<script setup lang="ts">
// 文件编辑工作台弹窗（M20 页面形态）：FaModal 近全屏壳，内容面板复用 WorkspaceContent。
// 页面内各自挂载（文件管理/容器详情等）；与 M57 全局浮层 YdQuickDock 相互独立——
// 两边各持一份 store 实例（见 store/modules/fileEditor.ts 的 provide/inject 作用域），状态与会话互不影响。
import WorkspaceContent from './WorkspaceContent.vue'

const store = useFileEditorStore()
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
  >
    <WorkspaceContent />
  </FaModal>
</template>
