package dockerx

// httpGetBounded 受限 GET：查询串编码、超时与响应体积上限（VL 只读代理用）。

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var vlHTTPClient = &http.Client{Timeout: 60 * time.Second}

func httpGetBounded(ctx context.Context, base string, params map[string]string, maxBytes int64) (string, int, error) {
	v := url.Values{}
	for k, val := range params {
		if val != "" {
			v.Set(k, val)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?"+v.Encode(), nil)
	if err != nil {
		return "", 0, err
	}
	resp, err := vlHTTPClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("读取响应失败: %w", err)
	}
	return string(body), resp.StatusCode, nil
}
