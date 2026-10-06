<script setup lang="ts">
import { useEventBus } from '@/composables/useEventBus'
import dayjs from '@/utils/dayjs'
import { ua } from '@/utils/ua'
import Provider from './ui/provider/index.vue'
import AiFloatLayer from '@/components/AiFloatLayer.vue'
import 'dayjs/locale/zh-cn'

const route = useRoute()

const appSettingsStore = useAppSettingsStore()

const { auth } = useAppAuth()
const { generateTitle } = useAppMenu()

document.body.setAttribute('data-os', ua.getOS().name || '')

// B7：全局事件总线订阅（登录后收到任务/通知事件时 toast 提示）
const { onEvent } = useEventBus()
watch(() => useAppAccountStore().isLogin, (logged) => {
  if (!logged) {
    return
  }
  onEvent((e: { topic: string, type: string, title?: string, payload?: any }) => {
    if (e.topic === 'task') {
      e.type === 'failed'
        ? useFaToast().error(e.title || '任务失败')
        : useFaToast().success(e.title || '任务完成')
    }
    else if (e.topic === 'notification' && e.type !== 'info') {
      useFaToast().warning(e.title || '新通知', { description: e.payload?.content })
    }
  })
}, { immediate: true })

const isAuth = computed(() => {
  return route.matched.every((item) => {
    return auth(item.meta.auth ?? '')
  })
})

// 设置网页 title
watch([
  () => appSettingsStore.settings.app.dynamicTitle,
  () => appSettingsStore.title,
], () => {
  nextTick(() => {
    if (appSettingsStore.settings.app.dynamicTitle && appSettingsStore.title) {
      document.title = `${generateTitle(appSettingsStore.title)} - ${import.meta.env.VITE_APP_TITLE}`
    }
    else {
      document.title = import.meta.env.VITE_APP_TITLE
    }
  })
}, {
  immediate: true,
  deep: true,
})

onMounted(() => {
  appSettingsStore.setMode(document.documentElement.clientWidth)
  dayjs.locale('zh-cn')
  window.addEventListener('resize', () => {
    appSettingsStore.setMode(document.documentElement.clientWidth)
  })
})
</script>

<template>
  <Provider>
    <RouterView v-slot="{ Component }">
      <AppNotSupportedMobile v-if="!appSettingsStore.settings.app.mobile && appSettingsStore.mode === 'mobile'" />
      <Component :is="Component" v-else-if="isAuth" />
      <AppNotAllowed v-else />
    </RouterView>
    <AppBackToTop />
    <!-- B18：全局 AI 浮层（登录后任意页面可用） -->
    <AiFloatLayer v-if="isAuth" />
    <FaToast :theme="appSettingsStore.currentColorScheme" />
    <AppSystemInfo />
  </Provider>
</template>
