// SkillsManager AI 技能包（B18+）：SKILL.md 文件格式（YAML frontmatter + Markdown 正文），
// 存储于 /opt/ypanel/ai/skills/<name>/SKILL.md，可经文件通道/scp 直接编辑。
// 启用状态存 settings（逗号分隔白名单）。仅启用的技能注入对话 system。
package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/ypanel/shared/dto"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/errs"
)

// SkillsManager 技能包管理。
type SkillsManager struct {
	nodes    *NodeService
	settings *SettingService
}

// NewSkillsManager 创建。
func NewSkillsManager(nodes *NodeService, settings *SettingService) *SkillsManager {
	return &SkillsManager{nodes: nodes, settings: settings}
}

const skillsDir = "/opt/ypanel/ai/skills"

// AISkill 技能条目。
type AISkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Body        string `json:"body"`
	Enabled     bool   `json:"enabled"`
}

func (s *SkillsManager) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

func (s *SkillsManager) exec(ctx context.Context, command string, timeout int) (*dto.ExecResp, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	return agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: command, TimeoutSecs: timeout})
}

// enabledSet 启用白名单。
func (s *SkillsManager) enabledSet() map[string]bool {
	raw := s.settings.Get("ai.skills_enabled", "")
	out := map[string]bool{}
	for _, n := range strings.Split(raw, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out[n] = true
		}
	}
	return out
}

// SetEnabled 设置技能启停。
func (s *SkillsManager) SetEnabled(name string, enabled bool) error {
	cur := s.enabledSet()
	if enabled {
		cur[name] = true
	} else {
		delete(cur, name)
	}
	names := make([]string, 0, len(cur))
	for n := range cur {
		names = append(names, n)
	}
	sort.Strings(names)
	return s.settings.Set("ai.skills_enabled", strings.Join(names, ","))
}

// parseSKILLMD 解析 frontmatter（name/description）与正文。
func parseSKILLMD(content string) (name, desc, body string) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return "", "", content
	}
	end := strings.Index(content[3:], "\n---")
	if end < 0 {
		return "", "", content
	}
	fm := content[3 : end+3]
	body = strings.TrimSpace(content[end+7:])
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.Trim(strings.TrimPrefix(line, "name:"), ` "`)
		}
		if strings.HasPrefix(line, "description:") {
			desc = strings.Trim(strings.TrimPrefix(line, "description:"), ` "`)
		}
	}
	return name, desc, body
}

// List 技能列表（远端目录扫描，按文件边界解析）。
func (s *SkillsManager) List(ctx context.Context) ([]AISkill, error) {
	out, err := s.exec(ctx, fmt.Sprintf(
		`for f in %s/*/SKILL.md; do [ -f "$f" ] && echo "===FILE:$f===" && cat "$f" && echo; done`, skillsDir), 30)
	if err != nil {
		return nil, err
	}
	enabled := s.enabledSet()
	list := []AISkill{}
	for _, doc := range strings.Split(out.Output, "===FILE:") {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		eq := strings.Index(doc, "===")
		if eq < 0 {
			continue
		}
		name, desc, body := parseSKILLMD(doc[eq+3:])
		if name == "" {
			continue
		}
		list = append(list, AISkill{Name: name, Description: desc, Body: body, Enabled: enabled[name]})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list, nil
}

// EnabledBodies 启用技能的正文（注入对话 system）。
func (s *SkillsManager) EnabledBodies(ctx context.Context) []string {
	list, err := s.List(ctx)
	if err != nil {
		return nil
	}
	var out []string
	for _, sk := range list {
		if sk.Enabled {
			out = append(out, "【技能:"+sk.Name+"】"+sk.Description+"\n"+sk.Body)
		}
	}
	return out
}

// Save 创建/更新技能（内容经 base64 中转防 shell 引号注入）。
func (s *SkillsManager) Save(ctx context.Context, name, description, body string) error {
	if name == "" || strings.ContainsAny(name, "/ ;|&$<>*?~") {
		return errs.Wrap(errs.ErrBadRequest, "技能名不合法")
	}
	content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n", name, description, body)
	b64 := base64.StdEncoding.EncodeToString([]byte(content))
	dir := path.Join(skillsDir, name)
	target := path.Join(dir, "SKILL.md")
	out, err := s.exec(ctx, fmt.Sprintf("mkdir -p '%s' && echo %s | base64 -d > '%s'", dir, b64, target), 30)
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "技能保存失败: "+strings.TrimSpace(out.Output))
	}
	return nil
}

// Remove 删除技能目录。
func (s *SkillsManager) Remove(ctx context.Context, name string) error {
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return errs.Wrap(errs.ErrBadRequest, "技能名不合法")
	}
	_, err := s.exec(ctx, fmt.Sprintf("rm -rf '%s'", path.Join(skillsDir, name)), 15)
	return err
}

