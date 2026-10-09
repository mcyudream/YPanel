<script setup lang="ts">
// 顶栏桌面工作台入口：一键进出 webos 桌面（与经典面板随时互切）。
import { useRouter } from 'vue-router'

defineOptions({
  name: 'ToolbarDesktopEntry',
})

const appSettingsStore = useAppSettingsStore()
const router = useRouter()
</script>

<template>
  <div v-if="appSettingsStore.mode === 'pc'" class="flex items-center">
    <FaButton variant="ghost" size="icon-sm" :title="$t('layout.desktopEntry.title')" @click="router.push('/desktop')">
      <FaIcon name="i-lucide:layout-dashboard" class="size-4" />
    </FaButton>
  </div>
</template>
