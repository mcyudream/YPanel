// Package dbdriver 被管数据库的直连驱动层（库/用户管理）。
// 安全：标识符（库名/用户名）经白名单正则校验后才可进入 DDL；数据值一律参数化。
package dbdriver

import (
	"context"
	"regexp"

	"github.com/ypanel/shared/errs"
)

// identPattern 库名/用户名白名单。
var identPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)

// hostPattern 连接主机白名单（MySQL host 可为 %/.- 等）。
var hostPattern = regexp.MustCompile(`^[a-zA-Z0-9._%:-]{1,253}$`)

// ValidateIdent 校验标识符。
func ValidateIdent(s string) error {
	if !identPattern.MatchString(s) {
		return errs.Wrap(errs.ErrBadRequest, "标识符不合法（字母/下划线开头，仅字母数字下划线，≤63 位）: "+s)
	}
	return nil
}

// ValidateHost 校验主机。
func ValidateHost(s string) error {
	if !hostPattern.MatchString(s) {
		return errs.Wrap(errs.ErrBadRequest, "主机不合法: "+s)
	}
	return nil
}

// DatabaseInfo 数据库信息。
type DatabaseInfo struct {
	Name    string `json:"name"`
	SizeMB  float64 `json:"sizeMb"`
	Charset string `json:"charset"`
}

// UserInfo 用户信息。
type UserInfo struct {
	Name  string `json:"name"`
	Host  string `json:"host"`
	Extra string `json:"extra"` // 类型相关说明（权限摘要等）
}

// Driver 各库驱动统一接口。host/port 指向实例暴露地址。
type Driver interface {
	Ping(ctx context.Context) error
	ListDatabases(ctx context.Context) ([]DatabaseInfo, error)
	CreateDatabase(ctx context.Context, name, charset string) error
	DropDatabase(ctx context.Context, name string) error
	ListUsers(ctx context.Context) ([]UserInfo, error)
	CreateUser(ctx context.Context, name, host, password string) error
	DropUser(ctx context.Context, name, host string) error
	ChangePassword(ctx context.Context, name, host, password string) error
	// B3：远程访问开关（SQL 层授权/回收远端管理用户）
	EnableRemote(ctx context.Context, password string) error
	DisableRemote(ctx context.Context) error
	Close()
}

// New 按类型创建驱动。
func New(dbType, host string, port int, user, password string) (Driver, error) {
	switch dbType {
	case "mysql":
		return newMySQL(host, port, user, password)
	case "postgres":
		return newPostgres(host, port, user, password)
	case "redis":
		return newRedis(host, port, password)
	case "mongo":
		return newMongo(host, port, user, password)
	default:
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的数据库类型: "+dbType)
	}
}
