package dbdriver

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/ypanel/shared/errs"
)

// redisDBFromName "db3" → 3。
func redisDBFromName(name string) (int, error) {
	var idx int
	if _, err := fmt.Sscanf(name, "db%d", &idx); err != nil || idx < 0 || idx > 15 {
		return 0, errs.Wrap(errs.ErrBadRequest, "Redis 库名格式为 db0-db15")
	}
	return idx, nil
}

func (d *redisDriver) clientForDB(idx int) *redis.Client {
	opts := d.rdb.Options()
	return redis.NewClient(&redis.Options{Addr: opts.Addr, Password: opts.Password, DB: idx})
}

// ListTables Redis：db0-15 的 key 数概览。
func (d *redisDriver) ListTables(ctx context.Context, database string) ([]TableInfo, error) {
	idx, err := redisDBFromName(database)
	if err != nil {
		return nil, err
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	n, err := db.DBSize(ctx).Result()
	if err != nil {
		return nil, err
	}
	return []TableInfo{{Name: database, Rows: n}}, nil
}

// Query Redis：SCAN key 列表（前 limit 个），值类型标注。
func (d *redisDriver) Query(ctx context.Context, database, sqlText string, limit int) (*QueryResult, error) {
	idx, err := redisDBFromName(database)
	if err != nil {
		return nil, err
	}
	// sqlText 可作为 MATCH 模式（空 = *）
	pattern := strings.TrimSpace(strings.TrimRight(sqlText, ";"))
	if pattern == "" {
		pattern = "*"
	}
	if pattern == "select" || pattern == "show" {
		pattern = "*"
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	result := &QueryResult{Columns: []string{"key", "type"}}
	iter := db.Scan(ctx, 0, pattern, int64(limit)).Iterator()
	for iter.Next(ctx) && len(result.Rows) < limit {
		key := iter.Val()
		typ, _ := db.Type(ctx, key).Result()
		result.Rows = append(result.Rows, []interface{}{key, typ})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
