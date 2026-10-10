<script setup lang="ts">
// YdQuickDock：全局快速工作台浮层（M57）——右缘贴边按钮 + 滑入面板（文件树/编辑器/终端三合一）。
// 挂在 App.vue（RouterView 之外）：切路由不卸载；首次打开才挂载内容（everOpened），
// 最小化仅 v-show 隐藏（终端 WS 跨路由留存），关闭走 store.closeAll() 全量断开。
import WorkspaceContent from '@/views/file_management/editor/WorkspaceContent.vue'

defineOptions({ name: 'YdQuickDock' })

// 独立 store 实例（与页面内文件编辑弹窗的 fileEditor store 互不影响），
// 并 provide 给子树：WorkspaceContent 及其内部组件经 useFileEditorStore() 解析到本实例。
const store = useFileEditorDockStore()
provideFileEditorStore(store)

// ---- 面板宽度（vw）：左缘把手拖拽调整，localStorage 持久化 ----
const DOCK_W_KEY = 'ypanel-quickdock-width'
const dockWidth = ref(initWidth())

function initWidth() {
  const raw = Number(localStorage.getItem(DOCK_W_KEY))
  return Number.isFinite(raw) && raw >= 36 && raw <= 97 ? raw : 92
}

let dragging = false
function onHandleDown(ev: PointerEvent) {
  ev.preventDefault()
  dragging = true
  window.addEventListener('pointermove', onHandleMove)
  window.addEventListener('pointerup', onHandleUp)
}
function onHandleMove(ev: PointerEvent) {
  if (!dragging) {
    return
  }
  const vw = (1 - ev.clientX / window.innerWidth) * 100
  dockWidth.value = Math.min(97, Math.max(36, Math.round(vw * 10) / 10))
}
function onHandleUp() {
  dragging = false
  window.removeEventListener('pointermove', onHandleMove)
  window.removeEventListener('pointerup', onHandleUp)
  try {
    localStorage.setItem(DOCK_W_KEY, String(dockWidth.value))
  }
  catch {}
}
onBeforeUnmount(() => {
  window.removeEventListener('pointermove', onHandleMove)
  window.removeEventListener('pointerup', onHandleUp)
})

// 会话活动徽标：有打开的文件 tab 或终端会话时点亮
const hasSessions = computed(() => Object.keys(store.tabs).length > 0 || store.terminalSessions > 0)
</script>

<template>
  <div>
    <!-- 右缘贴边按钮：面板展开时隐藏 -->
    <button
      v-show="!store.visible"
      type="button"
      class="yd-dock-tab group fixed top-1/2 right-0 z-1600 flex -translate-y-1/2 cursor-pointer flex-col items-center gap-1.5 rounded-l-lg border border-r-0 bg-background px-1.5 py-3 text-foreground shadow-lg transition-colors hover:bg-accent"
      :title="$t('components.ydQuickDock.tip')"
      @click="store.openWorkspace()"
    >
      <span v-if="hasSessions" class="absolute top-1.5 right-1.5 size-1.5 rounded-full bg-primary" />
      <FaIcon name="i-lucide:folder-tree" class="text-base text-primary" />
      <span class="yd-dock-tab-text text-xs font-medium">{{ $t('components.ydQuickDock.label') }}</span>
    </button>

    <!-- 滑入面板：首次打开后常驻挂载（会话留存），显隐用 class 切换 + CSS transform——
         不用 Vue Transition：其完成依赖 rAF，遮挡/后台窗口下 leave 永不完成会卡在可见态（见 exp）；
         class 切换的隐藏语义（visibility/pointer-events）即时生效，滑入动画仅作视觉增强 -->
    <template v-if="store.everOpened">
      <section
        class="yd-dock-panel fixed inset-y-[3vh] right-0 z-1600 flex flex-col overflow-hidden border-l bg-background shadow-2xl"
        :class="store.visible ? 'yd-dock-panel-open' : 'yd-dock-panel-closed'"
        :style="{ width: `${dockWidth}vw` }"
      >
        <div
          class="absolute inset-y-0 left-0 z-10 w-1 cursor-col-resize bg-transparent transition-colors hover:bg-primary/40"
          :title="$t('components.ydQuickDock.resizeTip')"
          @pointerdown="onHandleDown"
        />
        <WorkspaceContent />
      </section>
    </template>
  </div>
</template>

<style scoped>
.yd-dock-tab-text {
  writing-mode: vertical-rl;
  letter-spacing: 0.2em;
}

.yd-dock-panel {
  transition: transform 0.3s ease;
}

.yd-dock-panel-closed {
  transform: translateX(102%);
  visibility: hidden;
  pointer-events: none;
}
</style>
