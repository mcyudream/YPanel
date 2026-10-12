<script setup lang="ts">
// M68 Neo4j 图工作台：标签/关系侧栏 + 节点分页浏览（表格）+ Cypher 控制台 + 节点编辑 + 索引/约束。
import { ref, watch, computed } from 'vue'
const toast = useFaToast()
import apiDBA, { type DbQueryResult } from '@/api/modules/dbadmin'

const props = defineProps<{ instanceId: number, db: string }>()

const labels = ref<{ name: string, nodes: number }[]>([])
const relTypes = ref<{ name: string, nodes: number }[]>([])
const activeLabel = ref('')
const metaLoading = ref(false)

const nodes = ref<Record<string, any>[]>([])
const nodesTotal = ref(0)
const page = ref(1)
const pageSize = 20
const nodesLoading = ref(false)

// ---- 编辑弹窗 ----
const editOpen = ref(false)
const editProps = ref('{}')
const editId = ref('')
const editSaving = ref(false)
const createOpen = ref(false)
const createLabel = ref('')
const createProps = ref('{}')

// ---- Cypher 控制台 ----
const cypher = ref('MATCH (n) RETURN n LIMIT 20')
const cypherResult = ref<DbQueryResult | null>(null)
const cypherRunning = ref(false)

const totalPages = computed(() => Math.max(1, Math.ceil(nodesTotal.value / pageSize)))

async function loadMeta() {
  metaLoading.value = true
  try {
    const meta = await apiDBA.neo4jMeta(props.instanceId, props.db)
    labels.value = meta.labels || []
    relTypes.value = meta.relTypes || []
    if (!labels.value.find(l => l.name === activeLabel.value))
      activeLabel.value = labels.value[0]?.name || ''
  }
  catch (e: any) {
    toast.error(e?.message || '加载图元失败')
  }
  finally {
    metaLoading.value = false
  }
}

async function loadNodes() {
  if (!activeLabel.value)
    return
  nodesLoading.value = true
  try {
    const r = await apiDBA.neo4jNodes(props.instanceId, props.db, activeLabel.value, (page.value - 1) * pageSize, pageSize)
    nodes.value = (r.nodes || []).map(s => JSON.parse(s))
    nodesTotal.value = r.total
  }
  catch (e: any) {
    toast.error(e?.message || '加载节点失败')
  }
  finally {
    nodesLoading.value = false
  }
}

watch(() => [props.instanceId, props.db], () => {
  activeLabel.value = ''
  cypherResult.value = null
  loadMeta()
}, { immediate: true })

watch(activeLabel, () => {
  page.value = 1
  loadNodes()
})

watch(page, loadNodes)

const propKeys = computed(() => {
  const keys = new Set<string>()
  for (const n of nodes.value) {
    for (const k of Object.keys(n)) {
      if (!k.startsWith('_'))
        keys.add(k)
    }
  }
  return [...keys].slice(0, 6)
})

function cellOf(row: Record<string, any>, key: string) {
  const v = row[key]
  if (v === null || v === undefined)
    return ''
  if (typeof v === 'object')
    return JSON.stringify(v)
  return String(v)
}

function openEdit(row: Record<string, any>) {
  editId.value = JSON.stringify({ elementId: row._elementId, id: row._id })
  const propsOnly: Record<string, any> = {}
  for (const [k, v] of Object.entries(row)) {
    if (!k.startsWith('_'))
      propsOnly[k] = v
  }
  editProps.value = JSON.stringify(propsOnly, null, 2)
  editOpen.value = true
}

async function doSaveEdit() {
  editSaving.value = true
  try {
    JSON.parse(editProps.value)
  }
  catch {
    toast.error('属性 JSON 不合法')
    editSaving.value = false
    return
  }
  try {
    await apiDBA.neo4jNodeUpdate(props.instanceId, { db: props.db, id: editId.value, props: editProps.value })
    toast.success('已更新节点属性')
    editOpen.value = false
    await loadNodes()
  }
  catch (e: any) {
    toast.error(e?.message || '更新失败')
  }
  finally {
    editSaving.value = false
  }
}

async function doCreate() {
  if (!createLabel.value.trim()) {
    toast.warning('标签必填')
    return
  }
  try {
    JSON.parse(createProps.value)
  }
  catch {
    toast.error('属性 JSON 不合法')
    return
  }
  try {
    await apiDBA.neo4jNodeCreate(props.instanceId, { db: props.db, label: createLabel.value, props: createProps.value })
    toast.success('节点已创建')
    createOpen.value = false
    activeLabel.value = createLabel.value
    await loadMeta()
  }
  catch (e: any) {
    toast.error(e?.message || '创建失败')
  }
}

async function doDelete(row: Record<string, any>, detach = true) {
  try {
    await apiDBA.neo4jNodeDelete(props.instanceId, { db: props.db, id: JSON.stringify({ elementId: row._elementId, id: row._id }), detach })
    toast.success('节点已删除')
    await Promise.all([loadNodes(), loadMeta()])
  }
  catch (e: any) {
    toast.error(e?.message || '删除失败')
  }
}

async function runCypher() {
  if (!cypher.value.trim())
    return
  cypherRunning.value = true
  try {
    cypherResult.value = await apiDBA.neo4jCypher(props.instanceId, props.db, cypher.value, 200)
  }
  catch (e: any) {
    toast.error(e?.message || '执行失败')
  }
  finally {
    cypherRunning.value = false
  }
}

// ---- 索引/约束 ----
const schemaOpen = ref(false)
const schemaRows = ref<Record<string, unknown>[]>([])
const schemaTab = ref<'indexes' | 'constraints'>('indexes')

async function openSchema(tab: 'indexes' | 'constraints') {
  schemaTab.value = tab
  try {
    schemaRows.value = tab === 'indexes'
      ? await apiDBA.neo4jIndexes(props.instanceId, props.db)
      : await apiDBA.neo4jConstraints(props.instanceId, props.db)
  }
  catch (e: any) {
    toast.error(e?.message || '加载失败')
  }
  schemaOpen.value = true
}

// 索引创建
const idxOpen = ref(false)
const idxForm = ref({ name: '', label: '', props: '', unique: false })

function openIndexCreate() {
  idxForm.value = { name: '', label: activeLabel.value, props: '', unique: false }
  idxOpen.value = true
}

async function doIndexCreate() {
  const propsList = idxForm.value.props.split(',').map(s => s.trim()).filter(Boolean)
  if (!idxForm.value.label || !propsList.length) {
    toast.warning('标签与属性（逗号分隔）必填')
    return
  }
  try {
    await apiDBA.neo4jIndexCreate(props.instanceId, { db: props.db, name: idxForm.value.name || undefined, label: idxForm.value.label, props: propsList, unique: idxForm.value.unique })
    toast.success(idxForm.value.unique ? '唯一约束已创建' : '索引已创建')
    idxOpen.value = false
    await openSchema(idxForm.value.unique ? 'constraints' : 'indexes')
  }
  catch (e: any) {
    toast.error(e?.message || '创建失败')
  }
}

async function doIndexDrop(name: unknown) {
  try {
    await apiDBA.neo4jIndexDrop(props.instanceId, { db: props.db, name: String(name) })
    toast.success('已删除')
    await openSchema(schemaTab.value)
  }
  catch (e: any) {
    toast.error(e?.message || '删除失败')
  }
}
</script>

<template>
  <div class="grid h-full min-h-0 grid-cols-1 gap-3 lg:grid-cols-[220px_1fr]">
    <!-- 侧栏：标签 + 关系类型 -->
    <div class="min-h-0 overflow-auto rounded-lg border bg-background p-2">
      <div class="mb-1 flex items-center justify-between px-1">
        <span class="text-xs font-medium text-muted-foreground">标签</span>
        <FaButton variant="ghost" size="sm" @click="createOpen = true">
          + 节点
        </FaButton>
      </div>
      <button
        v-for="l in labels" :key="l.name" type="button"
        class="mb-0.5 flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm hover:bg-accent/50"
        :class="{ 'bg-accent font-medium': activeLabel === l.name }"
        @click="activeLabel = l.name"
      >
        <span class="size-2 shrink-0 rounded-full bg-sky-500" />
        <span class="min-w-0 flex-1 truncate">{{ l.name }}</span>
        <span class="text-[10px] text-muted-foreground">{{ l.nodes }}</span>
      </button>
      <div class="mb-1 mt-3 px-1 text-xs font-medium text-muted-foreground">
        关系类型
      </div>
      <div v-for="r in relTypes" :key="r.name" class="flex items-center gap-2 rounded px-2 py-1 text-xs text-muted-foreground">
        <span class="size-2 shrink-0 rounded-full bg-violet-500" />
        <span class="min-w-0 flex-1 truncate">{{ r.name }}</span>
        <span class="text-[10px]">{{ r.nodes }}</span>
      </div>
    </div>

    <!-- 主区 -->
    <div class="flex min-h-0 min-w-0 flex-col gap-3">
      <!-- 节点表 -->
      <FaCard :padding="false" class="min-h-0 flex-1 overflow-hidden">
        <div class="flex flex-wrap items-center gap-2 border-b px-3 py-2">
          <span class="text-sm font-medium">{{ activeLabel || '（选择标签）' }}</span>
          <span class="text-xs text-muted-foreground">共 {{ nodesTotal }} 节点</span>
          <span class="flex-1" />
          <FaButton variant="ghost" size="sm" @click="openSchema('indexes')">
            索引
          </FaButton>
          <FaButton variant="ghost" size="sm" @click="openSchema('constraints')">
            约束
          </FaButton>
        </div>
        <div class="max-h-[46vh] overflow-auto">
          <table class="w-full text-sm">
            <thead class="sticky top-0 bg-muted/60 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2 font-medium">
                  id
                </th>
                <th v-for="k in propKeys" :key="k" class="px-3 py-2 font-medium">
                  {{ k }}
                </th>
                <th class="px-3 py-2 text-right font-medium">
                  操作
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="nodesLoading">
                <td :colspan="propKeys.length + 2" class="px-3 py-8 text-center text-muted-foreground">
                  加载中…
                </td>
              </tr>
              <tr v-else-if="!nodes.length">
                <td :colspan="propKeys.length + 2" class="px-3 py-8 text-center text-muted-foreground">
                  {{ activeLabel ? '该标签暂无节点' : '从左侧选择标签浏览节点' }}
                </td>
              </tr>
              <tr v-for="(row, i) in nodes" v-else :key="row._elementId || i" class="border-t border-border/60 hover:bg-muted/30">
                <td class="px-3 py-2 font-mono text-xs text-muted-foreground">
                  {{ row._id }}
                </td>
                <td v-for="k in propKeys" :key="k" class="max-w-[220px] truncate px-3 py-2 font-mono text-xs" :title="cellOf(row, k)">
                  {{ cellOf(row, k) }}
                </td>
                <td class="px-3 py-2 text-right">
                  <div class="inline-flex gap-1">
                    <FaButton variant="ghost" size="sm" @click="openEdit(row)">
                      编辑
                    </FaButton>
                    <FaButton variant="ghost" size="sm" class="text-destructive!" @click="doDelete(row)">
                      删除
                    </FaButton>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="flex items-center gap-2 border-t px-3 py-1.5 text-xs text-muted-foreground">
          <FaButton variant="ghost" size="sm" :disabled="page <= 1" @click="page--">
            上一页
          </FaButton>
          <span>{{ page }} / {{ totalPages }}</span>
          <FaButton variant="ghost" size="sm" :disabled="page >= totalPages" @click="page++">
            下一页
          </FaButton>
        </div>
      </FaCard>

      <!-- Cypher 控制台 -->
      <FaCard :padding="false" class="shrink-0">
        <div class="flex items-center gap-2 border-b px-3 py-2">
          <span class="text-sm font-medium">Cypher</span>
          <span class="text-xs text-muted-foreground">读语句只读会话、写语句自动计数反馈</span>
          <span class="flex-1" />
          <FaButton size="sm" :loading="cypherRunning" @click="runCypher">
            执行 (Ctrl+Enter)
          </FaButton>
        </div>
        <div class="p-2">
          <textarea
            v-model="cypher"
            rows="3"
            class="w-full resize-y rounded-md border bg-muted/30 p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-ring"
            @keydown.ctrl.enter.prevent="runCypher"
          />
          <div v-if="cypherResult" class="mt-2 max-h-[24vh] overflow-auto rounded border">
            <table class="w-full text-xs">
              <thead class="sticky top-0 bg-muted/60 text-left text-muted-foreground">
                <tr>
                  <th v-for="c in cypherResult.columns" :key="c" class="px-2 py-1.5 font-medium">
                    {{ c }}
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, ri) in cypherResult.rows" :key="ri" class="border-t border-border/60">
                  <td v-for="(cell, ci) in row" :key="ci" class="max-w-[320px] truncate px-2 py-1.5 font-mono" :title="typeof cell === 'object' ? JSON.stringify(cell) : String(cell)">
                    {{ typeof cell === 'object' ? JSON.stringify(cell) : cell }}
                  </td>
                </tr>
                <tr v-if="!cypherResult.rows.length">
                  <td :colspan="cypherResult.columns.length" class="px-2 py-3 text-center text-muted-foreground">
                    {{ cypherResult.elapsed }} · 无行返回
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </FaCard>
    </div>

    <!-- 节点编辑 -->
    <FaModal v-model:open="editOpen" title="编辑节点属性" :ok-loading="editSaving" @ok="doSaveEdit" @cancel="editOpen = false">
      <p class="mb-2 text-xs text-muted-foreground">
        合并语义（SET +=）：键值覆盖、新键追加、显式 null 删除属性。定位：{{ editId }}
      </p>
      <textarea v-model="editProps" rows="10" class="w-full rounded-md border bg-muted/30 p-2 font-mono text-xs" />
    </FaModal>

    <!-- 新建节点 -->
    <FaModal v-model:open="createOpen" title="新建节点" @ok="doCreate" @cancel="createOpen = false">
      <div class="flex flex-col gap-3">
        <label class="flex flex-col gap-1 text-sm">
          <span>标签 <span class="text-destructive">*</span></span>
          <FaInput v-model="createLabel" placeholder="如 Person" />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span>属性 JSON</span>
          <textarea v-model="createProps" rows="6" class="w-full rounded-md border bg-muted/30 p-2 font-mono text-xs" />
        </label>
      </div>
    </FaModal>

    <!-- 索引/约束 -->
    <FaModal v-model:open="schemaOpen" :title="schemaTab === 'indexes' ? '索引' : '约束'" :footer="false" width="min(92vw, 680px)">
      <div class="mb-2 flex gap-2">
        <FaButton size="sm" :variant="schemaTab === 'indexes' ? 'default' : 'outline'" @click="openSchema('indexes')">
          索引
        </FaButton>
        <FaButton size="sm" :variant="schemaTab === 'constraints' ? 'default' : 'outline'" @click="openSchema('constraints')">
          约束
        </FaButton>
        <span class="flex-1" />
        <FaButton size="sm" @click="openIndexCreate">
          新建{{ schemaTab === 'indexes' ? '索引' : '唯一约束' }}
        </FaButton>
      </div>
      <div class="max-h-[56vh] overflow-auto">
        <table class="w-full text-xs">
          <thead class="bg-muted/60 text-left text-muted-foreground">
            <tr>
              <th class="px-2 py-1.5 font-medium">
                名称
              </th>
              <th class="px-2 py-1.5 font-medium">
                类型
              </th>
              <th class="px-2 py-1.5 font-medium">
                标签/关系
              </th>
              <th class="px-2 py-1.5 font-medium">
                属性
              </th>
              <th class="px-2 py-1.5 text-right font-medium">
                操作
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(r, i) in schemaRows" :key="i" class="border-t border-border/60">
              <td class="px-2 py-1.5 font-mono">
                {{ r.name }}
              </td>
              <td class="px-2 py-1.5">
                {{ r.type || r.state }}
              </td>
              <td class="px-2 py-1.5 font-mono">
                {{ (r.labelsOrTypes as string[] | undefined)?.join(', ') }}
              </td>
              <td class="px-2 py-1.5 font-mono">
                {{ (r.properties as string[] | undefined)?.join(', ') }}
              </td>
              <td class="px-2 py-1.5 text-right">
                <FaButton variant="ghost" size="sm" class="text-destructive!" @click="doIndexDrop(r.name)">
                  删除
                </FaButton>
              </td>
            </tr>
            <tr v-if="!schemaRows.length">
              <td colspan="5" class="px-2 py-6 text-center text-muted-foreground">
                暂无{{ schemaTab === 'indexes' ? '索引' : '约束' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaModal>

    <!-- 新建索引/约束 -->
    <FaModal v-model:open="idxOpen" :title="idxForm.unique ? '新建唯一约束' : '新建索引'" @ok="doIndexCreate" @cancel="idxOpen = false">
      <div class="flex flex-col gap-3">
        <label class="flex flex-col gap-1 text-sm">
          <span>标签 <span class="text-destructive">*</span></span>
          <FaInput v-model="idxForm.label" />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span>属性（逗号分隔）<span class="text-destructive">*</span></span>
          <FaInput v-model="idxForm.props" placeholder="如 email" />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span>名称（可选，缺省自动生成）</span>
          <FaInput v-model="idxForm.name" />
        </label>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="idxForm.unique" type="checkbox" class="accent-primary">
          <span>唯一约束（UNIQUENESS）</span>
        </label>
      </div>
    </FaModal>
  </div>
</template>
