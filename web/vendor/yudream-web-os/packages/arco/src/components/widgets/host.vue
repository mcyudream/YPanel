<script setup lang="ts">
import type { WidgetInstance } from '@yudream/yudream-webos-core'
import { useWebOS, useWidgets } from '@yudream/yudream-webos-vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

/**
 * YwWidgetHost — 小组件宿主，按 placement 双形态：
 * - desktop：卡片按各自跨度（2x2/4x2/4x4）直接铺在桌面上，编辑模式抖动 + 拖拽换位 + 尺寸切换，无滚动条；
 * - sidebar（默认）：桌面右侧小组件栏（macOS 15 风格），列内滚动。
 * 编辑入口：桌面右键「编辑小组件」（webos:widgets:edit 事件）或侧栏底部按钮。
 */
const emit = defineEmits<{
  openGallery: []
}>()

const { widgets, instances, editing, setEditing, remove, move, resize } = useWidgets()
const os = useWebOS()

const isDesktop = computed(() => widgets.placement === 'desktop')
const hasInstances = computed(() => instances.value.length > 0)

/* ── 桌面平铺：网格常量与布局 ── */
const UNIT = 84
const GAP = 8
const STEP = UNIT + GAP
/** 卡片首行顶部 = 菜单栏高 + 呼吸间距（与层 inset 一致） */
const WIDGET_TOP = 38
const SPAN: Record<string, [number, number]> = { small: [2, 2], medium: [4, 2], large: [4, 4] }

function spanOf(inst: WidgetInstance): [number, number] {
  return (SPAN[inst.size] ?? [2, 2]) as [number, number]
}

function cardStyle(inst: WidgetInstance) {
  const [w, h] = spanOf(inst)
  return {
    left: `${inst.position.col * STEP}px`,
    top: `${inst.position.row * STEP}px`,
    width: `${w * UNIT + (w - 1) * GAP}px`,
    height: `${h * UNIT + (h - 1) * GAP}px`,
  }
}

/* 拖拽换位（仅编辑模式）：pointer capture + 抬起时吸附网格 */
const dragId = ref<string | null>(null)
const dragStart = ref({ x: 0, y: 0, left: 0, top: 0 })

function nextSize(inst: WidgetInstance): string | null {
  const def = widgets.getDefinition(inst.widgetId)
  const sizes = def?.sizes ?? []
  if (sizes.length < 2) {
    return null
  }
  return sizes[(sizes.indexOf(inst.size) + 1) % sizes.length] ?? null
}

function cycleSize(inst: WidgetInstance) {
  const next = nextSize(inst)
  if (next) {
    resize(inst.instanceId, next as typeof inst.size)
  }
}

function onCardContextmenu(inst: WidgetInstance, e: MouseEvent) {
  e.preventDefault()
  const next = nextSize(inst)
  os.ui.menu({
    x: e.clientX,
    y: e.clientY,
    items: [
      ...(next ? [{ label: `切换为${next === 'small' ? '小' : next === 'medium' ? '中' : '大'}尺寸`, icon: 'i-lucide-scaling', onSelect: () => resize(inst.instanceId, next as typeof inst.size) }] : []),
      { label: '移除小组件', icon: 'i-lucide-trash-2', danger: true, onSelect: () => remove(inst.instanceId) },
    ],
  })
}

function onDragStart(inst: WidgetInstance, e: PointerEvent) {
  if (!editing.value || e.button !== 0) {
    return
  }
  const el = e.currentTarget as HTMLElement
  const r = el.getBoundingClientRect()
  dragId.value = inst.instanceId
  dragStart.value = { x: e.clientX, y: e.clientY, left: r.left, top: r.top }
  el.setPointerCapture(e.pointerId)
}

function onDragMove(inst: WidgetInstance, e: PointerEvent) {
  if (dragId.value !== inst.instanceId) {
    return
  }
  const el = e.currentTarget as HTMLElement
  const dx = e.clientX - dragStart.value.x
  const dy = e.clientY - dragStart.value.y
  el.style.transform = `translate(${dx}px, ${dy})`
}

function onDragEnd(inst: WidgetInstance, e: PointerEvent) {
  if (dragId.value !== inst.instanceId) {
    return
  }
  dragId.value = null
  const el = e.currentTarget as HTMLElement
  el.style.transform = ''
  const [w, h] = spanOf(inst)
  const dx = e.clientX - dragStart.value.x
  const dy = e.clientY - dragStart.value.y
  const newLeft = dragStart.value.left + dx
  const newTop = dragStart.value.top + dy
  const maxCol = Math.max(0, Math.floor((window.innerWidth - w * UNIT - (w - 1) * GAP) / STEP))
  const maxRow = Math.max(0, Math.floor((window.innerHeight - WIDGET_TOP - h * UNIT - (h - 1) * GAP) / STEP))
  const col = Math.min(maxCol, Math.max(0, Math.round(newLeft / STEP)))
  const row = Math.min(maxRow, Math.max(0, Math.round((newTop - WIDGET_TOP) / STEP)))
  move(inst.instanceId, { col, row })
}

function toggleEdit() {
  setEditing(!editing.value)
}

onMounted(() => {
  window.addEventListener('webos:widgets:edit', toggleEdit)
})

onBeforeUnmount(() => {
  window.removeEventListener('webos:widgets:edit', toggleEdit)
})
</script>

<template>
  <!-- 桌面平铺形态：卡片铺在桌面上，编辑模式拖拽换位/移除/切换尺寸 -->
  <aside v-if="isDesktop" class="yw-widget-desktop" :data-editing="editing">
    <div
      v-for="inst in instances"
      :key="inst.instanceId"
      class="yw-widget-card is-desktop"
      :class="[`is-${inst.size}`, { 'is-editing': editing, 'is-dragging': dragId === inst.instanceId }]"
      :style="cardStyle(inst)"
      @pointerdown="onDragStart(inst, $event)"
      @pointermove="onDragMove(inst, $event)"
      @pointerup="onDragEnd(inst, $event)"
      @contextmenu="onCardContextmenu(inst, $event)"
    >
      <button v-if="editing" class="yw-widget-remove" title="移除" @click="remove(inst.instanceId)">
        <i class="i-lucide-minus" />
      </button>
      <button
        v-if="editing && nextSize(inst)"
        class="yw-widget-resize"
        title="切换尺寸"
        @click="cycleSize(inst)"
      >
        <i class="i-lucide-scaling" />
      </button>
      <div class="yw-widget-body">
        <slot name="widget" :instance="inst" />
      </div>
    </div>

    <button v-if="editing" class="yw-widget-edit-btn yw-widget-desktop-done" @click="toggleEdit()">
      完成
    </button>
    <button v-if="editing" class="yw-widget-edit-btn yw-widget-desktop-add" @click="emit('openGallery')">
      <i class="i-lucide-plus" />
      添加小组件
    </button>
  </aside>

  <!-- 侧栏形态（macOS 15 风格） -->
  <aside v-else class="yw-widget-host" data-editing="false">
    <div class="yw-widget-list">
      <p v-if="!hasInstances" class="yw-widget-empty">
        {{ editing ? '从下方画廊添加小组件' : '小组件将显示在这里' }}
      </p>
      <div
        v-for="inst in instances"
        :key="inst.instanceId"
        class="yw-widget-card"
        :class="[`is-${inst.size}`, { 'is-editing': editing }]"
      >
        <button v-if="editing" class="yw-widget-remove" title="移除" @click="remove(inst.instanceId)">
          <i class="i-lucide-minus" />
        </button>
        <div class="yw-widget-body">
          <slot name="widget" :instance="inst" />
        </div>
      </div>
    </div>

    <div v-if="editing" class="yw-widget-gallery-hint">
      <slot name="gallery" />
    </div>

    <footer class="yw-widget-host-footer">
      <button v-if="editing" class="yw-widget-edit-btn" @click="emit('openGallery')">
        <i class="i-lucide-plus" />
        添加小组件
      </button>
      <button class="yw-widget-edit-btn" @click="setEditing(!editing); if (!editing) emit('openGallery')">
        {{ editing ? '完成' : '编辑小组件' }}
      </button>
    </footer>
  </aside>
</template>

<style scoped>
.yw-widget-host {
  position: absolute;
  top: calc(var(--yw-menubar-h) + 14px);
  bottom: 14px;
  left: 14px;
  z-index: 10;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 180px;
  overflow-y: auto;

  /* 容器穿透：只有卡片与按钮可点，不挡桌面图标双击 */
  pointer-events: none;
}

.yw-widget-host .yw-widget-card,
.yw-widget-host .yw-widget-edit-btn,
.yw-widget-host .yw-widget-gallery-hint {
  pointer-events: auto;
}

.yw-widget-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.yw-widget-empty {
  padding: 16px;
  font-size: 12px;
  color: #fff;
  text-align: center;
  white-space: pre-line;
  text-shadow: 0 1px 3px rgb(0 0 0 / 50%);
}

.yw-widget-card {
  position: relative;
  min-height: 150px;
  overflow: hidden;
  background: var(--yw-control-bg);
  border-radius: var(--yw-radius-widget);
  box-shadow:
    0 0 0 0.5px var(--yw-separator),
    0 8px 28px rgb(0 0 0 / 22%),
    0 2px 8px rgb(0 0 0 / 12%);
}

.yw-widget-card.is-editing {
  animation: yw-widget-jiggle 0.3s ease-in-out infinite alternate;
}

@keyframes yw-widget-jiggle {
  from { transform: rotate(-0.6deg); }
  to { transform: rotate(0.6deg); }
}

.yw-widget-remove {
  position: absolute;
  top: 6px;
  left: 6px;
  z-index: 2;
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  font-size: 13px;
  color: #fff;
  cursor: pointer;
  background: rgb(0 0 0 / 55%);
  border: none;
  border-radius: 50%;
}

.yw-widget-body {
  height: 100%;
  padding: 12px;
}

.yw-widget-gallery-hint {
  min-height: 0;
}

.yw-widget-host-footer {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.yw-widget-edit-btn {
  display: flex;
  gap: 6px;
  align-items: center;
  justify-content: center;
  padding: 6px 12px;
  font-size: 12px;
  color: #fff;
  cursor: default;
  background: rgb(0 0 0 / 35%);
  border: none;
  border-radius: var(--yw-radius-capsule);
  backdrop-filter: blur(var(--yw-blur-light));
}

.yw-widget-edit-btn:hover {
  background: rgb(0 0 0 / 50%);
}

/* ── 桌面平铺形态 ── */
.yw-widget-desktop {
  position: absolute;
  inset: calc(var(--yw-menubar-h) + 14px) 0 0;
  z-index: 5;

  /* 层穿透：卡片本身可交互，不挡桌面图标与右键 */
  pointer-events: none;
}

.yw-widget-desktop .yw-widget-card {
  position: absolute;
  pointer-events: auto;
}

.yw-widget-desktop .yw-widget-card.is-editing {
  cursor: grab;
}

.yw-widget-desktop .yw-widget-card.is-dragging {
  cursor: grabbing;
  opacity: 0.85;
  animation: none;
}

.yw-widget-desktop .yw-widget-remove {
  right: 6px;
  left: auto;
}

.yw-widget-resize {
  position: absolute;
  top: 6px;
  left: 6px;
  z-index: 2;
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  font-size: 12px;
  color: #fff;
  cursor: pointer;
  background: rgb(0 0 0 / 55%);
  border: none;
  border-radius: 6px;
}

.yw-widget-desktop-done {
  position: absolute;
  bottom: 18px;
  left: 50%;
  pointer-events: auto;
  transform: translateX(-50%);
}

.yw-widget-desktop-add {
  position: absolute;
  right: 18px;
  bottom: 18px;
  pointer-events: auto;
}
</style>
