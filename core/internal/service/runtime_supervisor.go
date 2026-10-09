// runtime_supervisor.go PHP 运行时容器内 supervisor 进程管理 + FPM 慢日志视图。
package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

var supProcNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// supervisorAvailable 探测容器内是否具备 supervisor（旧镜像需重建）。
func (s *RuntimeService) supervisorAvailable(ctx context.Context, row *model.Runtime) error {
	out, err := s.exec(ctx, 20, "docker exec -i %s sh -c 'command -v supervisorctl >/dev/null 2>&1 && echo ok || echo missing'", containerNameOr(row))
	if err != nil {
		return err
	}
	if strings.TrimSpace(out.Output) != "ok" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "该运行环境镜像未内置 supervisor，请先在概览中执行「重建镜像」后使用")
	}
	return nil
}

// supervisorCtl 在容器内执行 supervisorctl 并返回输出。
func (s *RuntimeService) supervisorCtl(ctx context.Context, row *model.Runtime, args string) (string, error) {
	out, err := s.exec(ctx, 60, "docker exec -i %s supervisorctl %s", containerNameOr(row), args)
	if err != nil {
		return "", err
	}
	if out.ExitCode != 0 {
		return out.Output, errs.Wrapc(errs.CodeFileOpFailed, "supervisorctl 失败: "+tailOutput(out.Output, 400))
	}
	return out.Output, nil
}

// SupervisorProcessInfo 进程条目。
type SupervisorProcessInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"` // pid/uptime 等
}

// SupervisorList 进程列表（supervisorctl status）。
func (s *RuntimeService) SupervisorList(ctx context.Context, id uint) ([]SupervisorProcessInfo, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Origin == "external" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 不支持进程管理")
	}
	if err := s.supervisorAvailable(ctx, row); err != nil {
		return nil, err
	}
	out, err := s.supervisorCtl(ctx, row, "status")
	if err != nil && out == "" {
		return nil, err
	}
	list := make([]SupervisorProcessInfo, 0, 4)
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(strings.TrimSpace(line))
		if len(f) < 2 {
			continue
		}
		info := SupervisorProcessInfo{Name: f[0], Status: f[1]}
		if len(f) > 2 {
			info.Detail = strings.Join(f[2:], " ")
		}
		list = append(list, info)
	}
	return list, nil
}

// SupervisorUpsertInput 新增/更新自定义进程。
type SupervisorUpsertInput struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	AutoStart   *bool  `json:"autoStart"`
	AutoRestart *bool  `json:"autoRestart"`
}

// SupervisorUpsert 写入 supervisor.d/<name>.ini 并 update 生效。
func (s *RuntimeService) SupervisorUpsert(ctx context.Context, id uint, in SupervisorUpsertInput) error {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return err
	}
	if row.Origin == "external" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 不支持进程管理")
	}
	if !supProcNameRe.MatchString(in.Name) {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "进程名不合法（字母/数字/中划线，≤64）")
	}
	if in.Name == "php-fpm" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "php-fpm 为内置进程，不可覆盖")
	}
	cmd := strings.TrimSpace(in.Command)
	if cmd == "" || len(cmd) > 500 || strings.ContainsAny(cmd, "\n\r") {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "启动命令不合法（非空、≤500 字符、无换行）")
	}
	autoStart, autoRestart := true, true
	if in.AutoStart != nil {
		autoStart = *in.AutoStart
	}
	if in.AutoRestart != nil {
		autoRestart = *in.AutoRestart
	}
	boolStr := map[bool]string{true: "true", false: "false"}
	ini := fmt.Sprintf(`[program:%s]
command=%s
directory=/var/www
autostart=%s
autorestart=%s
startsecs=3
stopasgroup=true
killasgroup=true
stdout_logfile=/var/log/supervisor/%s.log
stdout_logfile_maxbytes=10MB
stderr_logfile=/var/log/supervisor/%s.err.log
stderr_logfile_maxbytes=10MB
`, in.Name, cmd, boolStr[autoStart], boolStr[autoRestart], in.Name, in.Name)
	// ini 经 files/write 通道写入宿主挂载目录（不进 shell），supervisorctl 仅接收白名单校验过的进程名
	if err := s.writeRemote(ctx, s.dir(row)+"/supervisor/supervisor.d/"+in.Name+".ini", ini); err != nil {
		return err
	}
	if err := s.supervisorAvailable(ctx, row); err != nil {
		return err
	}
	if _, err := s.supervisorCtl(ctx, row, "update"); err != nil {
		return err
	}
	return nil
}

// SupervisorOperate 启动/停止/重启单个进程。
func (s *RuntimeService) SupervisorOperate(ctx context.Context, id uint, name, action string) error {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return err
	}
	if row.Origin == "external" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 不支持进程管理")
	}
	if !supProcNameRe.MatchString(name) {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "进程名不合法")
	}
	if !map[string]bool{"start": true, "stop": true, "restart": true}[action] {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的操作")
	}
	if err := s.supervisorAvailable(ctx, row); err != nil {
		return err
	}
	_, err = s.supervisorCtl(ctx, row, action+" "+name)
	return err
}

// SupervisorDelete 删除进程（删 ini + update）。
func (s *RuntimeService) SupervisorDelete(ctx context.Context, id uint, name string) error {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return err
	}
	if row.Origin == "external" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 不支持进程管理")
	}
	if !supProcNameRe.MatchString(name) {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "进程名不合法")
	}
	if name == "php-fpm" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "php-fpm 为内置进程，不可删除")
	}
	if _, err := s.exec(ctx, 30, "rm -f %s/supervisor/supervisor.d/%s.ini", s.dir(row), name); err != nil {
		return err
	}
	if err := s.supervisorAvailable(ctx, row); err != nil {
		return err
	}
	_, err = s.supervisorCtl(ctx, row, "update")
	return err
}

// SupervisorLog 自定义进程日志（容器内 /var/log/supervisor/<name>.log）。
func (s *RuntimeService) SupervisorLog(ctx context.Context, id uint, name string) (map[string]any, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if !supProcNameRe.MatchString(name) {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "进程名不合法")
	}
	out, err := s.exec(ctx, 30, "docker exec -i %s sh -c 'tail -n 200 /var/log/supervisor/%s.log 2>/dev/null; tail -n 100 /var/log/supervisor/%s.err.log 2>/dev/null'", containerNameOr(row), name, name)
	if err != nil {
		return nil, err
	}
	return map[string]any{"log": out.Output}, nil
}

// ---- FPM 慢日志 ----

// SlowLog 慢日志内容（host 侧 log/slow.log，模板已配 request_slowlog_timeout=5s）。
func (s *RuntimeService) SlowLog(ctx context.Context, id uint) (map[string]any, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Origin == "external" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 请在原主机查看慢日志")
	}
	out, err := s.exec(ctx, 20, "tail -n 400 %s/log/slow.log 2>/dev/null; true", s.dir(row))
	if err != nil {
		return nil, err
	}
	return map[string]any{"log": strings.TrimRight(out.Output, "\n"), "empty": strings.TrimSpace(out.Output) == ""}, nil
}

// SlowLogClear 清空慢日志。
func (s *RuntimeService) SlowLogClear(ctx context.Context, id uint) error {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return err
	}
	if row.Origin == "external" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 请在原主机管理慢日志")
	}
	_, err = s.exec(ctx, 20, "truncate -s 0 %s/log/slow.log 2>/dev/null; true", s.dir(row))
	return err
}
