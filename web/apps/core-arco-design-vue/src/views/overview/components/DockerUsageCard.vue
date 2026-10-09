<script setup lang="ts">
import { fmtBytes } from '@/utils/format'
import echarts from '@/utils/echarts'
import apiDockerUsage from '@/api/modules/dockerusage'
import type { DockerUsage } from '@/api/modules/dockerusage'
import { i18n } from '@/locales'
import { OVERVIEW_PALETTE } from '../palette'

// M44 Docker 用量统计卡（自加载：system df 实算，懒加载 + 手动刷新，不进轮询）。
const router = useRouter()
const toast = useFaToast()

const usage = ref<DockerUsage>()
const usageError = ref('')
const usageCategory = ref<'containers' | 'images' | 'volumes'>('containers')
const usagePruneConfirm = ref(false)
const usagePruning = ref(false)

const usageCategories = [
  { key: 'containers', label: i18n.global.t('overview.docker.containers') },
  { key: 'images', label: i18n.global.t('overview.docker.images') },
  { key: 'volumes', label: i18n.global.t('overview.docker.volumes') },
] as const

const usageChartEl = ref<HTMLElement | null>(null)
let usageChartInstance: echarts.ECharts | null = null

async function loadUsage() {
  try {
    usage.value = await apiDockerUsage.usage()
    usageError.value = ''
    await nextTick()
    renderUsageChart()
  }
  catch (e: any) {
    usageError.value = e?.message || i18n.global.t('overview.collectFailed')
  }
}

function renderUsageChart() {
  const u = usage.value
  if (!u || !usageChartEl.value) {
    return
  }
  const items = usageCategory.value === 'containers' ? u.containerItems : usageCategory.value === 'images' ? u.imageItems : u.volumeItems
  const data = items
    .filter(i => i.size > 0)
    .sort((a, b) => b.size - a.size)
    .map(i => ({ name: i.name, value: i.size, sub: i.sub }))
  if (!usageChartInstance) {
    usageChartInstance = echarts.init(usageChartEl.value)
  }
  usageChartInstance.setOption({
    animation: false,
    tooltip: { formatter: (p: any) => `${p.name}<br/>${fmtBytes(p.value)}${p.data?.sub ? ` · ${p.data.sub}` : ''}` },
    series: [{
      type: 'treemap',
      data,
      roam: false,
      nodeClick: false,
      breadcrumb: { show: false },
      label: { show: true, formatter: '{b}', fontSize: 11, overflow: 'truncate', color: '#fff' },
      itemStyle: { borderColor: 'transparent', borderWidth: 0, gapWidth: 2 },
      color: OVERVIEW_PALETTE,
    }],
  }, true)
}

watch(usageCategory, () => renderUsageChart())

async function doPruneBuildCache() {
  usagePruning.value = true
  try {
    const res = await apiDockerUsage.buildCachePrune()
    usagePruneConfirm.value = false
    toast.success(i18n.global.t('overview.docker.pruneDone', { size: fmtBytes(res.freed) }))
    await loadUsage()
  }
  catch (e: any) {
    toast.error(i18n.global.t('overview.docker.pruneFailed'), { description: e?.message })
  }
  finally {
    usagePruning.value = false
  }
}

onMounted(() => {
  loadUsage()
})

onBeforeUnmount(() => {
  usageChartInstance?.dispose()
})
</script>

<template>
  <div class="rounded-xl border bg-background p-4">
    <div class="mb-3 flex items-center justify-between">
      <div class="flex items-center gap-2 text-sm font-medium">
        <YdMorphIcon name="chart-pie" :size="16" />
        {{ $t('overview.docker.title') }}
        <span class="text-xs font-normal text-muted-foreground">{{ $t('overview.docker.subtitle') }}</span>
      </div>
      <div class="flex items-center gap-3 text-xs text-muted-foreground">
        <span v-if="usage">{{ $t('overview.docker.lastUpdate', { time: new Date(usage.collectedAt).toLocaleString(i18n.global.locale.value === 'en-US' ? 'en-US' : 'zh-CN', { hour12: false }) }) }}</span>
        <button
          type="button"
          class="inline-flex cursor-pointer items-center gap-1 rounded-md border px-2 py-0.5 transition-colors hover:bg-accent/50"
          :title="$t('overview.docker.refreshTip')"
          @click="loadUsage()"
        >
          <YdMorphIcon name="refresh-cw" :size="12" /> {{ $t('common.refresh') }}
        </button>
      </div>
    </div>

    <div v-if="usageError" class="py-8 text-center text-sm text-muted-foreground">
      {{ $t('overview.docker.loadFailed', { msg: usageError }) }}
    </div>
    <div v-else-if="!usage" class="py-8 text-center text-sm text-muted-foreground">
      {{ $t('overview.docker.collecting') }}
    </div>
    <div v-else class="flex flex-col gap-6 xl:flex-row">
      <div class="min-w-0 flex-1">
        <div class="mb-2 flex gap-1.5">
          <button
            v-for="c in usageCategories"
            :key="c.key"
            type="button"
            class="cursor-pointer rounded-md px-2.5 py-1 text-xs transition-colors"
            :class="usageCategory === c.key ? 'bg-primary text-primary-foreground' : 'border hover:bg-accent/50'"
            @click="usageCategory = c.key"
          >
            {{ c.label }}
          </button>
        </div>
        <div ref="usageChartEl" class="h-72 w-full"></div>
      </div>
      <div class="grid shrink-0 grid-cols-2 gap-x-10 gap-y-5 xl:w-[400px]">
        <div>
          <div class="text-xs text-muted-foreground">
            {{ $t('overview.docker.containers') }}
          </div>
          <div class="mt-1 text-xl font-semibold tabular-nums">
            {{ fmtBytes(usage.containersRwSize) }}
          </div>
          <div class="mt-0.5 text-xs text-muted-foreground">
            {{ $t('overview.docker.containersCount', { n: usage.containersCount }) }}
          </div>
          <button type="button" class="mt-1 cursor-pointer text-xs text-primary hover:underline" @click="router.push('/container/list')">
            {{ $t('common.detail') }}
          </button>
        </div>
        <div>
          <div class="text-xs text-muted-foreground">
            {{ $t('overview.docker.images') }}
          </div>
          <div class="mt-1 text-xl font-semibold tabular-nums">
            {{ fmtBytes(usage.imagesTotalSize) }}
          </div>
          <div class="mt-0.5 text-xs text-muted-foreground">
            {{ $t('overview.docker.imagesCount', { n: usage.imagesCount }) }}
          </div>
          <button type="button" class="mt-1 cursor-pointer text-xs text-primary hover:underline" @click="router.push('/container/images')">
            {{ $t('common.detail') }}
          </button>
        </div>
        <div>
          <div class="text-xs text-muted-foreground">
            {{ $t('overview.docker.volumes') }}
          </div>
          <div class="mt-1 text-xl font-semibold tabular-nums">
            {{ fmtBytes(usage.volumesTotalSize) }}
          </div>
          <div class="mt-0.5 text-xs text-muted-foreground">
            {{ $t('overview.docker.volumesCount', { n: usage.volumesCount }) }}
          </div>
          <button type="button" class="mt-1 cursor-pointer text-xs text-primary hover:underline" @click="router.push('/container/volumes')">
            {{ $t('common.detail') }}
          </button>
        </div>
        <div>
          <div class="text-xs text-muted-foreground">
            {{ $t('overview.docker.buildCache') }}
          </div>
          <div class="mt-1 text-xl font-semibold tabular-nums">
            {{ fmtBytes(usage.buildCacheSize) }}
          </div>
          <div class="mt-0.5 text-xs text-muted-foreground">
            {{ $t('overview.docker.buildCacheCount', { n: usage.buildCacheCount }) }}
          </div>
          <button type="button" class="mt-1 cursor-pointer text-xs text-red-500 hover:underline" @click="usagePruneConfirm = true">
            {{ $t('common.clear') }}
          </button>
        </div>
        <div>
          <div class="text-xs text-muted-foreground">
            {{ $t('overview.docker.networks') }}
          </div>
          <div class="mt-1 text-xl font-semibold tabular-nums">
            {{ usage.networksCount }}
          </div>
          <button type="button" class="mt-1 cursor-pointer text-xs text-primary hover:underline" @click="router.push('/container/networks')">
            {{ $t('common.detail') }}
          </button>
        </div>
        <div>
          <div class="text-xs text-muted-foreground">
            {{ $t('overview.docker.hostPorts') }}
          </div>
          <div class="mt-1 text-xl font-semibold tabular-nums">
            {{ usage.hostPortsCount }}
          </div>
          <div class="mt-0.5 text-xs text-muted-foreground">
            {{ $t('overview.docker.hostPortsTip') }}
          </div>
        </div>
      </div>
    </div>

    <FaModal v-model="usagePruneConfirm" :title="$t('overview.docker.pruneTitle')">
      <div class="text-sm">
        {{ $t('overview.docker.pruneConfirm') }}
      </div>
      <template #footer>
        <FaButton variant="outline" @click="usagePruneConfirm = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="usagePruning" @click="doPruneBuildCache">
          {{ $t('overview.docker.pruneButton') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
