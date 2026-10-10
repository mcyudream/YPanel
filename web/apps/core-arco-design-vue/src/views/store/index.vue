<script setup lang="ts">
import type { StoreAppItem, StoreFormField, StoreListQuery, StoreSource, StoreTag, StoreVersion } from '@/api/modules/store'
import type { AppTask } from '@/api/modules/task'
import { marked } from 'marked'
import apiCompose from '@/api/modules/compose'
import dbApi, { type DbInstance } from '@/api/modules/database'
import apiSystem from '@/api/modules/system'
import { dockerExtApi } from '@/api/modules/dockerext'
import api from '@/api'
import { appIconSrc, isPortField, storeApi, type StoreInstallInfo } from '@/api/modules/store'
import { taskApi } from '@/api/modules/task'
import { useTaskCenterStore } from '@/store/modules/taskCenter'
import { i18n, tr } from '@/locales'
import YdLogViewer from '@/components/YdLogViewer/index.vue'
import YdDangerDelete from '@/components/YdDangerDelete/index.vue'
import YdOwnerDialog from '@/components/YdOwnerDialog/index.vue'

defineOptions({
  name: 'StoreIndex',
})

const taskCenter = useTaskCenterStore()

type TabKey = 'all' | 'installed' | 'notInstalled' | 'upgradable' | 'sources'

const TABS: { key: TabKey, label: string }[] = [
  { key: 'all', label: i18n.global.t('common.all') },
  { key: 'installed', label: i18n.global.t('store.tabInstalled') },
  { key: 'notInstalled', label: i18n.global.t('store.tabNotInstalled') },
  { key: 'upgradable', label: i18n.global.t('store.tabUpgradable') },
  { key: 'sources', label: i18n.global.t('store.tabSources') },
]

function kindLabel(kind: string) {
  return tr(`store.kind.${kind}`, kind)
}
function typeLabel(type: string) {
  return tr(`store.type.${type}`, type)
}

const activeTab = ref<TabKey>('all')
const search = ref('')
const activeTag = ref('')
const sourceFilter = ref<number | 0>(0)
const orderBy = ref<'name' | 'lastModified' | ''>('')
const page = ref(1)
const size = ref(20)
const total = ref(0)
const items = ref<StoreAppItem[]>([])
const tags = ref<StoreTag[]>([])
const sources = ref<StoreSource[]>([])
const loading = ref(false)
const syncing = ref(false)
const upgradableCount = ref(0)

async function load() {
  if (activeTab.value === 'sources') {
    await loadSources()
    return
  }
  loading.value = true
  try {
    const q: StoreListQuery = {
      search: search.value,
      tag: activeTag.value,
      sourceId: sourceFilter.value || undefined,
      status: activeTab.value,
      orderBy: orderBy.value || undefined,
      order: orderBy.value === 'name' ? 'asc' : 'desc',
      page: page.value,
      pageSize: size.value,
    }
    const out = await storeApi.list(q)
    items.value = out.items
    total.value = out.total
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.listLoadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
  if (activeTab.value === 'installed') {
    await loadInstalled()
  }
  refreshUpgradableCount()
}

// 角标数 = 全量可升级数（借分页接口 total，pageSize=1 最轻）
async function refreshUpgradableCount() {
  try {
    const out = await storeApi.list({ status: 'upgradable', page: 1, pageSize: 1 })
    upgradableCount.value = out.total
  }
  catch {
    // 角标失败不打扰主流程
  }
}

async function loadTags() {
  try {
    tags.value = await storeApi.tags()
  }
  catch {}
}

async function loadSources() {
  try {
    sources.value = await storeApi.sources()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.sourceListLoadFailed'), { description: e?.message })
  }
}

async function syncAll(force = false) {
  syncing.value = true
  try {
    const out = await storeApi.sync(force)
    const parts = Object.entries(out || {}).map(([k, v]) => `${k}: ${v}`)
    useFaToast().success(i18n.global.t('store.syncDone', { n: parts.length }), { description: parts.join('，') })
    await Promise.all([load(), loadTags()])
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.syncFailed'), { description: e?.message })
  }
  finally {
    syncing.value = false
  }
}

watch([search, activeTag, sourceFilter, orderBy], () => {
  page.value = 1
  load()
})
watch(activeTab, () => {
  page.value = 1
  load()
})
watch([page, size], () => load())

onMounted(() => {
  load()
  loadTags()
  loadSources()
})

// ---------- 详情抽屉 ----------
const detailVisible = ref(false)
const detailLoading = ref(false)
const detailApp = ref<StoreAppItem | null>(null)
const detailVersions = ref<StoreVersion[]>([])
const detailReadme = ref('')
const detailInstall = ref<{ version: string, composeProject: string } | null>(null)

function sourceName(id: number) {
  return sources.value.find(s => s.id === id)?.name || i18n.global.t('store.sourceN', { n: id })
}

async function openDetail(item: StoreAppItem) {
  detailApp.value = item
  detailVersions.value = []
  detailReadme.value = ''
  detailInstall.value = null
  detailVisible.value = true
  detailLoading.value = true
  try {
    const out = await storeApi.get(item.sourceId, item.key)
    detailVersions.value = out.versions
    detailInstall.value = out.installed ? { version: out.install.version, composeProject: out.install.composeProject } : null
    detailReadme.value = out.app.readMe
      ? (marked.parse(out.app.readMe, { async: false, breaks: true }) as string)
      : ''
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.detailLoadFailed'), { description: e?.message })
  }
  finally {
    detailLoading.value = false
  }
}

// ---------- 安装向导 ----------
const installVisible = ref(false)
const installing = ref(false)
const installTarget = ref<StoreAppItem | null>(null)
const installForm = ref({ version: '', name: '', domain: '', params: {} as Record<string, string> })
const installFields = ref<StoreFormField[]>([])
const installProxyEnv = ref('')
const installVersions = ref<StoreVersion[]>([])
// 高级选项：网络 / 时区 / hosts / 目标节点（M55）
const installAdvanced = ref(false)
const installNode = ref('')
const onlineNodes = ref<{ id: string, name: string, arch?: string }[]>([])
const dockerNets = ref<{ name: string }[]>([])
const netSel = ref('ypanel_default')
const netNew = ref('')
const installTZ = ref(true)
const installTZValue = ref('Asia/Shanghai')
const installMountHosts = ref(false)
// M32：外接数据库实例（识别到 *_HOST 数据库参数时提供「使用已有实例」选择）
const dbSource = ref<'default' | 'external'>('default')
const installPrefilled = ref(false)
const dbInstances = ref<DbInstance[]>([])
const dbTarget = ref<{ instanceId: number, database: string, user: string, createIfMissing: boolean }>({ instanceId: 0, database: '', user: '', createIfMissing: true })

const dbHostKey = computed(() => {
  for (const f of installFields.value) {
    const k = (f.envKey || '').toUpperCase()
    const stem = k.replace(/^PANEL_/, '').replace(/_HOST$/, '')
    if (k.endsWith('_HOST') && /(DB|MYSQL|SQL|MONGO|DATABASE|MARIA)/.test(stem)) {
      return f.envKey
    }
  }
  return ''
})

// 数据库纳管模式下的字段前缀（如 DATABASE），这些原始字段由选择器接管注入
const dbFieldPrefix = computed(() => (dbHostKey.value ? dbHostKey.value.replace(/_HOST$/, '') : ''))

function isDbManagedField(envKey?: string) {
  if (!dbFieldPrefix.value || dbSource.value !== 'external') {
    return false
  }
  const k = (envKey || '').toUpperCase()
  return k === dbFieldPrefix.value || k.startsWith(dbFieldPrefix.value + '_')
}

async function loadDBInstances() {
  if (dbInstances.value.length) {
    return
  }
  try {
    dbInstances.value = ((await dbApi.list()) || []).filter(i => i.type === 'mysql' || i.type === 'postgres')
  }
  catch {}
}
const installHosts = ref<{ host: string, ip: string }[]>([])
// 导入宿主机 /etc/hosts
const hostImportVisible = ref(false)
const hostImportLoading = ref(false)
const hostEntries = ref<{ ip: string, hosts: string[] }[]>([])
const hostPicked = ref<Record<string, boolean>>({})

async function openHostImport() {
  hostImportVisible.value = true
  hostImportLoading.value = true
  hostPicked.value = {}
  try {
    hostEntries.value = await apiSystem.hostEntries()
    for (const e of hostEntries.value) {
      for (const h of e.hosts) {
        hostPicked.value[`${e.ip}|${h}`] = installHosts.value.some(x => x.host === h && x.ip === e.ip)
      }
    }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.hostsReadFailed'), { description: e?.message })
  }
  finally {
    hostImportLoading.value = false
  }
}

function confirmHostImport() {
  for (const e of hostEntries.value) {
    for (const h of e.hosts) {
      const k = `${e.ip}|${h}`
      if (hostPicked.value[k] && !installHosts.value.some(x => x.host === h && x.ip === e.ip)) {
        installHosts.value.push({ host: h, ip: e.ip })
      }
    }
  }
  hostImportVisible.value = false
}
// 任务化安装：提交后切换为日志视图
const installTaskId = ref(0)
const installTask = ref<AppTask | null>(null)
const installLogLoading = ref(false)
let installPollTimer: ReturnType<typeof setInterval> | null = null

function fieldLabel(f: { label: Record<string, string>, envKey?: string }) {
  return f.label?.zh || f.label?.en || f.envKey || ''
}

function applyInstallVersion(versions: StoreVersion[], versionId: string, prev?: StoreInstallInfo) {
  const target = versions.find(v => v.id === versionId) || versions[0]
  if (!target) {
    return
  }
  installForm.value.version = target.id
  installFields.value = (target.formFields || []).filter(f => f.envKey)
  dbSource.value = 'default'
  dbTarget.value = { instanceId: 0, database: '', user: '', createIfMissing: true }
  const params: Record<string, string> = {}
  for (const f of installFields.value) {
    params[f.envKey] = fieldDefault(f)
  }
  // P3 重装沿用参数：同应用已装实例的上次参数覆盖默认值（字段仍可编辑）
  installPrefilled.value = !!prev
  if (prev && prev.name) {
    installForm.value.name = prev.name
  }
  if (prev) {
    for (const [k, v] of Object.entries(prev.params || {})) {
      if (k in params && v) {
        params[k] = v
      }
    }
  }
  installForm.value.params = params
  // php 应用声明 database 时默认走纳管实例（自动建库注入）；compose 应用保持「应用自带」默认
  if (installTarget.value?.kind === 'php' && dbHostKey.value) {
    dbSource.value = 'external'
    void loadDBInstances()
  }
}

function stopInstallPolling() {
  if (installPollTimer) {
    clearInterval(installPollTimer)
    installPollTimer = null
  }
}

async function refreshInstallTask() {
  if (!installTaskId.value) {
    return
  }
  installLogLoading.value = true
  try {
    installTask.value = await taskApi.get(installTaskId.value)
    if (installTask.value.status !== 'running') {
      stopInstallPolling()
      if (installTask.value.status === 'success') {
        useFaToast().success(i18n.global.t('store.installDone', { name: installTask.value.ref }))
      }
      else {
        useFaToast().error(i18n.global.t('store.installFailed'), { description: installTask.value.error })
      }
      await Promise.all([load(), loadTags()])
    }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.taskStatusFailed'), { description: e?.message })
    stopInstallPolling()
  }
  finally {
    installLogLoading.value = false
  }
}

async function loadNetworks() {
  try {
    dockerNets.value = (await dockerExtApi.networks()).map(n => ({ name: n.name }))
  api.get('api/v1/nodes', { silent: true }).then((r) => {
    onlineNodes.value = (r.data as any[]).filter((x: any) => x.id === 'local' || x.online)
  }).catch(() => {})
  }
  catch {}
}

async function openInstall(item: StoreAppItem, versionId = '') {
  installTarget.value = item
  installProxyEnv.value = item.reverseProxy
  netSel.value = 'ypanel_default'
  netNew.value = ''
  installTZ.value = true
  installTZValue.value = 'Asia/Shanghai'
  installMountHosts.value = false
  installHosts.value = []
  void loadNetworks()
  installForm.value = { version: versionId || item.latestVersion || '', name: item.key, domain: '', params: {} }
  installTaskId.value = 0
  installTask.value = null
  // P3 重装沿用参数：确保已装列表就绪，取同应用已装实例（默认 tab 下 installedInfos 为空需补拉）
  if (!installedInfos.value.length) {
    try {
      await loadInstalled()
    }
    catch {}
  }
  const prevInstall = installedInfos.value.find(i => i.sourceId === item.sourceId && i.key === item.key)
  // 优先复用详情抽屉已加载的版本；否则异步拉取
  if (detailApp.value?.sourceId === item.sourceId && detailApp.value?.key === item.key && detailVersions.value.length) {
    installVersions.value = detailVersions.value
    applyInstallVersion(detailVersions.value, installForm.value.version, prevInstall)
  }
  else {
    installVersions.value = []
    installFields.value = []
    storeApi.get(item.sourceId, item.key).then((out) => {
      installVersions.value = out.versions
      applyInstallVersion(out.versions, installForm.value.version, prevInstall)
    }).catch(() => {})
  }
  installVisible.value = true
}

async function doInstall() {
  const target = installTarget.value
  if (!target) {
    return
  }
  installing.value = true
  try {
    const out = await storeApi.install({
      sourceId: target.sourceId,
      key: target.key,
      version: installForm.value.version,
      name: installForm.value.name,
      // 统一字符串化（number 输入框可能产出 number）
      params: Object.fromEntries(Object.entries(installForm.value.params).map(([k, v]) => [k, String(v ?? '')])),
      domain: installForm.value.domain || undefined,
        nodeId: installNode.value || '',
      network: netSel.value === '__create__' ? netNew.value : netSel.value,
      createNetwork: netSel.value === '__create__',
      timezone: installTZ.value ? installTZValue.value : '',
      extraHosts: installHosts.value.filter(h => h.host && h.ip).map(h => `${h.host}:${h.ip}`),
      mountHostsFile: installMountHosts.value,
      externalDB: dbSource.value === 'external' && dbTarget.value.instanceId
        ? {
            instanceId: dbTarget.value.instanceId,
            database: dbTarget.value.database || undefined,
            user: dbTarget.value.user || undefined,
            createIfMissing: dbTarget.value.createIfMissing,
          }
        : undefined,
    })
    // 任务化：切换到日志视图并轮询
    installTaskId.value = out.taskId
    installTask.value = null
    useFaToast().success(i18n.global.t('store.installTaskCreated'))
    void refreshInstallTask()
    stopInstallPolling()
    installPollTimer = setInterval(() => {
      if (!installTask.value || installTask.value.status === 'running') {
        void refreshInstallTask()
      }
      else {
        stopInstallPolling()
      }
    }, 2000)
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.installTaskCreateFailed'), { description: e?.message })
  }
  finally {
    installing.value = false
  }
}

watch(installVisible, (v) => {
  if (!v) {
    stopInstallPolling()
  }
})

const uninstallVisible = ref(false)
const uninstallTarget = ref('')
const uninstalling = ref(false)
const UNINSTALL_OPTS = computed(() => [
  { key: 'purgeData', label: i18n.global.t('store.optPurgeData'), desc: i18n.global.t('store.optPurgeDataDesc') },
  { key: 'rmi', label: i18n.global.t('store.optRmi'), desc: i18n.global.t('store.optRmiDesc') },
  { key: 'cascadeDB', label: i18n.global.t('store.optCascadeDB'), desc: i18n.global.t('store.optCascadeDBDesc') },
])

function uninstall(p: string, info?: StoreInstallInfo) {
  uninstallTarget.value = p
  uninstallSource.value = info ?? null
  uninstallVisible.value = true
}

async function doUninstall(checked: Record<string, boolean>) {
  uninstalling.value = true
  try {
    await storeApi.uninstall(uninstallTarget.value, {
      nodeId: uninstallSource.value?.nodeId,
      purgeData: !!checked.purgeData,
      rmi: !!checked.rmi,
      cascadeDB: !!checked.cascadeDB,
    })
    useFaToast().success(i18n.global.t('store.uninstallTaskCreated'))
    uninstallVisible.value = false
    await Promise.all([load(), loadTags()])
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.uninstallFailed'), { description: e?.message })
  }
  finally {
    uninstalling.value = false
  }
}

async function upgrade(item: StoreAppItem) {
  openInstall(item, item.latestVer)
}

// ---------- M54-P3 属主分配 ----------
const ownerVisible = ref(false)
const ownerTarget = ref<StoreInstallInfo | null>(null)

function openOwner(info: StoreInstallInfo) {
  ownerTarget.value = info
  ownerVisible.value = true
}

async function doSetOwner(uid: number) {
  if (!ownerTarget.value) {
    return
  }
  try {
    await storeApi.setOwner(ownerTarget.value.composeProject, uid)
    useFaToast().success(i18n.global.t('common.owner.saved'))
    await loadInstalled()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('common.owner.saveFailed'), { description: e?.message })
  }
}

// ---------- 已安装 Tab（1Panel 风格卡片：状态/启停/重启/重建/参数/日志/外链） ----------
const installedInfos = ref<StoreInstallInfo[]>([])
// M55 节点筛选：全部/本机/各节点（前端过滤，installedDetailed 本就跨节点聚合）
const installedNodeFilter = ref('all')
const storeNodes = ref<{ id: string, name: string, online: boolean }[]>([])
const installedNodeOptions = computed(() => {
  const opts = [{ label: i18n.global.t('store.nodeAll'), value: 'all' }, { label: i18n.global.t('nodes.localPanel'), value: 'local' }]
  for (const n of storeNodes.value) {
    if (n.id !== 'local' && !opts.some(o => o.value === n.id)) {
      opts.push({ label: n.name, value: n.id })
    }
  }
  return opts
})
const installedNodeFilterLabel = computed(() =>
  installedNodeOptions.value.find(o => o.value === installedNodeFilter.value)?.label || i18n.global.t('store.nodeAll'),
)
const installedNodeMenu = computed(() => [installedNodeOptions.value.map(o => ({
  label: o.label,
  handle: () => { installedNodeFilter.value = o.value },
}))])
onMounted(() => {
  api.get('api/v1/nodes', { silent: true }).then((r) => {
    storeNodes.value = (r.data as any[]).map((x: any) => ({ id: x.id, name: x.name, online: x.online }))
  }).catch(() => {})
})
const installedFiltered = computed(() => {
  if (installedNodeFilter.value === 'all') {
    return installedInfos.value
  }
  return installedInfos.value.filter(i => (i.nodeId || 'local') === installedNodeFilter.value)
})
const uninstallSource = ref<StoreInstallInfo | null>(null)
const installedLoading = ref(false)
const actingOn = ref('')

async function loadInstalled() {
  installedLoading.value = true
  try {
    installedInfos.value = await storeApi.installed()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.installedListLoadFailed'), { description: e?.message })
  }
  finally {
    installedLoading.value = false
  }
}

async function installedAction(info: StoreInstallInfo, action: 'start' | 'stop' | 'restart' | 'rebuild') {
  actingOn.value = `${action}-${info.composeProject}`
  try {
    await storeApi.installedAction(info.composeProject, action, info.nodeId)
    const actionLabel = action === 'rebuild' ? i18n.global.t('store.rebuild') : i18n.global.t(`common.${action}`)
    useFaToast().success(i18n.global.t('store.actionDone', { name: info.composeProject, action: actionLabel }))
    await loadInstalled()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.opFailed'), { description: e?.message })
  }
  finally {
    actingOn.value = ''
  }
}

// 参数（.env 编辑，保存重建生效）
const paramsVisible = ref(false)
const paramsProject = ref('')
const paramsSaving = ref(false)
const paramsText = ref('')

async function openParams(info: StoreInstallInfo) {
  paramsProject.value = info.composeProject
  paramsVisible.value = true
  paramsText.value = i18n.global.t('common.loading')
  try {
    const env = await storeApi.installEnv(info.composeProject)
    paramsText.value = Object.entries(env).map(([k, v]) => `${k}=${v}`).join('\n')
  }
  catch (e: any) {
    paramsText.value = ''
    useFaToast().error(i18n.global.t('store.paramsLoadFailed'), { description: e?.message })
  }
}

async function saveParams() {
  paramsSaving.value = true
  try {
    await storeApi.saveInstallEnv(paramsProject.value, paramsText.value)
    useFaToast().success(i18n.global.t('store.paramsSaved'))
    paramsVisible.value = false
    await loadInstalled()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.saveFailed'), { description: e?.message })
  }
  finally {
    paramsSaving.value = false
  }
}

// 日志（compose logs tail）
const logsVisible = ref(false)
const logsProject = ref('')
const logsText = ref('')
const logsLoading = ref(false)

async function openLogs(info: StoreInstallInfo) {
  logsProject.value = info.composeProject
  logsVisible.value = true
  await refreshLogs()
}

async function refreshLogs() {
  const token = useAppAccountStore().token
  if (!token) {
    return
  }
  logsLoading.value = true
  try {
    const url = apiCompose.logsURL(logsProject.value, undefined, token, 500)
    const res = await fetch(url)
    logsText.value = await res.text()
  }
  catch (e: any) {
    logsText.value = i18n.global.t('store.logFetchFailed', { msg: e?.message || e })
  }
  finally {
    logsLoading.value = false
  }
}

function openExternal(info: StoreInstallInfo) {
  if (!info.ports.length) {
    return
  }
  window.open(`http://${location.hostname}:${info.ports[0]}`, '_blank', 'noopener')
}

function fmtDuration(t: string) {
  const ms = Date.now() - new Date(t).getTime()
  const d = Math.floor(ms / 86400000)
  const h = Math.floor((ms % 86400000) / 3600000)
  return d > 0 ? i18n.global.t('store.durationDh', { d, h }) : i18n.global.t('store.durationH', { h })
}

// ---------- 安装表单控件辅助 ----------

const CHARS = 'abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789'

function randomPassword(len = 16) {
  let s = ''
  for (let i = 0; i < len; i++) {
    s += CHARS[Math.floor(Math.random() * CHARS.length)]
  }
  return s
}

function randomPort() {
  return String(32768 + Math.floor(Math.random() * 28231))
}

function fieldIsEditable(f: StoreFormField) {
  return f.edit !== false && !f.disabled && f.type !== 'service' && f.type !== 'apps'
}

// 按字段语义生成默认值（密码随机 / 端口随机高位，对齐 1Panel 行为）
function fieldDefault(f: StoreFormField) {
  // 后端 Go 序列化把无 default 的字段输出为 null（非 undefined），必须用 == null 同时兜住两种
  if (f.type === 'password' && f.random !== false && f.default == null) {
    return randomPassword(f.randomLen || 16)
  }
  if (isPortField(f) && f.type === 'number') {
    return randomPort()
  }
  if (f.default !== undefined && f.default !== null) {
    return String(f.default)
  }
  return ''
}

function randomizeField(f: StoreFormField) {
  if (f.type === 'password') {
    installForm.value.params[f.envKey] = randomPassword(f.randomLen || 16)
  }
  else if (isPortField(f)) {
    installForm.value.params[f.envKey] = randomPort()
  }
}

// ---------- 源管理 ----------
const sourceModalVisible = ref(false)
const sourceSaving = ref(false)
const sourceEditing = ref<StoreSource | null>(null)
const sourceForm = ref({ name: '', type: 'yp-git', url: '', branch: 'main', authToken: '', remark: '' })

function openSourceModal(src?: StoreSource) {
  sourceEditing.value = src || null
  sourceForm.value = src
    ? { name: src.name, type: src.type, url: src.url, branch: src.branch, authToken: '', remark: src.remark }
    : { name: '', type: 'yp-git', url: '', branch: 'main', authToken: '', remark: '' }
  sourceModalVisible.value = true
}

async function saveSource() {
  sourceSaving.value = true
  try {
    if (sourceEditing.value) {
      await storeApi.updateSource(sourceEditing.value.id, sourceForm.value)
      useFaToast().success(i18n.global.t('store.sourceUpdated'))
    }
    else {
      await storeApi.createSource(sourceForm.value)
      useFaToast().success(i18n.global.t('store.sourceAdded'))
    }
    sourceModalVisible.value = false
    await loadSources()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.saveFailed'), { description: e?.message })
  }
  finally {
    sourceSaving.value = false
  }
}

async function toggleSource(src: StoreSource) {
  try {
    await storeApi.setSourceEnabled(src.id, !src.enabled)
    await loadSources()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.opFailed'), { description: e?.message })
  }
}

function removeSource(src: StoreSource) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('store.deleteSourceTitle'),
    content: i18n.global.t('store.deleteSourceConfirm', { name: src.name }),
    onConfirm: async () => {
      try {
        await storeApi.deleteSource(src.id)
        useFaToast().success(i18n.global.t('store.sourceDeleted'))
        await Promise.all([loadSources(), loadTags()])
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('store.deleteFailed'), { description: e?.message })
      }
    },
  })
}

async function syncOne(src: StoreSource) {
  try {
    const out = await storeApi.syncSource(src.id, true)
    useFaToast().success(i18n.global.t('store.sourceSyncDone', { name: out.source, n: out.total }))
    await Promise.all([loadSources(), loadTags()])
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('store.syncFailed'), { description: e?.message })
    await loadSources()
  }
}

function statusText(s: StoreSource) {
  if (!s.url && s.type === 'yp-git') {
    return i18n.global.t('store.stUnset')
  }
  switch (s.status) {
    case 'ok': return i18n.global.t('store.stOk')
    case 'error': return i18n.global.t('store.stError', { msg: s.message })
    default: return i18n.global.t('store.stPending')
  }
}
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="package" :size="24" />
          <span>{{ $t('store.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('store.description') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" :loading="syncing" @click="syncAll(true)">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('store.syncAll') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 状态 Tab -->
      <div class="mb-3 flex flex-wrap items-center gap-1 border-b border-border pb-2">
        <button
          v-for="t in TABS"
          :key="t.key"
          type="button"
          class="cursor-pointer rounded-md px-3 py-1.5 text-sm transition-colors"
          :class="activeTab === t.key ? 'bg-primary/10 font-medium text-primary' : 'text-muted-foreground hover:bg-accent/50'"
          @click="activeTab = t.key"
        >
          {{ t.label }}
          <span
            v-if="t.key === 'upgradable' && upgradableCount > 0"
            class="ml-1 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] leading-none text-white"
          >
            {{ upgradableCount > 99 ? '99+' : upgradableCount }}
          </span>
        </button>
      </div>

      <!-- 已安装：1Panel 风格大卡片（状态/启停/重启/重建/参数/日志/外链/卸载） -->
      <template v-if="activeTab === 'installed'">
        <div class="mb-3 flex items-center gap-2">
          <FaIcon name="i-lucide:server" class="text-sm text-muted-foreground" />
          <FaDropdown :items="installedNodeMenu">
            <FaButton variant="outline" size="sm" class="h-8">
              <FaIcon name="i-lucide:server" class="mr-1 text-xs text-muted-foreground" />
              {{ installedNodeFilterLabel }}
              <FaIcon name="i-lucide:chevron-down" class="ml-1 text-xs text-muted-foreground" />
            </FaButton>
          </FaDropdown>
        </div>
        <div v-if="installedLoading && !installedInfos.length" class="grid gap-4 md:grid-cols-2">
          <div v-for="i in 4" :key="i" class="h-36 animate-pulse rounded-lg border bg-muted/30" />
        </div>
        <div v-else-if="!installedInfos.length" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
          {{ $t('store.noInstalled') }}
        </div>
        <div v-else class="grid gap-4 md:grid-cols-2">
          <div
            v-for="info in installedFiltered"
            :key="info.id"
            class="flex flex-col rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
          >
            <div class="flex items-start gap-3">
              <YdAppIcon
                :image="appIconSrc(info.iconUrl)"
                :name="info.name"
                :size="44"
                class="mt-0.5 rounded-lg"
              />
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-1.5">
                  <span class="truncate font-medium">{{ info.name }}</span>
                  <span v-if="info.nodeId && info.nodeId !== 'local'" class="rounded-full bg-sky-500/10 px-1.5 py-0.5 text-[10px] text-sky-600">@{{ info.nodeId }}</span>
                  <span
                    class="inline-block whitespace-nowrap rounded-full px-2 py-0.5 text-xs"
                    :class="info.running ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-500'"
                  >
                    {{ info.running ? $t('store.running') : $t('store.stopped') }}
                  </span>
                  <span v-if="info.upgradable" class="rounded bg-orange-500/15 px-1.5 py-0.5 text-xs text-orange-600">{{ $t('store.upgradableWithVersion', { v: info.latestVersion }) }}</span>
                </div>
                <div class="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                  <span class="rounded border px-1.5 py-0.5">{{ $t('store.versionN', { v: info.version }) }}</span>
                  <span v-for="p in info.ports" :key="p" class="rounded border px-1.5 py-0.5">{{ $t('store.portN', { v: p }) }}</span>
                </div>
                <div class="mt-1 text-xs text-muted-foreground">{{ $t('store.installedDur', { t: fmtDuration(info.createdAt) }) }} · {{ info.composeProject }}</div>
              </div>
              <div class="flex shrink-0 flex-col gap-1.5">
                <FaButton size="sm" variant="outline" :disabled="!info.ports.length" :title="$t('store.openServiceTitle')" @click="openExternal(info)">
                  <FaIcon name="i-lucide:external-link" class="text-xs" /> {{ $t('store.open') }}
                </FaButton>
              </div>
            </div>
            <div class="mt-3 flex flex-wrap items-center gap-1.5 border-t pt-2.5" @click.stop>
              <FaButton
                size="sm" variant="outline" :loading="actingOn === `restart-${info.composeProject}`"
                @click="installedAction(info, 'restart')"
              >
                {{ $t('common.restart') }}
              </FaButton>
              <FaButton
                v-if="info.running" size="sm" variant="outline" :loading="actingOn === `stop-${info.composeProject}`"
                @click="installedAction(info, 'stop')"
              >
                {{ $t('common.stop') }}
              </FaButton>
              <FaButton
                v-else size="sm" variant="outline" :loading="actingOn === `start-${info.composeProject}`"
                @click="installedAction(info, 'start')"
              >
                {{ $t('common.start') }}
              </FaButton>
              <FaButton
                size="sm" variant="outline" :loading="actingOn === `rebuild-${info.composeProject}`" :title="$t('store.rebuildTitle')"
                @click="installedAction(info, 'rebuild')"
              >
                {{ $t('store.rebuild') }}
              </FaButton>
              <FaButton size="sm" variant="outline" @click="openParams(info)">{{ $t('store.params') }}</FaButton>
              <FaButton size="sm" variant="outline" @click="openLogs(info)">{{ $t('store.logs') }}</FaButton>
              <FaButton size="sm" variant="outline" :title="$t('common.owner.title')" @click="openOwner(info)">{{ $t('common.owner.short') }}</FaButton>
              <FaButton size="sm" variant="outline" class="ml-auto text-red-500!" @click="uninstall(info.composeProject, info)">
                {{ $t('store.uninstall') }}
              </FaButton>
            </div>
          </div>
        </div>
      </template>

      <!-- 应用列表（全部 / 未安装 / 可升级） -->
      <template v-else-if="activeTab !== 'sources'">
        <div class="mb-4 flex flex-wrap items-center gap-2">
          <div class="flex flex-wrap gap-1.5">
            <button
              type="button"
              class="cursor-pointer rounded-md border px-2.5 py-1 text-sm transition-colors"
              :class="activeTag === '' ? 'border-primary bg-primary/10 text-primary' : 'border-border hover:bg-accent/50'"
              @click="activeTag = ''"
            >
              {{ $t('common.all') }}
            </button>
            <button
              v-for="t in tags"
              :key="t.name"
              type="button"
              class="cursor-pointer rounded-md border px-2.5 py-1 text-sm transition-colors"
              :class="activeTag === t.name ? 'border-primary bg-primary/10 text-primary' : 'border-border hover:bg-accent/50'"
              @click="activeTag = activeTag === t.name ? '' : t.name"
            >
              {{ t.name }} <span class="text-xs opacity-60">{{ t.count }}</span>
            </button>
          </div>
          <div class="ml-auto flex items-center gap-2">
            <YdSelect
              v-model="sourceFilter"
              :options="[{ value: 0, label: $t('store.allSources') }, ...sources.filter(x => x.enabled).map(s => ({ value: s.id, label: s.name }))]"
            />
            <YdSelect
              v-model="orderBy"
              :options="[{ value: '', label: $t('store.sortDefault') }, { value: 'name', label: $t('common.name') }, { value: 'lastModified', label: $t('store.sortRecent') }]"
            />
            <FaInput v-model="search" :placeholder="$t('store.searchPh')" class="w-52">
              <template #start>
                <FaIcon name="i-lucide:search" class="text-muted-foreground" />
              </template>
            </FaInput>
          </div>
        </div>

        <div v-if="loading && !items.length" class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div v-for="i in 8" :key="i" class="h-28 animate-pulse rounded-lg border bg-muted/30" />
        </div>
        <div v-else-if="!items.length" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
          {{ activeTab === 'upgradable' ? $t('store.allUpToDate') : $t('store.noMatch') }}
        </div>
        <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div
            v-for="a in items"
            :key="`${a.sourceId}-${a.key}`"
            class="flex cursor-pointer flex-col rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
            @click="openDetail(a)"
          >
            <div class="flex items-start gap-3">
              <YdAppIcon
                :image="appIconSrc(a.iconUrl) || storeApi.iconUrl(a.sourceId, a.key)"
                :name="a.name"
                :size="40"
                class="mt-0.5 rounded-lg"
              />
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1.5">
                  <span class="truncate font-medium">{{ a.name }}</span>
                  <span v-if="a.installed" class="shrink-0 rounded bg-green-500/15 px-1.5 py-0.5 text-xs text-green-600">{{ $t('store.installed') }}</span>
                  <span v-else-if="a.upgradable" class="shrink-0 rounded bg-orange-500/15 px-1.5 py-0.5 text-xs text-orange-600">{{ $t('store.upgradable') }}</span>
                </div>
                <div class="truncate text-xs text-muted-foreground" :title="a.title">{{ a.title }}</div>
              </div>
            </div>
            <div class="mt-2 line-clamp-2 min-h-10 flex-1 text-xs text-muted-foreground">{{ a.description }}</div>
            <div class="mt-3 border-t pt-2" @click.stop>
              <div class="flex flex-wrap gap-1">
                <span class="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">{{ kindLabel(a.kind) }}</span>
                <span v-for="t in (a.tags || '').split(',').filter(Boolean).slice(0, 2)" :key="t" class="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">{{ t }}</span>
                <span class="max-w-full truncate rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground" :title="sourceName(a.sourceId)">{{ sourceName(a.sourceId) }}</span>
              </div>
              <div class="mt-2 flex flex-wrap justify-end gap-1.5">
                <FaButton v-if="a.upgradable" size="sm" variant="outline" @click="upgrade(a)">{{ $t('store.upgradeV', { v: a.latestVer }) }}</FaButton>
                <FaButton v-if="a.installed" size="sm" variant="outline" @click="uninstall(a.installInfo?.composeProject || '')">{{ $t('store.uninstall') }}</FaButton>
                <FaButton v-auth="['store:write']" size="sm" :variant="a.installed ? 'outline' : 'default'" @click="openInstall(a)">
                  {{ a.installed ? $t('store.installAgain') : $t('store.install') }}
                </FaButton>
              </div>
            </div>
          </div>
        </div>

        <div v-if="total > 0" class="mt-4 flex justify-end">
          <FaPagination v-model:page="page" v-model:size="size" :total="total" @page-change="load" @size-change="load" />
        </div>
      </template>

      <!-- 源管理 -->
      <template v-else>
        <div class="mb-4 flex items-center justify-between">
          <div class="text-sm text-muted-foreground">
            {{ $t('store.sourcesDesc') }}
          </div>
          <FaButton v-auth="['store:write']" size="sm" @click="openSourceModal()">
            <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('store.addSource') }}
          </FaButton>
        </div>
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/40 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-2.5 font-medium">{{ $t('common.name') }}</th>
                <th class="px-4 py-2.5 font-medium">{{ $t('common.type') }}</th>
                <th class="px-4 py-2.5 font-medium">{{ $t('store.address') }}</th>
                <th class="px-4 py-2.5 font-medium">{{ $t('store.appCount') }}</th>
                <th class="px-4 py-2.5 font-medium">{{ $t('common.status') }}</th>
                <th class="px-4 py-2.5 font-medium">{{ $t('store.lastSync') }}</th>
                <th class="px-4 py-2.5 text-right font-medium">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in sources" :key="s.id" class="border-t border-border">
                <td class="px-4 py-2.5">
                  {{ s.name }}
                  <span v-if="s.builtin" class="ml-1 rounded bg-primary/10 px-1.5 py-0.5 text-xs text-primary">{{ $t('store.builtin') }}</span>
                  <span v-if="!s.enabled" class="ml-1 rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">{{ $t('store.sourceDisabled') }}</span>
                </td>
                <td class="px-4 py-2.5 text-muted-foreground">{{ typeLabel(s.type) }}{{ s.branch ? ` · ${s.branch}` : '' }}</td>
                <td class="max-w-60 truncate px-4 py-2.5 font-mono text-xs text-muted-foreground" :title="s.url">{{ s.url || '—' }}</td>
                <td class="px-4 py-2.5">{{ s.appCount }}</td>
                <td class="max-w-52 truncate px-4 py-2.5" :class="s.status === 'error' ? 'text-red-500' : 'text-muted-foreground'" :title="statusText(s)">
                  {{ statusText(s) }}
                </td>
                <td class="px-4 py-2.5 text-xs text-muted-foreground">{{ s.lastSyncAt ? new Date(s.lastSyncAt).toLocaleString() : '—' }}</td>
                <td class="px-4 py-2.5">
                  <div class="flex justify-end gap-1">
                    <FaButton size="sm" variant="ghost" @click="syncOne(s)">{{ $t('store.sync') }}</FaButton>
                    <FaButton size="sm" variant="ghost" @click="openSourceModal(s)">{{ $t('common.edit') }}</FaButton>
                    <FaButton size="sm" variant="ghost" @click="toggleSource(s)">{{ s.enabled ? $t('common.disabled') : $t('common.enabled') }}</FaButton>
                    <FaButton v-if="!s.builtin" size="sm" variant="ghost" class="text-red-500!" @click="removeSource(s)">{{ $t('common.delete') }}</FaButton>
                  </div>
                </td>
              </tr>
              <tr v-if="!sources.length">
                <td colspan="7" class="px-4 py-8 text-center text-muted-foreground">{{ $t('store.noSources') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </FaPageMain>

    <!-- 详情抽屉 -->
    <FaDrawer v-model="detailVisible" :title="$t('store.detailTitle', { name: detailApp?.name || '' })" class="max-w-2xl!">
      <div v-if="detailApp" class="flex flex-col gap-4">
        <div class="flex items-start gap-4">
          <YdAppIcon
            :image="appIconSrc(detailApp.iconUrl) || storeApi.iconUrl(detailApp.sourceId, detailApp.key)"
            :name="detailApp.name"
            :size="64"
            class="rounded-xl border bg-background p-1"
          />
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-lg font-medium">{{ detailApp.name }}</span>
              <span v-if="detailInstall" class="rounded bg-green-500/15 px-2 py-0.5 text-xs text-green-600">{{ $t('store.installedV', { v: detailInstall.version }) }}</span>
              <span v-if="detailApp.upgradable" class="rounded bg-orange-500/15 px-2 py-0.5 text-xs text-orange-600">{{ $t('store.upgradableToV', { v: detailApp.latestVer }) }}</span>
            </div>
            <div class="mt-1 text-sm text-muted-foreground">{{ detailApp.title || detailApp.description }}</div>
            <div class="mt-2 flex flex-wrap gap-1.5 text-xs text-muted-foreground">
              <span class="rounded bg-muted px-1.5 py-0.5">{{ $t('store.sourceIs', { name: sourceName(detailApp.sourceId) }) }}</span>
              <span class="rounded bg-muted px-1.5 py-0.5">{{ kindLabel(detailApp.kind) }}</span>
              <span v-if="detailApp.author" class="rounded bg-muted px-1.5 py-0.5">{{ $t('store.authorIs', { name: detailApp.author }) }}</span>
              <span v-if="detailApp.arch" class="rounded bg-muted px-1.5 py-0.5">{{ $t('store.archIs', { name: detailApp.arch }) }}</span>
            </div>
          </div>
        </div>

        <div v-if="detailInstall" class="flex items-center justify-between rounded-lg border border-green-500/30 bg-green-500/5 px-3 py-2 text-sm">
          <span>{{ $t('store.runningProject') }}<span class="font-mono">{{ detailInstall.composeProject }}</span></span>
          <div class="flex gap-2">
            <FaButton size="sm" variant="outline" @click="uninstall(detailInstall?.composeProject || '')">{{ $t('store.uninstall') }}</FaButton>
          </div>
        </div>

        <!-- 链接信息（对齐 1Panel：官网 / 开源社区 / 文档） -->
        <div v-if="detailApp.website || detailApp.sourceUrl || detailApp.document" class="grid grid-cols-3 overflow-hidden rounded-lg border text-sm">
          <a
            v-if="detailApp.website"
            :href="detailApp.website"
            target="_blank"
            rel="noopener noreferrer"
            class="border-r border-border px-3 py-2 transition-colors hover:bg-accent/40"
          >
            <div class="text-xs text-muted-foreground">{{ $t('store.website') }}</div>
            <div class="mt-0.5 flex items-center gap-1 truncate text-primary">
              {{ $t('store.link') }} <FaIcon name="i-lucide:external-link" class="size-3 shrink-0" />
            </div>
          </a>
          <a
            v-if="detailApp.sourceUrl"
            :href="detailApp.sourceUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="border-r border-border px-3 py-2 transition-colors hover:bg-accent/40"
          >
            <div class="text-xs text-muted-foreground">{{ $t('store.community') }}</div>
            <div class="mt-0.5 flex items-center gap-1 truncate text-primary">
              {{ $t('store.link') }} <FaIcon name="i-lucide:external-link" class="size-3 shrink-0" />
            </div>
          </a>
          <a
            v-if="detailApp.document"
            :href="detailApp.document"
            target="_blank"
            rel="noopener noreferrer"
            class="px-3 py-2 transition-colors hover:bg-accent/40"
          >
            <div class="text-xs text-muted-foreground">{{ $t('store.docs') }}</div>
            <div class="mt-0.5 flex items-center gap-1 truncate text-primary">
              {{ $t('store.link') }} <FaIcon name="i-lucide:external-link" class="size-3 shrink-0" />
            </div>
          </a>
        </div>

        <div>
          <div class="mb-1.5 text-sm font-medium">{{ $t('store.version') }}</div>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="v in detailVersions"
              :key="v.id"
              class="rounded-md border px-2 py-1 text-xs"
              :class="v.id === (detailInstall?.version || detailApp.latestVersion) ? 'border-primary bg-primary/10 text-primary' : 'border-border text-muted-foreground'"
            >
              {{ v.id }}
            </span>
            <span v-if="!detailVersions.length && detailLoading" class="text-xs text-muted-foreground">{{ $t('common.loading') }}</span>
          </div>
        </div>

        <div v-if="detailApp.description" class="rounded-lg border bg-muted/20 p-3 text-sm text-muted-foreground">
          {{ detailApp.description }}
        </div>

        <div v-if="detailReadme">
          <div class="mb-1.5 text-sm font-medium">{{ $t('store.readme') }}</div>
          <div class="max-h-[50vh] overflow-auto rounded-lg border p-4 text-sm leading-6" v-html="detailReadme" />
        </div>

        <div class="flex justify-end border-t pt-3">
          <FaButton v-if="!detailInstall" v-auth="['store:write']" size="sm" :disabled="detailLoading" @click="openInstall(detailApp, detailApp.latestVersion)">
            {{ $t('store.installV', { v: detailApp.latestVersion }) }}
          </FaButton>
          <FaButton v-else size="sm" variant="outline" @click="openInstall(detailApp, detailApp.latestVer)">
            {{ $t('store.upgradeToV', { v: detailApp.latestVer }) }}
          </FaButton>
        </div>
      </div>
    </FaDrawer>

    <!-- 安装向导 -->
    <FaModal v-model="installVisible" :title="installTaskId ? $t('store.installRunningTitle', { name: installTarget?.name || '' }) : $t('store.installTitle', { name: installTarget?.name || '' })" class="max-w-2xl!" :destroy-on-close="true" :close-on-click-modal="false">
      <!-- 阶段一：参数 -->
      <div v-if="!installTaskId" class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-28 shrink-0 text-sm text-muted-foreground">{{ $t('store.appName') }}</span>
          <FaInput v-model="installForm.name" :placeholder="$t('store.appNamePh')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-28 shrink-0 text-sm text-muted-foreground">{{ $t('nodes.targetNode') }}</span>
          <YdSelect
            v-model="installNode"
            :options="[{ label: $t('nodes.localPanel'), value: '' }, ...onlineNodes.filter(x => x.id !== 'local').map(n => ({ label: `${n.name}（${n.arch || 'linux'}）`, value: n.id }))]"
            button-class="min-w-0 flex-1"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-28 shrink-0 text-sm text-muted-foreground">{{ $t('store.version') }}</span>
          <YdSelect v-model="installForm.version" :options="installVersions.map(v => ({ label: v.name || v.id, value: v.id }))" button-class="flex-1" />
        </div>
        <div v-if="installPrefilled" class="-mt-1 text-xs text-muted-foreground">
          {{ $t('store.prefillHint') }}
        </div>
        <div v-if="dbHostKey" class="flex items-start gap-3">
          <span class="w-28 shrink-0 pt-2 text-sm text-muted-foreground">{{ $t('store.database') }}</span>
          <div class="flex min-w-0 flex-1 flex-col gap-2">
            <YdSelect
              v-model="dbSource"
              :options="[
                { label: installTarget?.kind === 'php' && dbHostKey ? $t('store.dbManual') : $t('store.dbDefault'), value: 'default' },
                { label: $t('store.dbExternal'), value: 'external' },
              ]"
              button-class="w-full"
              @update:model-value="dbSource === 'external' && loadDBInstances()"
            />
            <div v-if="dbSource === 'external'" class="space-y-2 rounded-md border border-dashed p-2.5">
              <YdSelect
                v-model="dbTarget.instanceId"
                :options="[{ label: $t('store.dbPickInstance'), value: 0, disabled: true }, ...dbInstances.map(i => ({ label: `${i.name}（${i.type} :${i.port}）`, value: i.id }))]"
                button-class="w-full"
              />
              <div class="flex gap-2">
                <FaInput v-model="dbTarget.database" :placeholder="$t('store.dbNamePh')" class="flex-1" />
                <FaInput v-model="dbTarget.user" :placeholder="$t('store.dbUserPh')" class="flex-1" />
              </div>
              <label class="flex cursor-pointer items-center gap-1.5 text-xs text-muted-foreground">
                <input v-model="dbTarget.createIfMissing" type="checkbox" class="accent-[var(--primary)]" />
                {{ $t('store.dbAutoCreate') }}
              </label>
            </div>
          </div>
        </div>
        <template v-for="f in installFields" :key="f.envKey">
          <!-- 关联服务 / 复合字段（一期只读说明） -->
          <div v-if="f.type === 'service' || f.type === 'apps'" class="flex items-center gap-3">
            <span class="w-28 shrink-0 text-sm text-muted-foreground">{{ fieldLabel(f) }}</span>
            <span class="flex-1 rounded-md border border-dashed px-2 py-1.5 text-xs text-muted-foreground">
              {{ f.default ? String(f.default) : $t('store.autoLinkService') }}{{ f.description ? `（${f.description}）` : '' }}
            </span>
          </div>
          <div v-else-if="!isDbManagedField(f.envKey)" class="flex items-center gap-3">
            <span class="w-28 shrink-0 text-sm text-muted-foreground">
              {{ fieldLabel(f) }}<span v-if="f.required" class="text-red-500">*</span>
            </span>
            <!-- 枚举：下拉选择 -->
            <YdSelect
              v-if="f.type === 'select' && f.values?.length"
              v-model="installForm.params[f.envKey]"
              :options="f.values.map(o => ({ label: o.label || o.value, value: o.value }))"
              :disabled="!fieldIsEditable(f)"
              button-class="flex-1"
            />
            <!-- 密码 / 端口：输入 + 随机 -->
            <div v-else class="flex flex-1 items-center gap-1.5">
              <FaInput
                v-model="installForm.params[f.envKey]"
                :type="f.type === 'password' ? 'password' : f.type === 'number' ? 'number' : 'text'"
                :disabled="!fieldIsEditable(f)"
                :placeholder="f.required ? $t('store.required') : $t('store.optional')"
                class="flex-1"
              />
              <FaButton
                v-if="fieldIsEditable(f) && (f.type === 'password' || isPortField(f))"
                variant="outline" size="icon-sm" :title="$t('store.randomGen')"
                @click="randomizeField(f)"
              >
                <FaIcon name="i-lucide:dices" class="text-sm" />
              </FaButton>
            </div>
          </div>
          <div v-if="f.description" class="-mt-2 pl-31 text-xs text-muted-foreground">{{ f.description }}</div>
        </template>
        <div v-if="installProxyEnv" class="flex items-center gap-3">
          <span class="w-28 shrink-0 text-sm text-muted-foreground">{{ $t('store.oneClickProxy') }}</span>
          <FaInput v-model="installForm.domain" :placeholder="$t('store.proxyPh')" class="flex-1" />
        </div>

        <!-- 高级选项 -->
        <button
          type="button"
          class="flex cursor-pointer items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
          @click="installAdvanced = !installAdvanced"
        >
          <FaIcon :name="installAdvanced ? 'i-lucide:chevron-down' : 'i-lucide:chevron-right'" class="text-xs" />
          {{ $t('store.advanced') }}
        </button>
        <div v-if="installAdvanced" class="flex flex-col gap-3 rounded-md border border-dashed p-3">
          <div class="flex items-center gap-3">
            <span class="w-28 shrink-0 text-sm text-muted-foreground">{{ $t('store.network') }}</span>
            <div class="flex flex-1 items-center gap-2">
              <YdSelect
                v-model="netSel"
                :options="[
                  { label: $t('store.netYpanel'), value: 'ypanel_default' },
                  { label: $t('store.netHost'), value: 'host' },
                  ...dockerNets.filter(x => !['ypanel_default', 'host', 'bridge', 'none'].includes(x.name) && !x.name.startsWith('br-')).map(n => ({ label: n.name + $t('store.netExisting'), value: n.name })),
                  { label: $t('store.netCreate'), value: '__create__' },
                ]"
                button-class="min-w-0 flex-1"
              />
              <FaInput
                v-if="netSel === '__create__'"
                v-model="netNew"
                :placeholder="$t('store.netNamePh')"
                class="w-40!"
              />
            </div>
          </div>
          <div class="flex items-center gap-3">
            <span class="w-28 shrink-0 text-sm text-muted-foreground">{{ $t('store.timezone') }}</span>
            <label class="flex cursor-pointer items-center gap-1.5 text-sm">
              <input v-model="installTZ" type="checkbox" class="accent-[rgb(var(--primary))]">
              {{ $t('store.injectTZ') }}
            </label>
            <FaInput v-if="installTZ" v-model="installTZValue" placeholder="Asia/Shanghai" class="w-44!" />
          </div>
          <div class="flex items-start gap-3">
            <span class="w-28 shrink-0 pt-2 text-sm text-muted-foreground">{{ $t('store.hostsMap') }}</span>
            <div class="flex-1">
              <div v-for="(h, hi) in installHosts" :key="hi" class="mb-1.5 flex items-center gap-1.5">
                <FaInput v-model="h.host" :placeholder="$t('store.hostPh')" class="flex-1" />
                <span class="text-xs text-muted-foreground">→</span>
                <FaInput v-model="h.ip" placeholder="IP" class="w-36!" />
                <FaButton variant="ghost" size="icon-sm" @click="installHosts.splice(hi, 1)">
                  <FaIcon name="i-lucide:x" class="text-xs" />
                </FaButton>
              </div>
              <div class="flex gap-1.5">
                <FaButton variant="outline" size="sm" @click="openHostImport()">
                  <FaIcon name="i-lucide:download" class="mr-1" /> {{ $t('store.importHosts') }}
                </FaButton>
                <FaButton variant="outline" size="sm" @click="installHosts.push({ host: '', ip: '' })">
                  <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('store.addMapping') }}
                </FaButton>
              </div>
            </div>
          </div>
          <label class="flex cursor-pointer items-center gap-1.5 text-sm">
            <input v-model="installMountHosts" type="checkbox" class="accent-[rgb(var(--primary))]">
            {{ $t('store.mountHosts') }}
          </label>
          <div class="text-xs text-muted-foreground">
            {{ $t('store.advNote') }}
          </div>
          <div v-if="netSel === 'host'" class="rounded-md border border-amber-500/30 bg-amber-500/5 p-2 text-xs text-amber-600">
            {{ $t('store.hostModeNote') }}
          </div>
        </div>
        <div class="text-xs text-muted-foreground">
          {{ $t('store.installNote') }}
        </div>
      </div>
      <!-- 阶段二：任务日志 -->
      <div v-else class="flex flex-col gap-3">
        <div class="flex items-center gap-3 text-sm">
          <span class="rounded-full px-2 py-0.5 text-xs" :class="installTask?.status === 'success' ? 'bg-emerald-500/10 text-emerald-600' : installTask?.status === 'failed' ? 'bg-red-500/10 text-red-600' : 'bg-blue-500/10 text-blue-600'">
            {{ installTask?.status === 'success' ? $t('store.installSuccess') : installTask?.status === 'failed' ? $t('store.installFailed') : $t('store.inProgress') }}
          </span>
          <span class="text-xs text-muted-foreground">{{ $t('store.taskN', { n: installTaskId }) }}</span>
          <FaButton variant="link" size="sm" @click="taskCenter.open(installTaskId)">
            {{ $t('store.openInTaskCenter') }}
          </FaButton>
        </div>
        <YdLogViewer :logs="installTask?.logText || ''" height="320px" :loading="installLogLoading" />
      </div>
      <template #footer>
        <template v-if="!installTaskId">
          <FaButton variant="outline" @click="installVisible = false">{{ $t('common.cancel') }}</FaButton>
          <FaButton :loading="installing" @click="doInstall">{{ $t('store.createInstallTask') }}</FaButton>
        </template>
        <template v-else>
          <FaButton variant="outline" @click="installVisible = false">
            {{ installTask?.status === 'running' ? $t('store.runInBackground') : $t('common.close') }}
          </FaButton>
          <FaButton v-if="installTask?.status === 'failed'" @click="installTaskId = 0">{{ $t('store.backToParams') }}</FaButton>
        </template>
      </template>
    </FaModal>

    <!-- 导入宿主机 hosts -->
    <FaModal v-model="hostImportVisible" :title="$t('store.importHosts')" class="max-w-lg!" :destroy-on-close="true">
      <div class="flex flex-col gap-2">
        <div class="text-xs text-muted-foreground">
          {{ $t('store.importHostsDesc') }}
        </div>
        <div class="max-h-72 overflow-y-auto rounded-md border">
          <div v-if="hostImportLoading" class="p-6 text-center text-sm text-muted-foreground">
            {{ $t('common.loading') }}
          </div>
          <div v-else-if="!hostEntries.length" class="p-6 text-center text-sm text-muted-foreground">
            {{ $t('store.noHostEntries') }}
          </div>
          <div
            v-for="e in hostEntries"
            :key="e.ip"
            class="flex items-center gap-2 border-b px-3 py-2 text-sm last:border-b-0 hover:bg-accent/30"
          >
            <div class="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
              <label
                v-for="h in e.hosts"
                :key="e.ip + '|' + h"
                class="flex cursor-pointer items-center gap-1.5"
              >
                <input
                  v-model="hostPicked[e.ip + '|' + h]"
                  type="checkbox"
                  class="accent-[rgb(var(--primary))]"
                >
                <span class="truncate">{{ h }}</span>
              </label>
            </div>
            <span class="shrink-0 font-mono text-xs text-muted-foreground">{{ e.ip }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="hostImportVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton @click="confirmHostImport">{{ $t('store.importSelected') }}</FaButton>
      </template>
    </FaModal>

    <!-- 已安装：卸载（选项 + 名称确认） -->
    <YdDangerDelete
      v-model:visible="uninstallVisible"
      :title="$t('store.uninstallTitle', { name: uninstallTarget })"
      :name="uninstallTarget"
      :options="UNINSTALL_OPTS"
      :loading="uninstalling"
      :confirm-text="$t('store.uninstall')"
      @confirm="doUninstall"
    />

    <!-- 已安装：参数编辑（.env，保存重建生效） -->
    <FaModal v-model="paramsVisible" :title="$t('store.paramsTitle', { name: paramsProject })" class="max-w-2xl!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <textarea
          v-model="paramsText"
          class="h-72 w-full resize-y rounded-md border border-input bg-muted/30 p-3 font-mono text-xs leading-5 outline-none focus:border-primary"
          spellcheck="false"
        />
        <div class="text-xs text-muted-foreground">
          {{ $t('store.paramsNote') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="paramsVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="paramsSaving" @click="saveParams">{{ $t('store.saveRebuild') }}</FaButton>
      </template>
    </FaModal>

    <!-- 已安装：日志 -->
    <FaModal v-model="logsVisible" :title="$t('store.logsTitle', { name: logsProject })" class="max-w-3xl!" :destroy-on-close="true">
      <YdLogViewer :logs="logsText" height="55vh" :loading="logsLoading" />
      <template #footer>
        <FaButton variant="outline" @click="logsVisible = false">{{ $t('common.close') }}</FaButton>
        <FaButton :loading="logsLoading" @click="refreshLogs">{{ $t('common.refresh') }}</FaButton>
      </template>
    </FaModal>

    <!-- 添加/编辑源 -->
    <FaModal v-model="sourceModalVisible" :title="sourceEditing ? $t('store.editSource') : $t('store.addSource')" class="max-w-xl!">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.name') }}</span>
          <FaInput v-model="sourceForm.name" :disabled="!!sourceEditing?.builtin" :placeholder="$t('store.sourceNamePh')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.type') }}</span>
          <YdSelect
            v-model="sourceForm.type"
            :options="[
              { label: $t('store.srcTypeYpGit'), value: 'yp-git' },
              { label: $t('store.srcTypeYpUrl'), value: 'yp-url' },
              { label: $t('store.srcTypeOnepanel'), value: 'onepanel' },
            ]"
            :disabled="!!sourceEditing?.builtin"
            button-class="flex-1"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('store.address') }}</span>
          <FaInput
            v-model="sourceForm.url"
            :placeholder="sourceForm.type === 'yp-git' ? 'https://github.com/you/yp-apps.git' : sourceForm.type === 'onepanel' ? 'https://apps-assets.fit2cloud.com/dev/1panel.json.zip' : 'https://example.com/index.json'"
            class="flex-1"
          />
        </div>
        <div v-if="sourceForm.type === 'yp-git'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('store.branch') }}</span>
          <FaInput v-model="sourceForm.branch" placeholder="main" class="flex-1" />
        </div>
        <div v-if="sourceForm.type === 'yp-git'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">Token</span>
          <FaInput v-model="sourceForm.authToken" type="password" :placeholder="$t('store.tokenPh')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="sourceForm.remark" :placeholder="$t('store.optional')" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="sourceModalVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="sourceSaving" @click="saveSource">{{ $t('common.save') }}</FaButton>
      </template>
    </FaModal>

    <!-- M54-P3 属主分配 -->
    <YdOwnerDialog
      v-model="ownerVisible"
      :title="$t('common.owner.storeTitle', { name: ownerTarget?.composeProject || '' })"
      :current-owner-id="ownerTarget?.ownerId ?? 0"
      @confirm="doSetOwner"
    />
  </div>
</template>
