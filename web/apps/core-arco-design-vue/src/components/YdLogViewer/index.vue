<script setup lang="ts">
// YdLogViewer 统一日志查看器：时间/级别提取到最左两栏、级别颜色区分、级别/关键字筛选、自动滚动。
// 行格式约定：`2006-01-02 15:04:05 [LEVEL] 消息`（TaskService / agent 输出均兼容）。
import { computed, ref, watch, nextTick } from 'vue'

defineOptions({
  name: 'YdLogViewer',
})

const props = withDefaults(defineProps<{
  logs?: string
  height?: string
  loading?: boolean
  autoScroll?: boolean
  maxLines?: number
}>(), {
  logs: '',
  height: '360px',
  loading: false,
  autoScroll: true,
  maxLines: 5000,
})

interface LogLine {
  raw: string
  time: string
  level: string
  content: string
}

const LEVEL_META: Record<string, { label: string, cls: string }> = {
  ERROR: { label: 'ERROR', cls: 'text-red-500' },
  FATAL: { label: 'FATAL', cls: 'text-red-500' },
  WARN: { label: 'WARN', cls: 'text-orange-400' },
  WARNING: { label: 'WARN', cls: 'text-orange-400' },
  INFO: { label: 'INFO', cls: 'text-blue-400' },
  DEBUG: { label: 'DEBUG', cls: 'text-muted-foreground' },
  TRACE: { label: 'TRACE', cls: 'text-muted-foreground' },
  SUCCESS: { label: 'DONE', cls: 'text-emerald-500' },
}

const LINE_RE = /^(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:[.,]\d+)?)\s+\[?(\w+)\]?\s*(.*)$/
// 常见无时间戳输出的级别行（agent/docker 输出）
const LEVEL_ONLY_RE = /^\[(\w+)\]\s*(.*)$/

const lines = computed<LogLine[]>(() => {
  const rawLines = (props.logs || '').split('\n')
  if (rawLines.length > props.maxLines) {
    rawLines.splice(0, rawLines.length - props.maxLines)
  }
  const out: LogLine[] = []
  let lastLevel = ''
  for (const raw of rawLines) {
    const m = raw.match(LINE_RE)
    if (m) {
      const level = m[2].toUpperCase()
      lastLevel = level
      out.push({ raw, time: m[1].replace('T', ' '), level, content: m[3] || '' })
      continue
    }
    const m2 = raw.match(LEVEL_ONLY_RE)
    if (m2) {
      const level = m2[1].toUpperCase()
      lastLevel = level
      out.push({ raw, time: '', level, content: m2[2] || '' })
      continue
    }
    out.push({ raw, time: '', level: lastLevel, content: raw })
  }
  return out
})

const levelsInLog = computed(() => {
  const set = new Set<string>()
  for (const l of lines.value) {
    if (l.level && LEVEL_META[l.level]) {
      set.add(l.level)
    }
  }
  return [...set].sort()
})

const levelFilter = ref('all')
const keyword = ref('')
const autoScrollRef = ref(props.autoScroll)
const container = ref<HTMLElement | null>(null)

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return lines.value.filter((l) => {
    if (levelFilter.value !== 'all' && l.level !== levelFilter.value) {
      return false
    }
    if (kw && !l.content.toLowerCase().includes(kw) && !l.raw.toLowerCase().includes(kw)) {
      return false
    }
    return true
  })
})

function levelClass(level: string) {
  return LEVEL_META[level]?.cls || 'text-foreground'
}

function levelLabel(level: string) {
  return LEVEL_META[level]?.label || ''
}

watch(filtered, () => {
  if (!autoScrollRef.value) {
    return
  }
  void nextTick(() => {
    const el = container.value
    if (el) {
      el.scrollTop = el.scrollHeight
    }
  })
}, { immediate: true })

defineExpose({ lines: filtered })
</script>

<template>
  <div class="flex min-w-0 flex-col overflow-hidden rounded-lg border bg-muted/30">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-2 border-b bg-background/60 px-2 py-1.5 text-xs">
      <select v-model="levelFilter" class="h-7 rounded border bg-background px-1.5 outline-none">
        <option value="all">全部级别</option>
        <option v-for="lv in levelsInLog" :key="lv" :value="lv">
          {{ LEVEL_META[lv]?.label || lv }}
        </option>
      </select>
      <FaInput v-model="keyword" placeholder="过滤关键字…" class="h-7 w-40! text-xs" />
      <label class="ml-auto flex cursor-pointer items-center gap-1 text-muted-foreground">
        <input v-model="autoScrollRef" type="checkbox" class="accent-[rgb(var(--primary))]">
        自动滚动
      </label>
      <span class="text-muted-foreground">{{ filtered.length }} 行</span>
    </div>
    <!-- 日志区 -->
    <div ref="container" class="min-w-0 flex-1 overflow-auto px-2 py-1.5 font-mono text-xs leading-5" :style="{ height }">
      <div v-if="loading && !filtered.length" class="py-6 text-center text-muted-foreground">
        日志加载中…
      </div>
      <div v-else-if="!filtered.length" class="py-6 text-center text-muted-foreground">
        暂无日志
      </div>
      <table v-else class="w-full border-collapse">
        <tbody>
          <tr v-for="(l, i) in filtered" :key="i" class="align-top hover:bg-accent/30">
            <td class="w-36 whitespace-nowrap pr-2 text-right text-muted-foreground/80 select-all">{{ l.time }}</td>
            <td class="w-12 whitespace-nowrap pr-2 font-medium" :class="levelClass(l.level)">{{ levelLabel(l.level) }}</td>
            <td class="whitespace-pre-wrap break-all" :class="levelClass(l.level)">{{ l.content }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
