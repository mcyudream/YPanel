<script setup lang="ts">
import type { CronTask, CronTaskLog } from '@/api/modules/cron'
import type { StorageAccount } from '@/api/modules/storage'
import type { ScriptItem } from '@/api/modules/cron'
import apiCron, { scriptApi, scriptRunApi } from '@/api/modules/cron'
import { storageApi } from '@/api/modules/storage'
import { i18n } from '@/locales'

defineOptions({
  name: 'CronIndex',
})

const tasks = ref<CronTask[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await apiCron.list(1, 100)
    tasks.value = res.items
  }
  finally {
    loading.value = false
  }
}

// ---- 创建/编辑 ----
const editorVisible = ref(false)
const isCreate = ref(false)
const editorId = ref<number>(0)
const form = ref({ name: '', cron: '*/5 * * * *', command: '', timeoutSecs: 300, type: 'shell', payload: '' })
const saving = ref(false)
// M34：备份类任务的结构化参数（含远程上传选项）
const backupFields = ref({ dbId: '', siteName: '', srcDir: '', name: '', project: '', storageAccountId: 0, keep: 0 })
const structuredTypes = ['db_backup', 'site_backup', 'dir_backup', 'compose_backup']
const isBackupType = computed(() => structuredTypes.includes(form.value.type))
const storageAccounts = ref<StorageAccount[]>([])
// M36：新类型参数
const curlUrls = ref('')
const certId = ref('')
const noPayloadTypes = ['cut_website_log', 'clean']

function loadStorageAccounts() {
  storageApi.list().then((a) => {
    storageAccounts.value = a
  }).catch(() => {
    storageAccounts.value = []
  })
}

function resetBackupFields() {
  backupFields.value = { dbId: '', siteName: '', srcDir: '', name: '', project: '', storageAccountId: 0, keep: 0 }
}

function openCreate() {
  isCreate.value = true
  editorId.value = 0
  form.value = { name: '', cron: '*/5 * * * *', command: '', timeoutSecs: 300, type: 'shell', payload: '' }
  resetBackupFields()
  editorVisible.value = true
}

function openEdit(t: CronTask) {
  isCreate.value = false
  editorId.value = t.id
  form.value = { name: t.name, cron: t.cron, command: t.command, timeoutSecs: t.timeoutSecs, type: t.type || 'shell', payload: t.payload || '' }
  resetBackupFields()
  if (structuredTypes.includes(form.value.type)) {
    try {
      const p = JSON.parse(form.value.payload || '{}')
      backupFields.value = {
        dbId: p.dbId ? String(p.dbId) : '',
        siteName: p.siteName || '',
        srcDir: p.srcDir || '',
        name: p.name || '',
        project: p.project || '',
        storageAccountId: p.storageAccountId || 0,
        keep: p.keep || 0,
      }
    }
    catch {
      // 旧格式（裸 ID）兜底：db_backup 的 payload 直接是实例 ID
      if (/^\d+$/.test(form.value.payload.trim())) {
        backupFields.value.dbId = form.value.payload.trim()
      }
    }
  }
  editorVisible.value = true
}

function buildPayload(): string | null {
  const f = backupFields.value
  const common = { storageAccountId: f.storageAccountId || undefined, keep: f.keep || undefined }
  switch (form.value.type) {
    case 'db_backup':
      if (!f.dbId) {
        useFaToast().warning(i18n.global.t('cron.requireDbId'))
        return null
      }
      return JSON.stringify({ dbId: Number(f.dbId), ...common })
    case 'site_backup':
      if (!f.siteName) {
        useFaToast().warning(i18n.global.t('cron.requireSiteName'))
        return null
      }
      return JSON.stringify({ siteName: f.siteName, ...common })
    case 'dir_backup':
      if (!f.srcDir) {
        useFaToast().warning(i18n.global.t('cron.requireSrcDir'))
        return null
      }
      return JSON.stringify({ srcDir: f.srcDir, name: f.name || undefined, ...common })
    case 'compose_backup':
      if (!f.project) {
        useFaToast().warning(i18n.global.t('cron.requireProject'))
        return null
      }
      return JSON.stringify({ project: f.project, ...common })
    case 'curl': {
      const urls = curlUrls.value.split('\n').map(s => s.trim()).filter(Boolean)
      if (!urls.length) {
        useFaToast().warning(i18n.global.t('cron.requireUrl'))
        return null
      }
      return JSON.stringify({ urls })
    }
    case 'cert_renew':
      if (!certId.value) {
        useFaToast().warning(i18n.global.t('cron.requireCertId'))
        return null
      }
      return JSON.stringify({ certId: Number(certId.value) })
    case 'cut_website_log':
    case 'clean':
      return ''
  }
  return form.value.payload
}

async function save() {
  if (!form.value.name || !form.value.cron) {
    useFaToast().warning(i18n.global.t('cron.requireNameCron'))
    return
  }
  if (isBackupType.value || noPayloadTypes.includes(form.value.type) || form.value.type === 'curl' || form.value.type === 'cert_renew') {
    const payload = buildPayload()
    if (payload === null) {
      return
    }
    form.value.payload = payload
  }
  else if (!form.value.command) {
    useFaToast().warning(i18n.global.t('cron.requireCommand'))
    return
  }
  saving.value = true
  try {
    if (isCreate.value) {
      await apiCron.create(form.value)
      useFaToast().success(i18n.global.t('cron.created'))
    }
    else {
      await apiCron.update(editorId.value, form.value)
      useFaToast().success(i18n.global.t('cron.saved'))
    }
    editorVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('cron.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

async function toggle(t: CronTask) {
  try {
    await apiCron.update(t.id, { enabled: !t.enabled })
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('cron.opFailed'), { description: e?.message })
  }
}

async function runNow(t: CronTask) {
  try {
    await apiCron.run(t.id)
    useFaToast().success(i18n.global.t('cron.runTriggered'))
    setTimeout(load, 800)
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('cron.runFailed'), { description: e?.message })
  }
}

function remove(t: CronTask) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('cron.deleteTitle'),
    content: i18n.global.t('cron.deleteConfirm', { name: t.name }),
    onConfirm: async () => {
      try {
        await apiCron.remove(t.id)
        useFaToast().success(i18n.global.t('cron.deleted'))
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('cron.deleteFailed'), { description: e?.message })
      }
    },
  })
}

// ---- 日志 ----
const logsVisible = ref(false)
const logsTaskId = ref<number | undefined>(undefined)
const logs = ref<CronTaskLog[]>([])
const logsTotal = ref(0)
const logsPage = ref(1)
const logsLoading = ref(false)
const expanded = ref<CronTaskLog | null>(null)

async function openLogs(taskId?: number) {
  logsTaskId.value = taskId
  logsPage.value = 1
  logsVisible.value = true
  await loadLogs()
}

async function loadLogs() {
  logsLoading.value = true
  try {
    const res = await apiCron.logs(logsTaskId.value, logsPage.value, 20)
    logs.value = res.items
    logsTotal.value = res.total
  }
  finally {
    logsLoading.value = false
  }
}

function fmtTime(iso?: string | null) {
  return iso ? new Date(iso).toLocaleString('zh-CN', { hour12: false }) : '—'
}

onMounted(() => {
  load()
  loadStorageAccounts()
})

// ---- 脚本库（M36） ----
const scriptsVisible = ref(false)
const scripts = ref<ScriptItem[]>([])
const scriptMode = ref<'list' | 'edit'>('list')
const scriptForm = ref({ id: 0, name: '', content: '' })
const scriptSaving = ref(false)
const runBusyId = ref(0)
const runResult = ref<{ name: string, success: boolean, output: string } | null>(null)

async function openScripts() {
  scriptsVisible.value = true
  scriptMode.value = 'list'
  runResult.value = null
  await loadScripts()
}

async function loadScripts() {
  try {
    scripts.value = await scriptApi.list()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('cron.scriptListLoadFailed'), { description: e?.message })
  }
}

function newScript() {
  scriptForm.value = { id: 0, name: '', content: '' }
  scriptMode.value = 'edit'
}

function editScript(s: ScriptItem) {
  scriptForm.value = { id: s.id, name: s.name, content: s.content }
  scriptMode.value = 'edit'
}

async function saveScript() {
  if (!scriptForm.value.name || !scriptForm.value.content) {
    useFaToast().warning(i18n.global.t('cron.scriptRequired'))
    return
  }
  scriptSaving.value = true
  try {
    if (scriptForm.value.id) {
      await scriptApi.update(scriptForm.value.id, scriptForm.value)
    }
    else {
      await scriptApi.create(scriptForm.value)
    }
    useFaToast().success(i18n.global.t('cron.savedShort'))
    scriptMode.value = 'list'
    await loadScripts()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('cron.saveFailed'), { description: e?.message })
  }
  finally {
    scriptSaving.value = false
  }
}

function removeScript(s: ScriptItem) {
  useFaModal().confirm({
    title: i18n.global.t('cron.scriptDeleteTitle'),
    content: i18n.global.t('cron.scriptDeleteConfirm', { name: s.name }),
    onConfirm: async () => {
      await scriptApi.remove(s.id)
      useFaToast().success(i18n.global.t('cron.deleted'))
      await loadScripts()
    },
  })
}

async function runScript(s: ScriptItem) {
  runBusyId.value = s.id
  runResult.value = null
  try {
    const out = await scriptRunApi.run(s.id)
    runResult.value = { name: s.name, success: out.success, output: out.output || i18n.global.t('cron.noOutput') }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('cron.execFailed'), { description: e?.message })
  }
  finally {
    runBusyId.value = 0
  }
}
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="calendar-clock" :size="24" />
          <span>{{ $t('cron.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('cron.description') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="openScripts()">
          <FaIcon name="i-lucide:scroll" class="mr-1" /> {{ $t('cron.scripts') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openLogs()">
          <FaIcon name="i-lucide:scroll-text" class="mr-1" /> {{ $t('cron.allRecords') }}
        </FaButton>
        <FaButton v-auth="['cron:write']" size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('cron.newTask') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('cron.task') }}</th>
              <th class="px-3 py-2">cron</th>
              <th class="hidden px-3 py-2 lg:table-cell">{{ $t('cron.command') }}</th>
              <th class="px-3 py-2">{{ $t('common.status') }}</th>
              <th class="hidden px-3 py-2 xl:table-cell">{{ $t('cron.lastRun') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !tasks.length">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('common.loading') }}
              </td>
            </tr>
            <tr v-else-if="!tasks.length">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('cron.noTasks') }}
              </td>
            </tr>
            <tr v-for="t in tasks" :key="t.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2">
                <div class="font-medium">{{ t.name }}</div>
                <div class="text-xs text-muted-foreground">{{ $t('cron.timeout', { n: t.timeoutSecs }) }}</div>
              </td>
              <td class="px-3 py-2 font-mono text-xs">
                {{ t.cron }}
              </td>
              <td class="hidden max-w-72 truncate px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell" :title="t.command">
                {{ t.command }}
              </td>
              <td class="px-3 py-2">
                <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="t.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  <span class="inline-block size-1.5 rounded-full" :class="t.enabled ? 'animate-pulse bg-current' : 'bg-current'" />
                  {{ t.enabled ? $t('common.enabled') : $t('common.disabled') }}
                </span>
                <span
                  v-if="t.lastSuccess !== null && t.lastSuccess !== undefined"
                  class="ml-1.5 rounded-full px-2 py-0.5 text-xs"
                  :class="t.lastSuccess ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'"
                >
                  {{ t.lastSuccess ? $t('common.success') : $t('common.failed') }}
                </span>
              </td>
              <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground xl:table-cell">
                {{ fmtTime(t.lastRunAt) }}
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton v-auth="['cron:write']" variant="outline" size="sm" @click="runNow(t)">
                    {{ $t('cron.run') }}
                  </FaButton>
                  <FaButton variant="ghost" size="sm" @click="openLogs(t.id)">
                    {{ $t('cron.record') }}
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" :title="t.enabled ? $t('common.disabled') : $t('common.enabled')" @click="toggle(t)">
                    <FaIcon :name="t.enabled ? 'i-lucide:pause' : 'i-lucide:play'" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" :title="$t('common.edit')" @click="openEdit(t)">
                    <FaIcon name="i-lucide:pen-line" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" :title="$t('common.delete')" @click="remove(t)">
                    <FaIcon name="i-lucide:trash" class="text-sm" />
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <!-- 创建/编辑 -->
    <FaModal v-model="editorVisible" :title="isCreate ? $t('cron.createTitle') : $t('cron.editTitle', { name: form.name })" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.taskName') }}</span>
          <FaInput v-model="form.name" :placeholder="$t('cron.namePh')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.taskType') }}</span>
          <select v-model="form.type" class="h-9 flex-1 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
            <option value="shell">{{ $t('cron.typeShell') }}</option>
            <option value="db_backup">{{ $t('cron.typeDbBackup') }}</option>
            <option value="site_backup">{{ $t('cron.typeSiteBackup') }}</option>
            <option value="dir_backup">{{ $t('cron.typeDirBackup') }}</option>
            <option value="compose_backup">{{ $t('cron.typeComposeBackup') }}</option>
            <option value="curl">{{ $t('cron.typeCurl') }}</option>
            <option value="cut_website_log">{{ $t('cron.typeCutLog') }}</option>
            <option value="clean">{{ $t('cron.typeClean') }}</option>
            <option value="cert_renew">{{ $t('cron.typeCertRenew') }}</option>
            <option value="container_op">{{ $t('cron.typeContainerOp') }}</option>
            <option value="script">{{ $t('cron.typeScript') }}</option>
          </select>
        </div>
        <div v-if="form.type === 'shell'" class="flex items-start gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.command') }}</span>
          <textarea
            v-model="form.command"
            class="h-24 w-full flex-1 resize-y rounded-md border border-input bg-background p-2 font-mono text-[13px] outline-none focus:ring-1 focus:ring-primary"
            :placeholder="$t('cron.cmdPh')"
            spellcheck="false"
          />
        </div>
        <div v-if="form.type === 'db_backup'" class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.instanceId') }}</span>
          <FaInput v-model="backupFields.dbId" type="number" :placeholder="$t('cron.instanceIdPh')" class="flex-1" />
        </div>
        <div v-if="form.type === 'site_backup'" class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.siteName') }}</span>
          <FaInput v-model="backupFields.siteName" :placeholder="$t('cron.siteNamePh')" class="flex-1" />
        </div>
        <template v-if="form.type === 'dir_backup'">
          <div class="flex items-center gap-3">
            <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.srcDir') }}</span>
            <FaInput v-model="backupFields.srcDir" :placeholder="$t('cron.srcDirPh')" class="flex-1" />
          </div>
          <div class="flex items-center gap-3">
            <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.archiveName') }}</span>
            <FaInput v-model="backupFields.name" :placeholder="$t('cron.archiveNamePh')" class="flex-1" />
          </div>
        </template>
        <div v-if="form.type === 'compose_backup'" class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.projectName') }}</span>
          <FaInput v-model="backupFields.project" :placeholder="$t('cron.projectNamePh')" class="flex-1" />
        </div>
        <template v-if="isBackupType">
          <div class="flex items-center gap-3">
            <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.remoteStorage') }}</span>
            <div class="flex flex-1 items-center gap-2">
              <YdSelect
                v-model="backupFields.storageAccountId" class="w-52"
                :options="[{ label: $t('cron.localOnly'), value: 0 }, ...storageAccounts.map(a => ({ label: a.name, value: a.id }))]"
              />
              <template v-if="backupFields.storageAccountId">
                <span class="text-xs text-muted-foreground">{{ $t('cron.keepCount') }}</span>
                <FaInput v-model="backupFields.keep" type="number" class="w-20" :placeholder="$t('cron.keepPh')" />
              </template>
              <span v-if="!storageAccounts.length" class="text-xs text-muted-foreground">
                {{ $t('cron.noStorage') }}
              </span>
            </div>
          </div>
        </template>
        <div v-if="form.type === 'curl'" class="flex items-start gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.urlList') }}</span>
          <textarea
            v-model="curlUrls"
            class="h-20 w-full flex-1 resize-y rounded-md border border-input bg-background p-2 font-mono text-[13px] outline-none focus:ring-1 focus:ring-primary"
            :placeholder="$t('cron.urlListPh')"
            spellcheck="false"
          />
        </div>
        <div v-if="form.type === 'cert_renew'" class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.certId') }}</span>
          <FaInput v-model="certId" type="number" :placeholder="$t('cron.certIdPh')" class="flex-1" />
        </div>
        <div v-if="form.type === 'cut_website_log'" class="flex items-center gap-3 text-xs text-muted-foreground">
          <span class="w-24 shrink-0">{{ $t('cron.note') }}</span>
          <span>{{ $t('cron.cutLogNote') }}</span>
        </div>
        <div v-if="form.type === 'clean'" class="flex items-center gap-3 text-xs text-muted-foreground">
          <span class="w-24 shrink-0">{{ $t('cron.note') }}</span>
          <span>{{ $t('cron.cleanNote') }}</span>
        </div>
        <div v-if="form.type === 'container_op'" class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.containerOp') }}</span>
          <FaInput v-model="form.payload" placeholder='{"container":"web","action":"restart"}' class="flex-1" />
        </div>
        <div v-if="form.type === 'script'" class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.scriptId') }}</span>
          <FaInput v-model="form.payload" type="number" :placeholder="$t('cron.scriptIdPh')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.cronExpr') }}</span>
          <FaInput v-model="form.cron" :placeholder="$t('cron.cronExprPh')" class="flex-1" />
        </div>
        <div class="flex gap-1.5">
          <button
            v-for="preset in ['*/1 * * * *', '*/5 * * * *', '0 * * * *', '0 3 * * *', '0 3 * * 1']"
            :key="preset"
            type="button"
            class="cursor-pointer rounded-md border px-2 py-0.5 font-mono text-xs text-muted-foreground transition-colors hover:bg-accent/50"
            @click="form.cron = preset"
          >
            {{ preset }}
          </button>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('cron.timeoutSecs') }}</span>
          <FaInput v-model="form.timeoutSecs" type="number" class="w-32" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="editorVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="saving" @click="save">
          {{ $t('common.save') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 执行记录 -->
    <FaModal v-model="logsVisible" :title="logsTaskId ? $t('cron.recordsTitle', { id: logsTaskId }) : $t('cron.allRecords')" class="max-w-4xl!" :destroy-on-close="true">
      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('cron.startTime') }}</th>
              <th class="px-3 py-2">{{ $t('cron.task') }}</th>
              <th class="px-3 py-2">{{ $t('cron.trigger') }}</th>
              <th class="px-3 py-2">{{ $t('cron.result') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('cron.duration') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="logsLoading && !logs.length">
              <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
                {{ $t('common.loading') }}
              </td>
            </tr>
            <tr v-else-if="!logs.length">
              <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
                {{ $t('cron.noRecords') }}
              </td>
            </tr>
            <template v-for="l in logs" :key="l.id">
              <tr class="cursor-pointer border-t hover:bg-accent/30" @click="expanded = expanded?.id === l.id ? null : l">
                <td class="px-3 py-2 text-xs tabular-nums text-muted-foreground">
                  {{ fmtTime(l.startAt) }}
                </td>
                <td class="px-3 py-2">
                  {{ l.taskName }}
                </td>
                <td class="px-3 py-2 text-xs text-muted-foreground">
                  {{ l.trigger === 'cron' ? $t('cron.triggerCron') : $t('cron.triggerManual') }}
                </td>
                <td class="px-3 py-2">
                  <span class="rounded-full px-2 py-0.5 text-xs" :class="l.success ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'">
                    {{ l.success ? $t('common.success') : $t('common.failed') }}
                  </span>
                </td>
                <td class="px-3 py-2 text-right text-xs tabular-nums text-muted-foreground">
                  {{ (l.durationMs / 1000).toFixed(1) }}s
                </td>
              </tr>
              <tr v-if="expanded?.id === l.id">
                <td colspan="5" class="bg-muted/30 p-0">
                  <pre class="max-h-64 overflow-auto p-3 font-mono text-xs leading-relaxed">{{ l.output || $t('cron.noOutput') }}</pre>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
      <div class="mt-3 flex justify-end">
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <FaButton variant="outline" size="sm" :disabled="logsPage <= 1" @click="logsPage--; loadLogs()">
            {{ $t('cron.prevPage') }}
          </FaButton>
          <span>{{ logsPage }} / {{ Math.max(1, Math.ceil(logsTotal / 20)) }} · {{ $t('common.total', { n: logsTotal }) }}</span>
          <FaButton variant="outline" size="sm" :disabled="logsPage >= Math.ceil(logsTotal / 20)" @click="logsPage++; loadLogs()">
            {{ $t('cron.nextPage') }}
          </FaButton>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="logsVisible = false">
          {{ $t('common.close') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- M36：脚本库 -->
    <FaModal v-model="scriptsVisible" :title="$t('cron.scripts')" class="max-w-3xl!" :destroy-on-close="true">
      <div v-if="scriptMode === 'list'">
        <div class="mb-2 flex items-center gap-2">
          <span class="text-xs text-muted-foreground">{{ $t('cron.scriptsDesc') }}</span>
          <FaButton size="sm" class="ml-auto" @click="newScript">
            <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('cron.newScript') }}
          </FaButton>
        </div>
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('common.name') }}</th>
                <th class="hidden px-3 py-2 md:table-cell">{{ $t('cron.contentPreview') }}</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!scripts.length">
                <td colspan="3" class="px-3 py-8 text-center text-muted-foreground">
                  {{ $t('cron.noScripts') }}
                </td>
              </tr>
              <tr v-for="s in scripts" :key="s.id" class="border-t hover:bg-accent/30">
                <td class="px-3 py-2 font-medium">
                  {{ s.name }}
                </td>
                <td class="hidden max-w-64 truncate px-3 py-2 font-mono text-xs text-muted-foreground md:table-cell">
                  {{ s.content.split('\n')[0] }}
                </td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" :loading="runBusyId === s.id" @click="runScript(s)">
                    {{ $t('cron.run') }}
                  </FaButton>
                  <FaButton variant="ghost" size="sm" @click="editScript(s)">
                    {{ $t('common.edit') }}
                  </FaButton>
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeScript(s)">
                    {{ $t('common.delete') }}
                  </FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="runResult" class="mt-3 rounded-lg border p-3" :class="runResult.success ? 'border-emerald-500/30' : 'border-red-500/40'">
          <div class="mb-1 text-xs" :class="runResult.success ? 'text-emerald-600' : 'text-red-500'">
            {{ runResult.name }} · {{ runResult.success ? $t('common.success') : $t('common.failed') }}
          </div>
          <pre class="max-h-40 overflow-auto font-mono text-xs leading-relaxed">{{ runResult.output }}</pre>
        </div>
      </div>
      <div v-else class="flex flex-col gap-3">
        <FaInput v-model="scriptForm.name" :placeholder="$t('cron.scriptNamePh')" class="w-full" />
        <textarea
          v-model="scriptForm.content"
          class="h-56 w-full resize-y rounded-md border border-input bg-background p-2 font-mono text-[13px] outline-none focus:ring-1 focus:ring-primary"
          :placeholder="$t('cron.scriptContentPh')"
          spellcheck="false"
        />
      </div>
      <template #footer>
        <template v-if="scriptMode === 'list'">
          <FaButton variant="outline" @click="scriptsVisible = false">
            {{ $t('common.close') }}
          </FaButton>
        </template>
        <template v-else>
          <FaButton variant="outline" @click="scriptMode = 'list'">
            {{ $t('cron.backToList') }}
          </FaButton>
          <FaButton :loading="scriptSaving" @click="saveScript">
            {{ $t('common.save') }}
          </FaButton>
        </template>
      </template>
    </FaModal>
  </div>
</template>
