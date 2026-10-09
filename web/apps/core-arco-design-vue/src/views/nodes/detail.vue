<script setup lang="ts">
import echarts from '@/utils/echarts'
import { fmtBytes } from '@/utils/format'
import type { NodeItem } from '@/api/modules/node'
import apiNode from '@/api/modules/node'
import apiSystem, { type MetricRecord, type SystemOverview } from '@/api/modules/system'
import { i18n } from '@/locales'
import { closestWindowId, useYwEmbed } from '@/views/desktop/embed'

defineOptions({
  name: 'NodesDetail',
})

// 桌面工作台承载时经 props 传入（launchOptions），经典模式走路由参数
const props = defineProps<{
  /** 节点 ID（webos 窗口承载时由 launchOptions 注入，优先于路由参数） */
  id?: string
}>()

const route = useRoute()
const router = useRouter()
const toast = useFaToast()

const nodeId = computed(() => String(props.id || route.params.id || 'local'))
const node = ref<NodeItem | null>(null)
const overview = ref<SystemOverview | null>(null)
const overviewLoading = ref(false)

// ---- 节点信息与实时概览 ----
async function loadNode() {
  try {
    const list = await apiNode.list()
    node.value = list.find(n => n.id === nodeId.value) || null
  }
  catch {}
}

async function loadOverview() {
  overviewLoading.value = true
  try {
    overview.value = await apiSystem.overview(nodeId.value === 'local' ? undefined : nodeId.value)
  }
  catch {
    // 离线节点：清空实时数据，在线状态以节点列表为准
    overview.value = null
  }
  finally {
    overviewLoading.value = false
  }
}

// ---- 历史趋势 ----
const ranges = [
  { label: i18n.global.t('nodes.range1h'), seconds: 3600 },
  { label: i18n.global.t('nodes.range6h'), seconds: 6 * 3600 },
  { label: i18n.global.t('nodes.range24h'), seconds: 24 * 3600 },
  { label: i18n.global.t('nodes.range7d'), seconds: 7 * 24 * 3600 },
  { label: i18n.global.t('nodes.range30d'), seconds: 30 * 24 * 3600 },
]
const activeSeconds = ref(3600)
const loading = ref(false)
const samples = ref<MetricRecord[]>([])

const cpuChartRef = useTemplateRef<HTMLElement>('cpuChart')
const netChartRef = useTemplateRef<HTMLElement>('netChart')
let cpuChart: echarts.ECharts | null = null
let netChart: echarts.ECharts | null = null

// 大范围时抽样到 ≤720 点，避免曲线过密
function decimate(list: MetricRecord[], max = 720): MetricRecord[] {
  if (list.length <= max) {
    return list
  }
  const step = Math.ceil(list.length / max)
  const out: MetricRecord[] = []
  for (let i = 0; i < list.length; i += step) {
    out.push(list[i])
  }
  return out
}

function timeLabel(iso: string) {
  const d = new Date(iso)
  if (activeSeconds.value <= 24 * 3600) {
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false })
  }
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
}

async function loadHistory() {
  loading.value = true
  try {
    samples.value = decimate(await apiSystem.historyPersisted(activeSeconds.value, nodeId.value === 'local' ? undefined : nodeId.value))
    await nextTick()
    render()
  }
  catch (e: any) {
    toast.error(i18n.global.t('nodes.historyLoadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

function render() {
  const list = samples.value
  const times = list.map(s => timeLabel(s.at))
  if (cpuChartRef.value && !cpuChart) {
    cpuChart = echarts.init(cpuChartRef.value)
  }
  if (netChartRef.value && !netChart) {
    netChart = echarts.init(netChartRef.value)
  }
  const hasSwap = list.some(s => (s.swap ?? 0) > 0)
  const nameCpu = i18n.global.t('nodes.legendCpu')
  const nameMem = i18n.global.t('nodes.legendMem')
  const nameSwap = i18n.global.t('nodes.legendSwap')
  const nameLoad = i18n.global.t('nodes.legendLoad')
  cpuChart?.setOption({
    animation: false,
    grid: { left: 45, right: 45, top: 35, bottom: 25 },
    legend: { data: hasSwap ? [nameCpu, nameMem, nameSwap, nameLoad] : [nameCpu, nameMem, nameLoad], top: 0, textStyle: { fontSize: 11 } },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
    yAxis: [
      { type: 'value', max: 100, axisLabel: { formatter: '{value}%', fontSize: 10 } },
      { type: 'value', axisLabel: { fontSize: 10 }, splitLine: { show: false } },
    ],
    series: [
      { name: nameCpu, type: 'line', showSymbol: false, data: list.map(s => s.cpu), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
      { name: nameMem, type: 'line', showSymbol: false, data: list.map(s => s.mem), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
      ...(hasSwap
        ? [{ name: nameSwap, type: 'line', showSymbol: false, data: list.map(s => s.swap ?? 0), lineStyle: { width: 1, type: 'dashed' as const } }]
        : []),
      { name: nameLoad, type: 'line', yAxisIndex: 1, showSymbol: false, data: list.map(s => Number(s.load1.toFixed(2))), lineStyle: { width: 1 } },
    ],
  }, { notMerge: true })
  const nameDown = i18n.global.t('nodes.legendDown')
  const nameUp = i18n.global.t('nodes.legendUp')
  netChart?.setOption({
    animation: false,
    grid: { left: 65, right: 20, top: 35, bottom: 25 },
    legend: { data: [nameDown, nameUp], top: 0, textStyle: { fontSize: 11 } },
    tooltip: { trigger: 'axis', valueFormatter: (v: number) => `${fmtBytes(v)}/s` },
    xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
    yAxis: { type: 'value', axisLabel: { formatter: (v: number) => fmtBytes(v), fontSize: 10 } },
    series: [
      { name: nameDown, type: 'line', showSymbol: false, data: list.map(s => s.rxSpeed), lineStyle: { width: 1 }, areaStyle: { opacity: 0.08 } },
      { name: nameUp, type: 'line', showSymbol: false, data: list.map(s => s.txSpeed), lineStyle: { width: 1 }, areaStyle: { opacity: 0.08 } },
    ],
  }, { notMerge: true })
}

function switchRange(seconds: number) {
  activeSeconds.value = seconds
  loadHistory()
}

// ---- 区间统计摘要 ----
const summary = computed(() => {
  const list = samples.value
  if (!list.length) {
    return null
  }
  const avg = (f: (s: MetricRecord) => number) => list.reduce((a, s) => a + f(s), 0) / list.length
  return {
    count: list.length,
    cpuAvg: avg(s => s.cpu).toFixed(1),
    cpuMax: Math.max(...list.map(s => s.cpu)).toFixed(1),
    memAvg: avg(s => s.mem).toFixed(1),
    memMax: Math.max(...list.map(s => s.mem)).toFixed(1),
    rxMax: Math.max(...list.map(s => s.rxSpeed)),
    txMax: Math.max(...list.map(s => s.txSpeed)),
  }
})

// ---- 格式化 ----
function fmtUptime(sec?: number) {
  if (!sec) return '—'
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  return d > 0 ? i18n.global.t('nodes.uptimeDh', { d, h }) : i18n.global.t('nodes.uptimeHm', { h, m: Math.floor((sec % 3600) / 60) })
}

function barClass(pct: number) {
  if (pct >= 90) return 'bg-red-500'
  if (pct >= 70) return 'bg-amber-500'
  return 'bg-emerald-500'
}

// 桌面承载：下钻开新窗（router.push 会顶掉 /desktop 路由）、返回=关自己窗；经典模式保持路由
const ywEmbed = useYwEmbed()
// 根元素 ref：用于窗口内定位自身窗 id。不能用 getCurrentInstance——computed 首次求值发生在
// 点击期而非渲染期，届时拿不到实例，selfWinId 恒为 null，返回会误走 router.push 逃逸桌面
const rootRef = ref<HTMLElement | null>(null)
const selfWinId = computed(() => closestWindowId(rootRef.value))

function onBack() {
  if (ywEmbed && selfWinId.value) {
    ywEmbed.closeWindow(selfWinId.value)
    return
  }
  router.push('/nodes')
}

function goFiles() {
  if (ywEmbed) {
    ywEmbed.openApp('file', { title: i18n.global.t('desktop.apps.file'), launchOptions: nodeId.value === 'local' ? {} : { node: nodeId.value } })
    return
  }
  router.push(nodeId.value === 'local' ? '/file_management' : `/file_management?node=${nodeId.value}`)
}
function goProcs() {
  if (ywEmbed) {
    ywEmbed.openApp('processes', { title: i18n.global.t('desktop.apps.processes'), launchOptions: nodeId.value === 'local' ? {} : { node: nodeId.value } })
    return
  }
  router.push(nodeId.value === 'local' ? '/processes' : `/processes?node=${nodeId.value}`)
}

function handleResize() {
  cpuChart?.resize()
  netChart?.resize()
}

let overviewTimer: ReturnType<typeof setInterval> | null = null
let historyTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  loadNode()
  loadOverview()
  loadHistory()
  overviewTimer = setInterval(loadOverview, 10000)
  historyTimer = setInterval(loadHistory, 60000)
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  if (overviewTimer) clearInterval(overviewTimer)
  if (historyTimer) clearInterval(historyTimer)
  window.removeEventListener('resize', handleResize)
  cpuChart?.dispose()
  netChart?.dispose()
})
</script>

<template>
  <div ref="rootRef">
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <FaButton variant="ghost" size="icon-sm" :title="$t('nodes.backToList')" @click="onBack">
            <FaIcon name="i-lucide:arrow-left" class="text-sm" />
          </FaButton>
          <YdMorphIcon name="server" :size="24" />
          <span>{{ node?.name || (nodeId === 'local' ? $t('nodes.local') : nodeId) }}</span>
          <span
            v-if="node"
            class="rounded-full px-2 py-0.5 text-xs"
            :class="node.online ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'"
          >
            {{ node.online ? $t('nodes.online') : $t('nodes.offline') }}
          </span>
        </div>
      </template>
      <template #description>
        <span>{{ [node?.os, node?.arch, node?.version, node?.addr].filter(Boolean).join(' · ') || $t('nodes.probe') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="goFiles">
          <FaIcon name="i-lucide:folder-open" class="mr-1" /> {{ $t('nodes.filesTitle') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="goProcs">
          <FaIcon name="i-lucide:cpu" class="mr-1" /> {{ $t('nodes.procsTitle') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 实时概览 -->
      <div v-if="overview" class="mb-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-lg border p-3">
          <div class="flex items-center justify-between text-xs text-muted-foreground">
            <span>{{ $t('nodes.cpuCores', { n: overview.cpu.logicalCount }) }}</span>
            <span class="truncate pl-2 font-mono text-[10px]" :title="overview.cpu.modelName">{{ overview.cpu.modelName }}</span>
          </div>
          <div class="mt-1 font-mono text-2xl tabular-nums">{{ overview.cpu.usagePercent.toFixed(1) }}%</div>
          <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
            <div class="h-full rounded-full transition-all" :class="barClass(overview.cpu.usagePercent)" :style="{ width: `${Math.min(100, overview.cpu.usagePercent)}%` }" />
          </div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">{{ $t('nodes.memory') }}</div>
          <div class="mt-1 font-mono text-2xl tabular-nums">{{ overview.memory.usagePercent.toFixed(1) }}%</div>
          <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
            <div class="h-full rounded-full transition-all" :class="barClass(overview.memory.usagePercent)" :style="{ width: `${Math.min(100, overview.memory.usagePercent)}%` }" />
          </div>
          <div class="mt-1 text-xs text-muted-foreground">{{ fmtBytes(overview.memory.used) }} / {{ fmtBytes(overview.memory.total) }}</div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">Swap</div>
          <div class="mt-1 font-mono text-2xl tabular-nums">
            {{ overview.swap?.total ? `${overview.swap.usagePercent.toFixed(1)}%` : '—' }}
          </div>
          <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
            <div v-if="overview.swap?.total" class="h-full rounded-full transition-all" :class="barClass(overview.swap.usagePercent)" :style="{ width: `${Math.min(100, overview.swap.usagePercent)}%` }" />
          </div>
          <div class="mt-1 text-xs text-muted-foreground">
            {{ overview.swap?.total ? `${fmtBytes(overview.swap.used)} / ${fmtBytes(overview.swap.total)}` : $t('nodes.swapOff') }}
          </div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">{{ $t('nodes.loadUptime') }}</div>
          <div class="mt-1 font-mono text-2xl tabular-nums">{{ overview.load.load1.toFixed(2) }}</div>
          <div class="mt-2 text-xs text-muted-foreground">
            5min {{ overview.load.load5.toFixed(2) }} · 15min {{ overview.load.load15.toFixed(2) }}
          </div>
          <div class="mt-1 text-xs text-muted-foreground">{{ $t('nodes.ranFor', { t: fmtUptime(overview.uptime) }) }}</div>
        </div>
        <div class="rounded-lg border p-3 md:col-span-2">
          <div class="text-xs text-muted-foreground">{{ $t('nodes.network') }}</div>
          <div class="mt-1 grid grid-cols-2 gap-2 font-mono text-sm tabular-nums">
            <span>↓ {{ fmtBytes(overview.network.rxSpeedBps) }}/s</span>
            <span>↑ {{ fmtBytes(overview.network.txSpeedBps) }}/s</span>
            <span class="text-xs text-muted-foreground">{{ $t('nodes.rxTotal', { v: fmtBytes(overview.network.rxTotal) }) }}</span>
            <span class="text-xs text-muted-foreground">{{ $t('nodes.txTotal', { v: fmtBytes(overview.network.txTotal) }) }}</span>
          </div>
        </div>
        <div class="rounded-lg border p-3 md:col-span-2">
          <div class="text-xs text-muted-foreground">{{ $t('nodes.disks', { n: overview.disks.length }) }}</div>
          <div class="mt-2 max-h-28 space-y-2 overflow-auto pr-1">
            <div v-for="d in overview.disks" :key="d.mountpoint" class="text-xs">
              <div class="flex items-center justify-between">
                <span class="truncate font-mono" :title="d.mountpoint">{{ d.mountpoint }}</span>
                <span class="ml-2 shrink-0 tabular-nums text-muted-foreground">{{ d.usagePercent.toFixed(0) }}% · {{ fmtBytes(d.used) }} / {{ fmtBytes(d.total) }}</span>
              </div>
              <div class="mt-0.5 h-1 overflow-hidden rounded-full bg-muted">
                <div class="h-full rounded-full transition-all" :class="barClass(d.usagePercent)" :style="{ width: `${Math.min(100, d.usagePercent)}%` }" />
              </div>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="mb-4 rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
        {{ !node || node.online ? $t('nodes.realtimeLoading') : $t('nodes.nodeOfflineNoData') }}
      </div>

      <!-- 历史趋势 -->
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div class="text-sm font-medium">
          {{ $t('nodes.history') }}
          <span class="ml-2 text-xs font-normal text-muted-foreground">{{ $t('nodes.historyNote') }}</span>
        </div>
        <div class="flex flex-wrap items-center gap-1">
          <button
            v-for="r in ranges"
            :key="r.seconds"
            type="button"
            class="cursor-pointer rounded-md border px-2.5 py-1 text-xs transition-colors"
            :class="activeSeconds === r.seconds ? 'border-primary bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-accent/50'"
            @click="switchRange(r.seconds)"
          >
            {{ r.label }}
          </button>
        </div>
      </div>

      <div v-if="summary" class="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">{{ $t('nodes.cpuAvgMax') }}</div>
          <div class="mt-1 font-mono text-lg tabular-nums">{{ summary.cpuAvg }}% <span class="text-sm text-muted-foreground">/ {{ summary.cpuMax }}%</span></div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">{{ $t('nodes.memAvgMax') }}</div>
          <div class="mt-1 font-mono text-lg tabular-nums">{{ summary.memAvg }}% <span class="text-sm text-muted-foreground">/ {{ summary.memMax }}%</span></div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">{{ $t('nodes.rxPeak') }}</div>
          <div class="mt-1 font-mono text-lg tabular-nums">{{ fmtBytes(summary.rxMax) }}/s</div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">{{ $t('nodes.txPeak') }}</div>
          <div class="mt-1 font-mono text-lg tabular-nums">{{ fmtBytes(summary.txMax) }}/s</div>
        </div>
      </div>

      <div v-if="loading && !samples.length" class="py-16 text-center text-sm text-muted-foreground">
        {{ $t('common.loading') }}
      </div>
      <div v-else-if="!samples.length" class="py-16 text-center text-sm text-muted-foreground">
        {{ $t('nodes.noSamples') }}
      </div>
      <template v-else>
        <div ref="cpuChart" class="h-72 w-full" />
        <div ref="netChart" class="mt-2 h-56 w-full" />
      </template>
    </FaPageMain>
  </div>
</template>
