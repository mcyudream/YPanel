<script setup lang="ts">
import type { ContainerItem } from '@/api/modules/container'
import { LabelComposeProject } from '@/api/modules/container'
import apiContainer from '@/api/modules/container'
import { dockerExtApi } from '@/api/modules/dockerext'
import FileEditorWorkspace from '@/views/file_management/editor/Workspace.vue'
import ContainerForm from '../components/ContainerForm.vue'

// 容器列表（M23）：批量操作 / 状态与项目过滤 / 品牌 logo / 快速操作。
const router = useRouter()
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()

const containers = ref<ContainerItem[]>([])
const loading = ref(false)
const disabled = ref(false)
const disabledMsg = ref('')
const acting = ref('')
const autoRefresh = ref(true)
const search = ref('')
const stateFilter = ref<'all' | 'running' | 'stopped'>('all')
const projectFilter = ref('all')

const selected = ref<Set<string>>(new Set())

const composeProjects = computed(() => {
  const set = new Set<string>()
  for (const c of containers.value) {
    const p = c.labels?.[LabelComposeProject]
    if (p) {
      set.add(p)
    }
  }
  return [...set].sort()
})

const filtered = computed(() => {
  const kw = search.value.trim().toLowerCase()
  return containers.value.filter((c) => {
    if (stateFilter.value === 'running' && c.state !== 'running') {
      return false
    }
    if (stateFilter.value === 'stopped' && c.state === 'running') {
      return false
    }
    if (projectFilter.value !== 'all' && c.labels?.[LabelComposeProject] !== projectFilter.value) {
      return false
    }
    if (kw && !c.name.toLowerCase().includes(kw) && !c.image.toLowerCase().includes(kw) && !c.id.toLowerCase().includes(kw)) {
      return false
    }
    return true
  })
})

const allChecked = computed(() => filtered.value.length > 0 && filtered.value.every(c => selected.value.has(c.id)))

// ---- 分页（筛选/搜索变化自动回第 1 页） ----
const page = ref(1)
const size = ref(20)
const paged = computed(() => filtered.value.slice((page.value - 1) * size.value, page.value * size.value))
watch([search, stateFilter, projectFilter], () => {
  page.value = 1
})

function toggleAll() {
  if (allChecked.value) {
    selected.value.clear()
  }
  else {
    for (const c of filtered.value) {
      selected.value.add(c.id)
    }
  }
  selected.value = new Set(selected.value)
}

function toggle(id: string) {
  const next = new Set(selected.value)
  if (next.has(id)) {
    next.delete(id)
  }
  else {
    next.add(id)
  }
  selected.value = next
}

function projectOf(c: ContainerItem) {
  return c.labels?.[LabelComposeProject] || ''
}

const stateStyle: Record<string, string> = {
  running: 'text-emerald-600 bg-emerald-500/10',
  exited: 'text-muted-foreground bg-muted',
  paused: 'text-amber-600 bg-amber-500/10',
  created: 'text-blue-600 bg-blue-500/10',
  restarting: 'text-amber-600 bg-amber-500/10',
  dead: 'text-red-600 bg-red-500/10',
}

async function load(silent = false) {
  if (!silent) {
    loading.value = true
  }
  try {
    containers.value = await apiContainer.list()
    disabled.value = false
    // 清掉已消失的选中项
    const ids = new Set(containers.value.map(c => c.id))
    const next = new Set([...selected.value].filter(x => ids.has(x)))
    if (next.size !== selected.value.size) {
      selected.value = next
    }
  }
  catch (e: any) {
    if (e?.code === 5002) {
      disabled.value = true
      disabledMsg.value = e?.message || '当前节点未检测到 Docker'
    }
    else {
      disabledMsg.value = e?.message || '加载失败'
    }
  }
  finally {
    loading.value = false
  }
}

async function action(c: ContainerItem, act: 'start' | 'stop' | 'restart') {
  acting.value = c.id + act
  try {
    await apiContainer.action(c.id, act)
    toast.success(`已${act === 'start' ? '启动' : act === 'stop' ? '停止' : '重启'} ${c.name}`)
    await load(true)
  }
  catch (e: any) {
    toast.error('操作失败', { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function removeOne(c: ContainerItem) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除容器',
    content: `确认删除容器 ${c.name}？可写层数据将一并删除，不可恢复。`,
    onConfirm: () => batchRemove([c]),
  })
}

const bulkRunning = ref(false)

async function bulkAction(act: 'start' | 'stop' | 'restart') {
  const list = containers.value.filter(c => selected.value.has(c.id))
  if (!list.length) {
    return
  }
  bulkRunning.value = true
  const results = await Promise.allSettled(list.map(c => apiContainer.action(c.id, act)))
  const ok = results.filter(r => r.status === 'fulfilled').length
  const failed = results.length - ok
  if (failed) {
    toast.warning(`批量${act === 'start' ? '启动' : act === 'stop' ? '停止' : '重启'}完成：${ok} 成功，${failed} 失败`)
  }
  else {
    toast.success(`已批量${act === 'start' ? '启动' : act === 'stop' ? '停止' : '重启'} ${ok} 个容器`)
  }
  await load(true)
  bulkRunning.value = false
}

async function batchRemove(list: ContainerItem[]) {
  bulkRunning.value = true
  try {
    const results = await Promise.allSettled(list.map(c => apiContainer.remove(c.id, true)))
    const ok = results.filter(r => r.status === 'fulfilled').length
    toast.success(`已删除 ${ok}/${list.length} 个容器`)
    selected.value = new Set()
    await load(true)
  }
  finally {
    bulkRunning.value = false
  }
}

function bulkRemove() {
  const list = containers.value.filter(c => selected.value.has(c.id))
  if (!list.length) {
    return
  }
  const modal = useFaModal()
  modal.confirm({
    title: '批量删除容器',
    content: `确认删除选中的 ${list.length} 个容器？可写层数据将一并删除，不可恢复。`,
    onConfirm: () => batchRemove(list),
  })
}

function openExec(c: ContainerItem) {
  fileEditorStore.openWorkspace(undefined, 'local', c.id)
  fileEditorStore.layout.terminalVisible = true
}

async function pruneContainers() {
  const modal = useFaModal()
  modal.confirm({
    title: '清理停止容器',
    content: '确认清理全部已停止容器？',
    onConfirm: async () => {
      try {
        const out = await dockerExtApi.pruneContainers()
        useFaToast().success(out)
        await load(true)
      }
      catch (e: any) {
        toast.error('清理失败', { description: e?.message })
      }
    },
  })
}

function fmtPorts(c: ContainerItem) {
  if (!c.ports?.length) {
    return ''
  }
  return c.ports.filter(p => p.hostPort).map(p => `${p.hostPort}→${p.containerPort}/${p.proto}`).join(' ')
}

const createVisible = ref(false)
// 编辑（删除重建式）：compose 管理的容器禁用入口
const editVisible = ref(false)
const editId = ref('')

function openEdit(c: ContainerItem) {
  editId.value = c.id
  editVisible.value = true
}

const isCompose = (c: ContainerItem) => !!c.labels?.[LabelComposeProject]

let timer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  load()
  timer = setInterval(() => {
    if (autoRefresh.value && !acting.value && !bulkRunning.value && !document.hidden) {
      load(true)
    }
  }, 10000)
})
onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<template>
  <div>
    <FaPageMain>
  <div>
    <!-- 工具条 -->
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <FaInput v-model="search" placeholder="搜索名称/镜像/ID…" class="h-8 w-52!" />
      <div class="flex overflow-hidden rounded-md border text-xs">
        <button
          v-for="s in [{ v: 'all', l: '全部' }, { v: 'running', l: '运行中' }, { v: 'stopped', l: '已停止' }]" :key="s.v"
          type="button"
          class="px-2.5 py-1.5 transition-colors"
          :class="stateFilter === s.v ? 'bg-primary text-primary-foreground' : 'hover:bg-accent/50'"
          @click="stateFilter = s.v as any"
        >
          {{ s.l }}
        </button>
      </div>
      <select v-model="projectFilter" class="h-8 rounded-md border bg-background px-2 text-xs outline-none">
        <option value="all">全部项目</option>
        <option v-for="p in composeProjects" :key="p" :value="p">
          {{ p }}
        </option>
      </select>
      <label class="flex items-center gap-1 text-xs text-muted-foreground">
        <input v-model="autoRefresh" type="checkbox" class="accent-[var(--primary)]">
        自动刷新
      </label>
      <div class="ml-auto flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" title="刷新" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton variant="outline" size="sm" @click="pruneContainers">
          清理停止容器
        </FaButton>
        <FaButton size="sm" @click="createVisible = true">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 创建容器
        </FaButton>
      </div>
    </div>

    <!-- 批量操作条 -->
    <div v-if="selected.size" class="mb-3 flex flex-wrap items-center gap-2 rounded-md border bg-primary/5 px-3 py-2">
      <span class="text-sm">已选 <b class="tabular-nums">{{ selected.size }}</b> 个</span>
      <FaButton variant="outline" size="sm" :disabled="bulkRunning" @click="bulkAction('start')">
        批量启动
      </FaButton>
      <FaButton variant="outline" size="sm" :disabled="bulkRunning" @click="bulkAction('stop')">
        批量停止
      </FaButton>
      <FaButton variant="outline" size="sm" :disabled="bulkRunning" @click="bulkAction('restart')">
        批量重启
      </FaButton>
      <FaButton variant="outline" size="sm" class="text-red-500!" :disabled="bulkRunning" @click="bulkRemove">
        批量删除
      </FaButton>
      <FaButton variant="ghost" size="sm" @click="selected = new Set()">
        取消选择
      </FaButton>
    </div>

    <div v-if="disabled" class="mb-4 flex items-center gap-2 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
      <YdMorphIcon name="triangle-alert" :size="18" />
      {{ disabledMsg }}：未检测到可用的 Docker 环境（/var/run/docker.sock）。
    </div>
    <div v-else-if="disabledMsg" class="mb-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
      {{ disabledMsg }}
    </div>

    <!-- 列表 -->
    <div class="overflow-x-auto rounded-lg border">
      <table class="w-full min-w-200 text-sm">
        <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
          <tr>
            <th class="w-9 px-3 py-2">
              <input type="checkbox" :checked="allChecked" class="accent-[var(--primary)]" @change="toggleAll">
            </th>
            <th class="px-3 py-2">名称</th>
            <th class="px-3 py-2">镜像</th>
            <th class="px-3 py-2">状态</th>
            <th class="hidden px-3 py-2 lg:table-cell">端口</th>
            <th class="hidden px-3 py-2 xl:table-cell">项目</th>
            <th class="px-3 py-2 text-right">快速操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !filtered.length">
            <td colspan="7" class="px-3 py-10 text-center text-muted-foreground">
              加载中…
            </td>
          </tr>
          <tr v-else-if="!filtered.length">
            <td colspan="7" class="px-3 py-10 text-center text-muted-foreground">
              暂无容器
            </td>
          </tr>
          <tr v-for="c in paged" :key="c.id" class="border-t transition-colors hover:bg-accent/30" :class="selected.has(c.id) ? 'bg-primary/5' : ''">
            <td class="px-3 py-2">
              <input type="checkbox" :checked="selected.has(c.id)" class="accent-[var(--primary)]" @change="toggle(c.id)">
            </td>
            <td class="max-w-64 px-3 py-2">
              <div class="flex items-center gap-2">
                <YdAppIcon :name="c.image" :size="20" />
                <button type="button" class="cursor-pointer truncate font-mono text-[13px] font-medium hover:text-primary" :title="c.name" @click="router.push(`/container/detail/${c.id}`)">
                  {{ c.name }}
                </button>
              </div>
            </td>
            <td class="max-w-48 truncate px-3 py-2 font-mono text-xs text-muted-foreground" :title="c.image">
              {{ c.image }}
            </td>
            <td class="px-3 py-2">
              <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs whitespace-nowrap" :class="stateStyle[c.state] || stateStyle.exited">
                <span class="inline-block size-1.5 rounded-full" :class="c.state === 'running' ? 'animate-pulse bg-current' : 'bg-current'" />
                {{ c.state }}<span v-if="c.status" class="hidden opacity-70 xl:inline">· {{ c.status }}</span>
              </span>
            </td>
            <td class="hidden px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell">
              {{ fmtPorts(c) || '—' }}
            </td>
            <td class="hidden px-3 py-2 xl:table-cell">
              <button
                v-if="projectOf(c)" type="button"
                class="cursor-pointer rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary hover:bg-primary/20"
                @click="router.push(`/container/app/${projectOf(c)}`)"
              >
                {{ projectOf(c) }}
              </button>
              <span v-else class="text-xs text-muted-foreground">—</span>
            </td>
            <td class="px-3 py-2">
              <div class="flex items-center justify-end gap-1">
                <FaButton v-if="c.state !== 'running'" variant="outline" size="sm" :disabled="acting === c.id + 'start'" @click="action(c, 'start')">
                  启动
                </FaButton>
                <template v-else>
                  <FaButton variant="outline" size="sm" :disabled="acting === c.id + 'stop'" @click="action(c, 'stop')">
                    停止
                  </FaButton>
                  <FaButton variant="outline" size="sm" :disabled="acting === c.id + 'restart'" @click="action(c, 'restart')">
                    重启
                  </FaButton>
                </template>
                <FaButton v-if="c.state === 'running'" variant="ghost" size="icon-sm" title="终端 + 文件" @click="openExec(c)">
                  <FaIcon name="i-lucide:square-terminal" class="text-sm" />
                </FaButton>
                <FaButton
                  variant="ghost" size="icon-sm"
                  :class="isCompose(c) ? 'cursor-not-allowed opacity-40' : ''"
                  :disabled="isCompose(c)" :title="isCompose(c) ? '由 Compose 编排管理，请在编排/应用处修改' : '编辑'"
                  @click="!isCompose(c) && openEdit(c)"
                >
                  <FaIcon name="i-lucide:pencil" class="text-sm" />
                </FaButton>
                <FaButton variant="ghost" size="icon-sm" title="详情" @click="router.push(`/container/detail/${c.id}`)">
                  <FaIcon name="i-lucide:info" class="text-sm" />
                </FaButton>
                <FaButton variant="ghost" size="icon-sm" title="删除" class="text-red-500!" :disabled="acting === c.id + 'rm'" @click="removeOne(c)">
                  <FaIcon name="i-lucide:trash" class="text-sm" />
                </FaButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <FaPagination v-model:page="page" v-model:size="size" :total="filtered.length" class="mt-3" />

    <ContainerForm v-model="createVisible" mode="create" @created="load(true)" />
    <ContainerForm v-model="editVisible" mode="edit" :container-id="editId" @saved="load(true)" />
    <FileEditorWorkspace />
  </div>
    </FaPageMain>
  </div>
</template>
