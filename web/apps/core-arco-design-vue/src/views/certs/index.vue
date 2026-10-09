<script setup lang="ts">
import type { Certificate, DnsAccount, AcmeAccount } from '@/api/modules/cert'
import { certApi } from '@/api/modules/cert'
import { i18n, tr } from '@/locales'

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
const issueSaving = ref(false)
const issueForm = ref({ domain: '', altDomains: '', remark: '', autoRenew: true, acmeAccountId: 0, dnsAccountId: 0 })

function openIssue() {
  issueForm.value = { domain: '', altDomains: '', remark: '', autoRenew: true, acmeAccountId: acmeAccounts.value[0]?.id || 0, dnsAccountId: dnsAccounts.value[0]?.id || 0 }
  issueVisible.value = true
}

async function doIssue() {
  if (!issueForm.value.domain.trim()) {
    toast.warning(i18n.global.t('certs.requireMainDomain'))
    return
  }
  if (!issueForm.value.dnsAccountId) {
    toast.warning(i18n.global.t('certs.requireDnsAccount'))
    return
  }
  issueSaving.value = true
  toast.info(i18n.global.t('certs.issuing'))
  try {
    await certApi.issue(issueForm.value)
    toast.success(i18n.global.t('certs.issued'))
    issueVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('certs.issueFailed'), { description: e?.message })
  }
  finally {
    issueSaving.value = false
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
const acmeSaving = ref(false)
const acmeForm = ref({ email: '', caType: 'letsencrypt', keyType: 'ec-256' })

function openAcme() {
  loadAccounts()
  acmeForm.value = { email: '', caType: 'letsencrypt', keyType: 'ec-256' }
  acmeVisible.value = true
}

async function doCreateAcme() {
  if (!acmeForm.value.email.includes('@')) {
    toast.warning(i18n.global.t('certs.requireEmail'))
    return
  }
  acmeSaving.value = true
  try {
    await certApi.createAcmeAccount(acmeForm.value)
    toast.success(i18n.global.t('certs.acmeCreated'))
    acmeForm.value = { email: '', caType: 'letsencrypt', keyType: 'ec-256' }
    await loadAccounts()
  }
  catch (e: any) {
    toast.error(i18n.global.t('certs.createFailed'), { description: e?.message })
  }
  finally {
    acmeSaving.value = false
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

function renew(c: Certificate) {
  toast.info(i18n.global.t('certs.renewing', { domain: c.domain }))
  certApi.renew(c.id).then(() => {
    toast.success(i18n.global.t('certs.renewed'))
    load()
  }).catch((e: any) => {
    toast.error(i18n.global.t('certs.renewFailed'), { description: e?.message })
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

      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">ID</th>
              <th class="px-3 py-2">{{ $t('certs.domainCol') }}</th>
              <th class="px-3 py-2">{{ $t('certs.altDomainsCol') }}</th>
              <th class="px-3 py-2">{{ $t('certs.providerCol') }}</th>
              <th class="px-3 py-2">{{ $t('common.status') }}</th>
              <th class="px-3 py-2">{{ $t('certs.issuerCol') }}</th>
              <th class="hidden px-3 py-2 lg:table-cell">{{ $t('common.remark') }}</th>
              <th class="px-3 py-2">{{ $t('certs.autoRenewCol') }}</th>
              <th class="cursor-pointer px-3 py-2 select-none" :title="$t('certs.sortToggle')" @click="sortAsc = !sortAsc">
                {{ $t('certs.expiryCol') }}
                <FaIcon :name="sortAsc ? 'i-lucide:chevron-up' : 'i-lucide:chevron-down'" class="text-[10px]" />
              </th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !certs.length">
              <td colspan="10" class="px-3 py-10 text-center text-muted-foreground">{{ $t('common.loading') }}</td>
            </tr>
            <tr v-else-if="!filteredCerts.length">
              <td colspan="10" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('certs.noCerts') }}
              </td>
            </tr>
            <tr v-for="c in filteredCerts" :key="c.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2 text-xs text-muted-foreground">{{ c.id }}</td>
              <td class="px-3 py-2">
                <div class="font-medium">{{ c.domain }}</div>
                <div v-if="c.sites?.length" class="text-xs text-muted-foreground">{{ $t('certs.boundTo', { sites: c.sites.join('、') }) }}</div>
              </td>
              <td class="max-w-48 truncate px-3 py-2 font-mono text-xs text-muted-foreground" :title="(c.altDomains || []).join(', ')">
                {{ (c.altDomains || []).join(', ') || '—' }}
              </td>
              <td class="px-3 py-2 text-xs">
                {{ tr(`certs.providerLabel.${c.provider}`, c.provider) }}
                <span v-if="c.provider === 'acme'" class="text-muted-foreground">
                  {{ $t('certs.providerHint', { via: c.dnsAccountId ? $t('certs.viaDns') : $t('certs.viaEnv') }) }}
                </span>
              </td>
              <td class="px-3 py-2">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="statusCls[c.status] || 'bg-muted text-muted-foreground'">
                  {{ tr(`certs.status.${c.status}`, c.status) }}
                </span>
              </td>
              <td class="px-3 py-2 text-xs text-muted-foreground">{{ c.issuer || '—' }}</td>
              <td class="hidden max-w-32 truncate px-3 py-2 text-xs lg:table-cell" :title="c.remark">{{ c.remark || '—' }}</td>
              <td class="px-3 py-2">
                <label v-if="c.provider === 'acme'" class="inline-flex cursor-pointer items-center">
                  <input type="checkbox" :checked="c.autoRenew" @change="toggleAutoRenew(c)">
                </label>
                <span v-else class="text-xs text-muted-foreground">—</span>
              </td>
              <td class="px-3 py-2 text-xs" :class="expiry(c).cls">{{ expiry(c).text }}</td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton variant="ghost" size="sm" @click="openDetail(c)">{{ $t('common.detail') }}</FaButton>
                  <FaButton v-if="c.provider === 'acme'" variant="ghost" size="sm" @click="renew(c)">{{ $t('certs.renew') }}</FaButton>
                  <FaButton variant="ghost" size="sm" @click="openEdit(c)">{{ $t('common.edit') }}</FaButton>
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeCert(c)">{{ $t('common.delete') }}</FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
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
          <select v-model.number="issueForm.acmeAccountId" class="h-9 flex-1 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
            <option :value="0">{{ $t('certs.defaultAcme') }}</option>
            <option v-for="a in acmeAccounts" :key="a.id" :value="a.id">{{ a.email }}（{{ caLabel[a.caType] }}）</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('certs.dnsAccounts') }}</span>
          <select v-model.number="issueForm.dnsAccountId" class="h-9 flex-1 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
            <option v-if="!dnsAccounts.length" :value="0" disabled>{{ $t('certs.noDnsOption') }}</option>
            <option v-for="a in dnsAccounts" :key="a.id" :value="a.id">{{ a.name }}（{{ tr(`certs.provider.${a.provider}`, a.provider) }}）</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="issueForm.remark" :placeholder="$t('certs.issueRemarkPlaceholder')" class="flex-1" />
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
              <select v-model="dnsForm.provider" class="h-9 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
                <option value="aliyun">{{ $t('certs.provider.aliyun') }}</option>
                <option value="dnspod">{{ $t('certs.provider.dnspod') }}</option>
                <option value="cloudflare">{{ $t('certs.provider.cloudflare') }}</option>
              </select>
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
              <select v-model="acmeForm.caType" class="h-9 flex-1 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
                <option value="letsencrypt">Let's Encrypt</option>
                <option value="zerossl">ZeroSSL</option>
                <option value="buypass">Buypass</option>
              </select>
              <select v-model="acmeForm.keyType" class="h-9 flex-1 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
                <option value="ec-256">EC 256</option>
                <option value="ec-384">EC 384</option>
                <option value="rsa-2048">RSA 2048</option>
                <option value="rsa-4096">RSA 4096</option>
              </select>
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
