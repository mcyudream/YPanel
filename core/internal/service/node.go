package service

import (
	"context"
	"os"

	"github.com/ypanel/agent/server"
	"github.com/ypanel/shared/errs"
)

// Node 一个受管节点（M0 仅本机节点 local）。
type Node struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"-"` // agent 基址
	Token   string `json:"-"`
}

// NodeService 节点注册表：M0 管理进程内嵌 agent；多节点扩展点。
type NodeService struct {
	local *Node
}

// NewNodeService 启动本机 agent（loopback 随机端口 + 随机 PSK）。
func NewNodeService(ctx context.Context) (*NodeService, error) {
	token := randomToken()
	srv := server.New(server.Config{Token: token})
	base, wait, err := srv.Start(ctx)
	if err != nil {
		return nil, err
	}
	go wait()
	hostname, _ := osHostname()
	return &NodeService{local: &Node{ID: "local", Name: hostname, BaseURL: base, Token: token}}, nil
}

// Local 本机节点。
func (s *NodeService) Local() *Node { return s.local }

// ByID 按 ID 取节点（M0 仅 local）。
func (s *NodeService) ByID(id string) (*Node, error) {
	if id == "" || id == "local" {
		return s.local, nil
	}
	return nil, errs.New(errs.CodeNotFound, "error.nodeNotFound", "节点不存在")
}

func randomToken() string {
	return randomHex(32)
}

func osHostname() (string, error) {
	return os.Hostname()
}
