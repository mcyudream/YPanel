<script setup lang="ts">
import apiContainer from '@/api/modules/container'
import { LabelComposeProject } from '@/api/modules/container'
import { i18n, tr } from '@/locales'
import ContainerForm from './components/ContainerForm.vue'
import LogViewer from './components/LogViewer.vue'
import StatsCharts from './components/StatsCharts.vue'
import { closestWindowId, useYwEmbed } from '@/views/desktop/embed'
import { useVpnAccess } from '@/composables/useVpnAccess'
import FileEditorWorkspace from '@/views/file_management/editor/Workspace.vue'

defineOptions({
  name: 'ContainerDetail',
})

// 桌面工作台承载时经 props 传入（launchOptions），经典模式走路由参数
const props = defineProps<{
  /** 容器 ID（webos 窗口承载时由 launchOptions 注入，优先于路由参数） */
  id?: string
}>()

const route = useRoute()
const router = useRouter()
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()

const id = computed(() => String(props.id ?? (route.params.id || '')))

// 桌面承载：返回=关自己窗（列表窗仍在）；经典模式保持路由
const ywEmbed = useYwEmbed()
// 根元素 ref：用于窗口内定位自身窗 id。不能用 getCurrentInstance——computed 首次求值发生在
// 点击期而非渲染期，届时拿不到实例，selfWinId 恒为 null，返回会误走 router.push 逃逸桌面
const rootRef = ref<HTMLElement | null>(null)
const selfWinId = computed(() => closestWindowId(rootRef.value))

function onBack() {
  if (ywEmbed && selfWinId.value) {
    ywEmbed.closeWindow(selfWinId.value)
    return
  }
  router.push('/container/list')
}

const inspect = ref<Record<string, any> | null>(null)
const lastErr = ref<{ action: string, message: string, at: string } | null>(null)
const loading = ref(false)
const acting = ref('')
const activeTab = ref<'overview' | 'logs' | 'stats' | 'terminal' | 'files' | 'inspect'>('overview')

const name = computed(() => (inspect.value?.Name || '').replace(/^\//, '') || id.value.slice(0, 12))
const image = computed(() => inspect.value?.Config?.Image || '')
const state = computed(() => inspect.value?.State?.Status || '')
const running = computed(() => state.value === 'running')

// 可写层宿主目录（inspect 已含 GraphDriver）：供「可写层直读」文件源
const upperDir = computed(() => inspect.value?.GraphDriver?.Data?.UpperDir || '')
const fileSource = ref<'archive' | 'upperdir'>('archive')

async function load() {
  loading.value = true
  try {
    inspect.value = await apiContainer.inspect(id.value)
    // 最近一次启动失败原因（面板内存留存；启动成功后服务端已清除）
    lastErr.value = await apiContainer.lastOpErr(id.value).catch(() => null)
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.detail.loadFailed'), { description: e?.message })
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
    toast.success(i18n.global.t(act === 'start' ? 'container.common.started' : act === 'stop' ? 'container.common.stopped' : 'container.common.restarted', { name: name.value }))
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.opFailed'), { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function remove() {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.detail.deleteTitle'),
    content: i18n.global.t('container.detail.deleteConfirm', { name: name.value }),
    onConfirm: async () => {
      try {
        await apiContainer.remove(id.value, true)
        toast.success(i18n.global.t('container.common.deletedName', { name: name.value }))
        onBack()
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.common.deleteFailed'), { description: e?.message })
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
// 组网开启时：端口映射给出经虚拟 IP 的一键跳转
const { portJumpUrl } = useVpnAccess()

function openPortJump(url: string) {
  if (url) {
    window.open(url, '_blank', 'noopener')
  }
}

const portItems = computed(() => {
  const ports = inspect.value?.NetworkSettings?.Ports || {}
  const out: { text: string, hostIp: string, hostPort: string, proto: string }[] = []
  for (const [key, binds] of Object.entries<any>(ports)) {
    if (!binds?.length) {
      continue
    }
    for (const b of binds) {
      const shown = b.HostIp === '::' ? '[::]' : b.HostIp || '0.0.0.0'
      out.push({
        text: `${shown}:${b.HostPort} → ${key}`,
        hostIp: b.HostIp || '',
        hostPort: b.HostPort,
        proto: key.split('/')[1] || 'tcp',
      })
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

// compose 编排管理的容器：重建后脱离编排追踪，编辑入口禁用（引导去编排/应用处修改）
const composeManaged = computed(() => !!inspect.value?.Config?.Labels?.[LabelComposeProject])

// 编辑（ContainerForm edit 模式）：保存重建后容器 ID 会变，跳转到新详情
const editVisible = ref(false)
function onSaved(newId: string) {
  if (newId && newId !== id.value) {
    router.replace(`/container/detail/${newId}`)
    return
  }
  load()
}

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
  <div ref="rootRef">
    <FaPageHeader>
      <template #title>
        <div class="flex min-w-0 items-center gap-2.5">
          <FaButton variant="ghost" size="icon-sm" :title="$t('container.detail.backToListTitle')" @click="onBack">
            <FaIcon name="i-lucide:arrow-left" class="text-base" />
          </FaButton>
          <YdAppIcon :name="image" :size="26" />
          <span class="truncate font-mono">{{ name }}</span>
          <span class="inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="stateStyle[state] || stateStyle.exited">
            <span class="inline-block size-1.5 rounded-full" :class="running ? 'animate-pulse bg-current' : 'bg-current'" />
            {{ tr(`container.state.${state}`) }}<span v-if="healthStatus" class="opacity-70">· {{ healthStatus }}</span>
          </span>
        </div>
      </template>
      <template #description>
        <span class="font-mono text-xs">{{ image }} · {{ id }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" :title="$t('common.refresh')" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton v-if="!running" size="sm" :loading="acting === 'start'" @click="action('start')">
          {{ $t('common.start') }}
        </FaButton>
        <template v-else>
          <FaButton variant="outline" size="sm" :loading="acting === 'stop'" @click="action('stop')">
            {{ $t('common.stop') }}
          </FaButton>
          <FaButton variant="outline" size="sm" :loading="acting === 'restart'" @click="action('restart')">
            {{ $t('common.restart') }}
          </FaButton>
        </template>
        <FaButton
          variant="outline" size="sm"
          :class="composeManaged ? 'cursor-not-allowed opacity-50' : ''"
          :disabled="composeManaged" :title="composeManaged ? $t('container.common.managedByCompose') : $t('container.detail.editContainerConfig')"
          @click="!composeManaged && (editVisible = true)"
        >
          <FaIcon name="i-lucide:pencil" class="mr-1" /> {{ $t('common.edit') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openWorkspaceFiles">
          <FaIcon name="i-lucide:square-terminal" class="mr-1" /> {{ $t('container.detail.terminalAndFiles') }}
        </FaButton>
        <FaButton variant="outline" size="sm" class="text-red-500!" @click="remove">
          <FaIcon name="i-lucide:trash-2" class="mr-1" /> {{ $t('common.delete') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 最近启动/重启失败原因（排障指引） -->
      <div v-if="lastErr && !running" class="mb-3 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400">
        <div class="flex items-center gap-1.5 font-medium">
          <FaIcon name="i-lucide:triangle-alert" class="text-sm" />
          {{ $t('container.detail.lastFailed', { action: lastErr.action === 'restart' ? $t('common.restart') : $t('common.start'), time: new Date(lastErr.at).toLocaleString('zh-CN', { hour12: false }) }) }}
        </div>
        <div class="mt-1 break-all font-mono text-xs leading-relaxed">
          {{ lastErr.message }}
        </div>
        <div class="mt-1.5 text-xs opacity-80">
          {{ $t('container.detail.troubleshootTip') }}
        </div>
      </div>

      <FaTabs
        v-model="activeTab" :list="[
          { label: $t('container.detail.tabOverview'), value: 'overview' },
          { label: $t('container.detail.tabLogs'), value: 'logs' },
          { label: $t('container.detail.tabStats'), value: 'stats' },
          { label: $t('container.detail.tabTerminal'), value: 'terminal' },
          { label: $t('container.detail.tabFiles'), value: 'files' },
          { label: $t('container.detail.tabInspect'), value: 'inspect' },
        ]"
      />

      <!-- 概览 -->
      <div v-if="activeTab === 'overview' && inspect" class="mt-3 flex flex-col gap-3">
        <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">{{ $t('container.common.image') }}</div>
            <div class="mt-1 truncate font-mono text-xs" :title="image">{{ image }}</div>
          </div>
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">{{ $t('container.common.createdAt') }}</div>
            <div class="mt-1 text-xs tabular-nums">{{ inspect.Created ? new Date(inspect.Created).toLocaleString('zh-CN', { hour12: false }) : '—' }}</div>
          </div>
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">{{ $t('container.detail.restartAndLimits') }}</div>
            <div class="mt-1 font-mono text-xs">
              {{ inspect.HostConfig?.RestartPolicy?.Name || 'no' }}
              <span class="text-muted-foreground">
                · {{ memLimitMB ? `${memLimitMB}MB` : $t('container.detail.memUnlimited') }} · {{ cpusLimit ? $t('container.detail.cpusUnit', { n: cpusLimit }) : $t('container.detail.cpuUnlimited') }}{{ inspect.HostConfig?.Privileged ? ` · ${$t('container.detail.privilegedShort')}` : '' }}
              </span>
            </div>
          </div>
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">{{ $t('container.detail.ttyAndWorkdir') }}</div>
            <div class="mt-1 font-mono text-xs">
              {{ inspect.Config?.Tty ? $t('container.detail.ttyYes') : $t('container.detail.ttyNo') }} · {{ inspect.Config?.WorkingDir || '/' }}
            </div>
          </div>
        </div>

        <div class="grid gap-3 lg:grid-cols-2">
          <div class="rounded-md border p-3">
            <div class="mb-2 text-xs font-medium text-muted-foreground">{{ $t('container.common.portMappings') }}</div>
            <div v-if="portItems.length" class="space-y-1 font-mono text-xs">
              <div v-for="(p, i) in portItems" :key="i" class="flex items-center gap-1.5">
                <span>{{ p.text }}</span>
                <button
                  v-if="portJumpUrl(p.hostIp, p.hostPort, p.proto)" type="button"
                  class="cursor-pointer text-muted-foreground transition-colors hover:text-primary"
                  :title="$t('container.common.portJumpTip', { url: portJumpUrl(p.hostIp, p.hostPort, p.proto) })"
                  @click="openPortJump(portJumpUrl(p.hostIp, p.hostPort, p.proto))"
                >
                  <FaIcon name="i-lucide:external-link" class="size-3 shrink-0" />
                </button>
              </div>
            </div>
            <div v-else class="text-xs text-muted-foreground">
              {{ $t('container.detail.noPorts') }}
            </div>
          </div>
          <div class="rounded-md border p-3">
            <div class="mb-2 text-xs font-medium text-muted-foreground">{{ $t('container.common.network') }}</div>
            <table v-if="netLines.length" class="w-full text-xs">
              <thead class="text-left text-muted-foreground">
                <tr><th class="pb-1">{{ $t('container.common.network') }}</th><th class="pb-1">IP</th><th class="pb-1">{{ $t('container.detail.gateway') }}</th></tr>
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
              {{ $t('container.detail.noNetwork') }}
            </div>
          </div>
        </div>

        <div class="rounded-md border p-3">
          <div class="mb-2 text-xs font-medium text-muted-foreground">{{ $t('container.detail.mounts') }}</div>
          <div v-if="(inspect.Mounts || []).length" class="space-y-1 font-mono text-xs">
            <div v-for="(m, i) in inspect.Mounts" :key="i" class="truncate" :title="`${m.Source || m.Name} → ${m.Destination}`">
              <span class="rounded bg-muted px-1 text-[10px] text-muted-foreground">{{ m.Type }}</span>
              {{ m.Source || m.Name }} → {{ m.Destination }}{{ m.RW === false ? ' (ro)' : '' }}
            </div>
          </div>
          <div v-else class="text-xs text-muted-foreground">
            {{ $t('container.detail.noMounts') }}
          </div>
        </div>

        <div class="grid gap-3 lg:grid-cols-2">
          <div class="rounded-md border p-3">
            <div class="mb-2 text-xs font-medium text-muted-foreground">{{ $t('container.detail.envVars', { n: envLines.length }) }}</div>
            <div class="max-h-40 space-y-0.5 overflow-auto font-mono text-xs">
              <div v-for="(e, i) in envLines" :key="i" class="break-all">
                {{ e }}
              </div>
              <div v-if="!envLines.length" class="text-muted-foreground">
                {{ $t('common.none') }}
              </div>
            </div>
          </div>
          <div class="rounded-md border p-3">
            <div class="mb-2 text-xs font-medium text-muted-foreground">{{ $t('container.detail.labels', { n: labelEntries.length }) }}</div>
            <div class="max-h-40 space-y-0.5 overflow-auto font-mono text-xs">
              <div v-for="[k, v] in labelEntries" :key="k" class="break-all">
                <span class="text-muted-foreground">{{ k }}</span> = {{ v }}
              </div>
              <div v-if="!labelEntries.length" class="text-muted-foreground">
                {{ $t('common.none') }}
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
          <FaInput v-model="customShell" :placeholder="$t('container.detail.customShellPlaceholder')" class="h-8 w-44!" @keyup.enter="switchShell" />
          <FaButton size="sm" variant="outline" @click="switchShell">
            {{ $t('container.detail.open') }}
          </FaButton>
        </div>
        <div class="min-h-0 flex-1 overflow-hidden rounded-md border bg-background p-1">
          <YdTerminal :key="`${id}-${shell}-${sessionKey}`" :endpoint="endpoint" :active="activeTab === 'terminal'" class="h-full" />
        </div>
      </div>

      <!-- 文件 -->
      <div v-if="activeTab === 'files'" class="mt-3 flex h-[calc(100vh-320px)] min-h-100 flex-col gap-2">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-xs text-muted-foreground">{{ $t('container.detail.fileSource') }}</span>
          <div class="flex overflow-hidden rounded-md border text-xs">
            <button
              type="button" class="px-2.5 py-1 transition-colors"
              :class="fileSource === 'archive' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent/50'"
              @click="fileSource = 'archive'"
            >
              {{ $t('container.detail.srcArchive') }}
            </button>
            <button
              v-if="upperDir" type="button" class="px-2.5 py-1 transition-colors"
              :class="fileSource === 'upperdir' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent/50'"
              @click="fileSource = 'upperdir'"
            >
              {{ $t('container.detail.srcUpperdir') }}
            </button>
          </div>
          <span class="text-xs text-muted-foreground">
            {{ fileSource === 'upperdir' ? $t('container.detail.upperdirHint') : upperDir ? $t('container.detail.archiveFallbackHint') : $t('container.detail.noUpperdirHint') }}
          </span>
        </div>
        <div class="min-h-0 flex-1">
          <YdFileTree
            v-if="fileSource === 'archive'" :key="`arch-${id}`" :container-id="id" :title="$t('container.detail.filesTitle')"
            @open-file="(e: any) => fileEditorStore.openWorkspace(e.path, 'local', id)"
          />
          <YdFileTree
            v-else :key="`up-${id}`" node="local" :initial-path="upperDir" :title="$t('container.detail.upperdirTitle')"
            @open-file="(e: any) => fileEditorStore.openWorkspace(e.path, 'local')"
          />
        </div>
      </div>

      <!-- Inspect -->
      <div v-if="activeTab === 'inspect'" class="mt-3">
        <YdJsonViewer :data="inspect" :default-expand-level="2" />
      </div>
    </FaPageMain>

    <!-- 编辑（保存后重建） -->
    <ContainerForm v-model="editVisible" mode="edit" :container-id="id" @saved="onSaved" />

    <!-- 文件工作台（容器模式弹窗） -->
    <FileEditorWorkspace />
  </div>
</template>
