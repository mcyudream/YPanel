// aitools_summary.go 工具输出的人类可读摘要（M32）：时间线卡片/审计展示用，
// 模型上下文仍拿完整结果。列表类按工具定制行渲染，其余 JSON 键值平铺，非 JSON 原样。
package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

// rowRendererOf 按工具名返回列表行的定制渲染（无定制返回 nil 走通用键值平铺）。
func rowRendererOf(tool string) func(map[string]any) string {
	renderers := map[string]func(map[string]any) string{
		"list_containers": func(m map[string]any) string {
			return fmt.Sprintf("%s[%s]", strOf(m, "name", "id"), strOf(m, "state", "status"))
		},
		"list_images": func(m map[string]any) string {
			return strOf(m, "name", "repository", "id")
		},
		"list_networks": func(m map[string]any) string {
			return strOf(m, "name", "id")
		},
		"list_volumes": func(m map[string]any) string {
			return strOf(m, "name")
		},
		"list_compose_projects": func(m map[string]any) string {
			return fmt.Sprintf("%s(%s)", strOf(m, "name"), fmt.Sprintf("%s/%s 运行", strOf(m, "running"), strOf(m, "total")))
		},
		"list_files": func(m map[string]any) string {
			name := strOf(m, "name")
			if isDir(m) {
				return name + "/"
			}
			if size, ok := m["size"].(float64); ok && size > 0 {
				return fmt.Sprintf("%s(%s)", name, humanSize(size))
			}
			return name
		},
		"search_files": func(m map[string]any) string {
			return strOf(m, "path", "name")
		},
		"list_processes": func(m map[string]any) string {
			return fmt.Sprintf("%s(pid %s, cpu %s%%)", strOf(m, "name"), strOf(m, "pid"), strOf(m, "cpu"))
		},
		"list_services": func(m map[string]any) string {
			return fmt.Sprintf("%s(%s)", strOf(m, "name"), strOf(m, "active"))
		},
		"list_database_instances": func(m map[string]any) string {
			return fmt.Sprintf("%s(%s@%s:%s)", strOf(m, "name"), strOf(m, "type"), strOf(m, "host"), strOf(m, "port"))
		},
		"list_databases":    func(m map[string]any) string { return strOf(m, "name") },
		"list_tables":       func(m map[string]any) string { return strOf(m, "name", "tableName") },
		"list_db_columns":   func(m map[string]any) string { return strOf(m, "name", "column", "field") },
		"list_db_users":     func(m map[string]any) string { return strOf(m, "name", "user") },
		"list_db_backups":   func(m map[string]any) string { return strOf(m, "name", "file", "filename") },
		"redis_scan_keys":   func(m map[string]any) string { return fmt.Sprintf("%s(%s)", strOf(m, "name"), strOf(m, "type")) },
		"list_sites":        func(m map[string]any) string { return strOf(m, "name", "domain") },
		"list_certs":        func(m map[string]any) string { return strOf(m, "domain", "name") },
		"list_runtimes":     func(m map[string]any) string { return strOf(m, "name", "type") },
		"store_search_apps": func(m map[string]any) string { return strOf(m, "title", "name", "key") },
		"list_installed_apps": func(m map[string]any) string {
			return fmt.Sprintf("%s(%s)", strOf(m, "name", "appName"), strOf(m, "project", "composeProject"))
		},
		"list_nodes":   func(m map[string]any) string { return strOf(m, "name") },
		"list_tasks":   func(m map[string]any) string { return strOf(m, "name") },
		"list_fail2ban": func(m map[string]any) string { return strOf(m, "name", "jail") },
	}
	if r, ok := renderers[tool]; ok {
		return r
	}
	return nil
}

func strOf(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return "?"
}

func isDir(m map[string]any) bool {
	b, ok := m["isDir"].(bool)
	return ok && b
}

func humanSize(bytes float64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1fGB", bytes/(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1fMB", bytes/(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1fKB", bytes/(1<<10))
	default:
		return fmt.Sprintf("%.0fB", bytes)
	}
}

// toolSummary 工具输出摘要：列表渲染「共 N 条：行1；行2…」，对象键值平铺，文本原样。
func toolSummary(def aiToolDef, out string) string {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return "(无输出)"
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return truncText(strings.ReplaceAll(trimmed, "\n", " "), 300)
	}
	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		return truncText(strings.ReplaceAll(trimmed, "\n", " "), 300)
	}
	render := rowRendererOf(def.Name)
	switch data := v.(type) {
	case []any:
		return renderRows(def, len(data), data, render)
	case map[string]any:
		total := -1
		if t, ok := data["total"].(float64); ok {
			total = int(t)
		}
		if items, ok := data["items"].([]any); ok {
			n := len(items)
			if total < 0 {
				total = n
			}
			return renderRows(def, total, items, render)
		}
		// 单对象：键值平铺（跳过长文本/嵌套）
		return truncText(compactKV(data), 300)
	}
	return truncText(strings.ReplaceAll(trimmed, "\n", " "), 300)
}

func renderRows(def aiToolDef, total int, items []any, render func(map[string]any) string) string {
	head := fmt.Sprintf("共 %d 条", total)
	if len(items) < total {
		head += fmt.Sprintf("（显示前 %d 条）", len(items))
	}
	if len(items) == 0 {
		return head
	}
	parts := make([]string, 0, len(items))
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			if render != nil {
				parts = append(parts, render(m))
			} else {
				parts = append(parts, compactKV(m))
			}
		} else {
			parts = append(parts, fmt.Sprint(it))
		}
	}
	return truncText(head+"："+strings.Join(parts, "；"), 420)
}

// compactKV 单对象紧凑键值平铺（最多 8 键，跳过嵌套与长文本）。
func compactKV(m map[string]any) string {
	parts := make([]string, 0, 8)
	for k, v := range m {
		if len(parts) >= 8 {
			break
		}
		switch val := v.(type) {
		case string:
			if val != "" && len(val) < 80 && !sensitiveKey(k) {
				parts = append(parts, fmt.Sprintf("%s=%s", k, val))
			}
		case float64:
			parts = append(parts, fmt.Sprintf("%s=%v", k, val))
		case bool:
			parts = append(parts, fmt.Sprintf("%s=%v", k, val))
		}
	}
	if len(parts) == 0 {
		return "(对象)"
	}
	return "{" + strings.Join(parts, " ") + "}"
}
