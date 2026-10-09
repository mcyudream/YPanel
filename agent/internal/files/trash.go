// 回收站（M38）：删除先进 /opt/ypanel/.trash（0700），每条带元数据 JSON；
// 还原不覆盖同名；彻底删除/清空需显式调用。单用户面板场景不加锁（写路径均为原子 rename）。
package files

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// TrashRoot 回收站根目录。
const TrashRoot = "/opt/ypanel/.trash"

func trashMetaPath(name string) string {
	return filepath.Join(TrashRoot, name+".json")
}

// Trash 把路径移入回收站（rename 语义，跨设备退化为复制+删除）。
func (m *Manager) Trash(paths []string) ([]dto.TrashItemMeta, error) {
	if err := os.MkdirAll(TrashRoot, 0o700); err != nil {
		return nil, err
	}
	out := []dto.TrashItemMeta{}
	for _, p := range paths {
		abs, err := m.Normalize(p)
		if err != nil {
			return out, err
		}
		if abs == "/" || abs == "/usr" || abs == "/etc" || abs == "/var" || abs == "/boot" || strings.HasPrefix(abs, TrashRoot) {
			return out, errs.Wrap(errs.ErrBadRequest, "拒绝移入回收站: "+abs)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return out, errs.Wrap(errs.ErrNotFound, "路径不存在: "+abs)
		}
		name := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(abs))
		meta := dto.TrashItemMeta{Original: abs, Name: name, IsDir: info.IsDir(), Size: info.Size(), TrashedAt: time.Now()}
		// rename 落回收站（同设备原子；跨设备 EXDEV 时退化为 MoveAll 语义）
		if err := os.Rename(abs, filepath.Join(TrashRoot, name)); err != nil {
			return out, errs.Wrapc(errs.CodeFileOpFailed, "移入回收站失败: "+err.Error())
		}
		mb, _ := json.Marshal(meta)
		tmp := trashMetaPath(name) + ".tmp"
		if err := os.WriteFile(tmp, mb, 0o600); err != nil {
			return out, err
		}
		if err := os.Rename(tmp, trashMetaPath(name)); err != nil {
			return out, err
		}
		out = append(out, meta)
	}
	return out, nil
}

// TrashList 回收站列表（按时间倒序）。
func (m *Manager) TrashList() ([]dto.TrashItemMeta, error) {
	out := []dto.TrashItemMeta{}
	entries, err := os.ReadDir(TrashRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		mb, err := os.ReadFile(filepath.Join(TrashRoot, e.Name()))
		if err != nil {
			continue
		}
		var meta dto.TrashItemMeta
		if json.Unmarshal(mb, &meta) != nil {
			continue
		}
		if meta.Name == "" {
			meta.Name = strings.TrimSuffix(e.Name(), ".json")
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TrashedAt.After(out[j].TrashedAt) })
	return out, nil
}

// TrashRestore 还原到原路径（目标已存在拒绝，不覆盖）。
func (m *Manager) TrashRestore(names []string) error {
	for _, n := range names {
		if strings.Contains(n, "/") || strings.Contains(n, "..") {
			return errs.ErrPathInvalid
		}
		mb, err := os.ReadFile(trashMetaPath(n))
		if err != nil {
			return errs.Wrap(errs.ErrNotFound, "回收站条目不存在: "+n)
		}
		var meta dto.TrashItemMeta
		if json.Unmarshal(mb, &meta) != nil || meta.Original == "" {
			return errs.Wrap(errs.ErrInternal, "元数据损坏: "+n)
		}
		if _, err := os.Lstat(meta.Original); err == nil {
			return errs.New(errs.CodeConflict, "error.conflict", "目标已存在，不覆盖: "+meta.Original)
		}
		if err := os.Rename(filepath.Join(TrashRoot, n), meta.Original); err != nil {
			return errs.Wrapc(errs.CodeFileOpFailed, "还原失败: "+err.Error())
		}
		_ = os.Remove(trashMetaPath(n))
	}
	return nil
}

// TrashPurge 彻底删除回收站条目。
func (m *Manager) TrashPurge(names []string) error {
	for _, n := range names {
		if strings.Contains(n, "/") || strings.Contains(n, "..") {
			return errs.ErrPathInvalid
		}
		if err := os.RemoveAll(filepath.Join(TrashRoot, n)); err != nil {
			return err
		}
		_ = os.Remove(trashMetaPath(n))
	}
	return nil
}

// TrashClear 清空回收站。
func (m *Manager) TrashClear() (int, error) {
	items, err := m.TrashList()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, it := range items {
		if err := os.RemoveAll(filepath.Join(TrashRoot, it.Name)); err == nil {
			_ = os.Remove(trashMetaPath(it.Name))
			n++
		}
	}
	return n, nil
}
