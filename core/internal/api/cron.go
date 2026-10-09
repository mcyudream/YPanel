package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// CronAPI 计划任务接口。
type CronAPI struct {
	DB   *gorm.DB
	Cron *service.Cron
}

// List GET /api/v1/cron/tasks
func (a *CronAPI) List(c *gin.Context) {
	page, size := pageParams(c)
	var total int64
	var rows []model.CronTask
	q := a.DB.Model(&model.CronTask{})
	if err := q.Count(&total).Error; err != nil {
		respErr(c, err)
		return
	}
	if err := q.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		respErr(c, err)
		return
	}
	items := make([]dto.CronTask, 0, len(rows))
	for i := range rows {
		t := &rows[i]
		items = append(items, dto.CronTask{
			ID: t.ID, Name: t.Name, Cron: t.Cron, Command: t.Command, Type: t.Type, Payload: t.Payload,
			Enabled: t.Enabled, TimeoutSecs: t.TimeoutSecs,
			LastRunAt: t.LastRunAt, LastSuccess: t.LastSuccess, CreatedAt: t.CreatedAt,
		})
	}
	respOK(c, dto.NewPageResp(total, items))
}

// Create POST /api/v1/cron/tasks
func (a *CronAPI) Create(c *gin.Context) {
	req, ok := bind[dto.CronTaskCreateReq](c)
	if !ok {
		return
	}
	if _, err := cron.ParseStandard(req.Cron); err != nil {
		respErr(c, errs.Wrap(errs.ErrBadRequest, "cron 表达式不合法: "+err.Error()))
		return
	}
	timeout := req.TimeoutSecs
	if timeout <= 0 {
		timeout = 300
	}
		taskType := req.Type
	if taskType == "" {
		taskType = "shell"
	}
	row := model.CronTask{Name: req.Name, Cron: req.Cron, Command: req.Command, Type: taskType, Payload: req.Payload, Enabled: true, TimeoutSecs: timeout, NodeID: req.NodeID}
	if err := a.DB.Create(&row).Error; err != nil {
		respErr(c, err)
		return
	}
	a.Cron.Reload()
	respOK(c, row)
}

// Update PUT /api/v1/cron/tasks/:id
func (a *CronAPI) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errs.ErrBadRequest)
		return
	}
	req, ok := bind[dto.CronTaskUpdateReq](c)
	if !ok {
		return
	}
	if req.Cron != nil {
		if _, err := cron.ParseStandard(*req.Cron); err != nil {
			respErr(c, errs.Wrap(errs.ErrBadRequest, "cron 表达式不合法: "+err.Error()))
			return
		}
	}
	var row model.CronTask
	if err := a.DB.First(&row, id).Error; err != nil {
		respErr(c, errs.New(errs.CodeNotFound, "error.taskNotFound", "任务不存在"))
		return
	}
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Cron != nil {
		updates["cron"] = *req.Cron
	}
	if req.Command != nil {
		updates["command"] = *req.Command
	}
	if req.TimeoutSecs != nil {
		updates["timeout_secs"] = *req.TimeoutSecs
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if len(updates) > 0 {
		if err := a.DB.Model(&row).Updates(updates).Error; err != nil {
			respErr(c, err)
			return
		}
	}
	a.Cron.Reload()
	respOK(c, row)
}

// Delete DELETE /api/v1/cron/tasks/:id
func (a *CronAPI) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errs.ErrBadRequest)
		return
	}
	if err := a.DB.Delete(&model.CronTask{}, id).Error; err != nil {
		respErr(c, err)
		return
	}
	a.Cron.Reload()
	respOK(c, struct{}{})
}

// Run POST /api/v1/cron/tasks/:id/run
func (a *CronAPI) Run(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errs.ErrBadRequest)
		return
	}
	if err := a.Cron.RunNow(uint(id)); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Logs GET /api/v1/cron/logs?taskId=&page=&pageSize=
func (a *CronAPI) Logs(c *gin.Context) {
	page, size := pageParams(c)
	var total int64
	var rows []model.CronTaskLog
	q := a.DB.Model(&model.CronTaskLog{})
	if v := c.Query("taskId"); v != "" {
		q = q.Where("task_id = ?", v)
	}
	if err := q.Count(&total).Error; err != nil {
		respErr(c, err)
		return
	}
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		respErr(c, err)
		return
	}
	items := make([]dto.CronTaskLogItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, dto.CronTaskLogItem{
			ID: r.ID, TaskID: r.TaskID, TaskName: r.TaskName, Trigger: r.Trigger,
			StartAt: r.StartAt, EndAt: r.EndAt, DurationMs: r.DurationMs,
			Success: r.Success, Output: r.Output,
		})
	}
	respOK(c, dto.NewPageResp(total, items))
}
