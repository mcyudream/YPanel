// AIService AI 运维助手（B18 v2）：langchaingo 能力层。
// 多供应商（openai 兼容/anthropic）+ 工具循环（面板工具集）+ 知识库检索注入 +
// 场景感知（当前页面数据摘要）+ SSE 流式。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/tools"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
)

// AIProvider 供应商配置。
type AIProvider struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	APIType   string `json:"apiType"` // openai（兼容）/ anthropic
	BaseURL   string `json:"baseURL"`
	APIKey    string `json:"apiKey"`
	Model     string `json:"model"`
	IsDefault bool   `json:"isDefault"`
}

// AIKnowledge 知识库条目。
type AIKnowledge struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// ChatMessage 对话消息。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIService AI 助手。
type AIService struct {
	db    *gorm.DB
	nodes *NodeService
}

// NewAIService 创建。
func NewAIService(db *gorm.DB, nodes *NodeService) *AIService {
	return &AIService{db: db, nodes: nodes}
}

// ---- 供应商 CRUD ----

func (s *AIService) toOut(p *model.AIProvider, withKey bool) AIProvider {
	out := AIProvider{ID: p.ID, Name: p.Name, APIType: p.APIType, BaseURL: p.BaseURL, Model: p.Model, IsDefault: p.IsDefault}
	if withKey {
		out.APIKey = p.APIKey
	}
	return out
}

// ListProviders 供应商列表（不回显 key）。
func (s *AIService) ListProviders() []AIProvider {
	rows := []model.AIProvider{}
	_ = s.db.Order("is_default desc, id").Find(&rows).Error
	out := []AIProvider{}
	for i := range rows {
		out = append(out, s.toOut(&rows[i], false))
	}
	return out
}

// Presets 内置供应商预设。
func (s *AIService) Presets() []AIProvider {
	return []AIProvider{
		{ID: 0, Name: "智谱 GLM", APIType: "openai", BaseURL: "https://open.bigmodel.cn/api/coding/paas/v4", Model: "glm-4.6"},
		{ID: 0, Name: "OpenAI", APIType: "openai", BaseURL: "https://api.openai.com/v1", Model: "gpt-4o"},
		{ID: 0, Name: "Anthropic", APIType: "anthropic", BaseURL: "https://api.anthropic.com", Model: "claude-sonnet-4-20250514"},
		{ID: 0, Name: "DeepSeek", APIType: "openai", BaseURL: "https://api.deepseek.com/v1", Model: "deepseek-chat"},
		{ID: 0, Name: "Ollama 本地", APIType: "openai", BaseURL: "http://127.0.0.1:11434/v1", Model: "qwen2.5:7b"},
	}
}

// SaveProvider 新建/更新。
func (s *AIService) SaveProvider(p *model.AIProvider) error {
	if p.APIType != "openai" && p.APIType != "anthropic" {
		return errWrapAI("API 类型仅支持 openai/anthropic")
	}
	if p.IsDefault {
		s.db.Model(&model.AIProvider{}).Where("is_default = ?", true).Update("is_default", false)
	}
	if p.ID == 0 {
		return s.db.Create(p).Error
	}
	return s.db.Model(p).Updates(map[string]any{
		"name": p.Name, "api_type": p.APIType, "base_url": p.BaseURL,
		"api_key": p.APIKey, "model": p.Model, "is_default": p.IsDefault,
	}).Error
}

// DeleteProvider 删除。
func (s *AIService) DeleteProvider(id uint) error {
	return s.db.Delete(&model.AIProvider{}, id).Error
}

// ProviderByID 单条。
func (s *AIService) ProviderByID(id uint) (*model.AIProvider, error) {
	var row model.AIProvider
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// NodeLocal 本地节点信息（B18 工作空间端点用）。
func (s *AIService) NodeLocal() (map[string]any, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": node.ID, "name": node.Name, "baseURL": node.BaseURL, "token": node.Token}, nil
}

// DefaultProvider 默认供应商。
func (s *AIService) DefaultProvider() (*model.AIProvider, error) {
	var rows []model.AIProvider
	if err := s.db.Order("is_default desc, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errWrapAI("未配置 AI 供应商")
	}
	return &rows[0], nil
}

// ---- 知识库 ----

// ListKnowledge 知识库列表。
func (s *AIService) ListKnowledge() []model.AIKnowledge {
	out := []model.AIKnowledge{}
	_ = s.db.Order("id desc").Find(&out).Error
	return out
}

// SaveKnowledge 新建/更新。
func (s *AIService) SaveKnowledge(k *model.AIKnowledge) error {
	if k.Title == "" {
		return errWrapAI("标题必填")
	}
	if k.ID == 0 {
		return s.db.Create(k).Error
	}
	return s.db.Model(k).Updates(map[string]any{"title": k.Title, "body": k.Body}).Error
}

// DeleteKnowledge 删除。
func (s *AIService) DeleteKnowledge(id uint) error {
	return s.db.Delete(&model.AIKnowledge{}, id).Error
}

// searchKnowledge 关键词检索（命中数最多的前 3 条）。
func (s *AIService) searchKnowledge(query string) []model.AIKnowledge {
	all := s.ListKnowledge()
	if len(all) == 0 {
		return nil
	}
	words := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127)
	})
	if len(words) == 0 {
		return nil
	}
	type scored struct {
		k model.AIKnowledge
		n int
	}
	var ss []scored
	for _, k := range all {
		low := strings.ToLower(k.Title + " " + k.Body)
		n := 0
		for _, w := range words {
			if strings.Contains(low, w) {
				n++
			}
		}
		if n > 0 {
			ss = append(ss, scored{k, n})
		}
	}
	for i := 1; i < len(ss) && i < 3; i++ {
		for j := i; j > 0 && ss[j].n > ss[j-1].n; j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
	limit := 3
	if len(ss) < limit {
		limit = len(ss)
	}
	out := make([]model.AIKnowledge, 0, limit)
	for _, x := range ss[:limit] {
		out = append(out, x.k)
	}
	return out
}

// ---- 场景感知 ----

// SceneContext 按页面路径返回数据摘要。
func (s *AIService) SceneContext(ctx context.Context, path string) map[string]any {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	switch {
	case strings.HasPrefix(path, "/container"):
		out, err := agentclient.GetJSON[[]dto.ContainerItem](ac, ctx, "/agent/v1/docker/containers")
		if err == nil {
			return map[string]any{"page": "容器管理", "containers": out}
		}
	case strings.HasPrefix(path, "/sites"):
		sites := []model.Site{}
		_ = s.db.Find(&sites).Error
		return map[string]any{"page": "网站管理", "sites": sites}
	case strings.HasPrefix(path, "/database"):
		out, err := agentclient.GetJSON[[]map[string]any](ac, ctx, "/agent/v1/database/instances")
		if err == nil {
			return map[string]any{"page": "数据库", "instances": out}
		}
	case path == "/" || path == "" || strings.HasPrefix(path, "/overview"):
		ov, err := agentclient.GetJSON[dto.SystemOverview](ac, ctx, "/agent/v1/sysinfo/overview")
		if err == nil {
			return map[string]any{"page": "主机概览", "overview": ov}
		}
	}
	return map[string]any{"page": path}
}

// buildSceneSummary 场景摘要文本。
func (s *AIService) buildSceneSummary(ctx context.Context, path string) string {
	sc := s.SceneContext(ctx, path)
	b, _ := json.Marshal(sc)
	if len(b) > 4096 {
		return "用户当前所在面板页面实时数据（JSON 截断）：" + string(b[:4096])
	}
	return "用户当前所在面板页面实时数据（JSON）：" + string(b)
}

// ---- 工具系统 ----

// workspaceDir AI 工作空间沙箱目录。
const workspaceDir = "/opt/ypanel/ai-workspace"

// errWrapAI 局部错误包装。
func errWrapAI(msg string) error {
	return fmt.Errorf("AI: %s", msg)
}

// aiTool 单个工具（实现 langchaingo tools.Tool 接口）。
type aiTool struct {
	name        string
	description string
	fn          func(ctx context.Context, input string) (string, error)
}

func (t *aiTool) Name() string        { return t.name }
func (t *aiTool) Description() string { return t.description }
func (t *aiTool) Call(ctx context.Context, input string) (string, error) {
	return t.fn(ctx, input)
}

// toolsFor 组装工具集。
func (s *AIService) toolsFor(ctx context.Context) []tools.Tool {
	run := func(command string) (string, error) {
		node, nerr := s.nodes.ByID("local")
		if nerr != nil {
			return "", nerr
		}
		ac := agentclient.New(node.BaseURL, node.Token)
		res, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: command, TimeoutSecs: 120})
		if err != nil {
			return "", err
		}
		out := strings.TrimSpace(res.Output)
		if res.ExitCode != 0 {
			return fmt.Sprintf("exit=%d\n%s", res.ExitCode, out), nil
		}
		if out == "" {
			out = "(无输出)"
		}
		return out, nil
	}

	return []tools.Tool{
		&aiTool{
			name:        "get_overview",
			description: "获取服务器实时概览：CPU/内存/磁盘/负载/网络速率。无参数，input 传空。",
			fn: func(_ context.Context, _ string) (string, error) {
				node, nerr := s.nodes.ByID("local")
				if nerr != nil {
					return "", nerr
				}
				ac := agentclient.New(node.BaseURL, node.Token)
				ov, err := agentclient.GetJSON[dto.SystemOverview](ac, ctx, "/agent/v1/sysinfo/overview")
				if err != nil {
					return "", err
				}
				b, _ := json.Marshal(map[string]any{
					"cpuPercent": ov.CPU.UsagePercent, "cores": ov.CPU.LogicalCount,
					"memPercent": ov.Memory.UsagePercent, "memUsedGB": float64(ov.Memory.Used) / 1e9,
					"load1": ov.Load.Load1, "disk": ov.Disks,
					"rxSpeedBps": ov.Network.RxSpeedBps, "txSpeedBps": ov.Network.TxSpeedBps,
				})
				return string(b), nil
			},
		},
		&aiTool{
			name:        "list_containers",
			description: "列出全部 Docker 容器（名称/镜像/状态/端口）。无参数，input 传空。",
			fn: func(_ context.Context, _ string) (string, error) {
				node, nerr := s.nodes.ByID("local")
				if nerr != nil {
					return "", nerr
				}
				ac := agentclient.New(node.BaseURL, node.Token)
				out, err := agentclient.GetJSON[[]dto.ContainerItem](ac, ctx, "/agent/v1/docker/containers")
				if err != nil {
					return "", err
				}
				b, _ := json.Marshal(out)
				return string(b), nil
			},
		},
		&aiTool{
			name:        "container_action",
			description: "对容器执行操作。input 为 JSON：{\"name\":\"容器名\",\"action\":\"start|stop|restart\"}",
			fn: func(_ context.Context, input string) (string, error) {
				var p struct {
					Name   string `json:"name"`
					Action string `json:"action"`
				}
				if err := json.Unmarshal([]byte(input), &p); err != nil {
					return "", err
				}
				if p.Action != "start" && p.Action != "stop" && p.Action != "restart" {
					return "", fmt.Errorf("不支持的操作: %s", p.Action)
				}
				return run(fmt.Sprintf("docker %s %s", p.Action, p.Name))
			},
		},
		&aiTool{
			name:        "list_sites",
			description: "列出全部网站站点（名称/域名/类型/启用状态）。无参数，input 传空。",
			fn: func(_ context.Context, _ string) (string, error) {
				sites := []model.Site{}
				if err := s.db.Find(&sites).Error; err != nil {
					return "", err
				}
				b, _ := json.Marshal(sites)
				return string(b), nil
			},
		},
		&aiTool{
			name:        "read_file",
			description: "读取服务器上的文本文件内容（前 100KB）。input 为文件绝对路径。",
			fn: func(_ context.Context, input string) (string, error) {
				p := strings.TrimSpace(input)
				if p == "" || !strings.HasPrefix(p, "/") {
					return "", fmt.Errorf("需要绝对路径")
				}
				return run(fmt.Sprintf("head -c 102400 '%s'", strings.ReplaceAll(p, "'", "'\\''")))
			},
		},
		&aiTool{
			name:        "run_in_workspace",
			description: "在 AI 工作空间沙箱目录（" + workspaceDir + "）中执行 shell 命令，用于运行脚本/代码/写文件。input 为 shell 命令字符串。",
			fn: func(_ context.Context, input string) (string, error) {
				command := strings.TrimSpace(input)
				if command == "" {
					return "", fmt.Errorf("命令为空")
				}
				return run(fmt.Sprintf("mkdir -p '%s' && cd '%s' && %s", workspaceDir, workspaceDir, command))
			},
		},
	}
}
