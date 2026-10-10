package api

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/shared/errs"
)

// escape query 转义。
func escape(s string) string { return url.QueryEscape(s) }

// errBadRequest 快捷构造。
func errBadRequest(msg string) error { return errs.Wrap(errs.ErrBadRequest, msg) }

// errAgentUnreach 快捷构造。
func errAgentUnreach(err error) error {
	if err == nil {
		return errs.ErrAgentUnreach
	}
	return errs.Wrap(errs.ErrAgentUnreach, err.Error())
}

// uploadMultipart 重建 multipart 并转发至 agent。
func (f *FileAPI) uploadMultipart(c *gin.Context, r io.Reader, filename, dir string) (any, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		var werr error
		if part, perr := mw.CreateFormFile("file", filename); perr == nil {
			_, werr = io.Copy(part, r)
		} else {
			werr = perr
		}
		_ = mw.Close()
		_ = pw.CloseWithError(werr)
	}()
	cl, cerr := f.client(c)
	if cerr != nil {
		return nil, cerr
	}
	req, err := cl.NewRequest(c.Request.Context(), http.MethodPost, "/agent/v1/files/upload?path="+escape(dir), pr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := cl.HTTP.Do(req)
	if err != nil {
		return nil, errAgentUnreach(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, errAgentUnreach(nil)
	}
	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    map[string]any  `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, errAgentUnreach(err)
	}
	if env.Code != 0 {
		return nil, &errs.Error{Code: env.Code, Message: env.Message}
	}
	return env.Data, nil
}
