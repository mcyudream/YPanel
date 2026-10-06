<script setup lang="ts">
import { fmtBytes } from '@/utils/format'
import echarts from '@/utils/echarts'
import type { SystemOverview } from '@/api/modules/system'
import apiSystem from '@/api/modules/system'

const router = useRouter()

defineOptions({
  name: 'OverviewIndex',
})

const overview = ref<SystemOverview>()
const loading = ref(true)
const errorMsg = ref('')

// ---- 格式化 ----


function fmtUptime(sec: number): string {
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return d > 0 ? `${d} 天 ${h} 时 ${m} 分` : `${h} 时 ${m} 分`
}

// ---- 实时轮询 ----
let timer: ReturnType<typeof setInterval> | null = null

async function refresh() {
  try {
    overview.value = await apiSystem.overview()
    errorMsg.value = ''
  }
  catch (e: any) {
    errorMsg.value = e?.message || '采集失败'
  }
  finally {
    loading.value = false
  }
}

onMounted(() => {
  refresh()
  timer = setInterval(refresh, 3000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
  chart?.dispose()
})

// ---- 历史趋势图（echarts 懒初始化）----
const chartRef = useTemplateRef<HTMLElement>('chartRef')
let chart: echarts.ECharts | null = null

async function loadHistory() {
  try {
    const samples = await apiSystem.history(600)
    if (!chartRef.value) {
      return
    }
    if (!chart) {
      chart = echarts.init(chartRef.value)
    }
    const times = samples.map(s => new Date(s.at).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' }))
    chart.setOption({
      animation: false,
      grid: { left: 45, right: 60, top: 30, bottom: 25 },
      legend: { data: ['CPU %', '内存 %', '下行', '上行'], top: 0, textStyle: { fontSize: 11 } },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
      yAxis: [
        { type: 'value', max: 100, axisLabel: { formatter: '{value}%', fontSize: 10 } },
        { type: 'value', axisLabel: { formatter: (v: number) => fmtBytes(v), fontSize: 10 }, splitLine: { show: false } },
      ],
      series: [
        { name: 'CPU %', type: 'line', showSymbol: false, data: samples.map(s => s.cpuPercent), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
        { name: '内存 %', type: 'line', showSymbol: false, data: samples.map(s => s.memPercent), lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.08 } },
        { name: '下行', type: 'line', yAxisIndex: 1, showSymbol: false, data: samples.map(s => s.rxSpeedBps), lineStyle: { width: 1 } },
        { name: '上行', type: 'line', yAxisIndex: 1, showSymbol: false, data: samples.map(s => s.txSpeedBps), lineStyle: { width: 1 } },
      ],
    })
  }
  catch {}
}

onMounted(() => {
  loadHistory()
  const t = setInterval(loadHistory, 10000)
  onBeforeUnmount(() => clearInterval(t))
})

// ---- 展示辅助 ----
function diskColor(p: number) {
  if (p >= 90) {
    return 'text-red-500'
  }
  if (p >= 75) {
    return 'text-amber-500'
  }
  return 'text-emerald-500'
}

const stateCards = computed(() => {
  const o = overview.value
  if (!o) {
    return []
  }
  return [
    { label: 'CPU 使用率', value: `${o.cpu.usagePercent}%`, sub: `${o.cpu.logicalCount} 核 · ${o.cpu.modelName || '未知型号'}`, icon: 'cpu', percent: o.cpu.usagePercent },
    { label: '内存使用率', value: `${o.memory.usagePercent}%`, sub: `${fmtBytes(o.memory.used)} / ${fmtBytes(o.memory.total)}`, icon: 'memory-stick', percent: o.memory.usagePercent },
    { label: '网络速率', value: `${fmtBytes(o.network.rxSpeedBps)}/s`, sub: `↓ ${fmtBytes(o.network.rxSpeedBps)}/s · ↑ ${fmtBytes(o.network.txSpeedBps)}/s`, icon: 'arrow-down-up', percent: null },
    { label: '系统负载', value: `${o.load.load1.toFixed(2)}`, sub: `5m ${o.load.load5.toFixed(2)} · 15m ${o.load.load15.toFixed(2)}`, icon: 'activity', percent: null },
  ]
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="gauge" :size="24" />
          <span>主机概览</span>
        </div>
      </template>
      <template #description>
        <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
          <button
            type="button"
            class="inline-flex cursor-pointer items-center gap-1 rounded-md border px-2 py-0.5 transition-colors hover:bg-accent/50"
            title="切换到桌面工作台"
            @click="router.push('/desktop')"
          >
            <YdMorphIcon name="layout-grid" :size="13" /> 桌面工作台
          </button>
          <span class="flex items-center gap-1"><YdMorphIcon name="monitor" :size="14" /> {{ overview?.hostname || '—' }}</span>
          <span class="flex items-center gap-1"><YdMorphIcon name="hard-drive" :size="14" /> {{ overview?.platform || '—' }} · {{ overview?.arch }}</span>
          <span class="flex items-center gap-1"><YdMorphIcon name="clock" :size="14" /> 已运行 {{ overview ? fmtUptime(overview.uptime) : '—' }}</span>
        </div>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="errorMsg" class="mb-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
        {{ errorMsg }}
      </div>

      <!-- 状态卡片 -->
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="card in stateCards"
          :key="card.label"
          class="rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
        >
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">{{ card.label }}</span>
            <YdMorphIcon :name="card.icon" :size="18" class="text-primary opacity-70" />
          </div>
          <div class="mt-2 text-2xl font-bold tabular-nums">
            {{ card.value }}
          </div>
          <div class="mt-1 truncate text-xs text-muted-foreground" :title="card.sub">
            {{ card.sub }}
          </div>
          <div v-if="card.percent !== null" class="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
            <div
              class="h-full rounded-full bg-primary transition-all duration-500"
              :class="[card.percent >= 90 ? 'bg-red-500' : card.percent >= 75 ? 'bg-amber-500' : '']"
              :style="{ width: `${Math.min(100, card.percent)}%` }"
            />
          </div>
        </div>
      </div>

      <!-- 趋势图 -->
      <div class="mt-4 rounded-lg border bg-background p-4">
        <div class="mb-2 flex items-center gap-2 text-sm font-medium">
          <YdMorphIcon name="chart-line" :size="16" />
          近 10 分钟趋势
        </div>
        <div ref="chartRef" class="h-64 w-full" />
      </div>

      <!-- 磁盘 -->
      <div class="mt-4 rounded-lg border bg-background p-4">
        <div class="mb-3 flex items-center gap-2 text-sm font-medium">
          <YdMorphIcon name="hard-drive" :size="16" />
          磁盘用量
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <div v-for="d in overview?.disks || []" :key="d.mountpoint" class="flex items-center gap-3">
            <span class="w-28 truncate font-mono text-xs text-muted-foreground" :title="d.mountpoint">{{ d.mountpoint }}</span>
            <div class="h-2 flex-1 overflow-hidden rounded-full bg-muted">
              <div
                class="h-full rounded-full transition-all duration-500"
                :class="diskColor(d.usagePercent)"
                :style="{ width: `${Math.min(100, d.usagePercent)}%`, backgroundColor: 'currentColor' }"
              />
            </div>
            <span class="w-40 text-right text-xs tabular-nums text-muted-foreground">
              {{ fmtBytes(d.used) }} / {{ fmtBytes(d.total) }} ({{ d.usagePercent }}%)
            </span>
          </div>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
