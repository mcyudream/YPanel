<script setup lang="ts">
// YdDockerNodeSelect 容器域节点选择器（M55）：写入 dockerNode 模块级上下文并整页刷新。
// M63 起展示与列表自拉收敛到统一节点选择器 YdNodeSelect，本组件只承载域上下文语义。
import { getDockerNode, setDockerNode } from '@/api/dockerNode'
import YdNodeSelect from '@/components/YdNodeSelect/index.vue'

defineOptions({ name: 'YdDockerNodeSelect' })

const current = ref(getDockerNode() || 'local')

function onLoaded(list: Array<{ id: string }>) {
  if (current.value !== 'local' && !list.some(n => n.id === current.value)) {
    current.value = 'local'
    setDockerNode('')
  }
}

function pick(id: string) {
  setDockerNode(id === 'local' ? '' : id)
  location.reload()
}
</script>

<template>
  <YdNodeSelect :model-value="current" @loaded="onLoaded" @update:model-value="pick" />
</template>
