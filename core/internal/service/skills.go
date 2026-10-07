// SkillsManager AI 技能包（B18+）：SKILL.md 文件格式（YAML frontmatter + Markdown 正文），
// 存储于 /opt/ypanel/ai/skills/<name>/SKILL.md，可经文件通道/scp 直接编辑。
// 启用状态存 settings（逗号分隔白名单）。仅启用的技能注入对话 system。
package service

import (
	"archive/zip"
	"bytes"
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

// UploadZip 上传技能包压缩包：解压定位 SKILL.md（zip 根或唯一一级子目录），
// 按 frontmatter name 把整个技能目录重整为 /opt/ypanel/ai/skills/<name>/（SKILL.md 在根）。
// 重名覆盖；防 zip slip；限制：zip ≤10MB、条目 ≤200、单文件 ≤2MB。
func (s *SkillsManager) UploadZip(ctx context.Context, data []byte) (string, string, error) {
	if len(data) == 0 || len(data) > 10<<20 {
		return "", "", errs.Wrap(errs.ErrBadRequest, "压缩包为空或超过 10MB")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", "", errs.Wrap(errs.ErrBadRequest, "不是有效的 zip 压缩包")
	}
	if len(zr.File) > 200 {
		return "", "", errs.Wrap(errs.ErrBadRequest, "压缩包含文件过多（上限 200）")
	}
	// 定位 SKILL.md 与其所在基准目录
	base := ""
	found := false
	for _, f := range zr.File {
		n := path.Clean(f.Name)
		if n == "SKILL.md" {
			base, found = "", true
			break
		}
		if strings.HasSuffix(n, "/SKILL.md") {
			parent := path.Dir(n)
			if !strings.Contains(parent, "/") {
				base, found = parent, true
				break
			}
		}
	}
	if !found {
		return "", "", errs.Wrap(errs.ErrBadRequest, "压缩包内未找到 SKILL.md（支持根目录或一级子目录）")
	}
	// 读 SKILL.md 解析 name
	var skillMD *zip.File
	relFiles := make([]*zip.File, 0, len(zr.File))
	var total uint64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		n := path.Clean(f.Name)
		if strings.HasPrefix(n, "__MACOSX/") || strings.HasSuffix(n, ".DS_Store") {
			continue
		}
		if base != "" && !strings.HasPrefix(n, base+"/") {
			continue
		}
		if f.UncompressedSize64 > 2<<20 {
			return "", "", errs.Wrap(errs.ErrBadRequest, "单文件超过 2MB: "+n)
		}
		total += f.UncompressedSize64
		if total > 20<<20 {
			return "", "", errs.Wrap(errs.ErrBadRequest, "解压后总量超过 20MB")
		}
		rel := strings.TrimPrefix(n, base)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "SKILL.md" {
			skillMD = f
			continue
		}
		if rel == "" || strings.HasPrefix(rel, "..") {
			continue
		}
		relFiles = append(relFiles, f)
	}
	if skillMD == nil {
		return "", "", errs.Wrap(errs.ErrBadRequest, "压缩包内未找到 SKILL.md")
	}
	rc, err := skillMD.Open()
	if err != nil {
		return "", "", err
	}
	mdBuf := new(bytes.Buffer)
	if _, err := mdBuf.ReadFrom(rc); err != nil {
		_ = rc.Close()
		return "", "", err
	}
	_ = rc.Close()
	name, desc, _ := parseSKILLMD(mdBuf.String())
	if name == "" || strings.ContainsAny(name, "/ ;|&$<>*?~") {
		return "", "", errs.Wrap(errs.ErrBadRequest, "SKILL.md frontmatter 缺少合法的 name 字段")
	}
	// 重整：先清旧目录，再写 SKILL.md + 其余文件（相对路径）
	if err := s.Remove(ctx, name); err != nil {
		return "", "", err
	}
	writeFile := func(rel, contentB64 string) error {
		target := path.Join(skillsDir, name, rel)
		dir := path.Dir(target)
		out, err := s.exec(ctx, fmt.Sprintf("mkdir -p '%s' && echo %s | base64 -d > '%s'", dir, contentB64, target), 30)
		if err != nil {
			return err
		}
		if out.ExitCode != 0 {
			return errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "写入技能文件失败: "+strings.TrimSpace(out.Output))
		}
		return nil
	}
	b64 := base64.StdEncoding.EncodeToString(mdBuf.Bytes())
	if err := writeFile("SKILL.md", b64); err != nil {
		return "", "", err
	}
	for _, f := range relFiles {
		rc, err := f.Open()
		if err != nil {
			return "", "", err
		}
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(rc); err != nil {
			_ = rc.Close()
			return "", "", err
		}
		_ = rc.Close()
		if err := writeFile(path.Clean(strings.TrimPrefix(f.Name, base)), base64.StdEncoding.EncodeToString(buf.Bytes())); err != nil {
			return "", "", err
		}
	}
	return name, desc, nil
}
