package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/wsbus"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// Cron 计划任务调度器：core 侧调度，执行经 agent 受控 exec。
type Cron struct {
	db     *gorm.DB
	nodes  *NodeService
	SiteBk *SiteBackupService
	DBSvc  *DatabaseService

	mu      sync.Mutex
	inner   *cron.Cron
	entries map[uint]cron.EntryID
	running sync.Map // taskID → struct{}（防同任务并发）
}

// logOutputCap 执行记录输出上限（64KB，尾部截取）。
const logOutputCap = 64 * 1024

// NewCron 创建调度器。
func NewCron(db *gorm.DB, nodes *NodeService) *Cron {
	return &Cron{db: db, nodes: nodes, entries: map[uint]cron.EntryID{}}
}

// Start 装载全部启用任务并启动调度。
func (c *Cron) Start() error {
	c.inner = cron.New(cron.WithChain(cron.Recover(cron.DefaultLogger), cron.SkipIfStillRunning(cron.DefaultLogger)))
	if err := c.loadAll(); err != nil {
		return err
	}
	c.inner.Start()
	slog.Info("cron 调度器已启动")
	return nil
}

// Stop 停止调度（等待在跑任务收尾由 ctx/超时兜底）。
func (c *Cron) Stop() {
	if c.inner != nil {
		c.inner.Stop()
	}
}

// loadAll 清空并重载全部启用任务。
func (c *Cron) loadAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, entry := range c.entries {
		c.inner.Remove(entry)
		delete(c.entries, id)
	}
	var tasks []model.CronTask
	if err := c.db.Where("enabled = ?", true).Find(&tasks).Error; err != nil {
		return err
	}
	for i := range tasks {
		t := tasks[i]
		entry, err := c.inner.AddFunc(t.Cron, func() {
			c.RunTask(&t, "cron")
		})
		if err != nil {
			// 单条表达式非法不阻塞整体；保存入口已校验，此处兜底记录
			slog.Warn("cron 表达式装载失败", "task", t.Name, "err", err)
			continue
		}
		c.entries[t.ID] = entry
	}
	return nil
}

// Reload 任务变更后重载。
func (c *Cron) Reload() {
	if err := c.loadAll(); err != nil {
		slog.Error("cron 重载失败", "err", err)
	}
}

// RunTask 执行一次任务（cron 触发或手动），落执行记录；同任务运行中则跳过。
func (c *Cron) RunTask(t *model.CronTask, trigger string) {
	if _, busy := c.running.LoadOrStore(t.ID, struct{}{}); busy {
		slog.Warn("cron 任务仍在运行，跳过本次触发", "task", t.Name)
		return
	}
	defer c.running.Delete(t.ID)

	node, err := c.nodes.ByID("local")
	if err != nil {
		slog.Error("cron 取节点失败", "err", err)
		return
	}
	ac := agentclient.New(node.BaseURL, node.Token)

	start := time.Now()
	now := start
	logRow := &model.CronTaskLog{TaskID: t.ID, TaskName: t.Name, Trigger: trigger, StartAt: start}
	if err := c.db.Create(logRow).Error; err != nil {
		slog.Error("cron 执行记录创建失败", "err", err)
		return
	}

	// B4：任务类型分支（shell 为默认；其余类型由内部服务执行或按 payload 组装命令）
	var output string
	var timedOut, exitFail bool
	var execErr error
	switch t.Type {
	case "db_backup":
		var payload struct {
			DbId uint `json:"dbId"`
		}
		_ = json.Unmarshal([]byte(t.Payload), &payload)
		if c.DBSvc == nil {
			execErr = fmt.Errorf("数据库服务未就绪")
			break
		}
		var res map[string]any
		res, execErr = c.DBSvc.CreateBackup(context.Background(), payload.DbId)
		output = fmt.Sprintf("备份完成: %v", res["file"])
	case "site_backup":
		var payload struct {
			SiteName string `json:"siteName"`
		}
		_ = json.Unmarshal([]byte(t.Payload), &payload)
		if c.SiteBk == nil {
			execErr = fmt.Errorf("站点备份服务未就绪")
			break
		}
		var res map[string]any
		res, execErr = c.SiteBk.Backup(context.Background(), payload.SiteName)
		output = fmt.Sprintf("站点备份完成: %v", res["file"])
	case "container_op":
		var payload struct {
			Container string `json:"container"`
			Action    string `json:"action"`
		}
		_ = json.Unmarshal([]byte(t.Payload), &payload)
		if payload.Action != "start" && payload.Action != "stop" && payload.Action != "restart" {
			execErr = fmt.Errorf("不支持的容器操作: %s", payload.Action)
			break
		}
		var res *dto.ExecResp
		res, execErr = agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, context.Background(),
			"POST", "/agent/v1/exec", &dto.ExecReq{Command: fmt.Sprintf("docker %s %s", payload.Action, payload.Container), TimeoutSecs: t.TimeoutSecs})
		if res != nil {
			output, timedOut, exitFail = res.Output, res.TimedOut, res.ExitCode != 0
		}
	case "script":
		content := t.Command
		var payload struct {
			ScriptId uint `json:"scriptId"`
		}
		_ = json.Unmarshal([]byte(t.Payload), &payload)
		if payload.ScriptId > 0 {
			var sc model.Script
			if err := c.db.First(&sc, payload.ScriptId).Error; err == nil {
				content = sc.Content
			} else {
				execErr = fmt.Errorf("脚本不存在: %d", payload.ScriptId)
			}
		}
		if execErr == nil {
			var res *dto.ExecResp
			res, execErr = agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, context.Background(),
				"POST", "/agent/v1/exec", &dto.ExecReq{Command: content, TimeoutSecs: t.TimeoutSecs})
			if res != nil {
				output, timedOut, exitFail = res.Output, res.TimedOut, res.ExitCode != 0
			}
		}
	default:
		var res *dto.ExecResp
		res, execErr = agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, context.Background(),
			"POST", "/agent/v1/exec", &dto.ExecReq{Command: t.Command, TimeoutSecs: t.TimeoutSecs})
		if res != nil {
			output, timedOut, exitFail = res.Output, res.TimedOut, res.ExitCode != 0
		}
	}

	end := time.Now()
	logRow.EndAt = &end
	logRow.DurationMs = end.Sub(start).Milliseconds()
	if execErr != nil {
		logRow.Success = false
		logRow.Output = tail(execErr.Error(), logOutputCap)
	} else {
		logRow.Success = !timedOut && !exitFail
		logRow.Output = tail(output, logOutputCap)
	}
	if err := c.db.Model(logRow).Updates(map[string]any{
		"end_at": logRow.EndAt, "duration_ms": logRow.DurationMs,
		"success": logRow.Success, "output": logRow.Output,
	}).Error; err != nil {
		slog.Error("cron 执行记录更新失败", "err", err)
	}
	lastSuccess := logRow.Success
	if err := c.db.Model(&model.CronTask{}).Where("id = ?", t.ID).
		Updates(map[string]any{"last_run_at": now, "last_success": lastSuccess}).Error; err != nil {
		slog.Error("cron 任务状态更新失败", "err", err)
	}
	if !logRow.Success {
		slog.Warn("cron 任务执行失败", "task", t.Name, "trigger", trigger, "output", firstLine(logRow.Output))
	}
	// B7：任务事件推送（WS 总线）
	typ := "success"
	title := fmt.Sprintf("计划任务完成: %s", t.Name)
	if !logRow.Success {
		typ = "failed"
		title = fmt.Sprintf("计划任务失败: %s", t.Name)
	}
	wsbus.Default.Publish("task", typ, title, map[string]any{
		"taskId": t.ID, "name": t.Name, "trigger": trigger, "success": logRow.Success,
	})
}

// RunNow 手动立即执行（异步）。
func (c *Cron) RunNow(id uint) error {
	var t model.CronTask
	if err := c.db.First(&t, id).Error; err != nil {
		return errs.New(errs.CodeNotFound, "error.taskNotFound", "任务不存在")
	}
	go c.RunTask(&t, "manual")
	return nil
}

// tail 取字符串尾部（保留最新输出）。
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…[截断]\n" + s[len(s)-n:]
}

// firstLine 取首行（日志摘要）。
func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
