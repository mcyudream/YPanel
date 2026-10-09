// MySQLAdmin MySQL 管理深化（M39）：授权矩阵 / 参数配置 / 状态监控。
// 仅 mysql 驱动实现；接口挂 Driver 之外（能力探测用类型断言）。
package dbdriver

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ypanel/shared/errs"
)

// MySQLAdmin MySQL 专属管理能力。
type MySQLAdmin interface {
	// GrantMatrix 读指定库的授权矩阵（user@host → 权限集合）。
	GrantMatrix(ctx context.Context, db string) ([]GrantRow, error)
	// GrantPrivs 授予/回收库级权限（grant=false 为 REVOKE）。
	GrantPrivs(ctx context.Context, db, user, host string, privs []string, grant bool) error
	// Variables 服务器参数（可按名称子串过滤）。
	Variables(ctx context.Context, filter string) ([]KVPair, error)
	// SetGlobalVariable 白名单内在线修改。
	SetGlobalVariable(ctx context.Context, name, value string) error
	// StatusStats 精选状态指标。
	StatusStats(ctx context.Context) (map[string]float64, error)
}

// GrantRow 授权矩阵行。
type GrantRow struct {
	User  string   `json:"user"`
	Host  string   `json:"host"`
	Privs []string `json:"privs"`
}

// KVPair 通用键值。
type KVPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// grantablePrivs 库级可授权权限白名单。
var grantablePrivs = map[string]bool{
	"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
	"CREATE": true, "DROP": true, "ALTER": true, "INDEX": true,
	"CREATE VIEW": true, "SHOW VIEW": true, "REFERENCES": true, "ALL PRIVILEGES": true,
}

// adminIdentPattern 标识符白名单（库名/用户名/主机名；driver.go 已有 identPattern 则复用其语义）。
var adminIdentPattern = regexp.MustCompile(`^[a-zA-Z0-9_$]+$`)

// variableWhitelist 在线可改参数白名单（M39）。
var variableWhitelist = map[string]bool{
	"max_connections": true, "wait_timeout": true, "interactive_timeout": true,
	"slow_query_log": true, "long_query_time": true, "slow_query_log_file": false,
	"character_set_server": true, "collation_server": true,
	"innodb_buffer_pool_size": true, "innodb_flush_log_at_trx_commit": true,
	"max_allowed_packet": true, "tmp_table_size": true, "max_heap_table_size": true,
	"sort_buffer_size": true, "join_buffer_size": true, "read_rnd_buffer_size": true,
	"table_open_cache": true, "thread_cache_size": true, "open_files_limit": true,
	"log_bin_trust_function_creators": true, "sql_mode": true,
}

func quoteIdent(s string) string {
	return "`" + s + "`"
}

// GrantMatrix 实现。
func (d *mysqlDriver) GrantMatrix(ctx context.Context, db string) ([]GrantRow, error) {
	if !adminIdentPattern.MatchString(db) {
		return nil, errs.Wrap(errs.ErrBadRequest, "库名不合法")
	}
	// MySQL 8：schema_privileges 列为 GRANTEE（形如 'user'@'host'）/ PRIVILEGE_TYPE；5.7 同构
	rows, err := d.db.QueryContext(ctx,
		`SELECT grantee, privilege_type FROM information_schema.schema_privileges WHERE table_schema = ?`, db)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	idx := map[string]*GrantRow{}
	order := []string{}
	for rows.Next() {
		var grantee, priv string
		if err := rows.Scan(&grantee, &priv); err != nil {
			return nil, err
		}
		// GRANTEE 形如 'user'@'host'：先按最后一个 @ 分段，两段各自去引号
		i := strings.LastIndex(grantee, "@")
		if i < 0 {
			continue
		}
		user, host := strings.Trim(grantee[:i], "'"), strings.Trim(grantee[i+1:], "'")
		key := user + "@" + host
		g, ok := idx[key]
		if !ok {
			g = &GrantRow{User: user, Host: host}
			idx[key] = g
			order = append(order, key)
		}
		g.Privs = append(g.Privs, priv)
	}
	out := make([]GrantRow, 0, len(order))
	for _, k := range order {
		out = append(out, *idx[k])
	}
	return out, rows.Err()
}

// GrantPrivs 实现。
func (d *mysqlDriver) GrantPrivs(ctx context.Context, db, user, host string, privs []string, grant bool) error {
	if !adminIdentPattern.MatchString(db) || !adminIdentPattern.MatchString(user) {
		return errs.Wrap(errs.ErrBadRequest, "库名/用户名不合法")
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "%"
	}
	if host == "%" {
		host = "`%`" // GRANT 语句中 % 需引号
	} else if !adminIdentPattern.MatchString(host) {
		return errs.Wrap(errs.ErrBadRequest, "主机不合法")
	}
	if len(privs) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "权限列表为空")
	}
	// MySQL 8 禁止 GRANT 隐式建用户：先确认账号存在，给出明确业务错误
	var exists int
	if err := d.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM mysql.user WHERE user = ? AND host = ?", user, strings.Trim(host, "`")).Scan(&exists); err == nil && exists == 0 {
		return errs.New(errs.CodeNotFound, "error.notFound", "账号 "+user+"@"+host+" 不存在，请先在「用户」页创建")
	}
	clean := make([]string, 0, len(privs))
	for _, p := range privs {
		p = strings.ToUpper(strings.TrimSpace(p))
		if !grantablePrivs[p] {
			return errs.Wrap(errs.ErrBadRequest, "不支持的权限: "+p)
		}
		clean = append(clean, p)
	}
	verb := "GRANT"
	if !grant {
		verb = "REVOKE"
	}
	// REVOKE PRIVILEGES ON db.* FROM user；GRANT 后接 WITH GRANT OPTION 省略
	sql := fmt.Sprintf("%s %s ON %s.* TO %s@%s", verb, strings.Join(clean, ", "), quoteIdent(db), quoteIdent(user), host)
	if !grant {
		sql = fmt.Sprintf("REVOKE %s ON %s.* FROM %s@%s", strings.Join(clean, ", "), quoteIdent(db), quoteIdent(user), host)
	}
	if _, err := d.db.ExecContext(ctx, sql); err != nil {
		// 回收时该用户在该库本就无授权（MySQL 1141）视为已达目的
		if !grant && strings.Contains(err.Error(), "no such grant") {
			return nil
		}
		return err
	}
	if grant {
		_, _ = d.db.ExecContext(ctx, "FLUSH PRIVILEGES")
	}
	return nil
}

// Variables 实现。
func (d *mysqlDriver) Variables(ctx context.Context, filter string) ([]KVPair, error) {
	q := `SELECT variable_name, variable_value FROM performance_schema.global_variables`
	var args []any
	if filter != "" {
		q += ` WHERE variable_name LIKE ?`
		args = append(args, "%"+filter+"%")
	}
	q += ` ORDER BY variable_name LIMIT 500`
	rows, err := d.db.QueryContext(ctx, q, args...)
	if err != nil {
		// 旧版本无 performance_schema 时回退 SHOW VARIABLES
		rows2, err2 := d.db.QueryContext(ctx, "SHOW VARIABLES")
		if err2 != nil {
			return nil, err
		}
		rows = rows2
	}
	defer func() { _ = rows.Close() }()
	out := []KVPair{}
	for rows.Next() {
		var kv KVPair
		if err := rows.Scan(&kv.Name, &kv.Value); err != nil {
			return nil, err
		}
		if filter != "" && !strings.Contains(strings.ToLower(kv.Name), strings.ToLower(filter)) {
			continue
		}
		out = append(out, kv)
	}
	return out, rows.Err()
}

// SetGlobalVariable 实现。
func (d *mysqlDriver) SetGlobalVariable(ctx context.Context, name, value string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if !variableWhitelist[name] {
		return errs.Wrap(errs.ErrBadRequest, "该参数不在允许在线修改的白名单内: "+name)
	}
	if strings.ContainsAny(value, ";'`\\") || len(value) > 128 {
		return errs.Wrap(errs.ErrBadRequest, "参数值不合法")
	}
	// 值仅允许数字/布尔/标识符/简单串（白名单语义即数字或枚举词）
	if !regexp.MustCompile(`^[a-zA-Z0-9_.,:\-+]+$`).MatchString(value) {
		return errs.Wrap(errs.ErrBadRequest, "参数值仅允许数字/枚举词")
	}
	_, err := d.db.ExecContext(ctx, fmt.Sprintf("SET GLOBAL %s = %s", name, value))
	return err
}

// StatusStats 实现（含慢查询相关全局变量）。
func (d *mysqlDriver) StatusStats(ctx context.Context) (map[string]float64, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT variable_name, IFNULL(variable_value,'0') FROM performance_schema.global_status
		 WHERE variable_name IN ('Threads_connected','Threads_running','Queries','Com_select','Com_insert','Com_update','Com_delete','Innodb_buffer_pool_read_requests','Innodb_buffer_pool_reads','Slow_queries','Uptime','Max_used_connections')
		 UNION ALL
		 SELECT variable_name, IFNULL(variable_value,'0') FROM performance_schema.global_variables
		 WHERE variable_name IN ('slow_query_log','long_query_time','max_connections')`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]float64{}
	for rows.Next() {
		var name string
		var val string
		if err := rows.Scan(&name, &val); err != nil {
			return nil, err
		}
		var f float64
		if strings.EqualFold(val, "ON") {
			f = 1
		} else if strings.EqualFold(val, "OFF") {
			f = 0
		} else {
			_, _ = fmt.Sscanf(val, "%g", &f)
		}
		out[strings.ToLower(name)] = f
	}
	return out, rows.Err()
}
