<script setup lang="ts">
import type { VNode } from '@yudream/yudream-webos-core'
import { useFinder, useVFS, useWebOS } from '@yudream/yudream-webos-vue'
import { computed, onMounted, ref, watch } from 'vue'

/**
 * YwFinder — 访达。四视图（图标/列表/分栏/画廊）+ 侧栏（收藏）+ 工具栏 + 面包屑 + 搜索 + 排序。
 * 文件数据来自 VFS（/ 根），双击文件按 registry.fileHandlers 路由到应用。
 */
withDefaults(defineProps<{
  win?: { id: string }
}>(), {})

const { vfs } = useVFS()
const { state, navigate, back, forward, setView, select } = useFinder()
const { registry, openApp } = useWebOS()

const nodes = ref<VNode[]>([])
const loading = ref(false)
const keyword = ref('')

const SIDEBAR_SECTIONS = [
  {
    title: '收藏',
    items: [
      { label: '文稿', path: '/Documents', icon: 'i-lucide-file-text' },
      { label: '下载', path: '/Downloads', icon: 'i-lucide-download' },
      { label: '图片', path: '/Pictures', icon: 'i-lucide-image' },
      { label: '桌面', path: '/Desktop', icon: 'i-lucide-monitor' },
    ],
  },
  {
    title: '位置',
    items: [
      { label: '系统', path: '/system', icon: 'i-lucide-hard-drive' },
      { label: '根目录', path: '/', icon: 'i-lucide-database' },
    ],
  },
]

async function load() {
  loading.value = true
  try {
    // 确保常用目录存在
    for (const p of ['/Documents', '/Downloads', '/Pictures', '/Desktop']) {
      await vfs.mkdir(p).catch(() => {})
    }
    nodes.value = await vfs.readdir(state.path)
  }
  catch {
    nodes.value = []
  }
  finally {
    loading.value = false
  }
}

onMounted(load)
watch(() => state.path, load)

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  const list = [...nodes.value].sort((a, b) => (a.kind === b.kind ? a.name.localeCompare(b.name) : a.kind === 'directory' ? -1 : 1))
  return kw ? list.filter(n => n.name.toLowerCase().includes(kw)) : list
})

const breadcrumbs = computed(() => {
  const parts = state.path.split('/').filter(Boolean)
  const crumbs = [{ name: '根目录', path: '/' }]
  let acc = ''
  for (const p of parts) {
    acc += `/${p}`
    crumbs.push({ name: p, path: acc })
  }
  return crumbs
})

const selectedIds = computed(() => new Set(state.selection))

function openNode(node: VNode) {
  if (node.kind === 'directory') {
    navigate(node.path)
    return
  }
  // fileHandlers 路由：双击文件 → registry 查应用 → 透传 launchOptions
  const ext = node.name.split('.').pop()?.toLowerCase() ?? ''
  const handler = registry.queryByFileType(ext)[0]
  if (handler) {
    openApp(handler.id, { files: [node.path] })
  }
  else {
    openApp('text-editor', { files: [node.path] })
  }
}

function onItemContextmenu(ev: MouseEvent, node: VNode) {
  ev.preventDefault()
  const { ui } = useWebOS()
  ui.menu({
    x: ev.clientX,
    y: ev.clientY,
    items: [
      { label: '打开', icon: 'i-lucide-external-link', onSelect: () => openNode(node) },
      { label: '重命名', disabled: true },
      { separator: true, label: '' },
      { label: '移到废纸篓', icon: 'i-lucide-trash-2', danger: true, onSelect: () => { void vfs.move(node.path, `/.Trash/${node.name}`) } },
    ],
  })
}
</script>

<template>
  <div class="yw-finder">
    <!-- 工具栏 -->
    <div class="yw-finder-toolbar">
      <div class="yw-finder-nav">
        <button class="yw-finder-btn" :disabled="state.historyIndex === 0" title="返回" @click="back">
          <i class="i-lucide-chevron-left" />
        </button>
        <button class="yw-finder-btn" :disabled="state.historyIndex >= state.history.length - 1" title="前进" @click="forward">
          <i class="i-lucide-chevron-right" />
        </button>
      </div>

      <div class="yw-finder-views">
        <button
          v-for="v in (['icon', 'list', 'column', 'gallery'] as const)"
          :key="v"
          class="yw-finder-view-btn"
          :class="{ 'is-active': state.view === v }"
          :title="v"
          @click="setView(v)"
        >
          <i :class="`i-lucide-layout-${v}`" />
        </button>
      </div>

      <input v-model="keyword" class="yw-finder-search" placeholder="搜索" spellcheck="false">
    </div>

    <div class="yw-finder-main">
      <!-- 侧栏 -->
      <aside class="yw-finder-sidebar">
        <div v-for="sec in SIDEBAR_SECTIONS" :key="sec.title" class="yw-finder-side-sec">
          <div class="yw-finder-side-title">
            {{ sec.title }}
          </div>
          <button
            v-for="item in sec.items"
            :key="item.path"
            class="yw-finder-side-item"
            :class="{ 'is-active': state.path === item.path }"
            @click="navigate(item.path)"
          >
            <i :class="item.icon" />
            {{ item.label }}
          </button>
        </div>
      </aside>

      <!-- 内容 -->
      <div class="yw-finder-content">
        <div class="yw-finder-breadcrumbs">
          <template v-for="(crumb, i) in breadcrumbs" :key="crumb.path">
            <i v-if="i > 0" class="yw-finder-crumb-sep i-lucide-chevron-right" />
            <button class="yw-finder-crumb" @click="navigate(crumb.path)">
              {{ crumb.name }}
            </button>
          </template>
        </div>

        <div v-if="loading" class="yw-finder-status">
          加载中…
        </div>
        <p v-else-if="!filtered.length" class="yw-finder-empty">
          此文件夹为空
        </p>

        <!-- 图标视图 -->
        <div v-else-if="state.view === 'icon' || state.view === 'gallery'" class="yw-finder-grid" :class="{ 'is-gallery': state.view === 'gallery' }">
          <button
            v-for="node in filtered"
            :key="node.path"
            class="yw-finder-item"
            :class="{ 'is-selected': selectedIds.has(node.path) }"
            @click="select([node.path])"
            @dblclick="openNode(node)"
            @contextmenu="onItemContextmenu($event, node)"
          >
            <i :class="node.kind === 'directory' ? 'i-lucide-folder' : 'i-lucide-file'" class="yw-finder-item-icon" />
            <span class="yw-finder-item-name">{{ node.name }}</span>
          </button>
        </div>

        <!-- 列表视图 -->
        <table v-else-if="state.view === 'list'" class="yw-finder-table">
          <thead>
            <tr>
              <th>名称</th>
              <th>修改时间</th>
              <th>大小</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="node in filtered"
              :key="node.path"
              :class="{ 'is-selected': selectedIds.has(node.path) }"
              @click="select([node.path])"
              @dblclick="openNode(node)"
            >
              <td>
                <i :class="node.kind === 'directory' ? 'i-lucide-folder' : 'i-lucide-file'" class="yw-finder-row-icon" />
                {{ node.name }}
              </td>
              <td>{{ new Date(node.modifiedAt).toLocaleString('zh-CN') }}</td>
              <td>{{ node.kind === 'directory' ? '—' : `${node.size} B` }}</td>
            </tr>
          </tbody>
        </table>

        <!-- 分栏视图（简化：路径树 + 图标列表） -->
        <div v-else class="yw-finder-columns">
          <div v-for="crumb in breadcrumbs" :key="crumb.path" class="yw-finder-column">
            <button
              v-for="node in nodes.filter(n => n.path !== crumb.path)"
              :key="node.path"
              class="yw-finder-col-item"
              @click="navigate(node.kind === 'directory' ? node.path : crumb.path); if (node.kind !== 'directory') openNode(node)"
            >
              <i :class="node.kind === 'directory' ? 'i-lucide-folder' : 'i-lucide-file'" />
              {{ node.name }}
            </button>
          </div>
        </div>

        <footer class="yw-finder-statusbar">
          {{ filtered.length }} 个项目
        </footer>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yw-finder {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  font-size: 13px;
  color: oklch(var(--yw-foreground));
  background: var(--yw-window-bg);
}

.yw-finder-toolbar {
  display: flex;
  flex-shrink: 0;
  gap: 12px;
  align-items: center;
  height: 44px;
  padding: 0 12px;
  border-bottom: 1px solid var(--yw-separator);
}

.yw-finder-nav {
  display: flex;
  gap: 2px;
}

.yw-finder-btn {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  font-size: 16px;
  color: oklch(var(--yw-foreground));
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 6px;
}

.yw-finder-btn:disabled {
  opacity: 0.35;
}

.yw-finder-btn:not(:disabled):hover {
  background: var(--yw-label-4);
}

.yw-finder-views {
  display: flex;
  gap: 2px;
  padding: 2px;
  background: var(--yw-label-4);
  border-radius: 7px;
}

.yw-finder-view-btn {
  display: grid;
  place-items: center;
  width: 26px;
  height: 24px;
  font-size: 14px;
  color: oklch(var(--yw-foreground));
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 5px;
}

.yw-finder-view-btn.is-active {
  background: var(--yw-control-bg);
  box-shadow: 0 1px 2px rgb(0 0 0 / 15%);
}

.yw-finder-search {
  flex: 1;
  max-width: 220px;
  height: 26px;
  padding: 0 12px;
  margin-left: auto;
  font-size: 12px;
  color: oklch(var(--yw-foreground));
  outline: none;
  background: rgb(120 120 128 / 14%);
  border: none;
  border-radius: 7px;
  box-shadow: 0 0 0 0.5px var(--yw-separator);
}

.yw-finder-main {
  display: flex;
  flex: 1;
  overflow: hidden;
}

/* 侧栏：玻璃浮动面板（参考 Finder 规格） */
.yw-finder-sidebar {
  display: flex;
  flex-shrink: 0;
  flex-direction: column;
  gap: 14px;
  width: 190px;
  padding: 8px;
  margin: 8px 0 8px 8px;
  overflow-y: auto;
  user-select: none;
  background: var(--yw-glass-panel-bg);
  border-radius: var(--yw-radius-sidebar);
  backdrop-filter: blur(var(--yw-blur-strong)) saturate(170%);
}

.yw-finder-side-title {
  padding: 0 8px;
  margin-bottom: 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--yw-label-2);
}

.yw-finder-side-item {
  display: flex;
  gap: 8px;
  align-items: center;
  width: 100%;
  height: 28px;
  padding: 0 8px;
  font-size: 13px;
  color: oklch(var(--yw-foreground));
  text-align: left;
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 7px;
}

.yw-finder-side-item i {
  font-size: 15px;
  color: oklch(var(--yw-primary));
}

.yw-finder-side-item:hover {
  background: var(--yw-label-4);
}

.yw-finder-side-item.is-active {
  color: #fff;
  background: oklch(var(--yw-primary));
}

.yw-finder-side-item.is-active i {
  color: #fff;
}

/* 内容 */
.yw-finder-content {
  display: flex;
  flex: 1;
  flex-direction: column;
  overflow: hidden;
}

.yw-finder-breadcrumbs {
  display: flex;
  flex-shrink: 0;
  gap: 2px;
  align-items: center;
  padding: 8px 14px 4px;
  font-size: 12px;
}

.yw-finder-crumb {
  padding: 2px 6px;
  color: var(--yw-label-2);
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 4px;
}

.yw-finder-crumb:hover {
  color: oklch(var(--yw-foreground));
}

.yw-finder-crumb:last-child {
  font-weight: 600;
  color: oklch(var(--yw-foreground));
}

.yw-finder-crumb-sep {
  font-size: 12px;
  color: var(--yw-label-3);
}

.yw-finder-grid {
  display: grid;
  flex: 1;
  grid-template-columns: repeat(auto-fill, 96px);
  gap: 6px;
  align-content: start;
  padding: 8px 14px;
  overflow-y: auto;
}

.yw-finder-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
  padding: 8px 4px;
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 8px;
}

.yw-finder-item.is-selected {
  background: var(--yw-accent-selection);
}

.yw-finder-item-icon {
  font-size: 44px;
  color: oklch(72% 0.14 250deg);
}

.yw-finder-item-name {
  max-width: 90px;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  text-align: center;
  white-space: nowrap;
}

.yw-finder-table {
  flex: 1;
  margin: 0 14px;
  font-size: 12px;
  text-align: left;
  border-collapse: collapse;
}

.yw-finder-table th {
  padding: 6px 8px;
  font-weight: 500;
  color: var(--yw-label-2);
  border-bottom: 1px solid var(--yw-separator);
}

.yw-finder-table td {
  padding: 5px 8px;
  border-bottom: 0.5px solid var(--yw-separator);
}

.yw-finder-table tr.is-selected {
  background: var(--yw-accent-selection);
}

.yw-finder-row-icon {
  margin-right: 6px;
  vertical-align: -2px;
  color: oklch(72% 0.14 250deg);
}

.yw-finder-columns {
  display: flex;
  flex: 1;
  overflow-x: auto;
}

.yw-finder-column {
  width: 200px;
  padding: 8px;
  overflow-y: auto;
  border-right: 1px solid var(--yw-separator);
}

.yw-finder-col-item {
  display: flex;
  gap: 6px;
  align-items: center;
  width: 100%;
  height: 26px;
  padding: 0 6px;
  font-size: 12px;
  color: oklch(var(--yw-foreground));
  text-align: left;
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 5px;
}

.yw-finder-col-item:hover {
  background: var(--yw-label-4);
}

.yw-finder-empty,
.yw-finder-status {
  flex: 1;
  padding: 40px;
  color: var(--yw-label-3);
  text-align: center;
}

.yw-finder-statusbar {
  flex-shrink: 0;
  padding: 5px 14px;
  font-size: 11px;
  color: var(--yw-label-2);
  border-top: 1px solid var(--yw-separator);
}
</style>
