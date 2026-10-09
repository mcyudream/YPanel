package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// AlertAPI 告警接口。
type AlertAPI struct {
	Alerts   *service.AlertService
	Settings *service.SettingService // M37 SMTP 全局配置
}

// ListRules GET /api/v1/alert/rules
func (a *AlertAPI) ListRules(c *gin.Context) {
	respOK(c, a.Alerts.ListRules())
}

// CreateRule POST /api/v1/alert/rules
func (a *AlertAPI) CreateRule(c *gin.Context) {
	req, ok := bind[struct {
		Name        string `json:"name" binding:"required"`
		Metric      string `json:"metric" binding:"required,oneof=cpu memory disk load network cert_expiry site_expiry node_expiry log"`
		Threshold   int    `json:"threshold" binding:"required,min=1"`
		WebhookURL  string `json:"webhookUrl"`
		WebhookType string `json:"webhookType" binding:"required,oneof=feishu dingtalk wecom generic telegram bark email"`
		SilentStart string `json:"silentStart"`
		SilentEnd   string `json:"silentEnd"`
		Query       string `json:"query"`
		WindowSec   int    `json:"windowSec"`
	}](c)
	if !ok {
		return
	}
	row, err := a.Alerts.CreateRule(req.Name, req.Metric, req.Threshold, req.WebhookURL, req.WebhookType, req.SilentStart, req.SilentEnd, req.Query, req.WindowSec)
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
	allowed := map[string]bool{"enabled": true, "threshold": true, "webhookUrl": true, "silentStart": true, "silentEnd": true, "query": true, "windowSec": true}
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

// GetSMTPSettings GET /api/v1/alert/smtp（M37：密码不回显）
func (a *AlertAPI) GetSMTPSettings(c *gin.Context) {
	cfg := service.LoadSMTPConfig(a.Settings)
	respOK(c, gin.H{
		"host": cfg.Host, "port": cfg.Port, "user": cfg.User,
		"from": cfg.From, "ssl": cfg.SSL, "to": cfg.ToAddr,
		"hasPass": cfg.Pass != "",
	})
}

// PutSMTPSettings PUT /api/v1/alert/smtp {host,port,user,pass(空=保留),from,ssl,to}
func (a *AlertAPI) PutSMTPSettings(c *gin.Context) {
	req, ok := bind[struct {
		Host string `json:"host"`
		Port int    `json:"port"`
		User string `json:"user"`
		Pass string `json:"pass"`
		From string `json:"from"`
		SSL  bool   `json:"ssl"`
		To   string `json:"to"`
	}](c)
	if !ok {
		return
	}
	set := func(k, v string) { _ = a.Settings.Set("smtp."+k, v) }
	set("host", req.Host)
	set("port", strconv.Itoa(req.Port))
	set("user", req.User)
	if req.Pass != "" {
		set("pass", req.Pass)
	}
	set("from", req.From)
	set("ssl", map[bool]string{true: "true", false: "false"}[req.SSL])
	set("to", req.To)
	respOK(c, struct{}{})
}

// TestSMTP POST /api/v1/alert/smtp/test {to?}（M37：测试发送）
func (a *AlertAPI) TestSMTP(c *gin.Context) {
	req, ok := bind[struct {
		To string `json:"to"`
	}](c)
	if !ok {
		return
	}
	cfg := service.LoadSMTPConfig(a.Settings)
	if req.To != "" {
		cfg.ToAddr = req.To
	}
	if err := service.SendMail(c.Request.Context(), cfg, "YPanel SMTP 测试", "这是一封来自 YPanel 的测试邮件，收到即说明告警邮件通道配置正确。"); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"ok": true})
}
