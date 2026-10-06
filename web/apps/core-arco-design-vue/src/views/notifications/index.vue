<script setup lang="ts">
import type { NotificationItem } from '@/api/modules/m9'
import { notificationApi } from '@/api/modules/m9'

defineOptions({
  name: 'NotificationsIndex',
})

const toast = useFaToast()

const notifications = ref<NotificationItem[]>([])
const unread = ref(0)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    notifications.value = await notificationApi.list(100)
    unread.value = await notificationApi.unread()
  }
  catch (e: any) {
    toast.error('加载通知失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function markAll() {
  await notificationApi.markRead()
  await load()
}

async function markOne(n: NotificationItem) {
  await notificationApi.markRead(n.id)
  await load()
}

const levelStyle: Record<string, string> = {
  info: 'bg-blue-500/10 text-blue-600',
  success: 'bg-emerald-500/10 text-emerald-600',
  warning: 'bg-amber-500/10 text-amber-600',
  error: 'bg-red-500/10 text-red-600',
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="bell" :size="24" />
          <span>通知中心</span>
          <span v-if="unread" class="rounded-full bg-red-500/10 px-2 py-0.5 text-xs text-red-600">{{ unread }} 未读</span>
        </div>
      </template>
      <template #description>
        <span>站内通知（告警/关键事件）</span>
      </template>
      <FaButton variant="outline" size="sm" :disabled="!unread" @click="markAll">
        全部已读
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div class="rounded-lg border">
        <div class="border-b bg-muted/40 px-4 py-2 text-sm font-medium">
          站内通知
        </div>
        <div v-if="loading && !notifications.length" class="p-8 text-center text-sm text-muted-foreground">
          加载中…
        </div>
        <div v-else-if="!notifications.length" class="p-8 text-center text-sm text-muted-foreground">
          暂无通知
        </div>
        <div
          v-for="n in notifications"
          :key="n.id"
          class="flex cursor-pointer items-start gap-3 border-t px-4 py-3 transition-colors hover:bg-accent/30"
          :class="n.read ? 'opacity-60' : ''"
          @click="!n.read && markOne(n)"
        >
          <span class="mt-0.5 rounded-full px-2 py-0.5 text-xs" :class="levelStyle[n.level] || levelStyle.info">
            {{ n.level }}
          </span>
          <div class="min-w-0 flex-1">
            <div class="text-sm" :class="n.read ? '' : 'font-medium'">
              <span v-if="!n.read" class="mr-1.5 inline-block size-1.5 rounded-full bg-red-500 align-middle" />
              {{ n.title }}
            </div>
            <div class="mt-0.5 text-xs text-muted-foreground">
              {{ n.content }}
            </div>
          </div>
          <span class="shrink-0 text-xs tabular-nums text-muted-foreground">{{ new Date(n.createdAt).toLocaleString('zh-CN', { hour12: false }) }}</span>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
