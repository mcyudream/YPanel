package api

// M33 日志中心：集中存储（VictoriaLogs）多节点聚合接口。
// 地址全自动管理（agent 按镜像发现本节点 VL），无手工配置；历史/常用查询 SQLite 持久化。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
	"github.com/ypanel/core/internal/service"
)

// LogCentralAPI 集中日志接口。
type LogCentralAPI struct {
	LC *service.LogCentralService
}

// GetStatus GET /api/v1/logs/central/status → 各节点 VL 自动发现状态
func (l *LogCentralAPI) GetStatus(c *gin.Context) {
	respOK(c, gin.H{"instances": l.LC.Status(c.Request.Context())})
}

// GetRetention GET /api/v1/logs/central/retention → 各节点 VL 当前保留期
func (l *LogCentralAPI) GetRetention(c *gin.Context) {
	respOK(c, l.LC.Retentions(c.Request.Context()))
}

type centralRetentionReq struct {
	Period string `json:"period"`
}

// SetRetention PUT /api/v1/logs/central/retention {period} → 统一设置并重建 VL 生效
func (l *LogCentralAPI) SetRetention(c *gin.Context) {
	req, ok := bind[centralRetentionReq](c)
	if !ok {
		return
	}
	applied, failures := l.LC.SetRetention(c.Request.Context(), req.Period)
	respOK(c, gin.H{"applied": applied, "failures": failures})
}

type centralQueryReq struct {
	Query  string `json:"query"`
	Start  string `json:"start"`
	End    string `json:"end"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// Query POST /api/v1/logs/central/query {query,start,end,limit,offset} → 多节点聚合
func (l *LogCentralAPI) Query(c *gin.Context) {
	req, ok := bind[centralQueryReq](c)
	if !ok {
		return
	}
	out, err := l.LC.Query(c.Request.Context(), service.CentralQuery{
		Query: req.Query, Start: req.Start, End: req.End, Limit: req.Limit, Offset: req.Offset,
	})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

type centralStreamsReq struct {
	Field string `json:"field"`
	Query string `json:"query"`
	Start string `json:"start"`
	End   string `json:"end"`
	Limit int    `json:"limit"`
}

// StreamValues POST /api/v1/logs/central/streams {field,query,start,end,limit} → 跨节点合并
func (l *LogCentralAPI) StreamValues(c *gin.Context) {
	req, ok := bind[centralStreamsReq](c)
	if !ok {
		return
	}
	out, err := l.LC.StreamValues(c.Request.Context(), req.Field, req.Query, req.Start, req.End, req.Limit)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

type centralHitsReq struct {
	Query string `json:"query"`
	Start string `json:"start"`
	End   string `json:"end"`
	Step  string `json:"step"`
}

// Hits POST /api/v1/logs/central/hits {query,start,end,step} → 聚合直方图 + 命中总数
func (l *LogCentralAPI) Hits(c *gin.Context) {
	req, ok := bind[centralHitsReq](c)
	if !ok {
		return
	}
	out, err := l.LC.Hits(c.Request.Context(), req.Query, req.Start, req.End, req.Step)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// History GET /api/v1/logs/central/history?limit=
func (l *LogCentralAPI) History(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	out, err := l.LC.History(limit)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

type centralHistoryReq struct {
	QueryText      string     `json:"queryText"`
	ContainersJSON string     `json:"containersJson"`
	Keyword        string     `json:"keyword"`
	IsRegex        bool       `json:"isRegex"`
	RangeType      string     `json:"rangeType"`
	StartAt        *time.Time `json:"startAt"`
	EndAt          *time.Time `json:"endAt"`
}

// HistoryRecord POST /api/v1/logs/central/history（搜索成功后记录）
func (l *LogCentralAPI) HistoryRecord(c *gin.Context) {
	req, ok := bind[centralHistoryReq](c)
	if !ok {
		return
	}
	item, err := l.LC.HistoryRecord(model.LogSearchQuery{
		QueryText: req.QueryText, ContainersJSON: req.ContainersJSON, Keyword: req.Keyword,
		IsRegex: req.IsRegex, RangeType: req.RangeType, StartAt: req.StartAt, EndAt: req.EndAt,
	})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, item)
}

type centralHistoryPinReq struct {
	Pinned bool   `json:"pinned"`
	Remark string `json:"remark"`
}

// HistoryPin PUT /api/v1/logs/central/history/:id/pin {pinned,remark}
func (l *LogCentralAPI) HistoryPin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errs.New(errs.CodeBadRequest, "error.badRequest", "ID 不合法"))
		return
	}
	req, ok := bind[centralHistoryPinReq](c)
	if !ok {
		return
	}
	if err := l.LC.HistoryPin(uint(id), req.Pinned, req.Remark); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// HistoryDelete DELETE /api/v1/logs/central/history/:id?all=1
func (l *LogCentralAPI) HistoryDelete(c *gin.Context) {
	if c.Query("all") == "1" {
		if err := l.LC.HistoryDelete(0, true); err != nil {
			respErr(c, err)
			return
		}
		respOK(c, struct{}{})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errs.New(errs.CodeBadRequest, "error.badRequest", "ID 不合法"))
		return
	}
	if err := l.LC.HistoryDelete(uint(id), false); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
