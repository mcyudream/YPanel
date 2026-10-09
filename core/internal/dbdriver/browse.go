// 浏览与查询能力：表清单 / 结构 / 分页查询 / 行编辑 / Redis / Mongo 专属浏览。
// 各能力接口按需实现，service 层按类型断言取用。
package dbdriver

import (
	"context"
	"strings"

	"github.com/ypanel/shared/errs"
)

// TableInfo 表/集合信息。
type TableInfo struct {
	Name    string  `json:"name"`
	Rows    int64   `json:"rows"`
	SizeMB  float64 `json:"sizeMb"`
	Comment string  `json:"comment,omitempty"`
	Kind    string  `json:"kind,omitempty"` // table / view / collection
}

// QueryResult 自由 SQL 查询结果。
type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Elapsed string          `json:"elapsed"`
}

// PagedResult 表浏览分页结果（含总数）。
type PagedResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Total   int64           `json:"total"`
	Offset  int             `json:"offset"`
	Limit   int             `json:"limit"`
	Elapsed string          `json:"elapsed"`
}

// ColumnInfo 字段结构。
type ColumnInfo struct {
	Name     string  `json:"name"`
	DataType string  `json:"dataType"`
	Nullable bool    `json:"nullable"`
	Key      string  `json:"key,omitempty"` // PRI / UNI / MUL
	Default  *string `json:"default,omitempty"`
	Comment  string  `json:"comment,omitempty"`
	Extra    string  `json:"extra,omitempty"` // auto_increment 等
}

// IndexInfo 索引信息（SQL 库）。
type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Primary bool     `json:"primary"`
}

// IndexRequest 创建索引请求。
type IndexRequest struct {
	DB      string   `json:"db"`
	Schema  string   `json:"schema,omitempty"`
	Table   string   `json:"table"`
	Name    string   `json:"name,omitempty"` // 空则自动生成
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

// RowEdit 行编辑请求（主键定位，值参数化）。
type RowEdit struct {
	DB     string         `json:"db"`
	Schema string         `json:"schema,omitempty"`
	Table  string         `json:"table"`
	PK     map[string]any `json:"pk"`
	Values map[string]any `json:"values,omitempty"`
}

// ---- SQL 库能力 ----

// SchemasProvider 多 schema 支持（PG）；MySQL 不实现。
type SchemasProvider interface {
	Schemas(ctx context.Context, database string) ([]string, error)
}

// SQLBrowser SQL 库浏览：表清单 / 自由只读查询 / 表浏览分页。
type SQLBrowser interface {
	ListTables(ctx context.Context, database, schema string) ([]TableInfo, error)
	RunQuery(ctx context.Context, database, sqlText string, limit int) (*QueryResult, error)
	BrowsePage(ctx context.Context, req BrowsePageReq) (*PagedResult, error)
}

// BrowsePageReq 表浏览分页请求。
type BrowsePageReq struct {
	DB      string `json:"db"`
	Schema  string `json:"schema,omitempty"`
	Table   string `json:"table"`
	SortCol string `json:"sortCol,omitempty"` // 空 = 主键序
	SortDir string `json:"sortDir,omitempty"` // asc / desc
	Offset  int    `json:"offset"`
	Limit   int    `json:"limit"`
}

// MetaExplorer 结构浏览能力。
type MetaExplorer interface {
	DescribeColumns(ctx context.Context, database, schema, table string) ([]ColumnInfo, error)
	TableDDL(ctx context.Context, database, schema, table string) (string, error)
	ListIndexes(ctx context.Context, database, schema, table string) ([]IndexInfo, error)
	CreateIndex(ctx context.Context, req IndexRequest) error
	DropIndex(ctx context.Context, database, schema, table, name string) error
}

// RowEditor 行级写回（主键定位，无主键表拒绝）。
type RowEditor interface {
	PrimaryKey(ctx context.Context, database, schema, table string) ([]string, error)
	InsertRow(ctx context.Context, req RowEdit) error
	UpdateRow(ctx context.Context, req RowEdit) error
	DeleteRow(ctx context.Context, req RowEdit) error
}

// ---- Redis 能力 ----

// RedisKey key 概览。
type RedisKey struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// RedisZSetItem 有序集合成员。
type RedisZSetItem struct {
	Member string  `json:"member"`
	Score  float64 `json:"score"`
}

// RedisEntry 流条目。
type RedisStreamEntry struct {
	ID     string            `json:"id"`
	Fields map[string]string `json:"fields"`
}

// RedisKeyDetail key 详情（类型化值）。
type RedisKeyDetail struct {
	Name    string             `json:"name"`
	Type    string             `json:"type"`
	TTL     int64              `json:"ttl"`    // 秒；-1 无过期
	Length  int64              `json:"length"` // 元素数或字节数
	String  string             `json:"string,omitempty"`
	Hash    map[string]string  `json:"hash,omitempty"`
	List    []string           `json:"list,omitempty"`
	Set     []string           `json:"set,omitempty"`
	ZSet    []RedisZSetItem    `json:"zset,omitempty"`
	Stream  []RedisStreamEntry `json:"stream,omitempty"`
	Binary  bool               `json:"binary,omitempty"` // string 值非 UTF-8
	Preview bool               `json:"preview"`          // 值被截断（超预览上限）
}

// RedisWrite key 写入（整体替换语义）。
type RedisWrite struct {
	DB    string `json:"db"`
	Name  string `json:"name"`
	Type  string `json:"type"` // string/hash/list/set/zset
	Value any    `json:"value"`
}

// RedisBrowser Redis 浏览与操作。
type RedisBrowser interface {
	ScanKeys(ctx context.Context, database, pattern string, cursor uint64, count int64) ([]RedisKey, uint64, error)
	KeyDetail(ctx context.Context, database, name string) (*RedisKeyDetail, error)
	WriteKey(ctx context.Context, req RedisWrite) error
	DeleteKey(ctx context.Context, database, name string) error
	SetTTL(ctx context.Context, database, name string, ttlSecs int64) error // -1 = persist
	ExecCommand(ctx context.Context, database, command string) (string, error)
	FlushDatabase(ctx context.Context, database string) error
	ExportDumps(ctx context.Context, database string, cursor uint64, count int64) ([]RedisDump, uint64, error)
	ImportKeys(ctx context.Context, database string, items []RedisDump) (int, error)
}

// ---- Mongo 能力 ----

// MongoIndex mongo 索引。
type MongoIndex struct {
	Name       string         `json:"name"`
	Keys       map[string]int `json:"keys"`
	Unique     bool           `json:"unique"`
	SizeMB     float64        `json:"sizeMb"`
	Accesses   int64          `json:"accesses"`        // $indexStats accesses.ops（自服务启动以来）
	AccessedAt string         `json:"accessedAt,omitempty"` // 最近一次使用时间
}

// MongoDocPage 文档分页。
type MongoDocPage struct {
	Docs  []string `json:"docs"` // 每文档一个 canonical ExtJSON 字符串（$oid/$date 往返保真）
	Total int64    `json:"total"`
	Skip  int      `json:"skip"`
	Limit int      `json:"limit"`
}

// MongoBrowser MongoDB 浏览与管理。
type MongoBrowser interface {
	FindDocs(ctx context.Context, database, coll, filterJSON, projectJSON, sortField, sortDir string, skip, limit int) (*MongoDocPage, error)
	Aggregate(ctx context.Context, database, coll string, stages []string, maxDocs int) (*MongoDocPage, error)
	GetDoc(ctx context.Context, database, coll, idJSON string) (string, error)
	InsertDoc(ctx context.Context, database, coll, docJSON string) error
	UpdateDoc(ctx context.Context, database, coll, idJSON, docJSON string) error
	DeleteDoc(ctx context.Context, database, coll, idJSON string) error
	ListIndexes(ctx context.Context, database, coll string) ([]MongoIndex, error)
	CreateIndex(ctx context.Context, database, coll, name string, keys map[string]int, unique bool) error
	DropIndex(ctx context.Context, database, coll, name string) error
	CreateCollection(ctx context.Context, database, name string) error
	DropCollection(ctx context.Context, database, name string) error
	RenameCollection(ctx context.Context, database, from, to string) error
}

// ---- 只读校验（多语句拆分 + 逐句白名单）----

// readOnlyPrefixes 只读语句白名单前缀。
var readOnlyPrefixes = []string{"select", "show", "desc", "describe", "explain"}

// ValidateReadOnly 校验 SQL 为只读：剥注释后按引号外分号拆句，逐句白名单。
// 单前缀判断会被 "select 1; drop table x" 或注释开头绕过，必须逐句。
func ValidateReadOnly(sql string) error {
	stmts := splitStatements(stripSQLComments(sql))
	if len(stmts) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "语句为空")
	}
	for _, s := range stmts {
		t := strings.ToLower(strings.TrimSpace(s))
		ok := false
		for _, p := range readOnlyPrefixes {
			if t == p || strings.HasPrefix(t, p+" ") || strings.HasPrefix(t, p+"(") {
				ok = true
				break
			}
		}
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "只允许 SELECT/SHOW/DESC/EXPLAIN 语句（检测到其它语句或存在多条语句）")
		}
	}
	return nil
}

// stripSQLComments 剥除 -- 行注释与 /* */ 块注释（引号内不剥）。
func stripSQLComments(sql string) string {
	var b strings.Builder
	inSingle, inDouble, inBacktick := false, false, false
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		switch {
		case inSingle:
			b.WriteByte(ch)
			if ch == '\'' {
				// '' 转义
				if i+1 < len(sql) && sql[i+1] == '\'' {
					b.WriteByte('\'')
					i++
				} else {
					inSingle = false
				}
			}
		case inDouble:
			b.WriteByte(ch)
			if ch == '"' {
				inDouble = false
			}
		case inBacktick:
			b.WriteByte(ch)
			if ch == '`' {
				inBacktick = false
			}
		case ch == '\'' :
			inSingle = true
			b.WriteByte(ch)
		case ch == '"':
			inDouble = true
			b.WriteByte(ch)
		case ch == '`':
			inBacktick = true
			b.WriteByte(ch)
		case ch == '-' && i+1 < len(sql) && sql[i+1] == '-':
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			if i < len(sql) {
				b.WriteByte('\n')
			}
		case ch == '/' && i+1 < len(sql) && sql[i+1] == '*':
			i += 2
			for i+1 < len(sql) && !(sql[i] == '*' && sql[i+1] == '/') {
				i++
			}
			i++ // 跳过结尾 '/'
			b.WriteByte(' ')
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

// splitStatements 按引号外分号拆句。
func splitStatements(sql string) []string {
	var out []string
	var b strings.Builder
	inSingle, inDouble, inBacktick := false, false, false
	flush := func() {
		if strings.TrimSpace(b.String()) != "" {
			out = append(out, b.String())
		}
		b.Reset()
	}
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		switch {
		case inSingle:
			b.WriteByte(ch)
			if ch == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					b.WriteByte('\'')
					i++
				} else {
					inSingle = false
				}
			}
		case inDouble:
			b.WriteByte(ch)
			if ch == '"' {
				inDouble = false
			}
		case inBacktick:
			b.WriteByte(ch)
			if ch == '`' {
				inBacktick = false
			}
		case ch == '\'':
			inSingle = true
			b.WriteByte(ch)
		case ch == '"':
			inDouble = true
			b.WriteByte(ch)
		case ch == '`':
			inBacktick = true
			b.WriteByte(ch)
		case ch == ';':
			flush()
		default:
			b.WriteByte(ch)
		}
	}
	flush()
	return out
}

// DatabaseGranter 库级授权（M32：可选能力，mysql/postgres 实现；redis/mongo 不适用）。
type DatabaseGranter interface {
	GrantDatabase(ctx context.Context, database, user, host string) error
}
