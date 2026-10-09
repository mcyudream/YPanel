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
	GwPort        int    // 内网浏览器网关端口（会话式反代，M51）；<=0 = Port+1
	DataDir       string // 数据目录（SQLite、日志、临时文件）
	AdminPassword string // 初始 admin 密码（仅首启建号时使用；空则随机生成落盘）
	ResetAdmin    string // 重置指定用户密码并退出
	TLSCert       string // 面板 HTTPS 证书路径（空=HTTP）
	TLSKey        string // 面板 HTTPS 私钥路径
	AgentAddr     string // 外部 agent 地址（多节点；空 = 内嵌 loopback agent）
	Version       string
}

// Load 从 flag + env 装配配置。
func Load(version string) *Config {
	cfg := &Config{Version: version}
	flag.IntVar(&cfg.Port, "port", envInt("YPANEL_PORT", 8880), "面板监听端口")
	flag.IntVar(&cfg.GwPort, "gw-port", envInt("YPANEL_GW_PORT", -1), "内网浏览器网关端口（默认 面板端口+1）")
	flag.StringVar(&cfg.DataDir, "data", envStr("YPANEL_DATA_DIR", "./data"), "数据目录")
	flag.StringVar(&cfg.AdminPassword, "admin-password", os.Getenv("YPANEL_ADMIN_PASSWORD"), "初始 admin 密码（仅首启）")
	flag.StringVar(&cfg.ResetAdmin, "reset-admin", "", "重置指定用户名的密码（交互输入）")
	flag.StringVar(&cfg.AgentAddr, "agent-addr", os.Getenv("YPANEL_AGENT_ADDR"), "外部 agent 地址（多节点）")
	flag.StringVar(&cfg.TLSCert, "tls-cert", os.Getenv("YPANEL_TLS_CERT"), "面板 HTTPS 证书路径（空=HTTP）")
	flag.StringVar(&cfg.TLSKey, "tls-key", os.Getenv("YPANEL_TLS_KEY"), "面板 HTTPS 私钥路径")
	flag.Parse()
	if cfg.GwPort <= 0 {
		cfg.GwPort = cfg.Port + 1
	}
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

// GwAddr 内网浏览器网关监听地址。
func (c *Config) GwAddr() string {
	return ":" + strconv.Itoa(c.GwPort)
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
