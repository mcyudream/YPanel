package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
)

// NotificationAPI 通知中心接口。
type NotificationAPI struct {
	Notif      *service.NotificationService
	ParseToken func(string) error // SSE query token 校验（EventSource 无法带 header）
}

// Stream GET /api/v1/notifications/stream?token=（SSE 实时推送：notification / unread 两类事件，25s 心跳）。
func (a *NotificationAPI) Stream(c *gin.Context) {
	if a.ParseToken == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if err := a.ParseToken(c.Query("token")); err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	ch := a.Notif.Subscribe()
	defer a.Notif.Unsubscribe(ch)
	h := c.Writer.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	// 连接建立即推送当前未读数
	if b, err := json.Marshal(map[string]any{"type": "unread", "count": a.Notif.UnreadCount()}); err == nil {
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", b)
		flusher.Flush()
	}
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", msg); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

// List GET /api/v1/notifications?limit=
func (a *NotificationAPI) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	respOK(c, a.Notif.List(limit))
}

// Unread GET /api/v1/notifications/unread
func (a *NotificationAPI) Unread(c *gin.Context) {
	respOK(c, gin.H{"count": a.Notif.UnreadCount()})
}

// MarkRead POST /api/v1/notifications/read?id=（id 空 = 全部）
func (a *NotificationAPI) MarkRead(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Query("id"), 10, 64)
	if err := a.Notif.MarkRead(uint(id)); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// PanelBackupAPI 面板备份接口。
type PanelBackupAPI struct {
	BP *service.PanelBackupService
}

// List GET /api/v1/panel/backups
func (a *PanelBackupAPI) List(c *gin.Context) {
	out, err := a.BP.List(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Create POST /api/v1/panel/backups（body 可选 {storageAccountId,keep}）
func (a *PanelBackupAPI) Create(c *gin.Context) {
	var opts service.BackupUploadOpts
	if c.Request.ContentLength > 0 {
		var req struct {
			StorageAccountId uint `json:"storageAccountId"`
			Keep             int  `json:"keep"`
		}
		if berr := c.ShouldBindJSON(&req); berr == nil {
			opts = service.BackupUploadOpts{StorageAccountID: req.StorageAccountId, Keep: req.Keep}
		}
	}
	out, err := a.BP.Create(c.Request.Context(), opts)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Delete DELETE /api/v1/panel/backups?file=
func (a *PanelBackupAPI) Delete(c *gin.Context) {
	if err := a.BP.Delete(c.Request.Context(), c.Query("file")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// RestoreHint GET /api/v1/panel/backups/restore-hint
func (a *PanelBackupAPI) RestoreHint(c *gin.Context) {
	respOK(c, gin.H{"hint": service.RestoreHint})
}

// AuditAPI 操作审计查询。
type AuditAPI struct {
	DB   *gorm.DB
	Hist *service.HistoryRecorder
}

// List GET /api/v1/audit/ops?page=&pageSize=
func (a *AuditAPI) List(c *gin.Context) {
	page, size := pageParams(c)
	var total int64
	var rows []model.AuditLog
	q := a.DB.Model(&model.AuditLog{})
	if err := q.Count(&total).Error; err != nil {
		respErr(c, err)
		return
	}
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		respErr(c, err)
		return
	}
	respOK(c, dto.NewPageResp(total, rows))
}

// HistoryDB GET /api/v1/system/history/persisted?seconds=&node=（历史监控持久化查询，全节点）
func (a *AuditAPI) History(c *gin.Context) {
	seconds, _ := strconv.Atoi(c.DefaultQuery("seconds", "3600"))
	// M37：超过原始保留期（30 天）自动切换小时聚合（MetricHourly，365 天）
	if seconds > 30*24*3600 {
		respOK(c, a.Hist.QueryHourly(c.Request.Context(), seconds, c.DefaultQuery("node", "local")))
		return
	}
	respOK(c, a.Hist.Query(c.Request.Context(), seconds, c.DefaultQuery("node", "local")))
}
