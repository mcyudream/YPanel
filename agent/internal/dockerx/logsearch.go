package dockerx

// M33 日志中心 P1：多容器日志服务端搜索。
// 并发逐容器拉取日志流（Timestamps + Since/Until 有界区间），行级正则过滤，
// 每容器环形缓冲保"最后 N 条命中"，跨容器按时间戳归并有界输出。

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

const (
	SearchMaxContainers = 50   // 单次搜索容器数上限
	searchMaxPatternLen = 256  // 正则长度上限
	searchPerLimitDef   = 2000 // 单容器默认保留条数
	searchPerLimitMax   = 10000
	searchTotalLimitDef = 10000 // 归并后默认总量上限
	searchTotalLimitMax = 50000
	searchConcurrency   = 8 // 单容器流并发数
)

// levelPat 从日志行内容探测级别 token（词边界，避免 "notification" 误中 notice）。
var levelPat = regexp.MustCompile(`(?i)\b(trace|debug|info|notice|warn(?:ing)?|err(?:or)?|fatal|crit(?:ical)?|panic)\b`)

// DetectLevel 归一化级别标签；空串 = 未知。
func DetectLevel(content string) string {
	m := levelPat.FindString(content)
	if m == "" {
		return ""
	}
	switch strings.ToLower(m) {
	case "warning", "warn":
		return "WARN"
	case "error", "err":
		return "ERROR"
	case "fatal", "critical", "crit", "panic":
		return "FATAL"
	default:
		return strings.ToUpper(m)
	}
}

// Search 多容器日志搜索（结果有界：单容器 perLimit + 归并 totalLimit，均保最新）。
// 单容器失败不阻断整体，错误按容器标注进 Errors。
func (m *Manager) Search(ctx context.Context, req *dto.LogsSearchReq) (*dto.LogsSearchResp, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	if len(req.Containers) == 0 {
		return nil, badReq("未选择容器")
	}
	if len(req.Containers) > SearchMaxContainers {
		return nil, badReq("单次搜索容器数不能超过 " + strconv.Itoa(SearchMaxContainers))
	}
	if !validTimeArg(req.Since) || !validTimeArg(req.Until) {
		return nil, badReq("时间范围格式不合法（支持 RFC3339 / unix 秒 / 相对时长如 15m）")
	}
	re, err := compilePattern(req.Pattern)
	if err != nil {
		return nil, err
	}
	perLimit := clampLimit(req.PerLimit, searchPerLimitDef, searchPerLimitMax)
	totalLimit := clampLimit(req.TotalLimit, searchTotalLimitDef, searchTotalLimitMax)

	type jobResult struct {
		hits    []logHit
		scanned int64
		errMsg  string
	}
	results := make([]jobResult, len(req.Containers))
	sem := make(chan struct{}, searchConcurrency)
	var wg sync.WaitGroup
	for i, id := range req.Containers {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			hits, scanned, serr := m.searchOne(ctx, cli, id, req, re, perLimit)
			res := jobResult{hits: hits, scanned: scanned}
			if serr != nil {
				res.errMsg = id + ": " + serr.Error()
			}
			results[i] = res
		}(i, id)
	}
	wg.Wait()

	resp := &dto.LogsSearchResp{Items: make([]dto.LogSearchItem, 0)}
	all := make([]logHit, 0, 1024)
	for i := range results {
		resp.Scanned += results[i].scanned
		if results[i].errMsg != "" {
			resp.Errors = append(resp.Errors, results[i].errMsg)
		}
		all = append(all, results[i].hits...)
	}
	// (ts, seq) 升序归并；超总量保最新
	sort.Slice(all, func(a, b int) bool {
		if !all[a].ts.Equal(all[b].ts) {
			return all[a].ts.Before(all[b].ts)
		}
		return all[a].seq < all[b].seq
	})
	if len(all) > totalLimit {
		all = all[len(all)-totalLimit:]
		resp.Truncated = true
	}
	for _, h := range all {
		resp.Items = append(resp.Items, h.item)
	}
	return resp, nil
}

// logHit 单条命中（ts/seq 用于跨容器归并排序）。
type logHit struct {
	ts   time.Time
	seq  uint64
	item dto.LogSearchItem
}

func (m *Manager) searchOne(ctx context.Context, cli *client.Client, id string, req *dto.LogsSearchReq, re *regexp.Regexp, perLimit int) ([]logHit, int64, error) {
	ins, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return nil, 0, err
	}
	name := strings.TrimPrefix(ins.Container.Name, "/")
	tty := ins.Container.Config != nil && ins.Container.Config.Tty

	opts := client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
		Timestamps: true,
		Since:      req.Since,
		Until:      req.Until,
	}
	reader, err := cli.ContainerLogs(ctx, id, opts)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = reader.Close() }()

	lp := &lineParser{limit: perLimit, re: re, negate: req.Negate}
	var cerr error
	if tty {
		// TTY 容器无帧头，原样文本流
		_, cerr = io.Copy(lp, reader)
	} else if _, e := stdcopy.StdCopy(lp, lp, reader); e != nil {
		cerr = e
	}
	if cerr != nil && ctx.Err() == nil {
		return nil, lp.scanned, cerr
	}
	lp.flush()

	hits := lp.hits()
	for i := range hits {
		hits[i].item.Container = name
		hits[i].item.ID = shortID(id)
	}
	return hits, lp.scanned, nil
}

// lineParser 帧内行级解析：剥时间戳 → 正则过滤 → 级别探测 → 环形缓冲（保最后 limit 条）。
type lineParser struct {
	limit   int
	re      *regexp.Regexp
	negate  bool
	ring    []logHit
	pos     int // ring 满后下一个覆盖位置（即最旧条目）
	buf     []byte
	lastTs  time.Time
	seq     uint64
	scanned int64
}

func (p *lineParser) Write(b []byte) (int, error) {
	total := len(b)
	for {
		idx := bytes.IndexByte(b, '\n')
		if idx < 0 {
			p.buf = append(p.buf, b...)
			return total, nil
		}
		var line []byte
		if len(p.buf) > 0 {
			line = append(p.buf, b[:idx]...)
			p.buf = p.buf[:0]
		} else {
			line = b[:idx]
		}
		p.feed(string(line))
		b = b[idx+1:]
	}
}

// flush 处理流尾无换行的最后一行。
func (p *lineParser) flush() {
	if len(p.buf) > 0 {
		line := string(p.buf)
		p.buf = p.buf[:0]
		p.feed(line)
	}
}

func (p *lineParser) feed(line string) {
	if line == "" {
		return
	}
	p.scanned++
	ts, content := splitTimestamp(line, p.lastTs)
	p.lastTs = ts
	if p.re != nil && p.re.MatchString(content) == p.negate {
		// negate=false：未命中丢弃；negate=true：命中即排除
		return
	}
	limit := p.limit
	if limit <= 0 {
		limit = searchPerLimitDef
	}
	hit := logHit{
		ts:  ts,
		seq: p.seq,
		item: dto.LogSearchItem{
			Ts:    ts,
			Level: DetectLevel(content),
			Line:  content,
		},
	}
	p.seq++
	if len(p.ring) < limit {
		p.ring = append(p.ring, hit)
		return
	}
	p.ring[p.pos] = hit
	p.pos = (p.pos + 1) % limit
}

// hits 按逻辑顺序（旧→新）物化环形缓冲。
func (p *lineParser) hits() []logHit {
	out := make([]logHit, len(p.ring))
	if p.pos == 0 || p.pos >= len(p.ring) {
		copy(out, p.ring)
		return out
	}
	copy(out, p.ring[p.pos:])
	copy(out[len(p.ring)-p.pos:], p.ring[:p.pos])
	return out
}

// splitTimestamp 剥离行首 docker 时间戳（RFC3339Nano，含时区偏移）；无时间戳行继承 lastTs。
func splitTimestamp(line string, lastTs time.Time) (time.Time, string) {
	i := strings.IndexByte(line, ' ')
	if i > 0 {
		if ts, err := time.Parse(time.RFC3339Nano, line[:i]); err == nil {
			return ts, strings.TrimLeft(line[i+1:], " ")
		}
	}
	return lastTs, line
}

// validTimeArg 时间参数合法性（透传 docker 前的快速校验）。
func validTimeArg(s string) bool {
	if s == "" {
		return true
	}
	if len(s) > 64 {
		return false
	}
	if _, err := strconv.ParseInt(s, 10, 64); err == nil { // unix 秒
		return true
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	_, err := time.ParseDuration(s) // 相对时长：15m / 1h30m
	return err == nil
}

func compilePattern(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	if len(pattern) > searchMaxPatternLen {
		return nil, badReq("正则表达式过长（上限 " + strconv.Itoa(searchMaxPatternLen) + " 字符）")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, badReq("正则表达式无效: " + err.Error())
	}
	return re, nil
}

func clampLimit(v, def, max int) int {
	switch {
	case v <= 0:
		return def
	case v > max:
		return max
	default:
		return v
	}
}

func badReq(msg string) error {
	return errs.New(errs.CodeBadRequest, "error.badRequest", msg)
}
