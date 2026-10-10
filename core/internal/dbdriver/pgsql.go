package dbdriver

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ypanel/shared/errs"
)

type pgDriver struct {
	mu    sync.Mutex
	host  string
	port  int
	user  string
	pwd   string
	pools map[string]*pgxpool.Pool // 按库名缓存连接池（含基座 "postgres"）
}

// pgExtNamePattern 扩展名白名单（uuid-ossp / pg_trgm 等；仅用于校验，SQL 侧 %q 引用）。
var pgExtNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,63}$`)

func newPostgres(host string, port int, user, password string) (Driver, error) {
	d := &pgDriver{host: host, port: port, user: user, pwd: password, pools: map[string]*pgxpool.Pool{}}
	if _, err := d.poolFor(context.Background(), "postgres"); err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return d, nil
}

// dsnOf 构造指定库的 DSN（不能对整串 Replace：用户名可能等于库名，"//postgres:" 会误伤）。
func (d *pgDriver) dsnOf(database string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", d.user, d.pwd, d.host, d.port, database)
}

// poolFor 取指定库的连接池（惰性创建并缓存）。
func (d *pgDriver) poolFor(ctx context.Context, database string) (*pgxpool.Pool, error) {
	if database == "" {
		database = "postgres"
	}
	if err := ValidateIdent(database); err != nil {
		return nil, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if p, ok := d.pools[database]; ok {
		return p, nil
	}
	pool, err := pgxpool.New(ctx, d.dsnOf(database))
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	d.pools[database] = pool
	return pool, nil
}

// dropPool 关闭并移除某库的池（删库后调用）。
func (d *pgDriver) dropPool(database string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if p, ok := d.pools[database]; ok {
		p.Close()
		delete(d.pools, database)
	}
}

func (d *pgDriver) Ping(ctx context.Context) error {
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return err
	}
	return p.Ping(ctx)
}

func (d *pgDriver) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return nil, err
	}
	rows, err := p.Query(ctx, `
		SELECT d.datname, pg_database_size(d.datname), pg_encoding_to_char(d.encoding)
		FROM pg_database d
		WHERE NOT d.datistemplate AND d.datname != 'postgres'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DatabaseInfo{}
	for rows.Next() {
		var i DatabaseInfo
		var sizeBytes int64
		if err := rows.Scan(&i.Name, &sizeBytes, &i.Charset); err != nil {
			return nil, err
		}
		i.SizeMB = float64(sizeBytes) / 1024 / 1024
		out = append(out, i)
	}
	return out, rows.Err()
}

func (d *pgDriver) CreateDatabase(ctx context.Context, name, charset string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	_ = charset // PG 编码由模板库决定，M4 不暴露
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return err
	}
	// CREATE DATABASE 不允许在事务块内，pgx Exec 单语句即自动提交
	_, err = p.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %q`, name))
	return err
}

// CreateExtension 在指定库创建扩展（幂等；商店纳管建库后自动建扩展用）。
// 扩展名可含连字符（uuid-ossp），不走 ValidateIdent——用专用白名单，%q 引用防注入。
func (d *pgDriver) CreateExtension(ctx context.Context, database, name string) error {
	if !pgExtNamePattern.MatchString(name) {
		return errs.Wrap(errs.ErrBadRequest, "扩展名不合法（字母/数字/连字符/下划线，≤63 位）: "+name)
	}
	p, err := d.poolFor(ctx, database)
	if err != nil {
		return err
	}
	_, err = p.Exec(ctx, fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS %q`, name))
	return err
}

func (d *pgDriver) DropDatabase(ctx context.Context, name string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return err
	}
	// 断开既有连接后删除
	if _, err := p.Exec(ctx,
		fmt.Sprintf(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()`, name)); err != nil {
		return err
	}
	if _, err := p.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %q`, name)); err != nil {
		return err
	}
	d.dropPool(name)
	return nil
}

func (d *pgDriver) ListUsers(ctx context.Context) ([]UserInfo, error) {
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return nil, err
	}
	rows, err := p.Query(ctx, `
		SELECT rolname, rolsuper::text, rolcanlogin::text
		FROM pg_roles WHERE rolname NOT LIKE 'pg\_%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UserInfo{}
	for rows.Next() {
		var u UserInfo
		var super, canLogin string
		if err := rows.Scan(&u.Name, &super, &canLogin); err != nil {
			return nil, err
		}
		u.Extra = "login=" + canLogin
		if super == "true" {
			u.Extra += ",superuser"
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (d *pgDriver) CreateUser(ctx context.Context, name, host, password string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	// PG16+ 保留 pg_ 前缀角色名（SQLSTATE 42939）
	if strings.HasPrefix(strings.ToLower(name), "pg_") {
		return errs.Wrap(errs.ErrBadRequest, "PostgreSQL 角色名不允许以 pg_ 开头（系统保留）")
	}
	// PG 无 host 概念，忽略入参
	// 密码已过白名单（无引号），单引号字面量拼接安全
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return err
	}
	_, err = p.Exec(ctx, fmt.Sprintf(`CREATE ROLE %q LOGIN PASSWORD '%s'`, name, password))
	return err
}

func (d *pgDriver) DropUser(ctx context.Context, name, host string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return err
	}
	_, err = p.Exec(ctx, fmt.Sprintf(`DROP ROLE IF EXISTS %q`, name))
	return err
}

func (d *pgDriver) ChangePassword(ctx context.Context, name, host, password string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return err
	}
	_, err = p.Exec(ctx, fmt.Sprintf(`ALTER ROLE %q PASSWORD '%s'`, name, password))
	return err
}

func (d *pgDriver) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, p := range d.pools {
		p.Close()
	}
	d.pools = map[string]*pgxpool.Pool{}
}

// EnableRemote 创建远端管理角色（B3）。
func (d *pgDriver) EnableRemote(ctx context.Context, password string) error {
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return err
	}
	_, err = p.Exec(ctx, fmt.Sprintf("CREATE ROLE remote WITH LOGIN SUPERUSER PASSWORD '%s'", password))
	return err
}

// DisableRemote 回收远端管理角色（B3）。
func (d *pgDriver) DisableRemote(ctx context.Context) error {
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return err
	}
	_, err = p.Exec(ctx, "DROP ROLE IF EXISTS remote")
	return err
}

// GrantDatabase 授权账号对指定库的全部权限（M32）：PG 直接改库 owner（一条语句获得全权，含 schema）。
func (d *pgDriver) GrantDatabase(ctx context.Context, database, user, host string) error {
	if err := ValidateIdent(database); err != nil {
		return err
	}
	if err := ValidateIdent(user); err != nil {
		return err
	}
	if strings.HasPrefix(strings.ToLower(user), "pg_") {
		return errs.Wrap(errs.ErrBadRequest, "PostgreSQL 角色名不允许以 pg_ 开头（系统保留）")
	}
	pool, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return err
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, fmt.Sprintf(`ALTER DATABASE "%s" OWNER TO "%s"`, database, user))
	return err
}
