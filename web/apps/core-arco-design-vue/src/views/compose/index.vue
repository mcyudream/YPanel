<script setup lang="ts">
import type { ComposeProject } from '@/api/modules/compose'
import apiCompose from '@/api/modules/compose'

defineOptions({
  name: 'ComposeIndex',
})

const appAccountStore = useAppAccountStore()

const projects = ref<ComposeProject[]>([])
const loading = ref(false)
const dockerDisabled = ref(false)
const dockerMsg = ref('')
const acting = ref('')

let timer: ReturnType<typeof setInterval> | null = null

async function load() {
  loading.value = true
  try {
    projects.value = await apiCompose.list()
    dockerDisabled.value = false
  }
  catch (e: any) {
    if (e?.code === 5002) {
      dockerDisabled.value = true
      dockerMsg.value = '未检测到可用的 Docker 环境（/var/run/docker.sock），Compose 管理不可用'
    }
    else {
      dockerMsg.value = e?.message || '加载失败'
    }
  }
  finally {
    loading.value = false
  }
}

async function doUp(p: ComposeProject) {
  acting.value = `up-${p.name}`
  try {
    const out = await apiCompose.up(p.name, p.managed ? '' : p.dir)
    useFaToast().success(`已启动 ${p.name}`)
    if (out) {
      lastOutput.value = out
      outputVisible.value = true
    }
    await load()
  }
  catch (e: any) {
    useFaToast().error('启动失败', { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

async function doDown(p: ComposeProject) {
  acting.value = `down-${p.name}`
  try {
    const out = await apiCompose.down(p.name, p.managed ? '' : p.dir)
    useFaToast().success(`已停止 ${p.name}`)
    if (out) {
      lastOutput.value = out
      outputVisible.value = true
    }
    await load()
  }
  catch (e: any) {
    useFaToast().error('停止失败', { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function confirmDown(p: ComposeProject) {
  const modal = useFaModal()
  modal.confirm({
    title: '停止项目',
    content: `确认 down 项目 ${p.name}？其容器/网络将被移除（数据卷保留）。`,
    onConfirm: () => doDown(p),
  })
}

// ---- 创建/编辑 ----
const editorVisible = ref(false)
const editorName = ref('')
const editorManaged = ref(true)
const editorDir = ref('')
const editorContent = ref('')
const editorSaving = ref(false)
const isCreate = ref(false)

const TEMPLATE = `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
    restart: unless-stopped
`

function openCreate() {
  isCreate.value = true
  editorName.value = ''
  editorManaged.value = true
  editorDir.value = ''
  editorContent.value = TEMPLATE
  editorVisible.value = true
}

async function openEdit(p: ComposeProject) {
  isCreate.value = false
  editorName.value = p.name
  editorManaged.value = p.managed
  editorDir.value = p.dir
  try {
    const cfg = await apiCompose.config(p.name, p.managed ? '' : p.dir)
    editorContent.value = cfg.content
    editorVisible.value = true
  }
  catch (e: any) {
    useFaToast().error('读取配置失败', { description: e?.message })
  }
}

async function saveEditor() {
  editorSaving.value = true
  try {
    if (isCreate.value) {
      if (!/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/.test(editorName.value)) {
        useFaToast().warning('项目名仅允许字母/数字/中划线/下划线')
        return
      }
      await apiCompose.write(editorName.value, editorContent.value)
      useFaToast().success('项目已创建，正在启动…')
      editorVisible.value = false
      await apiCompose.up(editorName.value, '')
      await load()
    }
    else {
      if (!editorManaged.value) {
        useFaToast().warning('外部项目不可在线编辑（可在目标机修改其配置文件）')
        return
      }
      await apiCompose.write(editorName.value, editorContent.value)
      useFaToast().success('已保存，正在重建…')
      editorVisible.value = false
      await apiCompose.up(editorName.value, '')
      await load()
    }
  }
  catch (e: any) {
    useFaToast().error('保存失败', { description: e?.message })
  }
  finally {
    editorSaving.value = false
  }
}

// ---- 日志 ----
const logsVisible = ref(false)
const logsTarget = ref<ComposeProject | null>(null)
const logsContent = ref('')
const logsFollowing = ref(false)
const logsService = ref('')
let logsAbort: AbortController | null = null

async function openLogs(p: ComposeProject) {
  logsTarget.value = p
  logsContent.value = ''
  logsService.value = ''
  logsVisible.value = true
  loadLogs(p, false)
}

function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}

async function loadLogs(p: ComposeProject, follow: boolean) {
  logsAbort?.abort()
  logsAbort = new AbortController()
  logsFollowing.value = follow
  try {
    const url = `${wsBase()}/${apiCompose.logsURL(p.name, p.managed ? '' : p.dir, appAccountStore.token, 500, follow, logsService.value)}`
    const resp = await fetch(url, {
      headers: { Authorization: `Bearer ${appAccountStore.token}` },
      signal: logsAbort.signal,
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
      logsContent.value += decoder.decode(value, { stream: true })
      if (logsContent.value.length > 2_000_000) {
        logsContent.value = logsContent.value.slice(-1_000_000)
      }
      await nextTick()
      const el = document.getElementById('compose-logs-box')
      if (el) {
        el.scrollTop = el.scrollHeight
      }
    }
  }
  catch (e: any) {
    if (e?.name !== 'AbortError') {
      logsContent.value += `\n[日志流错误] ${e?.message || e}`
    }
  }
  finally {
    logsFollowing.value = false
  }
}

function closeLogs() {
  logsAbort?.abort()
  logsVisible.value = false
}

// ---- up/down 输出 ----
const outputVisible = ref(false)
const lastOutput = ref('')

onMounted(() => {
  load()
  timer = setInterval(() => {
    if (!dockerDisabled.value && !acting.value) {
      load()
    }
  }, 5000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
  logsAbort?.abort()
})

</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="layers" :size="24" />
          <span>Compose 编排</span>
        </div>
      </template>
      <template #description>
        <span>docker compose 项目：托管目录 /opt/ypanel/compose，外部项目自动识别</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" title="刷新" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton size="sm" :disabled="dockerDisabled" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 新建项目
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="dockerDisabled" class="mb-4 flex items-center gap-2 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
        <YdMorphIcon name="triangle-alert" :size="18" />
        {{ dockerMsg }}
      </div>
      <div v-else-if="dockerMsg" class="mb-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
        {{ dockerMsg }}
      </div>

      <div v-if="!dockerDisabled && !projects.length" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
        暂无 compose 项目，点击右上角"新建项目"开始
      </div>

      <div class="grid gap-4 xl:grid-cols-2">
        <div
          v-for="p in projects"
          :key="p.name + p.dir"
          class="rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
        >
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <YdMorphIcon name="layers" :size="16" class="text-primary opacity-70" />
                <span class="truncate font-mono text-sm font-medium">{{ p.name }}</span>
                <span
                  class="shrink-0 rounded-full px-2 py-0.5 text-xs"
                  :class="p.managed ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'"
                >
                  {{ p.managed ? '托管' : '外部' }}
                </span>
              </div>
              <div class="mt-1 truncate font-mono text-xs text-muted-foreground" :title="p.dir">{{ p.dir }}</div>
            </div>
            <span class="shrink-0 rounded-full px-2 py-0.5 text-xs tabular-nums" :class="p.running > 0 ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ p.running }}/{{ p.total }} 运行
            </span>
          </div>

          <div v-if="p.services.length" class="mt-3 flex flex-wrap gap-1.5">
            <span
              v-for="s in p.services"
              :key="s.name"
              class="inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-xs"
              :title="`${s.image} · ${s.state}`"
            >
              <span class="inline-block size-1.5 rounded-full" :class="s.state === 'running' ? 'bg-emerald-500' : 'bg-muted-foreground/40'" />
              {{ s.name }}
            </span>
          </div>

          <div class="mt-3 flex flex-wrap items-center gap-1.5 border-t pt-3">
            <FaButton v-if="p.running === 0" variant="outline" size="sm" :disabled="acting === `up-${p.name}`" @click="doUp(p)">
              启动
            </FaButton>
            <template v-else>
              <FaButton variant="outline" size="sm" :disabled="acting === `up-${p.name}`" title="按当前配置重建并启动" @click="doUp(p)">
                重建
              </FaButton>
              <FaButton variant="outline" size="sm" :disabled="acting === `down-${p.name}`" @click="confirmDown(p)">
                停止
              </FaButton>
            </template>
            <FaButton variant="ghost" size="sm" @click="openLogs(p)">
              <FaIcon name="i-lucide:scroll-text" class="mr-1" /> 日志
            </FaButton>
            <FaButton v-if="p.managed" variant="ghost" size="sm" @click="openEdit(p)">
              <FaIcon name="i-lucide:pen-line" class="mr-1" /> 编辑
            </FaButton>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 创建/编辑 yaml -->
    <FaModal
      v-model="editorVisible"
      :title="isCreate ? '新建 compose 项目' : `编辑：${editorName}`"
      class="max-w-4xl!"
      :destroy-on-close="true"
    >
      <div class="flex flex-col gap-3">
        <div v-if="isCreate" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">项目名</span>
          <FaInput v-model="editorName" placeholder="字母/数字/中划线/下划线" class="flex-1" />
        </div>
        <div v-if="!isCreate && !editorManaged" class="rounded-md border border-amber-300 bg-amber-50 p-2 text-xs text-amber-700 dark:bg-amber-950/30">
          外部项目（{{ editorDir }}）只读展示，保存将在托管目录创建同名副本。
        </div>
        <textarea
          v-model="editorContent"
          class="h-96 w-full resize-y rounded-md border border-input bg-background p-3 font-mono text-[13px] leading-relaxed outline-none focus:ring-1 focus:ring-primary"
          spellcheck="false"
        />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="editorVisible = false">
          取消
        </FaButton>
        <FaButton :loading="editorSaving" @click="saveEditor">
          {{ isCreate ? '创建并启动' : '保存并重建' }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 日志 -->
    <FaModal
      v-model="logsVisible"
      :title="`日志：${logsTarget?.name || ''}`"
      class="max-w-5xl!"
      :destroy-on-close="true"
      @close="closeLogs"
    >
      <div class="mb-2 flex flex-wrap items-center gap-2">
        <FaButton size="sm" :variant="logsFollowing ? 'default' : 'outline'" @click="logsTarget && loadLogs(logsTarget, !logsFollowing)">
          <FaIcon name="i-lucide:radio" class="mr-1" :class="logsFollowing ? 'animate-pulse' : ''" />
          {{ logsFollowing ? '跟踪中（点击停止）' : '跟踪日志' }}
        </FaButton>
        <FaInput v-model="logsService" placeholder="服务名（可空）" class="w-40" @keyup.enter="logsTarget && loadLogs(logsTarget, false)" />
        <FaButton variant="outline" size="sm" @click="logsTarget && loadLogs(logsTarget, false)">
          刷新
        </FaButton>
      </div>
      <pre id="compose-logs-box" class="h-96 overflow-auto rounded-md bg-muted/60 p-3 font-mono text-xs leading-relaxed">{{ logsContent || '暂无日志' }}</pre>
      <template #footer>
        <FaButton variant="outline" @click="closeLogs">
          关闭
        </FaButton>
      </template>
    </FaModal>

    <!-- 命令输出 -->
    <FaModal v-model="outputVisible" title="命令输出" class="max-w-3xl!" :destroy-on-close="true">
      <pre class="h-72 overflow-auto rounded-md bg-muted/60 p-3 font-mono text-xs">{{ lastOutput }}</pre>
      <template #footer>
        <FaButton variant="outline" @click="outputVisible = false">
          关闭
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
