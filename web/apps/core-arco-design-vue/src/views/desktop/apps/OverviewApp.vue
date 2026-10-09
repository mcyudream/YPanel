<script setup lang="ts">
// 主机概览（webos 原生形态）：「关于本机」式卡片布局，YwCard + --yw-* token，
// 与桌面视觉同语言；经典面板的概览页（echarts 重图表）保留给经典模式。
import { YwCard } from '@yudream/yudream-webos-arco'
import apiSystem from '@/api/modules/system'
import type { SystemOverview } from '@/api/modules/system'
import { i18n } from '@/locales'
import { onBeforeUnmount, onMounted, ref, computed } from 'vue'

const info = ref<SystemOverview | null>(null)
const errorMsg = ref('')
let timer: ReturnType<typeof setInterval> | null = null

function fmtBytes(n: number): string {
  if (n >= 1024 ** 4) {
    return `${(n / 1024 ** 4).toFixed(2)} TB`
  }
  if (n >= 1024 ** 3) {
    return `${(n / 1024 ** 3).toFixed(1)} GB`
  }
  if (n >= 1024 ** 2) {
    return `${(n / 1024 ** 2).toFixed(0)} MB`
  }
  return `${(n / 1024).toFixed(0)} KB`
}

function fmtUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) {
    return i18n.global.t('desktop.overview.uptimeDHM', { d, h })
  }
  if (h > 0) {
    return i18n.global.t('desktop.overview.uptimeHM', { h, m })
  }
  return i18n.global.t('desktop.overview.uptimeM', { m })
}

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
    info.value = await apiSystem.overview()
    errorMsg.value = ''
  }
  catch (e: any) {
    errorMsg.value = e?.message || i18n.global.t('desktop.overview.collectFailed')
  }
}

onMounted(() => {
  void load()
  timer = setInterval(() => void load(), 10000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})

function usageColor(p: number): string {
  if (p >= 90) {
    return '#ff5257'
  }
  if (p >= 70) {
    return '#ff9f0a'
  }
  return 'oklch(var(--yw-primary))'
}

// 模板不写内联对象数组：编译产物里的 `[{` 片段会被 UnoCSS 提取为未闭合候选类名（见 exp）
const loadRows = computed(() => info.value
  ? [
      { k: i18n.global.t('desktop.overview.load1m'), v: info.value.load.load1 },
      { k: i18n.global.t('desktop.overview.load5m'), v: info.value.load.load5 },
      { k: i18n.global.t('desktop.overview.load15m'), v: info.value.load.load15 },
    ]
  : [])
</script>

<template>
  <div class="yp-overview">
    <div v-if="errorMsg" class="yp-overview-error">
      {{ errorMsg }}
    </div>

    <template v-if="info">
      <!-- 主卡：关于本机 -->
      <YwCard class="about">
        <div class="about-main">
          <div class="about-badge">
            <i class="i-lucide-server" />
          </div>
          <div class="about-text">
            <div class="about-hostname">
              {{ info.hostname }}
            </div>
            <div class="about-os">
              {{ info.os }} · {{ info.arch }}
            </div>
          </div>
          <div class="about-uptime">
            <span class="about-uptime-value">{{ fmtUptime(info.uptime) }}</span>
            <span class="about-uptime-label">{{ $t('desktop.overview.uptime') }}</span>
          </div>
        </div>
        <div class="about-grid">
          <div class="kv">
            <span class="k">{{ $t('desktop.overview.kernel') }}</span><span class="v">{{ info.kernelVersion }}</span>
          </div>
          <div class="kv">
            <span class="k">CPU</span><span class="v">{{ info.cpu.modelName }} · {{ $t('desktop.overview.cores', { n: info.cpu.logicalCount }) }}</span>
          </div>
          <div class="kv">
            <span class="k">{{ $t('desktop.overview.sysTime') }}</span><span class="v">{{ new Date(info.collectedAt).toLocaleString() }}</span>
          </div>
          <div class="kv">
            <span class="k">{{ $t('desktop.overview.netTotal') }}</span>
            <span class="v">↓ {{ fmtBytes(info.network.rxTotal) }} · ↑ {{ fmtBytes(info.network.txTotal) }}</span>
          </div>
        </div>
      </YwCard>

      <!-- 仪表行：CPU / 内存 / 负载 -->
      <div class="gauges">
        <YwCard title="CPU">
          <div class="gauge-wrap">
            <div class="gauge-item">
              <svg viewBox="0 0 44 44" class="gauge-ring">
                <circle class="ring-track" cx="22" cy="22" r="19" />
                <circle
                  class="ring-fill" cx="22" cy="22" r="19"
                  :style="{ stroke: usageColor(info.cpu.usagePercent), strokeDashoffset: String(119.4 * (1 - info.cpu.usagePercent / 100)) }"
                />
              </svg>
              <div class="gauge-center">
                <b>{{ info.cpu.usagePercent.toFixed(1) }}%</b>
              </div>
            </div>
            <div class="gauge-note">
              {{ info.cpu.modelName || '—' }}
            </div>
          </div>
        </YwCard>

        <YwCard :title="$t('desktop.overview.memory')">
          <div class="gauge-wrap">
            <div class="gauge-item">
              <svg viewBox="0 0 44 44" class="gauge-ring">
                <circle class="ring-track" cx="22" cy="22" r="19" />
                <circle
                  class="ring-fill" cx="22" cy="22" r="19"
                  :style="{ stroke: usageColor(info.memory.usagePercent), strokeDashoffset: String(119.4 * (1 - info.memory.usagePercent / 100)) }"
                />
              </svg>
              <div class="gauge-center">
                <b>{{ info.memory.usagePercent.toFixed(1) }}%</b>
              </div>
            </div>
            <div class="gauge-note">
              {{ fmtBytes(info.memory.used) }} / {{ fmtBytes(info.memory.total) }}（Swap {{ fmtBytes(info.swap.used) }}）
            </div>
          </div>
        </YwCard>

        <YwCard :title="$t('desktop.overview.load')">
          <div class="load-rows">
            <div v-for="l in loadRows" :key="l.k" class="load-row">
              <span class="load-k">{{ l.k }}</span>
              <span class="load-v">{{ l.v.toFixed(2) }}</span>
            </div>
            <div class="load-row">
              <span class="load-k">{{ $t('desktop.overview.realtimeNet') }}</span>
              <span class="load-v">↓ {{ fmtSpeed(info.network.rxSpeedBps) }} · ↑ {{ fmtSpeed(info.network.txSpeedBps) }}</span>
            </div>
          </div>
        </YwCard>
      </div>

      <!-- 磁盘 -->
      <YwCard :title="$t('desktop.overview.disk')">
        <div class="disks">
          <div v-for="d in info.disks" :key="d.mountpoint" class="disk-row">
            <span class="disk-mount" :title="`${d.fsType} · ${d.mountpoint}`">{{ d.mountpoint }}</span>
            <div class="disk-bar">
              <div class="disk-fill" :style="{ width: `${Math.min(100, d.usagePercent)}%`, background: usageColor(d.usagePercent) }" />
            </div>
            <span class="disk-text">{{ fmtBytes(d.used) }} / {{ fmtBytes(d.total) }} · {{ d.usagePercent.toFixed(0) }}%</span>
          </div>
        </div>
      </YwCard>
    </template>
  </div>
</template>

<style scoped>
.yp-overview {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
}

.yp-overview-error {
  padding: 10px 14px;
  font-size: 13px;
  color: #ff5257;
  background: rgb(255 82 87 / 10%);
  border-radius: 10px;
}

.about-main {
  display: flex;
  gap: 14px;
  align-items: center;
}

.about-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 46px;
  font-size: 22px;
  color: #fff;
  background: linear-gradient(135deg, oklch(var(--yw-primary)), oklch(var(--yw-primary) / 0.7));
  border-radius: 12px;
}

.about-text {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.about-hostname {
  overflow: hidden;
  font-size: 20px;
  font-weight: 700;
  color: oklch(var(--yw-foreground));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.about-os {
  font-size: 12px;
  color: oklch(var(--yw-muted-foreground));
}

.about-uptime {
  display: flex;
  flex-shrink: 0;
  flex-direction: column;
  gap: 2px;
  align-items: flex-end;
}

.about-uptime-value {
  font-size: 15px;
  font-weight: 600;
  color: oklch(var(--yw-foreground));
}

.about-uptime-label {
  font-size: 11px;
  color: oklch(var(--yw-muted-foreground));
}

.about-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 24px;
  margin-top: 14px;
}

.kv {
  display: flex;
  gap: 10px;
  align-items: baseline;
  min-width: 0;
}

.k {
  flex-shrink: 0;
  font-size: 12px;
  color: oklch(var(--yw-muted-foreground));
}

.v {
  overflow: hidden;
  font-size: 12px;
  font-weight: 500;
  color: oklch(var(--yw-foreground));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gauges {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.gauge-wrap {
  display: flex;
  gap: 16px;
  align-items: center;
}

.gauge-item {
  position: relative;
  flex-shrink: 0;
  width: 76px;
  height: 76px;
}

.gauge-ring {
  width: 100%;
  height: 100%;
}

/* SVG 表现属性走 scoped CSS 类：attributify 会把含空格/括号的静态属性值提取成非法选择器（见 exp） */
.ring-track {
  fill: none;
  stroke: color-mix(in oklab, oklch(var(--yw-foreground)) 14%, transparent);
  stroke-width: 3;
}

.ring-fill {
  fill: none;
  stroke-width: 3;
  stroke-linecap: round;
  stroke-dasharray: 119.4;
  transform: rotate(-90deg);
  transform-origin: center;
}

.gauge-center {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.gauge-center b {
  font-size: 13px;
  color: oklch(var(--yw-foreground));
  font-variant-numeric: tabular-nums;
}

.gauge-note {
  font-size: 12px;
  line-height: 1.5;
  color: oklch(var(--yw-muted-foreground));
}

.load-rows {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.load-row {
  display: flex;
  gap: 12px;
  align-items: baseline;
  justify-content: space-between;
}

.load-k {
  font-size: 12px;
  color: oklch(var(--yw-muted-foreground));
}

.load-v {
  font-size: 12px;
  font-weight: 600;
  color: oklch(var(--yw-foreground));
  font-variant-numeric: tabular-nums;
}

.disks {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.disk-row {
  display: grid;
  grid-template-columns: 140px 1fr auto;
  gap: 12px;
  align-items: center;
}

.disk-mount {
  overflow: hidden;
  font-size: 12px;
  color: oklch(var(--yw-foreground));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.disk-bar {
  overflow: hidden;
  height: 6px;
  background: var(--yw-separator, rgb(128 128 128 / 25%));
  border-radius: 3px;
}

.disk-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.4s ease;
}

.disk-text {
  flex-shrink: 0;
  font-size: 11px;
  color: oklch(var(--yw-muted-foreground));
  font-variant-numeric: tabular-nums;
}
</style>
