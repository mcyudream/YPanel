// ypagent 独立分发模式入口（多节点场景）。
//
// 两种启动方式：
//
//	配对模式（推荐）：ypagent -core http://core:8880 -code ABCD1234 [-name node-1] [-addr 0.0.0.0:9527]
//	凭据续启：        ypagent [-addr ...]（自动读取 <data>/agent.json）
//	手工 PSK：        ypagent -token <PSK> -addr ...
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ypanel/agent/internal/pair"
	"github.com/ypanel/agent/server"
)

var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	coreURL := flag.String("core", os.Getenv("YPANEL_CORE_URL"), "core 地址（配对模式）")
	code := flag.String("code", os.Getenv("YPANEL_PAIR_CODE"), "一次性配对码（配对模式）")
	name := flag.String("name", os.Getenv("YPANEL_NODE_NAME"), "节点名（配对模式，默认主机名小写）")
	token := flag.String("token", os.Getenv("YPANEL_AGENT_TOKEN"), "手工 PSK（跳过配对）")
	addr := flag.String("addr", "0.0.0.0:9527", "监听地址")
	dataDir := flag.String("data", defaultDataDir(), "数据目录（凭据文件位置）")
	pairOnly := flag.Bool("pair-only", false, "仅执行配对并保存凭据后退出（供安装脚本使用）")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var cred *pair.Credentials
	switch {
	case *token != "":
		hostname, _ := os.Hostname()
		cred = &pair.Credentials{CoreURL: "", Name: lower(hostname), Token: *token}
		slog.Info("使用手工 PSK 启动", "addr", *addr)
	case *coreURL != "" && *code != "":
		nodeName := *name
		if nodeName == "" {
			hostname, _ := os.Hostname()
			nodeName = lower(hostname)
		}
		c, err := pair.Pair(ctx, *coreURL, *code, nodeName, *addr, version)
		if err != nil {
			slog.Error("配对失败", "err", err)
			os.Exit(1)
		}
		cred = c
		credPath := filepath.Join(*dataDir, "agent.json")
		if err := pair.Save(credPath, cred); err != nil {
			slog.Warn("凭据落盘失败（重启需重新配对）", "err", err)
		} else {
			slog.Info("配对成功，凭据已保存", "file", credPath)
		}
		if *pairOnly {
			// 安装脚本通道：配对落凭据即退出，正式服务由 systemd 无参启动（走已存凭据）
			slog.Info("pair-only 完成，退出")
			return
		}
	default:
		credPath := filepath.Join(*dataDir, "agent.json")
		c, err := pair.Load(credPath)
		if err != nil {
			slog.Error("缺少启动参数：-core+-code（配对）或 -token；未找到已存凭据", "file", credPath)
			os.Exit(2)
		}
		cred = c
		slog.Info("使用已存凭据启动", "core", cred.CoreURL, "name", cred.Name)
	}

	slog.Info("agent 启动", "version", version, "addr", *addr)
	_, wait, err := server.New(server.Config{Token: cred.Token, ListenAddr: *addr}).Start(ctx)
	if err != nil {
		slog.Error("agent 启动失败", "err", err)
		os.Exit(1)
	}

	// 心跳（仅配对节点；手工 PSK 模式无 core 可报）
	if cred.CoreURL != "" {
		go pair.HeartbeatLoop(ctx, cred, version, 30*time.Second, func(err error) {
			slog.Error("心跳致命错误，agent 退出", "err", err)
			stop()
		})
	}
	wait()
	slog.Info("agent 已停止")
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

func defaultDataDir() string {
	if v := os.Getenv("YPANEL_AGENT_DATA"); v != "" {
		return v
	}
	return "/etc/ypanel"
}
