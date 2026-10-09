<script setup lang="ts">
import type { MetricSample, SystemOverview } from '@/api/modules/system'
import apiSystem from '@/api/modules/system'
import { fmtBytes } from '@/utils/format'
import { i18n } from '@/locales'

// 监控侧栏：系统/CPU(每核)/内存环形/网络(速率+趋势)/磁盘/负载，按节点轮询 overview+history。
// active=false（面板收起）时暂停轮询。
const props = withDefaults(defineProps<{
  node?: string
  active?: boolean
}>(), {
  node: 'local',
  active: true,
})

const overview = ref<SystemOverview | null>(null)
const history = ref<MetricSample[]>([])
const loadError = ref('')

let overviewTimer: ReturnType<typeof setInterval> | null = null
let historyTimer: ReturnType<typeof setInterval> | null = null

async function loadOverview() {
  try {
    overview.value = await apiSystem.overview(props.node)
    loadError.value = ''
  }
  catch (e: unknown) {
    loadError.value = e && typeof e === 'object' && 'message' in e ? String((e as { message: unknown }).message) : i18n.global.t('components.ydMonitorSidebar.collectFailed')
  }
}

async function loadHistory() {
  try {
    history.value = await apiSystem.history(300, props.node)
  }
  catch {}
}

function startPolling() {
  stopPolling()
  loadOverview()
  loadHistory()
  overviewTimer = setInterval(loadOverview, 4000)
  historyTimer = setInterval(loadHistory, 30000)
}

function stopPolling() {
  if (overviewTimer) {
    clearInterval(overviewTimer)
    overviewTimer = null
  }
  if (historyTimer) {
    clearInterval(historyTimer)
    historyTimer = null
  }
}

watch(() => props.node, () => {
  overview.value = null
  history.value = []
  if (props.active) {
    startPolling()
  }
}, { immediate: true })

watch(() => props.active, (v) => {
  if (v) {
    startPolling()
  }
  else {
    stopPolling()
  }
})

onMounted(() => {
  if (props.active) {
    startPolling()
  }
})

onBeforeUnmount(stopPolling)

// ---- 格式化 ----
function fmtUptime(sec: number) {
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  if (d > 0) {
    return i18n.global.t('components.ydMonitorSidebar.uptimeDh', { d, h })
  }
  if (h > 0) {
    return i18n.global.t('components.ydMonitorSidebar.uptimeHm', { h, m })
  }
  return i18n.global.t('components.ydMonitorSidebar.uptimeM', { m })
}

function pct(used: number, total: number) {
  return total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0
}

function usageColor(p: number) {
  return p >= 90 ? '#ef4444' : p >= 70 ? '#f59e0b' : '#10b981'
}

const memPct = computed(() => overview.value ? pct(overview.value.memory.used, overview.value.memory.total) : 0)
// r=26 圆环周长
const MEM_RING_C = 2 * Math.PI * 26

const cores = computed(() => (overview.value?.cpu.perCore ?? []).slice(0, 16))

// 趋势条（CSS 迷你柱状，取 history 末尾 N 个采样）
function sparkline(key: 'cpuPercent' | 'rxSpeedBps' | 'txSpeedBps', n = 60) {
  const samples = history.value.slice(-n)
  const max = Math.max(...samples.map(s => s[key]), 0.0001)
  return samples.map(s => Math.max(4, Math.round((s[key] / max) * 100)))
}
</script>

<template>
  <div class="flex size-full min-h-0 min-w-0 flex-col overflow-hidden">
    <!-- 头部 -->
    <div class="flex h-8 shrink-0 items-center gap-1 border-b bg-muted/40 px-2 text-[13px]">
      <YdMorphIcon name="activity" :size="14" class="text-muted-foreground" />
      <span class="text-xs font-medium text-muted-foreground">{{ $t('components.ydMonitorSidebar.title') }}</span>
      <span class="ml-auto truncate text-[11px] text-muted-foreground" :title="overview?.hostname">
        {{ overview?.hostname || '—' }}
      </span>
    </div>

    <div class="min-h-0 flex-1 space-y-2 overflow-auto p-2">
      <div v-if="loadError && !overview" class="rounded-md border border-red-300 bg-red-50 p-2 text-xs text-red-600 dark:bg-red-950/30 dark:text-red-400">
        {{ loadError }}
      </div>

      <template v-if="overview">
        <!-- 系统 -->
        <div class="rounded-lg border bg-card p-2.5">
          <div class="mb-1.5 flex items-center gap-1.5 text-xs font-medium">
            <YdMorphIcon name="info" :size="13" class="text-emerald-500" />
            {{ $t('components.ydMonitorSidebar.system') }}
          </div>
          <div class="truncate text-sm" :title="overview.os">{{ overview.os || overview.platform }}</div>
          <div class="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-[11px] text-muted-foreground">
            <span>{{ overview.arch }}</span>
            <span>CPU ×{{ overview.cpu.logicalCount }}</span>
            <span>{{ $t('components.ydMonitorSidebar.uptime', { time: fmtUptime(overview.uptime) }) }}</span>
          </div>
          <div class="mt-1 flex gap-3 text-[11px] text-muted-foreground">
            <span>{{ $t('components.ydMonitorSidebar.load', { l1: overview.load.load1.toFixed(2), l5: overview.load.load5.toFixed(2), l15: overview.load.load15.toFixed(2) }) }}</span>
          </div>
        </div>

        <!-- CPU -->
        <div class="rounded-lg border bg-card p-2.5">
          <div class="mb-1.5 flex items-center justify-between">
            <div class="flex items-center gap-1.5 text-xs font-medium">
              <YdMorphIcon name="cpu" :size="13" class="text-emerald-500" />
              CPU
            </div>
            <span class="text-sm font-semibold tabular-nums" :style="{ color: usageColor(Math.round(overview.cpu.usagePercent)) }">
              {{ overview.cpu.usagePercent.toFixed(1) }}%
            </span>
          </div>
          <div class="h-1.5 overflow-hidden rounded-full bg-muted">
            <div class="h-full rounded-full transition-all duration-500" :style="{ width: `${overview.cpu.usagePercent}%`, background: usageColor(Math.round(overview.cpu.usagePercent)) }" />
          </div>
          <div class="mt-2 space-y-1">
            <div v-for="(c, i) in cores" :key="i" class="flex items-center gap-1.5">
              <span class="w-6 shrink-0 text-right text-[10px] tabular-nums text-muted-foreground">{{ i }}</span>
              <div class="h-1 flex-1 overflow-hidden rounded-full bg-muted">
                <div class="h-full rounded-full bg-emerald-500 transition-all duration-500" :style="{ width: `${Math.min(100, c)}%` }" />
              </div>
              <span class="w-10 shrink-0 text-right text-[10px] tabular-nums text-muted-foreground">{{ c.toFixed(1) }}%</span>
            </div>
            <div v-if="(overview.cpu.perCore?.length ?? 0) > 16" class="text-right text-[10px] text-muted-foreground">
              {{ $t('components.ydMonitorSidebar.first16Cores') }}
            </div>
          </div>
        </div>

        <!-- 内存 -->
        <div class="rounded-lg border bg-card p-2.5">
          <div class="mb-1.5 flex items-center justify-between">
            <div class="flex items-center gap-1.5 text-xs font-medium">
              <YdMorphIcon name="memory-stick" :size="13" class="text-emerald-500" />
              {{ $t('components.ydMonitorSidebar.memory') }}
            </div>
            <span class="text-[11px] tabular-nums text-muted-foreground">{{ fmtBytes(overview.memory.used) }} / {{ fmtBytes(overview.memory.total) }}</span>
          </div>
          <div class="flex items-center gap-3">
            <svg width="64" height="64" viewBox="0 0 64 64" class="shrink-0">
              <circle cx="32" cy="32" r="26" fill="none" stroke="currentColor" class="text-muted" stroke-width="7" />
              <circle
                cx="32" cy="32" r="26" fill="none" :stroke="usageColor(memPct)" stroke-width="7"
                stroke-linecap="round" :stroke-dasharray="MEM_RING_C" :stroke-dashoffset="MEM_RING_C * (1 - memPct / 100)"
                transform="rotate(-90 32 32)" class="transition-all duration-500"
              />
              <text x="32" y="36" text-anchor="middle" class="fill-current text-[13px] font-semibold" :style="{ fill: usageColor(memPct) }">{{ memPct }}%</text>
            </svg>
            <div class="min-w-0 flex-1 space-y-1 text-[11px] text-muted-foreground">
              <div class="flex justify-between">
                <span>{{ $t('components.ydMonitorSidebar.used') }}</span><span class="tabular-nums">{{ fmtBytes(overview.memory.used) }}</span>
              </div>
              <div class="flex justify-between">
                <span>{{ $t('components.ydMonitorSidebar.available') }}</span><span class="tabular-nums">{{ fmtBytes(overview.memory.available) }}</span>
              </div>
              <div v-if="overview.swap.total > 0" class="flex justify-between">
                <span>Swap</span><span class="tabular-nums">{{ fmtBytes(overview.swap.used) }} / {{ fmtBytes(overview.swap.total) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 网络 -->
        <div class="rounded-lg border bg-card p-2.5">
          <div class="mb-1.5 flex items-center gap-1.5 text-xs font-medium">
            <YdMorphIcon name="network" :size="13" class="text-emerald-500" />
            {{ $t('components.ydMonitorSidebar.network') }}
          </div>
          <div class="grid grid-cols-2 gap-2 text-[11px]">
            <div class="rounded-md bg-muted/50 p-1.5">
              <div class="flex items-center gap-1 text-muted-foreground">
                <span class="inline-block size-1.5 rounded-full bg-emerald-500" /> {{ $t('components.ydMonitorSidebar.upload') }}
              </div>
              <div class="mt-0.5 font-semibold tabular-nums">{{ fmtBytes(overview.network.txSpeedBps) }}/s</div>
              <div class="text-[10px] text-muted-foreground">{{ $t('components.ydMonitorSidebar.total', { n: fmtBytes(overview.network.txTotal) }) }}</div>
            </div>
            <div class="rounded-md bg-muted/50 p-1.5">
              <div class="flex items-center gap-1 text-muted-foreground">
                <span class="inline-block size-1.5 rounded-full bg-sky-500" /> {{ $t('components.ydMonitorSidebar.download') }}
              </div>
              <div class="mt-0.5 font-semibold tabular-nums">{{ fmtBytes(overview.network.rxSpeedBps) }}/s</div>
              <div class="text-[10px] text-muted-foreground">{{ $t('components.ydMonitorSidebar.total', { n: fmtBytes(overview.network.rxTotal) }) }}</div>
            </div>
          </div>
          <div class="mt-2 flex h-8 items-end gap-px" :title="$t('components.ydMonitorSidebar.downloadTrend')">
            <div
              v-for="(h, i) in sparkline('rxSpeedBps')"
              :key="i"
              class="min-w-0 flex-1 rounded-t-sm bg-sky-500/70"
              :style="{ height: `${h}%` }"
            />
          </div>
        </div>

        <!-- 磁盘 -->
        <div class="rounded-lg border bg-card p-2.5">
          <div class="mb-1.5 flex items-center gap-1.5 text-xs font-medium">
            <YdMorphIcon name="hard-drive" :size="13" class="text-emerald-500" />
            {{ $t('components.ydMonitorSidebar.disk') }}
          </div>
          <div class="space-y-2">
            <div v-for="d in overview.disks" :key="d.mountpoint">
              <div class="flex items-center justify-between text-[11px]">
                <span class="truncate font-mono" :title="d.mountpoint">{{ d.mountpoint }}</span>
                <span class="shrink-0 tabular-nums text-muted-foreground">
                  {{ fmtBytes(d.used) }} / {{ fmtBytes(d.total) }} · {{ d.usagePercent.toFixed(0) }}%
                </span>
              </div>
              <div class="mt-0.5 h-1.5 overflow-hidden rounded-full bg-muted">
                <div class="h-full rounded-full transition-all duration-500" :style="{ width: `${d.usagePercent}%`, background: usageColor(d.usagePercent) }" />
              </div>
              <div class="mt-0.5 text-[10px] text-muted-foreground">
                {{ d.fsType }}
              </div>
            </div>
          </div>
        </div>
      </template>

      <div v-else-if="!loadError" class="px-2 py-8 text-center text-xs text-muted-foreground">
        {{ $t('components.ydMonitorSidebar.collecting') }}
      </div>
    </div>
  </div>
</template>
