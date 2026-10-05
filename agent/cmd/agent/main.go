// agent 独立分发模式的入口（多节点场景）。
// 单机合并部署时 core 直接进程内启动本服务，不经由此入口。
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ypanel/agent/server"
)

var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	token := flag.String("token", os.Getenv("YPANEL_AGENT_TOKEN"), "PSK 令牌（env YPANEL_AGENT_TOKEN）")
	addr := flag.String("addr", "0.0.0.0:9527", "监听地址（独立模式）")
	flag.Parse()

	if *token == "" {
		slog.Error("缺少 --token / YPANEL_AGENT_TOKEN")
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("agent standalone 启动", "version", version, "addr", *addr)
	_, wait, err := server.New(server.Config{Token: *token, ListenAddr: *addr}).Start(ctx)
	if err != nil {
		slog.Error("agent 启动失败", "err", err)
		os.Exit(1)
	}
	wait()
	slog.Info("agent 已停止")
}
