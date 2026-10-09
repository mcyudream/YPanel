<script setup lang="ts">
import type { PHPExtensionsResp, RuntimeItem } from '@/api/modules/runtime'
import apiRT from '@/api/modules/runtime'
import apiCompose from '@/api/modules/compose'
import { taskApi } from '@/api/modules/task'
import { useFileEditorStore } from '@/store/modules/fileEditor'
import { useTaskCenterStore } from '@/store/modules/taskCenter'
import { i18n, tr } from '@/locales'
import YdLogViewer from '@/components/YdLogViewer/index.vue'
import YdDirPicker from '@/components/YdDirPicker/index.vue'

defineOptions({
  name: 'RuntimesIndex',
})

const fileEditorStore = useFileEditorStore()
const taskCenter = useTaskCenterStore()
const appAccountStore = useAppAccountStore()

function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}

// ---- 列表 ----
const runtimes = ref<RuntimeItem[]>([])
const loading = ref(false)

const typeLabels: Record<string, string> = { php: 'PHP', node: 'Node', python: 'Python', java: 'Java', go: 'Go' }
const typeBadgeCls: Record<string, string> = {
  php: 'bg-violet-500/10 text-violet-600',
  node: 'bg-emerald-500/10 text-emerald-600',
  python: 'bg-sky-500/10 text-sky-600',
  java: 'bg-orange-500/10 text-orange-600',
  go: 'bg-cyan-500/10 text-cyan-600',
}

function statusMeta(r: RuntimeItem) {
  if (r.status === 'error') {
    return { label: tr('runtimes.status.error'), cls: 'bg-red-500/10 text-red-600' }
  }
  if (r.status === 'building' || r.status === 'creating') {
    return { label: r.status === 'building' ? tr('runtimes.status.building') : tr('runtimes.status.creating'), cls: 'bg-amber-500/10 text-amber-600 animate-pulse' }
  }
  return r.running
    ? { label: tr('runtimes.status.running'), cls: 'bg-emerald-500/10 text-emerald-600' }
    : { label: tr('runtimes.status.stopped'), cls: 'bg-muted text-muted-foreground' }
}

async function load() {
  loading.value = true
  try {
    runtimes.value = await apiRT.list()
  }
  finally {
    loading.value = false
  }
}

// 存在构建/创建中行时自动轮询刷新
let pollTimer: ReturnType<typeof setInterval> | null = null
function ensurePolling() {
  if (pollTimer) {
    return
  }
  pollTimer = setInterval(async () => {
    const busy = runtimes.value.some(r => r.status === 'building' || r.status === 'creating')
    if (!busy) {
      return
    }
    try {
      runtimes.value = await apiRT.list()
      if (current.value) {
        const hit = runtimes.value.find(r => r.id === current.value!.id)
        if (hit) {
          current.value = hit
        }
      }
    }
    catch {}
  }, 3000)
}
onMounted(() => {
  load()
  ensurePolling()
})
onBeforeUnmount(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})

// ---- 创建（多类型）----
const createVisible = ref(false)
const creating = ref(false)
const catalog = ref<{ catalog: { name: string, desc: string }[], templates: { name: string, desc: string, extensions: string[] }[] }>({ catalog: [], templates: [] })

const phpVersionOptions = [
  { label: 'PHP 7.4 (fpm-alpine)', value: '7.4' },
  { label: 'PHP 8.0 (fpm-alpine)', value: '8.0' },
  { label: 'PHP 8.1 (fpm-alpine)', value: '8.1' },
  { label: 'PHP 8.2 (fpm-alpine)', value: '8.2' },
  { label: 'PHP 8.3 (fpm-alpine)', value: '8.3' },
  { label: 'PHP 8.4 (fpm-alpine)', value: '8.4' },
]
const codeVersionOptions: Record<string, { label: string, value: string }[]> = {
  node: [
    { label: 'Node 18 (LTS)', value: '18' },
    { label: 'Node 20 (LTS)', value: '20' },
    { label: 'Node 22 (LTS)', value: '22' },
  ],
  python: [
    { label: 'Python 3.10', value: '3.10' },
    { label: 'Python 3.11', value: '3.11' },
    { label: 'Python 3.12', value: '3.12' },
    { label: 'Python 3.13', value: '3.13' },
  ],
  java: [
    { label: 'Java 8 (Temurin JRE)', value: '8' },
    { label: 'Java 11 (Temurin JRE)', value: '11' },
    { label: 'Java 17 (Temurin JRE)', value: '17' },
    { label: 'Java 21 (Temurin JRE)', value: '21' },
  ],
  go: [
    { label: 'Go 1.22', value: '1.22' },
    { label: 'Go 1.23', value: '1.23' },
    { label: 'Go 1.24', value: '1.24' },
  ],
}
const pkgMgrOptions = computed(() => [
  { label: i18n.global.t('runtimes.pkgMgrAuto'), value: 'auto' },
  { label: 'npm', value: 'npm' },
  { label: 'yarn', value: 'yarn' },
  { label: 'pnpm', value: 'pnpm' },
])

const createForm = ref({
  type: 'php',
  name: '',
  version: '8.2',
  extensions: [] as string[],
  remark: '',
  codeDir: '',
  startCmd: '',
  autoInstall: true,
  pkgMgr: 'auto',
  ports: [] as { host: string, container: string }[],
})

const createVersionOptions = computed(() =>
  createForm.value.type === 'php' ? phpVersionOptions : (codeVersionOptions[createForm.value.type] || []),
)

function switchCreateType() {
  const t = createForm.value.type
  createForm.value.version = t === 'php' ? '8.2' : (codeVersionOptions[t]?.[1]?.value || codeVersionOptions[t]?.[0]?.value || '')
}

async function openCreate() {
  createForm.value = {
    type: 'php', name: '', version: '8.2', extensions: [], remark: '',
    codeDir: '', startCmd: '', autoInstall: true, pkgMgr: 'auto', ports: [],
  }
  createVisible.value = true
  if (!catalog.value.catalog.length) {
    try {
      const resp = await apiRT.phpCatalog()
      catalog.value = resp
    }
    catch {}
  }
}

function applyTemplate(extensions: string[]) {
  for (const e of extensions) {
    if (!createForm.value.extensions.includes(e)) {
      createForm.value.extensions.push(e)
    }
  }
}

function toggleExt(name: string) {
  const i = createForm.value.extensions.indexOf(name)
  if (i >= 0) {
    createForm.value.extensions.splice(i, 1)
  }
  else {
    createForm.value.extensions.push(name)
  }
}

function addPortRow() {
  createForm.value.ports.push({ host: '', container: '' })
}
function removePortRow(i: number) {
  createForm.value.ports.splice(i, 1)
}

// 目录选择器
const dirPickerVisible = ref(false)
function onDirPicked(path: string) {
  createForm.value.codeDir = path
}

async function doCreate() {
  creating.value = true
  try {
    const f = createForm.value
    const ports = f.ports
      .map(p => ({ host: Number(p.host), container: Number(p.container) }))
      .filter(p => Number.isInteger(p.host) && Number.isInteger(p.container) && p.host > 0 && p.container > 0)
    const out = await apiRT.create({
      name: f.name,
      type: f.type,
      version: f.version,
      extensions: f.type === 'php' ? f.extensions : undefined,
      remark: f.remark || undefined,
      codeDir: f.type === 'php' ? undefined : f.codeDir,
      startCmd: f.type === 'php' ? undefined : f.startCmd,
      autoInstall: f.type === 'php' ? undefined : f.autoInstall,
      pkgMgr: f.type === 'node' ? f.pkgMgr : undefined,
      ports: f.type === 'php' ? undefined : ports,
    })
    useFaToast().success(i18n.global.t('runtimes.createStarted'))
    createVisible.value = false
    taskCenter.open(out.taskId)
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.createFailed'), { description: e?.message })
  }
  finally {
    creating.value = false
  }
}

// ---- 接管本机 PHP（保留 M16 能力）----
const extVisible = ref(false)
const extForm = ref({ name: '', version: '8.2', fcgiAddr: '127.0.0.1:9000', remark: '' })
const extSaving = ref(false)

async function doAttach() {
  extSaving.value = true
  try {
    await apiRT.attachExternal(extForm.value)
    useFaToast().success(i18n.global.t('runtimes.attached'))
    extVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.attachFailed'), { description: e?.message })
  }
  finally {
    extSaving.value = false
  }
}

// ---- 启停/删除 ----
async function toggle(r: RuntimeItem) {
  try {
    if (r.running) {
      await apiRT.stop(r.id)
      useFaToast().success(i18n.global.t('runtimes.status.stopped'))
    }
    else {
      await apiRT.start(r.id)
      useFaToast().success(i18n.global.t('runtimes.started'))
    }
    await load()
    if (current.value) {
      const hit = runtimes.value.find(x => x.id === current.value!.id)
      if (hit) {
        current.value = hit
      }
    }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.opFailed'), { description: e?.message })
  }
}

function toggleCurrent() {
  if (current.value) {
    toggle(current.value)
  }
}

function remove(r: RuntimeItem) {
  const external = r.origin === 'external'
  const modal = useFaModal()
  modal.confirm({
    title: external ? i18n.global.t('runtimes.detachTitle') : i18n.global.t('runtimes.deleteTitle'),
    content: external
      ? i18n.global.t('runtimes.detachConfirm', { name: r.name, addr: r.fcgiAddr })
      : i18n.global.t('runtimes.deleteConfirm', { type: typeLabels[r.type] || r.type, version: r.version, name: r.name }),
    onConfirm: async () => {
      try {
        await apiRT.remove(r.id)
        useFaToast().success(external ? i18n.global.t('runtimes.detached') : i18n.global.t('runtimes.deleted'))
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('runtimes.deleteFailed'), { description: e?.message })
      }
    },
  })
}

// ---- 详情抽屉 ----
const drawerVisible = ref(false)
const current = ref<RuntimeItem | null>(null)
const drawerTab = ref('overview')

watch(drawerTab, (t) => {
  if (!current.value) {
    return
  }
  if (t === 'processes') {
    loadSupervisor()
  }
  else if (t === 'slowlog') {
    loadSlowLog()
  }
  else if (t === 'backups') {
    loadBackups()
  }
})

const drawerTabs = computed(() => {
  if (!current.value || current.value.origin === 'external') {
    return [{ label: i18n.global.t('runtimes.tab.overview'), value: 'overview' }]
  }
  if (current.value.type === 'php') {
    return [
      { label: i18n.global.t('runtimes.tab.overview'), value: 'overview' },
      { label: i18n.global.t('runtimes.tab.extensions'), value: 'extensions' },
      { label: i18n.global.t('runtimes.tab.config'), value: 'config' },
      { label: i18n.global.t('runtimes.tab.fpm'), value: 'fpm' },
      { label: i18n.global.t('runtimes.tab.processes'), value: 'processes' },
      { label: i18n.global.t('runtimes.tab.slowlog'), value: 'slowlog' },
      { label: i18n.global.t('runtimes.tab.backups'), value: 'backups' },
    ]
  }
  const tabs = [
    { label: i18n.global.t('runtimes.tab.overview'), value: 'overview' },
    { label: i18n.global.t('runtimes.tab.logs'), value: 'logs' },
    { label: i18n.global.t('runtimes.tab.backups'), value: 'backups' },
  ]
  if (current.value.type === 'node') {
    tabs.splice(2, 0, { label: i18n.global.t('runtimes.tab.modules'), value: 'modules' })
  }
  return tabs
})

async function manage(r: RuntimeItem) {
  current.value = r
  drawerTab.value = 'overview'
  drawerVisible.value = true
  if (r.origin !== 'external') {
    if (r.type === 'php') {
      loadExtensions(r)
      loadQuickConfig(r)
      loadFpmConfig(r)
    }
    else {
      loadLogs()
      if (r.type === 'node') {
        loadNodeModules(r)
      }
    }
  }
}

function refreshCurrent() {
  if (current.value) {
    const hit = runtimes.value.find(x => x.id === current.value!.id)
    if (hit) {
      current.value = hit
    }
  }
}

// ---- 通用任务跟踪 ----
async function watchTask(taskId: number, title: string, after?: () => void) {
  taskCenter.open(taskId)
  const started = Date.now()
  const timer = setInterval(async () => {
    try {
      const t = await taskApi.get(taskId)
      if (t.status !== 'running') {
        clearInterval(timer)
        if (t.status === 'success') {
          useFaToast().success(i18n.global.t('runtimes.taskDone', { title }))
        }
        else {
          useFaToast().error(i18n.global.t('runtimes.taskFailed', { title }), { description: t.error || i18n.global.t('runtimes.seeTaskLog') })
        }
        after?.()
      }
      else if (Date.now() - started > 20 * 60 * 1000) {
        clearInterval(timer)
      }
    }
    catch {
      clearInterval(timer)
    }
  }, 3000)
}

// ---- PHP 扩展 ----
const extData = ref<PHPExtensionsResp | null>(null)
const extLoading = ref(false)
async function loadExtensions(r: RuntimeItem) {
  extLoading.value = true
  try {
    extData.value = await apiRT.phpExtensions(r.id)
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.extListFailed'), { description: e?.message })
  }
  finally {
    extLoading.value = false
  }
}

async function operateExt(name: string, install: boolean) {
  if (!current.value) {
    return
  }
  const modal = useFaModal()
  modal.confirm({
    title: install ? i18n.global.t('runtimes.extInstallTitle', { name }) : i18n.global.t('runtimes.extUninstallTitle', { name }),
    content: install
      ? i18n.global.t('runtimes.extInstallConfirm', { name })
      : i18n.global.t('runtimes.extUninstallConfirm', { name }),
    onConfirm: async () => {
      try {
        const out = install
          ? await apiRT.phpExtensionInstall(current.value!.id, name)
          : await apiRT.phpExtensionUninstall(current.value!.id, name)
        watchTask(out.taskId, install ? i18n.global.t('runtimes.extInstallTask', { name }) : i18n.global.t('runtimes.extUninstallTask', { name }), () => {
          if (current.value) {
            loadExtensions(current.value)
          }
        })
      }
      catch (e: any) {
        useFaToast().error(install ? i18n.global.t('runtimes.extInstallTaskFailed') : i18n.global.t('runtimes.extUninstallTaskFailed'), { description: e?.message })
      }
    },
  })
}

// ---- php.ini 快捷配置 ----
const quickForm = ref({ memoryLimit: '', uploadMaxSize: '', maxExecutionTime: '', disableFunctionsText: '' })
const quickSaving = ref(false)
async function loadQuickConfig(r: RuntimeItem) {
  try {
    const cfg = await apiRT.phpConfig(r.id)
    quickForm.value = {
      memoryLimit: cfg.memoryLimit,
      uploadMaxSize: cfg.uploadMaxSize,
      maxExecutionTime: cfg.maxExecutionTime,
      disableFunctionsText: (cfg.disableFunctions || []).join(', '),
    }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.configLoadFailed'), { description: e?.message })
  }
}

async function saveQuickConfig() {
  if (!current.value) {
    return
  }
  quickSaving.value = true
  try {
    const fns = quickForm.value.disableFunctionsText.split(/[,\n]/).map(s => s.trim()).filter(Boolean)
    await apiRT.phpConfigSave(current.value.id, {
      memoryLimit: quickForm.value.memoryLimit,
      uploadMaxSize: quickForm.value.uploadMaxSize,
      maxExecutionTime: quickForm.value.maxExecutionTime,
      disableFunctions: fns,
    })
    useFaToast().success(i18n.global.t('runtimes.configSaved'))
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.saveRolledBack'), { description: e?.message })
  }
  finally {
    quickSaving.value = false
  }
}

// ---- FPM 进程池 ----
const fpmForm = ref<Record<string, string>>({})
const fpmSaving = ref(false)
async function loadFpmConfig(r: RuntimeItem) {
  try {
    fpmForm.value = await apiRT.fpmConfig(r.id)
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.fpmLoadFailed'), { description: e?.message })
  }
}

async function saveFpmConfig() {
  if (!current.value) {
    return
  }
  fpmSaving.value = true
  try {
    await apiRT.fpmConfigSave(current.value.id, { ...fpmForm.value })
    useFaToast().success(i18n.global.t('runtimes.fpmSaved'))
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.saveRolledBack'), { description: e?.message })
  }
  finally {
    fpmSaving.value = false
  }
}

// ---- FPM 状态 ----
const fpmStatusItems = ref<{ key: string, value: string }[]>([])
const fpmStatusLoading = ref(false)
async function loadFpmStatus() {
  if (!current.value) {
    return
  }
  fpmStatusLoading.value = true
  try {
    fpmStatusItems.value = (await apiRT.fpmStatus(current.value.id)).items
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.fpmStatusFailed'), { description: e?.message })
  }
  finally {
    fpmStatusLoading.value = false
  }
}

// ---- 代码运行时日志 ----
const codeLogs = ref('')
const logsLoading = ref(false)
async function loadLogs() {
  if (!current.value) {
    return
  }
  logsLoading.value = true
  try {
    const token = appAccountStore.token
    const dir = `/opt/ypanel/runtime/${current.value.type}/${current.value.name}`
    const url = `${wsBase()}/${apiCompose.logsURL(current.value.composeProject, dir, token, 500, false)}`
    const resp = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    if (!resp.ok) {
      throw new Error(`HTTP ${resp.status}`)
    }
    codeLogs.value = await resp.text()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.logFailed'), { description: e?.message })
  }
  finally {
    logsLoading.value = false
  }
}

// ---- Node 依赖管理 ----
const nodeModules = ref<{ name: string, version: string, dev: boolean }[]>([])
const nodePkgMgr = ref('auto')
const nodeHasPkg = ref(false)
const nodeModulesLoading = ref(false)
const nodeInstallName = ref('')
const nodeModuleBusy = ref(false)

async function loadNodeModules(r: RuntimeItem) {
  nodeModulesLoading.value = true
  try {
    const resp = await apiRT.nodeModules(r.id)
    nodeModules.value = resp.modules || []
    nodePkgMgr.value = resp.pkgMgr || 'auto'
    nodeHasPkg.value = resp.packageJson
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.modulesLoadFailed'), { description: e?.message })
  }
  finally {
    nodeModulesLoading.value = false
  }
}

async function operateNodeModule(operate: 'install' | 'uninstall' | 'update', name: string) {
  if (!current.value) {
    return
  }
  nodeModuleBusy.value = true
  try {
    const out = await apiRT.nodeModuleOperate(current.value.id, operate, name)
    watchTask(out.taskId, i18n.global.t('runtimes.moduleTask', { name: name || i18n.global.t('runtimes.wordInstall'), op: operate }), () => {
      if (current.value) {
        loadNodeModules(current.value)
        loadLogs()
      }
    })
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.opFailed'), { description: e?.message })
  }
  finally {
    nodeModuleBusy.value = false
  }
}

// ---- 原文编辑（M20 工作台，保存后需重启）----
function editFile(r: RuntimeItem, type: 'php' | 'fpm') {
  const p = type === 'php'
    ? `/opt/ypanel/runtime/php/${r.name}/conf/php.ini`
    : `/opt/ypanel/runtime/php/${r.name}/conf/php-fpm.conf`
  fileEditorStore.openWorkspace(p, 'local')
}

async function restartRuntime(r: RuntimeItem) {
  try {
    await apiRT.restart(r.id)
    useFaToast().success(i18n.global.t('runtimes.restarted'))
    await load()
    refreshCurrent()
    if (r.type !== 'php') {
      loadLogs()
    }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.opFailed'), { description: e?.message })
  }
}

// ---- 重建镜像 ----
function rebuildRuntime(r: RuntimeItem) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('runtimes.rebuild'),
    content: i18n.global.t('runtimes.rebuildConfirm', { name: r.name }),
    onConfirm: async () => {
      try {
        const out = await apiRT.rebuild(r.id)
        watchTask(out.taskId, i18n.global.t('runtimes.rebuildTask'), () => {
          load()
        })
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('runtimes.rebuildTaskFailed'), { description: e?.message })
      }
    },
  })
}

// ---- 进程管理（supervisor）----
const supList = ref<{ name: string, status: string, detail: string }[]>([])
const supLoading = ref(false)
const supError = ref('')
const supForm = ref({ name: '', command: '', autoStart: true, autoRestart: true })
const supSaving = ref(false)
const supLogName = ref('')
const supLog = ref('')

async function loadSupervisor() {
  if (!current.value) {
    return
  }
  supLoading.value = true
  supError.value = ''
  try {
    supList.value = await apiRT.supervisorList(current.value.id)
  }
  catch (e: any) {
    supError.value = e?.message || i18n.global.t('runtimes.supListFailed')
    supList.value = []
  }
  finally {
    supLoading.value = false
  }
}

async function saveSupervisor() {
  if (!current.value) {
    return
  }
  supSaving.value = true
  try {
    await apiRT.supervisorUpsert(current.value.id, { ...supForm.value })
    useFaToast().success(i18n.global.t('runtimes.supSaved'))
    supForm.value = { name: '', command: '', autoStart: true, autoRestart: true }
    loadSupervisor()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.saveFailed'), { description: e?.message })
  }
  finally {
    supSaving.value = false
  }
}

async function operateSupervisor(name: string, action: 'start' | 'stop' | 'restart') {
  if (!current.value) {
    return
  }
  try {
    await apiRT.supervisorOperate(current.value.id, name, action)
    useFaToast().success(i18n.global.t('runtimes.supOpDone', { name, action }))
    loadSupervisor()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.opFailed'), { description: e?.message })
  }
}

function removeSupervisor(name: string) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('runtimes.supDeleteTitle', { name }),
    content: i18n.global.t('runtimes.supDeleteConfirm'),
    onConfirm: async () => {
      if (!current.value) {
        return
      }
      try {
        await apiRT.supervisorRemove(current.value.id, name)
        useFaToast().success(i18n.global.t('runtimes.deleted'))
        loadSupervisor()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('runtimes.deleteFailed'), { description: e?.message })
      }
    },
  })
}

async function showSupervisorLog(name: string) {
  if (!current.value) {
    return
  }
  try {
    supLogName.value = name
    supLog.value = (await apiRT.supervisorLog(current.value.id, name)).log
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.logFailed'), { description: e?.message })
  }
}

// ---- 慢日志 ----
const slowLogText = ref('')
const slowLogLoading = ref(false)
async function loadSlowLog() {
  if (!current.value) {
    return
  }
  slowLogLoading.value = true
  try {
    const resp = await apiRT.slowLog(current.value.id)
    slowLogText.value = resp.log || ''
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.slowLogFailed'), { description: e?.message })
  }
  finally {
    slowLogLoading.value = false
  }
}

async function clearSlowLog() {
  if (!current.value) {
    return
  }
  try {
    await apiRT.slowLogClear(current.value.id)
    useFaToast().success(i18n.global.t('runtimes.cleared'))
    loadSlowLog()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.clearFailed'), { description: e?.message })
  }
}

// ---- 备份恢复 ----
const backups = ref<{ file: string, size: string, at: string }[]>([])
const backupsLoading = ref(false)
async function loadBackups() {
  if (!current.value) {
    return
  }
  backupsLoading.value = true
  try {
    backups.value = await apiRT.backupsList(current.value.id)
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('runtimes.backupsLoadFailed'), { description: e?.message })
  }
  finally {
    backupsLoading.value = false
  }
}

function createBackup() {
  if (!current.value) {
    return
  }
  const id = current.value.id
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('runtimes.backupCreateTitle'),
    content: i18n.global.t('runtimes.backupCreateConfirm'),
    onConfirm: async () => {
      try {
        const out = await apiRT.backupsCreate(id)
        watchTask(out.taskId, i18n.global.t('runtimes.backupCreateTask'), () => {
          loadBackups()
        })
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('runtimes.backupTaskFailed'), { description: e?.message })
      }
    },
  })
}

function restoreBackup(file: string) {
  if (!current.value) {
    return
  }
  const id = current.value.id
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('runtimes.restoreTitle', { file }),
    content: i18n.global.t('runtimes.restoreConfirm'),
    onConfirm: async () => {
      try {
        const out = await apiRT.backupsRestore(id, file)
        watchTask(out.taskId, i18n.global.t('runtimes.restoreTask'), () => {
          load()
          loadBackups()
        })
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('runtimes.restoreTaskFailed'), { description: e?.message })
      }
    },
  })
}

function removeBackup(file: string) {
  if (!current.value) {
    return
  }
  const id = current.value.id
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('runtimes.backupDeleteTitle', { file }),
    content: i18n.global.t('runtimes.backupDeleteConfirm'),
    onConfirm: async () => {
      try {
        await apiRT.backupsRemove(id, file)
        useFaToast().success(i18n.global.t('runtimes.deleted'))
        loadBackups()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('runtimes.deleteFailed'), { description: e?.message })
      }
    },
  })
}
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="file-code" :size="24" />
          <span>{{ $t('runtimes.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('runtimes.desc') }}</span>
      </template>
      <div class="flex gap-2">
        <FaButton size="sm" variant="outline" @click="extVisible = true">
          <FaIcon name="i-lucide:plug-zap" class="mr-1" /> {{ $t('runtimes.attach') }}
        </FaButton>
        <FaButton size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('runtimes.create') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="mb-3 flex justify-end">
        <YdDockerNodeSelect />
      </div>
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div v-for="r in runtimes" :key="r.id" class="rounded-lg border bg-background p-4">
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-2">
              <YdMorphIcon name="file-code" :size="22" class="text-indigo-500" />
              <div>
                <div class="flex items-center gap-1.5">
                  <span class="font-medium">{{ r.name }}</span>
                  <span
                    class="rounded px-1.5 py-0.5 text-xs"
                    :class="r.origin === 'external' ? 'bg-blue-500/10 text-blue-600' : 'bg-muted text-muted-foreground'"
                  >
                    {{ r.origin === 'external' ? $t('runtimes.originExternal') : $t('runtimes.originContainer') }}
                  </span>
                  <span class="rounded px-1.5 py-0.5 text-xs" :class="typeBadgeCls[r.type] || 'bg-muted text-muted-foreground'">
                    {{ typeLabels[r.type] || r.type }} {{ r.version }}
                  </span>
                </div>
                <div class="mt-0.5 max-w-64 truncate text-xs text-muted-foreground" :title="r.origin === 'external' ? r.fcgiAddr : (r.codeDir || r.containerName)">
                  {{ r.origin === 'external' ? r.fcgiAddr : (r.codeDir || r.containerName) }}
                  <span v-if="r.extensions?.length"> · {{ r.extensions.join(' / ') }}</span>
                </div>
              </div>
            </div>
            <span class="shrink-0 rounded-full px-2 py-0.5 text-xs" :class="statusMeta(r).cls" :title="r.status === 'error' ? r.message : ''">
              {{ statusMeta(r).label }}
            </span>
          </div>
          <div v-if="r.status === 'error' && r.message" class="mt-2 line-clamp-2 rounded bg-red-500/5 px-2 py-1 font-mono text-xs text-red-600">
            {{ r.message }}
          </div>
          <div class="mt-3 flex items-center justify-between border-t pt-3">
            <span class="font-mono text-xs text-muted-foreground">{{ r.composeProject }}</span>
            <div class="flex gap-1">
              <FaButton variant="outline" size="sm" @click="manage(r)">{{ $t('runtimes.manage') }}</FaButton>
              <FaButton v-if="r.origin !== 'external'" variant="outline" size="sm" @click="toggle(r)">
                {{ r.running ? $t('common.stop') : $t('common.start') }}
              </FaButton>
              <FaButton variant="outline" size="sm" class="text-red-500!" @click="remove(r)">
                {{ r.origin === 'external' ? $t('runtimes.detach') : $t('common.delete') }}
              </FaButton>
            </div>
          </div>
        </div>
        <div v-if="!runtimes.length && !loading" class="rounded-lg border p-10 text-center text-sm text-muted-foreground md:col-span-2 xl:col-span-3">
          {{ $t('runtimes.empty') }}
        </div>
      </div>
    </FaPageMain>

    <!-- 详情抽屉 -->
    <FaDrawer v-model="drawerVisible" :title="current ? $t('runtimes.drawerTitle', { name: current.name }) : $t('runtimes.title')">
      <template v-if="current">
        <div v-if="current.status === 'error' && current.message" class="mb-3">
          <FaAlert variant="destructive" :title="current.message" />
        </div>
        <FaTabs v-model="drawerTab" :list="drawerTabs" />

        <!-- 概览 -->
        <div v-if="drawerTab === 'overview'" class="mt-3 flex flex-col gap-3">
          <div class="grid grid-cols-2 gap-3">
            <div class="rounded-md border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('runtimes.typeVersion') }}</div>
              <div class="mt-1 text-sm">{{ typeLabels[current.type] || current.type }} {{ current.version }}</div>
            </div>
            <div class="rounded-md border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('common.status') }}</div>
              <div class="mt-1 text-sm">{{ statusMeta(current).label }}</div>
            </div>
            <div class="col-span-2 rounded-md border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('runtimes.imageContainer') }}</div>
              <div class="mt-1 truncate font-mono text-xs" :title="current.image">
                {{ current.origin === 'external' ? $t('runtimes.externalFcgi', { addr: current.fcgiAddr }) : `${current.image} · ${current.containerName}` }}
              </div>
            </div>
            <div v-if="current.codeDir" class="col-span-2 rounded-md border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('runtimes.codeDir') }}</div>
              <div class="mt-1 font-mono text-xs">{{ current.codeDir }}</div>
            </div>
            <div v-if="current.startCmd" class="col-span-2 rounded-md border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('runtimes.startCmd') }}</div>
              <div class="mt-1 font-mono text-xs">{{ current.startCmd }}</div>
            </div>
            <div v-if="current.ports?.length" class="col-span-2 rounded-md border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('runtimes.portsCol') }}</div>
              <div class="mt-1 flex flex-wrap gap-2">
                <span v-for="(p, i) in current.ports" :key="i" class="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
                  {{ p.host }} → {{ p.container }}/{{ p.protocol || 'tcp' }}
                </span>
              </div>
            </div>
            <div class="col-span-2 rounded-md border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('runtimes.composeProject') }}</div>
              <div class="mt-1 font-mono text-xs">{{ current.composeProject }}</div>
            </div>
          </div>
          <div v-if="current.origin !== 'external'" class="flex flex-wrap gap-2">
            <FaButton size="sm" variant="outline" @click="toggleCurrent">
              {{ current.running ? $t('common.stop') : $t('common.start') }}
            </FaButton>
            <FaButton size="sm" variant="outline" @click="restartRuntime(current)">
              <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('common.restart') }}
            </FaButton>
            <FaButton v-if="current.type === 'php'" size="sm" variant="outline" @click="rebuildRuntime(current)">
              <FaIcon name="i-lucide:hammer" class="mr-1" /> {{ $t('runtimes.rebuild') }}
            </FaButton>
            <template v-if="current.type === 'php'">
              <FaButton size="sm" variant="outline" @click="editFile(current, 'php')">{{ $t('runtimes.editPhpIni') }}</FaButton>
              <FaButton size="sm" variant="outline" @click="editFile(current, 'fpm')">{{ $t('runtimes.editFpmConf') }}</FaButton>
            </template>
          </div>
          <div class="text-xs text-muted-foreground">
            <template v-if="current.type === 'php'">
              {{ $t('runtimes.phpEditHint') }}
            </template>
            <template v-else>
              {{ $t('runtimes.codeHint') }}
            </template>
          </div>
        </div>

        <!-- 扩展（仅 PHP）-->
        <div v-if="drawerTab === 'extensions' && current.origin !== 'external'" class="mt-3 flex flex-col gap-3">
          <FaButton size="sm" variant="outline" :loading="extLoading" @click="loadExtensions(current)">
            <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('runtimes.extRefresh') }}
          </FaButton>
          <FaDivider>{{ $t('runtimes.extInstalled') }}</FaDivider>
          <div class="flex flex-wrap gap-1.5">
            <FaTooltip v-for="e in extData?.catalog.filter(x => x.installed)" :key="e.name" :text="e.desc">
              <span class="inline-flex items-center gap-1 rounded-md border bg-emerald-500/5 px-2 py-1 text-xs">
                {{ e.name }}
                <button
                  class="text-muted-foreground hover:text-red-500" @click="operateExt(e.name, false)"
                >
                  <FaIcon name="i-lucide:x" />
                </button>
              </span>
            </FaTooltip>
            <span v-if="!extData?.catalog.some(x => x.installed)" class="text-xs text-muted-foreground">{{ $t('runtimes.extEmpty') }}</span>
          </div>
          <FaDivider>{{ $t('runtimes.extAvailable') }}</FaDivider>
          <div class="flex flex-col gap-1">
            <div
              v-for="e in extData?.catalog.filter(x => !x.installed)" :key="e.name"
              class="flex items-center justify-between rounded-md border px-3 py-2"
            >
              <div>
                <span class="text-sm">{{ e.name }}</span>
                <span class="ml-2 text-xs text-muted-foreground">{{ e.desc }}</span>
              </div>
              <FaButton size="sm" variant="outline" :disabled="current.status !== 'running'" @click="operateExt(e.name, true)">
                {{ $t('runtimes.install') }}
              </FaButton>
            </div>
            <span v-if="extData && !extData.catalog.some(x => !x.installed)" class="text-xs text-muted-foreground">
              {{ $t('runtimes.extAllInstalled') }}
            </span>
          </div>
        </div>

        <!-- 配置（仅 PHP）-->
        <div v-if="drawerTab === 'config' && current.origin !== 'external'" class="mt-3 flex flex-col gap-4">
          <FaDivider>{{ $t('runtimes.phpQuickSettings') }}</FaDivider>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
            <div>
              <div class="mb-1 text-xs text-muted-foreground">memory_limit</div>
              <FaInput v-model="quickForm.memoryLimit" placeholder="256M" class="w-full" />
            </div>
            <div>
              <div class="mb-1 text-xs text-muted-foreground">{{ $t('runtimes.uploadLimit') }}</div>
              <FaInput v-model="quickForm.uploadMaxSize" placeholder="50M" class="w-full" />
            </div>
            <div>
              <div class="mb-1 text-xs text-muted-foreground">{{ $t('runtimes.execTimeout') }}</div>
              <FaInput v-model="quickForm.maxExecutionTime" placeholder="60" class="w-full" />
            </div>
            <div class="md:col-span-3">
              <div class="mb-1 text-xs text-muted-foreground">{{ $t('runtimes.disableFunctions') }}</div>
              <FaInput v-model="quickForm.disableFunctionsText" placeholder="exec, shell_exec, system" class="w-full" />
            </div>
          </div>
          <div>
            <FaButton size="sm" :loading="quickSaving" :disabled="current.status !== 'running'" @click="saveQuickConfig">{{ $t('runtimes.saveAutoRestart') }}</FaButton>
          </div>

          <FaDivider>{{ $t('runtimes.fpmPool') }}</FaDivider>
          <div class="grid grid-cols-2 gap-3 md:grid-cols-3">
            <div>
              <div class="mb-1 text-xs text-muted-foreground">{{ $t('runtimes.pmMode') }}</div>
              <FaSelect
                v-model="fpmForm.pm" :options="[
                  { label: $t('runtimes.pmDynamic'), value: 'dynamic' },
                  { label: $t('runtimes.pmStatic'), value: 'static' },
                  { label: $t('runtimes.pmOndemand'), value: 'ondemand' },
                ]" class="w-full"
              />
            </div>
            <div>
              <div class="mb-1 text-xs text-muted-foreground">max_children</div>
              <FaInput v-model="fpmForm['pm.max_children']" class="w-full" />
            </div>
            <div>
              <div class="mb-1 text-xs text-muted-foreground">start_servers</div>
              <FaInput v-model="fpmForm['pm.start_servers']" class="w-full" />
            </div>
            <div>
              <div class="mb-1 text-xs text-muted-foreground">min_spare_servers</div>
              <FaInput v-model="fpmForm['pm.min_spare_servers']" class="w-full" />
            </div>
            <div>
              <div class="mb-1 text-xs text-muted-foreground">max_spare_servers</div>
              <FaInput v-model="fpmForm['pm.max_spare_servers']" class="w-full" />
            </div>
            <div>
              <div class="mb-1 text-xs text-muted-foreground">max_requests</div>
              <FaInput v-model="fpmForm['pm.max_requests']" class="w-full" />
            </div>
          </div>
          <div>
            <FaButton size="sm" :loading="fpmSaving" :disabled="current.status !== 'running'" @click="saveFpmConfig">{{ $t('runtimes.saveAutoRestart') }}</FaButton>
          </div>
        </div>

        <!-- FPM 状态 -->
        <div v-if="drawerTab === 'fpm'" class="mt-3 flex flex-col gap-3">
          <div>
            <FaButton size="sm" variant="outline" :loading="fpmStatusLoading" :disabled="current.origin !== 'container' || current.status !== 'running'" @click="loadFpmStatus">
              <FaIcon name="i-lucide:activity" class="mr-1" /> {{ $t('runtimes.refreshStatus') }}
            </FaButton>
          </div>
          <div v-if="fpmStatusItems.length" class="rounded-md border">
            <div class="flex items-center justify-between border-b px-3 py-1.5 text-xs text-muted-foreground">
              <span>{{ $t('runtimes.metric') }}</span><span>{{ $t('runtimes.value') }}</span>
            </div>
            <div v-for="it in fpmStatusItems" :key="it.key" class="flex items-center justify-between border-b px-3 py-1.5 text-sm last:border-b-0">
              <span class="text-muted-foreground">{{ it.key }}</span>
              <span class="font-mono text-xs">{{ it.value }}</span>
            </div>
          </div>
          <div v-else class="text-xs text-muted-foreground">
            {{ $t('runtimes.fpmStatusHint') }}
          </div>
        </div>

        <!-- 进程管理（supervisor，仅 PHP）-->
        <div v-if="drawerTab === 'processes'" class="mt-3 flex flex-col gap-3">
          <div class="flex items-center justify-between">
            <FaButton size="sm" variant="outline" :loading="supLoading" @click="loadSupervisor">
              <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('common.refresh') }}
            </FaButton>
          </div>
          <FaAlert v-if="supError" variant="destructive" :title="supError" />
          <div v-else class="flex flex-col gap-1">
            <div v-for="p in supList" :key="p.name" class="flex items-center justify-between rounded-md border px-3 py-2">
              <div>
                <span class="font-mono text-sm">{{ p.name }}</span>
                <span class="ml-2 text-xs" :class="p.status === 'RUNNING' ? 'text-emerald-600' : 'text-muted-foreground'">{{ p.status }}</span>
                <span class="ml-2 font-mono text-xs text-muted-foreground">{{ p.detail }}</span>
              </div>
              <div class="flex gap-1">
                <FaButton v-if="p.name !== 'php-fpm'" size="sm" variant="outline" @click="showSupervisorLog(p.name)">{{ $t('runtimes.logBtn') }}</FaButton>
                <FaButton v-if="p.name !== 'php-fpm'" size="sm" variant="outline" @click="operateSupervisor(p.name, 'restart')">{{ $t('common.restart') }}</FaButton>
                <FaButton v-if="p.name !== 'php-fpm' && p.status !== 'RUNNING'" size="sm" variant="outline" @click="operateSupervisor(p.name, 'start')">{{ $t('common.start') }}</FaButton>
                <FaButton v-if="p.name !== 'php-fpm' && p.status === 'RUNNING'" size="sm" variant="outline" @click="operateSupervisor(p.name, 'stop')">{{ $t('common.stop') }}</FaButton>
                <FaButton v-if="p.name !== 'php-fpm'" size="sm" variant="outline" class="text-red-500!" @click="removeSupervisor(p.name)">{{ $t('common.delete') }}</FaButton>
              </div>
            </div>
            <span v-if="!supList.length && !supLoading" class="text-xs text-muted-foreground">{{ $t('runtimes.noProcs') }}</span>
          </div>
          <FaDivider>{{ $t('runtimes.addProc') }}</FaDivider>
          <div class="flex items-end gap-2">
            <div class="w-40">
              <div class="mb-1 text-xs text-muted-foreground">{{ $t('runtimes.procName') }}</div>
              <FaInput v-model="supForm.name" :placeholder="$t('runtimes.procNamePlaceholder')" class="w-full" />
            </div>
            <div class="flex-1">
              <div class="mb-1 text-xs text-muted-foreground">{{ $t('runtimes.procCmd') }}</div>
              <FaInput v-model="supForm.command" placeholder="php /var/www/artisan queue:work" class="w-full" />
            </div>
            <FaButton size="sm" :loading="supSaving" :disabled="current.status !== 'running'" @click="saveSupervisor">{{ $t('runtimes.saveEffect') }}</FaButton>
          </div>
          <div class="flex gap-4 text-xs text-muted-foreground">
            <label class="flex items-center gap-1"><FaSwitch v-model="supForm.autoStart" /> {{ $t('runtimes.autoStart') }}</label>
            <label class="flex items-center gap-1"><FaSwitch v-model="supForm.autoRestart" /> {{ $t('runtimes.autoRestart') }}</label>
          </div>
          <div v-if="supLogName" class="rounded-md border p-2">
            <div class="mb-1 flex items-center justify-between">
              <span class="text-xs text-muted-foreground">{{ $t('runtimes.supLog', { name: supLogName }) }}</span>
              <FaButton size="sm" variant="link" @click="showSupervisorLog(supLogName)">{{ $t('common.refresh') }}</FaButton>
            </div>
            <YdLogViewer :logs="supLog" height="200px" />
          </div>
        </div>

        <!-- 慢日志（仅 PHP）-->
        <div v-if="drawerTab === 'slowlog'" class="mt-3 flex flex-col gap-2">
          <div class="flex gap-2">
            <FaButton size="sm" variant="outline" :loading="slowLogLoading" @click="loadSlowLog">
              <FaIcon name="i-lucide:snail" class="mr-1" /> {{ $t('runtimes.slowLogRefresh') }}
            </FaButton>
            <FaButton size="sm" variant="outline" class="text-red-500!" @click="clearSlowLog">{{ $t('common.clear') }}</FaButton>
          </div>
          <div class="text-xs text-muted-foreground">{{ $t('runtimes.slowLogHint') }}</div>
          <YdLogViewer :logs="slowLogText || $t('runtimes.noSlowLog')" height="380px" />
        </div>

        <!-- 备份恢复 -->
        <div v-if="drawerTab === 'backups'" class="mt-3 flex flex-col gap-3">
          <div class="flex items-center justify-between">
            <FaButton size="sm" :disabled="current.origin === 'external'" @click="createBackup">{{ $t('runtimes.backupCreateTitle') }}</FaButton>
            <FaButton size="sm" variant="outline" :loading="backupsLoading" @click="loadBackups">
              <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('common.refresh') }}
            </FaButton>
          </div>
          <div class="text-xs text-muted-foreground">{{ $t('runtimes.backupsHint') }}</div>
          <div v-if="backups.length" class="flex flex-col gap-1">
            <div v-for="b in backups" :key="b.file" class="flex items-center justify-between rounded-md border px-3 py-2">
              <div>
                <span class="font-mono text-xs">{{ b.file }}</span>
                <span class="ml-2 text-xs text-muted-foreground">{{ b.size }} · {{ b.at }}</span>
              </div>
              <div class="flex gap-1">
                <FaButton size="sm" variant="outline" @click="restoreBackup(b.file)">{{ $t('runtimes.restore') }}</FaButton>
                <FaButton size="sm" variant="outline" class="text-red-500!" @click="removeBackup(b.file)">{{ $t('common.delete') }}</FaButton>
              </div>
            </div>
          </div>
          <div v-else class="text-xs text-muted-foreground">{{ $t('runtimes.noBackups') }}</div>
        </div>

        <!-- 日志（代码运行时）-->
        <div v-if="drawerTab === 'logs'" class="mt-3 flex flex-col gap-2">
          <div>
            <FaButton size="sm" variant="outline" :loading="logsLoading" @click="loadLogs">
              <FaIcon name="i-lucide:scroll-text" class="mr-1" /> {{ $t('runtimes.logRefresh') }}
            </FaButton>
          </div>
          <YdLogViewer :logs="codeLogs" height="420px" />
        </div>

        <!-- 依赖（Node）-->
        <div v-if="drawerTab === 'modules'" class="mt-3 flex flex-col gap-3">
          <div class="flex items-end gap-2">
            <div class="flex-1">
              <div class="mb-1 text-xs text-muted-foreground">{{ $t('runtimes.moduleName') }}</div>
              <FaInput v-model="nodeInstallName" :placeholder="$t('runtimes.modulePlaceholder')" class="w-full" />
            </div>
            <FaButton size="sm" :disabled="nodeModuleBusy" @click="operateNodeModule('install', nodeInstallName)">{{ $t('runtimes.install') }}</FaButton>
            <FaButton size="sm" variant="outline" :loading="nodeModulesLoading" @click="current && loadNodeModules(current)">
              <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('common.refresh') }}
            </FaButton>
          </div>
          <div class="text-xs text-muted-foreground">{{ $t('runtimes.pkgMgrLabel', { mgr: nodePkgMgr }) }}</div>
          <div v-if="!nodeHasPkg" class="text-xs text-muted-foreground">
            {{ $t('runtimes.noPackageJson') }}
          </div>
          <div v-else class="flex flex-col gap-1">
            <div v-for="m in nodeModules" :key="m.name" class="flex items-center justify-between rounded-md border px-3 py-2">
              <div>
                <span class="font-mono text-sm">{{ m.name }}</span>
                <span class="ml-2 font-mono text-xs text-muted-foreground">{{ m.version }}</span>
                <span v-if="m.dev" class="ml-1 rounded bg-muted px-1 py-0.5 text-xs text-muted-foreground">dev</span>
              </div>
              <div class="flex gap-1">
                <FaButton size="sm" variant="outline" :disabled="nodeModuleBusy" @click="operateNodeModule('update', m.name)">{{ $t('runtimes.update') }}</FaButton>
                <FaButton size="sm" variant="outline" class="text-red-500!" :disabled="nodeModuleBusy" @click="operateNodeModule('uninstall', m.name)">
                  {{ $t('runtimes.uninstall') }}
                </FaButton>
              </div>
            </div>
          </div>
        </div>
      </template>
    </FaDrawer>

    <!-- 目录选择器 -->
    <YdDirPicker v-model:visible="dirPickerVisible" initial-path="/opt" :title="$t('runtimes.pickCodeDir')" @select="onDirPicked" />

    <!-- 接管本机 PHP -->
    <FaModal v-model="extVisible" :title="$t('runtimes.attach')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.name') }}</span>
          <FaInput v-model="extForm.name" :placeholder="$t('runtimes.attachNamePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('runtimes.version') }}</span>
          <FaSelect
            v-model="extForm.version" :options="[
              { label: 'PHP 8.2', value: '8.2' },
              { label: 'PHP 8.3', value: '8.3' },
              { label: 'PHP 8.1', value: '8.1' },
              { label: 'PHP 7.4', value: '7.4' },
              { label: $t('common.unknown'), value: 'unknown' },
            ]" class="flex-1"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">FastCGI</span>
          <FaInput v-model="extForm.fcgiAddr" :placeholder="$t('runtimes.fcgiAddrPlaceholder')" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          {{ $t('runtimes.attachHint') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="extVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="extSaving" @click="doAttach">{{ $t('runtimes.attachAction') }}</FaButton>
      </template>
    </FaModal>

    <!-- 创建运行环境 -->
    <FaModal v-model="createVisible" :title="$t('runtimes.create')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.type') }}</span>
          <FaSelect
            v-model="createForm.type" class="flex-1" :options="[
              { label: $t('runtimes.typePhp'), value: 'php' },
              { label: $t('runtimes.typeNode'), value: 'node' },
              { label: $t('runtimes.typePython'), value: 'python' },
              { label: $t('runtimes.typeJava'), value: 'java' },
              { label: $t('runtimes.typeGo'), value: 'go' },
            ]" @change="switchCreateType"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.name') }}</span>
          <FaInput v-model="createForm.name" :placeholder="$t('runtimes.namePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('runtimes.version') }}</span>
          <FaSelect v-model="createForm.version" :options="createVersionOptions" class="flex-1" />
        </div>

        <!-- PHP：扩展 -->
        <div v-if="createForm.type === 'php'">
          <div class="mb-1.5 text-sm text-muted-foreground">{{ $t('runtimes.extBuildHint') }}</div>
          <div v-if="catalog.templates.length" class="mb-2 flex flex-wrap gap-1.5">
            <FaTooltip v-for="t in catalog.templates" :key="t.name" :text="`${t.desc}：${t.extensions.join(', ')}`">
              <button
                type="button"
                class="rounded-md border border-dashed px-2 py-1 text-xs text-muted-foreground transition-colors hover:border-primary hover:text-primary"
                @click="applyTemplate(t.extensions)"
              >
                <FaIcon name="i-lucide:package" class="mr-1" />{{ t.name }}
              </button>
            </FaTooltip>
          </div>
          <div class="flex flex-wrap gap-1.5">
            <button
              v-for="e in catalog.catalog" :key="e.name" type="button"
              class="rounded-md border px-2 py-1 text-xs transition-colors"
              :class="createForm.extensions.includes(e.name)
                ? 'border-primary bg-primary/10 text-primary'
                : 'bg-background text-muted-foreground hover:border-primary/50'"
              :title="e.desc"
              @click="toggleExt(e.name)"
            >
              {{ e.name }}
            </button>
            <span v-if="!catalog.catalog.length" class="text-xs text-muted-foreground">{{ $t('runtimes.catalogLoading') }}</span>
          </div>
        </div>

        <!-- 代码运行时：目录/命令/端口 -->
        <template v-else>
          <div class="flex items-center gap-2">
            <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('runtimes.codeDir') }}</span>
            <FaInput v-model="createForm.codeDir" :placeholder="$t('runtimes.codeDirPlaceholder')" class="min-w-0 flex-1" />
            <FaButton size="sm" variant="outline" @click="dirPickerVisible = true">
              <FaIcon name="i-lucide:folder-open" class="mr-1" /> {{ $t('runtimes.browse') }}
            </FaButton>
          </div>
          <div class="flex items-center gap-3">
            <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('runtimes.startCmd') }}</span>
            <FaInput
              v-model="createForm.startCmd" class="flex-1"
              :placeholder="createForm.type === 'node' ? 'npm start / node dist/main.js' : (createForm.type === 'python' ? 'python main.py' : (createForm.type === 'java' ? 'java -jar app.jar' : 'go run .'))"
            />
          </div>
          <div class="flex items-center gap-3">
            <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('runtimes.autoInstall') }}</span>
            <FaSwitch v-model="createForm.autoInstall" />
            <span class="text-xs text-muted-foreground">{{ $t('runtimes.autoInstallHint') }}</span>
          </div>
          <div v-if="createForm.type === 'node'" class="flex items-center gap-3">
            <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('runtimes.pkgMgr') }}</span>
            <FaSelect v-model="createForm.pkgMgr" :options="pkgMgrOptions" class="flex-1" />
          </div>
          <div>
            <div class="mb-1.5 flex items-center justify-between">
              <span class="text-sm text-muted-foreground">{{ $t('runtimes.portsCol') }}</span>
              <FaButton size="sm" variant="outline" @click="addPortRow">
                <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('common.add') }}
              </FaButton>
            </div>
            <div class="flex flex-col gap-2">
              <div v-for="(p, i) in createForm.ports" :key="i" class="flex items-center gap-2">
                <FaInput v-model="p.host" :placeholder="$t('runtimes.hostPortPlaceholder')" class="flex-1" />
                <span class="text-xs text-muted-foreground">→</span>
                <FaInput v-model="p.container" :placeholder="$t('runtimes.containerPortPlaceholder')" class="flex-1" />
                <FaButton size="sm" variant="outline" class="text-red-500!" @click="removePortRow(i)">
                  <FaIcon name="i-lucide:x" />
                </FaButton>
              </div>
              <span v-if="!createForm.ports.length" class="text-xs text-muted-foreground">
                {{ $t('runtimes.portsHint') }}
              </span>
            </div>
          </div>
        </template>

        <div class="text-xs text-muted-foreground">
          <template v-if="createForm.type === 'php'">
            {{ $t('runtimes.phpCreateHint') }}
          </template>
          <template v-else>
            {{ $t('runtimes.codeCreateHint') }}
          </template>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="creating" @click="doCreate">{{ createForm.type === 'php' ? $t('runtimes.createAndBuild') : $t('runtimes.createAndStart') }}</FaButton>
      </template>
    </FaModal>
  </div>
</template>
