// 远程访问状态回读（remote 管理用户是否存在）。
package dbdriver

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

func (d *mysqlDriver) RemoteEnabled(ctx context.Context) (bool, error) {
	var n int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM mysql.user WHERE user = 'remote'").Scan(&n)
	return n > 0, err
}

func (d *pgDriver) RemoteEnabled(ctx context.Context) (bool, error) {
	p, err := d.poolFor(ctx, "postgres")
	if err != nil {
		return false, err
	}
	var n int
	err = p.QueryRow(ctx, "SELECT COUNT(*) FROM pg_roles WHERE rolname = 'remote'").Scan(&n)
	return n > 0, err
}

func (d *redisDriver) RemoteEnabled(ctx context.Context) (bool, error) {
	lines, err := d.rdb.ACLList(ctx).Result()
	if err != nil {
		return false, err
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "user remote ") {
			return true, nil
		}
	}
	return false, nil
}

func (d *mongoDriver) RemoteEnabled(ctx context.Context) (bool, error) {
	var res struct {
		Users []struct {
			User string `bson:"user"`
		} `bson:"users"`
	}
	err := d.client.Database("admin").RunCommand(ctx, bson.D{{Key: "usersInfo", Value: "remote"}}).Decode(&res)
	if err != nil {
		return false, err
	}
	for _, u := range res.Users {
		if u.User == "remote" {
			return true, nil
		}
	}
	return false, nil
}
