package dbdriver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ypanel/shared/errs"
)

// pgTableRef schema 表引用（标识符已过白名单，%q 引用兼容大小写）。
func pgTableRef(schema, table string) (string, error) {
	if err := ValidateIdent(table); err != nil {
		return "", err
	}
	if schema == "" {
		schema = "public"
	}
	if err := ValidateIdent(schema); err != nil {
		return "", err
	}
	return fmt.Sprintf(`%q.%q`, schema, table), nil
}

// Schemas 非系统 schema 列表。
func (d *pgDriver) Schemas(ctx context.Context, database string) ([]string, error) {
	p, err := d.poolFor(ctx, database)
	if err != nil {
		return nil, err
	}
	rows, err := p.Query(ctx, `SELECT nspname FROM pg_namespace
		WHERE nspname NOT LIKE 'pg\_%' AND nspname != 'information_schema' ORDER BY nspname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListTables 指定库 schema 的表与视图（含行数估计/大小/注释）。
func (d *pgDriver) ListTables(ctx context.Context, database, schema string) ([]TableInfo, error) {
	p, err := d.poolFor(ctx, database)
	if err != nil {
		return nil, err
	}
	if schema == "" {
		schema = "public"
	}
	if err := ValidateIdent(schema); err != nil {
		return nil, err
	}
	rows, err := p.Query(ctx, `
		SELECT c.relname,
		       GREATEST(c.reltuples::bigint, 0),
		       pg_total_relation_size(c.oid),
		       COALESCE(obj_description(c.oid), ''),
		       CASE WHEN c.relkind = 'v' THEN 'view' ELSE 'table' END
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relkind IN ('r','p','v')
		ORDER BY c.relname`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TableInfo{}
	for rows.Next() {
		var t TableInfo
		var size int64
		if err := rows.Scan(&t.Name, &t.Rows, &size, &t.Comment, &t.Kind); err != nil {
			return nil, err
		}
		t.SizeMB = float64(size) / 1024 / 1024
		out = append(out, t)
	}
	return out, rows.Err()
}

// RunQuery 只读查询（在指定库执行）。
func (d *pgDriver) RunQuery(ctx context.Context, database, sqlText string, limit int) (*QueryResult, error) {
	if err := ValidateReadOnly(sqlText); err != nil {
		return nil, err
	}
	p, err := d.poolFor(ctx, database)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	start := time.Now()
	finalSQL := strings.TrimRight(strings.TrimSpace(sqlText), ";")
	rows, err := p.Query(ctx, finalSQL)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer rows.Close()
	result := &QueryResult{Elapsed: time.Since(start).Round(time.Millisecond).String()}
	for i, fd := range rows.FieldDescriptions() {
		_ = i
		result.Columns = append(result.Columns, string(fd.Name))
	}
	for rows.Next() && len(result.Rows) < limit {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, vals)
	}
	return result, rows.Err()
}

// BrowsePage 表浏览真分页（总数 + 主键/指定列排序）。
func (d *pgDriver) BrowsePage(ctx context.Context, req BrowsePageReq) (*PagedResult, error) {
	ref, err := pgTableRef(req.Schema, req.Table)
	if err != nil {
		return nil, err
	}
	p, err := d.poolFor(ctx, req.DB)
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
		order = fmt.Sprintf(` ORDER BY %q %s`, req.SortCol, strings.ToUpper(req.SortDir))
	} else if pks, _ := d.PrimaryKey(ctx, req.DB, req.Schema, req.Table); len(pks) > 0 {
		order = fmt.Sprintf(` ORDER BY %q ASC`, pks[0])
	}
	start := time.Now()
	var total int64
	if err := p.QueryRow(ctx, "SELECT COUNT(*) FROM "+ref).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := p.Query(ctx, fmt.Sprintf("SELECT * FROM %s%s LIMIT %d OFFSET %d", ref, order, req.Limit, req.Offset))
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer rows.Close()
	out := &PagedResult{Total: total, Offset: req.Offset, Limit: req.Limit, Elapsed: time.Since(start).Round(time.Millisecond).String()}
	for i, fd := range rows.FieldDescriptions() {
		_ = i
		out.Columns = append(out.Columns, string(fd.Name))
	}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		out.Rows = append(out.Rows, vals)
	}
	return out, rows.Err()
}

// DescribeColumns 字段结构（含主键标记与注释）。
func (d *pgDriver) DescribeColumns(ctx context.Context, database, schema, table string) ([]ColumnInfo, error) {
	if err := ValidateIdent(table); err != nil {
		return nil, err
	}
	sc := schemaOrDefault(schema)
	p, err := d.poolFor(ctx, database)
	if err != nil {
		return nil, err
	}
	// col_description 用字符串拼接参数做 regclass cast（标识符形态在表达式里会被当表引用）
	rows, err := p.Query(ctx, `
		SELECT c.column_name, c.data_type, c.is_nullable, c.column_default,
		       EXISTS (SELECT 1 FROM information_schema.table_constraints tc
		               JOIN information_schema.key_column_usage kcu
		                 ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		                WHERE tc.constraint_type = 'PRIMARY KEY' AND tc.table_schema = c.table_schema
		                  AND tc.table_name = c.table_name AND kcu.column_name = c.column_name),
		       COALESCE(col_description(($1||'.'||$2)::regclass, c.ordinal_position), '')
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position`, sc, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ColumnInfo{}
	for rows.Next() {
		var c ColumnInfo
		var nullable string
		var isPK bool
		var def *string
		if err := rows.Scan(&c.Name, &c.DataType, &nullable, &def, &isPK, &c.Comment); err != nil {
			return nil, err
		}
		c.Nullable = nullable == "YES"
		c.Default = def
		if isPK {
			c.Key = "PRI"
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func schemaOrDefault(s string) string {
	if s == "" {
		return "public"
	}
	return s
}

// TableDDL 由系统目录拼装建表语句（PG 无服务端 SHOW CREATE 等价物）。
func (d *pgDriver) TableDDL(ctx context.Context, database, schema, table string) (string, error) {
	cols, err := d.DescribeColumns(ctx, database, schema, table)
	if err != nil {
		return "", err
	}
	if len(cols) == 0 {
		return "", errs.Wrap(errs.ErrBadRequest, "表不存在或无字段")
	}
	if schema == "" {
		schema = "public"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "CREATE TABLE %q.%q (\n", schema, table)
	pkCols := make([]string, 0, 2)
	for i, c := range cols {
		line := "    " + quotePGIdent(c.Name) + " " + c.DataType
		if !c.Nullable {
			line += " NOT NULL"
		}
		if c.Default != nil {
			line += " DEFAULT " + *c.Default
		}
		if c.Key == "PRI" {
			pkCols = append(pkCols, quotePGIdent(c.Name))
		}
		if i < len(cols)-1 || len(pkCols) > 0 {
			line += ","
		}
		b.WriteString(line + "\n")
	}
	if len(pkCols) > 0 {
		b.WriteString("    PRIMARY KEY (" + strings.Join(pkCols, ", ") + ")\n")
	}
	b.WriteString(");\n")
	for _, c := range cols {
		if c.Comment != "" {
			fmt.Fprintf(&b, "COMMENT ON COLUMN %s.%s IS '%s';\n", quotePGIdentFull(schema, table), quotePGIdent(c.Name), strings.ReplaceAll(c.Comment, "'", "''"))
		}
	}
	return b.String(), nil
}

func quotePGIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func quotePGIdentFull(schema, table string) string {
	return quotePGIdent(schema) + "." + quotePGIdent(table)
}

// ListIndexes 索引清单。
func (d *pgDriver) ListIndexes(ctx context.Context, database, schema, table string) ([]IndexInfo, error) {
	if err := ValidateIdent(table); err != nil {
		return nil, err
	}
	p, err := d.poolFor(ctx, database)
	if err != nil {
		return nil, err
	}
	rows, err := p.Query(ctx, `
		SELECT i.relname, ix.indisprimary, ix.indisunique,
		       array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum))
		FROM pg_class t
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_index ix ON ix.indrelid = t.oid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1 AND t.relname = $2
		GROUP BY i.relname, ix.indisprimary, ix.indisunique
		ORDER BY i.relname`, schemaOrDefault(schema), table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []IndexInfo{}
	for rows.Next() {
		var idx IndexInfo
		if err := rows.Scan(&idx.Name, &idx.Primary, &idx.Unique, &idx.Columns); err != nil {
			return nil, err
		}
		out = append(out, idx)
	}
	return out, rows.Err()
}

// CreateIndex 创建索引。
func (d *pgDriver) CreateIndex(ctx context.Context, req IndexRequest) error {
	ref, err := pgTableRef(req.Schema, req.Table)
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
	p, err := d.poolFor(ctx, req.DB)
	if err != nil {
		return err
	}
	cols := make([]string, 0, len(req.Columns))
	for _, c := range req.Columns {
		if err := ValidateIdent(c); err != nil {
			return err
		}
		cols = append(cols, quotePGIdent(c))
	}
	unique := ""
	if req.Unique {
		unique = "UNIQUE "
	}
	// CREATE INDEX 的索引名不允许 schema 前缀（跟随目标表所在 schema）
	_, err = p.Exec(ctx, fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", unique, quotePGIdent(name), ref, strings.Join(cols, ", ")))
	return err
}

// DropIndex 删除索引（主键索引不支持经此删除）。
func (d *pgDriver) DropIndex(ctx context.Context, database, schema, table, name string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	p, err := d.poolFor(ctx, database)
	if err != nil {
		return err
	}
	idxRef := quotePGIdent(schemaOrDefault(schema)) + "." + quotePGIdent(name)
	_, err = p.Exec(ctx, "DROP INDEX IF EXISTS "+idxRef)
	return err
}

// PrimaryKey 主键列（按序）。
func (d *pgDriver) PrimaryKey(ctx context.Context, database, schema, table string) ([]string, error) {
	if err := ValidateIdent(table); err != nil {
		return nil, err
	}
	p, err := d.poolFor(ctx, database)
	if err != nil {
		return nil, err
	}
	rows, err := p.Query(ctx, `
		SELECT a.attname
		FROM pg_index ix
		JOIN pg_class c ON c.oid = ix.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1 AND c.relname = $2 AND ix.indisprimary
		ORDER BY array_position(ix.indkey, a.attnum)`, schemaOrDefault(schema), table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// pgRowEditRef 行编辑公共校验。
func pgRowEditRef(req RowEdit) (string, []string, error) {
	ref, err := pgTableRef(req.Schema, req.Table)
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

// InsertRow 插入行（值参数化）。
func (d *pgDriver) InsertRow(ctx context.Context, req RowEdit) error {
	if err := ValidateIdent(req.Table); err != nil {
		return err
	}
	if len(req.Values) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "缺少数据")
	}
	p, err := d.poolFor(ctx, req.DB)
	if err != nil {
		return err
	}
	cols := make([]string, 0, len(req.Values))
	ph := make([]string, 0, len(req.Values))
	vals := make([]any, 0, len(req.Values))
	i := 1
	for col, v := range req.Values {
		if err := ValidateIdent(col); err != nil {
			return err
		}
		cols = append(cols, quotePGIdent(col))
		ph = append(ph, fmt.Sprintf("$%d", i))
		vals = append(vals, v)
		i++
	}
	sql := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)", quotePGIdent(schemaOrDefault(req.Schema)), quotePGIdent(req.Table), strings.Join(cols, ","), strings.Join(ph, ","))
	_, err = p.Exec(ctx, sql, vals...)
	return err
}

// UpdateRow 按主键更新。
func (d *pgDriver) UpdateRow(ctx context.Context, req RowEdit) error {
	ref, pkCols, err := pgRowEditRef(req)
	if err != nil {
		return err
	}
	if len(req.Values) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "缺少更新数据")
	}
	p, err := d.poolFor(ctx, req.DB)
	if err != nil {
		return err
	}
	sets := make([]string, 0, len(req.Values))
	vals := make([]any, 0, len(req.Values)+len(req.PK))
	i := 1
	for col, v := range req.Values {
		if err := ValidateIdent(col); err != nil {
			return err
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", quotePGIdent(col), i))
		vals = append(vals, v)
		i++
	}
	wheres := make([]string, 0, len(pkCols))
	for _, col := range pkCols {
		wheres = append(wheres, fmt.Sprintf("%s = $%d", quotePGIdent(col), i))
		vals = append(vals, req.PK[col])
		i++
	}
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s", ref, strings.Join(sets, ","), strings.Join(wheres, " AND "))
	res, err := p.Exec(ctx, sql, vals...)
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	if res.RowsAffected() == 0 {
		return errs.Wrap(errs.ErrBadRequest, "未找到匹配行（主键值可能已变更）")
	}
	return nil
}

// DeleteRow 按主键删除。
func (d *pgDriver) DeleteRow(ctx context.Context, req RowEdit) error {
	ref, pkCols, err := pgRowEditRef(req)
	if err != nil {
		return err
	}
	p, err := d.poolFor(ctx, req.DB)
	if err != nil {
		return err
	}
	wheres := make([]string, 0, len(pkCols))
	vals := make([]any, 0, len(pkCols))
	i := 1
	for _, col := range pkCols {
		wheres = append(wheres, fmt.Sprintf("%s = $%d", quotePGIdent(col), i))
		vals = append(vals, req.PK[col])
		i++
	}
	sql := fmt.Sprintf("DELETE FROM %s WHERE %s", ref, strings.Join(wheres, " AND "))
	res, err := p.Exec(ctx, sql, vals...)
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	if res.RowsAffected() == 0 {
		return errs.Wrap(errs.ErrBadRequest, "未找到匹配行")
	}
	return nil
}
