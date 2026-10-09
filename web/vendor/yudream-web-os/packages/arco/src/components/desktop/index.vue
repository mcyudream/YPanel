<script setup lang="ts">
import type { DesktopItem, WallpaperMeta } from '@yudream/yudream-webos-core'
import { useWebOS } from '@yudream/yudream-webos-vue'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { resolveBrowserComponent, useAppsStore, useWindowsStore } from '../../stores/compat'
import { useThemeStore } from '../../stores/theme'
import YwAppIcon from '../app-icon/index.vue'
import YwWindow from '../window/index.vue'
import { bindDesktopDrag } from './drag'

const props = withDefaults(defineProps<{
  wallpaper?: WallpaperMeta | null
  /** 图标网格列数（0 = 自动按容器高度填充） */
  iconColumns?: number
  /** 图标停靠角（macOS 惯例右上，自上而下列优先） */
  iconGravity?: 'top-right' | 'top-left'
}>(), {
  wallpaper: null,
  iconColumns: 0,
  iconGravity: 'top-right',
})

const emit = defineEmits<{
  iconContextmenu: [ev: { event: MouseEvent, appId: string }]
  desktopContextmenu: [event: MouseEvent]
}>()

const appsStore = useAppsStore()
const windowsStore = useWindowsStore()
const themeStore = useThemeStore()
const os = useWebOS()

const rootEl = ref<HTMLElement | null>(null)
const gridEl = ref<HTMLElement | null>(null)
const selectedKey = ref<string | null>(null)

/** 拖拽 ghost 状态 */
const dragState = ref<{ id: string, pointerId: number, offsetX: number, offsetY: number, x: number, y: number } | null>(null)

function metrics() {
  return {
    cellWidth: 84,
    cellHeight: 92,
    gap: 4,
    width: gridEl.value?.clientWidth ?? 1200,
    height: gridEl.value?.clientHeight ?? 800,
    gravity: props.iconGravity,
    menubarHeight: 24,
  }
}

/** ── 壁纸 ── */
const wallpaperStyle = computed((): Record<string, string> => {
  const w = props.wallpaper ?? themeStore.wallpaper
  if (!w) {
    return {
      background: 'linear-gradient(160deg, #0b3b66 0%, #1265a8 38%, #2d8fd0 68%, #6db6e8 100%)',
    }
  }
  if (w.src.startsWith('linear-gradient') || w.src.startsWith('radial-gradient')) {
    return { background: w.src }
  }
  return {
    backgroundImage: `url(${w.src})`,
    backgroundSize: w.fit === 'tile' ? 'auto' : (w.fit ?? 'cover'),
    backgroundRepeat: w.fit === 'tile' ? 'repeat' : 'no-repeat',
    backgroundPosition: 'center',
  }
})

/** ── 桌面项：DesktopModel 为数据源 ── */
interface RenderItem {
  id: string
  kind: 'app' | 'folder' | 'file'
  appId?: string
  name: string
  icon: string
  iconBg?: string
  col: number
  row: number
}

const desktopItems = ref<RenderItem[]>([])
const visualCols = ref(12)

function syncFromModel() {
  const width = gridEl.value?.clientWidth ?? 1200
  const height = gridEl.value?.clientHeight ?? 800
  visualCols.value = props.iconColumns || Math.max(1, Math.floor((width + 4) / (84 + 4)))
  // 每列行数上限交给模型（firstFreeCell 换列排布）
  os.desktop.rowsPerColumn = Math.max(1, Math.floor((height + 4) / (92 + 4)))
  const out: RenderItem[] = []
  for (const item of os.desktop.list()) {
    const pos = item.position as { col: number, row: number }
    if (item.type === 'app') {
      const app = appsStore.all[item.refId]
      out.push({
        id: item.id,
        kind: 'app',
        appId: item.refId,
        name: item.name || app?.name || item.refId,
        icon: item.icon || app?.icon || 'i-lucide-circle',
        iconBg: app?.iconBg,
        col: pos.col,
        row: pos.row,
      })
    }
    else if (item.type === 'folder') {
      out.push({ id: item.id, kind: 'folder', name: item.name, icon: 'i-lucide-folder', col: pos.col, row: pos.row })
    }
    else if (item.type === 'file') {
      out.push({ id: item.id, kind: 'file', name: item.name, icon: 'i-lucide-file-text', col: pos.col, row: pos.row })
    }
  }
  desktopItems.value = out
}

/** 把已注册应用播种进 DesktopModel（已有 app 项跳过；启动台是覆盖层不落桌面） */
function seedApps() {
  const existing = new Set(os.desktop.list().filter(x => x.type === 'app').map(x => x.refId))
  for (const app of appsStore.apps) {
    if (existing.has(app.id) || app.id === 'launchpad') {
      continue
    }
    // desktop.show=false = 不上桌面（仅 Dock/启动台/搜索可达）；dock+launchpad 双隐藏 = 彻底隐藏应用
    if (app.desktop?.show === false) {
      continue
    }
    if (app.dock?.showInDock === false && app.launchpad?.show === false) {
      continue
    }
    os.desktop.add({ type: 'app', refId: app.id, name: app.name, icon: app.icon, position: { col: 0, row: 0 } })
  }
}

/** 模型坐标 → 渲染像素（gravity=right 时列镜像） */
function cellStyle(item: RenderItem): Record<string, string> {
  const visualCol = props.iconGravity === 'top-right' ? visualCols.value - 1 - item.col : item.col
  return {
    left: `${visualCol * (84 + 4)}px`,
    top: `${item.row * (92 + 4)}px`,
  }
}

const dragGhost = computed(() => {
  const s = dragState.value
  return s ? { transform: `translate(${s.x}px, ${s.y}px)` } : null
})

const draggingItem = computed(() => {
  const s = dragState.value
  return s ? desktopItems.value.find(x => x.id === s.id) : null
})

const drag = bindDesktopDrag({
  container: null as unknown as HTMLElement,
  model: null as unknown as typeof os.desktop,
  metrics,
  getState: () => dragState.value,
  setState: v => (dragState.value = v),
  onCommit: () => syncFromModel(),
})

function onCellPointerDown(ev: PointerEvent, item: DesktopItem) {
  drag.onPointerDown(ev, item)
}

let detachDrag: (() => void) | null = null

/** 恢复窗口无组件载荷时从注册表重建 */
function reconcilePayloads() {
  for (const w of [...windowsStore.windows]) {
    if (windowsStore.payloadOf(w.id)) {
      continue
    }
    const app = appsStore.all[w.appId]
    if (!app) {
      continue
    }
    if (app.component !== undefined) {
      const comp = app.id === 'browser' ? resolveBrowserComponent() : app.component
      windowsStore.attachPayload(w.id, { appId: app.id, component: comp, title: app.name })
    }
    else {
      windowsStore.close(w.id)
    }
  }
}

function syncViewport() {
  windowsStore.setViewport({ width: window.innerWidth, height: window.innerHeight, menubarHeight: 24 })
  syncFromModel()
}

let mediaQuery: MediaQueryList | null = null

onMounted(async () => {
  syncViewport()
  window.addEventListener('resize', syncViewport)
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  themeStore.setSystemDark(mediaQuery.matches)
  themeStore.apply()
  windowsStore.restore()
  // 布局持久化由 provider 统一装配（desktop.layout scope）；此处等它恢复后播种应用
  void os.persist.get<any[]>(`desktop.layout`).then((saved) => {
    if (Array.isArray(saved)) {
      for (const item of saved) {
        os.desktop.add(item)
      }
    }
    seedApps()
    syncFromModel()
  })
  os.desktop.onChange(() => {
    syncFromModel()
  })
  watch(() => appsStore.apps.length, () => seedApps())
  if (gridEl.value) {
    drag.container = gridEl.value
    drag.model = os.desktop
    drag.attach()
    detachDrag = drag.detach
  }
})

watch(() => appsStore.all, reconcilePayloads, { deep: true })

onBeforeUnmount(() => {
  window.removeEventListener('resize', syncViewport)
  mediaQuery?.removeEventListener('change', () => {})
  detachDrag?.()
})

/** ── 交互 ── */
function onIconClick(item: RenderItem) {
  selectedKey.value = item.id
}

function onIconOpen(item: RenderItem) {
  selectedKey.value = null
  if (item.kind === 'app') {
    appsStore.openApp(item.appId!)
  }
  else if (item.kind === 'folder') {
    os.openApp('finder', { path: `/Desktop/${item.name}` })
  }
  else {
    os.openApp('text-editor', { files: [`/Desktop/${item.name}`] })
  }
}

function onIconContextmenu(ev: MouseEvent, item: RenderItem) {
  ev.preventDefault()
  ev.stopPropagation()
  selectedKey.value = item.id
  os.ui.menu({
    x: ev.clientX,
    y: ev.clientY,
    items: [
      { label: '打开', icon: 'i-lucide-external-link', onSelect: () => onIconOpen(item) },
      { label: '重命名', icon: 'i-lucide-pencil', disabled: item.kind === 'app', onSelect: () => startRename(item) },
      { separator: true, label: '' },
      { label: '移到废纸篓', icon: 'i-lucide-trash-2', danger: true, disabled: item.kind === 'app', onSelect: () => void trashItem(item) },
    ],
  })
}

async function trashItem(item: RenderItem) {
  if (os.config.desktop.bindVFS && item.kind !== 'app') {
    await os.vfs.mkdir('/.Trash').catch(() => {})
    await os.vfs.move(`/Desktop/${item.name}`, `/.Trash/${item.name}`).catch(() => {})
  }
  os.desktop.remove(item.id)
}

function onDesktopPointerDown(ev: PointerEvent) {
  if (ev.target === rootEl.value || (ev.target as HTMLElement).classList?.contains('yw-desktop-icons')) {
    selectedKey.value = null
  }
}

/** ── 新建（文件夹 / 文本文件）── */
const creating = reactive({ open: false, kind: 'folder' as 'folder' | 'file', name: '' })

function startCreate(kind: 'folder' | 'file') {
  creating.open = true
  creating.kind = kind
  creating.name = kind === 'folder' ? '新建文件夹' : '新建文本.txt'
}

async function confirmCreate() {
  const name = creating.name.trim()
  creating.open = false
  if (!name) {
    return
  }
  if (os.config.desktop.bindVFS) {
    const path = `/Desktop/${name}`
    if (creating.kind === 'folder') {
      await os.vfs.mkdir(path).catch(() => {})
    }
    else {
      await os.vfs.write(path, '').catch(() => {})
    }
  }
  const pos = firstFreeVisual()
  const modelCol = props.iconGravity === 'top-right' ? Math.max(0, visualCols.value - 1 - pos.col) : pos.col
  os.desktop.add({ type: creating.kind, refId: creating.kind === 'folder' ? `/Desktop/${name}` : '', name, position: { col: modelCol, row: pos.row } })
}

function firstFreeVisual(): { col: number, row: number } {
  const taken = new Set(desktopItems.value.map(x => `${x.col},${x.row}`))
  const rows = Math.max(1, Math.floor(((gridEl.value?.clientHeight ?? 800) - 40) / (92 + 4)))
  for (let col = 0; col < 32; col++) {
    for (let row = 0; row < rows; row++) {
      if (!taken.has(`${col},${row}`)) {
        return { col, row }
      }
    }
  }
  return { col: 0, row: 0 }
}

/** 重命名 */
const renaming = reactive({ id: '', name: '' })

function startRename(item: RenderItem) {
  renaming.id = item.id
  renaming.name = item.name
}

async function confirmRename() {
  const item = os.desktop.get(renaming.id)
  const old = item?.name
  const next = renaming.name.trim()
  renaming.id = ''
  if (!item || !next || next === old) {
    return
  }
  if (os.config.desktop.bindVFS && item.type !== 'app' && old) {
    await os.vfs.move(`/Desktop/${old}`, `/Desktop/${next}`).catch(() => {})
  }
  item.name = next
  syncFromModel()
  syncFromModel()
}

/** 桌面右键菜单（带实功能，供模板与宿主复用） */
function showDesktopMenu(ev: MouseEvent) {
  os.ui.menu({
    x: ev.clientX,
    y: ev.clientY,
    items: [
      { label: '新建文件夹', icon: 'i-lucide-folder-plus', onSelect: () => startCreate('folder') },
      { label: '新建文本文件', icon: 'i-lucide-file-plus', onSelect: () => startCreate('file') },
      { separator: true, label: '' },
      { label: '整理图标', icon: 'i-lucide-layout-grid', onSelect: () => os.desktop.autoArrange() },
      { label: '按名称排序', icon: 'i-lucide-arrow-down-a-z', onSelect: () => os.desktop.sortBy('name') },
      { separator: true, label: '' },
      { label: '平铺全部窗口', icon: 'i-lucide-layout-panel-left', onSelect: () => windowsStore.tileAll() },
      { separator: true, label: '' },
      { label: '编辑小组件', icon: 'i-lucide-puzzle', onSelect: () => window.dispatchEvent(new CustomEvent('webos:widgets:edit')) },
      { label: '系统设置…', icon: 'i-lucide-settings', onSelect: () => os.openApp('settings') },
    ],
  })
}

defineExpose({ showDesktopMenu })
</script>

<template>
  <div
    ref="rootEl"
    class="yw-desktop"
    :style="wallpaperStyle"
    @pointerdown="onDesktopPointerDown"
    @contextmenu.prevent="showDesktopMenu($event); emit('desktopContextmenu', $event)"
  >
    <div class="yw-desktop-dim" />

    <!-- 图标网格：绝对定位单元格，支持拖拽换位 -->
    <div ref="gridEl" class="yw-desktop-icons">
      <div
        v-for="item in desktopItems"
        :key="item.id"
        class="yw-desktop-cell"
        :style="cellStyle(item)"
        @pointerdown="onCellPointerDown($event, os.desktop.get(item.id)!)"
        @click="onIconClick(item)"
        @dblclick="onIconOpen(item)"
        @contextmenu="onIconContextmenu($event, item)"
      >
        <YwAppIcon
          :app-key="item.kind === 'app' ? item.appId! : item.id"
          :icon="item.icon"
          :icon-bg="item.iconBg"
          :title="item.name"
          :selected="selectedKey === item.id"
        />
      </div>

      <!-- 拖拽 ghost -->
      <div v-if="dragGhost && draggingItem" class="yw-desktop-cell yw-desktop-ghost" :style="dragGhost">
        <YwAppIcon
          :app-key="draggingItem.kind === 'app' ? draggingItem.appId! : draggingItem.id"
          :icon="draggingItem.icon"
          :icon-bg="draggingItem.iconBg"
          :title="draggingItem.name"
        />
      </div>
    </div>

    <!-- 新建对话框 -->
    <div v-if="creating.open" class="yw-desktop-dialog" @pointerdown.self="creating.open = false">
      <div class="yw-desktop-dialog-card">
        <b>{{ creating.kind === 'folder' ? '新建文件夹' : '新建文本文件' }}</b>
        <input v-model="creating.name" class="yw-desktop-input" @keydown.enter="confirmCreate">
        <div class="yw-desktop-actions">
          <button @click="creating.open = false">
            取消
          </button>
          <button class="is-primary" @click="confirmCreate">
            创建
          </button>
        </div>
      </div>
    </div>

    <!-- 重命名对话框 -->
    <div v-if="renaming.id" class="yw-desktop-dialog" @pointerdown.self="renaming.id = ''">
      <div class="yw-desktop-dialog-card">
        <b>重命名</b>
        <input v-model="renaming.name" class="yw-desktop-input" @keydown.enter="confirmRename">
        <div class="yw-desktop-actions">
          <button @click="renaming.id = ''">
            取消
          </button>
          <button class="is-primary" @click="confirmRename">
            确定
          </button>
        </div>
      </div>
    </div>

    <!-- 窗口层 -->
    <div class="yw-desktop-windows">
      <YwWindow
        v-for="win in windowsStore.visible"
        :key="win.id"
        :win="win"
        @focus="windowsStore.focus"
      />
    </div>
  </div>
</template>

<style scoped>
.yw-desktop-cell {
  position: absolute;
  width: var(--yw-deskicon-cell-w, 84px);
  height: var(--yw-deskicon-cell-h, 92px);
  touch-action: none;
  transition: left 0.18s var(--yw-ease-out), top 0.18s var(--yw-ease-out);
}

.yw-desktop-cell.is-dragging {
  opacity: 0.35;
}

.yw-desktop-cell :deep(.yw-app-icon) {
  width: 100%;
}

.yw-desktop-ghost {
  z-index: 50;
  pointer-events: none;
  opacity: 0.85;
}

.yw-desktop-dialog {
  position: absolute;
  inset: 0;
  z-index: 9400;
  display: grid;
  place-items: center;
  background: rgb(0 0 0 / 25%);
}

.yw-desktop-dialog-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 300px;
  padding: 16px;
  font-size: 13px;
  color: oklch(var(--yw-foreground));
  background: oklch(var(--yw-popover));
  border-radius: var(--yw-radius-window);
  box-shadow: var(--yw-shadow-window);
}

.yw-desktop-input {
  height: 30px;
  padding: 0 10px;
  color: oklch(var(--yw-foreground));
  outline: none;
  background: rgb(120 120 128 / 14%);
  border: none;
  border-radius: 7px;
}

.yw-desktop-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.yw-desktop-actions button {
  padding: 5px 16px;
  font-size: 12px;
  color: oklch(var(--yw-foreground));
  cursor: default;
  background: var(--yw-label-4);
  border: none;
  border-radius: 7px;
}

.yw-desktop-actions button.is-primary {
  color: #fff;
  background: oklch(var(--yw-primary));
}
</style>

<style scoped>
/* legacy: icons container */
.yw-desktop {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: linear-gradient(160deg, #1e293b 0%, #334155 100%);
}

.yw-desktop-dim {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.dark .yw-desktop-dim {
  background: rgb(0 0 0 / 12%);
}

.yw-desktop-icons {
  position: absolute;
  inset: calc(var(--yw-menubar-h, 24px) + 12px) 14px 14px;
  pointer-events: none;
}

.yw-desktop-icons .yw-desktop-cell {
  pointer-events: auto;
}

.yw-desktop-windows {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.yw-desktop-windows > * {
  pointer-events: auto;
}
</style>
