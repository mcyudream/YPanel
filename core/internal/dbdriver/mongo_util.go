package dbdriver

import "strings"

// stringsCollectionName 从查询语句提取集合名（首词）。
func stringsCollectionName(sqlText string) string {
	f := strings.Fields(strings.TrimSpace(strings.TrimRight(sqlText, ";")))
	if len(f) == 0 {
		return ""
	}
	return f[0]
}
