// FileExtService 文件管理扩展（M38）：收藏夹 / 分享链接 / 远程 URL 下载（core 侧 SSRF 逐跳校验）。
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// FileExtService 文件扩展服务。
type FileExtService struct {
	db    *gorm.DB
	nodes *NodeService
	tasks *TaskService // 远程下载任务化（可 nil）
}

// NewFileExtService 创建。
func NewFileExtService(db *gorm.DB, nodes *NodeService, tasks *TaskService) *FileExtService {
	return &FileExtService{db: db, nodes: nodes, tasks: tasks}
}

func (s *FileExtService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// ---- 收藏夹 ----

// Favorites 收藏列表。
func (s *FileExtService) Favorites() []model.FileFavorite {
	out := []model.FileFavorite{}
	_ = s.db.Order("id desc").Find(&out).Error
	return out
}

// FavoriteAdd 添加收藏（目录或文件）。
func (s *FileExtService) FavoriteAdd(p string) (*model.FileFavorite, error) {
	p = path.Clean(p)
	if !path.IsAbs(p) || strings.Contains(p, "..") {
		return nil, errs.Wrap(errs.ErrBadRequest, "路径不合法")
	}
	var count int64
	_ = s.db.Model(&model.FileFavorite{}).Where("path = ?", p).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "已在收藏夹")
	}
	row := &model.FileFavorite{Path: p, Name: path.Base(p)}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// FavoriteRemove 移除收藏。
func (s *FileExtService) FavoriteRemove(id uint) error {
	return s.db.Delete(&model.FileFavorite{}, id).Error
}

// ---- 分享链接 ----

// newShareToken 生成 32 hex token 与其 SHA256。
func newShareToken() (raw, hash string, err error) {
	b := make([]byte, 16)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}

// ShareCreate 创建分享（days=0 永久；path 必须为文件）。
func (s *FileExtService) ShareCreate(ctx context.Context, p string, days int) (map[string]any, error) {
	p = path.Clean(p)
	if !path.IsAbs(p) || strings.Contains(p, "..") {
		return nil, errs.Wrap(errs.ErrBadRequest, "路径不合法")
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	st, serr := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+strings.ReplaceAll(path.Dir(p), " ", "%20"))
	if serr != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "无法访问该路径")
	}
	found := false
	isDir := false
	for _, e := range st.Entries {
		if e.Path == p {
			found = true
			isDir = e.IsDir
			break
		}
	}
	if !found {
		return nil, errs.Wrap(errs.ErrNotFound, "文件不存在")
	}
	if isDir {
		return nil, errs.Wrap(errs.ErrBadRequest, "仅支持分享文件")
	}
	raw, hash, err := newShareToken()
	if err != nil {
		return nil, err
	}
	row := &model.FileShare{TokenHash: hash, Path: p, Name: path.Base(p), Enabled: true}
	if days > 0 {
		t := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		row.ExpireAt = &t
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": row.ID, "token": raw, "name": row.Name, "expireAt": row.ExpireAt}, nil
}

// ShareList 分享管理列表（不回 token）。
func (s *FileExtService) ShareList() []map[string]any {
	rows := []model.FileShare{}
	_ = s.db.Order("id desc").Find(&rows).Error
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		expired := r.ExpireAt != nil && r.ExpireAt.Before(time.Now())
		out = append(out, map[string]any{
			"id": r.ID, "path": r.Path, "name": r.Name, "expireAt": r.ExpireAt,
			"enabled": r.Enabled, "valid": r.Enabled && !expired,
		})
	}
	return out
}

// ShareRevoke 撤销分享。
func (s *FileExtService) ShareRevoke(id uint) error {
	return s.db.Delete(&model.FileShare{}, id).Error
}

// ShareResolve 公开访问解析：token → agent 文件流（过期/禁用 404）。返回 (文件名, 流, 清理函数, 错误)。
func (s *FileExtService) ShareResolve(token string) (string, io.ReadCloser, func(), error) {
	sum := sha256.Sum256([]byte(token))
	var row model.FileShare
	if err := s.db.Where("token_hash = ?", hex.EncodeToString(sum[:])).First(&row).Error; err != nil {
		return "", nil, func() {}, errs.New(errs.CodeNotFound, "error.notFound", "分享不存在或已失效")
	}
	if !row.Enabled || (row.ExpireAt != nil && row.ExpireAt.Before(time.Now())) {
		return "", nil, func() {}, errs.New(errs.CodeNotFound, "error.notFound", "分享已过期或已撤销")
	}
	ac, err := s.client()
	if err != nil {
		return "", nil, func() {}, err
	}
	req, rerr := ac.NewRequest(context.Background(), http.MethodGet, "/agent/v1/files/download?path="+strings.ReplaceAll(row.Path, " ", "%20"), nil)
	if rerr != nil {
		return "", nil, func() {}, rerr
	}
	resp, derr := ac.HTTP.Do(req)
	if derr != nil {
		return "", nil, func() {}, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "文件读取失败")
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return "", nil, func() {}, errs.New(errs.CodeNotFound, "error.notFound", "文件不存在")
	}
	cleanup := func() { _ = resp.Body.Close() }
	return row.Name, resp.Body, cleanup, nil
}

// ---- 远程 URL 下载（SSRF 防护：仅 http/https；逐跳解析并拒绝内网/保留地址；≤3 跳） ----

// checkPublicHost 解析主机并校验全部 IP 均为公网。
func checkPublicHost(host string) error {
	if strings.EqualFold(host, "localhost") {
		return errs.Wrap(errs.ErrBadRequest, "拒绝访问内网地址")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return errs.Wrap(errs.ErrBadRequest, "域名解析失败: "+err.Error())
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
			return errs.Wrap(errs.ErrBadRequest, "拒绝访问内网/保留地址")
		}
	}
	return nil
}

// progressReader 计数读取器：累计已读字节数回调（下载进度）。
type progressReader struct {
	r      io.Reader
	onRead func(n int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.onRead(int64(n))
	}
	return n, err
}

// RemoteDownloadTask 远程下载任务化：立即回 taskId，下载进度（百分比/速度，2s 节流）进任务日志，
// 任务中心与文件页均可跟踪；完成后任务日志记录落盘路径。
func (s *FileExtService) RemoteDownloadTask(rawURL, destDir string) (map[string]any, error) {
	if s.tasks == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "任务服务不可用")
	}
	name := path.Base(mustParsePath(rawURL))
	task, err := s.tasks.StartTask("file-download", "远程下载 "+name, destDir, 2*time.Hour,
		func(ctx context.Context, logf TaskLogf) error {
			saved, err := s.remoteDownloadRun(ctx, logf, rawURL, destDir)
			if err != nil {
				return err
			}
			logf("info", "下载完成：%s/%s", destDir, saved)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

// mustParsePath URL path 提取（解析失败回退时间戳名，仅用于任务标题展示）。
func mustParsePath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path == "" {
		return fmt.Sprintf("download-%d", time.Now().Unix())
	}
	return u.Path
}

// remoteDownloadRun 经 core 下载（逐跳 SSRF 复核）并流式落盘 agent 目录。返回文件名；
// 下载进度经 plog 写任务日志（2s 节流，Content-Length 已知时带百分比）。
func (s *FileExtService) remoteDownloadRun(ctx context.Context, logf TaskLogf, rawURL, destDir string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errs.Wrap(errs.ErrBadRequest, "仅支持 http(s) 直链")
	}
	if !path.IsAbs(destDir) || strings.Contains(destDir, "..") {
		return "", errs.Wrap(errs.ErrBadRequest, "目标目录不合法")
	}
	name := path.Base(u.Path)
	if name == "" || name == "/" || name == "." {
		name = fmt.Sprintf("download-%d", time.Now().Unix())
	}
	// HEAD 预检大小（≤2GB）
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 手动逐跳
		},
		Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}},
	}
	addr := rawURL
	var resp *http.Response
	for hop := 0; hop < 4; hop++ {
		if err := checkPublicHost(u.Hostname()); err != nil {
			return "", err
		}
		req, rerr := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
		if rerr != nil {
			return "", rerr
		}
		req.Header.Set("User-Agent", "YPanel/1.0")
		resp, err = client.Do(req)
		if err != nil {
			return "", errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "下载失败: "+err.Error())
		}
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := resp.Header.Get("Location")
			_ = resp.Body.Close()
			if loc == "" {
				return "", errs.Wrap(errs.ErrBadRequest, "重定向缺少 Location")
			}
			next, perr := u.Parse(loc)
			if perr != nil {
				return "", errs.Wrap(errs.ErrBadRequest, "重定向地址不合法")
			}
			addr, u = next.String(), next
			continue
		}
		break
	}
	if resp == nil {
		return "", errs.Wrap(errs.ErrBadRequest, "下载失败")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", errs.Wrapc(errs.CodeFileOpFailed, fmt.Sprintf("远端返回 HTTP %d", resp.StatusCode))
	}
	if resp.ContentLength > 2<<30 {
		return "", errs.Wrap(errs.ErrBadRequest, "文件超过 2GB 上限")
	}
	logf("info", "开始下载 %s → %s", rawURL, destDir)
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	// 进度：计数读取器 + 2s 节流日志（任务中心前端 2s 轮询，无需更密）
	plog := NewProgressLogger(logf, 2*time.Second)
	var done, lastDone int64
	last := time.Now()
	total := resp.ContentLength
	body := io.Reader(resp.Body)
	if logf != nil {
		body = &progressReader{r: resp.Body, onRead: func(n int64) {
			done += n
			span := time.Since(last).Seconds()
			if span < 2 || done < lastDone {
				return
			}
			speed := float64(done-lastDone) / span / 1048576
			if total > 0 {
				plog("已下载 %.1f/%.1f MB（%.1f MB/s，%.0f%%）", float64(done)/1048576, float64(total)/1048576, speed, float64(done)/float64(total)*100)
			} else {
				plog("已下载 %.1f MB（%.1f MB/s）", float64(done)/1048576, speed)
			}
			lastDone, last = done, time.Now()
		}}
	}
	if err := UploadMultipartToAgent(ac, ctx, destDir, name, body); err != nil {
		return "", err
	}
	if logf != nil {
		logf("info", "传输完成 %.1f MB", float64(done)/1048576)
	}
	return name, nil
}
