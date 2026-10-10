<script setup lang="ts">
import type { NatExternalRule, NatInterface, NatRule } from '@/api/modules/nat'
import apiNat from '@/api/modules/nat'
import nodeApi, { type NodeItem } from '@/api/modules/node'
import { i18n } from '@/locales'

defineOptions({
  name: 'NatIndex',
})

interface NatForm {
  name: string
  protocol: 'tcp' | 'udp'
  ipFamily: 4 | 6
  listenPort: number | ''
  listenPortEnd: number | ''
  targetIp: string
  targetPort: number | ''
  targetPortEnd: number | ''
  iface: string
  destIp: string
  enabled: boolean
  sort: number
}

function emptyForm(): NatForm {
  return {
    name: '', protocol: 'tcp', ipFamily: 4,
    listenPort: '', listenPortEnd: '',
    targetIp: '', targetPort: '', targetPortEnd: '',
    iface: '', destIp: '', enabled: true, sort: 0,
  }
}

const nodes = ref<NodeItem[]>([])
const nodeId = ref('local')
const rules = ref<NatRule[]>([])
const loading = ref(false)

const modalVisible = ref(false)
const editingId = ref(0)
const saving = ref(false)
const form = ref<NatForm>(emptyForm())
const ifaces = ref<NatInterface[]>([])
const checking = ref(false)
const checkResult = ref<{ ok: boolean, text: string } | null>(null)
const applying = ref(false)
const quickAddr = ref('')

// M58 外部规则（只读 + 接管导入）
const importRuleId = ref('')
const importMode = computed(() => !!importRuleId.value)
const extRules = ref<NatExternalRule[]>([])
const extAvailable = ref(true)
const extLoading = ref(false)
const extScanned = ref(false)

const nodeOptions = computed(() => nodes.value.map(n => ({
  label: `${n.name}${n.remote ? '' : i18n.global.t('nat.localNode')}${n.online ? '' : i18n.global.t('nat.offlineNode')}`,
  value: n.id,
  disabled: !n.online,
})))

// 入站网卡选项：非内部网卡优先，内部（lo/docker）置后标注；排除 veth@pair 等含 @ 的虚拟对端名
const ifaceOptions = computed(() => {
  const names = [...new Set(ifaces.value.map(i => i.name))].filter(n => !n.includes('@'))
  names.sort((a, b) => Number(isInternal(a)) - Number(isInternal(b)))
  return [{ label: i18n.global.t('nat.allIfacesCard'), value: '' }, ...names.map(n => ({ label: isInternal(n) ? `${n}${i18n.global.t('nat.internalSuffix')}` : n, value: n }))]
})
function isInternal(name: string) {
  return name === 'lo' || name.startsWith('docker') || name.startsWith('veth') || name.startsWith('br-')
}

// 一键本机地址：与所选 IP 族匹配，非内部优先
const localAddrOptions = computed(() => {
  const list = ifaces.value
    .filter(i => i.family === form.value.ipFamily)
    .sort((a, b) => Number(a.internal) - Number(b.internal))
  return list.map(i => ({ label: `${i.addr}（${i.name}${i.internal ? i18n.global.t('nat.internalComma') : ''}）`, value: i.addr }))
})

function fmtPort(p: number, pe: number) {
  return pe ? `${p}-${pe}` : `${p}`
}

async function loadNodes() {
  try {
    nodes.value = await nodeApi.list()
    if (!nodes.value.some(n => n.id === nodeId.value)) {
      nodeId.value = 'local'
    }
  }
  catch {}
}

async function load() {
  loading.value = true
  try {
    rules.value = await apiNat.list(nodeId.value)
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('nat.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function loadIfaces() {
  try {
    ifaces.value = await apiNat.interfaces(nodeId.value)
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('nat.ifaceQueryFailed'), { description: e?.message })
  }
}

function openCreate() {
  editingId.value = 0
  importRuleId.value = ''
  form.value = emptyForm()
  checkResult.value = null
  quickAddr.value = ''
  modalVisible.value = true
  loadIfaces()
}

function openEdit(r: NatRule) {
  editingId.value = r.id
  importRuleId.value = ''
  form.value = {
    name: r.name, protocol: r.protocol, ipFamily: r.ipFamily,
    listenPort: r.listenPort, listenPortEnd: r.listenPortEnd || '',
    targetIp: r.targetIp, targetPort: r.targetPort, targetPortEnd: r.targetPortEnd || '',
    iface: r.iface || '', destIp: r.destIp || '', enabled: r.enabled, sort: r.sort,
  }
  checkResult.value = null
  quickAddr.value = ''
  modalVisible.value = true
  loadIfaces()
}

// M58 接管导入：字段预填自内核规则（导入模式仅名称可改，字段以后端探测结果为准）
function openImport(er: NatExternalRule) {
  editingId.value = 0
  importRuleId.value = er.id
  form.value = {
    name: er.comment || `${i18n.global.t('nat.badgeTakeover')}-${er.dportStart}`,
    protocol: er.proto === 'udp' ? 'udp' : 'tcp',
    ipFamily: er.family,
    listenPort: er.dportStart, listenPortEnd: er.dportEnd || '',
    targetIp: er.toIp, targetPort: er.toPortStart, targetPortEnd: er.toPortEnd || '',
    iface: er.iface || '', destIp: er.destIp || '', enabled: true, sort: 0,
  }
  checkResult.value = null
  quickAddr.value = ''
  modalVisible.value = true
  loadIfaces()
}

async function loadExternals() {
  extLoading.value = true
  try {
    const res = await apiNat.external(nodeId.value)
    extRules.value = res?.rules || []
    extAvailable.value = !!res?.available
    extScanned.value = true
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('nat.extLoadFailed'), { description: e?.message })
  }
  finally {
    extLoading.value = false
  }
}

function extSrcLabel(source: NatExternalRule['source']) {
  const key = { manual: 'extSrcManual', docker: 'extSrcDocker', firewalld: 'extSrcFirewalld', ufw: 'extSrcUfw', libvirt: 'extSrcLibvirt', k8s: 'extSrcK8s', custom: 'extSrcCustom' }[source]
  return key ? i18n.global.t(`nat.${key}`) : source
}

const extSrcClass: Record<NatExternalRule['source'], string> = {
  manual: 'bg-amber-500/10 text-amber-600',
  docker: 'bg-blue-500/10 text-blue-600',
  firewalld: 'bg-purple-500/10 text-purple-600',
  ufw: 'bg-purple-500/10 text-purple-600',
  libvirt: 'bg-cyan-500/10 text-cyan-600',
  k8s: 'bg-indigo-500/10 text-indigo-600',
  custom: 'bg-muted text-muted-foreground',
}

function extMatchText(er: NatExternalRule) {
  const parts: string[] = []
  if (er.destIp) {
    parts.push(`-d ${er.destIp}`)
  }
  if (er.iface) {
    parts.push(`-i ${er.iface}`)
  }
  if (er.dportStart) {
    parts.push(`:${fmtPort(er.dportStart, er.dportEnd)} /${er.proto || '?'}`)
  }
  return parts.join('  ') || er.spec
}

function extTargetText(er: NatExternalRule) {
  return er.toIp ? `${er.toIp}:${fmtPort(er.toPortStart, er.toPortEnd)}` : '—'
}

// 表单端口数值化 + 基础校验（语义校验由后端权威执行）
function coercePorts() {
  return {
    lp: Number(form.value.listenPort),
    lpe: form.value.listenPortEnd === '' ? 0 : Number(form.value.listenPortEnd),
    tp: Number(form.value.targetPort),
    tpe: form.value.targetPortEnd === '' ? 0 : Number(form.value.targetPortEnd),
  }
}

function formError(): string {
  if (!form.value.name.trim()) {
    return i18n.global.t('nat.requireName')
  }
  // 接管导入模式：字段以后端探测结果为准，前端仅校验名称
  if (importMode.value) {
    return ''
  }
  const { lp, lpe, tp, tpe } = coercePorts()
  if (!Number.isInteger(lp) || lp < 1 || lp > 65535) {
    return i18n.global.t('nat.invalidListenPort')
  }
  if (lpe !== 0 && (!Number.isInteger(lpe) || lpe < lp || lpe > 65535)) {
    return i18n.global.t('nat.invalidListenRange')
  }
  if (!form.value.targetIp.trim()) {
    return i18n.global.t('nat.requireTargetIp')
  }
  if (!Number.isInteger(tp) || tp < 1 || tp > 65535) {
    return i18n.global.t('nat.invalidTargetPort')
  }
  if (tpe !== 0 && (!Number.isInteger(tpe) || tpe < tp || tpe > 65535)) {
    return i18n.global.t('nat.invalidTargetRange')
  }
  const lSize = lpe === 0 ? 1 : lpe - lp + 1
  const tSize = tpe === 0 ? 1 : tpe - tp + 1
  if ((lSize === 1) !== (tSize === 1) || (lSize > 1 && lSize !== tSize)) {
    return i18n.global.t('nat.rangeSemantics')
  }
  return ''
}

async function doCheck() {
  const { lp, lpe } = coercePorts()
  if (!Number.isInteger(lp) || lp < 1 || lp > 65535) {
    checkResult.value = { ok: false, text: i18n.global.t('nat.checkRequirePort') }
    return
  }
  checking.value = true
  try {
    const occupied = await apiNat.checkPort(nodeId.value, form.value.protocol, lp, lpe || 0)
    if (!occupied.length) {
      checkResult.value = { ok: true, text: i18n.global.t('nat.portFree', { port: fmtPort(lp, lpe || 0), proto: form.value.protocol }) }
    }
    else {
      checkResult.value = { ok: false, text: i18n.global.t('nat.portOccupied', { list: occupied.map(o => `${o.port}/${o.proto}${o.process ? ` ${o.process}` : ''}`).join('、') }) }
    }
  }
  catch (e: any) {
    checkResult.value = { ok: false, text: e?.message || i18n.global.t('nat.checkFailed') }
  }
  finally {
    checking.value = false
  }
}

function toastWarnings(warnings: string[]) {
  if (warnings.length) {
    useFaToast().warning(i18n.global.t('nat.appliedWithWarnings'), { description: warnings.join('\n'), duration: 8000 })
  }
}

async function doSave() {
  const err = formError()
  if (err) {
    useFaToast().warning(err)
    return
  }
  // 接管导入：字段由后端按探测结果权威生成，仅回传名称
  if (importMode.value) {
    saving.value = true
    try {
      const res = await apiNat.import(nodeId.value, [{ ruleId: importRuleId.value, name: form.value.name.trim() }])
      useFaToast().success(i18n.global.t('nat.extImported'))
      toastWarnings(res?.warnings || [])
      modalVisible.value = false
      await Promise.all([load(), loadExternals()])
    }
    catch (e: any) {
      useFaToast().error(i18n.global.t('nat.extImportFailed'), { description: e?.message })
    }
    finally {
      saving.value = false
    }
    return
  }
  const { lp, lpe, tp, tpe } = coercePorts()
  saving.value = true
  try {
    const res = await apiNat.save({
      id: editingId.value || undefined,
      nodeId: nodeId.value,
      name: form.value.name.trim(),
      protocol: form.value.protocol,
      ipFamily: form.value.ipFamily,
      listenPort: lp,
      listenPortEnd: lpe,
      targetIp: form.value.targetIp.trim(),
      targetPort: tp,
      targetPortEnd: tpe,
      iface: form.value.iface,
      destIp: form.value.destIp.trim(),
      enabled: form.value.enabled,
      sort: Number(form.value.sort) || 0,
    })
    useFaToast().success(editingId.value ? i18n.global.t('nat.updated') : i18n.global.t('nat.created'))
    toastWarnings(res?.warnings || [])
    modalVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('nat.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

async function toggle(r: NatRule) {
  try {
    if (r.enabled) {
      await apiNat.disable(r.id)
      useFaToast().success(i18n.global.t('nat.disabledRebuilt'))
    }
    else {
      await apiNat.enable(r.id)
      useFaToast().success(i18n.global.t('nat.enabledApplied'))
    }
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('nat.opFailed'), { description: e?.message })
    await load()
  }
}

function remove(r: NatRule) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('nat.deleteRuleTitle'),
    content: i18n.global.t('nat.deleteConfirm', { name: r.name, listen: fmtPort(r.listenPort, r.listenPortEnd), target: `${r.targetIp}:${fmtPort(r.targetPort, r.targetPortEnd)}` }),
    onConfirm: async () => {
      try {
        await apiNat.remove(r.id)
        useFaToast().success(i18n.global.t('nat.deleted'))
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('nat.deleteFailed'), { description: e?.message })
        await load()
      }
    },
  })
}

async function doApply() {
  applying.value = true
  try {
    const res = await apiNat.apply(nodeId.value)
    useFaToast().success(i18n.global.t('nat.reapplied'))
    toastWarnings(res?.warnings || [])
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('nat.reapplyFailed'), { description: e?.message })
  }
  finally {
    applying.value = false
  }
}

// 一键本机：选中地址回填目标 IP；切换 IP 族后快速选择失效需重选
watch(quickAddr, (v) => {
  if (v) {
    form.value.targetIp = v
  }
})
watch(() => form.value.ipFamily, () => {
  quickAddr.value = ''
})

watch(nodeId, () => {
  load()
  loadExternals()
})

onMounted(() => {
  loadNodes()
  load()
  loadExternals()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="network" :size="24" />
          <span>{{ $t('nat.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('nat.desc') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaSelect v-model="nodeId" :options="nodeOptions" class="w-44" />
        <FaButton variant="outline" size="sm" :loading="loading" @click="load">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('common.refresh') }}
        </FaButton>
        <FaButton variant="outline" size="sm" :loading="applying" @click="doApply">
          <FaIcon name="i-lucide:zap" class="mr-1" /> {{ $t('nat.reapplyBtn') }}
        </FaButton>
        <FaButton size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('nat.createRule') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('common.name') }}</th>
              <th class="px-3 py-2">{{ $t('nat.forwardCol') }}</th>
              <th class="px-3 py-2">{{ $t('nat.protoFamilyCol') }}</th>
              <th class="px-3 py-2">{{ $t('nat.ifaceCol') }}</th>
              <th class="px-3 py-2">{{ $t('nat.sortCol') }}</th>
              <th class="px-3 py-2">{{ $t('common.status') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!rules.length">
              <td colspan="7" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('nat.noRules') }}
              </td>
            </tr>
            <tr v-for="r in rules" :key="r.id" class="border-t hover:bg-accent/30">
              <td class="px-3 py-2">
                <div class="flex items-center gap-1.5">
                  <span class="font-medium">{{ r.name }}</span>
                  <span v-if="r.srcSpec" class="rounded-full bg-amber-500/10 px-2 py-0.5 text-xs text-amber-600" :title="r.srcSpec">{{ $t('nat.badgeTakeover') }}</span>
                </div>
                <div v-if="r.iface" class="font-mono text-xs text-muted-foreground">via {{ r.iface }}</div>
              </td>
              <td class="px-3 py-2">
                <span class="font-mono text-xs">
                  <span class="rounded bg-muted px-1.5 py-0.5">:{{ fmtPort(r.listenPort, r.listenPortEnd) }}</span>
                  <FaIcon name="i-lucide:arrow-right" class="mx-1 text-muted-foreground" />
                  <span class="rounded bg-muted px-1.5 py-0.5">{{ r.targetIp }}:{{ fmtPort(r.targetPort, r.targetPortEnd) }}</span>
                </span>
              </td>
              <td class="px-3 py-2">
                <span class="rounded-full bg-blue-500/10 px-2 py-0.5 text-xs text-blue-600">{{ r.protocol }}</span>
                <span class="ml-1 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">IPv{{ r.ipFamily }}</span>
              </td>
              <td class="px-3 py-2 text-xs text-muted-foreground">
                {{ r.iface || $t('nat.allIfaces') }}
              </td>
              <td class="px-3 py-2 tabular-nums text-xs text-muted-foreground">
                {{ r.sort }}
              </td>
              <td class="px-3 py-2">
                <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="r.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  <span class="inline-block size-1.5 rounded-full bg-current" :class="r.enabled ? 'animate-pulse' : ''" />
                  {{ r.enabled ? $t('common.enabled') : $t('common.disabled') }}
                </span>
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
      <div class="mt-3 text-xs text-muted-foreground">
        {{ $t('nat.footNote') }}
      </div>

      <!-- M58 本机已有规则：只读展示 + 主链手工 DNAT 接管导入 -->
      <div class="mt-6 flex items-center justify-between gap-3">
        <div class="min-w-0">
          <div class="text-sm font-medium">{{ $t('nat.extTitle') }}</div>
          <div class="mt-0.5 truncate text-xs text-muted-foreground">{{ $t('nat.extDesc') }}</div>
        </div>
        <FaButton variant="outline" size="sm" :loading="extLoading" @click="loadExternals">
          <FaIcon name="i-lucide:radar" class="mr-1" /> {{ $t('nat.extScan') }}
        </FaButton>
      </div>
      <div class="mt-2 overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('nat.extColSource') }}</th>
              <th class="px-3 py-2">{{ $t('nat.extColChain') }}</th>
              <th class="px-3 py-2">{{ $t('nat.extColMatch') }}</th>
              <th class="px-3 py-2">{{ $t('nat.extColTarget') }}</th>
              <th class="px-3 py-2">{{ $t('nat.extColStatus') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!extRules.length">
              <td colspan="6" class="px-3 py-8 text-center text-muted-foreground">
                {{ extAvailable ? $t('nat.extNone') : $t('nat.extUnavailable') }}
              </td>
            </tr>
            <tr v-for="er in extRules" :key="er.id" class="border-t hover:bg-accent/30" :title="er.spec">
              <td class="px-3 py-2">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="extSrcClass[er.source]">{{ extSrcLabel(er.source) }}</span>
              </td>
              <td class="px-3 py-2 font-mono text-xs text-muted-foreground">
                {{ er.chain }}
              </td>
              <td class="px-3 py-2 font-mono text-xs">
                {{ extMatchText(er) }}
              </td>
              <td class="px-3 py-2 font-mono text-xs">
                {{ extTargetText(er) }}
              </td>
              <td class="px-3 py-2">
                <span
                  v-if="er.importable"
                  class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600"
                >{{ $t('nat.extImportable') }}</span>
                <span v-else class="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground" :title="er.reason">{{ $t('nat.extReadonly') }}</span>
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton v-if="er.importable" variant="outline" size="sm" @click="openImport(er)">
                    <FaIcon name="i-lucide:import" class="mr-1" /> {{ $t('nat.extImport') }}
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <FaModal v-model="modalVisible" :title="importMode ? $t('nat.extImport') : (editingId ? $t('nat.editRule') : $t('nat.createRule'))" :destroy-on-close="true" class="lg:w-[560px]">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('common.name') }}</span>
          <FaInput v-model="form.name" :placeholder="$t('nat.namePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('nat.protocol') }}</span>
          <FaSelect v-model="form.protocol" :options="[{ label: 'TCP', value: 'tcp' }, { label: 'UDP', value: 'udp' }]" class="w-32" :disabled="importMode" />
          <span class="ml-2 text-sm text-muted-foreground">{{ $t('nat.ipFamily') }}</span>
          <FaSelect v-model="form.ipFamily" :options="[{ label: 'IPv4', value: 4 }, { label: 'IPv6', value: 6 }]" class="w-32" :disabled="importMode" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('nat.listenPort') }}</span>
          <FaInput v-model="form.listenPort" :placeholder="$t('nat.portStartPlaceholder')" class="w-36" :disabled="importMode" />
          <span class="text-xs text-muted-foreground">{{ $t('nat.to') }}</span>
          <FaInput v-model="form.listenPortEnd" :placeholder="$t('nat.portEndPlaceholder')" class="w-36" :disabled="importMode" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('nat.targetIp') }}</span>
          <FaInput v-model="form.targetIp" :placeholder="form.ipFamily === 4 ? $t('nat.targetIp4Placeholder') : $t('nat.targetIp6Placeholder')" class="flex-1" :disabled="importMode" />
          <FaSelect
            v-if="!importMode && localAddrOptions.length"
            v-model="quickAddr"
            :options="localAddrOptions"
            :placeholder="$t('nat.quickLocal')"
            class="w-44 shrink-0"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('nat.targetPort') }}</span>
          <FaInput v-model="form.targetPort" :placeholder="$t('nat.startPlaceholder')" class="w-36" :disabled="importMode" />
          <span class="text-xs text-muted-foreground">{{ $t('nat.to') }}</span>
          <FaInput v-model="form.targetPortEnd" :placeholder="$t('nat.portEndPlaceholder')" class="w-36" :disabled="importMode" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('nat.destMatch') }}</span>
          <FaInput v-model="form.destIp" :placeholder="$t('nat.destMatchPh')" class="flex-1" :disabled="importMode" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('nat.ifaceCol') }}</span>
          <FaSelect v-model="form.iface" :options="ifaceOptions" class="flex-1" :disabled="importMode" />
        </div>
        <div v-if="importMode" class="rounded-md bg-amber-500/10 px-3 py-2 text-xs text-amber-600">
          {{ $t('nat.extImportTip') }}
        </div>
        <div v-else class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('nat.sortStatus') }}</span>
          <FaInput v-model="form.sort" placeholder="0" class="w-20" />
          <FaButton :variant="form.enabled ? 'default' : 'outline'" size="sm" @click="form.enabled = !form.enabled">
            {{ form.enabled ? $t('common.enabled') : $t('common.disabled') }}
          </FaButton>
          <span class="text-xs text-muted-foreground">{{ $t('nat.smallFirst') }}</span>
        </div>
        <div v-if="!importMode" class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ $t('nat.portCheck') }}</span>
          <FaButton variant="outline" size="sm" :loading="checking" @click="doCheck">
            {{ $t('nat.checkBtn') }}
          </FaButton>
          <span v-if="checkResult" class="min-w-0 flex-1 truncate text-xs" :class="checkResult.ok ? 'text-emerald-600' : 'text-red-600'" :title="checkResult.text">{{ checkResult.text }}</span>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="modalVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="saving" @click="doSave">
          {{ importMode ? $t('nat.extImport') : (editingId ? $t('nat.saveApply') : $t('nat.createApply')) }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
