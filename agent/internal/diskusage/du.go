// Package diskusage 目录占用统计（du 语义：不跟随符号链接、不跨挂载点、跳过无权限子树）。
package diskusage

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// cacheTTL 结果缓存：目录实算耗时与文件数成正比，长缓存让下钻/切换秒开；
// 前端「刷新」走 force 绕过缓存重算。agent 重启即失效（纯内存）。
const cacheTTL = 24 * time.Hour

// walkParallel 子目录并行实算的并发上限。
const walkParallel = 8

// cacheMax 条目上限：防止极端导航路径把缓存撑爆；超限整体清空（下次重算）。
const cacheMax = 256

var (
	mu    sync.Mutex
	cache = map[string]cacheEntry{}
)

type cacheEntry struct {
	resp *dto.DiskUsageTree
	at   time.Time
}

// Tree 计算 path 下各直接子项的占用。path 必须是绝对路径（前端只回传本接口给出的路径）；
// force=true 跳过缓存读取（重算后回写缓存）。
func Tree(ctx context.Context, path string, force bool) (*dto.DiskUsageTree, error) {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) || clean != path || strings.Contains(clean, "..") {
		return nil, errs.ErrBadRequest
	}
	mu.Lock()
	if !force {
		if c, ok := cache[clean]; ok && time.Since(c.at) < cacheTTL {
			mu.Unlock()
			return c.resp, nil
		}
	}
	if len(cache) >= cacheMax {
		cache = map[string]cacheEntry{}
	}
	mu.Unlock()

	entries, err := os.ReadDir(clean)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "read dir: "+err.Error())
	}
	type job struct{ name, full string }
	dirs := make([]job, 0, len(entries))
	var loose int64
	for _, e := range entries {
		if e.Type()&fs.ModeSymlink != 0 {
			continue // 符号链接不计入（防环、防重复计数）
		}
		if e.IsDir() {
			dirs = append(dirs, job{e.Name(), filepath.Join(clean, e.Name())})
		} else if info, ierr := e.Info(); ierr == nil {
			loose += info.Size()
		}
	}

	out := make([]dto.DiskUsageTreeItem, len(dirs))
	sem := make(chan struct{}, walkParallel)
	var wg sync.WaitGroup
	for i, d := range dirs {
		wg.Add(1)
		go func(i int, d job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			// 子目录本身是挂载点（如 /proc /sys /run）则不下钻：与所在目录不同设备即独立文件系统
			if same, serr := sameDevice(d.full, clean); serr == nil && !same {
				return // Size 保持 0，前端按 size>0 过滤
			}
			var n int64
			walkSize(ctx, d.full, &n)
			out[i] = dto.DiskUsageTreeItem{Name: d.name, Path: d.full, Size: n, IsDir: true}
		}(i, d)
	}
	wg.Wait()

	items := make([]dto.DiskUsageTreeItem, 0, len(out)+1)
	items = append(items, out...)
	sort.Slice(items, func(a, b int) bool { return items[a].Size > items[b].Size })
	if loose > 0 {
		items = append(items, dto.DiskUsageTreeItem{Name: "(散落文件)", Path: clean, Size: loose})
	}
	var total int64
	for _, it := range items {
		total += it.Size
	}
	resp := &dto.DiskUsageTree{Path: clean, Total: total, Items: items, CollectedAt: time.Now()}
	mu.Lock()
	cache[clean] = cacheEntry{resp: resp, at: time.Now()}
	mu.Unlock()
	return resp, nil
}

// walkSize 递归累计目录占用；无权限/已消失的子树跳过（du 行为），ctx 取消即中止。
func walkSize(ctx context.Context, root string, n *int64) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if p != root {
				if same, serr := sameDevice(p, root); serr == nil && !same {
					return fs.SkipDir // 不跨挂载点（du -x 语义，避开 /proc /sys 等伪文件系统）
				}
			}
			return nil
		}
		if info, ierr := d.Info(); ierr == nil {
			*n += info.Size()
		}
		return nil
	})
}
