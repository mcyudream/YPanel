<script setup lang="ts">
import type { Src2ServiceSpec, Src2Suggestion } from '@/api/modules/compose'
import apiCompose from '@/api/modules/compose'
import apiCred, { type GitCredential } from '@/api/modules/gitcred'
import { i18n } from '@/locales'

// M26 P2：从源码创建应用向导（仓库 → 检测候选多选（monorepo/多模块）→ 构建任务）。
// 弹窗自含（exp：FaModal 插槽内 defineModel 断链——组件自身持 FaModal，面板随打开重建）。
const visible = defineModel<boolean>({ default: false })

const emit = defineEmits<{
  (e: 'created', projectName: string): void
}>()

const toast = useFaToast()
const taskCenter = useTaskCenterStore()
const appAccountStore = useAppAccountStore()

const step = ref<'repo' | 'services'>('repo')
const gitUrl = ref('')
const branch = ref('')
const credentialId = ref(0) // 0=自动匹配，-1=不使用凭据
const previewing = ref(false)
const commit = ref('')
const suggestions = ref<Src2Suggestion[]>([])
const checked = ref<Record<number, boolean>>({})
const projectName = ref('')
const submitting = ref(false)

interface Row extends Src2ServiceSpec {
  versionRaw: string
  packaging: string
  modules: string[]
  pkgMgr: string
  module: string
  distDir: string
}
const rows = ref<Row[]>([])

const langs = computed(() => [
  { value: 'dockerfile', label: i18n.global.t('container.src.langDockerfile') },
  { value: 'go', label: 'Go' },
  { value: 'node', label: i18n.global.t('container.src.langNode') },
  { value: 'node-build', label: i18n.global.t('container.src.langNodeBuild') },
  { value: 'java-maven', label: i18n.global.t('container.src.langJavaMaven') },
  { value: 'java-gradle', label: i18n.global.t('container.src.langJavaGradle') },
  { value: 'python', label: i18n.global.t('container.src.langPython') },
  { value: 'php', label: i18n.global.t('container.src.langPhp') },
  { value: 'static', label: i18n.global.t('container.src.langStatic') },
])

watch(visible, (v) => {
  if (v) {
    reset()
    loadCreds()
  }
})

function reset() {
  step.value = 'repo'
  gitUrl.value = ''
  branch.value = ''
  credentialId.value = 0
  previewing.value = false
  commit.value = ''
  suggestions.value = []
  checked.value = {}
  projectName.value = ''
  rows.value = []
  credFormVisible.value = false
}

// ---- 凭据 ----
const creds = ref<GitCredential[]>([])
const credFormVisible = ref(false)
const credForm = ref({ name: '', type: 'token' as 'token' | 'ssh', host: '', username: '', secret: '', remark: '' })
const credSaving = ref(false)

const credOptions = computed(() => [
  { label: i18n.global.t('container.src.credAuto'), value: 0 },
  { label: i18n.global.t('container.src.credNone'), value: -1 },
  ...creds.value.map(c => ({ label: `${c.name}（${c.host}）`, value: c.id })),
])

const credTypeOptions = computed(() => [
  { label: i18n.global.t('container.src.credTypeToken'), value: 'token' },
  { label: i18n.global.t('container.src.credTypeSsh'), value: 'ssh' },
])

async function loadCreds() {
  creds.value = await apiCred.list().catch(() => [])
}

async function saveCred() {
  if (!credForm.value.name || !credForm.value.host || !credForm.value.secret) {
    toast.warning(i18n.global.t('container.src.vCredRequired'))
    return
  }
  credSaving.value = true
  try {
    await apiCred.create({ ...credForm.value })
    toast.success(i18n.global.t('container.src.credSaved'))
    credFormVisible.value = false
    credForm.value = { name: '', type: 'token', host: '', username: '', secret: '', remark: '' }
    await loadCreds()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.saveFailed'), { description: e?.message })
  }
  finally {
    credSaving.value = false
  }
}

async function removeCred(id: number) {
  await apiCred.remove(id).catch(() => null)
  await loadCreds()
}

// ---- 检测（SSE 流式进度：校验/凭据/克隆用时/检出/标记扫描实时回显） ----
const previewLogs = ref<string[]>([])
const previewLogBox = useTemplateRef<HTMLElement>('previewLogBox')

function appendLog(text: string) {
  previewLogs.value.push(text)
  nextTick(() => {
    const el = previewLogBox.value
    if (el) {
      el.scrollTop = el.scrollHeight
    }
  })
}

function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}

async function doPreview() {
  if (!gitUrl.value.trim()) {
    toast.warning(i18n.global.t('container.src.gitUrlRequired'))
    return
  }
  previewing.value = true
  previewLogs.value = []
  try {
    const token = appAccountStore.token
    const url = `${wsBase()}/${apiCompose.src2PreviewStreamURL(gitUrl.value.trim(), branch.value.trim() || undefined, credentialId.value > 0 ? credentialId.value : undefined)}`
    const resp = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    if (!resp.ok || !resp.body) {
      throw new Error(`HTTP ${resp.status}`)
    }
    const reader = resp.body.getReader()
    const dec = new TextDecoder()
    let buf = ''
    let finished = false
    let fatal: Error | null = null
    for (;;) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }
      buf += dec.decode(value, { stream: true })
      const blocks = buf.split('\n\n')
      buf = blocks.pop() || ''
      for (const b of blocks) {
        const line = b.split('\n').find(l => l.startsWith('data: '))
        if (!line) {
          continue
        }
        const ev = JSON.parse(line.slice(6))
        if (ev.type === 'log') {
          appendLog(ev.text)
        } else if (ev.type === 'error') {
          fatal = new Error(ev.message)
        } else if (ev.type === 'done') {
          applySuggestions(ev.commit || '', ev.suggestions || [])
          finished = true
        }
      }
      if (finished || fatal) {
        break
      }
    }
    if (fatal) {
      throw fatal
    }
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.src.detectFailed'), { description: e?.message })
  }
  finally {
    previewing.value = false
  }
}

function applySuggestions(commitSha: string, sugg: Src2Suggestion[]) {
  commit.value = commitSha
  suggestions.value = sugg
  if (!sugg.length) {
    toast.warning(i18n.global.t('container.src.noLangMarkers'))
    return
  }
  rows.value = sugg.map(s => ({
    name: s.service,
    dir: s.dir,
    lang: s.lang,
    version: s.version || '',
    versionRaw: s.version || '',
    hostPort: s.hostPort,
    containerPort: s.port,
    startCmd: s.startCmd || '',
    module: '',
    buildCmd: s.buildCmd || '',
    distDir: 'dist',
    packaging: s.packaging || '',
    modules: s.modules || [],
    pkgMgr: s.pkgMgr || '',
  }))
  checked.value = {}
  sugg.forEach((_, i) => {
    checked.value[i] = true
  })
  if (!projectName.value) {
    projectName.value = repoName(gitUrl.value)
  }
  step.value = 'services'
}

function repoName(u: string) {
  const tail = u.replace(/\/+$/, '').split('/').pop() || 'app'
  return tail.replace(/\.git$/, '').replace(/[^a-zA-Z0-9_-]/g, '-').replace(/^-+/, '').slice(0, 48) || 'app'
}

// 语言切换时联动端口（构建静态/静态/PHP 容器固定 80）
function onLangChange(r: Row) {
  if (r.lang === 'node-build' || r.lang === 'static' || r.lang === 'php') {
    r.containerPort = 80
  } else if (r.lang === 'node') {
    r.containerPort = 3000
  }
}

const checkedCount = computed(() => Object.values(checked.value).filter(Boolean).length)
const checkedRows = computed(() => rows.value.filter((_, i) => checked.value[i]))

// ---- 提交 ----
async function submit() {
  if (!projectName.value || !checkedCount.value) {
    toast.warning(i18n.global.t('container.src.vProject'))
    return
  }
  const services: Src2ServiceSpec[] = checkedRows.value.map(r => ({
    name: r.name,
    dir: r.dir,
    lang: r.lang,
    version: r.versionRaw || undefined,
    hostPort: Number(r.hostPort) || 0,
    containerPort: Number(r.containerPort) || 0,
    startCmd: r.startCmd || undefined,
    module: r.module || undefined,
    buildCmd: (r.lang === 'node-build' && r.buildCmd) || undefined,
    distDir: (r.lang === 'node-build' && r.distDir) || undefined,
  }))
  submitting.value = true
  try {
    const res = await apiCompose.src2Create({
      name: projectName.value.trim(),
      gitUrl: gitUrl.value.trim(),
      branch: branch.value.trim() || undefined,
      credentialId: credentialId.value > 0 ? credentialId.value : undefined,
      services,
    })
    toast.success(i18n.global.t('container.src.buildStarted'))
    taskCenter.open(res.taskId)
    visible.value = false
    emit('created', projectName.value.trim())
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.createFailed'), { description: e?.message })
  }
  finally {
    submitting.value = false
  }
}

function dirLabel(d: string) {
  return d || i18n.global.t('container.src.dirRoot')
}

function portHint(r: Row) {
  const c = (r.lang === 'node-build' || r.lang === 'static' || r.lang === 'php') ? 80 : (Number(r.containerPort) || 0)
  return `${Number(r.hostPort) || 0}:${c || '—'}`
}
</script>

<template>
  <FaModal v-model="visible" :title="$t('container.src.title')" class="max-w-4xl!" :destroy-on-close="true">
    <!-- 第一步：仓库 + 凭据 -->
    <div v-if="step === 'repo'">
      <div class="space-y-3">
        <div>
          <div class="mb-1 text-xs text-muted-foreground">{{ $t('container.src.gitUrlLabel') }}</div>
          <FaInput v-model="gitUrl" placeholder="https://github.com/user/repo.git" class="w-full" />
        </div>
        <div class="flex gap-3">
          <div class="flex-1">
            <div class="mb-1 text-xs text-muted-foreground">{{ $t('container.src.branchLabel') }}</div>
            <FaInput v-model="branch" placeholder="main" class="w-full" />
          </div>
          <div class="flex-1">
            <div class="mb-1 text-xs text-muted-foreground">{{ $t('container.src.credLabel') }}</div>
            <FaSelect v-model="credentialId" :options="credOptions" class="w-full" />
          </div>
        </div>

        <!-- 流式进度（校验/凭据/克隆用时/检出/标记扫描） -->
        <div v-if="previewLogs.length" class="rounded-md border bg-muted/30 p-2">
          <pre ref="previewLogBox" class="max-h-40 overflow-auto font-mono text-[11px] leading-5 text-muted-foreground">{{ previewLogs.join('\n') }}</pre>
        </div>

        <!-- 凭据管理 -->
        <div class="rounded-md border p-3">
          <div class="mb-2 flex items-center gap-2">
            <span class="text-xs font-medium">{{ $t('container.src.credLib', { n: creds.length }) }}</span>
            <FaButton size="sm" variant="outline" class="ml-auto" @click="credFormVisible = !credFormVisible">
              <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('container.src.addCred') }}
            </FaButton>
          </div>
          <div v-if="!creds.length" class="text-xs text-muted-foreground">
            {{ $t('container.src.noCreds') }}
          </div>
          <div v-else class="space-y-1">
            <div v-for="c in creds" :key="c.id" class="flex items-center gap-2 text-xs">
              <span class="rounded px-1.5 py-0.5" :class="c.type === 'ssh' ? 'bg-violet-500/10 text-violet-600' : 'bg-blue-500/10 text-blue-600'">{{ c.type }}</span>
              <span class="font-medium">{{ c.name }}</span>
              <span class="text-muted-foreground">{{ c.host }}</span>
              <span v-if="c.username" class="text-muted-foreground">/ {{ c.username }}</span>
              <button type="button" class="ml-auto cursor-pointer text-red-500 hover:underline" @click="removeCred(c.id)">{{ $t('common.delete') }}</button>
            </div>
          </div>
          <div v-if="credFormVisible" class="mt-3 space-y-2 border-t pt-3">
            <div class="flex gap-2">
              <FaInput v-model="credForm.name" :placeholder="$t('container.src.namePlaceholder')" class="w-36" />
              <FaSelect v-model="credForm.type" :options="credTypeOptions" class="w-28" />
              <FaInput v-model="credForm.host" :placeholder="$t('container.src.hostPlaceholder')" class="flex-1" />
            </div>
            <div class="flex gap-2">
              <FaInput v-if="credForm.type === 'token'" v-model="credForm.username" :placeholder="$t('container.src.usernamePlaceholder')" class="w-44" />
              <FaInput v-model="credForm.secret" :placeholder="$t('container.src.secretPlaceholder')" class="flex-1" />
            </div>
            <div class="text-right">
              <FaButton size="sm" :loading="credSaving" @click="saveCred">{{ $t('container.src.saveCred') }}</FaButton>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 第二步：候选确认（卡片式，勾选展开参数；monorepo 多选=多服务同项目） -->
    <div v-else class="space-y-3">
      <div class="flex flex-wrap items-center gap-2 text-xs">
        <span class="max-w-70 truncate font-mono text-muted-foreground" :title="gitUrl">{{ gitUrl }}</span>
        <span v-if="commit" class="rounded bg-muted px-1.5 py-0.5 font-mono text-muted-foreground">{{ commit.slice(0, 10) }}</span>
        <span class="text-muted-foreground">{{ $t('container.src.detected', { n: rows.length }) }}</span>
        <FaButton size="sm" variant="ghost" class="ml-auto" @click="step = 'repo'">
          <FaIcon name="i-lucide:arrow-left" class="mr-1" /> {{ $t('container.src.prevStep') }}
        </FaButton>
      </div>

      <div class="flex items-center gap-2">
        <span class="shrink-0 text-xs text-muted-foreground">{{ $t('container.src.projectNameLabel') }}</span>
        <FaInput v-model="projectName" class="w-60" :placeholder="$t('container.src.projectNamePlaceholder')" />
        <span class="text-xs text-muted-foreground">{{ $t('container.src.projectHint') }}</span>
      </div>

      <div class="max-h-[52vh] space-y-2.5 overflow-y-auto pr-1">
        <div
          v-for="(r, i) in rows" :key="r.dir"
          class="rounded-lg border p-3 transition-colors"
          :class="checked[i] ? 'border-primary/40 bg-primary/[0.03]' : 'opacity-55'"
        >
          <!-- 卡片头：勾选 + 目录/标记 + 语言栈 -->
          <div class="flex flex-wrap items-center gap-2.5">
            <input v-model="checked[i]" type="checkbox" class="accent-[var(--primary)]">
            <div class="w-52 min-w-0">
              <div class="truncate font-mono text-[13px] font-medium" :title="dirLabel(r.dir)">{{ dirLabel(r.dir) }}</div>
              <div class="truncate text-[11px] text-muted-foreground">
                {{ r.dir ? r.dir.split('/').pop() : $t('container.src.rootDir') }}<template v-if="suggestions[i]?.marker"> · {{ suggestions[i].marker }}</template>
              </div>
            </div>
            <FaSelect v-model="r.lang" :options="langs" class="w-56" @update:model-value="onLangChange(r)" />
            <div class="ml-auto flex items-center gap-2">
              <span
                v-if="r.packaging === 'pom'"
                class="rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] text-amber-600"
                :title="$t('container.src.multiModuleTitle')"
              >{{ $t('container.src.multiModule') }}</span>
              <span v-if="checked[i]" class="font-mono text-[11px] text-muted-foreground">{{ portHint(r) }}</span>
            </div>
          </div>

          <!-- 参数区：勾选后展开 -->
          <div v-if="checked[i]" class="mt-2.5 grid grid-cols-2 gap-x-3 gap-y-2 border-t pt-2.5 lg:grid-cols-4">
            <div>
              <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.serviceName') }}</div>
              <FaInput v-model="r.name" class="w-full" />
            </div>
            <!-- maven 多模块：选运行模块 -->
            <div v-if="r.lang === 'java-maven' && r.packaging === 'pom' && r.modules.length">
              <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.runModule') }}</div>
              <FaSelect v-model="r.module" :options="[{ label: $t('container.src.pleaseSelect'), value: '' }, ...r.modules.map(m => ({ label: m, value: m }))]" class="w-full" />
            </div>
            <!-- node 双模式 -->
            <template v-if="r.lang === 'node'">
              <div>
                <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.startCmd') }}</div>
                <FaInput v-model="r.startCmd" class="w-full" :placeholder="r.pkgMgr === 'pnpm' ? 'pnpm start' : 'npm start'" />
              </div>
              <div>
                <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.nodeVersion') }}</div>
                <FaInput v-model="r.versionRaw" class="w-full" :placeholder="r.versionRaw || '22'" />
              </div>
            </template>
            <template v-else-if="r.lang === 'node-build'">
              <div class="lg:col-span-2">
                <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.buildCmd') }}</div>
                <FaInput v-model="r.buildCmd" class="w-full" placeholder="pnpm build" />
              </div>
              <div>
                <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.distDir') }}</div>
                <FaInput v-model="r.distDir" class="w-full" placeholder="dist" />
              </div>
              <div>
                <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.nodeVersion') }}</div>
                <FaInput v-model="r.versionRaw" class="w-full" :placeholder="r.versionRaw || '22'" />
              </div>
            </template>
            <template v-else-if="r.lang === 'java-maven' || r.lang === 'java-gradle'">
              <div>
                <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.javaVersion') }}</div>
                <FaInput v-model="r.versionRaw" class="w-full" :placeholder="r.versionRaw || '21'" />
              </div>
            </template>
            <template v-else-if="r.lang === 'python' || r.lang === 'go'">
              <div>
                <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.startCmd') }}</div>
                <FaInput v-model="r.startCmd" class="w-full" :placeholder="r.lang === 'python' ? 'python main.py' : 'go run .'" />
              </div>
            </template>
            <!-- static/php/dockerfile 无额外参数 -->
            <div v-if="r.lang !== 'node-build' && r.lang !== 'static' && r.lang !== 'php' && r.lang !== 'dockerfile'">
              <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.hostPort') }}</div>
              <FaInput v-model="r.hostPort" class="w-full" />
            </div>
            <div v-if="r.lang !== 'node-build' && r.lang !== 'static' && r.lang !== 'php' && r.lang !== 'dockerfile'">
              <div class="mb-1 text-[11px] text-muted-foreground">{{ $t('container.src.containerPort') }}</div>
              <FaInput v-model="r.containerPort" class="w-full" />
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex w-full items-center">
        <span class="text-xs text-muted-foreground">
          {{ step === 'services' ? $t('container.src.selectedServices', { n: checkedCount }) : $t('container.src.detectHint') }}
        </span>
        <div class="ml-auto flex gap-2">
          <FaButton variant="outline" @click="visible = false">{{ $t('common.cancel') }}</FaButton>
          <FaButton v-if="step === 'repo'" :loading="previewing" @click="doPreview">{{ $t('container.src.detect') }}</FaButton>
          <FaButton v-else :loading="submitting" @click="submit">{{ $t('container.src.startBuild') }}</FaButton>
        </div>
      </div>
    </template>
  </FaModal>
</template>
