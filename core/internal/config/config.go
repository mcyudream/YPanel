// Package config 启动配置：flag 与环境变量双通道，环境变量前缀 YPANEL_。
package config

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
)

// Config 运行配置。
type Config struct {
	Port          int    // 面板监听端口
	DataDir       string // 数据目录（SQLite、日志、临时文件）
	AdminPassword string // 初始 admin 密码（仅首启建号时使用；空则随机生成落盘）
	ResetAdmin    string // 重置指定用户密码并退出
	AgentAddr     string // 外部 agent 地址（多节点；空 = 内嵌 loopback agent）
	Version       string
}

// Load 从 flag + env 装配配置。
func Load(version string) *Config {
	cfg := &Config{Version: version}
	flag.IntVar(&cfg.Port, "port", envInt("YPANEL_PORT", 8880), "面板监听端口")
	flag.StringVar(&cfg.DataDir, "data", envStr("YPANEL_DATA_DIR", "./data"), "数据目录")
	flag.StringVar(&cfg.AdminPassword, "admin-password", os.Getenv("YPANEL_ADMIN_PASSWORD"), "初始 admin 密码（仅首启）")
	flag.StringVar(&cfg.ResetAdmin, "reset-admin", "", "重置指定用户名的密码（交互输入）")
	flag.StringVar(&cfg.AgentAddr, "agent-addr", os.Getenv("YPANEL_AGENT_ADDR"), "外部 agent 地址（多节点）")
	flag.Parse()
	return cfg
}

// DBPath SQLite 文件路径。
func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "ypanel.db")
}

// Addr 监听地址。
func (c *Config) Addr() string {
	return ":" + strconv.Itoa(c.Port)
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
