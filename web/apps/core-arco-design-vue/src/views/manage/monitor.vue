<script setup lang="ts">
import echarts from '@/utils/echarts'
import { fmtBytes } from '@/utils/format'
import type { MetricRecord } from '@/api/modules/system'
import apiSystem from '@/api/modules/system'

defineOptions({
  name: 'ManageMonitor',
})

const toast = useFaToast()

// 时间范围：1h/6h/24h/7d/30d
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
// 实例变量不能与模板 ref 名（cpuChart/netChart）同名：SFC 编译器会把 ref="cpuChart"
// 编译成对同名 setup 变量的引用而非字符串 ref，useTemplateRef 桥接随之失效（仅生产构建触发）
let cpuChartInstance: echarts.ECharts | null = null
let netChartInstance: echarts.ECharts | null = null



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

async function load() {
  loading.value = true
  try {
    samples.value = decimate(await apiSystem.historyPersisted(activeSeconds.value))
    // 图表容器在 v-else 分支内，需等 DOM 更新后 ref 才存在，否则 init 被静默跳过
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
  if (cpuChartRef.value && !cpuChartInstance) {
    cpuChartInstance = echarts.init(cpuChartRef.value)
  }
  if (netChartRef.value && !netChartInstance) {
    netChartInstance = echarts.init(netChartRef.value)
  }
  cpuChartInstance?.setOption({
    animation: false,
    grid: { left: 45, right: 20, top: 35, bottom: 25 },
    legend: { data: ['CPU %', '内存 %', '负载'], top: 0, textStyle: { fontSize: 11 } },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
    yAxis: [
      { type: 'value', max: 100, axisLabel: { formatter: '{value}%', fontSize: 10 } },
      { type: 'value', axisLabel: { fontSize: 10 }, splitLine: { show: false } },
    ],
    series: [
      { name: 'CPU %', type: 'line', showSymbol: false, data: list.map(s => s.cpu), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
      { name: '内存 %', type: 'line', showSymbol: false, data: list.map(s => s.mem), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
      { name: '负载', type: 'line', yAxisIndex: 1, showSymbol: false, data: list.map(s => Number(s.load1.toFixed(2))), lineStyle: { width: 1 } },
    ],
  }, { notMerge: true })
  netChartInstance?.setOption({
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
  load()
}

// 区间统计摘要
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

function handleResize() {
  cpuChartInstance?.resize()
  netChartInstance?.resize()
}

onMounted(() => {
  load()
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  cpuChartInstance?.dispose()
  netChartInstance?.dispose()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="activity" :size="24" />
          <span>历史监控</span>
        </div>
      </template>
      <template #description>
        <span>CPU / 内存 / 负载 / 网络趋势（每分钟采样，保留 30 天）</span>
      </template>
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
    </FaPageHeader>

    <FaPageMain>
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
        该区间暂无采样数据（采集器每分钟落库一次，新面板需等待片刻）
      </div>
      <template v-else>
        <div ref="cpuChart" class="h-72 w-full" />
        <div ref="netChart" class="mt-2 h-56 w-full" />
      </template>
    </FaPageMain>
  </div>
</template>
