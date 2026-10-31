<script setup lang="ts">
import type { MiddlewareInstance, MiddlewareType } from '@/api/modules/middleware'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import mwApi from '@/api/modules/middleware'
import YdNodeSelect from '@/components/YdNodeSelect/index.vue'

const toast = useFaToast()
const router = useRouter()
const loading = ref(false)
const items = ref<MiddlewareInstance[]>([])
const types = ref<MiddlewareType[]>([])
const keyword = ref('')
const node = ref('local')
const loadError = ref('')
const discoveryError = ref('')
let requestID = 0

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return kw ? items.value.filter(i => `${i.name} ${i.label} ${i.type} ${i.host} ${i.containerName} ${i.composeProject} ${i.remark}`.toLowerCase().includes(kw)) : items.value
})

async function load() {
  const request = ++requestID
  loading.value = true
  loadError.value = ''
  try {
    const res = await mwApi.list(node.value)
    if (request !== requestID) {
      return
    }
    items.value = res.data.items
    discoveryError.value = res.data.discoveryError || ''
  }
  catch (e: any) {
    if (request === requestID) {
      loadError.value = e?.message || '列表读取失败'
    }
  }
  finally {
    if (request === requestID) {
      loading.value = false
    }
  }
}

watch(node, () => {
  items.value = []
  discoveryError.value = ''
  void load()
})
onMounted(() => {
  void load()
  void mwApi.types().then((res) => {
    types.value = res.data
  }).catch((e: any) => toast.error(e?.message || '中间件类型读取失败'))
})
onBeforeUnmount(() => ++requestID)

function onNodesLoaded(nodes: { id: string }[]) {
  if (nodes.length && !nodes.some(n => n.id === node.value)) {
    node.value = nodes[0]!.id
  }
}

const sourceLabels = { middleware: '已登记', database: '数据库纳管', discovered: '自动发现' }
const stateLabels: Record<string, string> = {
  running: '运行中',
  exited: '已停止',
  created: '已创建',
  restarting: '重启中',
  paused: '已暂停',
  dead: '异常',
  removing: '移除中',
  missing: '容器未找到',
  unknown: '读取失败',
  external: '外部连接',
}

function openDatabase() {
  void router.push('/database')
}

// 登记外部实例只展示本域支持的类型；数据库域沿用已有接入流程。
const showAdd = ref(false)
const addForm = ref({ name: '', type: 'rabbitmq', host: '', port: 0, user: '', password: '', remark: '' })
const addSaving = ref(false)

function openAdd() {
  const type = types.value[0]
  addForm.value = { name: '', type: type?.type || 'rabbitmq', host: '', port: type?.defaultPort || 0, user: '', password: '', remark: '' }
  showAdd.value = true
}

function onAddTypeChange() {
  addForm.value.port = types.value.find(t => t.type === addForm.value.type)?.defaultPort || 0
}

async function doAdd() {
  if (!addForm.value.name.trim() || !addForm.value.host.trim()) {
    toast.warning('实例名与主机地址必填')
    return
  }
  addSaving.value = true
  try {
    await mwApi.create({ ...addForm.value, port: addForm.value.port || undefined })
    toast.success('已登记并完成连通验证')
    showAdd.value = false
    await load()
  }
  catch (e: any) {
    toast.error(e?.message || '登记失败')
  }
  finally {
    addSaving.value = false
  }
}

async function doTest(row: MiddlewareInstance) {
  try {
    await mwApi.test(row.id, row.nodeId)
    toast.success(`连接正常：${row.name}`)
  }
  catch (e: any) {
    toast.error(e?.message || '连接失败')
  }
}

async function doDelete(row: MiddlewareInstance) {
  try {
    await mwApi.remove(row.id, row.nodeId)
    toast.success('已移除登记，容器继续保留')
    await load()
  }
  catch (e: any) {
    toast.error(e?.message || '移除登记失败')
  }
}

async function doRefreshVersion(row: MiddlewareInstance) {
  try {
    const res = await mwApi.versionRefresh(row.id, row.nodeId)
    toast.success(`版本：${res.data.version || '未知（镜像未标注版本）'}`)
    await load()
  }
  catch (e: any) {
    toast.error(e?.message || '探测失败')
  }
}

const adopting = ref('')
async function doAdopt(row: MiddlewareInstance) {
  adopting.value = row.key
  try {
    await mwApi.adopt({ node: row.nodeId, containerName: row.containerName })
    toast.success(`已纳管 ${row.containerName}`)
    await load()
  }
  catch (e: any) {
    toast.error(e?.message || '纳管失败')
  }
  finally {
    adopting.value = ''
  }
}
</script>

<template>
  <div class="p-4 flex flex-col gap-4">
    <header class="flex flex-wrap gap-2 items-center">
      <div class="flex-1 min-w-0">
        <h1 class="text-lg font-semibold">
          中间件
        </h1>
        <p class="text-xs text-muted-foreground mt-1">
          自动发现当前节点的容器，并关联已有纳管记录；外部登记在各节点均可见。
        </p>
      </div>
      <div class="flex flex-wrap gap-2 items-center max-md:w-full">
        <YdNodeSelect v-model="node" @loaded="onNodesLoaded" />
        <FaInput v-model="keyword" placeholder="搜索实例、容器或类型" class="max-md:flex-1 w-48!" />
        <FaButton variant="outline" size="sm" :loading="loading" @click="load">
          刷新发现
        </FaButton>
        <FaButton size="sm" :disabled="!types.length" @click="openAdd">
          登记外部实例
        </FaButton>
      </div>
    </header>

    <div v-if="loadError || discoveryError" role="alert" class="text-sm p-3 border border-destructive/30 rounded-lg bg-destructive/5">
      {{ loadError || discoveryError }}
      <p class="text-xs text-muted-foreground mt-1">
        {{ loadError ? '请刷新重试；如有上次读取的记录，会继续保留。' : '已保留登记与数据库纳管记录，容器状态暂不可用。请检查所选节点连接和 Docker 服务后刷新。' }}
      </p>
    </div>

    <FaCard :padding="false">
      <div class="overflow-x-auto">
        <table class="text-sm w-full">
          <thead class="text-xs text-muted-foreground text-left bg-muted/40">
            <tr>
              <th class="font-medium px-4 py-2.5">
                实例 / 容器
              </th>
              <th class="font-medium px-4 py-2.5">
                类型
              </th>
              <th class="font-medium px-4 py-2.5">
                状态
              </th>
              <th class="font-medium px-4 py-2.5 hidden md:table-cell">
                连接地址
              </th>
              <th class="font-medium px-4 py-2.5">
                来源
              </th>
              <th class="font-medium px-4 py-2.5 hidden lg:table-cell">
                版本
              </th>
              <th class="font-medium px-4 py-2.5 text-right">
                操作
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !items.length">
              <td colspan="7" class="text-muted-foreground px-4 py-10 text-center">
                正在读取登记记录和节点容器…
              </td>
            </tr>
            <tr v-else-if="!filtered.length">
              <td colspan="7" class="text-muted-foreground px-4 py-10 text-center">
                <template v-if="loadError || discoveryError">
                  读取未完成，请查看上方错误并重试。
                </template>
                <template v-else-if="keyword.trim()">
                  没有匹配的实例或容器
                </template>
                <template v-else>
                  当前节点未发现可识别的中间件。可登记外部 RabbitMQ/Memcached；S3、Neo4j、Redis 等可在数据库管理中接入。
                </template>
              </td>
            </tr>
            <tr v-for="row in filtered" :key="row.key" class="border-t border-border/60 hover:bg-muted/30">
              <td class="px-4 py-2.5">
                <div class="font-medium">
                  {{ row.name }}
                </div>
                <div v-if="row.containerName && row.containerName !== row.name" class="text-xs text-muted-foreground font-mono mt-0.5">
                  {{ row.containerName }}
                </div>
                <div v-if="row.composeProject" class="text-xs text-muted-foreground mt-0.5">
                  {{ row.composeProject }}
                </div>
                <div v-if="row.image" class="text-xs text-muted-foreground mt-0.5 max-w-64 truncate" :title="row.image">
                  {{ row.image }}
                </div>
              </td>
              <td class="px-4 py-2.5">
                <span class="text-xs px-1.5 py-0.5 rounded bg-muted whitespace-nowrap">{{ row.label || row.type }}</span>
              </td>
              <td class="text-xs px-4 py-2.5 whitespace-nowrap" :class="row.state === 'running' ? 'text-primary' : 'text-muted-foreground'">
                {{ stateLabels[row.state] || row.state }}
              </td>
              <td class="text-xs font-mono px-4 py-2.5 hidden md:table-cell">
                {{ row.host }}:{{ row.port }}
              </td>
              <td class="text-xs px-4 py-2.5 whitespace-nowrap">
                {{ sourceLabels[row.source] }}
                <div v-if="row.source === 'discovered'" class="text-muted-foreground mt-1">
                  尚未纳管
                </div>
                <div v-else-if="row.origin === 'external'" class="text-muted-foreground mt-1">
                  外部接入
                </div>
              </td>
              <td class="text-xs px-4 py-2.5 hidden lg:table-cell" :class="row.reusable === false ? 'text-destructive' : 'text-muted-foreground'">
                {{ row.version || '未知' }}
                <div v-if="row.reusable === false">
                  低于复用基线 {{ row.minReuseVersion }}
                </div>
              </td>
              <td class="px-4 py-2.5 text-right">
                <div class="inline-flex gap-1 whitespace-nowrap">
                  <template v-if="row.source === 'middleware'">
                    <FaButton variant="ghost" size="sm" @click="doTest(row)">
                      测试
                    </FaButton>
                    <FaButton v-if="row.origin === 'container'" variant="ghost" size="sm" @click="doRefreshVersion(row)">
                      探测版本
                    </FaButton>
                    <FaButton variant="ghost" size="sm" class="text-destructive!" @click="doDelete(row)">
                      移除登记
                    </FaButton>
                  </template>
                  <FaButton v-else-if="row.adoptable" size="sm" :loading="adopting === row.key" @click="doAdopt(row)">
                    纳管
                  </FaButton>
                  <FaButton v-else-if="row.managedBy === 'database'" variant="outline" size="sm" @click="openDatabase">
                    {{ row.source === 'database' ? '数据库管理' : '前往接入' }}
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaCard>

    <FaModal v-model="showAdd" title="登记外部中间件" :footer="false" class="max-w-2xl!" :close-on-click-overlay="!addSaving">
      <p class="text-xs text-muted-foreground mb-4">
        登记 RabbitMQ / Memcached 的连接信息。S3、Neo4j、Redis 等沿用数据库管理的接入记录。
      </p>
      <div class="gap-3 grid grid-cols-1 sm:grid-cols-2">
        <label class="text-sm flex flex-col gap-1">
          <span>类型</span>
          <YdSelect v-model="addForm.type" :options="types.map(t => ({ label: t.label, value: t.type }))" @change="onAddTypeChange" />
        </label>
        <label class="text-sm flex flex-col gap-1">
          <span>实例名 <span class="text-destructive">*</span></span>
          <FaInput v-model="addForm.name" placeholder="如 rabbitmq-main" />
        </label>
        <label class="text-sm flex flex-col gap-1">
          <span>主机 <span class="text-destructive">*</span></span>
          <FaInput v-model="addForm.host" placeholder="面板可访问的 IP / 域名" />
        </label>
        <label class="text-sm flex flex-col gap-1">
          <span>端口</span>
          <FaInput v-model.number="addForm.port" type="number" />
        </label>
        <label class="text-sm flex flex-col gap-1">
          <span>用户名（可选）</span>
          <FaInput v-model="addForm.user" />
        </label>
        <label class="text-sm flex flex-col gap-1">
          <span>密码（可选）</span>
          <FaInput v-model="addForm.password" type="password" />
        </label>
        <label class="text-sm flex flex-col gap-1 sm:col-span-2">
          <span>备注</span>
          <FaInput v-model="addForm.remark" />
        </label>
      </div>
      <div class="mt-5 flex gap-2 justify-end">
        <FaButton variant="outline" :disabled="addSaving" @click="showAdd = false">
          取消
        </FaButton>
        <FaButton :loading="addSaving" @click="doAdd">
          验证并登记
        </FaButton>
      </div>
    </FaModal>
  </div>
</template>
