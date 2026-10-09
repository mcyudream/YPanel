// DBAdminService 数据库管理台（插件一期：进程内模块，SDK 风格边界）。
// 职责：应库而宜的浏览/查询/结构/索引/数据编辑（SQL 库行级、Redis key、Mongo 文档），写路径全审计。
package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/dbdriver"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// 超时上限（管理台低频操作，短连接模式）。
const (
	dbAdminReadTimeout   = 15 * time.Second
	dbAdminQueryTimeout  = 30 * time.Second
	dbAdminWriteTimeout  = 15 * time.Second
	dbAdminConsoleBgTime = 10 * time.Second
)

// auditDetailMax 审计详情截断长度。
const auditDetailMax = 4000

// DBAdminService db-admin 插件服务。
type DBAdminService struct {
	db  *gorm.DB
	dbs *DatabaseService
}

// NewDBAdminService 创建。
func NewDBAdminService(db *gorm.DB, dbs *DatabaseService) *DBAdminService {
	return &DBAdminService{db: db, dbs: dbs}
}

// Instances 可浏览实例列表（复用 M4）。
func (s *DBAdminService) Instances(ctx context.Context) ([]map[string]any, error) {
	return s.dbs.List(ctx)
}

// Databases 库列表（复用 M4）。
func (s *DBAdminService) Databases(ctx context.Context, instanceID uint) ([]dbdriver.DatabaseInfo, error) {
	return s.dbs.Databases(ctx, instanceID)
}

// withDriver 取驱动短连接并执行（统一超时与 Ping 预检）。
func (s *DBAdminService) withDriver(ctx context.Context, instanceID uint, timeout time.Duration, fn func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error) error {
	inst, err := s.dbs.ByID(instanceID)
	if err != nil {
		return err
	}
	drv, err := s.dbs.driverFor(inst)
	if err != nil {
		return err
	}
	defer drv.Close()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := drv.Ping(ctx); err != nil {
		return errs.Wrap(errs.ErrAgentUnreach, "连接实例失败: "+err.Error())
	}
	return fn(ctx, inst, drv)
}

// Latency 实例连通延迟（毫秒；err 则返回 -1）。
func (s *DBAdminService) Latency(ctx context.Context, instanceID uint) (int64, error) {
	inst, err := s.dbs.ByID(instanceID)
	if err != nil {
		return 0, err
	}
	drv, err := s.dbs.driverFor(inst)
	if err != nil {
		return -1, err
	}
	defer drv.Close()
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	start := time.Now()
	if err := drv.Ping(cctx); err != nil {
		return -1, errs.Wrap(errs.ErrAgentUnreach, "连接实例失败: "+err.Error())
	}
	return time.Since(start).Milliseconds(), nil
}

// audit 写审计（失败不影响主流程，但不静默——落日志语义由 gorm 错误返回忽略+注释说明）。
func (s *DBAdminService) audit(username string, inst *model.DatabaseInstance, kind, target, detail string, success bool) {
	if len(detail) > auditDetailMax {
		detail = detail[:auditDetailMax]
	}
	entry := model.DatabaseAuditLog{
		Username: username, InstanceID: inst.ID, InstanceName: inst.Name,
		DbType: inst.Type, Kind: kind, Target: target, Detail: detail, Success: success,
	}
	// 审计写失败仅丢一条记录，不回滚业务操作
	_ = s.db.Create(&entry).Error
}

// ---- 结构浏览 ----

// Schemas schema 列表（PG 多 schema；MySQL 空）。
func (s *DBAdminService) Schemas(ctx context.Context, instanceID uint, database string) ([]string, error) {
	var out []string
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		sp, ok := drv.(dbdriver.SchemasProvider)
		if !ok {
			out = []string{}
			return nil
		}
		var err error
		out, err = sp.Schemas(ctx, database)
		return err
	})
	return out, err
}

// Tables 表/集合清单。
func (s *DBAdminService) Tables(ctx context.Context, instanceID uint, database, schema string) ([]dbdriver.TableInfo, error) {
	var out []dbdriver.TableInfo
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		switch drv.(type) {
		case dbdriver.MongoBrowser:
			// mongo：集合清单为单参签名
			lt, ok := drv.(interface {
				ListTables(ctx context.Context, database string) ([]dbdriver.TableInfo, error)
			})
			if !ok {
				return errs.Wrap(errs.ErrBadRequest, "该实例不支持集合浏览")
			}
			var err error
			out, err = lt.ListTables(ctx, database)
			return err
		case dbdriver.RedisBrowser:
			// Redis 无表概念（key 浏览在工作台），返回空清单而非报错
			out = []dbdriver.TableInfo{}
			return nil
		default:
			browser, ok := drv.(dbdriver.SQLBrowser)
			if !ok {
				return errs.Wrap(errs.ErrBadRequest, "该类型暂不支持表浏览")
			}
			var err error
			out, err = browser.ListTables(ctx, database, schema)
			return err
		}
	})
	return out, err
}

// Columns 字段结构。
func (s *DBAdminService) Columns(ctx context.Context, instanceID uint, database, schema, table string) ([]dbdriver.ColumnInfo, error) {
	var out []dbdriver.ColumnInfo
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		meta, ok := drv.(dbdriver.MetaExplorer)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持结构查看")
		}
		var err error
		out, err = meta.DescribeColumns(ctx, database, schema, table)
		return err
	})
	return out, err
}

// TableDDL 建表语句。
func (s *DBAdminService) TableDDL(ctx context.Context, instanceID uint, database, schema, table string) (string, error) {
	var out string
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		meta, ok := drv.(dbdriver.MetaExplorer)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持 DDL 查看")
		}
		var err error
		out, err = meta.TableDDL(ctx, database, schema, table)
		return err
	})
	return out, err
}

// Indexes SQL 索引清单。
func (s *DBAdminService) Indexes(ctx context.Context, instanceID uint, database, schema, table string) ([]dbdriver.IndexInfo, error) {
	var out []dbdriver.IndexInfo
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		meta, ok := drv.(dbdriver.MetaExplorer)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持索引查看")
		}
		var err error
		out, err = meta.ListIndexes(ctx, database, schema, table)
		return err
	})
	return out, err
}

// PrimaryKey 主键列。
func (s *DBAdminService) PrimaryKey(ctx context.Context, instanceID uint, database, schema, table string) ([]string, error) {
	var out []string
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		editor, ok := drv.(dbdriver.RowEditor)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持行编辑")
		}
		var err error
		out, err = editor.PrimaryKey(ctx, database, schema, table)
		return err
	})
	return out, err
}

// ---- 查询 ----

// RunQuery 自由只读 SQL。
func (s *DBAdminService) RunQuery(ctx context.Context, instanceID uint, database, sqlText string, limit int) (*dbdriver.QueryResult, error) {
	var out *dbdriver.QueryResult
	err := s.withDriver(ctx, instanceID, dbAdminQueryTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		browser, ok := drv.(dbdriver.SQLBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持 SQL 查询（Redis 请用命令控制台，Mongo 请用文档查询）")
		}
		var err error
		out, err = browser.RunQuery(ctx, database, sqlText, limit)
		return err
	})
	return out, err
}

// BrowsePage 表浏览分页。
func (s *DBAdminService) BrowsePage(ctx context.Context, instanceID uint, req dbdriver.BrowsePageReq) (*dbdriver.PagedResult, error) {
	var out *dbdriver.PagedResult
	err := s.withDriver(ctx, instanceID, dbAdminQueryTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		browser, ok := drv.(dbdriver.SQLBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持表浏览")
		}
		var err error
		out, err = browser.BrowsePage(ctx, req)
		return err
	})
	return out, err
}

// ---- SQL 库写路径（审计） ----

// InsertRow 插入行。
func (s *DBAdminService) InsertRow(ctx context.Context, instanceID uint, username string, req dbdriver.RowEdit) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		editor, ok := drv.(dbdriver.RowEditor)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持行编辑")
		}
		err := editor.InsertRow(ctx, req)
		target := req.DB + "." + req.Table
		detail := "INSERT " + keysSummary(req.Values)
		s.audit(username, inst, "row_insert", target, detail, err == nil)
		return err
	})
}

// UpdateRow 按主键更新。
func (s *DBAdminService) UpdateRow(ctx context.Context, instanceID uint, username string, req dbdriver.RowEdit) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		editor, ok := drv.(dbdriver.RowEditor)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持行编辑")
		}
		err := editor.UpdateRow(ctx, req)
		target := req.DB + "." + req.Table
		detail := "UPDATE " + pkSummary(req.PK) + " SET " + keysSummary(req.Values)
		s.audit(username, inst, "row_update", target, detail, err == nil)
		return err
	})
}

// DeleteRow 按主键删除。
func (s *DBAdminService) DeleteRow(ctx context.Context, instanceID uint, username string, req dbdriver.RowEdit) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		editor, ok := drv.(dbdriver.RowEditor)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持行编辑")
		}
		err := editor.DeleteRow(ctx, req)
		target := req.DB + "." + req.Table
		s.audit(username, inst, "row_delete", target, "DELETE "+pkSummary(req.PK), err == nil)
		return err
	})
}

// CreateIndex 创建索引。
func (s *DBAdminService) CreateIndex(ctx context.Context, instanceID uint, username string, req dbdriver.IndexRequest) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		meta, ok := drv.(dbdriver.MetaExplorer)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持索引管理")
		}
		err := meta.CreateIndex(ctx, req)
		target := req.DB + "." + req.Table
		detail := "CREATE INDEX " + req.Name + " (" + strings.Join(req.Columns, ",") + ") unique=" + boolText(req.Unique)
		s.audit(username, inst, "index_create", target, detail, err == nil)
		return err
	})
}

// DropIndex 删除索引。
func (s *DBAdminService) DropIndex(ctx context.Context, instanceID uint, username, database, schema, table, name string) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		meta, ok := drv.(dbdriver.MetaExplorer)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该类型不支持索引管理")
		}
		err := meta.DropIndex(ctx, database, schema, table, name)
		s.audit(username, inst, "index_drop", database+"."+table, "DROP INDEX "+name, err == nil)
		return err
	})
}

// ---- Redis 写路径（审计） ----

// ScanKeys key 分页扫描。
func (s *DBAdminService) ScanKeys(ctx context.Context, instanceID uint, database, pattern string, cursor uint64, count int64) ([]dbdriver.RedisKey, uint64, error) {
	var keys []dbdriver.RedisKey
	var next uint64
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		rb, ok := drv.(dbdriver.RedisBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Redis")
		}
		var err error
		keys, next, err = rb.ScanKeys(ctx, database, pattern, cursor, count)
		return err
	})
	return keys, next, err
}

// KeyDetail key 详情。
func (s *DBAdminService) KeyDetail(ctx context.Context, instanceID uint, database, name string) (*dbdriver.RedisKeyDetail, error) {
	var out *dbdriver.RedisKeyDetail
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		rb, ok := drv.(dbdriver.RedisBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Redis")
		}
		var err error
		out, err = rb.KeyDetail(ctx, database, name)
		return err
	})
	return out, err
}

// WriteRedisKey 写 key。
func (s *DBAdminService) WriteRedisKey(ctx context.Context, instanceID uint, username string, req dbdriver.RedisWrite) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		rb, ok := drv.(dbdriver.RedisBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Redis")
		}
		err := rb.WriteKey(ctx, req)
		s.audit(username, inst, "redis_write", req.DB+"/"+req.Name, "WRITE "+req.Type, err == nil)
		return err
	})
}

// DeleteRedisKey 删 key。
func (s *DBAdminService) DeleteRedisKey(ctx context.Context, instanceID uint, username, database, name string) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		rb, ok := drv.(dbdriver.RedisBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Redis")
		}
		err := rb.DeleteKey(ctx, database, name)
		s.audit(username, inst, "redis_delete", database+"/"+name, "DEL", err == nil)
		return err
	})
}

// SetRedisTTL 设置/清除过期。
func (s *DBAdminService) SetRedisTTL(ctx context.Context, instanceID uint, username, database, name string, ttlSecs int64) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		rb, ok := drv.(dbdriver.RedisBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Redis")
		}
		err := rb.SetTTL(ctx, database, name, ttlSecs)
		kind := "redis_ttl"
		detail := "EXPIRE"
		if ttlSecs < 0 {
			detail = "PERSIST"
		}
		s.audit(username, inst, kind, database+"/"+name, detail, err == nil)
		return err
	})
}

// RedisExec 命令控制台。
func (s *DBAdminService) RedisExec(ctx context.Context, instanceID uint, username, database, command string) (string, error) {
	var out string
	err := s.withDriver(ctx, instanceID, dbAdminConsoleBgTime, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		rb, ok := drv.(dbdriver.RedisBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Redis")
		}
		var err error
		out, err = rb.ExecCommand(ctx, database, command)
		s.audit(username, inst, "redis_exec", database, firstLine(command), err == nil)
		return err
	})
	return out, err
}

// ---- Mongo 写路径（审计） ----

// MongoFindDocs 文档分页查询。
func (s *DBAdminService) MongoFindDocs(ctx context.Context, instanceID uint, database, coll, filterJSON, projectJSON, sortField, sortDir string, skip, limit int) (*dbdriver.MongoDocPage, error) {
	var out *dbdriver.MongoDocPage
	err := s.withDriver(ctx, instanceID, dbAdminQueryTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(dbdriver.MongoBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		var err error
		out, err = mb.FindDocs(ctx, database, coll, filterJSON, projectJSON, sortField, sortDir, skip, limit)
		return err
	})
	return out, err
}

// MongoAggregate 聚合管道执行（只读，$out/$merge 已拦截）。
func (s *DBAdminService) MongoAggregate(ctx context.Context, instanceID uint, database, coll string, stages []string, maxDocs int) (*dbdriver.MongoDocPage, error) {
	var out *dbdriver.MongoDocPage
	err := s.withDriver(ctx, instanceID, dbAdminQueryTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(dbdriver.MongoBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		var err error
		out, err = mb.Aggregate(ctx, database, coll, stages, maxDocs)
		return err
	})
	return out, err
}

// MongoGetDoc 单文档。
func (s *DBAdminService) MongoGetDoc(ctx context.Context, instanceID uint, database, coll, idJSON string) (string, error) {
	var out string
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(dbdriver.MongoBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		var err error
		out, err = mb.GetDoc(ctx, database, coll, idJSON)
		return err
	})
	return out, err
}

// MongoInsertDoc 插入文档。
func (s *DBAdminService) MongoInsertDoc(ctx context.Context, instanceID uint, username, database, coll, docJSON string) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(dbdriver.MongoBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		err := mb.InsertDoc(ctx, database, coll, docJSON)
		s.audit(username, inst, "mongo_doc_insert", database+"."+coll, summaryOf(docJSON), err == nil)
		return err
	})
}

// MongoUpdateDoc 替换文档。
func (s *DBAdminService) MongoUpdateDoc(ctx context.Context, instanceID uint, username, database, coll, idJSON, docJSON string) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(dbdriver.MongoBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		err := mb.UpdateDoc(ctx, database, coll, idJSON, docJSON)
		s.audit(username, inst, "mongo_doc_update", database+"."+coll, summaryOf(docJSON), err == nil)
		return err
	})
}

// MongoDeleteDoc 删除文档。
func (s *DBAdminService) MongoDeleteDoc(ctx context.Context, instanceID uint, username, database, coll, idJSON string) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(dbdriver.MongoBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		err := mb.DeleteDoc(ctx, database, coll, idJSON)
		s.audit(username, inst, "mongo_doc_delete", database+"."+coll, "id="+idJSON, err == nil)
		return err
	})
}

// MongoIndexes mongo 索引清单。
func (s *DBAdminService) MongoIndexes(ctx context.Context, instanceID uint, database, coll string) ([]dbdriver.MongoIndex, error) {
	var out []dbdriver.MongoIndex
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(dbdriver.MongoBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		var err error
		out, err = mb.ListIndexes(ctx, database, coll)
		return err
	})
	return out, err
}

// MongoCreateIndex 创建索引。
func (s *DBAdminService) MongoCreateIndex(ctx context.Context, instanceID uint, username, database, coll, name string, keys map[string]int, unique bool) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(dbdriver.MongoBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		err := mb.CreateIndex(ctx, database, coll, name, keys, unique)
		s.audit(username, inst, "mongo_index_create", database+"."+coll, "CREATE INDEX "+name+" unique="+boolText(unique), err == nil)
		return err
	})
}

// MongoDropIndex 删除索引。
func (s *DBAdminService) MongoDropIndex(ctx context.Context, instanceID uint, username, database, coll, name string) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(dbdriver.MongoBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		err := mb.DropIndex(ctx, database, coll, name)
		s.audit(username, inst, "mongo_index_drop", database+"."+coll, "DROP INDEX "+name, err == nil)
		return err
	})
}

// MongoCollectionAction 集合管理动作（create/drop/rename）。
func (s *DBAdminService) MongoCollectionAction(ctx context.Context, instanceID uint, username, action, database, name, to string) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(dbdriver.MongoBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		var err error
		switch action {
		case "create":
			err = mb.CreateCollection(ctx, database, name)
		case "drop":
			err = mb.DropCollection(ctx, database, name)
		case "rename":
			if strings.TrimSpace(to) == "" {
				err = errs.Wrap(errs.ErrBadRequest, "缺少重命名目标")
			} else {
				err = mb.RenameCollection(ctx, database, name, to)
			}
		default:
			err = errs.Wrap(errs.ErrBadRequest, "不支持的动作: "+action)
		}
		detail := strings.ToUpper(action) + " " + name
		if to != "" {
			detail += " → " + to
		}
		s.audit(username, inst, "mongo_collection_"+action, database, detail, err == nil)
		return err
	})
}

// ---- 审计查询 ----

// Audits 审计记录分页。
func (s *DBAdminService) Audits(ctx context.Context, instanceID uint, page, size int) ([]model.DatabaseAuditLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	q := s.db.Model(&model.DatabaseAuditLog{})
	if instanceID > 0 {
		q = q.Where("instance_id = ?", instanceID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	out := []model.DatabaseAuditLog{}
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// ---- 小工具 ----

func boolText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// pkSummary 主键定位摘要（不含值原文，仅列名）。
func pkSummary(pk map[string]any) string {
	cols := make([]string, 0, len(pk))
	for c := range pk {
		cols = append(cols, c)
	}
	return "WHERE(" + strings.Join(cols, ",") + ")"
}

// keysSummary 列清单摘要。
func keysSummary(vals map[string]any) string {
	cols := make([]string, 0, len(vals))
	for c := range vals {
		cols = append(cols, c)
	}
	return "(" + strings.Join(cols, ",") + ")"
}

// summaryOf JSON 摘要（截断）。
func summaryOf(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
