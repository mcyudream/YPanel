package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/agent/server"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// Node 一个受管节点。
type Node struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"-"`
	Token   string `json:"-"`
}

// onlineWindow 心跳在线窗口。
const onlineWindow = 90 * time.Second

var nodeNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}[a-z0-9]$`)

// NodeService 节点注册表：local（进程内嵌）+ 远程节点（DB）。
type NodeService struct {
	local *Node
	db    *gorm.DB
}

// NewNodeService 启动本机 agent（loopback 随机端口 + 随机 PSK）。
func NewNodeService(ctx context.Context, db *gorm.DB) (*NodeService, error) {
	token := randomHex(32)
	srv := server.New(server.Config{Token: token})
	base, wait, err := srv.Start(ctx)
	if err != nil {
		return nil, err
	}
	go wait()
	hostname, _ := os.Hostname()
	return &NodeService{local: &Node{ID: "local", Name: hostname, BaseURL: base, Token: token}, db: db}, nil
}

// Local 本机节点。
func (s *NodeService) Local() *Node { return s.local }

// ByID 取节点：local 或 DB 中远程节点。
func (s *NodeService) ByID(id string) (*Node, error) {
	if id == "" || id == "local" {
		return s.local, nil
	}
	var row model.Node
	if err := s.db.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.nodeNotFound", "节点不存在")
	}
	return &Node{ID: fmt.Sprintf("%d", row.ID), Name: row.Name, BaseURL: row.Addr, Token: row.Token}, nil
}

// ListNodes 节点列表（含 local 与远程，实时在线状态）。
func (s *NodeService) ListNodes() []map[string]any {
	out := []map[string]any{{
		"id": "local", "name": s.local.Name, "remote": false, "online": true,
	}}
	var rows []model.Node
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return out
	}
	for _, r := range rows {
		out = append(out, map[string]any{
			"id":         fmt.Sprintf("%d", r.ID),
			"name":       r.Name,
			"remote":     true,
			"addr":       r.Addr,
			"hostname":   r.Hostname,
			"os":         r.OS,
			"arch":       r.Arch,
			"version":    r.Version,
			"online":     time.Since(r.LastSeenAt) < onlineWindow,
			"lastSeenAt": r.LastSeenAt,
		})
	}
	return out
}

// pairingCodeChars 配对码字符集（去混淆字符）。
const pairingCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GeneratePairingCode 生成一次性配对码（10 分钟有效）。
func (s *NodeService) GeneratePairingCode() (string, error) {
	code := make([]byte, 8)
	for i := range code {
		b := make([]byte, 1)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		code[i] = pairingCodeChars[int(b[0])%len(pairingCodeChars)]
	}
	row := model.PairingCode{Code: string(code), ExpiredAt: time.Now().Add(10 * time.Minute)}
	if err := s.db.Create(&row).Error; err != nil {
		return "", err
	}
	return row.Code, nil
}

// Pair 节点配对：校验一次性配对码 → 注册/覆盖节点 → 返回专属 PSK。
func (s *NodeService) Pair(code, name, addr, hostname, osName, arch, version string) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return "", errs.Wrap(errs.ErrBadRequest, "配对码为空")
	}
	if !nodeNamePattern.MatchString(name) {
		return "", errs.Wrap(errs.ErrBadRequest, "节点名不合法（小写字母/数字/中划线）")
	}
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		return "", errs.Wrap(errs.ErrBadRequest, "节点地址需以 http(s):// 开头")
	}

	token := randomHex(32)
	now := time.Now()
	node := model.Node{
		Name: name, Addr: strings.TrimRight(addr, "/"), Token: token,
		Hostname: hostname, OS: osName, Arch: arch, Version: version, LastSeenAt: now,
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.PairingCode{}).
			Where("code = ? AND used = ? AND expired_at > ?", code, false, time.Now()).
			Updates(map[string]any{"used": true})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errs.Wrap(errs.ErrBadRequest, "配对码无效、已使用或已过期")
		}
		var existing model.Node
		if err := tx.Where("name = ?", name).First(&existing).Error; err == nil {
			return tx.Model(&existing).Updates(map[string]any{
				"addr": node.Addr, "token": node.Token, "hostname": node.Hostname,
				"os": node.OS, "arch": node.Arch, "version": node.Version, "last_seen_at": now,
			}).Error
		}
		return tx.Create(&node).Error
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

// Heartbeat 节点心跳（Bearer PSK constant-time 校验）并刷新在线时间。
func (s *NodeService) Heartbeat(name, token string) error {
	var row model.Node
	if err := s.db.Where("name = ?", name).First(&row).Error; err != nil {
		return errs.Wrap(errs.ErrBadRequest, "节点未注册")
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(row.Token)) != 1 {
		return errs.ErrForbidden
	}
	return s.db.Model(&row).Update("last_seen_at", time.Now()).Error
}

// DeleteNode 删除远程节点。
func (s *NodeService) DeleteNode(id string) error {
	if id == "local" {
		return errs.Wrap(errs.ErrBadRequest, "本机节点不可删除")
	}
	return s.db.Delete(&model.Node{}, id).Error
}
