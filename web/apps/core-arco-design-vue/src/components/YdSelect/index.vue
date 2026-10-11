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
  /** 裸形态（M60.1）：ghost 触发器无边框，用于发送框分段工具条内 */
  bare?: boolean
  /** 自适应图标形态：≥md 显示文字标签，<md 收缩为纯图标触发器（title 保留语义） */
  icon?: string
}>(), {
  modelValue: undefined,
  placeholder: undefined,
  disabled: false,
  size: 'sm',
  buttonClass: '',
  bare: false,
  icon: undefined,
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
      :variant="bare ? 'ghost' : 'outline'"
      :size="size"
      :disabled="disabled"
      :class="[sizeCls, buttonClass, icon ? 'yd-sel-trigger' : '']"
      class="max-w-full justify-between gap-2 font-normal"
      :title="icon ? (current?.label ?? placeholder ?? '') : undefined"
    >
      <FaIcon v-if="icon" :name="icon" class="yd-sel-icon shrink-0 text-base" />
      <span class="truncate" :class="[current ? '' : 'text-muted-foreground', icon ? 'yd-sel-label' : '']">{{ current?.label ?? (placeholder ?? $t('components.ydSelect.placeholder')) }}</span>
      <FaIcon name="i-lucide:chevron-down" class="yd-sel-chevron shrink-0 text-xs text-muted-foreground" />
    </FaButton>
  </FaDropdown>
</template>

<style scoped>
/* 自适应图标形态（icon prop）：视口 <md 或宿主 .yd-sender-compact（容器实测放不下，容器宽 ≠ 视口宽，
   如桌面悬浮窗 520px）时触发器收缩为纯图标——两路信号任一生效即图标态 */
.yd-sel-icon {
  display: none;
}

@media (max-width: 767.9px) {
  .yd-sel-trigger {
    width: 2rem;
    justify-content: center;
    padding-inline: 0;
  }

  .yd-sel-icon {
    display: block;
  }

  .yd-sel-label,
  .yd-sel-chevron {
    display: none;
  }
}

.yd-sender-compact .yd-sel-trigger {
  width: 2rem;
  justify-content: center;
  padding-inline: 0;
}

.yd-sender-compact .yd-sel-icon {
  display: block;
}

.yd-sender-compact .yd-sel-label,
.yd-sender-compact .yd-sel-chevron {
  display: none;
}
</style>
