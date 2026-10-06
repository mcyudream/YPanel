<script setup lang="ts">
// 桌面工作台：组件内窗口层（拖拽/最小化/最大化/任务栏）。
// 模式偏好记忆：localStorage（刷新后按偏好进入）。
import { defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import { useWorkbenchStore } from '@/store/modules/app/workbench'

defineOptions({
  name: 'DesktopIndex',
})

const router = useRouter()
const wb = useWorkbenchStore()

// 窗口内容组件映射（懒加载复用经典面板页面组件）
const appLoaders: Record<string, () => Promise<any>> = {
  overview: () => import('@/views/overview/index.vue'),
  file: () => import('@/views/file_management/index.vue'),
  container: () => import('@/views/container/index.vue'),
  terminal: () => import('@/views/terminal/index.vue'),
  database: () => import('@/views/database/index.vue'),
  sites: () => import('@/views/sites/index.vue'),
  compose: () => import('@/views/compose/index.vue'),
  cron: () => import('@/views/cron/index.vue'),
  icons: () => import('@/views/icons/index.vue'),
}

// defineAsyncComponent 包装（缓存，避免每次渲染重建组件）
const asyncCompCache: Record<string, ReturnType<typeof defineAsyncComponent>> = {}
function compOf(key: string) {
  if (!asyncCompCache[key]) {
    asyncCompCache[key] = defineAsyncComponent(appLoaders[key])
  }
  return asyncCompCache[key]
}

// ---- 拖拽 ----
const dragState = ref<{
  id: number
  startX: number
  startY: number
  originX: number
  originY: number
} | null>(null)

function onTitlebarMousedown(e: MouseEvent, id: number) {
  const w = wb.windows.find(i => i.id === id)
  if (!w || w.maximized) {
    return
  }
  wb.focus(id)
  dragState.value = {
    id,
    startX: e.clientX,
    startY: e.clientY,
    originX: w.x,
    originY: w.y,
  }
  e.preventDefault()
}

function onMousemove(e: MouseEvent) {
  const d = dragState.value
  if (!d) {
    return
  }
  const w = wb.windows.find(i => i.id === d.id)
  if (!w) {
    return
  }
  const nx = d.originX + (e.clientX - d.startX)
  const ny = d.originY + (e.clientY - d.startY)
  wb.move(d.id, Math.max(-w.width + 120, Math.min(window.innerWidth - 120, nx)), Math.max(0, Math.min(window.innerHeight - 80, ny)))
}

function onMouseup() {
  dragState.value = null
}

onMounted(() => {
  window.addEventListener('mousemove', onMousemove)
  window.addEventListener('mouseup', onMouseup)
  // 默认打开主机概览
  if (!wb.windows.length) {
    wb.open('overview')
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', onMousemove)
  window.removeEventListener('mouseup', onMouseup)
})

function backToClassic() {
  localStorage.setItem('ypanel.mode', 'classic')
  router.push('/')
}
</script>

<template>
  <div class="fixed inset-0 z-50 select-none overflow-hidden bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900">
    <!-- 顶栏：品牌 + 返回 -->
    <div class="absolute inset-x-0 top-0 z-[9998] flex h-12 items-center justify-between bg-black/30 px-4 text-white backdrop-blur">
      <div class="flex items-center gap-2">
        <YdMorphIcon name="layout-grid" :size="18" />
        <span class="text-sm font-medium">YPanel 桌面工作台</span>
      </div>
      <FaButton variant="ghost" size="sm" class="text-white!" @click="backToClassic">
        <YdMorphIcon name="panel-left" :size="14" class="mr-1" /> 返回经典面板
      </FaButton>
    </div>

    <!-- 图标网格 -->
    <div class="absolute inset-x-0 top-14 z-0 grid max-w-max grid-cols-4 gap-4 p-6 sm:grid-cols-6">
      <button
        v-for="app in wb.apps"
        :key="app.key"
        type="button"
        class="flex w-20 cursor-pointer flex-col items-center gap-1.5 rounded-lg p-2 text-white transition-colors hover:bg-white/10"
        @dblclick="wb.open(app.key)"
        @click="wb.open(app.key)"
      >
        <YdMorphIcon :name="app.icon" :size="34" color="#ffffff" :stroke-width="1.6" />
        <span class="text-center text-xs leading-tight">{{ app.title }}</span>
      </button>
    </div>

    <!-- 窗口层 -->
    <template v-for="w in wb.windows" :key="w.id">
      <div
        v-show="!w.minimized"
        class="absolute flex flex-col overflow-hidden rounded-lg border border-white/10 bg-background shadow-2xl"
        :class="wb.focusedId === w.id ? 'ring-1 ring-primary' : 'opacity-95'"
        :style="w.maximized
          ? { left: '0px', top: '48px', width: '100%', height: 'calc(100% - 48px - 48px)', zIndex: w.z }
          : { left: `${w.x}px`, top: `${w.y}px`, width: `${w.width}px`, height: `${w.height}px`, zIndex: w.z }"
        @mousedown="wb.focus(w.id)"
      >
        <!-- 标题栏 -->
        <div
          class="flex h-9 shrink-0 cursor-move items-center justify-between border-b bg-muted/60 px-2"
          @mousedown="onTitlebarMousedown($event, w.id)"
          @dblclick="wb.toggleMaximize(w.id)"
        >
          <div class="flex items-center gap-2 px-1 text-sm font-medium">
            <YdMorphIcon :name="w.icon" :size="15" />
            {{ w.title }}
          </div>
          <div class="flex items-center gap-0.5">
            <button
              type="button"
              class="flex size-7 cursor-pointer items-center justify-center rounded transition-colors hover:bg-accent"
              title="最小化"
              @mousedown.stop
              @click="wb.toggleMinimize(w.id)"
            >
              <FaIcon name="i-lucide:minus" class="text-xs" />
            </button>
            <button
              type="button"
              class="flex size-7 cursor-pointer items-center justify-center rounded transition-colors hover:bg-accent"
              :title="w.maximized ? '还原' : '最大化'"
              @mousedown.stop
              @click="wb.toggleMaximize(w.id)"
            >
              <FaIcon :name="w.maximized ? 'i-lucide:minimize-2' : 'i-lucide:maximize-2'" class="text-xs" />
            </button>
            <button
              type="button"
              class="flex size-7 cursor-pointer items-center justify-center rounded transition-colors hover:bg-red-500/20"
              title="关闭"
              @mousedown.stop
              @click="wb.close(w.id)"
            >
              <FaIcon name="i-lucide:x" class="text-xs" />
            </button>
          </div>
        </div>
        <!-- 内容 -->
        <div class="min-h-0 flex-1 overflow-auto bg-background">
          <component :is="compOf(w.key)" />
        </div>
      </div>
    </template>

    <!-- 任务栏 -->
    <div class="absolute inset-x-0 bottom-0 z-[9998] flex h-12 items-center gap-1 border-t border-white/10 bg-black/40 px-2 backdrop-blur">
      <button
        v-for="app in wb.apps"
        :key="app.key"
        type="button"
        class="flex size-10 cursor-pointer items-center justify-center rounded-md text-white/80 transition-colors hover:bg-white/10"
        :title="app.title"
        @click="wb.open(app.key)"
      >
        <YdMorphIcon :name="app.icon" :size="18" color="#ffffff" />
      </button>
      <div class="mx-2 h-6 w-px bg-white/20" />
      <button
        v-for="w in wb.windows"
        :key="w.id"
        type="button"
        class="flex h-10 cursor-pointer items-center gap-1.5 rounded-md px-2.5 text-xs text-white/90 transition-colors"
        :class="wb.focusedId === w.id && !w.minimized ? 'bg-white/20' : 'hover:bg-white/10'"
        @click="w.minimized ? wb.toggleMinimize(w.id) : wb.focus(w.id)"
      >
        <YdMorphIcon :name="w.icon" :size="14" color="#ffffff" />
        {{ w.title }}
      </button>
    </div>
  </div>
</template>
