// HostsService hosts 可视化编辑（集中记录库 + 多节点托管块分发，M28）。
// 机制：/etc/hosts 尾部维护「ypanel-managed」标记块，应用 = 读原文件 → 剥离旧块 → 追加新块 →
// 原子写回（备份 .bak、保留权限属主）；块外内容（系统/手工条目）零接触。
// 内容经 base64 通道传递（防 shell 注入）；执行与 M24/M27 同构（agent exec）。
package service

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

const (
	hostsFilePath  = "/etc/hosts"
	hostsBlockBegin = "# >>> ypanel-managed >>>"
	hostsBlockEnd   = "# <<< ypanel-managed <<<"
)

// hostsMaxNames 单条记录主机名上限。
const hostsMaxNames = 20

// HostsService hosts 管理服务。
type HostsService struct {
	db    *gorm.DB
	nodes *NodeService
	mu    sync.Mutex // 串行化记录变更触发的全节点重放
}

// NewHostsService 创建。
func NewHostsService(db *gorm.DB, nodes *NodeService) *HostsService {
	return &HostsService{db: db, nodes: nodes}
}

// ---------- 校验 ----------

// hostsValidateRecord 记录校验与归一化（IP 规范化、主机名小写空格分隔）。
func hostsValidateRecord(r *model.HostRecord) error {
	r.Comment = strings.TrimSpace(r.Comment)
	ip := net.ParseIP(strings.TrimSpace(r.IP))
	if ip == nil {
		return svcBadReq("IP 地址不合法")
	}
	r.IP = ip.String()
	names := parseHostNames(r.Hostnames)
	if len(names) == 0 {
		return svcBadReq("至少填写一个主机名")
	}
	if len(names) > hostsMaxNames {
		return svcBadReq(fmt.Sprintf("单条记录主机名不得超过 %d 个", hostsMaxNames))
	}
	for _, n := range names {
		if len(n) > 253 || !dnsHostnamePattern.MatchString(n) {
			return svcBadReq("主机名不合法: " + n)
		}
	}
	r.Hostnames = strings.Join(names, " ")
	if len(r.Comment) > 128 {
		return svcBadReq("备注不超过 128 字符")
	}
	return nil
}

// parseHostNames 主机名解析：逗号/空白分隔，去空、小写、去重（保持顺序）。
func parseHostNames(s string) []string {
	seen := map[string]bool{}
	names := []string{}
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '，'
	}) {
		n := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(part), "."))
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	return names
}

// hostsHostnameConflict 启用记录间主机名重复判定（hosts 首条命中语义，重复易困惑，予以拦截）。
func hostsHostnameConflict(existing []model.HostRecord, r *model.HostRecord) bool {
	if !r.Enabled {
		return false
	}
	names := map[string]bool{}
	for _, n := range parseHostNames(r.Hostnames) {
		names[n] = true
	}
	for _, e := range existing {
		if !e.Enabled || e.ID == r.ID {
			continue
		}
		for _, n := range parseHostNames(e.Hostnames) {
			if names[n] {
				return true
			}
		}
	}
	return false
}

// ---------- 渲染（纯函数） ----------

// renderHostsBlock 渲染托管块（含标记行，确定性输出）。
func renderHostsBlock(records []model.HostRecord) string {
	var b strings.Builder
	b.WriteString(hostsBlockBegin + "\n")
	b.WriteString("# 由 YPanel Hosts 管理生成（块外内容不受影响；手改本块会被下次应用覆盖）\n")
	enabled := make([]model.HostRecord, 0, len(records))
	for _, r := range records {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	sortHostRecords(enabled)
	if len(enabled) == 0 {
		b.WriteString("# （当前无启用记录）\n")
	}
	for _, r := range enabled {
		line := r.IP + "\t" + r.Hostnames
		if r.Comment != "" {
			line += "\t# " + r.Comment
		}
		b.WriteString(line + "\n")
	}
	b.WriteString(hostsBlockEnd + "\n")
	return b.String()
}

// sortHostRecords 按 sort,id 升序（就地排序）。
func sortHostRecords(rows []model.HostRecord) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0; j-- {
			a, c := rows[j-1], rows[j]
			if a.Sort < c.Sort || (a.Sort == c.Sort && a.ID <= c.ID) {
				break
			}
			rows[j-1], rows[j] = c, a
		}
	}
}

// stripHostsBlock 剥离旧托管块（begin 起到 end 行；有 begin 无 end 视为块延伸到文件尾）。
func stripHostsBlock(content string) string {
	i := strings.Index(content, hostsBlockBegin)
	if i < 0 {
		return content
	}
	j := strings.Index(content[i:], hostsBlockEnd)
	if j < 0 {
		return content[:i]
	}
	return content[:i] + content[i+j+len(hostsBlockEnd):]
}

// composeHostsFile 合成新文件：剥离旧块后追加新块（幂等：块一致时输出不变）。
func composeHostsFile(orig string, block string) string {
	stripped := strings.TrimRight(stripHostsBlock(orig), "\n")
	if block == "" {
		return stripped + "\n"
	}
	if stripped == "" {
		return block
	}
	return stripped + "\n\n" + block
}

// extractHostsBlock 提取现存的托管块（无则返回空串；含块尾换行，与渲染输出可直接比对）。
func extractHostsBlock(content string) string {
	i := strings.Index(content, hostsBlockBegin)
	if i < 0 {
		return ""
	}
	j := strings.Index(content[i:], hostsBlockEnd)
	if j < 0 {
		return content[i:]
	}
	end := i + j + len(hostsBlockEnd)
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return content[i:end]
}

// ---------- 节点应用 ----------

// enabledRecords 启用记录（sort,id 升序）。
func (s *HostsService) enabledRecords() ([]model.HostRecord, error) {
	rows := []model.HostRecord{}
	if err := s.db.Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// targetNodeIds 托管节点清单。
func (s *HostsService) targetNodeIds() ([]string, error) {
	ids := []string{}
	if err := s.db.Model(&model.HostTarget{}).Order("id ASC").Pluck("node_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// ApplyToNode 将期望托管块下发到节点（块已一致时跳过写入）。
func (s *HostsService) ApplyToNode(ctx context.Context, nodeId string) error {
	records, err := s.enabledRecords()
	if err != nil {
		return err
	}
	block := renderHostsBlock(records)
	orig, code, err := execOnNode(ctx, s.nodes, nodeId, "cat "+hostsFilePath, 20)
	if err != nil {
		return err
	}
	if code != 0 {
		return svcBadReq("读取 " + hostsFilePath + " 失败（节点 " + nodeId + "）")
	}
	if extractHostsBlock(orig) == block {
		return nil
	}
	script := buildB64WriteScript(hostsFilePath, []byte(composeHostsFile(orig, block)))
	out, code, err := execOnNode(ctx, s.nodes, nodeId, script, 30)
	if err != nil {
		return err
	}
	if code != 0 {
		return svcBadReq("写入 " + hostsFilePath + " 失败: " + tailOutput(out, 400))
	}
	return nil
}

// RemoveNode 解除托管：剥离托管块还原文件并删除目标记录。
func (s *HostsService) RemoveNode(ctx context.Context, nodeId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	orig, code, err := execOnNode(ctx, s.nodes, nodeId, "cat "+hostsFilePath, 20)
	if err != nil {
		return err
	}
	if code != 0 {
		return svcBadReq("读取 " + hostsFilePath + " 失败（节点 " + nodeId + "）")
	}
	if extractHostsBlock(orig) != "" {
		script := buildB64WriteScript(hostsFilePath, []byte(composeHostsFile(orig, "")))
		out, code, err := execOnNode(ctx, s.nodes, nodeId, script, 30)
		if err != nil {
			return err
		}
		if code != 0 {
			return svcBadReq("写回 " + hostsFilePath + " 失败: " + tailOutput(out, 400))
		}
	}
	return s.db.Where("node_id = ?", nodeId).Delete(&model.HostTarget{}).Error
}

// replayAllLocked 重放全部托管节点（一处失败即中断返回）。
func (s *HostsService) replayAllLocked(ctx context.Context) error {
	ids, err := s.targetNodeIds()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.ApplyToNode(ctx, id); err != nil {
			return svcBadReq("节点 " + id + " 应用失败: " + err.Error())
		}
	}
	return nil
}

// ---------- 状态 ----------

// HostsNodeStatus 节点托管状态。
type HostsNodeStatus struct {
	NodeID   string `json:"nodeId"`
	Targeted bool   `json:"targeted"`
	State    string `json:"state"` // match / notApplied / drift / offline / error
	Message  string `json:"message"`
}

// StatusAll 全部托管节点的漂移检测。
func (s *HostsService) StatusAll(ctx context.Context) ([]HostsNodeStatus, error) {
	ids, err := s.targetNodeIds()
	if err != nil {
		return nil, err
	}
	records, err := s.enabledRecords()
	if err != nil {
		return nil, err
	}
	expected := renderHostsBlock(records)
	res := []HostsNodeStatus{}
	for _, id := range ids {
		st := HostsNodeStatus{NodeID: id, Targeted: true}
		out, code, err := execOnNode(ctx, s.nodes, id, "cat "+hostsFilePath, 20)
		switch {
		case err != nil:
			st.State = "offline"
			st.Message = err.Error()
		case code != 0:
			st.State = "error"
			st.Message = tailOutput(out, 200)
		default:
			actual := extractHostsBlock(out)
			if actual == expected {
				st.State = "match"
			} else if actual == "" {
				st.State = "notApplied"
			} else {
				st.State = "drift"
				st.Message = "托管块与面板记录不一致（可能在节点上被手工修改），可重新应用覆盖"
			}
		}
		res = append(res, st)
	}
	return res, nil
}

// ---------- 应用与记录 CRUD ----------

// HostsApplyResult 单节点应用结果。
type HostsApplyResult struct {
	NodeID  string `json:"nodeId"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// Apply 应用/纳管节点：写 target + 下发托管块（部分失败按节点返回明细）。
func (s *HostsService) Apply(ctx context.Context, nodeIds []string) ([]HostsApplyResult, error) {
	if len(nodeIds) == 0 {
		return nil, svcBadReq("请选择要应用的节点")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// 先全部落 target，再逐节点应用（应用失败的节点保留 target，修复后可重新应用）
	for _, id := range nodeIds {
		if _, err := s.nodes.ByID(id); err != nil {
			return nil, err
		}
		var cnt int64
		if err := s.db.Model(&model.HostTarget{}).Where("node_id = ?", id).Count(&cnt).Error; err != nil {
			return nil, err
		}
		if cnt == 0 {
			if err := s.db.Create(&model.HostTarget{NodeID: id}).Error; err != nil {
				return nil, err
			}
		}
	}
	res := []HostsApplyResult{}
	for _, id := range nodeIds {
		r := HostsApplyResult{NodeID: id, OK: true}
		if err := s.ApplyToNode(ctx, id); err != nil {
			r.OK = false
			r.Message = err.Error()
		}
		res = append(res, r)
	}
	return res, nil
}

// ListRecords 记录列表（sort,id 升序）。
func (s *HostsService) ListRecords() ([]model.HostRecord, error) {
	return s.enabledRecords()
}

// SaveRecord 新建（ID=0）/更新；自动重放全部托管节点，失败回滚。
func (s *HostsService) SaveRecord(ctx context.Context, r *model.HostRecord) (*model.HostRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := hostsValidateRecord(r); err != nil {
		return nil, err
	}
	existing, err := s.enabledRecords()
	if err != nil {
		return nil, err
	}
	if hostsHostnameConflict(existing, r) {
		return nil, svcBadReq("主机名与其他启用记录重复（hosts 按首条命中，重复易困惑）: " + r.Hostnames)
	}
	isUpdate := r.ID != 0
	var before model.HostRecord
	if isUpdate {
		if err := s.db.First(&before, r.ID).Error; err != nil {
			return nil, errs.Wrap(errs.ErrNotFound, "记录不存在")
		}
	}
	if isUpdate {
		if err := s.db.Model(&before).Updates(map[string]any{
			"ip": r.IP, "hostnames": r.Hostnames, "comment": r.Comment,
			"enabled": r.Enabled, "sort": r.Sort,
		}).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.Create(r).Error; err != nil {
			return nil, err
		}
	}
	if err := s.replayAllLocked(ctx); err != nil {
		s.rollbackRecord(&before, r, isUpdate)
		return nil, svcBadReq("记录已回滚（应用失败）: " + err.Error())
	}
	return r, nil
}

// DeleteRecord 删除；自动重放，失败回滚。
func (s *HostsService) DeleteRecord(ctx context.Context, id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var row model.HostRecord
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.Wrap(errs.ErrNotFound, "记录不存在")
	}
	if err := s.db.Delete(&row).Error; err != nil {
		return err
	}
	if err := s.replayAllLocked(ctx); err != nil {
		if e := s.db.Create(&row).Error; e != nil {
			slog.Error("hosts 记录删除回滚失败", "id", row.ID, "err", e.Error())
		}
		_ = s.replayAllLocked(ctx)
		return svcBadReq("删除已回滚（应用失败）: " + err.Error())
	}
	return nil
}

// SetRecordEnabled 启停；自动重放，失败还原状态。
func (s *HostsService) SetRecordEnabled(ctx context.Context, id uint, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var row model.HostRecord
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.Wrap(errs.ErrNotFound, "记录不存在")
	}
	if row.Enabled == enabled {
		return nil
	}
	if err := s.db.Model(&row).Update("enabled", enabled).Error; err != nil {
		return err
	}
	if err := s.replayAllLocked(ctx); err != nil {
		if e := s.db.Model(&row).Update("enabled", !enabled).Error; e != nil {
			slog.Error("hosts 记录状态还原失败", "id", row.ID, "err", e.Error())
		}
		_ = s.replayAllLocked(ctx)
		return svcBadReq("已还原记录状态（应用失败）: " + err.Error())
	}
	return nil
}

// rollbackRecord 应用失败后的 DB 回滚。
func (s *HostsService) rollbackRecord(before *model.HostRecord, cur *model.HostRecord, isUpdate bool) {
	if isUpdate {
		if err := s.db.Save(before).Error; err != nil {
			slog.Error("hosts 记录回滚失败", "id", before.ID, "err", err.Error())
		}
	} else {
		if err := s.db.Delete(cur).Error; err != nil {
			slog.Error("hosts 记录回滚失败", "id", cur.ID, "err", err.Error())
		}
	}
	if err := s.replayAllLocked(context.Background()); err != nil {
		slog.Warn("hosts 记录回滚后重放旧块失败", "err", err.Error())
	}
}
