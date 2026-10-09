// ScriptService 脚本库（B13/M36）：计划任务引用的脚本片段管理 + 手动运行留痕。
package service

import (
	"context"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
	"gorm.io/gorm"
)

// ScriptService 脚本库。
type ScriptService struct {
	db    *gorm.DB
	nodes *NodeService
}

// NewScriptService 创建。
func NewScriptService(db *gorm.DB, nodes *NodeService) *ScriptService {
	return &ScriptService{db: db, nodes: nodes}
}

// Run 手动执行脚本（M36）：agent exec 执行 + 执行记录留痕（CronTaskLog，TaskID=0）。
func (s *ScriptService) Run(ctx context.Context, id uint) (map[string]any, error) {
	var row model.Script
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "脚本不存在")
	}
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	start := time.Now()
	logRow := &model.CronTaskLog{TaskID: 0, TaskName: "脚本:" + row.Name, Trigger: "manual", StartAt: start}
	_ = s.db.Create(logRow).Error
	out, eerr := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: row.Content, TimeoutSecs: 300})
	end := time.Now()
	logRow.EndAt = &end
	logRow.DurationMs = end.Sub(start).Milliseconds()
	if eerr != nil {
		logRow.Success = false
		logRow.Output = tail(eerr.Error(), logOutputCap)
	} else {
		logRow.Success = !out.TimedOut && out.ExitCode == 0
		logRow.Output = tail(out.Output, logOutputCap)
	}
	_ = s.db.Model(logRow).Updates(map[string]any{
		"end_at": logRow.EndAt, "duration_ms": logRow.DurationMs,
		"success": logRow.Success, "output": logRow.Output,
	}).Error
	if eerr != nil {
		return nil, eerr
	}
	return map[string]any{"success": logRow.Success, "output": logRow.Output}, nil
}

// List 全量列表（脚本数量有限，不分页）。
func (s *ScriptService) List() []model.Script {
	out := []model.Script{}
	_ = s.db.Order("id desc").Find(&out).Error
	return out
}

// Create 创建脚本。
func (s *ScriptService) Create(name, content string) (*model.Script, error) {
	if name == "" || content == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "脚本名与内容必填")
	}
	row := &model.Script{Name: name, Content: content}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// Update 更新脚本。
func (s *ScriptService) Update(id uint, name, content string) (*model.Script, error) {
	var row model.Script
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "脚本不存在")
	}
	if name != "" {
		row.Name = name
	}
	if content != "" {
		row.Content = content
	}
	if err := s.db.Model(&row).Updates(map[string]any{"name": row.Name, "content": row.Content}).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Delete 删除脚本。
func (s *ScriptService) Delete(id uint) error {
	return s.db.Delete(&model.Script{}, id).Error
}
