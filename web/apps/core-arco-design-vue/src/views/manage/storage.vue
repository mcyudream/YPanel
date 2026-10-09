<script setup lang="ts">
import type { StorageAccount, StorageObject } from '@/api/modules/storage'
import { storageApi } from '@/api/modules/storage'
import { i18n } from '@/locales'

defineOptions({
  name: 'ManageStorage',
})

const toast = useFaToast()
const modal = useFaModal()

const accounts = ref<StorageAccount[]>([])
const loading = ref(false)
const testingId = ref(0)

// 编辑/新建弹窗
const dialogVisible = ref(false)
const editing = ref<StorageAccount | null>(null)
const saving = ref(false)

type StorageAccountInput = {
  name: string
  type: 's3' | 'webdav' | 'sftp'
  endpoint: string
  region: string
  bucket: string
  accessKey: string
  secret: string
  backupPath: string
  useSSL: boolean
  remark: string
}

const form = ref<StorageAccountInput>(emptyForm())

// 远端对象浏览
const browseAccount = ref<StorageAccount | null>(null)
const browseVisible = ref(false)
const browseCategory = ref('')
const browseObjects = ref<StorageObject[]>([])
const browseLoading = ref(false)
const fetchingKey = ref('')

function emptyForm(): StorageAccountInput {
  return {
    name: '', type: 's3', endpoint: '', region: '', bucket: '', accessKey: '',
    secret: '', backupPath: 'ypanel-backups', useSSL: true, remark: '',
  }
}

async function load() {
  loading.value = true
  try {
    accounts.value = await storageApi.list()
  }
  catch (e: any) {
    toast.error(i18n.global.t('storage.loadFail'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  form.value = emptyForm()
  dialogVisible.value = true
}

function openEdit(a: StorageAccount) {
  editing.value = a
  form.value = {
    name: a.name, type: a.type, endpoint: a.endpoint, region: a.region, bucket: a.bucket,
    accessKey: a.accessKey, secret: '', backupPath: a.backupPath, useSSL: a.useSSL, remark: a.remark,
  }
  dialogVisible.value = true
}

async function save() {
  saving.value = true
  try {
    if (editing.value) {
      await storageApi.update(editing.value.id, form.value)
      toast.success(i18n.global.t('storage.saved'))
    }
    else {
      await storageApi.create(form.value)
      toast.success(i18n.global.t('storage.created'))
    }
    dialogVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('storage.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

async function testConn(a: StorageAccount) {
  testingId.value = a.id
  try {
    const out = await storageApi.test(a.id)
    toast.success(i18n.global.t('storage.testOk', { ms: out.latencyMs }))
  }
  catch (e: any) {
    toast.error(i18n.global.t('storage.testFail'), { description: e?.message })
  }
  finally {
    testingId.value = 0
  }
}

function removeAccount(a: StorageAccount) {
  modal.confirm({
    title: i18n.global.t('storage.removeTitle'),
    content: i18n.global.t('storage.removeConfirm', { name: a.name }),
    onConfirm: async () => {
      await storageApi.remove(a.id)
      toast.success(i18n.global.t('storage.deleted'))
      await load()
    },
  })
}

async function openBrowse(a: StorageAccount) {
  browseAccount.value = a
  browseVisible.value = true
  await loadObjects()
}

async function loadObjects() {
  if (!browseAccount.value)
    return
  browseLoading.value = true
  try {
    browseObjects.value = await storageApi.objects(browseAccount.value.id, browseCategory.value.trim())
  }
  catch (e: any) {
    toast.error(i18n.global.t('storage.browseFail'), { description: e?.message })
  }
  finally {
    browseLoading.value = false
  }
}

async function fetchObject(o: StorageObject) {
  if (!browseAccount.value)
    return
  fetchingKey.value = o.name
  try {
    const out = await storageApi.fetch(browseAccount.value.id, o.name, '/opt/ypanel/backups/restore')
    toast.success(i18n.global.t('storage.fetched', { path: out.path }))
  }
  catch (e: any) {
    toast.error(i18n.global.t('storage.fetchFail'), { description: e?.message })
  }
  finally {
    fetchingKey.value = ''
  }
}

function removeObject(o: StorageObject) {
  if (!browseAccount.value)
    return
  const acct = browseAccount.value
  modal.confirm({
    title: i18n.global.t('storage.objRemoveTitle'),
    content: i18n.global.t('storage.objRemoveConfirm', { name: o.name }),
    onConfirm: async () => {
      await storageApi.deleteObject(acct.id, o.name)
      toast.success(i18n.global.t('storage.deleted'))
      await loadObjects()
    },
  })
}

const typeLabels = computed<Record<string, string>>(() => ({
  s3: i18n.global.t('storage.typeS3'),
  webdav: i18n.global.t('storage.typeWebdav'),
  sftp: i18n.global.t('storage.typeSftp'),
}))

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="save" :size="24" />
          <span>{{ $t('storage.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('storage.desc') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton size="sm" @click="openCreate">
          <YdMorphIcon name="save" :size="14" class="mr-1" /> {{ $t('storage.create') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2">{{ $t('common.name') }}</th>
              <th class="hidden w-28 px-4 py-2 sm:table-cell">{{ $t('common.type') }}</th>
              <th class="hidden px-4 py-2 md:table-cell">{{ $t('storage.endpoint') }}</th>
              <th class="hidden w-44 px-4 py-2 lg:table-cell">{{ $t('storage.pathHeader') }}</th>
              <th class="w-52 px-4 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !accounts.length">
              <td colspan="5" class="px-4 py-10 text-center text-muted-foreground">
                {{ $t('common.loading') }}
              </td>
            </tr>
            <tr v-else-if="!accounts.length">
              <td colspan="5" class="px-4 py-10 text-center text-muted-foreground">
                {{ $t('storage.empty') }}
              </td>
            </tr>
            <tr v-for="a in accounts" :key="a.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-4 py-2.5">
                <div class="font-medium">{{ a.name }}</div>
                <div v-if="a.remark" class="text-xs text-muted-foreground">
                  {{ a.remark }}
                </div>
              </td>
              <td class="hidden px-4 py-2.5 text-xs text-muted-foreground sm:table-cell">
                {{ typeLabels[a.type] || a.type }}
              </td>
              <td class="hidden px-4 py-2.5 font-mono text-xs text-muted-foreground md:table-cell">
                {{ a.type === 'webdav' ? a.endpoint : `${a.useSSL ? 'https' : 'http'}://${a.endpoint}` }}
                <span v-if="a.bucket"> / {{ a.bucket }}</span>
              </td>
              <td class="hidden px-4 py-2.5 font-mono text-xs text-muted-foreground lg:table-cell">
                {{ a.backupPath }}
              </td>
              <td class="px-4 py-2.5 text-right">
                <FaButton variant="ghost" size="sm" :loading="testingId === a.id" @click="testConn(a)">
                  {{ $t('common.test') }}
                </FaButton>
                <FaButton variant="ghost" size="sm" @click="openBrowse(a)">
                  {{ $t('storage.browse') }}
                </FaButton>
                <FaButton variant="ghost" size="sm" @click="openEdit(a)">
                  {{ $t('common.edit') }}
                </FaButton>
                <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeAccount(a)">
                  {{ $t('common.delete') }}
                </FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <!-- 新建/编辑账号 -->
    <FaModal v-model="dialogVisible" :title="editing ? $t('storage.editTitle', { name: editing.name }) : $t('storage.create')" :destroy-on-close="true">
      <div class="space-y-3">
        <div class="grid grid-cols-2 gap-3">
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('common.name') }}</span>
            <FaInput v-model="form.name" :placeholder="$t('storage.namePlaceholder')" class="w-full" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('common.type') }}</span>
            <YdSelect
              v-model="form.type" class="w-full" size="default"
              :options="[{ label: $t('storage.typeS3Full'), value: 's3' }, { label: $t('storage.typeWebdav'), value: 'webdav' }, { label: $t('storage.typeSftp'), value: 'sftp' }]"
            />
          </label>
        </div>
        <label class="block space-y-1">
          <span class="text-xs text-muted-foreground">
            {{ form.type === 'webdav' ? $t('storage.endpointWebdav') : form.type === 'sftp' ? $t('storage.endpointSftp') : $t('storage.endpointS3') }}
          </span>
          <FaInput v-model="form.endpoint" :placeholder="form.type === 's3' ? 'minio.example.com:9000' : form.type === 'sftp' ? '192.168.1.10:22' : 'https://nas.local:5006/dav'" class="w-full" />
        </label>
        <div v-if="form.type === 's3'" class="grid grid-cols-2 gap-3">
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('storage.bucket') }}</span>
            <FaInput v-model="form.bucket" placeholder="ypanel" class="w-full" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('storage.region') }}</span>
            <FaInput v-model="form.region" placeholder="us-east-1 / oss-cn-hangzhou" class="w-full" />
          </label>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ form.type === 's3' ? $t('storage.accessKey') : $t('storage.username') }}</span>
            <FaInput v-model="form.accessKey" class="w-full" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ form.type === 's3' ? $t('storage.secretKey') : form.type === 'sftp' ? $t('storage.secretSftp') : $t('storage.password') }}{{ editing ? $t('storage.keepSecretHint') : '' }}</span>
            <FaInput v-model="form.secret" type="password" class="w-full" :placeholder="editing ? $t('storage.keepSecretPlaceholder') : ''" />
          </label>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('storage.backupPath') }}</span>
            <FaInput v-model="form.backupPath" placeholder="ypanel-backups" class="w-full" />
          </label>
          <label v-if="form.type === 's3'" class="flex items-end gap-2 pb-2">
            <input v-model="form.useSSL" type="checkbox" class="accent-[var(--primary)]">
            <span class="text-xs text-muted-foreground">{{ $t('storage.useSSL') }}</span>
          </label>
        </div>
        <label class="block space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="form.remark" class="w-full" />
        </label>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <FaButton variant="outline" size="sm" @click="dialogVisible = false">
            {{ $t('common.cancel') }}
          </FaButton>
          <FaButton size="sm" :loading="saving" @click="save">
            {{ $t('common.save') }}
          </FaButton>
        </div>
      </template>
    </FaModal>

    <!-- 远端对象浏览 -->
    <FaModal v-model="browseVisible" :title="$t('storage.browseTitle', { name: browseAccount?.name || '' })" :destroy-on-close="true">
      <div class="space-y-3">
        <div class="flex items-center gap-2">
          <FaInput v-model="browseCategory" :placeholder="$t('storage.categoryPlaceholder')" class="flex-1" @keyup.enter="loadObjects" />
          <FaButton variant="outline" size="sm" :loading="browseLoading" @click="loadObjects">
            {{ $t('common.refresh') }}
          </FaButton>
        </div>
        <div class="max-h-96 overflow-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('storage.objectName') }}</th>
                <th class="hidden w-24 px-3 py-2 sm:table-cell">{{ $t('common.size') }}</th>
                <th class="hidden w-44 px-3 py-2 md:table-cell">{{ $t('common.time') }}</th>
                <th class="w-32 px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="browseLoading && !browseObjects.length">
                <td colspan="4" class="px-3 py-8 text-center text-muted-foreground">
                  {{ $t('common.loading') }}
                </td>
              </tr>
              <tr v-else-if="!browseObjects.length">
                <td colspan="4" class="px-3 py-8 text-center text-muted-foreground">
                  {{ $t('storage.objEmpty') }}
                </td>
              </tr>
              <tr v-for="o in browseObjects" :key="o.name" class="border-t transition-colors hover:bg-accent/30">
                <td class="px-3 py-2 font-mono text-xs break-all">
                  {{ o.name }}
                </td>
                <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground sm:table-cell">
                  {{ o.sizeMb.toFixed(2) }} MB
                </td>
                <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground md:table-cell">
                  {{ new Date(o.modTime).toLocaleString('zh-CN', { hour12: false }) }}
                </td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" :loading="fetchingKey === o.name" @click="fetchObject(o)">
                    {{ $t('storage.fetch') }}
                  </FaButton>
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeObject(o)">
                    {{ $t('common.delete') }}
                  </FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="text-xs text-muted-foreground">
          {{ $t('storage.fetchHint') }}
        </p>
      </div>
    </FaModal>
  </div>
</template>
