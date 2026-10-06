<script setup lang="ts">
import type { SplitpanesResizedPayload } from 'splitpanes'
import type { NodeItem } from '@/api/modules/node'
import type { FileEntry } from '@/api/modules/file'
import apiNode from '@/api/modules/node'
import { useFileEditorStore } from '@/store/modules/fileEditor'
import { Pane, Splitpanes } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import { useLocalStorage } from '@vueuse/core'
import YdFileTree from '../YdFileTree/index.vue'
import YdMonitorSidebar from '../YdMonitorSidebar/index.vue'
import Panel from './Panel.vue'

// 终端工作台：VS Code 式三栏（左文件树 | 中终端面板 | 右监控），splitpanes 可拖宽、可折叠，布局持久化。
defineOptions({
  name: 'YdTerminalWorkspace',
})

const fileEditorStore = useFileEditorStore()

const nodes = ref<NodeItem[]>([])
const node = useLocalStorage('ypanel.terminal.node', 'local')
const treeVisible = useLocalStorage('ypanel.terminal.tree', true)
const monitorVisible = useLocalStorage('ypanel.terminal.monitor', true)
const treeWidth = useLocalStorage('ypanel.terminal.treeWidth', 18)
const monitorWidth = useLocalStorage('ypanel.terminal.monitorWidth', 22)

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
  <Splitpanes class="yd-term-workspace h-full w-full" @resized="onResized">
    <Pane
      v-if="treeVisible"
      :size="treeWidth"
      min-size="10"
      max-size="40"
      class="border-r bg-muted/20"
    >
      <YdFileTree :node="node" title="文件" @open-file="openInEditor" />
    </Pane>

    <Pane min-size="30" class="min-w-0">
      <Panel
        v-model:node="node"
        :nodes="nodes"
        :tree-visible="treeVisible"
        :monitor-visible="monitorVisible"
        @toggle-tree="treeVisible = !treeVisible"
        @toggle-monitor="monitorVisible = !monitorVisible"
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
</template>

<style>
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
