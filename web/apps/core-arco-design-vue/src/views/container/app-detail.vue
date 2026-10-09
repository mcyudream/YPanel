<script setup lang="ts">
import type { ComposeProject, ComposeTopology } from '@/api/modules/compose'
import apiCompose from '@/api/modules/compose'
import apiContainer, { LabelComposeProject, LabelComposeService, type ContainerItem } from '@/api/modules/container'
import apiFile from '@/api/modules/file'
import { storeApi, type StoreAppItem, type StoreInstall } from '@/api/modules/store'
import { i18n, tr } from '@/locales'
import FileEditorWorkspace from '@/views/file_management/editor/Workspace.vue'
import { computed } from 'vue'
import { closestWindowId, useYwEmbed } from '@/views/desktop/embed'

// 应用详情（M23）：服务列表 + 配置文件（工作台编辑 + 服务端版本历史）+ 项目容器。
// 桌面工作台承载时经 props 传入（launchOptions），经典模式走路由参数
const props = defineProps<{
  /** compose 项目名（webos 窗口承载时由 launchOptions 注入，优先于路由参数） */
  project?: string
}>()

const route = useRoute()
const router = useRouter()
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()
const appAccountStore = useAppAccountStore()

const project = computed(() => String(props.project ?? (route.params.project || '')))

// 桌面承载：返回=关自己窗、下钻容器=开新窗；经典模式保持路由
const ywEmbed = useYwEmbed()
// 根元素 ref：用于窗口内定位自身窗 id。不能用 getCurrentInstance——computed 首次求值发生在
// 点击期而非渲染期，届时拿不到实例，selfWinId 恒为 null，关闭窗会静默失效
const rootRef = ref<HTMLElement | null>(null)
const selfWinId = computed(() => closestWindowId(rootRef.value))

function closeSelf() {
  const id = selfWinId.value
  if (id && ywEmbed) {
    ywEmbed.closeWindow(id)
  }
}

function openContainerDetailWin(containerId: string) {
  if (ywEmbed) {
    ywEmbed.openApp('container-detail', { title: i18n.global.t('container.common.containerTitle', { name: containerId.slice(0, 12) }), launchOptions: { id: containerId } })
    return
  }
  router.push(`/container/detail/${containerId}`)
}

function onBack() {
  if (ywEmbed) {
    closeSelf()
    return
  }
  router.push('/container/apps')
}

const info = ref<ComposeProject | null>(null)
const install = ref<StoreInstall | null>(null)
const iconUrl = ref('')
const appName = ref('')
const containers = ref<ContainerItem[]>([])
const dirFiles = ref<{ name: string, path: string, isDir: boolean }[]>([])
const loading = ref(false)
const acting = ref('')

async function load() {
  loading.value = true
  try {
    const [projects, installs] = await Promise.all([
      apiCompose.list(),
      storeApi.installed().catch(() => [] as StoreInstall[]),
    ])
    info.value = projects.find(p => p.name === project.value) || null
    if (info.value) {
      // services 可能为 null（新装/无服务项目），归一防御
      info.value = { ...info.value, services: info.value.services || [] }
    }
    install.value = installs.find(i => i.composeProject === project.value) || null
    if (install.value) {
      const res = await storeApi.list({ pageSize: 500 }).catch(() => null)
      const apps: StoreAppItem[] = res?.items || []
      const app = apps.find(a => a.key === install.value!.key)
      // 图标优先商店元数据外链，否则走后端代理端点
      iconUrl.value = app?.iconUrl?.startsWith('http')
        ? app.iconUrl
        : storeApi.iconUrl(install.value.sourceId, install.value.key)
      appName.value = app?.title || app?.name || ''
    }
    const all = await apiContainer.list().catch(() => [] as ContainerItem[])
    containers.value = all.filter(c => c.labels?.[LabelComposeProject] === project.value)
    // 配置目录文件（托管项目）
    if (info.value?.managed) {
      const list = await apiFile.list(info.value.dir, 'local').catch(() => null)
      dirFiles.value = (list?.entries || [])
        .filter(e => !e.isDir)
        .map(e => ({ name: e.name, path: e.path, isDir: e.isDir }))
    }
    // 拓扑展开时跟随刷新（未展开不打扰，避免定时器白白跑两条 docker 命令）
    if (topoOpen.value) {
      loadTopo()
    }
  }
  finally {
    loading.value = false
  }
}

watch(project, () => load(), { immediate: true })

const running = computed(() => info.value?.running || 0)
const total = computed(() => info.value?.total || 0)

// 项目主 compose 文件：目录里可能是 compose.yaml / docker-compose.yml 等命名
const mainYaml = computed(() => {
  if (!info.value?.managed) {
    return ''
  }
  const dir = info.value.dir.replace(/\/$/, '')
  const hit = dirFiles.value.find(f => /^compose\.ya?ml$|^docker-compose\.ya?ml$/.test(f.name))
  return hit ? hit.path : `${dir}/docker-compose.yml`
})

async function doUp() {
  acting.value = 'up'
  try {
    await apiCompose.up(info.value!.name, info.value!.managed ? '' : info.value!.dir)
    toast.success(i18n.global.t('container.common.started', { name: project.value }))
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.startFailed'), { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function confirmDown() {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.common.stopAppTitle'),
    content: i18n.global.t('container.common.stopAppConfirm', { name: project.value }),
    onConfirm: async () => {
      acting.value = 'down'
      try {
        await apiCompose.down(info.value!.name, info.value!.managed ? '' : info.value!.dir)
        toast.success(i18n.global.t('container.common.stopped', { name: project.value }))
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

function containerOf(service: string): ContainerItem | undefined {
  // compose 服务名（label com.docker.compose.service）与展示名（容器名/项目约定）可能不同，两级匹配
  return containers.value.find(c => c.labels?.[LabelComposeService] === service)
    || containers.value.find(c => c.name === service)
}

// ---- 服务级操作（容器 action + compose 服务级重建） ----
async function serviceAction(service: string, action: 'start' | 'stop' | 'restart' | 'up') {
  const key = `svc-${service}-${action}`
  acting.value = key
  try {
    await apiCompose.serviceAction(info.value!.name, service, action, info.value!.managed ? '' : info.value!.dir)
    const msgKey = action === 'up' ? 'container.common.svcRebuilt' : action === 'start' ? 'container.common.svcStarted' : action === 'stop' ? 'container.common.svcStopped' : 'container.common.svcRestarted'
    toast.success(i18n.global.t(msgKey, { name: service }))
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.appDetail.svcActionFailed', { action }), { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function dropdownItems(service: string) {
  const c = containerOf(service)
  const running = c?.state === 'running'
  return [[
    ...(running ? [] : [{ label: i18n.global.t('common.start'), icon: 'i-lucide:play', handle: () => serviceAction(service, 'start') }]),
    ...(running ? [{ label: i18n.global.t('common.restart'), icon: 'i-lucide:rotate-cw', handle: () => serviceAction(service, 'restart') }] : []),
    ...(running ? [{ label: i18n.global.t('common.stop'), icon: 'i-lucide:square', handle: () => serviceAction(service, 'stop') }] : []),
    { label: i18n.global.t('container.common.recreateByCompose'), icon: 'i-lucide:hammer', handle: () => serviceAction(service, 'up') },
  ]]
}

// ---- 服务日志弹窗（compose logs 按服务流式） ----
const svcLogsVisible = ref(false)
const svcLogsTarget = ref('')
const svcLogsContent = ref('')
const svcLogsFollowing = ref(false)
let svcLogsAbort: AbortController | null = null

function openServiceLogs(service: string) {
  svcLogsTarget.value = service
  svcLogsContent.value = ''
  svcLogsVisible.value = true
  loadServiceLogs(service, false)
}

function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}

async function loadServiceLogs(service: string, follow: boolean) {
  svcLogsAbort?.abort()
  svcLogsAbort = new AbortController()
  svcLogsFollowing.value = follow
  try {
    const token = appAccountStore.token
    const url = `${wsBase()}/${apiCompose.logsURL(info.value!.name, info.value!.managed ? '' : info.value!.dir, token, 1000, follow, service)}`
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

function openServiceTerminal(service: string) {
  const c = containerOf(service)
  if (!c) {
    toast.warning(i18n.global.t('container.appDetail.svcNoContainer', { name: service }))
    return
  }
  fileEditorStore.openWorkspace(undefined, 'local', c.id)
  fileEditorStore.layout.terminalVisible = true
}

function openFile(path: string, dir?: string) {
  fileEditorStore.openWorkspace(path, 'local', undefined, dir)
}

const stateStyle: Record<string, string> = {
  running: 'text-emerald-600 bg-emerald-500/10',
  exited: 'text-muted-foreground bg-muted',
  paused: 'text-amber-600 bg-amber-500/10',
  created: 'text-blue-600 bg-blue-500/10',
  restarting: 'text-amber-600 bg-amber-500/10',
  dead: 'text-red-600 bg-red-500/10',
}

// ---- 服务拓扑（M26 P1：depends_on 依赖视图，多服务时展示） ----
const topoOpen = ref(false)
const topo = ref<ComposeTopology | null>(null)
const topoLoading = ref(false)
const topoError = ref('')
const showTopo = computed(() => (info.value?.services?.length || 0) > 1)

async function loadTopo() {
  if (!info.value) {
    return
  }
  topoLoading.value = true
  topoError.value = ''
  try {
    topo.value = await apiCompose.topology(info.value.name, info.value.managed ? '' : info.value.dir)
  }
  catch (e: any) {
    topoError.value = e?.message || i18n.global.t('container.appDetail.topoLoadFailed')
  }
  finally {
    topoLoading.value = false
  }
}

watch(topoOpen, (v) => {
  if (v && !topo.value && !topoLoading.value) {
    loadTopo()
  }
})

// 点击拓扑节点：滚动定位到服务表行并短暂高亮
const highlightSvc = ref('')
function onTopoNodeClick(name: string) {
  highlightSvc.value = name
  document.getElementById(`svc-row-${name}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  setTimeout(() => {
    if (highlightSvc.value === name) {
      highlightSvc.value = ''
    }
  }, 1800)
}

// ---- 版本历史 ----
const revisionVisible = ref(false)

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
          <FaButton variant="ghost" size="icon-sm" :title="$t('container.appDetail.closeTitle')" @click="onBack">
            <FaIcon name="i-lucide:arrow-left" class="text-base" />
          </FaButton>
          <YdAppIcon :image="iconUrl" :name="project" :size="26" />
          <span class="truncate">{{ appName || project }}</span>
          <span
            class="shrink-0 rounded-full px-2 py-0.5 text-xs"
            :class="install ? 'bg-violet-500/10 text-violet-600' : info?.managed ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'"
          >
            {{ install ? $t('container.common.srcStore') : info?.managed ? $t('container.common.srcManaged') : $t('container.common.srcExternal') }}
          </span>
          <span v-if="install" class="shrink-0 rounded-full bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">
            v{{ install.version }}
          </span>
        </div>
      </template>
      <template #description>
        <span class="font-mono text-xs">{{ info?.dir || project }}<template v-if="total > 0"> · {{ $t('container.appDetail.runningStats', { running, total }) }}</template><template v-else> · {{ $t('container.appDetail.undeployedHint') }}</template></span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" :title="$t('common.refresh')" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton v-if="running === 0" size="sm" :loading="acting === 'up'" @click="doUp">
          {{ $t('common.start') }}
        </FaButton>
        <template v-else>
          <FaButton variant="outline" size="sm" :loading="acting === 'up'" :title="$t('container.common.recreateTitle')" @click="doUp">
            {{ $t('container.common.recreate') }}
          </FaButton>
          <FaButton variant="outline" size="sm" :loading="acting === 'down'" @click="confirmDown">
            {{ $t('common.stop') }}
          </FaButton>
        </template>
        <FaButton v-if="mainYaml" variant="outline" size="sm" @click="openFile(mainYaml, info?.dir)">
          <FaIcon name="i-lucide:pen-line" class="mr-1" /> {{ $t('container.common.editConfig') }}
        </FaButton>
        <FaButton v-if="mainYaml" variant="outline" size="sm" @click="revisionVisible = true">
          <FaIcon name="i-lucide:history" class="mr-1" /> {{ $t('container.common.revisionHistory') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="!info" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
        {{ $t('container.appDetail.notFound', { name: project }) }}
      </div>
      <template v-else>
        <!-- 服务列表 -->
        <div class="mb-3 text-xs font-medium text-muted-foreground">
          {{ $t('container.appDetail.services', { n: info.services.length }) }}
        </div>
        <div class="mb-6 overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('container.appDetail.service') }}</th>
                <th class="px-3 py-2">{{ $t('container.common.image') }}</th>
                <th class="px-3 py-2">{{ $t('common.status') }}</th>
                <th class="hidden px-3 py-2 lg:table-cell">{{ $t('container.common.containerCol') }}</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!info.services.length">
                <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
                  {{ $t('container.appDetail.noServices') }}
                </td>
              </tr>
              <tr
                v-for="s in info.services" :key="s.name" :id="`svc-row-${s.name}`"
                class="border-t transition-colors hover:bg-accent/30"
                :class="highlightSvc === s.name ? 'bg-primary/5 ring-1 ring-inset ring-primary/40' : ''"
              >
                <td class="px-3 py-2">
                  <div class="flex items-center gap-2">
                    <YdAppIcon :name="s.image" :size="18" />
                    <span class="font-mono text-[13px] font-medium">{{ s.name }}</span>
                  </div>
                </td>
                <td class="max-w-48 truncate px-3 py-2 font-mono text-xs text-muted-foreground" :title="s.image">
                  {{ s.image }}
                </td>
                <td class="px-3 py-2">
                  <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="stateStyle[s.state] || stateStyle.exited">
                    <span class="inline-block size-1.5 rounded-full" :class="s.state === 'running' ? 'animate-pulse bg-current' : 'bg-current'" />
                    {{ tr(`container.state.${s.state}`) }}
                  </span>
                </td>
                <td class="hidden px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell">
                  {{ containerOf(s.name)?.name || '—' }}
                </td>
                <td class="px-3 py-2">
                  <div class="flex items-center justify-end gap-1">
                    <FaButton variant="ghost" size="sm" :title="$t('container.common.svcLogs')" @click="openServiceLogs(s.name)">
                      <FaIcon name="i-lucide:scroll-text" class="text-sm" />
                    </FaButton>
                    <FaButton variant="ghost" size="sm" :title="$t('container.common.openTerminal')" @click="openServiceTerminal(s.name)">
                      <FaIcon name="i-lucide:square-terminal" class="text-sm" />
                    </FaButton>
                    <FaButton
                      v-if="containerOf(s.name)" variant="ghost" size="sm" :title="$t('container.common.containerDetail')"
                      @click="openContainerDetailWin(containerOf(s.name)!.id)"
                    >
                      <FaIcon name="i-lucide:info" class="text-sm" />
                    </FaButton>
                    <FaDropdown
                      :items="dropdownItems(s.name)"
                    >
                      <FaButton variant="outline" size="sm" :loading="acting === `svc-${s.name}-start` || acting === `svc-${s.name}-restart` || acting === `svc-${s.name}-stop` || acting === `svc-${s.name}-up`" :title="$t('container.common.powerActions')">
                        <FaIcon name="i-lucide:power" class="text-sm" />
                      </FaButton>
                    </FaDropdown>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 服务拓扑（M26 P1：depends_on 依赖视图，多服务时展示） -->
        <template v-if="showTopo">
          <div class="mb-3 flex items-center gap-1 text-xs font-medium text-muted-foreground">
            <button type="button" class="flex cursor-pointer items-center gap-1 transition-colors hover:text-primary" @click="topoOpen = !topoOpen">
              <FaIcon :name="topoOpen ? 'i-lucide:chevron-down' : 'i-lucide:chevron-right'" class="text-sm" />
              {{ $t('container.appDetail.topology') }}
            </button>
            <FaButton v-if="topoOpen" variant="ghost" size="icon-sm" :title="$t('container.appDetail.refreshTopo')" @click="loadTopo()">
              <FaIcon name="i-lucide:refresh-cw" class="text-xs" :class="topoLoading ? 'animate-spin' : ''" />
            </FaButton>
            <span class="text-[11px] font-normal">{{ $t('container.appDetail.topoHint') }}</span>
          </div>
          <div v-show="topoOpen" class="mb-6 rounded-lg border p-4">
            <div v-if="topoError" class="py-6 text-center text-xs text-red-500">
              {{ topoError }}
            </div>
            <div v-else-if="topoLoading && !topo" class="py-6 text-center text-xs text-muted-foreground">
              {{ $t('container.appDetail.topoLoading') }}
            </div>
            <YdTopology v-else-if="topo" :nodes="topo.nodes" :edges="topo.edges" @node-click="onTopoNodeClick" />
          </div>
        </template>

        <!-- 配置文件 -->
        <template v-if="info.managed">
          <div class="mb-3 text-xs font-medium text-muted-foreground">
            {{ $t('container.appDetail.configFiles') }}
          </div>
          <div class="mb-6 flex flex-wrap gap-2">
            <button
              v-for="f in dirFiles" :key="f.path" type="button"
              class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border bg-background px-3 py-1.5 font-mono text-xs transition-colors hover:border-primary hover:text-primary"
              @click="openFile(f.path, info.dir)"
            >
              <FaIcon name="i-lucide:file-text" class="text-sm" />
              {{ f.name }}
            </button>
            <span v-if="!dirFiles.length" class="text-xs text-muted-foreground">
              {{ $t('container.appDetail.configDirEmpty') }}
            </span>
          </div>
        </template>

        <!-- 项目容器 -->
        <div class="mb-3 text-xs font-medium text-muted-foreground">
          {{ $t('container.appDetail.projectContainers', { n: containers.length }) }}
        </div>
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('container.common.containerCol') }}</th>
                <th class="px-3 py-2">{{ $t('common.status') }}</th>
                <th class="hidden px-3 py-2 lg:table-cell">{{ $t('container.common.ports') }}</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!containers.length">
                <td colspan="4" class="px-3 py-8 text-center text-muted-foreground">
                  {{ $t('container.common.noContainers') }}
                </td>
              </tr>
              <tr v-for="c in containers" :key="c.id" class="border-t transition-colors hover:bg-accent/30">
                <td class="px-3 py-2 font-mono text-[13px]">
                  {{ c.name }}
                </td>
                <td class="px-3 py-2">
                  <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="stateStyle[c.state] || stateStyle.exited">
                    {{ tr(`container.state.${c.state}`) }}
                  </span>
                </td>
                <td class="hidden px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell">
                  {{ c.ports?.filter(p => p.hostPort).map(p => `${p.hostPort}→${p.containerPort}/${p.proto}`).join('  ') || '—' }}
                </td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" @click="openContainerDetailWin(c.id)">
                    {{ $t('common.detail') }}
                  </FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </FaPageMain>

    <!-- 版本历史（组件自含弹窗） -->
    <YdRevisionHistory
      v-if="mainYaml" v-model="revisionVisible"
      node="local" :path="mainYaml" @restored="load()"
    />

    <!-- 服务日志 -->
    <FaModal
      v-model="svcLogsVisible"
      :title="$t('container.common.svcLogsTitle', { name: svcLogsTarget })"
      class="max-w-5xl!"
      :destroy-on-close="true"
      @close="closeServiceLogs"
    >
      <div class="mb-2 flex items-center gap-2">
        <FaButton size="sm" :variant="svcLogsFollowing ? 'default' : 'outline'" @click="loadServiceLogs(svcLogsTarget, !svcLogsFollowing)">
          <FaIcon name="i-lucide:radio" class="mr-1" :class="svcLogsFollowing ? 'animate-pulse' : ''" />
          {{ svcLogsFollowing ? $t('container.common.following') : $t('container.common.followLogs') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="loadServiceLogs(svcLogsTarget, false)">
          {{ $t('common.refresh') }}
        </FaButton>
        <span v-if="containerOf(svcLogsTarget)" class="text-xs text-muted-foreground">
          {{ $t('container.appDetail.containerPrefix') }}{{ containerOf(svcLogsTarget)!.name }} · {{ tr(`container.state.${containerOf(svcLogsTarget)!.state}`) }}
        </span>
      </div>
      <YdLogViewer :logs="svcLogsContent" height="384px" :loading="svcLogsFollowing" />
      <template #footer>
        <FaButton variant="outline" @click="closeServiceLogs">
          {{ $t('common.close') }}
        </FaButton>
      </template>
    </FaModal>

    <FileEditorWorkspace />
  </div>
</template>
