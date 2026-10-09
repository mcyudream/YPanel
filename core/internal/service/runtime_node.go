// runtime_node.go Node 运行时模块管理（package.json 依赖查看 + npm/yarn/pnpm 增删）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ypanel/shared/errs"
)

// NodeModuleInfo package.json 依赖项。
type NodeModuleInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dev     bool   `json:"dev"`
}

// NodeModulesResp node_modules 视图。
type NodeModulesResp struct {
	Modules   []NodeModuleInfo `json:"modules"`
	PkgMgr    string           `json:"pkgMgr"`
	PackageJS bool             `json:"packageJson"`
}

var nodePkgNameRe = regexp.MustCompile(`^(@[a-z0-9-~][a-z0-9-._~]*/)?[a-z0-9-~][a-z0-9-._~]{0,214}$`)

// NodeModules 列出 package.json 依赖（deps + devDeps）。
func (s *RuntimeService) NodeModules(ctx context.Context, id uint) (*NodeModulesResp, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Type != "node" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "仅 Node 运行时支持模块管理")
	}
	env := envJSON(row)
	resp := &NodeModulesResp{Modules: make([]NodeModuleInfo, 0, 8), PkgMgr: cmpOr(env.PkgMgr, "auto")}
	out, err := s.exec(ctx, 30, "docker exec -i %s sh -c 'cat /app/package.json'", containerNameOr(row))
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return resp, nil // 无 package.json：返回空列表
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(out.Output), &pkg); err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "package.json 解析失败: "+tailOutput(out.Output, 200))
	}
	resp.PackageJS = true
	for name, ver := range pkg.Dependencies {
		resp.Modules = append(resp.Modules, NodeModuleInfo{Name: name, Version: ver})
	}
	for name, ver := range pkg.DevDependencies {
		resp.Modules = append(resp.Modules, NodeModuleInfo{Name: name, Version: ver, Dev: true})
	}
	return resp, nil
}

// OperateNodeModule 安装/卸载/更新依赖（任务化）。
func (s *RuntimeService) OperateNodeModule(ctx context.Context, id uint, operate, pkg, pkgMgr string) (map[string]any, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Type != "node" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "仅 Node 运行时支持模块管理")
	}
	if !map[string]bool{"install": true, "uninstall": true, "update": true}[operate] {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的操作")
	}
	env := envJSON(row)
	mgr := cmpOr(pkgMgr, env.PkgMgr)
	if mgr == "auto" || mgr == "" {
		mgr = "npm"
	}
	if !map[string]bool{"npm": true, "yarn": true, "pnpm": true}[mgr] {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的包管理器: "+mgr)
	}
	sub := map[string][2]string{
		"npm":   {"install", "uninstall"},
		"yarn":  {"add", "remove"},
		"pnpm":  {"add", "remove"},
	}[mgr]
	var verb string
	switch operate {
	case "install":
		verb = sub[0]
	case "uninstall":
		verb = sub[1]
	case "update":
		verb = map[string]string{"npm": "update", "yarn": "upgrade", "pnpm": "update"}[mgr]
	}
	if pkg != "" && !nodePkgNameRe.MatchString(strings.TrimSuffix(pkg, "@latest")) {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "包名不合法: "+pkg)
	}
	pkgArg := ""
	if pkg != "" {
		pkgArg = " " + pkg
	}
	task, err := s.tasks.StartTask("runtime-node-module", fmt.Sprintf("%s 依赖 %s（%s）", operate, cmpOr(pkg, "全部"), mgr), fmt.Sprintf("runtime-%d", id), 20*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			logf("info", "执行 %s %s%s", mgr, verb, pkgArg)
			out, err := s.exec(tctx, 1200, "docker exec -i %s %s %s%s", containerNameOr(row), mgr, verb, pkgArg)
			if err != nil {
				return err
			}
			logf("info", "%s", tailOutput(out.Output, 1200))
			if out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, mgr+" "+verb+" 失败: "+tailOutput(out.Output, 600))
			}
			logf("info", "完成")
			return nil
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}
