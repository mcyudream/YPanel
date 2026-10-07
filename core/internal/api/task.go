package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// TaskAPI 统一任务中心接口。
type TaskAPI struct {
	Tasks *service.TaskService
}

// List GET /api/v1/tasks?status=&type=&page=&pageSize=
func (a *TaskAPI) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	out, err := a.Tasks.List(c.Query("type"), c.Query("status"), page, pageSize)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Get GET /api/v1/tasks/:id（含日志）
func (a *TaskAPI) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("任务 ID 不合法"))
		return
	}
	row, err := a.Tasks.Get(uint(id))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// Delete DELETE /api/v1/tasks/:id
func (a *TaskAPI) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("任务 ID 不合法"))
		return
	}
	if err := a.Tasks.Delete(uint(id)); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Clear DELETE /api/v1/tasks（清理已结束任务）
func (a *TaskAPI) Clear(c *gin.Context) {
	n, err := a.Tasks.Clear()
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"cleared": n})
}
