package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// VPNService EasyTier 组网（工具域）：一键安装编排 + 配置全面接管 + 运行状态。
//
// 范围与商店安装一致（local 节点单实例）。安装委托 StoreService（1p 源，任务化），
// 配置以 settings 持久化的期望态为唯一事实源：应用 = 渲染 TOML 写入
// /opt/ypanel/compose/<项目>/data/config.toml + 重启容器。
// 注意：容器名 = compose 项目名（1p 模板 CONTAINER_NAME=${CONTAINER_NAME}）；宿主机需有 /dev/net/tun。
type VPNService struct {
	db       *gorm.DB
	nodes    *NodeService
	settings *SettingService
	tasks    *TaskService
	store    *StoreService
}

const (
	vpnSettingKey   = "easytier.config"
	vpnAppKey       = "easytier"
	vpnComposeDir   = "/opt/ypanel/compose/"
	vpnTaskInitType = "vpn-easytier-init"
)

// NewVPNService 创建（store 用于安装委托与 agent 通道复用）。
func NewVPNService(db *gorm.DB, nodes *NodeService, settings *SettingService, tasks *TaskService, store *StoreService) *VPNService {
	return &VPNService{db: db, nodes: nodes, settings: settings, tasks: tasks, store: store}
}

// VPNProxyNetwork 子网映射：把本节点可达的物理网段接入虚拟网。
// CIDR 为物理段；Remap 非空时映射为另一虚拟段（避免远端本地段冲突）。
type VPNProxyNetwork struct {
	CIDR    string `json:"cidr"`
	Remap   string `json:"remap,omitempty"`
	Enabled bool   `json:"enabled"`
}

// VPNConfig 组网期望配置（settings JSON 持久化；渲染为目标 config.toml）。
type VPNConfig struct {
	RawMode        bool              `json:"rawMode"`
	Raw            string            `json:"raw,omitempty"` // 高级模式：整文件原文
	InstanceName   string            `json:"instanceName"`
	Hostname       string            `json:"hostname"`
	NetworkName    string            `json:"networkName"`
	NetworkSecret  string            `json:"networkSecret"`
	VirtualIP      string            `json:"virtualIp"`
	DHCP           bool              `json:"dhcp"`
	Peers          []string          `json:"peers"`
	Listeners      []string          `json:"listeners"`
	ProxyNetworks  []VPNProxyNetwork `json:"proxyNetworks"`
	ExitNodes      []string          `json:"exitNodes"`
	EnableExitNode bool              `json:"enableExitNode"`
	LatencyFirst   bool              `json:"latencyFirst"`
	DevName        string            `json:"devName"`
}

// VPNStatus 运行状态。
type VPNStatus struct {
	Installed   bool   `json:"installed"`
	Running     bool   `json:"running"`
	Project     string `json:"project"`
	Version     string `json:"version"`
	VirtualIP   string `json:"virtualIp"`
	PeerCount   int    `json:"peerCount"`
	NetworkName string `json:"networkName"`
	Configured  bool   `json:"configured"` // settings 已有期望配置
	HasTUN      bool   `json:"hasTun"`
}

// VPNTable easytier-cli 表格通用结构（列随版本漂移，泛化解析）。
type VPNTable struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

// VPNInstallInput 一键安装种子参数（缺省取默认配置）。
type VPNInstallInput struct {
	NetworkName   string `json:"networkName"`
	NetworkSecret string `json:"networkSecret"`
	VirtualIP     string `json:"virtualIp"`
	DHCP          *bool  `json:"dhcp"`
}

// ---------- 状态与查询 ----------

// Status 查询安装/运行状态（不产生副作用）。
func (s *VPNService) Status(ctx context.Context) (*VPNStatus, error) {
	st := &VPNStatus{Configured: s.settings.Get(vpnSettingKey, "") != ""}
	var inst model.AppStoreInstall
	if err := s.db.Where("`key` = ?", vpnAppKey).Order("id DESC").First(&inst).Error; err != nil {
		return st, nil // 未安装
	}
	st.Installed = true
	st.Project = inst.ComposeProject
	st.Version = inst.Version
	var cfg VPNConfig
	if st.Configured {
		_ = json.Unmarshal([]byte(s.settings.Get(vpnSettingKey, "")), &cfg)
		st.NetworkName = cfg.NetworkName
	}
	out, err := s.exec(ctx, 30, "docker inspect -f {{.State.Status}} "+inst.ComposeProject+" 2>/dev/null")
	if err != nil {
		return st, nil
	}
	if strings.TrimSpace(out.Output) != "running" {
		return st, nil
	}
	st.Running = true
	rows, err := s.cliRows(ctx, inst.ComposeProject, "peer")
	if err != nil || rows == nil {
		return st, nil
	}
	st.PeerCount = len(rows)
	for _, row := range rows {
		if row["cost"] == "Local" {
			st.VirtualIP = strings.SplitN(row["ipv4"], "/", 2)[0]
			st.HasTUN = st.VirtualIP != ""
			break
		}
	}
	return st, nil
}

// Peers 虚拟网内节点表。
func (s *VPNService) Peers(ctx context.Context) (*VPNTable, error) {
	proj, err := s.activeProject()
	if err != nil {
		return nil, err
	}
	return s.cliTable(ctx, proj, "peer")
}

// Routes 虚拟网路由表（含子网映射下发的段）。
func (s *VPNService) Routes(ctx context.Context) (*VPNTable, error) {
	proj, err := s.activeProject()
	if err != nil {
		return nil, err
	}
	return s.cliTable(ctx, proj, "route")
}

// ---------- 配置 ----------

// RenderedConfig 当前期望配置渲染出的 config.toml 全文（RawMode 时返回 Raw 原文）。
// 供前端"高级模式导入"使用；未持久化过配置时按默认值渲染。
func (s *VPNService) RenderedConfig() (string, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return "", err
	}
	if cfg.RawMode {
		return cfg.Raw, nil
	}
	return renderVPNTOML(cfg)
}

// GetConfig 读取期望配置（未设置时返回默认值）。
func (s *VPNService) GetConfig() (*VPNConfig, error) {
	raw := s.settings.Get(vpnSettingKey, "")
	if raw == "" {
		return vpnDefaultConfig(), nil
	}
	var cfg VPNConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "组网配置解析失败: "+err.Error())
	}
	return &cfg, nil
}

// SaveConfig 校验 → 持久化 → 应用（写 config.toml + 重启容器）。
func (s *VPNService) SaveConfig(ctx context.Context, cfg *VPNConfig) (*VPNStatus, error) {
	// 校验全部通过后才允许持久化：坏 TOML 一旦落库，「重置」（重读服务器状态）
	// 会把坏配置原样拉回表单，用户会被锁死在错误里（2026-10-08 真机踩过）。
	if err := s.validateConfig(cfg); err != nil {
		return nil, err
	}
	if !cfg.RawMode {
		if _, err := renderVPNTOML(cfg); err != nil {
			return nil, err
		}
	} else if err := validateVPNTOMLText(cfg.Raw); err != nil {
		return nil, err
	}
	if err := s.persistConfig(cfg); err != nil {
		return nil, err
	}
	if err := s.ApplyConfig(ctx, func(level, format string, args ...any) {}); err != nil {
		return nil, err
	}
	return s.Status(ctx)
}

// ApplyConfig 应用当前期望配置（安装钩子/手动应用共用）。
func (s *VPNService) ApplyConfig(ctx context.Context, logf TaskLogf) error {
	raw := s.settings.Get(vpnSettingKey, "")
	if raw == "" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "尚无组网期望配置")
	}
	var cfg VPNConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return errs.Wrap(errs.ErrBadRequest, "组网配置解析失败: "+err.Error())
	}
	text := cfg.Raw
	if !cfg.RawMode {
		var rerr error
		text, rerr = renderVPNTOML(&cfg)
		if rerr != nil {
			return rerr
		}
	}
	if err := validateVPNTOMLText(text); err != nil {
		return err
	}
	proj, err := s.activeProject()
	if err != nil {
		return err
	}
	ac, err := s.store.client()
	if err != nil {
		return err
	}
	logf("info", "写入配置 %s/data/config.toml", vpnComposeDir+proj)
	if _, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: vpnComposeDir + proj + "/data/config.toml", Content: text}); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "写入配置失败: "+err.Error())
	}
	logf("info", "重启容器 %s", proj)
	if _, err := s.exec(ctx, 180, "docker restart "+proj); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "重启容器失败: "+err.Error())
	}
	time.Sleep(3 * time.Second)
	out, err := s.exec(ctx, 30, "docker inspect -f {{.State.Status}} "+proj+" 2>/dev/null")
	if err == nil && strings.TrimSpace(out.Output) == "running" {
		if rows, terr := s.cliRows(ctx, proj, "peer"); terr == nil && rows != nil {
			for _, row := range rows {
				if row["cost"] == "Local" && row["ipv4"] != "" {
					logf("success", "生效确认：虚拟 IP %s（%d 个节点在线）", row["ipv4"], len(rows))
					return nil
				}
			}
		}
		logf("warn", "容器已运行，但尚未观察到虚拟网卡（首节点静态 IP 需配置 ipv4；对端未接入时 DHCP 不会分配）")
		return nil
	}
	logf("warn", "重启后容器未处于运行态，请查看日志页排查")
	return nil
}

// ---------- 一键安装 ----------

// Install 未安装时一键安装：落默认期望配置 → 商店安装（1p 源）→ 装完自动应用。
// 已安装时等价于应用配置。返回父任务 ID（前端轮询 Status 即可）。
func (s *VPNService) Install(ctx context.Context, in VPNInstallInput) (map[string]any, error) {
	var inst model.AppStoreInstall
	installed := s.db.Where("`key` = ?", vpnAppKey).Order("id DESC").First(&inst).Error == nil
	if !installed {
		cfg := vpnDefaultConfig()
		if in.NetworkName != "" {
			cfg.NetworkName = in.NetworkName
		}
		if in.NetworkSecret != "" {
			cfg.NetworkSecret = in.NetworkSecret
		}
		if in.VirtualIP != "" {
			cfg.VirtualIP = in.VirtualIP
		}
		if in.DHCP != nil {
			cfg.DHCP = *in.DHCP
		}
		if err := s.validateConfig(cfg); err != nil {
			return nil, err
		}
		if err := s.persistConfig(cfg); err != nil {
			return nil, err
		}
	}
	task, err := s.tasks.StartTask(vpnTaskInitType, "初始化 EasyTier 组网", "easytier", 35*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			return s.runInit(tctx, logf, installed)
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

func (s *VPNService) runInit(ctx context.Context, logf TaskLogf, alreadyInstalled bool) error {
	var inst model.AppStoreInstall
	installed := alreadyInstalled ||
		s.db.Where("`key` = ?", vpnAppKey).Order("id DESC").First(&inst).Error == nil
	if !installed {
		logf("info", "未检测到安装记录，开始从应用商店安装 EasyTier…")
		var app model.AppStoreApp
		if err := s.db.Joins("JOIN app_store_sources src ON src.id = app_store_apps.source_id").
			Where("app_store_apps.key = ? AND src.enabled = ?", vpnAppKey, true).
			Order("app_store_apps.id ASC").First(&app).Error; err != nil {
			return errs.New(errs.CodeNotFound, "error.appNotFound", "应用商店中未找到 EasyTier（请先在商店同步源）")
		}
		out, err := s.store.Install(ctx, StoreInstallInput{SourceID: app.SourceID, Key: app.Key, Name: "easytier"})
		if err != nil {
			return err
		}
		taskID := out["taskId"]
		logf("info", "商店安装任务 #%v 已启动，等待完成…", taskID)
		if err := s.waitTask(ctx, out["taskId"], 30*time.Minute); err != nil {
			return err
		}
		logf("info", "商店安装完成")
	}
	logf("info", "应用组网配置…")
	return s.ApplyConfig(ctx, logf)
}

// ---------- 内部 ----------

// activeProject 取已安装实例的 compose 项目名（容器同名）。
func (s *VPNService) activeProject() (string, error) {
	var inst model.AppStoreInstall
	if err := s.db.Where("`key` = ?", vpnAppKey).Order("id DESC").First(&inst).Error; err != nil {
		return "", errs.New(errs.CodeNotFound, "error.appNotFound", "EasyTier 未安装")
	}
	return inst.ComposeProject, nil
}

// exec 经 agent 受控执行命令。
func (s *VPNService) exec(ctx context.Context, timeoutSecs int, cmd string) (dto.ExecResp, error) {
	ac, err := s.store.client()
	if err != nil {
		return dto.ExecResp{}, err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeoutSecs})
	if err != nil {
		return dto.ExecResp{}, err
	}
	return *out, nil
}

// cliTable 执行 easytier-cli 子命令并解析为通用表（JSON 优先，markdown 表格兜底）。
func (s *VPNService) cliTable(ctx context.Context, proj, sub string) (*VPNTable, error) {
	rows, err := s.cliRows(ctx, proj, sub)
	if err != nil {
		return nil, err
	}
	return vpnRowsToTable(sub, rows), nil
}

// cliRows 返回键值行（列对齐稳定）。
func (s *VPNService) cliRows(ctx context.Context, proj, sub string) ([]map[string]string, error) {
	out, err := s.exec(ctx, 30, "docker exec "+proj+" easytier-cli -o json "+sub+" 2>/dev/null")
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(out.Output)
	if strings.HasPrefix(text, "[") {
		var rows []map[string]any
		if json.Unmarshal([]byte(text), &rows) == nil {
			return vpnJSONRows(rows), nil
		}
	}
	t := parseVPNMarkdownTable(out.Output)
	res := make([]map[string]string, 0, len(t.Rows))
	for _, row := range t.Rows {
		m := map[string]string{}
		for i, c := range t.Columns {
			if i < len(row) {
				m[c] = row[i]
			}
		}
		res = append(res, m)
	}
	return res, nil
}

// 列顺序以 easytier v2.6 实测 JSON 键为准，缺省键追加在后。
var (
	vpnPeerCols  = []string{"cidr", "ipv4", "hostname", "cost", "lat_ms", "loss_rate", "rx_bytes", "tx_bytes", "tunnel_proto", "nat_type", "id", "version"}
	vpnRouteCols = []string{"ipv4", "hostname", "proxy_cidrs", "next_hop_ipv4", "next_hop_hostname", "next_hop_lat", "path_len", "path_latency", "version"}
)

func vpnJSONRows(rows []map[string]any) []map[string]string {
	res := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		m := make(map[string]string, len(r))
		for k, v := range r {
			m[k] = vpnCell(v)
		}
		res = append(res, m)
	}
	return res
}

func vpnRowsToTable(sub string, rows []map[string]string) *VPNTable {
	t := &VPNTable{Columns: []string{}, Rows: make([][]string, 0, len(rows))}
	pref := vpnRouteCols
	if sub == "peer" {
		pref = vpnPeerCols
	}
	present := map[string]bool{}
	for _, r := range rows {
		for k := range r {
			present[k] = true
		}
	}
	for _, c := range pref {
		if present[c] {
			t.Columns = append(t.Columns, c)
		}
	}
	extra := map[string]bool{}
	for k := range present {
		if !slices.Contains(pref, k) {
			extra[k] = true
		}
	}
	for _, r := range rows {
		for k := range r {
			if extra[k] && !slices.Contains(t.Columns, k) {
				t.Columns = append(t.Columns, k)
			}
		}
	}
	for _, r := range rows {
		row := make([]string, 0, len(t.Columns))
		for _, c := range t.Columns {
			row = append(row, r[c])
		}
		t.Rows = append(t.Rows, row)
	}
	return t
}

func vpnCell(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprint(x)
	}
}

// waitTask 轮询子任务直到成功/失败。
func (s *VPNService) waitTask(ctx context.Context, taskID any, timeout time.Duration) error {
	id, ok := taskID.(uint)
	if !ok {
		if f, ferr := toFloat(taskID); ferr == nil {
			id = uint(f)
		} else {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "安装任务 ID 缺失")
		}
	}
	deadline := time.After(timeout)
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return errs.New(errs.CodeBadRequest, "error.badRequest", "等待安装任务超时")
		case <-tick.C:
		}
		row, err := s.tasks.Get(id)
		if err != nil {
			return err
		}
		switch row.Status {
		case TaskSuccess:
			return nil
		case TaskFailed:
			return errs.New(errs.CodeBadRequest, "error.badRequest", "商店安装任务失败: "+truncStr(row.Error, 400))
		}
	}
}

// validateConfig 结构化配置校验（值域白名单先行，渲染无需转义）。
func (s *VPNService) validateConfig(cfg *VPNConfig) error {
	safe := vpnTextSafe
	for k, v := range map[string]string{
		"instanceName": cfg.InstanceName, "hostname": cfg.Hostname,
		"networkName": cfg.NetworkName, "networkSecret": cfg.NetworkSecret, "devName": cfg.DevName,
	} {
		if v != "" && !safe.MatchString(v) {
			return errs.New(errs.CodeBadRequest, "error.badRequest", k+" 含非法字符（不允许引号/反斜杠/换行）")
		}
	}
	for _, u := range append(append([]string{}, cfg.Peers...), cfg.Listeners...) {
		parsed, err := url.Parse(u)
		if err != nil || parsed.Host == "" || !vpnURLSchemes[strings.ToLower(parsed.Scheme)] {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "地址不合法: "+u+"（协议 tcp/udp/wg/ws/wss/quic）")
		}
	}
	if !cfg.DHCP {
		if _, _, err := net.ParseCIDR(ensureVPNCIDR(cfg.VirtualIP)); cfg.VirtualIP != "" && err != nil {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "虚拟 IP 不合法: "+cfg.VirtualIP)
		}
		if cfg.VirtualIP == "" {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "需设置本机虚拟 IP，或开启 DHCP（DHCP 需组网内已有节点）")
		}
	}
	for _, ip := range cfg.ExitNodes {
		if net.ParseIP(ip) == nil {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "出口节点 IP 不合法: "+ip)
		}
	}
	for _, pn := range cfg.ProxyNetworks {
		if _, _, err := net.ParseCIDR(strings.TrimSpace(pn.CIDR)); err != nil {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "子网映射 CIDR 不合法: "+pn.CIDR)
		}
		if pn.Remap != "" {
			if _, _, err := net.ParseCIDR(strings.TrimSpace(pn.Remap)); err != nil {
				return errs.New(errs.CodeBadRequest, "error.badRequest", "映射虚拟 CIDR 不合法: "+pn.Remap)
			}
		}
	}
	return nil
}

func (s *VPNService) persistConfig(cfg *VPNConfig) error {
	buf, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.settings.Set(vpnSettingKey, string(buf))
}

// vpnDefaultConfig 默认期望配置（与 1p 模板端口对齐，静态 IP 走 easytier 常用段）。
func vpnDefaultConfig() *VPNConfig {
	return &VPNConfig{
		InstanceName: "easytier",
		Hostname:     "easytier",
		NetworkName:  "ypanel-net",
		VirtualIP:    "10.144.144.1",
		Listeners:    []string{"tcp://0.0.0.0:11010", "udp://0.0.0.0:11010"},
	}
}

var (
	vpnTextSafe   = regexp.MustCompile(`^[^"\\\r\n]{1,128}$`)
	vpnURLSchemes = map[string]bool{"tcp": true, "udp": true, "wg": true, "ws": true, "wss": true, "quic": true}
)

// renderVPNTOML 渲染 config.toml（字段值已经白名单校验，无需 TOML 转义）。
// 字段名以 EasyTier v2.6.x Config 序列化为准：proxy_network（数组表）、network_identity、flags。
// 注意 rpc_portal 不是 v2 配置字段（写入会被忽略），不渲染。
func renderVPNTOML(cfg *VPNConfig) (string, error) {
	var b strings.Builder
	put := func(k, v string) { b.WriteString(k + " = \"" + v + "\"\n") }
	if cfg.InstanceName != "" {
		put("instance_name", cfg.InstanceName)
	}
	if cfg.Hostname != "" {
		put("hostname", cfg.Hostname)
	}
	if cfg.DHCP {
		b.WriteString("dhcp = true\n")
	} else if cfg.VirtualIP != "" {
		b.WriteString("dhcp = false\n")
		put("ipv4", ensureVPNCIDR(cfg.VirtualIP))
	}
	listeners := cfg.Listeners
	if len(listeners) == 0 {
		listeners = vpnDefaultConfig().Listeners
	}
	b.WriteString("listeners = [" + vpnQuotedList(listeners) + "]\n")
	if len(cfg.ExitNodes) > 0 {
		b.WriteString("exit_nodes = [" + vpnQuotedList(cfg.ExitNodes) + "]\n")
	}
	for _, p := range cfg.Peers {
		b.WriteString("\n[[peer]]\n")
		put("uri", p)
	}
	for _, pn := range cfg.ProxyNetworks {
		if !pn.Enabled {
			continue
		}
		b.WriteString("\n[[proxy_network]]\n")
		put("cidr", strings.TrimSpace(pn.CIDR))
		if pn.Remap != "" {
			put("mapped_cidr", strings.TrimSpace(pn.Remap))
		}
	}
	b.WriteString("\n[network_identity]\n")
	put("network_name", cfg.NetworkName)
	put("network_secret", cfg.NetworkSecret)
	flags := []string{}
	if cfg.LatencyFirst {
		flags = append(flags, "latency_first = true")
	}
	if cfg.EnableExitNode {
		flags = append(flags, "enable_exit_node = true")
	}
	if cfg.DevName != "" {
		flags = append(flags, "dev_name = \""+cfg.DevName+"\"")
	}
	if len(flags) > 0 {
		b.WriteString("\n[flags]\n" + strings.Join(flags, "\n") + "\n")
	}
	return b.String(), nil
}

// validateVPNTOMLText TOML 语法校验（结构化与原文模式共用；原文模式仅校验语法）。
func validateVPNTOMLText(text string) error {
	var m map[string]any
	if err := toml.Unmarshal([]byte(text), &m); err != nil {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "配置不是合法 TOML: "+err.Error())
	}
	return nil
}

func vpnQuotedList(list []string) string {
	quoted := make([]string, 0, len(list))
	for _, v := range list {
		quoted = append(quoted, "\""+strings.TrimSpace(v)+"\"")
	}
	return strings.Join(quoted, ", ")
}

// ensureVPNCIDR 裸 IP 补 /24（EasyTier 对纯 ipv4 默认按 /24 补全）。
func ensureVPNCIDR(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.Contains(s, "/") {
		return s
	}
	return s + "/24"
}

// parseVPNMarkdownTable 解析 easytier-cli 输出的 markdown 表格（| 分隔，第二行为分隔线）。
func parseVPNMarkdownTable(out string) *VPNTable {
	t := &VPNTable{Columns: []string{}, Rows: [][]string{}}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		sep := true
		for _, c := range cells {
			if c != "" && strings.Trim(c, "-: ") != "" {
				sep = false
				break
			}
		}
		if sep || len(cells) == 0 {
			continue
		}
		if len(t.Columns) == 0 {
			t.Columns = cells
			continue
		}
		t.Rows = append(t.Rows, cells)
	}
	return t
}

// toFloat JSON 数字（float64）宽容转换。
func toFloat(v any) (float64, error) {
	if f, ok := v.(float64); ok {
		return f, nil
	}
	if i, ok := v.(int); ok {
		return float64(i), nil
	}
	if u, ok := v.(uint); ok {
		return float64(u), nil
	}
	return 0, fmt.Errorf("not a number: %T", v)
}
