<script setup lang="ts">
import type { FileEditorGroup } from '@/store/modules/fileEditor'
import { useFileEditorStore } from '@/store/modules/fileEditor'

// 编辑器组标签栏：点击激活、脏标记、中键/×关闭、拖拽排序与跨组拖动。
const props = defineProps<{
  group: FileEditorGroup
  active: boolean
}>()

const store = useFileEditorStore()

function tabOf(tabId: string | null) {
  return tabId ? store.tabs[tabId] : undefined
}

function onClick(tabId: string) {
  store.activateTab(props.group.id, tabId)
}

function onClose(tabId: string) {
  if (store.closeTab(tabId, props.group.id) === 'confirm') {
    confirmClose(tabId)
  }
}

function onTabMouseDown(e: MouseEvent, tabId: string) {
  if (e.button === 1) {
    e.preventDefault()
    onClose(tabId)
  }
}

function confirmClose(tabId: string) {
  useFaModal().confirm({
    title: '未保存的修改',
    content: `“${tabId.split('::').pop()}” 有未保存的修改，关闭将丢失。`,
    onConfirm: () => {
      store.closeTab(tabId, props.group.id, true)
    },
  })
}

// ---- 拖拽（排序 / 跨组移动） ----
const dragOverTabId = ref<string | null>(null)

function onDragStart(e: DragEvent, tabId: string) {
  e.dataTransfer?.setData('application/x-ypanel-tab', JSON.stringify({ tabId, groupId: props.group.id }))
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
  }
}

function onDrop(e: DragEvent, targetTabId: string) {
  e.preventDefault()
  e.stopPropagation()
  dragOverTabId.value = null
  const raw = e.dataTransfer?.getData('application/x-ypanel-tab')
  if (!raw) {
    return
  }
  const { tabId, groupId } = JSON.parse(raw) as { tabId: string, groupId: number }
  if (groupId === props.group.id) {
    // 组内排序
    const from = props.group.tabIds.indexOf(tabId)
    const to = props.group.tabIds.indexOf(targetTabId)
    if (from !== -1 && to !== -1 && from !== to) {
      props.group.tabIds.splice(to, 0, ...props.group.tabIds.splice(from, 1))
    }
    store.activateTab(props.group.id, tabId)
  }
  else {
    store.moveTabToGroup(tabId, props.group.id, props.group.tabIds.indexOf(targetTabId))
  }
}

function onDragOver(e: DragEvent, tabId: string) {
  if (e.dataTransfer?.types.includes('application/x-ypanel-tab')) {
    e.preventDefault()
    dragOverTabId.value = tabId
  }
}
</script>

<template>
  <div class="flex items-stretch overflow-x-auto border-b bg-muted/40 text-[13px]" @dragover.prevent>
    <button
      v-for="tabId in group.tabIds"
      :key="tabId"
      type="button"
      draggable="true"
      class="group relative inline-flex max-w-52 shrink-0 cursor-pointer items-center gap-1.5 border-r px-3 py-1.5 transition-colors"
      :class="[
        group.activeTabId === tabId
          ? 'bg-background text-foreground'
          : 'text-muted-foreground hover:bg-accent/50',
        dragOverTabId === tabId ? 'border-l-2 border-l-primary' : '',
      ]"
      :title="tabOf(tabId)?.path"
      @click="onClick(tabId)"
      @mousedown="onTabMouseDown($event, tabId)"
      @dragstart="onDragStart($event, tabId)"
      @dragover="onDragOver($event, tabId)"
      @dragleave="dragOverTabId = null"
      @drop="onDrop($event, tabId)"
    >
      <YdMorphIcon name="file" :size="13" class="shrink-0 text-muted-foreground" />
      <span class="truncate">{{ tabOf(tabId)?.name }}</span>
      <span v-if="tabOf(tabId)?.dirty" class="text-primary" title="未保存">●</span>
      <span
        class="inline-flex size-4 shrink-0 cursor-pointer items-center justify-center rounded-full opacity-0 transition-opacity group-hover:opacity-60 hover:!opacity-100 hover:bg-accent"
        title="关闭 (中键)"
        @click.stop="onClose(tabId)"
      >
        <FaIcon name="i-lucide:x" class="text-[10px]" />
      </span>
    </button>
  </div>
</template>
