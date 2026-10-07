<script setup lang="ts">
// 顶栏通知中心：铃铛 + 未读角标 + SSE 实时推送 + Popover 最近通知。
import type { NotificationItem } from '@/api/modules/ops'
import { notificationApi } from '@/api/modules/ops'
import { useNotificationCenterStore } from '@/store/modules/notificationCenter'

defineOptions({
  name: 'ToolbarNotificationCenter',
})

const appSettingsStore = useAppSettingsStore()
const appAccountStore = useAppAccountStore()

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
        historyUnread.value++
        if (modalVisible.value) {
          history.value = [{ id: data.id!, level: data.level!, title: data.title!, content: data.content!, read: false, createdAt: data.createdAt! }, ...history.value]
        }
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
  modalVisible.value = true
  void loadHistory()
}

// ---------- 历史弹窗 ----------
const notificationCenter = useNotificationCenterStore()
const modalVisible = computed({
  get: () => notificationCenter.visible,
  set: v => (notificationCenter.visible = v),
})
const history = ref<NotificationItem[]>([])
const historyUnread = ref(0)
const historyLoading = ref(false)
const historyKeyword = ref('')

const historyFiltered = computed(() => {
  const kw = historyKeyword.value.trim().toLowerCase()
  if (!kw) {
    return history.value
  }
  return history.value.filter(n =>
    (n.title || '').toLowerCase().includes(kw) || (n.content || '').toLowerCase().includes(kw))
})

async function loadHistory() {
  historyLoading.value = true
  try {
    const [list, count] = await Promise.all([notificationApi.list(100), notificationApi.unread()])
    history.value = list
    historyUnread.value = count
  }
  catch (e: any) {
    useFaToast().error('加载通知失败', { description: e?.message })
  }
  finally {
    historyLoading.value = false
  }
}

async function markAllHistory() {
  await notificationApi.markRead()
  await loadHistory()
  void loadInitial()
}

async function markOne(n: NotificationItem) {
  if (n.read) {
    return
  }
  await notificationApi.markRead(n.id)
  n.read = true
  historyUnread.value = Math.max(0, historyUnread.value - 1)
  const local = recent.value.find(x => x.id === n.id)
  if (local) {
    local.read = true
  }
  void loadInitial()
}

const historyLevelStyle: Record<string, string> = {
  info: 'bg-blue-500/10 text-blue-600',
  success: 'bg-emerald-500/10 text-emerald-600',
  warning: 'bg-amber-500/10 text-amber-600',
  error: 'bg-red-500/10 text-red-600',
}

// 打开弹窗 / 外部 refreshTick 触发刷新
watch(modalVisible, (v) => {
  if (v) {
    void loadHistory()
  }
})
watch(() => notificationCenter.refreshTick, () => {
  if (modalVisible.value) {
    void loadHistory()
  }
})

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

    <!-- 通知历史弹窗（全局，概览页「查看全部」同样唤起） -->
    <FaModal v-model="modalVisible" title="通知中心" class="max-w-2xl!" :close-on-click-modal="false">
      <div class="flex flex-col gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <span v-if="historyUnread" class="rounded-full bg-red-500/10 px-2 py-0.5 text-xs text-red-600">{{ historyUnread }} 未读</span>
          <FaInput v-model="historyKeyword" placeholder="搜索通知…" class="w-44!" />
          <FaButton class="ml-auto" variant="outline" size="sm" :disabled="!historyUnread" @click="markAllHistory">
            全部已读
          </FaButton>
        </div>
        <div class="max-h-[55vh] overflow-y-auto rounded-lg border">
          <div v-if="historyLoading && !history.length" class="p-8 text-center text-sm text-muted-foreground">
            加载中…
          </div>
          <div v-else-if="!historyFiltered.length" class="p-8 text-center text-sm text-muted-foreground">
            暂无通知
          </div>
          <div
            v-for="n in historyFiltered"
            :key="n.id"
            class="flex cursor-pointer items-start gap-3 border-t px-4 py-3 transition-colors first:border-t-0 hover:bg-accent/30"
            :class="n.read ? 'opacity-60' : ''"
            @click="markOne(n)"
          >
            <span class="mt-0.5 shrink-0 rounded-full px-2 py-0.5 text-xs" :class="historyLevelStyle[n.level] || historyLevelStyle.info">
              {{ n.level }}
            </span>
            <div class="min-w-0 flex-1">
              <div class="text-sm" :class="n.read ? '' : 'font-medium'">
                <span v-if="!n.read" class="mr-1.5 inline-block size-1.5 rounded-full bg-red-500 align-middle" />
                {{ n.title }}
              </div>
              <div class="mt-0.5 break-all text-xs text-muted-foreground">
                {{ n.content }}
              </div>
            </div>
            <span class="shrink-0 text-xs tabular-nums text-muted-foreground">{{ new Date(n.createdAt).toLocaleString('zh-CN', { hour12: false }) }}</span>
          </div>
        </div>
        <div class="text-xs text-muted-foreground">
          点击未读通知即标记已读
        </div>
      </div>
    </FaModal>
  </div>
</template>
