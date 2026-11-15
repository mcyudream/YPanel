package api

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/service"
)

func aiQueryInt(c *gin.Context, key string) int { v, _ := strconv.Atoi(c.Query(key)); return v }
func (a *AIAPI) ChatDefaults(c *gin.Context)    { respOK(c, a.AI.AIChatDefaults(callerCtx(c))) }
func (a *AIAPI) SaveChatDefaults(c *gin.Context) {
	v, ok := bind[service.AIChatSettings](c)
	if !ok {
		return
	}
	if err := a.AI.SaveAIChatDefaults(callerCtx(c), *v); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, v)
}
func (a *AIAPI) UpdateConversation(c *gin.Context) {
	id := aiIDParam(c)
	if id == 0 {
		respErr(c, errBadRequest("会话 ID 无效"))
		return
	}
	v, ok := bind[struct {
		Title    *string                 `json:"title"`
		Pinned   *bool                   `json:"pinned"`
		Archived *bool                   `json:"archived"`
		Settings *service.AIChatSettings `json:"settings"`
	}](c)
	if !ok {
		return
	}
	if err := a.AI.UpdateConversation(callerCtx(c), id, v.Title, v.Pinned, v.Archived, v.Settings); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": id})
}
func (a *AIAPI) ForkConversation(c *gin.Context) {
	id := aiIDParam(c)
	if id == 0 {
		respErr(c, errBadRequest("会话 ID 无效"))
		return
	}
	v, ok := bind[struct {
		Through string `json:"through"`
		Include bool   `json:"include"`
	}](c)
	if !ok {
		return
	}
	result, err := a.AI.ForkConversation(callerCtx(c), id, v.Through, v.Include)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": result})
}
func (a *AIAPI) CompactConversation(c *gin.Context) {
	id := aiIDParam(c)
	if id == 0 {
		respErr(c, errBadRequest("会话 ID 无效"))
		return
	}
	if err := a.AI.CompactConversation(callerCtx(c), id); err != nil {
		respErr(c, err)
		return
	}
	out, err := a.AI.ConversationSnapshot(callerCtx(c), id, 0, 50)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
func (a *AIAPI) UploadChatAttachment(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 21<<20)
	if err := c.Request.ParseMultipartForm(2 << 20); err != nil {
		respErr(c, errBadRequest("上传内容过大或格式无效（单文件上限 20MB）"))
		return
	}
	if c.Request.MultipartForm != nil {
		defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	}
	fh, err := c.FormFile("file")
	if err != nil {
		respErr(c, errBadRequest("缺少文件"))
		return
	}
	f, err := fh.Open()
	if err != nil {
		respErr(c, err)
		return
	}
	defer func() { _ = f.Close() }()
	out, err := a.AI.UploadChatAttachment(callerCtx(c), fh.Filename, f)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
func (a *AIAPI) DownloadChatAttachment(c *gin.Context) {
	row, path, err := a.AI.AttachmentFile(callerCtx(c), c.Param("id"))
	if err != nil {
		respErr(c, err)
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, no-store")
	disposition := "attachment"
	if row.Kind == "image" {
		disposition = "inline"
	}
	c.Header("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": row.Name}))
	c.Header("Content-Type", row.MIME)
	c.File(path)
}
func (a *AIAPI) DeleteChatAttachment(c *gin.Context) {
	if err := a.AI.DeleteChatAttachment(callerCtx(c), c.Param("id")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

func (a *AIAPI) SaveMemory(c *gin.Context) {
	v, ok := bind[model.AIMemory](c)
	if !ok {
		return
	}
	if err := a.AI.SaveMemory(callerCtx(c), *v); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
func (a *AIAPI) DeleteMemory(c *gin.Context) {
	id := aiIDParam(c)
	if id == 0 {
		respErr(c, errBadRequest("记忆 ID 无效"))
		return
	}
	if err := a.AI.DeleteMemory(callerCtx(c), id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

func (a *AIAPI) ShareKnowledgeDoc(c *gin.Context) {
	id := aiIDParam(c)
	if id == 0 {
		respErr(c, errBadRequest("文档 ID 无效"))
		return
	}
	v, ok := bind[struct {
		Shared bool `json:"shared"`
	}](c)
	if !ok {
		return
	}
	if err := a.AI.ShareKnowledgeDoc(callerCtx(c), id, v.Shared); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
