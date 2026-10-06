package dbdriver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ypanel/shared/errs"
)

// ListTables mysql 表清单（含行数与大小）。
func (d *mysqlDriver) ListTables(ctx context.Context, database string) ([]TableInfo, error) {
	if err := ValidateIdent(database); err != nil {
		return nil, err
	}
	rows, err := d.db.QueryContext(ctx,
		`SELECT table_name, IFNULL(table_rows,0), IFNULL((data_length+index_length),0)
		 FROM information_schema.tables WHERE table_schema = ?`, database)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []TableInfo{}
	for rows.Next() {
		var t TableInfo
		var size int64
		if err := rows.Scan(&t.Name, &t.Rows, &size); err != nil {
			return nil, err
		}
		t.SizeMB = float64(size) / 1024 / 1024
		out = append(out, t)
	}
	return out, rows.Err()
}

// Query 只读查询（USE 切库 + LIMIT 限制）。
func (d *mysqlDriver) Query(ctx context.Context, database, sqlText string, limit int) (*QueryResult, error) {
	if err := ValidateReadOnly(sqlText); err != nil {
		return nil, err
	}
	if err := ValidateIdent(database); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if _, err := d.db.ExecContext(ctx, fmt.Sprintf("USE %s", database)); err != nil {
		return nil, err
	}
	start := time.Now()
	finalSQL := strings.TrimRight(strings.TrimSpace(sqlText), ";")
	// LIMIT 仅适用于 SELECT（SHOW/DESC 等不支持追加 LIMIT）
	if strings.HasPrefix(strings.ToLower(finalSQL), "select") && !strings.Contains(strings.ToLower(finalSQL), " limit ") {
		finalSQL += " LIMIT 500"
	}
	rows, err := d.db.QueryContext(ctx, finalSQL)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := &QueryResult{Columns: cols, Elapsed: time.Since(start).Round(time.Millisecond).String()}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		for i, v := range vals {
			if b, ok := v.([]byte); ok {
				vals[i] = string(b)
			}
		}
		result.Rows = append(result.Rows, vals)
	}
	return result, rows.Err()
}
