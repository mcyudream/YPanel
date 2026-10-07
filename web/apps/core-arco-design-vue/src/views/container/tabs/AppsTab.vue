<script setup lang="ts">
import type { ComposeProject } from '@/api/modules/compose'
import type { ContainerItem } from '@/api/modules/container'
import apiCompose from '@/api/modules/compose'
import apiContainer, { LabelComposeProject, LabelComposeService } from '@/api/modules/container'
import apiFile from '@/api/modules/file'
import { storeApi, type StoreAppItem, type StoreInstall } from '@/api/modules/store'
import FileEditorWorkspace from '@/views/file_management/editor/Workspace.vue'

// 应用 tab（M23，开发者默认视角）：compose 项目 + 商店安装聚合为应用卡片；
// 卡片内直接平铺子容器（服务），支持电源下拉（启动/重启/停止/重建）、日志、终端。
const router = useRouter()
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()
const appAccountStore = useAppAccountStore()

const projects = ref<ComposeProject[]>([])
const installs = ref<StoreInstall[]>([])
const storeApps = ref<StoreAppItem[]>([])
const containers = ref<ContainerItem[]>([])
const loading = ref(false)
const dockerDisabled = ref(false)
const dockerMsg = ref('')
const acting = ref('')

async function load() {
  loading.value = true
  try {
    const [ps, inst, cs] = await Promise.all([
      apiCompose.list(),
      storeApi.installed().catch(() => [] as StoreInstall[]),
      apiContainer.list().catch(() => [] as ContainerItem[]),
    ])
    projects.value = ps
    installs.value = inst
    containers.value = cs
    if (inst.length && !storeApps.value.length) {
      const res = await storeApi.list({ pageSize: 500 }).catch(() => null)
      storeApps.value = res?.items || []
    }
    dockerDisabled.value = false
  }
  catch (e: any) {
    if (e?.code === 5002) {
      dockerDisabled.value = true
      dockerMsg.value = '未检测到可用的 Docker 环境（/var/run/docker.sock），应用管理不可用'
    }
    else {
      dockerMsg.value = e?.message || '加载失败'
    }
  }
  finally {
    loading.value = false
  }
}

function containersOf(projectName: string): ContainerItem[] {
  return containers.value.filter(c => c.labels?.[LabelComposeProject] === projectName)
}

function containerOf(projectName: string, service: string): ContainerItem | undefined {
  const list = containersOf(projectName)
  // compose 服务名（label）与展示名（容器名）可能不同，两级匹配
  return list.find(c => c.labels?.[LabelComposeService] === service)
    || list.find(c => c.name === service || c.name === `${projectName}-${service}` || c.name.startsWith(`${projectName}_${service}`))
}

// ---- 卡片内子容器（服务）操作：电源下拉 / 日志 / 终端 ----
async function serviceAction(p: ComposeProject, service: string, action: 'start' | 'stop' | 'restart' | 'up') {
  const key = `svc-${p.name}-${service}-${action}`
  acting.value = key
  try {
    await apiCompose.serviceAction(p.name, service, action, p.managed ? '' : p.dir)
    const label = action === 'up' ? '重建' : action === 'start' ? '启动' : action === 'stop' ? '停止' : '重启'
    toast.success(`已${label}服务 ${service}`)
    await load()
  }
  catch (e: any) {
    toast.error(`服务 ${service} ${action} 失败`, { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function dropdownItems(p: ComposeProject, service: string) {
  const c = containerOf(p.name, service)
  const running = c?.state === 'running'
  return [[
    ...(running ? [] : [{ label: '启动', icon: 'i-lucide:play', handle: () => serviceAction(p, service, 'start') }]),
    ...(running ? [{ label: '重启', icon: 'i-lucide:rotate-cw', handle: () => serviceAction(p, service, 'restart') }] : []),
    ...(running ? [{ label: '停止', icon: 'i-lucide:square', handle: () => serviceAction(p, service, 'stop') }] : []),
    { label: '重建（按编排定义）', icon: 'i-lucide:hammer', handle: () => serviceAction(p, service, 'up') },
  ]]
}

function openServiceTerminal(p: ComposeProject, service: string) {
  const c = containerOf(p.name, service)
  if (!c) {
    toast.warning(`服务 ${service} 当前没有容器`)
    return
  }
  fileEditorStore.openWorkspace(undefined, 'local', c.id)
  fileEditorStore.layout.terminalVisible = true
}

// ---- 服务日志弹窗（compose logs 按服务流式） ----
const svcLogsVisible = ref(false)
const svcLogsProject = ref<ComposeProject | null>(null)
const svcLogsTarget = ref('')
const svcLogsContent = ref('')
const svcLogsFollowing = ref(false)
let svcLogsAbort: AbortController | null = null

function openServiceLogs(p: ComposeProject, service: string) {
  svcLogsProject.value = p
  svcLogsTarget.value = service
  svcLogsContent.value = ''
  svcLogsVisible.value = true
  loadServiceLogs(p, service, false)
}

function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}

async function loadServiceLogs(p: ComposeProject, service: string, follow: boolean) {
  svcLogsAbort?.abort()
  svcLogsAbort = new AbortController()
  svcLogsFollowing.value = follow
  try {
    const token = appAccountStore.token
    const url = `${wsBase()}/${apiCompose.logsURL(p.name, p.managed ? '' : p.dir, token, 1000, follow, service)}`
    const resp = await fetch(url, {
      headers: { Authorization: `Bearer ${token}` },
      signal: svcLogsAbort.signal,
    })
    if (!resp.ok || !resp.body) {
      throw new Error(`HTTP ${resp.status}`)
    }
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    for (;;) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }
      svcLogsContent.value += decoder.decode(value, { stream: true })
      if (svcLogsContent.value.length > 2_000_000) {
        svcLogsContent.value = svcLogsContent.value.slice(-1_000_000)
      }
      await nextTick()
      const el = document.getElementById('svc-logs-box')
      if (el) {
        el.scrollTop = el.scrollHeight
      }
    }
  }
  catch (e: any) {
    if (e?.name !== 'AbortError') {
      svcLogsContent.value += `\n[日志流错误] ${e?.message || e}`
    }
  }
  finally {
    svcLogsFollowing.value = false
  }
}

function closeServiceLogs() {
  svcLogsAbort?.abort()
  svcLogsVisible.value = false
}

interface AppCard {
  project: ComposeProject
  install?: StoreInstall
  iconUrl?: string
  appName?: string
}

const cards = computed<AppCard[]>(() => {
  return projects.value.map((p) => {
    const install = installs.value.find(i => i.composeProject === p.name)
    const app = install ? storeApps.value.find(a => a.key === install.key) : undefined
    // 图标优先商店元数据外链，否则走后端代理端点（多源商店的相对 icon 路径）
    const icon = app?.iconUrl?.startsWith('http') ? app.iconUrl : (install ? storeApi.iconUrl(install.sourceId, install.key) : '')
    return { project: p, install, iconUrl: icon, appName: app?.title || app?.name }
  })
})

function sourceLabel(p: ComposeProject, install?: StoreInstall) {
  if (install) {
    return '商店'
  }
  return p.managed ? '托管' : '外部'
}

// 应用整体状态：全部运行=运行中；部分=部分运行；0=已停止
function appState(p: ComposeProject) {
  if (p.running === 0) {
    return { label: '已停止', cls: 'bg-muted text-muted-foreground' }
  }
  if (p.running >= p.total) {
    return { label: '运行中', cls: 'bg-emerald-500/10 text-emerald-600' }
  }
  return { label: `部分运行`, cls: 'bg-amber-500/10 text-amber-600' }
}

async function doUp(p: ComposeProject) {
  acting.value = `up-${p.name}`
  try {
    await apiCompose.up(p.name, p.managed ? '' : p.dir)
    toast.success(`已启动 ${p.name}`)
    await load()
  }
  catch (e: any) {
    toast.error('启动失败', { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function confirmDown(p: ComposeProject) {
  const modal = useFaModal()
  modal.confirm({
    title: '停止应用',
    content: `确认停止 ${p.name}？其容器/网络将被移除（数据卷保留）。`,
    onConfirm: async () => {
      acting.value = `down-${p.name}`
      try {
        await apiCompose.down(p.name, p.managed ? '' : p.dir)
        toast.success(`已停止 ${p.name}`)
        await load()
      }
      catch (e: any) {
        toast.error('停止失败', { description: e?.message })
      }
      finally {
        acting.value = ''
      }
    },
  })
}

// ---- 配置编辑：统一走文件工作台（M23） ----
// 主 compose 文件名不固定（compose.yaml / docker-compose.yml…），点击时探测目录取实际文件
const MAIN_YAML_RE = /^compose\.ya?ml$|^docker-compose\.ya?ml$/

async function mainYamlOf(p: ComposeProject): Promise<string | null> {
  try {
    const list = await apiFile.list(p.dir.replace(/\/$/, ''), 'local')
    const hit = (list.entries || []).find(e => !e.isDir && MAIN_YAML_RE.test(e.name))
    return hit ? hit.path : null
  }
  catch {
    return null
  }
}

async function openEdit(p: ComposeProject) {
  if (!p.managed) {
    toast.warning('外部项目不可在线编辑（可在目标机修改其配置文件）')
    return
  }
  const path = await mainYamlOf(p)
  if (!path) {
    toast.error('未找到项目的 compose 配置文件', { description: p.dir })
    return
  }
  fileEditorStore.openWorkspace(path, 'local')
}

async function openRevisionHistory(p: ComposeProject) {
  const path = await mainYamlOf(p)
  if (!path) {
    toast.error('未找到项目的 compose 配置文件', { description: p.dir })
    return
  }
  revisionTarget.value = p.name
  revisionPath.value = path
  revisionVisible.value = true
}

// ---- 新建项目（保留一次性模板创建，编辑走工作台） ----
const createVisible = ref(false)
const createName = ref('')
const createContent = ref('')
const createSaving = ref(false)

const TEMPLATE = `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
    restart: unless-stopped
`

function openCreate() {
  createName.value = ''
  createContent.value = TEMPLATE
  createVisible.value = true
}

async function submitCreate() {
  createSaving.value = true
  try {
    if (!/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/.test(createName.value)) {
      toast.warning('项目名仅允许字母/数字/中划线/下划线')
      return
    }
    await apiCompose.write(createName.value, createContent.value)
    toast.success('项目已创建，正在启动…')
    createVisible.value = false
    await apiCompose.up(createName.value, '')
    await load()
  }
  catch (e: any) {
    toast.error('创建失败', { description: e?.message })
  }
  finally {
    createSaving.value = false
  }
}

// ---- 版本历史弹窗 ----
const revisionVisible = ref(false)
const revisionTarget = ref('')
const revisionPath = ref('')

let timer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  load()
  timer = setInterval(() => {
    if (!dockerDisabled.value && !acting.value && !document.hidden) {
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
    <div v-if="dockerDisabled" class="mb-4 flex items-center gap-2 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
      <YdMorphIcon name="triangle-alert" :size="18" />
      {{ dockerMsg }}
    </div>
    <div v-else-if="dockerMsg" class="mb-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
      {{ dockerMsg }}
    </div>

    <div class="mb-3 flex items-center gap-2">
      <span class="text-xs text-muted-foreground">compose 项目与应用商店安装统一为「应用」；点击卡片进详情看服务/日志/终端/配置版本</span>
      <FaButton class="ml-auto" size="sm" :disabled="dockerDisabled" @click="openCreate">
        <FaIcon name="i-lucide:plus" class="mr-1" /> 新建项目
      </FaButton>
    </div>

    <div v-if="!dockerDisabled && !cards.length" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
      暂无应用，点击右上角「新建项目」开始
    </div>

    <div class="grid gap-4 xl:grid-cols-2">
      <div
        v-for="c in cards"
        :key="c.project.name + c.project.dir"
        class="cursor-pointer rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
        @click="router.push(`/container/app/${c.project.name}`)"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="flex min-w-0 items-center gap-2.5">
            <YdAppIcon :image="c.iconUrl" :name="c.project.name" :size="30" />
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium">{{ c.appName || c.project.name }}</span>
                <span
                  class="shrink-0 rounded-full px-2 py-0.5 text-xs"
                  :class="c.install ? 'bg-violet-500/10 text-violet-600' : c.project.managed ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'"
                >
                  {{ sourceLabel(c.project, c.install) }}
                </span>
                <span v-if="c.install" class="shrink-0 rounded-full bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">
                  v{{ c.install.version }}
                </span>
              </div>
              <div class="mt-0.5 truncate font-mono text-xs text-muted-foreground">
                {{ c.project.name }}
              </div>
            </div>
          </div>
          <span class="shrink-0 rounded-full px-2 py-0.5 text-xs tabular-nums" :class="appState(c.project).cls">
            {{ appState(c.project).label }} {{ c.project.running }}/{{ c.project.total }}
          </span>
        </div>

        <!-- 子容器（服务）行：状态 + 电源下拉 + 日志/终端/详情 -->
        <div v-if="(c.project.services || []).length" class="mt-3 divide-y rounded-md border bg-muted/20" @click.stop>
          <div v-for="s in (c.project.services || [])" :key="s.name" class="flex items-center gap-2 px-2.5 py-1.5">
            <span
              class="inline-block size-1.5 shrink-0 rounded-full"
              :class="s.state === 'running' ? 'animate-pulse bg-emerald-500' : 'bg-muted-foreground/40'"
              :title="s.state"
            />
            <div class="min-w-0 flex-1 leading-tight">
              <div class="truncate font-mono text-xs font-medium">
                {{ s.name }}
              </div>
              <div class="truncate text-[11px] text-muted-foreground" :title="`${s.image} · ${s.state}`">
                {{ containerOf(c.project.name, s.name)?.name || '—' }} · {{ s.state }}
              </div>
            </div>
            <FaButton variant="ghost" size="icon-sm" title="服务日志" @click="openServiceLogs(c.project, s.name)">
              <FaIcon name="i-lucide:scroll-text" class="text-sm" />
            </FaButton>
            <FaButton variant="ghost" size="icon-sm" title="进入终端" @click="openServiceTerminal(c.project, s.name)">
              <FaIcon name="i-lucide:square-terminal" class="text-sm" />
            </FaButton>
            <FaButton
              v-if="containerOf(c.project.name, s.name)" variant="ghost" size="icon-sm" title="容器详情"
              @click="router.push(`/container/detail/${containerOf(c.project.name, s.name)!.id}`)"
            >
              <FaIcon name="i-lucide:info" class="text-sm" />
            </FaButton>
            <FaDropdown :items="dropdownItems(c.project, s.name)">
              <FaButton
                variant="ghost" size="icon-sm"
                :loading="acting === `svc-${c.project.name}-${s.name}-start` || acting === `svc-${c.project.name}-${s.name}-restart` || acting === `svc-${c.project.name}-${s.name}-stop` || acting === `svc-${c.project.name}-${s.name}-up`"
                title="电源操作"
              >
                <FaIcon name="i-lucide:power" class="text-sm" />
              </FaButton>
            </FaDropdown>
          </div>
        </div>

        <div class="mt-3 flex flex-wrap items-center gap-1.5 border-t pt-3" @click.stop>
          <FaButton v-if="c.project.running === 0" variant="outline" size="sm" :disabled="acting === `up-${c.project.name}`" @click="doUp(c.project)">
            启动
          </FaButton>
          <template v-else>
            <FaButton variant="outline" size="sm" :disabled="acting === `up-${c.project.name}`" title="按当前配置重建并启动" @click="doUp(c.project)">
              重建
            </FaButton>
            <FaButton variant="outline" size="sm" :disabled="acting === `down-${c.project.name}`" @click="confirmDown(c.project)">
              停止
            </FaButton>
          </template>
          <FaButton variant="ghost" size="sm" @click="router.push(`/container/app/${c.project.name}`)">
            详情
          </FaButton>
          <FaButton v-if="c.project.managed" variant="ghost" size="sm" @click="openEdit(c.project)">
            <FaIcon name="i-lucide:pen-line" class="mr-1" /> 编辑配置
          </FaButton>
          <FaButton v-if="c.project.managed" variant="ghost" size="sm" title="版本历史" @click="openRevisionHistory(c.project)">
            <FaIcon name="i-lucide:history" class="mr-1" /> 版本
          </FaButton>
        </div>
      </div>
    </div>

    <!-- 新建项目 -->
    <FaModal v-model="createVisible" title="新建 compose 项目" class="max-w-4xl!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">项目名</span>
          <FaInput v-model="createName" placeholder="字母/数字/中划线/下划线" class="flex-1" />
        </div>
        <textarea
          v-model="createContent"
          class="h-96 w-full resize-y rounded-md border bg-background p-3 font-mono text-[13px] leading-relaxed outline-none focus:ring-1 focus:ring-primary"
          spellcheck="false"
        />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">
          取消
        </FaButton>
        <FaButton :loading="createSaving" @click="submitCreate">
          创建并启动
        </FaButton>
      </template>
    </FaModal>

    <!-- 版本历史（组件自含弹窗） -->
    <YdRevisionHistory
      v-if="revisionPath" v-model="revisionVisible"
      node="local" :path="revisionPath" @restored="load()"
    />

    <!-- 服务日志 -->
    <FaModal
      v-model="svcLogsVisible"
      :title="`服务日志：${svcLogsTarget}`"
      class="max-w-5xl!"
      :destroy-on-close="true"
      @close="closeServiceLogs"
    >
      <div class="mb-2 flex items-center gap-2">
        <FaButton size="sm" :variant="svcLogsFollowing ? 'default' : 'outline'" @click="svcLogsProject && loadServiceLogs(svcLogsProject, svcLogsTarget, !svcLogsFollowing)">
          <FaIcon name="i-lucide:radio" class="mr-1" :class="svcLogsFollowing ? 'animate-pulse' : ''" />
          {{ svcLogsFollowing ? '跟踪中（点击停止）' : '跟踪日志' }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="svcLogsProject && loadServiceLogs(svcLogsProject, svcLogsTarget, false)">
          刷新
        </FaButton>
        <span v-if="svcLogsProject && containerOf(svcLogsProject.name, svcLogsTarget)" class="text-xs text-muted-foreground">
          容器：{{ containerOf(svcLogsProject.name, svcLogsTarget)!.name }} · {{ containerOf(svcLogsProject.name, svcLogsTarget)!.state }}
        </span>
      </div>
      <pre id="svc-logs-box" class="h-96 overflow-auto rounded-md bg-muted/60 p-3 font-mono text-xs leading-relaxed">{{ svcLogsContent || '暂无日志' }}</pre>
      <template #footer>
        <FaButton variant="outline" @click="closeServiceLogs">
          关闭
        </FaButton>
      </template>
    </FaModal>

    <FileEditorWorkspace />
  </div>
</template>
