<script setup lang="ts">
import type { ToolbarSettings } from '@fantastic-admin/settings'
import { pascalCase } from 'scule'

defineOptions({
  name: 'ToolbarTools',
})

const props = defineProps<{
  mode: 'left-side' | 'right-side'
}>()

const modules = import.meta.glob('./*/index.vue', { import: 'default', eager: true })

const appSettingsStore = useAppSettingsStore()

const toolbarTools = {
  'left-side': ['breadcrumb'],
  // taskCenter / desktopEntry 非设置开关项，固定显示（红点由组件内部控制）
  'right-side': ['menuSearch', 'taskCenter', 'desktopEntry', 'localeSwitch', 'notificationCenter', 'fullscreen', 'pageReload', 'colorScheme'],
} satisfies Record<typeof props.mode, string[]>

const tools = computed(() => toolbarTools[props.mode])

function checkVisible(item: boolean | { enable: boolean }) {
  if (typeof item === 'boolean') {
    return item
  }
  else if (typeof item === 'object' && item !== null) {
    return item.enable ?? false
  }
  return false
}
</script>

<template>
  <template v-for="item in tools" :key="item">
    <Component
      :is="modules[`./${pascalCase(item)}/index.vue`]"
      v-if="item === 'taskCenter' || item === 'desktopEntry' || item === 'localeSwitch' || checkVisible(appSettingsStore.settings.toolbar[item as keyof ToolbarSettings])"
    />
  </template>
</template>
