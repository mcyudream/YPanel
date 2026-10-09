package dbdriver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ypanel/shared/errs"
)

// tableRefMySQL 库表引用（标识符已过白名单后反引号引用）。
func tableRefMySQL(database, table string) (string, error) {
	if err := ValidateIdent(database); err != nil {
		return "", err
	}
	if err := ValidateIdent(table); err != nil {
		return "", err
	}
	return fmt.Sprintf("`%s`.`%s`", database, table), nil
}

// ListTables mysql 表清单（含行数/大小/注释，区分表与视图）。
func (d *mysqlDriver) ListTables(ctx context.Context, database, schema string) ([]TableInfo, error) {
	if err := ValidateIdent(database); err != nil {
		return nil, err
	}
	rows, err := d.db.QueryContext(ctx,
		`SELECT table_name, IFNULL(table_rows,0), IFNULL((data_length+index_length),0), IFNULL(table_comment,''), table_type
		 FROM information_schema.tables WHERE table_schema = ? ORDER BY table_name`, database)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []TableInfo{}
	for rows.Next() {
		var t TableInfo
		var size int64
		var typ string
		if err := rows.Scan(&t.Name, &t.Rows, &size, &t.Comment, &typ); err != nil {
			return nil, err
		}
		t.SizeMB = float64(size) / 1024 / 1024
		if typ == "VIEW" {
			t.Kind = "view"
		} else {
			t.Kind = "table"
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Schemas mysql 无 schema 概念。
func (d *mysqlDriver) Schemas(context.Context, string) ([]string, error) {
	return []string{}, nil
}

// RunQuery 自由只读查询（USE 切库 + 自动 LIMIT）。
func (d *mysqlDriver) RunQuery(ctx context.Context, database, sqlText string, limit int) (*QueryResult, error) {
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
		finalSQL += fmt.Sprintf(" LIMIT %d", limit)
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

// BrowsePage 表浏览真分页（总数 + 主键/指定列排序）。
func (d *mysqlDriver) BrowsePage(ctx context.Context, req BrowsePageReq) (*PagedResult, error) {
	ref, err := tableRefMySQL(req.DB, req.Table)
	if err != nil {
		return nil, err
	}
	if req.Limit <= 0 || req.Limit > 500 {
		req.Limit = 200
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	order := ""
	if req.SortCol != "" {
		if req.SortDir != "desc" {
			req.SortDir = "asc"
		}
		if err := ValidateIdent(req.SortCol); err != nil {
			return nil, err
		}
		order = fmt.Sprintf(" ORDER BY `%s` %s", req.SortCol, strings.ToUpper(req.SortDir))
	} else if pks, _ := d.PrimaryKey(ctx, req.DB, "", req.Table); len(pks) > 0 {
		order = fmt.Sprintf(" ORDER BY `%s` ASC", pks[0])
	}
	start := time.Now()
	var total int64
	if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+ref).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := d.db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s%s LIMIT %d OFFSET %d", ref, order, req.Limit, req.Offset))
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := &PagedResult{Columns: cols, Total: total, Offset: req.Offset, Limit: req.Limit, Elapsed: time.Since(start).Round(time.Millisecond).String()}
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
		out.Rows = append(out.Rows, vals)
	}
	return out, rows.Err()
}

// DescribeColumns 字段结构。
func (d *mysqlDriver) DescribeColumns(ctx context.Context, database, _ string, table string) ([]ColumnInfo, error) {
	if err := ValidateIdent(database); err != nil {
		return nil, err
	}
	if err := ValidateIdent(table); err != nil {
		return nil, err
	}
	rows, err := d.db.QueryContext(ctx,
		`SELECT column_name, column_type, is_nullable, column_default, column_key, IFNULL(extra,''), IFNULL(column_comment,'')
		 FROM information_schema.columns WHERE table_schema = ? AND table_name = ? ORDER BY ordinal_position`, database, table)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []ColumnInfo{}
	for rows.Next() {
		var c ColumnInfo
		var nullable, key string
		var def *string
		if err := rows.Scan(&c.Name, &c.DataType, &nullable, &def, &key, &c.Extra, &c.Comment); err != nil {
			return nil, err
		}
		c.Nullable = nullable == "YES"
		c.Key = key
		c.Default = def
		out = append(out, c)
	}
	return out, rows.Err()
}

// TableDDL 建表语句（SHOW CREATE TABLE）。
func (d *mysqlDriver) TableDDL(ctx context.Context, database, _ string, table string) (string, error) {
	ref, err := tableRefMySQL(database, table)
	if err != nil {
		return "", err
	}
	var name, ddl string
	if err := d.db.QueryRowContext(ctx, "SHOW CREATE TABLE "+ref).Scan(&name, &ddl); err != nil {
		return "", err
	}
	return ddl, nil
}

// ListIndexes 索引清单（information_schema 固定列，兼容各版本；聚合同名多列）。
func (d *mysqlDriver) ListIndexes(ctx context.Context, database, _ string, table string) ([]IndexInfo, error) {
	if err := ValidateIdent(database); err != nil {
		return nil, err
	}
	if err := ValidateIdent(table); err != nil {
		return nil, err
	}
	rows, err := d.db.QueryContext(ctx,
		`SELECT index_name, non_unique, seq_in_index, column_name
		 FROM information_schema.statistics
		 WHERE table_schema = ? AND table_name = ?
		 ORDER BY index_name, seq_in_index`, database, table)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []IndexInfo{}
	pos := map[string]int{}
	for rows.Next() {
		var indexName, colName string
		var nonUnique, seqInIndex int64
		if err := rows.Scan(&indexName, &nonUnique, &seqInIndex, &colName); err != nil {
			return nil, err
		}
		idx := -1
		for i := range out {
			if out[i].Name == indexName {
				idx = i
				break
			}
		}
		if idx < 0 {
			out = append(out, IndexInfo{Name: indexName, Unique: nonUnique == 0, Primary: indexName == "PRIMARY"})
			idx = len(out) - 1
			pos[indexName] = 0
		}
		// 按 seq_in_index 占位插入，保持列序
		want := int(seqInIndex)
		for len(out[idx].Columns) < want {
			out[idx].Columns = append(out[idx].Columns, "")
		}
		out[idx].Columns[want-1] = colName
		pos[indexName] = want
	}
	// 清理占位空串
	for i := range out {
		cols := make([]string, 0, len(out[i].Columns))
		for _, c := range out[i].Columns {
			if c != "" {
				cols = append(cols, c)
			}
		}
		out[i].Columns = cols
	}
	return out, rows.Err()
}

// CreateIndex 创建索引。
func (d *mysqlDriver) CreateIndex(ctx context.Context, req IndexRequest) error {
	ref, err := tableRefMySQL(req.DB, req.Table)
	if err != nil {
		return err
	}
	if len(req.Columns) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "索引至少需要一列")
	}
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("idx_%s_%s", req.Table, req.Columns[0])
	}
	if err := ValidateIdent(name); err != nil {
		return err
	}
	cols := make([]string, 0, len(req.Columns))
	for _, c := range req.Columns {
		if err := ValidateIdent(c); err != nil {
			return err
		}
		cols = append(cols, "`"+c+"`")
	}
	unique := ""
	if req.Unique {
		unique = "UNIQUE "
	}
	_, err = d.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD %sINDEX `%s` (%s)", ref, unique, name, strings.Join(cols, ",")))
	return err
}

// DropIndex 删除索引（主键不支持经此删除）。
func (d *mysqlDriver) DropIndex(ctx context.Context, database, _ string, table, name string) error {
	ref, err := tableRefMySQL(database, table)
	if err != nil {
		return err
	}
	if name == "PRIMARY" {
		return errs.Wrap(errs.ErrBadRequest, "主键不支持删除")
	}
	if err := ValidateIdent(name); err != nil {
		return err
	}
	_, err = d.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s DROP INDEX `%s`", ref, name))
	return err
}

// PrimaryKey 主键列（按序）。
func (d *mysqlDriver) PrimaryKey(ctx context.Context, database, _ string, table string) ([]string, error) {
	if err := ValidateIdent(database); err != nil {
		return nil, err
	}
	if err := ValidateIdent(table); err != nil {
		return nil, err
	}
	rows, err := d.db.QueryContext(ctx,
		`SELECT column_name FROM information_schema.key_column_usage
		 WHERE table_schema = ? AND table_name = ? AND constraint_name = 'PRIMARY' ORDER BY ordinal_position`, database, table)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// rowEditRef 行编辑公共校验：库表白名单 + 主键列白名单。
func rowEditRef(req RowEdit) (string, []string, error) {
	ref, err := tableRefMySQL(req.DB, req.Table)
	if err != nil {
		return "", nil, err
	}
	if len(req.PK) == 0 {
		return "", nil, errs.Wrap(errs.ErrBadRequest, "缺少主键条件")
	}
	pkCols := make([]string, 0, len(req.PK))
	for col := range req.PK {
		if err := ValidateIdent(col); err != nil {
			return "", nil, err
		}
		pkCols = append(pkCols, col)
	}
	return ref, pkCols, nil
}

// InsertRow 插入行（列与值参数化）。
func (d *mysqlDriver) InsertRow(ctx context.Context, req RowEdit) error {
	if err := ValidateIdent(req.DB); err != nil {
		return err
	}
	if err := ValidateIdent(req.Table); err != nil {
		return err
	}
	if len(req.Values) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "缺少数据")
	}
	cols := make([]string, 0, len(req.Values))
	ph := make([]string, 0, len(req.Values))
	vals := make([]any, 0, len(req.Values))
	for col, v := range req.Values {
		if err := ValidateIdent(col); err != nil {
			return err
		}
		cols = append(cols, "`"+col+"`")
		ph = append(ph, "?")
		vals = append(vals, v)
	}
	sql := fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES (%s)", req.DB, req.Table, strings.Join(cols, ","), strings.Join(ph, ","))
	_, err := d.db.ExecContext(ctx, sql, vals...)
	return err
}

// UpdateRow 按主键更新（SET 与 WHERE 全参数化）。
func (d *mysqlDriver) UpdateRow(ctx context.Context, req RowEdit) error {
	ref, pkCols, err := rowEditRef(req)
	if err != nil {
		return err
	}
	if len(req.Values) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "缺少更新数据")
	}
	sets := make([]string, 0, len(req.Values))
	vals := make([]any, 0, len(req.Values)+len(req.PK))
	for col, v := range req.Values {
		if err := ValidateIdent(col); err != nil {
			return err
		}
		sets = append(sets, "`"+col+"` = ?")
		vals = append(vals, v)
	}
	wheres := make([]string, 0, len(pkCols))
	for _, col := range pkCols {
		wheres = append(wheres, "`"+col+"` = ?")
		vals = append(vals, req.PK[col])
	}
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s LIMIT 1", ref, strings.Join(sets, ","), strings.Join(wheres, " AND "))
	res, err := d.db.ExecContext(ctx, sql, vals...)
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errs.Wrap(errs.ErrBadRequest, "未找到匹配行（主键值可能已变更）")
	}
	return nil
}

// DeleteRow 按主键删除。
func (d *mysqlDriver) DeleteRow(ctx context.Context, req RowEdit) error {
	ref, pkCols, err := rowEditRef(req)
	if err != nil {
		return err
	}
	wheres := make([]string, 0, len(pkCols))
	vals := make([]any, 0, len(pkCols))
	for _, col := range pkCols {
		wheres = append(wheres, "`"+col+"` = ?")
		vals = append(vals, req.PK[col])
	}
	sql := fmt.Sprintf("DELETE FROM %s WHERE %s LIMIT 1", ref, strings.Join(wheres, " AND "))
	res, err := d.db.ExecContext(ctx, sql, vals...)
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errs.Wrap(errs.ErrBadRequest, "未找到匹配行")
	}
	return nil
}
