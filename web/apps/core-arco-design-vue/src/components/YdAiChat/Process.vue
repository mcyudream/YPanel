<script setup lang="ts">
// YdAiProcess：工具/步骤执行时间线（对齐 YdChatProcess 视觉：✓ 图标 + 名称 + 明细）。
const props = withDefaults(defineProps<{
  steps: Array<{ type: string, name?: string, detail?: string }>
  streaming?: boolean
}>(), {
  streaming: false,
})

const typeLabel: Record<string, string> = {
  scene: '读取页面数据',
  action: '调用工具',
  tool: '执行工具',
  tool_result: '工具结果',
  thought: '思考',
}

function label(s: { type: string, name?: string }) {
  return typeLabel[s.type] || s.type
}

function icon(s: { type: string }) {
  switch (s.type) {
    case 'scene': return 'i-lucide:eye'
    case 'tool_result': return 'i-lucide:check'
    case 'thought': return 'i-lucide:brain'
    default: return 'i-lucide:wrench'
  }
}

const lastIdx = computed(() => props.steps.length - 1)
</script>

<template>
  <div class="mb-2 space-y-1 rounded-md bg-muted/40 px-2.5 py-1.5 text-xs text-muted-foreground">
    <div v-for="(s, i) in steps" :key="i" class="flex items-start gap-1.5">
      <FaIcon :name="icon(s)" class="mt-0.5 text-[10px]" :class="i === lastIdx && streaming ? 'animate-pulse' : ''" />
      <span class="min-w-0 flex-1 truncate">
        <template v-if="s.name && s.type === 'tool_result'">{{ s.name }}：{{ s.detail }}</template>
        <template v-else-if="s.name">{{ label(s) }}：{{ s.name }}</template>
        <template v-else>{{ label(s) }}{{ s.detail ? '：' + s.detail : '' }}</template>
      </span>
      <FaIcon
        v-if="i === lastIdx && streaming"
        name="i-lucide:loader-circle"
        class="mt-0.5 animate-spin text-[10px]"
      />
    </div>
  </div>
</template>
