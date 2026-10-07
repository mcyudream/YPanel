<script setup lang="ts">
import type { ProcessItem, ProcessSort, ServiceItem } from '@/api/modules/nodeexec'
import { procApi } from '@/api/modules/nodeexec'
import apiNode from '@/api/modules/node'
import type { NodeItem } from '@/api/modules/node'

defineOptions({
  name: 'ProcessesIndex',
})

const route = useRoute()
const router = useRouter()

const tab = ref<'processes' | 'services'>('processes')
const processes = ref<ProcessItem[]>([])
const services = ref<ServiceItem[]>([])
const loadedTabs = ref<{ processes: boolean, services: boolean }>({ processes: false, services: false })
const loading = ref(false)
const keyword = ref('')
const acting = ref('')
const loadError = ref('')

// ---- 节点选择 ----
const nodes = ref<NodeItem[]>([])
const nodeId = ref<'local' | string>((route.query.node as string) || 'local')
const nodeOptions = computed(() => {
  const online = nodes.value.filter(n => n.online)
  return [{ id: 'local', name: `${nodes.value.find(n => n.id === 'local')?.name || '本机'}（本机）` }, ...online.filter(n => n.id !== 'local').map(n => ({ id: n.id, name: n.name }))]
})

async function loadNodes() {
  try {
    nodes.value = await apiNode.list()
  }
  catch {}
}

// ---- 排序 ----
type ProcSortKey = ProcessSort
const procSortKey = ref<ProcSortKey>('cpu')
const procSortOrder = ref<'asc' | 'desc'>('desc')
const svcSortKey = ref<'name' | 'active'>('name')
const svcSortOrder = ref<'asc' | 'desc'>('asc')

function toggleProcSort(key: ProcSortKey) {
  if (procSortKey.value === key) {
    procSortOrder.value = procSortOrder.value === 'asc' ? 'desc' : 'asc'
  }
  else {
    procSortKey.value = key
    procSortOrder.value = key === 'name' ? 'asc' : 'desc'
  }
}

function toggleSvcSort(key: 'name' | 'active') {
  if (svcSortKey.value === key) {
    svcSortOrder.value = svcSortOrder.value === 'asc' ? 'desc' : 'asc'
  }
  else {
    svcSortKey.value = key
    svcSortOrder.value = 'asc'
  }
}

// ---- 加载 ----
// 各 tab 独立拉取：初始与切换节点时并行拉全部（tab 计数才准确），轮询只刷当前 tab
async function loadTab(which: 'processes' | 'services') {
  if (which === 'processes') {
    processes.value = await procApi.processes({
      node: nodeId.value,
      sort: procSortKey.value,
      order: procSortOrder.value,
      limit: 500,
    })
  }
  else {
    services.value = await procApi.services(nodeId.value)
  }
  loadedTabs.value[which] = true
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    await loadTab(tab.value)
  }
  catch (e: any) {
    loadError.value = e?.message || '加载失败'
  }
  finally {
    loading.value = false
  }
}

async function loadAll() {
  loading.value = true
  loadError.value = ''
  try {
    await Promise.all([loadTab('processes'), loadTab('services')])
  }
  catch (e: any) {
    loadError.value = e?.message || '加载失败'
  }
  finally {
    loading.value = false
  }
}

// ---- 筛选 / 排序 / 分页（本地） ----
const filteredProcesses = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) {
    return processes.value
  }
  return processes.value.filter(p => p.name.toLowerCase().includes(kw) || p.cmdline.toLowerCase().includes(kw))
})

const filteredServices = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  const list = kw
    ? services.value.filter(s => s.name.toLowerCase().includes(kw) || s.desc.toLowerCase().includes(kw))
    : [...services.value]
  const dir = svcSortOrder.value === 'asc' ? 1 : -1
  if (svcSortKey.value === 'active') {
    // 活跃服务排前（desc 时排后），同级按名称
    return list.sort((a, b) => ((b.active === 'active' ? 1 : 0) - (a.active === 'active' ? 1 : 0)) * dir || a.name.localeCompare(b.name))
  }
  return list.sort((a, b) => a.name.localeCompare(b.name) * dir)
})

const page = ref(1)
const size = ref(20)

const pagedProcesses = computed(() => filteredProcesses.value.slice((page.value - 1) * size.value, page.value * size.value))
const pagedServices = computed(() => filteredServices.value.slice((page.value - 1) * size.value, page.value * size.value))
const pagedTotal = computed(() => (tab.value === 'processes' ? filteredProcesses.value.length : filteredServices.value.length))
void pagedTotal

watch(keyword, () => {
  page.value = 1
})
watch([procSortKey, procSortOrder], () => {
  page.value = 1
  loadTab('processes').catch(() => {})
})
watch(tab, () => {
  page.value = 1
  load()
})

function pickNode(id: string) {
  nodeId.value = id
  router.replace({ query: { ...route.query, node: id === 'local' ? undefined : id } })
  page.value = 1
  loadAll()
}

function fmtRss(n: number) {
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

async function kill(p: ProcessItem) {
  const modal = useFaModal()
  modal.confirm({
    title: '结束进程',
    content: `确认强制结束进程 ${p.name} (PID ${p.pid})？`,
    onConfirm: async () => {
      try {
        await procApi.kill(p.pid, nodeId.value)
        useFaToast().success('已结束')
        await load()
      }
      catch (e: any) {
        useFaToast().error('操作失败', { description: e?.message })
      }
    },
  })
}

async function svcAction(s: ServiceItem, action: 'start' | 'stop' | 'restart') {
  acting.value = s.name + action
  try {
    await procApi.serviceAction(s.name, action, nodeId.value)
    useFaToast().success(`已${action === 'start' ? '启动' : action === 'stop' ? '停止' : '重启'} ${s.name}`)
    await load()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

let timer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await loadNodes()
  loadAll()
  timer = setInterval(() => {
    if (!acting.value) {
      load()
    }
  }, 5000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="cpu" :size="24" />
          <span>进程与服务</span>
        </div>
      </template>
      <template #description>
        <span>进程与 systemd 服务管理（可选节点）</span>
      </template>
      <div class="flex flex-wrap items-center gap-2">
        <select
          :value="nodeId"
          class="h-8 rounded-md border bg-background px-2 text-sm outline-none"
          @change="pickNode(($event.target as HTMLSelectElement).value)"
        >
          <option v-for="n in nodeOptions" :key="n.id" :value="n.id">
            {{ n.name }}
          </option>
        </select>
        <FaInput v-model="keyword" placeholder="筛选…" class="w-44" />
        <FaButton variant="outline" size="icon-sm" title="刷新" @click="load">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <FaTabs
        v-model="tab" :list="[
          { label: `进程 (${filteredProcesses.length})`, value: 'processes' },
          { label: `服务 (${filteredServices.length})`, value: 'services' },
        ]"
      />

      <div v-if="loadError" class="mt-3 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
        {{ loadError }}（节点离线或 agent 不可达）
      </div>

      <!-- 进程 -->
      <template v-if="tab === 'processes'">
        <div class="mt-3 overflow-x-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">
                  <button type="button" class="inline-flex cursor-pointer items-center gap-0.5 select-none hover:text-foreground" @click="toggleProcSort('pid')">
                    PID
                    <FaIcon v-if="procSortKey === 'pid'" :name="procSortOrder === 'asc' ? 'i-lucide:arrow-up' : 'i-lucide:arrow-down'" class="text-[11px]" />
                  </button>
                </th>
                <th class="px-3 py-2">
                  <button type="button" class="inline-flex cursor-pointer items-center gap-0.5 select-none hover:text-foreground" @click="toggleProcSort('name')">
                    名称
                    <FaIcon v-if="procSortKey === 'name'" :name="procSortOrder === 'asc' ? 'i-lucide:arrow-up' : 'i-lucide:arrow-down'" class="text-[11px]" />
                  </button>
                </th>
                <th class="px-3 py-2">
                  <button type="button" class="inline-flex cursor-pointer items-center gap-0.5 select-none hover:text-foreground" @click="toggleProcSort('cpu')">
                    CPU%
                    <FaIcon v-if="procSortKey === 'cpu'" :name="procSortOrder === 'asc' ? 'i-lucide:arrow-up' : 'i-lucide:arrow-down'" class="text-[11px]" />
                  </button>
                </th>
                <th class="px-3 py-2">
                  <button type="button" class="inline-flex cursor-pointer items-center gap-0.5 select-none hover:text-foreground" @click="toggleProcSort('mem')">
                    内存%
                    <FaIcon v-if="procSortKey === 'mem'" :name="procSortOrder === 'asc' ? 'i-lucide:arrow-up' : 'i-lucide:arrow-down'" class="text-[11px]" />
                  </button>
                </th>
                <th class="hidden px-3 py-2 md:table-cell">
                  <button type="button" class="inline-flex cursor-pointer items-center gap-0.5 select-none hover:text-foreground" @click="toggleProcSort('rss')">
                    RSS
                    <FaIcon v-if="procSortKey === 'rss'" :name="procSortOrder === 'asc' ? 'i-lucide:arrow-up' : 'i-lucide:arrow-down'" class="text-[11px]" />
                  </button>
                </th>
                <th class="hidden px-3 py-2 lg:table-cell">用户</th>
                <th class="px-3 py-2 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading && !pagedProcesses.length">
                <td colspan="7" class="px-3 py-10 text-center text-muted-foreground">
                  加载中…
                </td>
              </tr>
              <tr v-else-if="!pagedProcesses.length">
                <td colspan="7" class="px-3 py-10 text-center text-muted-foreground">
                  无匹配进程
                </td>
              </tr>
              <tr v-for="p in pagedProcesses" :key="p.pid" class="border-t hover:bg-accent/30">
                <td class="px-3 py-1.5 font-mono text-xs">{{ p.pid }}</td>
                <td class="max-w-40 truncate px-3 py-1.5 font-mono text-[13px]" :title="p.cmdline">{{ p.name }}</td>
                <td class="px-3 py-1.5 text-xs tabular-nums">{{ p.cpu.toFixed(1) }}</td>
                <td class="px-3 py-1.5 text-xs tabular-nums">{{ p.mem.toFixed(1) }}</td>
                <td class="hidden px-3 py-1.5 text-xs tabular-nums text-muted-foreground md:table-cell">{{ fmtRss(p.memRss) }}</td>
                <td class="hidden px-3 py-1.5 font-mono text-xs text-muted-foreground lg:table-cell">{{ p.user }}</td>
                <td class="px-3 py-1.5 text-right">
                  <FaButton variant="outline" size="sm" @click="kill(p)">结束</FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <FaPagination v-model:page="page" v-model:size="size" :total="filteredProcesses.length" class="mt-3" />
      </template>

      <!-- 服务 -->
      <template v-else>
        <div class="mt-3 overflow-x-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">
                  <button type="button" class="inline-flex cursor-pointer items-center gap-0.5 select-none hover:text-foreground" @click="toggleSvcSort('name')">
                    服务
                    <FaIcon v-if="svcSortKey === 'name'" :name="svcSortOrder === 'asc' ? 'i-lucide:arrow-up' : 'i-lucide:arrow-down'" class="text-[11px]" />
                  </button>
                </th>
                <th class="hidden px-3 py-2 md:table-cell">加载</th>
                <th class="px-3 py-2">
                  <button type="button" class="inline-flex cursor-pointer items-center gap-0.5 select-none hover:text-foreground" @click="toggleSvcSort('active')">
                    状态
                    <FaIcon v-if="svcSortKey === 'active'" :name="svcSortOrder === 'asc' ? 'i-lucide:arrow-up' : 'i-lucide:arrow-down'" class="text-[11px]" />
                  </button>
                </th>
                <th class="hidden px-3 py-2 lg:table-cell">描述</th>
                <th class="px-3 py-2 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading && !pagedServices.length">
                <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
                  加载中…
                </td>
              </tr>
              <tr v-else-if="!pagedServices.length">
                <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
                  无匹配服务
                </td>
              </tr>
              <tr v-for="s in pagedServices" :key="s.name" class="border-t hover:bg-accent/30">
                <td class="px-3 py-1.5 font-mono text-[13px]">{{ s.name }}</td>
                <td class="hidden px-3 py-1.5 text-xs text-muted-foreground md:table-cell">{{ s.load }}</td>
                <td class="px-3 py-1.5">
                  <span class="rounded-full px-2 py-0.5 text-xs" :class="s.active === 'active' ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                    {{ s.active }}
                  </span>
                </td>
                <td class="hidden max-w-64 truncate px-3 py-1.5 text-xs text-muted-foreground lg:table-cell" :title="s.desc">{{ s.desc }}</td>
                <td class="px-3 py-1.5 text-right">
                  <div class="inline-flex items-center gap-1">
                    <FaButton v-if="s.active !== 'active'" variant="outline" size="sm" :disabled="acting === s.name + 'start'" @click="svcAction(s, 'start')">启动</FaButton>
                    <FaButton v-if="s.active === 'active'" variant="outline" size="sm" :disabled="acting === s.name + 'stop'" @click="svcAction(s, 'stop')">停止</FaButton>
                    <FaButton v-if="s.active === 'active'" variant="ghost" size="sm" :disabled="acting === s.name + 'restart'" @click="svcAction(s, 'restart')">重启</FaButton>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <FaPagination v-model:page="page" v-model:size="size" :total="filteredServices.length" class="mt-3" />
      </template>
    </FaPageMain>
  </div>
</template>
