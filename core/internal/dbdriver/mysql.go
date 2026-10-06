package dbdriver

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"

	"github.com/ypanel/shared/errs"
)

type mysqlDriver struct {
	db *sql.DB
}

func newMySQL(host string, port int, user, password string) (Driver, error) {
	cfg := mysql.Config{
		User:   user,
		Passwd: password,
		Net:    "tcp",
		Addr:   fmt.Sprintf("%s:%d", host, port),
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return &mysqlDriver{db: db}, nil
}

func (d *mysqlDriver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d *mysqlDriver) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT schema_name, IFNULL(default_character_set_name,''), IFNULL(SUM(data_length+index_length),0)
		 FROM information_schema.schemata
		 LEFT JOIN information_schema.tables t ON t.table_schema = schema_name
		 WHERE schema_name NOT IN ('information_schema','mysql','performance_schema','sys')
		 GROUP BY schema_name, default_character_set_name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []DatabaseInfo{}
	for rows.Next() {
		var i DatabaseInfo
		var sizeBytes int64
		if err := rows.Scan(&i.Name, &i.Charset, &sizeBytes); err != nil {
			return nil, err
		}
		i.SizeMB = float64(sizeBytes) / 1024 / 1024
		out = append(out, i)
	}
	return out, rows.Err()
}

func (d *mysqlDriver) CreateDatabase(ctx context.Context, name, charset string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	if charset == "" {
		charset = "utf8mb4"
	}
	if err := ValidateIdent(charset); err != nil {
		return err
	}
	// DDL 标识符不可参数化；已过白名单
	_, err := d.db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET %s", name, charset))
	return err
}

func (d *mysqlDriver) DropDatabase(ctx context.Context, name string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	_, err := d.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name))
	return err
}

func (d *mysqlDriver) ListUsers(ctx context.Context) ([]UserInfo, error) {
	rows, err := d.db.QueryContext(ctx, "SELECT user, host FROM mysql.user WHERE user NOT IN ('mysql.infoschema','mysql.session','mysql.sys')")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []UserInfo{}
	for rows.Next() {
		var u UserInfo
		if err := rows.Scan(&u.Name, &u.Host); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (d *mysqlDriver) CreateUser(ctx context.Context, name, host, password string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	if host == "" {
		host = "%"
	}
	if err := ValidateHost(host); err != nil {
		return err
	}
	// 密码已过 passwordPattern 白名单（无引号/反斜杠），单引号字面量拼接安全；MySQL DDL 不支持占位符
	_, err := d.db.ExecContext(ctx, fmt.Sprintf("CREATE USER IF NOT EXISTS `%s`@`%s` IDENTIFIED BY '%s'", name, host, password))
	return err
}

func (d *mysqlDriver) DropUser(ctx context.Context, name, host string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	if host == "" {
		host = "%"
	}
	if err := ValidateHost(host); err != nil {
		return err
	}
	_, err := d.db.ExecContext(ctx, fmt.Sprintf("DROP USER IF EXISTS `%s`@`%s`", name, host))
	return err
}

func (d *mysqlDriver) ChangePassword(ctx context.Context, name, host, password string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	if host == "" {
		host = "%"
	}
	if err := ValidateHost(host); err != nil {
		return err
	}
	_, err := d.db.ExecContext(ctx, fmt.Sprintf("ALTER USER `%s`@`%s` IDENTIFIED BY '%s'", name, host, password))
	return err
}

func (d *mysqlDriver) Close() {
	_ = d.db.Close()
}

// EnableRemote 创建 % 授权的远端管理用户（B3）。
func (d *mysqlDriver) EnableRemote(ctx context.Context, password string) error {
	_, err := d.db.ExecContext(ctx, fmt.Sprintf("CREATE USER IF NOT EXISTS `remote`@`%%` IDENTIFIED BY '%s'", password))
	if err != nil {
		return err
	}
	_, err = d.db.ExecContext(ctx, "GRANT ALL PRIVILEGES ON *.* TO `remote`@`%` WITH GRANT OPTION")
	return err
}

// DisableRemote 回收远端管理用户（B3）。
func (d *mysqlDriver) DisableRemote(ctx context.Context) error {
	_, err := d.db.ExecContext(ctx, "DROP USER IF EXISTS `remote`@`%`")
	return err
}
