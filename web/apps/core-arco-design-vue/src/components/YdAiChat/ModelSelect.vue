<script setup lang="ts">
// YdAiModelSelect：供应商+模型合并单选（M60，对话页/浮层共用）。
// 各供应商模型平铺为「模型名」（重名自动追加「 · 供应商」），选中一次回写 providerId+model。
import { computed } from 'vue'
import type { AIProvider } from '@/api/modules/ai'

const props = defineProps<{
  providers: AIProvider[]
  providerId: number
  model: string
  /** 裸形态：分段工具条内 ghost 触发器 */
  bare?: boolean
}>()

const emit = defineEmits<{
  'update:providerId': [number]
  'update:model': [string]
}>()

function modelsOf(p: AIProvider) {
  const list = (p.models || '').split(',').map(m => m.trim()).filter(Boolean)
  if (p.model && !list.includes(p.model)) {
    list.unshift(p.model)
  }
  return list
}

function providerName(pid: number) {
  return props.providers.find(p => p.id === pid)?.name || `#${pid}`
}

// 平铺所有 (供应商 × 模型) 组合；同名模型出现在多个供应商时以「 · 供应商」消歧
const options = computed(() => {
  const count = new Map<string, number>()
  const pairs: Array<{ pid: number, m: string }> = []
  for (const p of props.providers) {
    for (const m of modelsOf(p)) {
      count.set(m, (count.get(m) || 0) + 1)
      pairs.push({ pid: p.id, m })
    }
  }
  return pairs.map(({ pid, m }) => ({
    value: `${pid}:${m}`,
    label: count.get(m)! > 1 ? `${m} · ${providerName(pid)}` : m,
  }))
})

const current = computed(() =>
  props.providerId > 0 && props.model ? `${props.providerId}:${props.model}` : '',
)

function onSelect(v: string | number) {
  const s = String(v)
  const i = s.indexOf(':')
  if (i <= 0) {
    return
  }
  emit('update:providerId', Number(s.slice(0, i)))
  emit('update:model', s.slice(i + 1))
}
</script>

<template>
  <YdSelect
    :model-value="current"
    :options="options"
    size="sm"
    :bare="bare"
    :placeholder="$t('components.ydAiChat.selectModel')"
    button-class="max-w-45 font-mono text-[11px]"
    icon="i-lucide:cpu"
    @update:model-value="onSelect"
  />
</template>
