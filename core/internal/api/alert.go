package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// AlertAPI 告警接口。
type AlertAPI struct {
	Alerts *service.AlertService
}

// ListRules GET /api/v1/alert/rules
func (a *AlertAPI) ListRules(c *gin.Context) {
	respOK(c, a.Alerts.ListRules())
}

// CreateRule POST /api/v1/alert/rules
func (a *AlertAPI) CreateRule(c *gin.Context) {
	req, ok := bind[struct {
		Name        string `json:"name" binding:"required"`
		Metric      string `json:"metric" binding:"required,oneof=cpu memory disk"`
		Threshold   int    `json:"threshold" binding:"required,min=1,max=100"`
		WebhookURL  string `json:"webhookUrl" binding:"required"`
		WebhookType string `json:"webhookType" binding:"required,oneof=feishu dingtalk wecom generic"`
	}](c)
	if !ok {
		return
	}
	row, err := a.Alerts.CreateRule(req.Name, req.Metric, req.Threshold, req.WebhookURL, req.WebhookType)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// UpdateRule PUT /api/v1/alert/rules/:id
func (a *AlertAPI) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[map[string]any](c)
	if !ok {
		return
	}
	allowed := map[string]bool{"enabled": true, "threshold": true, "webhookUrl": true}
	for k := range *req {
		if !allowed[k] {
			respErr(c, errBadRequest("字段不允许更新: "+k))
			return
		}
	}
	if err := a.Alerts.UpdateRule(uint(id), *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// DeleteRule DELETE /api/v1/alert/rules/:id
func (a *AlertAPI) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.Alerts.DeleteRule(uint(id)); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
