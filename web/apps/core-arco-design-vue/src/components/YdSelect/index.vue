<script setup lang="ts">
import { computed } from 'vue'
import { FaButton, FaDropdown, FaIcon } from '@fantastic-admin/components'

/**
 * YdSelect — 下拉选择（yd- 封装）：FaDropdown 弹层，深浅主题跟随，
 * 替代原生 <select>（原生展开层是浏览器 UI，不可主题化且位置脱管）。
 * drop-in：options 支持 string/number 或 {label, value, disabled}；v-model 透传原始值类型。
 */
interface Option {
  label: string
  value: string | number
  disabled?: boolean
}

const props = withDefaults(defineProps<{
  options: (Option | string | number)[]
  modelValue?: string | number
  placeholder?: string
  disabled?: boolean
  size?: 'sm' | 'default' | 'icon-sm'
  /** 触发按钮附加类（宽度等） */
  buttonClass?: string
}>(), {
  modelValue: undefined,
  placeholder: undefined,
  disabled: false,
  size: 'sm',
  buttonClass: '',
})

const emit = defineEmits<{
  'update:modelValue': [string | number]
}>()

const normalized = computed<Option[]>(() =>
  props.options.map(o => typeof o === 'object' ? o : { label: String(o), value: o }),
)

const current = computed(() => normalized.value.find(o => o.value === props.modelValue))

const menuItems = computed(() => [normalized.value.map(o => ({
  label: o.label,
  disabled: o.disabled,
  handle: () => emit('update:modelValue', o.value),
}))])

const sizeCls = computed(() => ({ default: 'h-9 text-sm', sm: 'h-8 text-sm', 'icon-sm': 'h-7 text-xs' }[props.size] ?? 'h-8'))
</script>

<template>
  <FaDropdown :items="menuItems">
    <FaButton
      variant="outline"
      :size="size"
      :disabled="disabled"
      :class="[sizeCls, buttonClass]"
      class="max-w-full justify-between gap-2 font-normal"
    >
      <span class="truncate" :class="current ? '' : 'text-muted-foreground'">{{ current?.label ?? (placeholder ?? $t('components.ydSelect.placeholder')) }}</span>
      <FaIcon name="i-lucide:chevron-down" class="shrink-0 text-xs text-muted-foreground" />
    </FaButton>
  </FaDropdown>
</template>
