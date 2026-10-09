<script setup lang="ts">
import type { DesktopItem, WallpaperMeta } from '@yudream/yudream-webos-core'
import { useWebOS } from '@yudream/yudream-webos-vue'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { resolveBrowserComponent, useAppsStore, useWindowsStore } from '../../stores/compat'
import { useThemeStore } from '../../stores/theme'
import YwAppIcon from '../app-icon/index.vue'
import YwWindow from '../window/index.vue'
import { bindDesktopDrag } from './drag'
import { tileBackground } from '../icon-tile/colors'

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
  kind: 'app' | 'folder' | 'file' | 'widget'
  appId?: string
  /** 仅 widget：实例/定义标识 */
  instanceId?: string
  widgetId?: string
  name: string
  icon: string
  iconBg?: string
  col: number
  row: number
  /** 多格跨度（模型坐标），缺省 1x1 */
  spanW: number
  spanH: number
  /** 仅 folder：子项快照（卡片快捷预览/浮层） */
  children: Array<{ id: string, kind: 'app' | 'file', appId?: string, name: string, icon: string, iconBg?: string }>
}

const desktopItems = ref<RenderItem[]>([])
const visualCols = ref(12)
/** 自动重排守卫：一轮 out-of-range 只触发一次重排（防极端窄面板循环） */
let repackGuard = false

function childIcon(child: DesktopItem): { icon: string, iconBg?: string } {
  const app = child.type === 'app' ? appsStore.all[child.refId] : undefined
  return { icon: child.icon || app?.icon || (child.type === 'folder' ? 'i-lucide-folder' : 'i-lucide-file-text'), iconBg: app?.iconBg }
}

function syncFromModel() {
  const width = gridEl.value?.clientWidth ?? 1200
  const height = gridEl.value?.clientHeight ?? 800
  visualCols.value = props.iconColumns || Math.max(1, Math.floor((width + 4) / (84 + 4)))
  // 每列行数上限交给模型（firstFreeCell 换列排布）
  os.desktop.rowsPerColumn = Math.max(1, Math.floor((height + 4) / (92 + 4)))
  // 面板收窄后，落在可见列之外的项会变成「看不见的洞」——自动重排进可见区。
  // 守卫位防止极端窄面板下（放不下全部项）syncFromModel ↔ autoArrange 循环。
  const outOfRange = os.desktop.list().some((x) => {
    const s = x.span ?? { w: 1, h: 1 }
    return (x.position as { col: number }).col + s.w - 1 > visualCols.value - 1
  })
  if (outOfRange && os.desktop.list().length && !repackGuard) {
    repackGuard = true
    os.desktop.autoArrange()
    return
  }
  if (!outOfRange) {
    repackGuard = false
  }
  const out: RenderItem[] = []
  for (const item of os.desktop.list()) {
    const pos = item.position as { col: number, row: number }
    const span = item.span ?? { w: 1, h: 1 }
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
        spanW: span.w,
        spanH: span.h,
        children: [],
      })
    }
    else if (item.type === 'folder') {
      out.push({
        id: item.id,
        kind: 'folder',
        name: item.name,
        icon: 'i-lucide-folder',
        col: pos.col,
        row: pos.row,
        spanW: span.w,
        spanH: span.h,
        children: (item.children ?? []).map(c => ({
          id: c.id,
          kind: c.type === 'app' ? 'app' : 'file',
          appId: c.type === 'app' ? c.refId : undefined,
          name: c.name,
          ...childIcon(c),
        })),
      })
    }
    else if (item.type === 'file') {
      out.push({ id: item.id, kind: 'file', name: item.name, icon: 'i-lucide-file-text', col: pos.col, row: pos.row, spanW: span.w, spanH: span.h, children: [] })
    }
    else if (item.type === 'widget') {
      const inst = os.widgets.listInstances().find(i => i.instanceId === item.refId)
      const def = inst ? os.widgets.getDefinition(inst.widgetId) : undefined
      out.push({ id: item.id, kind: 'widget', name: def?.name ?? item.name, icon: 'i-lucide-puzzle', col: pos.col, row: pos.row, spanW: span.w, spanH: span.h, children: [], instanceId: item.refId, widgetId: inst?.widgetId })
    }
  }
  desktopItems.value = out
}

/** 小组件实例 ↔ 桌面模型项双向对账（widget 项为真实占格项） */
const WIDGET_SPAN: Record<string, { w: number, h: number }> = { small: { w: 2, h: 2 }, medium: { w: 4, h: 2 }, large: { w: 4, h: 4 } }
let widgetSyncing = false

function syncWidgets() {
  if (widgetSyncing) {
    return
  }
  widgetSyncing = true
  try {
    const instances = os.widgets.listInstances()
    for (const mw of os.desktop.list().filter(x => x.type === 'widget')) {
      if (!instances.some(i => i.instanceId === mw.refId)) {
        os.desktop.remove(mw.id)
      }
    }
    for (const inst of instances) {
      const id = `yww-${inst.instanceId}`
      const span = WIDGET_SPAN[inst.size] ?? { w: 2, h: 2 }
      const existing = os.desktop.get(id)
      if (!existing) {
        const def = os.widgets.getDefinition(inst.widgetId)
        os.desktop.add({ id, type: 'widget', refId: inst.instanceId, name: def?.name ?? inst.widgetId, span, position: inst.position })
      }
      else {
        existing.span = span
      }
    }
  }
  finally {
    widgetSyncing = false
  }
}

function widgetInst(item: RenderItem) {
  return os.widgets.listInstances().find(i => i.instanceId === item.instanceId)
}

function widgetComp(item: RenderItem): unknown {
  const inst = widgetInst(item)
  return inst ? os.widgets.getDefinition(inst.widgetId)?.component : undefined
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

/** 卡片右下把手拖拽调大小的预览跨度（声明前置供 cellStyle 引用） */
const resizePreview = ref<{ id: string, w: number, h: number, leftVisual: number } | null>(null)
let resizeCleanup: (() => void) | null = null

/** gravity=right 时模型列向左增长：跨格项的视觉左缘 = 其覆盖的最后一个模型列的视觉列 */
function visualLeftOf(item: { col: number, spanW?: number }): number {
  const w = item.spanW ?? 1
  const lastModelCol = item.col + w - 1
  return props.iconGravity === 'top-right' ? visualCols.value - 1 - lastModelCol : item.col
}

/** 模型坐标 → 渲染像素；span 决定格子尺寸（拖拽调大小时用预览跨度） */
function cellStyle(item: RenderItem): Record<string, string> {
  const p = resizePreview.value?.id === item.id ? resizePreview.value : null
  const w = p ? p.w : item.spanW
  const h = p ? p.h : item.spanH
  return {
    left: `${(p ? p.leftVisual : visualLeftOf(item)) * (84 + 4)}px`,
    top: `${item.row * (92 + 4)}px`,
    width: `${w * 84 + (w - 1) * 4}px`,
    height: `${h * 92 + (h - 1) * 4}px`,
  }
}

/** 卡片右下把手拖拽调大小：pointermove 实时预览跨度，pointerup 提交（resizeItem 挤开占位者） */
function startCardResize(ev: PointerEvent, item: RenderItem) {
  ev.preventDefault()
  ev.stopPropagation()
  const grid = gridEl.value?.getBoundingClientRect()
  if (!grid) {
    return
  }
  // rtl：向视觉右扩大 = 基列左移，视觉左缘固定；最多扩到 col 0
  const leftVisual = props.iconGravity === 'top-right'
    ? visualCols.value - 1 - (item.col + item.spanW - 1)
    : item.col
  const maxW = Math.min(4, props.iconGravity === 'top-right' ? item.spanW + item.col : visualCols.value - item.col)
  const maxH = Math.min(4, os.desktop.rowsPerColumn || 4)
  const onMove = (e: PointerEvent) => {
    const vcol = Math.floor((e.clientX - grid.left) / (84 + 4))
    const row = Math.floor((e.clientY - grid.top) / (92 + 4))
    resizePreview.value = {
      id: item.id,
      w: Math.max(1, Math.min(maxW, vcol - leftVisual + 1)),
      h: Math.max(1, Math.min(maxH, row - item.row + 1)),
      leftVisual,
    }
  }
  const onUp = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    window.removeEventListener('pointercancel', onUp)
    resizeCleanup = null
    const p = resizePreview.value
    resizePreview.value = null
    if (p && (p.w !== item.spanW || p.h !== item.spanH)) {
      os.desktop.resizeItem(p.id, { w: p.w, h: p.h })
    }
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
  window.addEventListener('pointercancel', onUp)
  resizeCleanup = onUp
}

const dragGhost = computed(() => {
  const s = dragState.value
  if (!s) {
    return null
  }
  // ghost 跟随被拖项的跨度（跨格卡拖动时不再是 1×1 小图）
  const d = desktopItems.value.find(x => x.id === s.id)
  const w = d ? d.spanW * 84 + (d.spanW - 1) * 4 : 84
  const h = d ? d.spanH * 92 + (d.spanH - 1) * 4 : 92
  return { transform: `translate(${s.x}px, ${s.y}px)`, width: `${w}px`, height: `${h}px` }
})

const draggingItem = computed(() => {
  const s = dragState.value
  return s ? desktopItems.value.find(x => x.id === s.id) : null
})

/** 拖拽悬停命中的文件夹（高亮反馈 + 落点移入） */
const hoverFolderId = ref<string | null>(null)

/** 落点格指示（随被拖项跨度） */
const dropHint = ref<{ col: number, row: number } | null>(null)

function dropHintStyle(h: { col: number, row: number }): Record<string, string> {
  const d = draggingItem.value
  const w = d?.spanW ?? 1
  const hh = d?.spanH ?? 1
  const left = props.iconGravity === 'top-right' ? visualCols.value - 1 - (h.col + w - 1) : h.col
  return {
    left: `${left * (84 + 4)}px`,
    top: `${h.row * (92 + 4)}px`,
    width: `${w * 84 + (w - 1) * 4}px`,
    height: `${hh * 92 + (hh - 1) * 4}px`,
  }
}

function resolveDropTarget(id: string, col: number, row: number): string | null {
  for (const item of desktopItems.value) {
    if (item.kind !== 'folder' || item.id === id) {
      continue
    }
    if (col >= item.col && col < item.col + item.spanW && row >= item.row && row < item.row + item.spanH) {
      return item.id
    }
  }
  return null
}

const drag = bindDesktopDrag({
  container: null as unknown as HTMLElement,
  model: null as unknown as typeof os.desktop,
  metrics,
  getState: () => dragState.value,
  setState: v => (dragState.value = v),
  onCommit: (id) => {
    const it = os.desktop.get(id)
    if (it?.type === 'widget') {
      os.widgets.moveInstance(it.refId, it.position as { col: number, row: number })
    }
    syncFromModel()
  },
  resolveDropTarget: (id, col, row) => resolveDropTarget(id, col, row),
  setHoverFolder: (id) => { hoverFolderId.value = id },
  setDropHint: (h) => { dropHint.value = h },
})

/** 多格应用大卡：图标砖铺满整卡（图标字形随卡缩放），底色沿用 icon-tile 色板 */
function largeTileStyle(item: RenderItem): Record<string, string> {
  return { background: tileBackground(item.appId ?? item.id, item.iconBg) }
}

function isImageIcon(item: RenderItem): boolean {
  return /^(?:https?:|data:|\/|\.)/.test(item.icon)
}

/** 文件夹卡：末格「打开」之外能外露的子项数 */
function folderMore(item: RenderItem): number {
  return item.children.length - (item.spanW * item.spanH - 1)
}

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
let detachWidgetsWatch: (() => void) | null = null

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
    syncWidgets()
    syncFromModel()
  })
  os.desktop.onChange(() => {
    syncFromModel()
  })
  // 小组件增删/尺寸变化 → 对账出模型项（真实占格）+ 重渲染
  detachWidgetsWatch = os.widgets.onChange(() => {
    syncWidgets()
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
  detachWidgetsWatch?.()
  resizeCleanup?.()
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
    openFolder(item)
  }
  else {
    openFileItem(item.id, item.name)
  }
}

/** 桌面文本文件 → 内置编辑器：同一文件只开一窗（已开则聚焦还原），窗口标题=文件名 */
function openFileItem(id: string, name: string) {
  const existing = os.wm.windowsOfApp('text-editor').find(w => (w.launchOptions as Record<string, unknown> | undefined)?.desktopId === id)
  if (existing) {
    os.wm.focus(existing.id)
    return
  }
  os.wm.open('text-editor', { title: name, launchOptions: { desktopId: id } })
}

function openChild(child: RenderItem['children'][number]) {
  if (child.kind === 'app') {
    appsStore.openApp(child.appId!)
  }
  else {
    openFileItem(child.id, child.name)
  }
}

function onIconContextmenu(ev: MouseEvent, item: RenderItem) {
  ev.preventDefault()
  ev.stopPropagation()
  selectedKey.value = item.id
  if (item.kind === 'widget') {
    const inst = widgetInst(item)
    const def = inst ? os.widgets.getDefinition(inst.widgetId) : undefined
    const sizeItems = (def?.sizes ?? []).map(sz => ({
      label: sz === 'small' ? '小（2×2）' : sz === 'medium' ? '中（4×2）' : '大（4×4）',
      onSelect: () => os.widgets.resizeInstance(inst!.instanceId, sz),
    }))
    os.ui.menu({
      x: ev.clientX,
      y: ev.clientY,
      items: [
        ...(sizeItems.length ? [{ label: '大小 ▸', icon: 'i-lucide-scaling', onSelect: () => os.ui.menu({ x: ev.clientX, y: ev.clientY, items: sizeItems }) }] : []),
        { separator: true, label: '' },
        { label: '移除小组件', icon: 'i-lucide-trash-2', danger: true, disabled: !inst, onSelect: () => os.widgets.removeInstance(inst!.instanceId) },
      ],
    })
    emit('iconContextmenu', { event: ev, appId: item.id })
    return
  }
  const items: any[] = [{ label: '打开', icon: 'i-lucide-external-link', onSelect: () => onIconOpen(item) }]
  if (item.kind === 'folder' && item.spanW * item.spanH > 1) {
    items.push({
      label: '整理此文件夹',
      icon: 'i-lucide-square-stack',
      onSelect: () => {
        openFolder(item)
      },
    })
  }
  if (item.kind !== 'app') {
    items.push({ label: '重命名', icon: 'i-lucide-pencil', onSelect: () => startRename(item) })
  }
  // 大小：文件夹卡片 1×1/2×2/3×2/4×3；应用图标 1×1/2×1/1×2/2×2/4×2
  const presets: Array<{ label: string, w: number, h: number }> = item.kind === 'folder'
    ? [
        { label: '1 × 1（普通图标）', w: 1, h: 1 },
        { label: '2 × 2', w: 2, h: 2 },
        { label: '3 × 2', w: 3, h: 2 },
        { label: '4 × 3', w: 4, h: 3 },
      ]
    : [
        { label: '1 × 1（普通图标）', w: 1, h: 1 },
        { label: '2 × 1（宽）', w: 2, h: 1 },
        { label: '1 × 2（高）', w: 1, h: 2 },
        { label: '2 × 2', w: 2, h: 2 },
        { label: '4 × 2（大卡）', w: 4, h: 2 },
      ]
  items.push({
    label: '大小 ▸',
    icon: 'i-lucide-scaling',
    onSelect: () => {
      os.ui.menu({
        x: ev.clientX,
        y: ev.clientY,
        items: presets.map(p => ({
          label: p.label,
          onSelect: () => {
            const res = os.desktop.resizeItem(item.id, { w: p.w, h: p.h })
            if (!res.ok) {
              os.ui.message('error', '该区域被占用（图标或小组件），换个位置再试')
            }
          },
        })),
      })
    },
  })
  if (item.kind === 'folder' && item.children.length) {
    items.push({
      label: '解散文件夹',
      icon: 'i-lucide-folder-open',
      onSelect: () => { os.desktop.dissolveFolder(item.id) },
    })
  }
  items.push(
    { separator: true, label: '' },
    { label: item.kind === 'folder' ? '移到废纸篓（含内容）' : '移到废纸篓', icon: 'i-lucide-trash-2', danger: true, disabled: item.kind === 'app', onSelect: () => void trashItem(item) },
  )
  os.ui.menu({ x: ev.clientX, y: ev.clientY, items })
  emit('iconContextmenu', { event: ev, appId: item.appId ?? item.id })
}

async function trashItem(item: RenderItem) {
  if (os.config.desktop.bindVFS && item.kind !== 'app') {
    await os.vfs.mkdir('/.Trash').catch(() => {})
    await os.vfs.move(`/Desktop/${item.name}`, `/.Trash/${item.name}`).catch(() => {})
  }
  if (item.kind === 'folder') {
    closeFolder()
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
  // 位置交给模型：firstFreeCell 会避开其它图标与小组件占格
  os.desktop.add({ type: creating.kind, refId: creating.kind === 'folder' ? `/Desktop/${name}` : '', name, position: { col: 0, row: 0 } })
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
}

/** ── 文件夹浮层：完整子项网格 ── */
const openFolderId = ref<string | null>(null)
const openFolderStyle = ref<Record<string, string>>({})

function openFolder(item: RenderItem) {
  const root = rootEl.value?.getBoundingClientRect()
  const grid = gridEl.value?.getBoundingClientRect()
  if (root && grid) {
    const x = grid.left - root.left + visualLeftOf(item) * 88
    const y = grid.top - root.top + item.row * 96
    const width = Math.min(3 * 88 + 16, root.width - 24)
    openFolderStyle.value = {
      left: `${Math.max(12, Math.min(x - 8, root.width - width - 12))}px`,
      top: `${Math.max(12, Math.min(y, root.height - 320))}px`,
      width: `${width}px`,
    }
  }
  openFolderId.value = item.id
}

function closeFolder() {
  openFolderId.value = null
}

const openFolderItem = computed(() => {
  if (!openFolderId.value) {
    return null
  }
  return desktopItems.value.find(x => x.id === openFolderId.value && x.kind === 'folder') ?? null
})

function onChildContextmenu(ev: MouseEvent, folder: RenderItem, child: RenderItem['children'][number]) {
  ev.preventDefault()
  ev.stopPropagation()
  os.ui.menu({
    x: ev.clientX,
    y: ev.clientY,
    items: [
      { label: '打开', icon: 'i-lucide-external-link', onSelect: () => openChild(child) },
      { label: '移出文件夹', icon: 'i-lucide-log-out', onSelect: () => { os.desktop.removeFromFolder(folder.id, child.id) } },
    ],
  })
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
        :class="{
          'is-folder-card': item.kind === 'folder' && item.spanW * item.spanH > 1,
          'is-hover-folder': hoverFolderId === item.id,
          'is-large-app': item.kind === 'app' && item.spanW * item.spanH > 1,
          'is-dragging': dragState?.id === item.id,
        }"
        :style="cellStyle(item)"
        @pointerdown="onCellPointerDown($event, os.desktop.get(item.id)!)"
        @click="onIconClick(item)"
        @dblclick="onIconOpen(item)"
        @contextmenu="onIconContextmenu($event, item)"
      >
        <!-- 文件夹卡片（多格）：W×H 预览网格铺满，末格固定「打开」，名称挂卡下 -->
        <template v-if="item.kind === 'folder' && item.spanW * item.spanH > 1">
          <div class="yw-folder-card">
            <span v-if="item.children.length" class="yw-folder-badge">{{ item.children.length }}</span>
            <div class="yw-folder-grid" :style="{ gridTemplateColumns: `repeat(${item.spanW}, 1fr)` }">
              <button
                v-for="child in item.children.slice(0, item.spanW * item.spanH - 1)"
                :key="child.id"
                class="yw-folder-mini"
                :title="child.name"
                @click.stop="openChild(child)"
                @dblclick.stop
                @contextmenu.stop="onChildContextmenu($event, item, child)"
              >
                <YwAppIcon :app-key="child.appId ?? child.id" :icon="child.icon" :icon-bg="child.iconBg" :title="child.name" />
              </button>
              <button
                class="yw-folder-open"
                :title="`打开 ${item.name}`"
                @click.stop="openFolder(item)"
                @dblclick.stop
              >
                <i class="i-lucide-folder-open" />
                <span v-if="folderMore(item) > 0" class="yw-folder-open-more">+{{ folderMore(item) }}</span>
                <span v-else-if="!item.children.length" class="yw-folder-open-more">空</span>
              </button>
            </div>
          </div>
          <span class="yw-folder-label">{{ item.name }}</span>
        </template>
        <!-- 小组件：定义组件直接铺在格内（真实占格项） -->
        <div v-else-if="item.kind === 'widget'" class="yw-widget-item">
          <div class="yw-widget-item-card">
            <div class="yw-widget-item-body">
              <component :is="widgetComp(item)" v-if="widgetComp(item)" :config="widgetInst(item)?.config ?? {}" />
            </div>
          </div>
        </div>
        <!-- 多格应用大卡：图标砖铺满整卡 -->
        <div
          v-else-if="item.kind === 'app' && item.spanW * item.spanH > 1"
          class="yw-large-tile"
          :style="largeTileStyle(item)"
        >
          <img v-if="isImageIcon(item)" :src="item.icon" alt="" draggable="false">
          <i v-else :class="item.icon" />
        </div>
        <!-- 普通图标（1×1） -->
        <YwAppIcon
          v-else
          :app-key="item.kind === 'app' ? item.appId! : item.id"
          :icon="item.icon"
          :icon-bg="item.iconBg"
          :title="item.name"
          :selected="selectedKey === item.id"
        />
        <!-- 卡片右下拖拽调大小把手（悬停浮现） -->
        <span
          v-if="item.spanW * item.spanH > 1 && item.kind !== 'widget'"
          class="yw-card-resize"
          :class="{ 'is-on-label': item.kind === 'folder' }"
          title="拖拽调整大小"
          @pointerdown.stop="startCardResize($event, item)"
        ><i /></span>
      </div>

      <!-- 落点格指示 -->
      <div v-if="dropHint && dragState" class="yw-drop-hint" :style="dropHintStyle(dropHint)" />

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

    <!-- 文件夹浮层：完整子项网格 -->
    <template v-if="openFolderItem">
      <div class="yw-folder-flyout-backdrop" @pointerdown.self="closeFolder" />
      <div class="yw-folder-flyout" :style="openFolderStyle">
        <div class="yw-folder-flyout-head">
          <i class="i-lucide-folder" />
          <b>{{ openFolderItem.name }}</b>
          <span>{{ openFolderItem.children.length }}</span>
          <button class="yw-folder-flyout-close" @click="closeFolder">
            <i class="i-lucide-x" />
          </button>
        </div>
        <div class="yw-folder-flyout-grid">
          <button
            v-for="child in openFolderItem.children"
            :key="child.id"
            class="yw-folder-mini"
            :title="child.name"
            @click="openChild(child)"
            @contextmenu="onChildContextmenu($event, openFolderItem, child)"
          >
            <YwAppIcon :app-key="child.appId ?? child.id" :icon="child.icon" :icon-bg="child.iconBg" :title="child.name" />
          </button>
          <div v-if="!openFolderItem.children.length" class="yw-folder-flyout-empty">
            空文件夹——把图标拖到桌面文件夹上即可归组
          </div>
        </div>
      </div>
    </template>

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
  will-change: transform;
}

/* 格内小组件卡（原生占格项） */
.yw-widget-item {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  padding: 4px; /* 卡片间 8px 呼吸间距（两侧各 4） */
  pointer-events: auto;
}

.yw-widget-item-card {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  overflow: hidden;
  cursor: default;
  background: color-mix(in oklch, oklch(var(--yw-popover, 100% 0 0)) 48%, transparent);
  border: 1px solid rgb(255 255 255 / 12%);
  border-radius: var(--yw-radius-widget, 16px);
  backdrop-filter: blur(16px) saturate(1.3);
  box-shadow: 0 8px 24px rgb(0 0 0 / 18%);
  transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
}

/* 与文件夹卡同族的悬停反馈：描边提亮 + 轻浮起 */
.yw-desktop-cell:hover .yw-widget-item-card {
  border-color: rgb(255 255 255 / 24%);
  box-shadow: 0 10px 28px rgb(0 0 0 / 26%);
  transform: translateY(-1px);
}

.yw-widget-item-body {
  flex: 1;
  min-height: 0;
  padding: 12px;
}

/* 落点格指示（拖动时的目标高亮框） */
.yw-drop-hint {
  position: absolute;
  z-index: 49;
  pointer-events: none;
  will-change: transform;
  background: oklch(var(--yw-primary, 60% 0.15 250) / 22%);
  border: 2px dashed oklch(var(--yw-primary, 60% 0.15 250) / 75%);
  border-radius: 14px;
}

/* ── 文件夹卡片（多格）：对齐小组件卡片的视觉 ── */
.yw-desktop-cell.is-folder-card,
.yw-desktop-cell.is-large-app {
  pointer-events: auto;
}

.yw-folder-card {
  position: relative;
  display: flex;
  flex-direction: column;
  height: calc(100% - 20px);
  padding: 6px 0;
  cursor: default;
  background: color-mix(in oklch, oklch(var(--yw-popover, 100% 0 0)) 26%, transparent);
  border: 1px solid rgb(255 255 255 / 14%);
  border-radius: 14px;
  backdrop-filter: blur(18px) saturate(1.4);
  box-shadow: 0 8px 24px rgb(0 0 0 / 18%);
  transition: border-color 0.15s, box-shadow 0.15s;
}

/* 卡片右下拖拽调大小把手（悬停浮现） */
.yw-card-resize {
  position: absolute;
  right: 2px;
  bottom: 2px;
  z-index: 3;
  width: 16px;
  height: 16px;
  cursor: nwse-resize;
  opacity: 0;
  background:
    linear-gradient(135deg,
      transparent 0 40%,
      rgb(255 255 255 / 80%) 40% 50%,
      transparent 50% 64%,
      rgb(255 255 255 / 80%) 64% 74%,
      transparent 74%);
  border-radius: 4px;
  transition: opacity 0.15s;
}

.yw-desktop-cell:hover .yw-card-resize {
  opacity: 0.9;
}

.yw-card-resize.is-on-label {
  bottom: 22px;
}

/* 名称挂卡片下方（对齐系统图标标签） */
.yw-folder-label {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 20px;
  overflow: hidden;
  font-size: 12px;
  font-weight: 500;
  line-height: 20px;
  color: #fff;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-shadow:
    0 1px 2px rgb(0 0 0 / 70%),
    0 0 8px rgb(0 0 0 / 45%);
  pointer-events: none;
}

/* 总数角标（右上角，小米式） */
.yw-folder-badge {
  position: absolute;
  top: -8px;
  right: -8px;
  z-index: 1;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  font-size: 10px;
  line-height: 18px;
  color: #fff;
  text-align: center;
  background: oklch(var(--yw-primary, 60% 0.15 250));
  border: 1.5px solid rgb(255 255 255 / 85%);
  border-radius: 99px;
}

.yw-desktop-cell.is-hover-folder .yw-folder-card {
  border-color: oklch(var(--yw-primary));
  box-shadow: 0 0 0 2px oklch(var(--yw-primary) / 55%), 0 8px 24px rgb(0 0 0 / 18%);
}

.yw-folder-grid {
  display: grid;
  flex: 1;
  gap: 2px;
}

.yw-folder-mini {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  min-width: 0;
  padding: 2px;
  cursor: default;
  background: none;
  border: none;
  border-radius: 8px;
}

.yw-folder-mini :deep(.yw-app-icon) {
  width: 100%;
  height: 100%;
  justify-content: center;
  padding: 0;
}

/* 外露图标不显示文字（名称在浮层里看） */
.yw-folder-mini :deep(.yw-app-icon-label) {
  display: none;
}

/* 末格「打开文件夹」 */
.yw-folder-open {
  position: relative;
  display: grid;
  place-items: center;
  width: 48px;
  height: 48px;
  place-self: center;
  cursor: default;
  background: rgb(255 255 255 / 10%);
  border: 1px dashed rgb(255 255 255 / 30%);
  border-radius: 12px;
  transition: background 0.15s;
}

.yw-folder-open:hover {
  background: rgb(255 255 255 / 20%);
}

.yw-folder-open i {
  width: 22px;
  height: 22px;
  color: #fff;
}

.yw-folder-open-more {
  position: absolute;
  right: 4px;
  bottom: 3px;
  font-size: 9px;
  color: #fff;
  opacity: 0.85;
}

/* 多格应用大卡：图标砖铺满整卡，视觉与 icon-tile 同源（圆角/高光/投影） */
.yw-large-tile {
  position: relative;
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: 24px;
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 40%),
    inset 0 0 0 0.5px rgb(255 255 255 / 25%),
    inset 0 -6px 12px rgb(0 0 0 / 18%),
    0 3px 8px rgb(0 0 0 / 30%),
    0 1px 3px rgb(0 0 0 / 20%);
}

.yw-large-tile::after {
  content: '';
  position: absolute;
  inset: 0 0 52% 0;
  pointer-events: none;
  background: linear-gradient(rgb(255 255 255 / 26%), rgb(255 255 255 / 3%));
}

.yw-large-tile i {
  width: auto;
  height: 56%;
  aspect-ratio: 1;
  color: #fff;
  filter: drop-shadow(0 2px 4px rgb(0 0 0 / 30%));
}

.yw-large-tile img {
  width: 80%;
  height: 80%;
  object-fit: contain;
}

/* ── 文件夹浮层 ── */
.yw-folder-flyout-backdrop {
  position: absolute;
  inset: 0;
  z-index: 9000;
}

.yw-folder-flyout {
  position: absolute;
  z-index: 9001;
  max-height: 320px;
  padding: 10px;
  overflow: auto;
  background: oklch(var(--yw-popover) / 92%);
  border: 1px solid rgb(255 255 255 / 12%);
  border-radius: 14px;
  box-shadow: var(--yw-shadow-window);
}

.yw-folder-flyout-head {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-bottom: 8px;
  font-size: 12px;
  color: oklch(var(--yw-foreground));
}

.yw-folder-flyout-head span {
  color: oklch(var(--yw-muted-foreground, 50% 0 0));
}

.yw-folder-flyout-close {
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  margin-left: auto;
  cursor: default;
  background: rgb(120 120 128 / 18%);
  border: none;
  border-radius: 6px;
}

.yw-folder-flyout-grid {
  display: grid;
  grid-template-columns: repeat(3, 72px);
  gap: 6px;
}

.yw-folder-flyout-empty {
  padding: 14px 6px;
  font-size: 11px;
  color: oklch(var(--yw-muted-foreground, 50% 0 0));
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
  user-select: none; /* 拖动图标/卡片时不得拉出文字选区 */
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
  z-index: 125;
  pointer-events: none;
}

.yw-desktop-windows :deep(.yw-window) {
  pointer-events: auto;
}
</style>
