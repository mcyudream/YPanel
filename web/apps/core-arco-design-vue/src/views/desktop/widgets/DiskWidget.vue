<script setup lang="ts">
// 磁盘占用小组件（small/medium）：挂载点用量条 top3，真数据 30s 轮询。
import apiSystem from '@/api/modules/system'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

interface DiskRow {
  mountpoint: string
  usagePercent: number
  used: string
  total: string
}

const disks = ref<DiskRow[]>([])
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
    const o = await apiSystem.overview()
    disks.value = [...o.disks]
      .sort((a, b) => b.usagePercent - a.usagePercent)
      .slice(0, 3)
      .map(d => ({
        mountpoint: d.mountpoint,
        usagePercent: Math.round(d.usagePercent),
        used: fmtBytes(d.used),
        total: fmtBytes(d.total),
      }))
  }
  catch {}
}

onMounted(() => {
  void load()
  timer = setInterval(() => void load(), 30000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})

const rows = computed(() => disks.value)

function barColor(p: number): string {
  if (p >= 90) {
    return '#ff5257'
  }
  if (p >= 70) {
    return '#ff9f0a'
  }
  return 'oklch(var(--yw-primary))'
}
</script>

<template>
  <div class="yw-widget-body">
    <div class="yw-widget-head">
      <i class="yw-widget-head-icon i-lucide-hard-drive" />
      <span class="yw-widget-head-title">{{ $t('desktop.widgets.yp-disk') }}</span>
    </div>

    <div v-if="!rows.length" class="empty">
      {{ $t('common.noData') }}
    </div>
    <div v-else class="rows">
      <div v-for="d in rows" :key="d.mountpoint" class="row">
        <div class="row-top">
          <span class="mount" :title="d.mountpoint">{{ d.mountpoint }}</span>
          <span class="pct">{{ d.usagePercent }}%</span>
        </div>
        <div class="bar">
          <div class="bar-fill" :style="{ width: `${d.usagePercent}%`, background: barColor(d.usagePercent) }" />
        </div>
        <div class="row-sub">
          {{ d.used }} / {{ d.total }}
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yw-widget-body {
  height: 100%;
}

.empty {
  padding: 12px 0;
  font-size: 11px;
  color: oklch(var(--yw-muted-foreground));
  text-align: center;
}

.rows {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.row-top {
  display: flex;
  gap: 8px;
  align-items: baseline;
  justify-content: space-between;
}

.mount {
  overflow: hidden;
  max-width: 100px;
  font-size: 11px;
  font-weight: 500;
  color: oklch(var(--yw-foreground));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pct {
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 600;
  color: oklch(var(--yw-foreground));
}

.bar {
  overflow: hidden;
  height: 4px;
  margin-top: 3px;
  background: var(--yw-separator, rgb(128 128 128 / 25%));
  border-radius: 2px;
}

.bar-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 0.4s ease;
}

.row-sub {
  margin-top: 2px;
  font-size: 10px;
  color: oklch(var(--yw-muted-foreground));
}
</style>
