<script setup lang="ts" generic="Row extends Record<string, any>">
import { computed, ref, watch } from 'vue'
import { i18n } from '@/locales'

/**
 * YdDataTable — 数据表格（yd- 封装，M67）：
 * 列定义驱动 + 表头拖拽换序（storageKey 记忆，仅中间列可拖，固定列位置锁死）+
 * 固定列（sticky left/right，width 必填）+ 受控分页（page/pageSize/total 齐备时渲染页脚）。
 * 单元格渲染：`#cell-<key>` 插槽（作用域 { item }），无插槽回退显示 item[key]；
 * 表头自定义用 `#head-<key>`（如全选勾选框）。行内 <tr> 由组件渲染，列序由组件控制，
 * 页面只需按列提供插槽内容——换序后单元格自动跟随。
 */
export interface YdDataTableColumn {
  key: string
  label?: string
  /** px 宽度；固定列必填（sticky 偏移按此累计），普通列可省略 */
  width?: number
  sortable?: boolean
  fixed?: 'left' | 'right'
  align?: 'left' | 'right' | 'center'
  /** 移动端隐藏（hidden md:table-cell）；平板隐藏用 'lg'。固定列不适用 */
  hidden?: 'md' | 'lg' | 'xl'
}

const props = withDefaults(defineProps<{
  columns: YdDataTableColumn[]
  rows: Row[]
  /** 列序记忆 localStorage 键（空=不记忆） */
  storageKey?: string
  loading?: boolean
  rowKey?: string
  emptyText?: string
  // 受控排序（箭头显示）
  sortKey?: string
  sortOrder?: 'asc' | 'desc'
  /** 行附加类（如选中背景） */
  rowClass?: (item: Row) => string
  // 受控分页：三者齐备渲染页脚（size 为 v-model:size 的 prop 部分，必须声明否则
  // 透传到根元素，[size] 属性会触发布局塌缩——生产网站页实证）
  page?: number
  size?: number
  pageSize?: number
  total?: number
}>(), {
  storageKey: '',
  loading: false,
  rowKey: 'id',
  emptyText: '',
  sortKey: '',
  sortOrder: 'asc',
  rowClass: undefined,
  page: undefined,
  size: undefined,
  pageSize: undefined,
  total: undefined,
})

const emit = defineEmits<{
  'update:page': [page: number]
  'update:size': [size: number]
  sort: [key: string]
}>()

const t = (k: string, v?: Record<string, any>) => i18n.global.t(k, v ?? {})

// ---- 列序：固定列锁死（定义序），中间列可拖拽换序并记忆 ----
const middleOrder = ref<string[]>([])

function loadOrder() {
  const def = props.columns.filter(c => !c.fixed).map(c => c.key)
  let saved: string[] | null = null
  if (props.storageKey) {
    try {
      const raw = JSON.parse(localStorage.getItem(`ydtable:${props.storageKey}`) || 'null')
      if (Array.isArray(raw)) {
        saved = raw
      }
    }
    catch {}
  }
  if (!saved) {
    middleOrder.value = def
    return
  }
  // 记忆序与定义求交并补新列（列定义变更后记忆不丢列）
  const known = saved.filter(k => def.includes(k))
  middleOrder.value = [...known, ...def.filter(k => !known.includes(k))]
}

loadOrder()
watch(() => [props.columns, props.storageKey], loadOrder)

function saveOrder() {
  if (props.storageKey) {
    localStorage.setItem(`ydtable:${props.storageKey}`, JSON.stringify(middleOrder.value))
  }
}

const byKey = computed(() => Object.fromEntries(props.columns.map(c => [c.key, c])))
const fixedLeft = computed(() => props.columns.filter(c => c.fixed === 'left'))
const fixedRight = computed(() => props.columns.filter(c => c.fixed === 'right'))
const orderedColumns = computed<YdDataTableColumn[]>(() => [
  ...fixedLeft.value,
  ...middleOrder.value.map(k => byKey.value[k]).filter(Boolean),
  ...fixedRight.value,
])

// sticky 偏移：同侧固定列宽度累计（约定 fixed 列必填 width 且不设 hidden）
function stickyStyle(col: YdDataTableColumn): Record<string, string> | undefined {
  if (!col.fixed || !col.width) {
    return undefined
  }
  if (col.fixed === 'left') {
    const before = fixedLeft.value.slice(0, fixedLeft.value.indexOf(col))
    return { left: `${before.reduce((n, c) => n + (c.width || 0), 0)}px` }
  }
  const after = fixedRight.value.slice(fixedRight.value.indexOf(col) + 1)
  return { right: `${after.reduce((n, c) => n + (c.width || 0), 0)}px` }
}

// ---- 表头拖拽换序（仅中间列） ----
const dragKey = ref('')
const dragOverKey = ref('')

function onDrop(targetKey: string) {
  const from = middleOrder.value.indexOf(dragKey.value)
  const to = middleOrder.value.indexOf(targetKey)
  dragKey.value = ''
  dragOverKey.value = ''
  if (from < 0 || to < 0 || from === to) {
    return
  }
  const next = [...middleOrder.value]
  next.splice(to, 0, next.splice(from, 1)[0])
  middleOrder.value = next
  saveOrder()
}

// ---- 单元格回退渲染 ----
function cellText(item: Row, key: string): string {
  const v = item[key]
  if (v === null || v === undefined || v === '') {
    return '—'
  }
  if (typeof v === 'boolean') {
    return v ? '✓' : '—'
  }
  return String(v)
}

const alignCls = (a?: string) => ({ left: 'text-left', right: 'text-right', center: 'text-center' }[a || 'left'])
const hiddenCls = (h?: string) => (h ? `hidden ${h}:table-cell` : '')

// 受控分页转发
const pageModel = computed({
  get: () => props.page ?? 1,
  set: (v: number) => emit('update:page', v),
})
const sizeModel = computed({
  get: () => props.size ?? props.pageSize ?? 20,
  set: (v: number) => emit('update:size', v),
})
const hasPager = computed(() => props.page !== undefined && props.size !== undefined && props.total !== undefined)
</script>

<template>
  <div class="flex w-full flex-col gap-2">
    <div class="relative overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
          <tr class="border-b border-border">
            <th
              v-for="col in orderedColumns"
              :key="col.key"
              :style="stickyStyle(col)"
              :class="[
                'px-3 py-2 select-none whitespace-nowrap',
                alignCls(col.align),
                hiddenCls(col.hidden),
                col.fixed ? 'sticky z-10 bg-muted' : '',
                dragOverKey === col.key ? 'border-x-2 border-dashed border-primary/40' : '',
                col.sortable ? 'cursor-pointer hover:text-foreground' : '',
              ]"
              :draggable="!col.fixed"
              :title="!col.fixed ? t('common.dragToReorder') : undefined"
              @dragstart="!col.fixed && (dragKey = col.key)"
              @dragover.prevent="!col.fixed && dragKey && (dragOverKey = col.key)"
              @dragend="dragKey = ''; dragOverKey = ''"
              @drop.prevent="onDrop(col.key)"
              @click="col.sortable && emit('sort', col.key)"
            >
              <slot :name="`head-${col.key}`">
                <span :class="sortKey === col.key && col.sortable ? 'font-medium text-foreground' : ''">
                  {{ col.label }}{{ col.sortable ? (sortKey === col.key ? (sortOrder === 'asc' ? ' ↑' : ' ↓') : ' ⇅') : '' }}
                </span>
              </slot>
            </th>
          </tr>
        </thead>
        <tbody>
          <template v-if="rows.length">
            <tr v-for="item in rows" :key="String(item[rowKey])" class="border-b border-border/60 hover:bg-muted/40" :class="rowClass?.(item)">
              <td
                v-for="col in orderedColumns"
                :key="col.key"
                :style="stickyStyle(col)"
                :class="[
                  'px-3 py-2 align-middle',
                  alignCls(col.align),
                  hiddenCls(col.hidden),
                  col.fixed ? 'sticky z-10 bg-background' : '',
                ]"
              >
                <slot :name="`cell-${col.key}`" :item="item">{{ cellText(item, col.key) }}</slot>
              </td>
            </tr>
          </template>
          <tr v-else-if="!loading">
            <td :colspan="orderedColumns.length" class="px-3 py-10 text-center text-sm text-muted-foreground">
              <slot name="empty">{{ emptyText || t('common.noData') }}</slot>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="loading" class="py-6 text-center text-sm text-muted-foreground">{{ t('common.loading') }}</div>
    </div>

    <div v-if="hasPager" class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-3 text-xs text-muted-foreground">
        <span>{{ t('common.totalItems', { n: total }) }}</span>
        <slot name="footerLeft" />
      </div>
      <FaPagination v-model:page="pageModel" v-model:size="sizeModel" :total="total || 0" />
    </div>
  </div>
</template>
