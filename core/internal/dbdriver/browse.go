// 只读浏览扩展：表清单 / 结构 / 只读查询（mysql、postgres）；redis/mongo 用各自语义。
package dbdriver

import (
	"context"
	"strings"

	"github.com/ypanel/shared/errs"
)

// TableInfo 表/集合信息。
type TableInfo struct {
	Name   string `json:"name"`
	Rows   int64  `json:"rows"`
	SizeMB float64 `json:"sizeMb"`
}

// QueryResult 只读查询结果。
type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Elapsed string          `json:"elapsed"`
}

// readOnlyPrefixes 只读语句白名单前缀。
var readOnlyPrefixes = []string{"select", "show", "desc", "describe", "explain"}

// ValidateReadOnly 校验 SQL 为只读语句。
func ValidateReadOnly(sql string) error {
	t := strings.TrimSpace(strings.ToLower(sql))
	for _, p := range readOnlyPrefixes {
		if strings.HasPrefix(t, p+" ") || t == p {
			return nil
		}
	}
	return errs.Wrap(errs.ErrBadRequest, "只读查询器仅允许 SELECT/SHOW/DESC/EXPLAIN 语句")
}

// TableBrowser 表浏览能力（mysql/postgres/redis/mongo 各自实现）。
type TableBrowser interface {
	ListTables(ctx context.Context, database string) ([]TableInfo, error)
	Query(ctx context.Context, database, sqlText string, limit int) (*QueryResult, error)
}

