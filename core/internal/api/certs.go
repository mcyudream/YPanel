// 证书库接口（B23，对齐 1Panel 证书页）：证书 CRUD/签发/上传/续签 + DNS 账户 + ACME 账户。
package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// CertAPI 证书库接口。
type CertAPI struct {
	Certs  *service.CertificateService
	Groups *service.SiteGroupService
}

func certIDParam(c *gin.Context, label string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errBadRequest(label+" ID 不合法"))
		return 0, false
	}
	return uint(id), true
}

// ---- 证书 ----

// List GET /api/v1/certs
func (a *CertAPI) List(c *gin.Context) {
	out, err := a.Certs.List(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Issue POST /api/v1/certs/issue（ACME，长耗时）
func (a *CertAPI) Issue(c *gin.Context) {
	req, ok := bind[service.CertIssueInput](c)
	if !ok {
		return
	}
	out, err := a.Certs.Issue(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Upload POST /api/v1/certs/upload
func (a *CertAPI) Upload(c *gin.Context) {
	req, ok := bind[service.CertUploadInput](c)
	if !ok {
		return
	}
	out, err := a.Certs.Upload(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SelfSigned POST /api/v1/certs/selfsigned
func (a *CertAPI) SelfSigned(c *gin.Context) {
	req, ok := bind[service.CertSelfSignedInput](c)
	if !ok {
		return
	}
	out, err := a.Certs.SelfSigned(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Detail GET /api/v1/certs/:id
func (a *CertAPI) Detail(c *gin.Context) {
	id, ok := certIDParam(c, "证书")
	if !ok {
		return
	}
	out, err := a.Certs.Detail(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Update PUT /api/v1/certs/:id（备注/自动续签）
func (a *CertAPI) Update(c *gin.Context) {
	id, ok := certIDParam(c, "证书")
	if !ok {
		return
	}
	req, ok := bind[service.CertUpdateInput](c)
	if !ok {
		return
	}
	if err := a.Certs.Update(c.Request.Context(), id, *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Renew POST /api/v1/certs/:id/renew（ACME，长耗时）
func (a *CertAPI) Renew(c *gin.Context) {
	id, ok := certIDParam(c, "证书")
	if !ok {
		return
	}
	if err := a.Certs.Renew(c.Request.Context(), id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Remove DELETE /api/v1/certs/:id
func (a *CertAPI) Remove(c *gin.Context) {
	id, ok := certIDParam(c, "证书")
	if !ok {
		return
	}
	if err := a.Certs.Delete(c.Request.Context(), id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ---- DNS 账户 ----

// ListDnsAccounts GET /api/v1/certs/dns-accounts
func (a *CertAPI) ListDnsAccounts(c *gin.Context) {
	out, err := a.Certs.ListDnsAccounts()
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// CreateDnsAccount POST /api/v1/certs/dns-accounts
func (a *CertAPI) CreateDnsAccount(c *gin.Context) {
	req, ok := bind[service.DnsAccountInput](c)
	if !ok {
		return
	}
	out, err := a.Certs.CreateDnsAccount(*req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// UpdateDnsAccount PUT /api/v1/certs/dns-accounts/:id
func (a *CertAPI) UpdateDnsAccount(c *gin.Context) {
	id, ok := certIDParam(c, "DNS 账户")
	if !ok {
		return
	}
	req, ok := bind[service.DnsAccountInput](c)
	if !ok {
		return
	}
	if err := a.Certs.UpdateDnsAccount(id, *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// DeleteDnsAccount DELETE /api/v1/certs/dns-accounts/:id
func (a *CertAPI) DeleteDnsAccount(c *gin.Context) {
	id, ok := certIDParam(c, "DNS 账户")
	if !ok {
		return
	}
	if err := a.Certs.DeleteDnsAccount(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ---- Acme 账户 ----

// ListAcmeAccounts GET /api/v1/certs/acme-accounts
func (a *CertAPI) ListAcmeAccounts(c *gin.Context) {
	out, err := a.Certs.ListAcmeAccounts()
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// CreateAcmeAccount POST /api/v1/certs/acme-accounts
func (a *CertAPI) CreateAcmeAccount(c *gin.Context) {
	req, ok := bind[service.AcmeAccountInput](c)
	if !ok {
		return
	}
	out, err := a.Certs.CreateAcmeAccount(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// DeleteAcmeAccount DELETE /api/v1/certs/acme-accounts/:id
func (a *CertAPI) DeleteAcmeAccount(c *gin.Context) {
	id, ok := certIDParam(c, "ACME 账户")
	if !ok {
		return
	}
	if err := a.Certs.DeleteAcmeAccount(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ---- 站点分组 ----

// ListGroups GET /api/v1/site-groups
func (a *CertAPI) ListGroups(c *gin.Context) {
	out, err := a.Groups.List()
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// CreateGroup POST /api/v1/site-groups {name}
func (a *CertAPI) CreateGroup(c *gin.Context) {
	req, ok := bind[struct {
		Name string `json:"name"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Groups.Create(req.Name)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// UpdateGroup PUT /api/v1/site-groups/:id {name}
func (a *CertAPI) UpdateGroup(c *gin.Context) {
	id, ok := certIDParam(c, "分组")
	if !ok {
		return
	}
	req, ok := bind[struct {
		Name string `json:"name"`
	}](c)
	if !ok {
		return
	}
	if err := a.Groups.Update(id, req.Name); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// DeleteGroup DELETE /api/v1/site-groups/:id
func (a *CertAPI) DeleteGroup(c *gin.Context) {
	id, ok := certIDParam(c, "分组")
	if !ok {
		return
	}
	if err := a.Groups.Delete(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SetDefaultGroup POST /api/v1/site-groups/:id/default
func (a *CertAPI) SetDefaultGroup(c *gin.Context) {
	id, ok := certIDParam(c, "分组")
	if !ok {
		return
	}
	if err := a.Groups.SetDefault(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
