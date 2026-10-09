<script setup lang="ts">
import type { AppDefinition } from '@yudream/yudream-webos-core'
import { useWebOS } from '@yudream/yudream-webos-vue'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useAppsStore } from '../../stores/apps'
import YwIconTile from '../icon-tile/index.vue'

const props = withDefaults(defineProps<{
  /** 每页行数 */
  rows?: number
  /** 每行列数 */
  columns?: number
}>(), {
  rows: 4,
  columns: 7,
})

const emit = defineEmits<{
  close: []
}>()

const appsStore = useAppsStore()
const os = useWebOS()

const keyword = ref('')
const page = ref(0)

/** 过滤后的应用（title/keywords/category 匹配） */
const filtered = computed<AppDefinition[]>(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) {
    return appsStore.launchpad
  }
  return appsStore.launchpad.filter(a =>
    a.name.toLowerCase().includes(kw)
    || (a.keywords ?? '').toLowerCase().includes(kw)
    || a.category?.toLowerCase().includes(kw))
})

/** 分页 */
const pages = computed<AppDefinition[][]>(() => {
  const size = props.rows * props.columns
  const out: AppDefinition[][] = []
  for (let i = 0; i < filtered.value.length; i += size) {
    out.push(filtered.value.slice(i, i + size))
  }
  return out.length ? out : [[]]
})

watch(filtered, () => {
  page.value = 0
})

function open(app: AppDefinition) {
  appsStore.openApp(app.id)
  emit('close')
}

/** 右键：打开 / 添加到桌面（落点由 DesktopModel 自动找可见空位） */
function onAppContextmenu(ev: MouseEvent, app: AppDefinition) {
  ev.preventDefault()
  os.ui.menu({
    x: ev.clientX,
    y: ev.clientY,
    items: [
      { label: '打开', icon: 'i-lucide-external-link', onSelect: () => open(app) },
      {
        label: '添加到桌面',
        icon: 'i-lucide-monitor-plus',
        onSelect: () => {
          os.desktop.add({ type: 'app', refId: app.id, name: app.name, icon: app.icon, position: { col: 0, row: 0 } })
          os.ui.message('success', `«${app.name}» 已添加到桌面`)
        },
      },
    ],
  })
}

const inputEl = ref<HTMLInputElement | null>(null)
onMounted(() => inputEl.value?.focus())

function onKeydown(ev: KeyboardEvent) {
  if (ev.key === 'Escape') {
    emit('close')
  }
  else if (ev.key === 'ArrowRight' && page.value < pages.value.length - 1) {
    page.value++
  }
  else if (ev.key === 'ArrowLeft' && page.value > 0) {
    page.value--
  }
}

/* ── 滚轮 / 横向拖动换页 ── */
let wheelLockUntil = 0

function onWheel(ev: WheelEvent) {
  const now = Date.now()
  if (now < wheelLockUntil || pages.value.length < 2) {
    return
  }
  const d = Math.abs(ev.deltaX) > Math.abs(ev.deltaY) ? ev.deltaX : ev.deltaY
  if (Math.abs(d) < 24) {
    return
  }
  wheelLockUntil = now + 450
  if (d > 0 && page.value < pages.value.length - 1) {
    page.value++
  }
  else if (d < 0 && page.value > 0) {
    page.value--
  }
}

/** 横向拖动换页：超过阈值即翻页，并在 click 捕获段吞掉这次点击（防止误开应用） */
const swipe = reactive({ active: false, x: 0, y: 0, done: false })
let suppressClick = false

function onGridPointerDown(ev: PointerEvent) {
  if (ev.button !== 0) {
    return
  }
  swipe.active = true
  swipe.x = ev.clientX
  swipe.y = ev.clientY
  swipe.done = false
}

function onGridPointerMove(ev: PointerEvent) {
  if (!swipe.active) {
    return
  }
  const dx = ev.clientX - swipe.x
  const dy = ev.clientY - swipe.y
  if (Math.abs(dx) > 48 && Math.abs(dx) > Math.abs(dy) * 1.4) {
    swipe.done = true
  }
}

function onGridPointerUp(ev: PointerEvent) {
  if (!swipe.active) {
    return
  }
  const dx = ev.clientX - swipe.x
  swipe.active = false
  if (!swipe.done) {
    return
  }
  suppressClick = true
  setTimeout(() => {
    suppressClick = false
  }, 100)
  if (dx < 0 && page.value < pages.value.length - 1) {
    page.value++
  }
  else if (dx > 0 && page.value > 0) {
    page.value--
  }
}

function onClickCapture(ev: MouseEvent) {
  if (suppressClick) {
    ev.stopPropagation()
    ev.preventDefault()
    suppressClick = false
  }
}
</script>

<template>
  <div
    class="yw-launchpad"
    role="dialog"
    aria-label="启动台"
    @pointerdown.self="emit('close')"
    @keydown="onKeydown"
    @wheel="onWheel"
  >
    <input
      ref="inputEl"
      v-model="keyword"
      class="yw-launchpad-search"
      placeholder="搜索应用…"
    >

    <Transition name="yw-lp-page" mode="out-in">
      <div
        :key="page"
        class="yw-launchpad-grid"
        :style="{ gridTemplateColumns: `repeat(${columns}, 1fr)`, gridTemplateRows: `repeat(${rows}, 1fr)` }"
        @pointerdown="onGridPointerDown"
        @pointermove="onGridPointerMove"
        @pointerup="onGridPointerUp"
        @pointercancel="onGridPointerUp"
        @click.capture="onClickCapture"
      >
        <button
          v-for="app in pages[page]"
          :key="app.id"
          class="yw-launchpad-app"
          @click="open(app)"
          @contextmenu.prevent="onAppContextmenu($event, app)"
        >
          <YwIconTile :app-key="app.id" :icon="app.icon" :icon-bg="app.iconBg" :size="64" />
          <span class="yw-launchpad-label">{{ app.name }}</span>
        </button>
      </div>
    </Transition>

    <div v-if="pages.length > 1" class="yw-launchpad-dots">
      <button
        v-for="(_, i) in pages"
        :key="i"
        class="yw-launchpad-dot"
        :class="{ 'is-active': i === page }"
        @click="page = i"
      />
    </div>
  </div>
</template>

<style scoped>
.yw-launchpad {
  position: fixed;
  inset: 0;
  z-index: 9400;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48px 8vw 24px;
  background: rgb(0 0 0 / 45%);
  backdrop-filter: blur(30px) saturate(1.3);
  animation: yw-lp-in 0.18s ease;
}

@keyframes yw-lp-in {
  from {
    opacity: 0;
    transform: scale(1.04);
  }

  to {
    opacity: 1;
    transform: none;
  }
}

.yw-launchpad-search {
  box-sizing: border-box;
  width: 280px;
  padding: 8px 16px;
  margin-bottom: 40px;
  font-size: 14px;

  /* 恒定浅色实底 + 深字：任何壁纸下都清晰（macOS 启动台行为） */
  color: #1d1d1f;
  text-align: center;
  outline: none;
  background: rgb(255 255 255 / 92%);
  border: none;
  border-radius: 999px;
  box-shadow: 0 4px 16px rgb(0 0 0 / 25%);
}

.yw-launchpad-search::placeholder {
  color: rgb(60 60 67 / 45%);
}

.yw-launchpad-grid {
  display: grid;
  flex: 1;
  gap: 8px;
  place-items: start center;
  width: 100%;
}

.yw-launchpad-app {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: center;
  padding: 12px;
  cursor: default;
  background: transparent;
  border: none;
  border-radius: var(--yw-radius-md);
  transition: transform 0.15s ease, background-color 0.15s ease;
}

.yw-launchpad-app:hover {
  background: oklch(var(--yw-accent) / 40%);
  transform: scale(1.05);
}

.yw-launchpad-icon {
  font-size: 52px;
  color: rgb(255 255 255 / 92%);
  filter: drop-shadow(0 1px 3px rgb(0 0 0 / 45%));
}

.yw-launchpad-app img {
  object-fit: contain;
  border-radius: 14px;
}

.yw-launchpad-label {
  max-width: 110px;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 13px;

  /* 恒白 + 投影：模糊壁纸上任何区域都清晰 */
  color: #fff;
  white-space: nowrap;
  text-shadow:
    0 1px 3px rgb(0 0 0 / 70%),
    0 0 8px rgb(0 0 0 / 35%);
}

.yw-launchpad-dots {
  display: flex;
  gap: 10px;
  margin-top: 16px;
}

.yw-launchpad-dot {
  width: 8px;
  height: 8px;
  padding: 0;
  cursor: pointer;
  background: oklch(var(--yw-foreground) / 30%);
  border: none;
  border-radius: 50%;
}

.yw-launchpad-dot.is-active {
  background: oklch(var(--yw-foreground) / 85%);
}

.yw-lp-page-enter-active,
.yw-lp-page-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.yw-lp-page-enter-from {
  opacity: 0;
  transform: translateX(24px);
}

.yw-lp-page-leave-to {
  opacity: 0;
  transform: translateX(-24px);
}
</style>
