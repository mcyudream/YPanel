<script setup lang="ts">
import type { ComposeProject } from '@/api/modules/compose'
import type { ContainerItem } from '@/api/modules/container'
import apiCompose from '@/api/modules/compose'
import apiContainer, { LabelComposeProject, LabelComposeService } from '@/api/modules/container'
import apiFile from '@/api/modules/file'
import { storeApi, type StoreAppItem, type StoreInstall } from '@/api/modules/store'
import SrcCreateWizard from '@/views/container/components/SrcCreateWizard.vue'
import { i18n, tr } from '@/locales'
import { useYwEmbed } from '@/views/desktop/embed'

// 应用 tab（M23，开发者默认视角）：compose 项目 + 商店安装聚合为应用卡片（紧凑）。
// 子容器（服务）通过「子容器」按钮在弹窗表格中查看与操作（电源下拉/日志/终端/详情）；
// 未部署（容器已移除）项目默认隐藏，开关显示后可删除清理（down + 移除编排目录，含数据）。
const router = useRouter()
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()
const appAccountStore = useAppAccountStore()

// webos 桌面承载时：详情下钻开新窗（router.push 会顶掉 /desktop 路由）；经典模式保持路由跳转
const ywEmbed = useYwEmbed()

function openAppDetail(projectName: string) {
  if (ywEmbed) {
    ywEmbed.openApp('container-app-detail', { title: projectName, launchOptions: { project: projectName } })
    return
  }
  router.push(`/container/app/${projectName}`)
}

function openContainerDetail(containerId: string) {
  if (ywEmbed) {
    ywEmbed.openApp('container-detail', { title: i18n.global.t('container.common.containerTitle', { name: containerId.slice(0, 12) }), launchOptions: { id: containerId } })
    return
  }
  router.push(`/container/detail/${containerId}`)
}

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
      dockerMsg.value = i18n.global.t('container.apps.noDockerMsg')
    }
    else {
      dockerMsg.value = e?.message || i18n.global.t('container.common.loadFailed')
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

interface AppCard {
  project: ComposeProject
  install?: StoreInstall
  iconUrl?: string
  appName?: string
}

const cards = computed<AppCard[]>(() => {
  return projects.value.map((raw) => {
    // agent 对无服务/新装项目可能返回 services: null，统一归一
    const p: ComposeProject = { ...raw, services: raw.services || [] }
    const install = installs.value.find(i => i.composeProject === p.name)
    const app = install ? storeApps.value.find(a => a.key === install.key) : undefined
    // 图标优先商店元数据外链，否则走后端代理端点（多源商店的相对 icon 路径）
    const icon = app?.iconUrl?.startsWith('http') ? app.iconUrl : (install ? storeApi.iconUrl(install.sourceId, install.key) : '')
    return { project: p, install, iconUrl: icon, appName: app?.title || app?.name }
  })
})

// 未部署（total=0，容器已移除）项目默认隐藏，开关显示后可删除清理
const showUndeployed = ref(true) // M55 默认全部展示（含未部署）
const undeployedCount = computed(() => cards.value.filter(c => c.project.total === 0).length)
const visibleCards = computed(() => showUndeployed.value ? cards.value : cards.value.filter(c => c.project.total > 0))

async function deleteProject(p: ComposeProject) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.apps.deleteProjectTitle'),
    content: i18n.global.t('container.apps.deleteProjectConfirm', { name: p.name, dir: p.dir }),
    onConfirm: async () => {
      acting.value = `del-${p.name}`
      try {
        await apiCompose.deleteProject(p.name)
        toast.success(i18n.global.t('container.apps.projectDeleted', { name: p.name }))
        await load()
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.common.deleteFailed'), { description: e?.message })
      }
      finally {
        acting.value = ''
      }
    },
  })
}

function sourceLabel(p: ComposeProject, install?: StoreInstall) {
  if (install) {
    return i18n.global.t('container.common.srcStore')
  }
  return p.managed ? i18n.global.t('container.common.srcManaged') : i18n.global.t('container.common.srcExternal')
}

// 应用整体状态：全部运行=运行中；部分=部分运行；有定义但 0 运行=未运行（total=0 表示容器未创建）
function appState(p: ComposeProject) {
  if (p.running === 0) {
    return { label: p.total === 0 ? i18n.global.t('container.apps.stateUndeployed') : i18n.global.t('container.apps.stateStopped'), cls: 'bg-muted text-muted-foreground' }
  }
  if (p.running >= p.total) {
    return { label: i18n.global.t('container.apps.stateRunning'), cls: 'bg-emerald-500/10 text-emerald-600' }
  }
  return { label: i18n.global.t('container.apps.statePartial'), cls: 'bg-amber-500/10 text-amber-600' }
}

async function doUp(p: ComposeProject) {
  acting.value = `up-${p.name}`
  try {
    await apiCompose.up(p.name, p.managed ? '' : p.dir)
    toast.success(i18n.global.t('container.common.started', { name: p.name }))
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.startFailed'), { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function confirmDown(p: ComposeProject) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.common.stopAppTitle'),
    content: i18n.global.t('container.common.stopAppConfirm', { name: p.name }),
    onConfirm: async () => {
      acting.value = `down-${p.name}`
      try {
        await apiCompose.down(p.name, p.managed ? '' : p.dir)
        toast.success(i18n.global.t('container.common.stopped', { name: p.name }))
        await load()
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.common.stopFailed'), { description: e?.message })
      }
      finally {
        acting.value = ''
      }
    },
  })
}

// ---- 子容器弹窗：服务表格 + 电源下拉 + 日志/终端/详情 ----
const containersModalOpen = ref(false)
const containersModal = ref<ComposeProject | null>(null)
const modalActing = ref('')

const modalRows = computed(() => {
  const p = containersModal.value
  if (!p) {
    return []
  }
  return (p.services || []).map(s => ({ service: s, container: containerOf(p.name, s.name) }))
})

async function modalServiceAction(service: string, action: 'start' | 'stop' | 'restart' | 'up') {
  const p = containersModal.value
  if (!p) {
    return
  }
  modalActing.value = `${service}-${action}`
  try {
    await apiCompose.serviceAction(p.name, service, action, p.managed ? '' : p.dir)
    const msgKey = action === 'up' ? 'container.common.svcRebuilt' : action === 'start' ? 'container.common.svcStarted' : action === 'stop' ? 'container.common.svcStopped' : 'container.common.svcRestarted'
    toast.success(i18n.global.t(msgKey, { name: service }))
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.apps.svcActionFailed', { name: service, action }), { description: e?.message })
  }
  finally {
    modalActing.value = ''
  }
}

function modalDropdownItems(service: string) {
  const c = modalRows.value.find(x => x.service.name === service)?.container
  const running = c?.state === 'running'
  return [[
    ...(running ? [] : [{ label: i18n.global.t('common.start'), icon: 'i-lucide:play', handle: () => modalServiceAction(service, 'start') }]),
    ...(running ? [{ label: i18n.global.t('common.restart'), icon: 'i-lucide:rotate-cw', handle: () => modalServiceAction(service, 'restart') }] : []),
    ...(running ? [{ label: i18n.global.t('common.stop'), icon: 'i-lucide:square', handle: () => modalServiceAction(service, 'stop') }] : []),
    { label: i18n.global.t('container.common.recreateByCompose'), icon: 'i-lucide:hammer', handle: () => modalServiceAction(service, 'up') },
  ]]
}

function openServiceTerminal(p: ComposeProject, service: string) {
  const c = containerOf(p.name, service)
  if (!c) {
    toast.warning(i18n.global.t('container.apps.svcNoContainer', { name: service }))
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
      svcLogsContent.value += `\n${i18n.global.t('container.common.logStreamError', { msg: e?.message || e })}`
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
    toast.warning(i18n.global.t('container.apps.externalNotEditable'))
    return
  }
  const path = await mainYamlOf(p)
  if (!path) {
    toast.error(i18n.global.t('container.apps.composeNotFound'), { description: p.dir })
    return
  }
  fileEditorStore.openWorkspace(path, 'local', undefined, p.dir)
}

async function openRevisionHistory(p: ComposeProject) {
  const path = await mainYamlOf(p)
  if (!path) {
    toast.error(i18n.global.t('container.apps.composeNotFound'), { description: p.dir })
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

// ---- 从源码创建（M26 P2） ----
const srcWizardVisible = ref(false)

async function onSrcCreated() {
  await load()
}

async function submitCreate() {
  createSaving.value = true
  try {
    if (!/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/.test(createName.value)) {
      toast.warning(i18n.global.t('container.apps.nameInvalid'))
      return
    }
    await apiCompose.write(createName.value, createContent.value)
    toast.success(i18n.global.t('container.apps.projectCreated'))
    createVisible.value = false
    await apiCompose.up(createName.value, '')
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.createFailed'), { description: e?.message })
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
  svcLogsAbort?.abort()
})
</script>

<template>
  <div>
    <FaPageMain>
  <div>
    <div v-if="dockerDisabled" class="mb-4 flex items-center gap-2 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
      <YdMorphIcon name="triangle-alert" :size="18" />
      {{ dockerMsg }}
    </div>
    <div v-else-if="dockerMsg" class="mb-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
      {{ dockerMsg }}
    </div>

    <div class="mb-3 flex flex-wrap items-center gap-3">
      <YdDockerNodeSelect />
      <label class="flex items-center gap-1.5 text-xs text-muted-foreground">
        <input v-model="showUndeployed" type="checkbox" class="accent-[var(--primary)]">
        {{ $t('container.apps.showUndeployed') }}<template v-if="!showUndeployed && undeployedCount"> ({{ undeployedCount }})</template>
      </label>
      <span class="text-xs text-muted-foreground">{{ $t('container.apps.cardHint') }}</span>
      <FaButton class="ml-auto" size="sm" :disabled="dockerDisabled" variant="outline" @click="srcWizardVisible = true">
        <FaIcon name="i-lucide:git-branch" class="mr-1" /> {{ $t('container.apps.createFromSrc') }}
      </FaButton>
      <FaButton size="sm" :disabled="dockerDisabled" @click="openCreate">
        <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('container.apps.newProject') }}
      </FaButton>
    </div>

    <div v-if="!dockerDisabled && !visibleCards.length" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
      {{ $t('container.apps.empty') }}
    </div>

    <div class="grid gap-4 xl:grid-cols-2">
      <div
        v-for="c in visibleCards"
        :key="c.project.name + c.project.dir"
        class="cursor-pointer rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
        @click="openAppDetail(c.project.name)"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="flex min-w-0 items-center gap-2.5">
            <YdAppIcon :image="c.iconUrl" :name="c.project.name" :size="30" />
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium">{{ c.appName || c.project.name }}</span>
                  <span v-if="(c.install as any)?.nodeId && (c.install as any)?.nodeId !== 'local'" class="rounded-full bg-sky-500/10 px-1.5 py-0.5 text-[10px] text-sky-600">@{{ (c.install as any)?.nodeId }}</span>
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
            {{ appState(c.project).label }}<span v-if="c.project.total > 0"> {{ c.project.running }}/{{ c.project.total }}</span>
          </span>
        </div>

        <!-- 服务摘要（紧凑）：名称 + 状态，入口在「子容器」按钮 -->
        <div v-if="(c.project.services || []).length" class="mt-2.5 flex flex-wrap items-center gap-1.5">
          <span
            v-for="s in (c.project.services || [])"
            :key="s.name"
            class="inline-flex max-w-full items-center gap-1 rounded-md border px-1.5 py-0.5 text-xs"
            :title="`${s.image} · ${s.state}`"
          >
            <span class="inline-block size-1.5 rounded-full" :class="s.state === 'running' ? 'bg-emerald-500' : 'bg-muted-foreground/40'" />
            <span class="truncate">{{ s.name }}</span>
          </span>
        </div>

        <div class="mt-3 flex flex-wrap items-center gap-1.5 border-t pt-3" @click.stop>
          <FaButton
            v-if="(c.project.services || []).length" variant="outline" size="sm"
            @click="containersModal = c.project; containersModalOpen = true"
          >
            <FaIcon name="i-lucide:boxes" class="mr-1" /> {{ $t('container.apps.subContainers', { n: (c.project.services || []).length }) }}
          </FaButton>
          <FaButton v-if="c.project.running === 0" variant="outline" size="sm" :disabled="acting === `up-${c.project.name}`" @click="doUp(c.project)">
            {{ $t('common.start') }}
          </FaButton>
          <template v-else>
            <FaButton variant="outline" size="sm" :disabled="acting === `up-${c.project.name}`" :title="$t('container.common.recreateTitle')" @click="doUp(c.project)">
              {{ $t('container.common.recreate') }}
            </FaButton>
            <FaButton variant="outline" size="sm" :disabled="acting === `down-${c.project.name}`" @click="confirmDown(c.project)">
              {{ $t('common.stop') }}
            </FaButton>
          </template>
          <FaButton variant="ghost" size="sm" @click="openAppDetail(c.project.name)">
            {{ $t('common.detail') }}
          </FaButton>
          <FaButton v-if="c.project.managed" variant="ghost" size="sm" @click="openEdit(c.project)">
            <FaIcon name="i-lucide:pen-line" class="mr-1" /> {{ $t('container.common.editConfig') }}
          </FaButton>
          <FaButton v-if="c.project.managed" variant="ghost" size="sm" :title="$t('container.common.revisionHistory')" @click="openRevisionHistory(c.project)">
            <FaIcon name="i-lucide:history" class="mr-1" /> {{ $t('container.apps.revision') }}
          </FaButton>
          <FaButton
            v-if="c.project.managed && c.project.total === 0" variant="ghost" size="sm"
            class="ml-auto text-red-500!" :disabled="acting === `del-${c.project.name}`" :title="$t('container.apps.deleteProjectTip')"
            @click="deleteProject(c.project)"
          >
            <FaIcon name="i-lucide:trash" class="mr-1" /> {{ $t('common.delete') }}
          </FaButton>
        </div>
      </div>
    </div>

    <!-- 子容器弹窗：服务表格 + 电源下拉 + 日志/终端/详情 -->
    <FaModal
      v-model="containersModalOpen"
      :title="$t('container.apps.subContainersTitle', { name: containersModal?.name || '' })"
      class="max-w-4xl!"
      :destroy-on-close="true"
    >
      <div class="overflow-x-auto rounded-md border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('container.apps.svcOrContainer') }}</th>
              <th class="px-3 py-2">{{ $t('common.status') }}</th>
              <th class="hidden px-3 py-2 md:table-cell">{{ $t('container.common.image') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!modalRows.length">
              <td colspan="4" class="px-3 py-8 text-center text-muted-foreground">
                {{ $t('container.apps.undeployedRow') }}
              </td>
            </tr>
            <tr v-for="row in modalRows" :key="row.service.name" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2">
                <div class="font-mono text-[13px] font-medium">
                  {{ row.service.name }}
                </div>
                <div class="truncate font-mono text-xs text-muted-foreground">
                  {{ row.container?.name || '—' }}
                </div>
              </td>
              <td class="px-3 py-2">
                <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs whitespace-nowrap" :class="row.container?.state === 'running' ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  <span class="inline-block size-1.5 rounded-full" :class="row.container?.state === 'running' ? 'animate-pulse bg-current' : 'bg-current'" />
                  {{ row.container ? tr(`container.state.${row.container.state}`) : $t('container.state.notCreated') }}
                </span>
              </td>
              <td class="hidden max-w-40 truncate px-3 py-2 font-mono text-xs text-muted-foreground md:table-cell" :title="row.service.image">
                {{ row.service.image }}
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton variant="ghost" size="icon-sm" :title="$t('container.common.svcLogs')" @click="containersModal && openServiceLogs(containersModal, row.service.name)">
                    <FaIcon name="i-lucide:scroll-text" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" :title="$t('container.common.openTerminal')" @click="containersModal && openServiceTerminal(containersModal, row.service.name)">
                    <FaIcon name="i-lucide:square-terminal" class="text-sm" />
                  </FaButton>
                  <FaButton
                    v-if="row.container" variant="ghost" size="icon-sm" :title="$t('container.common.containerDetail')"
                    @click="openContainerDetail(row.container.id)"
                  >
                    <FaIcon name="i-lucide:info" class="text-sm" />
                  </FaButton>
                  <FaDropdown :items="modalDropdownItems(row.service.name)">
                    <FaButton
                      variant="ghost" size="icon-sm" :title="$t('container.common.powerActions')"
                      :loading="modalActing === `${row.service.name}-start` || modalActing === `${row.service.name}-restart` || modalActing === `${row.service.name}-stop` || modalActing === `${row.service.name}-up`"
                    >
                      <FaIcon name="i-lucide:power" class="text-sm" />
                    </FaButton>
                  </FaDropdown>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="containersModalOpen = false">
          {{ $t('common.close') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 服务日志 -->
    <FaModal
      v-model="svcLogsVisible"
      :title="$t('container.common.svcLogsTitle', { name: svcLogsTarget })"
      class="max-w-5xl!"
      :destroy-on-close="true"
      @close="closeServiceLogs"
    >
      <div class="mb-2 flex items-center gap-2">
        <FaButton size="sm" :variant="svcLogsFollowing ? 'default' : 'outline'" @click="svcLogsProject && loadServiceLogs(svcLogsProject, svcLogsTarget, !svcLogsFollowing)">
          <FaIcon name="i-lucide:radio" class="mr-1" :class="svcLogsFollowing ? 'animate-pulse' : ''" />
          {{ svcLogsFollowing ? $t('container.common.following') : $t('container.common.followLogs') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="svcLogsProject && loadServiceLogs(svcLogsProject, svcLogsTarget, false)">
          {{ $t('common.refresh') }}
        </FaButton>
      </div>
      <YdLogViewer :logs="svcLogsContent" height="384px" :loading="svcLogsFollowing" />
      <template #footer>
        <FaButton variant="outline" @click="closeServiceLogs">
          {{ $t('common.close') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 新建项目 -->
    <FaModal v-model="createVisible" :title="$t('container.apps.newProjectModalTitle')" class="max-w-4xl!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('container.apps.projectName') }}</span>
          <FaInput v-model="createName" :placeholder="$t('container.apps.namePlaceholder')" class="flex-1" />
        </div>
        <textarea
          v-model="createContent"
          class="h-96 w-full resize-y rounded-md border bg-background p-3 font-mono text-[13px] leading-relaxed outline-none focus:ring-1 focus:ring-primary"
          spellcheck="false"
        />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="createSaving" @click="submitCreate">
          {{ $t('container.common.createAndStart') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 版本历史（组件自含弹窗） -->
    <YdRevisionHistory
      v-if="revisionPath" v-model="revisionVisible"
      node="local" :path="revisionPath" @restored="load()"
    />

    <!-- 从源码创建（M26 P2，组件自含弹窗） -->
    <SrcCreateWizard v-model="srcWizardVisible" @created="onSrcCreated" />

  </div>
    </FaPageMain>
  </div>
</template>
