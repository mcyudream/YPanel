package dbdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ypanel/shared/errs"
)

// extToAny 解析 ExtJSON 并归一化：UnmarshalExtJSON 到 any 时 $oid/$date 落成 primitive.D，
// 直接作为 _id/文档值写入会变形（ObjectId 变嵌套文档），必须递归转换。
func extToAny(data []byte, out any) error {
	if err := bson.UnmarshalExtJSON(data, false, out); err != nil {
		return err
	}
	return nil
}

// normalizeExt 递归转换 ExtJSON 解析产物。
// 注意 bson.M/bson.D 是定义类型：json.Unmarshal 产物（map[string]any/[]any）不会命中它们的 case，
// 必须同时匹配底层类型（导入通道的文档来自 json.Unmarshal）。
func normalizeExt(v any) any {
	switch x := v.(type) {
	case bson.D:
		if oid, ok := extObjectID(x); ok {
			return oid
		}
		if t, ok := extDateTime(x); ok {
			return t
		}
		out := make(bson.D, 0, len(x))
		for _, e := range x {
			out = append(out, bson.E{Key: e.Key, Value: normalizeExt(e.Value)})
		}
		return out
	case bson.M:
		return normalizeMap(x)
	case map[string]any:
		return normalizeMap(x)
	case bson.A:
		return normalizeSlice(x)
	case []any:
		return normalizeSlice(x)
	default:
		return v
	}
}

func normalizeMap(x map[string]any) any {
	// 单键 $oid/$date 还原（map 形态，与 D 分支同义）
	if len(x) == 1 {
		if v, ok := x["$oid"]; ok {
			if sv, ok := v.(string); ok {
				if oid, err := primitive.ObjectIDFromHex(sv); err == nil {
					return oid
				}
			}
		}
		if v, ok := x["$date"]; ok {
			if t, ok2 := extDateValue(v); ok2 {
				return t
			}
		}
	}
	out := make(bson.M, len(x))
	for k, vv := range x {
		out[k] = normalizeExt(vv)
	}
	return out
}

func normalizeSlice(x []any) any {
	out := make(bson.A, len(x))
	for i, vv := range x {
		out[i] = normalizeExt(vv)
	}
	return out
}

func extObjectID(d bson.D) (primitive.ObjectID, bool) {
	if len(d) == 1 && d[0].Key == "$oid" {
		if s, ok := d[0].Value.(string); ok {
			if oid, err := primitive.ObjectIDFromHex(s); err == nil {
				return oid, true
			}
		}
	}
	return primitive.NilObjectID, false
}

func extDateTime(d bson.D) (time.Time, bool) {
	if len(d) == 1 && d[0].Key == "$date" {
		return extDateValue(d[0].Value)
	}
	return time.Time{}, false
}

func extDateValue(v any) (time.Time, bool) {
	switch x := v.(type) {
	case string: // relaxed 形态
		if t, err := time.Parse(time.RFC3339, x); err == nil {
			return t, true
		}
	case bson.D: // canonical 形态 {"$numberLong": "毫秒"}
		if len(x) == 1 && x[0].Key == "$numberLong" {
			if s, ok := x[0].Value.(string); ok {
				if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
					return time.UnixMilli(ms), true
				}
			}
		}
	case int64:
		return time.UnixMilli(x), true
	}
	return time.Time{}, false
}

// validateMongoNS 库与集合名校验（集合名允许点号分隔，如 logs.2024）。
func validateMongoNS(database, coll string) error {
	if err := ValidateIdent(database); err != nil {
		return err
	}
	if coll == "" || len(coll) > 120 {
		return errs.Wrap(errs.ErrBadRequest, "集合名不合法")
	}
	for _, r := range coll {
		if !(r == '.' || r == '-' || r == '_' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return errs.Wrap(errs.ErrBadRequest, "集合名含不合法字符: "+coll)
		}
	}
	return nil
}

// ListTables 集合清单（含文档数/存储大小统计，视图标注 kind=view）。
func (d *mongoDriver) ListTables(ctx context.Context, database string) ([]TableInfo, error) {
	if err := ValidateIdent(database); err != nil {
		return nil, err
	}
	specs, err := d.client.Database(database).ListCollectionSpecifications(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	out := make([]TableInfo, 0, len(specs))
	for _, spec := range specs {
		t := TableInfo{Name: spec.Name, Kind: "collection"}
		if spec.Type == "view" {
			t.Kind = "view"
		}
		var st struct {
			Count        int64 `bson:"count"`
			StorageStats struct {
				Size int64 `bson:"size"`
			} `bson:"storageStats"`
		}
		// 视图无 collStats，忽略错误
		if err := d.client.Database(database).RunCommand(ctx, bson.D{{Key: "collStats", Value: spec.Name}}).Decode(&st); err == nil {
			t.Rows = st.Count
			t.SizeMB = float64(st.StorageStats.Size) / 1024 / 1024
		}
		out = append(out, t)
	}
	return out, nil
}

// parseFilterJSON 解析查询过滤 JSON（ExtJSON，$oid/$date 归一化为原生类型）。
func parseFilterJSON(filterJSON string) (bson.D, error) {
	filter := bson.D{}
	if strings.TrimSpace(filterJSON) == "" {
		return filter, nil
	}
	if err := extToAny([]byte(filterJSON), &filter); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "过滤 JSON 不合法: "+err.Error())
	}
	return normalizeExt(filter).(bson.D), nil
}

// parseDocID 解析文档 _id（ExtJSON 值：{"$oid":...} / 字符串 / 数字）。
func parseDocID(idJSON string) (any, error) {
	idJSON = strings.TrimSpace(idJSON)
	if idJSON == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "缺少文档 _id")
	}
	var id any
	if err := bson.UnmarshalExtJSON([]byte(idJSON), false, &id); err != nil {
		// 兼容裸字符串（非 JSON 引号形式）
		return idJSON, nil
	}
	return normalizeExt(id), nil
}

// docToExtJSON 文档序列化为 canonical ExtJSON（ObjectId/日期往返保真）。
func docToExtJSON(doc bson.M) (string, error) {
	raw, err := bson.MarshalExtJSON(doc, false, false)
	if err != nil {
		return "", errs.Wrapc(errs.CodeInternal, err.Error())
	}
	return string(raw), nil
}

// FindDocs 文档分页查询（filter/project/sort/skip/limit + 总数）。
func (d *mongoDriver) FindDocs(ctx context.Context, database, coll, filterJSON, projectJSON, sortField, sortDir string, skip, limit int) (*MongoDocPage, error) {
	if err := validateMongoNS(database, coll); err != nil {
		return nil, err
	}
	filter, err := parseFilterJSON(filterJSON)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if skip < 0 {
		skip = 0
	}
	c := d.client.Database(database).Collection(coll)
	opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))
	if strings.TrimSpace(projectJSON) != "" {
		var proj bson.D
		if err := extToAny([]byte(projectJSON), &proj); err != nil {
			return nil, errs.Wrap(errs.ErrBadRequest, "投影 JSON 不合法: "+err.Error())
		}
		opts.SetProjection(normalizeExt(proj).(bson.D))
	}
	if strings.TrimSpace(sortField) != "" {
		if err := validateMongoNS(database, sortField); err != nil {
			return nil, errs.Wrap(errs.ErrBadRequest, "排序字段不合法")
		}
		dir := 1
		if sortDir == "desc" {
			dir = -1
		}
		opts.SetSort(bson.D{{Key: sortField, Value: dir}})
	}
	total, err := c.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}
	cursor, err := c.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	out := &MongoDocPage{Total: total, Skip: skip, Limit: limit, Docs: []string{}}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		s, err := docToExtJSON(doc)
		if err != nil {
			return nil, err
		}
		out.Docs = append(out.Docs, s)
	}
	return out, cursor.Err()
}

// aggregateWriteStages 管道中禁止出现的写目标 stage（聚合读通道不允许落库）。
var aggregateWriteStages = map[string]bool{"$out": true, "$merge": true, "$planCacheStats": true}

// Aggregate 聚合管道执行（只读：拦截 $out/$merge；结果上限 maxDocs）。
func (d *mongoDriver) Aggregate(ctx context.Context, database, coll string, stages []string, maxDocs int) (*MongoDocPage, error) {
	if err := validateMongoNS(database, coll); err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		return nil, errs.Wrap(errs.ErrBadRequest, "聚合管道至少需要一个 stage")
	}
	if maxDocs <= 0 || maxDocs > 500 {
		maxDocs = 100
	}
	pipeline := make(mongo.Pipeline, 0, len(stages))
	for i, s := range stages {
		var stage bson.D
		if err := extToAny([]byte(s), &stage); err != nil {
			return nil, errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("第 %d 个 stage JSON 不合法: %s", i+1, err.Error()))
		}
		if len(stage) != 1 {
			return nil, errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("第 %d 个 stage 必须是单操作符文档（如 {\"$match\": {...}}）", i+1))
		}
		if aggregateWriteStages[stage[0].Key] {
			return nil, errs.Wrap(errs.ErrBadRequest, "聚合管道不允许 "+stage[0].Key+"（写目标阶段）")
		}
		pipeline = append(pipeline, bson.D{{Key: stage[0].Key, Value: normalizeExt(stage[0].Value)}})
	}
	c := d.client.Database(database).Collection(coll)
	cursor, err := c.Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true))
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer cursor.Close(ctx)
	out := &MongoDocPage{Skip: 0, Limit: maxDocs, Docs: []string{}}
	for cursor.Next(ctx) && len(out.Docs) < maxDocs {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		s, err := docToExtJSON(doc)
		if err != nil {
			return nil, err
		}
		out.Docs = append(out.Docs, s)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	out.Total = int64(len(out.Docs))
	return out, nil
}

// GetDoc 单文档（ExtJSON）。
func (d *mongoDriver) GetDoc(ctx context.Context, database, coll, idJSON string) (string, error) {
	if err := validateMongoNS(database, coll); err != nil {
		return "", err
	}
	id, err := parseDocID(idJSON)
	if err != nil {
		return "", err
	}
	var doc bson.M
	if err := d.client.Database(database).Collection(coll).FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&doc); err != nil {
		return "", errs.Wrap(errs.ErrBadRequest, "文档不存在")
	}
	return docToExtJSON(doc)
}

// InsertDoc 插入文档（含 _id 则用之；$oid/$date 归一化）。
func (d *mongoDriver) InsertDoc(ctx context.Context, database, coll, docJSON string) error {
	if err := validateMongoNS(database, coll); err != nil {
		return err
	}
	var doc bson.M
	if err := extToAny([]byte(docJSON), &doc); err != nil {
		return errs.Wrap(errs.ErrBadRequest, "文档 JSON 不合法: "+err.Error())
	}
	_, err := d.client.Database(database).Collection(coll).InsertOne(ctx, normalizeExt(doc).(bson.M))
	return err
}

// UpdateDoc 整体替换文档（保留提交内容中的 _id；$oid/$date 归一化）。
func (d *mongoDriver) UpdateDoc(ctx context.Context, database, coll, idJSON, docJSON string) error {
	if err := validateMongoNS(database, coll); err != nil {
		return err
	}
	id, err := parseDocID(idJSON)
	if err != nil {
		return err
	}
	var doc bson.M
	if err := extToAny([]byte(docJSON), &doc); err != nil {
		return errs.Wrap(errs.ErrBadRequest, "文档 JSON 不合法: "+err.Error())
	}
	res, err := d.client.Database(database).Collection(coll).ReplaceOne(ctx, bson.D{{Key: "_id", Value: id}}, normalizeExt(doc).(bson.M))
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errs.Wrap(errs.ErrBadRequest, "文档不存在")
	}
	return nil
}

// DeleteDoc 删除文档。
func (d *mongoDriver) DeleteDoc(ctx context.Context, database, coll, idJSON string) error {
	if err := validateMongoNS(database, coll); err != nil {
		return err
	}
	id, err := parseDocID(idJSON)
	if err != nil {
		return err
	}
	res, err := d.client.Database(database).Collection(coll).DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errs.Wrap(errs.ErrBadRequest, "文档不存在")
	}
	return nil
}

// ListIndexes 索引清单（含大小与使用统计，对齐 Compass 索引页）。
func (d *mongoDriver) ListIndexes(ctx context.Context, database, coll string) ([]MongoIndex, error) {
	if err := validateMongoNS(database, coll); err != nil {
		return nil, err
	}
	// 1) 规格（name/keys/unique）
	type idxSpec struct {
		Name   string `bson:"name"`
		Key    bson.D `bson:"key"`
		Unique bool   `bson:"unique"`
	}
	cur, err := d.client.Database(database).Collection(coll).Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	specs := []idxSpec{}
	for cur.Next(ctx) {
		var s idxSpec
		if err := cur.Decode(&s); err != nil {
			cur.Close(ctx)
			return nil, err
		}
		specs = append(specs, s)
	}
	cur.Close(ctx)

	// 2) 使用统计（$indexStats）：accesses.ops / accesses.since
	acc := map[string]struct {
		Ops   int64
		Since time.Time
	}{}
	var statsOut struct {
		Cursor struct {
			Batch []struct {
				Name     string `bson:"name"`
				Accesses struct {
					Ops   int64     `bson:"ops"`
					Since time.Time `bson:"since"`
				} `bson:"accesses"`
			} `bson:"batch"`
		} `bson:"cursor"`
	}
	err = d.client.Database(database).RunCommand(ctx, bson.D{
		{Key: "aggregate", Value: coll},
		{Key: "pipeline", Value: bson.A{bson.D{{Key: "$indexStats", Value: bson.M{}}}}},
		{Key: "cursor", Value: bson.M{}},
	}).Decode(&statsOut)
	if err == nil {
		for _, a := range statsOut.Cursor.Batch {
			acc[a.Name] = struct {
				Ops   int64
				Since time.Time
			}{a.Accesses.Ops, a.Accesses.Since}
		}
	}

	// 3) 索引大小（collStats.indexSizes）
	sizes := map[string]int64{}
	var cs struct {
		IndexSizes map[string]int64 `bson:"indexSizes"`
	}
	if err := d.client.Database(database).RunCommand(ctx, bson.D{{Key: "collStats", Value: coll}}).Decode(&cs); err == nil {
		sizes = cs.IndexSizes
	}

	// 4) 组装
	out := make([]MongoIndex, 0, len(specs))
	for _, s := range specs {
		keys := map[string]int{}
		for _, kv := range s.Key {
			switch v := kv.Value.(type) {
			case int32:
				keys[kv.Key] = int(v)
			case float64:
				keys[kv.Key] = int(v)
			case string:
				if v == "desc" {
					keys[kv.Key] = -1
				} else {
					keys[kv.Key] = 1
				}
			default:
				keys[kv.Key] = 1
			}
		}
		mi := MongoIndex{Name: s.Name, Keys: keys, Unique: s.Unique}
		if sz, ok := sizes[s.Name]; ok {
			mi.SizeMB = float64(sz) / 1024 / 1024
		}
		if a, ok := acc[s.Name]; ok {
			mi.Accesses = a.Ops
			if !a.Since.IsZero() {
				mi.AccessedAt = a.Since.Format("2006-01-02 15:04")
			}
		}
		out = append(out, mi)
	}
	return out, nil
}

// CreateIndex 创建索引。
func (d *mongoDriver) CreateIndex(ctx context.Context, database, coll, name string, keys map[string]int, unique bool) error {
	if err := validateMongoNS(database, coll); err != nil {
		return err
	}
	if len(keys) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "索引至少需要一个字段")
	}
	keyDoc := bson.D{}
	for f, dir := range keys {
		if err := validateMongoNS(database, f); err != nil {
			return errs.Wrap(errs.ErrBadRequest, "索引字段不合法: "+f)
		}
		d := 1
		if dir < 0 {
			d = -1
		}
		keyDoc = append(keyDoc, bson.E{Key: f, Value: d})
	}
	opts := options.Index().SetUnique(unique)
	if strings.TrimSpace(name) != "" {
		if err := validateMongoNS(database, name); err != nil {
			return errs.Wrap(errs.ErrBadRequest, "索引名不合法")
		}
		opts.SetName(name)
	}
	_, err := d.client.Database(database).Collection(coll).Indexes().CreateOne(ctx, mongo.IndexModel{Keys: keyDoc, Options: opts})
	return err
}

// DropIndex 删除索引（_id_ 不可删）。
func (d *mongoDriver) DropIndex(ctx context.Context, database, coll, name string) error {
	if err := validateMongoNS(database, coll); err != nil {
		return err
	}
	if name == "_id_" {
		return errs.Wrap(errs.ErrBadRequest, "_id_ 索引不可删除")
	}
	_, err := d.client.Database(database).Collection(coll).Indexes().DropOne(ctx, name)
	return err
}

// CreateCollection 创建集合。
func (d *mongoDriver) CreateCollection(ctx context.Context, database, name string) error {
	if err := validateMongoNS(database, name); err != nil {
		return err
	}
	return d.client.Database(database).CreateCollection(ctx, name)
}

// DropCollection 删除集合。
func (d *mongoDriver) DropCollection(ctx context.Context, database, name string) error {
	if err := validateMongoNS(database, name); err != nil {
		return err
	}
	return d.client.Database(database).Collection(name).Drop(ctx)
}

// RenameCollection 重命名集合。
func (d *mongoDriver) RenameCollection(ctx context.Context, database, from, to string) error {
	if err := validateMongoNS(database, from); err != nil {
		return err
	}
	if err := validateMongoNS(database, to); err != nil {
		return err
	}
	return d.client.Database(database).RunCommand(ctx, bson.D{
		{Key: "renameCollection", Value: database + "." + from},
		{Key: "to", Value: database + "." + to},
	}).Err()
}

// ImportDocs 导入 Compass 导出的 JSON 数组或 NDJSON 文档（ExtJSON 归一化），返回导入条数。
func (d *mongoDriver) ImportDocs(ctx context.Context, database, coll, content, format string) (int, error) {
	if err := validateMongoNS(database, coll); err != nil {
		return 0, err
	}
	if len(content) > 10<<20 {
		return 0, errs.Wrap(errs.ErrBadRequest, "导入内容超过 10MB 上限")
	}
	var docs []map[string]any
	switch format {
	case "", "json":
		// JSON 数组；兼容单文档对象
		trimmed := strings.TrimSpace(content)
		if strings.HasPrefix(trimmed, "{") {
			var one map[string]any
			if err := json.Unmarshal([]byte(trimmed), &one); err != nil {
				return 0, errs.Wrap(errs.ErrBadRequest, "JSON 不合法: "+err.Error())
			}
			docs = []map[string]any{one}
		} else {
			if err := json.Unmarshal([]byte(trimmed), &docs); err != nil {
				return 0, errs.Wrap(errs.ErrBadRequest, "JSON 数组不合法: "+err.Error())
			}
		}
	case "ndjson":
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var one map[string]any
			if err := json.Unmarshal([]byte(line), &one); err != nil {
				return 0, errs.Wrap(errs.ErrBadRequest, "NDJSON 第 "+strconv.Itoa(len(docs)+1)+" 行不合法: "+err.Error())
			}
			docs = append(docs, one)
		}
	default:
		return 0, errs.Wrap(errs.ErrBadRequest, "不支持的格式: "+format)
	}
	if len(docs) == 0 {
		return 0, errs.Wrap(errs.ErrBadRequest, "未解析到文档")
	}
	c := d.client.Database(database).Collection(coll)
	n := 0
	for start := 0; start < len(docs); start += 500 {
		end := start + 500
		if end > len(docs) {
			end = len(docs)
		}
		batch := make([]any, 0, end-start)
		for _, doc := range docs[start:end] {
			// bson.M 即 map[string]any：json 产物里的 {"$oid":...}/{"$date":...} 由 normalizeExt 递归还原原生类型
			batch = append(batch, normalizeExt(doc))
		}
		res, err := c.InsertMany(ctx, batch)
		if err != nil {
			return n, errs.Wrapc(errs.CodeFileOpFailed, "第 "+strconv.Itoa(start+1)+"-"+strconv.Itoa(end)+" 条批量插入失败: "+err.Error())
		}
		n += len(res.InsertedIDs)
	}
	return n, nil
}
