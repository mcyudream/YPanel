<script setup lang="ts">
import type { SnapZone, WindowInstance } from '@yudream/yudream-webos-core'

import type { ResizeDirection } from '../../composables/usePointerDrag'
import { computed, onBeforeUnmount, ref } from 'vue'
import { usePointerDrag, usePointerResize } from '../../composables/usePointerDrag'
import { useWindowsStore } from '../../stores/windows'

const props = defineProps<{
  win: WindowInstance
}>()

const windowsStore = useWindowsStore()

/** 拖拽起始矩形（相对位移，结束提交 store） */
const dragOrigin = ref(props.win.bounds)
/** 拖拽/缩放进行中：禁用过渡动画 */
const interacting = ref(false)

/** Snap 布局选择器：悬停最大化按钮 400ms 弹出，鼠标离开 500ms 收起（Win11 式） */
const SNAP_ZONES_HALF: Array<'left' | 'top' | 'right'> = ['left', 'top', 'right']
const SNAP_ZONES_QUARTER: Array<'top-left' | 'top-right' | 'bottom-left' | 'bottom-right'> = ['top-left', 'top-right', 'bottom-left', 'bottom-right']
const showSnapPicker = ref(false)
let armTimer: ReturnType<typeof setTimeout> | null = null
let hideTimer: ReturnType<typeof setTimeout> | null = null

function armSnapPicker() {
  if (armTimer) {
    clearTimeout(armTimer)
  }
  if (hideTimer) {
    clearTimeout(hideTimer)
    hideTimer = null
  }
  armTimer = setTimeout(() => {
    showSnapPicker.value = true
  }, 400)
}

function disarmSnapPicker() {
  if (armTimer) {
    clearTimeout(armTimer)
    armTimer = null
  }
  hideTimer = setTimeout(() => {
    showSnapPicker.value = false
  }, 500)
}

function cancelHide() {
  if (hideTimer) {
    clearTimeout(hideTimer)
    hideTimer = null
  }
}

function applySnap(zone: SnapZone) {
  showSnapPicker.value = false
  windowsStore.snap(props.win.id, zone)
}

onBeforeUnmount(() => {
  if (armTimer) {
    clearTimeout(armTimer)
  }
  if (hideTimer) {
    clearTimeout(hideTimer)
  }
})

const titleDrag = usePointerDrag({
  onMove: (dx, dy) => {
    windowsStore.moveTo(props.win.id, dragOrigin.value.x + dx, dragOrigin.value.y + dy)
  },
  onEnd: () => {
    interacting.value = false
    dragOrigin.value = { ...windowsStore.wm.get(props.win.id)!.bounds }
  },
})

function beginDrag(ev: PointerEvent) {
  if (props.win.state === 'maximized' || props.win.state === 'fullscreen') {
    return
  }
  interacting.value = true
  dragOrigin.value = { ...props.win.bounds }
  titleDrag.start(ev)
}

function beginResize(direction: ResizeDirection, ev: PointerEvent) {
  if (props.win.state === 'maximized' || props.win.state === 'fullscreen') {
    return
  }
  interacting.value = true
  const start = { ...props.win.bounds }
  const resize = usePointerResize({
    direction,
    rect: start,
    minWidth: 320,
    minHeight: 240,
    onResize: (rect) => {
      windowsStore.resizeTo(props.win.id, rect)
    },
    onEnd: () => {
      interacting.value = false
      dragOrigin.value = { ...windowsStore.wm.get(props.win.id)!.bounds }
    },
  })
  resize.start(ev)
}

const style = computed(() => {
  if (props.win.state === 'maximized' || props.win.state === 'fullscreen') {
    return {
      left: '0px',
      top: 'var(--yw-menubar-h)',
      width: '100%',
      height: 'calc(100% - var(--yw-menubar-h))',
      zIndex: props.win.zIndex,
    }
  }
  const b = props.win.bounds
  return {
    left: `${b.x}px`,
    top: `${b.y}px`,
    width: `${b.width}px`,
    height: `${b.height}px`,
    zIndex: props.win.zIndex,
  }
})

const isFocused = computed(() => windowsStore.focusedId === props.win.id)
const payload = computed(() => windowsStore.payloadOf(props.win.id))

function onFocus() {
  if (!isFocused.value) {
    windowsStore.focus(props.win.id)
  }
}
</script>

<template>
  <section
    class="yw-window"
    :class="{ 'is-focused': isFocused, 'is-maximized': win.state === 'maximized', 'is-interacting': interacting }"
    :style="style"
    role="dialog"
    :aria-label="win.title"
    :data-window-id="win.id"
    @pointerdown.capture="onFocus"
  >
    <header
      class="yw-window-titlebar"
      @pointerdown="beginDrag"
      @dblclick="windowsStore.toggleMaximize(win.id)"
    >
      <div class="yw-tl-group">
        <button
          class="yw-tl yw-tl--close"
          title="关闭"
          @pointerdown.stop
          @click.stop="windowsStore.close(win.id)"
        >
          <svg viewBox="0 0 12 12" aria-hidden="true"><path d="M3.5 3.5 L8.5 8.5 M8.5 3.5 L3.5 8.5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" /></svg>
        </button>
        <button
          class="yw-tl yw-tl--min"
          title="最小化"
          @pointerdown.stop
          @click.stop="windowsStore.toggleMinimize(win.id)"
        >
          <svg viewBox="0 0 12 12" aria-hidden="true"><path d="M3 6 H9" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" /></svg>
        </button>
        <div class="yw-tl-zoom-wrap" @pointerenter="armSnapPicker" @pointerleave="disarmSnapPicker">
          <button
            class="yw-tl yw-tl--zoom"
            title="最大化（悬停选布局）"
            @pointerdown.stop
            @click.stop="windowsStore.toggleMaximize(win.id)"
          >
            <svg viewBox="0 0 12 12" aria-hidden="true"><path d="M4.2 3 H8.2 V7 M7.8 9 H3.8 V5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" fill="none" /></svg>
          </button>
          <div v-if="showSnapPicker" class="yw-snap-picker" role="menu" aria-label="窗口布局" @pointerenter="cancelHide">
            <div class="yw-snap-row">
              <button
                v-for="zone in SNAP_ZONES_HALF"
                :key="zone"
                class="yw-snap-cell"
                :title="{ left: '左半屏', top: '全屏', right: '右半屏' }[zone]"
                @pointerdown.stop
                @click.stop="applySnap(zone)"
              >
                <span class="yw-snap-shape" :class="`is-${zone}`" />
              </button>
            </div>
            <div class="yw-snap-row">
              <button
                v-for="zone in SNAP_ZONES_QUARTER"
                :key="zone"
                class="yw-snap-cell"
                :title="{ 'top-left': '左上 1/4', 'top-right': '右上 1/4', 'bottom-left': '左下 1/4', 'bottom-right': '右下 1/4' }[zone]"
                @pointerdown.stop
                @click.stop="applySnap(zone)"
              >
                <span class="yw-snap-shape" :class="`is-${zone}`" />
              </button>
            </div>
          </div>
        </div>
      </div>
      <span class="yw-window-title" :class="{ 'is-dim': !isFocused }">{{ win.title }}</span>
      <div class="yw-window-spacer" />
    </header>

    <div class="yw-window-body">
      <slot :win="win" :payload="payload">
        <component :is="payload?.component" v-if="payload?.component" :win="win" />
        <div v-else class="yw-window-empty">
          该应用未提供组件内容
        </div>
      </slot>
    </div>

    <template v-if="win.state !== 'maximized' && win.state !== 'fullscreen'">
      <span class="yw-resize yw-resize--e" @pointerdown="beginResize('e', $event)" />
      <span class="yw-resize yw-resize--s" @pointerdown="beginResize('s', $event)" />
      <span class="yw-resize yw-resize--se" @pointerdown="beginResize('se', $event)" />
      <span class="yw-resize yw-resize--w" @pointerdown="beginResize('w', $event)" />
      <span class="yw-resize yw-resize--n" @pointerdown="beginResize('n', $event)" />
      <span class="yw-resize yw-resize--ne" @pointerdown="beginResize('ne', $event)" />
      <span class="yw-resize yw-resize--nw" @pointerdown="beginResize('nw', $event)" />
      <span class="yw-resize yw-resize--sw" @pointerdown="beginResize('sw', $event)" />
    </template>
  </section>
</template>

<style scoped>
.yw-window {
  position: absolute;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--yw-window-bg);
  border-radius: var(--yw-radius-window);
  box-shadow: var(--yw-shadow-window-inactive);
  transition:
    box-shadow var(--yw-dur-ui) var(--yw-ease-out),
    opacity var(--yw-dur-ui);
  animation: yw-window-in 0.24s var(--yw-ease-spring);
}

.yw-window.is-focused {
  box-shadow: var(--yw-shadow-window);
}

/* 最大化动画用 ease-ios；拖拽/缩放中关掉过渡防拖影 */
.yw-window.is-maximized {
  transition: all var(--yw-dur-panel) var(--yw-ease-ios);
}

.yw-window.is-interacting {
  transition: none;
}

@keyframes yw-window-in {
  from {
    opacity: 0;
    transform: scale(0.94) translateY(10px);
  }

  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.yw-window-titlebar {
  position: relative;
  display: flex;
  flex-shrink: 0;
  align-items: center;
  height: var(--yw-titlebar-h);
  padding: 0 14px;
  touch-action: none;
  cursor: default;
  user-select: none;
  background: var(--yw-titlebar-bg, transparent);
  border-bottom: 1px solid oklch(var(--yw-border) / 55%);
}

/* 交通灯：绝对定位左侧（macOS 布局），hover 显示符号 */
.yw-tl-group {
  position: absolute;
  top: 50%;
  left: var(--yw-tl-inset);
  z-index: 2;
  display: flex;
  gap: var(--yw-tl-gap);
  align-items: center;
  transform: translateY(-50%);
}

.yw-tl {
  display: grid;
  place-items: center;
  width: var(--yw-tl-size);
  height: var(--yw-tl-size);
  padding: 0;
  cursor: pointer;
  border: none;
  border-radius: 50%;
  box-shadow: inset 0 0 1px rgb(0 0 0 / 25%);
  transition: filter var(--yw-dur-micro);
}

.yw-tl svg {
  width: 100%;
  height: 100%;
  color: rgb(0 0 0 / 55%);
  opacity: 0;
  transition: opacity var(--yw-dur-micro);
}

.yw-tl-group:hover .yw-tl svg {
  opacity: 1;
}

.yw-tl:hover {
  filter: brightness(1.08);
}

.yw-tl:active {
  filter: brightness(0.85);
}

.yw-tl--close { background: var(--yw-tl-close); }
.yw-tl--min { background: var(--yw-tl-min); }
.yw-tl--zoom { background: var(--yw-tl-zoom); }

.yw-window:not(.is-focused) .yw-tl {
  background: var(--yw-tl-inactive);
}

.yw-window:not(.is-focused) .yw-tl svg {
  opacity: 0;
}

.yw-window-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: var(--yw-fs-title);
  font-weight: 600;
  color: oklch(var(--yw-foreground));
  text-align: center;
  white-space: nowrap;
  transition: color var(--yw-dur-fast);
}

.yw-window-title.is-dim {
  color: oklch(var(--yw-muted-foreground));
}

.yw-window-spacer {
  flex-shrink: 0;
  width: 60px;
}

.yw-window-body {
  flex: 1;
  overflow: auto;
  color: oklch(var(--yw-foreground));
  background: var(--yw-window-bg);
}

.yw-window-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: oklch(var(--yw-muted-foreground));
}

/* 八向缩放手柄（--yw-resize-hit 命中区） */
.yw-resize {
  position: absolute;

  /* 高于窗口内容（内容撑满后 xterm/splitpanes 等会盖住边缘把手） */
  z-index: 30;
  touch-action: none;
}

.yw-resize--e {
  top: 0;
  right: -3px;
  width: calc(var(--yw-resize-hit) + 1px);
  height: 100%;
  cursor: ew-resize;
}

.yw-resize--w {
  top: 0;
  left: -3px;
  width: calc(var(--yw-resize-hit) + 1px);
  height: 100%;
  cursor: ew-resize;
}

.yw-resize--s {
  bottom: -3px;
  left: 0;
  width: 100%;
  height: calc(var(--yw-resize-hit) + 1px);
  cursor: ns-resize;
}

.yw-resize--n {
  top: -3px;
  left: 0;
  width: 100%;
  height: calc(var(--yw-resize-hit) + 1px);
  cursor: ns-resize;
}

.yw-resize--se {
  right: -5px;
  bottom: -5px;
  width: 16px;
  height: 16px;
  cursor: nwse-resize;
}

.yw-resize--sw {
  bottom: -5px;
  left: -5px;
  width: 16px;
  height: 16px;
  cursor: nesw-resize;
}

.yw-resize--ne {
  top: -5px;
  right: -5px;
  width: 16px;
  height: 16px;
  cursor: nesw-resize;
}

.yw-resize--nw {
  top: -5px;
  left: -5px;
  width: 16px;
  height: 16px;
  cursor: nwse-resize;
}

/* ── Snap 布局选择器（悬停最大化按钮弹出）── */
.yw-tl-zoom-wrap {
  position: relative;
}

.yw-snap-picker {
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  z-index: 40;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px;
  background: var(--yw-glass-panel-bg, rgb(30 30 32 / 92%));
  border-radius: 10px;
  box-shadow: var(--yw-shadow-menu);
  backdrop-filter: blur(var(--yw-blur-strong, 24px)) saturate(160%);
  animation: yw-snap-in 0.16s var(--yw-ease-out);
}

@keyframes yw-snap-in {
  from {
    opacity: 0;
    transform: translateY(-4px) scale(0.96);
  }

  to {
    opacity: 1;
    transform: none;
  }
}

.yw-snap-row {
  display: flex;
  gap: 4px;
}

.yw-snap-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 30px;
  padding: 3px;
  cursor: pointer;
  background: var(--yw-label-4, rgb(255 255 255 / 8%));
  border: 1px solid transparent;
  border-radius: 6px;
  transition: background var(--yw-dur-micro), border-color var(--yw-dur-micro);
}

.yw-snap-cell:hover {
  background: oklch(var(--yw-primary) / 18%);
  border-color: oklch(var(--yw-primary));
}

.yw-snap-shape {
  display: block;
  width: 100%;
  height: 100%;
  background: oklch(var(--yw-foreground) / 40%);
  border-radius: 2px;
}

.yw-snap-shape.is-left {
  width: 50%;
  margin-right: 50%;
}

.yw-snap-shape.is-right {
  width: 50%;
  margin-left: 50%;
}

.yw-snap-shape.is-top {
  width: 100%;
}

.yw-snap-shape.is-top-left {
  width: 50%;
  height: 50%;
  margin-right: 50%;
}

.yw-snap-shape.is-top-right {
  width: 50%;
  height: 50%;
  margin-left: 50%;
}

.yw-snap-shape.is-bottom-left {
  width: 50%;
  height: 50%;
  margin-top: 50%;
  margin-right: 50%;
}

.yw-snap-shape.is-bottom-right {
  width: 50%;
  height: 50%;
  margin-top: 50%;
  margin-left: 50%;
}
</style>
