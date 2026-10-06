<script setup lang="ts">
import type { SplitpanesResizedPayload } from 'splitpanes'
import { Pane, Splitpanes } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import EditorGroup from './EditorGroup.vue'
import EditorToolbar from './EditorToolbar.vue'
import FileTreePanel from './FileTreePanel.vue'
import StatusBar from './StatusBar.vue'
import TerminalPanel from './TerminalPanel.vue'

// 文件编辑工作台：FaModal 近全屏弹窗，VS Code 式布局（侧栏 | 编辑器组 | 终端 | 状态栏）。
const store = useFileEditorStore()

// 侧栏宽度 / 终端高度拖拽回写
function onOuterResized(e: SplitpanesResizedPayload) {
  const idx = store.layout.sidebarSide === 'left' ? 0 : e.panes.length - 1
  if (store.layout.sidebarVisible && e.panes[idx]) {
    store.layout.sidebarWidth = Math.round(e.panes[idx].size * 10) / 10
  }
}

function onContentResized(e: SplitpanesResizedPayload) {
  const last = e.panes[e.panes.length - 1]
  if (last) {
    store.layout.terminalSize = Math.round(last.size * 10) / 10
  }
}

// ---- 全局快捷键（仅工作台打开时） ----
function onKeydown(e: KeyboardEvent) {
  if (!store.visible || !(e.ctrlKey || e.metaKey)) {
    return
  }
  const k = e.key.toLowerCase()
  if (k === 's') {
    e.preventDefault()
    if (e.shiftKey) {
      store.saveAll()
    }
    else if (store.activeTabId) {
      store.save(store.activeTabId)
    }
  }
  else if (k === 'b') {
    e.preventDefault()
    store.toggleSidebar()
  }
  else if (k === 'j') {
    e.preventDefault()
    store.toggleTerminal()
  }
  else if (k === '\\') {
    e.preventDefault()
    if (store.activeTabId) {
      store.splitFromTab(store.activeTabId)
    }
  }
  else if (k === 'f') {
    e.preventDefault()
    store.runActiveAction('actions.find')
  }
  else if (k === 'h') {
    e.preventDefault()
    store.runActiveAction('actions.replace')
  }
}

watch(() => store.visible, (v) => {
  if (v) {
    document.addEventListener('keydown', onKeydown, true)
  }
  else {
    document.removeEventListener('keydown', onKeydown, true)
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown, true)
})
</script>

<template>
  <FaModal
    v-model="store.visible"
    :header="false"
    :footer="false"
    :closable="false"
    :close-on-press-escape="false"
    :close-on-click-overlay="false"
    :border="false"
    :draggable="false"
    class="h-[94vh]! max-h-[94vh]! w-[97vw]! max-w-[97vw]! sm:top-[3vh]!"
    content-class="p-0! min-h-0! flex-1 overflow-hidden"
    :destroy-on-close="true"
    @opened="store.editorOpened = true"
    @close="store.editorOpened = false"
  >
    <div class="flex h-full flex-col overflow-hidden bg-background text-foreground">
      <EditorToolbar />

      <!-- 主区域：侧栏 | 内容 -->
      <div class="min-h-0 flex-1">
        <Splitpanes class="h-full" @resized="onOuterResized">
          <FileTreePanel v-if="store.layout.sidebarVisible && store.layout.sidebarSide === 'left'" />
          <Pane :size="store.layout.sidebarVisible ? 100 - store.layout.sidebarWidth : 100" min-size="30" class="min-h-0 min-w-0">
            <!-- 内容：编辑器组 + 终端 -->
            <Splitpanes v-if="store.layout.terminalVisible" class="h-full" :horizontal="store.layout.terminalSide === 'bottom'" @resized="onContentResized">
              <Pane :size="100 - store.layout.terminalSize" min-size="20" class="min-h-0 min-w-0">
                <Splitpanes class="h-full">
                  <EditorGroup
                    v-for="g in store.groups"
                    :key="g.id"
                    :group="g"
                    :size="100 / store.groups.length"
                  />
                </Splitpanes>
              </Pane>
              <TerminalPanel :horizontal="store.layout.terminalSide === 'bottom'" />
            </Splitpanes>
            <Splitpanes v-else class="h-full">
              <EditorGroup
                v-for="g in store.groups"
                :key="g.id"
                :group="g"
                :size="100 / store.groups.length"
              />
            </Splitpanes>
          </Pane>
          <FileTreePanel v-if="store.layout.sidebarVisible && store.layout.sidebarSide === 'right'" />
        </Splitpanes>
      </div>

      <StatusBar />
    </div>
  </FaModal>
</template>
