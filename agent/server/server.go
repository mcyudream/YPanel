// Package server agent 的 loopback HTTP 服务。
// 单机合并部署时 agent 进程内嵌运行（127.0.0.1 随机端口），多节点远程部署时由 cmd/agent 独立启动——
// 两种形态对 core 暴露完全一致的 HTTP 协议。
package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/ypanel/agent/internal/compose"
	"github.com/ypanel/agent/internal/dockerx"
	"github.com/ypanel/agent/internal/files"
	"github.com/ypanel/agent/internal/sysinfo"
	"github.com/ypanel/shared/errs"
)

// Config 服务配置。
type Config struct {
	Token      string // PSK，Authorization: Bearer 校验；空则拒绝启动
	ListenAddr string // 监听地址；空 = 127.0.0.1:0（合并部署 loopback 随机端口）
	ComposeDir string // compose 托管目录；空 = /opt/ypanel/compose
}

// Server agent HTTP 服务。
type Server struct {
	cfg     Config
	sys     *sysinfo.Collector
	files   *files.Manager
	dock    *dockerx.Manager
	compose *compose.Manager
	srv     *http.Server
}

// New 创建服务并启动采样器。
func New(cfg Config) *Server {
	composeDir := cfg.ComposeDir
	if composeDir == "" {
		composeDir = "/opt/ypanel/compose"
	}
	m := &Server{
		cfg:   cfg,
		sys:   sysinfo.New(2, 1800), // 2s 采样，保留 1 小时
		files: files.New(nil),        // 根为 "/"，全盘管理
		dock:  dockerx.New(),
	}
	// 目录创建失败时 compose 为 nil，接口层降级为能力不可用
	m.compose, _ = compose.New(composeDir, m.dock)
	return m
}

// Start 启动服务，立即返回实际监听地址（http://addr）；wait 阻塞至服务退出。
func (s *Server) Start(ctx context.Context) (base string, wait func(), err error) {
	if s.cfg.Token == "" {
		return "", nil, errors.New("agent token 不能为空")
	}
	s.sys.Start(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /agent/v1/health", s.handleHealth)
	mux.HandleFunc("GET /agent/v1/sysinfo/overview", s.auth(s.handleOverview))
	mux.HandleFunc("GET /agent/v1/sysinfo/history", s.auth(s.handleHistory))
	mux.HandleFunc("GET /agent/v1/files/list", s.auth(s.handleFileList))
	mux.HandleFunc("GET /agent/v1/files/read", s.auth(s.handleFileRead))
	mux.HandleFunc("GET /agent/v1/files/download", s.auth(s.handleFileDownload))
	mux.HandleFunc("POST /agent/v1/files/write", s.auth(s.handleFileWrite))
	mux.HandleFunc("POST /agent/v1/files/mkdir", s.auth(s.handleFileMkdir))
	mux.HandleFunc("POST /agent/v1/files/rename", s.auth(s.handleFileRename))
	mux.HandleFunc("POST /agent/v1/files/delete", s.auth(s.handleFileDelete))
	mux.HandleFunc("POST /agent/v1/files/upload", s.auth(s.handleFileUpload))
	mux.HandleFunc("GET /agent/v1/docker/containers", s.auth(s.handleDockerList))
	mux.HandleFunc("POST /agent/v1/docker/containers/{id}/{action}", s.auth(s.handleDockerAction))
	mux.HandleFunc("GET /agent/v1/docker/containers/{id}/logs", s.auth(s.handleDockerLogs))
	mux.HandleFunc("GET /agent/v1/terminal", s.auth(s.handleTerminal))
	mux.HandleFunc("GET /agent/v1/compose/projects", s.auth(s.handleComposeList))
	mux.HandleFunc("GET /agent/v1/compose/config", s.auth(s.handleComposeConfig))
	mux.HandleFunc("POST /agent/v1/compose/config", s.auth(s.handleComposeWrite))
	mux.HandleFunc("POST /agent/v1/compose/up", s.auth(s.handleComposeUp))
	mux.HandleFunc("POST /agent/v1/compose/down", s.auth(s.handleComposeDown))
	mux.HandleFunc("GET /agent/v1/compose/logs", s.auth(s.handleComposeLogs))
	mux.HandleFunc("POST /agent/v1/exec", s.auth(s.handleExec))
	mux.HandleFunc("GET /agent/v1/processes", s.auth(s.handleProcessList))
	mux.HandleFunc("POST /agent/v1/processes/kill", s.auth(s.handleProcessKill))
	mux.HandleFunc("GET /agent/v1/services", s.auth(s.handleServiceList))
	mux.HandleFunc("POST /agent/v1/services/{name}/{action}", s.auth(s.handleServiceAction))

	ln, err := net.Listen("tcp", s.listenAddr())
	if err != nil {
		return "", nil, fmt.Errorf("agent 监听失败: %w", err)
	}
	s.srv = &http.Server{Handler: mux}
	base = "http://" + ln.Addr().String()

	go func() {
		<-ctx.Done()
		_ = s.srv.Close()
	}()
	slog.Info("agent server started", "addr", base)
	served := make(chan error, 1)
	go func() {
		served <- s.srv.Serve(ln)
	}()
	return base, func() {
		if err := <-served; err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("agent server exited", "err", err)
		}
	}, nil
}

func (s *Server) listenAddr() string {
	if s.cfg.ListenAddr != "" {
		return s.cfg.ListenAddr
	}
	return "127.0.0.1:0"
}

// auth Bearer PSK 校验（constant time 比较）。
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.Token)) != 1 {
			writeErr(w, errs.ErrForbidden)
			return
		}
		next(w, r)
	}
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return ""
}

// ---- 响应辅助 ----

func writeOK[T any](w http.ResponseWriter, data T) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(errs.RespOK(data))
}

func writeOKEmpty(w http.ResponseWriter) {
	writeOK(w, struct{}{})
}

func writeErr(w http.ResponseWriter, err error) {
	var be *errs.Error
	if errors.As(err, &be) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(errs.RespErr(be))
		return
	}
	slog.Error("agent internal error", "err", err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(errs.RespErr(errs.ErrInternal))
}

func slogWarn(msg string, err error) {
	if err != nil {
		slog.Warn(msg, "err", err)
	}
}
