// DnsService 内网 DNS（dnsmasq 容器自管部署，M27）。
// 部署形态：/opt/ypanel/dns/ 下 docker-compose.yml + dnsmasq.conf，network_mode:host，
// 仅绑定内网 IP（listen-address + bind-interfaces）——用户公网侧为全端口转发，
// 绑 0.0.0.0 会把 53 暴露成开放解析器（DNS 放大攻击源），属安全硬约束。
// 下发：conf 原子替换（b64 通道防注入，写前备份）+ docker kill -s HUP 平滑重载。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

const (
	dnsSettingKey  = "dns.settings"
	dnsDir         = "/opt/ypanel/dns"
	dnsConfPath    = "/opt/ypanel/dns/dnsmasq.conf"
	dnsComposePath = "/opt/ypanel/dns/docker-compose.yml"
	dnsContainer   = "ypanel-dnsmasq"
	dnsProject     = "ypanel-dns"
)

// dnsPort 服务的监听端口（固定 53）。
const dnsPort = 53

// dnsHostnamePattern 主机名/域名（RFC-1123 标签，允许下划线兼容 _dmarc 等记录，拒绝通配符）。
var dnsHostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9_](?:[a-zA-Z0-9_-]{0,61}[a-zA-Z0-9_])?(?:\.[a-zA-Z0-9_](?:[a-zA-Z0-9_-]{0,61}[a-zA-Z0-9_])?)*$`)

// dnsImagePattern 镜像名白名单（registry/repo:tag）。
var dnsImagePattern = regexp.MustCompile(`^[a-zA-Z0-9._/-]+(?::[a-zA-Z0-9._-]+)?$`)

// DnsConfig 运行设置（SettingService 单键 JSON；v1 单部署目标）。
type DnsConfig struct {
	DeployNodeID string   `json:"deployNodeId"`
	ListenIP     string   `json:"listenIp"`
	Upstreams    []string `json:"upstreams"`
	CacheSize    int      `json:"cacheSize"`
	Image        string   `json:"image"`
	SiteAlign    bool     `json:"siteAlign"`   // 与站点域名对齐（影子记录，随站点变化自动更新）
	SiteAlignIP  string   `json:"siteAlignIp"` // 派生记录指向 IP；空 = 使用监听 IP
}

// withDefaults 补默认值（渲染与展示统一走它）。
func (c DnsConfig) withDefaults() DnsConfig {
	if len(c.Upstreams) == 0 {
		c.Upstreams = []string{"223.5.5.5", "119.29.29.29"}
	}
	if c.CacheSize == 0 {
		c.CacheSize = 1000
	}
	if c.Image == "" {
		c.Image = "dockurr/dnsmasq:latest"
	}
	return c
}

// DnsService 内网 DNS 服务。
type DnsService struct {
	db       *gorm.DB
	nodes    *NodeService
	settings *SettingService
	mu       sync.Mutex // 串行化 apply/deploy，避免并发写 conf
}

// NewDnsService 创建。
func NewDnsService(db *gorm.DB, nodes *NodeService, settings *SettingService) *DnsService {
	return &DnsService{db: db, nodes: nodes, settings: settings}
}

// ---------- 设置 ----------

// GetConfig 读取设置（缺省补默认）。
func (s *DnsService) GetConfig() DnsConfig {
	var cfg DnsConfig
	if raw := s.settings.Get(dnsSettingKey, ""); raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			cfg = DnsConfig{}
		}
	}
	return cfg.withDefaults()
}

func (s *DnsService) saveConfigLocked(cfg DnsConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.settings.Set(dnsSettingKey, string(raw))
}

// dnsValidateIPPrivate 监听地址必须为私网/回环（安全硬约束：拒绝 0.0.0.0 与公网地址）。
func dnsValidateIPPrivate(v string) error {
	ip := net.ParseIP(strings.TrimSpace(v))
	if ip == nil {
		return svcBadReq("监听 IP 不合法")
	}
	if ip.IsUnspecified() {
		return svcBadReq("禁止绑定 0.0.0.0/::（公网侧全端口转发下会暴露成开放解析器）")
	}
	if !ip.IsPrivate() && !ip.IsLoopback() {
		return svcBadReq("监听 IP 必须为私网地址（如 192.168.x.2），不接受公网地址")
	}
	return nil
}

// SaveConfig 校验并保存设置；已部署时自动应用（失败回滚设置）。
func (s *DnsService) SaveConfig(ctx context.Context, cfg DnsConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg.ListenIP = strings.TrimSpace(cfg.ListenIP)
	if cfg.ListenIP != "" {
		if err := dnsValidateIPPrivate(cfg.ListenIP); err != nil {
			return err
		}
	}
	if len(cfg.Upstreams) > 8 {
		return svcBadReq("上游 DNS 最多 8 条")
	}
	cleanUp := make([]string, 0, len(cfg.Upstreams))
	for _, u := range cfg.Upstreams {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if len(u) > 253 || !(net.ParseIP(u) != nil || dnsHostnamePattern.MatchString(u)) {
			return svcBadReq("上游 DNS 不合法: " + u)
		}
		cleanUp = append(cleanUp, u)
	}
	cfg.Upstreams = cleanUp
	if cfg.CacheSize < 0 || cfg.CacheSize > 10000 {
		return svcBadReq("缓存条目数须在 0-10000 内")
	}
	cfg.Image = strings.TrimSpace(cfg.Image)
	if cfg.Image != "" && !dnsImagePattern.MatchString(cfg.Image) {
		return svcBadReq("镜像名不合法")
	}
	cfg.SiteAlignIP = strings.TrimSpace(cfg.SiteAlignIP)
	if cfg.SiteAlignIP != "" {
		if err := dnsValidateIPPrivate(cfg.SiteAlignIP); err != nil {
			return err
		}
	}
	before := s.GetConfig()
	if err := s.saveConfigLocked(cfg); err != nil {
		return err
	}
	if before.DeployNodeID != "" {
		if err := s.applyLocked(ctx, before.DeployNodeID); err != nil {
			_ = s.saveConfigLocked(before)
			return svcBadReq("设置已回滚（应用失败）: " + err.Error())
		}
	}
	return nil
}

// ---------- 校验 ----------

// dnsValidateRecord 记录语义校验。
func dnsValidateRecord(r *model.DnsRecord) error {
	r.Domain = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(r.Domain), ".")))
	r.Target = strings.TrimSpace(r.Target)
	r.Comment = strings.TrimSpace(r.Comment)
	if len(r.Domain) == 0 || len(r.Domain) > 253 || !dnsHostnamePattern.MatchString(r.Domain) {
		return svcBadReq("域名不合法（RFC-1123 主机名，≤253 字符；dnsmasq 域后缀天然匹配子域，无需通配符）")
	}
	switch r.Type {
	case "address":
		ip := net.ParseIP(r.Target)
		if ip == nil {
			return svcBadReq("指向须为合法 IP 地址（address 记录）")
		}
		r.Target = ip.String()
	case "cname":
		if len(r.Target) > 253 || !dnsHostnamePattern.MatchString(r.Target) {
			return svcBadReq("指向须为合法主机名（cname 记录）")
		}
	case "txt":
		if len(r.Target) == 0 || len(r.Target) > 255 {
			return svcBadReq("文本长度须在 1-255 字符内（txt 记录）")
		}
		for _, c := range r.Target {
			if c < 0x20 || c == 0x7f {
				return svcBadReq("文本包含不可打印字符（txt 记录）")
			}
		}
	default:
		return svcBadReq("类型仅支持 address / cname / txt")
	}
	if len(r.Comment) > 128 {
		return svcBadReq("备注不超过 128 字符")
	}
	return nil
}

// dnsDupCheck 启用记录中 (type, domain) 重复拦截（同域名不同类型合法）。
func (s *DnsService) dnsDupCheck(r *model.DnsRecord) error {
	if !r.Enabled {
		return nil
	}
	var cnt int64
	if err := s.db.Model(&model.DnsRecord{}).Where("enabled = ? AND type = ? AND domain = ? AND id <> ?", true, r.Type, r.Domain, r.ID).
		Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return svcBadReq(fmt.Sprintf("已存在同类型同域名的启用记录：%s %s（dnsmasq 仅取其一，避免歧义）", r.Type, r.Domain))
	}
	return nil
}

// ---------- 站点对齐（影子记录） ----------

// DnsSiteAlignEntry 站点对齐派生条目。
type DnsSiteAlignEntry struct {
	Domain   string `json:"domain"`
	SiteName string `json:"siteName"`
}

// collectSiteDomains 从站点收集对齐域名（纯函数）：主域名 + 附加域名 JSON，
// 小写、去 `*.` 前缀（dnsmasq 域后缀天然匹配子域）、跨站点去重（先到先得）。
func collectSiteDomains(sites []model.Site) []DnsSiteAlignEntry {
	seen := map[string]bool{}
	entries := []DnsSiteAlignEntry{}
	for _, st := range sites {
		domains := []string{st.Domain}
		if strings.TrimSpace(st.Domains) != "" {
			var extra []string
			if err := json.Unmarshal([]byte(st.Domains), &extra); err == nil {
				domains = append(domains, extra...)
			}
		}
		for _, d := range domains {
			d = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(d), ".")))
			d = strings.TrimPrefix(d, "*.")
			if d == "" || seen[d] || !dnsHostnamePattern.MatchString(d) {
				continue
			}
			seen[d] = true
			entries = append(entries, DnsSiteAlignEntry{Domain: d, SiteName: st.Name})
		}
	}
	return entries
}

// filterSiteAlign 手动优先：跳过与手动启用 address 记录同域的派生项（纯函数）。
func filterSiteAlign(entries []DnsSiteAlignEntry, records []model.DnsRecord) []DnsSiteAlignEntry {
	manual := map[string]bool{}
	for _, r := range records {
		if r.Enabled && r.Type == "address" {
			manual[strings.ToLower(r.Domain)] = true
		}
	}
	out := []DnsSiteAlignEntry{}
	for _, e := range entries {
		if !manual[e.Domain] {
			out = append(out, e)
		}
	}
	return out
}

// siteAlignTarget 对齐指向 IP：空 = 监听 IP。
func siteAlignTarget(cfg DnsConfig) string {
	if strings.TrimSpace(cfg.SiteAlignIP) != "" {
		return strings.TrimSpace(cfg.SiteAlignIP)
	}
	return cfg.ListenIP
}

// enabledSites 对齐源：启用中的站点。
func (s *DnsService) enabledSites() ([]model.Site, error) {
	rows := []model.Site{}
	if err := s.db.Where("enabled = ?", true).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// OnSitesChanged 站点变更后的对齐重载（best-effort：未开启对齐/未部署时静默跳过）。
func (s *DnsService) OnSitesChanged(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.GetConfig()
	if !cfg.SiteAlign || cfg.DeployNodeID == "" {
		return nil
	}
	return s.applyLocked(ctx, cfg.DeployNodeID)
}

// ---------- 渲染（纯函数） ----------

// renderDnsmasqConf 渲染 dnsmasq.conf（确定性输出：无时间戳，便于测试与漂移比对）。
func renderDnsmasqConf(cfg DnsConfig, records []model.DnsRecord, sites []model.Site) string {
	cfg = cfg.withDefaults()
	var b strings.Builder
	b.WriteString("# 本文件由 YPanel 内网 DNS 自动生成（手改内容会在下次应用时被覆盖）\n")
	b.WriteString("# 安全约束：仅绑定内网地址；公网侧如有全端口转发，请勿放行 53 端口入站\n")
	b.WriteString("no-resolv\n")
	for _, u := range cfg.Upstreams {
		b.WriteString("server=" + u + "\n")
	}
	if cfg.ListenIP != "" {
		b.WriteString("listen-address=" + cfg.ListenIP + "\n")
		b.WriteString("bind-interfaces\n")
	}
	b.WriteString(fmt.Sprintf("cache-size=%d\n", cfg.CacheSize))
	b.WriteString("local-ttl=300\n")
	b.WriteString("domain-needed\n")
	b.WriteString("bogus-priv\n")
	enabled := []model.DnsRecord{}
	for _, r := range records {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	sortRecords(enabled)
	if len(enabled) > 0 {
		b.WriteString("\n# ---- 解析记录 ----\n")
	}
	for _, r := range enabled {
		switch r.Type {
		case "address":
			fmt.Fprintf(&b, "address=/%s/%s\n", r.Domain, r.Target)
		case "cname":
			fmt.Fprintf(&b, "cname=%s,%s\n", r.Domain, r.Target)
		case "txt":
			fmt.Fprintf(&b, "txt-record=%s,%s\n", r.Domain, r.Target)
		}
	}
	// 站点对齐派生记录（影子：随站点变化自动更新；与手动记录同域时手动优先）
	if cfg.SiteAlign {
		target := siteAlignTarget(cfg)
		derived := filterSiteAlign(collectSiteDomains(sites), records)
		if target != "" && len(derived) > 0 {
			b.WriteString("\n# ---- 站点对齐（自动派生，随站点变化更新） ----\n")
			for _, e := range derived {
				fmt.Fprintf(&b, "address=/%s/%s\n", e.Domain, target)
			}
		}
	}
	return b.String()
}

// sortRecords 按 sort,id 升序（就地排序）。
func sortRecords(rows []model.DnsRecord) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0; j-- {
			a, b := rows[j-1], rows[j]
			if a.Sort < b.Sort || (a.Sort == b.Sort && a.ID <= b.ID) {
				break
			}
			rows[j-1], rows[j] = b, a
		}
	}
}

// renderDnsCompose 渲染 docker-compose.yml（host 网络；conf 只读挂载）。
func renderDnsCompose(image string) string {
	return fmt.Sprintf(`services:
  dnsmasq:
    image: %s
    container_name: %s
    restart: unless-stopped
    network_mode: host
    volumes:
      - ./dnsmasq.conf:/etc/dnsmasq.conf:ro
`, image, dnsContainer)
}

// ---------- 探测 ----------

// DnsPort53Occupy 53 端口占用明细（含本机绑定地址，用于按绑定目标判定冲突）。
type DnsPort53Occupy struct {
	Port    int    `json:"port"`
	Proto   string `json:"proto"`
	Addr    string `json:"addr"` // 本地绑定地址（0.0.0.0 / :: / 127.0.0.53 等）
	Process string `json:"process"`
}

// DnsOverview 部署状态投影。
type DnsOverview struct {
	Deployed         bool                `json:"deployed"`  // conf 存在且容器存在
	Container        string              `json:"container"` // running / exited / ... / missing
	Image            string              `json:"image"`
	DirReady         bool                `json:"dirReady"` // conf 文件存在
	Port53           []DnsPort53Occupy   `json:"port53"`   // 当前 53 端口监听明细
	Port53Conflict   bool                `json:"port53Conflict"`
	Port53Note       string              `json:"port53Note"`
	ListenIP         string              `json:"listenIp"`
	Upstreams        []string            `json:"upstreams"`
	RecordCount      int64               `json:"recordCount"`
	SiteAlign        bool                `json:"siteAlign"`
	SiteAlignIP      string              `json:"siteAlignIp"` // 生效指向（未单设时=监听 IP）
	SiteAlignCount   int                 `json:"siteAlignCount"`
	SiteAlignSites   int                 `json:"siteAlignSites"`
	SiteAlignPreview []DnsSiteAlignEntry `json:"siteAlignPreview"`
	DeployNodeID     string              `json:"deployNodeId"`
	NodeMatch        bool                `json:"nodeMatch"` // 查询节点是否为部署节点
}

// Overview 查询部署状态（单次 exec 批量探测）。
func (s *DnsService) Overview(ctx context.Context, nodeId string) (*DnsOverview, error) {
	records := []model.DnsRecord{}
	if err := s.db.Order("sort ASC, id ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	cfg := s.GetConfig()
	script := "echo ===C===; docker ps -a --filter name=^" + dnsContainer + "$ --format '{{.State}} {{.Image}}' 2>/dev/null || true\n" +
		"echo ===D===; [ -f " + dnsConfPath + " ] && echo yes || echo no\n" +
		"echo ===TCP===; ss -H -lntp 2>/dev/null || true\n" +
		"echo ===UDP===; ss -H -lnup 2>/dev/null || true\n"
	out, code, err := execOnNode(ctx, s.nodes, nodeId, script, 30)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "状态探测失败: "+strings.TrimSpace(out))
	}
	ov := parseDnsOverview(out)
	ov.ListenIP = cfg.ListenIP
	ov.Upstreams = cfg.withDefaults().Upstreams
	ov.RecordCount = int64(len(records))
	ov.DeployNodeID = cfg.DeployNodeID
	ov.NodeMatch = cfg.DeployNodeID == nodeId
	ov.Deployed = ov.DirReady && ov.Container != "" && ov.Container != "missing"
	conflicts, others := dns53Classify(ov.Port53, cfg.ListenIP)
	ov.Port53Conflict = len(conflicts) > 0
	if len(others) > 0 {
		ov.Port53Note = "本机存在其他特定地址的 53 监听（如 systemd-resolved stub 127.0.0.53），与内网 IP 精确绑定不冲突"
	}
	ov.SiteAlign = cfg.SiteAlign
	ov.SiteAlignIP = siteAlignTarget(cfg)
	if cfg.SiteAlign {
		sites, err := s.enabledSites()
		if err != nil {
			return nil, err
		}
		derived := filterSiteAlign(collectSiteDomains(sites), records)
		ov.SiteAlignCount = len(derived)
		names := map[string]bool{}
		for _, e := range derived {
			names[e.SiteName] = true
		}
		ov.SiteAlignSites = len(names)
		if len(derived) > 200 {
			derived = derived[:200]
		}
		ov.SiteAlignPreview = derived
	} else {
		ov.SiteAlignPreview = []DnsSiteAlignEntry{}
	}
	return ov, nil
}

// parseDnsOverview 解析探测输出（纯函数）。
func parseDnsOverview(out string) *DnsOverview {
	ov := &DnsOverview{Container: "missing", Port53: []DnsPort53Occupy{}, Upstreams: []string{}}
	ov.Port53 = parseDnsPort53(out)
	section := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "===C===", "===D===", "===TCP===", "===UDP===":
			section = line
			continue
		}
		if line == "" {
			continue
		}
		switch section {
		case "===C===":
			f := strings.Fields(line)
			if len(f) >= 2 {
				ov.Container = f[0]
				ov.Image = f[1]
			}
		case "===D===":
			ov.DirReady = line == "yes"
		}
	}
	return ov
}

// parseDnsPort53 解析 ss 输出为 53 端口监听明细（含本地地址，纯函数）。
func parseDnsPort53(out string) []DnsPort53Occupy {
	res := []DnsPort53Occupy{}
	proto := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "===TCP===":
			proto = "tcp"
			continue
		case "===UDP===":
			proto = "udp"
			continue
		}
		if proto == "" || line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		// Local Address:Port 为第 4 列（形如 192.168.100.2:53 / [::]:53 / *:53）
		idx := strings.LastIndex(f[3], ":")
		if idx < 0 {
			continue
		}
		port, err := strconv.Atoi(f[3][idx+1:])
		if err != nil || port != dnsPort {
			continue
		}
		addr := strings.Trim(f[3][:idx], "[]")
		proc := ""
		if rest := strings.Join(f[5:], " "); strings.Contains(rest, "users:((") {
			proc = rest
		}
		res = append(res, DnsPort53Occupy{Port: port, Proto: proto, Addr: addr, Process: proc})
	}
	return res
}

// dns53IsConflict 判定既有 53 监听是否与目标绑定地址冲突（纯函数）。
// Linux 同端口仅「通配绑定」与「同地址精确绑定」互斥；systemd-resolved stub 等
// 特定地址监听（127.0.0.53）与内网 IP 的 listen-address 精确绑定可共存。
func dns53IsConflict(addr, listenIP string) bool {
	switch addr {
	case "0.0.0.0", "*", "::":
		return true
	}
	a, b := net.ParseIP(addr), net.ParseIP(listenIP)
	if a != nil && b != nil {
		return a.Equal(b)
	}
	return strings.EqualFold(addr, listenIP)
}

// dns53Classify 按「本服务进程放行 → 冲突/共存」分拣 53 监听（纯函数）。
func dns53Classify(entries []DnsPort53Occupy, listenIP string) (conflicts, others []DnsPort53Occupy) {
	conflicts, others = []DnsPort53Occupy{}, []DnsPort53Occupy{}
	for _, o := range entries {
		// 本服务自身（重新部署场景）不算冲突
		if strings.Contains(o.Process, "dnsmasq") {
			continue
		}
		if dns53IsConflict(o.Addr, listenIP) {
			conflicts = append(conflicts, o)
		} else {
			others = append(others, o)
		}
	}
	return conflicts, others
}

// dnsProcNamePattern 从 ss 的 users:((("name",pid=…)) 片段提取进程名。
var dnsProcNamePattern = regexp.MustCompile(`users:\(\("([^"]+)"`)

// procShortName 提取监听进程短名（无进程信息时返回空串）。
func procShortName(process string) string {
	if m := dnsProcNamePattern.FindStringSubmatch(process); len(m) > 1 {
		return m[1]
	}
	return ""
}

// dnsCheckPort53 部署前置检测：仅拦截与目标监听地址冲突的 53 占用，
// 其余特定地址监听（如 resolved stub）作为提示返回，不阻断部署。
func (s *DnsService) dnsCheckPort53(ctx context.Context, nodeId, listenIP string) (conflicts, others []DnsPort53Occupy, err error) {
	script := "echo ===TCP===; ss -H -lntp 2>/dev/null || true\n echo ===UDP===; ss -H -lnup 2>/dev/null || true"
	out, _, err := execOnNode(ctx, s.nodes, nodeId, script, 30)
	if err != nil {
		return nil, nil, err
	}
	conflicts, others = dns53Classify(parseDnsPort53(out), listenIP)
	return conflicts, others, nil
}

// ---------- 部署 / 应用 ----------

// Deploy 部署 dnsmasq 容器到目标节点（预检 → 写配置 → compose up）。
func (s *DnsService) Deploy(ctx context.Context, nodeId string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.GetConfig()
	if cfg.ListenIP == "" {
		return nil, svcBadReq("请先在设置中填写监听 IP（节点内网地址）")
	}
	// docker 存在性
	out, code, err := execOnNode(ctx, s.nodes, nodeId, "command -v docker >/dev/null 2>&1 && echo YPOK || echo YPERR", 15)
	if err != nil {
		return nil, err
	}
	if code != 0 || !strings.Contains(out, "YPOK") {
		return nil, svcBadReq("目标节点缺少 docker 命令，请先安装 Docker")
	}
	// 53 端口占用预检（按目标监听地址判定冲突：通配/同地址拦截，其余特定地址提示）
	conflicts, others, err := s.dnsCheckPort53(ctx, nodeId, cfg.ListenIP)
	if err != nil {
		return nil, err
	}
	warnings := []string{}
	if len(conflicts) > 0 {
		parts := []string{}
		for _, o := range conflicts {
			p := o.Proto + " " + o.Addr + ":53"
			if name := procShortName(o.Process); name != "" {
				p += "（" + name + "）"
			}
			parts = append(parts, p)
		}
		return nil, svcBadReq("端口 53 与监听地址 " + cfg.ListenIP + " 冲突: " + strings.Join(parts, "、") +
			"（通配或同地址绑定互斥；请先停用占用方，或改用其余内网 IP）")
	}
	if len(others) > 0 {
		warnings = append(warnings, "节点上存在其他特定地址的 53 监听（如 systemd-resolved stub），与内网 IP 精确绑定不冲突，已忽略")
	}
	// 写 compose + conf
	records := []model.DnsRecord{}
	if err := s.db.Order("sort ASC, id ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	sites, err := s.enabledSites()
	if err != nil {
		return nil, err
	}
	for path, content := range map[string][]byte{
		dnsComposePath: []byte(renderDnsCompose(cfg.withDefaults().Image)),
		dnsConfPath:    []byte(renderDnsmasqConf(cfg, records, sites)),
	} {
		out, code, err := execOnNode(ctx, s.nodes, nodeId, buildB64WriteScript(path, content), 30)
		if err != nil {
			return nil, err
		}
		if code != 0 {
			return nil, errs.Wrapc(errs.CodeFileOpFailed, "写入 "+path+" 失败: "+tailOutput(out, 400))
		}
	}
	// compose up（compose 插件优先，回退独立二进制）
	up := "cd " + dnsDir + " && if docker compose version >/dev/null 2>&1; then docker compose -p " + dnsProject + " up -d; else docker-compose -p " + dnsProject + " up -d; fi"
	out, code, err = execOnNode(ctx, s.nodes, nodeId, up, 900)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, svcBadReq("容器启动失败（镜像拉取或 compose 错误）: " + tailOutput(out, 400))
	}
	// 启动确认
	confirm := "sleep 1; ss -H -lntup 2>/dev/null | grep -c '[:.]53[[:space:]]' || true"
	out, _, err = execOnNode(ctx, s.nodes, nodeId, confirm, 20)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "0" {
		warnings = append(warnings, "容器已启动但 53 端口暂未监听（可能仍在启动），请稍后在状态卡确认")
	}
	// 记录部署节点
	cfg.DeployNodeID = nodeId
	if err := s.saveConfigLocked(cfg); err != nil {
		return warnings, err
	}
	return warnings, nil
}

// Undeploy 卸载（compose down + 可选清理文件），清除部署节点记录。
func (s *DnsService) Undeploy(ctx context.Context, nodeId string, removeFiles bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	down := "cd " + dnsDir + " && if docker compose version >/dev/null 2>&1; then docker compose -p " + dnsProject + " down; else docker-compose -p " + dnsProject + " down; fi"
	if _, code, err := execOnNode(ctx, s.nodes, nodeId, down, 120); err != nil {
		return err
	} else if code != 0 {
		return svcBadReq("容器停止失败（可能本就未部署，可勾选清理文件后重试）")
	}
	if removeFiles {
		if _, code, err := execOnNode(ctx, s.nodes, nodeId, "rm -rf "+dnsDir, 15); err != nil {
			return err
		} else if code != 0 {
			return svcBadReq("清理部署目录失败")
		}
	}
	cfg := s.GetConfig()
	if cfg.DeployNodeID == nodeId {
		cfg.DeployNodeID = ""
		return s.saveConfigLocked(cfg)
	}
	return nil
}

// ApplyNode 渲染配置 → 原子写入 → HUP 平滑重载。
func (s *DnsService) ApplyNode(ctx context.Context, nodeId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyLocked(ctx, nodeId)
}

func (s *DnsService) applyLocked(ctx context.Context, nodeId string) error {
	if s.GetConfig().DeployNodeID == "" {
		return svcBadReq("尚未部署内网 DNS 服务，请先部署")
	}
	records := []model.DnsRecord{}
	if err := s.db.Order("sort ASC, id ASC").Find(&records).Error; err != nil {
		return err
	}
	sites, err := s.enabledSites()
	if err != nil {
		return err
	}
	conf := renderDnsmasqConf(s.GetConfig(), records, sites)
	out, code, err := execOnNode(ctx, s.nodes, nodeId, buildB64WriteScript(dnsConfPath, []byte(conf)), 30)
	if err != nil {
		return err
	}
	if code != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "写入配置失败: "+tailOutput(out, 400))
	}
	out, code, err = execOnNode(ctx, s.nodes, nodeId, "docker kill -s HUP "+dnsContainer+" 2>&1", 20)
	if err != nil {
		return err
	}
	if code != 0 {
		return svcBadReq("重载失败（容器可能未运行，请检查部署状态）: " + tailOutput(out, 400))
	}
	return nil
}

// CheckResolve 解析测试：dig 优先，缺省回退 nslookup。
func (s *DnsService) CheckResolve(ctx context.Context, nodeId, domain string) (string, error) {
	domain = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(domain), ".")))
	if !dnsHostnamePattern.MatchString(domain) {
		return "", svcBadReq("域名不合法")
	}
	cfg := s.GetConfig()
	if cfg.ListenIP == "" {
		return "", svcBadReq("请先在设置中填写监听 IP")
	}
	script := fmt.Sprintf(`if command -v dig >/dev/null 2>&1; then
  dig @'%s' '%s' +short +time=3 +tries=1 2>&1
  exit 0
fi
if command -v nslookup >/dev/null 2>&1; then
  nslookup '%s' '%s' 2>&1
  exit 0
fi
echo 'YPERR:节点缺少 dig / nslookup（apt install dnsutils 或 yum install bind-utils）'
exit 64`, cfg.ListenIP, domain, domain, cfg.ListenIP)
	out, code, err := execOnNode(ctx, s.nodes, nodeId, script, 20)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", svcBadReq(tailOutput(out, 400))
	}
	if strings.TrimSpace(out) == "" {
		return "（空结果：无匹配记录或解析失败）", nil
	}
	return strings.TrimRight(out, "\n"), nil
}

// ---------- 记录 CRUD ----------

// ListRecords 记录列表（sort,id 升序）。
func (s *DnsService) ListRecords() ([]model.DnsRecord, error) {
	rows := []model.DnsRecord{}
	if err := s.db.Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SaveRecord 新建（ID=0）/更新；已部署时自动应用，失败回滚 DB。
func (s *DnsService) SaveRecord(ctx context.Context, r *model.DnsRecord) (*model.DnsRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := dnsValidateRecord(r); err != nil {
		return nil, err
	}
	if err := s.dnsDupCheck(r); err != nil {
		return nil, err
	}
	isUpdate := r.ID != 0
	var before model.DnsRecord
	if isUpdate {
		if err := s.db.First(&before, r.ID).Error; err != nil {
			return nil, errs.Wrap(errs.ErrNotFound, "记录不存在")
		}
	}
	if isUpdate {
		if err := s.db.Model(&before).Updates(map[string]any{
			"type": r.Type, "domain": r.Domain, "target": r.Target,
			"enabled": r.Enabled, "comment": r.Comment, "sort": r.Sort,
		}).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.Create(r).Error; err != nil {
			return nil, err
		}
	}
	if nodeID := s.GetConfig().DeployNodeID; nodeID != "" {
		if err := s.applyLocked(ctx, nodeID); err != nil {
			s.rollbackRecord(&before, r, isUpdate)
			return nil, svcBadReq("记录已回滚（应用失败）: " + err.Error())
		}
	}
	return r, nil
}

// DeleteRecord 删除；已部署时自动应用，失败回滚。
func (s *DnsService) DeleteRecord(ctx context.Context, id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var row model.DnsRecord
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.Wrap(errs.ErrNotFound, "记录不存在")
	}
	if err := s.db.Delete(&row).Error; err != nil {
		return err
	}
	if nodeID := s.GetConfig().DeployNodeID; nodeID != "" {
		if err := s.applyLocked(ctx, nodeID); err != nil {
			if e := s.db.Create(&row).Error; e != nil {
				slog.Error("DNS 记录删除回滚失败", "id", row.ID, "err", e.Error())
			}
			_ = s.applyLocked(ctx, nodeID)
			return svcBadReq("删除已回滚（应用失败）: " + err.Error())
		}
	}
	return nil
}

// SetRecordEnabled 启停；已部署时自动应用，失败还原状态。
func (s *DnsService) SetRecordEnabled(ctx context.Context, id uint, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var row model.DnsRecord
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.Wrap(errs.ErrNotFound, "记录不存在")
	}
	if row.Enabled == enabled {
		return nil
	}
	if err := s.db.Model(&row).Update("enabled", enabled).Error; err != nil {
		return err
	}
	if nodeID := s.GetConfig().DeployNodeID; nodeID != "" {
		if err := s.applyLocked(ctx, nodeID); err != nil {
			if e := s.db.Model(&row).Update("enabled", !enabled).Error; e != nil {
				slog.Error("DNS 记录状态还原失败", "id", row.ID, "err", e.Error())
			}
			_ = s.applyLocked(ctx, nodeID)
			return svcBadReq("已还原记录状态（应用失败）: " + err.Error())
		}
	}
	return nil
}

// rollbackRecord 应用失败后的 DB 回滚（还原/删除；旧态重放失败仅记日志）。
func (s *DnsService) rollbackRecord(before *model.DnsRecord, cur *model.DnsRecord, isUpdate bool) {
	if isUpdate {
		if err := s.db.Save(before).Error; err != nil {
			slog.Error("DNS 记录回滚失败", "id", before.ID, "err", err.Error())
		}
	} else {
		if err := s.db.Delete(cur).Error; err != nil {
			slog.Error("DNS 记录回滚失败", "id", cur.ID, "err", err.Error())
		}
	}
	if nodeID := s.GetConfig().DeployNodeID; nodeID != "" {
		if err := s.applyLocked(context.Background(), nodeID); err != nil {
			slog.Warn("DNS 记录回滚后重放旧配置失败", "node", nodeID, "err", err.Error())
		}
	}
}
