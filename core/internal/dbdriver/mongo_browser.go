package dbdriver

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/ypanel/shared/errs"
)

// ListTables mongo 集合清单。
func (d *mongoDriver) ListTables(ctx context.Context, database string) ([]TableInfo, error) {
	if err := ValidateIdent(database); err != nil {
		return nil, err
	}
	cols, err := d.client.Database(database).ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	out := make([]TableInfo, 0, len(cols))
	for _, c := range cols {
		out = append(out, TableInfo{Name: c})
	}
	return out, nil
}

// Query mongo：find 前 limit 个文档（JSON 序列化展示），sqlText 作过滤 JSON（空 = 全部）。
func (d *mongoDriver) Query(ctx context.Context, database, sqlText string, limit int) (*QueryResult, error) {
	if err := ValidateIdent(database); err != nil {
		return nil, err
	}
	collection := stringsCollectionName(sqlText)
	if collection == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "Mongo 查询格式：\"<集合名>\" 或集合名+空格+过滤JSON")
	}
	filterJSON := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(sqlText), collection))
	filter := bson.D{}
	if filterJSON != "" {
		if err := bson.UnmarshalExtJSON([]byte(filterJSON), false, &filter); err != nil {
			return nil, errs.Wrap(errs.ErrBadRequest, "过滤 JSON 不合法: "+err.Error())
		}
	}
	cursor, err := d.client.Database(database).Collection(collection).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	result := &QueryResult{Columns: []string{"document"}}
	for cursor.Next(ctx) && len(result.Rows) < limit {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, []interface{}{doc})
	}
	return result, cursor.Err()
}
