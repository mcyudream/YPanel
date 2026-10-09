// MCP V2（M45）：写工具 + 审批状态机。
// write 级 MCP 工具调用 → 创建 MCPOperation(pending) → 面板内批准 → 客户端凭 operationId 重放执行。
// 状态机：pending → approved → executing → succeeded/failed；pending 10 分钟过期。
package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// mcpWriteTools MCP V2 开放的写工具白名单（低风险高频四个起步）。
var mcpWriteTools = map[string]bool{
	"container_action": true, // start/stop/restart 容器
	"compose_action":   true, // compose up/down/restart
	"create_backup":    true, // 数据库/站点/面板备份
	"reload_nginx":     true,
}

// MCPOpPendingTimeout pending 过期时长。
const MCPOpPendingTimeout = 10 * time.Minute

// MCPOperationService MCP 写操作审批。
type MCPOperationService struct {
	db *gorm.DB
}

// NewMCPOperationService 创建。
func NewMCPOperationService(db *gorm.DB) *MCPOperationService {
	return &MCPOperationService{db: db}
}

// mcpOpID 生成操作 ID。
func mcpOpID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return time.Now().Format("20060102150405") + "-" + hex.EncodeToString(b)
}

// MCPOperationState 状态枚举。
const (
	MCPOpPending   = "pending"
	MCPOpApproved  = "approved"
	MCPOpExecuting = "executing"
	MCPOpSucceeded = "succeeded"
	MCPOpFailed    = "failed"
	MCPOpExpired   = "expired"
	MCPOpRejected  = "rejected"
)

// CreateOperation 创建写操作审批（返回 token 供客户端轮询）。
func (s *MCPOperationService) CreateOperation(tool, args string) (*model.MCPOperation, error) {
	if !mcpWriteTools[tool] {
		return nil, errs.Wrap(errs.ErrBadRequest, "该工具不在 MCP 写白名单内: "+tool)
	}
	op := &model.MCPOperation{
		Token:     mcpOpID() + "-" + mcpOpID()[:8],
		Tool:      tool,
		Args:      truncStr(args, 2000),
		State:     MCPOpPending,
		ExpiresAt: time.Now().Add(MCPOpPendingTimeout),
	}
	if err := s.db.Create(op).Error; err != nil {
		return nil, err
	}
	return op, nil
}

// Approve 批准（面板登录态操作）。
func (s *MCPOperationService) Approve(id uint, username string, approve bool) error {
	var op model.MCPOperation
	if err := s.db.First(&op, id).Error; err != nil {
		return errs.New(errs.CodeNotFound, "error.notFound", "操作不存在")
	}
	if op.State != MCPOpPending {
		return errs.Wrap(errs.ErrBadRequest, "操作已不在待审批状态: "+op.State)
	}
	state := MCPOpApproved
	if !approve {
		state = MCPOpRejected
	}
	return s.db.Model(&op).Updates(map[string]any{
		"state": state, "approved_by": username,
		"approved_at": time.Now(),
	}).Error
}

// ResolveByToken 客户端凭 token 查询/领取执行：
// pending+过期 → expired；approved → 置 executing 并返回（幂等：同 token 二次查返回 executing/终态）。
func (s *MCPOperationService) ResolveByToken(token string) (*model.MCPOperation, error) {
	var op model.MCPOperation
	if err := s.db.Where("token = ?", token).First(&op).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.notFound", "操作不存在")
	}
	if op.State == MCPOpPending && time.Now().After(op.ExpiresAt) {
		_ = s.db.Model(&op).Update("state", MCPOpExpired).Error
		op.State = MCPOpExpired
	}
	return &op, nil
}

// MarkExecuting 领取执行（CAS：仅 approved 可置 executing，防并发重放）。
func (s *MCPOperationService) MarkExecuting(id uint) (bool, error) {
	res := s.db.Model(&model.MCPOperation{}).
		Where("id = ? AND state = ?", id, MCPOpApproved).
		Update("state", MCPOpExecuting)
	return res.RowsAffected > 0, res.Error
}

// MarkDone 终态回写。
func (s *MCPOperationService) MarkDone(id uint, success bool, result string) error {
	state := MCPOpSucceeded
	if !success {
		state = MCPOpFailed
	}
	return s.db.Model(&model.MCPOperation{}).Where("id = ?", id).
		Updates(map[string]any{"state": state, "result": truncStr(result, 2000)}).Error
}

// List 待审批/历史列表（面板审批页）。
func (s *MCPOperationService) List(state string) []model.MCPOperation {
	q := s.db.Order("id desc").Limit(100)
	if state != "" {
		q = q.Where("state = ?", state)
	}
	out := []model.MCPOperation{}
	_ = q.Find(&out).Error
	return out
}
