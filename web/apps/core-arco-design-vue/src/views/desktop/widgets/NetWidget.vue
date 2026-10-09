<script setup lang="ts">
// 网络速率小组件（small）：上下行速率两行大数字 + 彩色圆底箭头，5s 轮询。
import apiSystem from '@/api/modules/system'
import { i18n } from '@/locales'
import { onBeforeUnmount, onMounted, reactive } from 'vue'

const state = reactive({
  rx: '—',
  tx: '—',
  offline: false,
})

let timer: ReturnType<typeof setInterval> | null = null

function fmtSpeed(bps: number): string {
  if (bps >= 1024 ** 2) {
    return `${(bps / 1024 ** 2).toFixed(1)} MB/s`
  }
  if (bps >= 1024) {
    return `${(bps / 1024).toFixed(1)} KB/s`
  }
  return `${Math.round(bps)} B/s`
}

async function load() {
  try {
    const o = await apiSystem.overview()
    state.rx = fmtSpeed(o.network.rxSpeedBps)
    state.tx = fmtSpeed(o.network.txSpeedBps)
    state.offline = false
  }
  catch {
    state.offline = true
  }
}

onMounted(() => {
  void load()
  timer = setInterval(() => void load(), 5000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})

const rows = [
  { key: 'down', label: i18n.global.t('desktop.net.down'), icon: 'i-lucide-arrow-down', color: '#30d158', value: () => state.rx },
  { key: 'up', label: i18n.global.t('desktop.net.up'), icon: 'i-lucide-arrow-up', color: '#0a84ff', value: () => state.tx },
]
</script>

<template>
  <div class="net-body">
    <div class="net-head">
      <i class="net-head-icon i-lucide-network" />
      <span class="net-head-title">{{ $t('desktop.widgets.yp-net') }}</span>
      <span class="net-dot" :style="{ background: state.offline ? '#ff453a' : '#30d158' }" />
    </div>

    <div class="rows">
      <div v-for="row in rows" :key="row.key" class="row">
        <span class="row-icon" :style="{ background: `${row.color}22`, color: row.color }">
          <i :class="row.icon" />
        </span>
        <div class="row-text">
          <div class="row-value">{{ row.value() }}</div>
          <div class="row-label">{{ row.label }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.net-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  height: 100%;
}

.rows {
  display: flex;
  flex: 1;
  flex-direction: column;
  justify-content: space-evenly;
  gap: 4px;
}

.row {
  display: flex;
  gap: 10px;
  align-items: center;
}

.row-icon {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 30px;
  height: 30px;
  font-size: 15px;
  border-radius: 50%;
}

.row-text {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 1px;
}

.row-value {
  font-size: 14px;
  font-weight: 650;
  line-height: 1.2;
  color: var(--yw-label);
  font-variant-numeric: tabular-nums;
}

.row-label {
  font-size: 10px;
  color: var(--yw-label-secondary);
}
</style>
