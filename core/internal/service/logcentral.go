package service

// M33 日志中心：集中存储（VictoriaLogs）多节点聚合。
// 地址全自动管理：agent 按镜像发现本节点 VL 容器（宿主映射端口），core 经 agent 通道
// 并发代理查询各节点 VL（只读白名单路径），结果归并（日志行打节点标、hits 求和、切面合并）。

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
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

// vlDiscoverTTL 发现结果缓存窗口（避免每次查询全节点 docker ps）。
const vlDiscoverTTL = 30 * time.Second

// logCentralEndpointKey 保留旧设置键读取（历史存量），新架构不再使用。
const logCentralEndpointKey = "logs.central.endpoint"

// LogCentralService 集中日志（VictoriaLogs）多节点聚合服务。
type LogCentralService struct {
	db       *gorm.DB
	settings *SettingService
	nodes    *NodeService
	http     *http.Client

	mu        sync.Mutex
	cacheTime time.Time
	cache     []VLInstance
}

// NewLogCentralService 创建。
func NewLogCentralService(db *gorm.DB, settings *SettingService, nodes *NodeService) *LogCentralService {
	return &LogCentralService{db: db, settings: settings, nodes: nodes, http: &http.Client{Timeout: 30 * time.Second}}
}

// VLInstance 一个节点的 VL 发现状态。
type VLInstance struct {
	NodeID    string `json:"nodeId"`
	NodeName  string `json:"nodeName"`
	Online    bool   `json:"online"`
	Found     bool   `json:"found"`
	Container string `json:"container,omitempty"`
	Port      string `json:"port,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Status 聚合状态：各节点 VL 自动发现结果（30s 缓存）。
func (s *LogCentralService) Status(ctx context.Context) []VLInstance {
	s.mu.Lock()
	if time.Since(s.cacheTime) < vlDiscoverTTL && s.cache != nil {
		out := s.cache
		s.mu.Unlock()
		return out
	}
	s.mu.Unlock()

	nodes := s.nodes.ListNodes()
	if len(nodes) > 20 {
		nodes = nodes[:20]
	}
	out := make([]VLInstance, 0, len(nodes))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, n := range nodes {
		wg.Add(1)
		go func(n map[string]any) {
			defer wg.Done()
			id, _ := n["id"].(string)
			name, _ := n["name"].(string)
			online, _ := n["online"].(bool)
			inst := VLInstance{NodeID: id, NodeName: name, Online: online}
			if !online {
				inst.Error = "离线"
				mu.Lock()
				out = append(out, inst)
				mu.Unlock()
				return
			}
			node, err := s.nodes.ByID(id)
			if err != nil {
				inst.Error = errs.From(err).Message
				mu.Lock()
				out = append(out, inst)
				mu.Unlock()
				return
			}
			disc, err := agentclient.GetJSON[dto.VLDiscovery](
				agentclient.New(node.BaseURL, node.Token), ctx, "/agent/v1/logs/vl/discover")
			if err != nil {
				inst.Error = errs.From(err).Message
				mu.Lock()
				out = append(out, inst)
				mu.Unlock()
				return
			}
			inst.Found = disc.Found
			inst.Container = disc.Container
			inst.Port = disc.Port
			if !disc.Found {
				inst.Error = "未安装 VictoriaLogs"
			}
			mu.Lock()
			out = append(out, inst)
			mu.Unlock()
		}(n)
	}
	wg.Wait()
	sort.Slice(out, func(a, b int) bool {
		if out[a].NodeID == "local" {
			return true
		}
		if out[b].NodeID == "local" {
			return false
		}
		return out[a].NodeID < out[b].NodeID
	})

	s.mu.Lock()
	s.cache = out
	s.cacheTime = time.Now()
	s.mu.Unlock()
	return out
}

// vlReady 可查询的实例集合（在线且发现 VL）。
func (s *LogCentralService) vlReady(ctx context.Context) []VLInstance {
	out := []VLInstance{}
	for _, inst := range s.Status(ctx) {
		if inst.Online && inst.Found {
			out = append(out, inst)
		}
	}
	return out
}

// vlRequest 经 agent 代理请求某节点 VL 只读接口，返回上游 body。
func (s *LogCentralService) vlRequest(ctx context.Context, inst VLInstance, path string, params map[string]string) (string, error) {
	node, err := s.nodes.ByID(inst.NodeID)
	if err != nil {
		return "", err
	}
	resp, err := agentclient.DoJSON[dto.VLProxyReq, dto.VLProxyResp](
		agentclient.New(node.BaseURL, node.Token), ctx,
		http.MethodPost, "/agent/v1/logs/vl/query",
		&dto.VLProxyReq{Path: path, Params: params})
	if err != nil {
		return "", err
	}
	if resp.Status != http.StatusOK {
		return "", fmt.Errorf("VL HTTP %d: %s", resp.Status, truncateStr(resp.Body, 200))
	}
	return resp.Body, nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// CentralQueryItem 归并后的一条命中行（多节点时带 node 标）。
type CentralQueryItem struct {
	Ts     string `json:"ts"`
	Stream string `json:"stream"`
	Msg    string `json:"msg"`
	Node   string `json:"node,omitempty"`
}

// CentralQueryResult 有界查询结果。
type CentralQueryResult struct {
	Items     []CentralQueryItem `json:"items"`
	Truncated bool               `json:"truncated"`
}

// CentralQuery 查询入参（start/end 透传 VL；offset/limit 为全局窗口：每节点取最新 offset+limit 条，
// 归并后切 [offset, offset+limit)——与数据在节点间的分布无关，翻页天然正确）。
type CentralQuery struct {
	Query  string `json:"query"`
	Start  string `json:"start"`
	End    string `json:"end"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// centralQueryLimitMax 单节点拉取上限。
const centralQueryLimitMax = 50000

// Query 多节点聚合 LogsQL 查询：每节点取最新 offset+limit 条，全局归并后切
// [offset, offset+limit) 窗口（升序返回），行带节点标；窗口算法与节点分布无关。
func (s *LogCentralService) Query(ctx context.Context, q CentralQuery) (*CentralQueryResult, error) {
	if strings.TrimSpace(q.Query) == "" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "查询语句为空")
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	if limit > centralQueryLimitMax {
		limit = centralQueryLimitMax
	}
	targets := s.vlReady(ctx)
	if len(targets) == 0 {
		return nil, errs.New(errs.CodeNotFound, "error.notFound", "未发现可用的 VictoriaLogs 实例（请先在节点上安装 VictoriaLogs 应用）")
	}
	fetchPer := limit + q.Offset
	if fetchPer > centralQueryLimitMax {
		fetchPer = centralQueryLimitMax
	}

	type nodeItems struct {
		items []CentralQueryItem
	}
	results := make([]nodeItems, len(targets))
	var wg sync.WaitGroup
	for i, inst := range targets {
		wg.Add(1)
		go func(i int, inst VLInstance) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			params := map[string]string{
				"query": q.Query, "start": q.Start, "end": q.End,
				"limit": strconv.Itoa(fetchPer),
			}
			body, err := s.vlRequest(ctx, inst, "/select/logsql/query", params)
			if err != nil {
				results[i] = nodeItems{}
				return
			}
			parsed, perr := parseCentralLines(strings.NewReader(body), fetchPer)
			if perr != nil {
				results[i] = nodeItems{}
				return
			}
			for j := range parsed.Items {
				parsed.Items[j].Node = inst.NodeName
			}
			results[i] = nodeItems{items: parsed.Items}
		}(i, inst)
	}
	wg.Wait()

	merged := make([]CentralQueryItem, 0, 1024)
	for i := range results {
		merged = append(merged, results[i].items...)
	}
	sort.SliceStable(merged, func(a, b int) bool { return merged[a].Ts < merged[b].Ts })

	// 全局升序数组里切最新侧窗口 [len-offset-limit, len-offset)
	out := &CentralQueryResult{Items: make([]CentralQueryItem, 0)}
	end := len(merged) - q.Offset
	start := end - limit
	if start < 0 {
		start = 0
	}
	if end > len(merged) {
		end = len(merged)
	}
	if end < start {
		end = start
	}
	out.Items = merged[start:end]
	return out, nil
}

// CentralHitSeries hits 的分组系列。
type CentralHitSeries struct {
	Fields     map[string]any `json:"fields"`
	Timestamps []string       `json:"timestamps"`
	Values     []uint64       `json:"values"`
	Total      uint64         `json:"total"`
	Node       string         `json:"node,omitempty"`
}

// CentralHits 聚合直方图：total 与分桶跨节点求和。
type CentralHits struct {
	Total      uint64             `json:"total"`
	Timestamps []string           `json:"timestamps"`
	Values     []uint64           `json:"values"`
	Series     []CentralHitSeries `json:"series"`
}

// Hits 聚合直方图（各节点 hits 按时间桶求和；series 带节点标）。
// 注意：VL /hits 的 step 为必填（缺省报 cannot parse duration from 'step='），空值默认 1m。
func (s *LogCentralService) Hits(ctx context.Context, query, start, end, step string) (*CentralHits, error) {
	if strings.TrimSpace(query) == "" {
		query = "*"
	}
	if strings.TrimSpace(step) == "" {
		step = "1m"
	}
	targets := s.vlReady(ctx)
	if len(targets) == 0 {
		return nil, errs.New(errs.CodeNotFound, "error.notFound", "未发现可用的 VictoriaLogs 实例")
	}
	out := &CentralHits{Series: make([]CentralHitSeries, 0)}
	bucket := map[string]uint64{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, inst := range targets {
		wg.Add(1)
		go func(inst VLInstance) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			params := map[string]string{"query": query, "start": start, "end": end, "step": step}
			body, err := s.vlRequest(ctx, inst, "/select/logsql/hits", params)
			if err != nil {
				return
			}
			var parsed struct {
				Hits []CentralHitSeries `json:"hits"`
			}
			if json.Unmarshal([]byte(body), &parsed) != nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, h := range parsed.Hits {
				h.Node = inst.NodeName
				out.Total += h.Total
				out.Series = append(out.Series, h)
				for i, ts := range h.Timestamps {
					if i < len(h.Values) {
						bucket[ts] += h.Values[i]
					}
				}
			}
		}(inst)
	}
	wg.Wait()
	for ts, v := range bucket {
		out.Timestamps = append(out.Timestamps, ts)
		out.Values = append(out.Values, v)
	}
	sort.SliceStable(out.Timestamps, func(a, b int) bool { return out.Timestamps[a] < out.Timestamps[b] })
	values := make([]uint64, len(out.Timestamps))
	for i, ts := range out.Timestamps {
		values[i] = bucket[ts]
	}
	out.Values = values
	return out, nil
}

// CentralStreamValue 流字段值（跨节点求和）。
type CentralStreamValue struct {
	Value string `json:"value"`
	Hits  uint64 `json:"hits"`
}

// StreamValues 聚合流字段值（默认 container_name，同名跨节点求和，降序截断）。
func (s *LogCentralService) StreamValues(ctx context.Context, field, query, start, end string, limit int) ([]CentralStreamValue, error) {
	if field == "" {
		field = "container_name"
	}
	if strings.TrimSpace(query) == "" {
		query = "*"
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	targets := s.vlReady(ctx)
	if len(targets) == 0 {
		return []CentralStreamValue{}, nil
	}
	bucket := map[string]uint64{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, inst := range targets {
		wg.Add(1)
		go func(inst VLInstance) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			params := map[string]string{"field": field, "query": query, "start": start, "end": end, "limit": strconv.Itoa(limit)}
			body, err := s.vlRequest(ctx, inst, "/select/logsql/stream_field_values", params)
			if err != nil {
				return
			}
			var parsed struct {
				Values []CentralStreamValue `json:"values"`
			}
			if json.Unmarshal([]byte(body), &parsed) != nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, v := range parsed.Values {
				bucket[v.Value] += v.Hits
			}
		}(inst)
	}
	wg.Wait()
	out := make([]CentralStreamValue, 0, len(bucket))
	for v, h := range bucket {
		out = append(out, CentralStreamValue{Value: v, Hits: h})
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Hits > out[b].Hits })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ---------- 保留策略（VL 自动清理过期数据，保留期经 .env + 容器重建生效） ----------

// VLRetention 节点 VL 当前保留期。
type VLRetention struct {
	NodeID   string `json:"nodeId"`
	NodeName string `json:"nodeName"`
	Found    bool   `json:"found"`
	Period   string `json:"period"` // 如 30d；空 = 读取失败
	Error    string `json:"error,omitempty"`
}

// envPathOf VL 应用的 .env 路径（容器名 = compose 项目名 = app-victoria-logs）。
func envPathOf(inst VLInstance) string {
	return "/opt/ypanel/compose/app-" + strings.TrimPrefix(inst.Container, "app-") + "/.env"
}

// readRetention 读节点 VL 的保留期（grep .env）。
func (s *LogCentralService) readRetention(ctx context.Context, inst VLInstance) (string, error) {
	out, _, err := execOnNode(ctx, s.nodes, inst.NodeID,
		"grep -E '^RETENTION_PERIOD=' "+envPathOf(inst)+" 2>/dev/null | cut -d= -f2 | tr -d '\r'", 15)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Retentions 各节点 VL 当前保留期。
func (s *LogCentralService) Retentions(ctx context.Context) []VLRetention {
	out := []VLRetention{}
	for _, inst := range s.Status(ctx) {
		r := VLRetention{NodeID: inst.NodeID, NodeName: inst.NodeName, Found: inst.Found}
		if !inst.Online {
			r.Error = "离线"
			out = append(out, r)
			continue
		}
		if !inst.Found {
			r.Error = "未安装"
			out = append(out, r)
			continue
		}
		period, err := s.readRetention(ctx, inst)
		if err != nil {
			r.Error = errs.From(err).Message
		}
		r.Period = period
		out = append(out, r)
	}
	return out
}

// SetRetention 统一设置保留期（写 .env RETENTION_PERIOD 并重建 VL 容器生效；VL 自动清理过期数据）。
func (s *LogCentralService) SetRetention(ctx context.Context, period string) (applied []string, failures []string) {
	period = normalizeRetention(period)
	if period == "" {
		return nil, []string{"保留期格式不合法（如 7d / 30d / 90d / 1y）"}
	}
	for _, inst := range s.vlReady(ctx) {
		if err := s.applyRetention(ctx, inst, period); err != nil {
			failures = append(failures, inst.NodeName+": "+errs.From(err).Message)
			continue
		}
		applied = append(applied, inst.NodeName)
	}
	return applied, failures
}

// normalizeRetention 保留期归一：纯数字视为天数；支持 d/w/m/y 后缀。
func normalizeRetention(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	if n, err := strconv.Atoi(raw); err == nil {
		if n <= 0 || n > 3650 {
			return ""
		}
		return raw + "d"
	}
	if len(raw) < 2 {
		return ""
	}
	suffix := raw[len(raw)-1]
	num := raw[:len(raw)-1]
	n, err := strconv.Atoi(num)
	if err != nil || n <= 0 {
		return ""
	}
	switch suffix {
	case 'd':
		if n > 3650 {
			return ""
		}
	case 'w':
		if n > 520 {
			return ""
		}
	case 'm':
		if n > 120 {
			return ""
		}
	case 'y':
		if n > 10 {
			return ""
		}
	default:
		return ""
	}
	return raw
}

// applyRetention 单节点：改 .env → compose up 重建（使新保留期参数生效）。
func (s *LogCentralService) applyRetention(ctx context.Context, inst VLInstance, period string) error {
	envPath := envPathOf(inst)
	old, _, err := execOnNode(ctx, s.nodes, inst.NodeID, "cat "+envPath+" 2>/dev/null", 15)
	if err != nil {
		return fmt.Errorf("读取 .env 失败: %w", err)
	}
	lines := strings.Split(strings.ReplaceAll(old, "\r\n", "\n"), "\n")
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(line, "RETENTION_PERIOD=") {
			lines[i] = "RETENTION_PERIOD=" + period
			replaced = true
			break
		}
	}
	if !replaced {
		lines = append(lines, "RETENTION_PERIOD="+period)
	}
	script := buildB64WriteScript(envPath, []byte(strings.Join(lines, "\n")))
	if _, _, err := execOnNode(ctx, s.nodes, inst.NodeID, script, 15); err != nil {
		return fmt.Errorf("写入 .env 失败: %w", err)
	}
	projDir := "/opt/ypanel/compose/app-" + strings.TrimPrefix(inst.Container, "app-")
	upOut, upCode, err := execOnNode(ctx, s.nodes, inst.NodeID,
		"cd "+projDir+" && docker compose up -d 2>&1 | tail -3", 180)
	if err != nil || upCode != 0 {
		return fmt.Errorf("重建容器失败: %s", strings.TrimSpace(upOut))
	}
	return nil
}

// ---------- 查询历史 / 常用查询（SQLite） ----------

// logHistoryCap 历史留存上限（超出清理最旧的非置顶记录）。
const logHistoryCap = 200

// History 获取列表：常用（pinned）在前，其余按时间倒序；limit<=0 = 全量。
func (s *LogCentralService) History(limit int) ([]model.LogSearchQuery, error) {
	out := []model.LogSearchQuery{}
	q := s.db.Order("pinned desc, id desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&out).Error; err != nil {
		return nil, errs.Wrap(errs.ErrInternal, err.Error())
	}
	return out, nil
}

// HistoryRecord 搜索成功后记录历史（同条件去重：删旧插新；超上限清理最旧非置顶）。
func (s *LogCentralService) HistoryRecord(item model.LogSearchQuery) (model.LogSearchQuery, error) {
	item.ID = 0
	item.Pinned = false
	if err := s.db.Where("query_text = ? AND keyword = ? AND is_regex = ? AND pinned = ?",
		item.QueryText, item.Keyword, item.IsRegex, false).Delete(&model.LogSearchQuery{}).Error; err != nil {
		return item, errs.Wrap(errs.ErrInternal, err.Error())
	}
	if err := s.db.Create(&item).Error; err != nil {
		return item, errs.Wrap(errs.ErrInternal, err.Error())
	}
	var ids []uint
	if err := s.db.Model(&model.LogSearchQuery{}).Where("pinned = ?", false).
		Order("id desc").Offset(logHistoryCap).Limit(1000).Pluck("id", &ids).Error; err == nil && len(ids) > 0 {
		_ = s.db.Where("id IN ?", ids).Delete(&model.LogSearchQuery{}).Error
	}
	return item, nil
}

// HistoryPin 置顶/取消置顶（置顶即常用查询，可命名）。
func (s *LogCentralService) HistoryPin(id uint, pinned bool, remark string) error {
	updates := map[string]any{"pinned": pinned, "remark": remark}
	if err := s.db.Model(&model.LogSearchQuery{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return errs.Wrap(errs.ErrInternal, err.Error())
	}
	return nil
}

// HistoryDelete 删除单条；all=true 清空非置顶历史。
func (s *LogCentralService) HistoryDelete(id uint, all bool) error {
	if all {
		return s.db.Where("pinned = ?", false).Delete(&model.LogSearchQuery{}).Error
	}
	return s.db.Delete(&model.LogSearchQuery{}, id).Error
}

// parseCentralLines 解析 /select/logsql/query 的 JSON lines 流（按时间升序、限量截断）。
func parseCentralLines(r io.Reader, limit int) (*CentralQueryResult, error) {
	out := &CentralQueryResult{Items: make([]CentralQueryItem, 0, 256)}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var row struct {
			Time   string `json:"_time"`
			Stream string `json:"_stream"`
			Msg    string `json:"_msg"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue // 非 JSON 行（异常输出）跳过
		}
		if len(out.Items) >= limit {
			out.Truncated = true
			break
		}
		out.Items = append(out.Items, CentralQueryItem{Ts: row.Time, Stream: row.Stream, Msg: row.Msg})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out.Items, func(a, b int) bool { return out.Items[a].Ts < out.Items[b].Ts })
	return out, nil
}
