// Package db SQLite 初始化与迁移。
package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ypanel/core/internal/model"
)

// Open 打开（必要时创建）数据库并执行迁移。
func Open(dataDir string) (*gorm.DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	dsn := "file:" + filepath.ToSlash(filepath.Join(dataDir, "ypanel.db")) + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
	}
	if err := gdb.AutoMigrate(&model.User{}, &model.LoginLog{}, &model.Setting{}, &model.CronTask{}, &model.CronTaskLog{}, &model.DatabaseInstance{}, &model.Site{}, &model.Node{}, &model.PairingCode{}, &model.AlertRule{}, &model.Notification{}, &model.AuditLog{}, &model.MetricRecord{}, &model.AppStoreApp{}, &model.AppStoreInstall{}, &model.Runtime{}, &model.Script{}, &model.AIProvider{}); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	return gdb, nil
}
