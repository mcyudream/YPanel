// FileCrossService 跨节点文件拷贝（M56）：core 中转流式（src agent download → io.Pipe →
// dst agent upload），零临时盘、不要求节点间互通、不经浏览器。目录走 agent 的 tar.gz 流
// （download 对目录自动打包），目标侧解包保留权限属主。sha256 双端校验。任务化承载进度。
package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// FileCrossService 跨节点文件拷贝服务。
type FileCrossService struct {
	nodes *NodeService
	tasks *TaskService
}

// NewFileCrossService 创建（tasks 可 nil——仅同步小拷贝可用）。
func NewFileCrossService(nodes *NodeService, tasks *TaskService) *FileCrossService {
	return &FileCrossService{nodes: nodes, tasks: tasks}
}

// FileCrossItem 单个拷贝条目。
type FileCrossItem struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
}

// CopyAcrossReq 跨节点拷贝请求。
type CopyAcrossReq struct {
	SrcNode   string          `json:"srcNode" binding:"required"`
	SrcPath   string          `json:"srcPath" binding:"required"`
	DstNode   string          `json:"dstNode" binding:"required"`
	DstDir    string          `json:"dstDir" binding:"required"`
	Overwrite bool            `json:"overwrite"`
	Items     []FileCrossItem `json:"items,omitempty"` // 批量（拖拽多选）；空 = 单项 srcPath
}

// clientFor 节点客户端（空/local=本机）。
func (s *FileCrossService) clientFor(nodeId string) (*agentclient.Client, error) {
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// CopyAcrossTask 创建跨节点拷贝任务。
func (s *FileCrossService) CopyAcrossTask(ctx context.Context, req CopyAcrossReq) (map[string]any, error) {
	srcNode := normalizeNodeID(req.SrcNode)
	dstNode := normalizeNodeID(req.DstNode)
	if srcNode == dstNode {
		return nil, svcBadReq("源与目标为同一节点（同节点复制请用文件管理器复制）")
	}
	items := req.Items
	if len(items) == 0 {
		items = []FileCrossItem{{Path: strings.TrimSuffix(req.SrcPath, "/"), Name: path.Base(strings.TrimSuffix(req.SrcPath, "/")), IsDir: false}}
	}
	if len(items) == 0 || items[0].Name == "" {
		return nil, svcBadReq("拷贝条目为空")
	}
	if s.tasks == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "任务服务不可用")
	}
	title := fmt.Sprintf("跨节点拷贝 %d 项：%s → %s", len(items), srcNode, dstNode)
	task, err := s.tasks.StartTask("file-copy", title, srcNode+"→"+dstNode, 2*time.Hour,
		func(ctx context.Context, logf TaskLogf) error {
			return s.runCopy(ctx, logf, srcNode, dstNode, req.DstDir, req.Overwrite, items)
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

// runCopy 任务体：逐条目 download 流 → upload 流。
func (s *FileCrossService) runCopy(ctx context.Context, logf TaskLogf, srcNode, dstNode, dstDir string, overwrite bool, items []FileCrossItem) error {
	src, err := s.clientFor(srcNode)
	if err != nil {
		return err
	}
	dst, err := s.clientFor(dstNode)
	if err != nil {
		return err
	}
	// 目标目录必须先建（校验/解包都依赖它；tar -C 不自建目录）
	if _, code, err := execOnNode(ctx, s.nodes, dstNode, fmt.Sprintf("mkdir -p '%s'", dstDir), 15); err != nil || code != 0 {
		return fmt.Errorf("创建目标目录失败")
	}
	done, skipped := 0, 0
	for i, it := range items {
		logf("info", "[%d/%d] %s（%s）", i+1, len(items), it.Name, mapStr(it.IsDir, "目录", "文件"))
		// 冲突预检：overwrite=false 时目标同名跳过
		if !overwrite {
			out, code, _ := execOnNode(ctx, s.nodes, dstNode,
				fmt.Sprintf("test -e '%s/%s' && echo EXISTS", dstDir, it.Name), 15)
			if code == 0 && strings.Contains(out, "EXISTS") {
				logf("warn", "目标已存在 %s（未勾选覆盖），跳过", it.Name)
				skipped++
				continue
			}
		}
		sum, size, isTar, err := s.streamCopy(ctx, logf, src, dst, srcNode, dstNode, srcNode, it.Path, dstDir, it.Name)
		if err != nil {
			return fmt.Errorf("%s: %w", it.Name, err)
		}
		_ = size
		// 目标侧 sha256 复核
		kind := mapStr(isTar, "tmp.tar.gz", it.Name)
		out, code, err := execOnNode(ctx, s.nodes, dstNode,
			fmt.Sprintf("cd %s && printf '%%s  %%s\\n' '%s' '%s' | sha256sum -c - >/dev/null && echo YPSUMOK", dstDir, sum, kind), 60)
		if err != nil || code != 0 || !strings.Contains(out, "YPSUMOK") {
			dbg, _, _ := execOnNode(ctx, s.nodes, dstNode,
				fmt.Sprintf("cd %s && ls -la && sha256sum '%s' 2>&1", dstDir, kind), 30)
			logf("error", "目标侧实际状态：\n%s\n（core 侧期望 sha256=%s）", tailStr(dbg, 600), sum)
			clean := fmt.Sprintf("rm -f '%s/%s'", dstDir, kind)
			_, _, _ = execOnNode(ctx, s.nodes, dstNode, clean, 15)
			return fmt.Errorf("%s: 目标侧 sha256 校验失败（已清理）", it.Name)
		}
		if isTar {
			// 解包到目标目录并清理临时包
			out, code, err := execOnNode(ctx, s.nodes, dstNode,
				fmt.Sprintf("tar xzf '%s/%s' -C '%s' && rm -f '%s/%s' && echo YPOK", dstDir, kind, dstDir, dstDir, kind), 600)
			if err != nil || code != 0 || !strings.Contains(out, "YPOK") {
				return fmt.Errorf("%s: 解包失败：%s", it.Name, tailStr(out, 300))
			}
		}
		done++
		logf("info", "%s ✓", it.Name)
	}
	logf("info", "完成：成功 %d，跳过 %d", done, skipped)
	return nil
}

// streamCopy 下载流 → 上传流（tee sha256）；返回 (sha256hex, size, 是否目录tar流)。
func (s *FileCrossService) streamCopy(ctx context.Context, logf TaskLogf, src, dst *agentclient.Client, srcNode, dstNode, tmpNode, srcPath, dstDir, name string) (string, int64, bool, error) {
	req, err := src.NewRequest(ctx, http.MethodGet, "/agent/v1/files/download?path="+escapeURL(srcPath), nil)
	if err != nil {
		return "", 0, false, err
	}
	resp, err := src.HTTP.Do(req)
	if err != nil {
		return "", 0, false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", 0, false, fmt.Errorf("源读取 HTTP %d", resp.StatusCode)
	}
	// agent 错误响应是 HTTP 200 + JSON 信封（writeErr）——按 Content-Type 识别，防止把错误 JSON 当文件内容拷走
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "application/json") {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return "", 0, false, fmt.Errorf("源文件不存在或不可读: %s", strings.TrimSpace(string(b)))
	}
	isTar := strings.Contains(resp.Header.Get("Content-Type"), "gzip")
	// 远端暂存名必须与 runCopy 的校验文件名一致（isTar 时固定 tmp.tar.gz）
	remoteName := name
	if isTar {
		remoteName = "tmp.tar.gz"
	}
	// 同步缓冲：完整读取并计算 sha256（双 pipe 并发写曾致截断体静默上传→双端 sha 不一致）
	h := sha256.New()
	body, err := io.ReadAll(io.TeeReader(resp.Body, h))
	if err != nil {
		return "", 0, isTar, fmt.Errorf("源流读取中断: %w", err)
	}
	sumHex := hex.EncodeToString(h.Sum(nil))
	if err := streamUpload(ctx, dst, dstDir, remoteName, body); err != nil {
		return "", 0, isTar, err
	}
	return sumHex, int64(len(body)), isTar, nil
}

// streamUpload multipart 上传（内容已在内存缓冲；早期 pipe 双 goroutine 版本存在
// 截断静默上传竞态——双端 sha 不一致的根因，已改 bytes 同步写）。
func streamUpload(ctx context.Context, dst *agentclient.Client, dir, filename string, content []byte) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(content); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	req, err := dst.NewRequest(ctx, http.MethodPost, "/agent/v1/files/upload?path="+escapeURL(dir), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := dst.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	var env dto.Resp[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return err
	}
	if env.Code != 0 {
		return &errs.Error{Code: env.Code, Message: env.Message}
	}
	return nil
}
