<script setup lang="ts">
import type { ComposeProject } from '@/api/modules/compose'
import apiCompose from '@/api/modules/compose'
import apiContainer, { LabelComposeProject, LabelComposeService, type ContainerItem } from '@/api/modules/container'
import apiFile from '@/api/modules/file'
import { storeApi, type StoreAppItem, type StoreInstall } from '@/api/modules/store'
import FileEditorWorkspace from '@/views/file_management/editor/Workspace.vue'

// 应用详情（M23）：服务列表 + 配置文件（工作台编辑 + 服务端版本历史）+ 项目容器。
const route = useRoute()
const router = useRouter()
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()
const appAccountStore = useAppAccountStore()

const project = computed(() => String(route.params.project || ''))

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
    toast.success(`已启动 ${project.value}`)
    await load()
  }
  catch (e: any) {
    toast.error('启动失败', { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function confirmDown() {
  const modal = useFaModal()
  modal.confirm({
    title: '停止应用',
    content: `确认停止 ${project.value}？其容器/网络将被移除（数据卷保留）。`,
    onConfirm: async () => {
      acting.value = 'down'
      try {
        await apiCompose.down(info.value!.name, info.value!.managed ? '' : info.value!.dir)
        toast.success(`已停止 ${project.value}`)
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
    const label = action === 'up' ? '重建' : action === 'start' ? '启动' : action === 'stop' ? '停止' : '重启'
    toast.success(`已${label}服务 ${service}`)
    await load()
  }
  catch (e: any) {
    toast.error(`服务 ${action} 失败`, { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function dropdownItems(service: string) {
  const c = containerOf(service)
  const running = c?.state === 'running'
  return [[
    ...(running ? [] : [{ label: '启动', icon: 'i-lucide:play', handle: () => serviceAction(service, 'start') }]),
    ...(running ? [{ label: '重启', icon: 'i-lucide:rotate-cw', handle: () => serviceAction(service, 'restart') }] : []),
    ...(running ? [{ label: '停止', icon: 'i-lucide:square', handle: () => serviceAction(service, 'stop') }] : []),
    { label: '重建（按编排定义）', icon: 'i-lucide:hammer', handle: () => serviceAction(service, 'up') },
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

function openServiceTerminal(service: string) {
  const c = containerOf(service)
  if (!c) {
    toast.warning(`服务 ${service} 当前没有运行中的容器`)
    return
  }
  fileEditorStore.openWorkspace(undefined, 'local', c.id)
  fileEditorStore.layout.terminalVisible = true
}

function openFile(path: string) {
  fileEditorStore.openWorkspace(path, 'local')
}

const stateStyle: Record<string, string> = {
  running: 'text-emerald-600 bg-emerald-500/10',
  exited: 'text-muted-foreground bg-muted',
  paused: 'text-amber-600 bg-amber-500/10',
  created: 'text-blue-600 bg-blue-500/10',
  restarting: 'text-amber-600 bg-amber-500/10',
  dead: 'text-red-600 bg-red-500/10',
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
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex min-w-0 items-center gap-2.5">
          <FaButton variant="ghost" size="icon-sm" title="返回" @click="router.push('/container?tab=apps')">
            <FaIcon name="i-lucide:arrow-left" class="text-base" />
          </FaButton>
          <YdAppIcon :image="iconUrl" :name="project" :size="26" />
          <span class="truncate">{{ appName || project }}</span>
          <span
            class="shrink-0 rounded-full px-2 py-0.5 text-xs"
            :class="install ? 'bg-violet-500/10 text-violet-600' : info?.managed ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'"
          >
            {{ install ? '商店' : info?.managed ? '托管' : '外部' }}
          </span>
          <span v-if="install" class="shrink-0 rounded-full bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">
            v{{ install.version }}
          </span>
        </div>
      </template>
      <template #description>
        <span class="font-mono text-xs">{{ info?.dir || project }} · {{ running }}/{{ total }} 运行中</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" title="刷新" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton v-if="running === 0" size="sm" :loading="acting === 'up'" @click="doUp">
          启动
        </FaButton>
        <template v-else>
          <FaButton variant="outline" size="sm" :loading="acting === 'up'" title="按当前配置重建并启动" @click="doUp">
            重建
          </FaButton>
          <FaButton variant="outline" size="sm" :loading="acting === 'down'" @click="confirmDown">
            停止
          </FaButton>
        </template>
        <FaButton v-if="mainYaml" variant="outline" size="sm" @click="openFile(mainYaml)">
          <FaIcon name="i-lucide:pen-line" class="mr-1" /> 编辑配置
        </FaButton>
        <FaButton v-if="mainYaml" variant="outline" size="sm" @click="revisionVisible = true">
          <FaIcon name="i-lucide:history" class="mr-1" /> 版本历史
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="!info" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
        应用 {{ project }} 不存在或已移除
      </div>
      <template v-else>
        <!-- 服务列表 -->
        <div class="mb-3 text-xs font-medium text-muted-foreground">
          服务（{{ info.services.length }}）
        </div>
        <div class="mb-6 overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">服务</th>
                <th class="px-3 py-2">镜像</th>
                <th class="px-3 py-2">状态</th>
                <th class="hidden px-3 py-2 lg:table-cell">容器</th>
                <th class="px-3 py-2 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!info.services.length">
                <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
                  应用未部署（启动后显示服务）
                </td>
              </tr>
              <tr v-for="s in info.services" :key="s.name" class="border-t transition-colors hover:bg-accent/30">
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
                    {{ s.state }}
                  </span>
                </td>
                <td class="hidden px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell">
                  {{ containerOf(s.name)?.name || '—' }}
                </td>
                <td class="px-3 py-2">
                  <div class="flex items-center justify-end gap-1">
                    <FaButton variant="ghost" size="sm" title="服务日志" @click="openServiceLogs(s.name)">
                      <FaIcon name="i-lucide:scroll-text" class="text-sm" />
                    </FaButton>
                    <FaButton variant="ghost" size="sm" title="进入终端" @click="openServiceTerminal(s.name)">
                      <FaIcon name="i-lucide:square-terminal" class="text-sm" />
                    </FaButton>
                    <FaButton
                      v-if="containerOf(s.name)" variant="ghost" size="sm" title="容器详情"
                      @click="router.push(`/container/detail/${containerOf(s.name)!.id}`)"
                    >
                      <FaIcon name="i-lucide:info" class="text-sm" />
                    </FaButton>
                    <FaDropdown
                      :items="dropdownItems(s.name)"
                    >
                      <FaButton variant="outline" size="sm" :loading="acting === `svc-${s.name}-start` || acting === `svc-${s.name}-restart` || acting === `svc-${s.name}-stop` || acting === `svc-${s.name}-up`" title="电源操作">
                        <FaIcon name="i-lucide:power" class="text-sm" />
                      </FaButton>
                    </FaDropdown>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 配置文件 -->
        <template v-if="info.managed">
          <div class="mb-3 text-xs font-medium text-muted-foreground">
            配置文件（点击在工作台打开，保存自动存版本）
          </div>
          <div class="mb-6 flex flex-wrap gap-2">
            <button
              v-for="f in dirFiles" :key="f.path" type="button"
              class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border bg-background px-3 py-1.5 font-mono text-xs transition-colors hover:border-primary hover:text-primary"
              @click="openFile(f.path)"
            >
              <FaIcon name="i-lucide:file-text" class="text-sm" />
              {{ f.name }}
            </button>
            <span v-if="!dirFiles.length" class="text-xs text-muted-foreground">
              配置目录为空或读取失败
            </span>
          </div>
        </template>

        <!-- 项目容器 -->
        <div class="mb-3 text-xs font-medium text-muted-foreground">
          项目容器（{{ containers.length }}）
        </div>
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">容器</th>
                <th class="px-3 py-2">状态</th>
                <th class="hidden px-3 py-2 lg:table-cell">端口</th>
                <th class="px-3 py-2 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!containers.length">
                <td colspan="4" class="px-3 py-8 text-center text-muted-foreground">
                  暂无容器
                </td>
              </tr>
              <tr v-for="c in containers" :key="c.id" class="border-t transition-colors hover:bg-accent/30">
                <td class="px-3 py-2 font-mono text-[13px]">
                  {{ c.name }}
                </td>
                <td class="px-3 py-2">
                  <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="stateStyle[c.state] || stateStyle.exited">
                    {{ c.state }}
                  </span>
                </td>
                <td class="hidden px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell">
                  {{ c.ports?.filter(p => p.hostPort).map(p => `${p.hostPort}→${p.containerPort}/${p.proto}`).join('  ') || '—' }}
                </td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" @click="router.push(`/container/detail/${c.id}`)">
                    详情
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
      :title="`服务日志：${svcLogsTarget}`"
      class="max-w-5xl!"
      :destroy-on-close="true"
      @close="closeServiceLogs"
    >
      <div class="mb-2 flex items-center gap-2">
        <FaButton size="sm" :variant="svcLogsFollowing ? 'default' : 'outline'" @click="loadServiceLogs(svcLogsTarget, !svcLogsFollowing)">
          <FaIcon name="i-lucide:radio" class="mr-1" :class="svcLogsFollowing ? 'animate-pulse' : ''" />
          {{ svcLogsFollowing ? '跟踪中（点击停止）' : '跟踪日志' }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="loadServiceLogs(svcLogsTarget, false)">
          刷新
        </FaButton>
        <span v-if="containerOf(svcLogsTarget)" class="text-xs text-muted-foreground">
          容器：{{ containerOf(svcLogsTarget)!.name }} · {{ containerOf(svcLogsTarget)!.state }}
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
