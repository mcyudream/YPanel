// AIAPI AI 助手接口（B18）：供应商 CRUD + SSE 流式对话。
package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/service"
)

// AIAPI AI 助手。
type AIAPI struct {
	AI *service.AIService
}

func aiIDParam(c *gin.Context) uint {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0
	}
	return uint(id)
}

// ListProviders GET /api/v1/ai/providers
func (a *AIAPI) ListProviders(c *gin.Context) {
	respOK(c, a.AI.ListProviders())
}

// Presets GET /api/v1/ai/presets
func (a *AIAPI) Presets(c *gin.Context) {
	respOK(c, a.AI.Presets())
}

// SaveProvider POST /api/v1/ai/providers
func (a *AIAPI) SaveProvider(c *gin.Context) {
	req, ok := bind[service.AIProvider](c)
	if !ok {
		return
	}
	row := model.AIProvider{
		ID: req.ID, Name: req.Name, APIType: req.APIType, BaseURL: req.BaseURL,
		APIKey: req.APIKey, Model: req.Model, IsDefault: req.IsDefault,
	}
	if err := a.AI.SaveProvider(&row); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, a.AI.ListProviders())
}

// DeleteProvider DELETE /api/v1/ai/providers/:id
func (a *AIAPI) DeleteProvider(c *gin.Context) {
	if id := aiIDParam(c); id == 0 {
		respErr(c, errBadRequest("供应商 ID 不合法"))
		return
	} else if err := a.AI.DeleteProvider(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Chat POST /api/v1/ai/chat {providerId?, messages:[{role,content}]}（SSE 流式）
func (a *AIAPI) Chat(c *gin.Context) {
	req, ok := bind[struct {
		ProviderID uint                  `json:"providerId"`
		Messages   []service.ChatMessage `json:"messages"`
	}](c)
	if !ok {
		return
	}
	var provider *service.AIProviderModel
	var err error
	if req.ProviderID > 0 {
		provider, err = a.AI.ProviderByID(req.ProviderID)
	} else {
		provider, err = a.AI.DefaultProvider()
	}
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.AI.StreamChat(c.Request.Context(), c.Writer, provider, req.Messages); err != nil {
		c.SSEvent("error", gin.H{"message": err.Error()})
	}
}
