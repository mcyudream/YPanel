package dockerx

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/client"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// 容器内文件操作：tar 归档接口（list/read/write/mkdir/upload/download）为主，
// rename/delete/chmod 用容器内 exec 命令数组（无 shell，免注入）。
// 容器路径为 POSIX 语义，统一用 path 包处理；归档 API 以容器根锚定，无宿主穿越面。

// containerReadLimit 单次读取上限，与宿主 files.readLimit 对齐（1 MiB 截断）。
const containerReadLimit = 1 << 20

var chmodModePattern = regexp.MustCompile(`^[0-7]{3,4}$`)

// ContainerFileList 列目录：CopyFrom 解 tar 头，只保留第一层子项。
func (m *Manager) ContainerFileList(ctx context.Context, id, dir string) ([]dto.FileEntry, error) {
	if !m.Available() {
		return nil, errs.ErrAgentDisabled
	}
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	dir = path.Clean(dir)
	res, err := cli.CopyFromContainer(ctx, id, client.CopyFromContainerOptions{SourcePath: dir})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "读取容器目录失败: "+err.Error())
	}
	defer func() { _ = res.Content.Close() }()

	tr := tar.NewReader(res.Content)
	entries := []dto.FileEntry{}
	rootPrefix := ""
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errs.Wrapc(errs.CodeFileOpFailed, "解析容器目录失败: "+err.Error())
		}
		// 归一化：剥 "./"、前导与尾部 "/"（docker 的目录条目名带尾部斜杠）
		name := strings.Trim(strings.TrimPrefix(strings.TrimPrefix(h.Name, "./"), "/"), "/")
		if rootPrefix == "" {
			// 首个条目即所请求目录本身，确定 tar 内前缀
			rootPrefix = name
			continue
		}
		rel := name
		if rootPrefix != "" {
			rel = strings.TrimPrefix(name, rootPrefix+"/")
		}
		if rel == "" || strings.Contains(rel, "/") {
			continue // 更深层条目跳过
		}
		entries = append(entries, fileEntryFromTarHeader(h, path.Join(dir, rel)))
	}
	return entries, nil
}

func fileEntryFromTarHeader(h *tar.Header, fullPath string) dto.FileEntry {
	owner, group := h.Uname, h.Gname
	if owner == "" {
		owner = strconv.Itoa(h.Uid)
	}
	if group == "" {
		group = strconv.Itoa(h.Gid)
	}
	return dto.FileEntry{
		Name:    path.Base(fullPath),
		Path:    fullPath,
		IsDir:   h.Typeflag == tar.TypeDir,
		Size:    h.Size,
		Mode:    h.FileInfo().Mode().String(),
		ModeOct: fmt.Sprintf("%04o", h.Mode&0o7777),
		Owner:   owner,
		Group:   group,
		ModTime: h.ModTime,
		Target:  h.Linkname,
	}
}

// ContainerFileRead 读文件（≤1MiB 截断，返回原始字节）。
func (m *Manager) ContainerFileRead(ctx context.Context, id, filePath string) ([]byte, bool, error) {
	if !m.Available() {
		return nil, false, errs.ErrAgentDisabled
	}
	cli, err := m.getClient()
	if err != nil {
		return nil, false, err
	}
	res, err := cli.CopyFromContainer(ctx, id, client.CopyFromContainerOptions{SourcePath: path.Clean(filePath)})
	if err != nil {
		return nil, false, errs.Wrapc(errs.CodeFileOpFailed, "读取容器文件失败: "+err.Error())
	}
	defer func() { _ = res.Content.Close() }()

	tr := tar.NewReader(res.Content)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, false, errs.Wrapc(errs.CodeFileOpFailed, "容器内文件不存在")
		}
		if err != nil {
			return nil, false, errs.Wrapc(errs.CodeFileOpFailed, "解析容器文件失败: "+err.Error())
		}
		if h.Typeflag == tar.TypeDir {
			continue
		}
		buf := make([]byte, containerReadLimit+1)
		n, rerr := io.ReadFull(tr, buf)
		if rerr != nil && rerr != io.ErrUnexpectedEOF && rerr != io.EOF {
			return nil, false, errs.Wrapc(errs.CodeFileOpFailed, "读取内容失败: "+rerr.Error())
		}
		truncated := n > containerReadLimit
		if truncated {
			n = containerReadLimit
		}
		return buf[:n], truncated, nil
	}
}

// ContainerFileWrite 写文件（保留既有权限；tar 写回父目录，UTF-8 直写）。
func (m *Manager) ContainerFileWrite(ctx context.Context, id, filePath string, content []byte) error {
	if !m.Available() {
		return errs.ErrAgentDisabled
	}
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	filePath = path.Clean(filePath)
	mode := int64(0o644)
	if st, err := cli.ContainerStatPath(ctx, id, client.ContainerStatPathOptions{Path: filePath}); err == nil {
		mode = int64(st.Stat.Mode) & 0o7777
	}
	var buf bytes.Buffer
	writeTarFile(&buf, path.Base(filePath), content, mode)
	if _, err := cli.CopyToContainer(ctx, id, client.CopyToContainerOptions{
		DestinationPath: path.Dir(filePath),
		Content:         &buf,
	}); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "写回容器失败: "+err.Error())
	}
	return nil
}

// ContainerFileMkdir 递归建目录（嵌套 tar 目录条目，锚定容器根）。
func (m *Manager) ContainerFileMkdir(ctx context.Context, id, dirPath string) error {
	if !m.Available() {
		return errs.ErrAgentDisabled
	}
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	dirPath = strings.Trim(path.Clean(dirPath), "/")
	if dirPath == "" || dirPath == "." {
		return nil // 根目录恒存在
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	rel := ""
	for _, seg := range strings.Split(dirPath, "/") {
		if seg == "" {
			continue
		}
		rel = path.Join(rel, seg)
		if err := tw.WriteHeader(&tar.Header{
			Name: rel + "/", Typeflag: tar.TypeDir, Mode: 0o755, ModTime: time.Now(),
		}); err != nil {
			return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
		}
	}
	if err := tw.Close(); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	if _, err := cli.CopyToContainer(ctx, id, client.CopyToContainerOptions{
		DestinationPath: "/",
		Content:         &buf,
	}); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "容器内建目录失败: "+err.Error())
	}
	return nil
}

// ContainerFileUpload 上传单文件（内容字节 tar 写入目录，重名覆盖）。
func (m *Manager) ContainerFileUpload(ctx context.Context, id, dir, name string, content []byte) error {
	if !m.Available() {
		return errs.ErrAgentDisabled
	}
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	writeTarFile(&buf, path.Base(name), content, 0o644)
	if _, err := cli.CopyToContainer(ctx, id, client.CopyToContainerOptions{
		DestinationPath: path.Clean(dir),
		Content:         &buf,
	}); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "上传到容器失败: "+err.Error())
	}
	return nil
}

// ContainerFileDownload 流式下载文件内容（解 tar 取首个条目，不整载内存）。
func (m *Manager) ContainerFileDownload(ctx context.Context, id, filePath string, w io.Writer) error {
	if !m.Available() {
		return errs.ErrAgentDisabled
	}
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	res, err := cli.CopyFromContainer(ctx, id, client.CopyFromContainerOptions{SourcePath: path.Clean(filePath)})
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "下载容器文件失败: "+err.Error())
	}
	defer func() { _ = res.Content.Close() }()

	tr := tar.NewReader(res.Content)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return errs.Wrapc(errs.CodeFileOpFailed, "容器内文件不存在")
		}
		if err != nil {
			return errs.Wrapc(errs.CodeFileOpFailed, "解析容器文件失败: "+err.Error())
		}
		if h.Typeflag == tar.TypeDir {
			continue
		}
		_, err = io.CopyN(w, tr, h.Size)
		return err
	}
}

// ContainerExecRun 容器内无 TTY 缓冲 exec（命令数组，无 shell 免注入）。返回退出码与合并输出。
func (m *Manager) ContainerExecRun(ctx context.Context, id string, cmd []string, timeout time.Duration) (int, string, error) {
	if !m.Available() {
		return -1, "", errs.ErrAgentDisabled
	}
	cli, err := m.getClient()
	if err != nil {
		return -1, "", err
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	created, err := cli.ExecCreate(ctx, id, client.ExecCreateOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return -1, "", errs.Wrapc(errs.CodeFileOpFailed, "exec 创建失败: "+err.Error())
	}
	attach, err := cli.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return -1, "", errs.Wrapc(errs.CodeFileOpFailed, "exec attach 失败: "+err.Error())
	}
	defer attach.Close()

	var out bytes.Buffer
	// TTY=false 时 stdout/stderr 多路复用，需 demux
	_, _ = stdcopy.StdCopy(&out, &out, attach.Reader)

	for {
		ins, ierr := cli.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
		if ierr != nil {
			return -1, out.String(), errs.Wrapc(errs.CodeFileOpFailed, "exec 查询失败: "+ierr.Error())
		}
		if !ins.Running {
			return ins.ExitCode, out.String(), nil
		}
		select {
		case <-ctx.Done():
			return -1, out.String(), errs.Wrapc(errs.CodeFileOpFailed, "容器内命令超时")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// ContainerFileRename 容器内重命名/移动（exec mv）。
func (m *Manager) ContainerFileRename(ctx context.Context, id, from, to string) error {
	code, out, err := m.ContainerExecRun(ctx, id, []string{"mv", from, to}, 30*time.Second)
	return containerFileExecErr("重命名", code, out, err)
}

// ContainerFileDelete 容器内删除（exec rm -rf）。
func (m *Manager) ContainerFileDelete(ctx context.Context, id string, paths []string) error {
	for _, p := range paths {
		code, out, err := m.ContainerExecRun(ctx, id, []string{"rm", "-rf", "--", p}, 60*time.Second)
		if err := containerFileExecErr("删除", code, out, err); err != nil {
			return err
		}
	}
	return nil
}

// ContainerFileChmod 容器内权限修改（exec chmod，3-4 位八进制）。
func (m *Manager) ContainerFileChmod(ctx context.Context, id, filePath, mode string) error {
	if !chmodModePattern.MatchString(mode) {
		return errs.Wrapc(errs.CodeBadRequest, "权限格式错误（3-4 位八进制）")
	}
	code, out, err := m.ContainerExecRun(ctx, id, []string{"chmod", mode, filePath}, 30*time.Second)
	return containerFileExecErr("权限修改", code, out, err)
}

func containerFileExecErr(op string, code int, out string, err error) error {
	if err != nil {
		return err
	}
	if code != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, fmt.Sprintf("容器内%s失败(exit %d): %s", op, code, strings.TrimSpace(out)))
	}
	return nil
}

func writeTarFile(w io.Writer, name string, content []byte, mode int64) error {
	tw := tar.NewWriter(w)
	if err := tw.WriteHeader(&tar.Header{
		Name:    name,
		Mode:    mode,
		Size:    int64(len(content)),
		ModTime: time.Now(),
	}); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}
	return tw.Close()
}
