// Package pair agent 与 core 的配对注册与心跳。
package pair

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ypanel/shared/errs"
)

// Credentials 落地凭据。
type Credentials struct {
	CoreURL string `json:"coreUrl"`
	Name    string `json:"name"`
	Token   string `json:"token"`
}

// Load 从凭据文件读取。
func Load(path string) (*Credentials, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.CoreURL == "" || c.Name == "" || c.Token == "" {
		return nil, fmt.Errorf("凭据文件不完整")
	}
	return &c, nil
}

// Save 写凭据文件（0600）。
func Save(path string, c *Credentials) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(path, b, 0o600)
}

// PairRequest 配对请求。
type PairRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Addr     string `json:"addr"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
}

// PairResponse 配对响应。
type PairResponse struct {
	Token              string `json:"token"`
	HeartbeatInterval  int    `json:"heartbeatInterval"`
}

// advertiseAddr 探测本机到 core 的出站 IP。
func advertiseAddr(coreURL, listenAddr string) string {
	host := strings.TrimPrefix(strings.TrimPrefix(coreURL, "http://"), "https://")
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// 取 host 的 IP 形式；域名则通过 UDP connect 探测路由源地址
	if net.ParseIP(host) != nil {
		conn, err := net.Dial("udp", net.JoinHostPort(host, "80"))
		if err == nil {
			defer func() { _ = conn.Close() }()
			if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP.To4() != nil && !addr.IP.IsLoopback() {
				return addr.IP.String()
			}
		}
	}
	// 兜底：任意非回环出口
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err == nil {
		defer func() { _ = conn.Close() }()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && !addr.IP.IsLoopback() {
			return addr.IP.String()
		}
	}
	return "127.0.0.1"
}

// Pair 执行配对，返回凭据。
func Pair(ctx context.Context, coreURL, code, name, listenAddr, version string) (*Credentials, error) {
	hostname, _ := os.Hostname()
	ip := advertiseAddr(coreURL, listenAddr)
	port := "9527"
	if _, p, err := net.SplitHostPort(listenAddr); err == nil && p != "" && p != "0" {
		port = p
	}
	req := PairRequest{
		Code: code, Name: name,
		Addr:     fmt.Sprintf("http://%s:%s", ip, port),
		Hostname: hostname,
		OS:       goos(), Arch: goarch(), Version: version,
	}
	var env struct {
		Code int `json:"code"`
		Data PairResponse `json:"data"`
		Message string `json:"message"`
	}
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(coreURL, "/")+"/api/v1/pair", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("连接 core 失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("core 响应解析失败: %w", err)
	}
	if env.Code != 0 {
		return nil, fmt.Errorf("配对失败: %s", env.Message)
	}
	return &Credentials{CoreURL: coreURL, Name: name, Token: env.Data.Token}, nil
}

// HeartbeatLoop 心跳循环：成功返回（节点被删/凭据失效）时结束；网络错误重试。
func HeartbeatLoop(ctx context.Context, cred *Credentials, interval time.Duration, onFatal func(error)) {
	t := time.NewTicker(interval)
	defer t.Stop()
	exec := func() error {
		req, err := http.NewRequestWithContext(ctx, "POST",
			strings.TrimRight(cred.CoreURL, "/")+"/api/v1/pair/heartbeat?name="+cred.Name, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+cred.Token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == http.StatusOK {
			var env struct {
				Code int `json:"code"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&env); err == nil && env.Code == errs.CodeForbidden {
				return fmt.Errorf("节点已被 core 移除")
			}
			return nil
		}
		return fmt.Errorf("心跳 HTTP %d", resp.StatusCode)
	}
	for {
		if err := exec(); err != nil {
			// 403 类致命错误（节点被删）退出；网络类静默重试
			if strings.Contains(err.Error(), "被 core 移除") {
				onFatal(err)
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// 运行时信息辅助。
func goos() string   { return runtimeGOOS }
func goarch() string { return runtimeGOARCH }
