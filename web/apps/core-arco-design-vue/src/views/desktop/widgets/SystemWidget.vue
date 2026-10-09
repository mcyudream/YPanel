<script setup lang="ts">
// 系统监视小组件（medium）：CPU/内存环形 + 负载 + 运行时长，真数据 10s 轮询。
// 卡片外壳由 YwWidgetHost 提供，本组件只负责内容（对齐库内置组件的视觉约定）。
import apiSystem from '@/api/modules/system'
import type { SystemOverview } from '@/api/modules/system'
import { i18n } from '@/locales'
import { computed, onBeforeUnmount, onMounted, reactive } from 'vue'

const state = reactive({
  cpu: 0,
  mem: 0,
  load1: 0,
  load5: 0,
  load15: 0,
  uptime: '',
  hostname: '',
  offline: false,
})

let timer: ReturnType<typeof setInterval> | null = null

function fmtUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  return d > 0 ? i18n.global.t('desktop.overview.uptimeDHM', { d, h }) : i18n.global.t('desktop.overview.uptimeH', { h })
}

async function load() {
  try {
    const o: SystemOverview = await apiSystem.overview()
    state.cpu = Math.round(o.cpu.usagePercent)
    state.mem = Math.round(o.memory.usagePercent)
    state.load1 = o.load.load1
    state.load5 = o.load.load5
    state.load15 = o.load.load15
    state.uptime = fmtUptime(o.uptime)
    state.hostname = o.hostname
    state.offline = false
  }
  catch {
    state.offline = true
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

function gaugeColor(p: number): string {
  if (p >= 90) {
    return '#ff5257'
  }
  if (p >= 70) {
    return '#ff9f0a'
  }
  return '#62ba46'
}

// 模板不写内联对象数组：编译产物里的 `[{` 片段会被 UnoCSS 提取为未闭合候选类名，
// 生成非法 CSS 导致整个 __uno.css 500（应用全裸奔）
const gaugeItems = computed(() => [{ label: 'CPU', v: state.cpu }, { label: i18n.global.t('desktop.overview.memory'), v: state.mem }])
</script>

<template>
  <div class="yw-widget-body">
    <div class="yw-widget-head">
      <i class="yw-widget-head-icon i-lucide-activity" />
      <span class="yw-widget-head-title">{{ $t('desktop.widgets.yp-system') }}</span>
      <span class="yw-widget-dot" :style="{ background: state.offline ? '#ff5257' : '#62ba46' }" />
    </div>

    <div class="gauges">
      <div class="gauge-row">
        <div v-for="item in gaugeItems" :key="item.label" class="gauge-item">
          <svg viewBox="0 0 44 44" class="gauge-svg">
            <circle class="gauge-track" cx="22" cy="22" r="19" />
            <circle
              class="gauge-fill" cx="22" cy="22" r="19"
              :style="{ stroke: gaugeColor(item.v), strokeDashoffset: String(119.4 * (1 - item.v / 100)) }"
            />
          </svg>
          <div class="gauge-text">
            <span class="gauge-value">{{ item.v }}%</span>
            <span class="gauge-label">{{ item.label }}</span>
          </div>
        </div>
      </div>

      <div class="meta">
        <div class="meta-row">
          <span class="meta-label">{{ $t('desktop.overview.load') }}</span>
          <span class="meta-value">{{ state.load1.toFixed(2) }} / {{ state.load5.toFixed(2) }} / {{ state.load15.toFixed(2) }}</span>
        </div>
        <div class="meta-row">
          <span class="meta-label">{{ $t('desktop.overview.uptime') }}</span>
          <span class="meta-value">{{ state.uptime || '—' }}</span>
        </div>
        <div class="meta-row">
          <span class="meta-label">{{ $t('desktop.system.hostname') }}</span>
          <span class="meta-value">{{ state.hostname || '—' }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yw-widget-body {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.gauges {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 8px;
  align-items: center;
}

.gauge-row {
  display: flex;
  gap: 14px;
  align-items: center;
  justify-content: center;
}

.gauge-item {
  position: relative;
  flex-shrink: 0;
  width: 52px;
  height: 52px;
}

.gauge-svg {
  width: 100%;
  height: 100%;
}

/* SVG 表现属性一律走 scoped CSS 类，禁止写成静态属性：attributify 预设会把含空格/括号的
   静态属性值提取成非法选择器（postcss Unclosed bracket，__uno.css 整体 500），见 exp */
.gauge-track {
  fill: none;
  /* 随主题的前景色低透明度：暗色卡上是淡白细环，浅色卡上是淡灰细环，避免亮白「白框」观感 */
  stroke: color-mix(in oklab, oklch(var(--yw-foreground)) 14%, transparent);
  stroke-width: 3;
}

.gauge-fill {
  fill: none;
  stroke-width: 3;
  stroke-linecap: round;
  stroke-dasharray: 119.4;
  transform: rotate(-90deg);
  transform-origin: center;
}

.gauge-text {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  gap: 0;
  align-items: center;
  justify-content: center;
}

.gauge-value {
  font-size: 11px;
  font-weight: 700;
  color: oklch(var(--yw-foreground));
}

.gauge-label {
  font-size: 8px;
  color: oklch(var(--yw-muted-foreground));
}

.meta {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.meta-row {
  display: flex;
  gap: 8px;
  align-items: baseline;
  justify-content: space-between;
}

.meta-label {
  flex-shrink: 0;
  font-size: 10px;
  color: oklch(var(--yw-muted-foreground));
}

.meta-value {
  overflow: hidden;
  font-size: 10px;
  font-weight: 500;
  color: oklch(var(--yw-foreground));
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
