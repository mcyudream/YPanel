// AIService AI 运维助手（B18 v2）：langchaingo 能力层。
// 多供应商（openai 兼容/anthropic）+ 工具循环（面板工具集）+ 知识库检索注入 +
// 场景感知（当前页面数据摘要）+ SSE 流式。
package service

import (
	"log/slog"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tmc/langchaingo/tools"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
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
	db        *gorm.DB
	nodes     *NodeService
	dbSvc     *DatabaseService
	adminSvc  *DBAdminService
	settings  *SettingService
	skills    *SkillsManager
	sites     *SiteService
	certs     *CertificateService
	runtimes  *RuntimeService
	dockerX   *DockerExtService
	fw        *FirewallService
	nat       *NatForwardService
	hosts     *HostsService
	dns       *DnsService
	cron      *Cron
	store     *StoreService
	f2b       *Fail2banService
	src2      *Src2ComposeService
	alert     *AlertService
	hist      *HistoryRecorder
	notif     *NotificationService
	panelBk   *PanelBackupService
	su        *SelfUpdateService
	vpn       *VPNService
	pendingAsks sync.Map // askID → *aiAskRequest（等待用户确认的危险操作）
	pendingQuestions sync.Map // questionID → *aiQuestionRequest（等待用户回答的交互提问）
}

// AIDeps AI 服务外部依赖（main 装配注入；M31 扩展全域能力所需服务）。
type AIDeps struct {
	Nodes    *NodeService
	DBSvc    *DatabaseService
	DBAdmin  *DBAdminService
	Settings *SettingService
	Skills   *SkillsManager
	Sites    *SiteService
	Certs    *CertificateService
	Runtimes *RuntimeService
	DockerX  *DockerExtService
	FW       *FirewallService
	NAT      *NatForwardService
	Hosts    *HostsService
	DNS      *DnsService
	Cron     *Cron
	Store    *StoreService
	F2B      *Fail2banService
	Src2     *Src2ComposeService
	Alert    *AlertService
	Hist     *HistoryRecorder
	Notif    *NotificationService
	PanelBK  *PanelBackupService
	SU       *SelfUpdateService
	VPN      *VPNService
}

// NewAIService 创建。
func NewAIService(db *gorm.DB, deps AIDeps) *AIService {
	return &AIService{
		db: db, nodes: deps.Nodes, dbSvc: deps.DBSvc, adminSvc: deps.DBAdmin,
		settings: deps.Settings, skills: deps.Skills,
		sites: deps.Sites, certs: deps.Certs, runtimes: deps.Runtimes, dockerX: deps.DockerX,
		fw: deps.FW, nat: deps.NAT, hosts: deps.Hosts, dns: deps.DNS, cron: deps.Cron,
		store: deps.Store, f2b: deps.F2B, src2: deps.Src2,
		alert: deps.Alert, hist: deps.Hist, notif: deps.Notif, panelBk: deps.PanelBK, su: deps.SU, vpn: deps.VPN,
		pendingAsks: sync.Map{},
	}
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

// NewAIService 创建（见 AIDeps）。

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

// SceneContext 按页面路径返回数据摘要；focus 非空时标注用户当前聚焦对象（桌面工作台焦点窗口感知）。
func (s *AIService) SceneContext(ctx context.Context, path string, focus string) map[string]any {
	sc := s.sceneData(ctx, path)
	if focus != "" {
		sc["focus"] = focus
	}
	return sc
}

func (s *AIService) sceneData(ctx context.Context, path string) map[string]any {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return map[string]any{"page": path}
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
func (s *AIService) buildSceneSummary(ctx context.Context, path string, focus string) string {
	sc := s.SceneContext(ctx, path, focus)
	b, _ := json.Marshal(sc)
	if len(b) > 4096 {
		return "用户当前所在面板页面实时数据（JSON 截断）：" + string(b[:4096])
	}
	text := "用户当前所在面板页面实时数据（JSON）：" + string(b)
	if focus != "" {
		text += "；用户当前正在查看的对象：" + focus
	}
	return text
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

// ---- 工具系统（M31 治理化：模块注册表 + 风险分级 + JSON Schema + ask 确认） ----

// 工具风险分级：read 查询自动执行；write 新增/编辑；danger 删除/停启/清理等不可逆或影响运行态的操作。
const (
	aiRiskRead   = "read"
	aiRiskWrite  = "write"
	aiRiskDanger = "danger"
)

// ask 确认模式（settings ai.ask_mode）：strict = write+danger 都需用户确认（默认）；
// danger_only = 仅 danger 级确认，write 自动执行。
const (
	aiAskStrict      = "strict"
	aiAskDangerOnly  = "danger_only"
	settingAIAskMode = "ai.ask_mode"
	aiAskTimeout     = 120 * time.Second
)

// 工具模块 key。
const (
	aiModMeta       = "meta"
	aiModSystem     = "system"
	aiModContainers = "docker_containers"
	aiModImages     = "docker_images"
	aiModNetworks   = "docker_networks"
	aiModVolumes    = "docker_volumes"
	aiModCompose    = "compose"
	aiModFiles      = "files"
	aiModProc       = "processes"
	aiModExec       = "exec"
	aiModDB         = "databases"
	aiModSites      = "sites_certs"
	aiModRuntime    = "runtimes"
	aiModNetSec     = "firewall_net"
	aiModTasks      = "tasks"
	aiModStore      = "store"
	aiModSrcBuild   = "srcbuild"
	aiModDiag       = "diagnostics"
	aiModMonitor    = "monitor_alert"
	aiModPanel      = "panel_ops"
	aiModAI         = "panel_ai"
)

// aiModule 模块元数据（load_tools 目录 + 管理页分组）。
type aiModule struct {
	Key   string
	Title string
	Desc  string
}

// aiModuleRegistry 功能模块注册表（顺序即目录展示顺序）。
var aiModuleRegistry = []aiModule{
	{aiModSystem, "系统概览", "主机 CPU/内存/磁盘/网络实时数据、磁盘占用分析、面板纳管节点列表"},
	{aiModContainers, "容器管理", "容器列表/详情/日志/资源占用，创建容器，启停重启，删除，清理已停止容器"},
	{aiModImages, "镜像管理", "镜像列表、拉取镜像、删除镜像、清理悬空镜像"},
	{aiModNetworks, "容器网络", "网络列表、创建网络、删除网络"},
	{aiModVolumes, "存储卷", "卷列表、创建卷、删除卷、清理未使用卷"},
	{aiModCompose, "编排应用", "compose 项目列表/日志/配置读写、上线、服务启停重启、下线、删除项目"},
	{aiModFiles, "文件管理", "主机文件浏览/搜索/读取，写入/建目录/改名/复制/压缩解压/权限，删除"},
	{aiModProc, "进程服务", "进程与服务列表、终止进程、服务启停重启"},
	{aiModExec, "命令执行", "主机任意 shell 命令执行与 AI 工作空间执行（最高风险，务必先向用户确认用途）"},
	{aiModDB, "数据库", "实例/库/表/用户查询与只读 SQL，建库/建用户，行级增删改，删库/删用户"},
	{aiModSites, "网站证书", "站点列表/访问日志/创建/配置编辑/启停/删除，证书列表/续签/删除"},
	{aiModRuntime, "运行环境", "PHP 等运行时列表/详情、启停重启、删除"},
	{aiModNetSec, "网络安全", "防火墙规则、NAT 转发、Hosts 记录、内网 DNS 记录的查询与增删"},
	{aiModTasks, "计划任务", "计划任务列表/新建/立即执行/删除，任务中心状态与日志查询（应用安装/源码构建等异步任务跟踪）"},
	{aiModStore, "应用商店", "按需求检索应用（如博客/图床/网盘/数据库）、查看应用详情与安装参数、一键安装/卸载、查看已装应用与升级状态"},
	{aiModSrcBuild, "源码构建", "从 git 仓库预检语言栈与构建参数，创建源码构建部署任务（自动生成 compose 并构建跑通）"},
	{aiModDiag, "诊断排查", "端口监听检查、HTTP 探测、DNS 解析验证、systemd 服务日志(journalctl)、从 URL 下载文件到服务器"},
	{aiModMonitor, "监控告警", "告警规则查询与创建、历史监控指标查询（CPU/内存趋势）、面板通知、登录审计日志"},
	{aiModPanel, "面板运维", "面板备份创建/查询/删除、检查面板更新与执行升级"},
	{aiModAI, "AI 自身", "长期记忆沉淀与知识库深读（常驻工具，无需加载）"},
}

// aiModuleTitle 模块 key → 标题（未知 key 原样返回）。
func aiModuleTitle(key string) string {
	for _, m := range aiModuleRegistry {
		if m.Key == key {
			return m.Title
		}
	}
	return key
}

// sceneAIModules 场景路径 → 预注入模块（用户所在功能页对话时该域工具直接可用，无需 load_tools）。
func sceneAIModules(path string) []string {
	switch {
	case strings.HasPrefix(path, "/container"):
		return []string{aiModContainers}
	case strings.HasPrefix(path, "/docker"):
		return []string{aiModImages, aiModNetworks, aiModVolumes}
	case strings.HasPrefix(path, "/compose") || strings.HasPrefix(path, "/app"):
		return []string{aiModCompose, aiModSrcBuild}
	case strings.HasPrefix(path, "/store"):
		return []string{aiModStore}
	case strings.HasPrefix(path, "/sites") || strings.HasPrefix(path, "/certs"):
		return []string{aiModSites}
	case strings.HasPrefix(path, "/runtimes"):
		return []string{aiModRuntime}
	case strings.HasPrefix(path, "/database"):
		return []string{aiModDB}
	case strings.HasPrefix(path, "/tools"):
		return []string{aiModTasks}
	case strings.HasPrefix(path, "/manage") || strings.HasPrefix(path, "/monitor"):
		return []string{aiModMonitor, aiModNetSec}
	case strings.HasPrefix(path, "/system"):
		return []string{aiModFiles, aiModProc, aiModNetSec}
	}
	return nil
}

// aiToolDef 单个工具声明：模块归属 + 风险分级 + JSON Schema 入参。
type aiToolDef struct {
	Name       string
	Desc       string
	Module     string
	Risk       string
	Parameters map[string]any
	Fn         func(ctx context.Context, args string) (string, error)
}

// aiTool langchaingo tools.Tool 适配器。
type aiTool struct{ def aiToolDef }

func (t *aiTool) Name() string        { return t.def.Name }
func (t *aiTool) Description() string { return t.def.Desc }
func (t *aiTool) Call(ctx context.Context, input string) (string, error) {
	return t.def.Fn(ctx, input)
}

// ---- 入参解析与出参封装 ----

// unwrapArgs 兼容历史形态：模型把入参包成 {"input":"<json 字符串>"} 时解开取内层。
func unwrapArgs(input string) string {
	s := strings.TrimSpace(input)
	var probe map[string]json.RawMessage
	if json.Unmarshal([]byte(s), &probe) == nil {
		if inner, ok := probe["input"]; ok && len(probe) == 1 {
			return string(inner)
		}
	}
	return s
}

// parseToolArgs 入参 JSON 解析（先解 wrapper 再反序列化）。
func parseToolArgs[T any](input string) (T, error) {
	var v T
	if err := json.Unmarshal([]byte(unwrapArgs(input)), &v); err != nil {
		return v, fmt.Errorf("入参应为 JSON 对象: %v", err)
	}
	return v, nil
}

// toolOut 工具结果 JSON 化。
func toolOut(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// truncText 按字符截断（模型上下文友好）。
func truncText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…(已截断)"
}

// ---- JSON Schema 构造 ----

func schObj(props map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func schStr(desc string) map[string]any  { return map[string]any{"type": "string", "description": desc} }
func schInt(desc string) map[string]any  { return map[string]any{"type": "integer", "description": desc} }
func schBool(desc string) map[string]any { return map[string]any{"type": "boolean", "description": desc} }

func schEnum(desc string, values ...string) map[string]any {
	return map[string]any{"type": "string", "description": desc, "enum": values}
}

func schArr(desc string, items map[string]any) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": items}
}

// ---- agent 通道助手 ----

// acLocal 本地节点 agentclient。
func (s *AIService) acLocal() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// aiNodeCtxKey 工具入参声明的目标节点（ctx 注入，helper 层统一消费）。
type aiNodeCtxKey struct{}

// withAINode 将工具入参的 node 字段注入 ctx（该次工具执行的所有 agent 调用打到目标节点）。
func withAINode(ctx context.Context, node string) context.Context {
	node = strings.TrimSpace(node)
	if node == "" || node == "local" {
		return ctx
	}
	return context.WithValue(ctx, aiNodeCtxKey{}, node)
}

// acFromCtx 工具 helper 的节点解析：ctx 携带目标节点用之（子节点 agent），否则 local。
func (s *AIService) acFromCtx(ctx context.Context) (*agentclient.Client, error) {
	if node, ok := ctx.Value(aiNodeCtxKey{}).(string); ok && node != "" {
		n, err := s.nodes.ByID(node)
		if err != nil || n == nil {
			return nil, fmt.Errorf("节点不存在: %s", node)
		}
		return agentclient.New(n.BaseURL, n.Token), nil
	}
	return s.acLocal()
}

// hostExec 经 agent 通道在主机执行 shell（默认 120s 超时）。
func (s *AIService) hostExec(ctx context.Context, command string) (string, error) {
	ac, err := s.acFromCtx(ctx)
	if err != nil {
		return "", err
	}
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

// agentGetJSON 调 agent JSON 接口（包级泛型函数：Go 方法不允许类型参数）。
func agentGetJSON[T any](s *AIService, ctx context.Context, path string) (*T, error) {
	ac, err := s.acFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return agentclient.GetJSON[T](ac, ctx, path)
}

// agentPostJSON 调 agent JSON 接口（POST）。
func agentPostJSON[Req any, Resp any](s *AIService, ctx context.Context, path string, body *Req) (*Resp, error) {
	ac, err := s.acFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return agentclient.DoJSON[Req, Resp](ac, ctx, "POST", path, body)
}

// agentText 调 agent 文本响应端点（容器/compose 日志等非信封端点）。
func (s *AIService) agentText(ctx context.Context, path string) (string, error) {
	ac, err := s.acFromCtx(ctx)
	if err != nil {
		return "", err
	}
	req, err := ac.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("agent HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 192<<10))
	if err != nil {
		return "", err
	}
	out := strings.TrimSpace(string(b))
	if out == "" {
		out = "(无日志)"
	}
	return out, nil
}

// hostExecTimeout 带自定义超时的主机命令执行（上限 300s）。
func (s *AIService) hostExecTimeout(ctx context.Context, command string, timeoutSecs int) (string, error) {
	if timeoutSecs < 1 || timeoutSecs > 300 {
		timeoutSecs = 120
	}
	ac, err := s.acFromCtx(ctx)
	if err != nil {
		return "", err
	}
	res, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: command, TimeoutSecs: timeoutSecs})
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

// agentDeleteJSON 调 agent DELETE 接口（DockerExt 透传通道）。
func (s *AIService) agentDeleteJSON(ctx context.Context, path string) (json.RawMessage, error) {
	return s.dockerX.Delete(ctx, path)
}

// ---- ask 确认（一次性审批，fail-closed：超时/取消/无应答一律拒绝） ----

// aiAskRequest 待用户确认的危险操作。
type aiAskRequest struct {
	ch chan bool
}

// aiQuestionRequest 待用户回答的交互提问（ask_user 工具）。
type aiQuestionRequest struct {
	ch chan string // 答案 JSON 文本
}

// ResolveAsk 落用户的确认/拒绝结果；ask 不存在（已超时清理）返回 false。
func (s *AIService) ResolveAsk(id string, approve bool) bool {
	v, ok := s.pendingAsks.Load(id)
	if !ok {
		return false
	}
	req := v.(*aiAskRequest)
	select {
	case req.ch <- approve:
	default:
	}
	return true
}

// ResolveQuestion 落用户的提问回答（answers JSON 文本）；问题不存在（已取消）返回 false。
func (s *AIService) ResolveQuestion(id string, answers string) bool {
	v, ok := s.pendingQuestions.Load(id)
	slog.Info("ai ask-question received", "id", id, "found", ok)
	if !ok {
		return false
	}
	req := v.(*aiQuestionRequest)
	select {
	case req.ch <- answers:
	default:
	}
	return true
}

func randAskID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// AskMode 当前确认模式。
func (s *AIService) AskMode() string {
	m := s.settings.Get(settingAIAskMode, aiAskStrict)
	if m != aiAskStrict && m != aiAskDangerOnly {
		return aiAskStrict
	}
	return m
}

// SetAskMode 设置确认模式（strict / danger_only）。
func (s *AIService) SetAskMode(mode string) error {
	if mode != aiAskStrict && mode != aiAskDangerOnly {
		return errWrapAI("确认模式仅支持 strict / danger_only")
	}
	return s.settings.Set(settingAIAskMode, mode)
}

// auditOperation 写/danger 操作审计落库（含被拒绝/超时的 ask）。
func (s *AIService) auditOperation(def aiToolDef, args, result, errMsg string, success bool, action string) {
	merged := maskSensitive(strings.TrimSpace(strings.TrimSpace(result) + "\n" + strings.TrimSpace(errMsg)))
	_ = s.db.Create(&model.AIOperationLog{
		Tool: def.Name, Module: def.Module, Risk: def.Risk, Action: action,
		Args: truncText(args, 2000), Result: truncText(merged, 480), Success: success,
	}).Error
}

// ---- 工具目录与聚合 ----

// aiToolsAll 按模块聚合的全量工具声明（不含 load_tools 元工具）。
// 出口统一包权限闸（M54）：调用者权限经 ctx 注入（AI 对话 / MCP 入口），无权限工具执行时报错——
// 对话与 MCP 两条链路都经 def.Fn 执行，此收口同时覆盖两者。
func (s *AIService) aiToolsAll(ctx context.Context) []aiToolDef {
	defs := make([]aiToolDef, 0, 96)
	defs = append(defs, s.aiToolsSystem(ctx)...)
	defs = append(defs, s.aiToolsContainers(ctx)...)
	defs = append(defs, s.aiToolsImages(ctx)...)
	defs = append(defs, s.aiToolsNetworks(ctx)...)
	defs = append(defs, s.aiToolsVolumes(ctx)...)
	defs = append(defs, s.aiToolsCompose(ctx)...)
	defs = append(defs, s.aiToolsFiles(ctx)...)
	defs = append(defs, s.aiToolsProc(ctx)...)
	defs = append(defs, s.aiToolsExec(ctx)...)
	defs = append(defs, s.aiToolsDatabases(ctx)...)
	defs = append(defs, s.aiToolsDBMore(ctx)...)
	defs = append(defs, s.aiToolsSites(ctx)...)
	defs = append(defs, s.aiToolsRuntimes(ctx)...)
	defs = append(defs, s.aiToolsNetSec(ctx)...)
	defs = append(defs, s.aiToolsTasks(ctx)...)
	defs = append(defs, s.aiToolsStore(ctx)...)
	defs = append(defs, s.aiToolsSrcBuild(ctx)...)
	defs = append(defs, s.aiToolsDiag(ctx)...)
	defs = append(defs, s.aiToolsMonitor(ctx)...)
	defs = append(defs, s.aiToolsPanelOps(ctx)...)
	defs = append(defs, s.aiToolsDBInstances(ctx)...)
	defs = append(defs, s.aiToolsContainerExtra(ctx)...)
	defs = append(defs, s.aiToolsSiteDomains(ctx)...)
	defs = append(defs, s.aiToolsRuntimeExtra(ctx)...)
	defs = append(defs, s.aiToolsStoreExtra(ctx)...)
	defs = append(defs, s.aiToolsVPN(ctx)...)
	defs = append(defs, s.aiToolsPanelAI(ctx)...)
	for i := range defs {
		fn := defs[i].Fn
		module, risk := defs[i].Module, defs[i].Risk
		defs[i].Fn = func(c context.Context, args string) (string, error) {
			if err := rbac.CheckTool(c, module, risk, args); err != nil {
				return "", err
			}
			return fn(c, args)
		}
	}
	return defs
}

// ToolFlags 工具开关表（无记录 = 启用）。
func (s *AIService) ToolFlags() map[string]bool {
	rows := []model.AIToolFlag{}
	_ = s.db.Find(&rows).Error
	m := make(map[string]bool, len(rows))
	for _, r := range rows {
		m[r.Name] = r.Enabled
	}
	return m
}

// aiToolsByModule 应用用户开关过滤后按模块聚合。
func (s *AIService) aiToolsByModule(ctx context.Context) map[string][]aiToolDef {
	flags := s.ToolFlags()
	m := make(map[string][]aiToolDef, len(aiModuleRegistry))
	for _, d := range s.aiToolsAll(ctx) {
		if enabled, ok := flags[d.Name]; ok && !enabled {
			continue
		}
		m[d.Module] = append(m[d.Module], d)
	}
	return m
}

// loadToolsDef 目录元工具：模型按需展开模块工具集（expanded 随会话循环存活，Fn 内更新）。
func (s *AIService) loadToolsDef(expanded map[string]bool, byMod map[string][]aiToolDef) aiToolDef {
	var catalog strings.Builder
	for _, m := range aiModuleRegistry {
		state := ""
		if expanded[m.Key] {
			state = "（已加载）"
		}
		fmt.Fprintf(&catalog, "- %s | %s | %s%s\n", m.Key, m.Title, m.Desc, state)
	}
	return aiToolDef{
		Name:   "load_tools",
		Module: aiModMeta,
		Risk:   aiRiskRead,
		Desc: "加载面板功能模块的工具集（当前会话默认只有常驻工具：系统概览/记忆/知识库等）。\n" +
			"模块目录（key | 名称 | 能力）：\n" + catalog.String() +
			"input 为 JSON：{\"module\":\"模块key\"}，多个用逗号分隔（如 \"databases,files\"）。加载成功后相关工具在后续直接可用；与用户当前所在页面相关的模块通常已预加载。",
		Parameters: schObj(map[string]any{
			"module": schStr("要加载的模块 key，多个用英文逗号分隔"),
		}, "module"),
		Fn: func(_ context.Context, input string) (string, error) {
			var p struct {
				Module string `json:"module"`
			}
			if err := json.Unmarshal([]byte(unwrapArgs(input)), &p); err != nil {
				return "", err
			}
			p.Module = strings.TrimSpace(p.Module)
			if p.Module == "" {
				return "", fmt.Errorf("缺少 module 参数")
			}
			known := map[string]bool{}
			for _, m := range aiModuleRegistry {
				known[m.Key] = true
			}
			var loaded, missing []string
			for _, key := range strings.Split(p.Module, ",") {
				key = strings.TrimSpace(key)
				if key == "" || key == aiModMeta {
					continue
				}
				if !known[key] {
					missing = append(missing, key)
					continue
				}
				if !expanded[key] {
					expanded[key] = true
				}
				if defs := byMod[key]; len(defs) > 0 {
					loaded = append(loaded, fmt.Sprintf("%s(%d 个)", key, len(defs)))
				} else {
					loaded = append(loaded, key+"（模块内工具全部被用户停用）")
				}
			}
			msg := "已加载模块工具：" + strings.Join(loaded, "、") + "，后续可直接调用。"
			if len(missing) > 0 {
				msg += " 未知的模块 key：" + strings.Join(missing, "、") + "。"
			}
			return msg, nil
		},
	}
}

// toolDefsFor 会话工具集：load_tools + 常驻（system/panel_ai）+ 场景预注入 + 已展开模块，全部经开关过滤。
// read_only 模式只保留 read 级工具（write/danger 不注入）。
func (s *AIService) toolDefsFor(ctx context.Context, scenePath string, expanded map[string]bool, mode string) []aiToolDef {
	byMod := s.aiToolsByModule(ctx)
	modules := []string{aiModSystem, aiModAI}
	modules = append(modules, sceneAIModules(scenePath)...)
	for key := range expanded {
		modules = append(modules, key)
	}
	seen := map[string]bool{aiModMeta: true}
	defs := []aiToolDef{s.loadToolsDef(expanded, byMod)}
	for _, key := range modules {
		if seen[key] {
			continue
		}
		seen[key] = true
		defs = append(defs, byMod[key]...)
	}
	if mode == aiModeReadOnly {
		readonly := defs[:0]
		for _, d := range defs {
			if d.Risk == aiRiskRead || d.Name == "load_tools" || d.Name == "ask_user" {
				readonly = append(readonly, d)
			}
		}
		defs = readonly
	}
	// 权限注入期过滤（M33）：调用者无对应权限点的工具直接不提供（而非执行时拒绝）；
	// 节点范围依赖具体入参，仍在执行期闸校验
	if caller, ok := rbac.CallerFrom(ctx); ok {
		allowed := defs[:0]
		for _, d := range defs {
			if rbac.Match(caller.PermSet, rbac.ToolPerm(d.Module, d.Risk)) {
				allowed = append(allowed, d)
			}
		}
		defs = allowed
	}
	return defs
}

// builtinTools 兼容旧接口（管理页清单/开关过滤基准）：全量工具的 langchaingo 形态。
// 模块按 aiModuleRegistry 注册顺序排列（map 遍历无序，直接迭代会导致管理页分组乱序）。
func (s *AIService) builtinTools(ctx context.Context) []tools.Tool {
	byMod := s.aiToolsByModule(ctx)
	defs := []aiToolDef{s.loadToolsDef(map[string]bool{}, byMod)}
	for _, m := range aiModuleRegistry {
		defs = append(defs, byMod[m.Key]...)
	}
	out := make([]tools.Tool, 0, len(defs))
	for i := range defs {
		out = append(out, &aiTool{def: defs[i]})
	}
	return out
}

// findToolDef 在会话工具集中定位工具声明。
func findToolDef(defs []aiToolDef, name string) (aiToolDef, bool) {
	for _, d := range defs {
		if d.Name == name {
			return d, true
		}
	}
	return aiToolDef{}, false
}

// ---- 管理页接口 ----

// ListTools 系统工具清单（名称/描述/模块/风险/开关）。
func (s *AIService) ListTools() []map[string]any {
	flags := s.ToolFlags()
	builtins := s.builtinTools(context.Background())
	out := make([]map[string]any, 0, len(builtins))
	for _, t := range builtins {
		enabled := true
		if v, ok := flags[t.Name()]; ok {
			enabled = v
		}
		def := t.(*aiTool).def
		out = append(out, map[string]any{
			"name": def.Name, "description": def.Desc, "module": def.Module,
			"moduleTitle": aiModuleTitle(def.Module), "risk": def.Risk, "enabled": enabled,
		})
	}
	return out
}

// SetToolFlag 设置工具开关。
func (s *AIService) SetToolFlag(name string, enabled bool) error {
	known := false
	for _, t := range s.aiToolsAll(context.Background()) {
		if t.Name == name {
			known = true
			break
		}
	}
	if name == "load_tools" {
		known = true
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

// ListOperationLogs 操作审计分页（写/danger 工具的执行与 ask 结果）。
func (s *AIService) ListOperationLogs(page, pageSize int) map[string]any {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	_ = s.db.Model(&model.AIOperationLog{}).Count(&total).Error
	rows := []model.AIOperationLog{}
	_ = s.db.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error
	return map[string]any{"total": total, "page": page, "pageSize": pageSize, "items": rows}
}
