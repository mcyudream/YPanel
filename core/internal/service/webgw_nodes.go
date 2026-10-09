// WebGw 多节点支持（M50）：按节点发现可浏览目标（容器发布端口）+ core 侧可达性预检。
// 目标仍由 core 直接反代（要求节点网络可路由：同网段 / VPN）；NAT 隔离节点需中继隧道，列后续批次。
package service

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// WebTarget 节点上的可浏览目标（容器发布端口的 http 候选）。
type WebTarget struct {
	Name      string `json:"name"`      // 容器名
	Container string `json:"container"` // 容器 ID（12 位）
	URL       string `json:"url"`       // http://<节点IP>:<宿主端口>
	HostPort  string `json:"hostPort"`
	Reachable bool   `json:"reachable"` // core 侧 TCP 预检结果
}

// SetNodes 注入节点服务（main 装配）。
func (s *WebGwService) SetNodes(nodes *NodeService) { s.nodes = nodes }

// NodeIP 取节点内网 IP（经该节点 agent：hostname -I 首个地址；local 走 loopback agent）。
func (s *WebGwService) NodeIP(ctx context.Context, nodeID string) (string, error) {
	node, err := s.nodes.ByID(nodeID)
	if err != nil {
		return "", err
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: "hostname -I 2>/dev/null | awk '{print $1}'", TimeoutSecs: 15})
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(out.Output)
	if ip == "" {
		return "", errs.Wrap(errs.ErrBadRequest, "节点未上报内网 IP")
	}
	return ip, nil
}

// NodeTargets 聚合节点上运行中容器的发布端口为 http 候选目标，并做 core 侧可达性预检。
func (s *WebGwService) NodeTargets(ctx context.Context, nodeID string) (map[string]any, error) {
	node, err := s.nodes.ByID(nodeID)
	if err != nil {
		return nil, err
	}
	ip, err := s.NodeIP(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	containers, err := agentclient.GetJSON[[]dto.ContainerItem](ac, ctx, "/agent/v1/docker/containers")
	if err != nil {
		return nil, err
	}
	targets := []WebTarget{}
	for _, c := range *containers {
		if c.State != "running" {
			continue
		}
		name := c.Name
		seen := map[string]bool{}
		for _, p := range c.Ports {
			if p.HostPort == "" || seen[p.HostPort] {
				continue
			}
			seen[p.HostPort] = true
			id12 := c.ID
			if len(id12) > 12 {
				id12 = id12[:12]
			}
			targets = append(targets, WebTarget{
				Name: name, Container: id12,
				URL: fmt.Sprintf("http://%s:%s", ip, p.HostPort), HostPort: p.HostPort,
			})
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Name != targets[j].Name {
			return targets[i].Name < targets[j].Name
		}
		return targets[i].HostPort < targets[j].HostPort
	})
	// 并发 TCP 预检（1s 超时）
	var wg sync.WaitGroup
	for i := range targets {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", strings.TrimPrefix(targets[k].URL, "http://"), time.Second)
			if err == nil {
				_ = conn.Close()
				targets[k].Reachable = true
			}
		}(i)
	}
	wg.Wait()
	return map[string]any{"node": nodeID, "ip": ip, "targets": targets}, nil
}
