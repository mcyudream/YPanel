// Package db SQLite 初始化与迁移。
package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
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
	// 应用商店 key 历史唯一索引让位于 (source_id,key) 复合唯一
	if gdb.Migrator().HasIndex(&model.AppStoreApp{}, "idx_app_store_apps_key") {
		if err := gdb.Migrator().DropIndex(&model.AppStoreApp{}, "idx_app_store_apps_key"); err != nil {
			return nil, fmt.Errorf("迁移应用索引失败: %w", err)
		}
	}
	if err := gdb.AutoMigrate(&model.User{}, &model.Role{}, &model.RolePermission{}, &model.RoleNode{}, &model.LoginLog{}, &model.Setting{}, &model.CronTask{}, &model.CronTaskLog{}, &model.DatabaseInstance{}, &model.Site{}, &model.SiteGroup{}, &model.SitePortLease{}, &model.Certificate{}, &model.DnsAccount{}, &model.AcmeAccount{}, &model.Node{}, &model.PairingCode{}, &model.AlertRule{}, &model.Notification{}, &model.AuditLog{}, &model.MetricRecord{}, &model.AppStoreSource{}, &model.AppStoreApp{}, &model.AppStoreInstall{}, &model.Runtime{}, &model.Script{}, &model.AIProvider{}, &model.AIKnowledge{}, &model.AIKnowledgeDoc{}, &model.AIKnowledgeChunk{}, &model.AIMemory{}, &model.AIConversation{}, &model.AIToolFlag{}, &model.AIOperationLog{}, &model.ConfigRevision{}, &model.NatForwardRule{}, &model.AppTask{}, &model.GitCredential{}, &model.DnsRecord{}, &model.HostRecord{}, &model.HostTarget{}, &model.DatabaseAuditLog{}, &model.MonitorProbe{}, &model.StorageAccount{}, &model.DockerEnvironment{}, &model.LogSearchQuery{}, &model.MetricHourly{}, &model.FileFavorite{}, &model.FileShare{}, &model.MCPOperation{}, &model.DiskGuardEvent{}); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	if err := seedRBAC(gdb); err != nil {
		return nil, fmt.Errorf("角色初始化失败: %w", err)
	}
	if err := seedStore(gdb); err != nil {
		return nil, fmt.Errorf("应用源初始化失败: %w", err)
	}
	return gdb, nil
}

// ypMainURL YPanel 官方应用源仓库（Gitee 与 GitHub 双平台镜像同步更新，默认走 Gitee）。
const ypMainURL = "https://gitee.com/mcyudream/YPanel-AppStore.git"

// seedRBAC 内置角色同步 + 存量用户 role_id 回填（幂等，M54）。
func seedRBAC(gdb *gorm.DB) error {
	return rbac.New(gdb).Seed()
}

// seedStore 内置源注册与旧数据归属迁移（幂等）。
func seedStore(gdb *gorm.DB) error {
	var onePanelSrc model.AppStoreSource
	if err := gdb.Where("name = ?", "1Panel 官方源").First(&onePanelSrc).Error; err != nil {
		onePanelSrc = model.AppStoreSource{
			Name: "1Panel 官方源", Type: "onepanel", URL: "https://apps-assets.fit2cloud.com/dev/1panel.json.zip",
			Enabled: true, Builtin: true, Remark: "1Panel 应用社区官方源（只读）",
		}
		if err := gdb.Create(&onePanelSrc).Error; err != nil {
			return err
		}
	}
	var ypSrc model.AppStoreSource
	if err := gdb.Where("name = ?", "YPanel 官方源").First(&ypSrc).Error; err != nil {
		ypSrc = model.AppStoreSource{
			Name: "YPanel 官方源", Type: "yp-git", URL: ypMainURL, Branch: "main",
			Enabled: true, Builtin: true, Remark: "YPanel 自有源（git 仓库，yp 格式；环境服务与可复用中间件）",
		}
		if err := gdb.Create(&ypSrc).Error; err != nil {
			return err
		}
	} else if ypSrc.URL == "" {
		// 早期版本内置源未带仓库地址，补默认官方仓库
		_ = gdb.Model(&ypSrc).Update("url", ypMainURL).Error
	}
	// 历史 source_id=0 的应用/安装记录归属 1Panel 官方源
	now := time.Now()
	_ = gdb.Model(&model.AppStoreApp{}).Where("source_id = 0").Update("source_id", onePanelSrc.ID).Error
	_ = gdb.Model(&model.AppStoreInstall{}).Where("source_id = 0").Update("source_id", onePanelSrc.ID).Error
	_ = gdb.Model(&model.AppStoreSource{}).Where("id = ? AND status = 'pending'", onePanelSrc.ID).
		Updates(map[string]any{"status": "ok", "last_sync_at": &now}).Error
	return nil
}
