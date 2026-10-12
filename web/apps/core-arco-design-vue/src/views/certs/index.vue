<script setup lang="ts">
import type { Certificate, DnsAccount, AcmeAccount } from '@/api/modules/cert'
import { certApi, parseAltDomains } from '@/api/modules/cert'
import YdDataTable from '@/components/YdDataTable/index.vue'
import type { YdDataTableColumn } from '@/components/YdDataTable/index.vue'
import siteApi from '@/api/modules/site'
import type { SiteItem } from '@/api/modules/site'
import { i18n, tr } from '@/locales'
import { useTaskJob } from '@/composables/useTaskJob'

// B23 证书管理页（对齐 1Panel「网站 → 证书」）：
// 申请（ACME/DNS 挑战）/ 上传 / 自签 / DNS 账户 / ACME 账户 + 证书全生命周期列表。

defineOptions({
  name: 'CertsIndex',
})

const toast = useFaToast()
const modal = useFaModal()

const certs = ref<Certificate[]>([])
const loading = ref(false)
const keyword = ref('')
const sortAsc = ref(false)

const dnsAccounts = ref<DnsAccount[]>([])
const acmeAccounts = ref<AcmeAccount[]>([])

async function load() {
  loading.value = true
  try {
    certs.value = await certApi.list()
  }
  catch (e: any) {
    toast.error(i18n.global.t('certs.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function loadAccounts() {
  dnsAccounts.value = await certApi.dnsAccounts().catch(() => [])
  acmeAccounts.value = await certApi.acmeAccounts().catch(() => [])
}

// DNS 账户下拉：无账户时保留原生的禁用占位项（value 0），有账户时仅列账户
const dnsAccountOptions = computed(() => [
  ...(!dnsAccounts.value.length ? [{ label: i18n.global.t('certs.noDnsOption'), value: 0, disabled: true }] : []),
  ...dnsAccounts.value.map(a => ({ label: `${a.name}（${tr(`certs.provider.${a.provider}`, a.provider)}）`, value: a.id })),
])

const filteredCerts = computed(() => {
  let list = [...certs.value]
  if (keyword.value.trim()) {
    const k = keyword.value.trim().toLowerCase()
    list = list.filter(c => c.domain.includes(k) || (c.remark || '').toLowerCase().includes(k) || (c.certName || '').includes(k))
  }
  list.sort((a, b) => {
    const va = a.notAfter ? new Date(a.notAfter).getTime() : 0
    const vb = b.notAfter ? new Date(b.notAfter).getTime() : 0
    return sortAsc.value ? va - vb : vb - va
  })
  return list
})

// ---- 状态展示 ----

const caLabel: Record<string, string> = { letsencrypt: "Let's Encrypt", zerossl: 'ZeroSSL', buypass: 'Buypass' }

function expiry(c: Certificate) {
  if (!c.notAfter)
    return { text: '—', cls: 'text-muted-foreground', days: 0 }
  const d = new Date(c.notAfter)
  const days = Math.floor((d.getTime() - Date.now()) / 86400000)
  const cls = days < 0 ? 'text-red-500' : days < 15 ? 'text-amber-500' : ''
  return { text: d.toISOString().slice(0, 10), cls, days }
}

const statusCls: Record<string, string> = {
  ok: 'bg-emerald-500/10 text-emerald-600',
  expiring: 'bg-amber-500/10 text-amber-600',
  expired: 'bg-red-500/10 text-red-600',
  error: 'bg-red-500/10 text-red-600',
}

// ---- 申请证书（ACME）----

const issueVisible = ref(false)
const issueForm = ref({ domain: '', altDomains: '', remark: '', autoRenew: true, acmeAccountId: 0, dnsAccountId: 0, deploySiteIds: [] as number[] })
const issueSites = ref<SiteItem[]>([])

function openIssue() {
  issueForm.value = { domain: '', altDomains: '', remark: '', autoRenew: true, acmeAccountId: acmeAccounts.value[0]?.id || 0, dnsAccountId: dnsAccounts.value[0]?.id || 0, deploySiteIds: [] }
  issueVisible.value = true
  siteApi.list().then((list: SiteItem[]) => {
    issueSites.value = list
  }).catch(() => {
    issueSites.value = []
  })
}

// ---- 列定义（M67 YdDataTable）：证书名左固定，操作右固定，中间列可拖拽换序 ----
const certColumns: YdDataTableColumn[] = [
  { key: 'id', label: 'ID', width: 60 },
  { key: 'domain', label: i18n.global.t('certs.domainCol'), width: 220, fixed: 'left' },
  { key: 'altDomains', label: i18n.global.t('certs.altDomainsCol') },
  { key: 'provider', label: i18n.global.t('certs.providerCol'), width: 150 },
  { key: 'status', label: i18n.global.t('common.status'), width: 90 },
  { key: 'issuer', label: i18n.global.t('certs.issuerCol'), width: 120 },
  { key: 'remark', label: i18n.global.t('common.remark'), hidden: 'lg' },
  { key: 'autoRenew', label: i18n.global.t('certs.autoRenewCol'), width: 90 },
  { key: 'expiry', label: i18n.global.t('certs.expiryCol'), width: 110, sortable: true },
  { key: 'actions', label: i18n.global.t('common.operation'), width: 260, fixed: 'right', align: 'right' },
]

// 申请走任务中心（M65）：秒回 taskId，签发过程 acme.sh 输出实时滚动
const { running: issueSaving, run: runIssueJob } = useTaskJob()

async function doIssue() {
  if (!issueForm.value.domain.trim()) {
    toast.warning(i18n.global.t('certs.requireMainDomain'))
    return
  }
  if (!issueForm.value.dnsAccountId) {
    toast.warning(i18n.global.t('certs.requireDnsAccount'))
    return
  }
  const out = await runIssueJob(() => certApi.issue({ ...issueForm.value, deploySiteIds: issueForm.value.deploySiteIds.length ? issueForm.value.deploySiteIds : undefined }), {
    onSuccess: async () => {
      toast.success(i18n.global.t('certs.issued'))
      await load()
    },
  })
  if (out) {
    issueVisible.value = false
    toast.info(i18n.global.t('common.taskSubmitted'))
  }
}

// ---- 上传证书 ----

const uploadVisible = ref(false)
const uploadSaving = ref(false)
const uploadForm = ref({ certName: '', domain: '', remark: '', certPem: '', keyPem: '' })

function openUpload() {
  uploadForm.value = { certName: '', domain: '', remark: '', certPem: '', keyPem: '' }
  uploadVisible.value = true
}

async function doUpload() {
  if (!uploadForm.value.domain.trim() || !uploadForm.value.certPem.trim() || !uploadForm.value.keyPem.trim()) {
    toast.warning(i18n.global.t('certs.requireUploadFields'))
    return
  }
  uploadSaving.value = true
  try {
    await certApi.upload(uploadForm.value)
    toast.success(i18n.global.t('certs.uploaded'))
    uploadVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('certs.uploadFailed'), { description: e?.message })
  }
  finally {
    uploadSaving.value = false
  }
}

// ---- 自签证书 ----

const selfVisible = ref(false)
const selfSaving = ref(false)
const selfForm = ref({ domain: '', remark: '', days: 365 })

function openSelf() {
  selfForm.value = { domain: '', remark: '', days: 365 }
  selfVisible.value = true
}

async function doSelf() {
  if (!selfForm.value.domain.trim()) {
    toast.warning(i18n.global.t('certs.requireDomain'))
    return
  }
  selfSaving.value = true
  try {
    await certApi.selfSigned(selfForm.value)
    toast.success(i18n.global.t('certs.selfGenerated'))
    selfVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('certs.selfFailed'), { description: e?.message })
  }
  finally {
    selfSaving.value = false
  }
}

// ---- DNS 账户管理 ----

const dnsVisible = ref(false)
const dnsSaving = ref(false)
const dnsForm = ref({ name: '', provider: 'aliyun', accessKey: '', secret: '' })

function openDns() {
  loadAccounts()
  dnsForm.value = { name: '', provider: 'aliyun', accessKey: '', secret: '' }
  dnsVisible.value = true
}

async function doCreateDns() {
  if (!dnsForm.value.name.trim() || !dnsForm.value.accessKey.trim() || !dnsForm.value.secret.trim()) {
    toast.warning(i18n.global.t('certs.requireDnsFields'))
    return
  }
  dnsSaving.value = true
  try {
    await certApi.createDnsAccount(dnsForm.value)
    toast.success(i18n.global.t('certs.dnsCreated'))
    dnsForm.value = { name: '', provider: 'aliyun', accessKey: '', secret: '' }
    await loadAccounts()
  }
  catch (e: any) {
    toast.error(i18n.global.t('certs.createFailed'), { description: e?.message })
  }
  finally {
    dnsSaving.value = false
  }
}

function removeDns(a: DnsAccount) {
  modal.confirm({
    title: i18n.global.t('certs.deleteDnsTitle'),
    content: i18n.global.t('certs.deleteDnsConfirm', { name: a.name }),
    onConfirm: async () => {
      try {
        await certApi.removeDnsAccount(a.id)
        await loadAccounts()
      }
      catch (e: any) {
        toast.error(i18n.global.t('certs.deleteFailed'), { description: e?.message })
      }
    },
  })
}

// ---- ACME 账户管理 ----

const acmeVisible = ref(false)
const acmeForm = ref({ email: '', caType: 'letsencrypt', keyType: 'ec-256' })

function openAcme() {
  loadAccounts()
  acmeForm.value = { email: '', caType: 'letsencrypt', keyType: 'ec-256' }
  acmeVisible.value = true
}

// ACME 注册走任务中心（M65）：注册输出实时可查，失败任务标红
const { running: acmeSaving, run: runAcmeJob } = useTaskJob()

async function doCreateAcme() {
  if (!acmeForm.value.email.includes('@')) {
    toast.warning(i18n.global.t('certs.requireEmail'))
    return
  }
  const out = await runAcmeJob(() => certApi.createAcmeAccount(acmeForm.value), {
    onSuccess: async () => {
      toast.success(i18n.global.t('certs.acmeCreated'))
      await loadAccounts()
    },
  })
  if (out) {
    acmeForm.value = { email: '', caType: 'letsencrypt', keyType: 'ec-256' }
    toast.info(i18n.global.t('common.taskSubmitted'))
  }
}

function removeAcme(a: AcmeAccount) {
  modal.confirm({
    title: i18n.global.t('certs.deleteAcmeTitle'),
    content: i18n.global.t('certs.deleteAcmeConfirm', { email: a.email, ca: caLabel[a.caType] }),
    onConfirm: async () => {
      try {
        await certApi.removeAcmeAccount(a.id)
        await loadAccounts()
      }
      catch (e: any) {
        toast.error(i18n.global.t('certs.deleteFailed'), { description: e?.message })
      }
    },
  })
}

// ---- 证书操作 ----

// 续签走任务中心（M65）：同一证书防重复提交（后端同 ref 单飞）
const { running: renewing, run: runRenewJob } = useTaskJob()

function renew(c: Certificate) {
  runRenewJob(() => certApi.renew(c.id), {
    onSuccess: async () => {
      toast.success(i18n.global.t('certs.renewed'))
      await load()
    },
  })
}

function removeCert(c: Certificate) {
  modal.confirm({
    title: i18n.global.t('certs.deleteTitle'),
    content: i18n.global.t('certs.deleteConfirm', { domain: c.domain, name: c.certName }),
    onConfirm: async () => {
      try {
        await certApi.remove(c.id)
        toast.success(i18n.global.t('certs.deleted'))
        await load()
      }
      catch (e: any) {
        toast.error(i18n.global.t('certs.deleteFailed'), { description: e?.message })
      }
    },
  })
}

function toggleAutoRenew(c: Certificate) {
  certApi.update(c.id, { autoRenew: !c.autoRenew }).then(() => {
    c.autoRenew = !c.autoRenew
  }).catch((e: any) => {
    toast.error(i18n.global.t('certs.updateFailed'), { description: e?.message })
  })
}

// 编辑（备注 + 自动续签）
const editVisible = ref(false)
const editSaving = ref(false)
const editForm = ref({ id: 0, domain: '', remark: '', autoRenew: true })

function openEdit(c: Certificate) {
  editForm.value = { id: c.id, domain: c.domain, remark: c.remark || '', autoRenew: c.autoRenew }
  editVisible.value = true
}

async function doEdit() {
  editSaving.value = true
  try {
    await certApi.update(editForm.value.id, { remark: editForm.value.remark, autoRenew: editForm.value.autoRenew })
    toast.success(i18n.global.t('certs.saved'))
    editVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('certs.saveFailed'), { description: e?.message })
  }
  finally {
    editSaving.value = false
  }
}

// 详情
const detailVisible = ref(false)
const detailLoading = ref(false)
const detailData = ref<{ cert: Certificate, text: string }>()

async function openDetail(c: Certificate) {
  detailVisible.value = true
  detailLoading.value = true
  try {
    detailData.value = await certApi.detail(c.id)
  }
  catch (e: any) {
    toast.error(i18n.global.t('certs.detailLoadFailed'), { description: e?.message })
  }
  finally {
    detailLoading.value = false
  }
}

onMounted(() => {
  load()
  loadAccounts()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="shield" :size="24" />
          <span>{{ $t('certs.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('certs.desc') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton size="sm" @click="openIssue">
          <FaIcon name="i-lucide:badge-check" class="mr-1" /> {{ $t('certs.issue') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openUpload">
          <FaIcon name="i-lucide:upload" class="mr-1" /> {{ $t('certs.upload') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openSelf">
          <FaIcon name="i-lucide:pen-tool" class="mr-1" /> {{ $t('certs.selfSigned') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openAcme">
          <FaIcon name="i-lucide:user-round" class="mr-1" /> {{ $t('certs.acmeAccounts') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openDns">
          <FaIcon name="i-lucide:cloud" class="mr-1" /> {{ $t('certs.dnsAccounts') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="mb-3 flex items-center gap-2">
        <FaInput v-model="keyword" :placeholder="$t('certs.searchPlaceholder')" class="w-56!" />
        <span class="ml-auto text-xs text-muted-foreground">{{ $t('common.total', { n: filteredCerts.length }) }}</span>
      </div>

      <YdDataTable
        :columns="certColumns"
        :rows="filteredCerts"
        storage-key="certs"
        :loading="loading && !certs.length"
        :empty-text="$t('certs.noCerts')"
        sort-key="expiry"
        :sort-order="sortAsc ? 'asc' : 'desc'"
        @sort="sortAsc = !sortAsc"
      >
        <template #cell-id="{ item: c }">
          <span class="text-xs text-muted-foreground">{{ c.id }}</span>
        </template>
        <template #cell-domain="{ item: c }">
          <div class="font-medium">{{ c.domain }}</div>
          <div v-if="c.sites?.length" class="text-xs text-muted-foreground">{{ $t('certs.boundTo', { sites: c.sites.join('、') }) }}</div>
        </template>
        <template #cell-altDomains="{ item: c }">
          <span class="block max-w-48 truncate font-mono text-xs text-muted-foreground" :title="parseAltDomains(c.altDomains).join(', ')">
            {{ parseAltDomains(c.altDomains).join(', ') || '—' }}
          </span>
        </template>
        <template #cell-provider="{ item: c }">
          <span class="text-xs">
            {{ tr(`certs.providerLabel.${c.provider}`, c.provider) }}
            <span v-if="c.provider === 'acme'" class="text-muted-foreground">
              {{ $t('certs.providerHint', { via: c.dnsAccountId ? $t('certs.viaDns') : $t('certs.viaEnv') }) }}
            </span>
          </span>
        </template>
        <template #cell-status="{ item: c }">
          <span class="rounded-full px-2 py-0.5 text-xs" :class="statusCls[c.status] || 'bg-muted text-muted-foreground'">
            {{ tr(`certs.status.${c.status}`, c.status) }}
          </span>
        </template>
        <template #cell-issuer="{ item: c }">
          <span class="text-xs text-muted-foreground">{{ c.issuer || '—' }}</span>
        </template>
        <template #cell-remark="{ item: c }">
          <span class="block max-w-32 truncate text-xs" :title="c.remark">{{ c.remark || '—' }}</span>
        </template>
        <template #cell-autoRenew="{ item: c }">
          <label v-if="c.provider === 'acme'" class="inline-flex cursor-pointer items-center">
            <input type="checkbox" :checked="c.autoRenew" @change="toggleAutoRenew(c)">
          </label>
          <span v-else class="text-xs text-muted-foreground">—</span>
        </template>
        <template #cell-expiry="{ item: c }">
          <span class="text-xs" :class="expiry(c).cls">{{ expiry(c).text }}</span>
        </template>
        <template #cell-actions="{ item: c }">
          <div class="flex items-center justify-end gap-1">
            <FaButton variant="ghost" size="sm" @click="openDetail(c)">{{ $t('common.detail') }}</FaButton>
            <FaButton v-if="c.provider === 'acme'" variant="ghost" size="sm" :loading="renewing" @click="renew(c)">{{ $t('certs.renew') }}</FaButton>
            <FaButton variant="ghost" size="sm" @click="openEdit(c)">{{ $t('common.edit') }}</FaButton>
            <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeCert(c)">{{ $t('common.delete') }}</FaButton>
          </div>
        </template>
      </YdDataTable>
    </FaPageMain>

    <!-- 申请证书 -->
    <FaModal v-model="issueVisible" :title="$t('certs.issue')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.mainDomain') }}</span>
          <FaInput v-model="issueForm.domain" :placeholder="$t('certs.mainDomainPlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-start gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.altDomainsCol') }}</span>
          <textarea v-model="issueForm.altDomains" class="h-14 flex-1 rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" :placeholder="$t('certs.altDomainsPlaceholder')" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.acmeAccounts') }}</span>
          <YdSelect v-model="issueForm.acmeAccountId" button-class="flex-1" :options="[{ label: $t('certs.defaultAcme'), value: 0 }, ...acmeAccounts.map(a => ({ label: `${a.email}（${caLabel[a.caType]}）`, value: a.id }))]" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.dnsAccounts') }}</span>
          <YdSelect v-model="issueForm.dnsAccountId" button-class="flex-1" :options="dnsAccountOptions" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="issueForm.remark" :placeholder="$t('certs.issueRemarkPlaceholder')" class="flex-1" />
        </div>
        <div class="flex flex-col gap-1">
          <span class="text-sm">{{ $t('certs.deploySites') }}<span class="ml-1 text-xs text-muted-foreground">{{ $t('certs.deploySitesHint') }}</span></span>
          <div v-if="issueSites.length" class="max-h-36 overflow-y-auto rounded-md border border-input p-2">
            <label v-for="st in issueSites" :key="st.id" class="flex cursor-pointer items-center gap-2 rounded px-1 py-0.5 text-sm hover:bg-muted/60">
              <input v-model="issueForm.deploySiteIds" type="checkbox" :value="st.id">
              <span class="truncate">{{ st.name }}<span class="ml-1 font-mono text-xs text-muted-foreground">{{ st.domain }}</span></span>
              <span v-if="st.certDomain" class="ml-auto shrink-0 text-xs text-muted-foreground">{{ $t('certs.deployCurrent') }}{{ st.certDomain }}</span>
            </label>
          </div>
          <div v-else class="rounded-md bg-muted/50 p-2 text-xs text-muted-foreground">{{ $t('certs.noSitesHint') }}</div>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="issueForm.autoRenew" type="checkbox"> {{ $t('certs.autoRenew') }}
          <span class="text-xs text-muted-foreground">{{ $t('certs.autoRenewHint') }}</span>
        </label>
        <div class="rounded-md bg-muted/50 p-3 text-xs text-muted-foreground">
          {{ $t('certs.dnsApiHint') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="issueVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="issueSaving" @click="doIssue">{{ $t('certs.confirmIssue') }}</FaButton>
      </template>
    </FaModal>

    <!-- 上传证书 -->
    <FaModal v-model="uploadVisible" :title="$t('certs.upload')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.certName') }}</span>
          <FaInput v-model="uploadForm.certName" :placeholder="$t('certs.certNamePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.mainDomain') }}</span>
          <FaInput v-model="uploadForm.domain" :placeholder="$t('certs.domainPlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-start gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.certPem') }}</span>
          <textarea v-model="uploadForm.certPem" class="h-24 flex-1 rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" :placeholder="$t('certs.certPemPlaceholder')" />
        </div>
        <div class="flex items-start gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.keyPem') }}</span>
          <textarea v-model="uploadForm.keyPem" class="h-24 flex-1 rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" :placeholder="$t('certs.keyPemPlaceholder')" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="uploadForm.remark" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="uploadVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="uploadSaving" @click="doUpload">{{ $t('common.upload') }}</FaButton>
      </template>
    </FaModal>

    <!-- 自签证书 -->
    <FaModal v-model="selfVisible" :title="$t('certs.selfSigned')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.domainCol') }}</span>
          <FaInput v-model="selfForm.domain" :placeholder="$t('certs.selfDomainPlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.validity') }}</span>
          <FaInput v-model.number="selfForm.days" class="w-32!" />
          <span class="text-xs text-muted-foreground">{{ $t('certs.days') }}</span>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="selfForm.remark" class="flex-1" />
        </div>
        <div class="rounded-md bg-muted/50 p-3 text-xs text-muted-foreground">
          {{ $t('certs.selfHint') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="selfVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="selfSaving" @click="doSelf">{{ $t('certs.generate') }}</FaButton>
      </template>
    </FaModal>

    <!-- DNS 账户 -->
    <FaModal v-model="dnsVisible" :title="$t('certs.dnsAccounts')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('common.name') }}</th>
                <th class="px-3 py-2">{{ $t('common.type') }}</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!dnsAccounts.length">
                <td colspan="3" class="px-3 py-6 text-center text-xs text-muted-foreground">{{ $t('certs.noDnsAccounts') }}</td>
              </tr>
              <tr v-for="a in dnsAccounts" :key="a.id" class="border-t">
                <td class="px-3 py-2">{{ a.name }}</td>
                <td class="px-3 py-2 text-xs text-muted-foreground">{{ tr(`certs.provider.${a.provider}`, a.provider) }}</td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeDns(a)">{{ $t('common.delete') }}</FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="rounded-md border p-3">
          <div class="mb-2 text-sm font-medium">{{ $t('certs.createDnsAccount') }}</div>
          <div class="flex flex-col gap-2">
            <div class="flex items-center gap-2">
              <FaInput v-model="dnsForm.name" :placeholder="$t('certs.dnsNamePlaceholder')" class="flex-1" />
              <YdSelect v-model="dnsForm.provider" :options="[{ label: $t('certs.provider.aliyun'), value: 'aliyun' }, { label: $t('certs.provider.dnspod'), value: 'dnspod' }, { label: $t('certs.provider.cloudflare'), value: 'cloudflare' }]" />
            </div>
            <FaInput v-model="dnsForm.accessKey" :placeholder="$t('certs.accessKeyPlaceholder')" />
            <FaInput v-model="dnsForm.secret" placeholder="AccessKey Secret / SecretKey / Global API Key" type="password" />
            <div class="flex justify-end">
              <FaButton size="sm" :loading="dnsSaving" @click="doCreateDns">{{ $t('common.create') }}</FaButton>
            </div>
          </div>
        </div>
        <div class="text-xs text-muted-foreground">{{ $t('certs.credHint') }}</div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="dnsVisible = false">{{ $t('common.close') }}</FaButton>
      </template>
    </FaModal>

    <!-- Acme 账户 -->
    <FaModal v-model="acmeVisible" :title="$t('certs.acmeAccounts')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('certs.emailCol') }}</th>
                <th class="px-3 py-2">CA</th>
                <th class="px-3 py-2">{{ $t('certs.keyAlgCol') }}</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!acmeAccounts.length">
                <td colspan="4" class="px-3 py-6 text-center text-xs text-muted-foreground">{{ $t('certs.noAcmeAccounts') }}</td>
              </tr>
              <tr v-for="a in acmeAccounts" :key="a.id" class="border-t">
                <td class="px-3 py-2">{{ a.email }}</td>
                <td class="px-3 py-2 text-xs text-muted-foreground">{{ caLabel[a.caType] || a.caType }}</td>
                <td class="px-3 py-2 font-mono text-xs text-muted-foreground">{{ a.keyType }}</td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeAcme(a)">{{ $t('common.delete') }}</FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="rounded-md border p-3">
          <div class="mb-2 text-sm font-medium">{{ $t('certs.createAcmeAccount') }}</div>
          <div class="flex flex-col gap-2">
            <FaInput v-model="acmeForm.email" :placeholder="$t('certs.emailPlaceholder')" />
            <div class="flex items-center gap-2">
              <YdSelect v-model="acmeForm.caType" button-class="flex-1" :options="[{ label: `Let's Encrypt`, value: 'letsencrypt' }, 'zerossl', 'buypass']" />
              <YdSelect v-model="acmeForm.keyType" button-class="flex-1" :options="[{ label: 'EC 256', value: 'ec-256' }, { label: 'EC 384', value: 'ec-384' }, { label: 'RSA 2048', value: 'rsa-2048' }, { label: 'RSA 4096', value: 'rsa-4096' }]" />
              <FaButton size="sm" :loading="acmeSaving" @click="doCreateAcme">{{ $t('common.create') }}</FaButton>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="acmeVisible = false">{{ $t('common.close') }}</FaButton>
      </template>
    </FaModal>

    <!-- 编辑证书 -->
    <FaModal v-model="editVisible" :title="$t('certs.editTitle')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.domainCol') }}</span>
          <span class="flex-1 font-mono text-sm">{{ editForm.domain }}</span>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="editForm.remark" class="flex-1" />
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="editForm.autoRenew" type="checkbox"> {{ $t('certs.autoRenew') }}
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="editVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="editSaving" @click="doEdit">{{ $t('common.save') }}</FaButton>
      </template>
    </FaModal>

    <!-- 证书详情 -->
    <FaModal v-model="detailVisible" :title="$t('certs.detailTitle')" :destroy-on-close="true">
      <div v-if="detailLoading" class="py-8 text-center text-sm text-muted-foreground">{{ $t('common.loading') }}</div>
      <div v-else-if="detailData" class="space-y-3">
        <div class="rounded-md border p-3 text-sm">
          <div class="grid grid-cols-[5rem_1fr] gap-y-2 text-xs">
            <span class="text-muted-foreground">{{ $t('certs.mainDomain') }}</span><span class="font-mono">{{ detailData.cert.domain }}</span>
            <span class="text-muted-foreground">{{ $t('certs.certName') }}</span><span class="font-mono">{{ detailData.cert.certName }}</span>
            <span class="text-muted-foreground">{{ $t('certs.issuerCol') }}</span><span>{{ detailData.cert.issuer || '—' }}</span>
            <span class="text-muted-foreground">{{ $t('certs.expiryCol') }}</span><span>{{ detailData.cert.notAfter ? new Date(detailData.cert.notAfter).toLocaleString() : '—' }}</span>
            <span class="text-muted-foreground">{{ $t('certs.boundSites') }}</span><span>{{ detailData.cert.sites?.join('、') || $t('certs.notBound') }}</span>
          </div>
        </div>
        <pre class="max-h-72 overflow-auto rounded-md bg-muted/50 p-3 font-mono text-[11px] leading-relaxed">{{ detailData.text || $t('certs.noCertFile') }}</pre>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="detailVisible = false">{{ $t('common.close') }}</FaButton>
      </template>
    </FaModal>
  </div>
</template>
