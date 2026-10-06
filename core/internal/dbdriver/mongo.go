package dbdriver

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ypanel/shared/errs"
)

type mongoDriver struct {
	client *mongo.Client
}

func newMongo(host string, port int, user, password string) (Driver, error) {
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=admin", user, password, host, port)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return &mongoDriver{client: client}, nil
}

func (d *mongoDriver) Ping(ctx context.Context) error {
	return d.client.Ping(ctx, nil)
}

func (d *mongoDriver) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	res, err := d.client.ListDatabases(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	out := []DatabaseInfo{}
	for _, db := range res.Databases {
		if db.Name == "admin" || db.Name == "config" || db.Name == "local" {
			continue
		}
		out = append(out, DatabaseInfo{Name: db.Name, SizeMB: float64(db.SizeOnDisk) / 1024 / 1024})
	}
	return out, nil
}

// CreateDatabase Mongo 无显式建库（首集合写入时创建）；此处以创建占位集合实现。
func (d *mongoDriver) CreateDatabase(ctx context.Context, name, charset string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	return d.client.Database(name).CreateCollection(ctx, "_placeholder")
}

// DropDatabase 删除整个数据库。
func (d *mongoDriver) DropDatabase(ctx context.Context, name string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	return d.client.Database(name).Drop(ctx)
}

// ListUsers 列出 admin 库用户。
func (d *mongoDriver) ListUsers(ctx context.Context) ([]UserInfo, error) {
	var res struct {
		Users []struct {
			User string `bson:"user"`
		} `bson:"users"`
	}
	if err := d.client.Database("admin").RunCommand(ctx, bson.D{{Key: "usersInfo", Value: 1}}).Decode(&res); err != nil {
		return nil, err
	}
	out := make([]UserInfo, 0, len(res.Users))
	for _, u := range res.Users {
		out = append(out, UserInfo{Name: u.User, Extra: "authSource=admin"})
	}
	return out, nil
}

func (d *mongoDriver) CreateUser(ctx context.Context, name, host, password string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	cmd := bson.D{
		{Key: "createUser", Value: name},
		{Key: "pwd", Value: password},
		{Key: "roles", Value: bson.A{bson.D{{Key: "role", Value: "readWriteAnyDatabase"}, {Key: "db", Value: "admin"}}}},
	}
	return d.client.Database("admin").RunCommand(ctx, cmd).Err()
}

func (d *mongoDriver) DropUser(ctx context.Context, name, host string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	return d.client.Database("admin").RunCommand(ctx, bson.D{{Key: "dropUser", Value: name}}).Err()
}

func (d *mongoDriver) ChangePassword(ctx context.Context, name, host, password string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	cmd := bson.D{{Key: "updateUser", Value: name}, {Key: "pwd", Value: password}}
	return d.client.Database("admin").RunCommand(ctx, cmd).Err()
}

func (d *mongoDriver) Close() {
	_ = d.client.Disconnect(context.Background())
}
