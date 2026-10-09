<script setup lang="ts">
import JsonNode from './JsonNode.vue'

// JSON 树节点（递归）。显式自导入保证递归引用不依赖文件名（见 docs/exp/frontend.md）。
// 展开态：无过滤词时由用户手动切换（初始按 expandLevel）；有过滤词时命中路径强制展开。
const props = withDefaults(defineProps<{
  name: string
  value: unknown
  depth?: number
  keyword?: string
  expandLevel?: number
  expandSignal?: number
  expandDirection?: 'all' | 'none'
  root?: boolean
  isLast?: boolean
}>(), {
  depth: 0,
  keyword: '',
  expandLevel: 2,
  expandSignal: 0,
  expandDirection: 'all',
  root: false,
  isLast: false,
})

// 单分支渲染上限，超出折叠，点击展开全部（防极端大数组卡渲染）
const BRANCH_LIMIT = 100

const isBranch = computed(() => props.value !== null && typeof props.value === 'object')
const isArr = computed(() => Array.isArray(props.value))

const entries = computed<[string, unknown][]>(() => {
  if (!isBranch.value) {
    return []
  }
  return isArr.value
    ? (props.value as unknown[]).map((v, i) => [String(i), v] as [string, unknown])
    : Object.entries(props.value as Record<string, unknown>)
})

const showAll = ref(false)
const visibleEntries = computed(() => showAll.value ? entries.value : entries.value.slice(0, BRANCH_LIMIT))
const hiddenCount = computed(() => Math.max(0, entries.value.length - visibleEntries.value.length))

const manual = ref(props.depth < props.expandLevel)
watch(() => props.expandSignal, () => {
  manual.value = props.expandDirection === 'all'
})
watch(() => props.expandLevel, (lv) => {
  manual.value = props.depth < lv
})

const expanded = computed(() => {
  if (props.keyword) {
    return subtreeMatch(props.value, props.name)
  }
  return manual.value
})

function toggle() {
  manual.value = !manual.value
}

function subtreeMatch(v: unknown, name: string): boolean {
  const kw = props.keyword.toLowerCase()
  if (!kw) {
    return false
  }
  if (!props.root && name.toLowerCase().includes(kw)) {
    return true
  }
  if (v === null || typeof v !== 'object') {
    return String(v).toLowerCase().includes(kw)
  }
  const list: [string, unknown][] = Array.isArray(v)
    ? (v as unknown[]).map((x, i) => [String(i), x] as [string, unknown])
    : Object.entries(v as Record<string, unknown>)
  return list.some(([k, x]) => subtreeMatch(x, k))
}

function selfMatch(): boolean {
  if (!props.keyword) {
    return false
  }
  const kw = props.keyword.toLowerCase()
  return (!props.root && props.name.toLowerCase().includes(kw))
    || (!isBranch.value && String(props.value).toLowerCase().includes(kw))
}

const valueClass = computed(() => {
  if (isBranch.value) {
    return ''
  }
  if (props.value === null || props.value === undefined) {
    return 'text-red-500 dark:text-red-400 italic'
  }
  switch (typeof props.value) {
    case 'string': return 'text-emerald-600 break-all dark:text-emerald-300'
    case 'number': return 'text-amber-600 dark:text-amber-300'
    case 'boolean': return 'text-purple-600 dark:text-purple-300'
    default: return 'text-muted-foreground'
  }
})

const valueText = computed(() => {
  if (isBranch.value) {
    return ''
  }
  if (props.value === undefined) {
    return 'undefined'
  }
  return typeof props.value === 'string' ? JSON.stringify(props.value) : String(props.value)
})

const branchOpen = computed(() => (isArr.value ? '[' : '{'))
const branchClose = computed(() => (isArr.value ? ']' : '}'))
</script>

<template>
  <div>
    <!-- 分支：对象/数组 -->
    <template v-if="isBranch">
      <button
        type="button" class="flex w-full cursor-pointer items-start gap-1 rounded text-left hover:bg-accent/40"
        :class="selfMatch() ? 'bg-amber-500/10' : ''" @click="toggle"
      >
        <FaIcon
          name="i-lucide:chevron-right"
          class="mt-0.5 shrink-0 text-muted-foreground transition-transform"
          :class="expanded ? 'rotate-90' : ''"
        />
        <span class="min-w-0 break-all">
          <span v-if="!root" class="text-sky-600 dark:text-sky-300">&quot;{{ name }}&quot;</span>
          <span v-if="!root" class="text-muted-foreground">: </span>
          <span class="text-muted-foreground">{{ branchOpen }}</span>
          <span v-if="!expanded" class="text-muted-foreground">{{ branchClose }}</span>
          <span v-if="!expanded" class="ml-1 rounded bg-muted px-1 text-[10px] text-muted-foreground">
            {{ $t(isArr ? 'components.ydJsonViewer.itemCount' : 'components.ydJsonViewer.keyCount', { n: entries.length }) }}
          </span>
        </span>
      </button>
      <div v-if="expanded" class="ml-2.5 border-l border-border/70 pl-3">
        <JsonNode
          v-for="([k, v], i) in visibleEntries" :key="k" :name="k" :value="v" :depth="depth + 1"
          :keyword="keyword" :expand-level="expandLevel" :expand-signal="expandSignal"
          :expand-direction="expandDirection" :is-last="i === entries.length - 1"
        />
        <button
          v-if="hiddenCount > 0" type="button"
          class="cursor-pointer text-muted-foreground hover:text-primary"
          @click="showAll = true"
        >
          {{ $t('components.ydJsonViewer.showHidden', { n: hiddenCount }) }}
        </button>
        <div class="text-muted-foreground">
          {{ branchClose }}<span v-if="!isLast">,</span>
        </div>
      </div>
    </template>

    <!-- 叶子：标量 -->
    <div v-else class="flex items-start gap-1 rounded" :class="selfMatch() ? 'bg-amber-500/10' : ''">
      <span class="w-4 shrink-0" />
      <span class="min-w-0 break-all">
        <span v-if="!root" class="text-sky-600 dark:text-sky-300">&quot;{{ name }}&quot;</span>
        <span v-if="!root" class="text-muted-foreground">: </span>
        <span :class="valueClass">{{ valueText }}</span>
        <span v-if="!isLast" class="text-muted-foreground">,</span>
      </span>
    </div>
  </div>
</template>
