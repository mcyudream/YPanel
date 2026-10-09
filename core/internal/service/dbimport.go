// 数据导入与实例迁移：SQL 文件执行（恢复通道）、Mongo Compass JSON 导入、
// 1Panel 备份文件导入（标准 dump 格式兼容）、同类型实例间按库全量迁移（任务化）。
// 安全：密码一律经 ExecReq.Env 注入 + 容器内 stdin/config 中转，宿主 argv 无明文；写路径全审计。
package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/dbdriver"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// importMaxContent 导入内容上限（SQL 文本 20MB / JSON 文本 10MB 在驱动侧再校验）。
const importMaxContent = 20 << 20

// migrateTimeout 单次迁移任务上限。
const migrateTimeout = 24 * time.Hour

// backupImportMax 备份文件导入上限（解码前）。
const backupImportMax = 100 << 20

// importFileBase 落盘文件名白名单。
var importFileBase = regexp.MustCompile(`^[A-Za-z0-9._-]{1,120}$`)

var backupExtByType = map[string][]string{
	"mysql":    {".sql", ".sql.gz"},
	"postgres": {".sql", ".sql.gz"},
	"redis":    {".rdb"},
	"mongo":    {".archive.gz", ".gz"},
}

// writeFileToHost 经 agent 把内容落盘到面板主机（encoding=base64 透传，避免二次编码）。
func (s *DatabaseService) writeFileToHost(ctx context.Context, filePath, contentB64 string) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: filePath, Content: contentB64, Encoding: "base64"})
	return err
}

// hostPathSafe 基本路径安全检查（导入来源为服务器已有文件）。
func hostPathSafe(p string) error {
	if !strings.HasPrefix(p, "/") || strings.Contains(p, "\x00") || strings.Contains(p, "..") {
		return errs.Wrap(errs.ErrBadRequest, "文件路径不合法（需为服务器绝对路径）")
	}
	return nil
}

// ---- SQL 文件导入执行（MySQL/PG，恢复通道） ----

// ImportSQL 导入 SQL 文件到指定库。content（≤20MB 文本）与 srcPath（服务器已有文件）二选一。
func (s *DBAdminService) ImportSQL(ctx context.Context, instanceID uint, username, database, content, srcPath string) (map[string]any, error) {
	inst, err := s.dbs.ByID(instanceID)
	if err != nil {
		return nil, err
	}
	if inst.Type != "mysql" && inst.Type != "postgres" {
		return nil, errs.Wrap(errs.ErrBadRequest, "SQL 导入仅支持 MySQL/PostgreSQL")
	}
	if err := dbdriver.ValidateIdent(database); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "目标库名不合法")
	}
	if content == "" && srcPath == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "请提供导入文件或服务器路径")
	}
	if len(content) > importMaxContent {
		return nil, errs.Wrap(errs.ErrBadRequest, "导入文件超过 20MB 上限")
	}

	file := srcPath
	if content != "" {
		file = path.Join(s.dbs.BackupDir(inst), fmt.Sprintf("import-%s-%d.sql", inst.Name, time.Now().Unix()))
		if err := s.dbs.writeFileToHost(ctx, file, base64.StdEncoding.EncodeToString([]byte(content))); err != nil {
			return nil, errs.Wrap(errs.ErrBadRequest, "落盘失败: "+err.Error())
		}
	} else if err := hostPathSafe(srcPath); err != nil {
		return nil, err
	}

	pwd, err := s.dbs.decryptPassword(inst.PasswordEnc)
	if err != nil {
		return nil, err
	}
	c := inst.ComposeProject
	host := inst.Host
	if host == "" {
		host = "127.0.0.1"
	}
	// stdin 中转密码 + SQL 流（与恢复通道同款两层管道）
	var cmd string
	switch inst.Type {
	case "mysql":
		if inst.Origin == "external" {
			cmd = fmt.Sprintf("sh -c 'command -v mysql >/dev/null || { echo \"本机缺少 mysql 客户端\"; exit 127; }; MYSQL_PWD=\"$YP_DB_PWD\" mysql -h %s -P %d -u %s %s' < %s", host, inst.Port, inst.RootUser, database, file)
		} else {
			cmd = fmt.Sprintf("{ printf '%%s\\n' \"$YP_DB_PWD\"; cat %s; } | docker exec -i %s sh -c 'read -r pw; MYSQL_PWD=\"$pw\" mysql %s'", file, c, database)
		}
	case "postgres":
		if inst.Origin == "external" {
			cmd = fmt.Sprintf("sh -c 'command -v psql >/dev/null || { echo \"本机缺少 psql 客户端\"; exit 127; }; PGPASSWORD=\"$YP_DB_PWD\" psql -q -h %s -p %d -U %s -d %s' < %s", host, inst.Port, inst.RootUser, database, file)
		} else {
			cmd = fmt.Sprintf("docker exec -i %s psql -q -U postgres -d %s < %s", c, database, file)
		}
	}
	out, err := s.dbs.ExecAgent(ctx, cmd, map[string]string{"YP_DB_PWD": pwd}, 3600)
	if err != nil {
		return nil, err
	}
	success := out.ExitCode == 0 && !out.TimedOut
	s.audit(username, inst, "sql_import", database, fmt.Sprintf("file=%s bytes=%d", file, len(content)), success)
	if !success {
		return map[string]any{"output": out.Output}, errs.Wrapc(errs.CodeFileOpFailed, "导入执行失败: "+firstLine(out.Output))
	}
	return map[string]any{"file": file, "output": tailOf(out.Output, 2000)}, nil
}

func tailOf(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}

// ---- Mongo Compass JSON 导入 ----

// MongoImportDocs 导入 Compass 导出的 JSON/NDJSON 到集合。
func (s *DBAdminService) MongoImportDocs(ctx context.Context, instanceID uint, username, database, coll, content, format string) (map[string]any, error) {
	var out *dbdriver.MongoDocPage
	n := 0
	err := s.withDriver(ctx, instanceID, dbAdminQueryTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		mb, ok := drv.(interface {
			ImportDocs(ctx context.Context, database, coll, content, format string) (int, error)
		})
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 MongoDB")
		}
		var err error
		n, err = mb.ImportDocs(ctx, database, coll, content, format)
		return err
	})
	_ = out
	if err != nil {
		return nil, err
	}
	inst, err := s.dbs.ByID(instanceID)
	if err == nil {
		s.audit(username, inst, "mongo_import", database+"."+coll, fmt.Sprintf("format=%s count=%d", format, n), true)
	}
	return map[string]any{"imported": n}, nil
}

// ---- 1Panel 备份文件导入（标准 dump 格式，落到实例备份目录后走恢复） ----

// ImportBackup 导入外部备份文件（1Panel 等标准 dump 格式）到实例备份目录。
func (s *DatabaseService) ImportBackup(ctx context.Context, id uint, username, filename, contentB64 string) (map[string]any, error) {
	inst, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	if !importFileBase.MatchString(filename) {
		return nil, errs.Wrap(errs.ErrBadRequest, "文件名不合法（仅字母数字._-）")
	}
	extOK := false
	lower := strings.ToLower(filename)
	for _, ext := range backupExtByType[inst.Type] {
		if strings.HasSuffix(lower, ext) {
			extOK = true
			break
		}
	}
	if !extOK {
		return nil, errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("文件类型与实例不匹配（%s 实例要求 %s）", inst.Type, strings.Join(backupExtByType[inst.Type], " / ")))
	}
	raw, err := base64.StdEncoding.DecodeString(contentB64)
	if err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "文件内容解码失败")
	}
	if len(raw) > backupImportMax {
		return nil, errs.Wrap(errs.ErrBadRequest, "文件超过 100MB 上限")
	}
	target := path.Join(s.BackupDir(inst), filename)
	if err := s.writeFileToHost(ctx, target, contentB64); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "落盘失败: "+err.Error())
	}
	s.auditUser(username, inst, "backup_import", inst.Name, fmt.Sprintf("file=%s bytes=%d", filename, len(raw)), true)
	return map[string]any{"file": filename, "path": target, "bytes": len(raw)}, nil
}

// auditUser 带用户名的迁移审计（DatabaseService 侧，避免循环依赖 DBAdminService）。
func (s *DatabaseService) auditUser(username string, inst *model.DatabaseInstance, kind, target, detail string, success bool) {
	entry := model.DatabaseAuditLog{
		Username: username, InstanceID: inst.ID, InstanceName: inst.Name,
		DbType: inst.Type, Kind: kind, Target: target, Detail: detail, Success: success,
	}
	_ = s.db.Create(&entry).Error
}

// ---- DBAdminService 对迁移能力的委托（API 面向插件入口） ----

// MigratePreview 迁移预检。
func (s *DBAdminService) MigratePreview(ctx context.Context, srcID, targetID uint) (*MigratePreview, error) {
	return s.dbs.MigratePreview(ctx, srcID, targetID)
}

// MigrateStart 启动迁移任务。
func (s *DBAdminService) MigrateStart(ctx context.Context, username string, srcID, targetID uint, dbs []string) (*model.AppTask, error) {
	return s.dbs.MigrateStart(ctx, username, srcID, targetID, dbs)
}
