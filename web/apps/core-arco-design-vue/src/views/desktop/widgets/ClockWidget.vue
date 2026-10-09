<script setup lang="ts">
// 时钟小组件（small）：本地时间大字 + 日期星期，每秒走针。
import { i18n, tr } from '@/locales'
import { onBeforeUnmount, onMounted, ref } from 'vue'

const now = ref(new Date())
let timer: ReturnType<typeof setInterval> | null = null

const time = computed(() => {
  const d = now.value
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
})
const dateText = computed(() => {
  const d = now.value
  const week = tr(`desktop.clock.week${d.getDay()}`)
  return i18n.global.t('desktop.clock.date', { m: d.getMonth() + 1, d: d.getDate(), w: week })
})

onMounted(() => {
  timer = setInterval(() => { now.value = new Date() }, 1000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<template>
  <div class="clock-body">
    <div class="clock-time">
      {{ time }}
    </div>
    <div class="clock-date">
      {{ dateText }}
    </div>
  </div>
</template>

<style scoped>
.clock-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  height: 100%;
  justify-content: center;
}

.clock-time {
  font-size: 26px;
  font-weight: 700;
  line-height: 1.1;
  color: oklch(var(--yw-foreground));
  font-variant-numeric: tabular-nums;
}

.clock-date {
  font-size: 11px;
  color: oklch(var(--yw-muted-foreground));
}
</style>
