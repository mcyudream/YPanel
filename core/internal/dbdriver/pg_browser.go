package dbdriver

import (
	"context"
	"time"

	"github.com/ypanel/shared/errs"
)

// ListTables pg 表清单（当前连接库 public schema）。
func (d *pgDriver) ListTables(ctx context.Context, database string) ([]TableInfo, error) {
	rows, err := d.pool.Query(ctx, `SELECT relname, GREATEST(n_live_tup,0), 0
		FROM pg_stat_all_tables WHERE schemaname='public'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TableInfo{}
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Name, &t.Rows, &t.SizeMB); err != nil {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

// Query 只读查询（当前库）。
func (d *pgDriver) Query(ctx context.Context, database, sqlText string, limit int) (*QueryResult, error) {
	if err := ValidateReadOnly(sqlText); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	start := time.Now()
	rows, err := d.pool.Query(ctx, sqlText)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer rows.Close()
	result := &QueryResult{}
	for i, fd := range rows.FieldDescriptions() {
		_ = i
		result.Columns = append(result.Columns, string(fd.Name))
	}
	for rows.Next() && len(result.Rows) < limit+1 {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, vals)
	}
	result.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return result, nil
}
