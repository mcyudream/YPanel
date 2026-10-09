package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/dbdriver"
	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/errs"
)

// DBAdminAPI db-admin 插件接口（应库而宜的浏览/查询/结构/索引/数据编辑）。
type DBAdminAPI struct {
	Admin *service.DBAdminService
}

// Instances GET /plugin/db-admin/instances
func (a *DBAdminAPI) Instances(c *gin.Context) {
	out, err := a.Admin.Instances(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Ping GET /plugin/db-admin/:id/ping → {ms}
func (a *DBAdminAPI) Ping(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	ms, err := a.Admin.Latency(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"ms": ms})
}

// Databases GET /plugin/db-admin/:id/databases
func (a *DBAdminAPI) Databases(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.Databases(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Schemas GET /plugin/db-admin/:id/schemas?db=
func (a *DBAdminAPI) Schemas(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.Schemas(c.Request.Context(), id, c.Query("db"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Tables GET /plugin/db-admin/:id/tables?db=&schema=
func (a *DBAdminAPI) Tables(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.Tables(c.Request.Context(), id, c.Query("db"), c.Query("schema"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Columns GET /plugin/db-admin/:id/columns?db=&table=&schema=
func (a *DBAdminAPI) Columns(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.Columns(c.Request.Context(), id, c.Query("db"), c.Query("schema"), c.Query("table"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// TableDDL GET /plugin/db-admin/:id/ddl?db=&table=&schema=
func (a *DBAdminAPI) TableDDL(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.TableDDL(c.Request.Context(), id, c.Query("db"), c.Query("schema"), c.Query("table"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"ddl": out})
}

// Indexes GET /plugin/db-admin/:id/indexes?db=&table=&schema=
func (a *DBAdminAPI) Indexes(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.Indexes(c.Request.Context(), id, c.Query("db"), c.Query("schema"), c.Query("table"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// PrimaryKey GET /plugin/db-admin/:id/pk?db=&table=&schema=
func (a *DBAdminAPI) PrimaryKey(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.PrimaryKey(c.Request.Context(), id, c.Query("db"), c.Query("schema"), c.Query("table"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Query POST /plugin/db-admin/:id/query {db, sql, limit?}
func (a *DBAdminAPI) Query(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB    string `json:"db" binding:"required"`
		SQL   string `json:"sql" binding:"required"`
		Limit int    `json:"limit"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Admin.RunQuery(c.Request.Context(), id, req.DB, req.SQL, req.Limit)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Browse POST /plugin/db-admin/:id/browse {db, schema?, table, sortCol?, sortDir?, offset?, limit?}
func (a *DBAdminAPI) Browse(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[dbdriver.BrowsePageReq](c)
	if !ok {
		return
	}
	if req.Table == "" || req.DB == "" {
		respErr(c, errBadRequest("db 与 table 必填"))
		return
	}
	out, err := a.Admin.BrowsePage(c.Request.Context(), id, *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// RowInsert POST /plugin/db-admin/:id/rows/insert
func (a *DBAdminAPI) RowInsert(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[dbdriver.RowEdit](c)
	if !ok {
		return
	}
	if err := a.Admin.InsertRow(c.Request.Context(), id, c.GetString(middleware.CtxUsername), *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// RowUpdate POST /plugin/db-admin/:id/rows/update
func (a *DBAdminAPI) RowUpdate(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[dbdriver.RowEdit](c)
	if !ok {
		return
	}
	if err := a.Admin.UpdateRow(c.Request.Context(), id, c.GetString(middleware.CtxUsername), *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// RowDelete POST /plugin/db-admin/:id/rows/delete
func (a *DBAdminAPI) RowDelete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[dbdriver.RowEdit](c)
	if !ok {
		return
	}
	if err := a.Admin.DeleteRow(c.Request.Context(), id, c.GetString(middleware.CtxUsername), *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// IndexCreate POST /plugin/db-admin/:id/indexes/create
func (a *DBAdminAPI) IndexCreate(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[dbdriver.IndexRequest](c)
	if !ok {
		return
	}
	if req.DB == "" || req.Table == "" {
		respErr(c, errBadRequest("db 与 table 必填"))
		return
	}
	if err := a.Admin.CreateIndex(c.Request.Context(), id, c.GetString(middleware.CtxUsername), *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// IndexDrop POST /plugin/db-admin/:id/indexes/drop {db, table, schema?, name}
func (a *DBAdminAPI) IndexDrop(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB     string `json:"db" binding:"required"`
		Table  string `json:"table" binding:"required"`
		Schema string `json:"schema"`
		Name   string `json:"name" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.DropIndex(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Schema, req.Table, req.Name); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// ---- Redis ----

// RedisKeys GET /plugin/db-admin/:id/redis/keys?db=&pattern=&cursor=&count=
func (a *DBAdminAPI) RedisKeys(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	cursor, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	count, _ := strconv.ParseInt(c.Query("count"), 10, 64)
	if count <= 0 || count > 1000 {
		count = 100
	}
	keys, next, err := a.Admin.ScanKeys(c.Request.Context(), id, c.Query("db"), c.Query("pattern"), cursor, count)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"keys": keys, "cursor": strconv.FormatUint(next, 10)})
}

// RedisKey GET /plugin/db-admin/:id/redis/key?db=&name=
func (a *DBAdminAPI) RedisKey(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.KeyDetail(c.Request.Context(), id, c.Query("db"), c.Query("name"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// RedisKeyWrite POST /plugin/db-admin/:id/redis/key {db, name, type, value}
func (a *DBAdminAPI) RedisKeyWrite(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[dbdriver.RedisWrite](c)
	if !ok {
		return
	}
	if err := a.Admin.WriteRedisKey(c.Request.Context(), id, c.GetString(middleware.CtxUsername), *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// RedisKeyDelete POST /plugin/db-admin/:id/redis/key/delete {db, name}
func (a *DBAdminAPI) RedisKeyDelete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB   string `json:"db" binding:"required"`
		Name string `json:"name" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.DeleteRedisKey(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Name); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// RedisKeyTTL POST /plugin/db-admin/:id/redis/key/ttl {db, name, ttl}（ttl<0 = persist）
func (a *DBAdminAPI) RedisKeyTTL(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB   string `json:"db" binding:"required"`
		Name string `json:"name" binding:"required"`
		TTL  int64  `json:"ttl"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.SetRedisTTL(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Name, req.TTL); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// RedisExec POST /plugin/db-admin/:id/redis/exec {db, command}
func (a *DBAdminAPI) RedisExec(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB      string `json:"db" binding:"required"`
		Command string `json:"command" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Admin.RedisExec(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Command)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"output": out})
}

// ---- Mongo ----

// MongoDocs POST /plugin/db-admin/:id/mongo/docs {db, collection, filter?, project?, sort?, dir?, skip?, limit?}
func (a *DBAdminAPI) MongoDocs(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB         string `json:"db" binding:"required"`
		Collection string `json:"collection" binding:"required"`
		Filter     string `json:"filter"`
		Project    string `json:"project"`
		Sort       string `json:"sort"`
		Dir        string `json:"dir"`
		Skip       int    `json:"skip"`
		Limit      int    `json:"limit"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Admin.MongoFindDocs(c.Request.Context(), id, req.DB, req.Collection, req.Filter, req.Project, req.Sort, req.Dir, req.Skip, req.Limit)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// MongoAggregate POST /plugin/db-admin/:id/mongo/aggregate {db, collection, stages: [json...], maxDocs?}
func (a *DBAdminAPI) MongoAggregate(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB         string   `json:"db" binding:"required"`
		Collection string   `json:"collection" binding:"required"`
		Stages     []string `json:"stages" binding:"required"`
		MaxDocs    int      `json:"maxDocs"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Admin.MongoAggregate(c.Request.Context(), id, req.DB, req.Collection, req.Stages, req.MaxDocs)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// MongoDoc GET /plugin/db-admin/:id/mongo/doc?db=&collection=&id=
func (a *DBAdminAPI) MongoDoc(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.MongoGetDoc(c.Request.Context(), id, c.Query("db"), c.Query("collection"), c.Query("id"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"doc": out})
}

// MongoDocInsert POST /plugin/db-admin/:id/mongo/doc {db, collection, doc}
func (a *DBAdminAPI) MongoDocInsert(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB         string `json:"db" binding:"required"`
		Collection string `json:"collection" binding:"required"`
		Doc        string `json:"doc" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.MongoInsertDoc(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Collection, req.Doc); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// MongoDocUpdate POST /plugin/db-admin/:id/mongo/doc/update {db, collection, id, doc}
func (a *DBAdminAPI) MongoDocUpdate(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB         string `json:"db" binding:"required"`
		Collection string `json:"collection" binding:"required"`
		ID         string `json:"id" binding:"required"`
		Doc        string `json:"doc" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.MongoUpdateDoc(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Collection, req.ID, req.Doc); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// MongoDocDelete POST /plugin/db-admin/:id/mongo/doc/delete {db, collection, id}
func (a *DBAdminAPI) MongoDocDelete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB         string `json:"db" binding:"required"`
		Collection string `json:"collection" binding:"required"`
		ID         string `json:"id" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.MongoDeleteDoc(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Collection, req.ID); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// MongoIndexes GET /plugin/db-admin/:id/mongo/indexes?db=&collection=
func (a *DBAdminAPI) MongoIndexes(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.MongoIndexes(c.Request.Context(), id, c.Query("db"), c.Query("collection"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// MongoIndexCreate POST /plugin/db-admin/:id/mongo/indexes/create {db, collection, name?, keys, unique?}
func (a *DBAdminAPI) MongoIndexCreate(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB         string         `json:"db" binding:"required"`
		Collection string         `json:"collection" binding:"required"`
		Name       string         `json:"name"`
		Keys       map[string]int `json:"keys" binding:"required"`
		Unique     bool           `json:"unique"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.MongoCreateIndex(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Collection, req.Name, req.Keys, req.Unique); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// MongoIndexDrop POST /plugin/db-admin/:id/mongo/indexes/drop {db, collection, name}
func (a *DBAdminAPI) MongoIndexDrop(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB         string `json:"db" binding:"required"`
		Collection string `json:"collection" binding:"required"`
		Name       string `json:"name" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.MongoDropIndex(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Collection, req.Name); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// MongoCollection POST /plugin/db-admin/:id/mongo/collection {db, action: create/drop/rename, name, to?}
func (a *DBAdminAPI) MongoCollection(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB     string `json:"db" binding:"required"`
		Action string `json:"action" binding:"required,oneof=create drop rename"`
		Name   string `json:"name" binding:"required"`
		To     string `json:"to"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.MongoCollectionAction(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.Action, req.DB, req.Name, req.To); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// ---- 审计 ----

// Audits GET /plugin/db-admin/audits?instanceId=&page=&size=（仅 admin）
func (a *DBAdminAPI) Audits(c *gin.Context) {
	if c.GetString(middleware.CtxRole) != "admin" {
		respErr(c, errs.ErrForbidden)
		return
	}
	instanceID, _ := strconv.ParseUint(c.Query("instanceId"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
	list, total, err := a.Admin.Audits(c.Request.Context(), uint(instanceID), page, size)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"items": list, "total": total})
}

// ImportSQL POST /plugin/db-admin/:id/sql/import {db, content?, srcPath?}
func (a *DBAdminAPI) ImportSQL(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB      string `json:"db" binding:"required"`
		Content string `json:"content"`
		SrcPath string `json:"srcPath"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Admin.ImportSQL(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Content, req.SrcPath)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// MongoImport POST /plugin/db-admin/:id/mongo/import {db, collection, content, format?}
func (a *DBAdminAPI) MongoImport(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB         string `json:"db" binding:"required"`
		Collection string `json:"collection" binding:"required"`
		Content    string `json:"content" binding:"required"`
		Format     string `json:"format"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Admin.MongoImportDocs(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Collection, req.Content, req.Format)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// MigratePreview GET /plugin/db-admin/migrate/preview?src=&target=
func (a *DBAdminAPI) MigratePreview(c *gin.Context) {
	src, err := strconv.ParseUint(c.Query("src"), 10, 64)
	target, err2 := strconv.ParseUint(c.Query("target"), 10, 64)
	if err != nil || err2 != nil || src == 0 || target == 0 {
		respErr(c, errBadRequest("src 与 target 必填"))
		return
	}
	out, err := a.Admin.MigratePreview(c.Request.Context(), uint(src), uint(target))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// MigrateStart POST /plugin/db-admin/migrate/start {src, target, dbs}
func (a *DBAdminAPI) MigrateStart(c *gin.Context) {
	req, ok := bind[struct {
		Src    uint   `json:"src" binding:"required"`
		Target uint   `json:"target" binding:"required"`
		DBs    []string `json:"dbs" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Admin.MigrateStart(c.Request.Context(), c.GetString(middleware.CtxUsername), req.Src, req.Target, req.DBs)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
