<script setup lang="ts">
import type { NotificationItem } from '@/api/modules/m9'
import { notificationApi, panelBackupApi } from '@/api/modules/m9'

defineOptions({
  name: 'NotificationsIndex',
})

const notifications = ref<NotificationItem[]>([])
const unread = ref(0)
const loading = ref(false)
const backups = ref<{ name: string, sizeMb: number, modTime: string, path: string }[]>([])
const backupBusy = ref(false)
const restoreHint = ref('')

async function load() {
  loading.value = true
  try {
    notifications.value = await notificationApi.list(100)
    unread.value = await notificationApi.unread()
    backups.value = await panelBackupApi.list()
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

async function doBackup() {
  backupBusy.value = true
  try {
    const out = await panelBackupApi.create()
    useFaToast().success(`面板备份完成：${out.file || ''}`)
    await load()
  }
  catch (e: any) {
    useFaToast().error('备份失败', { description: e?.message })
  }
  finally {
    backupBusy.value = false
  }
}

async function downloadBackup(b: { path: string }) {
  const token = localStorage.getItem('token') || ''
  window.open(panelBackupApi.downloadURL(b.path, token))
}

async function removeBackup(b: { name: string }) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除备份',
    content: `确认删除 ${b.name}？`,
    onConfirm: async () => {
      await panelBackupApi.remove(b.name)
      useFaToast().success('已删除')
      await load()
    },
  })
}

async function showHint() {
  restoreHint.value = await panelBackupApi.restoreHint()
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
        <span>站内通知（告警/关键事件）与面板备份</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" :disabled="!unread" @click="markAll">全部已读</FaButton>
        <FaButton size="sm" :loading="backupBusy" @click="doBackup">
          <YdMorphIcon name="save" :size="14" class="mr-1" /> 立即备份面板
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 通知列表 -->
      <div class="rounded-lg border">
        <div class="border-b bg-muted/40 px-4 py-2 text-sm font-medium">站内通知</div>
        <div v-if="!notifications.length" class="p-8 text-center text-sm text-muted-foreground">暂无通知</div>
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
            <div class="mt-0.5 text-xs text-muted-foreground">{{ n.content }}</div>
          </div>
          <span class="shrink-0 text-xs tabular-nums text-muted-foreground">{{ new Date(n.createdAt).toLocaleString('zh-CN', { hour12: false }) }}</span>
        </div>
      </div>

      <!-- 面板备份 -->
      <div class="mt-4 rounded-lg border">
        <div class="flex items-center justify-between border-b bg-muted/40 px-4 py-2">
          <span class="text-sm font-medium">面板备份（SQLite 快照）</span>
          <FaButton variant="ghost" size="sm" class="text-xs!" @click="showHint">恢复说明</FaButton>
        </div>
        <div v-if="restoreHint" class="whitespace-pre-line bg-muted/30 px-4 py-2 text-xs text-muted-foreground">{{ restoreHint }}</div>
        <table class="w-full text-sm">
          <tbody>
            <tr v-if="!backups.length">
              <td class="px-4 py-6 text-center text-muted-foreground">暂无备份</td>
            </tr>
            <tr v-for="b in backups" :key="b.name" class="border-t hover:bg-accent/30">
              <td class="px-4 py-2 font-mono text-[13px]">{{ b.name }}</td>
              <td class="px-4 py-2 text-xs tabular-nums text-muted-foreground">{{ b.sizeMb.toFixed(2) }} MB</td>
              <td class="px-4 py-2 text-xs tabular-nums text-muted-foreground">{{ new Date(b.modTime).toLocaleString('zh-CN', { hour12: false }) }}</td>
              <td class="px-4 py-2 text-right">
                <FaButton variant="ghost" size="sm" @click="downloadBackup(b)">下载</FaButton>
                <FaButton variant="outline" size="sm" @click="removeBackup(b)">删除</FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>
  </div>
</template>
