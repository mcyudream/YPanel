package dockerx

// M33 P2.6：VictoriaLogs 自动发现（按镜像识别本节点 VL 容器与宿主映射端口）。
// 供 agent 日志代理使用：面板经 agent 通道查询各节点 VL，无需手工配置地址。

import (
	"context"
	"strings"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// vlImagePrefix VictoriaLogs 官方镜像前缀。
const vlImagePrefix = "victoriametrics/victoria-logs"

// DiscoverVL 发现本节点运行中的 VictoriaLogs（取 9428 的宿主映射端口；多实例取第一个）。
func (m *Manager) DiscoverVL(ctx context.Context) (*dto.VLDiscovery, error) {
	list, err := m.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		if c.State != "running" || !strings.HasPrefix(c.Image, vlImagePrefix) {
			continue
		}
		for _, p := range c.Ports {
			if p.HostPort != "" && p.ContainerPort == "9428" {
				return &dto.VLDiscovery{
					Found:     true,
					Container: c.Name,
					Port:      p.HostPort,
					Image:     c.Image,
				}, nil
			}
		}
	}
	return &dto.VLDiscovery{Found: false}, nil
}

// VLProxy 白名单路径（仅查询面，禁止写入/删除类上游）。
var vlProxyPaths = map[string]bool{
	"/select/logsql/query":               true,
	"/select/logsql/hits":                true,
	"/select/logsql/stream_field_values": true,
}

// QueryVL 代理请求本节点 VL 的只读查询接口（GET 上游，响应有界）。
func (m *Manager) QueryVL(ctx context.Context, path string, params map[string]string) (*dto.VLProxyResp, error) {
	if !vlProxyPaths[path] {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的 VL 查询路径")
	}
	disc, err := m.DiscoverVL(ctx)
	if err != nil {
		return nil, err
	}
	if !disc.Found {
		return nil, errs.New(errs.CodeNotFound, "error.notFound", "本节点未发现运行中的 VictoriaLogs 容器")
	}
	body, status, err := httpGetBounded(ctx, "http://127.0.0.1:"+disc.Port+path, params, 32<<20)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "VL 请求失败: "+err.Error())
	}
	return &dto.VLProxyResp{Status: status, Body: body}, nil
}
