// Package files 节点文件操作：浏览/读写/上传下载/增删改。
// 安全：所有对外路径经 Normalize 校验（防穿越），读取限长，删除前二次校验非系统根。
package files

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// readLimit 单次文本读取上限（1 MiB，超出截断）。
const readLimit = 1 << 20

// Manager 文件操作管理器。roots 为允许访问的根目录（默认 ["/"]）。
type Manager struct {
	roots []string
}

// New 创建管理器。
func New(roots []string) *Manager {
	if len(roots) == 0 {
		roots = []string{"/"}
	}
	return &Manager{roots: roots}
}

// Normalize 校验并规范化路径：
//   - 必须落在某个授权 root 内（root="/" 时放行全部绝对路径）
//   - 清洗 ..、重复分隔符等穿越手法
func (m *Manager) Normalize(p string) (string, error) {
	if p == "" {
		return "", errs.ErrPathInvalid
	}
	if !filepath.IsAbs(p) {
		return "", errs.Wrap(errs.ErrPathInvalid, "仅允许绝对路径")
	}
	clean := filepath.Clean(p)
	for _, root := range m.roots {
		rootClean := filepath.Clean(root)
		// 空/根授权 = 全盘管理（单机面板默认形态）
		if rootClean == "/" || rootClean == string(filepath.Separator) {
			if filepath.IsAbs(clean) {
				return clean, nil
			}
			continue
		}
		if clean == rootClean || strings.HasPrefix(clean, rootClean+string(filepath.Separator)) {
			return clean, nil
		}
	}
	return "", errs.Wrap(errs.ErrPathInvalid, "路径超出授权范围")
}

// denyPaths 高危设备/内核文件，禁止读取（防死循环/内存洪泛）。
var denyPaths = map[string]bool{
	"/proc/kcore": true, "/proc/kmem": true, "/proc/mem": true,
	"/dev/mem": true, "/dev/kmem": true, "/dev/zero": true, "/dev/random": true,
	"/dev/urandom": true, "/dev/port": true,
}

// List 列目录。
func (m *Manager) List(path string) (*dto.FileListResp, error) {
	abs, err := m.Normalize(path)
	if err != nil {
		return nil, err
	}
	if denyPaths[abs] {
		return nil, errs.New(errs.CodeFileOpFailed, "error.fileDenied", "该文件禁止访问")
	}
	ents, err := os.ReadDir(abs)
	if err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, err.Error())
	}
	out := &dto.FileListResp{Path: abs, Entries: make([]dto.FileEntry, 0, len(ents))}
	for _, e := range ents {
		entry, err := m.statEntry(filepath.Join(abs, e.Name()), e.Type()&os.ModeSymlink != 0)
		if err != nil {
			continue // 无权限等单条失败不阻塞整体
		}
		out.Entries = append(out.Entries, entry)
	}
	sort.Slice(out.Entries, func(i, j int) bool {
		a, b := out.Entries[i], out.Entries[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		return a.Name < b.Name
	})
	return out, nil
}

func (m *Manager) statEntry(path string, isSymlink bool) (dto.FileEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return dto.FileEntry{}, err
	}
	e := dto.FileEntry{
		Name:    filepath.Base(path),
		Path:    path,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		Mode:    info.Mode().String(),
		ModTime: info.ModTime(),
	}
	if isSymlink {
		if target, err := os.Readlink(path); err == nil {
			e.Target = target
		}
	}
	fillOwner(path, &e)
	return e, nil
}

// Read 读文本文件，超限截断。
func (m *Manager) Read(path string) (*dto.FileReadResp, error) {
	abs, err := m.Normalize(path)
	if err != nil {
		return nil, err
	}
	if denyPaths[abs] {
		return nil, errs.New(errs.CodeFileOpFailed, "error.fileDenied", "该文件禁止访问")
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, err.Error())
	}
	if st.IsDir() {
		return nil, errs.New(errs.CodeFileOpFailed, "error.isDirectory", "目标为目录")
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, errs.Wrap(errs.ErrForbidden, err.Error())
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, readLimit+1)
	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	truncated := n > readLimit
	if truncated {
		n = readLimit
	}
	return &dto.FileReadResp{
		Path:      abs,
		Content:   string(buf[:n]),
		Size:      st.Size(),
		Truncated: truncated,
	}, nil
}

// Write 写文件（覆盖），自动创建父目录。
func (m *Manager) Write(path, content string) error {
	abs, err := m.Normalize(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return nil
}

// Mkdir 递归建目录。
func (m *Manager) Mkdir(path string) error {
	abs, err := m.Normalize(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return nil
}

// Rename 重命名/移动。
func (m *Manager) Rename(from, to string) error {
	absFrom, err := m.Normalize(from)
	if err != nil {
		return err
	}
	absTo, err := m.Normalize(to)
	if err != nil {
		return err
	}
	if absFrom == "/" || absTo == "/" {
		return errs.Wrap(errs.ErrBadRequest, "不允许操作系统根目录")
	}
	if _, err := os.Lstat(absFrom); err != nil {
		return errs.Wrap(errs.ErrNotFound, err.Error())
	}
	if err := os.Rename(absFrom, absTo); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return nil
}

// Delete 批量删除（递归）。
func (m *Manager) Delete(paths []string) error {
	for _, p := range paths {
		abs, err := m.Normalize(p)
		if err != nil {
			return err
		}
		if abs == "/" || abs == "/usr" || abs == "/etc" || abs == "/var" || abs == "/boot" {
			return errs.Wrap(errs.ErrBadRequest, "拒绝删除系统关键目录: "+abs)
		}
		if err := os.RemoveAll(abs); err != nil {
			return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
		}
	}
	return nil
}

// Upload 接收 multipart 上传并落盘（r 已定位到文件内容流）。
func (m *Manager) Upload(path string, filename string, r io.Reader) (string, error) {
	abs, err := m.Normalize(path)
	if err != nil {
		return "", err
	}
	if filename != "" {
		abs = filepath.Join(abs, filepath.Base(filename))
		abs, err = m.Normalize(abs)
		if err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	f, err := os.Create(abs)
	if err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(f, r); err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return abs, nil
}

// Download 打开文件供流式下载。调用方负责 Close。
func (m *Manager) Download(path string) (*os.File, os.FileInfo, error) {
	abs, err := m.Normalize(path)
	if err != nil {
		return nil, nil, err
	}
	if denyPaths[abs] {
		return nil, nil, errs.New(errs.CodeFileOpFailed, "error.fileDenied", "该文件禁止访问")
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, nil, errs.Wrap(errs.ErrNotFound, err.Error())
	}
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		_ = f.Close()
		return nil, nil, errs.Wrapc(errs.CodeFileOpFailed, "目标为目录，请使用压缩下载")
	}
	return f, st, nil
}

// Archive 将 path（文件或目录）打包为 tar.gz 写入 w，用于目录下载。
func (m *Manager) Archive(path string, w io.Writer) error {
	abs, err := m.Normalize(path)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	err = filepath.Walk(abs, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(abs, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, f); err != nil {
				_ = f.Close()
				return err
			}
			_ = f.Close()
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

// fillOwner 填充属主/属组（平台相关实现在 fileinfo_*.go）。
func fillOwner(path string, e *dto.FileEntry) {
	if st, ok := infoOf(path); ok {
		e.Owner = st.owner
		e.Group = st.group
		e.ModeOct = st.modeOct
	}
}

type ownerInfo struct {
	owner, group, modeOct string
}

var ucache = map[string]string{}

func cachedName(id, name string) string {
	if name == "" || name == id {
		return id
	}
	ucache[id] = name
	return name
}

func lookupUserName(id string) string {
	if v, ok := ucache[id]; ok {
		return v
	}
	if u, err := user.LookupId(id); err == nil {
		return cachedName(id, u.Username)
	}
	return id
}

func lookupGroupName(id string) string {
	if v, ok := ucache["g"+id]; ok {
		return v
	}
	if g, err := user.LookupGroup(id); err == nil {
		return cachedName("g"+id, g.Name)
	}
	return id
}

func infoOf(path string) (ownerInfo, bool) {
	st, ok := syscallStat(path)
	if !ok {
		return ownerInfo{}, false
	}
	uid := strconv.FormatUint(uint64(st.uid), 10)
	gid := strconv.FormatUint(uint64(st.gid), 10)
	return ownerInfo{
		owner:   lookupUserName(uid),
		group:   lookupGroupName(gid),
		modeOct: fmt.Sprintf("%04o", st.mode&0o7777),
	}, true
}
