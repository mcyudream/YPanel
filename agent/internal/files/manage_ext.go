// 文件管理扩展：chmod/压缩/解压/名称搜索。
package files

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// Chmod 修改文件/目录权限（3/4 位八进制）。
func (m *Manager) Chmod(_ context.Context, p, mode string) error {
	abs, err := m.Normalize(p)
	if err != nil {
		return err
	}
	if len(mode) != 3 && len(mode) != 4 {
		return errs.Wrap(errs.ErrBadRequest, "权限格式错误（3 或 4 位八进制）")
	}
	n := 0
	for _, c := range mode {
		if c < '0' || c > '7' {
			return errs.Wrap(errs.ErrBadRequest, "权限格式错误（八进制）")
		}
		n = n*8 + int(c-'0')
	}
	return os.Chmod(abs, os.FileMode(n))
}

// Compress 将 srcPath（文件或目录）打包为 tar.gz 写入 destPath。
func (m *Manager) Compress(_ context.Context, srcPath, destPath string) error {
	absSrc, err := m.Normalize(srcPath)
	if err != nil {
		return err
	}
	absDest, err := m.Normalize(destPath)
	if err != nil {
		return err
	}
	f, err := os.Create(absDest)
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer func() { _ = f.Close() }()
	return m.Archive(absSrc, f)
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
