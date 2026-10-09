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

const LINE_RE = /^(\d{4}[-/]\d{2}[-/]\d{2}[ T]\d{2}:\d{2}:\d{2}(?:[.,]\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s+\[?(\w+)\]?\s*(.*)$/
// 常见无时间戳输出的级别行（agent/docker 输出）
const LEVEL_ONLY_RE = /^\[(\w+)\]\s*(.*)$/
// 仅时间戳行（docker logs ISO-Z / nginx 斜杠日期等，无级别 token）
const TS_ONLY_RE = /^(\d{4}[-/]\d{2}[-/]\d{2}[ T]\d{2}:\d{2}:\d{2}(?:[.,]\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s+(.*)$/

const lines = computed<LogLine[]>(() => {
  const rawLines = (props.logs || '').split('\n')
  if (rawLines.length > props.maxLines) {
    rawLines.splice(0, rawLines.length - props.maxLines)
  }
  const out: LogLine[] = []
  let lastLevel = ''
  for (const raw of rawLines) {
    // 纯空行（含日志末尾换行产生的尾元素）不渲染，避免出现"只有级别"的空行
    if (raw.trim() === '') {
      continue
    }
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
    const m3 = raw.match(TS_ONLY_RE)
    if (m3) {
      out.push({ raw, time: m3[1].replace('T', ' '), level: lastLevel, content: m3[2] || '' })
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
      <YdSelect
        v-model="levelFilter"
        :options="[{ value: 'all', label: $t('components.ydLogViewer.allLevels') }, ...levelsInLog.map(lv => ({ value: lv, label: LEVEL_META[lv]?.label || lv }))]"
        size="sm"
      />
      <FaInput v-model="keyword" :placeholder="$t('components.ydLogViewer.filterPlaceholder')" class="h-7 w-40! text-xs" />
      <label class="ml-auto flex cursor-pointer items-center gap-1 text-muted-foreground">
        <input v-model="autoScrollRef" type="checkbox" class="accent-[rgb(var(--primary))]">
        {{ $t('components.ydLogViewer.autoScroll') }}
      </label>
      <span class="text-muted-foreground">{{ $t('components.ydLogViewer.lines', { n: filtered.length }) }}</span>
    </div>
    <!-- 日志区 -->
    <div ref="container" class="min-w-0 overflow-auto px-2 py-1.5 font-mono text-xs leading-5" :style="{ height }">
      <div v-if="loading && !filtered.length" class="py-6 text-center text-muted-foreground">
        {{ $t('components.ydLogViewer.loadingLogs') }}
      </div>
      <div v-else-if="!filtered.length" class="py-6 text-center text-muted-foreground">
        {{ $t('components.ydLogViewer.noLogs') }}
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
