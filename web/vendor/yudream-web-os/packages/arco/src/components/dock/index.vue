<script setup lang="ts">
import type { AppDefinition } from '@yudream/yudream-webos-core'
import { useWebOS } from '@yudream/yudream-webos-vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useAppsStore, useWindowsStore } from '../../stores/compat'
import YwIconTile from '../icon-tile/index.vue'

const props = withDefaults(defineProps<{
  /** 停靠位置 */
  position?: 'bottom' | 'left' | 'right'
  /** 悬停放大 */
  magnification?: boolean
  /** 图标尺寸 px */
  iconSize?: number
  /** 接受文件拖放的应用 id（拖到这些图标上会 emit file-drop） */
  dropApps?: string[]
}>(), {
  position: 'bottom',
  magnification: true,
  iconSize: 48,
  dropApps: () => [],
})

const emit = defineEmits<{
  showQuicklaunch: []
  /** 文件拖到 Dock 图标上（宿主决定行为：如拖到终端图标=新开终端填路径） */
  fileDrop: [{ appId: string, dataTransfer: DataTransfer }]
}>()

const appsStore = useAppsStore()
const windowsStore = useWindowsStore()
const { dock, ui } = useWebOS()

const horizontal = computed(() => props.position === 'bottom')

/** 运行中的 appId 集合 */
const running = computed(() => new Set(windowsStore.runningApps))

/** 有最大化/全屏窗口时抑制 Dock（全屏应用独占屏幕，Dock 滑出避让） */
const suppressed = computed(() => windowsStore.windows.some(w => w.state === 'maximized' || w.state === 'fullscreen'))

/** 避让中的贴边呼出：鼠标压到 Dock 所在侧边缘时临时滑入，移开自动缩回 */
const dockEl = ref<HTMLElement | null>(null)
const peek = ref(false)
const EDGE = 8

onMounted(() => {
  window.addEventListener('mousemove', onMouseMove)
})

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', onMouseMove)
})

function onMouseMove(e: MouseEvent) {
  if (!suppressed.value) {
    peek.value = false
    return
  }
  const atEdge = props.position === 'left'
    ? e.clientX <= EDGE
    : props.position === 'right'
      ? e.clientX >= window.innerWidth - EDGE
      : e.clientY >= window.innerHeight - EDGE
  if (atEdge) {
    peek.value = true
    return
  }
  // 已呼出时：鼠标离开 Dock 区域（上方留 24px 余量）才缩回，避免中途误缩
  if (peek.value) {
    const dockTop = props.position === 'bottom'
      ? (dockEl.value?.getBoundingClientRect().top ?? window.innerHeight)
      : window.innerWidth
    const away = props.position === 'bottom'
      ? e.clientY < dockTop - 24
      : e.clientX < dockTop - 24
    if (away) {
      peek.value = false
    }
  }
}

/** 文件拖放悬停的图标（结构化拖拽 MIME 命中且应用接受时高亮） */
const dropHoverApp = ref<string | null>(null)
const STRUCTURED_MIME_PREFIX = 'application/x-'

function acceptsDrop(appId: string) {
  return props.dropApps.includes(appId)
}

function onItemDragOver(appId: string, e: DragEvent) {
  if (!acceptsDrop(appId) || !(e.dataTransfer?.types ?? []).some(t => t.startsWith(STRUCTURED_MIME_PREFIX))) {
    return
  }
  e.preventDefault()
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'copy'
  }
  dropHoverApp.value = appId
}

function onItemDragLeave(appId: string) {
  if (dropHoverApp.value === appId) {
    dropHoverApp.value = null
  }
}

function onItemDrop(appId: string, e: DragEvent) {
  dropHoverApp.value = null
  if (!acceptsDrop(appId) || !e.dataTransfer) {
    return
  }
  e.preventDefault()
  emit('fileDrop', { appId, dataTransfer: e.dataTransfer })
}

/** Dock 项：固定应用 + 运行中的非固定应用 */
const items = computed(() => {
  const pinnedApps = dock.pinned
    .map(id => appsStore.all[id])
    .filter((a): a is AppDefinition => Boolean(a))
  const pinnedIds = new Set(pinnedApps.map(a => a.id))
  const runningUnpinned = appsStore.apps.filter(a => running.value.has(a.id) && !pinnedIds.has(a.id))
  return [...pinnedApps, ...runningUnpinned]
})

/** 分隔符：固定区与运行区之间 */
const separatorIndex = computed(() => dock.pinned.length)

const hoverKey = ref<string | null>(null)
const launchingKey = ref<string | null>(null)

function onItemClick(appId: string) {
  const wins = windowsStore.windows.filter(w => w.appId === appId)
  if (!wins.length) {
    launchingKey.value = appId
    setTimeout(() => {
      launchingKey.value = null
    }, 700)
    appsStore.openApp(appId)
    return
  }
  const win = wins[0]!
  if (windowsStore.focusedId === win.id && win.state !== 'minimized') {
    windowsStore.toggleMinimize(win.id)
  }
  else {
    windowsStore.focus(win.id)
  }
}

function onItemContextMenu(ev: MouseEvent, appId: string) {
  ev.preventDefault()
  const appWins = windowsStore.windows.filter(w => w.appId === appId)
  const pinnedIdx = dock.pinned.indexOf(appId)
  const multi = appsStore.all[appId]?.multiInstance
  ui.menu({
    x: ev.clientX,
    y: ev.clientY - 10,
    items: [
      // 新窗口：直接走 openApp（multiInstance 每次开新窗；聚焦已有窗走 onItemClick）
      ...(multi
        ? [{ label: '新窗口', icon: 'i-lucide-app-window', onSelect: () => appsStore.openApp(appId) }]
        : []),
      // 该应用全部已开窗口：点击聚焦（最小化的经 core focus 自动还原置顶）
      ...appWins.map(w => ({
        label: w.state === 'minimized' ? `${w.title}（最小化）` : (w.title ?? w.id),
        icon: w.id === windowsStore.focusedId ? 'i-lucide-circle-dot' : undefined,
        onSelect: () => windowsStore.focus(w.id),
      })),
      ...(appWins.length ? [{ separator: true, label: '' }] : []),
      ...(appWins.length
        ? [{ label: '关闭窗口', icon: 'i-lucide-x', danger: true, onSelect: () => windowsStore.close(appWins[0]!.id) }]
        : []),
      ...(pinnedIdx > -1
        ? [{ label: '从 Dock 移除', icon: 'i-lucide-minus-circle', onSelect: () => dock.unpin(appId) }]
        : [{ label: '固定到 Dock', icon: 'i-lucide-pin', onSelect: () => dock.pin(appId) }]),
    ],
  })
}
</script>

<template>
  <nav
    ref="dockEl"
    class="yw-dock"
    :class="[`is-${position}`, { 'has-magnification': magnification, 'is-suppressed': suppressed && !peek }]"
    :aria-orientation="horizontal ? 'horizontal' : 'vertical'"
  >
    <button
      class="yw-dock-item"
      @click="emit('showQuicklaunch')"
      @mouseenter="hoverKey = 'yw-quicklaunch'"
      @mouseleave="hoverKey = null"
    >
      <YwIconTile app-key="yw-quicklaunch" icon="i-lucide-search" :size="iconSize" />
      <span v-if="hoverKey === 'yw-quicklaunch'" class="yw-dock-tip" :class="{ 'is-vertical': !horizontal }">快速启动</span>
    </button>

    <template v-for="(app, i) in items" :key="app.id">
      <span v-if="i === separatorIndex && separatorIndex < items.length && separatorIndex > 0" class="yw-dock-sep" />
      <button
        class="yw-dock-item"
        :class="{
          'is-running': running.has(app.id),
          'is-focused': windowsStore.windows.some(w => w.appId === app.id && w.id === windowsStore.focusedId),
          'is-launching': launchingKey === app.id,
          'is-drop-hover': dropHoverApp === app.id,
        }"
        @click="onItemClick(app.id)"
        @contextmenu.prevent="onItemContextMenu($event, app.id)"
        @mouseenter="hoverKey = app.id"
        @mouseleave="hoverKey = null"
        @dragover="onItemDragOver(app.id, $event)"
        @dragleave="onItemDragLeave(app.id)"
        @drop="onItemDrop(app.id, $event)"
      >
        <YwIconTile :app-key="app.id" :icon="app.icon" :icon-bg="app.iconBg" :size="iconSize" />
        <span v-if="hoverKey === app.id" class="yw-dock-tip" :class="{ 'is-vertical': !horizontal }">{{ app.name }}</span>
        <span class="yw-dock-dot" />
      </button>
    </template>
  </nav>
</template>

<style scoped>
.yw-dock {
  position: fixed;
  z-index: 9000;
  display: flex;
  gap: var(--yw-dock-gap);
  padding: var(--yw-dock-pad);
  background: var(--yw-glass-dock-bg);
  border-radius: var(--yw-radius-dock);
  box-shadow: var(--yw-shadow-dock);
  backdrop-filter: blur(var(--yw-blur-medium)) saturate(170%);
  transition: transform var(--yw-dur-panel, 0.3s) var(--yw-ease-out), opacity var(--yw-dur-panel, 0.3s);
}

.yw-dock.is-bottom {
  bottom: var(--yw-dock-margin);
  left: 50%;
  flex-direction: row;
  align-items: flex-end;
  transform: translateX(-50%);
}

.yw-dock.is-left {
  top: 50%;
  left: var(--yw-dock-margin);
  flex-direction: column;
  transform: translateY(-50%);
}

.yw-dock.is-right {
  top: 50%;
  right: var(--yw-dock-margin);
  flex-direction: column;
  transform: translateY(-50%);
}

.yw-dock-item {
  position: relative;
  display: grid;
  flex-shrink: 0;
  place-items: center;
  padding: 0;
  cursor: pointer;
  background: transparent;
  border: none;
  transition: transform 0.18s var(--yw-ease-spring);
}

.yw-dock.has-magnification .yw-dock-item:hover {
  transform: scale(1.16);
}

.yw-dock.has-magnification.is-bottom .yw-dock-item:hover {
  transform: scale(1.16) translateY(-4px);
}

.yw-dock-item:active {
  transform: scale(0.94);
}

.yw-dock.is-suppressed {
  pointer-events: none;
  opacity: 0;
}

.yw-dock.is-bottom.is-suppressed {
  transform: translateY(110%);
}

.yw-dock.is-left.is-suppressed {
  transform: translateX(-110%);
}

.yw-dock.is-right.is-suppressed {
  transform: translateX(110%);
}

.yw-dock-item.is-drop-hover {
  outline: 2px solid oklch(var(--yw-primary));
  outline-offset: 2px;
  border-radius: var(--yw-radius-capsule, 14px);
}

.yw-dock-item.is-launching > * {
  animation: yw-dock-launch 0.65s var(--yw-ease-out);
}

@keyframes yw-dock-launch {
  0%,
  100% { transform: translateY(0); }
  40% { transform: translateY(-14px); }
  65% { transform: translateY(-2px); }
}

/* tooltip 胶囊 */
.yw-dock-tip {
  position: absolute;
  bottom: calc(100% + 12px);
  left: 50%;
  z-index: 10;
  padding: 4px 12px;
  font-size: var(--yw-fs-small);
  font-weight: 500;
  color: oklch(var(--yw-popover-foreground));
  white-space: nowrap;
  pointer-events: none;
  background: oklch(var(--yw-popover) / 88%);
  border-radius: var(--yw-radius-capsule);
  box-shadow: var(--yw-shadow-tooltip);
  backdrop-filter: blur(var(--yw-blur-light));
  transform: translateX(-50%);
  animation: yw-tip-in 0.14s var(--yw-ease-out);
}

.yw-dock-tip.is-vertical {
  inset: 50% calc(100% + 12px) auto auto;
  transform: translateY(-50%);
  animation: none;
}

@keyframes yw-tip-in {
  from {
    opacity: 0;
    transform: translateX(-50%) translateY(4px);
  }

  to {
    opacity: 1;
    transform: translateX(-50%) translateY(0);
  }
}

/* 运行指示点 */
.yw-dock-dot {
  position: absolute;
  bottom: -3px;
  left: 50%;
  width: 4px;
  height: 4px;
  background: oklch(var(--yw-foreground) / 75%);
  border-radius: 50%;
  opacity: 0;
  transform: translateX(-50%);
  transition: opacity var(--yw-dur-fast);
}

.yw-dock.is-left .yw-dock-dot,
.yw-dock.is-right .yw-dock-dot {
  inset: 50% -5px auto auto;
  transform: translateY(-50%);
}

.yw-dock-item.is-running .yw-dock-dot {
  opacity: 1;
}

/* 分隔线 */
.yw-dock-sep {
  flex-shrink: 0;
  align-self: stretch;
  width: 1px;
  margin: 4px 5px;
  background: oklch(var(--yw-foreground) / 18%);
}

.yw-dock.is-left .yw-dock-sep,
.yw-dock.is-right .yw-dock-sep {
  width: auto;
  height: 1px;
  margin: 5px 4px;
}
</style>
