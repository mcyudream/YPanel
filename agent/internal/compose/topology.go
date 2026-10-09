package compose

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// Topology 输出项目服务拓扑：
//   - config --format json 取编排定义（镜像/depends_on/发布端口，extends/include 已展开）；
//   - ps --all --format json 取容器实时状态与健康。
//
// 两个命令都显式传 --project-name（项目名取编排文件目录名时会静默查空，见 exp）。
// ps 失败不阻塞：拓扑仍可展示编排结构，只是无实时状态。
func (m *Manager) Topology(ctx context.Context, name, dir string) (dto.ComposeTopology, error) {
	cfg, err := m.resolveProject(name, dir)
	if err != nil {
		return dto.ComposeTopology{}, err
	}
	out := dto.ComposeTopology{
		Name:  name,
		Dir:   filepath.Dir(cfg),
		Nodes: make([]dto.ComposeTopologyNode, 0),
		Edges: make([]dto.ComposeTopologyEdge, 0),
	}

	cj, code, err := runDocker(ctx, composeArgs(cfg, "--project-name", name, "config", "--format", "json")...)
	if err != nil {
		return out, err
	}
	if code != 0 {
		return out, errs.Wrapc(errs.CodeFileOpFailed, "compose config 失败: "+tailRunes(cj, 400))
	}
	var parsed composeConfigJSON
	if err := json.Unmarshal([]byte(cj), &parsed); err != nil {
		return out, errs.Wrapc(errs.CodeFileOpFailed, "解析 compose config 输出失败: "+err.Error())
	}

	live := map[string]composePsItem{}
	pj, _, perr := runDocker(ctx, composeArgs(cfg, "--project-name", name, "ps", "--all", "--format", "json")...)
	if perr == nil {
		for _, it := range parseComposePsItems(pj) {
			if it.Service != "" {
				live[it.Service] = it
			}
		}
	}

	for svc, s := range parsed.Services {
		node := dto.ComposeTopologyNode{Name: svc, Image: s.Image, Ports: make([]string, 0, len(s.Ports))}
		if li, ok := live[svc]; ok {
			node.State = li.State
			node.Health = li.Health
		}
		for _, p := range s.Ports {
			pub := numStr(p.Published)
			if pub == "" {
				continue
			}
			proto := p.Protocol
			if proto == "" {
				proto = "tcp"
			}
			node.Ports = append(node.Ports, pub+":"+numStr(p.Target)+"/"+proto)
		}
		sort.Strings(node.Ports)
		out.Nodes = append(out.Nodes, node)

		for dep, d := range s.DependsOn {
			cond := d.Condition
			if cond == "" {
				cond = "service_started"
			}
			out.Edges = append(out.Edges, dto.ComposeTopologyEdge{From: svc, To: dep, Condition: cond})
		}
	}
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].Name < out.Nodes[j].Name })
	sort.Slice(out.Edges, func(i, j int) bool {
		if out.Edges[i].From != out.Edges[j].From {
			return out.Edges[i].From < out.Edges[j].From
		}
		return out.Edges[i].To < out.Edges[j].To
	})
	return out, nil
}

// ---- `docker compose config --format json` 解析 ----

type composeConfigJSON struct {
	Name     string                        `json:"name"`
	Services map[string]composeServiceJSON `json:"services"`
}

type composeServiceJSON struct {
	Image     string                        `json:"image"`
	DependsOn composeDependsMap             `json:"depends_on"`
	Ports     []composeConfigPortJSON       `json:"ports"`
}

// composeDependsMap 兼容 depends_on 长语法（map）与短语法（字符串数组，config 已归一但防御旧版）。
type composeDependsMap map[string]composeDependsJSON

func (m *composeDependsMap) UnmarshalJSON(b []byte) error {
	*m = composeDependsMap{}
	t := strings.TrimSpace(string(b))
	if t == "" || t == "null" {
		return nil
	}
	if strings.HasPrefix(t, "{") {
		var long map[string]composeDependsJSON
		if err := json.Unmarshal(b, &long); err != nil {
			return err
		}
		for k, v := range long {
			(*m)[k] = v
		}
		return nil
	}
	var short []string
	if err := json.Unmarshal(b, &short); err != nil {
		return err
	}
	for _, k := range short {
		(*m)[k] = composeDependsJSON{}
	}
	return nil
}

type composeDependsJSON struct {
	Condition string `json:"condition"`
}

type composeConfigPortJSON struct {
	Published json.Number `json:"published"`
	Target    json.Number `json:"target"`
	Protocol  string      `json:"protocol"`
}

// numStr json.Number 空值兜底为空串（published/target 各版本可能为数字或字符串）。
func numStr(n json.Number) string {
	if n == "" {
		return ""
	}
	return string(n)
}

// ---- `docker compose ps --format json` 解析：v2.21+ 输出 JSON 数组，旧版为逐行 NDJSON ----

type composePsItem struct {
	Service string `json:"Service"`
	State   string `json:"State"`
	Health  string `json:"Health"`
}

func parseComposePsItems(s string) []composePsItem {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	if strings.HasPrefix(t, "[") {
		var arr []composePsItem
		if json.Unmarshal([]byte(t), &arr) == nil {
			return arr
		}
		return nil
	}
	out := make([]composePsItem, 0, 8)
	for _, line := range strings.Split(t, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var it composePsItem
		if json.Unmarshal([]byte(line), &it) == nil {
			out = append(out, it)
		}
	}
	return out
}

// tailRunes 取尾部 n 个字符（错误输出通常在尾部：真实错误不在 Pulling 进度首行）。
func tailRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return "…" + string(r[len(r)-n:])
}
