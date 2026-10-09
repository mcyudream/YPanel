// GitCredAPI 私有仓库凭据接口（M26 P2）。secret 一律不回传（model json:"-"）。
package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

type GitCredAPI struct {
	Creds *service.GitCredService
}

// List GET /api/v1/git/credentials
func (g *GitCredAPI) List(ctx *gin.Context) {
	out, err := g.Creds.List(ctx.Request.Context())
	if err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, out)
}

// Create POST /api/v1/git/credentials {name,type,host,username,secret,remark}
func (g *GitCredAPI) Create(ctx *gin.Context) {
	req, ok := bind[struct {
		Name    string `json:"name" binding:"required"`
		Type    string `json:"type" binding:"required,oneof=token ssh"`
		Host    string `json:"host" binding:"required"`
		Username string `json:"username"`
		Secret  string `json:"secret" binding:"required"`
		Remark  string `json:"remark"`
	}](ctx)
	if !ok {
		return
	}
	out, err := g.Creds.Create(ctx.Request.Context(), req.Name, req.Type, req.Host, req.Username, req.Secret, req.Remark)
	if err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, out)
}

// Update PUT /api/v1/git/credentials/:id（secret 留空表示保留原值）
func (g *GitCredAPI) Update(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		respErr(ctx, errBadRequest("id 不合法"))
		return
	}
	req, ok := bind[struct {
		Name     *string `json:"name"`
		Host     *string `json:"host"`
		Username *string `json:"username"`
		Secret   *string `json:"secret"`
		Remark   *string `json:"remark"`
	}](ctx)
	if !ok {
		return
	}
	if err := g.Creds.Update(ctx.Request.Context(), uint(id), req.Name, req.Host, req.Username, req.Secret, req.Remark); err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, struct{}{})
}

// Delete DELETE /api/v1/git/credentials/:id
func (g *GitCredAPI) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		respErr(ctx, errBadRequest("id 不合法"))
		return
	}
	if err := g.Creds.Delete(ctx.Request.Context(), uint(id)); err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, struct{}{})
}

// Match GET /api/v1/git/credentials/match?url=（向导预览将命中的凭据）
func (g *GitCredAPI) Match(ctx *gin.Context) {
	out, err := g.Creds.Match(ctx.Request.Context(), ctx.Query("url"))
	if err != nil {
		respErr(ctx, err)
		return
	}
	if out == nil {
		respOK[any](ctx, nil)
		return
	}
	respOK(ctx, gin.H{"id": out.ID, "name": out.Name, "type": out.Type, "host": out.Host})
}
