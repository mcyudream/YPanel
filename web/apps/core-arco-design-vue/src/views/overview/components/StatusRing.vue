<script setup lang="ts">
// M44 状态环：负载 / CPU / 内存 / 磁盘四联仪表（内联 SVG，零 echarts 依赖）。
const props = withDefaults(defineProps<{
  percent: number
  label: string
  sub?: string
  size?: number
}>(), {
  sub: '',
  size: 120,
})

const emit = defineEmits<{
  click: []
}>()

const R = 52
const C = 2 * Math.PI * R

const display = computed(() => Math.min(100, Math.max(0, props.percent)))
const offset = computed(() => C * (1 - display.value / 100))
const toneClass = computed(() => {
  if (display.value >= 90) {
    return 'text-red-500'
  }
  if (display.value >= 75) {
    return 'text-amber-500'
  }
  return 'text-emerald-500'
})
</script>

<template>
  <button
    type="button"
    class="group flex cursor-pointer flex-col items-center justify-center rounded-xl border bg-background p-4 transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
    @click="emit('click')"
  >
    <div class="relative" :style="{ width: `${size}px`, height: `${size}px` }">
      <svg viewBox="0 0 120 120" class="size-full -rotate-90" :class="toneClass">
        <circle cx="60" cy="60" :r="R" fill="none" stroke-width="10" class="ring-track" />
        <circle
          cx="60" cy="60" :r="R" fill="none" stroke-width="10" stroke-linecap="round"
          class="ring-bar"
          :stroke-dasharray="C"
          :stroke-dashoffset="offset"
        />
      </svg>
      <div class="absolute inset-0 flex items-center justify-center">
        <span class="text-xl font-bold tabular-nums">{{ display.toFixed(1) }}</span>
        <span class="ml-0.5 text-xs font-normal text-muted-foreground">%</span>
      </div>
    </div>
    <div class="mt-2.5 text-sm font-medium">
      {{ label }}
    </div>
    <div v-if="sub" class="mt-0.5 max-w-full truncate text-xs text-muted-foreground" :title="sub">
      {{ sub }}
    </div>
  </button>
</template>

<style scoped>
.ring-track {
  stroke: currentColor;
  opacity: 0.15;
}

.ring-bar {
  stroke: currentColor;
  transition: stroke-dashoffset 0.7s ease;
}
</style>
