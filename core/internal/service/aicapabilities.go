package service

import (
	"encoding/json"
	"fmt"
	"github.com/ypanel/core/internal/model"
)

func validateAIModelCapabilities(raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > 256000 {
		return errWrapAI("模型能力配置过大")
	}
	var models map[string]AIModelCapabilities
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return errWrapAI("模型能力需为以模型名称为键的 JSON 对象")
	}
	if models == nil {
		return errWrapAI("模型能力不能为空对象以外的值")
	}
	for name, c := range models {
		if c.MaxTokensField != "" && c.MaxTokensField != "max_tokens" && c.MaxTokensField != "max_completion_tokens" {
			return errWrapAI("maxTokensField 仅支持 max_tokens / max_completion_tokens")
		}
		if name == "" || c.ContextLimit < 0 || c.ContextLimit > 2000000 || (c.ContextLimit > 0 && c.ContextLimit < 2048) || c.MaxOutputTokens < 0 || (c.MaxOutputTokens > 0 && c.ContextLimit > 0 && c.MaxOutputTokens >= c.ContextLimit) {
			return errWrapAI(fmt.Sprintf("模型 %s 的能力预算无效", name))
		}
	}
	return nil
}
func filterAISettingsTools(defs []aiToolDef, c AIChatSettings, p *model.AIProvider) []aiToolDef {
	cap := modelCapabilities(p)
	if !c.Tools || (cap.Tools != nil && !*cap.Tools) {
		return nil
	}
	out := defs[:0]
	for _, d := range defs {
		if !c.Memory && d.Name == "save_memory" {
			continue
		}
		if !c.Knowledge && d.Name == "read_knowledge" {
			continue
		}
		out = append(out, d)
	}
	return out
}
