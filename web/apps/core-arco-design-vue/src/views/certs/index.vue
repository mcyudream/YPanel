<script setup lang="ts">
import type { Certificate, DnsAccount, AcmeAccount } from '@/api/modules/cert'
import { certApi } from '@/api/modules/cert'

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
    toast.error('读取证书列表失败', { description: e?.message })
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

const providerLabel: Record<string, string> = { acme: 'DNS 账号', selfsigned: '自签', upload: '手动上传' }
const caLabel: Record<string, string> = { letsencrypt: "Let's Encrypt", zerossl: 'ZeroSSL', buypass: 'Buypass' }

function expiry(c: Certificate) {
  if (!c.notAfter)
    return { text: '—', cls: 'text-muted-foreground', days: 0 }
  const d = new Date(c.notAfter)
  const days = Math.floor((d.getTime() - Date.now()) / 86400000)
  const cls = days < 0 ? 'text-red-500' : days < 15 ? 'text-amber-500' : ''
  return { text: d.toISOString().slice(0, 10), cls, days }
}

const statusLabel: Record<string, { text: string, cls: string }> = {
  ok: { text: '正常', cls: 'bg-emerald-500/10 text-emerald-600' },
  expiring: { text: '即将过期', cls: 'bg-amber-500/10 text-amber-600' },
  expired: { text: '已过期', cls: 'bg-red-500/10 text-red-600' },
  error: { text: '异常', cls: 'bg-red-500/10 text-red-600' },
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
    toast.warning('请填写主域名')
    return
  }
  if (!issueForm.value.dnsAccountId) {
    toast.warning('请选择 DNS 账户（可在「DNS 账户」中创建）')
    return
  }
  issueSaving.value = true
  toast.info('正在申请证书，DNS 验证通常需要 1-3 分钟，请勿关闭…')
  try {
    await certApi.issue(issueForm.value)
    toast.success('证书申请成功')
    issueVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error('申请失败', { description: e?.message })
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
    toast.warning('主域名、证书内容、私钥内容均必填')
    return
  }
  uploadSaving.value = true
  try {
    await certApi.upload(uploadForm.value)
    toast.success('证书上传成功')
    uploadVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error('上传失败', { description: e?.message })
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
    toast.warning('请填写域名')
    return
  }
  selfSaving.value = true
  try {
    await certApi.selfSigned(selfForm.value)
    toast.success('自签证书已生成')
    selfVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error('自签失败', { description: e?.message })
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
    toast.warning('名称、AccessKey、Secret 均必填')
    return
  }
  dnsSaving.value = true
  try {
    await certApi.createDnsAccount(dnsForm.value)
    toast.success('DNS 账户已创建')
    dnsForm.value = { name: '', provider: 'aliyun', accessKey: '', secret: '' }
    await loadAccounts()
  }
  catch (e: any) {
    toast.error('创建失败', { description: e?.message })
  }
  finally {
    dnsSaving.value = false
  }
}

function removeDns(a: DnsAccount) {
  modal.confirm({
    title: '删除 DNS 账户',
    content: `确认删除 DNS 账户 ${a.name}？使用该账户续签的证书将受影响。`,
    onConfirm: async () => {
      try {
        await certApi.removeDnsAccount(a.id)
        await loadAccounts()
      }
      catch (e: any) {
        toast.error('删除失败', { description: e?.message })
      }
    },
  })
}

const providerText: Record<string, string> = { aliyun: '阿里云', dnspod: '腾讯云 DNSPod', cloudflare: 'Cloudflare' }

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
    toast.warning('请填写合法邮箱')
    return
  }
  acmeSaving.value = true
  try {
    await certApi.createAcmeAccount(acmeForm.value)
    toast.success('ACME 账户已创建')
    acmeForm.value = { email: '', caType: 'letsencrypt', keyType: 'ec-256' }
    await loadAccounts()
  }
  catch (e: any) {
    toast.error('创建失败', { description: e?.message })
  }
  finally {
    acmeSaving.value = false
  }
}

function removeAcme(a: AcmeAccount) {
  modal.confirm({
    title: '删除 ACME 账户',
    content: `确认删除 ACME 账户 ${a.email}（${caLabel[a.caType]}）？`,
    onConfirm: async () => {
      try {
        await certApi.removeAcmeAccount(a.id)
        await loadAccounts()
      }
      catch (e: any) {
        toast.error('删除失败', { description: e?.message })
      }
    },
  })
}

// ---- 证书操作 ----

function renew(c: Certificate) {
  toast.info(`正在续签 ${c.domain}，请勿关闭…`)
  certApi.renew(c.id).then(() => {
    toast.success('续签成功')
    load()
  }).catch((e: any) => {
    toast.error('续签失败', { description: e?.message })
  })
}

function removeCert(c: Certificate) {
  modal.confirm({
    title: '删除证书',
    content: `确认删除证书 ${c.domain}（${c.certName}）？绑定的站点 HTTPS 将受影响。`,
    onConfirm: async () => {
      try {
        await certApi.remove(c.id)
        toast.success('已删除')
        await load()
      }
      catch (e: any) {
        toast.error('删除失败', { description: e?.message })
      }
    },
  })
}

function toggleAutoRenew(c: Certificate) {
  certApi.update(c.id, { autoRenew: !c.autoRenew }).then(() => {
    c.autoRenew = !c.autoRenew
  }).catch((e: any) => {
    toast.error('更新失败', { description: e?.message })
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
    toast.success('已保存')
    editVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
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
    toast.error('读取详情失败', { description: e?.message })
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
          <span>证书</span>
        </div>
      </template>
      <template #description>
        <span>HTTPS 证书统一管理：ACME 申请（DNS 验证）/ 上传 / 自签，支持自动续签与站点绑定</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton size="sm" @click="openIssue">
          <FaIcon name="i-lucide:badge-check" class="mr-1" /> 申请证书
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openUpload">
          <FaIcon name="i-lucide:upload" class="mr-1" /> 上传证书
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openSelf">
          <FaIcon name="i-lucide:pen-tool" class="mr-1" /> 自签证书
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openAcme">
          <FaIcon name="i-lucide:user-round" class="mr-1" /> Acme 账户
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openDns">
          <FaIcon name="i-lucide:cloud" class="mr-1" /> DNS 账户
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="mb-3 flex items-center gap-2">
        <FaInput v-model="keyword" placeholder="搜索域名 / 备注" class="w-56!" />
        <span class="ml-auto text-xs text-muted-foreground">共 {{ filteredCerts.length }} 条</span>
      </div>

      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">ID</th>
              <th class="px-3 py-2">域名</th>
              <th class="px-3 py-2">其他域名</th>
              <th class="px-3 py-2">申请方式</th>
              <th class="px-3 py-2">状态</th>
              <th class="px-3 py-2">颁发组织</th>
              <th class="hidden px-3 py-2 lg:table-cell">备注</th>
              <th class="px-3 py-2">自动续签</th>
              <th class="cursor-pointer px-3 py-2 select-none" title="点击切换排序" @click="sortAsc = !sortAsc">
                过期时间
                <FaIcon :name="sortAsc ? 'i-lucide:chevron-up' : 'i-lucide:chevron-down'" class="text-[10px]" />
              </th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !certs.length">
              <td colspan="10" class="px-3 py-10 text-center text-muted-foreground">加载中…</td>
            </tr>
            <tr v-else-if="!filteredCerts.length">
              <td colspan="10" class="px-3 py-10 text-center text-muted-foreground">
                暂无证书。点击右上角「申请证书」（需 DNS 账户）或「上传证书」开始。
              </td>
            </tr>
            <tr v-for="c in filteredCerts" :key="c.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2 text-xs text-muted-foreground">{{ c.id }}</td>
              <td class="px-3 py-2">
                <div class="font-medium">{{ c.domain }}</div>
                <div v-if="c.sites?.length" class="text-xs text-muted-foreground">绑定：{{ c.sites.join('、') }}</div>
              </td>
              <td class="max-w-48 truncate px-3 py-2 font-mono text-xs text-muted-foreground" :title="(c.altDomains || []).join(', ')">
                {{ (c.altDomains || []).join(', ') || '—' }}
              </td>
              <td class="px-3 py-2 text-xs">
                {{ providerLabel[c.provider] || c.provider }}
                <span v-if="c.provider === 'acme'" class="text-muted-foreground">
                  （{{ c.dnsAccountId ? 'DNS 账户' : '环境变量' }}）
                </span>
              </td>
              <td class="px-3 py-2">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="statusLabel[c.status]?.cls || 'bg-muted text-muted-foreground'">
                  {{ statusLabel[c.status]?.text || c.status }}
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
                  <FaButton variant="ghost" size="sm" @click="openDetail(c)">详情</FaButton>
                  <FaButton v-if="c.provider === 'acme'" variant="ghost" size="sm" @click="renew(c)">续签</FaButton>
                  <FaButton variant="ghost" size="sm" @click="openEdit(c)">编辑</FaButton>
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeCert(c)">删除</FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <!-- 申请证书 -->
    <FaModal v-model="issueVisible" title="申请证书" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">主域名</span>
          <FaInput v-model="issueForm.domain" placeholder="如 example.com（自动附带 *.example.com 泛域名）" class="flex-1" />
        </div>
        <div class="flex items-start gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">其他域名</span>
          <textarea v-model="issueForm.altDomains" class="h-14 flex-1 rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" placeholder="可选，每行一个或逗号分隔" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">Acme 账户</span>
          <select v-model.number="issueForm.acmeAccountId" class="h-9 flex-1 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
            <option :value="0">默认（面板环境变量邮箱）</option>
            <option v-for="a in acmeAccounts" :key="a.id" :value="a.id">{{ a.email }}（{{ caLabel[a.caType] }}）</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">DNS 账户</span>
          <select v-model.number="issueForm.dnsAccountId" class="h-9 flex-1 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
            <option v-if="!dnsAccounts.length" :value="0" disabled>暂无 DNS 账户，请先在「DNS 账户」中创建</option>
            <option v-for="a in dnsAccounts" :key="a.id" :value="a.id">{{ a.name }}（{{ providerText[a.provider] }}）</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">备注</span>
          <FaInput v-model="issueForm.remark" placeholder="如：通配证书" class="flex-1" />
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="issueForm.autoRenew" type="checkbox"> 自动续签
          <span class="text-xs text-muted-foreground">由 acme.sh 定时任务自动续期并在续期后重载 nginx</span>
        </label>
        <div class="rounded-md bg-muted/50 p-3 text-xs text-muted-foreground">
          使用 DNS API 验证（支持阿里云 / 腾讯云 DNSPod / Cloudflare），无需开放 80 端口，可签发泛域名证书。
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="issueVisible = false">取消</FaButton>
        <FaButton :loading="issueSaving" @click="doIssue">确认申请</FaButton>
      </template>
    </FaModal>

    <!-- 上传证书 -->
    <FaModal v-model="uploadVisible" title="上传证书" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">证书名称</span>
          <FaInput v-model="uploadForm.certName" placeholder="可选，默认取主域名" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">主域名</span>
          <FaInput v-model="uploadForm.domain" placeholder="如 example.com" class="flex-1" />
        </div>
        <div class="flex items-start gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">证书（PEM）</span>
          <textarea v-model="uploadForm.certPem" class="h-24 flex-1 rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" placeholder="-----BEGIN CERTIFICATE-----&#10;…（fullchain 全链）" />
        </div>
        <div class="flex items-start gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">私钥（PEM）</span>
          <textarea v-model="uploadForm.keyPem" class="h-24 flex-1 rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" placeholder="-----BEGIN PRIVATE KEY-----&#10;…" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">备注</span>
          <FaInput v-model="uploadForm.remark" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="uploadVisible = false">取消</FaButton>
        <FaButton :loading="uploadSaving" @click="doUpload">上传</FaButton>
      </template>
    </FaModal>

    <!-- 自签证书 -->
    <FaModal v-model="selfVisible" title="自签证书" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">域名</span>
          <FaInput v-model="selfForm.domain" placeholder="如 example.com（内网/测试用）" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">有效期</span>
          <FaInput v-model.number="selfForm.days" class="w-32!" />
          <span class="text-xs text-muted-foreground">天</span>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">备注</span>
          <FaInput v-model="selfForm.remark" class="flex-1" />
        </div>
        <div class="rounded-md bg-muted/50 p-3 text-xs text-muted-foreground">
          自签证书浏览器会提示不受信任，仅建议内网或测试环境使用。
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="selfVisible = false">取消</FaButton>
        <FaButton :loading="selfSaving" @click="doSelf">生成</FaButton>
      </template>
    </FaModal>

    <!-- DNS 账户 -->
    <FaModal v-model="dnsVisible" title="DNS 账户" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">名称</th>
                <th class="px-3 py-2">类型</th>
                <th class="px-3 py-2 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!dnsAccounts.length">
                <td colspan="3" class="px-3 py-6 text-center text-xs text-muted-foreground">暂无 DNS 账户</td>
              </tr>
              <tr v-for="a in dnsAccounts" :key="a.id" class="border-t">
                <td class="px-3 py-2">{{ a.name }}</td>
                <td class="px-3 py-2 text-xs text-muted-foreground">{{ providerText[a.provider] || a.provider }}</td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeDns(a)">删除</FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="rounded-md border p-3">
          <div class="mb-2 text-sm font-medium">创建 DNS 账户</div>
          <div class="flex flex-col gap-2">
            <div class="flex items-center gap-2">
              <FaInput v-model="dnsForm.name" placeholder="名称，如 my-aliyun" class="flex-1" />
              <select v-model="dnsForm.provider" class="h-9 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
                <option value="aliyun">阿里云</option>
                <option value="dnspod">腾讯云 DNSPod</option>
                <option value="cloudflare">Cloudflare</option>
              </select>
            </div>
            <FaInput v-model="dnsForm.accessKey" placeholder="AccessKey ID（阿里云）/ SecretId（DNSPod）/ Email（Cloudflare Global Key）" />
            <FaInput v-model="dnsForm.secret" placeholder="AccessKey Secret / SecretKey / Global API Key" type="password" />
            <div class="flex justify-end">
              <FaButton size="sm" :loading="dnsSaving" @click="doCreateDns">创建</FaButton>
            </div>
          </div>
        </div>
        <div class="text-xs text-muted-foreground">凭据加密存储于面板数据库，仅用于 ACME DNS 验证。</div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="dnsVisible = false">关闭</FaButton>
      </template>
    </FaModal>

    <!-- Acme 账户 -->
    <FaModal v-model="acmeVisible" title="Acme 账户" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">邮箱</th>
                <th class="px-3 py-2">CA</th>
                <th class="px-3 py-2">密钥算法</th>
                <th class="px-3 py-2 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!acmeAccounts.length">
                <td colspan="4" class="px-3 py-6 text-center text-xs text-muted-foreground">暂无账户（申请证书可用面板默认配置）</td>
              </tr>
              <tr v-for="a in acmeAccounts" :key="a.id" class="border-t">
                <td class="px-3 py-2">{{ a.email }}</td>
                <td class="px-3 py-2 text-xs text-muted-foreground">{{ caLabel[a.caType] || a.caType }}</td>
                <td class="px-3 py-2 font-mono text-xs text-muted-foreground">{{ a.keyType }}</td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeAcme(a)">删除</FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="rounded-md border p-3">
          <div class="mb-2 text-sm font-medium">创建 Acme 账户</div>
          <div class="flex flex-col gap-2">
            <FaInput v-model="acmeForm.email" placeholder="邮箱" />
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
              <FaButton size="sm" :loading="acmeSaving" @click="doCreateAcme">创建</FaButton>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="acmeVisible = false">关闭</FaButton>
      </template>
    </FaModal>

    <!-- 编辑证书 -->
    <FaModal v-model="editVisible" title="编辑证书" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">域名</span>
          <span class="flex-1 font-mono text-sm">{{ editForm.domain }}</span>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">备注</span>
          <FaInput v-model="editForm.remark" class="flex-1" />
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="editForm.autoRenew" type="checkbox"> 自动续签
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="editVisible = false">取消</FaButton>
        <FaButton :loading="editSaving" @click="doEdit">保存</FaButton>
      </template>
    </FaModal>

    <!-- 证书详情 -->
    <FaModal v-model="detailVisible" title="证书详情" :destroy-on-close="true">
      <div v-if="detailLoading" class="py-8 text-center text-sm text-muted-foreground">加载中…</div>
      <div v-else-if="detailData" class="space-y-3">
        <div class="rounded-md border p-3 text-sm">
          <div class="grid grid-cols-[5rem_1fr] gap-y-2 text-xs">
            <span class="text-muted-foreground">主域名</span><span class="font-mono">{{ detailData.cert.domain }}</span>
            <span class="text-muted-foreground">证书名称</span><span class="font-mono">{{ detailData.cert.certName }}</span>
            <span class="text-muted-foreground">颁发组织</span><span>{{ detailData.cert.issuer || '—' }}</span>
            <span class="text-muted-foreground">过期时间</span><span>{{ detailData.cert.notAfter ? new Date(detailData.cert.notAfter).toLocaleString() : '—' }}</span>
            <span class="text-muted-foreground">绑定站点</span><span>{{ detailData.cert.sites?.join('、') || '未绑定' }}</span>
          </div>
        </div>
        <pre class="max-h-72 overflow-auto rounded-md bg-muted/50 p-3 font-mono text-[11px] leading-relaxed">{{ detailData.text || '（证书文件不存在）' }}</pre>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="detailVisible = false">关闭</FaButton>
      </template>
    </FaModal>
  </div>
</template>
