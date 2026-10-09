<script setup lang="ts">
import api from '@/api'
import type { NodeItem } from '@/api/modules/node'
import apiNode from '@/api/modules/node'
import { nodeAssetApi } from '@/api/modules/node'
import type { NodeExecResult } from '@/api/modules/nodeexec'
import { nodeExecApi } from '@/api/modules/nodeexec'
import { i18n } from '@/locales'
import { useYwEmbed } from '@/views/desktop/embed'

defineOptions({
  name: 'NodesIndex',
})

const router = useRouter()
// 桌面承载时：详情/文件/进程下钻开新窗（router.push 会顶掉 /desktop 路由）；经典模式保持路由跳转
const ywEmbed = useYwEmbed()

interface MetricEntry {
  id: string
  online: boolean
  cpu?: number
  mem?: number
  rxSpeed?: number
  txSpeed?: number
  load1?: number
  uptime?: number
  error?: string
}

const nodes = ref<NodeItem[]>([])
const metrics = ref<MetricEntry[]>([])
const loading = ref(false)

// 基础信息 + 实时指标合并为单卡片数据
const cards = computed(() => {
  return nodes.value.map((n) => {
    const m = metrics.value.find(x => x.id === n.id)
    return { ...n, metric: m }
  })
})

async function loadMetrics() {
  try {
    const res = await api.get('api/v1/nodes/metrics', { silent: true })
    metrics.value = res.data as MetricEntry[]
  }
  catch {}
}

function fmtSpeed(n?: number) {
  if (!n) return '0 B/s'
  const u = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  let i = 0
  let v = n
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(1)} ${u[i]}`
}

function fmtUptime(sec?: number) {
  if (!sec) return '—'
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  return d > 0 ? i18n.global.t('nodes.uptimeDh', { d, h }) : i18n.global.t('nodes.uptimeHm', { h, m: Math.floor((sec % 3600) / 60) })
}

const pairVisible = ref(false)
const pairCode = ref('')
const pairCommand = ref('')
let timer: ReturnType<typeof setInterval> | null = null

async function load() {
  loading.value = true
  try {
    nodes.value = await apiNode.list()
  }
  finally {
    loading.value = false
  }
}

// ---- M54 节点 agent 更新 ----
const latestAgentVersion = ref('')
const upgradingId = ref('')

async function loadLatest() {
  try {
    const res = await api.get('api/v1/system/update/check', { silent: true })
    latestAgentVersion.value = (res.data as any)?.latest || ''
  }
  catch {}
}

function versionLt(a?: string, b?: string) {
  if (!a || !b) {
    return false
  }
  const pa = a.replace(/^v/, '').split('.').map(x => Number.parseInt(x) || 0)
  const pb = b.replace(/^v/, '').split('.').map(x => Number.parseInt(x) || 0)
  for (let i = 0; i < 3; i++) {
    if ((pa[i] || 0) !== (pb[i] || 0)) {
      return (pa[i] || 0) < (pb[i] || 0)
    }
  }
  return false
}

function agentUpgradable(n: NodeItem) {
  // 旧版 agent 未上报版本时也提示可更新（升级一次后恢复版本上报）
  return Boolean(n.remote && n.online && latestAgentVersion.value && (!n.version || versionLt(n.version, latestAgentVersion.value)))
}

async function upgradeAgent(n: NodeItem) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('nodes.agentUpgradeTitle'),
    content: i18n.global.t('nodes.agentUpgradeConfirm', { name: n.name, from: n.version || '-', to: latestAgentVersion.value }),
    onConfirm: async () => {
      upgradingId.value = n.id
      try {
        const { taskId } = await apiNode.upgradeAgent(n.id)
        useFaToast().info(i18n.global.t('nodes.agentUpgradeStarted'))
        // 轮询任务终态（下载+推送+切换，可能数分钟）
        for (let i = 0; i < 300; i++) {
          await new Promise(r => setTimeout(r, 2000))
          try {
            const t = await api.get(`api/v1/tasks/${taskId}`, { silent: true }).then(r => r.data)
            if (t.status === 'success') {
              useFaToast().success(i18n.global.t('nodes.agentUpgradeDone', { version: latestAgentVersion.value }))
              break
            }
            if (t.status === 'failed') {
              useFaToast().error(i18n.global.t('nodes.agentUpgradeFailed'), { description: t.error || '' })
              break
            }
          }
          catch {}
        }
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('common.opFailed'), { description: e?.message })
      }
      finally {
        upgradingId.value = ''
      }
    },
  })
}

async function genCode() {
  try {
    const res = await apiNode.pairingCode()
    pairCode.value = res.code
    pairCommand.value = `ypagent -core ${location.origin} -code ${res.code}`
    pairVisible.value = true
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('nodes.genFailed'), { description: e?.message })
  }
}

// ---- 批量命令 ----
const execVisible = ref(false)
const execSelected = ref<string[]>(['local'])
const execCommand = ref('')
const execRunning = ref(false)
const execResults = ref<NodeExecResult[]>([])

function openExec() {
  execSelected.value = nodes.value.filter(n => n.online).map(n => n.id)
  execCommand.value = ''
  execResults.value = []
  execVisible.value = true
}

async function doExec() {
  if (!execSelected.value.length || !execCommand.value.trim()) {
    useFaToast().warning(i18n.global.t('nodes.selectAndCommand'))
    return
  }
  execRunning.value = true
  try {
    execResults.value = await nodeExecApi.exec(execSelected.value, execCommand.value)
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('nodes.execFailed'), { description: e?.message })
  }
  finally {
    execRunning.value = false
  }
}

function remove(n: NodeItem) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('nodes.deleteTitle'),
    content: i18n.global.t('nodes.deleteConfirm', { name: n.name }),
    onConfirm: async () => {
      try {
        await apiNode.remove(n.id)
        useFaToast().success(i18n.global.t('nodes.deleted'))
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('nodes.deleteFailed'), { description: e?.message })
      }
    },
  })
}

function goDetail(id: string) {
  if (ywEmbed) {
    const name = nodes.value.find(n => n.id === id)?.name || id
    ywEmbed.openApp('nodes-detail', { title: name, launchOptions: { id, name } })
    return
  }
  router.push(`/nodes/detail/${id}`)
}
function goFiles(id: string) {
  if (ywEmbed) {
    ywEmbed.openApp('file', { title: i18n.global.t('desktop.apps.file'), launchOptions: id === 'local' ? {} : { node: id } })
    return
  }
  router.push(id === 'local' ? '/file_management' : `/file_management?node=${id}`)
}
function goProcs(id: string) {
  if (ywEmbed) {
    ywEmbed.openApp('processes', { title: i18n.global.t('desktop.apps.processes'), launchOptions: id === 'local' ? {} : { node: id } })
    return
  }
  router.push(id === 'local' ? '/processes' : `/processes?node=${id}`)
}

loadMetrics()
// ---- M43：服务器资产 ----
const assetVisible = ref(false)
const assetSaving = ref(false)
const assetTarget = ref<{ id: number | string, name: string } | null>(null)
const assetForm = ref({ expireDate: '', monthlyPrice: '', trafficQuotaGB: 0, assetRemark: '' })

function openAsset(n: { id: string, name: string }) {
  assetTarget.value = { id: n.id, name: n.name }
  const row = n.id !== 'local' ? (n as any) : null
  assetForm.value = {
    expireDate: row?.expireDate ? String(row.expireDate).slice(0, 10) : '',
    monthlyPrice: row?.monthlyPrice || '',
    trafficQuotaGB: row?.trafficQuotaGB || 0,
    assetRemark: row?.assetRemark || '',
  }
  assetVisible.value = true
}

async function saveAsset() {
  if (!assetTarget.value) return
  assetSaving.value = true
  try {
    await nodeAssetApi.update(assetTarget.value.id, {
      expireDate: assetForm.value.expireDate || null,
      monthlyPrice: assetForm.value.monthlyPrice,
      trafficQuotaGB: Number(assetForm.value.trafficQuotaGB) || 0,
      assetRemark: assetForm.value.assetRemark,
    })
    useFaToast().success(i18n.global.t('nodes.assetSaved'))
    assetVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('nodes.saveFailed'), { description: e?.message })
  }
  finally {
    assetSaving.value = false
  }
}

function remainingValueText(n: any) {
  if (n.remainingValue === '' || n.remainingValue === undefined) return '—'
  return `${i18n.global.t('nodes.remainingValue')} ${n.remainingValue}`
}

onMounted(() => {
  load()
  loadLatest()
  timer = setInterval(() => {
    load()
    loadMetrics()
  }, 10000)
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
          <YdMorphIcon name="network" :size="24" />
          <span>{{ $t('nodes.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('nodes.description') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="openExec">
          <YdMorphIcon name="terminal-square" :size="14" class="mr-1" /> {{ $t('nodes.batchCmd') }}
        </FaButton>
        <FaButton size="sm" @click="genCode">
          <YdMorphIcon name="key-round" :size="14" class="mr-1" /> {{ $t('nodes.genPairCode') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="n in cards"
          :key="n.id"
          class="cursor-pointer rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
          @click="goDetail(n.id)"
        >
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-2">
              <YdMorphIcon name="server" :size="22" class="text-primary opacity-80" />
              <div>
                <div class="flex items-center gap-2 font-medium">
                  {{ n.name }}
                  <span v-if="!n.remote" class="rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">{{ $t('nodes.local') }}</span>
                </div>
                <div v-if="n.remote" class="font-mono text-xs text-muted-foreground">{{ n.addr }}</div>
              </div>
            </div>
            <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="n.online ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'">
              <span class="inline-block size-1.5 rounded-full" :class="n.online ? 'animate-pulse bg-current' : 'bg-current'" />
              {{ n.online ? $t('nodes.online') : $t('nodes.offline') }}
            </span>
          </div>

          <!-- 实时指标 -->
          <div v-if="n.online && n.metric" class="mt-3 grid grid-cols-3 gap-x-4 gap-y-1.5 text-xs text-muted-foreground">
            <span>CPU <b class="font-mono tabular-nums text-foreground">{{ n.metric.cpu?.toFixed(1) ?? '—' }}%</b></span>
            <span>{{ $t('nodes.memory') }} <b class="font-mono tabular-nums text-foreground">{{ n.metric.mem?.toFixed(1) ?? '—' }}%</b></span>
            <span>{{ $t('nodes.load') }} <b class="font-mono tabular-nums text-foreground">{{ n.metric.load1?.toFixed(2) ?? '—' }}</b></span>
            <span>↓ {{ fmtSpeed(n.metric.rxSpeed) }}</span>
            <span>↑ {{ fmtSpeed(n.metric.txSpeed) }}</span>
            <span>{{ $t('nodes.uptimeLabel', { t: fmtUptime(n.metric.uptime) }) }}</span>
          </div>
          <div v-if="(n as any).monthlyPrice || (n as any).expireDate" class="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
            <span>{{ (n as any).monthlyPrice }}</span>
            <span v-if="(n as any).expireDate">· {{ $t('nodes.expiryLeft', { date: String((n as any).expireDate).slice(0, 10), n: (n as any).daysLeft }) }}</span>
            <span class="text-amber-600">{{ remainingValueText(n) }}</span>
          </div>
          <div v-else-if="!n.online" class="mt-3 text-xs text-red-500">
            {{ n.metric?.error || $t('nodes.unreachable') }}
          </div>

          <!-- 系统信息 + 快捷入口 -->
          <div class="mt-3 flex items-center justify-between border-t pt-3">
            <span class="flex items-center gap-1.5 text-xs text-muted-foreground">
              {{ [n.os, n.arch, n.version].filter(Boolean).join(' · ') || '—' }}
              <span
                v-if="n.remote && agentUpgradable(n)"
                class="rounded-full bg-amber-500/10 px-1.5 py-0.5 text-[10px] text-amber-600"
                :title="$t('nodes.agentUpgradableTip', { latest: latestAgentVersion })"
              >{{ $t('nodes.agentUpgradable') }}</span>
            </span>
            <div class="flex items-center gap-1">
              <FaButton
                v-if="n.remote && agentUpgradable(n)" variant="ghost" size="icon-sm"
                :title="$t('nodes.agentUpgradeBtn')" :loading="upgradingId === n.id" @click.stop="upgradeAgent(n)"
              >
                <FaIcon name="i-lucide:arrow-up-circle" class="text-sm text-amber-600" />
              </FaButton>
              <FaButton variant="ghost" size="icon-sm" :title="$t('nodes.filesTitle')" @click.stop="goFiles(n.id)">
                <FaIcon name="i-lucide:folder-open" class="text-sm" />
              </FaButton>
              <FaButton variant="ghost" size="icon-sm" :title="$t('nodes.procsTitle')" @click.stop="goProcs(n.id)">
                <FaIcon name="i-lucide:cpu" class="text-sm" />
              </FaButton>
              <FaButton v-if="n.remote" variant="ghost" size="icon-sm" :title="$t('nodes.assetInfo')" @click.stop="openAsset(n)">
                <FaIcon name="i-lucide:wallet" class="text-sm" />
              </FaButton>
              <FaButton v-if="n.remote" variant="ghost" size="icon-sm" :title="$t('nodes.deleteNode')" @click.stop="remove(n)">
                <FaIcon name="i-lucide:trash-2" class="text-sm" />
              </FaButton>
            </div>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 批量命令 -->
    <FaModal v-model="execVisible" :title="$t('nodes.batchCmd')" class="max-w-3xl!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex flex-wrap gap-1.5">
          <label
            v-for="n in nodes.filter(x => x.online)"
            :key="n.id"
            class="flex cursor-pointer items-center gap-1.5 rounded-md border px-2 py-1 text-xs transition-colors"
            :class="execSelected.includes(n.id) ? 'border-primary bg-primary/10' : 'border-border'"
          >
            <input v-model="execSelected" type="checkbox" :value="n.id" class="hidden">
            {{ n.name }}
          </label>
        </div>
        <textarea
          v-model="execCommand"
          class="h-20 w-full resize-y rounded-md border border-input bg-background p-2 font-mono text-[13px] outline-none focus:ring-1 focus:ring-primary"
          :placeholder="$t('nodes.execPh')"
          spellcheck="false"
        />
        <FaButton :loading="execRunning" @click="doExec">{{ $t('nodes.execute') }}</FaButton>
        <div v-for="r in execResults" :key="r.nodeId" class="rounded-md border p-2">
          <div class="mb-1 flex items-center gap-2 text-xs">
            <span class="rounded-full px-2 py-0.5" :class="r.ok ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'">
              {{ r.name || r.nodeId }} · {{ r.ok ? $t('common.success') : $t('common.failed') }}
            </span>
            <span v-if="r.error" class="text-red-500">{{ r.error }}</span>
          </div>
          <pre v-if="r.output" class="max-h-40 overflow-auto whitespace-pre-wrap bg-muted/50 p-2 font-mono text-xs">{{ r.output }}</pre>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="execVisible = false">{{ $t('common.close') }}</FaButton>
      </template>
    </FaModal>

    <!-- 配对码 -->
    <FaModal v-model="pairVisible" :title="$t('nodes.pairTitle')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="text-sm text-muted-foreground">
          {{ $t('nodes.pairDesc') }}
        </div>
        <div class="rounded-md bg-muted/60 p-3 font-mono text-[13px] break-all select-all">
          {{ pairCommand }}
        </div>
        <div class="text-xs text-muted-foreground">
          {{ $t('nodes.pairCode') }} <code class="rounded bg-muted px-1">{{ pairCode }}</code> {{ $t('nodes.pairCodeNote') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="pairVisible = false">{{ $t('common.close') }}</FaButton>
      </template>
    </FaModal>

    <!-- M43：节点资产 -->
    <FaModal v-model="assetVisible" :title="$t('nodes.assetTitle', { name: assetTarget?.name || '' })" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="grid grid-cols-2 gap-3">
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('nodes.expiry') }}</span>
            <input v-model="assetForm.expireDate" type="date" class="h-9 w-full rounded-md border border-input bg-background px-2 text-sm outline-none focus:border-primary">
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('nodes.priceHint') }}</span>
            <FaInput v-model="assetForm.monthlyPrice" class="w-full" />
          </label>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('nodes.quotaHint') }}</span>
            <FaInput v-model="assetForm.trafficQuotaGB" type="number" class="w-full" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('common.remark') }}</span>
            <FaInput v-model="assetForm.assetRemark" class="w-full" />
          </label>
        </div>
        <p class="text-xs text-muted-foreground">{{ $t('nodes.remainingFormula') }}</p>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="assetVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="assetSaving" @click="saveAsset">{{ $t('common.save') }}</FaButton>
      </template>
    </FaModal>
  </div>
</template>