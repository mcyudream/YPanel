<script setup lang="ts">
import { i18n } from '@/locales'
import type { DiskGuardConfig, DiskGuardEvent, DiskGuardStatus } from '@/api/modules/diskguard'
import { diskGuardApi } from '@/api/modules/diskguard'

defineOptions({
  name: 'DiskGuardIndex',
})

const status = ref<DiskGuardStatus | null>(null)
const loading = ref(false)
const saving = ref(false)
const restoring = ref(false)
const restoringId = ref(0)

// 配置表单（exclude 以逗号串编辑）
const form = ref<DiskGuardConfig>({ enabled: true, thresholdGB: 5, exclude: [] })
const excludeText = ref('')

const activeEvents = computed(() => (status.value?.events || []).filter(e => !e.restoredAt))

async function load() {
  loading.value = true
  try {
    status.value = await diskGuardApi.status()
    if (status.value) {
      form.value = { ...status.value.config, exclude: [...(status.value.config.exclude || [])] }
      excludeText.value = (status.value.config.exclude || []).join(',')
    }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('diskguard.toast.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const cfg = await diskGuardApi.updateConfig({
      enabled: form.value.enabled,
      thresholdGB: Number(form.value.thresholdGB) || 0,
      exclude: excludeText.value.split(',').map(s => s.trim()).filter(Boolean),
    })
    form.value = { ...cfg, exclude: [...(cfg.exclude || [])] }
    excludeText.value = (cfg.exclude || []).join(',')
    useFaToast().success(i18n.global.t('diskguard.toast.saved'))
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('diskguard.toast.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

function restoreAll() {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('diskguard.modal.restoreTitle'),
    content: i18n.global.t('diskguard.modal.restoreAllConfirm', { nodes: activeEvents.value.length }),
    onConfirm: async () => {
      restoring.value = true
      try {
        const r = await diskGuardApi.restore(0)
        useFaToast().success(i18n.global.t('diskguard.toast.restoreDone', { started: r.started.length, failed: r.failed.length }))
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('diskguard.toast.restoreFailed'), { description: e?.message })
      }
      finally {
        restoring.value = false
      }
    },
  })
}

function restoreOne(ev: DiskGuardEvent) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('diskguard.modal.restoreTitle'),
    content: i18n.global.t('diskguard.modal.restoreConfirm', { node: ev.nodeName, count: ev.containers.length }),
    onConfirm: async () => {
      restoringId.value = ev.id
      try {
        const r = await diskGuardApi.restore(ev.id)
        useFaToast().success(i18n.global.t('diskguard.toast.restoreDone', { started: r.started.length, failed: r.failed.length }))
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('diskguard.toast.restoreFailed'), { description: e?.message })
      }
      finally {
        restoringId.value = 0
      }
    },
  })
}

function fmtBytes(n?: number) {
  if (n === undefined || n === null) {
    return '—'
  }
  const gb = n / 1024 ** 3
  if (gb >= 1) {
    return `${gb.toFixed(1)} GB`
  }
  const mb = n / 1024 ** 2
  if (mb >= 1) {
    return `${mb.toFixed(0)} MB`
  }
  return `${n} B`
}

function fmtTime(t?: string | null) {
  if (!t) {
    return i18n.global.t('diskguard.events.none')
  }
  return new Date(t).toLocaleString('zh-CN', { hour12: false })
}

function isLow(free?: number) {
  if (free === undefined || !status.value) {
    return false
  }
  return free < status.value.config.thresholdGB * 1024 ** 3
}

let timer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  load()
  timer = setInterval(load, 30_000)
})
onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="hard-drive" :size="24" />
          <span>{{ $t('diskguard.page.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('diskguard.page.desc') }}</span>
      </template>
      <FaButton variant="outline" size="sm" @click="load">
        <FaIcon name="i-lucide:refresh-cw" class="mr-1" :class="loading ? 'animate-spin' : ''" /> {{ $t('diskguard.page.refresh') }}
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <!-- 触发横幅 -->
      <div v-if="activeEvents.length" class="mb-4">
        <FaAlert variant="destructive" :title="$t('diskguard.banner.title')" :description="$t('diskguard.banner.desc')" />
      </div>

      <!-- 保护配置 -->
      <div class="mb-4 space-y-4 rounded-lg border p-5">
        <div class="flex items-center gap-2 text-sm font-medium">
          <FaIcon name="i-lucide:shield-check" class="text-primary" />
          {{ $t('diskguard.config.title') }}
        </div>
        <div class="flex items-center gap-3">
          <FaSwitch v-model="form.enabled" />
          <div>
            <div class="text-sm">{{ $t('diskguard.config.enabled') }}</div>
            <div class="text-xs text-muted-foreground">{{ $t('diskguard.config.enabledDesc') }}</div>
          </div>
        </div>
        <div>
          <div class="text-sm">{{ $t('diskguard.config.threshold') }}</div>
          <FaInput v-model="form.thresholdGB" type="number" class="mt-2 w-40" />
          <div class="mt-1 text-xs text-muted-foreground">{{ $t('diskguard.config.thresholdDesc') }}</div>
        </div>
        <div>
          <div class="text-sm">{{ $t('diskguard.config.exclude') }}</div>
          <FaInput v-model="excludeText" :placeholder="$t('diskguard.config.excludePlaceholder')" class="mt-2 w-full" />
          <div class="mt-1 text-xs text-muted-foreground">{{ $t('diskguard.config.excludeDesc') }}</div>
        </div>
        <div class="flex justify-end">
          <FaButton :loading="saving" @click="save">
            <FaIcon name="i-lucide:save" class="mr-1" /> {{ $t('diskguard.config.save') }}
          </FaButton>
        </div>
      </div>

      <!-- 节点状态 -->
      <div class="mb-4 rounded-lg border p-5">
        <div class="mb-3 flex items-center gap-2 text-sm font-medium">
          <FaIcon name="i-lucide:server" class="text-primary" />
          {{ $t('diskguard.nodes.title') }}
        </div>
        <div v-if="!status?.nodes?.length" class="py-6 text-center text-sm text-muted-foreground">
          {{ $t('diskguard.nodes.empty') }}
        </div>
        <div v-else class="flex flex-col gap-2">
          <div v-for="n in status.nodes" :key="n.nodeId" class="rounded-md border p-3">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-sm font-medium">{{ n.nodeName }}</span>
              <span
                class="rounded-full px-2 py-0.5 text-xs"
                :class="n.online ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
              >{{ n.online ? $t('diskguard.nodes.online') : $t('diskguard.nodes.offline') }}</span>
              <span
                class="rounded-full px-2 py-0.5 text-xs"
                :class="n.triggered ? 'bg-red-500/10 text-red-600' : 'bg-sky-500/10 text-sky-600'"
              >{{ n.triggered ? $t('diskguard.nodes.triggered') : $t('diskguard.nodes.normal') }}</span>
              <span v-if="n.agentTooOld" class="rounded-full bg-amber-500/10 px-2 py-0.5 text-xs text-amber-600">
                {{ $t('diskguard.nodes.tooOld') }}
              </span>
            </div>
            <div v-if="n.disks?.length" class="mt-2 flex flex-col gap-1 text-xs text-muted-foreground">
              <div v-for="d in n.disks" :key="d.mountpoint" class="flex items-center gap-2">
                <code class="rounded bg-muted px-1.5 py-0.5 font-mono">{{ d.mountpoint }}</code>
                <span :class="isLow(d.free) ? 'font-medium text-red-600' : ''">
                  {{ $t('diskguard.events.freeAtTrigger') }} {{ fmtBytes(d.free) }} / {{ fmtBytes(d.total) }}
                </span>
              </div>
            </div>
            <div v-else class="mt-2 text-xs text-muted-foreground">
              {{ $t('diskguard.nodes.noDisk') }}
            </div>
          </div>
        </div>
      </div>

      <!-- 触发事件 -->
      <div class="rounded-lg border p-5">
        <div class="mb-3 flex items-center gap-2">
          <FaIcon name="i-lucide:history" class="text-primary" />
          <span class="text-sm font-medium">{{ $t('diskguard.events.title') }}</span>
          <FaButton
            v-if="activeEvents.length" class="ml-auto" variant="destructive" size="sm" :loading="restoring"
            @click="restoreAll"
          >
            <FaIcon name="i-lucide:rotate-ccw" class="mr-1" /> {{ $t('diskguard.events.restoreAll') }}
          </FaButton>
        </div>
        <div v-if="!status?.events?.length" class="py-6 text-center text-sm text-muted-foreground">
          {{ $t('diskguard.events.empty') }}
        </div>
        <div v-else class="flex flex-col gap-2">
          <div
            v-for="ev in status.events" :key="ev.id"
            class="rounded-md border p-3"
            :class="ev.restoredAt ? '' : 'border-red-500/40 bg-red-500/5'"
          >
            <div class="flex flex-wrap items-center gap-2 text-sm">
              <span class="font-medium">{{ ev.nodeName }}</span>
              <span
                class="rounded-full px-2 py-0.5 text-xs"
                :class="ev.restoredAt ? 'bg-muted text-muted-foreground' : 'bg-red-500/10 text-red-600'"
              >{{ ev.restoredAt ? $t('diskguard.events.restoredAt') : $t('diskguard.nodes.triggered') }}</span>
              <span class="text-xs text-muted-foreground">
                {{ $t('diskguard.events.triggeredAt') }}: {{ fmtTime(ev.triggeredAt) }}
              </span>
              <span v-if="ev.restoredAt" class="text-xs text-muted-foreground">
                {{ $t('diskguard.events.restoredAt') }}: {{ fmtTime(ev.restoredAt) }}
              </span>
              <span class="text-xs text-muted-foreground">
                {{ $t('diskguard.events.freeAtTrigger') }}: {{ fmtBytes(ev.freeBytes) }}
              </span>
              <FaButton
                v-if="!ev.restoredAt" class="ml-auto" size="sm"
                :loading="restoringId === ev.id" @click="restoreOne(ev)"
              >
                {{ $t('diskguard.events.restore') }}
              </FaButton>
            </div>
            <div class="mt-1 text-xs text-muted-foreground">
              {{ $t('diskguard.events.containers') }} ({{ ev.containers.length }}):
              <span class="font-mono" :title="ev.containers.map(c => c.name).join(', ')">
                {{ ev.containers.map(c => c.name).slice(0, 12).join(', ') }}{{ ev.containers.length > 12 ? '…' : '' }}
              </span>
            </div>
            <div v-if="ev.remark" class="mt-1 text-xs text-amber-600">
              {{ ev.remark }}
            </div>
          </div>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
