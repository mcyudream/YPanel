<script setup lang="ts">
import type { DnsConfig, DnsOverview, DnsRecord, DnsRecordType } from '@/api/modules/dns'
import apiDns from '@/api/modules/dns'
import nodeApi, { type NodeItem } from '@/api/modules/node'
import { i18n, tr } from '@/locales'

defineOptions({
  name: 'DnsIndex',
})

interface DnsForm {
  type: DnsRecordType
  domain: string
  target: string
  comment: string
  enabled: boolean
  sort: number
}

function emptyForm(): DnsForm {
  return { type: 'address', domain: '', target: '', comment: '', enabled: true, sort: 0 }
}

const nodes = ref<NodeItem[]>([])
const nodeId = ref('local')
const ov = ref<DnsOverview | null>(null)
const ovLoading = ref(false)
const records = ref<DnsRecord[]>([])
const loading = ref(false)

const recordModal = ref(false)
const editingId = ref(0)
const saving = ref(false)
const form = ref<DnsForm>(emptyForm())
const ifaces = ref<{ name: string, family: 4 | 6, addr: string, internal: boolean }[]>([])
const quickAddr = ref('')

const settingsModal = ref(false)
const settingsSaving = ref(false)
const settingsForm = ref<DnsConfig>({ deployNodeId: '', listenIp: '', upstreams: [], cacheSize: 1000, image: 'dockurr/dnsmasq:latest', siteAlign: false, siteAlignIp: '' })
const quickListen = ref('')
const quickAlign = ref('')

const deploying = ref(false)
const applying = ref(false)
const undeployModal = ref(false)
const undeploying = ref(false)
const removeFiles = ref(false)

const checkDomain = ref('')
const checking = ref(false)
const checkOutput = ref('')

const nodeOptions = computed(() => nodes.value.map(n => ({
  label: `${n.name}${n.remote ? '' : i18n.global.t('dns.node.local')}${n.online ? '' : i18n.global.t('dns.node.offline')}`,
  value: n.id,
  disabled: !n.online,
})))

// 私网/回环地址（含内部网卡置后），供监听 IP 与 address 记录一键填写
const privateAddrs = computed(() => {
  return ifaces.value
    .filter(i => !i.addr.includes(':') || i.addr.startsWith('fd') || i.addr.startsWith('fc'))
    .filter(i => !(i.addr.startsWith('127.') || i.addr === '::1'))
    .sort((a, b) => Number(a.internal) - Number(b.internal))
    .map(i => ({ label: `${i.addr}（${i.name}${i.internal ? i18n.global.t('dns.iface.internal') : ''}）`, value: i.addr }))
})

const containerText = computed(() => {
  if (!ov.value) {
    return '—'
  }
  return tr(`dns.container.${ov.value.container}`, ov.value.container)
})

// 53 端口状态：按绑定目标判定的冲突=红色；本服务监听=正常；仅 stub 等共存监听=灰色提示
function procName(p: string) {
  const m = p.match(/users:\(\("([^"]+)"/)
  return m?.[1] ?? ''
}

const port53Text = computed(() => {
  if (!ov.value) {
    return { ok: true, text: '—' }
  }
  if (ov.value.port53Conflict) {
    const items = ov.value.port53
      .filter(o => !o.process.includes('dnsmasq'))
      .map(o => `${o.proto} ${o.addr}:53${procName(o.process) ? `（${procName(o.process)}）` : ''}`)
    return { ok: false, text: i18n.global.t('dns.port53.conflict', { items: items.join('、') }) }
  }
  const mine = ov.value.port53.some(o => o.process.includes('dnsmasq'))
  if (mine) {
    return { ok: true, text: i18n.global.t('dns.port53.mine') }
  }
  if (ov.value.port53.length) {
    return { ok: true, text: ov.value.port53Note || i18n.global.t('dns.port53.idle') }
  }
  return { ok: true, text: ov.value.deployed ? i18n.global.t('dns.port53.notListening') : i18n.global.t('dns.port53.idle') }
})

const targetPlaceholder = computed(() => {
  switch (form.value.type) {
    case 'address':
      return i18n.global.t('dns.ph.targetAddress')
    case 'cname':
      return i18n.global.t('dns.ph.targetCname')
    case 'txt':
      return i18n.global.t('dns.ph.targetTxt')
  }
  return ''
})

async function loadNodes() {
  try {
    nodes.value = await nodeApi.list()
    if (!nodes.value.some(n => n.id === nodeId.value)) {
      nodeId.value = 'local'
    }
  }
  catch {}
}

async function loadOverview() {
  ovLoading.value = true
  try {
    ov.value = await apiDns.overview(nodeId.value)
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dns.toast.loadOverviewFailed'), { description: e?.message })
  }
  finally {
    ovLoading.value = false
  }
}

async function load() {
  loading.value = true
  try {
    records.value = await apiDns.list()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dns.toast.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function loadIfaces() {
  try {
    ifaces.value = await apiDns.interfaces(nodeId.value)
  }
  catch {}
}

function typeBadge(type: DnsRecordType) {
  const map: Record<DnsRecordType, string> = {
    address: 'bg-emerald-500/10 text-emerald-600',
    cname: 'bg-blue-500/10 text-blue-600',
    txt: 'bg-amber-500/10 text-amber-600',
  }
  return map[type] ?? 'bg-muted text-muted-foreground'
}

function openCreate() {
  editingId.value = 0
  form.value = emptyForm()
  quickAddr.value = ''
  recordModal.value = true
  loadIfaces()
}

function openEdit(r: DnsRecord) {
  editingId.value = r.id
  form.value = { type: r.type, domain: r.domain, target: r.target, comment: r.comment, enabled: r.enabled, sort: r.sort }
  quickAddr.value = ''
  recordModal.value = true
  loadIfaces()
}

function formError(): string {
  if (!form.value.domain.trim()) {
    return i18n.global.t('dns.form.domainRequired')
  }
  if (!form.value.target.trim()) {
    return i18n.global.t('dns.form.targetRequired')
  }
  return ''
}

async function doSave() {
  const err = formError()
  if (err) {
    useFaToast().warning(err)
    return
  }
  saving.value = true
  try {
    await apiDns.save({
      id: editingId.value || undefined,
      type: form.value.type,
      domain: form.value.domain.trim(),
      target: form.value.target.trim(),
      comment: form.value.comment.trim(),
      enabled: form.value.enabled,
      sort: Number(form.value.sort) || 0,
    })
    useFaToast().success(i18n.global.t(editingId.value ? 'dns.toast.updated' : 'dns.toast.created'))
    recordModal.value = false
    await Promise.all([load(), loadOverview()])
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dns.toast.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

async function toggle(r: DnsRecord) {
  try {
    await (r.enabled ? apiDns.disable(r.id) : apiDns.enable(r.id))
    useFaToast().success(i18n.global.t(r.enabled ? 'dns.toast.disabledApplied' : 'dns.toast.enabledApplied'))
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dns.toast.opFailed'), { description: e?.message })
    await load()
  }
}

function remove(r: DnsRecord) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('dns.modal.removeTitle'),
    content: i18n.global.t('dns.modal.removeConfirm', { type: r.type, domain: r.domain, target: r.target }),
    onConfirm: async () => {
      try {
        await apiDns.remove(r.id)
        useFaToast().success(i18n.global.t('dns.toast.deleted'))
        await Promise.all([load(), loadOverview()])
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('dns.toast.deleteFailed'), { description: e?.message })
        await load()
      }
    },
  })
}

async function openSettings() {
  try {
    const cfg = await apiDns.getSettings()
    settingsForm.value = { ...cfg, upstreams: cfg.upstreams?.length ? [...cfg.upstreams] : [] }
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dns.toast.loadSettingsFailed'), { description: e?.message })
    return
  }
  quickListen.value = ''
  quickAlign.value = ''
  settingsModal.value = true
  loadIfaces()
}

async function doSaveSettings() {
  const cfg = settingsForm.value
  settingsSaving.value = true
  try {
    await apiDns.saveSettings({
      listenIp: cfg.listenIp.trim(),
      upstreams: cfg.upstreams.map(u => u.trim()).filter(Boolean),
      cacheSize: Number(cfg.cacheSize) || 1000,
      image: cfg.image.trim() || 'dockurr/dnsmasq:latest',
      siteAlign: cfg.siteAlign,
      siteAlignIp: cfg.siteAlignIp.trim(),
    })
    useFaToast().success(i18n.global.t(ov.value?.deployed ? 'dns.toast.settingsSavedApplied' : 'dns.toast.settingsSaved'))
    settingsModal.value = false
    await loadOverview()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dns.toast.saveFailed'), { description: e?.message })
  }
  finally {
    settingsSaving.value = false
  }
}

async function doDeploy() {
  deploying.value = true
  try {
    const res = await apiDns.deploy(nodeId.value)
    useFaToast().success(i18n.global.t('dns.toast.deployed'))
    if (res?.warnings?.length) {
      useFaToast().warning(i18n.global.t('dns.toast.deployWarn'), { description: res.warnings.join('\n'), duration: 8000 })
    }
    await loadOverview()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dns.toast.deployFailed'), { description: e?.message })
  }
  finally {
    deploying.value = false
  }
}

function doUndeploy() {
  removeFiles.value = false
  undeployModal.value = true
}

async function confirmUndeploy() {
  undeploying.value = true
  try {
    await apiDns.undeploy(nodeId.value, removeFiles.value)
    useFaToast().success(i18n.global.t('dns.toast.undeployed'))
    undeployModal.value = false
    await loadOverview()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dns.toast.undeployFailed'), { description: e?.message })
  }
  finally {
    undeploying.value = false
  }
}

async function doApply() {
  applying.value = true
  try {
    await apiDns.apply(nodeId.value)
    useFaToast().success(i18n.global.t('dns.toast.applied'))
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('dns.toast.applyFailed'), { description: e?.message })
  }
  finally {
    applying.value = false
  }
}

async function doCheck() {
  if (!checkDomain.value.trim()) {
    useFaToast().warning(i18n.global.t('dns.toast.domainRequired'))
    return
  }
  checking.value = true
  checkOutput.value = ''
  try {
    checkOutput.value = await apiDns.check(nodeId.value, checkDomain.value.trim())
  }
  catch (e: any) {
    checkOutput.value = i18n.global.t('dns.checkFailed', { msg: e?.message || '' })
  }
  finally {
    checking.value = false
  }
}

watch(quickAddr, (v) => {
  if (v) {
    form.value.target = v
  }
})
watch(quickListen, (v) => {
  if (v) {
    settingsForm.value.listenIp = v
  }
})
watch(quickAlign, (v) => {
  if (v) {
    settingsForm.value.siteAlignIp = v
  }
})
watch(nodeId, loadOverview)

onMounted(() => {
  loadNodes()
  loadOverview()
  load()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="radar" :size="24" />
          <span>{{ $t('dns.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('dns.desc') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaSelect v-model="nodeId" :options="nodeOptions" class="w-44" />
        <FaButton variant="outline" size="sm" :loading="ovLoading" @click="loadOverview">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('common.refresh') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openSettings">
          <FaIcon name="i-lucide:settings" class="mr-1" /> {{ $t('dns.actions.settings') }}
        </FaButton>
        <FaButton v-if="!ov?.deployed" size="sm" :loading="deploying" @click="doDeploy">
          <FaIcon name="i-lucide:rocket" class="mr-1" /> {{ $t('dns.actions.deploy') }}
        </FaButton>
        <template v-else>
          <FaButton variant="outline" size="sm" :loading="applying" @click="doApply">
            <FaIcon name="i-lucide:zap" class="mr-1" /> {{ $t('dns.actions.apply') }}
          </FaButton>
          <FaButton variant="outline" size="sm" @click="doUndeploy">
            <FaIcon name="i-lucide:power-off" class="mr-1" /> {{ $t('dns.actions.undeploy') }}
          </FaButton>
        </template>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 部署状态卡 -->
      <div class="grid grid-cols-2 gap-3 rounded-lg border p-4 md:grid-cols-3 lg:grid-cols-6">
        <div>
          <div class="text-xs text-muted-foreground">{{ $t('dns.card.service') }}</div>
          <div class="mt-1 flex items-center gap-1.5 text-sm font-medium">
            <span class="inline-block size-1.5 rounded-full" :class="ov?.deployed && ov.container === 'running' ? 'bg-emerald-500' : 'bg-muted-foreground/40'" />
            {{ containerText }}
          </div>
        </div>
        <div>
          <div class="text-xs text-muted-foreground">{{ $t('dns.card.node') }}</div>
          <div class="mt-1 truncate text-sm" :title="ov?.deployNodeId">
            {{ ov?.deployNodeId || $t('dns.state.undeployed') }}
          </div>
        </div>
        <div>
          <div class="text-xs text-muted-foreground">{{ $t('dns.card.listenIp') }}</div>
          <div class="mt-1 font-mono text-sm">
            {{ ov?.listenIp || $t('dns.state.notSet') }}
          </div>
        </div>
        <div>
          <div class="text-xs text-muted-foreground">{{ $t('dns.card.port53') }}</div>
          <div class="mt-1 truncate text-sm" :class="port53Text.ok ? '' : 'text-red-600'" :title="port53Text.text">
            {{ port53Text.text }}
          </div>
        </div>
        <div>
          <div class="text-xs text-muted-foreground">{{ $t('dns.card.upstreams') }}</div>
          <div class="mt-1 truncate font-mono text-xs" :title="ov?.upstreams?.join('、')">
            {{ ov?.upstreams?.join('、') || '—' }}
          </div>
        </div>
        <div>
          <div class="text-xs text-muted-foreground">{{ $t('dns.card.records') }}</div>
          <div class="mt-1 tabular-nums text-sm">
            {{ ov?.recordCount ?? 0 }}
          </div>
        </div>
      </div>
      <div v-if="!ov?.deployed" class="mt-2 text-xs text-muted-foreground">
        {{ $t('dns.hint.undeployed') }}
      </div>
      <div v-else-if="ov.deployed && !ov.nodeMatch" class="mt-2 text-xs text-amber-600">
        {{ $t('dns.hint.nodeMatch', { deployed: ov.deployNodeId, current: nodeId }) }}
      </div>

      <!-- 站点对齐预览 -->
      <div v-if="ov?.siteAlign" class="mt-2 rounded-lg border p-3 text-xs">
        <div class="flex flex-wrap items-center gap-2">
          <span class="font-medium text-foreground">{{ $t('dns.align.title') }}</span>
          <span class="text-muted-foreground">
            {{ $t('dns.align.summary', { count: ov.siteAlignCount, sites: ov.siteAlignSites, ip: ov.siteAlignIp || $t('dns.align.noIp') }) }}
          </span>
        </div>
        <div v-if="ov.siteAlignPreview.length" class="mt-2 flex flex-wrap gap-1.5">
          <span
            v-for="e in ov.siteAlignPreview"
            :key="e.domain"
            class="rounded-full bg-blue-500/10 px-2 py-0.5 font-mono text-[11px] text-blue-600"
            :title="$t('dns.align.fromSite', { site: e.siteName })"
          >
            {{ e.domain }}
          </span>
        </div>
        <div v-else class="mt-1.5 text-muted-foreground">
          {{ $t('dns.align.empty') }}
        </div>
      </div>

      <!-- 测试解析 -->
      <div class="mt-4 flex flex-wrap items-center gap-2">
        <span class="text-sm text-muted-foreground">{{ $t('dns.check.label') }}</span>
        <FaInput v-model="checkDomain" :placeholder="$t('dns.check.placeholder')" class="w-64" @keyup.enter="doCheck" />
        <FaButton variant="outline" size="sm" :loading="checking" @click="doCheck">
          <FaIcon name="i-lucide:search" class="mr-1" /> {{ $t('dns.check.query') }}
        </FaButton>
        <span v-if="ov?.listenIp" class="text-xs text-muted-foreground">{{ $t('dns.check.hint', { ip: ov.listenIp }) }}</span>
      </div>
      <pre v-if="checkOutput" class="mt-2 max-h-48 overflow-auto rounded-lg border bg-muted/30 p-3 font-mono text-xs">{{ checkOutput }}</pre>

      <!-- 解析记录表 -->
      <div class="mt-6 flex items-center justify-between">
        <div class="text-sm font-medium">
          {{ $t('dns.records.title') }}
        </div>
        <FaButton size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('dns.records.create') }}
        </FaButton>
      </div>
      <div class="mt-2 overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('dns.records.domain') }}</th>
              <th class="px-3 py-2">{{ $t('common.type') }}</th>
              <th class="px-3 py-2">{{ $t('dns.records.target') }}</th>
              <th class="px-3 py-2">{{ $t('common.status') }}</th>
              <th class="px-3 py-2">{{ $t('common.remark') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!records.length">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('dns.records.empty') }}
              </td>
            </tr>
            <tr v-for="r in records" :key="r.id" class="border-t hover:bg-accent/30">
              <td class="px-3 py-2 font-mono text-xs">
                {{ r.domain }}
              </td>
              <td class="px-3 py-2">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="typeBadge(r.type)">{{ r.type }}</span>
              </td>
              <td class="px-3 py-2">
                <span class="font-mono text-xs">{{ r.target }}</span>
              </td>
              <td class="px-3 py-2">
                <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="r.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  <span class="inline-block size-1.5 rounded-full bg-current" :class="r.enabled ? 'animate-pulse' : ''" />
                  {{ r.enabled ? $t('common.enabled') : $t('common.disabled') }}
                </span>
              </td>
              <td class="max-w-48 truncate px-3 py-2 text-xs text-muted-foreground" :title="r.comment">
                {{ r.comment || '—' }}
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton variant="ghost" size="icon-sm" :title="r.enabled ? $t('common.disabled') : $t('common.enabled')" @click="toggle(r)">
                    <FaIcon :name="r.enabled ? 'i-lucide:pause' : 'i-lucide:play'" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" :title="$t('common.edit')" @click="openEdit(r)">
                    <FaIcon name="i-lucide:pen-line" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" :title="$t('common.delete')" @click="remove(r)">
                    <FaIcon name="i-lucide:trash" class="text-sm" />
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 接入指引 -->
      <div class="mt-4 rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-xs leading-relaxed text-muted-foreground">
        <div class="mb-1 font-medium text-foreground">
          {{ $t('dns.guide.title') }}
        </div>
        <p>{{ $t('dns.guide.p1') }}</p>
        <p>{{ $t('dns.guide.p2') }}</p>
        <p>{{ $t('dns.guide.p3') }}</p>
        <p class="text-red-600/90">{{ $t('dns.guide.p4') }}</p>
      </div>
    </FaPageMain>

    <!-- 记录编辑弹窗 -->
    <FaModal v-model="recordModal" :title="editingId ? $t('dns.form.editTitle') : $t('dns.form.createTitle')" :destroy-on-close="true" class="lg:w-[560px]">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.type') }}</span>
          <FaSelect
            v-model="form.type"
            :options="[
              { label: $t('dns.form.typeAddress'), value: 'address' },
              { label: $t('dns.form.typeCname'), value: 'cname' },
              { label: $t('dns.form.typeTxt'), value: 'txt' },
            ]"
            class="flex-1"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('dns.form.domain') }}</span>
          <FaInput v-model="form.domain" :placeholder="$t('dns.form.domainPh')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('dns.form.target') }}</span>
          <FaInput v-model="form.target" :placeholder="targetPlaceholder" class="flex-1" />
          <FaSelect
            v-if="form.type === 'address' && privateAddrs.length"
            v-model="quickAddr"
            :options="privateAddrs"
            :placeholder="$t('dns.form.quickFill')"
            class="w-44 shrink-0"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('dns.form.comment') }}</span>
          <FaInput v-model="form.comment" :placeholder="$t('dns.form.commentPh')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('dns.form.sortStatus') }}</span>
          <FaInput v-model="form.sort" placeholder="0" class="w-20" />
          <FaButton :variant="form.enabled ? 'default' : 'outline'" size="sm" @click="form.enabled = !form.enabled">
            {{ form.enabled ? $t('common.enabled') : $t('common.disabled') }}
          </FaButton>
          <span class="text-xs text-muted-foreground">{{ $t('dns.form.smallFirst') }}</span>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="recordModal = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="saving" @click="doSave">
          {{ editingId ? $t('dns.form.saveUpdate') : $t('dns.form.saveCreate') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 设置弹窗 -->
    <FaModal v-model="settingsModal" :title="$t('dns.settings.title')" :destroy-on-close="true" class="lg:w-[560px]">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('dns.settings.listenIp') }}</span>
          <FaInput v-model="settingsForm.listenIp" :placeholder="$t('dns.settings.listenPh')" class="flex-1" />
          <FaSelect v-if="privateAddrs.length" v-model="quickListen" :options="privateAddrs" :placeholder="$t('dns.form.quickFill')" class="w-44 shrink-0" />
        </div>
        <div class="flex items-start gap-3">
          <span class="w-24 shrink-0 pt-2 text-sm text-muted-foreground">{{ $t('dns.settings.upstreams') }}</span>
          <div class="flex flex-1 flex-col gap-2">
            <div v-for="(_, i) in settingsForm.upstreams" :key="i" class="flex items-center gap-2">
              <FaInput v-model="settingsForm.upstreams[i]" :placeholder="$t('dns.settings.upstreamPh')" class="flex-1" />
              <FaButton variant="ghost" size="icon-sm" :title="$t('common.remove')" @click="settingsForm.upstreams.splice(i, 1)">
                <FaIcon name="i-lucide:x" class="text-sm" />
              </FaButton>
            </div>
            <div>
              <FaButton variant="outline" size="sm" :disabled="settingsForm.upstreams.length >= 8" @click="settingsForm.upstreams.push('')">
                <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('dns.settings.addUpstream') }}
              </FaButton>
              <span class="ml-2 text-xs text-muted-foreground">{{ $t('dns.settings.upstreamHint') }}</span>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('dns.settings.cache') }}</span>
          <FaInput v-model="settingsForm.cacheSize" placeholder="1000" class="w-32" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('dns.settings.image') }}</span>
          <FaInput v-model="settingsForm.image" placeholder="dockurr/dnsmasq:latest" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('dns.align.title') }}</span>
          <FaSwitch v-model="settingsForm.siteAlign" />
          <span class="text-xs text-muted-foreground">{{ $t('dns.settings.alignHint') }}</span>
        </div>
        <div v-if="settingsForm.siteAlign" class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('dns.settings.alignIp') }}</span>
          <FaInput v-model="settingsForm.siteAlignIp" :placeholder="$t('dns.settings.alignPh')" class="flex-1" />
          <FaSelect v-if="privateAddrs.length" v-model="quickAlign" :options="privateAddrs" :placeholder="$t('dns.form.quickFill')" class="w-44 shrink-0" />
        </div>
        <div class="text-xs text-muted-foreground">
          {{ $t('dns.settings.footerHint') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="settingsModal = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="settingsSaving" @click="doSaveSettings">
          {{ $t('common.save') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 卸载确认弹窗 -->
    <FaModal v-model="undeployModal" :title="$t('dns.undeploy.title')" :destroy-on-close="true" class="lg:w-[460px]">
      <div class="flex flex-col gap-3 text-sm">
        <p>{{ $t('dns.undeploy.desc', { node: nodeId }) }}</p>
        <div class="flex items-center gap-3">
          <FaSwitch v-model="removeFiles" />
          <span class="text-muted-foreground">{{ $t('dns.undeploy.removeFiles') }}</span>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="undeployModal = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="undeploying" @click="confirmUndeploy">
          {{ $t('dns.undeploy.confirm') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
