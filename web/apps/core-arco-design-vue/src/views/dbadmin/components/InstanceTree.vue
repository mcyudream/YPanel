<script setup lang="ts">
import type { DbAdminInstance, DbDatabaseInfo, DbTableInfo } from '@/api/modules/dbadmin'
import apiDBA from '@/api/modules/dbadmin'
import { i18n } from '@/locales'

// 左侧树：实例 → 库 →（PG 多 schema）→ 表/集合，懒加载 + 搜索过滤 + 延迟徽标。
// Redis 的"库"（dbN）没有表概念，渲染为纯叶子（点击仅选中，不请求表清单）。
export interface TreeSelection {
  instanceId: number
  type: string
  db: string
  schema: string
  table: string
  kind: string
  rows: number
}

const props = defineProps<{
  selection: TreeSelection
}>()

const emit = defineEmits<{
  select: [sel: TreeSelection]
}>()

const TYPE_DOT: Record<string, string> = {
  mysql: 'bg-sky-500',
  postgres: 'bg-teal-500',
  redis: 'bg-rose-500',
  mongo: 'bg-emerald-500',
}

const instances = ref<DbAdminInstance[]>([])
const latency = ref<Record<number, number>>({})
const expandedInst = ref<number>(0)
const expandedType = ref('')
const databases = ref<DbDatabaseInfo[]>([])
const expandedDb = ref('')
const schemas = ref<string[]>([])
const expandedSchema = ref('')
const tables = ref<DbTableInfo[]>([])
const loading = ref('')
const filter = ref('')

async function loadInstances() {
  instances.value = await apiDBA.instances()
  // 并发测延迟（8s 超时由 api 层控制），失败标记 -1
  instances.value.forEach(async (inst) => {
    try {
      latency.value[inst.id] = await apiDBA.ping(inst.id)
    }
    catch {
      latency.value[inst.id] = -1
    }
  })
  const first = instances.value.find(i => i.id === props.selection.instanceId) ?? instances.value[0]
  if (first) {
    await toggleInstance(first, props.selection.instanceId === first.id)
  }
}

function matchFilter(...parts: string[]) {
  const f = filter.value.trim().toLowerCase()
  if (!f) {
    return true
  }
  return parts.some(p => p.toLowerCase().includes(f))
}

async function toggleInstance(inst: DbAdminInstance, forceSelect = false) {
  if (!inst.running && !forceSelect) {
    useFaToast().warning(i18n.global.t('dbadmin.tree.notRunning'))
    return
  }
  if (expandedInst.value !== inst.id) {
    expandedInst.value = inst.id
    expandedType.value = inst.type
    expandedDb.value = ''
    expandedSchema.value = ''
    tables.value = []
    loading.value = `inst-${inst.id}`
    try {
      databases.value = await apiDBA.databases(inst.id)
    }
    catch (e: any) {
      useFaToast().error(i18n.global.t('dbadmin.tree.dbListFailed'), { description: e?.message })
      databases.value = []
    }
    finally {
      loading.value = ''
    }
  }
  emitSelect(inst.id, inst.type, '', '', '', '', 0)
}

async function toggleDb(inst: DbAdminInstance, dbName: string) {
  // Redis 库是纯叶子：只选中（右侧进入 key 浏览工作台），不展开、不请求表清单
  if (inst.type === 'redis') {
    emitSelect(inst.id, inst.type, dbName, '', '', '', 0)
    return
  }
  const isNew = expandedDb.value !== dbName || expandedInst.value !== inst.id
  expandedInst.value = inst.id
  expandedDb.value = dbName
  expandedSchema.value = ''
  tables.value = []
  if (isNew) {
    loading.value = `db-${dbName}`
    try {
      schemas.value = await apiDBA.schemas(inst.id, dbName)
    }
    catch {
      schemas.value = []
    }
    finally {
      loading.value = ''
    }
  }
  emitSelect(inst.id, inst.type, dbName, '', '', '', 0)
  if (!schemas.value.length) {
    await loadTables(inst, dbName, '')
  }
}

async function selectSchema(inst: DbAdminInstance, dbName: string, schema: string) {
  expandedSchema.value = schema
  await loadTables(inst, dbName, schema)
}

async function loadTables(inst: DbAdminInstance, dbName: string, schema: string) {
  loading.value = `tbl-${dbName}-${schema}`
  try {
    tables.value = await apiDBA.tables(inst.id, dbName, schema || undefined)
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dbadmin.tree.tableListFailed'), { description: e?.message })
    tables.value = []
  }
  finally {
    loading.value = ''
  }
}

async function selectTable(inst: DbAdminInstance, dbName: string, schema: string, t: DbTableInfo) {
  emitSelect(inst.id, inst.type, dbName, schema, t.name, t.kind ?? 'table', t.rows)
}

function emitSelect(instanceId: number, type: string, db: string, schema: string, table: string, kind: string, rows: number) {
  emit('select', { instanceId, type, db, schema, table, kind, rows })
}

function refreshTables() {
  const inst = instances.value.find(i => i.id === expandedInst.value)
  if (inst && expandedDb.value && inst.type !== 'redis') {
    loadTables(inst, expandedDb.value, expandedSchema.value)
  }
}

defineExpose({ refreshTables })

onMounted(loadInstances)
</script>

<template>
  <div class="flex size-full flex-col gap-2.5">
    <div class="relative">
      <input
        v-model="filter"
        class="w-full rounded-md border border-input bg-background px-2.5 py-1.5 text-xs outline-none focus:ring-1 focus:ring-primary"
        :placeholder="$t('dbadmin.tree.searchPlaceholder')"
      >
      <button
        type="button"
        class="absolute right-1.5 top-1.5 rounded p-0.5 text-muted-foreground hover:bg-accent/60"
        :title="$t('dbadmin.tree.refreshTables')"
        @click="refreshTables"
      >
        <YdMorphIcon name="refresh-cw" :size="13" />
      </button>
    </div>

    <div class="min-h-0 flex-1 overflow-auto pr-1">
      <div v-if="!instances.length" class="px-1 py-3 text-xs text-muted-foreground">
        {{ $t('dbadmin.tree.noInstances') }}
      </div>

      <div v-for="inst in instances" :key="inst.id" class="mb-1">
        <!-- 实例行 -->
        <button
          type="button"
          class="flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors"
          :class="selection.instanceId === inst.id && !selection.db
            ? 'bg-primary/10 font-medium text-primary'
            : 'hover:bg-accent/60'"
          @click="toggleInstance(inst)"
        >
          <YdMorphIcon :name="expandedInst === inst.id ? 'chevron-down' : 'chevron-right'" :size="13" class="shrink-0 text-muted-foreground" />
          <span class="size-2 shrink-0 rounded-full" :class="TYPE_DOT[inst.type] ?? 'bg-muted-foreground'" />
          <span class="truncate">{{ inst.name }}</span>
          <span
            v-if="latency[inst.id] !== undefined"
            class="ml-auto shrink-0 text-[10px] tabular-nums"
            :class="latency[inst.id] < 0 ? 'text-red-500' : latency[inst.id] > 300 ? 'text-amber-500' : 'text-muted-foreground'"
          >
            {{ latency[inst.id] < 0 ? $t('dbadmin.tree.offline') : `${latency[inst.id]}ms` }}
          </span>
        </button>

        <!-- 库层 -->
        <div v-if="expandedInst === inst.id" class="ml-[18px] border-l pl-1.5">
          <div v-if="loading === `inst-${inst.id}`" class="px-2 py-1 text-xs text-muted-foreground">{{ $t('common.loading') }}</div>
          <template v-for="d in databases.filter(d => matchFilter(d.name))" :key="d.name">
            <!-- 库行（Redis 为叶子，无展开箭头） -->
            <button
              type="button"
              class="flex w-full cursor-pointer items-center gap-1.5 rounded-md px-2 py-1 text-left text-xs transition-colors"
              :class="selection.instanceId === inst.id && selection.db === d.name && (!selection.table || inst.type === 'redis')
                ? 'bg-primary/10 font-medium text-primary'
                : 'text-foreground/90 hover:bg-accent/60'"
              @click="toggleDb(inst, d.name)"
            >
              <YdMorphIcon
                v-if="inst.type !== 'redis'"
                :name="expandedDb === d.name && expandedInst === inst.id ? 'chevron-down' : 'chevron-right'"
                :size="11"
                class="shrink-0 text-muted-foreground"
              />
              <span v-else class="w-[11px] shrink-0" />
              <span class="truncate font-mono">{{ d.name }}</span>
              <span class="ml-auto shrink-0 text-[10px] tabular-nums text-muted-foreground">
                {{ inst.type === 'redis' ? $t('dbadmin.tree.keysCount', { n: d.sizeMb }) : d.sizeMb > 0 ? `${d.sizeMb.toFixed(1)}M` : '' }}
              </span>
            </button>

            <!-- schema 层（PG） -->
            <div v-if="inst.type !== 'redis' && expandedDb === d.name && expandedInst === inst.id && schemas.length" class="ml-2.5 border-l pl-1.5">
              <template v-for="sc in schemas.filter(s => matchFilter(s, d.name))" :key="sc">
                <button
                  type="button"
                  class="flex w-full cursor-pointer items-center gap-1.5 rounded-md px-2 py-1 text-left text-xs text-muted-foreground transition-colors hover:bg-accent/60"
                  :class="{ 'bg-primary/10 !text-primary font-medium': selection.schema === sc && selection.db === d.name && !selection.table }"
                  @click="selectSchema(inst, d.name, sc)"
                >
                  <YdMorphIcon :name="expandedSchema === sc ? 'chevron-down' : 'chevron-right'" :size="11" class="shrink-0" />
                  <span class="truncate font-mono">{{ sc }}</span>
                </button>
                <div v-if="expandedSchema === sc" class="ml-2.5 border-l pl-1.5">
                  <button
                    v-for="t in tables.filter(t => matchFilter(t.name, d.name))"
                    :key="t.name"
                    type="button"
                    class="flex w-full cursor-pointer items-center gap-1.5 rounded-md px-2 py-1 text-left text-xs font-mono transition-colors hover:bg-accent/60"
                    :class="selection.instanceId === inst.id && selection.db === d.name && selection.table === t.name
                      ? 'bg-primary/10 font-medium text-primary'
                      : 'hover:bg-accent/60'"
                    :title="t.comment || t.name"
                    @click="selectTable(inst, d.name, sc, t)"
                  >
                    <YdMorphIcon :name="t.kind === 'view' ? 'eye' : 'table-2'" :size="11" class="shrink-0 text-muted-foreground" />
                    <span class="truncate">{{ t.name }}</span>
                    <span class="ml-auto shrink-0 text-[10px] tabular-nums text-muted-foreground">{{ t.rows }}</span>
                  </button>
                </div>
              </template>
            </div>

            <!-- 表层（无 schema） -->
            <div v-else-if="inst.type !== 'redis' && expandedDb === d.name && expandedInst === inst.id && !schemas.length" class="ml-2.5 border-l pl-1.5">
              <div v-if="loading === `tbl-${d.name}-`" class="px-2 py-1 text-xs text-muted-foreground">{{ $t('common.loading') }}</div>
              <button
                v-for="t in tables.filter(t => matchFilter(t.name, d.name))"
                :key="t.name"
                type="button"
                class="flex w-full cursor-pointer items-center gap-1.5 rounded-md px-2 py-1 text-left text-xs font-mono transition-colors hover:bg-accent/60"
                :class="selection.instanceId === inst.id && selection.db === d.name && selection.table === t.name
                  ? 'bg-primary/10 font-medium text-primary'
                  : 'hover:bg-accent/60'"
                :title="t.comment || t.name"
                @click="selectTable(inst, d.name, '', t)"
              >
                <YdMorphIcon :name="t.kind === 'view' ? 'eye' : 'table-2'" :size="11" class="shrink-0 text-muted-foreground" />
                <span class="truncate">{{ t.name }}</span>
                    <span class="ml-auto shrink-0 text-[10px] tabular-nums text-muted-foreground">{{ t.kind === 'collection' ? $t('dbadmin.tree.docCount', { n: t.rows }) : t.rows }}</span>
              </button>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>
