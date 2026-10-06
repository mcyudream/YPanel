package dbdriver

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ypanel/shared/errs"
)

type pgDriver struct {
	pool *pgxpool.Pool
}

func newPostgres(host string, port int, user, password string) (Driver, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres", user, password, host, port)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return &pgDriver{pool: pool}, nil
}

func (d *pgDriver) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func (d *pgDriver) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	rows, err := d.pool.Query(ctx, `
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
	_, err := d.pool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %q`, name))
	return err
}

func (d *pgDriver) DropDatabase(ctx context.Context, name string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	// 断开既有连接后删除
	if _, err := d.pool.Exec(ctx,
		fmt.Sprintf(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()`, name)); err != nil {
		return err
	}
	_, err := d.pool.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %q`, name))
	return err
}

func (d *pgDriver) ListUsers(ctx context.Context) ([]UserInfo, error) {
	rows, err := d.pool.Query(ctx, `
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
	_, err := d.pool.Exec(ctx, fmt.Sprintf(`CREATE ROLE %q LOGIN PASSWORD '%s'`, name, password))
	return err
}

func (d *pgDriver) DropUser(ctx context.Context, name, host string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	_, err := d.pool.Exec(ctx, fmt.Sprintf(`DROP ROLE IF EXISTS %q`, name))
	return err
}

func (d *pgDriver) ChangePassword(ctx context.Context, name, host, password string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	_, err := d.pool.Exec(ctx, fmt.Sprintf(`ALTER ROLE %q PASSWORD '%s'`, name, password))
	return err
}

func (d *pgDriver) Close() {
	d.pool.Close()
}

var _ = pgx.QueryExecModeCacheStatement

// EnableRemote 创建远端管理角色（B3）。
func (d *pgDriver) EnableRemote(ctx context.Context, password string) error {
	_, err := d.pool.Exec(ctx, fmt.Sprintf("CREATE ROLE remote WITH LOGIN SUPERUSER PASSWORD '%s'", password))
	return err
}

// DisableRemote 回收远端管理角色（B3）。
func (d *pgDriver) DisableRemote(ctx context.Context) error {
	_, err := d.pool.Exec(ctx, "DROP ROLE IF EXISTS remote")
	return err
}
