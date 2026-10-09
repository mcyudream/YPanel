<script setup lang="ts">
import type * as Monaco from 'monaco-editor'
import type { MongoDocPage, MongoIndex } from '@/api/modules/dbadmin'
import apiDBA from '@/api/modules/dbadmin'
import YdCodeEditor from '@/components/YdCodeEditor/index.vue'
import JsonNode from '@/components/YdJsonViewer/JsonNode.vue'
import { i18n } from '@/locales'
import { loadMonaco } from '@/utils/monacoLoader'

// MongoDB 工作台（对齐 Compass 核心体验）：
// Documents（列表/表格双视图 + filter/project/sort/分页 + 克隆/导出）｜Aggregations（管道构建器，只读）｜Indexes（定义/唯一/大小/使用统计）。
const props = defineProps<{
  instanceId: number
  db: string
  collection: string
}>()

const toast = useFaToast()
const modal = useFaModal()

const tab = ref<'docs' | 'agg' | 'idx'>('docs')

// ---- 文档浏览 ----
const filterText = ref('')
const projectText = ref('')
const sortField = ref('_id')
const sortDir = ref<'asc' | 'desc'>('desc')
const pageSize = ref(50)
const page = ref<MongoDocPage>()
const loading = ref(false)
const error = ref('')
const viewMode = ref<'list' | 'table'>('list')

const skip = computed(() => page.value?.skip ?? 0)

async function load() {
  if (!props.collection || !props.db) {
    page.value = undefined
    return
  }
  loading.value = true
  error.value = ''
  try {
    if (filterText.value.trim()) {
      JSON.parse(filterText.value)
    }
    if (projectText.value.trim()) {
      JSON.parse(projectText.value)
    }
    page.value = await apiDBA.mongoDocs(props.instanceId, {
      db: props.db, collection: props.collection,
      filter: filterText.value || undefined,
      project: projectText.value || undefined,
      sort: sortField.value || undefined, dir: sortDir.value,
      skip: skip.value, limit: pageSize.value,
    })
  }
  catch (e: any) {
    error.value = e?.message || i18n.global.t('dbadmin.common.queryFailed')
    page.value = undefined
  }
  finally {
    loading.value = false
  }
}

function gotoPage(p: number) {
  if (page.value) {
    page.value.skip = (p - 1) * pageSize.value
    load()
  }
}

const curPageNo = computed(() => Math.floor((page.value?.skip ?? 0) / pageSize.value) + 1)
const totalPages = computed(() => Math.max(1, Math.ceil((page.value?.total ?? 0) / pageSize.value)))

watch(() => [props.instanceId, props.db, props.collection] as const, () => {
  page.value = undefined
  load()
}, { immediate: true })

// ---- 展示转换 ----
function docId(docStr: string): string {
  try {
    const id = JSON.parse(docStr)._id
    return JSON.stringify(id)
  }
  catch {
    return ''
  }
}

// 展示用：ExtJSON 特殊类型转可读形态（编辑仍用原始 ExtJSON 串）
function toDisplayDoc(docStr: string): unknown {
  const DATE_RE = /^\d{4}-\d{2}-\d{2}T/
  function walk(v: unknown): unknown {
    if (Array.isArray(v)) {
      return v.map(walk)
    }
    if (v && typeof v === 'object') {
      const obj = v as Record<string, unknown>
      const keys = Object.keys(obj)
      if (keys.length === 1 && '$oid' in obj && typeof obj.$oid === 'string') {
        return `ObjectId("${obj.$oid}")`
      }
      if (keys.length === 1 && '$date' in obj) {
        const d = obj.$date
        if (typeof d === 'string' && DATE_RE.test(d)) {
          return `Date("${d}")`
        }
        if (d && typeof d === 'object' && '$numberLong' in (d as Record<string, unknown>)) {
          const ms = Number((d as Record<string, unknown>).$numberLong)
          if (!Number.isNaN(ms)) {
            return `Date("${new Date(ms).toISOString()}")`
          }
        }
      }
      const out: Record<string, unknown> = {}
      for (const [k, vv] of Object.entries(obj)) {
        out[k] = walk(vv)
      }
      return out
    }
    return v
  }
  try {
    return walk(JSON.parse(docStr))
  }
  catch {
    return docStr
  }
}

// 编辑器内展示用：仅格式化，不转换 ExtJSON（保存按原文提交）
function pretty(docStr: string): string {
  try {
    return JSON.stringify(JSON.parse(docStr), null, 2)
  }
  catch {
    return docStr
  }
}

// ---- 表格视图：动态列 + 单元格摘要 ----
const tableCols = computed<string[]>(() => {
  const cols: string[] = []
  for (const ds of page.value?.docs ?? []) {
    try {
      for (const k of Object.keys(JSON.parse(ds))) {
        if (!cols.includes(k)) {
          cols.push(k)
        }
      }
    }
    catch {
      // 跳过坏文档
    }
  }
  return ['_id', ...cols.filter(c => c !== '_id')].slice(0, 12)
})

function cellValue(docStr: string, col: string): unknown {
  try {
    return (JSON.parse(docStr) as Record<string, unknown>)[col]
  }
  catch {
    return undefined
  }
}

function cellSummary(v: unknown): string {
  if (v === null || v === undefined) {
    return '—'
  }
  if (Array.isArray(v)) {
    return `[ ] ${v.length} items`
  }
  if (typeof v === 'object') {
    const obj = v as Record<string, unknown>
    if (Object.keys(obj).length === 1 && '$oid' in obj) {
      return String(obj.$oid)
    }
    return `{ } ${Object.keys(obj).length} fields`
  }
  const s = String(v)
  return s.length > 36 ? `${s.slice(0, 36)}…` : s
}

// ---- 文档编辑（monaco json，ExtJSON 往返） ----
const editorOpen = ref(false)
const editorMode = ref<'insert' | 'edit' | 'clone'>('insert')
const editId = ref('')
const docModel = shallowRef<Monaco.editor.ITextModel | null>(null)

async function openEditor(mode: 'insert' | 'edit' | 'clone', docStr?: string) {
  editorMode.value = mode
  editId.value = mode === 'edit' && docStr ? docId(docStr) : ''
  const m = await loadMonaco()
  let text = '{\n  \n}'
  if (mode === 'edit' && docStr) {
    text = pretty(docStr)
  }
  else if (mode === 'clone' && docStr) {
    // 克隆：去掉 _id 预填
    try {
      const doc = JSON.parse(docStr) as Record<string, unknown>
      delete doc._id
      text = JSON.stringify(doc, null, 2)
    }
    catch {
      text = '{\n  \n}'
    }
  }
  docModel.value?.dispose()
  docModel.value = m.editor.createModel(text, 'json')
  editorOpen.value = true
}

async function submitDoc() {
  const text = docModel.value?.getValue().trim()
  if (!text) {
    toast.warning(i18n.global.t('dbadmin.mongo.emptyDoc'))
    return
  }
  try {
    JSON.parse(text)
  }
  catch {
    toast.error(i18n.global.t('dbadmin.mongo.invalidJson'))
    return
  }
  try {
    if (editorMode.value === 'insert' || editorMode.value === 'clone') {
      await apiDBA.mongoDocInsert(props.instanceId, { db: props.db, collection: props.collection, doc: text })
      toast.success(i18n.global.t('dbadmin.common.inserted'))
    }
    else {
      await apiDBA.mongoDocUpdate(props.instanceId, { db: props.db, collection: props.collection, id: editId.value, doc: text })
      toast.success(i18n.global.t('dbadmin.mongo.updated'))
    }
    editorOpen.value = false
    load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.saveFailed'), { description: e?.message })
  }
}

async function delDoc(docStr: string) {
  const id = docId(docStr)
  const ok = await modal.confirm({ title: i18n.global.t('dbadmin.mongo.deleteDoc'), content: i18n.global.t('dbadmin.mongo.deleteDocConfirm') })
  if (!ok) {
    return
  }
  try {
    await apiDBA.mongoDocDelete(props.instanceId, { db: props.db, collection: props.collection, id })
    toast.success(i18n.global.t('dbadmin.common.deleted'))
    load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.deleteFailed'), { description: e?.message })
  }
}

// ---- 导出当前结果（JSON） ----
function exportDocs() {
  const docs = (page.value?.docs ?? []).map(d => pretty(d)).join(',\n')
  const blob = new Blob([`[\n${docs}\n]`], { type: 'application/json' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `${props.collection || 'export'}-${Date.now()}.json`
  a.click()
  URL.revokeObjectURL(a.href)
}

// ---- 聚合管道（只读，$out/$merge 后端拦截） ----
const stages = ref<string[]>(['{ "$match": { } }'])
const aggResult = ref<MongoDocPage>()
const aggLoading = ref(false)
const aggError = ref('')
const MAX_DOCS = 200

function addStage() {
  stages.value.push('{ "$sort": { "_id": -1 } }')
}

function removeStage(i: number) {
  stages.value.splice(i, 1)
}

async function runAggregate() {
  aggError.value = ''
  aggResult.value = undefined
  const valid = stages.value.map(s => s.trim()).filter(Boolean)
  if (!valid.length) {
    toast.warning(i18n.global.t('dbadmin.mongo.stageRequired'))
    return
  }
  aggLoading.value = true
  try {
    aggResult.value = await apiDBA.mongoAggregate(props.instanceId, {
      db: props.db, collection: props.collection, stages: valid, maxDocs: MAX_DOCS,
    })
  }
  catch (e: any) {
    aggError.value = e?.message || i18n.global.t('dbadmin.mongo.aggFailed')
  }
  finally {
    aggLoading.value = false
  }
}

// ---- Compass JSON 导入 ----
const importOpen = ref(false)
const importFile = ref<File | null>(null)
const importing = ref(false)

function openImport() {
  importFile.value = null
  importOpen.value = true
}

async function submitImport() {
  if (!importFile.value) {
    toast.warning(i18n.global.t('dbadmin.mongo.chooseImportFile'))
    return
  }
  const content = await importFile.value.text()
  const trimmed = content.trimStart()
  const format = trimmed.startsWith('[') || trimmed.startsWith('{') ? 'json' : 'ndjson'
  importing.value = true
  try {
    const r = await apiDBA.mongoImport(props.instanceId, { db: props.db, collection: props.collection, content, format })
    toast.success(i18n.global.t('dbadmin.mongo.imported', { n: r.imported }))
    importOpen.value = false
    load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.importFailed'), { description: e?.message })
  }
  finally {
    importing.value = false
  }
}

// ---- 索引管理 ----
const indexes = ref<MongoIndex[]>([])
const idxLoading = ref(false)
const idxOpen = ref(false)
const idxForm = ref({ name: '', keysText: '', unique: false })

async function loadIndexes() {
  if (!props.collection) {
    indexes.value = []
    return
  }
  idxLoading.value = true
  try {
    indexes.value = await apiDBA.mongoIndexes(props.instanceId, props.db, props.collection)
  }
  catch {
    indexes.value = []
  }
  finally {
    idxLoading.value = false
  }
}

async function submitIndex() {
  const keys: Record<string, number> = {}
  for (const pair of idxForm.value.keysText.split(',')) {
    const [f, d] = pair.split(':').map(s => s.trim())
    if (f) {
      keys[f] = d === '-1' ? -1 : 1
    }
  }
  if (!Object.keys(keys).length) {
    toast.warning(i18n.global.t('dbadmin.mongo.keysRequired'))
    return
  }
  try {
    await apiDBA.mongoIndexCreate(props.instanceId, {
      db: props.db, collection: props.collection,
      name: idxForm.value.name || undefined, keys, unique: idxForm.value.unique,
    })
    toast.success(i18n.global.t('dbadmin.common.indexCreated'))
    idxOpen.value = false
    loadIndexes()
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.createFailed'), { description: e?.message })
  }
}

async function dropIndex(name: string) {
  if (name === '_id_') {
    toast.warning(i18n.global.t('dbadmin.mongo.idIndexProtected'))
    return
  }
  const ok = await modal.confirm({ title: i18n.global.t('dbadmin.common.indexDeleteTitle'), content: i18n.global.t('dbadmin.common.indexDeleteConfirm', { name }) })
  if (!ok) {
    return
  }
  try {
    await apiDBA.mongoIndexDrop(props.instanceId, { db: props.db, collection: props.collection, name })
    toast.success(i18n.global.t('dbadmin.common.deleted'))
    loadIndexes()
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.deleteFailed'), { description: e?.message })
  }
}

function keysText(idx: MongoIndex): string {
  return Object.entries(idx.keys).map(([k, v]) => `${k}:${v}`).join(', ')
}

// ---- 集合管理（本地输入弹窗） ----
const collOpen = ref(false)
const collAction = ref<'create' | 'rename'>('create')
const collName = ref('')

function openCollAction(action: 'create' | 'rename') {
  if (action === 'rename' && !props.collection) {
    return
  }
  collAction.value = action
  collName.value = action === 'rename' ? props.collection : ''
  collOpen.value = true
}

async function submitColl() {
  const name = collName.value.trim()
  if (!name) {
    toast.warning(i18n.global.t('dbadmin.mongo.collNameRequired'))
    return
  }
  try {
    if (collAction.value === 'create') {
      await apiDBA.mongoCollection(props.instanceId, { db: props.db, action: 'create', name })
      toast.success(i18n.global.t('dbadmin.mongo.collCreated'))
    }
    else {
      await apiDBA.mongoCollection(props.instanceId, { db: props.db, action: 'rename', name: props.collection, to: name })
      toast.success(i18n.global.t('dbadmin.mongo.collRenamed'))
    }
    collOpen.value = false
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.operationFailed'), { description: e?.message })
  }
}

async function dropCollection() {
  if (!props.collection) {
    return
  }
  const ok = await modal.confirm({ title: i18n.global.t('dbadmin.mongo.deleteCollection'), content: i18n.global.t('dbadmin.mongo.deleteCollectionConfirm', { name: props.collection }) })
  if (!ok) {
    return
  }
  try {
    await apiDBA.mongoCollection(props.instanceId, { db: props.db, action: 'drop', name: props.collection })
    toast.success(i18n.global.t('dbadmin.mongo.collDropped'))
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.deleteFailed'), { description: e?.message })
  }
}

watch(tab, (t) => {
  if (t === 'idx') {
    loadIndexes()
  }
})

onBeforeUnmount(() => {
  docModel.value?.dispose()
})
</script>

<template>
  <div class="flex size-full min-h-0 flex-col gap-2">
    <!-- Tab 行 + 集合操作 -->
    <div class="flex items-center gap-1">
      <button
        v-for="t in [
          { k: 'docs', label: `Documents${page ? ` ${page.total}` : ''}` },
          { k: 'agg', label: 'Aggregations' },
          { k: 'idx', label: 'Indexes' },
        ]"
        :key="t.k"
        type="button"
        class="rounded-md px-2.5 py-1 text-xs transition-colors"
        :class="tab === t.k ? 'bg-primary/10 font-medium text-primary' : 'text-muted-foreground hover:bg-accent/50'"
        @click="tab = t.k as any"
      >
        {{ t.label }}
      </button>
      <span class="flex-1" />
      <FaButton size="sm" variant="outline" @click="openCollAction('create')">{{ $t('dbadmin.mongo.createCollection') }}</FaButton>
      <FaButton size="sm" variant="outline" :disabled="!collection" @click="openCollAction('rename')">{{ $t('dbadmin.mongo.rename') }}</FaButton>
      <FaButton size="sm" variant="outline" status="danger" :disabled="!collection" @click="dropCollection">{{ $t('dbadmin.mongo.deleteCollection') }}</FaButton>
    </div>

    <div v-if="!collection" class="flex flex-1 items-center justify-center text-sm text-muted-foreground">
      {{ $t('dbadmin.mongo.selectCollectionHint') }}
    </div>

    <!-- ============ Documents ============ -->
    <template v-else-if="tab === 'docs'">
      <div class="flex flex-wrap items-center gap-1.5 text-xs">
        <input
          v-model="filterText"
          class="w-56 rounded-md border border-input bg-background px-2 py-1 font-mono outline-none focus:ring-1 focus:ring-primary"
          :placeholder="$t('dbadmin.mongo.filterPlaceholder')"
          @keydown.enter="page && (page.skip = 0, load())"
        >
        <input
          v-model="projectText"
          class="w-40 rounded-md border border-input bg-background px-2 py-1 font-mono outline-none focus:ring-1 focus:ring-primary"
          :placeholder="$t('dbadmin.mongo.projectPlaceholder')"
          @keydown.enter="page && (page.skip = 0, load())"
        >
        <input
          v-model="sortField"
          class="w-24 rounded-md border border-input bg-background px-2 py-1 font-mono outline-none focus:ring-1 focus:ring-primary"
          :placeholder="$t('dbadmin.mongo.sortField')"
          @keydown.enter="page && (page.skip = 0, load())"
        >
        <button
          type="button"
          class="rounded border px-1.5 py-1 hover:bg-accent/50"
          @click="sortDir = sortDir === 'asc' ? 'desc' : 'asc'"
        >
          {{ sortDir === 'asc' ? '↑' : '↓' }}
        </button>
        <select v-model.number="pageSize" class="h-[26px] rounded-md border border-input bg-background px-1 text-xs outline-none" @change="page && (page.skip = 0, load())">
          <option :value="25">{{ $t('dbadmin.mongo.perPage', { n: 25 }) }}</option>
          <option :value="50">{{ $t('dbadmin.mongo.perPage', { n: 50 }) }}</option>
          <option :value="100">{{ $t('dbadmin.mongo.perPage', { n: 100 }) }}</option>
        </select>
        <FaButton size="sm" :loading="loading" @click="page && (page.skip = 0, load())">{{ $t('dbadmin.mongo.query') }}</FaButton>
        <span class="flex-1" />
        <div class="flex items-center rounded-md border p-0.5">
          <button
            type="button"
            class="rounded px-1.5 py-0.5 text-xs"
            :class="viewMode === 'list' ? 'bg-primary/10 font-medium text-primary' : 'text-muted-foreground'"
            :title="$t('dbadmin.mongo.listView')"
            @click="viewMode = 'list'"
          >☰</button>
          <button
            type="button"
            class="rounded px-1.5 py-0.5 text-xs"
            :class="viewMode === 'table' ? 'bg-primary/10 font-medium text-primary' : 'text-muted-foreground'"
            :title="$t('dbadmin.mongo.tableView')"
            @click="viewMode = 'table'"
          >⊞</button>
        </div>
        <FaButton size="sm" @click="exportDocs">{{ $t('dbadmin.mongo.exportJson') }}</FaButton>
        <FaButton size="sm" variant="outline" @click="openImport">{{ $t('dbadmin.mongo.importJson') }}</FaButton>
        <FaButton size="sm" type="primary" @click="openEditor('insert')">{{ $t('dbadmin.mongo.insertDoc') }}</FaButton>
      </div>

      <div v-if="error" class="shrink-0 rounded-md border border-red-300 bg-red-50 p-2 text-xs text-red-600 dark:bg-red-950/30">
        {{ error }}
      </div>

      <!-- 列表视图 -->
      <div v-if="viewMode === 'list'" class="min-h-0 flex-1 space-y-2 overflow-auto">
        <div v-for="(docStr, i) in page?.docs ?? []" :key="i" class="rounded-md border">
          <div class="flex items-center gap-2 border-b bg-muted/40 px-3 py-1.5 text-xs text-muted-foreground">
            <span class="truncate font-mono" :title="$t('dbadmin.mongo.editFullHint')">{{ docId(docStr) }}</span>
            <span class="flex-1" />
            <FaButton variant="ghost" size="icon-sm" :title="$t('dbadmin.mongo.cloneHint')" @click="openEditor('clone', docStr)">
              <FaIcon name="i-lucide:copy" class="text-sm" />
            </FaButton>
            <FaButton variant="ghost" size="icon-sm" :title="$t('dbadmin.mongo.editDoc')" @click="openEditor('edit', docStr)">
              <FaIcon name="i-lucide:pencil" class="text-sm" />
            </FaButton>
            <FaButton variant="ghost" size="icon-sm" :title="$t('dbadmin.mongo.deleteDoc')" @click="delDoc(docStr)">
              <FaIcon name="i-lucide:trash-2" class="text-sm" />
            </FaButton>
          </div>
          <div class="max-h-64 overflow-auto p-3 font-mono text-xs leading-relaxed">
            <JsonNode name="doc" :value="toDisplayDoc(docStr)" root :expand-level="3" />
          </div>
        </div>
        <div v-if="!loading && !page?.docs.length" class="py-8 text-center text-sm text-muted-foreground">
          {{ $t('dbadmin.mongo.noDocs') }}
        </div>
      </div>

      <!-- 表格视图 -->
      <div v-else class="min-h-0 flex-1 overflow-auto rounded-md border">
        <table class="w-max text-xs">
          <thead class="sticky top-0 z-10 bg-muted/80 text-left backdrop-blur">
            <tr>
              <th class="w-10 border-b px-2 py-1.5 text-muted-foreground">#</th>
              <th v-for="c in tableCols" :key="c" class="whitespace-nowrap border-b px-2 py-1.5 font-mono">
                {{ c }}
              </th>
              <th class="w-20 border-b px-2 py-1.5 text-muted-foreground">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td :colspan="tableCols.length + 2" class="px-3 py-6 text-center text-muted-foreground">{{ $t('common.loading') }}</td>
            </tr>
            <tr v-else-if="!page?.docs.length">
              <td :colspan="tableCols.length + 2" class="px-3 py-6 text-center text-muted-foreground">{{ $t('dbadmin.mongo.noDocs') }}</td>
            </tr>
            <tr v-for="(docStr, i) in page?.docs ?? []" :key="i" class="group border-t hover:bg-accent/30">
              <td class="px-2 py-1 tabular-nums text-muted-foreground">{{ skip + i + 1 }}</td>
              <td v-for="c in tableCols" :key="c" class="max-w-56 truncate border-t px-2 py-1 font-mono" :title="c === '_id' ? docId(docStr) : ''">
                {{ cellSummary(cellValue(docStr, c)) }}
              </td>
              <td class="whitespace-nowrap border-t px-1 py-1">
                <button type="button" class="rounded p-1 text-muted-foreground hover:text-primary" :title="$t('common.edit')" @click="openEditor('edit', docStr)">
                  <FaIcon name="i-lucide:pencil" class="text-xs" />
                </button>
                <button type="button" class="rounded p-1 text-muted-foreground hover:text-primary" :title="$t('dbadmin.mongo.clone')" @click="openEditor('clone', docStr)">
                  <FaIcon name="i-lucide:copy" class="text-xs" />
                </button>
                <button type="button" class="rounded p-1 text-muted-foreground hover:text-red-500" :title="$t('common.delete')" @click="delDoc(docStr)">
                  <FaIcon name="i-lucide:trash-2" class="text-xs" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div class="flex items-center gap-2 text-xs text-muted-foreground">
        <span v-if="page">{{ $t('dbadmin.mongo.pageInfo', { total: page.total, cur: curPageNo, pages: totalPages }) }}</span>
        <button type="button" class="rounded border px-1.5 py-0.5 hover:bg-accent/50 disabled:opacity-40" :disabled="curPageNo <= 1" @click="gotoPage(curPageNo - 1)">{{ $t('dbadmin.prevPage') }}</button>
        <button type="button" class="rounded border px-1.5 py-0.5 hover:bg-accent/50 disabled:opacity-40" :disabled="curPageNo >= totalPages" @click="gotoPage(curPageNo + 1)">{{ $t('dbadmin.nextPage') }}</button>
      </div>
    </template>

    <!-- ============ Aggregations ============ -->
    <template v-else-if="tab === 'agg'">
      <div class="flex min-h-0 flex-1 gap-3">
        <!-- 管道编辑 -->
        <div class="flex w-[420px] shrink-0 flex-col gap-2 overflow-auto rounded-md border p-2">
          <div class="flex items-center text-xs text-muted-foreground">
            {{ $t('dbadmin.mongo.pipelineHint', { n: MAX_DOCS }) }}
          </div>
          <div v-for="(_, i) in stages" :key="i" class="rounded-md border p-1.5">
            <div class="mb-1 flex items-center gap-1 text-xs text-muted-foreground">
              <span class="rounded bg-muted px-1.5 py-0.5 font-mono">stage {{ i + 1 }}</span>
              <span class="flex-1" />
              <button type="button" class="rounded p-0.5 hover:text-red-500" :title="$t('dbadmin.mongo.deleteStage')" @click="removeStage(i)">
                <YdMorphIcon name="trash" :size="12" />
              </button>
            </div>
            <textarea
              v-model="stages[i]"
              class="h-20 w-full resize-y rounded border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
              spellcheck="false"
              placeholder='{ "$match": { "age": { "$gt": 18 } } }'
            />
          </div>
          <div class="flex items-center gap-2">
            <FaButton size="sm" variant="outline" @click="addStage">
              <YdMorphIcon name="plus" :size="12" /> {{ $t('dbadmin.mongo.addStage') }}
            </FaButton>
            <FaButton size="sm" type="primary" :loading="aggLoading" :disabled="!collection" @click="runAggregate">{{ $t('dbadmin.mongo.runPipeline') }}</FaButton>
          </div>
        </div>

        <!-- 结果 -->
        <div class="flex min-h-0 flex-1 flex-col gap-2">
          <div v-if="aggError" class="shrink-0 rounded-md border border-red-300 bg-red-50 p-2 text-xs text-red-600 dark:bg-red-950/30">
            {{ aggError }}
          </div>
          <div v-if="aggResult" class="shrink-0 text-xs text-muted-foreground">
            {{ $t('dbadmin.mongo.resultCount', { n: aggResult.total }) }}
          </div>
          <div class="min-h-0 flex-1 space-y-2 overflow-auto">
            <div v-for="(docStr, i) in aggResult?.docs ?? []" :key="i" class="rounded-md border p-2 font-mono text-xs leading-relaxed">
              <div class="max-h-48 overflow-auto">
                <JsonNode name="doc" :value="toDisplayDoc(docStr)" root :expand-level="2" />
              </div>
            </div>
            <div v-if="!aggLoading && !aggResult" class="flex size-full items-center justify-center text-sm text-muted-foreground">
              {{ $t('dbadmin.mongo.aggPlaceholder') }}
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- ============ Indexes ============ -->
    <template v-else>
      <div class="flex items-center gap-2">
        <FaButton size="sm" type="primary" @click="idxOpen = true">{{ $t('dbadmin.mongo.createIndex') }}</FaButton>
        <FaButton size="sm" variant="outline" :loading="idxLoading" @click="loadIndexes">{{ $t('common.refresh') }}</FaButton>
      </div>
      <div class="min-h-0 flex-1 overflow-auto rounded-md border">
        <table class="w-full text-xs">
          <thead class="bg-muted/50 text-left text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('dbadmin.mongo.colNameDef') }}</th>
              <th class="px-3 py-2">{{ $t('dbadmin.common.uniqueLabel') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.size') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('dbadmin.mongo.colAccessCount') }}</th>
              <th class="px-3 py-2">{{ $t('dbadmin.mongo.colLastAccess') }}</th>
              <th class="w-14 px-3 py-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="idx in indexes" :key="idx.name" class="border-t hover:bg-accent/30">
              <td class="px-3 py-1.5">
                <div class="font-mono">{{ idx.name }}</div>
                <div class="font-mono text-muted-foreground">{{ keysText(idx) }}</div>
              </td>
              <td class="px-3 py-1.5">
                <span v-if="idx.unique" class="rounded bg-emerald-500/15 px-1.5 py-0.5 text-[10px] text-emerald-600">UNIQUE</span>
                <span v-else-if="idx.name === '_id_'" class="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">{{ $t('dbadmin.mongo.system') }}</span>
              </td>
              <td class="px-3 py-1.5 text-right tabular-nums">{{ idx.sizeMb > 0 ? `${idx.sizeMb.toFixed(2)} MB` : '—' }}</td>
              <td class="px-3 py-1.5 text-right tabular-nums">{{ idx.accesses }}</td>
              <td class="px-3 py-1.5 font-mono text-muted-foreground">{{ idx.accessedAt || '—' }}</td>
              <td class="px-3 py-1.5 text-center">
                <button
                  v-if="idx.name !== '_id_'"
                  type="button"
                  class="rounded p-1 text-muted-foreground hover:text-red-500"
                  :title="$t('dbadmin.common.indexDeleteTitle')"
                  @click="dropIndex(idx.name)"
                >
                  <YdMorphIcon name="trash" :size="13" />
                </button>
              </td>
            </tr>
            <tr v-if="!indexes.length && !idxLoading">
              <td colspan="6" class="px-3 py-6 text-center text-muted-foreground">{{ $t('dbadmin.common.noIndexes') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 创建索引弹窗 -->
      <FaModal v-model="idxOpen" :destroy-on-close="true" :title="$t('dbadmin.mongo.createIndex')" :width="480">
        <div class="space-y-3 p-1 text-sm">
          <div class="flex items-center gap-2">
            <label class="w-24 text-right text-xs">{{ $t('dbadmin.common.indexName') }}</label>
            <input
              v-model="idxForm.name"
              class="flex-1 rounded border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
              :placeholder="$t('dbadmin.common.autoGenerate')"
            >
          </div>
          <div class="flex items-center gap-2">
            <label class="w-24 text-right text-xs">{{ $t('dbadmin.mongo.keysLabel') }}</label>
            <input
              v-model="idxForm.keysText"
              class="flex-1 rounded border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
              placeholder="name: 1, age: -1"
            >
          </div>
          <div class="flex items-center gap-2">
            <label class="w-24 text-right text-xs">{{ $t('dbadmin.common.uniqueIndex') }}</label>
            <FaSwitch v-model="idxForm.unique" />
          </div>
        </div>
        <template #footer>
          <FaButton size="sm" @click="idxOpen = false">{{ $t('common.cancel') }}</FaButton>
          <FaButton type="primary" size="sm" @click="submitIndex">{{ $t('common.create') }}</FaButton>
        </template>
      </FaModal>
    </template>

    <!-- Compass JSON 导入弹窗 -->
    <FaModal v-model="importOpen" :title="$t('dbadmin.mongo.importTitle')" :width="520">
      <div class="space-y-3 p-1 text-sm">
        <div class="text-xs text-muted-foreground">
          {{ $t('dbadmin.mongo.targetLabel') }}<span class="font-mono text-foreground">{{ db }}.{{ collection }}</span> {{ $t('dbadmin.mongo.importHint') }}
        </div>
        <input
          type="file"
          accept=".json,.jsonl,.ndjson"
          class="w-full rounded border border-input bg-background px-2 py-1.5 text-xs"
          @change="(e: any) => { importFile = e.target.files?.[0] ?? null }"
        >
        <p class="text-xs text-muted-foreground">{{ $t('dbadmin.mongo.importFileHint') }}</p>
      </div>
      <template #footer>
        <FaButton size="sm" @click="importOpen = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton type="primary" size="sm" :loading="importing" @click="submitImport">{{ $t('dbadmin.mongo.importSubmit') }}</FaButton>
      </template>
    </FaModal>

    <!-- 文档编辑弹窗 -->
    <FaModal v-model="editorOpen" :destroy-on-close="true" :title="editorMode === 'insert' ? $t('dbadmin.mongo.insertDoc') : editorMode === 'clone' ? $t('dbadmin.mongo.cloneDoc') : $t('dbadmin.mongo.editDoc')" :width="640">
      <div class="h-80 overflow-hidden rounded-md border">
        <YdCodeEditor v-if="docModel" :model="docModel" :minimap="false" />
      </div>
      <p class="mt-2 text-xs text-muted-foreground">
        {{ $t('dbadmin.mongo.docEditorHint') }}
      </p>
      <template #footer>
        <FaButton size="sm" @click="editorOpen = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton type="primary" size="sm" @click="submitDoc">{{ $t('common.save') }}</FaButton>
      </template>
    </FaModal>

    <!-- 集合创建/重命名弹窗 -->
    <FaModal v-model="collOpen" :destroy-on-close="true" :title="collAction === 'create' ? $t('dbadmin.mongo.createCollectionTitle') : $t('dbadmin.mongo.renameCollectionTitle')" :width="420">
      <div class="flex items-center gap-2 p-1">
        <label class="w-20 text-right text-xs">{{ $t('dbadmin.mongo.collName') }}</label>
        <input
          v-model="collName"
          class="flex-1 rounded border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
          @keydown.enter="submitColl"
        >
      </div>
      <template #footer>
        <FaButton size="sm" @click="collOpen = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton type="primary" size="sm" @click="submitColl">{{ $t('dbadmin.mongo.ok') }}</FaButton>
      </template>
    </FaModal>
  </div>
</template>
