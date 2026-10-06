package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// revisionKeepPerScope 每个 scope 保留的快照条数（超出滚动清理最旧）。
const revisionKeepPerScope = 50

// composeManagedDir 托管 compose 项目根目录；其下所有文件编辑均纳入版本快照。
const composeManagedDir = "/opt/ypanel/compose/"

// RevisionService 受管配置版本快照（M23）：面板写盘前自动快照旧内容，支持对比与回滚。
// 语义：快照 = 每次面板保存前的盘上旧内容；当前内容永远以盘上为准（外部编辑会在下一次保存时被收进历史）。
type RevisionService struct {
	db    *gorm.DB
	nodes *NodeService
}

// NewRevisionService 创建。
func NewRevisionService(db *gorm.DB, nodes *NodeService) *RevisionService {
	return &RevisionService{db: db, nodes: nodes}
}

// ScopeFor 受管路径 → 版本 scope（node:path）；非受管路径返回空串。
func ScopeFor(node, p string) string {
	if p != "/etc/docker/daemon.json" && !strings.HasPrefix(p, composeManagedDir) {
		return ""
	}
	return node + ":" + p
}

// RevisionMeta 版本列表条目（不含内容，列表不背大字段）。
type RevisionMeta struct {
	ID        uint      `json:"id"`
	Scope     string    `json:"scope"`
	Trigger   string    `json:"trigger"`
	Note      string    `json:"note"`
	Author    string    `json:"author"`
	Size      int       `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *RevisionService) client(node string) (*agentclient.Client, error) {
	n, err := s.nodes.ByID(node)
	if err != nil {
		return nil, err
	}
	return agentclient.New(n.BaseURL, n.Token), nil
}

// SnapshotBefore 从 agent 读当前盘上内容并保存一条快照。
// 文件不存在/读取失败/超过读取上限时静默跳过（新文件没有旧版可存；快照失败不阻塞写盘）。
func (s *RevisionService) SnapshotBefore(ctx context.Context, node, p, trigger, author string) {
	scope := ScopeFor(node, p)
	if scope == "" {
		return
	}
	ac, err := s.client(node)
	if err != nil {
		slog.Debug("revision: agent 不可用，跳过快照", "path", p, "err", err)
		return
	}
	r, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+url.QueryEscape(p))
	if err != nil || r.Truncated {
		// 文件不存在/读取失败/超限：静默跳过（快照不阻塞写盘）
		return
	}
	note := p
	if trigger == "save" {
		note = "保存前自动快照"
	}
	if err := s.save(ctx, scope, r.Content, trigger, note, author); err != nil {
		slog.Warn("revision: 快照落库失败", "scope", scope, "err", err)
	}
}

func (s *RevisionService) save(ctx context.Context, scope, content, trigger, note, author string) error {
	row := model.ConfigRevision{
		Scope: scope, Content: content, Trigger: trigger, Note: note, Author: author, CreatedAt: time.Now(),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return err
	}
	// 滚动清理：仅保留最近 revisionKeepPerScope 条
	var ids []uint
	if err := s.db.Model(&model.ConfigRevision{}).Where("scope = ?", scope).
		Order("id desc").Offset(revisionKeepPerScope).Limit(200).Pluck("id", &ids).Error; err == nil && len(ids) > 0 {
		_ = s.db.Where("id IN ?", ids).Delete(&model.ConfigRevision{}).Error
	}
	return nil
}

// List 按 node+path 列版本（不含内容），新→旧。
func (s *RevisionService) List(ctx context.Context, node, p string) ([]RevisionMeta, error) {
	scope := ScopeFor(node, p)
	if scope == "" {
		return nil, errs.Wrapc(errs.CodeBadRequest, "该路径不受版本管理")
	}
	rows := []model.ConfigRevision{}
	if err := s.db.Where("scope = ?", scope).Order("id desc").Limit(revisionKeepPerScope).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]RevisionMeta, 0, len(rows))
	for _, r := range rows {
		out = append(out, RevisionMeta{
			ID: r.ID, Scope: r.Scope, Trigger: r.Trigger, Note: r.Note, Author: r.Author,
			Size: len(r.Content), CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

// Get 单条（含内容）。
func (s *RevisionService) Get(ctx context.Context, id uint) (*model.ConfigRevision, error) {
	row := &model.ConfigRevision{}
	if err := s.db.First(row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.Wrapc(errs.CodeNotFound, "版本不存在")
		}
		return nil, err
	}
	return row, nil
}

// Restore 回滚：取旧版本内容 → agent 写盘 → 存 rollback 快照 → 返回内容。
func (s *RevisionService) Restore(ctx context.Context, id uint, username string) (*model.ConfigRevision, error) {
	rev, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	// scope = node:path
	idx := strings.Index(rev.Scope, ":")
	if idx <= 0 {
		return nil, errs.Wrapc(errs.CodeInternal, "版本 scope 不合法")
	}
	node, p := rev.Scope[:idx], rev.Scope[idx+1:]
	ac, err := s.client(node)
	if err != nil {
		return nil, err
	}
	if _, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write", &dto.FileWriteReq{Path: p, Content: rev.Content}); err != nil {
		return nil, fmt.Errorf("写盘失败: %w", err)
	}
	_ = s.save(ctx, rev.Scope, rev.Content, "rollback", "回滚自 #"+fmt.Sprint(rev.ID), username)
	return rev, nil
}
