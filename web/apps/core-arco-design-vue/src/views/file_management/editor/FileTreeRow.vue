<script setup lang="ts">
import type { FileTreeApi, FileTreeNode } from './types'

// 文件树行（递归）：懒加载展开、右键菜单。
const props = defineProps<{
  node: FileTreeNode
  depth: number
  filter: string
}>()

const tree = inject<FileTreeApi>('fileTreeApi')!

const matched = computed(() => {
  const kw = props.filter.trim().toLowerCase()
  if (!kw) {
    return true
  }
  if (props.node.entry.name.toLowerCase().includes(kw)) {
    return true
  }
  return !!props.node.children?.some(c => c.entry.name.toLowerCase().includes(kw))
})
</script>

<template>
  <template v-if="matched">
    <FaContextMenu :items="tree.menuItems(node)">
      <button
        type="button"
        class="flex w-full cursor-pointer items-center gap-1 rounded px-1 py-[3px] text-left text-[13px] transition-colors hover:bg-accent/50"
        :style="{ paddingLeft: `${depth * 14 + 6}px` }"
        :title="node.entry.target ? `${node.entry.path} → ${node.entry.target}` : node.entry.path"
        @click="node.entry.isDir ? tree.toggle(node) : tree.openFile(node)"
      >
        <FaIcon
          v-if="node.entry.isDir"
          :name="node.expanded ? 'i-lucide:chevron-down' : 'i-lucide:chevron-right'"
          class="w-3.5 shrink-0 text-xs text-muted-foreground"
        />
        <span v-else class="w-3.5 shrink-0" />
        <YdMorphIcon
          :name="node.entry.isDir ? (node.expanded ? 'folder-open' : 'folder') : 'file'"
          :size="14"
          class="shrink-0"
          :class="node.entry.isDir ? 'text-amber-500' : 'text-muted-foreground'"
        />
        <span class="truncate">{{ node.entry.name }}</span>
        <span v-if="node.loading" class="text-[10px] text-muted-foreground">…</span>
      </button>
    </FaContextMenu>

    <div v-if="node.entry.isDir && node.expanded && node.children">
      <FileTreeRow
        v-for="child in node.children"
        :key="child.entry.path"
        :node="child"
        :depth="depth + 1"
        :filter="filter"
      />
      <div
        v-if="!node.children.length"
        class="py-0.5 text-xs text-muted-foreground"
        :style="{ paddingLeft: `${(depth + 1) * 14 + 24}px` }"
      >
        （空目录）
      </div>
    </div>
  </template>
</template>
