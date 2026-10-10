<script setup lang="ts">
import { useEventBus } from '@/composables/useEventBus'
import dayjs from '@/utils/dayjs'
import { ua } from '@/utils/ua'
import { useI18n } from 'vue-i18n'
import { i18n, initLocale } from '@/locales'
import apiSettings from '@/api/modules/settings'
import Provider from './ui/provider/index.vue'
import AiFloatLayer from '@/components/AiFloatLayer.vue'
import 'dayjs/locale/zh-cn'

const route = useRoute()

const appSettingsStore = useAppSettingsStore()
const appAccountStore = useAppAccountStore()

const { auth } = useAppAuth()
const { generateTitle } = useAppMenu()

document.body.setAttribute('data-os', ua.getOS().name || '')

// B7：全局事件总线订阅（登录后收到任务/通知事件时 toast 提示）
const { onEvent } = useEventBus()
const { t } = useI18n()
watch(() => useAppAccountStore().isLogin, (logged) => {
  if (!logged) {
    return
  }
  // B26-full：登录后回读后端语言偏好（initLocale 内部 localStorage 优先）
  apiSettings.get().then((s) => {
    if (s && (s['panel.language'] === 'en' || s['panel.language'] === 'zh')) {
      initLocale(s['panel.language'])
    }
  }).catch(() => {})
  onEvent((e: { topic: string, type: string, title?: string, payload?: any }) => {
    if (e.topic === 'task') {
      e.type === 'failed'
        ? useFaToast().error(e.title || t('layout.app.taskFailed'))
        : useFaToast().success(e.title || t('layout.app.taskDone'))
    }
    else if (e.topic === 'notification' && e.type !== 'info') {
      useFaToast().warning(e.title || t('layout.app.newNotification'), { description: e.payload?.content })
    }
  })
}, { immediate: true })

const isAuth = computed(() => {
  return route.matched.every((item) => {
    return auth(item.meta.auth ?? '')
  })
})

// M57 快速工作台浮层：仅经典面板（桌面有窗口体系）+ 已登录 + 有文件或终端权限时显示
const dockAllowed = computed(() => {
  return useAppAccountStore().isLogin && auth(['file:read', 'terminal:access'])
})
const isDesktopRoute = computed(() => route.path.startsWith('/desktop'))

// 设置网页 title
watch([
  () => appSettingsStore.settings.app.dynamicTitle,
  () => appSettingsStore.title,
  () => i18n.global.locale.value,
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
    <!-- B18：全局 AI 浮层（登录后任意页面可用；登录页不渲染——isAuth 只判路由权限不判登录态） -->
    <AiFloatLayer v-if="isAuth && appAccountStore.isLogin" />
    <!-- M57：全局快速工作台浮层（文件树/编辑器/终端，最小化跨路由留存） -->
    <YdQuickDock v-if="isAuth && appAccountStore.isLogin && dockAllowed && !isDesktopRoute" />
    <FaToast :theme="appSettingsStore.currentColorScheme" />
    <AppSystemInfo />
  </Provider>
</template>
