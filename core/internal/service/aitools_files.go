// aitools_files.go AI 工具：主机文件管理（M31）。
package service

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ypanel/shared/dto"
)

// aiToolsFiles 文件管理工具集。
func (s *AIService) aiToolsFiles(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_files", Module: aiModFiles, Risk: aiRiskRead,
			Desc: "浏览主机目录。input JSON：{\"path\":\"/var/log\",\"search\":\"可选名称关键词\"}",
			Parameters: schObj(map[string]any{
				"path": schStr("目录绝对路径，默认 /"), "search": schStr("名称关键词过滤"),
			}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Path   string `json:"path"`
					Search string `json:"search"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Path == "" {
					p.Path = "/"
				}
				out, err := agentGetJSON[dto.FileListResp](s, ctx, "/agent/v1/files/list?path="+url.QueryEscape(p.Path))
				if err != nil {
					return "", err
				}
				entries := []dto.FileEntry{}
				if out != nil {
					entries = out.Entries
				}
				filtered := make([]dto.FileEntry, 0, len(entries))
				for _, e := range entries {
					if matchAny(p.Search, e.Name) {
						filtered = append(filtered, e)
					}
				}
				return toolOut(map[string]any{"path": p.Path, "total": len(filtered), "entries": filtered})
			},
		},
		{
			Name: "search_files", Module: aiModFiles, Risk: aiRiskRead,
			Desc: "按名称在目录树下搜索文件（最多返回 100 条）。input JSON：{\"dir\":\"起始目录\",\"keyword\":\"文件名关键词\"}",
			Parameters: schObj(map[string]any{
				"dir": schStr("起始目录，默认 /"), "keyword": schStr("文件名关键词"),
			}, "keyword"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Dir     string `json:"dir"`
					Keyword string `json:"keyword"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Dir == "" {
					p.Dir = "/"
				}
				out, err := agentGetJSON[[]dto.FileEntry](s, ctx,
					"/agent/v1/files/search?dir="+url.QueryEscape(p.Dir)+"&keyword="+url.QueryEscape(p.Keyword))
				if err != nil {
					return "", err
				}
				items := []dto.FileEntry{}
				if out != nil {
					items = *out
				}
				return toolOut(map[string]any{"total": len(items), "items": items})
			},
		},
		{
			Name: "read_file", Module: aiModFiles, Risk: aiRiskRead,
			Desc: "读取主机文本文件内容（超长自动截断，二进制文件会提示）。input JSON：{\"path\":\"/etc/hosts\"}",
			Parameters: schObj(map[string]any{"path": schStr("文件绝对路径")}, "path"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Path string `json:"path"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Path == "" {
					return "", fmt.Errorf("缺少 path")
				}
				out, err := agentGetJSON[dto.FileReadResp](s, ctx, "/agent/v1/files/read?path="+url.QueryEscape(p.Path))
				if err != nil {
					return "", err
				}
				content := truncText(out.Content, 24000)
				if out.Truncated {
					content += fmt.Sprintf("\n…(文件共 %d 字节，已截断)", out.Size)
				}
				return content, nil
			},
		},
		{
			Name: "write_file", Module: aiModFiles, Risk: aiRiskWrite,
			Desc: "写入/覆盖主机文本文件（整个文件内容替换；父目录需已存在）。input JSON：{\"path\":\"绝对路径\",\"content\":\"文本内容\"}",
			Parameters: schObj(map[string]any{
				"path": schStr("文件绝对路径"), "content": schStr("完整文件文本内容"),
			}, "path", "content"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[dto.FileWriteReq](input)
				if err != nil {
					return "", err
				}
				if _, err := agentPostJSON[dto.FileWriteReq, any](s, ctx, "/agent/v1/files/write", &p); err != nil {
					return "", err
				}
				return "已写入: " + p.Path + fmt.Sprintf("（%d 字符）", len(p.Content)), nil
			},
		},
		{
			Name: "make_dir", Module: aiModFiles, Risk: aiRiskWrite,
			Desc: "创建目录（递归）。input JSON：{\"path\":\"/opt/newdir\"}",
			Parameters: schObj(map[string]any{"path": schStr("目录绝对路径")}, "path"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[dto.FileMkdirReq](input)
				if err != nil {
					return "", err
				}
				if _, err := agentPostJSON[dto.FileMkdirReq, any](s, ctx, "/agent/v1/files/mkdir", &p); err != nil {
					return "", err
				}
				return "已创建目录: " + p.Path, nil
			},
		},
		{
			Name: "rename_path", Module: aiModFiles, Risk: aiRiskWrite,
			Desc: "重命名/移动文件或目录。input JSON：{\"from\":\"原路径\",\"to\":\"新路径\"}",
			Parameters: schObj(map[string]any{
				"from": schStr("原绝对路径"), "to": schStr("新绝对路径"),
			}, "from", "to"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[dto.FileRenameReq](input)
				if err != nil {
					return "", err
				}
				if _, err := agentPostJSON[dto.FileRenameReq, any](s, ctx, "/agent/v1/files/rename", &p); err != nil {
					return "", err
				}
				return "已重命名: " + p.From + " → " + p.To, nil
			},
		},
		{
			Name: "copy_path", Module: aiModFiles, Risk: aiRiskWrite,
			Desc: "复制文件/目录（递归）。input JSON：{\"from\":\"源\",\"to\":\"目标\",\"overwrite\":false}",
			Parameters: schObj(map[string]any{
				"from": schStr("源绝对路径"), "to": schStr("目标绝对路径"), "overwrite": schBool("目标存在时覆盖"),
			}, "from", "to"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[dto.FileCopyReq](input)
				if err != nil {
					return "", err
				}
				if _, err := agentPostJSON[dto.FileCopyReq, any](s, ctx, "/agent/v1/files/copy", &p); err != nil {
					return "", err
				}
				return "已复制: " + p.From + " → " + p.To, nil
			},
		},
		{
			Name: "compress_files", Module: aiModFiles, Risk: aiRiskWrite,
			Desc: "打包压缩文件/目录为 tar.gz。input JSON：{\"srcs\":[\"路径\",...],\"dest\":\"/path/out.tar.gz\"}",
			Parameters: schObj(map[string]any{
				"srcs": schArr("源路径列表", schStr("绝对路径")), "dest": schStr("输出 tar.gz 绝对路径"),
			}, "srcs", "dest"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Srcs []string `json:"srcs"`
					Dest string   `json:"dest"`
				}](input)
				if err != nil {
					return "", err
				}
				if len(p.Srcs) == 0 {
					return "", fmt.Errorf("缺少 srcs")
				}
				body := dto.FileCompressReq{Srcs: p.Srcs, Dest: p.Dest}
				if _, err := agentPostJSON[dto.FileCompressReq, any](s, ctx, "/agent/v1/files/compress", &body); err != nil {
					return "", err
				}
				return "已压缩: " + fmt.Sprint(p.Srcs) + " → " + p.Dest, nil
			},
		},
		{
			Name: "decompress_file", Module: aiModFiles, Risk: aiRiskWrite,
			Desc: "解压 tar.gz/zip 等压缩包到目标目录。input JSON：{\"archive\":\"压缩包路径\",\"destDir\":\"目标目录\"}",
			Parameters: schObj(map[string]any{
				"archive": schStr("压缩包绝对路径"), "destDir": schStr("解压目标目录"),
			}, "archive", "destDir"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[dto.FileDecompressReq](input)
				if err != nil {
					return "", err
				}
				if _, err := agentPostJSON[dto.FileDecompressReq, any](s, ctx, "/agent/v1/files/decompress", &p); err != nil {
					return "", err
				}
				return "已解压: " + p.Archive + " → " + p.DestDir, nil
			},
		},
		{
			Name: "change_perms", Module: aiModFiles, Risk: aiRiskDanger,
			Desc: "修改文件/目录权限（recursive 时影响整个子树，错误权限可能导致服务故障，会先向用户确认）。input JSON：{\"path\":\"路径\",\"mode\":\"0644\",\"recursive\":false}",
			Parameters: schObj(map[string]any{
				"path": schStr("路径"), "mode": schStr("八进制权限如 0644/0755"), "recursive": schBool("递归应用到子树"),
			}, "path", "mode"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[dto.FileChmodReq](input)
				if err != nil {
					return "", err
				}
				if _, err := agentPostJSON[dto.FileChmodReq, any](s, ctx, "/agent/v1/files/chmod", &p); err != nil {
					return "", err
				}
				return fmt.Sprintf("已修改权限: %s → %s", p.Path, p.Mode), nil
			},
		},
		{
			Name: "delete_paths", Module: aiModFiles, Risk: aiRiskDanger,
			Desc: "删除文件/目录（目录递归删除，不可恢复，会先向用户确认）。input JSON：{\"paths\":[\"/path/a\",\"/path/b\"]}",
			Parameters: schObj(map[string]any{
				"paths": schArr("要删除的绝对路径列表", schStr("绝对路径")),
			}, "paths"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[dto.FileDeleteReq](input)
				if err != nil {
					return "", err
				}
				if len(p.Paths) == 0 {
					return "", fmt.Errorf("缺少 paths")
				}
				if _, err := agentPostJSON[dto.FileDeleteReq, any](s, ctx, "/agent/v1/files/delete", &p); err != nil {
					return "", err
				}
				return "已删除: " + fmt.Sprint(p.Paths), nil
			},
		},
	}
}
