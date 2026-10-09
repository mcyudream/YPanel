// aitools_srcbuild.go AI 工具：源码构建部署场景（git 仓库 → 预检 → 构建任务 → 跑通）（M31 二批）。
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ypanel/shared/dto"
)

// aiToolsSrcBuild 源码构建工具集（场景：用户给一个 git 仓库，AI 检测语言栈并构建部署跑通）。
func (s *AIService) aiToolsSrcBuild(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "src_build_preview", Module: aiModSrcBuild, Risk: aiRiskRead,
			Desc: "预检 git 仓库：浅克隆并检测语言栈（go/node/java/python/php/static/dockerfile）、构建目录候选与建议参数，创建构建任务前必调。input JSON：{\"gitUrl\":\"https://github.com/x/y.git\",\"branch\":\"可选\",\"credentialId\":可选(0=按 host 自动匹配凭据库)}。返回 items（dir/marker/facts：语言版本/构建命令/包管理器/模块列表）。",
			Parameters: schObj(map[string]any{
				"gitUrl": schStr("git 仓库地址（https/ssh）"), "branch": schStr("分支，空=默认分支"),
				"credentialId": schInt("凭据库凭据 ID，0=按仓库 host 自动匹配"),
			}, "gitUrl"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[dto.Src2ComposePreviewReq](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.GitURL) == "" {
					return "", fmt.Errorf("缺少 gitUrl")
				}
				var logs []string
				resp, suggestions, err := s.src2.PreviewStream(ctx, p, func(text string) {
					if len(logs) < 40 {
						logs = append(logs, text)
					}
				})
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{
					"commit": resp.Commit, "items": resp.Items,
					"suggestions": suggestions, "process": strings.Join(logs, "\n"),
				})
			},
		},
		{
			Name: "src_build_create", Module: aiModSrcBuild, Risk: aiRiskWrite,
			Desc: "创建源码构建部署任务（克隆→渲染 compose→镜像构建→up，全程 30 分钟任务，会先向用户确认参数）。input JSON：{\"name\":\"b20j\",\"gitUrl\":\"...\",\"branch\":\"可选\",\"credentialId\":0,\"services\":[{\"name\":\"web\",\"dir\":\"\",\"lang\":\"go\",\"version\":\"1.23\",\"hostPort\":3459,\"containerPort\":8080,\"startCmd\":\"可选\"}]}。lang 枚举：go/node/node-build/java-maven/java-gradle/python/php/static/dockerfile；services 按预检结果填。返回 taskId，用 get_task 跟踪。",
			Parameters: schObj(map[string]any{
				"name": schStr("项目名（小写字母/数字/中划线，如 myapp-web）"), "gitUrl": schStr("仓库地址"),
				"branch": schStr("分支"), "credentialId": schInt("凭据 ID，0=自动匹配"),
				"services": schArr("服务规格（按 src_build_preview 结果填）", schObj(map[string]any{
					"name": schStr("服务名"), "dir": schStr("仓库内构建目录，根目录为空"),
					"lang": schStr("语言栈"), "version": schStr("语言版本"),
					"hostPort": schInt("宿主端口，0=不发布"), "containerPort": schInt("容器端口"),
					"startCmd": schStr("启动命令（node/python/go）"), "module": schStr("maven 模块相对路径"),
					"buildCmd": schStr("node-build 构建命令"), "distDir": schStr("node-build 产物目录"),
				})),
			}, "name", "gitUrl", "services"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[dto.Src2ComposeBuildReq](input)
				if err != nil {
					return "", err
				}
				if len(p.Services) == 0 {
					return "", fmt.Errorf("缺少 services（先 src_build_preview 获取候选）")
				}
				out, err := s.src2.Create(ctx, p)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "构建任务已创建，用 get_task 跟踪进度", "taskId": out.TaskID, "commit": out.Commit})
			},
		},
	}
}
