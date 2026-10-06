<script setup lang="ts">
import type { NatInterface, NatRule } from '@/api/modules/nat'
import apiNat from '@/api/modules/nat'
import nodeApi, { type NodeItem } from '@/api/modules/node'

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
  enabled: boolean
  sort: number
}

function emptyForm(): NatForm {
  return {
    name: '', protocol: 'tcp', ipFamily: 4,
    listenPort: '', listenPortEnd: '',
    targetIp: '', targetPort: '', targetPortEnd: '',
    iface: '', enabled: true, sort: 0,
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

const nodeOptions = computed(() => nodes.value.map(n => ({
  label: `${n.name}${n.remote ? '' : '（本机）'}${n.online ? '' : '（离线）'}`,
  value: n.id,
  disabled: !n.online,
})))

// 入站网卡选项：非内部网卡优先，内部（lo/docker）置后标注；排除 veth@pair 等含 @ 的虚拟对端名
const ifaceOptions = computed(() => {
  const names = [...new Set(ifaces.value.map(i => i.name))].filter(n => !n.includes('@'))
  names.sort((a, b) => Number(isInternal(a)) - Number(isInternal(b)))
  return [{ label: '所有网卡', value: '' }, ...names.map(n => ({ label: isInternal(n) ? `${n}（内部）` : n, value: n }))]
})
function isInternal(name: string) {
  return name === 'lo' || name.startsWith('docker') || name.startsWith('veth') || name.startsWith('br-')
}

// 一键本机地址：与所选 IP 族匹配，非内部优先
const localAddrOptions = computed(() => {
  const list = ifaces.value
    .filter(i => i.family === form.value.ipFamily)
    .sort((a, b) => Number(a.internal) - Number(b.internal))
  return list.map(i => ({ label: `${i.addr}（${i.name}${i.internal ? '，内部' : ''}）`, value: i.addr }))
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
    useFaToast().error('规则加载失败', { description: e?.message })
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
    useFaToast().error('网卡查询失败', { description: e?.message })
  }
}

function openCreate() {
  editingId.value = 0
  form.value = emptyForm()
  checkResult.value = null
  quickAddr.value = ''
  modalVisible.value = true
  loadIfaces()
}

function openEdit(r: NatRule) {
  editingId.value = r.id
  form.value = {
    name: r.name, protocol: r.protocol, ipFamily: r.ipFamily,
    listenPort: r.listenPort, listenPortEnd: r.listenPortEnd || '',
    targetIp: r.targetIp, targetPort: r.targetPort, targetPortEnd: r.targetPortEnd || '',
    iface: r.iface || '', enabled: r.enabled, sort: r.sort,
  }
  checkResult.value = null
  quickAddr.value = ''
  modalVisible.value = true
  loadIfaces()
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
  const { lp, lpe, tp, tpe } = coercePorts()
  if (!form.value.name.trim()) {
    return '请填写规则名称'
  }
  if (!Number.isInteger(lp) || lp < 1 || lp > 65535) {
    return '映射端口须为 1-65535 的整数'
  }
  if (lpe !== 0 && (!Number.isInteger(lpe) || lpe < lp || lpe > 65535)) {
    return '映射端口范围不合法（结束 ≥ 起始且 ≤ 65535，留空为单端口）'
  }
  if (!form.value.targetIp.trim()) {
    return '请填写目标 IP'
  }
  if (!Number.isInteger(tp) || tp < 1 || tp > 65535) {
    return '目标端口须为 1-65535 的整数'
  }
  if (tpe !== 0 && (!Number.isInteger(tpe) || tpe < tp || tpe > 65535)) {
    return '目标端口范围不合法'
  }
  const lSize = lpe === 0 ? 1 : lpe - lp + 1
  const tSize = tpe === 0 ? 1 : tpe - tp + 1
  if ((lSize === 1) !== (tSize === 1) || (lSize > 1 && lSize !== tSize)) {
    return '端口范围语义：仅允许「范围→同尺寸范围」或「单端口→单端口」'
  }
  return ''
}

async function doCheck() {
  const { lp, lpe } = coercePorts()
  if (!Number.isInteger(lp) || lp < 1 || lp > 65535) {
    checkResult.value = { ok: false, text: '请先填写合法的映射端口' }
    return
  }
  checking.value = true
  try {
    const occupied = await apiNat.checkPort(nodeId.value, form.value.protocol, lp, lpe || 0)
    if (!occupied.length) {
      checkResult.value = { ok: true, text: `端口 ${fmtPort(lp, lpe || 0)}/${form.value.protocol} 未被占用` }
    }
    else {
      checkResult.value = { ok: false, text: `占用：${occupied.map(o => `${o.port}/${o.proto}${o.process ? ` ${o.process}` : ''}`).join('、')}` }
    }
  }
  catch (e: any) {
    checkResult.value = { ok: false, text: e?.message || '检测失败' }
  }
  finally {
    checking.value = false
  }
}

function toastWarnings(warnings: string[]) {
  if (warnings.length) {
    useFaToast().warning('已生效，但有告警', { description: warnings.join('\n'), duration: 8000 })
  }
}

async function doSave() {
  const err = formError()
  if (err) {
    useFaToast().warning(err)
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
      enabled: form.value.enabled,
      sort: Number(form.value.sort) || 0,
    })
    useFaToast().success(editingId.value ? '规则已更新并下发' : '规则已创建并下发')
    toastWarnings(res?.warnings || [])
    modalVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error('保存失败', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

async function toggle(r: NatRule) {
  try {
    if (r.enabled) {
      await apiNat.disable(r.id)
      useFaToast().success('已停用（链已重建）')
    }
    else {
      await apiNat.enable(r.id)
      useFaToast().success('已启用并下发')
    }
    await load()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
    await load()
  }
}

function remove(r: NatRule) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除转发规则',
    content: `确认删除「${r.name}」（${fmtPort(r.listenPort, r.listenPortEnd)} → ${r.targetIp}:${fmtPort(r.targetPort, r.targetPortEnd)}）？将立即重建目标机规则链。`,
    onConfirm: async () => {
      try {
        await apiNat.remove(r.id)
        useFaToast().success('已删除')
        await load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
        await load()
      }
    },
  })
}

async function doApply() {
  applying.value = true
  try {
    const res = await apiNat.apply(nodeId.value)
    useFaToast().success('已整链重放')
    toastWarnings(res?.warnings || [])
  }
  catch (e: any) {
    useFaToast().error('重放失败', { description: e?.message })
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

watch(nodeId, load)

onMounted(() => {
  loadNodes()
  load()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="network" :size="24" />
          <span>NAT 转发</span>
        </div>
      </template>
      <template #description>
        <span>iptables DNAT 端口转发：宿主机映射端口 → 目标 IP:端口（v4/v6、端口范围、回程 MASQUERADE 自动处理）</span>
      </template>
      <div class="flex items-center gap-2">
        <FaSelect v-model="nodeId" :options="nodeOptions" class="w-44" />
        <FaButton variant="outline" size="sm" :loading="loading" @click="load">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> 刷新
        </FaButton>
        <FaButton variant="outline" size="sm" :loading="applying" @click="doApply">
          <FaIcon name="i-lucide:zap" class="mr-1" /> 整链重放
        </FaButton>
        <FaButton size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 新建规则
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">名称</th>
              <th class="px-3 py-2">转发（映射 → 目标）</th>
              <th class="px-3 py-2">协议 / 族</th>
              <th class="px-3 py-2">入站网卡</th>
              <th class="px-3 py-2">排序</th>
              <th class="px-3 py-2">状态</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!rules.length">
              <td colspan="7" class="px-3 py-10 text-center text-muted-foreground">
                当前节点暂无转发规则
              </td>
            </tr>
            <tr v-for="r in rules" :key="r.id" class="border-t hover:bg-accent/30">
              <td class="px-3 py-2">
                <div class="font-medium">{{ r.name }}</div>
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
                {{ r.iface || '所有' }}
              </td>
              <td class="px-3 py-2 tabular-nums text-xs text-muted-foreground">
                {{ r.sort }}
              </td>
              <td class="px-3 py-2">
                <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="r.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  <span class="inline-block size-1.5 rounded-full bg-current" :class="r.enabled ? 'animate-pulse' : ''" />
                  {{ r.enabled ? '启用' : '停用' }}
                </span>
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton variant="ghost" size="icon-sm" :title="r.enabled ? '停用' : '启用'" @click="toggle(r)">
                    <FaIcon :name="r.enabled ? 'i-lucide:pause' : 'i-lucide:play'" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" title="编辑" @click="openEdit(r)">
                    <FaIcon name="i-lucide:pen-line" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" title="删除" @click="remove(r)">
                    <FaIcon name="i-lucide:trash" class="text-sm" />
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-3 text-xs text-muted-foreground">
        说明：目标机需有 iptables / ip6tables、ss、ip 命令；ip_forward 未开启时会运行时自动开启（重启后失效，持久化属宿主机运维）。规则保存在面板库中，面板启动时自动重放。
      </div>
    </FaPageMain>

    <FaModal v-model="modalVisible" :title="editingId ? '编辑转发规则' : '新建转发规则'" :destroy-on-close="true" class="lg:w-[560px]">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">名称</span>
          <FaInput v-model="form.name" placeholder="规则备注名，如 游戏服务器" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">协议</span>
          <FaSelect v-model="form.protocol" :options="[{ label: 'TCP', value: 'tcp' }, { label: 'UDP', value: 'udp' }]" class="w-32" />
          <span class="ml-2 text-sm text-muted-foreground">IP 族</span>
          <FaSelect v-model="form.ipFamily" :options="[{ label: 'IPv4', value: 4 }, { label: 'IPv6', value: 6 }]" class="w-32" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">映射端口</span>
          <FaInput v-model="form.listenPort" placeholder="起始，如 8080" class="w-36" />
          <span class="text-xs text-muted-foreground">至</span>
          <FaInput v-model="form.listenPortEnd" placeholder="结束（留空=单端口）" class="w-36" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">目标 IP</span>
          <FaInput v-model="form.targetIp" :placeholder="form.ipFamily === 4 ? '如 10.0.0.5' : '如 fd00::5'" class="flex-1" />
          <FaSelect
            v-if="localAddrOptions.length"
            v-model="quickAddr"
            :options="localAddrOptions"
            placeholder="一键填入本机"
            class="w-44 shrink-0"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">目标端口</span>
          <FaInput v-model="form.targetPort" placeholder="起始" class="w-36" />
          <span class="text-xs text-muted-foreground">至</span>
          <FaInput v-model="form.targetPortEnd" placeholder="结束（留空=单端口）" class="w-36" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">入站网卡</span>
          <FaSelect v-model="form.iface" :options="ifaceOptions" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">排序 / 状态</span>
          <FaInput v-model="form.sort" placeholder="0" class="w-20" />
          <FaButton :variant="form.enabled ? 'default' : 'outline'" size="sm" @click="form.enabled = !form.enabled">
            {{ form.enabled ? '启用' : '停用' }}
          </FaButton>
          <span class="text-xs text-muted-foreground">小值先应用</span>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">端口检测</span>
          <FaButton variant="outline" size="sm" :loading="checking" @click="doCheck">
            检测映射端口占用
          </FaButton>
          <span v-if="checkResult" class="min-w-0 flex-1 truncate text-xs" :class="checkResult.ok ? 'text-emerald-600' : 'text-red-600'" :title="checkResult.text">{{ checkResult.text }}</span>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="modalVisible = false">
          取消
        </FaButton>
        <FaButton :loading="saving" @click="doSave">
          {{ editingId ? '保存并下发' : '创建并下发' }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
