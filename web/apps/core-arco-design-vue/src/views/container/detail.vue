<script setup lang="ts">
import apiContainer from '@/api/modules/container'
import FileEditorWorkspace from '@/views/file_management/editor/Workspace.vue'
import LogViewer from './components/LogViewer.vue'
import StatsCharts from './components/StatsCharts.vue'

defineOptions({
  name: 'ContainerDetail',
})

const route = useRoute()
const router = useRouter()
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()

const id = computed(() => String(route.params.id || ''))

const inspect = ref<Record<string, any> | null>(null)
const loading = ref(false)
const acting = ref('')
const activeTab = ref<'overview' | 'logs' | 'stats' | 'terminal' | 'files' | 'inspect'>('overview')

const name = computed(() => (inspect.value?.Name || '').replace(/^\//, '') || id.value.slice(0, 12))
const image = computed(() => inspect.value?.Config?.Image || '')
const state = computed(() => inspect.value?.State?.Status || '')
const running = computed(() => state.value === 'running')

async function load() {
  loading.value = true
  try {
    inspect.value = await apiContainer.inspect(id.value)
  }
  catch (e: any) {
    toast.error('获取容器详情失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

watch(id, () => load(), { immediate: true })

async function action(act: 'start' | 'stop' | 'restart') {
  acting.value = act
  try {
    await apiContainer.action(id.value, act)
    toast.success(`已${act === 'start' ? '启动' : act === 'stop' ? '停止' : '重启'} ${name.value}`)
    await load()
  }
  catch (e: any) {
    toast.error('操作失败', { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function remove() {
  const modal = useFaModal()
  modal.confirm({
    title: '删除容器',
    content: `确认删除容器 ${name.value}？可写层数据将一并删除，不可恢复。`,
    onConfirm: async () => {
      try {
        await apiContainer.remove(id.value, true)
        toast.success(`已删除 ${name.value}`)
        router.push('/container?tab=list')
      }
      catch (e: any) {
        toast.error('删除失败', { description: e?.message })
      }
    },
  })
}

function openWorkspaceFiles() {
  fileEditorStore.openWorkspace(undefined, 'local', id.value)
}

// ---- 终端 ----
const shell = ref('/bin/sh')
const customShell = ref('')
const sessionKey = ref(0)
const SHELLS = ['/bin/sh', '/bin/bash', '/bin/ash']
const endpoint = computed(() => ({ kind: 'exec' as const, containerId: id.value, cmd: shell.value }))
function switchShell() {
  if (customShell.value.trim()) {
    shell.value = customShell.value.trim()
    customShell.value = ''
  }
  sessionKey.value++
}

// ---- 概览数据 ----
const portLines = computed(() => {
  const ports = inspect.value?.NetworkSettings?.Ports || {}
  const out: string[] = []
  for (const [key, binds] of Object.entries<any>(ports)) {
    if (!binds?.length) {
      continue
    }
    for (const b of binds) {
      out.push(`${b.HostIp === '::' ? '[::]' : b.HostIp || '0.0.0.0'}:${b.HostPort} → ${key}`)
    }
  }
  return out
})

const netLines = computed(() => {
  const nets = inspect.value?.NetworkSettings?.Networks || {}
  return Object.entries<any>(nets).map(([n, v]) => ({
    name: n,
    ip: v.IPAddress || '—',
    gateway: v.Gateway || '—',
    mac: v.MacAddress || '',
  }))
})

const memLimitMB = computed(() => {
  const m = inspect.value?.HostConfig?.Memory || 0
  return m ? (m / 1048576).toFixed(0) : 0
})
const cpusLimit = computed(() => {
  const n = inspect.value?.HostConfig?.NanoCPUs || 0
  return n ? (n / 1e9).toFixed(2) : 0
})

const envLines = computed(() => (inspect.value?.Config?.Env || []) as string[])
const labelEntries = computed(() => Object.entries<any>(inspect.value?.Config?.Labels || {}))
const healthStatus = computed(() => inspect.value?.State?.Health?.Status || '')

const stateStyle: Record<string, string> = {
  running: 'text-emerald-600 bg-emerald-500/10',
  exited: 'text-muted-foreground bg-muted',
  paused: 'text-amber-600 bg-amber-500/10',
  created: 'text-blue-600 bg-blue-500/10',
  restarting: 'text-amber-600 bg-amber-500/10',
  dead: 'text-red-600 bg-red-500/10',
}

let timer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  timer = setInterval(() => {
    if (!acting.value && !document.hidden) {
      load()
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
    <FaPageHeader>
      <template #title>
        <div class="flex min-w-0 items-center gap-2.5">
          <FaButton variant="ghost" size="icon-sm" title="返回列表" @click="router.push('/container?tab=list')">
            <FaIcon name="i-lucide:arrow-left" class="text-base" />
          </FaButton>
          <YdAppIcon :name="image" :size="26" />
          <span class="truncate font-mono">{{ name }}</span>
          <span class="inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="stateStyle[state] || stateStyle.exited">
            <span class="inline-block size-1.5 rounded-full" :class="running ? 'animate-pulse bg-current' : 'bg-current'" />
            {{ state }}<span v-if="healthStatus" class="opacity-70">· {{ healthStatus }}</span>
          </span>
        </div>
      </template>
      <template #description>
        <span class="font-mono text-xs">{{ image }} · {{ id }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" title="刷新" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton v-if="!running" size="sm" :loading="acting === 'start'" @click="action('start')">
          启动
        </FaButton>
        <template v-else>
          <FaButton variant="outline" size="sm" :loading="acting === 'stop'" @click="action('stop')">
            停止
          </FaButton>
          <FaButton variant="outline" size="sm" :loading="acting === 'restart'" @click="action('restart')">
            重启
          </FaButton>
        </template>
        <FaButton variant="outline" size="sm" @click="openWorkspaceFiles">
          <FaIcon name="i-lucide:square-terminal" class="mr-1" /> 终端 + 文件工作台
        </FaButton>
        <FaButton variant="outline" size="sm" class="text-red-500!" @click="remove">
          <FaIcon name="i-lucide:trash-2" class="mr-1" /> 删除
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <FaTabs
        v-model="activeTab" :list="[
          { label: '概览', value: 'overview' },
          { label: '日志', value: 'logs' },
          { label: '统计', value: 'stats' },
          { label: '终端', value: 'terminal' },
          { label: '文件', value: 'files' },
          { label: 'Inspect', value: 'inspect' },
        ]"
      />

      <!-- 概览 -->
      <div v-if="activeTab === 'overview' && inspect" class="mt-3 flex flex-col gap-3">
        <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">镜像</div>
            <div class="mt-1 truncate font-mono text-xs" :title="image">{{ image }}</div>
          </div>
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">创建时间</div>
            <div class="mt-1 text-xs tabular-nums">{{ inspect.Created ? new Date(inspect.Created).toLocaleString('zh-CN', { hour12: false }) : '—' }}</div>
          </div>
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">重启策略 / 资源限制</div>
            <div class="mt-1 font-mono text-xs">
              {{ inspect.HostConfig?.RestartPolicy?.Name || 'no' }}
              <span class="text-muted-foreground">
                · {{ memLimitMB ? `${memLimitMB}MB` : '内存不限' }} · {{ cpusLimit ? `${cpusLimit}核` : 'CPU不限' }}{{ inspect.HostConfig?.Privileged ? ' · 特权' : '' }}
              </span>
            </div>
          </div>
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">TTY / 工作目录</div>
            <div class="mt-1 font-mono text-xs">
              {{ inspect.Config?.Tty ? '有 TTY' : '无 TTY' }} · {{ inspect.Config?.WorkingDir || '/' }}
            </div>
          </div>
        </div>

        <div class="grid gap-3 lg:grid-cols-2">
          <div class="rounded-md border p-3">
            <div class="mb-2 text-xs font-medium text-muted-foreground">端口映射</div>
            <div v-if="portLines.length" class="space-y-1 font-mono text-xs">
              <div v-for="(p, i) in portLines" :key="i">
                {{ p }}
              </div>
            </div>
            <div v-else class="text-xs text-muted-foreground">
              未发布端口
            </div>
          </div>
          <div class="rounded-md border p-3">
            <div class="mb-2 text-xs font-medium text-muted-foreground">网络</div>
            <table v-if="netLines.length" class="w-full text-xs">
              <thead class="text-left text-muted-foreground">
                <tr><th class="pb-1">网络</th><th class="pb-1">IP</th><th class="pb-1">网关</th></tr>
              </thead>
              <tbody class="font-mono">
                <tr v-for="n in netLines" :key="n.name">
                  <td class="py-0.5">{{ n.name }}</td>
                  <td class="py-0.5">{{ n.ip }}</td>
                  <td class="py-0.5 text-muted-foreground">{{ n.gateway }}</td>
                </tr>
              </tbody>
            </table>
            <div v-else class="text-xs text-muted-foreground">
              无网络
            </div>
          </div>
        </div>

        <div class="rounded-md border p-3">
          <div class="mb-2 text-xs font-medium text-muted-foreground">挂载</div>
          <div v-if="(inspect.Mounts || []).length" class="space-y-1 font-mono text-xs">
            <div v-for="(m, i) in inspect.Mounts" :key="i" class="truncate" :title="`${m.Source || m.Name} → ${m.Destination}`">
              <span class="rounded bg-muted px-1 text-[10px] text-muted-foreground">{{ m.Type }}</span>
              {{ m.Source || m.Name }} → {{ m.Destination }}{{ m.RW === false ? ' (ro)' : '' }}
            </div>
          </div>
          <div v-else class="text-xs text-muted-foreground">
            无挂载
          </div>
        </div>

        <div class="grid gap-3 lg:grid-cols-2">
          <div class="rounded-md border p-3">
            <div class="mb-2 text-xs font-medium text-muted-foreground">环境变量（{{ envLines.length }}）</div>
            <div class="max-h-40 space-y-0.5 overflow-auto font-mono text-xs">
              <div v-for="(e, i) in envLines" :key="i" class="break-all">
                {{ e }}
              </div>
              <div v-if="!envLines.length" class="text-muted-foreground">
                无
              </div>
            </div>
          </div>
          <div class="rounded-md border p-3">
            <div class="mb-2 text-xs font-medium text-muted-foreground">Labels（{{ labelEntries.length }}）</div>
            <div class="max-h-40 space-y-0.5 overflow-auto font-mono text-xs">
              <div v-for="[k, v] in labelEntries" :key="k" class="break-all">
                <span class="text-muted-foreground">{{ k }}</span> = {{ v }}
              </div>
              <div v-if="!labelEntries.length" class="text-muted-foreground">
                无
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 日志 -->
      <div v-if="activeTab === 'logs'" class="mt-3 h-[calc(100vh-320px)] min-h-100">
        <LogViewer :container-id="id" :container-name="name" :active="activeTab === 'logs'" />
      </div>

      <!-- 统计 -->
      <div v-if="activeTab === 'stats'" class="mt-3">
        <StatsCharts :container-id="id" :active="activeTab === 'stats'" />
      </div>

      <!-- 终端 -->
      <div v-if="activeTab === 'terminal'" class="mt-3 flex h-[calc(100vh-320px)] min-h-100 flex-col gap-2">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-xs text-muted-foreground">Shell</span>
          <div class="flex gap-1">
            <FaButton
              v-for="s in SHELLS" :key="s" size="sm"
              :variant="shell === s ? 'default' : 'outline'"
              @click="shell = s; sessionKey++"
            >
              {{ s }}
            </FaButton>
          </div>
          <FaInput v-model="customShell" placeholder="自定义命令…" class="h-8 w-44!" @keyup.enter="switchShell" />
          <FaButton size="sm" variant="outline" @click="switchShell">
            打开
          </FaButton>
        </div>
        <div class="min-h-0 flex-1 overflow-hidden rounded-md border bg-background p-1">
          <YdTerminal :key="`${id}-${shell}-${sessionKey}`" :endpoint="endpoint" :active="activeTab === 'terminal'" class="h-full" />
        </div>
      </div>

      <!-- 文件 -->
      <div v-if="activeTab === 'files'" class="mt-3 h-[calc(100vh-320px)] min-h-100">
        <YdFileTree :container-id="id" title="容器文件" @open-file="(e: any) => fileEditorStore.openWorkspace(e.path, 'local', id)" />
      </div>

      <!-- Inspect -->
      <div v-if="activeTab === 'inspect'" class="mt-3">
        <pre class="max-h-[calc(100vh-320px)] overflow-auto rounded-md bg-muted/60 p-3 font-mono text-xs leading-relaxed">{{ inspect ? JSON.stringify(inspect, null, 2) : '' }}</pre>
      </div>
    </FaPageMain>

    <!-- 文件工作台（容器模式弹窗） -->
    <FileEditorWorkspace />
  </div>
</template>
