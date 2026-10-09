<script setup lang="ts">
import type { DbAdminInstance, DbAuditItem } from '@/api/modules/dbadmin'
import apiDBA from '@/api/modules/dbadmin'
import InstanceTree, { type TreeSelection } from './components/InstanceTree.vue'
import MongoWorkbench from './components/MongoWorkbench.vue'
import RedisWorkbench from './components/RedisWorkbench.vue'
import SqlWorkbench from './components/SqlWorkbench.vue'
import { i18n, tr } from '@/locales'

// DB Admin v2：应库而宜的数据库管理台。
// 左侧实例→库→表树 + 右侧按库类型切换工作台（SQL / Redis / Mongo）。
defineOptions({
  name: 'DbAdminIndex',
})

const TYPE_LABEL: Record<string, string> = {
  mysql: 'MySQL', postgres: 'PostgreSQL', redis: 'Redis', mongo: 'MongoDB',
}

const instances = ref<DbAdminInstance[]>([])
const sel = ref<TreeSelection>({ instanceId: 0, type: '', db: '', schema: '', table: '', kind: '', rows: 0 })
const sqlWB = useTemplateRef<InstanceType<typeof SqlWorkbench>>('sqlWB')

const activeInst = computed(() => instances.value.find(i => i.id === sel.value.instanceId))

async function loadInstances() {
  instances.value = await apiDBA.instances()
}

function onSelect(s: TreeSelection) {
  sel.value = { ...s }
}

// 选中表后同步 SQL 补全（表名 + 该表字段）
async function syncHints() {
  const wb = sqlWB.value
  if (!wb || !sel.value.instanceId || !sel.value.db) {
    return
  }
  try {
    const tables = await apiDBA.tables(sel.value.instanceId, sel.value.db, sel.value.schema || undefined)
    const names = tables.filter(t => t.kind !== 'view').map(t => t.name)
    const views = tables.filter(t => t.kind === 'view').map(t => t.name)
    let cols: string[] = []
    if (sel.value.table && sel.value.type !== 'redis' && sel.value.type !== 'mongo') {
      cols = (await apiDBA.columns(sel.value.instanceId, sel.value.db, sel.value.table, sel.value.schema || undefined)).map(c => c.name)
    }
    wb.setHints(names, views, cols)
  }
  catch {
    // 补全非关键，失败静默
  }
}

watch(() => [sel.value.instanceId, sel.value.db, sel.value.schema, sel.value.table] as const, () => {
  syncHints()
})

// ---- 审计 ----
const auditsOpen = ref(false)
const audits = ref<DbAuditItem[]>([])
const auditsTotal = ref(0)
const auditsPage = ref(1)
const auditsLoading = ref(false)

async function loadAudits() {
  auditsLoading.value = true
  try {
    const r = await apiDBA.audits(undefined, auditsPage.value, 20)
    audits.value = r.items
    auditsTotal.value = r.total
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dbadmin.audit.loadFailed'), { description: e?.message })
  }
  finally {
    auditsLoading.value = false
  }
}

function openAudits() {
  auditsPage.value = 1
  auditsOpen.value = true
  loadAudits()
}

watch(auditsPage, () => {
  if (auditsOpen.value) {
    loadAudits()
  }
})

// 操作类型文案：动态枚举键，词条缺失时回落原值
function kindLabel(kind: string): string {
  return tr(`dbadmin.kind.${kind}`, kind)
}

// 操作类型徽标色：删/建/改/导/迁移/命令
function kindColor(kind: string): string {
  if (kind.includes('delete') || kind.includes('drop') || kind === 'redis_delete') {
    return 'bg-red-500/12 text-red-600'
  }
  if (kind.includes('insert') || kind.includes('create') || kind === 'backup_import' || kind === 'mongo_import') {
    return 'bg-emerald-500/12 text-emerald-600'
  }
  if (kind.includes('update') || kind === 'redis_write' || kind === 'redis_ttl' || kind === 'row_update') {
    return 'bg-amber-500/12 text-amber-600'
  }
  if (kind === 'sql_import') {
    return 'bg-sky-500/12 text-sky-600'
  }
  if (kind === 'migrate') {
    return 'bg-violet-500/12 text-violet-600'
  }
  return 'bg-slate-500/12 text-slate-600'
}

const PAD2 = (n: number) => String(n).padStart(2, '0')

function fmtTime(iso: string): string {
  const d = new Date(iso)
  return `${PAD2(d.getMonth() + 1)}-${PAD2(d.getDate())} ${PAD2(d.getHours())}:${PAD2(d.getMinutes())}:${PAD2(d.getSeconds())}`
}

onMounted(loadInstances)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="database" :size="24" />
          <span>{{ $t('dbadmin.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('dbadmin.description') }}</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="grid gap-3 lg:grid-cols-[300px_1fr]">
        <!-- 左树 -->
        <div class="h-[calc(100vh-260px)] rounded-lg border bg-background p-2">
          <InstanceTree :selection="sel" @select="onSelect" />
        </div>

        <!-- 右工作区 -->
        <div class="flex h-[calc(100vh-260px)] min-w-0 flex-col rounded-lg border bg-background">
          <div v-if="!sel.instanceId || !activeInst" class="flex flex-1 items-center justify-center text-sm text-muted-foreground">
            {{ $t('dbadmin.selectInstanceHint') }}
          </div>
          <template v-else>
            <div class="flex flex-wrap items-center gap-2 border-b px-3 py-2 text-xs text-muted-foreground">
              <span class="font-medium text-foreground">{{ activeInst.name }}</span>
              <span class="rounded bg-muted px-1.5 py-0.5">{{ TYPE_LABEL[activeInst.type] }}</span>
              <span v-if="sel.db" class="font-mono">{{ sel.db }}<template v-if="sel.schema">.{{ sel.schema }}</template></span>
              <span v-if="sel.table" class="font-mono">{{ sel.table }}</span>
              <span v-if="sel.rows" class="">{{ sel.kind === 'collection' ? $t('dbadmin.docsCount', { n: sel.rows }) : $t('dbadmin.rowsCount', { n: sel.rows }) }}</span>
              <span class="flex-1" />
              <FaButton variant="ghost" size="sm" @click="openAudits">
                <YdMorphIcon name="scroll-text" :size="13" /> {{ $t('dbadmin.audit.title') }}
              </FaButton>
            </div>
            <div class="min-h-0 flex-1 p-3">
              <SqlWorkbench
                v-if="activeInst.type === 'mysql' || activeInst.type === 'postgres'"
                ref="sqlWB"
                :instance-id="sel.instanceId"
                :db="sel.db"
                :schema="sel.schema"
                :table="sel.table"
              />
              <RedisWorkbench
                v-else-if="activeInst.type === 'redis'"
                :instance-id="sel.instanceId"
                :db="sel.db"
              />
              <MongoWorkbench
                v-else-if="activeInst.type === 'mongo'"
                :instance-id="sel.instanceId"
                :db="sel.db"
                :collection="sel.table"
              />
            </div>
          </template>
        </div>
      </div>

      <!-- 审计弹窗 -->
      <FaModal v-model="auditsOpen" :destroy-on-close="true" :title="$t('dbadmin.audit.title')" :width="640">
        <div class="max-h-[62vh] space-y-1 overflow-auto p-0.5">
          <div
            v-for="a in audits"
            :key="a.id"
            class="flex items-start gap-2 rounded-md px-2 py-1.5 text-xs transition-colors hover:bg-accent/40"
            :class="{ 'border-l-2 border-red-400': !a.success }"
            :title="`${$t('dbadmin.audit.operator', { name: a.username })} · ${a.success ? $t('common.success') : $t('common.failed')}`"
          >
            <span class="w-[104px] shrink-0 pt-0.5 font-mono tabular-nums text-muted-foreground">{{ fmtTime(a.createdAt) }}</span>
            <span class="shrink-0 rounded px-1.5 py-0.5 text-[10px] leading-4" :class="kindColor(a.kind)">{{ kindLabel(a.kind) }}</span>
            <div class="min-w-0 flex-1">
              <div class="flex items-baseline gap-1.5">
                <span class="shrink-0 font-medium">{{ a.instanceName }}</span>
                <span class="truncate font-mono text-muted-foreground" :title="a.target">{{ a.target }}</span>
              </div>
              <div class="truncate font-mono text-muted-foreground/90" :title="a.detail">{{ a.detail || '—' }}</div>
            </div>
            <span class="shrink-0 pt-0.5" :class="a.success ? 'text-emerald-600' : 'text-red-500'">{{ a.success ? '✓' : '✗' }}</span>
          </div>
          <div v-if="!audits.length && !auditsLoading" class="py-8 text-center text-muted-foreground">
            {{ $t('dbadmin.audit.empty') }}
          </div>
        </div>
        <template #footer>
          <div class="flex w-full items-center gap-2">
            <span class="text-xs text-muted-foreground">{{ $t('common.total', { n: auditsTotal }) }}</span>
            <span class="flex-1" />
            <FaButton size="sm" variant="outline" :disabled="auditsPage <= 1" @click="auditsPage--">{{ $t('dbadmin.prevPage') }}</FaButton>
            <span class="text-xs tabular-nums text-muted-foreground">{{ auditsPage }} / {{ Math.max(1, Math.ceil(auditsTotal / 20)) }}</span>
            <FaButton size="sm" variant="outline" :disabled="auditsPage * 20 >= auditsTotal" @click="auditsPage++">{{ $t('dbadmin.nextPage') }}</FaButton>
          </div>
        </template>
      </FaModal>
    </FaPageMain>
  </div>
</template>
