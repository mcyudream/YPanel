package service

import (
	"context"
	"log/slog"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/dbdriver"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// 备份根目录（agent 主机侧）。
const backupBaseDir = "/opt/ypanel/backups"

// namePatternDB 实例名白名单。
var namePatternDB = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}[a-z0-9]$`)

// passwordPattern 实例密码白名单（排除 shell/引号敏感字符）。
var passwordPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// 支持的数据库类型与默认配置。
var dbDefaults = map[string]struct {
	image    string
	port     int
	user     string
	svcName  string
}{
	"mysql":    {image: "mysql:8", port: 33061, user: "root", svcName: "mysql"},
	"postgres": {image: "postgres:16", port: 35432, user: "postgres", svcName: "postgres"},
	"redis":    {image: "redis:7", port: 36379, user: "default", svcName: "redis"},
	"mongo":    {image: "mongo:7", port: 37017, user: "root", svcName: "mongo"},
}

// DatabaseService 数据库实例管理。
type DatabaseService struct {
	db    *gorm.DB
	nodes *NodeService
	aesKey []byte
}

// NewDatabaseService 创建服务（加密密钥由 JWT 密钥派生）。
func NewDatabaseService(db *gorm.DB, nodes *NodeService, jwtSecret string) *DatabaseService {
	sum := sha256.Sum256([]byte("ypanel-dbcred:" + jwtSecret))
	return &DatabaseService{db: db, nodes: nodes, aesKey: sum[:]}
}

// encryptPassword AES-256-GCM 加密。
func (s *DatabaseService) encryptPassword(plain string) (string, error) {
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	enc := gcm.Seal(nil, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, enc...)), nil
}

// decryptPassword 解密。
func (s *DatabaseService) decryptPassword(enc string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errs.New(errs.CodeInternal, "error.internal", "凭据数据损坏")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *DatabaseService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// composeYAML 生成实例的 compose 配置。
func composeYAML(inst *model.DatabaseInstance, password string) (string, error) {
	def, ok := dbDefaults[inst.Type]
	if !ok {
		return "", errs.Wrap(errs.ErrBadRequest, "不支持的类型: "+inst.Type)
	}
	var b strings.Builder
	b.WriteString("services:\n")
	b.WriteString("  " + def.svcName + ":\n")
	b.WriteString("    image: " + def.image + "\n")
	b.WriteString("    container_name: " + inst.ComposeProject + "\n")
	fmt.Fprintf(&b, "    ports:\n      - \"%d:%d\"\n", inst.Port, defPort(inst.Type))
	b.WriteString("    volumes:\n      - ./data:" + dataDirOf(inst.Type) + "\n")
	b.WriteString("    restart: unless-stopped\n")
	switch inst.Type {
	case "mysql":
		fmt.Fprintf(&b, "    environment:\n      MYSQL_ROOT_PASSWORD: \"%s\"\n", password)
	case "postgres":
		fmt.Fprintf(&b, "    environment:\n      POSTGRES_PASSWORD: \"%s\"\n", password)
	case "redis":
		b.WriteString("    command: [\"redis-server\", \"--requirepass\", \"" + password + "\"]\n")
	case "mongo":
		fmt.Fprintf(&b, "    environment:\n      MONGO_INITDB_ROOT_USERNAME: \"%s\"\n      MONGO_INITDB_ROOT_PASSWORD: \"%s\"\n", inst.RootUser, password)
	}
	return b.String(), nil
}

func defPort(t string) int {
	switch t {
	case "mysql":
		return 3306
	case "postgres":
		return 5432
	case "redis":
		return 6379
	case "mongo":
		return 27017
	}
	return 0
}

func dataDirOf(t string) string {
	switch t {
	case "mysql":
		return "/var/lib/mysql"
	case "postgres":
		return "/var/lib/postgresql/data"
	case "redis":
		return "/data"
	case "mongo":
		return "/data/db"
	}
	return "/data"
}

// CreateInstance 创建实例：元数据 + compose 项目 + up。
func (s *DatabaseService) CreateInstance(ctx context.Context, name, dbType string, port int, password string) (*model.DatabaseInstance, error) {
	if !namePatternDB.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "实例名不合法（小写字母开头，小写字母/数字/中划线，3-32 位）")
	}
	if _, ok := dbDefaults[dbType]; !ok {
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的数据库类型: "+dbType)
	}
	var count int64
	_ = s.db.Model(&model.DatabaseInstance{}).Where("name = ?", name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.instanceExists", "实例名已存在")
	}
	if password == "" {
		password = randomHex(12)
	}
	if !passwordPattern.MatchString(password) {
		return nil, errs.Wrap(errs.ErrBadRequest, "密码仅允许字母/数字/下划线/中划线，8-64 位")
	}
	if port == 0 {
		port = dbDefaults[dbType].port
	}
	if port < 1024 || port > 65535 {
		return nil, errs.Wrap(errs.ErrBadRequest, "端口需在 1024-65535")
	}
	rootUser := dbDefaults[dbType].user

	enc, err := s.encryptPassword(password)
	if err != nil {
		return nil, err
	}
	inst := &model.DatabaseInstance{
		Name: name, Type: dbType, Port: port, RootUser: rootUser,
		PasswordEnc: enc, ComposeProject: "db-" + name,
	}
	if err := s.db.Create(inst).Error; err != nil {
		return nil, err
	}

	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	yaml, err := composeYAML(inst, password)
	if err != nil {
		return nil, err
	}
	if _, err := agentclient.DoJSON[dto.ComposeWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/compose/config",
		&dto.ComposeWriteReq{Name: inst.ComposeProject, Content: yaml}); err != nil {
		return nil, err
	}
	if _, err := agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/up",
		&dto.ComposeActionReq{Name: inst.ComposeProject}); err != nil {
		return nil, err
	}
	return inst, nil
}

// DeleteInstance 删除实例：down + 元数据删除（purge 时同时清理 compose 目录与备份）。
func (s *DatabaseService) DeleteInstance(ctx context.Context, id uint, purge bool) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	// down 失败不阻塞（容器可能已不存在）
	_, _ = agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/down",
		&dto.ComposeActionReq{Name: inst.ComposeProject})
	if purge {
		_, _ = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
			&dto.FileDeleteReq{Paths: []string{path.Join("/opt/ypanel/compose", inst.ComposeProject)}})
		_, _ = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
			&dto.FileDeleteReq{Paths: []string{path.Join(backupBaseDir, inst.Type, inst.Name)}})
	}
	return s.db.Delete(&model.DatabaseInstance{}, id).Error
}

// StartStop 启动/停止实例（compose up/down）。
func (s *DatabaseService) StartStop(ctx context.Context, id uint, up bool) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	action := "down"
	if up {
		action = "up"
	}
	_, err = agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/"+action,
		&dto.ComposeActionReq{Name: inst.ComposeProject})
	return err
}

// List 实例列表（附带实时运行状态）。
func (s *DatabaseService) List(ctx context.Context) ([]map[string]any, error) {
	var rows []model.DatabaseInstance
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	projects, _ := agentclient.GetJSON[[]dto.ComposeProject](ac, ctx, "/agent/v1/compose/projects")
	running := map[string]bool{}
	for _, p := range projectsSafe(projects) {
		running[p.Name] = p.Running > 0
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"id": r.ID, "name": r.Name, "type": r.Type, "port": r.Port,
			"user": r.RootUser, "composeProject": r.ComposeProject,
			"running": running[r.ComposeProject], "createdAt": r.CreatedAt,
		})
	}
	return out, nil
}

// Reveal 连接信息（含解密密码）。
func (s *DatabaseService) Reveal(id uint) (map[string]any, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	pwd, err := s.decryptPassword(inst.PasswordEnc)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"name": inst.Name, "type": inst.Type, "port": inst.Port,
		"user": inst.RootUser, "password": pwd, "host": "127.0.0.1",
	}, nil
}

// ByID 查实例。
func (s *DatabaseService) ByID(id uint) (*model.DatabaseInstance, error) {
	var inst model.DatabaseInstance
	if err := s.db.First(&inst, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.instanceNotFound", "实例不存在")
	}
	return &inst, nil
}

// driverFor 为实例创建直连驱动。
func (s *DatabaseService) driverFor(inst *model.DatabaseInstance) (dbdriver.Driver, error) {
	pwd, err := s.decryptPassword(inst.PasswordEnc)
	if err != nil {
		return nil, err
	}
	return dbdriver.New(inst.Type, "127.0.0.1", inst.Port, inst.RootUser, pwd)
}

// Databases 库列表。
func (s *DatabaseService) Databases(ctx context.Context, id uint) ([]dbdriver.DatabaseInfo, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	drv, err := s.driverFor(inst)
	if err != nil {
		return nil, err
	}
	defer drv.Close()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := drv.Ping(ctx); err != nil {
		return nil, errs.Wrap(errs.ErrAgentUnreach, "连接实例失败（实例可能未就绪）: "+err.Error())
	}
	return drv.ListDatabases(ctx)
}

// CreateDatabase 建库。
func (s *DatabaseService) CreateDatabase(ctx context.Context, id uint, name, charset string) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	drv, err := s.driverFor(inst)
	if err != nil {
		return err
	}
	defer drv.Close()
	return drv.CreateDatabase(ctx, name, charset)
}

// DropDatabase 删库。
func (s *DatabaseService) DropDatabase(ctx context.Context, id uint, name string) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	drv, err := s.driverFor(inst)
	if err != nil {
		return err
	}
	defer drv.Close()
	return drv.DropDatabase(ctx, name)
}

// Users 用户列表。
func (s *DatabaseService) Users(ctx context.Context, id uint) ([]dbdriver.UserInfo, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	drv, err := s.driverFor(inst)
	if err != nil {
		return nil, err
	}
	defer drv.Close()
	return drv.ListUsers(ctx)
}

// CreateUser 建用户。
func (s *DatabaseService) CreateUser(ctx context.Context, id uint, name, host, password string) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	if !passwordPattern.MatchString(password) {
		return errs.Wrap(errs.ErrBadRequest, "密码仅允许字母/数字/下划线/中划线，8-64 位")
	}
	drv, err := s.driverFor(inst)
	if err != nil {
		return err
	}
	defer drv.Close()
	return drv.CreateUser(ctx, name, host, password)
}

// DropUser 删用户。
func (s *DatabaseService) DropUser(ctx context.Context, id uint, name, host string) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	drv, err := s.driverFor(inst)
	if err != nil {
		return err
	}
	defer drv.Close()
	return drv.DropUser(ctx, name, host)
}

// ChangeUserPassword 改密。
func (s *DatabaseService) ChangeUserPassword(ctx context.Context, id uint, name, host, password string) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	if !passwordPattern.MatchString(password) {
		return errs.Wrap(errs.ErrBadRequest, "密码仅允许字母/数字/下划线/中划线，8-64 位")
	}
	drv, err := s.driverFor(inst)
	if err != nil {
		return err
	}
	defer drv.Close()
	if err := drv.ChangePassword(ctx, name, host, password); err != nil {
		return err
	}
	// 改的是实例凭据用户（root/postgres/default）时，同步元数据，避免后续连接失败
	if name == inst.RootUser || (inst.Type == "redis" && name == "default") {
		enc, err := s.encryptPassword(password)
		if err != nil {
			return err
		}
		if err := s.db.Model(&model.DatabaseInstance{}).Where("id = ?", inst.ID).
			Update("password_enc", enc).Error; err != nil {
			return err
		}
	}
	return nil
}

// BackupDir 实例备份目录。
func (s *DatabaseService) BackupDir(inst *model.DatabaseInstance) string {
	return path.Join(backupBaseDir, inst.Type, inst.Name)
}

// BackupFileName 生成备份文件名。
func (s *DatabaseService) BackupFileName(inst *model.DatabaseInstance) string {
	ext := map[string]string{"mysql": "sql", "postgres": "sql", "redis": "rdb", "mongo": "archive"}
	return fmt.Sprintf("%s-%s.%s", inst.Name, time.Now().Format("20060102-150405"), ext[inst.Type])
}

// CreateBackup 执行备份（agent exec 调容器内工具，输出落宿主备份目录）。
func (s *DatabaseService) CreateBackup(ctx context.Context, id uint) (map[string]any, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	pwd, err := s.decryptPassword(inst.PasswordEnc)
	if err != nil {
		return nil, err
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	// 建备份目录
	if _, err := agentclient.DoJSON[dto.FileMkdirReq, struct{}](ac, ctx, "POST", "/agent/v1/files/mkdir",
		&dto.FileMkdirReq{Path: s.BackupDir(inst)}); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "mkdir: "+err.Error())
	}

	file := s.BackupFileName(inst)
	target := path.Join(s.BackupDir(inst), file)
	c := inst.ComposeProject
	var cmd string
	switch inst.Type {
	case "mysql":
		cmd = fmt.Sprintf("docker exec %s sh -c 'mysqldump -uroot -p\"%s\" --all-databases --single-transaction' > %s", c, pwd, target)
	case "postgres":
		cmd = fmt.Sprintf("docker exec %s pg_dumpall -U postgres > %s", c, target)
	case "redis":
		// SAVE 输出与告警均丢弃，避免污染 RDB 流
		cmd = fmt.Sprintf("docker exec %s sh -c 'redis-cli -a \"%s\" SAVE >/dev/null 2>&1; cat /data/dump.rdb' > %s", c, pwd, target)
	case "mongo":
		cmd = fmt.Sprintf("docker exec %s sh -c 'mongodump --archive --gzip -u %s -p \"%s\" --authenticationDatabase admin' > %s", c, inst.RootUser, pwd, target)
	}
	if cmd == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的备份类型: "+inst.Type)
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 1800})
	if err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "exec: "+err.Error())
	}
	if out.ExitCode != 0 || out.TimedOut {
		return map[string]any{"output": out.Output}, errs.Wrapc(errs.CodeFileOpFailed, "备份执行失败: "+firstLine(out.Output))
	}
	return map[string]any{"file": file, "path": target, "output": out.Output}, nil
}

// Backups 备份列表（扫目录）。
func (s *DatabaseService) Backups(ctx context.Context, id uint) ([]map[string]any, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	out, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+backupDirEscape(s.BackupDir(inst)))
	if err != nil {
		// 目录不存在视为空
		return []map[string]any{}, nil
	}
	files := make([]map[string]any, 0, len(out.Entries))
	for _, e := range out.Entries {
		if e.IsDir {
			continue
		}
		files = append(files, map[string]any{
			"name": e.Name, "sizeMb": float64(e.Size) / 1024 / 1024, "modTime": e.ModTime, "path": e.Path,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		a, _ := files[i]["name"].(string)
		b, _ := files[j]["name"].(string)
		return a > b
	})
	return files, nil
}

// DeleteBackup 删除备份文件。
func (s *DatabaseService) DeleteBackup(ctx context.Context, id uint, file string) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	if strings.Contains(file, "/") || strings.Contains(file, "..") {
		return errs.ErrPathInvalid
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
		&dto.FileDeleteReq{Paths: []string{path.Join(s.BackupDir(inst), file)}})
	return err
}

// Restore 恢复备份（危险操作）。
func (s *DatabaseService) Restore(ctx context.Context, id uint, file string) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	if strings.Contains(file, "/") || strings.Contains(file, "..") {
		return errs.ErrPathInvalid
	}
	pwd, err := s.decryptPassword(inst.PasswordEnc)
	if err != nil {
		return err
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	src := path.Join(s.BackupDir(inst), file)
	slog.Info("db restore", "file", file, "src", src, "len", len(cmd0Placeholder()))
	c := inst.ComposeProject
	var cmd string
	switch inst.Type {
	case "mysql":
		cmd = fmt.Sprintf("docker exec -i %s sh -c 'mysql -uroot -p\"%s\"' < %s", c, pwd, src)
		slog.Info("db restore cmd", "cmdLen", len(cmd), "cmdPrefix", cmd[:min(80, len(cmd))])
	case "postgres":
		cmd = fmt.Sprintf("docker exec -i %s psql -U postgres < %s", c, src)
	case "redis":
		// RDB 恢复：拷入后重启实例加载
		cmd = fmt.Sprintf("docker cp %s %s:/data/dump.rdb && docker restart %s", src, c, c)
		_ = pwd
	case "mongo":
		cmd = fmt.Sprintf("docker exec -i %s sh -c 'mongorestore --archive --gzip -u %s -p \"%s\" --authenticationDatabase admin --drop' < %s", c, inst.RootUser, pwd, src)
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 1800})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 || out.TimedOut {
		return errs.Wrapc(errs.CodeFileOpFailed, "恢复执行失败: "+firstLine(out.Output))
	}
	return nil
}

// BackupDownloadURL 信息由 api 层组装（复用 files download 通道）。
func (s *DatabaseService) BackupPath(inst *model.DatabaseInstance, file string) string {
	return path.Join(s.BackupDir(inst), file)
}

func cmd0Placeholder() string { return "" }

func projectsSafe(p *[]dto.ComposeProject) []dto.ComposeProject {
	if p == nil {
		return nil
	}
	return *p
}

func backupDirEscape(s string) string { return s }
