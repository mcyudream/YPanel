<script setup lang="ts">
import apiContainer from '@/api/modules/container'

// 容器日志查看器：流式读取 + 跟踪 + 行数/时间戳 + 搜索过滤高亮 + 下载。
const props = defineProps<{
  containerId: string
  containerName?: string
  active?: boolean
}>()

const appAccountStore = useAppAccountStore()

const content = ref('')
const following = ref(false)
const tail = ref(500)
const showTs = ref(true)
const search = ref('')
const autoScroll = ref(true)
const boxRef = useTemplateRef<HTMLElement>('box')

let abort: AbortController | null = null

function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}

async function load(follow: boolean) {
  abort?.abort()
  abort = new AbortController()
  following.value = follow
  if (!follow) {
    content.value = ''
  }
  try {
    const url = `${wsBase()}/${apiContainer.logsURL(props.containerId, appAccountStore.token, tail.value, follow, showTs.value)}`
    const resp = await fetch(url, {
      headers: { Authorization: `Bearer ${appAccountStore.token}` },
      signal: abort.signal,
    })
    if (!resp.ok || !resp.body) {
      throw new Error(`HTTP ${resp.status}`)
    }
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    for (;;) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }
      content.value += decoder.decode(value, { stream: true })
      if (content.value.length > 4_000_000) {
        content.value = content.value.slice(-2_000_000)
      }
      if (autoScroll.value) {
        await nextTick()
        scrollBottom()
      }
    }
  }
  catch (e: any) {
    if (e?.name !== 'AbortError') {
      content.value += `\n[日志流错误] ${e?.message || e}`
    }
  }
  finally {
    following.value = false
  }
}

function scrollBottom() {
  const el = boxRef.value
  if (el) {
    el.scrollTop = el.scrollHeight
  }
}

function stop() {
  abort?.abort()
  following.value = false
}

// 搜索：行过滤 + 关键词高亮（内容先转义再替换，无注入面）
const displayLines = computed(() => {
  const kw = search.value.trim()
  const lines = content.value.split('\n')
  if (!kw) {
    return lines.map(l => ({ text: escapeHtml(l), html: null }))
  }
  const lower = kw.toLowerCase()
  return lines
    .filter(l => l.toLowerCase().includes(lower))
    .map((l) => {
      const esc = escapeHtml(l)
      const idx = l.toLowerCase().indexOf(lower)
      if (idx < 0) {
        return { text: esc, html: null }
      }
      // 在转义后的串中做同样长度偏移（转义不改变关键词本身为纯文本时的位置）
      const re = new RegExp(escapeRegExp(escapeHtml(kw)), 'gi')
      return { text: '', html: esc.replace(re, m => `<mark class="rounded-sm bg-yellow-300/70 px-0.5 text-black">${m}</mark>`) }
    })
})

function escapeHtml(s: string) {
  return s.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
}

function escapeRegExp(s: string) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function download() {
  const blob = new Blob([content.value], { type: 'text/plain;charset=utf-8' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `${props.containerName || props.containerId}-logs.txt`
  a.click()
  URL.revokeObjectURL(a.href)
}

function clearView() {
  content.value = ''
}

watch(() => props.containerId, () => {
  content.value = ''
  load(false)
})

// 首次激活即加载（tab 懒挂载时挂载即激活）
onMounted(() => {
  load(false)
})

onBeforeUnmount(() => {
  abort?.abort()
})
</script>

<template>
  <div class="flex h-full flex-col gap-2">
    <div class="flex flex-wrap items-center gap-2">
      <FaButton size="sm" :variant="following ? 'default' : 'outline'" @click="following ? stop() : load(true)">
        <FaIcon name="i-lucide:radio" class="mr-1" :class="following ? 'animate-pulse' : ''" />
        {{ following ? '跟踪中（点击停止）' : '跟踪日志' }}
      </FaButton>
      <FaButton size="sm" variant="outline" @click="load(false)">
        <FaIcon name="i-lucide:refresh-cw" class="mr-1" />
        刷新
      </FaButton>
      <label class="flex items-center gap-1 text-xs text-muted-foreground">
        行数
        <select
          v-model.number="tail"
          class="h-8 rounded-md border bg-background px-1.5 text-xs outline-none"
          @change="load(false)"
        >
          <option :value="100">100</option>
          <option :value="500">500</option>
          <option :value="1000">1000</option>
          <option :value="5000">5000</option>
          <option :value="10000">10000</option>
        </select>
      </label>
      <label class="flex items-center gap-1 text-xs text-muted-foreground">
        <input v-model="showTs" type="checkbox" class="accent-[var(--primary)]" @change="load(false)">
        时间戳
      </label>
      <label class="flex items-center gap-1 text-xs text-muted-foreground">
        <input v-model="autoScroll" type="checkbox" class="accent-[var(--primary)]">
        自动滚动
      </label>
      <div class="ml-auto flex items-center gap-2">
        <FaInput v-model="search" placeholder="搜索日志（过滤行）…" class="h-8 w-48!" />
        <FaButton variant="outline" size="icon-sm" title="下载日志" @click="download">
          <FaIcon name="i-lucide:download" class="text-sm" />
        </FaButton>
        <FaButton variant="outline" size="icon-sm" title="清空视图" @click="clearView">
          <FaIcon name="i-lucide:eraser" class="text-sm" />
        </FaButton>
      </div>
    </div>
    <div
      ref="box"
      class="min-h-0 flex-1 overflow-auto rounded-md bg-muted/60 p-3 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap"
    >
      <div v-for="(l, i) in displayLines" :key="i" class="min-h-[1.2em]">
        <span v-if="l.html" v-html="l.html" /><template v-else>{{ l.text }}</template>
      </div>
      <div v-if="!displayLines.length" class="text-muted-foreground">
        {{ content ? '（无匹配行）' : '暂无日志，点击「跟踪日志」或「刷新」加载' }}
      </div>
    </div>
    <div class="text-right text-xs text-muted-foreground">
      {{ content.length ? `${displayLines.length} 行（缓冲 ${(content.length / 1024).toFixed(0)} KB）` : '' }}
    </div>
  </div>
</template>
