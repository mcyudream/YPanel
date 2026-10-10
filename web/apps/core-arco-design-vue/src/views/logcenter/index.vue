<script setup lang="ts">
// M33 P2.5 日志中心：集中检索单页（VictoriaLogs/LogsQL v2）。
// 对齐 Kibana Discover（字段侧栏/行详情）、Grafana Logs（直方图点击缩放）、金融日志台（历史/常用/排序/高亮）。
// 实时检索（P1）UI 已下线（后端接口保留）；历史/常用查询 SQLite 持久化。
import { computed, onMounted, ref, watch, nextTick } from 'vue'
import { centralApi, historyApi } from '@/api/modules/logcenter'
import type { CentralQueryItem, CentralStreamValue, LogSearchQueryItem, VLInstance } from '@/api/modules/logcenter'
import { storeApi } from '@/api/modules/store'
import echarts from '@/utils/echarts'
import { i18n } from '@/locales'

defineOptions({
  name: 'LogcenterIndex',
})

const LEVEL_RE = /\b(trace|debug|info|notice|warn(?:ing)?|err(?:or)?|fatal|crit(?:ical)?|panic)\b/i
const LEVEL_META: Record<string, { label: string, cls: string }> = {
  ERROR: { label: 'ERROR', cls: 'text-red-500' },
  FATAL: { label: 'FATAL', cls: 'text-red-500 font-semibold' },
  WARN: { label: 'WARN', cls: 'text-orange-400' },
  NOTICE: { label: 'NOTICE', cls: 'text-sky-400' },
  INFO: { label: 'INFO', cls: 'text-blue-400' },
  DEBUG: { label: 'DEBUG', cls: 'text-muted-foreground' },
  TRACE: { label: 'TRACE', cls: 'text-muted-foreground' },
  PANIC: { label: 'PANIC', cls: 'text-red-500 font-semibold' },
}
const FIELDS = ['container_name', 'host', 'image'] as const
type FieldName = typeof FIELDS[number]

// ---- 聚合状态（VL 自动发现，面板自动管理，无手工地址） ----
const instances = ref<VLInstance[]>([])
const statusLoading = ref(false)
const readyInstances = computed(() => instances.value.filter(i => i.online && i.found))
const readyOn = computed(() => readyInstances.value.length > 0)
const multiNode = computed(() => readyInstances.value.length > 1)

async function loadStatus() {
  statusLoading.value = true
  try {
    instances.value = (await centralApi.status()).instances || []
  }
  catch {
    instances.value = []
  }
  finally {
    statusLoading.value = false
  }
  if (readyOn.value) {
    void reloadFields() // 状态就绪后加载字段切面（首次进入与刷新共用此路径）
    void loadRetention()
  }
}

// ---- 一键接入：对未安装 VL 的在线节点，经商店安装 victoria-logs + vector（全默认参数，任务化） ----
const LOG_STACK_KEYS = ['victoria-logs', 'vector'] as const
const installingNodes = ref<string[]>([])

function nodeInstallable(i: VLInstance) {
  return i.online && !i.found
}

// 在商店应用列表中定位应用所在源：VL/vector 为 YPanel 收录版（默认参数对接面板 VL），
// 同名应用存在于多个源时优先非 1Panel 源
async function resolveSourceId(key: string): Promise<number> {
  const t = i18n.global.t
  const [sources, list] = await Promise.all([storeApi.sources(), storeApi.list({ search: key })])
  const srcTypeOf = (id: number) => sources.find(s => s.id === id)?.type || ''
  const found = (list.items || []).filter(i => i.key === key)
    .sort((a, b) => Number(srcTypeOf(a.sourceId) === 'onepanel') - Number(srcTypeOf(b.sourceId) === 'onepanel'))
  if (!found[0])
    throw new Error(t('logcenter.status.appNotFound'))
  return found[0].sourceId
}

async function installStack(nodeId: string) {
  if (installingNodes.value.includes(nodeId))
    return
  const t = i18n.global.t
  installingNodes.value = [...installingNodes.value, nodeId]
  try {
    const sourceIds = await Promise.all(LOG_STACK_KEYS.map(key => resolveSourceId(key)))
    for (let i = 0; i < LOG_STACK_KEYS.length; i++) {
      await storeApi.install({ sourceId: sourceIds[i], key: LOG_STACK_KEYS[i], version: '', name: LOG_STACK_KEYS[i], params: {}, nodeId })
    }
    useFaToast().success(t('logcenter.status.installQueued'))
    // VL 容器启动 + agent 发现均有延迟，稍后自动刷新一次状态
    setTimeout(() => { void loadStatus() }, 45_000)
  }
  catch (e: any) {
    useFaToast().error(t('logcenter.status.installFail'), { description: e?.message })
  }
  finally {
    installingNodes.value = installingNodes.value.filter(n => n !== nodeId)
  }
}

// ---- 保留策略（VL 自动清理过期数据；改 .env + 重建容器生效） ----
const retentions = ref<{ nodeId: string, nodeName: string, found: boolean, period: string, error?: string }[]>([])
const retentionInput = ref('30d')
const applyingRetention = ref(false)

async function loadRetention() {
  try {
    retentions.value = await centralApi.retention()
  }
  catch {}
}

function applyRetention() {
  const period = retentionInput.value.trim()
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('logcenter.retention.title'),
    content: i18n.global.t('logcenter.retention.confirm', { period }),
    onConfirm: async () => {
      applyingRetention.value = true
      try {
        const res = await centralApi.setRetention(period)
        if (res.failures?.length) {
          useFaToast().error(i18n.global.t('logcenter.retention.partialFailed'), { description: res.failures.join('；') })
        }
        else {
          useFaToast().success(i18n.global.t('logcenter.retention.applied', { n: res.applied.length }))
        }
        await loadRetention()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('logcenter.retention.setFailed'), { description: e?.message })
      }
      finally {
        applyingRetention.value = false
      }
    },
  })
}

// ---- 查询条件 ----
const simpleMode = ref((localStorage.getItem('ypanel.logcenter.mode') ?? 'simple') !== 'advanced')
const timeRange = ref('1h')
const customStart = ref('')
const customEnd = ref('')
const filters = ref<Record<FieldName, string[]>>({ container_name: [], host: [], image: [] })
const keyword = ref('')
const isRegex = ref(false)
// 匹配模式：word=裸词（走倒排索引，多词 AND，最快）/ phrase=短语子串 / regex=正则（最慢）
const matchMode = ref<'word' | 'phrase' | 'regex'>(((localStorage.getItem('ypanel.logcenter.match') as any) || 'word'))

watch(matchMode, (v) => {
  localStorage.setItem('ypanel.logcenter.match', v)
  isRegex.value = v === 'regex'
  syncQueryText()
})
const limit = ref(1000)
const queryText = ref('*')
const queryTouched = ref(false)

watch(simpleMode, v => localStorage.setItem('ypanel.logcenter.mode', v ? 'simple' : 'advanced'))

function rfc3339Ago(range: string) {
  const mins: Record<string, number> = { '15m': 15, '1h': 60, '6h': 360, '24h': 1440, '7d': 10080 }
  if (range === 'custom') {
    if (!customStart.value) {
      return ''
    }
    return new Date(customStart.value).toISOString()
  }
  return new Date(Date.now() - (mins[range] || 60) * 60000).toISOString()
}

function endValue() {
  if (timeRange.value === 'custom' && customEnd.value) {
    return new Date(customEnd.value).toISOString()
  }
  return ''
}

function stepFor(range: string) {
  const step: Record<string, string> = { '15m': '30s', '1h': '1m', '6h': '5m', '24h': '15m', '7d': '2h' }
  return step[range] || ''
}

// LogsQL v2：{a=~"x|y",b="z"} + 裸过滤器（短语 / ~"re"）；无条件 = *
function buildQuery() {
  const sel: string[] = []
  for (const f of FIELDS) {
    const vals = filters.value[f]
    if (!vals.length) {
      continue
    }
    if (vals.length === 1) {
      sel.push(`${f}="${vals[0].replace(/(["\\])/g, '\\$1')}"`)
    }
    else {
      sel.push(`${f}=~"${vals.map(v => v.replace(/(["\\])/g, '\\$1')).join('|')}"`)
    }
  }
  const parts: string[] = []
  if (sel.length) {
    parts.push(`{${sel.join(',')}}`)
  }
  const kw = keyword.value.trim()
  if (kw) {
    if (matchMode.value === 'regex') {
      parts.push(`~${JSON.stringify(kw)}`)
    }
    else if (matchMode.value === 'phrase') {
      parts.push(JSON.stringify(kw))
    }
    else {
      // 词匹配：裸词走 VL 倒排索引（大日志量下最快），多词空格 AND；清理不安全字符
      for (const w of kw.replace(/["\\:{}[\]~|()*]/g, ' ').split(/\s+/).filter(Boolean)) {
        parts.push(w)
      }
    }
  }
  return parts.length ? parts.join(' ') : '*'
}

function syncQueryText() {
  if (!queryTouched.value) {
    queryText.value = buildQuery()
  }
}

function toggleFilter(field: FieldName, value: string) {
  const list = filters.value[field]
  filters.value[field] = list.includes(value) ? list.filter(v => v !== value) : [...list, value]
  queryTouched.value = false
  syncQueryText()
}

function clearFilters() {
  filters.value = { container_name: [], host: [], image: [] }
  queryTouched.value = false
  syncQueryText()
}

// ---- 字段侧栏 ----
const fieldKeyword = ref('')
const fieldValues = ref<Record<string, CentralStreamValue[]>>({})
const favFields = ref<FieldName[]>(JSON.parse(localStorage.getItem('ypanel.logcenter.favFields') || '[]'))

function toggleFavField(f: FieldName) {
  favFields.value = favFields.value.includes(f) ? favFields.value.filter(x => x !== f) : [...favFields.value, f]
  localStorage.setItem('ypanel.logcenter.favFields', JSON.stringify(favFields.value))
}

const sidebarFields = computed(() => {
  const kw = fieldKeyword.value.trim().toLowerCase()
  const fields = [...FIELDS].sort((a, b) =>
    Number(favFields.value.includes(b)) - Number(favFields.value.includes(a)))
  // 字段名或任一字段值命中即显示（仅 3 个字段，纯字段名过滤意义有限）
  return fields.filter(f => !kw || f.includes(kw) || (fieldValues.value[f] || []).some(v => v.value.toLowerCase().includes(kw)))
})

// 关键字下的值过滤：字段名命中时保留全部值，否则只留匹配值
function valuesOf(f: FieldName): CentralStreamValue[] {
  const kw = fieldKeyword.value.trim().toLowerCase()
  const vals = fieldValues.value[f] || []
  if (!kw || f.includes(kw)) {
    return vals
  }
  return vals.filter(v => v.value.toLowerCase().includes(kw))
}

const activeFilterCount = computed(() => FIELDS.reduce((n, f) => n + filters.value[f].length, 0))
const hasFilterState = computed(() => activeFilterCount.value > 0 || !!keyword.value.trim())

// 一键重置：清空字段过滤 + 关键字 + 匹配模式回词匹配（保留时间范围），并立即重查
function resetFilters() {
  filters.value = { container_name: [], host: [], image: [] }
  keyword.value = ''
  matchMode.value = 'word'
  queryTouched.value = false
  syncQueryText()
  void doSearch()
}

async function reloadFields() {
  if (!readyOn.value) {
    return
  }
  const start = rfc3339Ago(timeRange.value)
  const end = endValue()
  for (const f of FIELDS) {
    try {
      fieldValues.value[f] = (await centralApi.streams({ field: f, start, end, limit: 10 })).slice(0, 10)
    }
    catch {
      fieldValues.value[f] = []
    }
  }
}

// ---- 历史 / 常用 ----
const historyList = ref<LogSearchQueryItem[]>([])
const historyModal = ref(false)

const pinnedItems = computed(() => historyList.value.filter(h => h.pinned))
const recentItems = computed(() => historyList.value.filter(h => !h.pinned).slice(0, 8))

async function refreshHistory() {
  try {
    historyList.value = await historyApi.list()
  }
  catch {}
}

function applyHistory(h: LogSearchQueryItem) {
  historyModal.value = false
  queryText.value = h.queryText || '*'
  queryTouched.value = true
  keyword.value = h.keyword
  matchMode.value = h.isRegex ? 'regex' : 'word'
  isRegex.value = h.isRegex
  timeRange.value = h.rangeType || '1h'
  if (h.rangeType === 'custom' && h.startAt) {
    customStart.value = toLocalInput(new Date(h.startAt))
    customEnd.value = h.endAt ? toLocalInput(new Date(h.endAt)) : ''
  }
  try {
    const f = JSON.parse(h.containersJson || '{}')
    filters.value = { container_name: f.container_name || [], host: f.host || [], image: f.image || [] }
  }
  catch {
    filters.value = { container_name: [], host: [], image: [] }
  }
  void doSearch()
}

function itemLabel(h: LogSearchQueryItem) {
  return h.remark || h.keyword || h.queryText || i18n.global.t('logcenter.history.emptyQuery')
}

// ---- 搜索 ----
const searching = ref(false)
const searched = ref(false)
const items = ref<CentralQueryItem[]>([])
const truncated = ref(false)
const totalHits = ref(0)
const loadingMore = ref(false)
const sortDesc = ref(true)

async function doSearch() {
  if (!readyOn.value) {
    useFaToast().warning(i18n.global.t('logcenter.search.noInstance'))
    return
  }
  if (timeRange.value === 'custom' && !customStart.value) {
    useFaToast().warning(i18n.global.t('logcenter.search.customStartRequired'))
    return
  }
  searching.value = true
  const start = rfc3339Ago(timeRange.value)
  const end = endValue()
  const q = queryText.value.trim() || '*'
  try {
    const [qres, hres] = await Promise.all([
      centralApi.query({ query: q, start, end, limit: limit.value }),
      centralApi.hits({ query: q, start, end, step: stepFor(timeRange.value) }).catch(() => null),
    ])
    items.value = qres.items || []
    truncated.value = qres.truncated
    totalHits.value = hres?.total ?? items.value.length
    searched.value = true
    expandedKey.value = ''
    void nextTick(() => renderHist(hres))
    void historyApi.record({
      queryText: q,
      containersJson: JSON.stringify(filters.value),
      keyword: keyword.value.trim(),
      isRegex: isRegex.value,
      rangeType: timeRange.value,
      startAt: timeRange.value === 'custom' && customStart.value ? new Date(customStart.value).toISOString() : null,
      endAt: timeRange.value === 'custom' && customEnd.value ? new Date(customEnd.value).toISOString() : null,
    }).then(refreshHistory).catch(() => {})
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('logcenter.search.queryFailed'), { description: e?.message })
  }
  finally {
    searching.value = false
  }
}

async function loadMore() {
  if (loadingMore.value || !items.value.length) {
    return
  }
  loadingMore.value = true
  try {
    const res = await centralApi.query({
      query: queryText.value.trim() || '*',
      start: rfc3339Ago(timeRange.value),
      end: endValue(),
      limit: limit.value,
      offset: items.value.length,
    })
    items.value = [...items.value, ...(res.items || [])]
    truncated.value = truncated.value || res.truncated
  }
  catch (e: any) {
      useFaToast().error(i18n.global.t('logcenter.search.loadFailed'), { description: e?.message })
  }
  finally {
    loadingMore.value = false
  }
}

const displayItems = computed(() => (sortDesc.value ? [...items.value].reverse() : items.value))

// ---- 直方图（hits，点击柱子缩放范围） ----
const histEl = ref<HTMLElement | null>(null)
let histChart: echarts.ECharts | null = null
const lastStep = ref('')

function toLocalInput(d: Date) {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`
}

const STEP_MS: Record<string, number> = { '30s': 30e3, '1m': 60e3, '5m': 300e3, '15m': 900e3, '2h': 7200e3 }

// canvas 不解析 CSS 变量：从主题取 --primary 通道值拼 rgba（fa 主题变量为 "r g b" 形态）
function primaryBarColor() {
  try {
    const ch = getComputedStyle(document.documentElement).getPropertyValue('--primary').trim()
    if (ch && /^\d+[\s,]+\d+[\s,]+\d+$/.test(ch.replace(/,\s*/g, ' '))) {
      return `rgba(${ch.replace(/,\s*/g, ',')}, 0.5)`
    }
  }
  catch {}
  return 'rgba(99, 102, 241, 0.5)'
}

function renderHist(hits: { timestamps: string[], values: number[] } | null) {
  if (!hits || !hits.timestamps?.length) {
    histChart?.clear()
    return
  }
  if (!histEl.value) {
    return
  }
  histChart ||= echarts.init(histEl.value)
  lastStep.value = stepFor(timeRange.value)
  histChart.setOption({
    grid: { left: 8, right: 8, top: 8, bottom: 18 },
    xAxis: {
      type: 'category',
      data: hits.timestamps.map(t => t.replace('T', ' ').slice(5, 16)),
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { show: true, fontSize: 9, color: 'rgba(128,128,128,0.9)', interval: Math.ceil(hits.timestamps.length / 8) },
    },
    yAxis: { type: 'value', splitLine: { show: false }, axisLabel: { show: false } },
    tooltip: { trigger: 'axis', textStyle: { fontSize: 11 } },
    series: [{
      type: 'bar',
      data: hits.values,
      barWidth: '70%',
      itemStyle: { color: primaryBarColor(), borderRadius: [2, 2, 0, 0] },
    }],
  }, true)
  histChart.off('click')
  histChart.on('click', (p: any) => {
    const ts = hits.timestamps[p.dataIndex]
    if (!ts) {
      return
    }
    const start = new Date(ts)
    const stepMs = STEP_MS[lastStep.value] || 60e3
    timeRange.value = 'custom'
    customStart.value = toLocalInput(start)
    customEnd.value = toLocalInput(new Date(start.getTime() + stepMs))
    queryTouched.value = false
    syncQueryText()
    void doSearch()
  })
}

watch(histEl, (el) => {
  if (el && searched.value) {
    void nextTick(() => histChart?.resize())
  }
})

// ---- 结果展示 ----
const expandedKey = ref('')

function rowKey(item: CentralQueryItem, i: number) {
  return `${item.ts}:${i}`
}

function detectLevel(msg: string) {
  const m = msg.match(LEVEL_RE)
  if (!m) {
    return ''
  }
  const l = m[1].toLowerCase()
  if (l === 'warning' || l === 'warn') {
    return 'WARN'
  }
  if (l === 'error' || l === 'err') {
    return 'ERROR'
  }
  if (l === 'fatal' || l === 'critical' || l === 'crit' || l === 'panic') {
    return 'FATAL'
  }
  return l.toUpperCase()
}

function levelMeta(msg: string) {
  const lv = detectLevel(msg)
  return { meta: (lv && LEVEL_META[lv]) || { label: 'LOG', cls: 'text-muted-foreground' }, lv }
}

function containerOf(item: CentralQueryItem) {
  const m = item.stream.match(/container_name="([^"]*)"/)
  return m ? m[1] : (item.stream.replace(/[{}"]/g, '') || '-')
}

function fmtTime(ts: string) {
  return ts.replace('T', ' ').replace(/Z$/, '').replace(/\.\d{3,}$/, s => s.slice(0, 4))
}

// 关键字高亮分段（不使用 v-html；词模式高亮任一词的命中）
interface Seg { t: string, hit: boolean }

function segments(msg: string): Seg[] {
  const kw = keyword.value.trim()
  if (!kw || !searched.value) {
    return [{ t: msg, hit: false }]
  }
  const out: Seg[] = []
  if (matchMode.value === 'regex') {
    try {
      const re = new RegExp(kw, 'gi')
      let last = 0
      for (const m of msg.matchAll(re)) {
        const idx = m.index ?? 0
        if (idx > last) {
          out.push({ t: msg.slice(last, idx), hit: false })
        }
        out.push({ t: m[0] || '', hit: true })
        last = idx + (m[0]?.length || 0)
        if (!m[0]) {
          break
        }
      }
      out.push({ t: msg.slice(last), hit: false })
      return out
    }
    catch {
      return [{ t: msg, hit: false }]
    }
  }
  const lower = msg.toLowerCase()
  const hits: { s: number, e: number }[] = []
  if (matchMode.value === 'phrase') {
    const k = kw.toLowerCase()
    let i = 0
    while ((i = lower.indexOf(k, i)) >= 0) {
      hits.push({ s: i, e: i + kw.length })
      i += kw.length
    }
  }
  else {
    for (const w of kw.replace(/["\\:{}[\]~|()*]/g, ' ').split(/\s+/).filter(Boolean)) {
      const lw = w.toLowerCase()
      let i = 0
      while ((i = lower.indexOf(lw, i)) >= 0) {
        hits.push({ s: i, e: i + lw.length })
        i += lw.length
      }
    }
  }
  hits.sort((a, b) => a.s - b.s || a.e - b.e)
  let last = 0
  for (const h of hits) {
    if (h.s < last) {
      continue
    }
    if (h.s > last) {
      out.push({ t: msg.slice(last, h.s), hit: false })
    }
    out.push({ t: msg.slice(h.s, h.e), hit: true })
    last = h.e
  }
  out.push({ t: msg.slice(last), hit: false })
  return out
}

function streamKvs(stream: string): [string, string][] {
  const out: [string, string][] = []
  for (const m of stream.matchAll(/(\w+)="([^"]*)"/g)) {
    out.push([m[1], m[2]])
  }
  return out
}

function msgJson(msg: string): Record<string, any> | null {
  const t = msg.trim()
  if (!(t.startsWith('{') && t.endsWith('}'))) {
    return null
  }
  try {
    const v = JSON.parse(t)
    return typeof v === 'object' && v !== null ? v : null
  }
  catch {
    return null
  }
}

function copyText(text: string, tip = i18n.global.t('common.copied')) {
  void navigator.clipboard.writeText(text).then(() => {
    useFaToast().success(tip)
  })
}

function exportTxt() {
  const list = sortDesc.value ? [...items.value].reverse() : items.value
  const lines = list.map(i => `[${i.ts}] [${detectLevel(i.msg) || 'LOG'}] [${containerOf(i)}] ${i.msg}`)
  const blob = new Blob([lines.join('\n')], { type: 'text/plain;charset=utf-8' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `central-logs-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.txt`
  a.click()
  URL.revokeObjectURL(a.href)
}

// FaDropdown 快捷菜单：常用在前、历史在后（各取 8 条）
const quickMenus = computed(() => [[
  ...pinnedItems.value.slice(0, 8).map(h => ({ label: `★ ${itemLabel(h)}`, handle: () => applyHistory(h) })),
  ...recentItems.value.map(h => ({ label: itemLabel(h), handle: () => applyHistory(h) })),
]])

onMounted(async () => {
  await loadStatus()
  syncQueryText()
  await refreshHistory()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="scroll-text" :size="24" />
          <span>{{ $t('logcenter.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('logcenter.description') }}</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <!-- 未发现 VL：接入引导 -->
      <div v-if="!readyOn" class="mx-auto max-w-xl rounded-lg border bg-muted/30 p-5 text-sm">
        <div class="mb-2 flex items-center gap-2 font-medium">
          {{ $t('logcenter.status.notConnected') }}
          <FaButton size="sm" variant="outline" :loading="statusLoading" @click="loadStatus().then(() => { void reloadFields(); void refreshHistory() })">{{ $t('logcenter.status.recheck') }}</FaButton>
        </div>
        <ol class="list-decimal space-y-1 pl-5 text-muted-foreground">
          <li>{{ $t('logcenter.status.guide1a') }}<b class="text-foreground">VictoriaLogs</b>{{ $t('logcenter.status.guide1b') }}</li>
          <li>{{ $t('logcenter.status.guide2a') }}<b class="text-foreground">{{ $t('logcenter.status.guide2Name') }}</b>{{ $t('logcenter.status.guide2b') }}</li>
          <li>{{ $t('logcenter.status.guide3') }}</li>
        </ol>
        <div v-if="instances.length" class="mt-3 flex flex-wrap items-center gap-1.5 text-xs">
          <template v-for="i in instances" :key="i.nodeId">
            <span
              class="rounded-full px-2 py-0.5"
              :class="i.online && i.found ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
              :title="i.error || (i.container ? `${i.container}:${i.port}` : '')"
            >{{ i.nodeName }} · {{ i.online ? (i.found ? $t('logcenter.status.found') : (i.error || $t('logcenter.status.notInstalled'))) : $t('logcenter.status.offline') }}</span>
            <FaButton
              v-if="nodeInstallable(i)"
              variant="outline"
              size="sm"
              class="h-5 rounded-full px-2 text-xs"
              :loading="installingNodes.includes(i.nodeId)"
              @click="installStack(i.nodeId)"
            >{{ $t('logcenter.status.install') }}</FaButton>
          </template>
        </div>
      </div>

      <div v-else class="flex flex-col gap-3">
        <!-- 聚合状态条 -->
        <div class="flex flex-wrap items-center gap-1.5 text-xs">
          <span class="text-muted-foreground">{{ $t('logcenter.status.nodes') }}</span>
          <span
            v-for="i in instances" :key="i.nodeId"
            class="rounded-full px-2 py-0.5"
            :class="i.online && i.found ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
            :title="i.error || (i.container ? `${i.container}:${i.port}` : '')"
          >{{ i.nodeName }} · {{ i.online ? (i.found ? (readyInstances.length > 1 ? $t('logcenter.status.aggregating') : $t('logcenter.status.connected')) : (i.error || $t('logcenter.status.notInstalled'))) : $t('logcenter.status.offline') }}</span>
          <FaButton size="sm" variant="ghost" class="text-muted-foreground" :loading="statusLoading" @click="loadStatus().then(() => { void reloadFields() })">{{ $t('common.refresh') }}</FaButton>
          <!-- 保留策略 -->
          <span class="ml-2 text-muted-foreground">{{ $t('logcenter.retention.label') }}</span>
          <span
            v-for="rt in retentions" :key="rt.nodeId"
            class="rounded-full px-2 py-0.5"
            :class="rt.found ? 'bg-muted text-foreground' : 'bg-muted text-muted-foreground'"
            :title="rt.error || ''"
          >{{ rt.nodeName }} · {{ rt.period || (rt.error || '—') }}</span>
          <input
            v-model="retentionInput" placeholder="30d"
            class="h-7 w-20 rounded-md border border-input bg-background px-2 text-xs outline-none" :title="$t('logcenter.retention.inputTitle')"
          >
          <FaButton size="sm" variant="outline" :loading="applyingRetention" @click="applyRetention">{{ $t('common.apply') }}</FaButton>
        </div>

        <!-- 两栏：字段侧栏 + 检索区 -->
        <div class="flex min-w-0 items-start gap-3">
          <!-- 字段侧栏 -->
          <aside class="hidden w-52 shrink-0 flex-col gap-2 self-stretch rounded-lg border bg-muted/30 p-2.5 md:flex">
            <div class="flex items-center justify-between text-xs font-medium">
              <span>{{ $t('logcenter.fields.label') }}</span>
              <span v-if="activeFilterCount" class="cursor-pointer text-primary underline-offset-2 hover:underline" @click="clearFilters">{{ $t('logcenter.filters.clear', { n: activeFilterCount }) }}</span>
            </div>
            <FaInput v-model="fieldKeyword" :placeholder="$t('logcenter.fields.searchPlaceholder')" class="h-7 w-full! min-w-0! text-xs!" />
            <div v-for="f in sidebarFields" :key="f" class="flex flex-col gap-0.5">
              <div class="flex items-center gap-1 px-0.5 pt-1 text-xs">
                <span
                  class="cursor-pointer text-xs leading-none" :class="favFields.includes(f) ? 'text-orange-400' : 'text-muted-foreground/40 hover:text-muted-foreground'"
                  :title="favFields.includes(f) ? $t('logcenter.fields.unfollow') : $t('logcenter.fields.follow')"
                  @click="toggleFavField(f)"
                >★</span>
                <span class="font-mono text-muted-foreground">{{ f }}</span>
              </div>
              <button
                v-for="v in valuesOf(f)" :key="v.value"
                class="flex items-center justify-between gap-1 rounded px-1.5 py-1 text-left text-xs transition-colors hover:bg-accent/40"
                :class="filters[f].includes(v.value) ? 'bg-primary/10 text-primary' : ''"
                :title="$t('logcenter.filters.hitToggle', { n: v.hits.toLocaleString(), action: filters[f].includes(v.value) ? $t('logcenter.filters.removeFrom') : $t('logcenter.filters.addTo') })"
                @click="toggleFilter(f, v.value)"
              >
                <span class="min-w-0 truncate font-mono">{{ v.value }}</span>
                <span class="shrink-0 text-[10px] text-muted-foreground/70">{{ v.hits > 999 ? `${(v.hits / 1000).toFixed(1)}k` : v.hits }}</span>
              </button>
              <div v-if="!(fieldValues[f] || []).length" class="px-1.5 py-0.5 text-[10px] text-muted-foreground/60">{{ $t('logcenter.fields.noData') }}</div>
            </div>
          </aside>

          <!-- 检索区 -->
          <div class="flex min-w-0 flex-1 flex-col gap-2.5" :class="{ 'pointer-events-none opacity-50': !readyOn }">
            <!-- 工具条 1：模式 + 历史/常用 -->
            <div class="flex flex-wrap items-center gap-2 text-xs">
              <div class="flex overflow-hidden rounded-md border text-xs">
                <button
                  class="px-2.5 py-1 transition-colors" :class="simpleMode ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-accent/40'"
                  @click="simpleMode = true"
                >{{ $t('logcenter.toolbar.simple') }}</button>
                <button
                  class="px-2.5 py-1 transition-colors" :class="!simpleMode ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-accent/40'"
                  @click="simpleMode = false"
                >{{ $t('logcenter.toolbar.advanced') }}</button>
              </div>
              <FaDropdown :items="quickMenus">
                <FaButton size="sm" variant="outline">
                  {{ $t('logcenter.toolbar.historyQuick') }} <span class="text-muted-foreground">▾</span>
                </FaButton>
              </FaDropdown>
              <FaButton size="sm" variant="ghost" class="text-muted-foreground" @click="historyModal = true">{{ $t('logcenter.toolbar.manage') }}</FaButton>
              <span class="ml-auto text-[10px] text-muted-foreground/60">LogsQL v2 · VictoriaLogs</span>
            </div>

            <!-- 工具条 2：时间 + 上限 + 关键字 + 查询 -->
            <div class="flex flex-wrap items-center gap-2">
              <YdSelect
                v-model="timeRange"
                :options="[
                  { value: '15m', label: $t('logcenter.time.15m') },
                  { value: '1h', label: $t('logcenter.time.1h') },
                  { value: '6h', label: $t('logcenter.time.6h') },
                  { value: '24h', label: $t('logcenter.time.24h') },
                  { value: '7d', label: $t('logcenter.time.7d') },
                  { value: 'custom', label: $t('logcenter.time.custom') },
                ]"
                size="sm"
                @update:model-value="() => { queryTouched = false; syncQueryText(); void reloadFields() }"
              />
              <template v-if="timeRange === 'custom'">
                <input v-model="customStart" type="datetime-local" class="h-8 rounded-md border border-input bg-background px-2 text-xs outline-none">
                <span class="text-xs text-muted-foreground">{{ $t('logcenter.toolbar.to') }}</span>
                <input v-model="customEnd" type="datetime-local" class="h-8 rounded-md border border-input bg-background px-2 text-xs outline-none">
              </template>
              <YdSelect
                v-model="limit"
                :options="[
                  { value: 100, label: $t('logcenter.limit.l100') },
                  { value: 500, label: $t('logcenter.limit.l500') },
                  { value: 1000, label: $t('logcenter.limit.l1000') },
                  { value: 5000, label: $t('logcenter.limit.l5000') },
                  { value: 20000, label: $t('logcenter.limit.l20000') },
                ]"
                size="sm"
              />
              <FaInput
                v-model="keyword" :placeholder="$t('logcenter.toolbar.keywordPlaceholder')" class="h-8 w-48! text-xs"
                @update:model-value="() => { queryTouched = false; syncQueryText() }" @keyup.enter="doSearch"
              />
              <YdSelect
                v-model="matchMode"
                :options="[
                  { value: 'word', label: $t('logcenter.match.word') },
                  { value: 'phrase', label: $t('logcenter.match.phrase') },
                  { value: 'regex', label: $t('logcenter.match.regex') },
                ]"
                size="sm"
                :title="$t('logcenter.toolbar.matchTitle')"
              />
              <FaButton v-if="hasFilterState" size="sm" variant="outline" class="text-muted-foreground!" @click="resetFilters">
                <FaIcon name="i-lucide:rotate-ccw" class="mr-1" /> {{ $t('common.reset') }}
              </FaButton>
              <FaButton size="sm" :loading="searching" @click="doSearch">
                <FaIcon name="i-lucide:search" class="mr-1" /> {{ $t('logcenter.toolbar.search') }}
              </FaButton>
              <FaButton v-if="items.length" size="sm" variant="outline" @click="exportTxt">
                <FaIcon name="i-lucide:download" class="mr-1" /> {{ $t('common.export') }}
              </FaButton>
            </div>

            <!-- 高级 LogsQL -->
            <div v-if="!simpleMode" class="flex items-start gap-2">
              <span class="mt-1.5 shrink-0 text-xs text-muted-foreground">LogsQL</span>
              <textarea
                v-model="queryText" rows="2" spellcheck="false"
                class="min-w-0 flex-1 rounded-md border border-input bg-background px-2 py-1.5 font-mono text-xs outline-none focus:ring-1 focus:ring-[rgb(var(--primary))]"
                @input="queryTouched = true" @keydown.enter.exact.prevent="doSearch"
              />
            </div>

            <!-- 直方图 -->
            <div v-if="searched && totalHits > 0" class="rounded-lg border bg-muted/30 px-1 pt-1">
              <div ref="histEl" class="h-24 w-full" :title="$t('logcenter.results.histTitle')" />
            </div>

            <!-- 统计行 -->
            <div v-if="searched" class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <span>{{ $t('logcenter.results.hitsPrefix') }} <b class="text-foreground">{{ totalHits.toLocaleString() }}</b> {{ $t('logcenter.results.hitsSuffix') }} · {{ $t('logcenter.results.loaded', { n: items.length.toLocaleString() }) }}</span>
              <span v-if="truncated" class="rounded-full bg-orange-500/10 px-2 py-0.5 text-orange-500">{{ $t('logcenter.results.pageLimit') }}</span>
              <FaButton size="sm" variant="ghost" class="ml-auto text-muted-foreground" @click="sortDesc = !sortDesc">
                <FaIcon :name="sortDesc ? 'i-lucide:arrow-down-wide-narrow' : 'i-lucide:arrow-up-narrow-wide'" class="mr-1" />
                {{ sortDesc ? $t('logcenter.results.newestFirst') : $t('logcenter.results.oldestFirst') }}
              </FaButton>
            </div>

            <!-- 结果列表 -->
            <div class="overflow-hidden rounded-lg border bg-muted/30">
              <div v-if="searching" class="py-10 text-center text-sm text-muted-foreground">
                {{ $t('logcenter.results.querying') }}
              </div>
              <div v-else-if="!items.length" class="py-10 text-center text-sm text-muted-foreground">
                {{ searched ? $t('logcenter.results.noResults') : $t('logcenter.results.emptyHint') }}
              </div>
              <div v-else class="max-h-[56vh] overflow-auto px-2 py-1.5 font-mono text-xs leading-5">
                <table class="w-full border-collapse">
                  <tbody>
                    <template v-for="(item, i) in displayItems" :key="rowKey(item, i)">
                      <tr class="cursor-pointer align-top hover:bg-accent/30" @click="expandedKey = expandedKey === rowKey(item, i) ? '' : rowKey(item, i)">
                        <td class="w-36 whitespace-nowrap pr-2 text-right text-muted-foreground/80 select-all">{{ fmtTime(item.ts) }}</td>
                        <td class="max-w-40 truncate pr-2 text-muted-foreground" :title="item.node ? `${containerOf(item)} @ ${item.node}` : containerOf(item)">{{ containerOf(item) }}<span v-if="multiNode && item.node" class="ml-1 text-[10px] opacity-70">@{{ item.node }}</span></td>
                        <td class="w-16 whitespace-nowrap pr-2 font-medium" :class="levelMeta(item.msg).meta.cls">{{ levelMeta(item.msg).meta.label }}</td>
                        <td class="whitespace-pre-wrap break-all"><template v-for="(s, si) in segments(item.msg)" :key="si"><span v-if="s.hit" class="rounded bg-yellow-400/40 px-0.5 text-foreground">{{ s.t }}</span><template v-else>{{ s.t }}</template></template></td>
                      </tr>
                      <tr v-if="expandedKey === rowKey(item, i)">
                        <td colspan="4" class="bg-accent/20 px-2 py-2">
                          <div class="flex flex-col gap-2">
                            <div class="flex flex-wrap gap-1.5">
                              <span v-for="[k, v] in streamKvs(item.stream)" :key="k" class="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
                                <span class="opacity-70">{{ k }}</span>={{ v }}
                              </span>
                            </div>
                            <table v-if="msgJson(item.msg)" class="w-full border-collapse text-[11px]">
                              <tbody>
                                <tr v-for="(v, k) in msgJson(item.msg)" :key="k" class="align-top hover:bg-accent/30">
                                  <td class="w-52 whitespace-pre-wrap break-all pr-2 text-muted-foreground">{{ k }}</td>
                                  <td class="whitespace-pre-wrap break-all">{{ typeof v === 'object' ? JSON.stringify(v) : v }}</td>
                                </tr>
                              </tbody>
                            </table>
                            <div v-else class="break-all whitespace-pre-wrap text-muted-foreground">{{ item.msg }}</div>
                            <div class="flex gap-2">
                              <FaButton size="sm" variant="outline" @click.stop="copyText(item.msg, $t('logcenter.results.copiedRaw'))">{{ $t('logcenter.results.copyRaw') }}</FaButton>
                              <FaButton v-if="msgJson(item.msg)" size="sm" variant="outline" @click.stop="copyText(JSON.stringify(msgJson(item.msg), null, 2), $t('logcenter.results.copiedJson'))">{{ $t('logcenter.results.copyJson') }}</FaButton>
                            </div>
                          </div>
                        </td>
                      </tr>
                    </template>
                  </tbody>
                </table>
                <div v-if="items.length < totalHits" class="py-2 text-center">
                  <FaButton size="sm" variant="outline" :loading="loadingMore" @click="loadMore">{{ $t('logcenter.results.loadMore', { loaded: items.length, total: totalHits.toLocaleString() }) }}</FaButton>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 历史/常用管理 -->
    <FaModal v-model="historyModal" :title="$t('logcenter.history.title')" :destroy-on-close="true">
      <div class="flex flex-col gap-2">
        <div v-if="!historyList.length" class="py-6 text-center text-sm text-muted-foreground">
          {{ $t('logcenter.history.empty') }}
        </div>
        <div v-for="h in historyList" :key="h.id" class="flex items-center gap-2 rounded-md border px-2.5 py-2 text-xs">
          <span
            class="cursor-pointer text-sm leading-none" :class="h.pinned ? 'text-orange-400' : 'text-muted-foreground/30 hover:text-muted-foreground'"
            :title="$t('logcenter.history.pinTitle')" @click="historyApi.pin(h.id, !h.pinned, h.remark).then(refreshHistory)"
          >★</span>
          <div class="min-w-0 flex-1">
            <div class="truncate font-medium" :title="h.queryText">{{ itemLabel(h) }}</div>
            <div class="truncate text-[10px] text-muted-foreground" :title="h.queryText">{{ h.queryText }} · {{ h.rangeType }}</div>
          </div>
          <input
            v-if="h.pinned" :value="h.remark" :placeholder="$t('logcenter.history.remarkPlaceholder')"
            class="h-6 w-24 rounded border border-input bg-background px-1.5 text-xs outline-none"
            @change="(e: any) => historyApi.pin(h.id, true, e.target.value).then(refreshHistory)"
          >
          <FaButton size="sm" variant="outline" @click="applyHistory(h); historyModal = false">{{ $t('logcenter.history.use') }}</FaButton>
          <FaButton size="sm" variant="ghost" class="text-red-500!" @click="historyApi.remove(h.id).then(refreshHistory)">{{ $t('common.delete') }}</FaButton>
        </div>
      </div>
      <template #footer>
        <div class="flex items-center">
          <FaButton size="sm" variant="ghost" class="text-red-500!" @click="historyApi.clearAll().then(refreshHistory)">{{ $t('logcenter.history.clearHistory') }}</FaButton>
          <FaButton size="sm" variant="outline" class="ml-auto" @click="historyModal = false">{{ $t('common.close') }}</FaButton>
        </div>
      </template>
    </FaModal>
  </div>
</template>
