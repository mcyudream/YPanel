package dbdriver

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"

	"github.com/ypanel/shared/errs"
)

// previewLimit 列表/集合类值预览上限。
const previewLimit = 500

// redisCmdBlacklist 命令控制台黑名单（危险/破坏性/改变连接状态的命令）。
var redisCmdBlacklist = map[string]bool{
	"FLUSHALL": true, "FLUSHDB": true, "CONFIG": true, "DEBUG": true,
	"SLAVEOF": true, "REPLICAOF": true, "SHUTDOWN": true, "MONITOR": true,
	"KEYS": true, "SCRIPT": true, "SAVE": true, "BGSAVE": true, "BGREWRITEAOF": true,
	"SWAPDB": true, "CLUSTER": true, "MIGRATE": true, "RESTORE": true, "DUMP": true,
	"SELECT": true, "ACL": true, "MODULE": true, "COMMAND": true, "CLIENT": true,
	"MULTI": true, "EXEC": true, "DISCARD": true, "WATCH": true, "UNWATCH": true,
	"SUBSCRIBE": true, "UNSUBSCRIBE": true, "PSUBSCRIBE": true, "PUNSUBSCRIBE": true,
	"HOST": true, "POST": true, "LATENCY": true, "SLOWLOG": true,
}

// ScanKeys 游标分页扫描（返回下一游标，"0" 表示结束）。
func (d *redisDriver) ScanKeys(ctx context.Context, database, pattern string, cursor uint64, count int64) ([]RedisKey, uint64, error) {
	idx, err := redisDBFromName(database)
	if err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(pattern) == "" {
		pattern = "*"
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	keys, next, err := db.Scan(ctx, cursor, pattern, count).Result()
	if err != nil {
		return nil, 0, err
	}
	sort.Strings(keys)
	out := make([]RedisKey, 0, len(keys))
	for _, k := range keys {
		t, err := db.Type(ctx, k).Result()
		if err != nil {
			t = "unknown"
		}
		out = append(out, RedisKey{Name: k, Type: t})
	}
	return out, next, nil
}

// KeyDetail key 类型化详情。
func (d *redisDriver) KeyDetail(ctx context.Context, database, name string) (*RedisKeyDetail, error) {
	idx, err := redisDBFromName(database)
	if err != nil {
		return nil, err
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	typ, err := db.Type(ctx, name).Result()
	if err != nil {
		return nil, err
	}
	if typ == "none" {
		return nil, errs.Wrap(errs.ErrBadRequest, "key 不存在")
	}
	detail := &RedisKeyDetail{Name: name, Type: typ, TTL: -1}
	if ttl, err := db.TTL(ctx, name).Result(); err == nil && ttl > 0 {
		detail.TTL = int64(ttl.Seconds())
	}
	switch typ {
	case "string":
		val, err := db.Get(ctx, name).Bytes()
		if err != nil {
			return nil, err
		}
		detail.Length = int64(len(val))
		if !utf8.Valid(val) {
			detail.Binary = true
			detail.String = fmt.Sprintf("<binary %d bytes>", len(val))
		} else {
			detail.String = string(val)
			if len(val) > 64<<10 {
				detail.String = detail.String[:64<<10]
				detail.Preview = true
			}
		}
	case "hash":
		m, err := db.HGetAll(ctx, name).Result()
		if err != nil {
			return nil, err
		}
		detail.Hash = m
		detail.Length = int64(len(m))
	case "list":
		n, _ := db.LLen(ctx, name).Result()
		detail.Length = n
		vals, err := db.LRange(ctx, name, 0, previewLimit-1).Result()
		if err != nil {
			return nil, err
		}
		detail.List = vals
		detail.Preview = n > previewLimit
	case "set":
		n, _ := db.SCard(ctx, name).Result()
		detail.Length = n
		vals, err := db.SMembers(ctx, name).Result()
		if err != nil {
			return nil, err
		}
		if len(vals) > previewLimit {
			vals = vals[:previewLimit]
			detail.Preview = true
		}
		detail.Set = vals
	case "zset":
		n, _ := db.ZCard(ctx, name).Result()
		detail.Length = n
		vals, err := db.ZRangeWithScores(ctx, name, 0, previewLimit-1).Result()
		if err != nil {
			return nil, err
		}
		for _, z := range vals {
			detail.ZSet = append(detail.ZSet, RedisZSetItem{Member: fmt.Sprintf("%v", z.Member), Score: z.Score})
		}
		detail.Preview = n > previewLimit
	case "stream":
		n, _ := db.XLen(ctx, name).Result()
		detail.Length = n
		msgs, err := db.XRevRangeN(ctx, name, "+", "-", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, m := range msgs {
			fields := make(map[string]string, len(m.Values))
			for k, v := range m.Values {
				fields[k] = fmt.Sprintf("%v", v)
			}
			detail.Stream = append(detail.Stream, RedisStreamEntry{ID: m.ID, Fields: fields})
		}
		detail.Preview = n > 100
	default:
		return nil, errs.Wrap(errs.ErrBadRequest, "暂不支持查看该类型: "+typ)
	}
	return detail, nil
}

// WriteKey 写入（整体替换：先删后写）。
func (d *redisDriver) WriteKey(ctx context.Context, req RedisWrite) error {
	idx, err := redisDBFromName(req.DB)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Name) == "" {
		return errs.Wrap(errs.ErrBadRequest, "key 名不能为空")
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	switch req.Type {
	case "string":
		s, ok := req.Value.(string)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "string 值必须为字符串")
		}
		if err := db.Del(ctx, req.Name).Err(); err != nil {
			return err
		}
		return db.Set(ctx, req.Name, s, 0).Err()
	case "hash":
		m, ok := req.Value.(map[string]any)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "hash 值必须为对象（field → value）")
		}
		if err := db.Del(ctx, req.Name).Err(); err != nil {
			return err
		}
		if len(m) == 0 {
			return nil
		}
		pairs := make([]any, 0, len(m)*2)
		for f, v := range m {
			pairs = append(pairs, f, fmt.Sprintf("%v", v))
		}
		return db.HSet(ctx, req.Name, pairs...).Err()
	case "list":
		arr, ok := req.Value.([]any)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "list 值必须为数组")
		}
		if err := db.Del(ctx, req.Name).Err(); err != nil {
			return err
		}
		if len(arr) == 0 {
			return nil
		}
		items := make([]any, 0, len(arr))
		for _, v := range arr {
			items = append(items, fmt.Sprintf("%v", v))
		}
		return db.RPush(ctx, req.Name, items...).Err()
	case "set":
		arr, ok := req.Value.([]any)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "set 值必须为数组")
		}
		if err := db.Del(ctx, req.Name).Err(); err != nil {
			return err
		}
		if len(arr) == 0 {
			return nil
		}
		items := make([]any, 0, len(arr))
		for _, v := range arr {
			items = append(items, fmt.Sprintf("%v", v))
		}
		return db.SAdd(ctx, req.Name, items...).Err()
	case "zset":
		arr, ok := req.Value.([]any)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "zset 值必须为 [{member, score}] 数组")
		}
		if err := db.Del(ctx, req.Name).Err(); err != nil {
			return err
		}
		if len(arr) == 0 {
			return nil
		}
		members := make([]redis.Z, 0, len(arr))
		for _, v := range arr {
			obj, ok := v.(map[string]any)
			if !ok {
				return errs.Wrap(errs.ErrBadRequest, "zset 成员必须为 {member, score} 对象")
			}
			mem, _ := obj["member"].(string)
			score, err := toFloat(obj["score"])
			if err != nil {
				return errs.Wrap(errs.ErrBadRequest, "score 必须为数字")
			}
			members = append(members, redis.Z{Score: score, Member: mem})
		}
		return db.ZAdd(ctx, req.Name, members...).Err()
	default:
		return errs.Wrap(errs.ErrBadRequest, "不支持的写入类型: "+req.Type)
	}
}

func toFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case int64:
		return float64(x), nil
	case string:
		return strconv.ParseFloat(x, 64)
	default:
		return 0, errs.Wrap(errs.ErrBadRequest, "数字格式不合法")
	}
}

// DeleteKey 删除 key。
func (d *redisDriver) DeleteKey(ctx context.Context, database, name string) error {
	idx, err := redisDBFromName(database)
	if err != nil {
		return err
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	return db.Del(ctx, name).Err()
}

// SetTTL 设置过期（秒；-1 = 持久化）。
func (d *redisDriver) SetTTL(ctx context.Context, database, name string, ttlSecs int64) error {
	idx, err := redisDBFromName(database)
	if err != nil {
		return err
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	if ttlSecs < 0 {
		return db.Persist(ctx, name).Err()
	}
	return db.Expire(ctx, name, time.Duration(ttlSecs)*time.Second).Err()
}

// ExecCommand 命令控制台（黑名单拦截 + 简单引号解析）。
func (d *redisDriver) ExecCommand(ctx context.Context, database, command string) (string, error) {
	idx, err := redisDBFromName(database)
	if err != nil {
		return "", err
	}
	args, err := splitRedisCommand(command)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		return "", errs.Wrap(errs.ErrBadRequest, "命令为空")
	}
	if redisCmdBlacklist[strings.ToUpper(args[0])] {
		return "", errs.Wrap(errs.ErrBadRequest, "命令被禁止: "+args[0]+"（危险或影响连接状态的命令不允许在面板执行）")
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	argv := make([]any, len(args))
	for i, a := range args {
		argv[i] = a
	}
	val, err := db.Do(ctx, argv...).Result()
	if err != nil {
		// redis 业务错误（如 WRONGTYPE）也作为输出返回而非接口错误
		return "ERR " + err.Error(), nil
	}
	return formatRedisValue(val), nil
}

// splitRedisCommand 解析命令串为参数（支持单/双引号包裹含空格参数）。
func splitRedisCommand(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inSingle, inDouble, hasToken := false, false, false
	flush := func() {
		if hasToken {
			args = append(args, cur.String())
		}
		cur.Reset()
		hasToken = false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(ch)
			}
		case inDouble:
			if ch == '"' {
				inDouble = false
			} else if ch == '\\' && i+1 < len(s) {
				i++
				cur.WriteByte(s[i])
			} else {
				cur.WriteByte(ch)
			}
		case ch == '\'':
			inSingle, hasToken = true, true
		case ch == '"':
			inDouble, hasToken = true, true
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			flush()
		default:
			cur.WriteByte(ch)
			hasToken = true
		}
	}
	flush()
	return args, nil
}

// formatRedisValue 递归格式化命令输出。
func formatRedisValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case int64:
		return fmt.Sprintf("(integer) %d", x)
	case nil:
		return "(nil)"
	case []any:
		parts := make([]string, 0, len(x))
		for i, item := range x {
			parts = append(parts, fmt.Sprintf("%d) %q", i+1, formatRedisValue(item)))
		}
		return strings.Join(parts, "\n")
	case map[any]any:
		keys := make([]string, 0, len(x))
		m := map[string]string{}
		for k, item := range x {
			ks := fmt.Sprintf("%v", k)
			keys = append(keys, ks)
			m[ks] = formatRedisValue(item)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+": "+m[k])
		}
		return strings.Join(parts, "\n")
	case redis.StatusCmd:
		return x.Val()
	default:
		return fmt.Sprintf("%v", x)
	}
}

// RedisDump 单 key 的序列化导出（DUMP 原始值 + TTL）。
type RedisDump struct {
	Name string
	TTL  int64 // 秒；-1 永不过期
	Data []byte
}

// ExportDumps 导出指定逻辑库的一批 key（DUMP 原始形态），返回下一游标。
func (d *redisDriver) ExportDumps(ctx context.Context, database string, cursor uint64, count int64) ([]RedisDump, uint64, error) {
	idx, err := redisDBFromName(database)
	if err != nil {
		return nil, 0, err
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	keys, next, err := db.Scan(ctx, cursor, "*", count).Result()
	if err != nil {
		return nil, 0, err
	}
	out := make([]RedisDump, 0, len(keys))
	for _, k := range keys {
		data, err := db.Dump(ctx, k).Bytes()
		if err != nil {
			continue // 导出瞬间过期的 key 跳过
		}
		ttl := int64(-1)
		if t, err := db.TTL(ctx, k).Result(); err == nil && t > 0 {
			ttl = int64(t.Seconds())
		}
		out = append(out, RedisDump{Name: k, TTL: ttl, Data: data})
	}
	return out, next, nil
}

// ImportKeys 导入一批 key（RESTORE REPLACE，保真二进制），返回成功数。
func (d *redisDriver) ImportKeys(ctx context.Context, database string, items []RedisDump) (int, error) {
	idx, err := redisDBFromName(database)
	if err != nil {
		return 0, err
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	n := 0
	for _, it := range items {
		ttl := it.TTL
		if ttl < 0 {
			ttl = 0 // RESTORE 的 0 = 不过期
		}
		if err := db.RestoreReplace(ctx, it.Name, time.Duration(ttl)*time.Second, string(it.Data)).Err(); err != nil {
			continue
		}
		n++
	}
	return n, nil
}

// FlushDatabase 清空指定逻辑库（迁移覆盖目标库用）。
func (d *redisDriver) FlushDatabase(ctx context.Context, database string) error {
	idx, err := redisDBFromName(database)
	if err != nil {
		return err
	}
	db := d.clientForDB(idx)
	defer func() { _ = db.Close() }()
	return db.FlushDB(ctx).Err()
}
