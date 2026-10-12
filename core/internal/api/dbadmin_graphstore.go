// DB Admin Neo4j 图工作台与 S3 对象工作台端点（M68）。
package api

import (
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/shared/errs"
)

// queryInt 查询参数取 int（缺省 defaultVal）。
func queryInt(c *gin.Context, key string, defaultVal int) int {
	if v, err := strconv.Atoi(c.Query(key)); err == nil {
		return v
	}
	return defaultVal
}

// ---- Neo4j ----

// Neo4jMeta GET /plugin/db-admin/:id/neo4j/meta?db=
func (a *DBAdminAPI) Neo4jMeta(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.Neo4jGraphMeta(c.Request.Context(), id, c.Query("db"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Neo4jCypher POST /plugin/db-admin/:id/neo4j/cypher {db, cypher, limit?}
func (a *DBAdminAPI) Neo4jCypher(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB     string `json:"db" binding:"required"`
		Cypher string `json:"cypher" binding:"required"`
		Limit  int    `json:"limit"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Admin.Neo4jCypher(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Cypher, req.Limit)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Neo4jNodes GET /plugin/db-admin/:id/neo4j/nodes?db=&label=&skip=&limit=
func (a *DBAdminAPI) Neo4jNodes(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	skip := queryInt(c, "skip", 0)
	limit := queryInt(c, "limit", 50)
	out, err := a.Admin.Neo4jNodes(c.Request.Context(), id, c.Query("db"), c.Query("label"), skip, limit)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Neo4jNodeCreate POST /plugin/db-admin/:id/neo4j/node {db, label, props}
func (a *DBAdminAPI) Neo4jNodeCreate(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB    string `json:"db" binding:"required"`
		Label string `json:"label" binding:"required"`
		Props string `json:"props"`
	}](c)
	if !ok {
		return
	}
	nid, err := a.Admin.Neo4jNodeCreate(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Label, req.Props)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": nid})
}

// Neo4jNodeUpdate POST /plugin/db-admin/:id/neo4j/node/update {db, id, props}
func (a *DBAdminAPI) Neo4jNodeUpdate(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB    string `json:"db" binding:"required"`
		ID    string `json:"id" binding:"required"`
		Props string `json:"props" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.Neo4jNodeUpdate(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.ID, req.Props); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// Neo4jNodeDelete POST /plugin/db-admin/:id/neo4j/node/delete {db, id, detach?}
func (a *DBAdminAPI) Neo4jNodeDelete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB     string `json:"db" binding:"required"`
		ID     string `json:"id" binding:"required"`
		Detach bool   `json:"detach"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.Neo4jNodeDelete(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.ID, req.Detach); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// Neo4jIndexes GET /plugin/db-admin/:id/neo4j/indexes?db=&label=
func (a *DBAdminAPI) Neo4jIndexes(c *gin.Context) {
	a.neo4jSchemaList(c, true)
}

// Neo4jConstraints GET /plugin/db-admin/:id/neo4j/constraints?db=&label=
func (a *DBAdminAPI) Neo4jConstraints(c *gin.Context) {
	a.neo4jSchemaList(c, false)
}

// Neo4jIndexCreate POST /plugin/db-admin/:id/neo4j/index/create {db, name?, label, props, unique?}
func (a *DBAdminAPI) Neo4jIndexCreate(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB     string   `json:"db" binding:"required"`
		Name   string   `json:"name"`
		Label  string   `json:"label" binding:"required"`
		Props  []string `json:"props" binding:"required"`
		Unique bool     `json:"unique"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.Neo4jIndexCreate(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Name, req.Label, req.Props, req.Unique); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// Neo4jIndexDrop POST /plugin/db-admin/:id/neo4j/index/drop {db, name}
func (a *DBAdminAPI) Neo4jIndexDrop(c *gin.Context) {
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
	if err := a.Admin.Neo4jIndexDrop(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Name); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// neo4jSchemaList 索引/约束清单共用（仅服务层分别调用）。
func (a *DBAdminAPI) neo4jSchemaList(c *gin.Context, indexes bool) {
	// 索引与约束清单走 RunCypher 通道不合适（SHOW 语句结果列动态），这里直接复用 Cypher 端点语义。
	// 为保持权限与审计口径一致，统一走 Neo4jCypher（SHOW 为只读语句）。
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	cy := "SHOW INDEXES YIELD name, type, entityType, labelsOrTypes, properties, state RETURN name, type, entityType, labelsOrTypes, properties, state ORDER BY name"
	if !indexes {
		cy = "SHOW CONSTRAINTS YIELD name, type, entityType, labelsOrTypes, properties RETURN name, type, entityType, labelsOrTypes, properties ORDER BY name"
	}
	out, err := a.Admin.Neo4jCypher(c.Request.Context(), id, c.GetString(middleware.CtxUsername), c.DefaultQuery("db", "neo4j"), cy, 500)
	if err != nil {
		respErr(c, err)
		return
	}
	// 列式 → 行对象数组（前端表格友好）
	rows := make([]map[string]any, 0, len(out.Rows))
	for _, r := range out.Rows {
		m := map[string]any{}
		for i, col := range out.Columns {
			if i < len(r) {
				m[col] = r[i]
			}
		}
		rows = append(rows, m)
	}
	respOK(c, rows)
}

// ---- S3 ----

// S3Objects GET /plugin/db-admin/:id/s3/objects?db=&prefix=&startAfter=&limit=
func (a *DBAdminAPI) S3Objects(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	limit := queryInt(c, "limit", 200)
	out, err := a.Admin.S3Objects(c.Request.Context(), id, c.Query("db"), c.Query("prefix"), c.Query("startAfter"), limit)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// S3ObjectDelete POST /plugin/db-admin/:id/s3/object/delete {db, key}
func (a *DBAdminAPI) S3ObjectDelete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB  string `json:"db" binding:"required"`
		Key string `json:"key" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Admin.S3ObjectDelete(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.DB, req.Key); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// S3ObjectUpload POST /plugin/db-admin/:id/s3/object/upload multipart（db + key + file）
func (a *DBAdminAPI) S3ObjectUpload(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	bucket := c.PostForm("db")
	key := c.PostForm("key")
	if bucket == "" || key == "" {
		respErr(c, errs.Wrap(errs.ErrBadRequest, "bucket/db 与 key 必填"))
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		respErr(c, errs.Wrap(errs.ErrBadRequest, "缺少上传文件: "+err.Error()))
		return
	}
	f, err := fh.Open()
	if err != nil {
		respErr(c, err)
		return
	}
	defer func() { _ = f.Close() }()
	if err := a.Admin.S3ObjectUpload(c.Request.Context(), id, c.GetString(middleware.CtxUsername), bucket, key, f, fh.Size); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"key": key, "size": fh.Size})
}

// S3ObjectDownload GET /plugin/db-admin/:id/s3/object/download?db=&key= （流式）
func (a *DBAdminAPI) S3ObjectDownload(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	bucket := c.Query("db")
	key := c.Query("key")
	if bucket == "" || key == "" {
		respErr(c, errs.Wrap(errs.ErrBadRequest, "bucket/db 与 key 必填"))
		return
	}
	rc, size, err := a.Admin.S3ObjectRead(c.Request.Context(), id, c.GetString(middleware.CtxUsername), bucket, key)
	if err != nil {
		respErr(c, err)
		return
	}
	defer func() { _ = rc.Close() }()
	name := url.PathEscape(key)
	if i := len(name) - 1; i >= 0 {
		for j := i; j >= 0; j-- {
			if name[j] == '/' {
				name = name[j+1:]
				break
			}
		}
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", name))
	if size > 0 {
		c.DataFromReader(200, size, "application/octet-stream", rc, nil)
	} else {
		c.Status(200)
		_, _ = io.Copy(c.Writer, rc)
	}
}
