<script setup lang="ts">
import { fmtBytes } from '@/utils/format'
import echarts from '@/utils/echarts'
import apiSystem from '@/api/modules/system'
import type { SystemOverview, Dashboard } from '@/api/modules/system'
import { useNotificationCenterStore } from '@/store/modules/notificationCenter'
import { i18n } from '@/locales'
import WelcomeBanner from './components/WelcomeBanner.vue'
import StatusRing from './components/StatusRing.vue'
import AssetGrid from './components/AssetGrid.vue'
import DockerUsageCard from './components/DockerUsageCard.vue'
import DiskAnalysisCard from './components/DiskAnalysisCard.vue'

defineOptions({
  name: 'OverviewIndex',
})

const notificationCenter = useNotificationCenterStore()

const overview = ref<SystemOverview>()
const dashboard = ref<Dashboard>()
const errorMsg = ref('')

// ---- 实时轮询：指标 3s / 聚合 15s / 趋势 10s ----
let overviewTimer: ReturnType<typeof setInterval> | null = null
let dashboardTimer: ReturnType<typeof setInterval> | null = null
let historyTimer: ReturnType<typeof setInterval> | null = null

async function refreshOverview() {
  try {
    overview.value = await apiSystem.overview()
    errorMsg.value = ''
  }
  catch (e: any) {
    errorMsg.value = e?.message || i18n.global.t('overview.collectFailed')
  }
}

async function refreshDashboard() {
  try {
    dashboard.value = await apiSystem.dashboard()
  }
  catch {
    // 聚合失败静默：资产卡保持上次数据/占位（overview 失败已有全局提示）
  }
}

// ---- 趋势双图（echarts 懒初始化，同一数据源）----
const netChartEl = ref<HTMLElement | null>(null)
const sysChartEl = ref<HTMLElement | null>(null)
let netChartInstance: echarts.ECharts | null = null
let sysChartInstance: echarts.ECharts | null = null

async function loadHistory() {
  try {
    const samples = await apiSystem.history(600)
    if (samples.length === 0) {
      return
    }
    const locale = i18n.global.locale.value === 'en-US' ? 'en-US' : 'zh-CN'
    const times = samples.map(s => new Date(s.at).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit', second: '2-digit' }))
    if (netChartEl.value) {
      if (!netChartInstance) {
        netChartInstance = echarts.init(netChartEl.value)
      }
      netChartInstance.setOption({
        animation: false,
        grid: { left: 55, right: 15, top: 28, bottom: 25 },
        legend: { data: [i18n.global.t('overview.net.down'), i18n.global.t('overview.net.up')], top: 0, textStyle: { fontSize: 11 } },
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
        yAxis: { type: 'value', axisLabel: { formatter: (v: number) => fmtBytes(v), fontSize: 10 } },
        series: [
          { name: i18n.global.t('overview.net.down'), type: 'line', showSymbol: false, data: samples.map(s => s.rxSpeedBps), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
          { name: i18n.global.t('overview.net.up'), type: 'line', showSymbol: false, data: samples.map(s => s.txSpeedBps), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
        ],
      })
    }
    if (sysChartEl.value) {
      if (!sysChartInstance) {
        sysChartInstance = echarts.init(sysChartEl.value)
      }
      sysChartInstance.setOption({
        animation: false,
        grid: { left: 40, right: 15, top: 28, bottom: 25 },
        legend: { data: ['CPU %', i18n.global.t('overview.sys.memPercent')], top: 0, textStyle: { fontSize: 11 } },
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
        yAxis: { type: 'value', max: 100, axisLabel: { formatter: '{value}%', fontSize: 10 } },
        series: [
          { name: 'CPU %', type: 'line', showSymbol: false, data: samples.map(s => s.cpuPercent), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
          { name: i18n.global.t('overview.sys.memPercent'), type: 'line', showSymbol: false, data: samples.map(s => s.memPercent), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
        ],
      })
    }
  }
  catch {}
}

onMounted(() => {
  refreshOverview()
  refreshDashboard()
  loadHistory()
  overviewTimer = setInterval(refreshOverview, 3000)
  dashboardTimer = setInterval(refreshDashboard, 15000)
  historyTimer = setInterval(loadHistory, 10000)
})

onBeforeUnmount(() => {
  if (overviewTimer) {
    clearInterval(overviewTimer)
  }
  if (dashboardTimer) {
    clearInterval(dashboardTimer)
  }
  if (historyTimer) {
    clearInterval(historyTimer)
  }
  netChartInstance?.dispose()
  sysChartInstance?.dispose()
})

// ---- 状态四环 ----
type RingKind = 'load' | 'cpu' | 'mem' | 'disk'
const activeRing = ref<RingKind | null>(null)
// 关窗动画期间保持上一项内容，避免闪回"采集中"
const contentRing = ref<RingKind | null>(null)
watch(activeRing, (v) => {
  if (v) {
    contentRing.value = v
  }
})

const ringModalOpen = computed<boolean>({
  get: () => activeRing.value !== null,
  set: (v: boolean) => {
    if (!v) {
      activeRing.value = null
    }
  },
})

function diskToneClass(p: number) {
  if (p >= 90) {
    return 'text-red-500'
  }
  if (p >= 75) {
    return 'text-amber-500'
  }
  return 'text-emerald-500'
}

const rootDisk = computed(() => {
  const disks = overview.value?.disks || []
  return disks.find(d => d.mountpoint === '/') || disks.reduce<typeof disks[0] | undefined>((max, d) => (!max || d.usagePercent > max.usagePercent ? d : max), undefined)
})

const loadPercent = computed(() => {
  const o = overview.value
  if (!o || o.cpu.logicalCount === 0) {
    return 0
  }
  return Math.min(100, o.load.load1 / o.cpu.logicalCount * 100)
})

const loadEval = computed(() => {
  const p = loadPercent.value
  if (p >= 100) {
    return { text: i18n.global.t('overview.rings.loadHigh'), tone: 'text-red-500' }
  }
  if (p >= 75) {
    return { text: i18n.global.t('overview.rings.loadMid'), tone: 'text-amber-500' }
  }
  return { text: i18n.global.t('overview.rings.loadLow'), tone: 'text-emerald-500' }
})

const ringLabel = computed(() => ({
  load: i18n.global.t('overview.rings.load'),
  cpu: 'CPU',
  mem: i18n.global.t('overview.mem'),
  disk: i18n.global.t('overview.rings.disk'),
}))

const ringModalTitle = computed(() => (contentRing.value ? ringLabel.value[contentRing.value] : ''))
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="gauge" :size="24" />
          <span>{{ $t('overview.title') }}</span>
        </div>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="errorMsg" class="mb-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
        {{ errorMsg }}
      </div>

      <!-- 欢迎横幅 -->
      <WelcomeBanner :overview="overview ?? null" />

      <!-- 状态四环 -->
      <div class="mt-4 grid grid-cols-2 gap-3 xl:grid-cols-4">
        <StatusRing
          :percent="loadPercent"
          :label="ringLabel.load"
          :sub="overview ? `1m ${overview.load.load1.toFixed(2)}` : '—'"
          @click="activeRing = 'load'"
        />
        <StatusRing
          :percent="overview?.cpu.usagePercent ?? 0"
          :label="ringLabel.cpu"
          :sub="overview ? $t('overview.sys.cores', { n: overview.cpu.logicalCount }) : '—'"
          @click="activeRing = 'cpu'"
        />
        <StatusRing
          :percent="overview?.memory.usagePercent ?? 0"
          :label="ringLabel.mem"
          :sub="overview ? `${fmtBytes(overview.memory.used)} / ${fmtBytes(overview.memory.total)}` : '—'"
          @click="activeRing = 'mem'"
        />
        <StatusRing
          :percent="rootDisk?.usagePercent ?? 0"
          :label="ringLabel.disk"
          :sub="rootDisk ? rootDisk.mountpoint : '—'"
          @click="activeRing = 'disk'"
        />
      </div>

      <!-- 管理对象卡行 -->
      <div class="mt-4">
        <AssetGrid :dashboard="dashboard ?? null" />
      </div>

      <!-- 趋势双图 -->
      <div class="mt-4 grid gap-4 xl:grid-cols-2">
        <div class="rounded-xl border bg-background p-4">
          <div class="mb-2 flex items-center gap-2 text-sm font-medium">
            <YdMorphIcon name="arrow-down-up" :size="16" />
            {{ $t('overview.net.title') }}
            <span class="text-xs font-normal text-muted-foreground">{{ $t('overview.last10m') }}</span>
          </div>
          <div ref="netChartEl" class="h-48 w-full"></div>
        </div>
        <div class="rounded-xl border bg-background p-4">
          <div class="mb-2 flex items-center gap-2 text-sm font-medium">
            <YdMorphIcon name="chart-line" :size="16" />
            {{ $t('overview.trend.title') }}
            <span class="text-xs font-normal text-muted-foreground">{{ $t('overview.last10m') }}</span>
          </div>
          <div ref="sysChartEl" class="h-48 w-full"></div>
        </div>
      </div>

      <!-- Docker 用量 + 待关注（到期证书/通知） -->
      <div class="mt-4 grid gap-4 xl:grid-cols-3">
        <div class="xl:col-span-2">
          <DockerUsageCard />
        </div>
        <div class="flex flex-col gap-4">
          <!-- 到期证书 -->
          <div class="rounded-xl border bg-background p-4">
            <div class="mb-3 flex items-center gap-2 text-sm font-medium">
              <YdMorphIcon name="shield-check" :size="16" />
              {{ $t('overview.certs.title') }}
            </div>
            <div v-if="!dashboard" class="py-4 text-center text-xs text-muted-foreground">
              {{ $t('overview.sys.collecting') }}
            </div>
            <div v-else-if="dashboard.expiringCerts.length === 0" class="flex items-center gap-2 py-2 text-xs text-emerald-600 dark:text-emerald-400">
              <YdMorphIcon name="circle-check" :size="14" />
              {{ $t('overview.certs.allOk') }}
            </div>
            <div v-else class="flex flex-col gap-2">
              <button
                v-for="c in dashboard.expiringCerts"
                :key="c.id"
                type="button"
                class="flex cursor-pointer items-center gap-2 rounded-lg border px-2.5 py-2 text-left transition-colors hover:bg-accent/50"
                @click="$router.push('/certs')"
              >
                <div class="min-w-0 flex-1">
                  <div class="truncate text-xs font-medium" :title="c.certName">
                    {{ c.certName }}
                  </div>
                  <div class="truncate text-[11px] text-muted-foreground" :title="c.domain">
                    {{ c.domain }}
                  </div>
                </div>
                <span
                  class="shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium"
                  :class="c.status === 'expired' ? 'bg-red-100 text-red-600 dark:bg-red-950/50 dark:text-red-400' : c.daysLeft <= 7 ? 'bg-amber-100 text-amber-600 dark:bg-amber-950/50 dark:text-amber-400' : 'bg-blue-100 text-blue-600 dark:bg-blue-950/50 dark:text-blue-400'"
                >
                  {{ c.status === 'expired' ? $t('overview.certs.expired') : $t('overview.certs.daysLeft', { n: c.daysLeft }) }}
                </span>
              </button>
            </div>
          </div>

          <!-- 最近通知 -->
          <div v-if="dashboard?.recentNotifications?.length" class="rounded-xl border bg-background p-4">
            <div class="mb-3 flex items-center justify-between text-sm font-medium">
              <div class="flex items-center gap-2">
                <YdMorphIcon name="bell" :size="16" />
                {{ $t('overview.notify.title') }}
              </div>
              <FaButton variant="ghost" size="sm" @click="notificationCenter.open()">
                {{ $t('overview.notify.viewAll') }}
              </FaButton>
            </div>
            <div class="flex flex-col gap-2">
              <div v-for="n in dashboard.recentNotifications" :key="n.id" class="flex items-start gap-2 text-sm">
                <span
                  class="mt-1.5 inline-block size-1.5 shrink-0 rounded-full"
                  :class="n.level === 'error' ? 'bg-red-500' : n.level === 'warning' ? 'bg-orange-400' : n.level === 'success' ? 'bg-emerald-500' : 'bg-blue-400'"
                />
                <span class="shrink-0" :class="n.read ? 'opacity-60' : 'font-medium'">{{ n.title }}</span>
                <span class="min-w-0 flex-1 truncate text-xs text-muted-foreground" :title="n.content">{{ n.content }}</span>
                <span class="shrink-0 text-xs text-muted-foreground">{{ new Date(n.createdAt).toLocaleString(i18n.global.locale.value === 'en-US' ? 'en-US' : 'zh-CN', { hour12: false }) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 磁盘占用分析 -->
      <div class="mt-4">
        <DiskAnalysisCard :disks="overview?.disks ?? []" />
      </div>
    </FaPageMain>

    <!-- 环形明细弹窗 -->
    <FaModal v-model="ringModalOpen" :title="ringModalTitle">
      <!-- 负载 -->
      <div v-if="contentRing === 'load' && overview" class="flex flex-col gap-4">
        <div class="grid grid-cols-3 gap-3 text-center">
          <div class="rounded-lg border p-3">
            <div class="text-xs text-muted-foreground">1m</div>
            <div class="mt-1 text-xl font-bold tabular-nums">{{ overview.load.load1.toFixed(2) }}</div>
          </div>
          <div class="rounded-lg border p-3">
            <div class="text-xs text-muted-foreground">5m</div>
            <div class="mt-1 text-xl font-bold tabular-nums">{{ overview.load.load5.toFixed(2) }}</div>
          </div>
          <div class="rounded-lg border p-3">
            <div class="text-xs text-muted-foreground">15m</div>
            <div class="mt-1 text-xl font-bold tabular-nums">{{ overview.load.load15.toFixed(2) }}</div>
          </div>
        </div>
        <div class="text-center text-sm font-medium" :class="loadEval.tone">
          {{ loadEval.text }}
        </div>
      </div>
      <!-- CPU -->
      <div v-else-if="contentRing === 'cpu' && overview" class="flex flex-col gap-3">
        <div class="text-xs text-muted-foreground">
          {{ overview.cpu.modelName || $t('overview.sys.unknownModel') }} · {{ $t('overview.sys.cores', { n: overview.cpu.logicalCount }) }}
        </div>
        <div class="grid grid-cols-4 gap-2 md:grid-cols-8">
          <div v-for="(p, i) in overview.cpu.perCore" :key="i" class="flex flex-col items-center gap-1">
            <div class="flex h-16 w-full items-end overflow-hidden rounded bg-muted">
              <div
                class="w-full rounded-t transition-all duration-500"
                :class="diskToneClass(p)"
                :style="{ height: `${Math.min(100, Math.max(2, p))}%`, backgroundColor: 'currentColor' }"
              ></div>
            </div>
            <span class="text-[10px] tabular-nums text-muted-foreground">{{ p.toFixed(0) }}%</span>
          </div>
        </div>
      </div>
      <!-- 内存 -->
      <div v-else-if="contentRing === 'mem' && overview" class="flex flex-col gap-3 text-sm">
        <div class="flex justify-between">
          <span class="text-muted-foreground">{{ $t('overview.rings.memTotal') }}</span>
          <span class="tabular-nums">{{ fmtBytes(overview.memory.total) }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">{{ $t('overview.rings.memUsed') }}</span>
          <span class="tabular-nums">{{ fmtBytes(overview.memory.used) }}（{{ overview.memory.usagePercent.toFixed(1) }}%）</span>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">{{ $t('overview.rings.memAvail') }}</span>
          <span class="tabular-nums">{{ fmtBytes(overview.memory.available) }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">Swap</span>
          <span class="tabular-nums">{{ overview.swap.total > 0 ? `${fmtBytes(overview.swap.used)} / ${fmtBytes(overview.swap.total)}` : $t('overview.sys.swapDisabled') }}</span>
        </div>
      </div>
      <!-- 磁盘 -->
      <div v-else-if="contentRing === 'disk' && overview" class="flex flex-col gap-3">
        <div v-for="d in overview.disks" :key="d.mountpoint" class="flex items-center gap-3">
          <span class="w-28 truncate font-mono text-xs text-muted-foreground" :title="d.mountpoint">{{ d.mountpoint }}</span>
          <div class="h-2 flex-1 overflow-hidden rounded-full bg-muted">
            <div
              class="h-full rounded-full transition-all duration-500"
              :class="diskToneClass(d.usagePercent)"
              :style="{ width: `${Math.min(100, d.usagePercent)}%`, backgroundColor: 'currentColor' }"
            ></div>
          </div>
          <span class="w-36 text-right text-xs tabular-nums text-muted-foreground">
            {{ fmtBytes(d.used) }} / {{ fmtBytes(d.total) }} ({{ d.usagePercent }}%)
          </span>
        </div>
      </div>
      <div v-else class="py-6 text-center text-sm text-muted-foreground">
        {{ $t('overview.sys.collecting') }}
      </div>
    </FaModal>
  </div>
</template>
