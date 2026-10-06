package dbdriver

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/ypanel/shared/errs"
)

type redisDriver struct {
	rdb *redis.Client
}

func newRedis(host string, port int, password string) (Driver, error) {
	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", host, port), Password: password})
	return &redisDriver{rdb: rdb}, nil
}

func (d *redisDriver) Ping(ctx context.Context) error {
	return d.rdb.Ping(ctx).Err()
}

// ListDatabases Redis 以 16 个逻辑库呈现（0-15），大小为各库 key 数。
func (d *redisDriver) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	opts := d.rdb.Options()
	out := []DatabaseInfo{}
	for i := 0; i < 16; i++ {
		db := redis.NewClient(&redis.Options{Addr: opts.Addr, Password: opts.Password, DB: i})
		n, err := db.DBSize(ctx).Result()
		_ = db.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, DatabaseInfo{Name: fmt.Sprintf("db%d", i), SizeMB: float64(n)})
	}
	return out, nil
}

// CreateDatabase Redis 无建库语义（逻辑库固定 0-15）。
func (d *redisDriver) CreateDatabase(context.Context, string, string) error {
	return errs.Wrap(errs.ErrBadRequest, "Redis 为固定逻辑库（db0-db15），无需创建")
}

// DropDatabase Redis 语义为 FLUSHDB。
func (d *redisDriver) DropDatabase(ctx context.Context, name string) error {
	var idx int
	if _, err := fmt.Sscanf(name, "db%d", &idx); err != nil || idx < 0 || idx > 15 {
		return errs.Wrap(errs.ErrBadRequest, "Redis 库名格式为 db0-db15")
	}
	db := redis.NewClient(&redis.Options{
		Addr:     d.rdb.Options().Addr,
		Password: d.rdb.Options().Password,
		DB:       idx,
	})
	defer func() { _ = db.Close() }()
	return db.FlushDB(ctx).Err()
}

// ListUsers 列出 ACL 用户。
func (d *redisDriver) ListUsers(ctx context.Context) ([]UserInfo, error) {
	lines, err := d.rdb.ACLList(ctx).Result()
	if err != nil {
		return nil, err
	}
	out := []UserInfo{}
	for _, line := range lines {
		u := UserInfo{Name: "default"}
		if strings.HasPrefix(line, "user default ") {
			u.Extra = strings.TrimSpace(strings.TrimPrefix(line, "user default "))
		}
		out = append(out, u)
	}
	return out, nil
}

// CreateUser Redis 语义为设置 default 用户密码（ACL SETUSER）。
func (d *redisDriver) CreateUser(ctx context.Context, name, host, password string) error {
	return errs.Wrap(errs.ErrBadRequest, "Redis 仅支持 default 用户（用修改密码功能）")
}

func (d *redisDriver) DropUser(context.Context, string, string) error {
	return errs.Wrap(errs.ErrBadRequest, "Redis 不支持删除 default 用户")
}

// ChangePassword 设置 default 用户密码。
func (d *redisDriver) ChangePassword(ctx context.Context, name, host, password string) error {
	// 注意顺序：resetpass 会清掉所有密码，必须先执行再设置新密码；重启后以 compose 配置为准
	if err := d.rdb.ACLSetUser(ctx, "default", "resetpass").Err(); err != nil {
		return err
	}
	return d.rdb.ACLSetUser(ctx, "default", ">"+password, "+@all", "~*", "&*", "on").Err()
}

func (d *redisDriver) Close() {
	_ = d.rdb.Close()
}

// EnableRemote 创建远端 ACL 用户（B3）。
func (d *redisDriver) EnableRemote(ctx context.Context, password string) error {
	return d.rdb.ACLSetUser(ctx, "remote", "on", ">"+password, "+@all", "~*", "&*").Err()
}

// DisableRemote 回收远端 ACL 用户（B3）。
func (d *redisDriver) DisableRemote(ctx context.Context) error {
	return d.rdb.ACLDelUser(ctx, "remote").Err()
}
