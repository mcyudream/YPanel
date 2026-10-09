<script setup lang="ts">
// YdDockerNodeSelect 容器域节点选择器（M55）：写入 dockerNode 模块级上下文并整页刷新
// （容器域六个独立路由页 + 详情页的 API 全部经 withNode 路由，reload 后按新节点加载）。
import api from '@/api'
import { getDockerNode, setDockerNode } from '@/api/dockerNode'
import YdSelect from '@/components/YdSelect/index.vue'
import { i18n } from '@/locales'

defineOptions({ name: 'YdDockerNodeSelect' })

const emit = defineEmits<{ (e: 'change'): void }>()

interface NodeItem { id: string, name: string, online: boolean }

const current = ref(getDockerNode() || 'local')
const options = ref<{ label: string, value: string }[]>([])

onMounted(async () => {
  try {
    const res = await api.get('api/v1/nodes', { silent: true })
    const list = (res.data as NodeItem[]).filter(n => n.id === 'local' || n.online)
    options.value = list.map(n => ({
      value: n.id,
      label: n.id === 'local' ? `${n.name} · ${i18n.global.t('nodes.local')}` : n.name,
    }))
    if (!options.value.some(o => o.value === current.value)) {
      current.value = 'local'
      setDockerNode('')
    }
  }
  catch {}
})

function onChange(v: string | number) {
  const id = String(v)
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
  <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
    <FaIcon name="i-lucide:server" class="text-sm opacity-70" />
    <YdSelect v-model="current" :options="options" size="sm" @update:model-value="onChange" />
  </div>
</template>
