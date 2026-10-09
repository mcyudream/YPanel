package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// ProbeService 监控探针（M29）：站点/容器/HTTP/TCP 可达性探测，状态翻转推送通知。
// 范围与商店/组网一致（local 节点）。探测经 agent exec 执行：
// HTTP=curl（跟随重定向）、TCP=bash /dev/tcp、容器=docker inspect。
// URL 与关键字经 base64 注入命令，杜绝 shell 注入。
type ProbeService struct {
	db    *gorm.DB
	nodes *NodeService
	notif *NotificationService

	mu       sync.Mutex
	streaks  map[uint]int  // 连续失败次数
	up       map[uint]bool // 当前判定状态（内存；重启后重建，首轮不触发恢复通知）
	notified map[uint]bool // 本轮 down 已通知（恢复后复位）
	seen     map[uint]bool // 重启后是否已观察过（抑制首轮误报）
}

// NewProbeService 创建。
func NewProbeService(db *gorm.DB, nodes *NodeService, notif *NotificationService) *ProbeService {
	return &ProbeService{
		db: db, nodes: nodes, notif: notif,
		streaks: map[uint]int{}, up: map[uint]bool{}, notified: map[uint]bool{}, seen: map[uint]bool{},
	}
}

// Start 启动调度循环（10s tick，按各探针 interval 到期执行）。
func (s *ProbeService) Start(ctx context.Context) {
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.checkDue(ctx)
			}
		}
	}()
}

// ---------- CRUD ----------

// ProbeInput 新建/编辑入参。
type ProbeInput struct {
	Name         string `json:"name" binding:"required"`
	TargetType   string `json:"targetType" binding:"required"`
	TargetRef    string `json:"targetRef" binding:"required"`
	Method       string `json:"method"`
	ExpectStatus string `json:"expectStatus"`
	Keyword      string `json:"keyword"`
	IntervalSec  int    `json:"intervalSec"`
	TimeoutSec   int    `json:"timeoutSec"`
	Retries      int    `json:"retries"`
	WebhookURL   string `json:"webhookUrl"`
	WebhookType  string `json:"webhookType"`
}

// List 探针列表（附展示状态：未启用=paused）。
func (s *ProbeService) List() ([]map[string]any, error) {
	var rows []model.MonitorProbe
	if err := s.db.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		status := r.LastStatus
		if !r.Enabled {
			status = "paused"
		}
		out = append(out, map[string]any{
			"id": r.ID, "name": r.Name, "targetType": r.TargetType, "targetRef": r.TargetRef,
			"method": r.Method, "expectStatus": r.ExpectStatus, "keyword": r.Keyword,
			"intervalSec": r.IntervalSec, "timeoutSec": r.TimeoutSec, "retries": r.Retries,
			"webhookUrl": r.WebhookURL, "webhookType": r.WebhookType,
			"enabled": r.Enabled, "status": status,
			"lastCheckedAt": r.LastCheckedAt, "lastDownAt": r.LastDownAt, "lastError": r.LastError,
			"createdAt": r.CreatedAt,
		})
	}
	return out, nil
}

// Create 新建（默认 Enabled=false，默认不开启）。
func (s *ProbeService) Create(in ProbeInput) (*model.MonitorProbe, error) {
	if err := s.validate(&in); err != nil {
		return nil, err
	}
	row := &model.MonitorProbe{
		Name: in.Name, TargetType: in.TargetType, TargetRef: in.TargetRef,
		Method: in.Method, ExpectStatus: in.ExpectStatus, Keyword: in.Keyword,
		IntervalSec: in.IntervalSec, TimeoutSec: in.TimeoutSec, Retries: in.Retries,
		WebhookURL: in.WebhookURL, WebhookType: in.WebhookType,
		Enabled: false,
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// Update 编辑（不改动运行状态字段）。
func (s *ProbeService) Update(id uint, in ProbeInput) (*model.MonitorProbe, error) {
	var row model.MonitorProbe
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.notFound", "探针不存在")
	}
	if err := s.validate(&in); err != nil {
		return nil, err
	}
	updates := map[string]any{
		"name": in.Name, "target_type": in.TargetType, "target_ref": in.TargetRef,
		"method": in.Method, "expect_status": in.ExpectStatus, "keyword": in.Keyword,
		"interval_sec": in.IntervalSec, "timeout_sec": in.TimeoutSec, "retries": in.Retries,
		"webhook_url": in.WebhookURL, "webhook_type": in.WebhookType,
	}
	if err := s.db.Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	// 目标变化后重置判定状态，避免沿用旧目标的连败计数
	s.mu.Lock()
	delete(s.streaks, row.ID)
	delete(s.notified, row.ID)
	delete(s.seen, row.ID)
	s.up[row.ID] = true
	s.mu.Unlock()
	return s.reload(row.ID)
}

// SetEnabled 启用/停用（停用后展示 paused，不再调度）。
func (s *ProbeService) SetEnabled(id uint, enabled bool) error {
	var row model.MonitorProbe
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.New(errs.CodeNotFound, "error.notFound", "探针不存在")
	}
	if err := s.db.Model(&row).Update("enabled", enabled).Error; err != nil {
		return err
	}
	s.mu.Lock()
	if !enabled {
		s.streaks[id] = 0
		s.notified[id] = false
	}
	s.mu.Unlock()
	return nil
}

// Delete 删除。
func (s *ProbeService) Delete(id uint) error {
	if err := s.db.Delete(&model.MonitorProbe{}, id).Error; err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.streaks, id)
	delete(s.up, id)
	delete(s.notified, id)
	delete(s.seen, id)
	s.mu.Unlock()
	return nil
}

// Test 试跑一次（不改状态、不发通知）。
func (s *ProbeService) Test(in ProbeInput) (map[string]any, error) {
	if err := s.validate(&in); err != nil {
		return nil, err
	}
	p := s.probeFromInput(in)
	ok, detail := s.check(context.Background(), &p)
	return map[string]any{"ok": ok, "detail": detail, "checkedAt": time.Now()}, nil
}

// ---------- 调度与执行 ----------

func (s *ProbeService) checkDue(ctx context.Context) {
	var probes []model.MonitorProbe
	if err := s.db.Where("enabled = ?", true).Find(&probes).Error; err != nil {
		slog.Warn("probe: 读取探针失败", "err", err)
		return
	}
	now := time.Now()
	for i := range probes {
		p := &probes[i]
		if p.LastCheckedAt != nil && now.Sub(*p.LastCheckedAt) < time.Duration(p.IntervalSec)*time.Second {
			continue
		}
		s.runOnce(ctx, p)
	}
}

// runOnce 执行一次探测并维护状态机（连续失败≥Retries 判 down；翻转推通知）。
func (s *ProbeService) runOnce(ctx context.Context, p *model.MonitorProbe) {
	ok, detail := s.check(ctx, p)
	now := time.Now()

	s.mu.Lock()
	first := !s.seen[p.ID]
	s.seen[p.ID] = true
	streak := s.streaks[p.ID]
	prevUp := s.up[p.ID]
	if ok {
		streak = 0
	} else {
		streak++
	}
	s.streaks[p.ID] = streak
	retries := p.Retries
	if retries < 1 {
		retries = 1
	}
	confirmedDown := !ok && streak >= retries
	recovered := ok && !first && !prevUp && s.notified[p.ID]
	if recovered {
		s.notified[p.ID] = false
	}
	if confirmedDown && !s.notified[p.ID] {
		s.notified[p.ID] = true
	}
	s.up[p.ID] = ok
	s.mu.Unlock()

	// 展示状态：成功=up；失败且已确认=down；失败未确认=保持原状态（首轮失败为待定 ""，
	// 不能显示 up——首轮即失败时界面上"正常+报错"并存的误导已在真机踩过）
	status := "up"
	if !ok {
		switch {
		case confirmedDown:
			status = "down"
		case !first && prevUp:
			status = "up"
		default:
			status = ""
		}
	}

	updates := map[string]any{"last_checked_at": now, "last_status": status, "last_error": ""}
	if !ok {
		updates["last_error"] = truncStr(detail, 500)
	}
	if status == "down" {
		updates["last_down_at"] = now
	}
	if err := s.db.Model(&model.MonitorProbe{}).Where("id = ?", p.ID).Updates(updates).Error; err != nil {
		slog.Warn("probe: 状态落库失败", "id", p.ID, "err", err)
	}

	name := fmt.Sprintf("%s（%s）", p.Name, probeTargetLabel(p))
	if confirmedDown && s.notified[p.ID] {
		msg := fmt.Sprintf("【YPanel 探针告警】%s 连续 %d 次检测失败：%s", name, retries, detail)
		slog.Warn("probe: down", "id", p.ID, "name", p.Name, "detail", detail)
		if s.notif != nil {
			s.notif.Push("error", "探针告警", msg)
		}
		if p.WebhookURL != "" {
			postProbeWebhook(ctx, p.WebhookURL, p.WebhookType, msg)
		}
	}
	if recovered {
		msg := fmt.Sprintf("【YPanel 探针恢复】%s 检测已恢复正常", name)
		slog.Info("probe: recovered", "id", p.ID, "name", p.Name)
		if s.notif != nil {
			s.notif.Push("info", "探针恢复", msg)
		}
		if p.WebhookURL != "" {
			postProbeWebhook(ctx, p.WebhookURL, p.WebhookType, msg)
		}
	}
}

// check 按类型执行探测。ok=true 可达；detail 为失败原因或成功摘要。
func (s *ProbeService) check(ctx context.Context, p *model.MonitorProbe) (bool, string) {
	switch p.TargetType {
	case "http":
		return s.checkHTTP(ctx, p, p.TargetRef)
	case "site":
		var site model.Site
		if err := s.db.Where("name = ?", p.TargetRef).First(&site).Error; err != nil {
			return false, "站点不存在: " + p.TargetRef
		}
		return s.checkHTTP(ctx, p, probeSiteURL(site))
	case "tcp":
		return s.checkTCP(ctx, p)
	case "container":
		return s.checkContainer(ctx, p)
	default:
		return false, "未知探测类型: " + p.TargetType
	}
}

func (s *ProbeService) exec(ctx context.Context, timeoutSecs int, cmd string) (dto.ExecResp, error) {
	ac, err := s.client()
	if err != nil {
		return dto.ExecResp{}, err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeoutSecs + 5})
	if err != nil {
		return dto.ExecResp{}, err
	}
	return *out, nil
}

// checkHTTP curl 探测（跟随重定向）；期望状态码匹配 + 可选关键字。URL/关键字 base64 注入。
func (s *ProbeService) checkHTTP(ctx context.Context, p *model.MonitorProbe, rawURL string) (bool, string) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || strings.ContainsAny(rawURL, " \r\n\t") {
		return false, "URL 不合法（仅支持 http/https）: " + rawURL
	}
	timeout := clampProbeTimeout(p.TimeoutSec)
	b64u := base64.StdEncoding.EncodeToString([]byte(rawURL))
	codeCmd := fmt.Sprintf(`u=$(printf %%s '%s' | base64 -d); curl -sS -L --max-redirs 5 -m %d -o /dev/null -w '%%{http_code}' "$u" 2>&1`, b64u, timeout)
	out, err := s.exec(ctx, timeout, codeCmd)
	if err != nil {
		return false, "探测执行失败: " + err.Error()
	}
	if out.ExitCode != 0 {
		return false, "HTTP 请求失败: " + strings.TrimSpace(out.Output)
	}
	code, cerr := strconv.Atoi(strings.TrimSpace(out.Output))
	if cerr != nil {
		return false, "HTTP 状态码解析失败: " + strings.TrimSpace(out.Output)
	}
	if !probeStatusMatch(code, p.ExpectStatus) {
		return false, fmt.Sprintf("HTTP %d，期望 %s", code, p.ExpectStatus)
	}
	if kw := strings.TrimSpace(p.Keyword); kw != "" {
		b64k := base64.StdEncoding.EncodeToString([]byte(kw))
		kwCmd := fmt.Sprintf(`k=$(printf %%s '%s' | base64 -d); u=$(printf %%s '%s' | base64 -d); curl -sS -L --max-redirs 5 -m %d "$u" 2>/dev/null | grep -cF -- "$k"`, b64k, b64u, timeout)
		kout, kerr := s.exec(ctx, timeout, kwCmd)
		if kerr != nil || kout.ExitCode != 0 || strings.TrimSpace(kout.Output) == "0" {
			return false, "响应体未包含关键字: " + kw
		}
	}
	return true, fmt.Sprintf("HTTP %d", code)
}

// checkTCP bash /dev/tcp 探测端口可达。
func (s *ProbeService) checkTCP(ctx context.Context, p *model.MonitorProbe) (bool, string) {
	host, port, ok := strings.Cut(p.TargetRef, ":")
	host = strings.TrimSpace(host)
	if !ok || !probeHostPattern.MatchString(host) {
		return false, "TCP 目标不合法（host:port）: " + p.TargetRef
	}
	pn, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil || pn < 1 || pn > 65535 {
		return false, "端口不合法: " + port
	}
	timeout := clampProbeTimeout(p.TimeoutSec)
	cmd := fmt.Sprintf(`timeout %d bash -c 'exec 3<>/dev/tcp/%s/%d' 2>/dev/null`, timeout, host, pn)
	out, err := s.exec(ctx, timeout, cmd)
	if err != nil {
		return false, "探测执行失败: " + err.Error()
	}
	if out.ExitCode != 0 {
		return false, fmt.Sprintf("TCP %s:%d 连接失败（超时或拒绝）", host, pn)
	}
	return true, fmt.Sprintf("TCP %s:%d 可达", host, pn)
}

// checkContainer docker inspect 运行态探测。
func (s *ProbeService) checkContainer(ctx context.Context, p *model.MonitorProbe) (bool, string) {
	name := strings.TrimSpace(p.TargetRef)
	if !probeContainerPattern.MatchString(name) {
		return false, "容器名不合法: " + name
	}
	out, err := s.exec(ctx, 15, "docker inspect -f {{.State.Status}} "+name+" 2>/dev/null")
	if err != nil {
		return false, "探测执行失败: " + err.Error()
	}
	state := strings.TrimSpace(out.Output)
	if state == "" {
		return false, "容器不存在: " + name
	}
	if state != "running" {
		return false, "容器状态: " + state + "（非 running）"
	}
	return true, "容器运行中"
}

// ---------- 工具 ----------

func (s *ProbeService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

func (s *ProbeService) reload(id uint) (*model.MonitorProbe, error) {
	var row model.MonitorProbe
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *ProbeService) probeFromInput(in ProbeInput) model.MonitorProbe {
	return model.MonitorProbe{
		Name: in.Name, TargetType: in.TargetType, TargetRef: in.TargetRef,
		Method: in.Method, ExpectStatus: in.ExpectStatus, Keyword: in.Keyword,
		IntervalSec: in.IntervalSec, TimeoutSec: in.TimeoutSec, Retries: in.Retries,
		WebhookURL: in.WebhookURL, WebhookType: in.WebhookType, Enabled: true,
	}
}

// validate 入参校验（值域白名单 + 归一化默认值）。
func (s *ProbeService) validate(in *ProbeInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.TargetRef = strings.TrimSpace(in.TargetRef)
	if in.Name == "" || len(in.Name) > 64 {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "探针名称必填且不超过 64 字")
	}
	switch in.TargetType {
	case "site":
		var cnt int64
		s.db.Model(&model.Site{}).Where("name = ?", in.TargetRef).Count(&cnt)
		if cnt == 0 {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "站点不存在: "+in.TargetRef)
		}
	case "container":
		if !probeContainerPattern.MatchString(in.TargetRef) {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "容器名不合法: "+in.TargetRef)
		}
	case "http":
		u, err := url.Parse(in.TargetRef)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || strings.ContainsAny(in.TargetRef, " \r\n\t") {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "URL 不合法（仅支持 http/https）")
		}
	case "tcp":
		host, port, ok := strings.Cut(in.TargetRef, ":")
		if !ok || !probeHostPattern.MatchString(strings.TrimSpace(host)) {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "TCP 目标不合法（host:port）")
		}
		if pn, err := strconv.Atoi(strings.TrimSpace(port)); err != nil || pn < 1 || pn > 65535 {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "端口不合法: "+port)
		}
	default:
		return errs.New(errs.CodeBadRequest, "error.badRequest", "探测类型不合法: "+in.TargetType)
	}
	if in.Method == "" {
		in.Method = "GET"
	}
	in.Method = strings.ToUpper(in.Method)
	if !slices.Contains([]string{"GET", "HEAD", "POST"}, in.Method) {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "HTTP 方法仅支持 GET/HEAD/POST")
	}
	if in.ExpectStatus == "" {
		in.ExpectStatus = "2xx"
	}
	if !probeExpectPattern.MatchString(strings.ReplaceAll(in.ExpectStatus, " ", "")) {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "期望状态码格式不合法（如 2xx、200、200-299，逗号分隔）")
	}
	if in.IntervalSec <= 0 {
		in.IntervalSec = 60
	}
	if in.IntervalSec < 10 || in.IntervalSec > 86400 {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "检测间隔需在 10~86400 秒")
	}
	if in.TimeoutSec <= 0 {
		in.TimeoutSec = 5
	}
	if in.TimeoutSec < 1 || in.TimeoutSec > 60 {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "超时需在 1~60 秒")
	}
	if in.Retries <= 0 {
		in.Retries = 3
	}
	if in.Retries > 10 {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "失败重试次数最多 10")
	}
	if len(in.Keyword) > 128 {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "关键字最长 128 字")
	}
	if in.WebhookURL != "" {
		u, err := url.Parse(in.WebhookURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "Webhook URL 不合法")
		}
	}
	if in.WebhookType == "" {
		in.WebhookType = "generic"
	}
	return nil
}

// probeSiteURL 站点 → 探测 URL（CertDomain 非空走 https）。
func probeSiteURL(site model.Site) string {
	scheme := "http"
	if site.CertDomain != "" {
		scheme = "https"
	}
	u := scheme + "://" + site.Domain
	if site.Port > 0 && ((scheme == "http" && site.Port != 80) || (scheme == "https" && site.Port != 443)) {
		u += ":" + strconv.Itoa(site.Port)
	}
	return u
}

// probeStatusMatch 期望状态码匹配："2xx"、"200"、"200-299"，逗号混排。
func probeStatusMatch(code int, expect string) bool {
	expect = strings.ReplaceAll(expect, " ", "")
	if expect == "" {
		return true
	}
	for _, tok := range strings.Split(expect, ",") {
		tok = strings.ToLower(tok)
		switch {
		case strings.HasSuffix(tok, "xx"):
			if n, err := strconv.Atoi(strings.TrimSuffix(tok, "xx")); err == nil && code/100 == n {
				return true
			}
		case strings.Contains(tok, "-"):
			lo, hi, _ := strings.Cut(tok, "-")
			l, e1 := strconv.Atoi(lo)
			h, e2 := strconv.Atoi(hi)
			if e1 == nil && e2 == nil && code >= l && code <= h {
				return true
			}
		default:
			if n, err := strconv.Atoi(tok); err == nil && code == n {
				return true
			}
		}
	}
	return false
}

func clampProbeTimeout(t int) int {
	if t < 1 {
		return 5
	}
	if t > 60 {
		return 60
	}
	return t
}

func probeTargetLabel(p *model.MonitorProbe) string {
	switch p.TargetType {
	case "site":
		return "站点 " + p.TargetRef
	case "container":
		return "容器 " + p.TargetRef
	case "tcp":
		return "TCP " + p.TargetRef
	default:
		return p.TargetRef
	}
}

// postProbeWebhook 探针独立 webhook（复用告警的 webhook 载荷形态；AlertService.notify 暂未抽公共函数，避免热文件改动）。
func postProbeWebhook(ctx context.Context, rawURL, typ, msg string) {
	var payload any
	switch typ {
	case "feishu":
		payload = map[string]any{"msg_type": "text", "content": map[string]string{"text": msg}}
	case "dingtalk", "wecom":
		payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": msg}}
	case "telegram":
		if i := strings.Index(rawURL, "?chat_id="); i > 0 {
			payload = map[string]any{"chat_id": rawURL[i+len("?chat_id="):], "text": msg}
			rawURL = rawURL[:i]
		} else {
			return
		}
	default:
		payload = map[string]any{"text": msg, "source": "ypanel", "time": time.Now().Format(time.RFC3339)}
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", rawURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

var (
	probeHostPattern      = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$`)
	probeContainerPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
	probeExpectPattern    = regexp.MustCompile(`^([1-5]xx|[1-5][0-9]{2}|[1-5][0-9]{2}-[1-5][0-9]{2])(,([1-5]xx|[1-5][0-9]{2}|[1-5][0-9]{2}-[1-5][0-9]{2}))*$`)
)
