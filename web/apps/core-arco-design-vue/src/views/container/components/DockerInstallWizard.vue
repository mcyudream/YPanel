<script setup lang="ts">
import type { DockerPrecheck, InstallDryRunResp, InstallStep, SourceTest } from '@/api/modules/dockerinstall'
import { dockerInstallApi } from '@/api/modules/dockerinstall'
import { taskApi } from '@/api/modules/task'
import YdLogViewer from '@/components/YdLogViewer/index.vue'
import { i18n } from '@/locales'

// M52：Docker/Compose 一键安装向导（预检 → 选源 → 确认 → 执行 → 结果）。
// 弹窗自含（exp：FaModal 插槽内 defineModel 断链——组件自身持 FaModal，面板随打开重建）。
const visible = defineModel<boolean>({ default: false })

const emit = defineEmits<{
  (e: 'installed'): void
}>()

const toast = useFaToast()

type Phase = 'precheck' | 'source' | 'confirm' | 'running' | 'done'
const phase = ref<Phase>('precheck')
const prechecking = ref(false)
const pk = ref<DockerPrecheck | null>(null)

const source = ref('official')
const sourceTesting = ref(false)
const sourceTest = ref<SourceTest | null>(null)

const dryRunning = ref(false)
const dryRun = ref<InstallDryRunResp | null>(null)

const installing = ref(false)
const taskId = ref(0)
const taskStatus = ref<'running' | 'success' | 'failed'>('running')
const taskLog = ref('')
const taskError = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

// 安装完成后可选配置加速器（服务端合并写 daemon.json + 重启 docker）
const withMirror = ref(false)
const PRESET_MIRRORS = [
  { url: 'https://docker.mirrors.ustc.edu.cn', label: '中科大' },
  { url: 'https://hub-mirror.c.163.com', label: '网易' },
  { url: 'https://mirror.baidubce.com', label: '百度' },
]
const checkedMirrors = ref<string[]>([])
const customMirror = ref('')

const SOURCES = [
  { key: 'aliyun', labelKey: 'container.install.sourceAliyun' },
  { key: 'tuna', labelKey: 'container.install.sourceTuna' },
  { key: 'ustc', labelKey: 'container.install.sourceUstc' },
  { key: 'official', labelKey: 'container.install.sourceOfficial' },
]

watch(visible, (v) => {
  if (v) {
    reset()
  }
  else {
    stopPoll()
  }
})

onUnmounted(() => stopPoll())

function reset() {
  phase.value = 'precheck'
  pk.value = null
  source.value = 'official'
  sourceTest.value = null
  dryRun.value = null
  taskId.value = 0
  taskLog.value = ''
  taskError.value = ''
  withMirror.value = false
  checkedMirrors.value = []
  customMirror.value = ''
  doPrecheck()
}

function stopPoll() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function doPrecheck() {
  prechecking.value = true
  try {
    const res = await dockerInstallApi.precheck()
    pk.value = res.precheck
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.install.precheckFailed'), { description: e?.message })
    visible.value = false
  }
  finally {
    prechecking.value = false
  }
}

async function doTest() {
  sourceTesting.value = true
  sourceTest.value = null
  try {
    const res = await dockerInstallApi.precheck(source.value)
    sourceTest.value = res.sourceTest || null
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.install.testFailed'), { description: e?.message })
  }
  finally {
    sourceTesting.value = false
  }
}

async function doDryRun() {
  dryRunning.value = true
  dryRun.value = null
  try {
    const res = await dockerInstallApi.install({ source: source.value, dryRun: true })
    dryRun.value = res as InstallDryRunResp
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.install.dryRunFailed'), { description: e?.message })
    phase.value = 'source'
  }
  finally {
    dryRunning.value = false
  }
}

async function doInstall() {
  installing.value = true
  try {
    const mirrors = collectMirrors()
    const res = await dockerInstallApi.install({
      source: source.value,
      configureMirror: withMirror.value && mirrors.length > 0,
      mirrors: withMirror.value ? mirrors : undefined,
    })
    taskId.value = res.taskId
    phase.value = 'running'
    taskStatus.value = 'running'
    taskLog.value = ''
    taskError.value = ''
    startPoll()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.install.startFailed'), { description: e?.message })
  }
  finally {
    installing.value = false
  }
}

function collectMirrors() {
  const list = [...checkedMirrors.value]
  const custom = customMirror.value.trim()
  if (custom && !list.includes(custom)) {
    list.push(custom)
  }
  return list
}

function startPoll() {
  stopPoll()
  pollTimer = setInterval(async () => {
    if (!taskId.value) {
      return
    }
    try {
      const t = await taskApi.get(taskId.value)
      taskLog.value = t.logText || ''
      if (t.status !== 'running') {
        stopPoll()
        taskStatus.value = t.status
        taskError.value = t.error || ''
        if (t.status === 'success') {
          phase.value = 'done'
          emit('installed')
        }
      }
    }
    catch {}
  }, 2000)
}

const installSteps = computed<InstallStep[]>(() => dryRun.value?.steps || [])

function matrixOk(v: boolean | undefined) {
  return v ? 'ok' : 'bad'
}
</script>

<template>
  <FaModal v-model="visible" :title="$t('container.install.title')" class="max-w-2xl!" :destroy-on-close="true">
    <div class="flex flex-col gap-4">
      <!-- ① 预检 -->
      <div v-if="phase === 'precheck'">
        <div v-if="prechecking" class="flex items-center justify-center gap-2 py-10 text-sm text-muted-foreground">
          <FaIcon name="i-lucide:loader-circle" class="animate-spin text-lg" />
          {{ $t('container.install.prechecking') }}
        </div>
        <template v-else-if="pk">
          <div class="overflow-hidden rounded-md border text-sm">
            <div class="flex items-center justify-between border-b bg-muted/40 px-3 py-2">
              <span class="font-medium">{{ pk.prettyName || pk.distro || '—' }}</span>
              <span class="font-mono text-xs text-muted-foreground">{{ pk.arch }} / {{ pk.distro }} {{ pk.version }}</span>
            </div>
            <div class="grid grid-cols-2 gap-x-4 gap-y-1 px-3 py-2 text-xs">
              <div class="flex items-center gap-1">
                <span class="text-muted-foreground">systemd</span>
                <FaIcon :name="pk.systemd ? 'i-lucide:check' : 'i-lucide:x'" :class="pk.systemd ? 'text-emerald-500' : 'text-red-500'" />
              </div>
              <div class="flex items-center gap-1">
                <span class="text-muted-foreground">Docker</span>
                <FaIcon :name="matrixOk(pk.dockerInstalled) === 'ok' ? 'i-lucide:check' : 'i-lucide:x'" :class="pk.dockerInstalled ? 'text-emerald-500' : 'text-muted-foreground'" />
                <span v-if="pk.dockerVersion" class="truncate font-mono text-[11px] text-muted-foreground">{{ pk.dockerVersion }}</span>
              </div>
              <div class="flex items-center gap-1">
                <span class="text-muted-foreground">{{ $t('container.install.composePlugin') }}</span>
                <FaIcon :name="matrixOk(pk.composeInstalled) === 'ok' ? 'i-lucide:check' : 'i-lucide:x'" :class="pk.composeInstalled ? 'text-emerald-500' : 'text-muted-foreground'" />
                <span v-if="pk.composeVersion" class="truncate font-mono text-[11px] text-muted-foreground">{{ pk.composeVersion }}</span>
              </div>
              <div class="flex items-center gap-1">
                <span class="text-muted-foreground">{{ $t('container.install.dockerService') }}</span>
                <span :class="pk.dockerRunning ? 'text-emerald-600' : 'text-muted-foreground'">{{ pk.dockerRunning ? $t('container.install.running') : $t('container.install.stopped') }}</span>
              </div>
            </div>
          </div>
          <div v-if="!pk.supported" class="mt-3 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400">
            <FaIcon name="i-lucide:circle-x" class="mr-1" /> {{ pk.reason }}
          </div>
          <div v-else-if="pk.action === 'none'" class="mt-3 rounded-md border border-emerald-300 bg-emerald-50 p-3 text-sm text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-400">
            <FaIcon name="i-lucide:circle-check" class="mr-1" /> {{ $t('container.install.allReady') }}
          </div>
          <div v-else class="mt-3 rounded-md border border-sky-300 bg-sky-50 p-3 text-sm text-sky-700 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-400">
            <FaIcon name="i-lucide:info" class="mr-1" />
            {{ pk.action === 'full' ? $t('container.install.pathFull') : $t('container.install.pathComposeOnly') }}
            <div v-if="pk.note" class="mt-1 text-xs opacity-80">{{ pk.note }}</div>
          </div>
        </template>
      </div>

      <!-- ② 选源 -->
      <div v-else-if="phase === 'source'">
        <p class="text-xs text-muted-foreground">{{ $t('container.install.sourceHint') }}</p>
        <div class="mt-2 grid grid-cols-2 gap-2">
          <button
            v-for="s in SOURCES" :key="s.key"
            class="flex flex-col items-start rounded-md border p-3 text-left transition-colors hover:bg-accent/40"
            :class="source === s.key ? 'border-primary bg-primary/5 ring-1 ring-primary' : ''"
            @click="source = s.key; sourceTest = null"
          >
            <span class="flex items-center gap-1.5 text-sm font-medium">
              {{ $t(s.labelKey) }}
              <span v-if="s.key !== 'official'" class="rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[10px] text-emerald-600">{{ $t('container.install.domestic') }}</span>
            </span>
            <span class="mt-0.5 font-mono text-[11px] text-muted-foreground">{{ s.key === 'official' ? 'download.docker.com' : `mirrors.*` }}</span>
          </button>
        </div>
        <div class="mt-3 flex items-center gap-2">
          <FaButton variant="outline" size="sm" :loading="sourceTesting" @click="doTest">
            <FaIcon name="i-lucide:plug-zap" class="mr-1" /> {{ $t('container.install.testConn') }}
          </FaButton>
          <span v-if="sourceTest" class="truncate text-xs" :class="sourceTest.ok ? 'text-emerald-600' : 'text-red-500'" :title="sourceTest.detail || sourceTest.url">
            {{ sourceTest.ok ? $t('container.install.connOk') : sourceTest.detail || $t('container.install.connFail', { status: sourceTest.status }) }}
          </span>
        </div>
      </div>

      <!-- ③ 确认（dryRun 步骤清单 + 可选加速器） -->
      <div v-else-if="phase === 'confirm'">
        <div v-if="dryRunning" class="flex items-center justify-center gap-2 py-8 text-sm text-muted-foreground">
          <FaIcon name="i-lucide:loader-circle" class="animate-spin text-lg" /> {{ $t('container.install.generating') }}
        </div>
        <template v-else-if="dryRun">
          <div class="flex items-center gap-2 text-sm">
            <span class="font-medium">{{ $t('container.install.stepsTitle') }}</span>
            <span class="rounded-full px-2 py-0.5 text-xs" :class="dryRun.syntaxOk ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-500'">
              {{ dryRun.syntaxOk ? $t('container.install.syntaxOk') : $t('container.install.syntaxBad') }}
            </span>
            <span class="text-xs text-muted-foreground">{{ $t('container.install.stepsCount', { n: installSteps.length }) }}</span>
          </div>
          <ol class="mt-2 space-y-1.5">
            <li v-for="(st, i) in installSteps" :key="i" class="rounded-md border px-3 py-1.5 text-sm">
              <span class="mr-2 font-mono text-xs text-muted-foreground">{{ i + 1 }}.</span>{{ st.name }}
              <details class="mt-1">
                <summary class="cursor-pointer select-none text-[11px] text-muted-foreground">{{ $t('container.install.showCmd') }}</summary>
                <pre class="mt-1 max-h-40 overflow-auto whitespace-pre-wrap rounded bg-muted/50 p-2 font-mono text-[11px]">{{ st.cmd }}</pre>
              </details>
            </li>
          </ol>
          <div class="mt-3 rounded-md border p-3">
            <label class="flex items-center gap-2 text-sm">
              <input v-model="withMirror" type="checkbox" class="accent-(--primary)">
              {{ $t('container.install.mirrorOpt') }}
            </label>
            <div v-if="withMirror" class="mt-2 flex flex-col gap-2">
              <div class="flex flex-wrap gap-3 text-xs">
                <label v-for="m in PRESET_MIRRORS" :key="m.url" class="flex items-center gap-1">
                  <input v-model="checkedMirrors" type="checkbox" :value="m.url" class="accent-(--primary)">
                  {{ m.label }} <span class="font-mono text-muted-foreground">{{ m.url }}</span>
                </label>
              </div>
              <FaInput v-model="customMirror" :placeholder="$t('container.install.mirrorCustomPh')" class="w-full text-xs" />
              <p class="text-[11px] text-muted-foreground">{{ $t('container.install.mirrorHint') }}</p>
            </div>
          </div>
        </template>
      </div>

      <!-- ④ 执行 -->
      <div v-else-if="phase === 'running'">
        <div v-if="taskStatus === 'failed'" class="mb-3 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400">
          <FaIcon name="i-lucide:circle-x" class="mr-1" /> {{ $t('container.install.installFailed') }}
          <div v-if="taskError" class="mt-1 font-mono text-xs opacity-90">{{ taskError }}</div>
        </div>
        <YdLogViewer :logs="taskLog" height="320px" />
      </div>

      <!-- ⑤ 结果 -->
      <div v-else-if="phase === 'done'" class="flex flex-col items-center gap-2 py-6">
        <FaIcon name="i-lucide:circle-check" class="text-3xl text-emerald-500" />
        <p class="text-sm font-medium">{{ $t('container.install.installDone') }}</p>
        <p class="text-xs text-muted-foreground">{{ $t('container.install.installDoneHint') }}</p>
      </div>
    </div>

    <template #footer>
      <div class="flex w-full items-center">
        <div class="ml-auto flex items-center gap-2">
          <template v-if="phase === 'precheck'">
            <FaButton variant="outline" @click="visible = false">{{ $t('common.close') }}</FaButton>
            <FaButton v-if="pk?.supported && pk?.action !== 'none'" @click="phase = 'source'">
              {{ $t('container.install.next') }} <FaIcon name="i-lucide:chevron-right" class="ml-1" />
            </FaButton>
          </template>
          <template v-else-if="phase === 'source'">
            <FaButton variant="outline" @click="phase = 'precheck'">
              <FaIcon name="i-lucide:chevron-left" class="mr-1" /> {{ $t('container.install.prev') }}
            </FaButton>
            <FaButton @click="phase = 'confirm'; doDryRun()">{{ $t('container.install.next') }}</FaButton>
          </template>
          <template v-else-if="phase === 'confirm'">
            <FaButton variant="outline" :disabled="dryRunning" @click="phase = 'source'">
              <FaIcon name="i-lucide:chevron-left" class="mr-1" /> {{ $t('container.install.prev') }}
            </FaButton>
            <FaButton :loading="installing" :disabled="dryRunning || !dryRun?.syntaxOk" @click="doInstall">
              <FaIcon name="i-lucide:download" class="mr-1" /> {{ $t('container.install.startInstall') }}
            </FaButton>
          </template>
          <template v-else-if="phase === 'running'">
            <FaButton v-if="taskStatus === 'failed'" variant="outline" @click="phase = 'source'">
              {{ $t('container.install.retry') }}
            </FaButton>
            <FaButton v-if="taskStatus !== 'running'" @click="visible = false">{{ $t('common.close') }}</FaButton>
          </template>
          <template v-else-if="phase === 'done'">
            <FaButton @click="visible = false">{{ $t('container.install.done') }}</FaButton>
          </template>
        </div>
      </div>
    </template>
  </FaModal>
</template>
