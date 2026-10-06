<script setup lang="ts">
import type * as Monaco from 'monaco-editor'
import type { FileEditorGroup } from '@/store/modules/fileEditor'
import { useFileEditorStore } from '@/store/modules/fileEditor'
import { Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import EditorTabs from './EditorTabs.vue'

// 单个编辑器组：tab 栏 + monaco 编辑器 + 欢迎页；viewState 按 tab 记忆。
const props = defineProps<{
  group: FileEditorGroup
  /** 初始占比（%） */
  size: number
}>()

const store = useFileEditorStore()

const editorRef = useTemplateRef<{ getEditor: () => Monaco.editor.IStandaloneCodeEditor | null }>('codeEditor')

const activeTab = computed(() => (props.group.activeTabId ? store.tabs[props.group.activeTabId] : undefined))
const activeModel = computed(() => (props.group.activeTabId ? store.getModel(props.group.activeTabId) : null))

// viewState 按 tab 记忆（同组内切换标签恢复滚动/光标）
const viewStates = new Map<string, Monaco.editor.ICodeEditorViewState | null>()

watch(() => props.group.activeTabId, (next, prev) => {
  const editor = editorRef.value?.getEditor()
  if (editor && prev) {
    viewStates.set(prev, editor.saveViewState())
  }
  if (editor && next) {
    const st = viewStates.get(next)
    if (st) {
      nextTick(() => editor.restoreViewState(st))
    }
  }
})

function onReady(editor: Monaco.editor.IStandaloneCodeEditor) {
  store.registerEditor(props.group.id, editor)
  const tabId = props.group.activeTabId
  if (tabId) {
    const st = viewStates.get(tabId)
    if (st) {
      editor.restoreViewState(st)
    }
  }
}

onBeforeUnmount(() => {
  store.registerEditor(props.group.id, null)
})

// 编辑区接收跨组拖入
function onDropToBody(e: DragEvent) {
  const raw = e.dataTransfer?.getData('application/x-ypanel-tab')
  if (!raw) {
    return
  }
  const { tabId, groupId } = JSON.parse(raw) as { tabId: string, groupId: number }
  if (groupId !== props.group.id) {
    store.moveTabToGroup(tabId, props.group.id)
  }
}
</script>

<template>
  <Pane :size="size" min-size="20" class="flex min-h-0 flex-col overflow-hidden">
    <!-- 组头：tab 栏 + 组操作 -->
    <div class="flex items-stretch" :class="store.groups.length > 1 ? '' : 'flex-col'">
      <EditorTabs :group="group" :active="store.activeGroupId === group.id" class="flex-1" />
      <div v-if="store.groups.length > 1" class="flex items-center border-b bg-muted/40 px-1">
        <FaButton variant="ghost" size="icon-sm" title="关闭此编辑器组" @click="store.closeGroup(group.id)">
          <FaIcon name="i-lucide:x" class="text-xs" />
        </FaButton>
      </div>
    </div>

    <!-- 截断提示 -->
    <div
      v-if="activeTab?.truncated"
      class="border-b border-amber-300 bg-amber-50 px-3 py-1 text-xs text-amber-700 dark:border-amber-700 dark:bg-amber-950/30 dark:text-amber-400"
    >
      文件较大，仅加载前 1 MiB；保存将覆盖整个文件，请谨慎操作。
    </div>

    <!-- 编辑器 / 欢迎页 -->
    <div
      v-show="activeTab"
      class="min-h-0 flex-1"
      @dragover.prevent
      @drop="onDropToBody"
    >
      <YdCodeEditor
        ref="codeEditor"
        :model="activeModel"
        :word-wrap="store.layout.wordWrap"
        :minimap="store.layout.minimap"
        @ready="onReady"
        @cursor="store.cursor = $event"
      />
    </div>

    <div v-if="!activeTab" class="flex flex-1 select-none flex-col items-center justify-center gap-3 text-muted-foreground" @dragover.prevent @drop="onDropToBody">
      <YdMorphIcon name="file-code" :size="56" class="opacity-30" />
      <div class="text-sm">
        从左侧文件树打开文件，或把标签拖到这里切分
      </div>
      <div class="space-y-1 text-center text-xs opacity-70">
        <div>Ctrl+S 保存 · Ctrl+\ 切分编辑器</div>
        <div>Ctrl+B 侧栏 · Ctrl+J 终端面板</div>
      </div>
    </div>
  </Pane>
</template>
