package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// VpnAPI EasyTier 组网（工具域）接口。
type VpnAPI struct {
	Vpn *service.VPNService
}

// Status GET /api/v1/vpn/easytier/status
func (a *VpnAPI) Status(c *gin.Context) {
	out, err := a.Vpn.Status(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GetConfig GET /api/v1/vpn/easytier/config（返回期望配置与渲染后的 TOML 全文）
func (a *VpnAPI) GetConfig(c *gin.Context) {
	cfg, err := a.Vpn.GetConfig()
	if err != nil {
		respErr(c, err)
		return
	}
	rendered, _ := a.Vpn.RenderedConfig()
	respOK(c, gin.H{"config": cfg, "rendered": rendered})
}

// SaveConfig PUT /api/v1/vpn/easytier/config（校验 → 持久化 → 写文件 → 重启容器）
func (a *VpnAPI) SaveConfig(c *gin.Context) {
	req, ok := bind[service.VPNConfig](c)
	if !ok {
		return
	}
	out, err := a.Vpn.SaveConfig(c.Request.Context(), req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Install POST /api/v1/vpn/easytier/install（一键安装编排，任务化）
func (a *VpnAPI) Install(c *gin.Context) {
	req, ok := bind[service.VPNInstallInput](c)
	if !ok {
		return
	}
	out, err := a.Vpn.Install(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Apply POST /api/v1/vpn/easytier/apply（手动应用当前期望配置）
func (a *VpnAPI) Apply(c *gin.Context) {
	if err := a.Vpn.ApplyConfig(c.Request.Context(), func(level, format string, args ...any) {}); err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Vpn.Status(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Peers GET /api/v1/vpn/easytier/peers
func (a *VpnAPI) Peers(c *gin.Context) {
	out, err := a.Vpn.Peers(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Routes GET /api/v1/vpn/easytier/routes
func (a *VpnAPI) Routes(c *gin.Context) {
	out, err := a.Vpn.Routes(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
