// AIService AI 运维助手（B18 v2）：langchaingo 能力层。
// 多供应商（openai 兼容/anthropic）+ 工具循环（面板工具集）+ 知识库检索注入 +
// 场景感知（当前页面数据摘要）+ SSE 流式。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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
	Models    string `json:"models"` // 可用模型列表（逗号分隔）；空 = 仅 Model 一个
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
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	Reasoning string          `json:"reasoning,omitempty"`
	Steps     []ChatStep      `json:"steps,omitempty"`
	Knowledge []ChatKnowledge `json:"knowledge,omitempty"`
}

// ChatKnowledge 回答引用的知识库条目（随会话持久化）。
type ChatKnowledge struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

// ChatStep 过程步骤（工具/场景，随会话持久化）。
type ChatStep struct {
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Status  string `json:"status,omitempty"` // 失败/完成（空 = 未结束）
	Done    bool   `json:"done,omitempty"`
}

// AIService AI 助手。
type AIService struct {
	db       *gorm.DB
	nodes    *NodeService
	dbSvc    *DatabaseService
	adminSvc *DBAdminService
	settings *SettingService
	skills   *SkillsManager
}

// ListSkills 转发技能列表。
func (s *AIService) ListSkills(ctx context.Context) ([]AISkill, error) {
	return s.skills.List(ctx)
}

// SaveSkill 转发技能保存。
func (s *AIService) SaveSkill(ctx context.Context, name, description, body string) error {
	return s.skills.Save(ctx, name, description, body)
}

// SetSkillEnabled 转发技能启停。
func (s *AIService) SetSkillEnabled(name string, enabled bool) error {
	return s.skills.SetEnabled(name, enabled)
}

// RemoveSkill 转发技能删除。
func (s *AIService) RemoveSkill(ctx context.Context, name string) error {
	return s.skills.Remove(ctx, name)
}

// UploadSkillZip 上传技能包压缩包。
func (s *AIService) UploadSkillZip(ctx context.Context, data []byte) (string, string, error) {
	return s.skills.UploadZip(ctx, data)
}

// NewAIService 创建。
func NewAIService(db *gorm.DB, nodes *NodeService, dbSvc *DatabaseService, adminSvc *DBAdminService, settings *SettingService, skills *SkillsManager) *AIService {
	return &AIService{db: db, nodes: nodes, dbSvc: dbSvc, adminSvc: adminSvc, settings: settings, skills: skills}
}

// ---- 供应商 CRUD ----

func (s *AIService) toOut(p *model.AIProvider, withKey bool) AIProvider {
	out := AIProvider{ID: p.ID, Name: p.Name, APIType: p.APIType, BaseURL: p.BaseURL, Model: p.Model, Models: p.Models, IsDefault: p.IsDefault}
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
		{ID: 0, Name: "OpenAI", APIType: "openai", BaseURL: "https://api.openai.com/v1", Model: "gpt-4o"},
		{ID: 0, Name: "Anthropic Claude", APIType: "anthropic", BaseURL: "https://api.anthropic.com", Model: "claude-sonnet-4-20250514"},
		{ID: 0, Name: "Google Gemini", APIType: "openai", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", Model: "gemini-2.0-flash"},
		{ID: 0, Name: "DeepSeek", APIType: "openai", BaseURL: "https://api.deepseek.com/v1", Model: "deepseek-chat"},
		{ID: 0, Name: "智谱 GLM", APIType: "openai", BaseURL: "https://open.bigmodel.cn/api/paas/v4", Model: "glm-4.6"},
		{ID: 0, Name: "Moonshot Kimi", APIType: "openai", BaseURL: "https://api.moonshot.cn/v1", Model: "moonshot-v1-32k"},
		{ID: 0, Name: "阿里通义 Qwen", APIType: "openai", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", Model: "qwen-plus"},
		{ID: 0, Name: "字节豆包（火山方舟）", APIType: "openai", BaseURL: "https://ark.cn-beijing.volces.com/api/v3", Model: "doubao-pro-32k"},
		{ID: 0, Name: "MiniMax", APIType: "openai", BaseURL: "https://api.minimax.chat/v1", Model: "abab6.5s-chat"},
		{ID: 0, Name: "零一万物 Yi", APIType: "openai", BaseURL: "https://api.lingyiwanwu.com/v1", Model: "yi-large"},
		{ID: 0, Name: "百度文心", APIType: "openai", BaseURL: "https://qianfan.baidubce.com/v2", Model: "ernie-4.0-8k"},
		{ID: 0, Name: "讯飞星火", APIType: "openai", BaseURL: "https://spark-api-open.xf-yun.com/v1", Model: "generalv3.5"},
		{ID: 0, Name: "硅基流动 SiliconCloud", APIType: "openai", BaseURL: "https://api.siliconflow.cn/v1", Model: "deepseek-ai/DeepSeek-V3"},
		{ID: 0, Name: "xAI Grok", APIType: "openai", BaseURL: "https://api.x.ai/v1", Model: "grok-3"},
		{ID: 0, Name: "Mistral", APIType: "openai", BaseURL: "https://api.mistral.ai/v1", Model: "mistral-large-latest"},
		{ID: 0, Name: "Groq", APIType: "openai", BaseURL: "https://api.groq.com/openai/v1", Model: "llama-3.3-70b-versatile"},
		{ID: 0, Name: "Together", APIType: "openai", BaseURL: "https://api.together.xyz/v1", Model: "meta-llama/Llama-3.3-70B-Instruct-Turbo"},
		{ID: 0, Name: "OpenRouter", APIType: "openai", BaseURL: "https://openrouter.ai/api/v1", Model: "openai/gpt-4o"},
		{ID: 0, Name: "百川 Baichuan", APIType: "openai", BaseURL: "https://api.baichuan-ai.com/v1", Model: "Baichuan4"},
		{ID: 0, Name: "Cohere", APIType: "openai", BaseURL: "https://api.cohere.com/compatibility/v1", Model: "command-r-plus"},
		{ID: 0, Name: "Azure OpenAI", APIType: "openai", BaseURL: "https://<resource>.openai.azure.com", Model: "gpt-4o"},
		{ID: 0, Name: "Ollama 本地", APIType: "openai", BaseURL: "http://127.0.0.1:11434/v1", Model: "llama3.1"},
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
	updates := map[string]any{
		"name": p.Name, "api_type": p.APIType, "base_url": p.BaseURL,
		"model": p.Model, "models": p.Models, "is_default": p.IsDefault,
	}
	// apiKey 为空表示沿用已存密钥（前端编辑弹窗「留空不修改」）
	if p.APIKey != "" {
		updates["api_key"] = p.APIKey
	}
	return s.db.Model(p).Updates(updates).Error
}

// ProviderModels 拉取供应商可用模型列表（openai 兼容 GET {baseURL}/models）。
func (s *AIService) ProviderModels(id uint) ([]string, error) {
	p, err := s.ProviderByID(id)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(p.BaseURL, "/")+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errWrapAI(fmt.Sprintf("模型列表拉取失败: HTTP %d", resp.StatusCode))
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	return models, nil
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

// KnowledgeHit 知识检索命中（统一手工条目与文档分块形态）。
type KnowledgeHit struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// scoreWords 命中的分词数。
func scoreWords(words []string, text string) int {
	low := strings.ToLower(text)
	n := 0
	for _, w := range words {
		if strings.Contains(low, w) {
			n++
		}
	}
	return n
}

// searchKnowledge 关键词检索：手工条目 + 知识文档分块统一打分，取命中数最多的前 3 条。
func (s *AIService) searchKnowledge(query string) []KnowledgeHit {
	words := tokenizeKnowledgeQuery(query)
	if len(words) == 0 {
		return nil
	}
	type scored struct {
		hit KnowledgeHit
		n   int
	}
	var ss []scored
	for _, k := range s.ListKnowledge() {
		if n := scoreWords(words, k.Title+" "+k.Body); n > 0 {
			ss = append(ss, scored{KnowledgeHit{Title: "[条目] " + k.Title, Body: k.Body}, n})
		}
	}
	docs := map[uint]string{}
	docRows := []model.AIKnowledgeDoc{}
	_ = s.db.Find(&docRows).Error
	for _, d := range docRows {
		docs[d.ID] = d.Title
	}
	chunks := []model.AIKnowledgeChunk{}
	_ = s.db.Find(&chunks).Error
	for _, ch := range chunks {
		if n := scoreWords(words, ch.Heading+" "+ch.Body); n > 0 {
			ss = append(ss, scored{KnowledgeHit{
				Title: fmt.Sprintf("[文档] %s › %s", docs[ch.DocID], ch.Heading),
				Body:  ch.Body,
			}, n})
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
	out := make([]KnowledgeHit, 0, limit)
	for _, it := range ss[:limit] {
		out = append(out, it.hit)
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

// ---- 列表类工具的可选入参：搜索/排序/分页（模型按需取页，避免一次性回传全量撑爆上下文） ----

type listQuery struct {
	Search   string `json:"search"`
	State    string `json:"state"` // 容器专用：running / stopped 等
	Sort     string `json:"sort"`
	Order    string `json:"order"` // asc / desc
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

func parseListQuery(input string) listQuery {
	q := listQuery{Page: 1, PageSize: 20, Order: "asc"}
	s := strings.TrimSpace(input)
	_ = json.Unmarshal([]byte(s), &q)
	// langchaingo 未设 Parameters schema 时模型会把参数包成 {"input":"<json 字符串>"}，需解包
	var wrapper struct {
		Input string `json:"input"`
	}
	if json.Unmarshal([]byte(s), &wrapper) == nil && wrapper.Input != "" {
		_ = json.Unmarshal([]byte(wrapper.Input), &q)
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	if q.Order != "desc" {
		q.Order = "asc"
	}
	return q
}

// paginateList 过滤后的切片按 q 排序（less 为 nil 保持原序）并切页，返回当前页与总数。
func paginateList[T any](items []T, q listQuery, less func(a, b T) bool) ([]T, int) {
	if less != nil {
		sort.SliceStable(items, func(i, j int) bool {
			if q.Order == "desc" {
				return less(items[j], items[i])
			}
			return less(items[i], items[j])
		})
	}
	total := len(items)
	start := (q.Page - 1) * q.PageSize
	if start >= total {
		return []T{}, total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}
	return items[start:end], total
}

// matchAny 关键词命中任一字段即真（空关键词恒真，大小写不敏感）。
func matchAny(kw string, fields ...string) bool {
	if kw == "" {
		return true
	}
	kw = strings.ToLower(kw)
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), kw) {
			return true
		}
	}
	return false
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

// builtinTools 内置系统工具全集（不受开关过滤，管理页与工具循环共用）。
func (s *AIService) builtinTools(ctx context.Context) []tools.Tool {
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
			description: "列出 Docker 容器（分页，勿假设一次返回全部）。input 可选 JSON：{\"search\":\"名称/镜像/ID 关键词\",\"state\":\"running 或 stopped\",\"sort\":\"name|state|image|created\",\"order\":\"asc|desc\",\"page\":1,\"pageSize\":20}，全部可省略（默认第 1 页 20 条，按名称排序）。返回 {total,page,pageSize,items}；total 超过当前页时按需翻页或加 search 收窄。",
			fn: func(_ context.Context, input string) (string, error) {
				node, nerr := s.nodes.ByID("local")
				if nerr != nil {
					return "", nerr
				}
				ac := agentclient.New(node.BaseURL, node.Token)
				out, err := agentclient.GetJSON[[]dto.ContainerItem](ac, ctx, "/agent/v1/docker/containers")
				if err != nil {
					return "", err
				}
				items := []dto.ContainerItem{}
				if out != nil {
					items = *out
				}
				q := parseListQuery(input)
				filtered := make([]dto.ContainerItem, 0, len(items))
				for _, c := range items {
					if !matchAny(q.Search, c.Name, c.Image, c.ID) {
						continue
					}
					if q.State != "" && c.State != q.State {
						continue
					}
					filtered = append(filtered, c)
				}
				var less func(a, b dto.ContainerItem) bool
				switch q.Sort {
				case "state":
					less = func(a, b dto.ContainerItem) bool { return a.State < b.State }
				case "image":
					less = func(a, b dto.ContainerItem) bool { return a.Image < b.Image }
				case "created":
					less = func(a, b dto.ContainerItem) bool { return a.Created.Before(b.Created) }
				default:
					less = func(a, b dto.ContainerItem) bool { return a.Name < b.Name }
				}
				pageItems, total := paginateList(filtered, q, less)
				b, _ := json.Marshal(map[string]any{"total": total, "page": q.Page, "pageSize": q.PageSize, "items": pageItems})
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
			description: "列出网站站点（分页/搜索）。input 可选 JSON：{\"search\":\"站点名/域名关键词\",\"sort\":\"name|type|domain\",\"order\":\"asc|desc\",\"page\":1,\"pageSize\":20}，全部可省略。返回 {total,page,pageSize,items}。",
			fn: func(_ context.Context, input string) (string, error) {
				sites := []model.Site{}
				if err := s.db.Find(&sites).Error; err != nil {
					return "", err
				}
				q := parseListQuery(input)
				filtered := make([]model.Site, 0, len(sites))
				for _, st := range sites {
					if matchAny(q.Search, st.Name, st.Domain, st.Domains) {
						filtered = append(filtered, st)
					}
				}
				var less func(a, b model.Site) bool
				switch q.Sort {
				case "type":
					less = func(a, b model.Site) bool { return a.Type < b.Type }
				case "domain":
					less = func(a, b model.Site) bool { return a.Domain < b.Domain }
				default:
					less = func(a, b model.Site) bool { return a.Name < b.Name }
				}
				pageItems, total := paginateList(filtered, q, less)
				b, _ := json.Marshal(map[string]any{"total": total, "page": q.Page, "pageSize": q.PageSize, "items": pageItems})
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
		&aiTool{
			name:        "list_database_instances",
			description: "列出面板管理的数据库实例（分页/搜索）。input 可选 JSON：{\"search\":\"名称/类型/备注关键词\",\"sort\":\"name|type\",\"order\":\"asc|desc\",\"page\":1,\"pageSize\":20}，全部可省略。返回 {total,page,pageSize,items}；实例 ID 用于 query_database。",
			fn: func(_ context.Context, input string) (string, error) {
				out, err := s.dbSvc.List(ctx)
				if err != nil {
					return "", err
				}
				q := parseListQuery(input)
				filtered := make([]map[string]any, 0, len(out))
				for _, row := range out {
					if matchAny(q.Search, fmt.Sprint(row["name"]), fmt.Sprint(row["type"]), fmt.Sprint(row["remark"]), fmt.Sprint(row["host"])) {
						filtered = append(filtered, row)
					}
				}
				var less func(a, b map[string]any) bool
				switch q.Sort {
				case "type":
					less = func(a, b map[string]any) bool { return fmt.Sprint(a["type"]) < fmt.Sprint(b["type"]) }
				default:
					less = func(a, b map[string]any) bool { return fmt.Sprint(a["name"]) < fmt.Sprint(b["name"]) }
				}
				pageItems, total := paginateList(filtered, q, less)
				b, _ := json.Marshal(map[string]any{"total": total, "page": q.Page, "pageSize": q.PageSize, "items": pageItems})
				return string(b), nil
			},
		},
		&aiTool{
			name:        "query_database",
			description: "对数据库实例执行只读 SQL（仅 SELECT/SHOW/DESC/EXPLAIN，最多 40 行）。input 为 JSON：{\"instanceId\":1,\"database\":\"库名\",\"sql\":\"SELECT ...\"}。先调 list_database_instances 获取实例 ID。",
			fn: func(_ context.Context, input string) (string, error) {
				var p struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					SQL        string `json:"sql"`
				}
				if err := json.Unmarshal([]byte(input), &p); err != nil {
					return "", err
				}
				return s.aiQueryDatabase(ctx, p.InstanceID, p.Database, p.SQL)
			},
		},
		&aiTool{
			name:        "save_memory",
			description: "把本次对话中值得长期记住的运维经验/用户偏好/服务器特性沉淀为记忆（下次对话自动可用）。input 为一句话记忆内容。",
			fn: func(_ context.Context, input string) (string, error) {
				content := strings.TrimSpace(input)
				// 模型可能传 {"content":"..."} 或 {"input":"..."} 包装，取内层纯文本
				var probe struct {
					Content string `json:"content"`
					Input   string `json:"input"`
				}
				if json.Unmarshal([]byte(content), &probe) == nil && (probe.Content != "" || probe.Input != "") {
					if probe.Content != "" {
						content = strings.TrimSpace(probe.Content)
					} else {
						content = strings.TrimSpace(probe.Input)
					}
				}
				if content == "" {
					return "", fmt.Errorf("记忆内容为空")
				}
				if err := s.db.Create(&model.AIMemory{Content: content}).Error; err != nil {
					return "", err
				}
				return "已记住", nil
			},
		},
		&aiTool{
			name:        "read_knowledge",
			description: "按需深读知识库（回答引用了知识片段后如需更多上下文时使用）。input JSON 二选一：{\"search\":\"关键词\"} 返回命中的条目/文档章节清单（标题+摘要）；{\"doc\":\"文档标题关键词\"} 返回最匹配文档的全文（截断 6000 字）。",
			fn: func(_ context.Context, input string) (string, error) {
				var p struct {
					Search string `json:"search"`
					Doc    string `json:"doc"`
				}
				raw := strings.TrimSpace(input)
				_ = json.Unmarshal([]byte(raw), &p)
				var wrapper struct {
					Input string `json:"input"`
				}
				if json.Unmarshal([]byte(raw), &wrapper) == nil && wrapper.Input != "" {
					_ = json.Unmarshal([]byte(wrapper.Input), &p)
				}
				if p.Doc != "" {
					docs := []model.AIKnowledgeDoc{}
					if err := s.db.Where("title LIKE ? OR filename LIKE ?", "%"+p.Doc+"%", "%"+p.Doc+"%").Order("id desc").Find(&docs).Error; err != nil {
						return "", err
					}
					if len(docs) == 0 {
						return "未找到匹配文档：" + p.Doc, nil
					}
					content := docs[0].Content
					if len(content) > 6000 {
						content = content[:6000] + "…（已截断，全文见知识库）"
					}
					b, _ := json.Marshal(map[string]any{"title": docs[0].Title, "content": content})
					return string(b), nil
				}
				if p.Search == "" {
					return "需提供 search 或 doc 参数", nil
				}
				hits := s.searchKnowledge(p.Search)
				if len(hits) == 0 {
					return "知识库无命中：" + p.Search, nil
				}
				var sb strings.Builder
				for _, h := range hits {
					summary := h.Body
					if len(summary) > 160 {
						summary = summary[:160] + "…"
					}
					sb.WriteString("【" + h.Title + "】" + summary + "\n")
				}
				sb.WriteString("（如需某文档全文，用 {\"doc\":\"标题关键词\"} 再查）")
				return sb.String(), nil
			},
		},
	}
}

// toolsFor 组装工具集：内置系统工具按开关过滤 + MCP 工具。
func (s *AIService) toolsFor(ctx context.Context) []tools.Tool {
	flags := s.ToolFlags()
	out := make([]tools.Tool, 0, len(s.builtinTools(ctx)))
	for _, t := range s.builtinTools(ctx) {
		if enabled, ok := flags[t.Name()]; !ok || enabled {
			out = append(out, t)
		}
	}
	return out
}

// ToolFlags 工具开关表（未记录 = 启用）。
func (s *AIService) ToolFlags() map[string]bool {
	rows := []model.AIToolFlag{}
	_ = s.db.Find(&rows).Error
	m := make(map[string]bool, len(rows))
	for _, r := range rows {
		m[r.Name] = r.Enabled
	}
	return m
}

// ListTools 系统工具清单（管理页：名称/描述/开关）。
func (s *AIService) ListTools() []map[string]any {
	flags := s.ToolFlags()
	builtins := s.builtinTools(context.Background())
	out := make([]map[string]any, 0, len(builtins))
	for _, t := range builtins {
		enabled := true
		if v, ok := flags[t.Name()]; ok {
			enabled = v
		}
		out = append(out, map[string]any{
			"name": t.Name(), "description": t.Description(), "enabled": enabled,
		})
	}
	return out
}

// SetToolFlag 设置工具开关。
func (s *AIService) SetToolFlag(name string, enabled bool) error {
	known := false
	for _, t := range s.builtinTools(context.Background()) {
		if t.Name() == name {
			known = true
			break
		}
	}
	if !known {
		return errWrapAI("未知工具: " + name)
	}
	// 注意：GORM Save 对「设置了主键但行不存在」只执行 UPDATE（影响 0 行）且不报错，必须显式 upsert
	row := model.AIToolFlag{Name: name}
	if err := s.db.Where("name = ?", name).FirstOrCreate(&row).Error; err != nil {
		return err
	}
	return s.db.Model(&model.AIToolFlag{}).Where("name = ?", name).Update("enabled", enabled).Error
}
