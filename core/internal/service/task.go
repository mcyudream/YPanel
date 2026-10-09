// TaskService 统一任务中心：耗时操作（商店安装/卸载、镜像拉取等）异步执行并记录日志，
// 前端任务中心页与调用方（安装向导）均可按任务 ID 轮询日志与状态。
//
// 日志行统一格式（YdLogViewer 解析）：
//
//	2006-01-02 15:04:05 [INFO] 消息
//	2006-01-02 15:04:05 [ERROR] 消息
package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// 任务状态 / 类型常量。
const (
	TaskRunning = "running"
	TaskSuccess = "success"
	TaskFailed  = "failed"

	TaskStoreInstall   = "store-install"
	TaskStoreUninstall = "store-uninstall"
	TaskImagePull      = "image-pull"
	TaskDockerInstall  = "docker-install"
)

// TaskService 任务服务。
type TaskService struct {
	db *gorm.DB
	mu sync.Mutex // 日志追加串行化（SQLite 单写）
}

// NewTaskService 创建。
func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{db: db}
}

// TaskLogf 任务日志写入器（并发安全，追加到任务的 log_text）。
type TaskLogf func(level, format string, args ...any)

// StartTask 创建任务并异步执行（ctx 独立于请求：协程用 Background + 超时由任务自身控制）。
// 返回任务记录（running）。run 的返回值决定 success/failed。
func (s *TaskService) StartTask(typ, title, ref string, timeout time.Duration, run func(ctx context.Context, logf TaskLogf) error) (*model.AppTask, error) {
	row := &model.AppTask{Type: typ, Title: title, Ref: ref, Status: TaskRunning}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	taskID := row.ID
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		_ = cancel // 任务生命周期 = 进程生命周期，超时由 ctx 内部 deadline 控制
	}
	go func() {
		logf := s.logfFor(taskID)
		err := run(ctx, logf)
		updates := map[string]any{}
		if err != nil {
			updates["status"] = TaskFailed
			updates["error"] = truncStr(err.Error(), 2000)
			_ = s.appendRaw(taskID, "\n"+taskTime()+" [ERROR] "+truncStr(err.Error(), 500)+"\n")
		} else {
			updates["status"] = TaskSuccess
		}
		_ = s.db.Model(&model.AppTask{}).Where("id = ?", taskID).Updates(updates).Error
	}()
	return row, nil
}

func (s *TaskService) logfFor(taskID uint) TaskLogf {
	return func(level, format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		_ = s.appendRaw(taskID, taskTime()+" ["+strings.ToUpper(level)+"] "+strings.TrimRight(msg, "\n")+"\n")
	}
}

func taskTime() string { return time.Now().Format("2006-01-02 15:04:05") }

// appendRaw 追加原始日志（|| 拼接避免读改写竞态）。
func (s *TaskService) appendRaw(taskID uint, chunk string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Model(&model.AppTask{}).Where("id = ?", taskID).
		Update("log_text", gorm.Expr("COALESCE(log_text, '') || ?", chunk)).Error
}

// List 任务分页列表（不含日志正文）。
func (s *TaskService) List(typ, status string, page, pageSize int) (map[string]any, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	q := s.db.Model(&model.AppTask{}).Omit("log_text", "error")
	if typ != "" {
		q = q.Where("type = ?", typ)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	rows := []model.AppTask{}
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	return map[string]any{"total": total, "page": page, "pageSize": pageSize, "items": rows}, nil
}

// Get 任务详情（含日志）。
func (s *TaskService) Get(id uint) (*model.AppTask, error) {
	var row model.AppTask
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.taskNotFound", "任务不存在")
	}
	return &row, nil
}

// Delete 删除任务记录（运行中拒绝）。
func (s *TaskService) Delete(id uint) error {
	var row model.AppTask
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.New(errs.CodeNotFound, "error.taskNotFound", "任务不存在")
	}
	if row.Status == TaskRunning {
		return errs.Wrap(errs.ErrBadRequest, "任务运行中，不可删除")
	}
	return s.db.Delete(&model.AppTask{}, id).Error
}

// Clear 清理已结束任务。
func (s *TaskService) Clear() (int64, error) {
	res := s.db.Where("status <> ?", TaskRunning).Delete(&model.AppTask{})
	return res.RowsAffected, res.Error
}

// RunningCount 运行中任务数（供概览等处展示）。
func (s *TaskService) RunningCount() int64 {
	var n int64
	_ = s.db.Model(&model.AppTask{}).Where("status = ?", TaskRunning).Count(&n).Error
	return n
}
