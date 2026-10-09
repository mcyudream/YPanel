// AgentUpgradeService 被管节点 agent 一键更新（M54）。
// 流程：core 从 Release 下载 tar（与面板自更新同源，tar 内含 ypagent）→ sha256 校验 →
// 解出 ypagent → multipart 推到节点 updates 目录 → 节点侧 systemd-run 瞬态单元 detached
// 执行（备份→替换→systemctl restart ypagent；瞬态单元脱离 agent 自身 cgroup——
// 服务 restart 会杀 cgroup 内全部进程，nohup/setsid 均无效，见 exp/go-backend.md M45 结论）
// → 轮询节点心跳版本确认。任务化（agent-upgrade）承载进度日志。
package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// 节点 agent 固定安装位（与 deploy/quick_start.sh 节点模式一致）。
const (
	nodeAgentDir     = "/opt/ypagent"
	nodeAgentBin     = "/opt/ypagent/ypagent"
	nodeAgentUpdates = "/opt/ypagent/updates"
	nodeAgentService = "ypagent"
	agentUpdTimeout  = 1800 // 下载+推送+切换总超时（秒）
)

// AgentUpgradePlan 节点 agent 更新检查结果。
type AgentUpgradePlan struct {
	NodeID      string `json:"nodeId"`
	AgentVer    string `json:"agentVersion"`
	Latest      string `json:"latest,omitempty"`
	Updatable   bool   `json:"updatable"`
	Reason      string `json:"reason,omitempty"`
}

// CheckAgentUpdate 检查指定节点 agent 是否有可用更新。
func (s *SelfUpdateService) CheckAgentUpdate(ctx context.Context, nodeId string) (*AgentUpgradePlan, error) {
	nodeId = normalizeNodeID(nodeId)
	if nodeId == "local" {
		return nil, svcBadReq("本机 agent 与面板同进程，随面板自更新升级")
	}
	plan := &AgentUpgradePlan{NodeID: nodeId}
	// 节点版本：ListNodes 的 version 字段
	for _, n := range s.nodes.ListNodes() {
		if n["id"] == nodeId {
			plan.AgentVer, _ = n["version"].(string)
		}
	}
	if plan.AgentVer == "" {
		plan.Reason = "节点未上报版本（旧版 agent，升级后恢复）"
		plan.Updatable = true
	}
	chk, err := s.CheckOnline(ctx)
	if err != nil {
		return nil, err
	}
	plan.Latest = chk.Latest
	if plan.Latest == "" {
		plan.Reason = "暂无可用 Release（GitHub/Gitee 均不可达）"
		plan.Updatable = false
		return plan, nil
	}
	if isDev, _ := parseVersion(plan.AgentVer); isDev {
		plan.Updatable = true
		return plan, nil
	}
	if compareVersions(plan.AgentVer, plan.Latest) >= 0 {
		plan.Updatable = false
	}
	return plan, nil
}

// UpgradeAgentTask 创建节点 agent 升级任务（返回任务 ID）。
func (s *SelfUpdateService) UpgradeAgentTask(nodeId, source string) (map[string]any, error) {
	nodeId = normalizeNodeID(nodeId)
	if nodeId == "local" {
		return nil, svcBadReq("本机 agent 与面板同进程，请走面板在线更新")
	}
	if source != "github" && source != "gitee" {
		return nil, svcBadReq("未知更新源（github|gitee）")
	}
	if s.tasks == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "任务服务不可用")
	}
	task, err := s.tasks.StartTask("agent-upgrade", "更新节点 agent（node "+nodeId+"）", nodeId+":"+source, time.Duration(agentUpdTimeout)*time.Second,
		func(ctx context.Context, logf TaskLogf) error {
			return s.runAgentUpgrade(ctx, logf, nodeId, source)
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

// runAgentUpgrade 任务体：下载校验 → 推送 → detached 切换 → 心跳确认。
func (s *SelfUpdateService) runAgentUpgrade(ctx context.Context, logf TaskLogf, nodeId, source string) error {
	// 1) 取 Release 资产（tar 内含 ypagent，与面板同包）
	logf("info", "[1/5] 获取最新 Release（源：%s）…", source)
	rel := s.fetchLatest(ctx, source)
	if !rel.Reachable || rel.AssetURL == "" {
		return fmt.Errorf("Release 不可用: %s", rel.Error)
	}
	if !strings.HasPrefix(rel.AssetURL, "https://gitee.com/") && !strings.HasPrefix(rel.AssetURL, "https://github.com/") &&
		!strings.HasPrefix(rel.AssetURL, "https://objects.githubusercontent.com/") {
		return fmt.Errorf("资产地址域名不在白名单: %s", rel.AssetURL)
	}
	logf("info", "目标版本 %s（%s）", rel.Version, rel.AssetURL)

	// 2) core 侧下载 + sha256 校验 + 解出 ypagent
	logf("info", "[2/5] 下载并校验…")
	tmpDir, err := os.MkdirTemp("", "yp-agent-upd-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	tarPath := filepath.Join(tmpDir, "ypanel-linux.tar.gz")
	if err := httpDownload(ctx, rel.AssetURL, tarPath); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	if rel.SumURL != "" {
		sumPath := filepath.Join(tmpDir, "sha256sums.txt")
		if err := httpDownload(ctx, rel.SumURL, sumPath); err == nil {
			if err := verifySha256File(tarPath, sumPath); err != nil {
				return fmt.Errorf("sha256 校验失败: %w", err)
			}
			logf("info", "tar sha256 校验通过")
		} else {
			logf("info", "sha256sums.txt 获取失败，跳过整包校验")
		}
	}
	agentBin, err := extractFromTar(tarPath, "ypagent")
	if err != nil {
		return err
	}
	sum := sha256.Sum256(agentBin)
	agentSum := hex.EncodeToString(sum[:])
	logf("info", "ypagent 就绪（%.1f MB，sha256 %.16s…）", float64(len(agentBin))/1048576, agentSum)

	// 3) 推送到节点 updates 目录
	logf("info", "[3/5] 推送到节点…")
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return err
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	if err := agentUploadFile(ctx, ac, nodeAgentUpdates, "ypagent.new", agentBin); err != nil {
		return fmt.Errorf("推送失败: %w", err)
	}

	// 4) 节点侧校验 + systemd-run 瞬态单元切换（脱离 agent 自身 cgroup，restart 不杀脚本）
	logf("info", "[4/5] 节点侧校验并切换…")
	script := fmt.Sprintf(`set -e
cd %s
echo '%s  ypagent.new' | sha256sum -c - >/dev/null || { echo YPSUMFAIL; exit 1; }
systemd-run --unit=ypagent-upd-$(date +%%s) --collect sh -c 'sleep 1; cp %s %s.bak 2>/dev/null; mv %s/ypagent.new %s; chmod 0755 %s; systemctl restart %s' >/dev/null 2>&1
echo YPOK
`, nodeAgentUpdates, agentSum, nodeAgentBin, nodeAgentBin, nodeAgentUpdates, nodeAgentBin, nodeAgentBin, nodeAgentService)
	out, code, err := execOnNode(ctx, s.nodes, nodeId, script, 60)
	if err != nil || code != 0 || !strings.Contains(out, "YPOK") {
		if strings.Contains(out, "YPSUMFAIL") {
			return fmt.Errorf("节点侧 sha256 校验失败")
		}
		return fmt.Errorf("节点切换失败（exit %d）：%s", code, tailStr(out, 400))
	}

	// 5) 轮询心跳确认新版本生效（心跳 30s 间隔；重启后首个心跳带新版本）
	logf("info", "[5/5] 等待节点心跳确认（最长 90s）…")
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(6 * time.Second)
		for _, n := range s.nodes.ListNodes() {
			if n["id"] == nodeId {
				ver, _ := n["version"].(string)
				online, _ := n["online"].(bool)
				logf("info", "节点心跳：%s / online=%v", mapStr(ver != "", ver, "版本未知"), online)
				if online && (ver == rel.Version || ver == "") {
					logf("info", "节点 agent 更新完成：%s", mapStr(ver != "", ver, "agent 已重启（版本待下次心跳上报）"))
					return nil
				}
			}
		}
	}
	logf("info", "版本确认超时——若节点在线请稍后在节点列表复核版本（不影响已完成的切换）")
	return nil
}

// httpDownload 下载 URL 到文件（域名白名单由调用方负责）。
func httpDownload(ctx context.Context, url, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 15 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, resp.Body)
	return err
}

// verifySha256File 按 sha256sums.txt 校验文件（行格式：<hex>  <name>，取首行匹配文件名的）。
func verifySha256File(path, sumFile string) error {
	b, err := os.ReadFile(sumFile)
	if err != nil {
		return err
	}
	want := ""
	base := filepath.Base(path)
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.HasSuffix(fields[1], base) {
			want = strings.ToLower(fields[0])
			break
		}
	}
	if want == "" {
		return fmt.Errorf("sums 文件中无 %s 条目", base)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("sha256 不匹配（got %s… want %s…）", got[:16], want[:16])
	}
	return nil
}

// extractFromTar 从 tar.gz 中提取指定文件（包根一级），返回内容字节。
func extractFromTar(tarPath, name string) ([]byte, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("包内缺少 %s", name)
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == name && hdr.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}
}

// agentUploadFile multipart 推文件到 agent（落 <dir>/<filename>，agent 侧自动建目录）。
func agentUploadFile(ctx context.Context, ac *agentclient.Client, dir, filename string, content []byte) error {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		defer func() { _ = pw.Close() }()
		part, err := mw.CreateFormFile("file", filename)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if _, err := part.Write(content); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_ = mw.Close()
	}()
	req, err := ac.NewRequest(ctx, http.MethodPost,
		"/agent/v1/files/upload?path="+escapeURL2(dir), pr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := ac.HTTP.Do(req)
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
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent HTTP %d", resp.StatusCode)
	}
	return nil
}
