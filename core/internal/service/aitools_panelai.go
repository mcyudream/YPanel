// aitools_panelai.go AI 自身域工具：长期记忆 + 知识库深读（M31，常驻无需 load_tools）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ypanel/core/internal/model"
)

// aiToolsPanelAI AI 自身域工具集。
func (s *AIService) aiToolsPanelAI(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "ask_user", Module: aiModAI, Risk: aiRiskRead,
			Desc: "向用户提出澄清问题（需求有歧义或存在多种做法需要用户拍板时使用；一次 1-4 个问题，每题 2-4 个选项）。用户回答前会挂起等待，回答随结果返回；label 简短（≤12 字），description 一句话说明差异；不要在无歧义时滥用。input JSON：{\"questions\":[{\"question\":\"完整问题？\",\"header\":\"短标签\",\"multiSelect\":false,\"options\":[{\"label\":\"选项A\",\"description\":\"差异说明\"},{\"label\":\"选项B\",\"description\":\"...\"}]}]}",
			Parameters: schObj(map[string]any{
				"questions": schArr("问题列表（1-4 个）", schObj(map[string]any{
					"question":    schStr("完整问题文本"),
					"header":      schStr("短标签（≤12 字，如 部署方式）"),
					"multiSelect": schBool("是否多选，默认单选"),
					"options":     schArr("2-4 个选项", schObj(map[string]any{
						"label":       schStr("选项短文本"),
						"description": schStr("该选项含义/影响"),
					})),
				})),
			}, "questions"),
			Fn: func(_ context.Context, _ string) (string, error) {
				return "", nil // 实际执行在 aichat.go 循环特判（需 emit SSE 事件），此处不会走到
			},
		},
		{
			Name: "save_memory", Module: aiModAI, Risk: aiRiskWrite,
			Desc: "把本次对话中值得长期记住的运维经验/用户偏好/服务器特性沉淀为记忆（下次对话自动可用）。input 为一句话记忆内容（纯文本或 {\"content\":\"...\"}）。",
			Parameters: schObj(map[string]any{"content": schStr("记忆内容一句话")}),
			Fn: func(_ context.Context, input string) (string, error) {
				content := strings.TrimSpace(input)
				// 兼容 {"content":"..."} / {"input":"..."} 包装，取内层纯文本
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
		{
			Name: "read_knowledge", Module: aiModAI, Risk: aiRiskRead,
			Desc: "按需深读知识库（回答引用了知识片段后需更多上下文时使用）。input JSON 二选一：{\"search\":\"关键词\"} 返回命中清单；{\"doc\":\"文档标题关键词\"} 返回全文（截断 6000 字）。",
			Parameters: schObj(map[string]any{
				"search": schStr("关键词检索"), "doc": schStr("文档标题关键词，返回全文"),
			}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Search string `json:"search"`
					Doc    string `json:"doc"`
				}](input)
				if err != nil {
					return "", err
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
					return toolOut(map[string]any{"title": docs[0].Title, "content": content})
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
