// M49 系统安全工具 API：FTP 管理 / SSH 管理 / Fail2ban SSH 暴力破解防护。
package api

import (
	"fmt"
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// FtpAPI FTP（vsftpd）管理。
type FtpAPI struct {
	Ftp *service.FtpService
}

// Status GET /api/v1/ftp/status
func (a *FtpAPI) Status(c *gin.Context) {
	ftp, ferr := a.Ftp.WithNode(c.Query("nodeId"))
	if ferr != nil {
		respErr(c, ferr)
		return
	}
	out, err := ftp.Status(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Install POST /api/v1/ftp/install
func (a *FtpAPI) Install(c *gin.Context) {
	ftp, ferr := a.Ftp.WithNode(c.Query("nodeId"))
	if ferr != nil {
		respErr(c, ferr)
		return
	}
	out, err := ftp.Install(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Power POST /api/v1/ftp/power {action}
func (a *FtpAPI) Power(c *gin.Context) {
	req, ok := bind[struct {
		Action string `json:"action" binding:"required,oneof=start stop restart"`
	}](c)
	if !ok {
		return
	}
	ftp, ferr := a.Ftp.WithNode(c.Query("nodeId"))
	if ferr != nil {
		respErr(c, ferr)
		return
	}
	if err := ftp.Power(c.Request.Context(), req.Action); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SetPort POST /api/v1/ftp/port {port,pasvMin,pasvMax}
func (a *FtpAPI) SetPort(c *gin.Context) {
	req, ok := bind[struct {
		Port    int `json:"port" binding:"required,min=1024,max=65535"`
		PasvMin int `json:"pasvMin"`
		PasvMax int `json:"pasvMax"`
	}](c)
	if !ok {
		return
	}
	pasvMin, pasvMax := req.PasvMin, req.PasvMax
	if pasvMin == 0 && pasvMax == 0 {
		pasvMin, pasvMax = 40000, 40100
	}
	ftp, ferr := a.Ftp.WithNode(c.Query("nodeId"))
	if ferr != nil {
		respErr(c, ferr)
		return
	}
	if err := ftp.SetPort(c.Request.Context(), req.Port, pasvMin, pasvMax); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// GetConfig GET /api/v1/ftp/config?nodeId=（vsftpd.conf 原文，M53 配置编辑器）
func (a *FtpAPI) GetConfig(c *gin.Context) {
	ftp, ferr := a.Ftp.WithNode(c.Query("nodeId"))
	if ferr != nil {
		respErr(c, ferr)
		return
	}
	out, err := ftp.GetConfig(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"content": out})
}

// PutConfig PUT /api/v1/ftp/config?nodeId= {content}（写回 + 重启校验，失败回滚）
func (a *FtpAPI) PutConfig(c *gin.Context) {
	req, ok := bind[struct {
		Content string `json:"content" binding:"required"`
	}](c)
	if !ok {
		return
	}
	ftp, ferr := a.Ftp.WithNode(c.Query("nodeId"))
	if ferr != nil {
		respErr(c, ferr)
		return
	}
	if err := ftp.PutConfig(c.Request.Context(), req.Content); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SshAPI SSH 安全管理（配置/密钥/暴力破解）。
type SshAPI struct {
	Ssh *service.SshGuardService
	F2B *service.Fail2banService
}

// nodeID 取节点参数（默认 local）。
func nodeID(c *gin.Context) string { return c.DefaultQuery("nodeId", "local") }

// GetConfig GET /api/v1/ssh/config?nodeId=
func (a *SshAPI) GetConfig(c *gin.Context) {
	out, err := a.Ssh.GetConfig(c.Request.Context(), nodeID(c))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SetConfig PUT /api/v1/ssh/config?nodeId=
func (a *SshAPI) SetConfig(c *gin.Context) {
	req, ok := bind[struct {
		Port            *int    `json:"port"`
		PasswordAuth    *bool   `json:"passwordAuth"`
		PubkeyAuth      *bool   `json:"pubkeyAuth"`
		PermitRootLogin *string `json:"permitRootLogin"`
	}](c)
	if !ok {
		return
	}
	nid := nodeID(c)
	if err := a.Ssh.SetConfig(c.Request.Context(), nid, req.Port, req.PasswordAuth, req.PubkeyAuth, req.PermitRootLogin); err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Ssh.GetConfig(c.Request.Context(), nid)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Keys GET /api/v1/ssh/keys?nodeId=（密钥列表，含公钥内容与全节点分发状态）
func (a *SshAPI) Keys(c *gin.Context) {
	out, err := a.Ssh.ListKeys(c.Request.Context(), nodeID(c))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GenerateKey POST /api/v1/ssh/keys/generate {nodeId,type,name,comment}
func (a *SshAPI) GenerateKey(c *gin.Context) {
	req, ok := bind[struct {
		NodeId  string `json:"nodeId"`
		Type    string `json:"type"`
		Name    string `json:"name"`
		Comment string `json:"comment"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Ssh.GenerateKey(c.Request.Context(), req.NodeId, req.Type, req.Name, req.Comment)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ImportKey POST /api/v1/ssh/keys/import {nodeId,name,publicKey}
func (a *SshAPI) ImportKey(c *gin.Context) {
	req, ok := bind[struct {
		NodeId    string `json:"nodeId"`
		Name      string `json:"name"`
		PublicKey string `json:"publicKey" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Ssh.ImportKey(c.Request.Context(), req.NodeId, req.Name, req.PublicKey)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// DeleteKey POST /api/v1/ssh/keys/delete {nodeId,name,withPrivate}
func (a *SshAPI) DeleteKey(c *gin.Context) {
	req, ok := bind[struct {
		NodeId      string `json:"nodeId"`
		Name        string `json:"name" binding:"required"`
		WithPrivate bool   `json:"withPrivate"`
	}](c)
	if !ok {
		return
	}
	if err := a.Ssh.DeleteKey(c.Request.Context(), req.NodeId, req.Name, req.WithPrivate); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// DeployKey POST /api/v1/ssh/keys/deploy {keyNode,name,targetNode}
func (a *SshAPI) DeployKey(c *gin.Context) {
	req, ok := bind[struct {
		KeyNode    string `json:"keyNode"`
		Name       string `json:"name" binding:"required"`
		TargetNode string `json:"targetNode" binding:"required"`
	}](c)
	if !ok {
		return
	}
	result, err := a.Ssh.DeployKey(c.Request.Context(), req.KeyNode, req.Name, req.TargetNode)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"result": result})
}

// UndeployKey POST /api/v1/ssh/keys/undeploy {keyNode,name,targetNode}（从目标节点撤下公钥）
func (a *SshAPI) UndeployKey(c *gin.Context) {
	req, ok := bind[struct {
		KeyNode    string `json:"keyNode"`
		Name       string `json:"name" binding:"required"`
		TargetNode string `json:"targetNode" binding:"required"`
	}](c)
	if !ok {
		return
	}
	result, err := a.Ssh.UndeployKey(c.Request.Context(), req.KeyNode, req.Name, req.TargetNode)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"result": result})
}

// FailedAttempts GET /api/v1/ssh/attempts?nodeId=&limit=（SSH 失败登录聚合，标记已封禁）
func (a *SshAPI) FailedAttempts(c *gin.Context) {
	n := 100
	_, _ = fmt.Sscanf(c.DefaultQuery("limit", "100"), "%d", &n)
	out, err := a.Ssh.FailedAttemptsMarked(c.Request.Context(), nodeID(c), a.F2B, n)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
