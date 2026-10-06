<script setup lang="ts">
import type { DbAdminInstance, DbAdminTable } from '@/api/modules/dbadmin'
import apiDBA from '@/api/modules/dbadmin'

defineOptions({
  name: 'DbAdminIndex',
})

const instances = ref<DbAdminInstance[]>([])
const activeId = ref<number>(0)
const activeType = ref<string>('')
const databases = ref<{ name: string, sizeMb: number }[]>([])
const activeDb = ref('')
const tables = ref<DbAdminTable[]>([])
const loadingSide = ref(false)

const sqlText = ref('')
const queryResult = ref<{ columns: string[], rows: unknown[][] }>()
const queryElapsed = ref('')
const queryError = ref('')
const querying = ref(false)

const TYPE_LABEL: Record<string, string> = {
  mysql: 'MySQL', postgres: 'PostgreSQL', redis: 'Redis', mongo: 'MongoDB',
}

async function loadInstances() {
  instances.value = await apiDBA.instances()
  if (instances.value.length && !activeId.value) {
    selectInstance(instances.value[0])
  }
}

async function selectInstance(inst: DbAdminInstance) {
  if (!inst.running) {
    useFaToast().warning('实例未运行，无法浏览')
    return
  }
  activeId.value = inst.id
  activeType.value = inst.type
  activeDb.value = ''
  tables.value = []
  queryResult.value = undefined
  loadingSide.value = true
  try {
    databases.value = await apiDBA.databases(inst.id)
  }
  catch (e: any) {
    useFaToast().error('读取库列表失败', { description: e?.message })
  }
  finally {
    loadingSide.value = false
  }
}

async function selectDb(name: string) {
  activeDb.value = name
  try {
    tables.value = await apiDBA.tables(activeId.value, name)
  }
  catch (e: any) {
    useFaToast().error('读取表清单失败', { description: e?.message })
  }
}

async function runQuery(sql?: string) {
  queryError.value = ''
  const text = (sql ?? sqlText.value).trim()
  if (!text || !activeDb.value) {
    useFaToast().warning('请选择库并输入查询语句')
    return
  }
  querying.value = true
  try {
    const r = await apiDBA.query(activeId.value, activeDb.value, text)
    queryResult.value = { columns: r.columns, rows: r.rows }
    queryElapsed.value = r.elapsed
  }
  catch (e: any) {
    queryError.value = e?.message || '查询失败'
    queryResult.value = undefined
  }
  finally {
    querying.value = false
  }
}

function previewTable(t: DbAdminTable) {
  sqlText.value = `SELECT * FROM ${t.name}`
  runQuery()
}

onMounted(loadInstances)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="database" :size="24" />
          <span>数据库管理台</span>
        </div>
      </template>
      <template #description>
        <span>库/表浏览与只读 SQL 查询器（db-admin 插件；写操作请使用"数据库"页结构化接口）</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <!-- 实例选择 -->
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <span class="text-sm text-muted-foreground">实例：</span>
        <button
          v-for="i in instances"
          :key="i.id"
          type="button"
          class="cursor-pointer rounded-md border px-3 py-1 text-sm transition-colors"
          :class="activeId === i.id ? 'border-primary bg-primary/10' : 'border-border hover:bg-accent/50'"
          @click="selectInstance(i)"
        >
          {{ i.name }}
          <span class="ml-1 text-xs text-muted-foreground">{{ TYPE_LABEL[i.type] }}{{ i.running ? '' : '（停止）' }}</span>
        </button>
        <span v-if="!instances.length" class="text-sm text-muted-foreground">暂无实例，请先在"数据库"页创建</span>
      </div>

      <div v-if="activeId" class="grid gap-4 lg:grid-cols-[280px_1fr]">
        <!-- 侧栏：库/表 -->
        <div class="rounded-lg border bg-background p-3">
          <div class="mb-2 text-sm font-medium">数据库</div>
          <div class="mb-3 flex flex-wrap gap-1">
            <button
              v-for="d in databases"
              :key="d.name"
              type="button"
              class="cursor-pointer rounded-md border px-2 py-0.5 font-mono text-xs transition-colors"
              :class="activeDb === d.name ? 'border-primary bg-primary/10' : 'border-border hover:bg-accent/50'"
              @click="selectDb(d.name)"
            >
              {{ d.name }}
            </button>
            <span v-if="!databases.length" class="text-xs text-muted-foreground">无</span>
          </div>
          <div class="mb-1 text-sm font-medium">表 / 集合 {{ activeDb ? `（${activeDb}）` : '' }}</div>
          <div class="max-h-80 overflow-auto">
            <button
              v-for="t in tables"
              :key="t.name"
              type="button"
              class="flex w-full cursor-pointer items-center justify-between rounded px-2 py-1 text-left font-mono text-xs transition-colors hover:bg-accent/50"
              @click="previewTable(t)"
            >
              <span class="truncate">{{ t.name }}</span>
              <span class="shrink-0 text-muted-foreground">{{ t.rows }}</span>
            </button>
            <div v-if="activeDb && !tables.length" class="px-2 py-3 text-xs text-muted-foreground">无表</div>
          </div>
        </div>

        <!-- 查询器与结果 -->
        <div class="rounded-lg border bg-background p-3">
          <textarea
            v-model="sqlText"
            class="h-24 w-full resize-y rounded-md border border-input bg-background p-2 font-mono text-[13px] outline-none focus:ring-1 focus:ring-primary"
            placeholder="SELECT * FROM some_table LIMIT 100"
            spellcheck="false"
            @keydown.ctrl.enter="runQuery()"
          />
          <div class="mt-2 flex items-center justify-between">
            <span class="text-xs text-muted-foreground">Ctrl+Enter 执行 · 仅允许只读语句 · 最多返回 200 行</span>
            <FaButton size="sm" :loading="querying" @click="runQuery()">执行查询</FaButton>
          </div>

          <div v-if="queryError" class="mt-3 rounded-md border border-red-300 bg-red-50 p-2 text-xs text-red-600 dark:bg-red-950/30">
            {{ queryError }}
          </div>
          <div v-if="queryResult" class="mt-3">
            <div class="mb-1 text-xs text-muted-foreground">
              {{ queryResult.rows.length }} 行 · 耗时 {{ queryElapsed }}
            </div>
            <div class="max-h-96 overflow-auto rounded-md border">
              <table class="w-full text-xs">
                <thead class="bg-muted/50 text-left">
                  <tr>
                    <th v-for="c in queryResult.columns" :key="c" class="whitespace-nowrap px-2 py-1.5 font-mono">{{ c }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, ri) in queryResult.rows" :key="ri" class="border-t hover:bg-accent/30">
                    <td v-for="(cell, ci) in row" :key="ci" class="max-w-72 truncate px-2 py-1 font-mono">
                      {{ typeof cell === 'object' ? JSON.stringify(cell) : cell }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
