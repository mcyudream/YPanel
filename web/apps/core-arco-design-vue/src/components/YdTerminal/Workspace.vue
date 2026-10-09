<script setup lang="ts">
import type { SplitpanesResizedPayload } from 'splitpanes'
import type { NodeItem } from '@/api/modules/node'
import type { FileEntry } from '@/api/modules/file'
import apiNode from '@/api/modules/node'
import { useFileEditorStore } from '@/store/modules/fileEditor'
import { Pane, Splitpanes } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import { useLocalStorage } from '@vueuse/core'
import { ref } from 'vue'
import YdFileTree from '../YdFileTree/index.vue'
import YdMonitorSidebar from '../YdMonitorSidebar/index.vue'
import Panel from './Panel.vue'

// 终端工作台：VS Code 式三栏（左文件树 | 中终端面板 | 右监控），splitpanes 可拖宽、可折叠，布局持久化。
defineOptions({
  name: 'YdTerminalWorkspace',
})

const fileEditorStore = useFileEditorStore()
const panelRef = ref<InstanceType<typeof Panel>>()
const initialPaste = defineModel<string>('initialPaste', { default: '' })
/** 拖入数据的 MIME（文件管理器 dragstart 约定） */
const DRAG_MIME = 'application/x-ypanel-files'
/** 拖拽悬停高亮（终端区可视反馈） */
const dropHover = ref(false)

/** shell 引号包裹：含空白或特殊字符的路径用双引号 */
function quotePath(path: string): string {
  return /[\s';]/.test(path) ? `"${path.replaceAll('"', '\"')}"` : path
}

/** 拖入文件 → 活跃终端逐个填入路径（空格分隔，不执行） */
function onDropToTerminal(e: DragEvent) {
  dropHover.value = false
  // 文件拖入：填入引号包裹的路径（不执行）
  const raw = e.dataTransfer?.getData(DRAG_MIME)
  if (raw) {
    try {
      const payload = JSON.parse(raw) as { items: Array<{ path: string, name: string, isDir: boolean }> }
      const text = payload.items.map(x => quotePath(x.path)).join(' ')
      if (text) {
        panelRef.value?.pasteText(text)
      }
    }
    catch {}
    return
  }
  // 文本拖入（iTerm2 风格）：原样输入到活跃终端
  const text = e.dataTransfer?.getData('text/plain')
  if (text) {
    panelRef.value?.pasteText(text)
  }
}

const nodes = ref<NodeItem[]>([])
const node = useLocalStorage('ypanel.terminal.node', 'local')
const treeVisible = useLocalStorage('ypanel.terminal.tree', true)
const monitorVisible = useLocalStorage('ypanel.terminal.monitor', true)
const treeWidth = useLocalStorage('ypanel.terminal.treeWidth', 18)
const monitorWidth = useLocalStorage('ypanel.terminal.monitorWidth', 22)
/** 活动终端会话的当前目录（跟随开关在 Panel 内，空串=不跟随） */
const termCwd = ref('')

onMounted(async () => {
  try {
    nodes.value = await apiNode.list()
    // 记忆的节点已不存在则回落 local
    if (node.value !== 'local' && !nodes.value.some(n => n.id === node.value)) {
      node.value = 'local'
    }
  }
  catch {}
})

function onResized(e: SplitpanesResizedPayload) {
  // e.panes 仅含可见面板（v-if），按可见顺序回写：[tree?, center, monitor?]
  const visible: boolean[] = [treeVisible.value, true, monitorVisible.value]
  let pi = 0
  for (let i = 0; i < 3; i++) {
    if (!visible[i]) {
      continue
    }
    const size = Math.round(e.panes[pi]?.size * 10) / 10
    if (i === 0 && size) {
      treeWidth.value = size
    }
    if (i === 2 && size) {
      monitorWidth.value = size
    }
    pi++
  }
}

// 文件树双击/打开 → 编辑器工作台（节点一致）
function openInEditor(entry: FileEntry) {
  fileEditorStore.openWorkspace(entry.path, node.value)
}
</script>

<template>
  <div
    class="h-full w-full"
    :class="dropHover ? 'drop-hover' : ''"
    :data-drop-tip="$t('components.ydTerminal.dropToFillPath')"
    @dragover.prevent="dropHover = true"
    @dragleave.prevent="dropHover = false"
    @drop.prevent="onDropToTerminal"
  >
  <Splitpanes class="yd-term-workspace h-full w-full" @resized="onResized">
    <Pane
      v-if="treeVisible"
      :size="treeWidth"
      min-size="10"
      max-size="40"
      class="border-r bg-muted/20"
    >
      <YdFileTree :node="node" :cwd="termCwd" :title="$t('components.ydFileTree.files')" @open-file="openInEditor" />
    </Pane>

    <Pane min-size="30" class="min-w-0">
      <Panel
        ref="panelRef"
        v-model:node="node"
        v-model:initial-paste="initialPaste"
        :nodes="nodes"
        :tree-visible="treeVisible"
        :monitor-visible="monitorVisible"
        @toggle-tree="treeVisible = !treeVisible"
        @toggle-monitor="monitorVisible = !monitorVisible"
        @cwd="termCwd = $event"
      />
    </Pane>

    <Pane
      v-if="monitorVisible"
      :size="monitorWidth"
      min-size="12"
      max-size="45"
      class="border-l bg-muted/20"
    >
      <YdMonitorSidebar :node="node" :active="monitorVisible" />
    </Pane>
  </Splitpanes>
  </div>
</template>

<style>
.drop-hover > .yd-term-workspace {
  outline: 2px dashed oklch(var(--primary) / 60%);
  outline-offset: -4px;
}
.drop-hover > .yd-term-workspace::after {
  content: attr(data-drop-tip);
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: oklch(var(--primary));
  background: oklch(var(--primary) / 6%);
  pointer-events: none;
}
.yd-term-workspace { position: relative; }
.yd-term-workspace .splitpanes__splitter {
  position: relative;
  width: 5px;
  margin: 0 -2px;
  z-index: 5;
  cursor: col-resize;
}
.yd-term-workspace .splitpanes__splitter::before {
  content: '';
  position: absolute;
  inset: 0 2px;
  background: transparent;
  transition: background 0.15s;
}
.yd-term-workspace .splitpanes__splitter:hover::before,
.yd-term-workspace .splitpanes--dragging .splitpanes__splitter::before {
  background: oklch(var(--primary));
}
</style>
