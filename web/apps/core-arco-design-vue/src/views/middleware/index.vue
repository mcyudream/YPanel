<script setup lang="ts">
// M68 可复用中间件实例页（容器域二级叶子）：登记清单 / 连接测试 / 节点容器扫描发现与一键纳管。
import { onMounted, ref, computed } from 'vue'
const toast = useFaToast()
import mwApi from '@/api/modules/middleware'
import type { MiddlewareInstance, MiddlewareType, MiddlewareScanItem } from '@/api/modules/middleware'
import YdNodeSelect from '@/components/YdNodeSelect/index.vue'

const loading = ref(false)
const items = ref<MiddlewareInstance[]>([])
const types = ref<MiddlewareType[]>([])
const keyword = ref('')

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw)
    return items.value
  return items.value.filter(i => `${i.name} ${i.type} ${i.host} ${i.remark || ''}`.toLowerCase().includes(kw))
})

async function load() {
  loading.value = true
  try {
    const [list, tp] = await Promise.all([mwApi.list(), mwApi.types()])
    items.value = (list.data?.items ?? list.data ?? []) as MiddlewareInstance[]
    types.value = (tp.data ?? []) as MiddlewareType[]
  }
  catch (e: any) {
    toast.error(e?.message || '加载失败')
  }
  finally {
    loading.value = false
  }
}

onMounted(load)

function typeLabel(t: string) {
  return types.value.find(x => x.type === t)?.label || t
}

// ---- 添加（手动登记外部实例） ----
const showAdd = ref(false)
const addForm = ref({ name: '', type: 'rabbitmq', host: '', port: 0, user: '', password: '', remark: '' })
const addSaving = ref(false)

function openAdd() {
  addForm.value = { name: '', type: types.value[0]?.type || 'rabbitmq', host: '', port: 0, user: '', password: '', remark: '' }
  showAdd.value = true
}

function onAddTypeChange() {
  const tp = types.value.find(x => x.type === addForm.value.type)
  if (tp && !addForm.value.port)
    addForm.value.port = tp.defaultPort
}

async function doAdd() {
  if (!addForm.value.name.trim() || !addForm.value.host.trim()) {
    toast.warning('实例名与主机地址必填')
    return
  }
  addSaving.value = true
  try {
    await mwApi.create({ ...addForm.value, port: addForm.value.port || undefined })
    toast.success('已登记并连通验证')
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

// ---- 测试 / 删除 / 版本刷新 ----
async function doTest(row: MiddlewareInstance) {
  try {
    await mwApi.test(row.id)
    toast.success(`连接正常：${row.name}`)
  }
  catch (e: any) {
    toast.error(e?.message || '连接失败')
  }
}

async function doDelete(row: MiddlewareInstance) {
  try {
    await mwApi.remove(row.id)
    toast.success('已移除登记（实例本体不受影响）')
    await load()
  }
  catch (e: any) {
    toast.error(e?.message || '删除失败')
  }
}

async function doRefreshVersion(row: MiddlewareInstance) {
  try {
    const res = await mwApi.versionRefresh(row.id)
    toast.success(`版本：${(res.data as any)?.version || '未知'}`)
    await load()
  }
  catch (e: any) {
    toast.error(e?.message || '探测失败')
  }
}

// ---- 扫描发现 ----
const showScan = ref(false)
const scanning = ref(false)
const scanNode = ref('local')
const scanItems = ref<MiddlewareScanItem[]>([])
const adopting = ref('')

async function openScan() {
  showScan.value = true
  await doScan()
}

async function doScan() {
  scanning.value = true
  scanItems.value = []
  try {
    const res = await mwApi.scan(scanNode.value)
    scanItems.value = (res.data ?? []) as MiddlewareScanItem[]
  }
  catch (e: any) {
    toast.error(e?.message || '扫描失败')
  }
  finally {
    scanning.value = false
  }
}

async function doAdopt(item: MiddlewareScanItem) {
  adopting.value = item.containerName
  try {
    await mwApi.adopt({ node: scanNode.value, containerName: item.containerName })
    toast.success(`已纳管 ${item.containerName}`)
    item.reused = true
    item.adoptable = false
    await load()
  }
  catch (e: any) {
    toast.error(e?.message || '纳管失败')
  }
  finally {
    adopting.value = ''
  }
}

function versionBadge(row: { version?: string, reusable?: boolean }) {
  if (!row.version)
    return '未知'
  return row.reusable === false ? `${row.version}（低于基线）` : row.version
}
</script>

<template>
  <div class="flex flex-col gap-4 p-4">
    <header class="flex flex-wrap items-center gap-2">
      <h1 class="text-lg font-semibold">
        中间件
      </h1>
      <span class="text-xs text-muted-foreground">可复用的非数据库类共享中间件（应用安装向导按类型复用，不重复创建）</span>
      <div class="ml-auto flex items-center gap-2">
        <FaInput v-model="keyword" placeholder="搜索实例" class="w-48!" />
        <FaButton variant="outline" size="sm" @click="openScan">
          扫描发现
        </FaButton>
        <FaButton size="sm" @click="openAdd">
          登记实例
        </FaButton>
      </div>
    </header>

    <FaCard :padding="false" class="overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-muted/40 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2.5 font-medium">
                实例名
              </th>
              <th class="px-4 py-2.5 font-medium">
                类型
              </th>
              <th class="px-4 py-2.5 font-medium">
                连接地址
              </th>
              <th class="hidden px-4 py-2.5 font-medium md:table-cell">
                来源
              </th>
              <th class="px-4 py-2.5 font-medium">
                版本
              </th>
              <th class="hidden px-4 py-2.5 font-medium lg:table-cell">
                备注
              </th>
              <th class="px-4 py-2.5 text-right font-medium">
                操作
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !items.length">
              <td colspan="7" class="px-4 py-10 text-center text-muted-foreground">
                加载中…
              </td>
            </tr>
            <tr v-else-if="!filtered.length">
              <td colspan="7" class="px-4 py-10 text-center text-muted-foreground">
                暂无实例——右上角「登记实例」手动添加，或「扫描发现」一键纳管节点上已有中间件容器
              </td>
            </tr>
            <tr v-for="row in filtered" :key="row.id" class="border-t border-border/60 hover:bg-muted/30">
              <td class="px-4 py-2.5 font-medium">
                {{ row.name }}
              </td>
              <td class="px-4 py-2.5">
                <span class="rounded bg-muted px-1.5 py-0.5 text-xs">{{ typeLabel(row.type) }}</span>
              </td>
              <td class="px-4 py-2.5 font-mono text-xs">
                {{ row.host }}:{{ row.port }}
              </td>
              <td class="hidden px-4 py-2.5 md:table-cell">
                <span class="text-xs text-muted-foreground">{{ row.origin === 'container' ? `容器 ${row.containerName}` : '外部' }}</span>
              </td>
              <td class="px-4 py-2.5">
                <span class="text-xs" :class="row.reusable === false ? 'text-destructive' : 'text-muted-foreground'">{{ versionBadge(row) }}</span>
                <span v-if="row.minReuseVersion" class="ml-1 text-[10px] text-muted-foreground">基线 {{ row.minReuseVersion }}</span>
              </td>
              <td class="hidden px-4 py-2.5 text-xs text-muted-foreground lg:table-cell">
                {{ row.remark }}
              </td>
              <td class="px-4 py-2.5 text-right">
                <div class="inline-flex gap-1">
                  <FaButton variant="ghost" size="sm" @click="doTest(row)">
                    测试
                  </FaButton>
                  <FaButton variant="ghost" size="sm" @click="doRefreshVersion(row)">
                    探测版本
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
    </FaCard>

    <!-- 登记实例 -->
    <FaModal v-model:open="showAdd" title="登记中间件实例" :ok-loading="addSaving" @ok="doAdd" @cancel="showAdd = false">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="flex flex-col gap-1 text-sm">
          <span>类型</span>
          <YdSelect v-model="addForm.type" :options="types.map(t => ({ label: t.label, value: t.type }))" @change="onAddTypeChange" />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span>实例名 <span class="text-destructive">*</span></span>
          <FaInput v-model="addForm.name" placeholder="如 rabbitmq-main" />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span>主机 <span class="text-destructive">*</span></span>
          <FaInput v-model="addForm.host" placeholder="IP / 容器名" />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span>端口（0 = 类型默认）</span>
          <FaInput v-model.number="addForm.port" type="number" />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span>用户名（可选）</span>
          <FaInput v-model="addForm.user" />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span>密码（可选）</span>
          <FaInput v-model="addForm.password" type="password" />
        </label>
        <label class="flex flex-col gap-1 text-sm sm:col-span-2">
          <span>备注</span>
          <FaInput v-model="addForm.remark" />
        </label>
      </div>
    </FaModal>

    <!-- 扫描发现 -->
    <FaModal v-model:open="showScan" title="扫描节点中间件容器" :footer="false" width="min(92vw, 760px)">
      <div class="mb-3 flex items-center gap-2">
        <YdNodeSelect v-model="scanNode" class="w-56" @change="doScan" />
        <FaButton variant="outline" size="sm" :loading="scanning" @click="doScan">
          重新扫描
        </FaButton>
        <span class="text-xs text-muted-foreground">按镜像规则识别已注册中间件类型，从容器 env 推断凭据</span>
      </div>
      <div class="max-h-[60vh] overflow-auto">
        <div v-if="scanning" class="py-10 text-center text-muted-foreground">
          扫描中…
        </div>
        <div v-else-if="!scanItems.length" class="py-10 text-center text-muted-foreground">
          未发现可识别的中间件容器
        </div>
        <div v-for="item in scanItems" v-else :key="item.containerName" class="mb-2 flex items-center gap-3 rounded-lg border border-border/60 p-3">
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="rounded bg-muted px-1.5 py-0.5 text-xs">{{ item.label }}</span>
              <span class="truncate font-mono text-xs">{{ item.containerName }}</span>
              <span v-if="item.version" class="text-xs text-muted-foreground">v{{ item.version }}</span>
              <span v-if="item.reused" class="text-xs text-primary">已纳管</span>
              <span v-else-if="!item.reusable" class="text-xs text-destructive">版本低于可复用基线</span>
            </div>
            <div class="mt-0.5 truncate text-xs text-muted-foreground">
              {{ item.image }}<template v-if="item.composeProject"> · {{ item.composeProject }}</template>
            </div>
          </div>
          <FaButton v-if="item.adoptable" size="sm" :loading="adopting === item.containerName" @click="doAdopt(item)">
            纳管
          </FaButton>
        </div>
      </div>
    </FaModal>
  </div>
</template>
