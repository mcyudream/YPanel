package api

// M33 日志中心 P1：容器日志跨节点并发聚合搜索。
// nodes 为空时按 ?node= 单节点查询；多节点并发 fan-out 到各 agent，
// 结果按时间戳升序归并，超总量保最新；单节点失败不阻断整体。

import (
	"context"
	"net/http"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

const (
	logsSearchMaxNodes  = 20
	logsSearchTimeout   = 60 * time.Second
	logsSearchTotalMax  = 50000
	logsSearchTotalDef  = 10000
	logsSearchMaxPatten = 256
)

// LogsAPI 日志中心接口。
type LogsAPI struct {
	Nodes *service.NodeService
}

// logsSearchBody 在共享 DTO 上扩展节点集合。
type logsSearchBody struct {
	dto.LogsSearchReq
	Nodes []string `json:"nodes"`
}

// Search POST /api/v1/logs/search
func (l *LogsAPI) Search(c *gin.Context) {
	body, ok := bind[logsSearchBody](c)
	if !ok {
		return
	}
	if len(body.Containers) == 0 {
		respErr(c, logsBadReq("未选择容器"))
		return
	}
	// 正则本地预编译：快速失败，避免向 N 个节点扇出同样的坏参数
	if len(body.Pattern) > logsSearchMaxPatten {
		respErr(c, logsBadReq("正则表达式过长（上限 256 字符）"))
		return
	}
	if body.Pattern != "" {
		if _, err := regexp.Compile(body.Pattern); err != nil {
			respErr(c, logsBadReq("正则表达式无效: "+err.Error()))
			return
		}
	}

	targets, err := l.resolveNodes(c, body.Nodes)
	if err != nil {
		respErr(c, err)
		return
	}

	results := make([]nodeSearchResult, len(targets))
	var wg sync.WaitGroup
	for i, n := range targets {
		wg.Add(1)
		go func(i int, n *service.Node) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(c.Request.Context(), logsSearchTimeout)
			defer cancel()
			resp, err := agentclient.DoJSON[dto.LogsSearchReq, dto.LogsSearchResp](
				agentclient.New(n.BaseURL, n.Token), ctx,
				http.MethodPost, "/agent/v1/docker/logs/search", &body.LogsSearchReq)
			if err != nil {
				// 单节点失败不阻断整体：错误以节点名标注进结果
				results[i] = nodeSearchResult{Name: n.Name, Resp: &dto.LogsSearchResp{
					Items:  make([]dto.LogSearchItem, 0),
					Errors: []string{"节点 " + n.Name + ": " + errs.From(err).Message},
				}}
				return
			}
			results[i] = nodeSearchResult{Name: n.Name, Resp: resp}
			// 归并侧为命中行标注来源节点（跨节点同名容器可区分）
			for j := range resp.Items {
				resp.Items[j].Node = n.Name
			}
		}(i, n)
	}
	wg.Wait()

	respOK(c, mergeLogSearch(results, body.TotalLimit))
}

// resolveNodes 解析目标节点集合（去重、上限校验）。
func (l *LogsAPI) resolveNodes(c *gin.Context, ids []string) ([]*service.Node, error) {
	if len(ids) == 0 {
		node, err := l.Nodes.ByID(c.DefaultQuery("node", "local"))
		if err != nil {
			return nil, err
		}
		return []*service.Node{node}, nil
	}
	if len(ids) > logsSearchMaxNodes {
		return nil, logsBadReq("单次查询节点数不能超过 20")
	}
	seen := make(map[string]bool, len(ids))
	targets := make([]*service.Node, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		node, err := l.Nodes.ByID(id)
		if err != nil {
			return nil, err
		}
		targets = append(targets, node)
	}
	return targets, nil
}

// nodeSearchResult 跨包测试可见的节点结果载体。
type nodeSearchResult struct {
	Name string
	Resp *dto.LogsSearchResp
}

// mergeLogSearch 跨节点归并：按时间戳升序稳定合并，超总量保最新，截断与错误透出。
func mergeLogSearch(results []nodeSearchResult, totalLimit int) *dto.LogsSearchResp {
	switch {
	case totalLimit <= 0:
		totalLimit = logsSearchTotalDef
	case totalLimit > logsSearchTotalMax:
		totalLimit = logsSearchTotalMax
	}
	merged := &dto.LogsSearchResp{Items: make([]dto.LogSearchItem, 0)}
	all := make([]dto.LogSearchItem, 0, 1024)
	for _, r := range results {
		if r.Resp == nil {
			continue
		}
		if r.Resp.Truncated {
			merged.Truncated = true
		}
		merged.Scanned += r.Resp.Scanned
		merged.Errors = append(merged.Errors, r.Resp.Errors...)
		all = append(all, r.Resp.Items...)
	}
	// gin 已统一 UTC 序列化时间戳；跨节点时钟偏差内的次序以稳定排序兜底
	sort.SliceStable(all, func(a, b int) bool { return all[a].Ts.Before(all[b].Ts) })
	if len(all) > totalLimit {
		all = all[len(all)-totalLimit:]
		merged.Truncated = true
	}
	merged.Items = all
	return merged
}

func logsBadReq(msg string) error {
	return errs.New(errs.CodeBadRequest, "error.badRequest", msg)
}
