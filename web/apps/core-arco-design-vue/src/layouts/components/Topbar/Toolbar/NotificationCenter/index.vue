<script setup lang="ts">
// 顶栏通知中心：铃铛 + 未读角标 + SSE 实时推送 + Popover 最近通知。
import type { NotificationItem } from '@/api/modules/ops'
import { notificationApi } from '@/api/modules/ops'

defineOptions({
  name: 'ToolbarNotificationCenter',
})

const appSettingsStore = useAppSettingsStore()
const appAccountStore = useAppAccountStore()
const router = useRouter()

const unread = ref(0)
const recent = ref<NotificationItem[]>([])
const popVisible = ref(false)
let es: EventSource | null = null
let reconnectTimer: ReturnType<typeof setInterval> | null = null

const LEVEL_DOT: Record<string, string> = {
  error: 'bg-red-500',
  warning: 'bg-orange-400',
  success: 'bg-emerald-500',
  info: 'bg-blue-400',
}

async function loadInitial() {
  try {
    const [list, count] = await Promise.all([notificationApi.list(8), notificationApi.unread()])
    recent.value = list
    unread.value = count
  }
  catch {
    // 静默：初始加载失败等 SSE/下次重试
  }
}

// 打开面板时刷新（open 状态由 Popover 双向绑定）
watch(popVisible, (v) => {
  if (v) {
    void loadInitial()
  }
})

function connect() {
  if (es) {
    es.close()
  }
  const token = appAccountStore.token
  if (!token) {
    return
  }
  // 与 axios baseURL 对齐（dev 代理 / 生产同源）
  const base = (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY)
    ? '/proxy/'
    : (import.meta.env.VITE_APP_API_BASEURL || '/')
  es = new EventSource(`${base}api/v1/notifications/stream?token=${encodeURIComponent(token)}`)
  es.onmessage = (ev) => {
    try {
      const data = JSON.parse(ev.data) as { type: string, count?: number } & Partial<NotificationItem>
      if (data.type === 'unread') {
        unread.value = data.count || 0
      }
      else if (data.type === 'notification') {
        unread.value++
        recent.value = [{
          id: data.id!, level: data.level!, title: data.title!, content: data.content!,
          read: false, createdAt: data.createdAt!,
        }, ...recent.value].slice(0, 8)
      }
    }
    catch {}
  }
  // token 过期（401）时 EventSource 停止，定时重连
  es.onerror = () => {
    if (es && es.readyState === EventSource.CLOSED) {
      if (!reconnectTimer) {
        reconnectTimer = setInterval(() => {
          if (appAccountStore.isLogin) {
            connect()
          }
        }, 15000)
      }
    }
  }
  es.onopen = () => {
    if (reconnectTimer) {
      clearInterval(reconnectTimer)
      reconnectTimer = null
    }
  }
}

async function markAll() {
  try {
    await notificationApi.markRead()
    unread.value = 0
    recent.value = recent.value.map(n => ({ ...n, read: true }))
  }
  catch {}
}

function goAll() {
  popVisible.value = false
  router.push('/notifications')
}

function fmtTime(t: string) {
  const d = new Date(t)
  const today = new Date()
  const sameDay = d.toDateString() === today.toDateString()
  return sameDay
    ? d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false })
    : `${d.getMonth() + 1}/${d.getDate()} ${d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false })}`
}

onMounted(() => {
  void loadInitial()
  connect()
})

onBeforeUnmount(() => {
  es?.close()
  if (reconnectTimer) {
    clearInterval(reconnectTimer)
  }
})
</script>

<template>
  <div v-if="appSettingsStore.mode === 'pc'" class="flex items-center">
    <FaPopover v-model:open="popVisible" align="end" :side-offset="8">
      <FaButton variant="ghost" size="icon-sm" title="通知中心" class="relative!">
        <FaIcon name="i-lucide:bell" class="size-4" />
        <span
          v-if="unread > 0"
          class="absolute -right-0.5 -top-0.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] leading-none text-white ring-2 ring-background"
        >
          {{ unread > 99 ? '99+' : unread }}
        </span>
      </FaButton>
      <template #panel>
        <div class="w-80" @click.stop>
          <div class="flex items-center justify-between border-b px-3 py-2">
            <span class="text-sm font-medium">通知</span>
            <FaButton variant="ghost" size="sm" :disabled="!unread" @click="markAll">
              全部已读
            </FaButton>
          </div>
          <div class="max-h-80 overflow-y-auto">
            <div
              v-for="n in recent"
              :key="n.id"
              class="flex items-start gap-2 border-b px-3 py-2 last:border-b-0"
              :class="n.read ? 'opacity-60' : ''"
            >
              <span class="mt-1.5 inline-block size-1.5 shrink-0 rounded-full" :class="LEVEL_DOT[n.level] || 'bg-blue-400'" />
              <div class="min-w-0 flex-1">
                <div class="truncate text-sm">{{ n.title }}</div>
                <div class="truncate text-xs text-muted-foreground" :title="n.content">{{ n.content }}</div>
              </div>
              <span class="shrink-0 text-xs text-muted-foreground">{{ fmtTime(n.createdAt) }}</span>
            </div>
            <div v-if="!recent.length" class="px-3 py-8 text-center text-sm text-muted-foreground">
              暂无通知
            </div>
          </div>
          <div class="border-t px-3 py-1.5 text-center">
            <FaButton variant="ghost" size="sm" class="w-full" @click="goAll">
              查看全部
            </FaButton>
          </div>
        </div>
      </template>
    </FaPopover>
  </div>
</template>
