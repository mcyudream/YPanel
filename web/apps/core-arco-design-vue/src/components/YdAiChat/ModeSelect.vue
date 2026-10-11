<script setup lang="ts">
// YdAiModeSelect：AI 权限档位选择器（M60 四档，对话页/浮层共用）。
// value 与后端 aichat.go 权限档位枚举一一对应；默认 confirm（变更前确认）。
type AiChatMode = 'read_only' | 'confirm' | 'auto_edit' | 'full'

withDefaults(defineProps<{
  modelValue?: AiChatMode
  disabled?: boolean
  /** 裸形态：分段工具条内 ghost 触发器 */
  bare?: boolean
}>(), {
  modelValue: 'confirm',
  disabled: false,
  bare: false,
})

const emit = defineEmits<{
  'update:modelValue': [AiChatMode]
}>()

// labelKey 展示层词条化；value 为发后端的枚举值
const LEVELS: Array<{ value: AiChatMode, labelKey: string }> = [
  { value: 'read_only', labelKey: 'components.ydAiChat.levelReadOnly' },
  { value: 'confirm', labelKey: 'components.ydAiChat.levelConfirm' },
  { value: 'auto_edit', labelKey: 'components.ydAiChat.levelAutoEdit' },
  { value: 'full', labelKey: 'components.ydAiChat.levelFull' },
]
</script>

<template>
  <YdSelect
    :model-value="modelValue ?? 'confirm'"
    :options="LEVELS.map(l => ({ value: l.value, label: $t(l.labelKey) }))"
    :disabled="disabled"
    :bare="bare"
    size="sm"
    button-class="max-w-30"
    icon="i-lucide:shield"
    @update:model-value="v => emit('update:modelValue', v as AiChatMode)"
  />
</template>
