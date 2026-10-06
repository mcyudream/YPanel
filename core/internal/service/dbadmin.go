// DBAdminService 数据库管理台（插件一期：进程内模块，SDK 风格边界）。
// 职责：基于 M4 数据库实例提供库/表/集合浏览与只读查询；写操作走 M4 结构化接口。
package service

import (
	"context"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/dbdriver"
	"github.com/ypanel/shared/errs"
)

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

func (s *DBAdminService) driverFor(ctx context.Context, instanceID uint) (dbdriver.Driver, dbdriver.TableBrowser, error) {
	inst, err := s.dbs.ByID(instanceID)
	if err != nil {
		return nil, nil, err
	}
	drv, err := s.dbs.driverFor(inst)
	if err != nil {
		return nil, nil, err
	}
	browser, ok := drv.(dbdriver.TableBrowser)
	if !ok {
		drv.Close()
		return nil, nil, errs.Wrap(errs.ErrAgentDisabled, "该类型暂不支持浏览")
	}
	return drv, browser, nil
}

// Tables 表/集合清单。
func (s *DBAdminService) Tables(ctx context.Context, instanceID uint, database string) ([]dbdriver.TableInfo, error) {
	drv, browser, err := s.driverFor(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	defer drv.Close()
	ctx, cancel := context.WithTimeout(ctx, 15_000_000_000)
	defer cancel()
	return browser.ListTables(ctx, database)
}

// Query 只读查询。
func (s *DBAdminService) Query(ctx context.Context, instanceID uint, database, sqlText string) (*dbdriver.QueryResult, error) {
	drv, browser, err := s.driverFor(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	defer drv.Close()
	ctx, cancel := context.WithTimeout(ctx, 30_000_000_000)
	defer cancel()
	if err := drv.Ping(ctx); err != nil {
		return nil, errs.Wrap(errs.ErrAgentUnreach, "连接实例失败: "+err.Error())
	}
	return browser.Query(ctx, database, sqlText, 200)
}

// Databases 库列表（复用 M4）。
func (s *DBAdminService) Databases(ctx context.Context, instanceID uint) ([]dbdriver.DatabaseInfo, error) {
	return s.dbs.Databases(ctx, instanceID)
}
