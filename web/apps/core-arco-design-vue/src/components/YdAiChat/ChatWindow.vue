<script setup lang="ts">
// YdAiChatWindow：对话窗口壳（fixed 可拖动/缩放/最大化/全屏，规格对齐 YdChatWindow）。
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = withDefaults(defineProps<{
  title?: string
  width?: number
  height?: number
  minWidth?: number
  minHeight?: number
  resizable?: boolean
  maximizable?: boolean
  fullscreenable?: boolean
  /** 最大化/全屏按钮改为对外抛出 expand 事件 */
  expandOnly?: boolean
}>(), {
  title: 'AI 助手',
  width: 480,
  height: 640,
  minWidth: 360,
  minHeight: 420,
  resizable: true,
  maximizable: true,
  fullscreenable: true,
  expandOnly: false,
})

const emit = defineEmits<{ close: [], expand: [] }>()

const pos = ref({ x: window.innerWidth - props.width - 24, y: Math.max(16, (window.innerHeight - props.height) / 2 - 40) })
const size = ref({ w: props.width, h: props.height })
const maximized = ref(false)
const fullscreen = ref(false)

const style = computed(() => {
  if (fullscreen.value) {
    return { left: '0', top: '0', width: '100vw', height: '100vh', zIndex: 2000 }
  }
  if (maximized.value) {
    return { left: '8px', top: '8px', width: 'calc(100vw - 16px)', height: 'calc(100vh - 16px)', zIndex: 2000 }
  }
  return { left: `${pos.value.x}px`, top: `${pos.value.y}px`, width: `${size.value.w}px`, height: `${size.value.h}px`, zIndex: 2000 }
})

// 拖动
let dragStart: { x: number, y: number, px: number, py: number } | null = null
function onTitleDown(e: PointerEvent) {
  if (maximized.value || fullscreen.value) {
    return
  }
  dragStart = { x: e.clientX, y: e.clientY, px: pos.value.x, py: pos.value.y }
  window.addEventListener('pointermove', onDragMove)
  window.addEventListener('pointerup', onDragUp)
}
function onDragMove(e: PointerEvent) {
  if (!dragStart) {
    return
  }
  pos.value.x = Math.max(0, dragStart.px + e.clientX - dragStart.x)
  pos.value.y = Math.max(0, dragStart.py + e.clientY - dragStart.y)
}
function onDragUp() {
  dragStart = null
  window.removeEventListener('pointermove', onDragMove)
  window.removeEventListener('pointerup', onDragUp)
}

// 缩放手柄
let resizeStart: { x: number, y: number, w: number, h: number } | null = null
function onResizeDown(e: PointerEvent) {
  resizeStart = { x: e.clientX, y: e.clientY, w: size.value.w, h: size.value.h }
  window.addEventListener('pointermove', onResizeMove)
  window.addEventListener('pointerup', onResizeUp)
}
function onResizeMove(e: PointerEvent) {
  if (!resizeStart) {
    return
  }
  size.value.w = Math.max(props.minWidth, resizeStart.w + e.clientX - resizeStart.x)
  size.value.h = Math.max(props.minHeight, resizeStart.h + e.clientY - resizeStart.y)
}
function onResizeUp() {
  resizeStart = null
  window.removeEventListener('pointermove', onResizeMove)
  window.removeEventListener('pointerup', onResizeUp)
}

function toggleMaximize() {
  if (props.expandOnly) {
    emit('expand')
    return
  }
  maximized.value = !maximized.value
}
function toggleFullscreen() {
  if (props.expandOnly) {
    emit('expand')
    return
  }
  fullscreen.value = !fullscreen.value
}

function onEsc(e: KeyboardEvent) {
  if (e.key === 'Escape' && fullscreen.value) {
    fullscreen.value = false
  }
}
onMounted(() => window.addEventListener('keydown', onEsc))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onEsc)
  window.removeEventListener('pointermove', onDragMove)
  window.removeEventListener('pointerup', onDragUp)
  window.removeEventListener('pointermove', onResizeMove)
  window.removeEventListener('pointerup', onResizeUp)
})
</script>

<template>
  <div
    class="fixed flex flex-col overflow-hidden rounded-xl border bg-background shadow-2xl"
    :style="style"
  >
    <!-- 标题栏（拖动） -->
    <div
      class="flex cursor-move items-center justify-between border-b bg-muted/40 px-3 py-2 select-none"
      @pointerdown="onTitleDown"
    >
      <div class="flex items-center gap-2 text-sm font-medium">
        <FaIcon name="i-ri:sparkling-2-line" class="text-primary" />
        {{ title }}
      </div>
      <div class="flex items-center gap-1">
        <button
          v-if="maximizable"
          type="button"
          class="rounded p-1 text-muted-foreground hover:bg-accent"
          title="最大化"
          @click="toggleMaximize"
        >
          <FaIcon name="i-lucide:maximize-2" class="text-xs" />
        </button>
        <button
          v-if="fullscreenable"
          type="button"
          class="rounded p-1 text-muted-foreground hover:bg-accent"
          title="全屏"
          @click="toggleFullscreen"
        >
          <FaIcon name="i-lucide:expand" class="text-xs" />
        </button>
        <button
          type="button"
          class="rounded p-1 text-muted-foreground hover:bg-accent"
          title="关闭"
          @click="emit('close')"
        >
          <FaIcon name="i-lucide:x" class="text-xs" />
        </button>
      </div>
    </div>

    <!-- 主体插槽 -->
    <div class="flex min-h-0 flex-1 flex-col">
      <slot />
    </div>

    <!-- 缩放手柄 -->
    <span
      v-if="resizable && !maximized && !fullscreen"
      class="absolute right-0 bottom-0 size-4 cursor-nwse-resize"
      @pointerdown="onResizeDown"
    />
  </div>
</template>
