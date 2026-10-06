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
  return containers.value.find(c => c.labels?.[LabelComposeService] === service)
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
        <FaButton v-if="info?.managed" variant="outline" size="sm" @click="openFile(`${info.dir.replace(/\/$/, '')}/docker-compose.yml`)">
          <FaIcon name="i-lucide:pen-line" class="mr-1" /> 编辑配置
        </FaButton>
        <FaButton v-if="info?.managed" variant="outline" size="sm" @click="revisionVisible = true">
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
                    <FaButton variant="ghost" size="sm" title="终端 + 文件" @click="openServiceTerminal(s.name)">
                      <FaIcon name="i-lucide:square-terminal" class="text-sm" />
                    </FaButton>
                    <FaButton
                      v-if="containerOf(s.name)" variant="ghost" size="sm" title="容器详情"
                      @click="router.push(`/container/detail/${containerOf(s.name)!.id}`)"
                    >
                      <FaIcon name="i-lucide:info" class="text-sm" />
                    </FaButton>
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

    <!-- 版本历史 -->
    <FaModal v-model="revisionVisible" :title="`版本历史：${project} / docker-compose.yml`" class="max-w-3xl!" :destroy-on-close="true">
      <YdRevisionHistory
        v-if="info?.managed" v-model:visible="revisionVisible"
        node="local" :path="`${info.dir.replace(/\/$/, '')}/docker-compose.yml`" @restored="load()"
      />
      <template #footer>
        <FaButton variant="outline" @click="revisionVisible = false">
          关闭
        </FaButton>
      </template>
    </FaModal>

    <FileEditorWorkspace />
  </div>
</template>
