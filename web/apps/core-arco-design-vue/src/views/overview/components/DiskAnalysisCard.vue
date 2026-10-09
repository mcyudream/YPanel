<script setup lang="ts">
import { fmtBytes } from '@/utils/format'
import echarts from '@/utils/echarts'
import apiDiskUsage from '@/api/modules/diskusage'
import type { DiskUsageTree } from '@/api/modules/diskusage'
import { i18n } from '@/locales'
import { OVERVIEW_PALETTE } from '../palette'

// M44 磁盘占用分析卡（自加载：du 实算 + 24h 缓存；treemap 下钻，Arco 色板重风格）。
interface DiskInfo {
  mountpoint: string
}

const props = defineProps<{
  disks: DiskInfo[]
}>()

const duMount = ref('')
const duPath = ref('')
const duData = ref<DiskUsageTree>()
const duLoading = ref(false)
const duError = ref('')
const duChartEl = ref<HTMLElement | null>(null)
let duChartInstance: echarts.ECharts | null = null
let duChartDom: HTMLElement | null = null

const duCrumbs = computed(() => {
  if (!duPath.value) {
    return []
  }
  const crumbs = [{ label: duMount.value || '/', path: duMount.value || '/' }]
  const rel = duMount.value === '/' ? duPath.value : duPath.value.startsWith(duMount.value) ? duPath.value.slice(duMount.value.length) : ''
  let acc = duMount.value === '/' ? '' : duMount.value
  for (const part of rel.split('/').filter(Boolean)) {
    acc = `${acc === '/' ? '' : acc}/${part}`
    crumbs.push({ label: part, path: acc })
  }
  return crumbs
})

function openDuDisk(mount: string) {
  duMount.value = mount
  duPath.value = mount
  loadDu()
}

async function loadDu(force = false) {
  if (!duPath.value) {
    return
  }
  duLoading.value = true
  try {
    duData.value = await apiDiskUsage.tree(duPath.value, force)
    duError.value = ''
    await nextTick()
    renderDuChart()
  }
  catch (e: any) {
    duError.value = e?.message || i18n.global.t('overview.du.statFailed')
  }
  finally {
    duLoading.value = false
  }
}

function renderDuChart() {
  const d = duData.value
  if (!d || !duChartEl.value) {
    return
  }
  // 加载态分支切换会重建图表容器：实例跟随 DOM 身份重建，避免操作到已分离的旧实例
  if (!duChartInstance || duChartDom !== duChartEl.value) {
    duChartInstance?.dispose()
    duChartInstance = echarts.init(duChartEl.value)
    duChartDom = duChartEl.value
  }
  duChartInstance.setOption({
    animation: false,
    tooltip: { formatter: (p: any) => `${p.data.path}<br/>${fmtBytes(p.value)}` },
    series: [{
      type: 'treemap',
      data: d.items.filter(i => i.size > 0).map(i => ({ name: i.name, value: i.size, path: i.path, isDir: i.isDir })),
      roam: false,
      nodeClick: false,
      breadcrumb: { show: false },
      // 名称+大小随块渲染，小块放不下被裁剪（近似"块足够大才显示"）
      label: { show: true, formatter: (p: any) => `${p.name}\n${fmtBytes(p.value)}`, fontSize: 11, overflow: 'truncate', color: '#fff' },
      itemStyle: { borderColor: 'transparent', borderWidth: 0, gapWidth: 2 },
      color: OVERVIEW_PALETTE,
    }],
  }, true)
  duChartInstance.off('click')
  duChartInstance.on('click', (p: any) => {
    if (p.data?.isDir && p.data.path) {
      duPath.value = p.data.path
      loadDu()
    }
  })
}

// 挂载点就绪后自动选中第一个
watch(() => props.disks, (disks) => {
  if (disks?.length && !duPath.value) {
    openDuDisk(disks[0].mountpoint)
  }
}, { immediate: true })

onBeforeUnmount(() => {
  duChartInstance?.dispose()
})
</script>

<template>
  <div class="rounded-xl border bg-background p-4">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2 text-sm font-medium">
        <YdMorphIcon name="hard-drive" :size="16" />
        {{ $t('overview.du.title') }}
        <span class="text-xs font-normal text-muted-foreground">{{ $t('overview.du.tip') }}</span>
      </div>
      <div class="flex items-center gap-1.5">
        <button
          v-for="d in disks"
          :key="d.mountpoint"
          type="button"
          class="cursor-pointer rounded-md px-2.5 py-1 text-xs transition-colors"
          :class="duMount === d.mountpoint ? 'bg-primary text-primary-foreground' : 'border hover:bg-accent/50'"
          @click="openDuDisk(d.mountpoint)"
        >
          {{ d.mountpoint }}
        </button>
      </div>
    </div>

    <div v-if="duError" class="py-8 text-center text-sm text-muted-foreground">
      {{ $t('overview.du.statFailedMsg', { msg: duError }) }}
    </div>
    <div v-else-if="duLoading && !duData" class="py-8 text-center text-sm text-muted-foreground">
      {{ $t('overview.du.scanning') }}
    </div>
    <template v-else-if="duData">
      <div class="mb-2 flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
        <template v-for="(c, i) in duCrumbs" :key="c.path">
          <span v-if="i" class="opacity-50">/</span>
          <button
            type="button"
            class="cursor-pointer rounded px-1 py-0.5 transition-colors hover:bg-accent/50"
            :class="i === duCrumbs.length - 1 ? 'font-medium text-foreground' : 'text-primary hover:underline'"
            @click="duPath = c.path; loadDu()"
          >
            {{ c.label }}
          </button>
        </template>
        <span class="ml-2">{{ $t('overview.du.total', { size: fmtBytes(duData.total) }) }}</span>
        <span class="ml-auto">{{ $t('overview.du.dataTime', { time: new Date(duData.collectedAt).toLocaleTimeString('zh-CN', { hour12: false }) }) }}</span>
        <button
          type="button"
          class="ml-2 inline-flex cursor-pointer items-center gap-1 rounded-md border px-2 py-0.5 transition-colors hover:bg-accent/50"
          :title="$t('overview.du.refreshTip')"
          @click="loadDu(true)"
        >
          <YdMorphIcon name="refresh-cw" :size="12" /> {{ $t('common.refresh') }}
        </button>
      </div>
      <div ref="duChartEl" class="h-80 w-full"></div>
    </template>
  </div>
</template>
