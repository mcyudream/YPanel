<script setup lang="ts">
import type { ComposeProject } from '@/api/modules/compose'
import apiCompose from '@/api/modules/compose'
import apiFile from '@/api/modules/file'
import { storeApi, type StoreAppItem, type StoreInstall } from '@/api/modules/store'
import FileEditorWorkspace from '@/views/file_management/editor/Workspace.vue'

// 应用 tab（M23，开发者默认视角）：compose 项目 + 商店安装聚合为应用卡片。
const router = useRouter()
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()

const projects = ref<ComposeProject[]>([])
const installs = ref<StoreInstall[]>([])
const storeApps = ref<StoreAppItem[]>([])
const loading = ref(false)
const dockerDisabled = ref(false)
const dockerMsg = ref('')
const acting = ref('')

async function load() {
  loading.value = true
  try {
    const [ps, inst] = await Promise.all([
      apiCompose.list(),
      storeApi.installed().catch(() => [] as StoreInstall[]),
    ])
    projects.value = ps
    installs.value = inst
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

        <div v-if="c.project.services.length" class="mt-3 flex flex-wrap gap-1.5">
          <span
            v-for="s in c.project.services"
            :key="s.name"
            class="inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-xs"
            :title="`${s.image} · ${s.state}`"
          >
            <span class="inline-block size-1.5 rounded-full" :class="s.state === 'running' ? 'bg-emerald-500' : 'bg-muted-foreground/40'" />
            {{ s.name }}
          </span>
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

    <FileEditorWorkspace />
  </div>
</template>
