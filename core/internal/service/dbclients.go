// 数据库外接实例执行环境：dump/导入/备份/恢复统一经临时 Docker 容器执行（--network host，
// 容器与宿主共享网络栈，external 目标无论回环还是远程 IP 直达），宿主机不再依赖任何数据库客户端工具。
// 本文件提供镜像常量、Docker 可用性与镜像缓存检测、镜像预拉取（拉取走 daemon 配置的加速器）。
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ypanel/shared/errs"
)

// dbClientImage 外接实例通道的临时容器镜像。
// mysql:8.0 客户端向下兼容 5.x 服务端；pg_dump 新版本官方支持 dump 更老服务端；redis/mongo 取稳定大版本 tag。
var dbClientImage = map[string]string{
	"mysql":    "mysql:8.0",
	"postgres": "postgres:17",
	"redis":    "redis:7",
	"mongo":    "mongo:7",
}

// dbClientImageTools 各镜像内提供的工具（前端展示用）。
var dbClientImageTools = map[string][]string{
	"mysql":    {"mysqldump", "mysql"},
	"postgres": {"pg_dump", "psql"},
	"redis":    {"redis-cli"},
	"mongo":    {"mongodump", "mongorestore"},
}

// dbToolPullPre 命令前置：镜像缺失时先拉取（报错清晰；不缺时零开销）。
func dbToolPullPre(typ string) string {
	return fmt.Sprintf("docker image inspect %[1]s >/dev/null 2>&1 || docker pull %[1]s; ", dbClientImage[typ])
}

// execEnvImage 单类型执行环境状态。
type execEnvImage struct {
	Type  string   `json:"type"`
	Image string   `json:"image"`
	Tools []string `json:"tools"`
	Ready bool     `json:"ready"` // 镜像已缓存，可直接执行
}

// execEnvStatus 外接实例执行环境状态（面板宿主机 Docker + 镜像缓存）。
type execEnvStatus struct {
	DockerOK bool           `json:"dockerOk"`
	Hint     string         `json:"hint,omitempty"`
	Images   []execEnvImage `json:"images"`
}

// ExecEnvCheck 检测面板宿主机 Docker 可用性与各类型镜像缓存状态。
func (s *DatabaseService) ExecEnvCheck(ctx context.Context) (*execEnvStatus, error) {
	st := &execEnvStatus{Images: make([]execEnvImage, 0, len(dbClientImage))}
	out, err := s.ExecAgent(ctx, "command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1 && echo YP_DOCKER_OK || echo YP_DOCKER_MISSING", nil, 60)
	if err != nil {
		return nil, err
	}
	if strings.Contains(out.Output, "YP_DOCKER_MISSING") {
		st.Hint = "面板宿主机 Docker 不可用：外接实例的备份/迁移/导入/恢复经临时容器执行，依赖 Docker"
		return st, nil
	}
	cmd := "for i in"
	byImg := map[string]string{}
	for typ, img := range dbClientImage {
		cmd += " " + img
		byImg[img] = typ
	}
	cmd += "; do if docker image inspect \"$i\" >/dev/null 2>&1; then echo \"$i=1\"; else echo \"$i=0\"; fi; done"
	out, err = s.ExecAgent(ctx, cmd, nil, 120)
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "检测失败: "+firstLine(out.Output))
	}
	ready := map[string]bool{}
	for _, line := range strings.Split(out.Output, "\n") {
		kv := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(kv) == 2 {
			ready[kv[0]] = kv[1] == "1"
		}
	}
	for _, typ := range []string{"mysql", "postgres", "redis", "mongo"} {
		img := dbClientImage[typ]
		st.Images = append(st.Images, execEnvImage{
			Type: typ, Image: img, Tools: dbClientImageTools[typ], Ready: ready[img],
		})
	}
	st.DockerOK = true
	return st, nil
}

// ExecEnvPull 预拉取指定类型的执行镜像（首次使用前缓存到本地，避免操作时等待拉取）。
func (s *DatabaseService) ExecEnvPull(ctx context.Context, typ string) (*execEnvStatus, error) {
	img, ok := dbClientImage[typ]
	if !ok {
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的实例类型: "+typ)
	}
	out, err := s.ExecAgent(ctx, "docker pull "+img, nil, 1800)
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 || out.TimedOut {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, execFailMsg("镜像拉取", out))
	}
	return s.ExecEnvCheck(ctx)
}
