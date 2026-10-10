// aitools_more2.go AI 工具第二批补充（续）：数据库实例管理/容器文件与配置/站点域名/PHP 扩展/商店操作/组网。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ypanel/shared/dto"
)

// aiToolsDBInstances 数据库实例生命周期与配置。
func (s *AIService) aiToolsDBInstances(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "create_db_instance", Module: aiModDB, Risk: aiRiskWrite,
			Desc: "以容器方式创建数据库实例（拉镜像+起容器+自动纳管）。input JSON：{\"name\":\"实例名\",\"type\":\"mysql|postgres|redis|mongo\",\"port\":33061,\"password\":\"root 密码\"}",
			Parameters: schObj(map[string]any{
				"name": schStr("实例名"), "type": schEnum("数据库类型", "mysql", "postgres", "redis", "mongo"),
				"port": schInt("宿主映射端口"), "password": schStr("root 密码（8-64 位字母数字）"),
			}, "name", "type", "port", "password"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name     string `json:"name"`
					Type     string `json:"type"`
					Port     int    `json:"port"`
					Password string `json:"password"`
				}](input)
				if err != nil {
					return "", err
				}
				inst, err := s.dbSvc.CreateInstance(ctx, p.Name, p.Type, p.Port, p.Password)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("实例已创建: %s（%s :%d，ID %d）", inst.Name, inst.Type, inst.Port, inst.ID), nil
			},
		},
		{
			Name: "add_external_db_instance", Module: aiModDB, Risk: aiRiskWrite,
			Desc: "接管外部数据库实例（直连验证后纳管）。input JSON：{\"name\":\"实例名\",\"type\":\"mysql|postgres|redis|mongo\",\"host\":\"IP\",\"port\":3306,\"user\":\"root\",\"password\":\"密码\",\"remark\":\"备注\"}",
			Parameters: schObj(map[string]any{
				"name": schStr("实例名"), "type": schEnum("类型", "mysql", "postgres", "redis", "mongo"),
				"host": schStr("主机地址"), "port": schInt("端口"), "user": schStr("管理员账号"),
				"password": schStr("密码"), "remark": schStr("备注"),
			}, "name", "type", "host", "port", "user", "password"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name     string `json:"name"`
					Type     string `json:"type"`
					Host     string `json:"host"`
					Port     int    `json:"port"`
					User     string `json:"user"`
					Password string `json:"password"`
					Remark   string `json:"remark"`
				}](input)
				if err != nil {
					return "", err
				}
				inst, err := s.dbSvc.AddExternalInstance(ctx, p.Name, p.Type, p.Host, p.Port, p.User, p.Password, p.Remark)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("外部实例已接管: %s（%s@%s:%d）", inst.Name, inst.Type, inst.Host, inst.Port), nil
			},
		},
		{
			Name: "db_instance_action", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "启动/停止数据库实例（stop 会断开所有连接，会先向用户确认）。input JSON：{\"instanceId\":1,\"up\":true|false}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "up": schBool("true 启动 / false 停止"),
			}, "instanceId", "up"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint `json:"instanceId"`
					Up         bool `json:"up"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.dbSvc.StartStop(ctx, p.InstanceID, p.Up); err != nil {
					return "", err
				}
				state := "停止"
				if p.Up {
					state = "启动"
				}
				return fmt.Sprintf("实例 %d 已%s", p.InstanceID, state), nil
			},
		},
		{
			Name: "delete_db_instance", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "删除数据库实例纳管（可选同时删除容器数据与备份，不可恢复，会先向用户确认）。input JSON：{\"instanceId\":1,\"purgeData\":false,\"purgeBackups\":false}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "purgeData": schBool("同时删除数据"), "purgeBackups": schBool("同时删除备份"),
			}, "instanceId"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID   uint `json:"instanceId"`
					PurgeData    bool `json:"purgeData"`
					PurgeBackups bool `json:"purgeBackups"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.dbSvc.DeleteInstance(ctx, p.InstanceID, p.PurgeData, p.PurgeBackups); err != nil {
					return "", err
				}
				return fmt.Sprintf("实例 %d 已删除", p.InstanceID), nil
			},
		},
		{
			Name: "toggle_db_remote", Module: aiModDB, Risk: aiRiskWrite,
			Desc: "开启/关闭数据库实例远程访问（SQL 层授权或回收远端管理账号）。input JSON：{\"instanceId\":1,\"enable\":true|false}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "enable": schBool("true 开启远程 / false 关闭"),
			}, "instanceId", "enable"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint `json:"instanceId"`
					Enable     bool `json:"enable"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.dbSvc.RemoteAccess(ctx, p.InstanceID, p.Enable)
				if err != nil {
					return "", err
				}
				state := "关闭"
				if p.Enable {
					state = "开启"
				}
				return toolOut(map[string]any{"message": "远程访问已" + state, "detail": out})
			},
		},
		{
			Name: "set_redis_ttl", Module: aiModDB, Risk: aiRiskWrite,
			Desc: "设置 Redis 键过期时间（秒；0=永不过期）。input JSON：{\"instanceId\":1,\"database\":\"0\",\"name\":\"key\",\"ttlSecs\":3600}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("逻辑库编号"), "name": schStr("键名"), "ttlSecs": schInt("TTL 秒，0=永不过期"),
			}, "instanceId", "name", "ttlSecs"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Name       string `json:"name"`
					TTLSecs    int64  `json:"ttlSecs"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Database == "" {
					p.Database = "0"
				}
				if err := s.adminSvc.SetRedisTTL(ctx, p.InstanceID, "ai", p.Database, p.Name, p.TTLSecs); err != nil {
					return "", err
				}
				return fmt.Sprintf("已设置 %s TTL=%d 秒", p.Name, p.TTLSecs), nil
			},
		},
	}
}

// aiToolsContainerExtra 容器补充：容器内文件读写、资源热更新、daemon 配置、构建缓存清理。
func (s *AIService) aiToolsContainerExtra(ctx context.Context) []aiToolDef {
	base := "/agent/v1/docker/containers/"
	return []aiToolDef{
		{
			Name: "container_file_read", Module: aiModContainers, Risk: aiRiskRead,
			Desc: "读取容器内文件内容（排障容器内配置）。input JSON：{\"name\":\"容器名\",\"path\":\"/etc/nginx/nginx.conf\"}",
			Parameters: schObj(map[string]any{
				"name": schStr("容器名或 ID"), "path": schStr("容器内绝对路径"),
			}, "name", "path"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name string `json:"name"`
					Path string `json:"path"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := agentGetJSON[dto.FileReadResp](s, ctx, base+url.PathEscape(p.Name)+"/files/read?path="+url.QueryEscape(p.Path))
				if err != nil {
					return "", err
				}
				content := truncText(out.Content, 12000)
				if out.Truncated {
					content += fmt.Sprintf("\n…(共 %d 字节，已截断)", out.Size)
				}
				return content, nil
			},
		},
		{
			Name: "container_file_write", Module: aiModContainers, Risk: aiRiskWrite,
			Desc: "写入容器内文本文件（整文件覆盖；改完建议重启容器生效，会先向用户确认）。input JSON：{\"name\":\"容器名\",\"path\":\"容器内路径\",\"content\":\"文本内容\"}",
			Parameters: schObj(map[string]any{
				"name": schStr("容器名或 ID"), "path": schStr("容器内绝对路径"), "content": schStr("完整文本内容"),
			}, "name", "path", "content"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name    string `json:"name"`
					Path    string `json:"path"`
					Content string `json:"content"`
				}](input)
				if err != nil {
					return "", err
				}
				body := dto.FileWriteReq{Path: p.Path, Content: p.Content}
				if _, err := agentPostJSON[dto.FileWriteReq, any](s, ctx, base+url.PathEscape(p.Name)+"/files/write", &body); err != nil {
					return "", err
				}
				return fmt.Sprintf("已写入容器 %s 的 %s（%d 字符）", p.Name, p.Path, len(p.Content)), nil
			},
		},
		{
			Name: "container_update_resources", Module: aiModContainers, Risk: aiRiskWrite,
			Desc: "热更新容器资源限制/重启策略（docker update，免重建；compose 编排容器同样适用）。input JSON：{\"name\":\"容器名\",\"memoryMB\":512,\"cpus\":1.5,\"restart\":\"unless-stopped\"}（字段可省略=不修改）",
			Parameters: schObj(map[string]any{
				"name": schStr("容器名或 ID"), "memoryMB": schInt("内存上限 MB，0=不限"), "cpus": map[string]any{"type": "number", "description": "CPU 核数，0=不限"},
				"restart": schEnum("重启策略", "no", "always", "unless-stopped", "on-failure"),
			}, "name"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name     string  `json:"name"`
					MemoryMB int64   `json:"memoryMB"`
					Cpus     float64 `json:"cpus"`
					Restart  string  `json:"restart"`
				}](input)
				if err != nil {
					return "", err
				}
				body := map[string]any{"memoryMB": p.MemoryMB, "cpus": p.Cpus, "restart": p.Restart}
				out, err := agentPostJSON[map[string]any, json.RawMessage](s, ctx, base+url.PathEscape(p.Name)+"/update", &body)
				if err != nil {
					return "", err
				}
				msg := fmt.Sprintf("容器 %s 资源配置已更新", p.Name)
				if out != nil && len(*out) > 2 && string(*out) != "null" {
					msg += "（警告: " + truncText(string(*out), 300) + "）"
				}
				return msg, nil
			},
		},
		{
			Name: "get_docker_daemon_config", Module: aiModContainers, Risk: aiRiskRead,
			Desc: "读取 Docker daemon.json 配置（registry 镜像加速器/日志策略等）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := agentGetJSON[map[string]string](s, ctx, "/agent/v1/docker/daemon-config")
				if err != nil {
					return "", err
				}
				return (*out)["content"], nil
			},
		},
		{
			Name: "save_docker_daemon_config", Module: aiModContainers, Risk: aiRiskDanger,
			Desc: "保存 Docker daemon.json（需重启 docker 生效，影响全部容器，会先向用户确认）。input JSON：{\"content\":\"完整 JSON 文本\"}",
			Parameters: schObj(map[string]any{"content": schStr("完整 daemon.json 内容")}, "content"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Content string `json:"content"`
				}](input)
				if err != nil {
					return "", err
				}
				if !json.Valid([]byte(p.Content)) {
					return "", fmt.Errorf("daemon.json 不是合法 JSON")
				}
				body := map[string]string{"content": p.Content}
				if _, err := agentPostJSON[map[string]string, any](s, ctx, "/agent/v1/docker/daemon-config", &body); err != nil {
					return "", err
				}
				return "daemon.json 已保存（重启 docker 后生效）", nil
			},
		},
		{
			Name: "prune_build_cache", Module: aiModImages, Risk: aiRiskDanger,
			Desc: "清理 Docker 构建缓存（释放磁盘，会先向用户确认）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := s.dockerX.PassthroughPost(ctx, "/agent/v1/docker/buildcache/prune")
				if err != nil {
					return "", err
				}
				return truncText(string(out), 800), nil
			},
		},
	}
}

// aiToolsSiteDomains 站点域名管理。
func (s *AIService) aiToolsSiteDomains(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "get_site_domains", Module: aiModSites, Risk: aiRiskRead,
			Desc: "查看站点的域名配置（主域名/附加域名）。input JSON：{\"id\":站点ID}",
			Parameters: schObj(map[string]any{"id": schInt("站点 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				conf, err := s.sites.GetDomainConf(p.ID)
				if err != nil {
					return "", err
				}
				return toolOut(conf)
			},
		},
		{
			Name: "update_site_domains", Module: aiModSites, Risk: aiRiskWrite,
			Desc: "更新站点域名（改写 server_name 并 reload；主域名变更时若证书未覆盖新域名会给出告警）。input JSON：{\"id\":站点ID,\"domains\":[\"a.com\",\"b.com\"],\"primary\":\"主域名（须在 domains 内）\"}",
			Parameters: schObj(map[string]any{
				"id": schInt("站点 ID"), "domains": schArr("域名列表", schStr("域名")), "primary": schStr("主域名"),
			}, "id", "domains", "primary"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID       uint     `json:"id"`
					Domains  []string `json:"domains"`
					Primary  string   `json:"primary"`
				}](input)
				if err != nil {
					return "", err
				}
				conf, err := s.sites.UpdateDomainConf(ctx, p.ID, p.Domains, p.Primary)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "站点域名已更新并 reload", "domains": conf})
			},
		},
	}
}

// aiToolsRuntimeExtra 运行环境补充：详情与 PHP 扩展管理。
func (s *AIService) aiToolsRuntimeExtra(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "runtime_detail", Module: aiModRuntime, Risk: aiRiskRead,
			Desc: "查看运行环境详情（版本/端口/容器/PHP 扩展清单等）。input JSON：{\"id\":运行环境ID}",
			Parameters: schObj(map[string]any{"id": schInt("运行环境 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.runtimes.Detail(ctx, p.ID)
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
		{
			Name: "install_php_extension", Module: aiModRuntime, Risk: aiRiskWrite,
			Desc: "为 PHP 运行环境安装扩展（任务化编译安装并重启，耗时数分钟）。input JSON：{\"id\":运行环境ID,\"name\":\"扩展名（redis/gd/imagick 等，见 runtime_detail 扩展清单）\"}",
			Parameters: schObj(map[string]any{
				"id": schInt("运行环境 ID"), "name": schStr("扩展名"),
			}, "id", "name"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID   uint   `json:"id"`
					Name string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.runtimes.InstallPHPExtension(ctx, p.ID, p.Name)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "扩展安装完成", "detail": out})
			},
		},
		{
			Name: "uninstall_php_extension", Module: aiModRuntime, Risk: aiRiskDanger,
			Desc: "卸载 PHP 运行环境的扩展（依赖它的应用会受影响，会先向用户确认）。input JSON：{\"id\":运行环境ID,\"name\":\"扩展名\"}",
			Parameters: schObj(map[string]any{
				"id": schInt("运行环境 ID"), "name": schStr("扩展名"),
			}, "id", "name"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID   uint   `json:"id"`
					Name string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.runtimes.UninstallPHPExtension(ctx, p.ID, p.Name)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "扩展已卸载", "detail": out})
			},
		},
	}
}

// aiToolsStoreExtra 商店补充：已装应用操作/环境变量/源同步。
func (s *AIService) aiToolsStoreExtra(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "app_action", Module: aiModStore, Risk: aiRiskDanger,
			Desc: "对已安装应用执行 start/stop/restart/rebuild（rebuild 会重建容器，会先向用户确认）。input JSON：{\"project\":\"app-blog\",\"action\":\"start|stop|restart|rebuild\"}",
			Parameters: schObj(map[string]any{
				"project": schStr("应用项目名（app- 开头）"), "action": schEnum("操作", "start", "stop", "restart", "rebuild"),
			}, "project", "action"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					Action  string `json:"action"`
				}](input)
				if err != nil {
					return "", err
				}
				switch p.Action {
				case "start", "stop", "restart", "rebuild":
				default:
					return "", fmt.Errorf("不支持的操作: %s", p.Action)
				}
				if err := s.store.InstalledAction(ctx, p.Project, p.Action, ""); err != nil {
					return "", err
				}
				return fmt.Sprintf("已对 %s 执行 %s", p.Project, p.Action), nil
			},
		},
		{
			Name: "read_app_env", Module: aiModStore, Risk: aiRiskRead,
			Desc: "读取已安装应用的 .env 环境变量（含数据库密码等，密码类值界面打码）。input JSON：{\"project\":\"app-blog\"}",
			Parameters: schObj(map[string]any{"project": schStr("应用项目名")}, "project"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Project string `json:"project"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.store.InstallEnv(ctx, p.Project)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "save_app_env", Module: aiModStore, Risk: aiRiskWrite,
			Desc: "保存已安装应用的 .env 并重建容器生效（KEY=VALUE 每行一条；改错会导致应用起不来，会先向用户确认）。input JSON：{\"project\":\"app-blog\",\"content\":\"KEY=VALUE\\nKEY2=VALUE2\"}",
			Parameters: schObj(map[string]any{
				"project": schStr("应用项目名"), "content": schStr("完整 .env 内容（KEY=VALUE 每行一条）"),
			}, "project", "content"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					Content string `json:"content"`
				}](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Content) == "" {
					return "", fmt.Errorf("content 为空")
				}
				if err := s.store.SaveInstallEnv(ctx, p.Project, p.Content); err != nil {
					return "", err
				}
				return fmt.Sprintf("%s 的 .env 已保存并触发重建生效", p.Project), nil
			},
		},
		{
			Name: "sync_store_sources", Module: aiModStore, Risk: aiRiskWrite,
			Desc: "同步应用商店源（拉取最新应用清单与版本）。input JSON：{\"force\":false}",
			Parameters: schObj(map[string]any{"force": schBool("强制同步")}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Force bool `json:"force"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.store.SyncAll(ctx, p.Force)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "商店源同步完成", "detail": out})
			},
		},
	}
}

// aiToolsVPN 组网状态（EasyTier）。
func (s *AIService) aiToolsVPN(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "vpn_status", Module: aiModNetSec, Risk: aiRiskRead,
			Desc: "查看 EasyTier 组网状态（本机虚拟 IP/网络名/在线节点与 peers）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				status, err := s.vpn.Status(ctx)
				if err != nil {
					return "", err
				}
				peers, _ := s.vpn.Peers(ctx)
				return toolOut(map[string]any{"status": status, "peers": peers})
			},
		},
	}
}

var _ = dto.FileWriteReq{}
