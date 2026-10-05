// Package agentclient core 访问 agent 的统一 HTTP 客户端。
// 合并部署（loopback）与远程部署（多节点）走同一实现，仅 BaseURL 不同。
package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ypanel/shared/errs"
)

// Client 单节点客户端。
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// New 创建客户端（无超时：下载/日志为长流；各请求单独用 ctx 控时）。
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTP:    &http.Client{},
	}
}

func (c *Client) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	return req, nil
}

// DoJSON 发起 JSON 请求并解包 dto.Resp[T]。
func DoJSON[Req any, Resp any](c *Client, ctx context.Context, method, path string, body *Req) (*Resp, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, errs.Wrap(errs.ErrBadRequest, err.Error())
		}
		rd = bytes.NewReader(b)
	}
	req, err := c.NewRequest(ctx, method, path, rd)
	if err != nil {
		return nil, errs.Wrap(errs.ErrAgentUnreach, err.Error())
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return doResp[Resp](c, req)
}

// GetJSON GET 快捷方式。
func GetJSON[Resp any](c *Client, ctx context.Context, path string) (*Resp, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, errs.Wrap(errs.ErrAgentUnreach, err.Error())
	}
	return doResp[Resp](c, req)
}

func doResp[Resp any](c *Client, req *http.Request) (*Resp, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, errs.Wrap(errs.ErrAgentUnreach, err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	// 先解信封再解 data：agent 返回业务错误时 data 可能为空对象/空值，直接解目标类型会失败
	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, errs.Wrap(errs.ErrAgentUnreach, "agent 响应解析失败: "+err.Error())
	}
	if env.Code != 0 {
		return nil, &errs.Error{Code: env.Code, Message: env.Message}
	}
	var out Resp
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &out); err != nil {
			return nil, errs.Wrap(errs.ErrAgentUnreach, "agent 数据解析失败: "+err.Error())
		}
	}
	return &out, nil
}

// UploadFile 上传本地文件到 agent 指定目录。
func (c *Client) UploadFile(ctx context.Context, dir, localPath string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", errs.Wrap(errs.ErrBadRequest, err.Error())
	}
	defer func() { _ = f.Close() }()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		part, err := mw.CreateFormFile("file", filepath.Base(localPath))
		if err == nil {
			_, err = io.Copy(part, f)
		}
		_ = mw.Close()
		_ = pw.CloseWithError(err)
	}()
	req, err := c.NewRequest(ctx, http.MethodPost, "/agent/v1/files/upload?path="+dir, pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	out, err := doResp[map[string]string](c, req)
	if err != nil {
		return "", err
	}
	return (*out)["path"], nil
}

// WSURL 将 agent 的 WS 端点转为可拨号地址。
func (c *Client) WSURL(path string) string {
	wsScheme := "ws"
	base := c.BaseURL
	if strings.HasPrefix(base, "https://") {
		wsScheme, base = "wss", strings.TrimPrefix(base, "https://")
	}
	base = strings.TrimPrefix(base, "http://")
	return fmt.Sprintf("%s://%s%s", wsScheme, base, path)
}

// Header 返回鉴权头（WS 拨号用）。
func (c *Client) Header() http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+c.Token)
	return h
}
