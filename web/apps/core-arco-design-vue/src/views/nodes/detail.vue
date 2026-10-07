<script setup lang="ts">
import echarts from '@/utils/echarts'
import { fmtBytes } from '@/utils/format'
import type { NodeItem } from '@/api/modules/node'
import apiNode from '@/api/modules/node'
import apiSystem, { type MetricRecord, type SystemOverview } from '@/api/modules/system'

defineOptions({
  name: 'NodesDetail',
})

const route = useRoute()
const router = useRouter()
const toast = useFaToast()

const nodeId = computed(() => String(route.params.id || 'local'))
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
  { label: '近 1 小时', seconds: 3600 },
  { label: '近 6 小时', seconds: 6 * 3600 },
  { label: '近 24 小时', seconds: 24 * 3600 },
  { label: '近 7 天', seconds: 7 * 24 * 3600 },
  { label: '近 30 天', seconds: 30 * 24 * 3600 },
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
    toast.error('加载历史监控失败', { description: e?.message })
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
  cpuChart?.setOption({
    animation: false,
    grid: { left: 45, right: 45, top: 35, bottom: 25 },
    legend: { data: hasSwap ? ['CPU %', '内存 %', 'Swap %', '负载'] : ['CPU %', '内存 %', '负载'], top: 0, textStyle: { fontSize: 11 } },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
    yAxis: [
      { type: 'value', max: 100, axisLabel: { formatter: '{value}%', fontSize: 10 } },
      { type: 'value', axisLabel: { fontSize: 10 }, splitLine: { show: false } },
    ],
    series: [
      { name: 'CPU %', type: 'line', showSymbol: false, data: list.map(s => s.cpu), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
      { name: '内存 %', type: 'line', showSymbol: false, data: list.map(s => s.mem), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
      ...(hasSwap
        ? [{ name: 'Swap %', type: 'line', showSymbol: false, data: list.map(s => s.swap ?? 0), lineStyle: { width: 1, type: 'dashed' as const } }]
        : []),
      { name: '负载', type: 'line', yAxisIndex: 1, showSymbol: false, data: list.map(s => Number(s.load1.toFixed(2))), lineStyle: { width: 1 } },
    ],
  }, { notMerge: true })
  netChart?.setOption({
    animation: false,
    grid: { left: 65, right: 20, top: 35, bottom: 25 },
    legend: { data: ['下行', '上行'], top: 0, textStyle: { fontSize: 11 } },
    tooltip: { trigger: 'axis', valueFormatter: (v: number) => `${fmtBytes(v)}/s` },
    xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
    yAxis: { type: 'value', axisLabel: { formatter: (v: number) => fmtBytes(v), fontSize: 10 } },
    series: [
      { name: '下行', type: 'line', showSymbol: false, data: list.map(s => s.rxSpeed), lineStyle: { width: 1 }, areaStyle: { opacity: 0.08 } },
      { name: '上行', type: 'line', showSymbol: false, data: list.map(s => s.txSpeed), lineStyle: { width: 1 }, areaStyle: { opacity: 0.08 } },
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
  return d > 0 ? `${d} 天 ${h} 小时` : `${h} 小时 ${Math.floor((sec % 3600) / 60)} 分钟`
}

function barClass(pct: number) {
  if (pct >= 90) return 'bg-red-500'
  if (pct >= 70) return 'bg-amber-500'
  return 'bg-emerald-500'
}

function goFiles() {
  router.push(nodeId.value === 'local' ? '/file_management' : `/file_management?node=${nodeId.value}`)
}
function goProcs() {
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
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <FaButton variant="ghost" size="icon-sm" title="返回节点列表" @click="router.push('/nodes')">
            <FaIcon name="i-lucide:arrow-left" class="text-sm" />
          </FaButton>
          <YdMorphIcon name="server" :size="24" />
          <span>{{ node?.name || (nodeId === 'local' ? '本机' : nodeId) }}</span>
          <span
            v-if="node"
            class="rounded-full px-2 py-0.5 text-xs"
            :class="node.online ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'"
          >
            {{ node.online ? '在线' : '离线' }}
          </span>
        </div>
      </template>
      <template #description>
        <span>{{ [node?.os, node?.arch, node?.version, node?.addr].filter(Boolean).join(' · ') || '系统监控探针' }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="goFiles">
          <FaIcon name="i-lucide:folder-open" class="mr-1" /> 文件管理
        </FaButton>
        <FaButton variant="outline" size="sm" @click="goProcs">
          <FaIcon name="i-lucide:cpu" class="mr-1" /> 进程与服务
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 实时概览 -->
      <div v-if="overview" class="mb-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-lg border p-3">
          <div class="flex items-center justify-between text-xs text-muted-foreground">
            <span>CPU（{{ overview.cpu.logicalCount }} 核）</span>
            <span class="truncate pl-2 font-mono text-[10px]" :title="overview.cpu.modelName">{{ overview.cpu.modelName }}</span>
          </div>
          <div class="mt-1 font-mono text-2xl tabular-nums">{{ overview.cpu.usagePercent.toFixed(1) }}%</div>
          <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
            <div class="h-full rounded-full transition-all" :class="barClass(overview.cpu.usagePercent)" :style="{ width: `${Math.min(100, overview.cpu.usagePercent)}%` }" />
          </div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">内存</div>
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
            {{ overview.swap?.total ? `${fmtBytes(overview.swap.used)} / ${fmtBytes(overview.swap.total)}` : '未启用' }}
          </div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">负载 / 运行时间</div>
          <div class="mt-1 font-mono text-2xl tabular-nums">{{ overview.load.load1.toFixed(2) }}</div>
          <div class="mt-2 text-xs text-muted-foreground">
            5min {{ overview.load.load5.toFixed(2) }} · 15min {{ overview.load.load15.toFixed(2) }}
          </div>
          <div class="mt-1 text-xs text-muted-foreground">已运行 {{ fmtUptime(overview.uptime) }}</div>
        </div>
        <div class="rounded-lg border p-3 md:col-span-2">
          <div class="text-xs text-muted-foreground">网络</div>
          <div class="mt-1 grid grid-cols-2 gap-2 font-mono text-sm tabular-nums">
            <span>↓ {{ fmtBytes(overview.network.rxSpeedBps) }}/s</span>
            <span>↑ {{ fmtBytes(overview.network.txSpeedBps) }}/s</span>
            <span class="text-xs text-muted-foreground">累计收 {{ fmtBytes(overview.network.rxTotal) }}</span>
            <span class="text-xs text-muted-foreground">累计发 {{ fmtBytes(overview.network.txTotal) }}</span>
          </div>
        </div>
        <div class="rounded-lg border p-3 md:col-span-2">
          <div class="text-xs text-muted-foreground">磁盘（{{ overview.disks.length }} 个挂载点）</div>
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
        {{ !node || node.online ? '实时数据加载中…' : '节点离线，无法获取实时数据' }}
      </div>

      <!-- 历史趋势 -->
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div class="text-sm font-medium">
          历史趋势
          <span class="ml-2 text-xs font-normal text-muted-foreground">每分钟采样，保留 30 天</span>
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
          <div class="text-xs text-muted-foreground">CPU 均值 / 峰值</div>
          <div class="mt-1 font-mono text-lg tabular-nums">{{ summary.cpuAvg }}% <span class="text-sm text-muted-foreground">/ {{ summary.cpuMax }}%</span></div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">内存 均值 / 峰值</div>
          <div class="mt-1 font-mono text-lg tabular-nums">{{ summary.memAvg }}% <span class="text-sm text-muted-foreground">/ {{ summary.memMax }}%</span></div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">下行峰值</div>
          <div class="mt-1 font-mono text-lg tabular-nums">{{ fmtBytes(summary.rxMax) }}/s</div>
        </div>
        <div class="rounded-lg border p-3">
          <div class="text-xs text-muted-foreground">上行峰值</div>
          <div class="mt-1 font-mono text-lg tabular-nums">{{ fmtBytes(summary.txMax) }}/s</div>
        </div>
      </div>

      <div v-if="loading && !samples.length" class="py-16 text-center text-sm text-muted-foreground">
        加载中…
      </div>
      <div v-else-if="!samples.length" class="py-16 text-center text-sm text-muted-foreground">
        该区间暂无采样数据（采集器每分钟落库一次，接入后约 1 分钟生成首个采样点）
      </div>
      <template v-else>
        <div ref="cpuChart" class="h-72 w-full" />
        <div ref="netChart" class="mt-2 h-56 w-full" />
      </template>
    </FaPageMain>
  </div>
</template>
