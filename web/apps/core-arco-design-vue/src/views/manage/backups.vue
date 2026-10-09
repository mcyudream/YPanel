<script setup lang="ts">
import type { StorageAccount } from '@/api/modules/storage'
import { panelBackupApi } from '@/api/modules/ops'
import api from '@/api/index'
import { storageApi } from '@/api/modules/storage'
import { i18n } from '@/locales'

defineOptions({
  name: 'ManageBackups',
})

const toast = useFaToast()

const backups = ref<{ name: string, sizeMb: number, modTime: string, path: string }[]>([])
const loading = ref(false)
const backupBusy = ref(false)
const restoreHint = ref('')
// M34：远程上传选项
const storageAccounts = ref<StorageAccount[]>([])
const backupStorageId = ref(0)
const backupKeep = ref(0)

async function load() {
  loading.value = true
  try {
    backups.value = await panelBackupApi.list()
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.backups.loadFail'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function doBackup() {
  backupBusy.value = true
  try {
    const out = await panelBackupApi.create({
      storageAccountId: backupStorageId.value || undefined,
      keep: backupKeep.value || undefined,
    })
    toast.success(out.remoteKey
      ? i18n.global.t('manage.backups.doneRemote', { file: out.file || '', key: out.remoteKey })
      : i18n.global.t('manage.backups.done', { file: out.file || '' }))
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.backups.backupFail'), { description: e?.message })
  }
  finally {
    backupBusy.value = false
  }
}

function downloadBackup(b: { path: string }) {
  const token = localStorage.getItem('token') || ''
  window.open(panelBackupApi.downloadURL(b.path, token))
}

function removeBackup(b: { name: string }) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('manage.backups.deleteTitle'),
    content: i18n.global.t('manage.backups.deleteConfirm', { name: b.name }),
    onConfirm: async () => {
      await panelBackupApi.remove(b.name)
      toast.success(i18n.global.t('manage.saved'))
      await load()
    },
  })
}

async function showHint() {
  restoreHint.value = await panelBackupApi.restoreHint()
}

async function loadStorage() {
  try {
    storageAccounts.value = await storageApi.list()
  }
  catch {
    storageAccounts.value = []
  }
}

// ---- M45：快照恢复/回滚/导入 ----
const restoreVisible = ref(false)
const restoreTarget = ref<{ file: string } | null>(null)
const restorePlan = ref<{ script: string, rollback: string } | null>(null)
const restoreExecuting = ref(false)

async function openRestore(b: { name: string }) {
  restoreTarget.value = { file: b.name }
  try {
    const out = await api.get(`api/v1/snapshots/plan?file=${encodeURIComponent(b.name)}`)
    restorePlan.value = out.data
    restoreVisible.value = true
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.backups.planFail'), { description: e?.message })
  }
}

async function doRestore() {
  if (!restoreTarget.value) return
  restoreExecuting.value = true
  toast.info(i18n.global.t('manage.backups.restoring'))
  try {
    await api.post('api/v1/snapshots/execute', { file: restoreTarget.value.file, rollbackBefore: false })
  }
  catch {
    // 连接被 stop 掐断属预期
  }
  // 轮询健康
  const poll = setInterval(() => {
    fetch('/health').then(r => r.json()).then(() => {
      clearInterval(poll)
      restoreExecuting.value = false
      restoreVisible.value = false
      toast.success(i18n.global.t('manage.backups.restored'))
      load()
    }).catch(() => {})
  }, 3000)
}

async function doRollback() {
  restoreExecuting.value = true
  toast.info(i18n.global.t('manage.backups.rollingBack'))
  try {
    await api.post('api/v1/snapshots/execute', { file: restoreTarget.value?.file || '', rollbackBefore: true })
  }
  catch {}
  const poll = setInterval(() => {
    fetch('/health').then(r => r.json()).then(() => {
      clearInterval(poll)
      restoreExecuting.value = false
      restoreVisible.value = false
      toast.success(i18n.global.t('manage.backups.rolledBack'))
      load()
    }).catch(() => {})
  }, 3000)
}

async function doImportSnap(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  // 先上传到 tmp，再导入
  try {
    const form = new FormData()
    form.append('file', file)
    await api.post('api/v1/files/upload?path=%2Fopt%2Fypanel%2Ftmp', form, { headers: { 'Content-Type': 'multipart/form-data' } })
    await api.post('api/v1/snapshots/import', { path: `/opt/ypanel/tmp/${file.name}` })
    toast.success(i18n.global.t('manage.backups.imported'))
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.backups.importFail'), { description: e?.message })
  }
}

// ---- M46：多节点 + 保留份数 ----
const snapNode = ref('local')
const nodesList = ref<{ id: string, name: string, online: boolean }[]>([])
const keepVal = ref(10)
const pruneBusy = ref(false)

async function loadNodes() {
  try {
    const res = await api.get('api/v1/nodes', { silent: true })
    nodesList.value = (res.data || []).map((n: any) => ({ id: n.id === 0 || n.id === 'local' ? 'local' : String(n.id), name: n.name, online: n.online ?? true }))
  }
  catch {}
}

async function loadKeep() {
  try {
    const res = await api.get('api/v1/snapshots/keep', { silent: true })
    keepVal.value = res.data?.keep || 10
  }
  catch {}
}

async function saveKeep() {
  try {
    await api.put('api/v1/snapshots/keep', { keep: Number(keepVal.value) || 10 })
    toast.success(i18n.global.t('manage.backups.keepSaved', { n: keepVal.value }))
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.saveFailed'), { description: e?.message })
  }
}

async function pruneSnaps() {
  pruneBusy.value = true
  try {
    const res = await api.post('api/v1/snapshots/prune', { node: snapNode.value, keep: Number(keepVal.value) || 0 })
    toast.success(i18n.global.t('manage.backups.pruned', { n: res.data?.removed ?? 0 }))
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.backups.pruneFail'), { description: e?.message })
  }
  finally {
    pruneBusy.value = false
  }
}

// ---- M47：系统级快照 ----
const sysSnaps = ref<{ file: string, sizeMb: number, modTime: string }[]>([])
const sysLoading = ref(false)
const sysCreating = ref(false)
const sysOpts = ref({ websites: false, databases: false })
const sysRestoring = ref('')
const restoreConfirmVisible = ref(false)
const restoreFile = ref('')

async function loadSysSnaps() {
  sysLoading.value = true
  try {
    const res = await api.get(`api/v1/system/snapshots?node=${snapNode.value}`, { silent: true })
    sysSnaps.value = res.data || []
  }
  catch {}
  finally { sysLoading.value = false }
}

async function createSysSnap() {
  sysCreating.value = true
  try {
    const res = await api.post('api/v1/system/snapshots', { node: snapNode.value, ...sysOpts.value })
    toast.success(i18n.global.t('manage.backups.sysTaskCreated', { id: res.data?.taskId }))
    await loadSysSnaps()
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.createFailed'), { description: e?.message })
  }
  finally { sysCreating.value = false }
}

function confirmRestore(f: string) {
  restoreFile.value = f
  restoreConfirmVisible.value = true
}

async function doSysRestore() {
  sysRestoring.value = restoreFile.value
  toast.info(i18n.global.t('manage.backups.sysRestoreStarted'))
  try {
    await api.post('api/v1/system/snapshots/restore', { node: snapNode.value, file: restoreFile.value })
  }
  catch {}
  restoreConfirmVisible.value = false
  // 轮询 /health 自愈
  const poll = setInterval(() => {
    fetch('/health').then(r => r.json()).then(() => {
      clearInterval(poll)
      sysRestoring.value = ''
      toast.success(i18n.global.t('manage.backups.sysRestored'))
      loadSnaps()
      loadSysSnaps()
    }).catch(() => {})
  }, 4000)
}

async function deleteSysSnap(f: string) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('manage.backups.sysDeleteTitle'),
    content: i18n.global.t('manage.backups.sysDeleteConfirm', { name: f }),
    onConfirm: async () => {
      await api.delete(`api/v1/system/snapshots?node=${snapNode.value}&file=${encodeURIComponent(f)}`)
      toast.success(i18n.global.t('manage.saved'))
      await loadSysSnaps()
    },
  })
}

function downloadSysSnap(f: string) {
  const token = localStorage.getItem('token') || ''
  window.open(`api/v1/files/download?path=${encodeURIComponent('/opt/ypanel/backups/system/' + f)}&token=${encodeURIComponent(token)}`)
}

function loadSnaps() {
  loading.value = true
  try {
    api.get(`api/v1/snapshots?node=${snapNode.value}`, { silent: true }).then((res) => {
      backups.value = (res.data || []).map((x: any) => ({ name: x.file, sizeMb: x.sizeMb, modTime: x.modTime, path: x.file }))
    })
    loadSysSnaps()
  }
  finally {
    loading.value = false
  }
}

watch(snapNode, () => loadSnaps())

onMounted(() => {
  load()
  loadStorage()
  loadNodes()
  loadKeep()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="save" :size="24" />
          <span>{{ $t('manage.backups.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('manage.backups.desc') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="showHint">{{ $t('manage.backups.restoreHint') }}</FaButton>
        <label class="cursor-pointer rounded-md border px-3 py-1.5 text-sm transition-colors hover:bg-accent/50">
          {{ $t('manage.backups.importSnap') }}
          <input ref="importInputRef" type="file" class="hidden" accept=".db" @change="doImportSnap">
        </label>
        <FaButton size="sm" :loading="backupBusy" @click="doBackup">
          <YdMorphIcon name="save" :size="14" class="mr-1" /> {{ $t('manage.backups.backupNow') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- M47：系统级快照 -->
      <div class="mb-4 rounded-lg border p-4">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-sm font-medium">{{ $t('manage.backups.sysTitle') }}</span>
          <span class="text-xs text-muted-foreground">{{ $t('manage.backups.sysDesc') }}</span>
          <div class="ml-auto flex items-center gap-2">
            <label class="flex items-center gap-1 text-xs"><input v-model="sysOpts.websites" type="checkbox" class="accent-[var(--primary)]">{{ $t('manage.backups.sysWithWebsites') }}</label>
            <label class="flex items-center gap-1 text-xs"><input v-model="sysOpts.databases" type="checkbox" class="accent-[var(--primary)]">{{ $t('manage.backups.sysWithDatabases') }}</label>
            <FaButton size="sm" :loading="sysCreating" @click="createSysSnap">{{ $t('manage.backups.sysCreate') }}</FaButton>
          </div>
        </div>
        <div v-if="sysSnaps.length" class="mt-3 overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr><th class="px-3 py-2">{{ $t('manage.backups.file') }}</th><th class="w-24 px-3 py-2">{{ $t('common.size') }}</th><th class="w-44 px-3 py-2">{{ $t('manage.backups.createTime') }}</th><th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="f in sysSnaps" :key="f.file" class="border-t transition-colors hover:bg-accent/30">
                <td class="px-3 py-2 font-mono text-[13px]">{{ f.file }}</td>
                <td class="px-3 py-2 text-xs tabular-nums">{{ f.sizeMb.toFixed(1) }} MB</td>
                <td class="px-3 py-2 text-xs tabular-nums text-muted-foreground">{{ new Date(f.modTime).toLocaleString('zh-CN', { hour12: false }) }}</td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" @click="downloadSysSnap(f.file)">{{ $t('common.download') }}</FaButton>
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="confirmRestore(f.file)">{{ $t('manage.backups.restore') }}</FaButton>
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="deleteSysSnap(f.file)">{{ $t('common.delete') }}</FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="mt-2 text-xs text-muted-foreground">{{ $t('manage.backups.sysEmpty') }}</div>
      </div>

      <div class="mb-4 flex flex-wrap items-center gap-2 rounded-lg border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
        <span>{{ $t('manage.backups.node') }}</span>
        <YdSelect
          v-model="snapNode" size="sm" class="w-40"
          :options="nodesList.map(n => ({ label: n.name, value: n.id }))"
        />
        <span>{{ $t('manage.backups.keep') }}</span>
        <FaInput v-model="keepVal" type="number" class="w-20" />
        <FaButton variant="outline" size="sm" @click="saveKeep">{{ $t('common.save') }}</FaButton>
        <FaButton variant="outline" size="sm" :loading="pruneBusy" @click="pruneSnaps">{{ $t('manage.backups.prune') }}</FaButton>
      </div>
      <div v-if="storageAccounts.length" class="mb-4 flex flex-wrap items-center gap-2 rounded-lg border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
        <span>{{ $t('manage.backups.remoteUpload') }}</span>
        <YdSelect
          v-model="backupStorageId" size="sm" class="w-48"
          :options="[{ label: $t('manage.backups.localOnly'), value: 0 }, ...storageAccounts.map(a => ({ label: a.name, value: a.id }))]"
        />
        <template v-if="backupStorageId">
          <span>{{ $t('manage.backups.keep') }}</span>
          <FaInput v-model="backupKeep" type="number" class="w-20" :placeholder="$t('manage.backups.keepUnlimited')" />
        </template>
      </div>
      <div v-if="restoreHint" class="mb-4 rounded-lg border border-amber-300 bg-amber-50 p-4 text-xs whitespace-pre-line text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
        {{ restoreHint }}
      </div>

      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2">{{ $t('manage.backups.file') }}</th>
              <th class="hidden w-36 px-4 py-2 sm:table-cell">{{ $t('common.size') }}</th>
              <th class="hidden w-52 px-4 py-2 md:table-cell">{{ $t('manage.backups.createTime') }}</th>
              <th class="w-40 px-4 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !backups.length">
              <td colspan="4" class="px-4 py-10 text-center text-muted-foreground">
                {{ $t('common.loading') }}
              </td>
            </tr>
            <tr v-else-if="!backups.length">
              <td colspan="4" class="px-4 py-10 text-center text-muted-foreground">
                {{ $t('manage.backups.empty') }}
              </td>
            </tr>
            <tr v-for="b in backups" :key="b.name" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-4 py-2.5 font-mono text-[13px]">
                {{ b.name }}
              </td>
              <td class="hidden px-4 py-2.5 text-xs tabular-nums text-muted-foreground sm:table-cell">
                {{ b.sizeMb.toFixed(2) }} MB
              </td>
              <td class="hidden px-4 py-2.5 text-xs tabular-nums text-muted-foreground md:table-cell">
                {{ new Date(b.modTime).toLocaleString('zh-CN', { hour12: false }) }}
              </td>
              <td class="px-4 py-2.5 text-right">
                <FaButton variant="ghost" size="sm" @click="downloadBackup(b)">
                  {{ $t('common.download') }}
                </FaButton>
                <FaButton variant="ghost" size="sm" @click="openRestore(b)">{{ $t('manage.backups.restore') }}</FaButton>
                <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeBackup(b)">
                  {{ $t('common.delete') }}
                </FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <!-- M45：快照恢复确认 -->
    <FaModal v-model="restoreVisible" :title="$t('manage.backups.restoreTitle', { file: restoreTarget?.file || '' })" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="rounded-md border border-amber-300 bg-amber-50 p-3 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
          {{ $t('manage.backups.restoreFlow') }}
        </div>
        <pre v-if="restorePlan" class="max-h-48 overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs leading-relaxed">{{ restorePlan.script }}</pre>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="restoreVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton variant="outline" class="text-red-500!" :disabled="restoreExecuting" @click="doRollback">{{ $t('manage.backups.rollbackFirst') }}</FaButton>
        <FaButton :loading="restoreExecuting" @click="doRestore">{{ $t('manage.backups.confirmRestore') }}</FaButton>
      </template>
    </FaModal>

    <!-- M47：系统快照恢复确认 -->
    <FaModal v-model="restoreConfirmVisible" :title="$t('manage.backups.sysRestoreTitle', { file: restoreFile })" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="rounded-md border border-amber-300 bg-amber-50 p-3 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
          {{ $t('manage.backups.sysRestoreFlow') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="restoreConfirmVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="sysRestoring === restoreFile" @click="doSysRestore">{{ $t('manage.backups.confirmRestore') }}</FaButton>
      </template>
    </FaModal>
  </div>
</template>