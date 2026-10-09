<script setup lang="ts">
// Docker 概览小组件（medium）：容器/镜像/卷/网络计数 + 占用，挂载拉一次 + 120s 低频
// （system df 由 daemon 实算较慢，API 明确勿进高频轮询）。
import apiDockerUsage from '@/api/modules/dockerusage'
import type { DockerUsage } from '@/api/modules/dockerusage'
import { i18n } from '@/locales'
import { onBeforeUnmount, onMounted, reactive } from 'vue'

const state = reactive({
  containers: 0,
  images: 0,
  volumes: 0,
  networks: 0,
  size: '—',
  offline: false,
})

let timer: ReturnType<typeof setInterval> | null = null

function fmtBytes(n: number): string {
  if (n >= 1024 ** 3) {
    return `${(n / 1024 ** 3).toFixed(1)} GB`
  }
  if (n >= 1024 ** 2) {
    return `${(n / 1024 ** 2).toFixed(0)} MB`
  }
  return `${(n / 1024).toFixed(0)} KB`
}

async function load() {
  try {
    const u: DockerUsage = await apiDockerUsage.usage()
    state.containers = u.containersCount
    state.images = u.imagesCount
    state.volumes = u.volumesCount
    state.networks = u.networksCount
    state.size = fmtBytes(u.imagesTotalSize + u.containersRwSize + u.volumesTotalSize)
    state.offline = false
  }
  catch {
    state.offline = true
  }
}

onMounted(() => {
  void load()
  timer = setInterval(() => void load(), 120000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})

const cells = computed(() => [
  { icon: 'i-lucide-container', label: i18n.global.t('desktop.docker.containers'), v: state.containers },
  { icon: 'i-lucide-disc-3', label: i18n.global.t('desktop.docker.images'), v: state.images },
  { icon: 'i-lucide-archive', label: i18n.global.t('desktop.docker.volumes'), v: state.volumes },
  { icon: 'i-lucide-network', label: i18n.global.t('desktop.docker.networks'), v: state.networks },
])
</script>

<template>
  <div class="docker-body">
    <div class="yw-widget-head">
      <i class="yw-widget-head-icon i-lucide-container" />
      <span class="yw-widget-head-title">{{ $t('desktop.widgets.yp-docker') }}</span>
      <span class="yw-widget-dot" :style="{ background: state.offline ? '#ff5257' : '#62ba46' }" />
    </div>

    <div class="cells">
      <div v-for="c in cells" :key="c.label" class="cell">
        <i :class="c.icon" class="cell-icon" />
        <b class="cell-value">{{ c.v }}</b>
        <span class="cell-label">{{ c.label }}</span>
      </div>
    </div>
    <div class="foot">
      {{ $t('desktop.docker.totalUsage') }} <b>{{ state.size }}</b>
    </div>
  </div>
</template>

<style scoped>
.docker-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
  height: 100%;
}

.cells {
  display: grid;
  flex: 1;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
}

.cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  align-items: center;
  justify-content: center;
  padding: 6px 2px;
  background: oklch(var(--yw-foreground) / 0.06);
  border-radius: 8px;
}

.cell-icon {
  font-size: 14px;
  color: oklch(var(--yw-primary));
}

.cell-value {
  font-size: 15px;
  color: oklch(var(--yw-foreground));
  font-variant-numeric: tabular-nums;
}

.cell-label {
  font-size: 10px;
  color: oklch(var(--yw-muted-foreground));
}

.foot {
  font-size: 10px;
  color: oklch(var(--yw-muted-foreground));
}

.foot b {
  color: oklch(var(--yw-foreground));
}
</style>
