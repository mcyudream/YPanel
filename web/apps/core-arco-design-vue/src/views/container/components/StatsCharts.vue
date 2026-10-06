<script setup lang="ts">
import echarts from '@/utils/echarts'
import apiContainer from '@/api/modules/container'

// 容器实时资源图表（M23）：3s 轮询 docker stats 单次采样，滚动窗口折线。
const props = defineProps<{
  containerId: string
  active?: boolean
}>()

const cpuRef = useTemplateRef<HTMLElement>('cpuChart')
const netRef = useTemplateRef<HTMLElement>('netChart')
const error = ref('')
const latest = ref<{ cpuPct: number, memUsed: number, memLimit: number, rx: number, tx: number } | null>(null)

const WINDOW = 60
let cpuChart: echarts.ECharts | null = null
let netChart: echarts.ECharts | null = null
let timer: ReturnType<typeof setInterval> | null = null
let prev: { rx: number, tx: number, at: number } | null = null

const cpuData = ref<number[]>([])
const memData = ref<number[]>([])
const rxData = ref<number[]>([])
const txData = ref<number[]>([])
const timeData = ref<string[]>([])

function parse(s: Record<string, any>) {
  const cpuDelta = (s.cpu_stats?.cpu_usage?.total_usage ?? 0) - (s.precpu_stats?.cpu_usage?.total_usage ?? 0)
  const sysDelta = (s.cpu_stats?.system_cpu_usage ?? 0) - (s.precpu_stats?.system_cpu_usage ?? 0)
  const ncpu = s.cpu_stats?.online_cpus || 1
  const cpuPct = sysDelta > 0 ? (cpuDelta / sysDelta) * ncpu * 100 : 0
  const memUsed = Math.max(0, (s.memory_stats?.usage ?? 0) - (s.memory_stats?.stats?.cache ?? 0))
  const memLimit = s.memory_stats?.limit ?? 0
  let rx = 0
  let tx = 0
  for (const ifc of Object.values<any>(s.networks || {})) {
    rx += ifc.rx_bytes ?? 0
    tx += ifc.tx_bytes ?? 0
  }
  return { cpuPct, memUsed, memLimit, rx, tx }
}

function push(arr: { value: unknown[] }, v: unknown) {
  arr.value = [...arr.value.slice(-(WINDOW - 1)), v]
}

async function sample() {
  try {
    const s = await apiContainer.stats(props.containerId)
    error.value = ''
    const p = parse(s)
    latest.value = { cpuPct: p.cpuPct, memUsed: p.memUsed, memLimit: p.memLimit, rx: 0, tx: 0 }
    const now = Date.now()
    if (prev) {
      const dt = (now - prev.at) / 1000
      if (dt > 0) {
        latest.value.rx = Math.max(0, (p.rx - prev.rx) / dt)
        latest.value.tx = Math.max(0, (p.tx - prev.tx) / dt)
      }
    }
    prev = { rx: p.rx, tx: p.tx, at: now }
    push(cpuData, Number(p.cpuPct.toFixed(2)))
    push(memData, Number((p.memUsed / 1048576).toFixed(1)))
    push(rxData, Number((latest.value.rx / 1024).toFixed(1)))
    push(txData, Number((latest.value.tx / 1024).toFixed(1)))
    push(timeData, new Date(now).toLocaleTimeString('zh-CN', { hour12: false }))
    render()
  }
  catch (e: any) {
    error.value = e?.message || 'stats 采样失败'
    stopTimer()
  }
}

function baseOption(title: string, yFmt: (v: number) => string) {
  return {
    animation: false,
    grid: { left: 56, right: 16, top: 28, bottom: 22 },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: timeData.value, axisLabel: { fontSize: 10 } },
    yAxis: { type: 'value', axisLabel: { fontSize: 10, formatter: yFmt }, splitLine: { lineStyle: { opacity: 0.4 } } },
    title: { text: title, textStyle: { fontSize: 12 }, left: 8, top: 2 },
  }
}

function render() {
  if (cpuChart) {
    cpuChart.setOption({
      ...baseOption('CPU / 内存', (v: number) => v >= 1024 ? `${(v / 1024).toFixed(1)}G` : `${v}M`),
      series: [
        { name: 'CPU%', type: 'line', data: cpuData.value, showSymbol: false, smooth: true, lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.12 } },
        { name: '内存MB', type: 'line', data: memData.value, showSymbol: false, smooth: true, yAxisIndex: 0 },
      ],
      legend: { top: 2, right: 8, textStyle: { fontSize: 10 } },
    })
  }
  if (netChart) {
    netChart.setOption({
      ...baseOption('网络 IO（KB/s）', (v: number) => `${v}`),
      series: [
        { name: '下行', type: 'line', data: rxData.value, showSymbol: false, smooth: true, areaStyle: { opacity: 0.12 } },
        { name: '上行', type: 'line', data: txData.value, showSymbol: false, smooth: true },
      ],
      legend: { top: 2, right: 8, textStyle: { fontSize: 10 } },
    })
  }
}

function startTimer() {
  if (timer) {
    return
  }
  sample()
  timer = setInterval(sample, 3000)
}

function stopTimer() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

function init() {
  if (cpuRef.value && !cpuChart) {
    cpuChart = echarts.init(cpuRef.value)
  }
  if (netRef.value && !netChart) {
    netChart = echarts.init(netRef.value)
  }
}

watch(() => props.active, (v) => {
  if (v) {
    nextTick(() => {
      init()
      render()
      startTimer()
    })
  }
  else {
    stopTimer()
  }
}, { immediate: true })

const ro = new ResizeObserver(() => {
  cpuChart?.resize()
  netChart?.resize()
})

onMounted(() => {
  if (cpuRef.value) {
    ro.observe(cpuRef.value)
  }
  if (netRef.value) {
    ro.observe(netRef.value)
  }
})

onBeforeUnmount(() => {
  stopTimer()
  ro.disconnect()
  cpuChart?.dispose()
  netChart?.dispose()
  cpuChart = null
  netChart = null
})
</script>

<template>
  <div class="flex flex-col gap-3">
    <div v-if="latest" class="grid grid-cols-2 gap-3 md:grid-cols-4">
      <div class="rounded-md border p-2.5">
        <div class="text-xs text-muted-foreground">
          CPU
        </div>
        <div class="mt-0.5 font-mono text-lg tabular-nums">
          {{ latest.cpuPct.toFixed(1) }}%
        </div>
      </div>
      <div class="rounded-md border p-2.5">
        <div class="text-xs text-muted-foreground">
          内存
        </div>
        <div class="mt-0.5 font-mono text-lg tabular-nums">
          {{ (latest.memUsed / 1048576).toFixed(1) }}<span class="text-xs text-muted-foreground"> / {{ latest.memLimit ? (latest.memLimit / 1048576).toFixed(0) : '?' }} MB</span>
        </div>
      </div>
      <div class="rounded-md border p-2.5">
        <div class="text-xs text-muted-foreground">
          下行
        </div>
        <div class="mt-0.5 font-mono text-lg tabular-nums">
          {{ (latest.rx / 1024).toFixed(1) }} <span class="text-xs text-muted-foreground">KB/s</span>
        </div>
      </div>
      <div class="rounded-md border p-2.5">
        <div class="text-xs text-muted-foreground">
          上行
        </div>
        <div class="mt-0.5 font-mono text-lg tabular-nums">
          {{ (latest.tx / 1024).toFixed(1) }} <span class="text-xs text-muted-foreground">KB/s</span>
        </div>
      </div>
    </div>
    <div v-if="error" class="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-700 dark:bg-amber-950/30 dark:text-amber-400">
      {{ error }}（容器未运行或刚启动时采样可能失败，切回该页自动重试）
    </div>
    <div ref="cpuChart" class="h-56 w-full rounded-md border" />
    <div ref="netChart" class="h-48 w-full rounded-md border" />
  </div>
</template>
