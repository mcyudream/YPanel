// MiddlewareService 可复用中间件实例管理（M68）。
// 职责：实例登记 CRUD、连接测试、版本探测、节点容器扫描发现与一键纳管、凭据 reveal。
// 类型能力经 middlewareRegistry 描述符注册表驱动（见 middleware_registry.go 通用规范）。
// 密码加解密复用 DatabaseService 的实现（同包私有方法，同一密钥派生）。
package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// MiddlewareService 中间件实例服务。
type MiddlewareService struct {
	db    *gorm.DB
	dbs   *DatabaseService // 复用凭据加解密（aesKey 同派生）
	nodes *NodeService
}

// NewMiddlewareService 创建。
func NewMiddlewareService(db *gorm.DB, dbs *DatabaseService, nodes *NodeService) *MiddlewareService {
	return &MiddlewareService{db: db, dbs: dbs, nodes: nodes}
}

// clientFor 按节点路由 agent 客户端（与 DatabaseService 同款）。
func (s *MiddlewareService) clientFor(nodeId string) (*agentclient.Client, error) {
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// execAgentCmd 节点命令执行助手。
func execAgentCmd(ac *agentclient.Client, ctx context.Context, cmd string, timeoutSecs int) (dto.ExecResp, error) {
	res, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeoutSecs})
	if err != nil {
		return dto.ExecResp{}, err
	}
	return *res, nil
}

// ---- 查询 ----

// List 实例清单（含类型描述符 label 与可复用基线判定）。
func (s *MiddlewareService) List(ctx context.Context) ([]map[string]any, error) {
	var rows []model.MiddlewareInstance
	if err := s.db.Order("type asc, name asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		item := map[string]any{
			"id": r.ID, "name": r.Name, "type": r.Type, "origin": r.Origin,
			"host": r.Host, "port": r.Port, "user": r.User, "version": r.Version,
			"containerName": r.ContainerName, "composeProject": r.ComposeProject,
			"ownerId": r.OwnerID, "remark": r.Remark,
			"createdAt": r.CreatedAt,
		}
		if d, ok := middlewareDescriptorOf(r.Type); ok {
			item["label"] = d.Label
			item["minReuseVersion"] = d.MinReuseVersion
			item["reusable"] = middlewareVersionOK(r.Version, d.MinReuseVersion)
		} else {
			item["label"] = r.Type
			item["reusable"] = true
		}
		out = append(out, item)
	}
	return out, nil
}

// Types 已注册类型清单（前端添加表单与向导用）。
func (s *MiddlewareService) Types() []map[string]any {
	keys := make([]string, 0, len(middlewareRegistry))
	for k := range middlewareRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		d := middlewareRegistry[k]
		out = append(out, map[string]any{
			"type": d.Type, "label": d.Label, "defaultPort": d.DefaultPort,
			"minReuseVersion": d.MinReuseVersion, "injectPrefix": d.InjectPrefix,
		})
	}
	return out
}

// ---- 写路径 ----

// namePatternMW 实例名白名单（与数据库域同规）。
var namePatternMW = namePatternDB

// Create 手动登记外部实例。
func (s *MiddlewareService) Create(ctx context.Context, name, typ, host string, port int, user, password, remark string) (*model.MiddlewareInstance, error) {
	if !namePatternMW.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "实例名不合法（小写字母开头，小写字母/数字/中划线，3-32 位）")
	}
	d, ok := middlewareDescriptorOf(typ)
	if !ok {
		return nil, errs.Wrap(errs.ErrBadRequest, "未注册的中间件类型: "+typ)
	}
	if strings.TrimSpace(host) == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "主机地址必填")
	}
	port = middlewarePortOf(typ, port)
	var count int64
	_ = s.db.Model(&model.MiddlewareInstance{}).Where("name = ?", name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.instanceExists", "实例名已存在")
	}
	enc := ""
	if password != "" {
		e, err := s.dbs.encryptPassword(password)
		if err != nil {
			return nil, err
		}
		enc = e
	}
	inst := &model.MiddlewareInstance{
		Name: name, Type: d.Type, Origin: "external", Host: host, Port: port,
		User: user, PasswordEnc: enc, ComposeProject: "-", Remark: truncStr(remark, 255),
	}
	// 连通性预检（失败即拒，与数据库域同口径）
	if err := d.TestConn(ctx, inst.Host, inst.Port, user, password); err != nil {
		return nil, err
	}
	if err := s.db.Create(inst).Error; err != nil {
		return nil, err
	}
	return inst, nil
}

// Update 修改实例（密码留空=不变）。
func (s *MiddlewareService) Update(ctx context.Context, id uint, host string, port int, user, password, remark string) (*model.MiddlewareInstance, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	if host != "" {
		inst.Host = host
	}
	if port > 0 {
		inst.Port = port
	}
	if user != "" {
		inst.User = user
	}
	if password != "" {
		enc, err := s.dbs.encryptPassword(password)
		if err != nil {
			return nil, err
		}
		inst.PasswordEnc = enc
	}
	if remark != "" {
		inst.Remark = truncStr(remark, 255)
	}
	if err := s.db.Save(inst).Error; err != nil {
		return nil, err
	}
	return inst, nil
}

// Delete 删除登记（不动实例本体，仅摘除面板纳管记录）。
func (s *MiddlewareService) Delete(id uint) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	return s.db.Delete(inst).Error
}

// ByID 按 ID 取实例。
func (s *MiddlewareService) ByID(id uint) (*model.MiddlewareInstance, error) {
	var inst model.MiddlewareInstance
	if err := s.db.First(&inst, id).Error; err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "中间件实例不存在")
	}
	return &inst, nil
}

// Test 测指定实例连通性。
func (s *MiddlewareService) Test(ctx context.Context, id uint) error {
	inst, err := s.ByID(id)
	if err != nil {
		return err
	}
	pwd, _ := s.dbs.decryptPassword(inst.PasswordEnc)
	d, ok := middlewareDescriptorOf(inst.Type)
	if !ok {
		return errs.Wrap(errs.ErrBadRequest, "未注册的中间件类型: "+inst.Type)
	}
	return d.TestConn(ctx, inst.Host, inst.Port, inst.User, pwd)
}

// Reveal 凭据查看（密码明文，权限闸在 API 层）。
func (s *MiddlewareService) Reveal(id uint) (map[string]any, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	pwd, _ := s.dbs.decryptPassword(inst.PasswordEnc)
	return map[string]any{"id": inst.ID, "name": inst.Name, "user": inst.User, "password": pwd}, nil
}

// RefreshVersion 探测并回写实例版本（container origin 走容器内命令，external 取跳过）。
func (s *MiddlewareService) RefreshVersion(ctx context.Context, id uint, nodeId string) (string, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return "", err
	}
	d, ok := middlewareDescriptorOf(inst.Type)
	if !ok {
		return "", errs.Wrap(errs.ErrBadRequest, "未注册的中间件类型: "+inst.Type)
	}
	v := ""
	if inst.Origin == "container" && inst.ContainerName != "" {
		v = s.middlewareVersionProbe(ctx, nodeId, inst.ContainerName, d.VersionProbe)
	}
	if v == "" {
		return inst.Version, nil
	}
	inst.Version = v
	if err := s.db.Save(inst).Error; err != nil {
		return "", err
	}
	return v, nil
}

// ---- 扫描发现与纳管 ----

// MiddlewareScanItem 扫描发现条目。
type MiddlewareScanItem struct {
	Type          string   `json:"type"`
	Label         string   `json:"label"`
	ContainerName string   `json:"containerName"`
	Image         string   `json:"image"`
	Version       string   `json:"version"`       // 镜像 tag 推断
	Networks      []string `json:"networks"`
	ComposeDir    string   `json:"composeDir,omitempty"`
	ComposeProject string  `json:"composeProject,omitempty"`
	User          string   `json:"user,omitempty"`
	Password      string   `json:"password,omitempty"` // 容器 env 提取（仅纳管落库用，前端展示脱敏）
	Reused        bool     `json:"reused"`             // 已被纳管
	Reusable      bool     `json:"reusable"`           // 版本达基线
	Adoptable     bool     `json:"admittable"`
}

// Scan 扫描节点上可发现的中间件容器（镜像规则匹配 + env/ports 推参数）。
func (s *MiddlewareService) Scan(ctx context.Context, nodeId string) ([]MiddlewareScanItem, error) {
	ac, err := s.clientFor(nodeId)
	if err != nil {
		return nil, err
	}
	// 一次拿全容器：名称|镜像|网络|compose 项目目录（labels 缺失容错）
	cmd := `docker ps --format '{{.Names}}|{{.Image}}|{{.Label "com.docker.compose.project.working_dir"}}|{{.Label "com.docker.compose.project"}}'`
	res, err := execAgentCmd(ac, ctx, cmd, 30)
	if err != nil {
		return nil, err
	}
	// 已纳管容器集合（去重标记）
	var managed []model.MiddlewareInstance
	_ = s.db.Find(&managed).Error
	managedSet := map[string]bool{}
	for _, m := range managed {
		if m.ContainerName != "" {
			managedSet[m.ContainerName] = true
		}
	}
	out := make([]MiddlewareScanItem, 0)
	for _, line := range strings.Split(res.Output, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 2 {
			continue
		}
		name, image := parts[0], parts[1]
		workDir, project := "", ""
		if len(parts) >= 4 {
			workDir, project = parts[2], parts[3]
		}
		typ, d, ok := middlewareTypeByImage(image)
		if !ok {
			continue
		}
		item := MiddlewareScanItem{
			Type: typ, Label: d.Label, ContainerName: name, Image: image,
			Version:   versionFromImage(image),
			Reusable:  middlewareVersionOK(versionFromImage(image), d.MinReuseVersion),
			ComposeDir: workDir, ComposeProject: project,
			Reused: managedSet[name],
		}
		// 凭据 env 提取（匹配 descriptor 声明的键）
		if d.CredentialEnv[0] != "" || d.CredentialEnv[1] != "" {
			insp, err := execAgentCmd(ac, ctx, fmt.Sprintf("docker inspect %s --format '{{range .Config.Env}}{{println .}}{{end}}'", shellQuote(name)), 20)
			if err == nil {
				for _, el := range strings.Split(insp.Output, "\n") {
					el = strings.TrimRight(el, "\r")
					if k, v, found := strings.Cut(el, "="); found {
						if d.CredentialEnv[0] != "" && k == d.CredentialEnv[0] {
							item.User = v
						}
						if d.CredentialEnv[1] != "" && k == d.CredentialEnv[1] {
							item.Password = v
						}
					}
				}
			}
		}
		item.Adoptable = !item.Reused
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type+out[i].ContainerName < out[j].Type+out[j].ContainerName })
	return out, nil
}

// Adopt 纳管扫描条目为登记实例（container origin，容器名直连）。
func (s *MiddlewareService) Adopt(ctx context.Context, nodeId, containerName, customName string) (*model.MiddlewareInstance, error) {
	items, err := s.Scan(ctx, nodeId)
	if err != nil {
		return nil, err
	}
	var hit *MiddlewareScanItem
	for i := range items {
		if items[i].ContainerName == containerName {
			hit = &items[i]
			break
		}
	}
	if hit == nil {
		return nil, errs.Wrap(errs.ErrNotFound, "容器不在可发现清单中（镜像未匹配任何已注册中间件类型）")
	}
	if !hit.Adoptable {
		return nil, errs.New(errs.CodeConflict, "error.instanceExists", "该容器已纳管")
	}
	name := customName
	if name == "" {
		name = sanitizeInstanceName(hit.Type + "-" + containerName)
	}
	if !namePatternMW.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "实例名不合法（小写字母开头，小写字母/数字/中划线，3-32 位）: "+name)
	}
	var count int64
	_ = s.db.Model(&model.MiddlewareInstance{}).Where("name = ?", name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.instanceExists", "实例名已存在")
	}
	enc := ""
	if hit.Password != "" {
		e, err := s.dbs.encryptPassword(hit.Password)
		if err != nil {
			return nil, err
		}
		enc = e
	}
	inst := &model.MiddlewareInstance{
		Name: name, Type: hit.Type, Origin: "container",
		Host: hit.ContainerName, Port: middlewarePortOf(hit.Type, 0),
		User: hit.User, PasswordEnc: enc, Version: hit.Version,
		ContainerName: hit.ContainerName, ComposeProject: hit.ComposeProject,
	}
	if err := s.db.Create(inst).Error; err != nil {
		return nil, err
	}
	return inst, nil
}

// sanitizeInstanceName 容器名 → 合法实例名（非法字符转中划线、压长度、首字母合法性）。
func sanitizeInstanceName(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' {
			b.WriteRune(c)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 32 {
		out = out[:32]
	}
	out = strings.Trim(out, "-")
	if out == "" || out[0] < 'a' || out[0] > 'z' {
		out = "mw-" + out
	}
	if len(out) > 32 {
		out = out[:32]
	}
	return strings.Trim(out, "-")
}

// ---- 复用联动（商店安装向导）----

// SetOwner 数据范围属主设置。
func (s *MiddlewareService) SetOwner(id, ownerID uint) error {
	return s.db.Model(&model.MiddlewareInstance{}).Where("id = ?", id).Update("owner_id", ownerID).Error
}

// ReuseOptions 商店安装向导的复用候选（脱敏：不含凭据；含版本基线判定）。
func (s *MiddlewareService) ReuseOptions(ctx context.Context, typ, prefix string) ([]map[string]any, error) {
	rows, err := s.reuseCandidates(ctx, typ)
	if err != nil {
		return nil, err
	}
	d, _ := middlewareDescriptorOf(typ)
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"id":              r.ID,
			"name":            r.Name,
			"type":            r.Type,
			"origin":          r.Origin,
			"host":            r.Host,
			"port":            r.Port,
			"version":         r.Version,
			"containerName":   r.ContainerName,
			"label":           d.Label,
			"minReuseVersion": d.MinReuseVersion,
			"reusable":        middlewareVersionOK(r.Version, d.MinReuseVersion),
			"injectPrefix":    d.InjectPrefix,
			"declaredPrefix":  prefix,
		})
	}
	return out, nil
}

// ReuseCandidates 复用候选：按类型列出可复用实例（登记表；含版本基线判定与凭据解密供注入）。
// 返回 map 方便按 id 选用；仅本域内部使用（API 层不暴露凭据）。
func (s *MiddlewareService) reuseCandidates(ctx context.Context, typ string) ([]model.MiddlewareInstance, error) {
	d, ok := middlewareDescriptorOf(typ)
	if !ok {
		return nil, errs.Wrap(errs.ErrBadRequest, "未注册的中间件类型: "+typ)
	}
	var rows []model.MiddlewareInstance
	if err := s.db.Where("type = ?", d.Type).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// credentialOf 解密实例凭据（空密码返回空串）。
func (s *MiddlewareService) credentialOf(inst *model.MiddlewareInstance) (string, error) {
	if inst.PasswordEnc == "" {
		return "", nil
	}
	return s.dbs.decryptPassword(inst.PasswordEnc)
}
