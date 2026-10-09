<script setup lang="ts">
import type { DbColumnInfo, DbPagedResult, DbQueryResult } from '@/api/modules/dbadmin'
import apiDBA from '@/api/modules/dbadmin'
import { i18n } from '@/locales'
import { useQueryHistory } from '../composables/useQueryHistory'
import DataGrid from './DataGrid.vue'
import SqlEditor from './SqlEditor.vue'
import TableMeta from './TableMeta.vue'

// SQL 工作台（MySQL/PG）：SQL 编辑器 + 只读结果网格 ⇄ 表浏览（可编辑）+ 结构面板。
const props = defineProps<{
  instanceId: number
  db: string
  schema: string
  table: string
}>()

const toast = useFaToast()
const mode = ref<'sql' | 'browse'>('sql')

const sqlEditorRef = useTemplateRef<InstanceType<typeof SqlEditor>>('sqlEditor')
const treeTables = ref<string[]>([]) // 补全用表名（由壳注入或本组件维护）

// ---- 查询历史 ----
const history = useQueryHistory(() => props.instanceId)
const histOpen = ref(false)

// ---- SQL 查询 ----
const queryResult = ref<DbQueryResult>()
const queryError = ref('')
const querying = ref(false)

async function runQuery() {
  const sql = sqlEditorRef.value?.getValue().trim()
  if (!sql || !props.db) {
    toast.warning(i18n.global.t('dbadmin.workbench.selectDbAndQuery'))
    return
  }
  queryError.value = ''
  querying.value = true
  try {
    queryResult.value = await apiDBA.query(props.instanceId, props.db, sql)
    history.push(sql)
  }
  catch (e: any) {
    queryError.value = e?.message || i18n.global.t('dbadmin.common.queryFailed')
    queryResult.value = undefined
  }
  finally {
    querying.value = false
  }
}

function recall(item: { sql: string }) {
  sqlEditorRef.value?.setValue(item.sql)
  histOpen.value = false
}

// ---- SQL 文件导入（恢复通道执行） ----
const importOpen = ref(false)
const importMode = ref<'upload' | 'path'>('upload')
const importFile = ref<File | null>(null)
const importPath = ref('')
const importing = ref(false)
const importResult = ref('')

function openImport() {
  importFile.value = null
  importPath.value = ''
  importResult.value = ''
  importOpen.value = true
}

async function submitImport() {
  if (!props.db) {
    toast.warning(i18n.global.t('dbadmin.import.selectDbFirst'))
    return
  }
  let content = ''
  let srcPath = ''
  if (importMode.value === 'upload') {
    if (!importFile.value) {
      toast.warning(i18n.global.t('dbadmin.import.chooseFile'))
      return
    }
    if (/\.gz$/.test(importFile.value.name)) {
      toast.warning(i18n.global.t('dbadmin.import.gzUnsupported'))
      return
    }
    content = await importFile.value.text()
  }
  else {
    srcPath = importPath.value.trim()
    if (!srcPath) {
      toast.warning(i18n.global.t('dbadmin.import.pathRequired'))
      return
    }
  }
  importing.value = true
  try {
    await apiDBA.importSQL(props.instanceId, { db: props.db, content: content || undefined, srcPath: srcPath || undefined })
    importResult.value = i18n.global.t('dbadmin.import.done')
    toast.success(i18n.global.t('dbadmin.import.importedTo', { db: props.db }))
    importOpen.value = false
    if (mode.value === 'browse') {
      loadBrowse(0)
    }
  }
  catch (e: any) {
    importResult.value = e?.message || i18n.global.t('dbadmin.common.importFailed')
    toast.error(i18n.global.t('dbadmin.common.importFailed'), { description: importResult.value })
  }
  finally {
    importing.value = false
  }
}

// ---- 表浏览（真分页 + 编辑） ----
const browse = ref<DbPagedResult>()
const browseLoading = ref(false)
const browseColumnsMeta = ref<DbColumnInfo[]>([])
const browsePK = ref<string[]>([])
const sortCol = ref('')
const sortDir = ref<'asc' | 'desc'>('asc')
const browseError = ref('')

const LIMIT = 200

async function loadBrowse(offset = 0) {
  if (!props.table || !props.db) {
    browse.value = undefined
    return
  }
  browseLoading.value = true
  browseError.value = ''
  try {
    browse.value = await apiDBA.browse(props.instanceId, {
      db: props.db, schema: props.schema || undefined, table: props.table,
      sortCol: sortCol.value || undefined, sortDir: sortDir.value,
      offset, limit: LIMIT,
    })
  }
  catch (e: any) {
    browseError.value = e?.message || i18n.global.t('dbadmin.workbench.loadFailed')
    browse.value = undefined
  }
  finally {
    browseLoading.value = false
  }
}

function onSort(col: string) {
  if (sortCol.value === col) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  }
  else {
    sortCol.value = col
    sortDir.value = 'asc'
  }
  loadBrowse(0)
}

async function cellSave(pk: Record<string, unknown>, values: Record<string, unknown>) {
  try {
    await apiDBA.rowUpdate(props.instanceId, { db: props.db, schema: props.schema || undefined, table: props.table, pk, values })
    toast.success(i18n.global.t('dbadmin.common.saved'))
    loadBrowse(browse.value?.offset ?? 0)
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.saveFailed'), { description: e?.message })
  }
}

async function rowDelete(pk: Record<string, unknown>) {
  try {
    await apiDBA.rowDelete(props.instanceId, { db: props.db, schema: props.schema || undefined, table: props.table, pk })
    toast.success(i18n.global.t('dbadmin.common.deleted'))
    loadBrowse(browse.value?.offset ?? 0)
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.deleteFailed'), { description: e?.message })
  }
}

async function rowInsert(values: Record<string, unknown>) {
  try {
    await apiDBA.rowInsert(props.instanceId, { db: props.db, schema: props.schema || undefined, table: props.table, values })
    toast.success(i18n.global.t('dbadmin.common.inserted'))
    loadBrowse(0)
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.insertFailed'), { description: e?.message })
  }
}

function onColumnsLoaded(cols: DbColumnInfo[], pk: string[]) {
  browseColumnsMeta.value = cols
  browsePK.value = pk
}

watch(() => [props.instanceId, props.db, props.schema, props.table] as const, () => {
  sortCol.value = ''
  sortDir.value = 'asc'
  browse.value = undefined
  if (mode.value === 'browse') {
    loadBrowse(0)
  }
})

watch(mode, (m) => {
  if (m === 'browse') {
    loadBrowse(0)
  }
})

defineExpose({
  /** 树选中表后同步补全提示与浏览模式 */
  setHints(tables: string[], views: string[], columns: string[]) {
    sqlEditorRef.value?.setHints(tables, views, columns)
    treeTables.value = tables
  },
  refreshBrowse: loadBrowse,
})
</script>

<template>
  <div class="flex size-full min-h-0 flex-col gap-2">
    <div class="flex items-center gap-1">
      <button
        type="button"
        class="rounded-md px-2.5 py-1 text-xs transition-colors"
        :class="mode === 'sql' ? 'bg-primary/10 font-medium text-primary' : 'text-muted-foreground hover:bg-accent/50'"
        @click="mode = 'sql'"
      >
        {{ $t('dbadmin.workbench.sqlQuery') }}
      </button>
      <button
        type="button"
        class="rounded-md px-2.5 py-1 text-xs transition-colors"
        :class="mode === 'browse' ? 'bg-primary/10 font-medium text-primary' : 'text-muted-foreground hover:bg-accent/50'"
        :disabled="!table"
        @click="mode = 'browse'"
      >
        {{ table ? $t('dbadmin.workbench.browseTable', { table }) : $t('dbadmin.workbench.browseData') }}
      </button>
      <span class="flex-1" />
      <FaButton v-if="mode === 'sql'" size="sm" variant="outline" @click="openImport">{{ $t('dbadmin.workbench.importSql') }}</FaButton>
      <span v-if="mode === 'sql'" class="text-xs text-muted-foreground">
        {{ db }}{{ schema ? `.${schema}` : '' }} {{ $t('dbadmin.workbench.sqlHint') }}
      </span>
    </div>

    <!-- SQL 模式 -->
    <template v-if="mode === 'sql'">
      <div class="relative h-56 shrink-0 rounded-md border">
        <SqlEditor ref="sqlEditor" :read-only="false" @run="runQuery" />
        <!-- 历史浮层 -->
        <button
          type="button"
          class="absolute right-2 top-2 z-10 flex items-center gap-1 rounded border bg-background px-2 py-0.5 text-xs text-muted-foreground hover:bg-accent/50"
          @click="histOpen = !histOpen"
        >
          <YdMorphIcon name="clock-3" :size="12" /> {{ $t('dbadmin.workbench.history', { n: history.list.value.length }) }}
        </button>
        <div
          v-if="histOpen"
          class="absolute right-2 top-8 z-20 max-h-72 w-96 overflow-auto rounded-md border bg-background p-1 shadow-lg"
        >
          <button
            v-for="(item, i) in history.list.value"
            :key="i"
            type="button"
            class="block w-full truncate rounded px-2 py-1 text-left font-mono text-xs hover:bg-accent/50"
            :title="item.sql"
            @click="recall(item)"
          >
            {{ item.sql }}
          </button>
          <div v-if="!history.list.value.length" class="px-2 py-3 text-center text-xs text-muted-foreground">{{ $t('dbadmin.workbench.noHistory') }}</div>
          <div class="border-t p-1 text-right">
            <button type="button" class="text-xs text-muted-foreground hover:text-red-500" @click="history.clear()">{{ $t('dbadmin.workbench.clearHistory') }}</button>
          </div>
        </div>
      </div>
      <div v-if="queryError" class="shrink-0 rounded-md border border-red-300 bg-red-50 p-2 text-xs text-red-600 dark:bg-red-950/30">
        {{ queryError }}
      </div>
      <div class="min-h-0 flex-1">
        <DataGrid
          :columns="queryResult?.columns ?? []"
          :rows="queryResult?.rows ?? []"
          :loading="querying"
          :elapsed="queryResult?.elapsed"
        />
      </div>
    </template>


    <!-- 浏览模式 -->
    <template v-else>
      <div v-if="browseError" class="shrink-0 rounded-md border border-red-300 bg-red-50 p-2 text-xs text-red-600 dark:bg-red-950/30">
        {{ browseError }}
      </div>
      <div v-if="!table" class="flex flex-1 items-center justify-center text-sm text-muted-foreground">
        {{ $t('dbadmin.workbench.selectTableHint') }}
      </div>
      <div v-else class="min-h-0 flex-1">
        <DataGrid
          :columns="browse?.columns ?? []"
          :rows="browse?.rows ?? []"
          :total="browse?.total"
          :offset="browse?.offset"
          :limit="browse?.limit"
          :loading="browseLoading"
          :elapsed="browse?.elapsed"
          :editable="true"
          :pk-columns="browsePK"
          :column-meta="browseColumnsMeta"
          @page="loadBrowse"
          @sort="onSort"
          @cell-save="cellSave"
          @row-delete="rowDelete"
          @row-insert="rowInsert"
        />
      </div>
      <div v-if="table" class="h-64 shrink-0">
        <TableMeta
          :instance-id="instanceId"
          :db="db"
          :schema="schema"
          :table="table"
          @columns-loaded="onColumnsLoaded"
        />
      </div>
    </template>
    <!-- 导入 SQL 弹窗 -->
    <FaModal v-model="importOpen" :title="$t('dbadmin.import.title')" :width="520">
      <div class="space-y-3 p-1 text-sm">
        <div class="text-xs text-muted-foreground">
          {{ $t('dbadmin.import.targetDb') }}<span class="font-mono text-foreground">{{ db }}</span> {{ $t('dbadmin.import.hint') }}
        </div>
        <FaTabs
          v-model="importMode" :list="[
            { label: $t('dbadmin.import.upload'), value: 'upload' },
            { label: $t('dbadmin.import.path'), value: 'path' },
          ]"
        />
        <template v-if="importMode === 'upload'">
          <input
            type="file"
            accept=".sql,.sql.gz,.gz"
            class="w-full rounded border border-input bg-background px-2 py-1.5 text-xs"
            @change="(e: any) => { importFile = e.target.files?.[0] ?? null }"
          >
          <p class="text-xs text-muted-foreground">{{ $t('dbadmin.import.uploadHint') }}</p>
        </template>
        <template v-else>
          <input
            v-model="importPath"
            class="w-full rounded border border-input bg-background px-2 py-1.5 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
            placeholder="/opt/ypanel/backups/mysql/xx/backup.sql"
          >
          <p class="text-xs text-muted-foreground">{{ $t('dbadmin.import.pathHint') }}</p>
        </template>
        <div v-if="importResult" class="rounded bg-muted/50 p-2 text-xs">{{ importResult }}</div>
      </div>
      <template #footer>
        <FaButton size="sm" @click="importOpen = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton type="primary" size="sm" :loading="importing" @click="submitImport">{{ $t('dbadmin.import.submit') }}</FaButton>
      </template>
    </FaModal>
  </div>
</template>
