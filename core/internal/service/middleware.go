// MiddlewareService 可复用中间件实例管理（M68）。
// 职责：实例登记 CRUD、连接测试、版本探测、节点容器扫描发现与一键纳管、凭据 reveal。
// 类型能力经 middlewareRegistry 描述符注册表驱动（见 middleware_registry.go 通用规范）。
// 密码加解密复用 DatabaseService 的实现（同包私有方法，同一密钥派生）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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
	if res == nil {
		return dto.ExecResp{}, errs.Wrap(errs.ErrAgentUnreach, "节点未返回执行结果")
	}
	if res.TimedOut {
		return dto.ExecResp{}, errs.Wrap(errs.ErrAgentUnreach, "节点命令执行超时")
	}
	if res.ExitCode != 0 {
		return dto.ExecResp{}, errs.Wrap(errs.ErrAgentUnreach, fmt.Sprintf("节点命令执行失败（退出码 %d），请检查 Docker 服务及权限", res.ExitCode))
	}
	return *res, nil
}

// ---- 查询 ----

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
	if err := s.db.Model(&model.MiddlewareInstance{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return nil, err
	}
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
		OwnerID: middlewareOwnerID(ctx),
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
	inst, err := s.ByIDForCaller(ctx, id)
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
func (s *MiddlewareService) Delete(ctx context.Context, id uint) error {
	inst, err := s.ByIDForCaller(ctx, id)
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

// ByIDForCaller 将属主和实例自身节点作为权限边界，不能靠请求中的 node 替换归属。
func (s *MiddlewareService) ByIDForCaller(ctx context.Context, id uint) (*model.MiddlewareInstance, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	if !middlewareOwnerAllowed(ctx, inst.OwnerID) || inst.Origin == "container" && !middlewareNodeAllowed(ctx, inst.NodeID) {
		return nil, errs.Wrap(errs.ErrNotFound, "中间件实例不存在")
	}
	return inst, nil
}

// Test 测指定实例连通性。
func (s *MiddlewareService) Test(ctx context.Context, id uint) error {
	inst, err := s.ByIDForCaller(ctx, id)
	if err != nil {
		return err
	}
	pwd, err := s.credentialOf(inst)
	if err != nil {
		return err
	}
	d, ok := middlewareDescriptorOf(inst.Type)
	if !ok {
		return errs.Wrap(errs.ErrBadRequest, "未注册的中间件类型: "+inst.Type)
	}
	if inst.Origin != "container" || inst.ContainerName == "" {
		return d.TestConn(ctx, inst.Host, inst.Port, inst.User, pwd)
	}
	// 本机同网络允许容器名直连；远程实例不能误测本机同名容器。
	if normalizeNodeID(inst.NodeID) == "local" {
		if err := d.TestConn(ctx, inst.Host, inst.Port, inst.User, pwd); err == nil {
			return nil
		}
	}
	port, err := s.containerMappedPort(ctx, inst.NodeID, inst.ContainerName, inst.Port)
	if err != nil {
		return err
	}
	if port == 0 {
		return errs.Wrap(errs.ErrBadRequest, "该容器未发布服务端口，无法从面板测试连接")
	}
	node, err := s.nodes.ByID(inst.NodeID)
	if err != nil {
		return err
	}
	u, err := url.Parse(node.BaseURL)
	if err != nil {
		return err
	}
	return d.TestConn(ctx, u.Hostname(), port, inst.User, pwd)
}

// containerMappedPort 查容器服务端口在宿主的映射端口（无映射返回 0）。
func (s *MiddlewareService) containerMappedPort(ctx context.Context, nodeID, container string, port int) (int, error) {
	ac, err := s.clientFor(nodeID)
	if err != nil {
		return 0, err
	}
	res, err := execAgentCmd(ac, ctx, fmt.Sprintf("docker port %s %d/tcp", shellQuote(container), port), 15)
	if err != nil {
		return 0, err
	}
	// 形如 "11211/tcp -> 0.0.0.0:31211"
	for _, line := range strings.Split(res.Output, "\n") {
		if i := strings.LastIndex(line, ":"); i > 0 {
			var n int
			if _, err := fmt.Sscanf(strings.TrimSpace(line[i+1:]), "%d", &n); err == nil && n > 0 {
				return n, nil
			}
		}
	}
	return 0, nil
}

// Reveal 凭据查看（密码明文，权限闸在 API 层）。
func (s *MiddlewareService) Reveal(ctx context.Context, id uint) (map[string]any, error) {
	inst, err := s.ByIDForCaller(ctx, id)
	if err != nil {
		return nil, err
	}
	pwd, err := s.credentialOf(inst)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": inst.ID, "name": inst.Name, "user": inst.User, "password": pwd}, nil
}

// RefreshVersion 探测并回写实例版本（container origin 走容器内命令，external 取跳过）。
func (s *MiddlewareService) RefreshVersion(ctx context.Context, id uint, nodeId string) (string, error) {
	inst, err := s.ByIDForCaller(ctx, id)
	if err != nil {
		return "", err
	}
	d, ok := middlewareDescriptorOf(inst.Type)
	if !ok {
		return "", errs.Wrap(errs.ErrBadRequest, "未注册的中间件类型: "+inst.Type)
	}
	v := ""
	if inst.Origin == "container" && inst.ContainerName != "" {
		if nodeId != "" && normalizeNodeID(nodeId) != normalizeNodeID(inst.NodeID) {
			return "", errs.Wrap(errs.ErrBadRequest, "请求节点与实例归属节点不一致")
		}
		v, err = s.middlewareVersionProbe(ctx, inst.NodeID, inst.ContainerName, d.VersionProbe)
		if err != nil {
			return "", err
		}
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

// MiddlewareScanItem 扫描接口沿用清单元数据，明确 adoptable，不含凭据。
type MiddlewareScanItem struct {
	MiddlewareListItem
	Reused bool `json:"reused"`
}

func (s *MiddlewareService) Scan(ctx context.Context, nodeID string) ([]MiddlewareScanItem, error) {
	list, err := s.List(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if list.DiscoveryError != "" {
		return nil, errs.Wrap(errs.ErrAgentUnreach, list.DiscoveryError)
	}
	out := make([]MiddlewareScanItem, 0)
	for _, item := range list.Items {
		if item.Image != "" {
			out = append(out, MiddlewareScanItem{MiddlewareListItem: item, Reused: item.Source != "discovered"})
		}
	}
	return out, nil
}

// readCredentials 仅在显式纳管时提取指定的凭据键；不返回到扫描/列表响应。
func (s *MiddlewareService) readCredentials(ctx context.Context, nodeID, container string, d MiddlewareDescriptor) (string, string, error) {
	if d.CredentialEnv[0] == "" && d.CredentialEnv[1] == "" {
		return "", "", nil
	}
	ac, err := s.clientFor(nodeID)
	if err != nil {
		return "", "", err
	}
	res, err := execAgentCmd(ac, ctx, "docker inspect "+shellQuote(container)+" --format '{{json .Config.Env}}'", 20)
	if err != nil {
		return "", "", err
	}
	var env []string
	if err := json.Unmarshal([]byte(res.Output), &env); err != nil {
		return "", "", errs.Wrap(errs.ErrBadRequest, "读取容器凭据失败")
	}
	values := map[string]string{}
	for _, line := range env {
		if k, v, ok := strings.Cut(line, "="); ok {
			values[k] = v
		}
	}
	pick := func(index int) string {
		for _, key := range append([]string{d.CredentialEnv[index]}, d.CredentialAliases[index]...) {
			if value := values[key]; key != "" && value != "" {
				return value
			}
		}
		return ""
	}
	return pick(0), pick(1), nil
}

// Adopt 纳管扫描条目为登记实例（container origin，容器名直连）。
func (s *MiddlewareService) Adopt(ctx context.Context, nodeId, containerName, customName string) (*model.MiddlewareInstance, error) {
	nodeId = normalizeNodeID(nodeId)
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
	if hit.ManagedBy != "middleware" {
		return nil, errs.Wrap(errs.ErrBadRequest, "该类型由数据库管理页接入，请使用已有数据库纳管记录")
	}
	nodeId = hit.NodeID
	if !hit.Adoptable {
		return nil, errs.New(errs.CodeConflict, "error.instanceExists", "该容器已纳管")
	}
	name := customName
	if name == "" {
		name = sanitizeInstanceName(hit.Type + "-" + containerName)
		if nodeId != "local" {
			suffix := "-n" + nodeId
			if len(name) > 32-len(suffix) {
				name = name[:32-len(suffix)]
			}
			name = strings.TrimRight(name, "-") + suffix
		}
	}
	if !namePatternMW.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "实例名不合法（小写字母开头，小写字母/数字/中划线，3-32 位）: "+name)
	}
	var count int64
	if err := s.db.Model(&model.MiddlewareInstance{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.instanceExists", "实例名已存在")
	}
	d, _ := middlewareDescriptorOf(hit.Type)
	user, password, err := s.readCredentials(ctx, nodeId, containerName, d)
	if err != nil {
		return nil, err
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
		Name: name, Type: hit.Type, Origin: "container",
		Host: hit.ContainerName, Port: middlewarePortOf(hit.Type, 0),
		User: user, PasswordEnc: enc, Version: hit.Version,
		ContainerName: hit.ContainerName, ComposeProject: hit.ComposeProject,
		NodeID: nodeId, OwnerID: middlewareOwnerID(ctx),
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
			"nodeId":          normalizeNodeID(r.NodeID),
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
	out := make([]model.MiddlewareInstance, 0, len(rows))
	for _, r := range rows {
		if middlewareOwnerAllowed(ctx, r.OwnerID) && (r.Origin != "container" || middlewareNodeAllowed(ctx, r.NodeID)) {
			out = append(out, r)
		}
	}
	return out, nil
}

// credentialOf 解密实例凭据（空密码返回空串）。
func (s *MiddlewareService) credentialOf(inst *model.MiddlewareInstance) (string, error) {
	if inst.PasswordEnc == "" {
		return "", nil
	}
	return s.dbs.decryptPassword(inst.PasswordEnc)
}
