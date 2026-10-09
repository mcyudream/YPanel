<script setup lang="ts">
// YdDockerNodeSelect 容器域节点选择器（M55）：写入 dockerNode 模块级上下文并整页刷新。
// 形态对齐文件管理页节点下拉（FaDropdown + FaButton outline，工具条内联轻量样式）。
import api from '@/api'
import { getDockerNode, setDockerNode } from '@/api/dockerNode'
import { i18n } from '@/locales'

defineOptions({ name: 'YdDockerNodeSelect' })

const emit = defineEmits<{ (e: 'change'): void }>()

interface NodeItem { id: string, name: string, online: boolean }

const current = ref(getDockerNode() || 'local')
const nodes = ref<NodeItem[]>([])

const currentName = computed(() => {
  if (current.value === 'local') {
    return i18n.global.t('nodes.localPanel')
  }
  return nodes.value.find(n => n.id === current.value)?.name || current.value
})

const menuItems = computed(() => [nodes.value.map(n => ({
  label: n.id === 'local' ? i18n.global.t('nodes.localPanel') : n.name,
  disabled: n.id !== 'local' && !n.online,
  handle: () => pick(n.id),
}))])

onMounted(async () => {
  try {
    const res = await api.get('api/v1/nodes', { silent: true })
    nodes.value = res.data as NodeItem[]
    if (current.value !== 'local' && !nodes.value.some(n => n.id === current.value)) {
      current.value = 'local'
      setDockerNode('')
    }
  }
  catch {}
})

function pick(id: string) {
  if (id === current.value) {
    return
  }
  setDockerNode(id === 'local' ? '' : id)
  current.value = id
  emit('change')
  location.reload()
}
</script>

<template>
  <FaDropdown :items="menuItems">
    <FaButton variant="outline" size="sm" class="h-8">
      <FaIcon name="i-lucide:server" class="mr-1 text-xs text-muted-foreground" />
      {{ currentName }}
      <FaIcon name="i-lucide:chevron-down" class="ml-1 text-xs text-muted-foreground" />
    </FaButton>
  </FaDropdown>
</template>
