<script setup lang="ts">
import type { MigratePreview, DbBackup, DbDatabase, DbInstance, DbUser, DbKV, GrantRow } from '@/api/modules/database'
import apiDb, { dbTuningApi } from '@/api/modules/database'
import type { StorageAccount } from '@/api/modules/storage'
import { storageApi } from '@/api/modules/storage'
import { storeApi } from '@/api/modules/store'
import YdDangerDelete from '@/components/YdDangerDelete/index.vue'
import { i18n } from '@/locales'

defineOptions({
  name: 'DatabaseIndex',
})

const appAccountStore = useAppAccountStore()

const instances = ref<DbInstance[]>([])
const loading = ref(false)
let timer: ReturnType<typeof setInterval> | null = null

const TYPE_META: Record<string, { label: string, icon: string, color: string }> = {
  mysql: { label: 'MySQL', icon: 'database', color: '#00758f' },
  postgres: { label: 'PostgreSQL', icon: 'database-zap', color: '#336791' },
  redis: { label: 'Redis', icon: 'database-backup', color: '#dc382d' },
  mongo: { label: 'MongoDB', icon: 'database-zap', color: '#47a248' },
}

async function load() {
  loading.value = true
  try {
    instances.value = await apiDb.list()
  }
  finally {
    loading.value = false
  }
}

// ---- 创建 ----
const createVisible = ref(false)
const createForm = ref({ name: '', type: 'mysql', port: 0, password: '' })
const creating = ref(false)

function openCreate() {
  createForm.value = { name: '', type: 'mysql', port: 0, password: '' }
  createVisible.value = true
}

async function doCreate() {
  creating.value = true
  try {
    await apiDb.create(createForm.value)
    useFaToast().success(i18n.global.t('database.toast.creating'))
    createVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.createFailed'), { description: e?.message })
  }
  finally {
    creating.value = false
  }
}

// 危险删除（选项 + 名称确认）；商店接管来源可反向级联卸载商店应用
const delVisible = ref(false)
const delTarget = ref<DbInstance | null>(null)
const deleting = ref(false)
const delIsStore = computed(() => delTarget.value?.origin === 'container' && (delTarget.value?.composeProject || '').startsWith('app-'))
const DEL_OPTS = computed(() => {
  const t = delTarget.value
  if (!t || t.origin === 'external') {
    return []
  }
  const opts = [
    { key: 'data', label: i18n.global.t('database.del.optData'), desc: i18n.global.t('database.del.optDataDesc') },
    { key: 'backups', label: i18n.global.t('database.del.optBackups'), desc: i18n.global.t('database.del.optBackupsDesc') },
  ]
  if (delIsStore.value) {
    opts.push({ key: 'uninstallApp', label: i18n.global.t('database.del.optUninstall'), desc: i18n.global.t('database.del.optUninstallDesc') + (t.origin === 'container' ? i18n.global.t('database.del.optUninstallDescExtra') : '') })
  }
  return opts
})

function remove(inst: DbInstance) {
  try { localStorage.setItem('ydd-debug', 'remove() called ' + inst.name + ' @' + Date.now()) } catch {}
  delTarget.value = inst
  delVisible.value = true
  try { localStorage.setItem('ydd-debug', 'remove() done delVisible=' + delVisible.value + ' @' + Date.now()) } catch {}
}

async function doDelete(checked: Record<string, boolean>) {
  if (!delTarget.value) {
    return
  }
  deleting.value = true
  try {
    await apiDb.remove(delTarget.value.id, !!checked.data, !!checked.backups)
    if (checked.uninstallApp && delIsStore.value) {
      await storeApi.uninstall(delTarget.value.composeProject, { purgeData: !!checked.data })
      useFaToast().success(i18n.global.t('database.toast.deletedWithApp'))
    }
    else {
      useFaToast().success(delTarget.value.origin === 'external' ? i18n.global.t('database.toast.adoptReleased') : i18n.global.t('database.toast.deleted'))
    }
    delVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.deleteFailed'), { description: e?.message })
  }
  finally {
    deleting.value = false
  }
}

async function toggleRun(inst: DbInstance) {
  if (inst.origin === 'external') {
    useFaToast().info(i18n.global.t('database.toast.externalManaged'))
    return
  }
  try {
    if (inst.running) {
      await apiDb.stop(inst.id)
      useFaToast().success(i18n.global.t('database.stopped'))
    }
    else {
      await apiDb.start(inst.id)
      useFaToast().success(i18n.global.t('database.started'))
    }
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.operationFailed'), { description: e?.message })
  }
}

// ---- 接入外部实例 ----
const adopting = ref('')

async function doAdopt(inst: DbInstance) {
  adopting.value = inst.composeProject
  try {
    await apiDb.adopt(inst.composeProject)
    useFaToast().success(i18n.global.t('database.toast.adopted', { name: inst.name }))
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.adoptFailed'), { description: e?.message })
  }
  finally {
    adopting.value = ''
  }
}

const extVisible = ref(false)
const extForm = ref({ name: '', type: 'mysql', host: '127.0.0.1', port: 3306, user: 'root', password: '', remark: '' })
const extSaving = ref(false)

function openExternal() {
  extForm.value = { name: '', type: 'mysql', host: '127.0.0.1', port: 3306, user: 'root', password: '', remark: '' }
  extVisible.value = true
}

function extType(t: string) {
  const ports: Record<string, number> = { mysql: 3306, postgres: 5432, redis: 6379, mongo: 27017 }
  const users: Record<string, string> = { mysql: 'root', postgres: 'postgres', redis: 'default', mongo: 'root' }
  extForm.value.port = ports[t] || 3306
  extForm.value.user = users[t] || 'root'
}

async function doCreateExternal() {
  extSaving.value = true
  try {
    await apiDb.createExternal(extForm.value)
    useFaToast().success(i18n.global.t('database.toast.externalAdopted'))
    extVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.adoptExternalFailed'), { description: e?.message })
  }
  finally {
    extSaving.value = false
  }
}

// ---- 连接信息 ----
const connVisible = ref(false)
const conn = ref<{ user: string, password: string, host: string, port: number, container?: string, innerPort?: string, networks?: string, lanIp?: string, mapPort?: string } | null>(null)

async function showConn(inst: DbInstance) {
  try {
    conn.value = await apiDb.reveal(inst.id)
    connVisible.value = true
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.revealFailed'), { description: e?.message })
  }
}

// ---- 管理面板 ----
const active = ref<DbInstance | null>(null)
const tab = ref<'databases' | 'users' | 'tuning' | 'backups'>('databases')
const panelLoading = ref(false)
const databases = ref<DbDatabase[]>([])
const users = ref<DbUser[]>([])
const backups = ref<DbBackup[]>([])

function openPanel(inst: DbInstance) {
  active.value = inst
  tab.value = 'databases'
  refreshPanel()
  loadRemote()
}

const toast = useFaToast()

// ---- 实例迁移（同类型、按库选择） ----
const mOpen = ref(false)
const mTarget = ref<number>(0)
const mPreview = ref<MigratePreview | null>(null)
const mSelected = ref<string[]>([])
const mLoading = ref(false)
const mStarting = ref(false)

function openMigrate() {
  mTarget.value = 0
  mPreview.value = null
  mSelected.value = []
  mOpen.value = true
}

async function doPreview() {
  if (!active.value || !mTarget.value) {
    toast.warning(i18n.global.t('database.migrate.selectTarget'))
    return
  }
  mLoading.value = true
  try {
    mPreview.value = await apiDb.migratePreview(active.value.id, mTarget.value)
    if (!mPreview.value.compatible) {
      toast.error(i18n.global.t('database.migrate.typeMismatch'))
      return
    }
    mSelected.value = mPreview.value.dbs.map((d: { name: string }) => d.name)
  }
  catch (e: any) {
    toast.error(i18n.global.t('database.migrate.previewFailed'), { description: e?.message })
  }
  finally {
    mLoading.value = false
  }
}

async function doMigrate() {
  if (!active.value || !mTarget.value || !mSelected.value.length) {
    toast.warning(i18n.global.t('database.migrate.selectDb'))
    return
  }
  const ok = await useFaModal().confirm({
    title: i18n.global.t('database.migrate.confirmTitle'),
    content: i18n.global.t('database.migrate.confirmContent', { n: mSelected.value.length }),
  })
  if (!ok) {
    return
  }
  mStarting.value = true
  try {
    await apiDb.migrateStart({ src: active.value.id, target: mTarget.value, dbs: mSelected.value })
    toast.success(i18n.global.t('database.migrate.started'))
    mOpen.value = false
  }
  catch (e: any) {
    toast.error(i18n.global.t('database.migrate.startFailed'), { description: e?.message })
  }
  finally {
    mStarting.value = false
  }
}

// ---- 备份文件导入（1Panel 等标准 dump 格式） ----
const biFile = ref<File | null>(null)
const biBusy = ref(false)

async function doBackupImport() {
  if (!active.value || !biFile.value) {
    toast.warning(i18n.global.t('database.backup.selectFile'))
    return
  }
  const buf = await biFile.value.arrayBuffer()
  if (buf.byteLength > 100 * 1024 * 1024) {
    toast.warning(i18n.global.t('database.backup.fileTooLarge'))
    return
  }
  // 分块转 base64（避免大文件栈溢出）
  const bytes = new Uint8Array(buf)
  let bin = ''
  const chunk = 0x8000
  for (let i = 0; i < bytes.length; i += chunk) {
    bin += String.fromCharCode(...bytes.subarray(i, i + chunk))
  }
  biBusy.value = true
  try {
    await apiDb.backupImport(active.value.id, biFile.value.name, btoa(bin))
    toast.success(i18n.global.t('database.backup.imported'))
    biFile.value = null
    refreshPanel()
  }
  catch (e: any) {
    toast.error(i18n.global.t('database.backup.importFailed'), { description: e?.message })
  }
  finally {
    biBusy.value = false
  }
}

// ---- 远程访问开关（B3 remote 管理用户） ----
const remoteOn = ref(false)
const remoteBusy = ref(false)

async function loadRemote() {
  if (!active.value) {
    return
  }
  try {
    remoteOn.value = await apiDb.remoteAccessStatus(active.value.id)
  }
  catch {
    remoteOn.value = false
  }
}

async function toggleRemote(v: boolean | undefined) {
  const on = !!v
  if (!active.value) {
    return
  }
  remoteBusy.value = true
  try {
    await apiDb.remoteAccess(active.value.id, on)
    remoteOn.value = on
    useFaToast().success(on ? i18n.global.t('database.remote.created') : i18n.global.t('database.remote.removed'))
  }
  catch (e: any) {
    remoteOn.value = !on
    useFaToast().error(i18n.global.t('database.toast.operationFailed'), { description: e?.message })
  }
  finally {
    remoteBusy.value = false
  }
}

async function refreshPanel() {
  if (!active.value) {
    return
  }
  panelLoading.value = true
  const id = active.value.id
  try {
    if (tab.value === 'databases') {
      databases.value = await apiDb.databases(id)
    }
    else if (tab.value === 'users') {
      users.value = await apiDb.users(id)
    }
    else {
      backups.value = await apiDb.backups(id)
    }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.loadFailed'), { description: e?.message })
  }
  finally {
    panelLoading.value = false
  }
}

watch(tab, refreshPanel)

// 库操作
const dbModalVisible = ref(false)
const dbForm = ref({ name: '', charset: 'utf8mb4' })

function openCreateDb() {
  dbForm.value = { name: '', charset: 'utf8mb4' }
  dbModalVisible.value = true
}

async function doCreateDb() {
  if (!active.value) {
    return
  }
  try {
    await apiDb.createDatabase(active.value.id, dbForm.value.name, dbForm.value.charset)
    useFaToast().success(i18n.global.t('database.toast.created'))
    dbModalVisible.value = false
    refreshPanel()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.createFailed'), { description: e?.message })
  }
}

function dropDb(name: string) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('database.db.deleteTitle'),
    content: i18n.global.t('database.db.deleteConfirm', { name }),
    onConfirm: async () => {
      if (!active.value) {
        return
      }
      try {
        await apiDb.dropDatabase(active.value.id, name)
        useFaToast().success(i18n.global.t('database.toast.deleted'))
        refreshPanel()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('database.toast.deleteFailed'), { description: e?.message })
      }
    },
  })
}

// 用户操作
const userModalVisible = ref(false)
const userForm = ref({ name: '', host: '%', password: '' })

function openCreateUser() {
  userForm.value = { name: '', host: '%', password: '' }
  userModalVisible.value = true
}

async function doCreateUser() {
  if (!active.value) {
    return
  }
  try {
    await apiDb.createUser(active.value.id, userForm.value)
    useFaToast().success(i18n.global.t('database.toast.created'))
    userModalVisible.value = false
    refreshPanel()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.createFailed'), { description: e?.message })
  }
}

function dropUser(u: DbUser) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('database.user.deleteTitle'),
    content: i18n.global.t('database.user.deleteConfirm', { name: u.name, host: u.host ? `@${u.host}` : '' }),
    onConfirm: async () => {
      if (!active.value) {
        return
      }
      try {
        await apiDb.dropUser(active.value.id, u.name, u.host)
        useFaToast().success(i18n.global.t('database.toast.deleted'))
        refreshPanel()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('database.toast.deleteFailed'), { description: e?.message })
      }
    },
  })
}

const pwdModalVisible = ref(false)
const pwdForm = ref({ name: '', host: '', password: '' })

function openChangePwd(u: DbUser) {
  pwdForm.value = { name: u.name, host: u.host, password: '' }
  pwdModalVisible.value = true
}

async function doChangePwd() {
  if (!active.value) {
    return
  }
  try {
    await apiDb.changeUserPassword(active.value.id, pwdForm.value.name, pwdForm.value.password, pwdForm.value.host)
    useFaToast().success(i18n.global.t('database.toast.pwdChanged'))
    pwdModalVisible.value = false
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.updateFailed'), { description: e?.message })
  }
}

// 备份操作
const backupBusy = ref(false)
// M34：远程上传选项
const storageAccounts = ref<StorageAccount[]>([])
const backupStorageId = ref(0)
const backupKeep = ref(0)

async function loadStorageAccounts() {
  try {
    storageAccounts.value = await storageApi.list()
  }
  catch {
    storageAccounts.value = []
  }
}

async function doBackup() {
  if (!active.value) {
    return
  }
  backupBusy.value = true
  useFaToast().info(i18n.global.t('database.backup.running'))
  try {
    const out = await apiDb.createBackup(active.value.id, {
      storageAccountId: backupStorageId.value || undefined,
      keep: backupKeep.value || undefined,
    })
    if (out.remoteKey) {
      useFaToast().success(i18n.global.t('database.backup.doneRemote', { file: out.file, remoteKey: out.remoteKey }))
    }
    else if (out.file) {
      useFaToast().success(i18n.global.t('database.backup.done', { file: out.file }))
    }
    refreshPanel()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.backup.failed'), { description: e?.message })
  }
  finally {
    backupBusy.value = false
  }
}

function downloadBackup(b: DbBackup) {
  window.open(apiDb.backupDownloadURL(b.path, appAccountStore.token))
}

function removeBackup(b: DbBackup) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('database.backup.deleteTitle'),
    content: i18n.global.t('database.backup.deleteConfirm', { name: b.name }),
    onConfirm: async () => {
      if (!active.value) {
        return
      }
      try {
        await apiDb.deleteBackup(active.value.id, b.name)
        useFaToast().success(i18n.global.t('database.toast.deleted'))
        refreshPanel()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('database.toast.deleteFailed'), { description: e?.message })
      }
    },
  })
}

function restoreBackup(b: DbBackup) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('database.backup.restoreTitle'),
    content: i18n.global.t('database.backup.restoreConfirm', { name: b.name }),
    onConfirm: async () => {
      if (!active.value) {
        return
      }
      try {
        await apiDb.restoreBackup(active.value.id, b.name)
        useFaToast().success(i18n.global.t('database.backup.restored'))
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('database.backup.restoreFailed'), { description: e?.message })
      }
    },
  })
}


// ---- M39：MySQL 调优（参数/状态/慢查询）+ 授权矩阵 ----
const tuningVars = ref<DbKV[]>([])
const tuningFilter = ref('')
const tuningStatus = ref<Record<string, number>>({})
const varEditing = ref('')
const varEditVal = ref('')

const isMysql = computed(() => active.value?.type === 'mysql')

async function loadTuning() {
  if (!active.value)
    return
  try {
    const [vars, st] = await Promise.all([
      dbTuningApi.variables(active.value.id, tuningFilter.value),
      dbTuningApi.status(active.value.id).catch(() => ({})),
    ])
    tuningVars.value = vars
    tuningStatus.value = st
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.tuning.loadFailed'), { description: e?.message })
  }
}

watch([tab, tuningFilter], () => {
  if (tab.value === 'tuning' && isMysql.value)
    loadTuning()
})

const varAllowlist = ['max_connections', 'wait_timeout', 'interactive_timeout', 'slow_query_log', 'long_query_time', 'character_set_server', 'collation_server', 'innodb_buffer_pool_size', 'innodb_flush_log_at_trx_commit', 'max_allowed_packet', 'tmp_table_size', 'max_heap_table_size', 'sort_buffer_size', 'join_buffer_size', 'read_rnd_buffer_size', 'table_open_cache', 'thread_cache_size', 'open_files_limit', 'log_bin_trust_function_creators', 'sql_mode']

function editVar(kv: DbKV) {
  if (!varAllowlist.includes(kv.name.toLowerCase())) {
    useFaToast().warning(i18n.global.t('database.tuning.notAllowed'))
    return
  }
  varEditing.value = kv.name
  varEditVal.value = kv.value
}

async function saveVar() {
  if (!active.value || !varEditing.value)
    return
  try {
    await dbTuningApi.setVariable(active.value.id, varEditing.value, varEditVal.value)
    useFaToast().success(i18n.global.t('database.tuning.varApplied', { name: varEditing.value }))
    varEditing.value = ''
    await loadTuning()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.updateFailed'), { description: e?.message })
  }
}

const privVisible = ref(false)
const privDBs = ref<string[]>([])
const privDB = ref('')
const privUser = ref('')
const privHost = ref('')
const privMatrix = ref<GrantRow[]>([])
const privList = ['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER', 'INDEX', 'CREATE VIEW', 'SHOW VIEW', 'ALL PRIVILEGES']
const privChecked = ref<Set<string>>(new Set())
const privSaving = ref(false)

async function openPrivs(u: DbUser) {
  if (!active.value || active.value.type !== 'mysql') {
    useFaToast().warning(i18n.global.t('database.priv.mysqlOnly'))
    return
  }
  privUser.value = u.name
  privHost.value = u.host
  privVisible.value = true
  try {
    privDBs.value = (await apiDb.databases(active.value.id)).map(d => d.name)
    if (privDBs.value.length) {
      privDB.value = privDBs.value[0]
      await loadMatrix()
    }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.priv.dbListFailed'), { description: e?.message })
  }
}

async function loadMatrix() {
  if (!active.value || !privDB.value)
    return
  try {
    privMatrix.value = await dbTuningApi.grantMatrix(active.value.id, privDB.value)
    const mine = privMatrix.value.find(r => r.user === privUser.value && r.host === privHost.value)
    privChecked.value = new Set((mine?.privs || []).map(p => p.toUpperCase()))
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.priv.grantLoadFailed'), { description: e?.message })
  }
}

function togglePriv(p: string) {
  const n = new Set(privChecked.value)
  if (n.has(p))
    n.delete(p)
  else n.add(p)
  privChecked.value = n
}

async function applyPrivs(grant: boolean) {
  if (!active.value)
    return
  const privs = [...privChecked.value]
  if (!privs.length) {
    useFaToast().warning(i18n.global.t('database.priv.selectOne'))
    return
  }
  privSaving.value = true
  try {
    await dbTuningApi.setPrivileges(active.value.id, { db: privDB.value, user: privUser.value, host: privHost.value, privs, grant })
    useFaToast().success(grant ? i18n.global.t('database.priv.granted') : i18n.global.t('database.priv.revoked'))
    await loadMatrix()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('database.toast.operationFailed'), { description: e?.message })
  }
  finally {
    privSaving.value = false
  }
}


onMounted(() => {
  load()
  loadStorageAccounts()
  timer = setInterval(load, 8000)
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
        <div class="flex items-center gap-2">
          <YdMorphIcon name="database" :size="24" />
          <span>{{ $t('database.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('database.description') }}</span>
      </template>
      <div class="flex gap-2">
        <FaButton v-auth="['db:write']" size="sm" variant="outline" @click="openExternal">
          <FaIcon name="i-lucide:plug-zap" class="mr-1" /> {{ $t('database.actions.external') }}
        </FaButton>
        <FaButton v-auth="['db:write']" size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('database.actions.create') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 实例卡片 -->
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="inst in instances"
          :key="inst.id"
          class="rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
        >
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-2">
              <YdMorphIcon :name="TYPE_META[inst.type]?.icon || 'database'" :size="22" :color="TYPE_META[inst.type]?.color" />
              <div>
                <div class="flex items-center gap-1.5">
                  <span class="font-medium">{{ inst.name }}</span>
                  <span
                    class="rounded px-1.5 py-0.5 text-xs"
                    :class="inst.origin === 'external' ? 'bg-blue-500/10 text-blue-600' : 'bg-muted text-muted-foreground'"
                  >
                    {{ inst.origin === 'external' ? $t('database.origin.external') : inst.origin === 'store' ? $t('database.origin.store') : $t('database.origin.container') }}
                  </span>
                </div>
                <div class="text-xs text-muted-foreground">{{ TYPE_META[inst.type]?.label }} · {{ inst.host || '127.0.0.1' }}:{{ inst.port }}</div>
              </div>
            </div>
            <span class="rounded-full px-2 py-0.5 text-xs" :class="inst.running ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ inst.running ? $t('database.running') : $t('database.stopped') }}
            </span>
          </div>
          <div class="mt-3 flex flex-wrap items-center gap-1.5">
            <template v-if="inst.adoptable">
              <FaButton size="sm" :loading="adopting === inst.composeProject" @click="doAdopt(inst)">
                <FaIcon name="i-lucide:plug-zap" class="mr-1" /> {{ $t('database.actions.adopt') }}
              </FaButton>
              <span class="text-xs text-muted-foreground">{{ $t('database.adoptHint') }}</span>
            </template>
            <template v-else>
            <FaButton variant="outline" size="sm" @click="openPanel(inst)">
              {{ $t('database.actions.manage') }}
            </FaButton>
            <FaButton v-if="inst.origin !== 'external'" variant="ghost" size="sm" @click="toggleRun(inst)">
              {{ inst.running ? $t('common.stop') : $t('common.start') }}
            </FaButton>
            <FaButton variant="ghost" size="sm" @click="showConn(inst)">
              {{ $t('database.conn.title') }}
            </FaButton>
            <FaButton variant="ghost" size="sm" class="ml-auto text-red-500!" @click="remove(inst)">
              {{ inst.origin === 'external' ? $t('database.actions.release') : $t('common.delete') }}
            </FaButton>
            </template>
          </div>
        </div>
        <div v-if="!instances.length && !loading" class="rounded-lg border p-10 text-center text-sm text-muted-foreground md:col-span-2 xl:col-span-3">
          {{ $t('database.empty') }}
        </div>
      </div>

      <!-- 管理面板 -->
      <div v-if="active" class="mt-4 rounded-lg border bg-background">
        <div class="flex flex-wrap items-center gap-2 border-b px-4 py-3">
          <span class="font-medium">{{ $t('database.panelTitle', { name: active.name }) }}</span>
          <FaTabs
            v-model="tab" :list="[
              { label: $t('database.tabs.databases'), value: 'databases' },
              { label: $t('database.tabs.users'), value: 'users' },
              { label: $t('database.tabs.tuning'), value: 'tuning' },
              { label: $t('database.tabs.backups'), value: 'backups' },
            ]" class="ml-2"
          />
          <div class="ml-auto flex items-center gap-2">
            <FaButton variant="outline" size="sm" @click="openMigrate">{{ $t('database.migrate.title') }}</FaButton>
            <label class="flex items-center gap-1.5 text-xs text-muted-foreground" :title="$t('database.remote.tip')">
              {{ $t('database.remote.label') }}
              <FaSwitch :model-value="remoteOn" :disabled="remoteBusy" @update:model-value="toggleRemote" />
            </label>
            <template v-if="tab === 'databases'">
              <FaButton variant="outline" size="sm" @click="openCreateDb">{{ $t('database.db.createTitle') }}</FaButton>
            </template>
            <template v-else-if="tab === 'users'">
              <FaButton variant="outline" size="sm" @click="openCreateUser">{{ $t('database.user.createTitle') }}</FaButton>
            </template>
            <template v-else-if="tab === 'tuning'">
              <FaButton variant="outline" size="sm" :disabled="!isMysql" :title="isMysql ? $t('database.tuning.refreshStatus') : $t('database.tuning.mysqlOnlyShort')" @click="loadTuning">{{ $t('common.refresh') }}</FaButton>
            </template>
            <template v-else>
              <YdSelect
                v-if="storageAccounts.length" v-model="backupStorageId" size="sm" class="w-36"
                :options="[{ label: $t('database.backup.localOnly'), value: 0 }, ...storageAccounts.map(a => ({ label: `↑ ${a.name}`, value: a.id }))]"
              />
              <FaInput v-if="backupStorageId" v-model="backupKeep" type="number" class="w-16" :title="$t('database.backup.keepTip')" :placeholder="$t('database.backup.keepPlaceholder')" />
              <FaButton variant="outline" size="sm" :loading="backupBusy" @click="doBackup">{{ $t('database.backup.now') }}</FaButton>
              <label class="cursor-pointer rounded border border-input px-2 py-1 text-xs hover:bg-accent/50">
                {{ $t('database.backup.importAction') }}
                <input type="file" class="hidden" accept=".sql,.sql.gz,.rdb,.gz,.archive" @change="(e: any) => { biFile = e.target.files?.[0] ?? null; doBackupImport() }">
              </label>
            </template>
            <FaButton variant="ghost" size="icon-sm" :title="$t('common.refresh')" @click="refreshPanel">
              <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="panelLoading ? 'animate-spin' : ''" />
            </FaButton>
            <FaButton variant="ghost" size="icon-sm" :title="$t('common.close')" @click="active = null">
              <FaIcon name="i-lucide:x" class="text-sm" />
            </FaButton>
          </div>
        </div>

        <div v-if="active.type === 'redis' && tab === 'databases'" class="px-4 py-2 text-xs text-muted-foreground">
          {{ $t('database.hints.redisDbs') }}
        </div>
        <div v-if="active.type === 'redis' && tab === 'users'" class="px-4 py-2 text-xs text-muted-foreground">
          {{ $t('database.hints.redisUsers') }}
        </div>

        <!-- 数据库表 -->
        <table v-if="tab === 'databases'" class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2">{{ $t('common.name') }}</th>
              <th class="px-4 py-2">{{ $t('common.size') }}</th>
              <th class="hidden px-4 py-2 sm:table-cell">{{ $t('database.charset') }}</th>
              <th class="px-4 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in databases" :key="d.name" class="border-t hover:bg-accent/30">
              <td class="px-4 py-2 font-mono text-[13px]">{{ d.name }}</td>
              <td class="px-4 py-2 text-xs tabular-nums">{{ d.sizeMb.toFixed(2) }} {{ active.type === 'redis' ? 'keys' : 'MB' }}</td>
              <td class="hidden px-4 py-2 text-xs text-muted-foreground sm:table-cell">{{ d.charset || '—' }}</td>
              <td class="px-4 py-2 text-right">
                <FaButton variant="outline" size="sm" @click="dropDb(d.name)">{{ $t('common.delete') }}</FaButton>
              </td>
            </tr>
            <tr v-if="!databases.length && !panelLoading">
              <td colspan="4" class="px-4 py-8 text-center text-muted-foreground">{{ $t('database.db.noData') }}</td>
            </tr>
          </tbody>
        </table>

        <!-- 用户表 -->
        <table v-else-if="tab === 'users'" class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2">{{ $t('database.username') }}</th>
              <th class="hidden px-4 py-2 md:table-cell">{{ $t('database.host') }}</th>
              <th class="hidden px-4 py-2 md:table-cell">{{ $t('database.desc') }}</th>
              <th class="px-4 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in users" :key="u.name + u.host" class="border-t hover:bg-accent/30">
              <td class="px-4 py-2 font-mono text-[13px]">{{ u.name }}</td>
              <td class="hidden px-4 py-2 font-mono text-xs text-muted-foreground md:table-cell">{{ u.host || '—' }}</td>
              <td class="hidden px-4 py-2 font-mono text-xs text-muted-foreground md:table-cell">{{ u.extra }}</td>
              <td class="px-4 py-2 text-right">
                <FaButton variant="ghost" size="sm" @click="openChangePwd(u)">{{ $t('database.actions.changePwd') }}</FaButton>
                <FaButton v-if="isMysql" variant="ghost" size="sm" @click="openPrivs(u)">{{ $t('database.actions.privs') }}</FaButton>
                <FaButton v-if="u.name !== 'default' && u.name !== 'postgres'" variant="outline" size="sm" @click="dropUser(u)">{{ $t('common.delete') }}</FaButton>
              </td>
            </tr>
            <tr v-if="!users.length && !panelLoading">
              <td colspan="4" class="px-4 py-8 text-center text-muted-foreground">{{ $t('database.user.noData') }}</td>
            </tr>
          </tbody>
        </table>

        <!-- 备份表 -->
        <table v-else class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2">{{ $t('database.file') }}</th>
              <th class="px-4 py-2">{{ $t('common.size') }}</th>
              <th class="hidden px-4 py-2 md:table-cell">{{ $t('common.time') }}</th>
              <th class="px-4 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in backups" :key="b.name" class="border-t hover:bg-accent/30">
              <td class="px-4 py-2 font-mono text-[13px]">{{ b.name }}</td>
              <td class="px-4 py-2 text-xs tabular-nums">{{ b.sizeMb.toFixed(2) }} MB</td>
              <td class="hidden px-4 py-2 text-xs tabular-nums text-muted-foreground md:table-cell">{{ new Date(b.modTime).toLocaleString('zh-CN', { hour12: false }) }}</td>
              <td class="px-4 py-2 text-right">
                <FaButton variant="ghost" size="sm" @click="downloadBackup(b)">{{ $t('common.download') }}</FaButton>
                <FaButton variant="ghost" size="sm" @click="restoreBackup(b)">{{ $t('database.actions.restore') }}</FaButton>
                <FaButton variant="outline" size="sm" @click="removeBackup(b)">{{ $t('common.delete') }}</FaButton>
              </td>
            </tr>
            <tr v-if="!backups.length && !panelLoading">
              <td colspan="4" class="px-4 py-8 text-center text-muted-foreground">{{ $t('database.backup.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    

        <!-- M39：MySQL 调优面板 -->
        <div v-if="active && tab === 'tuning' && isMysql" class="mt-4 space-y-4">
          <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
            <div class="rounded-lg border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('database.tuning.threadsConnected') }}</div>
              <div class="text-xl font-semibold tabular-nums">{{ tuningStatus.threads_connected ?? '—' }}</div>
            </div>
            <div class="rounded-lg border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('database.tuning.threadsRunning') }}</div>
              <div class="text-xl font-semibold tabular-nums">{{ tuningStatus.threads_running ?? '—' }}</div>
            </div>
            <div class="rounded-lg border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('database.tuning.slowQueries') }}</div>
              <div class="text-xl font-semibold tabular-nums">{{ tuningStatus.slow_queries ?? '—' }}</div>
            </div>
            <div class="rounded-lg border p-3">
              <div class="text-xs text-muted-foreground">{{ $t('database.tuning.slowLog') }}</div>
              <div class="text-xl font-semibold">{{ tuningStatus.slow_query_log === 1 ? $t('common.on') : tuningStatus.slow_query_log === 0 ? $t('common.off') : '—' }}</div>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <FaInput v-model="tuningFilter" :placeholder="$t('database.tuning.filterPlaceholder')" class="w-56!" @keyup.enter="loadTuning" />
            <FaButton variant="outline" size="sm" @click="loadTuning">{{ $t('database.actions.query') }}</FaButton>
            <span class="text-xs text-muted-foreground">{{ $t('database.tuning.allowlistHint') }}</span>
          </div>
          <div class="max-h-96 overflow-auto rounded-lg border">
            <table class="w-full text-sm">
              <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
                <tr><th class="px-3 py-2">{{ $t('database.tuning.param') }}</th><th class="px-3 py-2">{{ $t('database.tuning.value') }}</th><th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th></tr>
              </thead>
              <tbody>
                <tr v-if="!tuningVars.length"><td colspan="3" class="px-3 py-6 text-center text-muted-foreground">{{ $t('database.tuning.noMatch') }}</td></tr>
                <tr v-for="kv in tuningVars" :key="kv.name" class="border-t hover:bg-accent/30">
                  <td class="px-3 py-1.5 font-mono text-xs">{{ kv.name }}</td>
                  <td class="px-3 py-1.5">
                    <template v-if="varEditing === kv.name">
                      <span class="inline-flex items-center gap-1">
                        <input v-model="varEditVal" class="h-7 w-40 rounded border border-input bg-background px-1.5 font-mono text-xs outline-none focus:border-primary" @keyup.enter="saveVar">
                        <FaButton size="sm" @click="saveVar">{{ $t('database.tuning.save') }}</FaButton>
                        <FaButton variant="ghost" size="sm" @click="varEditing = ''">{{ $t('common.cancel') }}</FaButton>
                      </span>
                    </template>
                    <template v-else><span class="font-mono text-xs">{{ kv.value }}</span></template>
                  </td>
                  <td class="px-3 py-1.5 text-right">
                    <FaButton v-if="varAllowlist.includes(kv.name.toLowerCase())" variant="ghost" size="sm" @click="editVar(kv)">{{ $t('database.tuning.edit') }}</FaButton>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
    </FaPageMain>

    <!-- 接入外部实例 -->
    <FaModal v-model="extVisible" :title="$t('database.ext.title')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.type') }}</span>
          <div class="flex flex-1 gap-1.5">
            <button
              v-for="(meta, t) in TYPE_META"
              :key="t"
              type="button"
              class="flex-1 cursor-pointer rounded-md border px-2 py-1.5 text-sm transition-colors"
              :class="extForm.type === t ? 'border-primary bg-primary/10' : 'border-border hover:bg-accent/50'"
              @click="extType(t as string); extForm.type = t"
            >
              {{ meta.label }}
            </button>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.instanceName') }}</span>
          <FaInput v-model="extForm.name" :placeholder="$t('database.ext.namePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.host') }}</span>
          <FaInput v-model="extForm.host" :placeholder="$t('database.ext.hostPlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.port') }}</span>
          <FaInput v-model="extForm.port" type="number" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.userLabel') }}</span>
          <FaInput v-model="extForm.user" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.password') }}</span>
          <FaInput v-model="extForm.password" type="password" :placeholder="$t('database.required')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="extForm.remark" :placeholder="$t('database.optional')" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          {{ $t('database.ext.hint') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="extVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="extSaving" @click="doCreateExternal">{{ $t('database.ext.submit') }}</FaButton>
      </template>
    </FaModal>

    <!-- 删除实例（选项 + 名称确认；商店接管来源可级联卸载应用） -->
    <YdDangerDelete
      v-model:visible="delVisible"
      :title="delTarget?.origin === 'external' ? $t('database.del.releaseTitle', { name: delTarget?.name || '' }) : $t('database.del.title', { name: delTarget?.name || '' })"
      :name="delTarget?.name || ''"
      :options="DEL_OPTS"
      :loading="deleting"
      :confirm-text="delTarget?.origin === 'external' ? $t('database.actions.release') : $t('common.delete')"
      @confirm="doDelete"
    />

    <!-- 创建实例 -->
    <FaModal v-model="createVisible" :title="$t('database.create.title')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.type') }}</span>
          <div class="flex flex-1 gap-1.5">
            <button
              v-for="(meta, t) in TYPE_META"
              :key="t"
              type="button"
              class="flex-1 cursor-pointer rounded-md border px-2 py-1.5 text-sm transition-colors"
              :class="createForm.type === t ? 'border-primary bg-primary/10' : 'border-border hover:bg-accent/50'"
              @click="createForm.type = t"
            >
              {{ meta.label }}
            </button>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.instanceName') }}</span>
          <FaInput v-model="createForm.name" :placeholder="$t('database.create.namePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.port') }}</span>
          <FaInput v-model="createForm.port" type="number" :placeholder="$t('database.create.portPlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.create.rootPassword') }}</span>
          <FaInput v-model="createForm.password" :placeholder="$t('database.create.pwdPlaceholder')" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          {{ $t('database.create.hint') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="creating" @click="doCreate">{{ $t('database.create.submit') }}</FaButton>
      </template>
    </FaModal>

    <!-- 连接信息 -->
    <FaModal v-model="connVisible" :title="$t('database.conn.title')" :destroy-on-close="true">
      <div v-if="conn" class="flex flex-col gap-2 font-mono text-sm">
        <div class="text-xs font-medium text-muted-foreground">{{ $t('database.conn.containerAccess') }}</div>
        <div>{{ $t('database.conn.addr') }}<span class="rounded bg-muted px-1.5">{{ conn.container ? `${conn.container}:${conn.innerPort}` : '—' }}</span></div>
        <div v-if="conn.networks" class="text-xs text-muted-foreground">{{ $t('database.conn.networks', { networks: conn.networks }) }}</div>
        <div class="mt-1 text-xs font-medium text-muted-foreground">{{ $t('database.conn.externalAccess') }}</div>
        <div>{{ $t('database.conn.addr') }}<span class="rounded bg-muted px-1.5">{{ conn.lanIp || conn.host }}:{{ conn.mapPort || conn.port }}</span></div>
        <div class="mt-1 text-xs font-medium text-muted-foreground">{{ $t('database.conn.rootCred') }}</div>
        <div>{{ $t('database.conn.account') }}<span class="rounded bg-muted px-1.5">{{ conn.user || '—' }}</span></div>
        <div>{{ $t('database.conn.pwd') }}<span class="rounded bg-muted px-1.5">{{ conn.password }}</span></div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="connVisible = false">{{ $t('common.close') }}</FaButton>
      </template>
    </FaModal>

    <!-- 新建数据库 -->
    <FaModal v-model="dbModalVisible" :title="$t('database.db.createTitle')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.name') }}</span>
          <FaInput v-model="dbForm.name" :placeholder="$t('database.ph.alphanumeric')" class="flex-1" @keyup.enter="doCreateDb" />
        </div>
        <div v-if="active?.type === 'mysql'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.charset') }}</span>
          <select v-model="dbForm.charset" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="utf8mb4">utf8mb4</option>
            <option value="utf8mb3">utf8mb3</option>
            <option value="ascii">ascii</option>
          </select>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="dbModalVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton @click="doCreateDb">{{ $t('common.create') }}</FaButton>
      </template>
    </FaModal>

    <!-- 新建用户 -->
    <FaModal v-model="userModalVisible" :title="$t('database.user.createTitle')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.username') }}</span>
          <FaInput v-model="userForm.name" :placeholder="$t('database.ph.alphanumeric')" class="flex-1" />
        </div>
        <div v-if="active?.type === 'mysql'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.host') }}</span>
          <FaInput v-model="userForm.host" :placeholder="$t('database.user.hostPlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('database.password') }}</span>
          <FaInput v-model="userForm.password" :placeholder="$t('database.pwdPlaceholder')" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="userModalVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton @click="doCreateUser">{{ $t('common.create') }}</FaButton>
      </template>
    </FaModal>

    <!-- 改密 -->
    <FaModal v-model="pwdModalVisible" :title="$t('database.pwd.title', { name: pwdForm.name })" :destroy-on-close="true">
      <FaInput v-model="pwdForm.password" :placeholder="$t('database.pwd.newPlaceholder')" class="w-full" />
      <template #footer>
        <FaButton variant="outline" @click="pwdModalVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton @click="doChangePwd">{{ $t('common.confirm') }}</FaButton>
      </template>
    </FaModal>
  <!-- 迁移向导 -->
  <FaModal v-model="mOpen" :title="$t('database.migrate.title')" :width="560">
    <div class="space-y-3 p-1 text-sm">
      <div class="text-xs text-muted-foreground">
        {{ $t('database.migrate.source') }}<span class="font-mono text-foreground">{{ active?.name }}{{ $t('database.migrate.typeWrap', { t: active?.type }) }}</span> {{ $t('database.migrate.hint') }}
      </div>
      <div class="flex items-center gap-2">
        <label class="w-20 text-right text-xs">{{ $t('database.migrate.target') }}</label>
        <select
          v-model.number="mTarget"
          class="flex-1 rounded border border-input bg-background px-2 py-1 text-xs outline-none focus:ring-1 focus:ring-primary"
        >
          <option :value="0" disabled>{{ $t('database.migrate.selectPlaceholder') }}</option>
          <option v-for="i in instances.filter(x => x.type === active?.type && x.id !== active?.id)" :key="i.id" :value="i.id">
            {{ i.name }}（{{ i.origin === 'external' ? $t('database.originShort.external') : $t('database.origin.container') }}）
          </option>
        </select>
        <FaButton variant="outline" size="sm" :loading="mLoading" :disabled="!mTarget" @click="doPreview">{{ $t('database.migrate.preview') }}</FaButton>
      </div>
      <div v-if="mPreview?.compatible" class="rounded border p-2">
        <div class="mb-1.5 flex items-center text-xs text-muted-foreground">
          {{ $t('database.migrate.dbsSelected', { sel: mSelected.length, total: mPreview.dbs.length }) }}
          <span class="flex-1" />
          <button type="button" class="text-xs text-primary hover:underline" @click="mSelected = mPreview.dbs.map((d: { name: string }) => d.name)">{{ $t('database.migrate.selectAll') }}</button>
        </div>
        <div class="max-h-48 space-y-1 overflow-auto">
          <label v-for="d in mPreview.dbs" :key="d.name" class="flex cursor-pointer items-center gap-2 rounded px-1 py-0.5 text-xs hover:bg-accent/50">
            <input v-model="mSelected" type="checkbox" :value="d.name" class="accent-primary">
            <span class="font-mono">{{ d.name }}</span>
            <span class="ml-auto text-muted-foreground tabular-nums">{{ d.sizeMb > 0 ? `${d.sizeMb.toFixed(1)} MB` : '' }}</span>
          </label>
        </div>
      </div>
      <div v-else-if="mPreview && !mPreview.compatible" class="rounded bg-red-50 p-2 text-xs text-red-600 dark:bg-red-950/30">
        {{ $t('database.migrate.incompatible', { src: mPreview.srcType, target: mPreview.targetType }) }}
      </div>
    </div>
    <template #footer>
      <FaButton size="sm" @click="mOpen = false">{{ $t('common.cancel') }}</FaButton>
      <FaButton type="primary" size="sm" :loading="mStarting" :disabled="!mSelected.length" @click="doMigrate">{{ $t('database.migrate.start') }}</FaButton>
    </template>
  </FaModal>

    <!-- M39：授权矩阵 -->
    <FaModal v-model="privVisible" :title="$t('database.priv.title', { target: `${privUser}@${privHost}` })" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-2">
          <span class="text-sm text-muted-foreground">{{ $t('database.database') }}</span>
          <select v-model="privDB" class="h-8 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary" @change="loadMatrix">
            <option v-for="d in privDBs" :key="d" :value="d">{{ d }}</option>
          </select>
          <span class="ml-auto text-xs text-muted-foreground">{{ $t('database.priv.hint') }}</span>
        </div>
        <div class="flex flex-wrap gap-2">
          <label v-for="p in privList" :key="p" class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border px-2 py-1 text-xs hover:bg-accent/50">
            <input type="checkbox" :checked="privChecked.has(p)" @change="togglePriv(p)" class="accent-[var(--primary)]">
            {{ p }}
          </label>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" :loading="privSaving" class="text-red-500!" @click="applyPrivs(false)">{{ $t('database.priv.revokeSelected') }}</FaButton>
        <FaButton :loading="privSaving" @click="applyPrivs(true)">{{ $t('database.priv.grantSelected') }}</FaButton>
      </template>
    </FaModal>
  </div>
</template>