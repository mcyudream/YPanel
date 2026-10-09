// Package service DashboardService 首页聚合（面板级计数 + local docker 计数 + 到期证书 + 最近通知）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
)

// DashboardService 首页聚合数据源：core DB 计数 + local agent docker 计数 + 通知。
type DashboardService struct {
	db     *gorm.DB
	nodes  *NodeService
	docker *DockerExtService
	notif  *NotificationService
}

// NewDashboardService 构造首页聚合服务。
func NewDashboardService(db *gorm.DB, nodes *NodeService, docker *DockerExtService, notif *NotificationService) *DashboardService {
	return &DashboardService{db: db, nodes: nodes, docker: docker, notif: notif}
}

// Get 汇总首页数据。docker 不可用时 DockerAvailable=false 且 docker 计数归零，整体仍返回 200。
func (s *DashboardService) Get(ctx context.Context) (*dto.Dashboard, error) {
	out := &dto.Dashboard{
		ExpiringCerts:       []dto.DashboardExpiringCert{},
		RecentNotifications: []map[string]any{},
		CollectedAt:         time.Now(),
	}

	// ---- DB 计数（单库小表，逐个快查；失败记日志不拖垮整页）----
	out.Sites = s.count(&model.Site{})
	out.Databases = s.count(&model.DatabaseInstance{})
	out.CronTasks = s.count(&model.CronTask{})
	s.certCounts(out)
	s.expiringCerts(out)

	// ---- 节点（ListNodes 已含 local + online 判定）----
	for _, n := range s.nodes.ListNodes() {
		out.NodesTotal++
		if online, _ := n["online"].(bool); online {
			out.NodesOnline++
		}
	}

	// ---- Docker（local agent 四个轻量 list 并行；containers 作可用性哨兵）----
	var wg sync.WaitGroup
	var containers []dto.ContainerItem
	var images, volumes, networks json.RawMessage
	dockerOK := true
	var dockerMu sync.Mutex

	wg.Add(4)
	go func() {
		defer wg.Done()
		raw, err := s.docker.Passthrough(ctx, "/agent/v1/docker/containers")
		if err != nil {
			dockerMu.Lock()
			dockerOK = false
			dockerMu.Unlock()
			slog.Warn("dashboard: list containers failed", "err", err)
			return
		}
		if err := json.Unmarshal(raw, &containers); err != nil {
			slog.Warn("dashboard: parse containers failed", "err", err)
		}
	}()
	go func() {
		defer wg.Done()
		raw, err := s.docker.Passthrough(ctx, "/agent/v1/docker/images")
		if err != nil {
			slog.Warn("dashboard: list images failed", "err", err)
			return
		}
		images = raw
	}()
	go func() {
		defer wg.Done()
		raw, err := s.docker.Passthrough(ctx, "/agent/v1/docker/volumes")
		if err != nil {
			slog.Warn("dashboard: list volumes failed", "err", err)
			return
		}
		volumes = raw
	}()
	go func() {
		defer wg.Done()
		raw, err := s.docker.Passthrough(ctx, "/agent/v1/docker/networks")
		if err != nil {
			slog.Warn("dashboard: list networks failed", "err", err)
			return
		}
		networks = raw
	}()
	wg.Wait()

	out.DockerAvailable = dockerOK
	if dockerOK {
		out.ContainersTotal = len(containers)
		for _, c := range containers {
			if c.State == "running" {
				out.ContainersRunning++
			}
		}
		out.Images = jsonArrayLen(images)
		out.Volumes = jsonArrayLen(volumes)
		out.Networks = jsonArrayLen(networks)
	}

	// ---- 最近通知 5 条 ----
	if s.notif != nil {
		for _, n := range s.notif.List(5) {
			out.RecentNotifications = append(out.RecentNotifications, map[string]any{
				"id":        n.ID,
				"level":     n.Level,
				"title":     n.Title,
				"content":   n.Content,
				"read":      n.Read,
				"createdAt": n.CreatedAt,
			})
		}
	}

	return out, nil
}

// count 单表计数（错误记日志返回 0）。
func (s *DashboardService) count(model any) int {
	var n int64
	if err := s.db.Model(model).Count(&n).Error; err != nil {
		slog.Warn("dashboard: count failed", "model", fmt.Sprintf("%T", model), "err", err)
		return 0
	}
	return int(n)
}

// certCounts 按状态分组计数证书。
func (s *DashboardService) certCounts(out *dto.Dashboard) {
	type row struct {
		Status string
		N      int64
	}
	var rows []row
	if err := s.db.Model(&model.Certificate{}).Select("status, count(*) as n").Group("status").Scan(&rows).Error; err != nil {
		slog.Warn("dashboard: cert counts failed", "err", err)
		return
	}
	for _, r := range rows {
		switch r.Status {
		case "ok":
			out.CertsOK = int(r.N)
		case "expiring":
			out.CertsExpiring = int(r.N)
		case "expired":
			out.CertsExpired = int(r.N)
		}
	}
}

// expiringCerts 30 天内到期证书（升序，最多 5 条；已过期排最前按时间升序自然成立）。
func (s *DashboardService) expiringCerts(out *dto.Dashboard) {
	var rows []model.Certificate
	deadline := time.Now().Add(30 * 24 * time.Hour)
	if err := s.db.Model(&model.Certificate{}).
		Where("not_after IS NOT NULL AND not_after <= ?", deadline).
		Order("not_after asc").Limit(5).Find(&rows).Error; err != nil {
		slog.Warn("dashboard: expiring certs failed", "err", err)
		return
	}
	now := time.Now()
	for _, r := range rows {
		if r.NotAfter == nil {
			continue
		}
		days := int(r.NotAfter.Sub(now).Hours() / 24)
		status := "expiring"
		if days < 0 {
			status = "expired"
		}
		name := r.CertName
		if name == "" {
			name = r.Domain
		}
		out.ExpiringCerts = append(out.ExpiringCerts, dto.DashboardExpiringCert{
			ID:       r.ID,
			CertName: name,
			Domain:   r.Domain,
			NotAfter: *r.NotAfter,
			DaysLeft: days,
			Status:   status,
		})
	}
}

// jsonArrayLen 解析 agent 返回的 JSON 数组长度（异常返回 0）。
func jsonArrayLen(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return 0
	}
	return len(arr)
}
