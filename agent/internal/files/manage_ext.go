// 文件管理扩展：chmod/chown/压缩/解压/名称搜索/属主枚举。
package files

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// parseFileMode 解析 3/4 位八进制权限串（4 位含 setuid/setgid/sticky）。
func parseFileMode(mode string) (os.FileMode, error) {
	if len(mode) != 3 && len(mode) != 4 {
		return 0, errs.Wrap(errs.ErrBadRequest, "权限格式错误（3 或 4 位八进制）")
	}
	n := 0
	for _, c := range mode {
		if c < '0' || c > '7' {
			return 0, errs.Wrap(errs.ErrBadRequest, "权限格式错误（八进制）")
		}
		n = n*8 + int(c-'0')
	}
	fm := os.FileMode(n & 0o777)
	if n&0o4000 != 0 {
		fm |= os.ModeSetuid
	}
	if n&0o2000 != 0 {
		fm |= os.ModeSetgid
	}
	if n&0o1000 != 0 {
		fm |= os.ModeSticky
	}
	return fm, nil
}

// Chmod 修改文件/目录权限（3/4 位八进制）。recursive 时应用到全部子项；
// symlink 自身无权限语义，跳过（对齐 chmod -R 默认不解引用）。
func (m *Manager) Chmod(_ context.Context, p, mode string, recursive bool) error {
	abs, err := m.Normalize(p)
	if err != nil {
		return err
	}
	fm, err := parseFileMode(mode)
	if err != nil {
		return err
	}
	if !recursive {
		return os.Chmod(abs, fm)
	}
	return filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		return os.Chmod(path, fm)
	})
}

// Chown 修改属主/属组。owner/group 传名字或数字 ID，空串表示该项不更改
// （os.Chown 传 -1 保持原值）。recursive 时子项一并修改，symlink 只改自身（Lchown）不深入。
func (m *Manager) Chown(_ context.Context, p, owner, group string, recursive bool) error {
	abs, err := m.Normalize(p)
	if err != nil {
		return err
	}
	uid, gid := -1, -1
	if owner != "" {
		if uid, err = resolveID(owner, false); err != nil {
			return err
		}
	}
	if group != "" {
		if gid, err = resolveID(group, true); err != nil {
			return err
		}
	}
	if !recursive {
		return os.Chown(abs, uid, gid)
	}
	return filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return os.Lchown(path, uid, gid)
		}
		return os.Chown(path, uid, gid)
	})
}

// resolveID 名字或数字 ID 解析为 uid/gid。
func resolveID(v string, isGroup bool) (int, error) {
	if id, err := strconv.Atoi(v); err == nil {
		return id, nil
	}
	if isGroup {
		g, err := user.LookupGroup(v)
		if err != nil {
			return 0, errs.New(errs.CodeBadRequest, "error.badRequest", "用户组不存在: "+v)
		}
		id, aerr := strconv.Atoi(g.Gid)
		if aerr != nil {
			return 0, errs.Wrapc(errs.CodeFileOpFailed, aerr.Error())
		}
		return id, nil
	}
	u, err := user.Lookup(v)
	if err != nil {
		return 0, errs.New(errs.CodeBadRequest, "error.badRequest", "用户不存在: "+v)
	}
	id, aerr := strconv.Atoi(u.Uid)
	if aerr != nil {
		return 0, errs.Wrapc(errs.CodeFileOpFailed, aerr.Error())
	}
	return id, nil
}

// ListOwners 枚举系统用户与组（解析 /etc/passwd、/etc/group，按数字 ID 升序；
// 非 Linux 或读取失败返回空列表，调用方仍可手输数字 ID）。
func (m *Manager) ListOwners() (*dto.FileOwnersResp, error) {
	users := parseIDFile("/etc/passwd")
	groups := parseIDFile("/etc/group")
	return &dto.FileOwnersResp{Users: users, Groups: groups}, nil
}

// parseIDFile 解析 name:x:id:... 形态的账号文件（跳过空行/注释/NIS 条目）。
func parseIDFile(path string) []dto.FileOwnerEntry {
	out := []dto.FileOwnerEntry{}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.SplitN(line, ":", 4)
		if len(f) < 3 || f[0] == "" || f[0] == "+" || f[0] == "-" || f[2] == "" {
			continue
		}
		out = append(out, dto.FileOwnerEntry{Name: f[0], ID: f[2]})
	}
	sort.Slice(out, func(i, j int) bool {
		ai, aerr := strconv.Atoi(out[i].ID)
		bi, berr := strconv.Atoi(out[j].ID)
		if aerr == nil && berr == nil {
			return ai < bi
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Compress 将多个源（文件/目录）打包为 tar.gz 写入 destPath。
// tar 内每个源以其 basename 为顶层条目；目标位于任一源内部拒绝（防自环）。
func (m *Manager) Compress(_ context.Context, destPath string, srcs []string) error {
	if len(srcs) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "缺少压缩源")
	}
	absSrcs := make([]string, 0, len(srcs))
	for _, s := range srcs {
		abs, err := m.Normalize(s)
		if err != nil {
			return err
		}
		if abs == string(filepath.Separator) {
			return errs.Wrap(errs.ErrBadRequest, "不允许压缩系统根目录")
		}
		for _, prev := range absSrcs {
			if abs == prev {
				return errs.Wrap(errs.ErrBadRequest, "压缩源重复: "+abs)
			}
		}
		absSrcs = append(absSrcs, abs)
	}
	absDest, err := m.Normalize(destPath)
	if err != nil {
		return err
	}
	for _, s := range absSrcs {
		if absDest == s || strings.HasPrefix(absDest, s+string(filepath.Separator)) {
			return errs.Wrap(errs.ErrBadRequest, "压缩目标不能位于源内部: "+s)
		}
	}
	f, err := os.Create(absDest)
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer func() { _ = f.Close() }()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, src := range absSrcs {
		if err := archiveTo(tw, src); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return gz.Close()
}

// archiveTo 将 path（文件/目录/symlink）写入 tar writer，条目以 basename 为顶层前缀。
func archiveTo(tw *tar.Writer, path string) error {
	base := filepath.Base(path)
	return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		link := ""
		if info.Mode()&os.ModeSymlink != 0 {
			link, err = os.Readlink(p)
			if err != nil {
				return err
			}
		}
		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(filepath.Join(base, rel))
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			_, cerr := io.Copy(tw, f)
			_ = f.Close()
			return cerr
		}
		return nil
	})
}

// Decompress 解压 tar.gz 到目标目录（防路径穿越）。
func (m *Manager) Decompress(archivePath, destDir string) error {
	absArc, err := m.Normalize(archivePath)
	if err != nil {
		return err
	}
	absDir, err := m.Normalize(destDir)
	if err != nil {
		return err
	}
	f, err := os.Open(absArc)
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		// 非 gzip：按纯 tar 处理
		if _, serr := f.Seek(0, io.SeekStart); serr != nil {
			return serr
		}
		return extractTar(f, absDir)
	}
	defer func() { _ = gz.Close() }()
	return extractTar(gz, absDir)
}

// Search 按名称递归搜索（深度限 8，结果限 100）。
func (m *Manager) Search(rootDir, keyword string, limit int) ([]dto.FileEntry, error) {
	abs, err := m.Normalize(rootDir)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	kw := strings.ToLower(keyword)
	out := []dto.FileEntry{}
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if len(out) >= limit || depth > 8 {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if len(out) >= limit {
				return
			}
			p := filepath.Join(dir, e.Name())
			if strings.Contains(strings.ToLower(e.Name()), kw) {
				if entry, se := m.statEntry(p, e.Type()&os.ModeSymlink != 0); se == nil {
					out = append(out, entry)
				}
			}
			if e.IsDir() {
				walk(p, depth+1)
			}
		}
	}
	walk(abs, 0)
	sortEntries(out)
	return out, nil
}

func sortEntries(entries []dto.FileEntry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
}

// extractTar 解压 tar 流（支持 gzip 由调用方解包）。
func extractTar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
		}
		target := filepath.Join(dest, filepath.Clean("/"+strings.ReplaceAll(hdr.Name, "\\", "/")))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return errs.Wrap(errs.ErrPathInvalid, "压缩包内路径穿越: "+hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.Symlink(hdr.Linkname, target); err != nil && !os.IsExist(err) {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return err
			}
			_ = f.Close()
		}
	}
}
