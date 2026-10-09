<script setup lang="ts">
// 通用数据网格：查询结果（只读）/ 表浏览（可编辑：单元格改写、行删除、行插入、CSV 导出）。
import { i18n } from '@/locales'

const props = defineProps<{
  columns: string[]
  rows: unknown[][]
  total?: number
  offset?: number
  limit?: number
  loading?: boolean
  elapsed?: string
  /** 表浏览模式开启编辑（需配合 pk 列名） */
  editable?: boolean
  pkColumns?: string[]
  columnMeta?: { name: string, dataType: string, nullable: boolean }[]
}>()

const emit = defineEmits<{
  page: [offset: number]
  cellSave: [pk: Record<string, unknown>, values: Record<string, unknown>]
  rowDelete: [pk: Record<string, unknown>]
  rowInsert: [values: Record<string, unknown>]
  sort: [col: string]
}>()

const toast = useFaToast()
const modal = useFaModal()

// ---- 单元格显示 ----
function display(v: unknown): string {
  if (v === null || v === undefined) {
    return 'NULL'
  }
  if (typeof v === 'object') {
    try {
      return JSON.stringify(v)
    }
    catch {
      return String(v)
    }
  }
  return String(v)
}

function isNull(v: unknown) {
  return v === null || v === undefined
}

// ---- 列宽拖动 ----
const colWidths = ref<Record<number, number>>({})
let resizing = { col: -1, startX: 0, startW: 0 }

function startResize(e: MouseEvent, idx: number) {
  resizing = { col: idx, startX: e.clientX, startW: colWidths.value[idx] ?? 160 }
  const onMove = (ev: MouseEvent) => {
    colWidths.value[resizing.col] = Math.max(60, resizing.startW + ev.clientX - resizing.startX)
  }
  const onUp = () => {
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
  }
  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
}

// ---- 脏单元格（批量保存） ----
const dirty = ref<Record<number, Record<string, unknown>>>({}) // rowIndex → col → value

function markDirty(ri: number, col: string, value: unknown) {
  if (!dirty.value[ri]) {
    dirty.value[ri] = {}
  }
  dirty.value[ri][col] = value
}

function pkOf(ri: number): Record<string, unknown> | null {
  if (!props.pkColumns?.length) {
    return null
  }
  const pk: Record<string, unknown> = {}
  for (const c of props.pkColumns) {
    const ci = props.columns.indexOf(c)
    if (ci < 0) {
      return null
    }
    pk[c] = props.rows[ri]?.[ci]
  }
  return pk
}

const dirtyCount = computed(() => Object.keys(dirty.value).length)

async function saveDirty() {
  for (const [riStr, values] of Object.entries(dirty.value)) {
    const pk = pkOf(Number(riStr))
    if (!pk) {
      toast.error(i18n.global.t('dbadmin.grid.rowMissingPk', { n: Number(riStr) + 1 }))
      continue
    }
    emit('cellSave', pk, values)
  }
  dirty.value = {}
}

function discardDirty() {
  dirty.value = {}
}

// ---- 行删除 ----
async function delRow(ri: number) {
  const pk = pkOf(ri)
  if (!pk) {
    toast.warning(i18n.global.t('dbadmin.grid.noPkDelete'))
    return
  }
  const ok = await modal.confirm({
    title: i18n.global.t('dbadmin.common.deleteRow'),
    content: i18n.global.t('dbadmin.grid.deleteRowConfirm', { n: ri + 1, pk: (props.pkColumns ?? []).map(c => `${c}=${display(pk[c])}`).join(', ') }),
  })
  if (ok) {
    emit('rowDelete', pk)
  }
}

// ---- 行插入 ----
const insertOpen = ref(false)
const insertValues = ref<Record<string, string>>({})

function openInsert() {
  insertValues.value = Object.fromEntries(props.columns.map(c => [c, '']))
  insertOpen.value = true
}

function submitInsert() {
  const values: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(insertValues.value)) {
    if (v !== '') {
      values[k] = coerce(k, v)
    }
  }
  if (!Object.keys(values).length) {
    toast.warning(i18n.global.t('dbadmin.grid.fillOneColumn'))
    return
  }
  emit('rowInsert', values)
  insertOpen.value = false
}

// 值类型推断：数字/布尔/JSON 对象，其余字符串
function coerce(col: string, raw: string): unknown {
  const meta = props.columnMeta?.find(c => c.name === col)
  const t = meta?.dataType.toLowerCase() ?? ''
  if (t.includes('int') || t.includes('numeric') || t.includes('decimal') || t.includes('real') || t.includes('double') || t.includes('float')) {
    const n = Number(raw)
    if (!Number.isNaN(n)) {
      return n
    }
  }
  if (t.includes('bool')) {
    return raw === 'true' || raw === '1' || raw === 't'
  }
  if (raw.startsWith('{') || raw.startsWith('[')) {
    try {
      return JSON.parse(raw)
    }
    catch {
      return raw
    }
  }
  return raw
}

// ---- CSV 导出 ----
function csvCell(v: unknown) {
  const s = v === null || v === undefined ? '' : display(v)
  return `"${s.replaceAll('"', '""')}"`
}

function exportCsv() {
  const lines = [props.columns.map(csvCell).join(',')]
  for (const row of props.rows) {
    lines.push(row.map(csvCell).join(','))
  }
  const blob = new Blob([`\uFEFF${lines.join('\n')}`], { type: 'text/csv;charset=utf-8' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `export-${Date.now()}.csv`
  a.click()
  URL.revokeObjectURL(a.href)
}

// ---- 分页 ----
const page = computed(() => {
  if (props.total === undefined) {
    return null
  }
  const off = props.offset ?? 0
  const lim = props.limit ?? 200
  return {
    cur: Math.floor(off / lim) + 1,
    pages: Math.max(1, Math.ceil(props.total / lim)),
    off,
    lim,
  }
})

function gotoPage(p: number) {
  if (page.value) {
    emit('page', (p - 1) * page.value.lim)
  }
}

// ---- 单元格编辑（双击就地 input，Enter/失焦记入脏集合，批量保存） ----
const editing = ref<{ ri: number, ci: number, text: string } | null>(null)

function beginCell(ri: number, ci: number) {
  if (!props.editable || !props.pkColumns?.length) {
    return
  }
  const original = props.rows[ri]?.[ci]
  editing.value = { ri, ci, text: isNull(original) ? '' : display(original) }
}

function commitCell() {
  const ed = editing.value
  if (!ed) {
    return
  }
  const col = props.columns[ed.ci]
  const original = props.rows[ed.ri]?.[ed.ci]
  const newText = ed.text
  // 与原值相同（或均为空）则不标记
  if (isNull(original) ? newText === '' : display(original) === newText) {
    editing.value = null
    return
  }
  markDirty(ed.ri, col, newText === '' ? null : coerce(col, newText))
  editing.value = null
}

function onCellKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    commitCell()
  }
  else if (e.key === 'Escape') {
    editing.value = null
  }
}
</script>

<template>
  <div class="flex size-full min-h-0 flex-col">
    <div class="min-h-0 flex-1 overflow-auto rounded-md border">
      <table class="w-max text-xs">
        <thead class="sticky top-0 z-10 bg-muted/80 text-left backdrop-blur">
          <tr>
            <th v-if="editable" class="w-8 border-b px-2 py-1.5" />
            <th
              v-for="(c, ci) in columns"
              :key="c"
              class="relative cursor-pointer whitespace-nowrap border-b px-2 py-1.5 font-mono hover:text-primary"
              :style="{ width: `${colWidths[ci] ?? 160}px`, minWidth: `${colWidths[ci] ?? 160}px` }"
              :title="$t('dbadmin.grid.sortBy', { col: c })"
              @click="emit('sort', c)"
            >
              {{ c }}
              <span
                v-if="pkColumns?.includes(c)"
                class="ml-1 rounded bg-amber-500/20 px-1 text-[9px] text-amber-600"
                :title="$t('dbadmin.grid.pk')"
              >PK</span>
              <span
                class="absolute right-0 top-0 h-full w-1 cursor-col-resize hover:bg-primary/40"
                @click.stop
                @mousedown.stop="startResize($event, ci)"
              />
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td :colspan="columns.length + 1" class="px-3 py-6 text-center text-muted-foreground">{{ $t('common.loading') }}</td>
          </tr>
          <tr v-else-if="!rows.length">
            <td :colspan="columns.length + 1" class="px-3 py-6 text-center text-muted-foreground">{{ $t('dbadmin.grid.noData') }}</td>
          </tr>
          <tr v-for="(row, ri) in rows" :key="ri" class="group border-t hover:bg-accent/30">
            <td v-if="editable" class="border-t px-1 text-center">
              <button
                type="button"
                class="rounded p-0.5 text-muted-foreground opacity-0 transition-opacity hover:text-red-500 group-hover:opacity-100"
                :title="$t('dbadmin.common.deleteRow')"
                @click="delRow(ri)"
              >
                <YdMorphIcon name="trash" :size="13" />
              </button>
            </td>
            <td v-for="(cell, ci) in row" :key="ci" class="border-t px-2 py-1 align-top">
              <div v-if="dirty[ri]?.[columns[ci]] !== undefined" class="rounded bg-amber-500/15 px-1 py-0.5">
                {{ display(dirty[ri][columns[ci]]) }}
              </div>
              <input
                v-else-if="editing && editing.ri === ri && editing.ci === ci"
                v-model="editing.text"
                :ref="(el: any) => el?.focus?.()"
                class="w-full rounded border border-primary bg-background px-1 py-0.5 font-mono text-xs outline-none"
                @keydown="onCellKeydown"
                @blur="commitCell"
              >
              <span
                v-else
                class="block max-w-72 truncate font-mono"
                :class="{ 'text-muted-foreground/60 italic': isNull(cell) }"
                :title="display(cell)"
                @dblclick="beginCell(ri, ci)"
              >{{ display(cell) }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 底部工具条 -->
    <div class="flex flex-wrap items-center gap-2 pt-2 text-xs text-muted-foreground">
      <span v-if="total !== undefined">{{ $t('dbadmin.grid.totalRows', { n: total }) }}</span>
      <span v-else>{{ $t('dbadmin.grid.rowsCount', { n: rows.length }) }}</span>
      <span v-if="elapsed">{{ $t('dbadmin.grid.elapsed', { t: elapsed }) }}</span>
      <template v-if="page">
        <span class="ml-2">{{ $t('dbadmin.grid.pageInfo', { cur: page.cur, pages: page.pages }) }}</span>
        <button
          type="button"
          class="rounded border px-1.5 py-0.5 hover:bg-accent/50 disabled:opacity-40"
          :disabled="page.cur <= 1"
          @click="gotoPage(page.cur - 1)"
        >{{ $t('dbadmin.prevPage') }}</button>
        <button
          type="button"
          class="rounded border px-1.5 py-0.5 hover:bg-accent/50 disabled:opacity-40"
          :disabled="page.cur >= page.pages"
          @click="gotoPage(page.cur + 1)"
        >{{ $t('dbadmin.nextPage') }}</button>
      </template>
      <span class="flex-1" />
      <span v-if="dirtyCount" class="text-amber-600">{{ $t('dbadmin.grid.pendingRows', { n: dirtyCount }) }}</span>
      <button v-if="dirtyCount" type="button" class="rounded border px-2 py-0.5 hover:bg-accent/50" @click="discardDirty">{{ $t('dbadmin.grid.discard') }}</button>
      <button v-if="dirtyCount" type="button" class="rounded bg-primary px-2 py-0.5 text-primary-foreground" @click="saveDirty">{{ $t('dbadmin.common.saveChanges') }}</button>
      <button v-if="editable" type="button" class="flex items-center gap-1 rounded border px-2 py-0.5 hover:bg-accent/50" @click="openInsert">
        <YdMorphIcon name="plus" :size="12" /> {{ $t('dbadmin.grid.insertRow') }}
      </button>
      <button type="button" class="flex items-center gap-1 rounded border px-2 py-0.5 hover:bg-accent/50" @click="exportCsv">
        <YdMorphIcon name="download" :size="12" /> {{ $t('dbadmin.grid.exportCsv') }}
      </button>
    </div>

    <!-- 插入行弹窗 -->
    <FaModal v-model="insertOpen" :destroy-on-close="true" :title="$t('dbadmin.grid.insertRow')" :width="520">
      <div class="max-h-[60vh] space-y-2 overflow-auto p-1">
        <div v-for="c in columns" :key="c" class="flex items-center gap-2">
          <label class="w-40 shrink-0 truncate text-right font-mono text-xs" :title="c">
            {{ c }}
            <span v-if="pkColumns?.includes(c)" class="text-amber-600">*</span>
          </label>
          <input
            v-model="insertValues[c]"
            class="flex-1 rounded border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
            :placeholder="columnMeta?.find(m => m.name === c)?.dataType ?? ''"
          >
        </div>
        <p class="pt-1 text-xs text-muted-foreground">{{ $t('dbadmin.grid.insertHint') }}</p>
      </div>
      <template #footer>
        <FaButton size="sm" @click="insertOpen = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton type="primary" size="sm" @click="submitInsert">{{ $t('dbadmin.grid.insert') }}</FaButton>
      </template>
    </FaModal>
  </div>
</template>
