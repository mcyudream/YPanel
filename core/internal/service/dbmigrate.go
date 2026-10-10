// 同类型数据库实例间按库全量迁移（结构+数据）：MySQL/PG/Mongo 走 dump 落盘中转，Redis 走 DUMP/RESTORE 逐键。
// 任务化长跑（任务中心可见进度与日志）；覆盖目标同名库需前端确认；密码全程 env/config 传递不进宿主 argv。
package service

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/dbdriver"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// MigratePreview 迁移预检：源库清单 + 兼容性。
type MigratePreview struct {
	SrcType    string               `json:"srcType"`
	TargetType string               `json:"targetType"`
	TargetName string               `json:"targetName"`
	Compatible bool                 `json:"compatible"`
	Dbs        []dbdriver.DatabaseInfo `json:"dbs"`
}

// MigratePreview 迁移预检（源可迁移库列表）。
func (s *DatabaseService) MigratePreview(ctx context.Context, srcID, targetID uint) (*MigratePreview, error) {
	src, err := s.ByID(srcID)
	if err != nil {
		return nil, err
	}
	dst, err := s.ByID(targetID)
	if err != nil {
		return nil, err
	}
	out := &MigratePreview{SrcType: src.Type, TargetType: dst.Type, TargetName: dst.Name}
	if src.Type != dst.Type {
		return out, nil
	}
	out.Compatible = true
	dbs, err := s.Databases(ctx, srcID)
	if err != nil {
		return nil, err
	}
	out.Dbs = dbs
	return out, nil
}

// MigrateStart 启动迁移任务（同类型；dbs 为源库/dbN 列表）。
func (s *DatabaseService) MigrateStart(ctx context.Context, username string, srcID, targetID uint, dbs []string) (*model.AppTask, error) {
	src, err := s.ByID(srcID)
	if err != nil {
		return nil, err
	}
	dst, err := s.ByID(targetID)
	if err != nil {
		return nil, err
	}
	if src.Type != dst.Type {
		return nil, errs.Wrap(errs.ErrBadRequest, "仅支持同类型实例间迁移")
	}
	if src.ID == dst.ID {
		return nil, errs.Wrap(errs.ErrBadRequest, "源与目标不能是同一实例")
	}
	if len(dbs) == 0 {
		return nil, errs.Wrap(errs.ErrBadRequest, "请选择要迁移的库")
	}
	for _, d := range dbs {
		if strings.TrimSpace(d) == "" {
			return nil, errs.Wrap(errs.ErrBadRequest, "库名不合法")
		}
	}
	tasks := s.tasks
	if tasks == nil {
		return nil, errs.Wrap(errs.ErrInternal, "任务服务未就绪")
	}
	title := fmt.Sprintf("迁移 %s(%s) → %s：%d 个库", src.Name, src.Type, dst.Name, len(dbs))
	return tasks.StartTask("db_migrate", title, fmt.Sprintf("%d->%d", srcID, targetID), migrateTimeout,
		func(ctx context.Context, logf TaskLogf) error {
			return s.migrateRun(ctx, logf, username, src, dst, dbs)
		})
}

// migrateRun 迁移执行体（每库：dump 落盘 → 目标导入；redis 逐键）。
func (s *DatabaseService) migrateRun(ctx context.Context, logf TaskLogf, username string, src, dst *model.DatabaseInstance, dbs []string) error {
	// 建源/目标实例备份目录（dump 中转落盘用；shell 重定向不会自动建目录，幂等）
	ac, err := s.client()
	if err != nil {
		return err
	}
	for _, inst := range []*model.DatabaseInstance{src, dst} {
		if _, err := agentclient.DoJSON[dto.FileMkdirReq, struct{}](ac, ctx, "POST", "/agent/v1/files/mkdir",
			&dto.FileMkdirReq{Path: s.BackupDir(inst)}); err != nil {
			return errs.Wrap(errs.ErrBadRequest, "mkdir: "+err.Error())
		}
	}
	srcPwd, err := s.decryptPassword(src.PasswordEnc)
	if err != nil {
		return err
	}
	dstPwd, err := s.decryptPassword(dst.PasswordEnc)
	if err != nil {
		return err
	}
	// 目标驱动（PG 建库/驱动级操作用）
	dstDrv, err := s.driverFor(dst)
	if err != nil {
		return err
	}
	defer dstDrv.Close()

	for i, db := range dbs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logf("info", "[%d/%d] 开始迁移 %s", i+1, len(dbs), db)
		var err error
		switch src.Type {
		case "redis":
			err = s.migrateRedisDB(ctx, logf, src, dst, srcPwd, dstPwd, db)
		case "mongo":
			err = s.migrateMongoDB(ctx, logf, src, dst, srcPwd, dstPwd, db)
		case "mysql":
			err = s.migrateSQLDB(ctx, logf, src, dst, srcPwd, dstPwd, db, dstDrv, true)
		case "postgres":
			err = s.migrateSQLDB(ctx, logf, src, dst, srcPwd, dstPwd, db, dstDrv, false)
		default:
			err = errs.Wrap(errs.ErrBadRequest, "不支持的类型: "+src.Type)
		}
		if err != nil {
			s.auditUser(username, dst, "migrate", db, fmt.Sprintf("src=%s fail: %s", src.Name, firstLine(err.Error())), false)
			return fmt.Errorf("迁移 %s 失败: %w", db, err)
		}
		s.auditUser(username, dst, "migrate", db, fmt.Sprintf("src=%s", src.Name), true)
		logf("info", "[%d/%d] %s 完成", i+1, len(dbs), db)
	}
	return nil
}

// dbDumpFile 迁移中转 dump 文件路径。
func (s *DatabaseService) dbDumpFile(inst *model.DatabaseInstance, db string) string {
	return path.Join(s.BackupDir(inst), fmt.Sprintf("migrate-%s-%d.dump", db, time.Now().Unix()))
}

// srcDumpShell 源库 dump 到宿主文件（落盘中转；密码经 env + stdin/config 传递）。
func (s *DatabaseService) srcDumpShell(ctx context.Context, logf TaskLogf, src *model.DatabaseInstance, pwd, db, file string) error {
	c := src.ComposeProject
	host := src.Host
	if host == "" {
		host = "127.0.0.1"
	}
	var cmd string
	switch src.Type {
	case "mysql":
		if src.Origin == "external" {
			cmd = fmt.Sprintf("sh -c 'command -v mysqldump >/dev/null || { echo \"本机缺少 mysqldump\"; exit 127; }; cf=$(mktemp); printf \"[client]\\npassword=%%s\\n\" \"$YP_DB_PWD\" > \"$cf\"; mysqldump --defaults-extra-file=\"$cf\" -h %s -P %d -u %s --databases %s --single-transaction; rc=$?; rm -f \"$cf\"; exit $rc' > %s", host, src.Port, src.RootUser, db, file)
		} else {
			cmd = fmt.Sprintf("printf '%%s\\n' \"$YP_DB_PWD\" | docker exec -i %s sh -c 'read -r pw; cf=$(mktemp); printf \"[client]\\npassword=%%s\\n\" \"$pw\" > \"$cf\"; MYSQL_PWD=\"$pw\" mysqldump --defaults-extra-file=\"$cf\" --databases %s --single-transaction; rc=$?; rm -f \"$cf\"; exit $rc' > %s", c, db, file)
		}
	case "postgres":
		if src.Origin == "external" {
			cmd = fmt.Sprintf("sh -c 'command -v pg_dump >/dev/null || { echo \"本机缺少 pg_dump\"; exit 127; }; PGPASSWORD=\"$YP_DB_PWD\" pg_dump -h %s -p %d -U %s -d %s' > %s", host, src.Port, src.RootUser, db, file)
		} else {
			cmd = fmt.Sprintf("docker exec %s pg_dump -U postgres -d %s > %s", c, db, file)
		}
	case "mongo":
		archive := ""
		if !strings.HasSuffix(file, ".archive") {
			file = file + ".archive.gz"
		}
		if src.Origin == "external" {
			cmd = fmt.Sprintf("sh -c 'command -v mongodump >/dev/null || { echo \"本机缺少 mongodump\"; exit 127; }; cf=$(mktemp); printf \"password: %%s\\n\" \"$YP_DB_PWD\" > \"$cf\"; mongodump --archive --gzip --db %s -u %s --config \"$cf\" --authenticationDatabase admin; rc=$?; rm -f \"$cf\"; exit $rc' > %s", db, src.RootUser, file)
		} else {
			cmd = fmt.Sprintf("printf '%%s\\n' \"$YP_DB_PWD\" | docker exec -i %s sh -c 'read -r pw; cf=$(mktemp); printf \"password: %%s\\n\" \"$pw\" > \"$cf\"; mongodump --archive --gzip --db %s -u %s --config \"$cf\" --authenticationDatabase admin; rc=$?; rm -f \"$cf\"; exit $rc' > %s", c, db, src.RootUser, file)
		}
		_ = archive
	default:
		return errs.Wrap(errs.ErrBadRequest, "类型不支持 dump: "+src.Type)
	}
	out, err := s.ExecAgent(ctx, cmd, map[string]string{"YP_DB_PWD": pwd}, 7200)
	if err != nil {
		return err
	}
	if out.ExitCode != 0 || out.TimedOut {
		return errs.Wrapc(errs.CodeFileOpFailed, "dump 失败: "+firstLine(out.Output))
	}
	logf("info", "  dump 完成 → %s", file)
	return nil
}

// migrateSQLDB MySQL/PG：dump 落盘 → 目标重建库 → 导入。
func (s *DatabaseService) migrateSQLDB(ctx context.Context, logf TaskLogf, src, dst *model.DatabaseInstance, srcPwd, dstPwd string, db string, dstDrv dbdriver.Driver, isMySQL bool) error {
	file := s.dbDumpFile(src, db)
	if !isMySQL {
		file += ".sql"
	} else {
		file += ".sql"
	}
	if err := s.srcDumpShell(ctx, logf, src, srcPwd, db, file); err != nil {
		return err
	}
	// 目标重建库（覆盖语义，前端已确认）
	if !isMySQL {
		_ = dstDrv.DropDatabase(ctx, db)
		if err := dstDrv.CreateDatabase(ctx, db, ""); err != nil {
			// 可能已存在（IF NOT EXISTS 语义因驱动而异），继续尝试导入
			logf("warn", "  目标建库返回: %s", err.Error())
		}
	}
	c := dst.ComposeProject
	host := dst.Host
	if host == "" {
		host = "127.0.0.1"
	}
	var cmd string
	switch {
	case isMySQL && dst.Origin == "container":
		cmd = fmt.Sprintf("{ printf '%%s\\n' \"$YP_DB_PWD\"; cat %s; } | docker exec -i %s sh -c 'read -r pw; MYSQL_PWD=\"$pw\" mysql'", file, c)
	case isMySQL:
		cmd = fmt.Sprintf("sh -c 'command -v mysql >/dev/null || { echo \"本机缺少 mysql 客户端\"; exit 127; }; cf=$(mktemp); printf \"[client]\\npassword=%%s\\n\" \"$YP_DB_PWD\" > \"$cf\"; mysql --defaults-extra-file=\"$cf\" -h %s -P %d -u %s < %s; rc=$?; rm -f \"$cf\"; exit $rc'", host, dst.Port, dst.RootUser, file)
	case !isMySQL && dst.Origin == "container":
		cmd = fmt.Sprintf("docker exec -i %s psql -q -U postgres -d %s < %s", c, db, file)
	default:
		cmd = fmt.Sprintf("sh -c 'command -v psql >/dev/null || { echo \"本机缺少 psql 客户端\"; exit 127; }; PGPASSWORD=\"$YP_DB_PWD\" psql -q -h %s -p %d -U %s -d %s' < %s", host, dst.Port, dst.RootUser, db, file)
	}
	out, err := s.ExecAgent(ctx, cmd, map[string]string{"YP_DB_PWD": dstPwd}, 14400)
	if err != nil {
		return err
	}
	if out.ExitCode != 0 || out.TimedOut {
		return errs.Wrapc(errs.CodeFileOpFailed, "目标导入失败: "+firstLine(out.Output))
	}
	return nil
}

// migrateMongoDB Mongo：dump archive 落盘 → 目标 mongorestore --drop。
func (s *DatabaseService) migrateMongoDB(ctx context.Context, logf TaskLogf, src, dst *model.DatabaseInstance, srcPwd, dstPwd string, db string) error {
	file := s.dbDumpFile(src, db) + ".archive.gz"
	if err := s.srcDumpShell(ctx, logf, src, srcPwd, db, file); err != nil {
		return err
	}
	c := dst.ComposeProject
	host := dst.Host
	if host == "" {
		host = "127.0.0.1"
	}
	var cmd string
	if dst.Origin == "container" {
		cmd = fmt.Sprintf("{ printf '%%s\\n' \"$YP_DB_PWD\"; cat %s; } | docker exec -i %s sh -c 'read -r pw; cf=$(mktemp); printf \"password: %%s\\n\" \"$pw\" > \"$cf\"; mongorestore --archive --gzip --drop -u %s --config \"$cf\" --authenticationDatabase admin; rc=$?; rm -f \"$cf\"; exit $rc'", file, c, dst.RootUser)
	} else {
		cmd = fmt.Sprintf("sh -c 'command -v mongorestore >/dev/null || { echo \"本机缺少 mongorestore\"; exit 127; }; cf=$(mktemp); printf \"password: %%s\\n\" \"$YP_DB_PWD\" > \"$cf\"; mongorestore --archive --gzip --drop -h %s --port %d -u %s --config \"$cf\" --authenticationDatabase admin; rc=$?; rm -f \"$cf\"; exit $rc' < %s", host, dst.Port, dst.RootUser, file)
	}
	out, err := s.ExecAgent(ctx, cmd, map[string]string{"YP_DB_PWD": dstPwd}, 14400)
	if err != nil {
		return err
	}
	if out.ExitCode != 0 || out.TimedOut {
		return errs.Wrapc(errs.CodeFileOpFailed, "目标恢复失败: "+firstLine(out.Output))
	}
	return nil
}

// migrateRedisDB Redis：逐键 DUMP/RESTORE（覆盖目标库）。
func (s *DatabaseService) migrateRedisDB(ctx context.Context, logf TaskLogf, src, dst *model.DatabaseInstance, srcPwd, dstPwd string, db string) error {
	srcDrv, err := s.driverFor(src)
	if err != nil {
		return err
	}
	defer srcDrv.Close()
	dstDrv, err := s.driverFor(dst)
	if err != nil {
		return err
	}
	defer dstDrv.Close()
	se, ok1 := srcDrv.(dbdriver.RedisBrowser)
	de, ok2 := dstDrv.(dbdriver.RedisBrowser)
	if !ok1 || !ok2 {
		return errs.Wrap(errs.ErrBadRequest, "Redis 驱动不支持迁移")
	}
	// 目标库覆盖：清空后导入
	if err := de.FlushDatabase(ctx, db); err != nil {
		logf("warn", "  目标库清空返回: %s", err.Error())
	}
	total := 0
	var cursor uint64
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		items, next, err := se.ExportDumps(ctx, db, cursor, 500)
		if err != nil {
			return err
		}
		if len(items) > 0 {
			n, err := de.ImportKeys(ctx, db, items)
			if err != nil {
				return err
			}
			total += n
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	logf("info", "  迁移 %d 个 key", total)
	return nil
}
