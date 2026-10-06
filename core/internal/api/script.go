// ScriptAPI 脚本库接口（B13）。
package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// ScriptAPI 脚本库。
type ScriptAPI struct {
	Scripts *service.ScriptService
}

// List GET /api/v1/scripts
func (a *ScriptAPI) List(c *gin.Context) {
	respOK(c, a.Scripts.List())
}

// Create POST /api/v1/scripts {name, content}
func (a *ScriptAPI) Create(c *gin.Context) {
	req, ok := bind[struct {
		Name    string `json:"name" binding:"required"`
		Content string `json:"content" binding:"required"`
	}](c)
	if !ok {
		return
	}
	row, err := a.Scripts.Create(req.Name, req.Content)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// Update PUT /api/v1/scripts/:id {name, content}
func (a *ScriptAPI) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errBadRequest("脚本 ID 不合法"))
		return
	}
	req, ok := bind[struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}](c)
	if !ok {
		return
	}
	row, err := a.Scripts.Update(uint(id), req.Name, req.Content)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// Delete DELETE /api/v1/scripts/:id
func (a *ScriptAPI) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errBadRequest("脚本 ID 不合法"))
		return
	}
	if err := a.Scripts.Delete(uint(id)); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
