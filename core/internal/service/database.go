package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/rbac"
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
	image   string
	port    int
	user    string
	svcName string
}{
	"mysql":    {image: "mysql:8", port: 33061, user: "root", svcName: "mysql"},
	"postgres": {image: "postgres:16", port: 35432, user: "postgres", svcName: "postgres"},
	"redis":    {image: "redis:7", port: 36379, user: "default", svcName: "redis"},
	"mongo":    {image: "mongo:7", port: 37017, user: "root", svcName: "mongo"},
}

// DatabaseService 数据库实例管理。
type DatabaseService struct {
	db      *gorm.DB
	nodes   *NodeService
	tasks   *TaskService
	aesKey  []byte
	storage *StorageService // 可选：备份产物远程上传（M34）
}

// SetStorage 注入远程存储服务（main 装配，避免构造签名变更）。
func (s *DatabaseService) SetStorage(st *StorageService) { s.storage = st }

// BackupUploadOpts 备份上传选项（M34）。
type BackupUploadOpts struct {
	StorageAccountID uint // 0 = 仅本地
	Keep             int  // 远端保留份数（>0 生效）
}

// NewDatabaseService 创建服务（加密密钥由 JWT 密钥派生）。
func NewDatabaseService(db *gorm.DB, nodes *NodeService, tasks *TaskService, jwtSecret string) *DatabaseService {
	sum := sha256.Sum256([]byte("ypanel-dbcred:" + jwtSecret))
	return &DatabaseService{db: db, nodes: nodes, tasks: tasks, aesKey: sum[:]}
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
	return s.clientFor("local")
}

// clientFor 按节点路由 agent 客户端（M55：商店节点安装的接管/探活在目标节点执行）。
func (s *DatabaseService) clientFor(nodeId string) (*agentclient.Client, error) {
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// composeYAML 生成实例的 compose 配置。
// 实例容器接入统一网络 ypanel_default（external）：同节点商店应用外接时才能以「容器名:内部端口」
// docker DNS 直连；不接入则实例只在自己的 compose 项目网络里，同节点外接只能退到 host 代指。
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
	b.WriteString("    networks:\n      - " + PanelNetwork + "\n")
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
	b.WriteString("networks:\n  " + PanelNetwork + ":\n    external: true\n")
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
		Name: name, Type: dbType, Origin: "container", Host: "127.0.0.1", Port: port, RootUser: rootUser,
		PasswordEnc: enc, ComposeProject: "db-" + name,
	}
	if err := s.db.Create(inst).Error; err != nil {
		return nil, err
	}

	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	// 实例容器接入统一网络（external 引用，网络必须已存在）
	if err := ensurePanelNetwork(ctx, ac); err != nil {
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

// hostPatternDB 外部实例主机（IPv4/IPv6/域名，不做连通性限制）。
var hostPatternDB = regexp.MustCompile(`^[a-zA-Z0-9._:-]{1,253}$`)

// AddExternalInstance 接入外部数据库实例（直连纳管，不创建容器）。
func (s *DatabaseService) AddExternalInstance(ctx context.Context, name, dbType, host string, port int, user, password, remark string) (*model.DatabaseInstance, error) {
	if !namePatternDB.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "实例名不合法（小写字母开头，小写字母/数字/中划线，3-32 位）")
	}
	if _, ok := dbDefaults[dbType]; !ok {
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的数据库类型: "+dbType)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if !hostPatternDB.MatchString(host) {
		return nil, errs.Wrap(errs.ErrBadRequest, "主机地址不合法")
	}
	if port < 1 || port > 65535 {
		return nil, errs.Wrap(errs.ErrBadRequest, "端口需在 1-65535")
	}
	if user == "" {
		user = dbDefaults[dbType].user
	}
	if password == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "外部实例必须提供连接密码")
	}
	var count int64
	_ = s.db.Model(&model.DatabaseInstance{}).Where("name = ?", name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.instanceExists", "实例名已存在")
	}
	enc, err := s.encryptPassword(password)
	if err != nil {
		return nil, err
	}
	inst := &model.DatabaseInstance{
		Name: name, Type: dbType, Origin: "external", Host: host, Port: port,
		RootUser: user, PasswordEnc: enc, ComposeProject: "-", Remark: truncStr(remark, 255),
	}
	// 连通性预检
	drv, err := dbdriver.New(dbType, inst.Host, inst.Port, inst.RootUser, password)
	if err == nil {
		cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		perr := drv.Ping(cctx)
		cancel()
		drv.Close()
		if perr != nil {
			return nil, errs.Wrap(errs.ErrAgentUnreach, "连接实例失败: "+perr.Error())
		}
	}
	if err := s.db.Create(inst).Error; err != nil {
		return nil, err
	}
	return inst, nil
}

// DeleteInstance 删除实例：down + 元数据删除（purgeData 清 compose 目录含数据；purgeBackups 清备份）。
func (s *DatabaseService) DeleteInstance(ctx context.Context, id uint, purgeData bool, purgeBackups bool) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	if inst.Origin == "external" {
		// 外部实例仅解除纳管，不触碰远端
		return s.db.Delete(&model.DatabaseInstance{}, id).Error
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	// down 失败不阻塞（容器可能已不存在）；商店接管来源的应用容器由商店卸载管线管理
	if inst.Origin != "store" {
		_, _ = agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/down",
			&dto.ComposeActionReq{Name: inst.ComposeProject})
	}
	if purgeData && inst.Origin != "store" {
		_, _ = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
			&dto.FileDeleteReq{Paths: []string{path.Join("/opt/ypanel/compose", inst.ComposeProject)}})
	}
	if purgeBackups {
		_, _ = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
			&dto.FileDeleteReq{Paths: []string{path.Join(backupBaseDir, inst.Type, inst.Name)}})
	}
	return s.db.Delete(&model.DatabaseInstance{}, id).Error
}

// StartStop 启动/停止实例（compose up/down；外部实例由本机服务自管）。
func (s *DatabaseService) StartStop(ctx context.Context, id uint, up bool) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	if inst.Origin == "external" {
		return errs.Wrap(errs.ErrBadRequest, "外部实例由其所在主机管理，面板不支持启停")
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
	// M54-P3：assigned 数据范围仅见 owner∈{uid,0}
	if caller, ok := rbac.CallerFrom(ctx); ok && caller.Assigned() {
		kept := rows[:0:0]
		for _, r := range rows {
			if r.OwnerID == 0 || r.OwnerID == caller.UserID {
				kept = append(kept, r)
			}
		}
		rows = kept
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	projects, _ := agentclient.GetJSON[[]dto.ComposeProject](ac, ctx, "/agent/v1/compose/projects")
	runningMap := map[string]bool{}
	for _, p := range projectsSafe(projects) {
		runningMap[p.Name] = p.Running > 0
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		running := false
		if r.Origin == "external" {
			// 外部实例以直连可达为准（异步探测代价高，仅在查看库列表时校验）
			running = true
		} else {
			running = runningMap[r.ComposeProject]
		}
		out = append(out, map[string]any{
			"id": r.ID, "name": r.Name, "type": r.Type, "origin": r.Origin, "host": r.Host,
			"port": r.Port, "user": r.RootUser, "remark": r.Remark,
			"composeProject": r.ComposeProject, "ownerId": r.OwnerID,
			"running": running, "createdAt": r.CreatedAt,
		})
	}
	// 商店已安装的数据库类应用，未接管则作为待接管条目展示
	out = append(out, s.adoptableItems(runningMap)...)
	return out, nil
}

// StartAutoAdopt 启动时扫描一次存量：商店已装、未纳管的数据库应用等就绪后自动接管。
func (s *DatabaseService) StartAutoAdopt(ctx context.Context) {
	go func() {
		var installs []model.AppStoreInstall
		if err := s.db.Where("key IN ?", dbServiceKeyList()).Find(&installs).Error; err != nil {
			return
		}
		for _, i := range installs {
			var count int64
			_ = s.db.Model(&model.DatabaseInstance{}).Where("compose_project = ?", i.ComposeProject).Count(&count).Error
			if count > 0 {
				continue
			}
			actx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			if _, err := s.AdoptWithRetry(actx, i.ComposeProject, i.NodeID, 6, 10*time.Second); err == nil {
				slog.Info("商店数据库应用已自动接管", "project", i.ComposeProject, "type", i.Key)
			}
			cancel()
		}
	}()
}

// AdoptWithRetry 带重试的接管（容器初始化未就绪时 Ping 失败，等间隔重试直至上限）。
func (s *DatabaseService) AdoptWithRetry(ctx context.Context, project, nodeId string, attempts int, interval time.Duration) (*model.DatabaseInstance, error) {
	var last error
	for i := 0; i < attempts; i++ {
		row, err := s.Adopt(ctx, project, nodeId)
		if err == nil {
			return row, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return nil, last
		case <-time.After(interval):
		}
	}
	return nil, last
}

// dbServiceKeys 可接管的商店数据库应用 key。
// 注意 1Panel 官方源 PostgreSQL 的 key 是 postgresql（驱动类型为 postgres，见 dbServiceTypeAlias）。
var dbServiceKeys = map[string]bool{"mysql": true, "postgres": true, "postgresql": true, "redis": true, "mongo": true}

// dbServiceTypeAlias 商店应用 key → 数据库驱动类型。
func dbServiceTypeAlias(key string) string {
	if key == "postgresql" {
		return "postgres"
	}
	return key
}

// dbServiceKeyList dbServiceKeys 的稳定序键表（查询条件复用，避免多处硬编码漂移）。
func dbServiceKeyList() []string {
	keys := make([]string, 0, len(dbServiceKeys))
	for k := range dbServiceKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// adoptableItems 商店已装、尚未纳管的数据库应用（running 映射由调用方传入复用）。
func (s *DatabaseService) adoptableItems(running map[string]bool) []map[string]any {
	var installs []model.AppStoreInstall
	if err := s.db.Where("key IN ?", dbServiceKeyList()).Find(&installs).Error; err != nil {
		return nil
	}
	out := []map[string]any{}
	for _, i := range installs {
		var count int64
		_ = s.db.Model(&model.DatabaseInstance{}).Where("compose_project = ?", i.ComposeProject).Count(&count).Error
		if count > 0 {
			continue
		}
		out = append(out, map[string]any{
			"id": 0, "name": i.Name, "type": dbServiceTypeAlias(i.Key), "origin": "store", "host": "127.0.0.1",
			"port": 0, "user": "", "remark": "商店应用（待接管）",
			"composeProject": i.ComposeProject, "running": running[i.ComposeProject],
			"adoptable": true, "createdAt": i.CreatedAt,
		})
	}
	return out
}

// Adopt 接管商店已安装的数据库应用：从应用 .env 解析凭据与端口，直连验证后入库纳管。
func (s *DatabaseService) Adopt(ctx context.Context, project, nodeId string) (*model.DatabaseInstance, error) {
	var inst model.AppStoreInstall
	if err := s.db.Where("compose_project = ?", project).First(&inst).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.installNotFound", "商店安装记录不存在")
	}
	if !dbServiceKeys[inst.Key] {
		return nil, errs.Wrap(errs.ErrBadRequest, "该应用不是可接管的数据库类型: "+inst.Key)
	}
	dbType := dbServiceTypeAlias(inst.Key) // postgresql → postgres（驱动/默认值按规范类型）
	var count int64
	_ = s.db.Model(&model.DatabaseInstance{}).Where("compose_project = ? OR name = ?", project, inst.Name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.instanceExists", "该应用已接管或实例名冲突")
	}
	// 读应用 .env 解析凭据与宿主端口（目标节点）
	ac, err := s.clientFor(nodeId)
	if err != nil {
		return nil, err
	}
	envOut, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+backupDirEscape("/opt/ypanel/compose/"+project+"/.env"))
	if err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "读取应用参数失败（.env 不存在）")
	}
	env := map[string]string{}
	for _, line := range strings.Split(envOut.Content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			env[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	pick := func(candidates ...string) string {
		for _, c := range candidates {
			if v := env[c]; v != "" {
				return v
			}
		}
		// 回退：任意 KEY 含 PASSWORD 的第一个
		for k, v := range env {
			if strings.Contains(strings.ToUpper(k), "PASSWORD") && v != "" {
				return v
			}
		}
		return ""
	}
	// 管理用户名：1P 官方包（如 postgresql 的 PANEL_DB_ROOT_USER）是随机生成的用户名，
	// 不能假设 root/postgres；读不到再回落该类型的默认管理用户。
	user := ""
	for _, c := range []string{"PANEL_DB_ROOT_USER", "POSTGRES_USER", "MYSQL_ROOT_USER", "MYSQL_USER", "MONGO_INITDB_ROOT_USERNAME"} {
		if v := env[c]; v != "" {
			user = v
			break
		}
	}
	if user == "" {
		user = dbDefaults[dbType].user
	}
	port := 0
	for _, c := range []string{"PANEL_APP_PORT_HTTP", "PANEL_APP_PORT", "POSTGRES_PORT", "PGSQL_PORT", "MYSQL_PORT", "REDIS_PORT", "MONGO_PORT", "DATABASE_PORT"} {
		if v := env[c]; v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n < 65536 {
				port = n
				break
			}
		}
	}
	if port == 0 { // 回退：任意含 PORT 的键（排序保证确定性）
		keys := make([]string, 0, len(env))
		for k := range env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if strings.Contains(strings.ToUpper(k), "PORT") {
				if n, err := strconv.Atoi(env[k]); err == nil && n > 0 && n < 65536 {
					port = n
					break
				}
			}
		}
	}
	password := pick("PANEL_DB_ROOT_PASSWORD", "MYSQL_ROOT_PASSWORD", "POSTGRES_PASSWORD", "MONGO_INITDB_ROOT_PASSWORD", "PANEL_REDIS_PASSWORD", "REDIS_PASSWORD")
	if password == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "未能从应用参数中解析出密码，请改用「接入外部实例」手动填写")
	}
	if port == 0 {
		return nil, errs.Wrap(errs.ErrBadRequest, "未能从应用参数中解析出宿主端口")
	}
	name := inst.Name
	if !namePatternDB.MatchString(name) {
		name = "app-" + strings.ToLower(inst.Name)
	}
	enc, err := s.encryptPassword(password)
	if err != nil {
		return nil, err
	}
	row := &model.DatabaseInstance{
		Name: name, Type: dbType, Origin: "container", Host: "127.0.0.1",
		Port: port, RootUser: user, PasswordEnc: enc,
		ComposeProject: project, Remark: "商店应用接管",
	}
	// 连通性验证（容器映射在宿主 127.0.0.1:port）
	drv, derr := dbdriver.New(dbType, row.Host, row.Port, row.RootUser, password)
	if derr == nil {
		cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		perr := drv.Ping(cctx)
		cancel()
		drv.Close()
		if perr != nil {
			return nil, errs.Wrap(errs.ErrAgentUnreach, "接管验证失败（实例未就绪或凭据不符）: "+perr.Error())
		}
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
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
	host := inst.Host
	if host == "" {
		host = "127.0.0.1"
	}
	out := map[string]any{
		"name": inst.Name, "type": inst.Type, "port": inst.Port,
		"user": inst.RootUser, "password": pwd, "host": host, "origin": inst.Origin,
	}
	// M32：合并容器/外部双视角端点（容器名:内部端口 供同网络容器使用；宿主内网 IP:映射端口 供外部访问）
	if info, err := s.ConnectInfo(context.Background(), id); err == nil {
		out["container"] = info.Container
		out["innerPort"] = info.InnerPort
		out["networks"] = info.Networks
		out["lanIp"] = info.LanIP
		out["mapPort"] = info.MapPort
	}
	return out, nil
}

// ByID 查实例。
// SetInstanceOwner 属主再分配（0=公共；仅 all 数据范围调用方可达，由 API 层把关）。
func (s *DatabaseService) SetInstanceOwner(id uint, ownerID uint) error {
	return s.db.Model(&model.DatabaseInstance{}).Where("id = ?", id).Update("owner_id", ownerID).Error
}

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
	host := inst.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return dbdriver.New(inst.Type, host, inst.Port, inst.RootUser, pwd)
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
// RemoteAccess B3：远程访问开关（SQL 层授权/回收远端管理用户）。
func (s *DatabaseService) RemoteAccess(ctx context.Context, id uint, enable bool) (map[string]any, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	driver, err := s.driverFor(inst)
	if err != nil {
		return nil, err
	}
	defer driver.Close()
	pwd, err := s.decryptPassword(inst.PasswordEnc)
	if err != nil {
		return nil, err
	}
	if enable {
		err = driver.EnableRemote(ctx, pwd)
	} else {
		err = driver.DisableRemote(ctx)
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{"enabled": enable, "user": "remote"}, nil
}

// RemoteAccessStatus 远程访问状态回读（remote 管理用户是否存在）。
func (s *DatabaseService) RemoteAccessStatus(ctx context.Context, id uint) (bool, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return false, err
	}
	driver, err := s.driverFor(inst)
	if err != nil {
		return false, err
	}
	defer driver.Close()
	sp, ok := driver.(dbdriver.RemoteStatusProvider)
	if !ok {
		return false, nil
	}
	return sp.RemoteEnabled(ctx)
}

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

// CreateBackup 执行备份（agent exec 调容器内工具，输出落宿主备份目录；opts 可选远程上传）。
func (s *DatabaseService) CreateBackup(ctx context.Context, id uint, opts ...BackupUploadOpts) (map[string]any, error) {
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
	host := inst.Host
	if host == "" {
		host = "127.0.0.1"
	}
	// 密码只经 ExecReq.Env 注入 agent sh 环境（$YP_DB_PWD 引用），argv 无明文；
	// 容器内经 stdin `read -r pw` 中转，mongo 走 tools --config 临时文件（用完即删）。
	// 外接实例统一走临时容器（--network host 直连宿主网络栈，宿主无需客户端工具）。
	var cmd string
	switch inst.Type {
	case "mysql":
		if inst.Origin == "external" {
			cmd = fmt.Sprintf(dbToolPullPre("mysql")+`printf '%%s\n' "$YP_DB_PWD" | docker run --rm -i --network host %s sh -c 'read -r pw; cf=$(mktemp); printf "[client]\npassword=%%s\n" "$pw" > "$cf"; mysqldump --defaults-extra-file="$cf" -h %s -P %d -u %s --all-databases --single-transaction; rc=$?; rm -f "$cf"; exit $rc' > %s`,
				dbClientImage["mysql"], host, inst.Port, inst.RootUser, target)
		} else {
			cmd = fmt.Sprintf("printf '%%s\\n' \"$YP_DB_PWD\" | docker exec -i %s sh -c 'read -r pw; MYSQL_PWD=\"$pw\" mysqldump -uroot --all-databases --single-transaction' > %s", c, target)
		}
	case "postgres":
		if inst.Origin == "external" {
			cmd = fmt.Sprintf(dbToolPullPre("postgres")+`printf '%%s\n' "$YP_DB_PWD" | docker run --rm -i --network host %s sh -c 'read -r pw; PGPASSWORD="$pw" pg_dumpall -h %s -p %d -U %s' > %s`,
				dbClientImage["postgres"], host, inst.Port, inst.RootUser, target)
		} else {
			cmd = fmt.Sprintf("docker exec %s pg_dumpall -U postgres > %s", c, target)
		}
	case "redis":
		if inst.Origin == "external" {
			cmd = fmt.Sprintf(dbToolPullPre("redis")+`printf '%%s\n' "$YP_DB_PWD" | docker run --rm -i --network host %s sh -c 'read -r pw; REDISCLI_AUTH="$pw" redis-cli -h %s -p %d --rdb /tmp/yp-dump.rdb; rc=$?; [ "$rc" -eq 0 ] && cat /tmp/yp-dump.rdb; exit $rc' > %s`,
				dbClientImage["redis"], host, inst.Port, target)
		} else {
			// SAVE 输出与告警均丢弃，避免污染 RDB 流
			cmd = fmt.Sprintf("printf '%%s\\n' \"$YP_DB_PWD\" | docker exec -i %s sh -c 'read -r pw; REDISCLI_AUTH=\"$pw\" redis-cli SAVE >/dev/null 2>&1; cat /data/dump.rdb' > %s", c, target)
		}
	case "mongo":
		// mongodump/mongorestore 无密码环境变量约定，统一走 --config 临时文件（umask 077 + 用完即删）
		if inst.Origin == "external" {
			cmd = fmt.Sprintf(dbToolPullPre("mongo")+`printf '%%s\n' "$YP_DB_PWD" | docker run --rm -i --network host %s sh -c 'read -r pw; cf=$(mktemp); printf "password: %%s\n" "$pw" > "$cf"; mongodump --archive --gzip --host %s --port %d -u %s --config "$cf" --authenticationDatabase admin; rc=$?; rm -f "$cf"; exit $rc' > %s`,
				dbClientImage["mongo"], host, inst.Port, inst.RootUser, target)
		} else {
			cmd = fmt.Sprintf("printf '%%s\\n' \"$YP_DB_PWD\" | docker exec -i %s sh -c 'read -r pw; cf=$(mktemp); printf \"password: %%s\\n\" \"$pw\" > \"$cf\"; mongodump --archive --gzip -u %s --config \"$cf\" --authenticationDatabase admin; rc=$?; rm -f \"$cf\"; exit $rc' > %s", c, inst.RootUser, target)
		}
	}
	if cmd == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的备份类型: "+inst.Type)
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 1800, Env: map[string]string{"YP_DB_PWD": pwd}})
	if err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "exec: "+err.Error())
	}
	if out.ExitCode != 0 || out.TimedOut {
		return map[string]any{"output": out.Output}, errs.Wrapc(errs.CodeFileOpFailed, "备份执行失败: "+firstLine(out.Output))
	}
	result := map[string]any{"file": file, "path": target, "output": out.Output}
	// M34：可选远程上传（失败不影响本地产物，但按失败返回，注明本地成功）
	if len(opts) > 0 && opts[0].StorageAccountID > 0 && s.storage != nil {
		key, uerr := s.storage.UploadAgentFile(ctx, opts[0].StorageAccountID, "databases/"+inst.Type+"/"+inst.Name, target, opts[0].Keep)
		if uerr != nil {
			return result, errs.Wrapc(errs.CodeFileOpFailed, "本地备份成功，远端上传失败: "+uerr.Error())
		}
		result["remoteKey"] = key
	}
	return result, nil
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
	c := inst.ComposeProject
	host := inst.Host
	if host == "" {
		host = "127.0.0.1"
	}
	// 与备份同款安全模式：密码经 ExecReq.Env 注入；容器内经 stdin 首行 read 中转
	//（restore 的 stdin 同时承载备份流，故用 { printf 密码; cat 文件; } 拼接）。
	var cmd string
	switch inst.Type {
	case "mysql":
		if inst.Origin == "external" {
			cmd = fmt.Sprintf(dbToolPullPre("mysql")+`{ printf '%%s\n' "$YP_DB_PWD"; cat %s; } | docker run --rm -i --network host %s sh -c 'read -r pw; cf=$(mktemp); printf "[client]\npassword=%%s\n" "$pw" > "$cf"; mysql --defaults-extra-file="$cf" -h %s -P %d -u %s; rc=$?; rm -f "$cf"; exit $rc'`,
				src, dbClientImage["mysql"], host, inst.Port, inst.RootUser)
		} else {
			cmd = fmt.Sprintf("{ printf '%%s\\n' \"$YP_DB_PWD\"; cat %s; } | docker exec -i %s sh -c 'read -r pw; MYSQL_PWD=\"$pw\" mysql -uroot'", src, c)
		}
	case "postgres":
		if inst.Origin == "external" {
			cmd = fmt.Sprintf(dbToolPullPre("postgres")+`{ printf '%%s\n' "$YP_DB_PWD"; cat %s; } | docker run --rm -i --network host %s sh -c 'read -r pw; PGPASSWORD="$pw" psql -q -h %s -p %d -U %s'`,
				src, dbClientImage["postgres"], host, inst.Port, inst.RootUser)
		} else {
			cmd = fmt.Sprintf("docker exec -i %s psql -U postgres < %s", c, src)
		}
	case "redis":
		if inst.Origin == "external" {
			return errs.Wrap(errs.ErrBadRequest, "外部 Redis 不支持面板恢复（请由所在主机执行 RDB 恢复）")
		}
		// RDB 恢复：拷入后重启实例加载
		cmd = fmt.Sprintf("docker cp %s %s:/data/dump.rdb && docker restart %s", src, c, c)
	case "mongo":
		if inst.Origin == "external" {
			cmd = fmt.Sprintf(dbToolPullPre("mongo")+`{ printf '%%s\n' "$YP_DB_PWD"; cat %s; } | docker run --rm -i --network host %s sh -c 'read -r pw; cf=$(mktemp); printf "password: %%s\n" "$pw" > "$cf"; mongorestore --archive --gzip --host %s --port %d -u %s --config "$cf" --authenticationDatabase admin --drop; rc=$?; rm -f "$cf"; exit $rc'`,
				src, dbClientImage["mongo"], host, inst.Port, inst.RootUser)
		} else {
			cmd = fmt.Sprintf("{ printf '%%s\\n' \"$YP_DB_PWD\"; cat %s; } | docker exec -i %s sh -c 'read -r pw; cf=$(mktemp); printf \"password: %%s\\n\" \"$pw\" > \"$cf\"; mongorestore --archive --gzip --config \"$cf\" --authenticationDatabase admin --drop; rc=$?; rm -f \"$cf\"; exit $rc'", src, c)
		}
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 1800, Env: map[string]string{"YP_DB_PWD": pwd}})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 || out.TimedOut {
		return errs.Wrapc(errs.CodeFileOpFailed, "恢复执行失败: "+firstLine(out.Output))
	}
	return nil
}

// audit 迁移等实例级写操作的审计（DBAdminService.audit 的实例侧对等物）。
func (s *DatabaseService) audit(inst *model.DatabaseInstance, kind, target, detail string, success bool) {
	entry := model.DatabaseAuditLog{
		Username: "", InstanceID: inst.ID, InstanceName: inst.Name,
		DbType: inst.Type, Kind: kind, Target: target, Detail: detail, Success: success,
	}
	_ = s.db.Create(&entry).Error
}

// ExecAgent 面板主机命令执行（密码经 env 注入，供导入/迁移通道复用）。
func (s *DatabaseService) ExecAgent(ctx context.Context, cmd string, env map[string]string, timeoutSecs int) (dto.ExecResp, error) {
	ac, err := s.client()
	if err != nil {
		return dto.ExecResp{}, err
	}
	res, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeoutSecs, Env: env})
	if err != nil {
		return dto.ExecResp{}, err
	}
	return *res, nil
}

// BackupDownloadURL 信息由 api 层组装（复用 files download 通道）。
func (s *DatabaseService) BackupPath(inst *model.DatabaseInstance, file string) string {
	return path.Join(s.BackupDir(inst), file)
}

func projectsSafe(p *[]dto.ComposeProject) []dto.ComposeProject {
	if p == nil {
		return nil
	}
	return *p
}

func backupDirEscape(s string) string { return s }

// ---- M32：连接信息双视角（容器网络 / 外部宿主） ----

// DBEndpoint 数据库实例的容器访问视角端点。
type DBEndpoint struct {
	Container string `json:"container"` // 容器名（同网络容器用容器名直连）
	InnerPort string `json:"innerPort"` // 容器内部端口（mysql 3306 / postgres 5432）
	Networks  string `json:"networks"`  // 容器接入的网络（空格分隔）
	LanIP     string `json:"lanIp"`     // 宿主内网 IP（外部/跨网络访问）
	MapPort   string `json:"mapPort"`   // 宿主映射端口
	RootUser  string `json:"rootUser"`
	RootPass  string `json:"rootPass"`
}

// ConnectInfo 实例连接信息（含 root 凭据与容器/外部双视角端点）。
// 容器视角：同网络容器用 container:innerPort；外部视角：lanIP:mapPort（host 为回环时 lan 即唯一可达路径）。
func (s *DatabaseService) ConnectInfo(ctx context.Context, id uint) (*DBEndpoint, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	pwd, err := s.decryptPassword(inst.PasswordEnc)
	if err != nil {
		return nil, err
	}
	out := &DBEndpoint{
		RootUser: inst.RootUser,
		RootPass: pwd,
		LanIP:    inst.Host,
		MapPort:  fmt.Sprint(inst.Port),
	}
	ac, err := s.client()
	if err != nil {
		return out, nil
	}
	inner := "3306"
	if inst.Type == "postgres" {
		inner = "5432"
	}
	// 容器定位三级兜底：compose 项目标签 → 实例名精确匹配 → 映射端口反查
	//（外部接管实例无 compose 标签；容器名常与实例名一致；publish 端口唯一性最强）
	locFilters := []string{}
	if inst.ComposeProject != "" && inst.ComposeProject != "-" {
		locFilters = append(locFilters, fmt.Sprintf(`--filter label=com.docker.compose.project=%s`, inst.ComposeProject))
	}
	locFilters = append(locFilters,
		fmt.Sprintf(`[ -z "$c" ] && c=$(docker ps --filter name=^%s$ --format '{{.Names}}' | head -1);`, inst.Name),
		fmt.Sprintf(`[ -z "$c" ] && c=$(docker ps --filter publish=%d --format '{{.Names}}' | head -1);`, inst.Port),
	)
	// 注意：filter 参数含空格不能进 for 词列表，逐级短路赋值
	res, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf(
			`c=""; %s echo "C=$c"; [ -n "$c" ] && docker inspect "$c" --format 'P={{range $k,$v := .Config.ExposedPorts}}{{$k}} {{end}}N={{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}'`,
			strings.Join(locFilters, " ")), TimeoutSecs: 25})
	if err == nil {
		// P= 与 N= 在 inspect 输出的同一行（模板段拼接），用正则提取而非行前缀
		if c := regexp.MustCompile(`C=(\S+)`).FindStringSubmatch(res.Output); len(c) > 1 {
			out.Container = c[1]
		}
		if pv := regexp.MustCompile(`P=([^ \n]+)`).FindStringSubmatch(res.Output); len(pv) > 1 {
			inner = strings.Split(pv[1], "/")[0]
		}
		if nv := regexp.MustCompile(`N=(.+)`).FindStringSubmatch(res.Output); len(nv) > 1 {
			out.Networks = strings.TrimSpace(nv[1])
		}
	}
	out.InnerPort = inner
	// 外部视角：host 为回环（面板容器实例默认）时用宿主内网 IP，否则原样
	if out.LanIP == "" || out.LanIP == "127.0.0.1" || out.LanIP == "localhost" || out.LanIP == "::1" {
		if res, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: "hostname -I | awk '{print $1}'", TimeoutSecs: 15}); err == nil {
			if ip := strings.TrimSpace(res.Output); ip != "" {
				out.LanIP = ip
			}
		}
	}
	return out, nil
}

// GrantUserDatabase 授权账号对指定库的全部权限（M32；仅 mysql/postgres 实例支持）。
func (s *DatabaseService) GrantUserDatabase(ctx context.Context, instanceID uint, database, user, host string) error {
	inst, err := s.ByID(instanceID)
	if err != nil {
		return err
	}
	drv, err := s.driverFor(inst)
	if err != nil {
		return err
	}
	defer drv.Close()
	g, ok := drv.(dbdriver.DatabaseGranter)
	if !ok {
		return errs.Wrap(errs.ErrBadRequest, "该实例类型不支持库级授权")
	}
	return g.GrantDatabase(ctx, database, user, host)
}

// CreateExtensions 纳管 PG 实例在指定库批量创建扩展（幂等）——商店安装「建库建扩展」一键化的底层。
func (s *DatabaseService) CreateExtensions(ctx context.Context, id uint, dbName string, exts []string) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	if inst.Type != "postgres" {
		return errs.Wrap(errs.ErrBadRequest, "仅 PostgreSQL 实例支持创建扩展")
	}
	drv, err := s.driverFor(inst)
	if err != nil {
		return err
	}
	defer drv.Close()
	pe, ok := drv.(interface {
		CreateExtension(ctx context.Context, database, name string) error
	})
	if !ok {
		return errs.Wrap(errs.ErrBadRequest, "该实例驱动不支持扩展创建")
	}
	for _, e := range exts {
		if err := pe.CreateExtension(ctx, dbName, e); err != nil {
			return fmt.Errorf("创建扩展 %s 失败: %w", e, err)
		}
	}
	return nil
}

// ---- M39：MySQL 管理深化（授权矩阵/参数/状态，经 MySQLAdmin 能力探测） ----

func (s *DatabaseService) adminFor(inst *model.DatabaseInstance) (dbdriver.MySQLAdmin, func(), error) {
	drv, err := s.driverFor(inst)
	if err != nil {
		return nil, func() {}, err
	}
	admin, ok := drv.(dbdriver.MySQLAdmin)
	if !ok {
		drv.Close()
		return nil, func() {}, errs.Wrap(errs.ErrBadRequest, "该实例类型不支持（仅 MySQL 支持管理深化）")
	}
	return admin, func() { drv.Close() }, nil
}

// GrantMatrix 授权矩阵。
func (s *DatabaseService) GrantMatrix(ctx context.Context, id uint, db string) ([]dbdriver.GrantRow, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	admin, done, err := s.adminFor(inst)
	if err != nil {
		return nil, err
	}
	defer done()
	return admin.GrantMatrix(ctx, db)
}

// GrantPrivs 授予/回收权限。
func (s *DatabaseService) GrantPrivs(ctx context.Context, id uint, db, user, host string, privs []string, grant bool) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	admin, done, err := s.adminFor(inst)
	if err != nil {
		return err
	}
	defer done()
	if grant {
		err = admin.GrantPrivs(ctx, db, user, host, privs, true)
	} else {
		err = admin.GrantPrivs(ctx, db, user, host, privs, false)
	}
	if err == nil {
		s.audit(inst, "grant", db+"."+user+"@"+host, fmt.Sprintf("privs=%v grant=%v", privs, grant), true)
	} else {
		s.audit(inst, "grant", db+"."+user+"@"+host, err.Error(), false)
	}
	return err
}

// Variables 参数列表。
func (s *DatabaseService) Variables(ctx context.Context, id uint, filter string) ([]dbdriver.KVPair, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	admin, done, err := s.adminFor(inst)
	if err != nil {
		return nil, err
	}
	defer done()
	return admin.Variables(ctx, filter)
}

// SetVariable 在线改参数。
func (s *DatabaseService) SetVariable(ctx context.Context, id uint, name, value string) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	admin, done, err := s.adminFor(inst)
	if err != nil {
		return err
	}
	defer done()
	if err := admin.SetGlobalVariable(ctx, name, value); err != nil {
		return err
	}
	s.audit(inst, "variable", name, "SET GLOBAL "+name+" = "+value, true)
	return nil
}

// StatusStats 状态指标。
func (s *DatabaseService) StatusStats(ctx context.Context, id uint) (map[string]float64, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	admin, done, err := s.adminFor(inst)
	if err != nil {
		return nil, err
	}
	defer done()
	return admin.StatusStats(ctx)
}
