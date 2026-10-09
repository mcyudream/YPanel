<script setup lang="ts">
import type { DbColumnInfo, DbIndexInfo } from '@/api/modules/dbadmin'
import apiDBA from '@/api/modules/dbadmin'
import { i18n } from '@/locales'

// 表结构面板：字段 / 索引 / DDL 三视图 + 索引创建/删除（写路径后端审计）。
const props = defineProps<{
  instanceId: number
  db: string
  schema: string
  table: string
}>()

const emit = defineEmits<{
  columnsLoaded: [cols: DbColumnInfo[], pk: string[]]
}>()

const tab = ref<'columns' | 'indexes' | 'ddl'>('columns')
const columns = ref<DbColumnInfo[]>([])
const pk = ref<string[]>([])
const indexes = ref<DbIndexInfo[]>([])
const ddl = ref('')
const loading = ref(false)

async function load() {
  if (!props.table) {
    return
  }
  loading.value = true
  try {
    if (tab.value === 'columns') {
      columns.value = await apiDBA.columns(props.instanceId, props.db, props.table, props.schema || undefined)
      pk.value = await apiDBA.primaryKey(props.instanceId, props.db, props.table, props.schema || undefined)
      emit('columnsLoaded', columns.value, pk.value)
    }
    else if (tab.value === 'indexes') {
      indexes.value = await apiDBA.indexes(props.instanceId, props.db, props.table, props.schema || undefined)
    }
    else {
      ddl.value = await apiDBA.ddl(props.instanceId, props.db, props.table, props.schema || undefined)
    }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dbadmin.meta.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

watch([tab, () => props.table, () => props.db, () => props.schema], load, { immediate: true })

// ---- 索引操作 ----
const toast = useFaToast()
const modal = useFaModal()
const idxOpen = ref(false)
const idxForm = ref({ name: '', columnsText: '', unique: false })

function openIndexCreate() {
  idxForm.value = { name: '', columnsText: '', unique: false }
  idxOpen.value = true
}

async function submitIndex() {
  const cols = idxForm.value.columnsText.split(',').map(s => s.trim()).filter(Boolean)
  if (!cols.length) {
    toast.warning(i18n.global.t('dbadmin.meta.columnsRequired'))
    return
  }
  try {
    await apiDBA.indexCreate(props.instanceId, {
      db: props.db, table: props.table, schema: props.schema || undefined,
      name: idxForm.value.name || undefined, columns: cols, unique: idxForm.value.unique,
    })
    toast.success(i18n.global.t('dbadmin.common.indexCreated'))
    idxOpen.value = false
    load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.createFailed'), { description: e?.message })
  }
}

async function dropIndex(name: string) {
  if (name === 'PRIMARY') {
    toast.warning(i18n.global.t('dbadmin.meta.pkProtected'))
    return
  }
  const ok = await modal.confirm({ title: i18n.global.t('dbadmin.common.indexDeleteTitle'), content: i18n.global.t('dbadmin.common.indexDeleteConfirm', { name }) })
  if (!ok) {
    return
  }
  try {
    await apiDBA.indexDrop(props.instanceId, { db: props.db, table: props.table, schema: props.schema || undefined, name })
    toast.success(i18n.global.t('dbadmin.meta.indexDropped'))
    load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.deleteFailed'), { description: e?.message })
  }
}
</script>

<template>
  <div class="flex size-full min-h-0 flex-col">
    <div class="mb-2 flex items-center gap-1">
      <button
        v-for="t in [
          { k: 'columns', label: table ? $t('dbadmin.meta.columnsTable', { table }) : $t('dbadmin.meta.columnsLabel') },
          { k: 'indexes', label: $t('dbadmin.meta.indexes') },
          { k: 'ddl', label: 'DDL' },
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
      <button
        v-if="tab === 'indexes'"
        type="button"
        class="flex items-center gap-1 rounded border px-2 py-0.5 text-xs hover:bg-accent/50"
        @click="openIndexCreate"
      >
        <YdMorphIcon name="plus" :size="12" /> {{ $t('dbadmin.meta.createIndex') }}
      </button>
    </div>

    <div class="min-h-0 flex-1 overflow-auto rounded-md border">
      <div v-if="loading" class="p-3 text-xs text-muted-foreground">{{ $t('common.loading') }}</div>

      <!-- 字段 -->
      <table v-else-if="tab === 'columns'" class="w-full text-xs">
        <thead class="bg-muted/60 text-left">
          <tr>
            <th class="px-2 py-1.5 font-mono">{{ $t('dbadmin.meta.colField') }}</th>
            <th class="px-2 py-1.5 font-mono">{{ $t('common.type') }}</th>
            <th class="px-2 py-1.5">{{ $t('dbadmin.meta.colNullable') }}</th>
            <th class="px-2 py-1.5">{{ $t('dbadmin.meta.colKey') }}</th>
            <th class="px-2 py-1.5">{{ $t('dbadmin.meta.colDefault') }}</th>
            <th class="px-2 py-1.5">{{ $t('common.remark') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in columns" :key="c.name" class="border-t hover:bg-accent/30">
            <td class="px-2 py-1 font-mono">{{ c.name }}</td>
            <td class="px-2 py-1 font-mono text-muted-foreground">{{ c.dataType }}</td>
            <td class="px-2 py-1">{{ c.nullable ? 'YES' : 'NO' }}</td>
            <td class="px-2 py-1">
              <span v-if="c.key" class="rounded bg-amber-500/20 px-1 text-[10px] text-amber-600">{{ c.key }}</span>
            </td>
            <td class="max-w-40 truncate px-2 py-1 font-mono" :title="c.default ?? ''">{{ c.default ?? '' }}</td>
            <td class="max-w-48 truncate px-2 py-1" :title="c.comment">{{ c.comment }}</td>
          </tr>
          <tr v-if="!columns.length">
            <td colspan="6" class="px-3 py-4 text-center text-muted-foreground">{{ $t('dbadmin.meta.noColumns') }}</td>
          </tr>
        </tbody>
      </table>

      <!-- 索引 -->
      <table v-else-if="tab === 'indexes'" class="w-full text-xs">
        <thead class="bg-muted/60 text-left">
          <tr>
            <th class="px-2 py-1.5 font-mono">{{ $t('dbadmin.common.indexName') }}</th>
            <th class="px-2 py-1.5">{{ $t('dbadmin.meta.colColumns') }}</th>
            <th class="px-2 py-1.5">{{ $t('dbadmin.common.uniqueLabel') }}</th>
            <th class="w-10" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="idx in indexes" :key="idx.name" class="border-t hover:bg-accent/30">
            <td class="px-2 py-1 font-mono">
              {{ idx.name }}
              <span v-if="idx.primary" class="ml-1 rounded bg-amber-500/20 px-1 text-[10px] text-amber-600">PK</span>
            </td>
            <td class="px-2 py-1 font-mono text-muted-foreground">{{ idx.columns.join(', ') }}</td>
            <td class="px-2 py-1">{{ idx.unique ? '✓' : '' }}</td>
            <td class="px-1 text-center">
              <button
                type="button"
                class="rounded p-0.5 text-muted-foreground hover:text-red-500"
                :title="$t('dbadmin.common.indexDeleteTitle')"
                @click="dropIndex(idx.name)"
              >
                <YdMorphIcon name="trash" :size="13" />
              </button>
            </td>
          </tr>
          <tr v-if="!indexes.length">
            <td colspan="4" class="px-3 py-4 text-center text-muted-foreground">{{ $t('dbadmin.common.noIndexes') }}</td>
          </tr>
        </tbody>
      </table>

      <!-- DDL -->
      <pre v-else class="overflow-auto p-3 font-mono text-xs leading-relaxed">{{ ddl || $t('dbadmin.meta.noDdl') }}</pre>
    </div>

    <!-- 新建索引弹窗 -->
    <FaModal v-model="idxOpen" :destroy-on-close="true" :title="$t('dbadmin.meta.createIndex')" :width="460">
      <div class="space-y-3 p-1 text-sm">
        <div class="flex items-center gap-2">
          <label class="w-24 text-right text-xs">{{ $t('dbadmin.common.indexName') }}</label>
          <input
            v-model="idxForm.name"
            class="flex-1 rounded border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
            :placeholder="$t('dbadmin.meta.autoGenerateDdl')"
          >
        </div>
        <div class="flex items-center gap-2">
          <label class="w-24 text-right text-xs">{{ $t('dbadmin.meta.columnsComma') }}</label>
          <input
            v-model="idxForm.columnsText"
            class="flex-1 rounded border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
            placeholder="col1, col2"
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
  </div>
</template>
