package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
	"gorm.io/gorm"
)

const (
	composeUpdateManual = "compose-pull-update"
	composeUpdateAuto   = "compose-auto-update"
)

var composeUpdateTaskTypes = []string{composeUpdateManual, composeUpdateAuto}

// NormalizeComposeUpdateTarget 同时供 HTTP 和 AI 使用，实际执行节点必须经过调用者节点范围校验。
func NormalizeComposeUpdateTarget(ctx context.Context, req *dto.ComposePullUpdateReq) error {
	req.NodeID = normalizeNodeID(strings.TrimSpace(req.NodeID))
	req.Project = strings.TrimSpace(req.Project)
	name := strings.TrimSpace(req.Name)
	if req.Project == "" {
		req.Project = name
	} else if name != "" && name != req.Project {
		return svcBadReq("项目参数 name 与 project 不一致")
	}
	if !src2NameRe.MatchString(req.Project) {
		return svcBadReq("请选择有效的编排项目（名称仅允许字母、数字、中划线、下划线，最多 64 个字符）")
	}
	if req.Dir != "" && (!path.IsAbs(req.Dir) || strings.ContainsAny(req.Dir, "\x00\r\n")) {
		return svcBadReq("编排目录必须是有效的绝对路径")
	}
	if caller, ok := rbac.CallerFrom(ctx); ok && !caller.AllNodes {
		if _, allowed := caller.NodeSet[req.NodeID]; !allowed {
			return errs.New(errs.CodeForbidden, "error.nodeForbidden", "无该节点的操作权限")
		}
	}
	return nil
}

func normalizeComposeUpdateHooks(req *dto.ComposePullUpdateReq) error {
	if req.HookFailurePolicy == "" {
		req.HookFailurePolicy = "stop"
	}
	if req.HookFailurePolicy != "stop" && req.HookFailurePolicy != "continue" {
		return svcBadReq("脚本失败策略只能选择中止或继续")
	}
	if utf8.RuneCountInString(req.PreHook) > 8192 || utf8.RuneCountInString(req.PostHook) > 8192 || strings.ContainsRune(req.PreHook+req.PostHook, '\x00') {
		return svcBadReq("更新脚本最多 8192 个字符，且不能包含空字符")
	}
	return nil
}

// composeExec 的 HTTP 成功仅代表命令已执行；非零退出码和超时必须单独判断。
func (s *Src2ComposeService) composeExec(ctx context.Context, nodeID, command string, timeout int) (string, error) {
	return s.composeExecRequest(ctx, nodeID, dto.ExecReq{Command: command, TimeoutSecs: timeout})
}

func (s *Src2ComposeService) composeExecRequest(ctx context.Context, nodeID string, req dto.ExecReq) (string, error) {
	ac, err := s.agentClientFor(nodeID)
	if err != nil {
		return "", err
	}
	res, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&req)
	if err != nil {
		return "", err
	}
	if res.TimedOut {
		return res.Output, fmt.Errorf("命令执行超时（上限 %d 秒）", req.TimeoutSecs)
	}
	if res.ExitCode != 0 {
		return res.Output, fmt.Errorf("命令退出码 %d：%s", res.ExitCode, tail(res.Output, 600))
	}
	return res.Output, nil
}

type composeUpdateImage struct {
	Service     string `json:"service"`
	Image       string `json:"image"`
	ID          string `json:"id"`
	ConfigFiles string `json:"configFiles"`
	WorkDir     string `json:"workDir"`
	OneOff      string `json:"oneOff"`
	ConfigHash  string `json:"configHash"`
}

// 只采集运行容器的必要字段，不读取/记录含密码的完整 inspect 输出。
func (s *Src2ComposeService) composeRunningImages(ctx context.Context, req dto.ComposePullUpdateReq) ([]composeUpdateImage, error) {
	format := `{"service":{{json (index .Config.Labels "com.docker.compose.service")}},"image":{{json .Config.Image}},"id":{{json .Image}},"configFiles":{{json (index .Config.Labels "com.docker.compose.project.config_files")}},"workDir":{{json (index .Config.Labels "com.docker.compose.project.working_dir")}},"oneOff":{{json (index .Config.Labels "com.docker.compose.oneoff")}},"configHash":{{json (index .Config.Labels "com.docker.compose.config-hash")}}}`
	command := "ids=$(docker ps -q --filter " + shellQuote("label=com.docker.compose.project="+req.Project) + ") || exit $?\n" +
		"if [ -n \"$ids\" ]; then docker inspect --format " + shellQuote(format) + " $ids; fi"
	out, err := s.composeExec(ctx, req.NodeID, command, 60)
	if err != nil {
		return nil, err
	}
	images := make([]composeUpdateImage, 0)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var image composeUpdateImage
		if err := json.Unmarshal([]byte(line), &image); err != nil {
			return nil, fmt.Errorf("无法读取运行容器的镜像信息: %w", err)
		}
		if strings.EqualFold(image.OneOff, "true") {
			continue
		}
		if !src2NameRe.MatchString(image.Service) || image.Image == "" || image.ID == "" {
			return nil, svcBadReq("运行容器缺少有效的编排服务或镜像信息")
		}
		images = append(images, image)
	}
	return images, nil
}

func composeUpdateServices(images []composeUpdateImage) []string {
	set := map[string]bool{}
	for _, image := range images {
		set[image.Service] = true
	}
	services := make([]string, 0, len(set))
	for service := range set {
		services = append(services, service)
	}
	sort.Strings(services)
	return services
}

func quoteComposeArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuote(arg)
	}
	return strings.Join(quoted, " ")
}

func (s *Src2ComposeService) composeUpdateCommand(ctx context.Context, req dto.ComposePullUpdateReq, images []composeUpdateImage) (string, error) {
	ac, err := s.agentClientFor(req.NodeID)
	if err != nil {
		return "", err
	}
	query := url.Values{"name": {req.Project}, "dir": {req.Dir}}
	cfg, err := agentclient.GetJSON[dto.ComposeConfigResp](ac, ctx, "/agent/v1/compose/config?"+query.Encode())
	if err != nil {
		return "", err
	}
	files := []string{cfg.File}
	dir := cfg.Dir
	// config_files 保留多文件编排的覆盖顺序，working_dir 可能不等于首个配置文件目录。
	if len(images) > 0 {
		if images[0].ConfigFiles != "" {
			files = strings.Split(images[0].ConfigFiles, ",")
		}
		if images[0].WorkDir != "" {
			dir = images[0].WorkDir
		}
	}
	if req.Dir != "" {
		dir = req.Dir
	}
	if !path.IsAbs(dir) || strings.ContainsAny(dir, "\x00\r\n") {
		return "", svcBadReq("无法确定编排项目的工作目录")
	}
	args := []string{"docker", "compose", "--project-name", req.Project, "--project-directory", dir}
	for _, file := range files {
		file = strings.TrimSpace(file)
		if !path.IsAbs(file) || strings.ContainsAny(file, "\x00\r\n") {
			return "", svcBadReq("无法确定编排项目的配置文件路径")
		}
		args = append(args, "--file", file)
	}
	return "cd " + shellQuote(dir) + " && " + quoteComposeArgs(args), nil
}

// runComposeUpdate 以运行容器实际镜像 ID 对比 pull 后标签 ID；仅重建有变化的运行服务。
func (s *Src2ComposeService) runComposeUpdate(ctx context.Context, req dto.ComposePullUpdateReq, logf TaskLogf) (string, error) {
	logf("info", "读取运行服务的镜像信息…")
	before, err := s.composeRunningImages(ctx, req)
	if err != nil {
		return "", err
	}
	if len(before) == 0 {
		return "项目没有运行中的服务，已跳过", nil
	}
	base, err := s.composeUpdateCommand(ctx, req, before)
	if err != nil {
		return "", err
	}
	if err := s.checkComposeUpdateConfig(ctx, req, base, before); err != nil {
		return "", err
	}
	logf("info", "拉取运行服务的镜像（跳过本地构建服务）…")
	if _, err := s.composeExec(ctx, req.NodeID, base+" pull --ignore-buildable --policy always "+quoteComposeArgs(composeUpdateServices(before)), 1800); err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, "镜像拉取失败，已中止更新: "+err.Error())
	}
	refs := make([]string, 0)
	seen := map[string]bool{}
	for _, image := range before {
		if !seen[image.Image] {
			refs = append(refs, image.Image)
			seen[image.Image] = true
		}
	}
	out, err := s.composeExec(ctx, req.NodeID, "docker image inspect --format '{{.Id}}' -- "+quoteComposeArgs(refs), 60)
	if err != nil {
		return "", fmt.Errorf("拉取后读取镜像信息失败: %w", err)
	}
	ids := strings.Fields(out)
	if len(ids) != len(refs) {
		return "", fmt.Errorf("拉取后镜像信息不完整，已中止更新")
	}
	current := map[string]string{}
	for i, ref := range refs {
		current[ref] = ids[i]
	}
	changed := make([]composeUpdateImage, 0)
	for _, image := range before {
		if image.ID != current[image.Image] {
			changed = append(changed, image)
		}
	}
	if len(changed) == 0 {
		return "镜像已是最新，无需重建容器", nil
	}
	warnings := false
	runHook := func(command, label string) error {
		if strings.TrimSpace(command) == "" {
			return nil
		}
		logf("info", "执行%s脚本…", label)
		// 脚本通过环境传递，避免 agent 的命令摘要日志记录脚本内的凭据。
		_, err := s.composeExecRequest(ctx, req.NodeID, dto.ExecReq{
			Command: `printf '%s\n' "$YPANEL_COMPOSE_UPDATE_HOOK" | sh -e`, TimeoutSecs: 300,
			Env: map[string]string{"YPANEL_COMPOSE_UPDATE_HOOK": command},
		})
		if err != nil {
			if req.HookFailurePolicy != "continue" {
				return fmt.Errorf("%s脚本失败，已中止后续步骤: %w", label, err)
			}
			warnings = true
			logf("warn", "%s脚本失败，按设置继续: %s", label, err)
		}
		return nil
	}
	if err := runHook(req.PreHook, "更新前"); err != nil {
		return "", err
	}
	services := composeUpdateServices(changed)
	if err := s.checkComposeUpdateConfig(ctx, req, base, before); err != nil {
		return "", err
	}
	logf("info", "重建镜像有变化的服务: %s", strings.Join(services, ", "))
	if _, err := s.composeExec(ctx, req.NodeID, base+" up -d --no-deps --no-build --pull never "+quoteComposeArgs(services), 600); err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, "容器重建失败: "+err.Error())
	}
	if err := runHook(req.PostHook, "更新后"); err != nil {
		return "", fmt.Errorf("镜像更新已应用（不会自动回滚），%w", err)
	}
	result := "已更新服务: " + strings.Join(services, ", ")
	if warnings {
		result += "；脚本失败，已按设置继续（见日志）"
	}
	return result, nil
}

// 镜像更新不能顺便应用未部署的端口、挂载、环境变量或镜像标签变更。
func (s *Src2ComposeService) checkComposeUpdateConfig(ctx context.Context, req dto.ComposePullUpdateReq, base string, images []composeUpdateImage) error {
	out, err := s.composeExec(ctx, req.NodeID, base+" config --hash '*'", 60)
	if err != nil {
		return fmt.Errorf("无法核对当前编排配置，已中止镜像更新: %w", err)
	}
	hashes := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 {
			hashes[f[0]] = f[1]
		}
	}
	for _, image := range images {
		if image.ConfigHash == "" || hashes[image.Service] != image.ConfigHash {
			return fmt.Errorf("服务 %s 的本地编排配置与运行容器不一致，已中止镜像更新；请先通过应用版本升级或重新部署应用配置", image.Service)
		}
	}
	return nil
}

func (s *Src2ComposeService) startComposeUpdateTask(ctx context.Context, req dto.ComposePullUpdateReq, taskType string) (*dto.ComposeUpdateTaskResp, error) {
	if err := NormalizeComposeUpdateTarget(ctx, &req); err != nil {
		return nil, err
	}
	if err := normalizeComposeUpdateHooks(&req); err != nil {
		return nil, err
	}
	if s.Tasks == nil || s.DB == nil {
		return nil, svcBadReq("更新任务服务不可用")
	}
	if _, err := s.Nodes.ByID(req.NodeID); err != nil {
		return nil, err
	}
	task, err := s.Tasks.StartProjectTask(taskType, "更新运行镜像 "+req.Project+"（节点 "+req.NodeID+"）", req.NodeID, req.Project, 45*time.Minute,
		func(runCtx context.Context, logf TaskLogf) error {
			now := time.Now()
			where := s.DB.Model(&model.ComposeAutoUpdate{}).Where("node_id = ? AND project = ?", req.NodeID, req.Project)
			if err := where.Updates(map[string]any{"last_checked_at": now, "last_result": "正在检查镜像更新"}).Error; err != nil {
				return err
			}
			result, runErr := s.runComposeUpdate(runCtx, req, logf)
			if runErr != nil {
				result = runErr.Error()
			} else {
				logf("info", "%s", result)
			}
			if err := s.DB.Model(&model.ComposeAutoUpdate{}).Where("node_id = ? AND project = ?", req.NodeID, req.Project).
				Update("last_result", truncStr(result, 255)).Error; err != nil {
				return errors.Join(runErr, fmt.Errorf("保存更新结果失败: %w", err))
			}
			return runErr
		})
	if err != nil {
		return nil, err
	}
	return &dto.ComposeUpdateTaskResp{TaskID: task.ID, Project: req.Project}, nil
}

func (s *Src2ComposeService) PullUpdateTask(ctx context.Context, req dto.ComposePullUpdateReq) (*dto.ComposeUpdateTaskResp, error) {
	return s.startComposeUpdateTask(ctx, req, composeUpdateManual)
}

func (s *Src2ComposeService) StartAutoUpdateLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.autoUpdateTick(ctx)
			}
		}
	}()
}

func (s *Src2ComposeService) autoUpdateTick(ctx context.Context) {
	var rows []model.ComposeAutoUpdate
	if err := s.DB.Where("enabled = ?", true).Order("last_checked_at ASC, id ASC").Find(&rows).Error; err != nil {
		slog.Error("读取编排自动更新配置失败", "err", err)
		return
	}
	for _, row := range rows {
		if ctx.Err() != nil {
			return
		}
		interval := time.Duration(max(5, row.IntervalMin)) * time.Minute
		if row.LastCheckedAt != nil && time.Since(*row.LastCheckedAt) < interval {
			continue
		}
		req := dto.ComposePullUpdateReq{NodeID: row.NodeID, Project: row.Project, Dir: row.Dir, PreHook: row.PreHook, PostHook: row.PostHook, HookFailurePolicy: row.HookFailurePolicy}
		if req.HookFailurePolicy == "" {
			req.HookFailurePolicy = "continue" // 迁移前的存量脚本保持原有失败策略
		}
		if _, err := s.startComposeUpdateTask(ctx, req, composeUpdateAuto); err != nil {
			var businessErr *errs.Error
			if !errors.As(err, &businessErr) || businessErr.Code != errs.CodeConflict {
				slog.Warn("调度编排自动更新失败", "node", row.NodeID, "project", row.Project, "err", err)
			}
			continue
		}
		break // 每分钟调度一个到期项目，按上次检查时间排序避免饥饿
	}
}

func (s *Src2ComposeService) GetAutoUpdate(ctx context.Context, nodeID, project string) (*dto.ComposeAutoUpdateResp, error) {
	req := dto.ComposePullUpdateReq{NodeID: nodeID, Project: project}
	if err := NormalizeComposeUpdateTarget(ctx, &req); err != nil {
		return nil, err
	}
	var row model.ComposeAutoUpdate
	err := s.DB.Where("node_id = ? AND project = ?", req.NodeID, req.Project).First(&row).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	out := &dto.ComposeAutoUpdateResp{NodeID: req.NodeID, Project: req.Project, IntervalMin: 60, HookFailurePolicy: "stop"}
	if err == nil {
		out.Exists, out.Enabled, out.IntervalMin = true, row.Enabled, row.IntervalMin
		out.Dir = row.Dir
		out.PreHook, out.PostHook, out.Remark = row.PreHook, row.PostHook, row.Remark
		out.HookFailurePolicy = row.HookFailurePolicy
		if out.HookFailurePolicy == "" {
			out.HookFailurePolicy = "continue"
		}
		out.LastCheckedAt, out.LastResult = row.LastCheckedAt, row.LastResult
	}
	var task model.AppTask
	err = s.DB.Where("type IN ? AND ref = ?", composeUpdateTaskTypes, req.NodeID+":"+req.Project).Order("id DESC").First(&task).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		out.LastTaskID, out.LastStatus = task.ID, task.Status
		out.LastCheckedAt = &task.CreatedAt
		if task.Status == TaskRunning {
			out.LastResult = "正在检查镜像更新"
		} else if task.Status == TaskFailed {
			out.LastResult = task.Error
		} else if out.LastResult == "" {
			out.LastResult = "执行完成，详情见任务日志"
			// 未保存自动更新配置时，手动更新的实际摘要保存在任务最后一条日志。
			if line := strings.TrimSpace(task.LogText); line != "" {
				line = line[strings.LastIndex(line, "\n")+1:]
				if _, message, ok := strings.Cut(line, " [INFO] "); ok {
					out.LastResult = message
				}
			}
		}
	}
	if out.Enabled && out.LastCheckedAt != nil {
		next := out.LastCheckedAt.Add(time.Duration(out.IntervalMin) * time.Minute)
		out.NextCheckAt = &next
	}
	return out, nil
}

func (s *Src2ComposeService) SaveAutoUpdate(ctx context.Context, req dto.ComposeAutoUpdateSaveReq) error {
	if err := NormalizeComposeUpdateTarget(ctx, &req.ComposePullUpdateReq); err != nil {
		return err
	}
	if err := normalizeComposeUpdateHooks(&req.ComposePullUpdateReq); err != nil {
		return err
	}
	if req.IntervalMin < 5 || req.IntervalMin > 1440 {
		return svcBadReq("检查间隔必须是 5–1440 分钟之间的整数")
	}
	if utf8.RuneCountInString(req.Remark) > 255 {
		return svcBadReq("备注最多 255 个字符")
	}
	if _, err := s.Nodes.ByID(req.NodeID); err != nil {
		return err
	}
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	var row model.ComposeAutoUpdate
	err := s.DB.Where("node_id = ? AND project = ?", req.NodeID, req.Project).First(&row).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = model.ComposeAutoUpdate{NodeID: req.NodeID, Project: req.Project, Dir: req.Dir, Enabled: req.Enabled, IntervalMin: req.IntervalMin,
			PreHook: req.PreHook, PostHook: req.PostHook, HookFailurePolicy: req.HookFailurePolicy, Remark: req.Remark}
		return s.DB.Create(&row).Error
	}
	// 仅更新配置列，避免覆盖正在运行的任务写回的检查时间/结果；map 保留 false 和空脚本。
	return s.DB.Model(&row).Updates(map[string]any{"dir": req.Dir, "enabled": req.Enabled, "interval_min": req.IntervalMin,
		"pre_hook": req.PreHook, "post_hook": req.PostHook, "hook_failure_policy": req.HookFailurePolicy, "remark": req.Remark}).Error
}
