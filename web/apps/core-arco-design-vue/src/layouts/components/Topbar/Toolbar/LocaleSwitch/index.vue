<script setup lang="ts">
// 语言切换（B26）：zh-CN / en-US，持久化到后端 panel.language。
// current 直接跟随 i18n 全局 locale（含启动时后端回读后的值），避免按钮态与实际语言脱节。
import { i18n, setLocale } from '@/locales'

defineOptions({
  name: 'ToolbarLocale',
})

const current = computed<'zh-CN' | 'en-US'>({
  get: () => i18n.global.locale.value as 'zh-CN' | 'en-US',
  set: v => void setLocale(v),
})

function toggle() {
  current.value = current.value === 'zh-CN' ? 'en-US' : 'zh-CN'
}
</script>

<template>
  <FaTooltip side="bottom" :content="current === 'zh-CN' ? 'English' : '中文'">
    <button
      type="button"
      class="flex size-8 cursor-pointer items-center justify-center rounded-md text-sm font-medium transition-colors hover:bg-accent"
      @click="toggle"
    >
      {{ current === 'zh-CN' ? 'EN' : '中' }}
    </button>
  </FaTooltip>
</template>
