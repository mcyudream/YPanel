<script setup lang="ts">
import type { VpnConfig, VpnStatus, VpnTable } from '@/api/modules/vpn'

import apiCompose from '@/api/modules/compose'
import { storeApi } from '@/api/modules/store'
import { vpnApi } from '@/api/modules/vpn'
import { useTaskCenterStore } from '@/store/modules/taskCenter'
import { i18n, tr } from '@/locales'
import YdDangerDelete from '@/components/YdDangerDelete/index.vue'
import YdLogViewer from '@/components/YdLogViewer/index.vue'

defineOptions({
  name: 'VpnIndex',
})

type TabKey = 'net' | 'proxy' | 'runtime' | 'advanced' | 'logs'

const TABS: { key: TabKey }[] = [
  { key: 'net' },
  { key: 'proxy' },
  { key: 'runtime' },
  { key: 'advanced' },
  { key: 'logs' },
]

const toast = useFaToast()
const taskCenter = useTaskCenterStore()

const status = ref<VpnStatus | null>(null)
const statusLoading = ref(false)
const activeTab = ref<TabKey>('net')

// ---------- 安装 ----------

const installing = ref(false)
const installTaskId = ref(0)
const seed = reactive({
  networkName: 'ypanel-net',
  networkSecret: '',
  virtualIp: '10.144.144.1',
  dhcp: false,
})

async function refreshStatus(silent = true) {
  statusLoading.value = true
  try {
    status.value = await vpnApi.status()
    if (!status.value.configured) {
      const res = await vpnApi.getConfig()
      applyConfig(res.config)
      seed.networkName = res.config.networkName || seed.networkName
      seed.virtualIp = res.config.virtualIp || seed.virtualIp
    }
  }
  catch (e: any) {
    if (!silent) {
      toast.error(i18n.global.t('vpn.toast.statusLoadFailed'), { description: e?.message })
    }
  }
  finally {
    statusLoading.value = false
  }
}

async function startInstall() {
  if (!seed.networkName.trim()) {
    toast.error(i18n.global.t('vpn.toast.nameRequired'))
    return
  }
  if (!seed.dhcp && !seed.virtualIp.trim()) {
    toast.error(i18n.global.t('vpn.toast.ipRequired'))
    return
  }
  installing.value = true
  try {
    const { taskId } = await vpnApi.install({
      networkName: seed.networkName.trim(),
      networkSecret: seed.networkSecret,
      virtualIp: seed.virtualIp.trim(),
      dhcp: seed.dhcp,
    })
    installTaskId.value = taskId
    toast.success(i18n.global.t('vpn.toast.installStarted'), { description: i18n.global.t('vpn.toast.installStartedDesc') })
    pollInstall()
  }
  catch (e: any) {
    toast.error(i18n.global.t('vpn.toast.installCreateFailed'), { description: e?.message })
  }
  finally {
    installing.value = false
  }
}

let installTimer: ReturnType<typeof setTimeout> | null = null
let installPolls = 0

function pollInstall() {
  if (installTimer) {
    clearTimeout(installTimer)
  }
  installPolls += 1
  if (installPolls > 100) {
    toast.error(i18n.global.t('vpn.toast.installTimeout'), { description: i18n.global.t('vpn.toast.installTimeoutDesc') })
    return
  }
  installTimer = setTimeout(async () => {
    await refreshStatus()
    if (status.value?.installed && status.value?.running) {
      toast.success(i18n.global.t('vpn.toast.ready'), { description: i18n.global.t('vpn.toast.readyDesc') })
      loadConfig()
      return
    }
    if (installTaskId.value) {
      pollInstall()
    }
  }, 3000)
}

// ---------- 配置 ----------

const config = reactive<VpnConfig>({
  rawMode: false,
  raw: '',
  instanceName: 'easytier',
  hostname: 'easytier',
  networkName: 'ypanel-net',
  networkSecret: '',
  virtualIp: '10.144.144.1',
  dhcp: false,
  peers: [],
  listeners: ['tcp://0.0.0.0:11010', 'udp://0.0.0.0:11010'],
  proxyNetworks: [],
  exitNodes: [],
  enableExitNode: false,
  latencyFirst: false,
  devName: '',
})
const rendered = ref('')
const peersText = ref('')
const listenersText = ref('')
const exitNodesText = ref('')
const configLoading = ref(false)
const saving = ref(false)

function applyConfig(cfg: VpnConfig) {
  Object.assign(config, cfg)
  peersText.value = (cfg.peers || []).join('\n')
  listenersText.value = (cfg.listeners || []).join('\n')
  exitNodesText.value = (cfg.exitNodes || []).join('\n')
  if (!config.proxyNetworks) {
    config.proxyNetworks = []
  }
}

async function loadConfig() {
  configLoading.value = true
  try {
    const res = await vpnApi.getConfig()
    applyConfig(res.config)
    rendered.value = res.rendered || ''
  }
  catch (e: any) {
    toast.error(i18n.global.t('vpn.toast.configLoadFailed'), { description: e?.message })
  }
  finally {
    configLoading.value = false
  }
}

function linesOf(text: string) {
  return text.split('\n').map(l => l.trim()).filter(Boolean)
}

async function saveConfig() {
  if (config.rawMode) {
    if (!(config.raw ?? '').trim()) {
      toast.error(i18n.global.t('vpn.toast.rawRequired'), { description: i18n.global.t('vpn.toast.rawRequiredDesc') })
      return
    }
  }
  else {
    if (!config.networkName.trim()) {
      toast.error(i18n.global.t('vpn.toast.nameRequired'))
      return
    }
    if (!config.dhcp && !config.virtualIp.trim()) {
      toast.error(i18n.global.t('vpn.toast.ipRequired'))
      return
    }
  }
  saving.value = true
  try {
    const payload: VpnConfig = {
      ...config,
      peers: linesOf(peersText.value),
      listeners: linesOf(listenersText.value),
      exitNodes: linesOf(exitNodesText.value),
    }
    status.value = await vpnApi.saveConfig(payload)
    toast.success(i18n.global.t('vpn.toast.saved'), { description: i18n.global.t('vpn.toast.savedDesc') })
    if (config.rawMode) {
      rendered.value = config.raw ?? ''
    }
    else {
      const res = await vpnApi.getConfig()
      rendered.value = res.rendered || ''
    }
  }
  catch (e: any) {
    const hint = config.rawMode
      ? i18n.global.t('vpn.toast.saveFailedHint')
      : ''
    toast.error(i18n.global.t('vpn.toast.saveFailed'), { description: (e?.message || '') + hint })
  }
  finally {
    saving.value = false
  }
}

// ---------- 子网映射 ----------

function addProxyNetwork() {
  config.proxyNetworks.push({ cidr: '', remap: '', enabled: true })
}

function removeProxyNetwork(i: number) {
  config.proxyNetworks.splice(i, 1)
}

// ---------- 生命周期操作 ----------

const acting = ref(false)

async function doAction(action: 'start' | 'stop' | 'restart') {
  if (!status.value?.project) {
    return
  }
  acting.value = true
  try {
    await storeApi.installedAction(status.value.project, action)
    toast.success(i18n.global.t('vpn.toast.actionDone', { action: tr(`common.${action}`, action) }))
    await refreshStatus()
  }
  catch (e: any) {
    toast.error(i18n.global.t('vpn.toast.opFailed'), { description: e?.message })
  }
  finally {
    acting.value = false
  }
}

const uninstallVisible = ref(false)
const uninstalling = ref(false)
const UNINSTALL_OPTS = computed(() => [
  { key: 'purgeData', label: i18n.global.t('vpn.uninstall.purgeData'), desc: i18n.global.t('vpn.uninstall.purgeDataDesc') },
  { key: 'rmi', label: i18n.global.t('vpn.uninstall.rmi'), desc: i18n.global.t('vpn.uninstall.rmiDesc') },
])

async function doUninstall(checked: Record<string, boolean>) {
  if (!status.value?.project) {
    return
  }
  uninstalling.value = true
  try {
    await storeApi.uninstall(status.value.project, checked)
    toast.success(i18n.global.t('vpn.toast.uninstallStarted'), { description: i18n.global.t('vpn.toast.uninstallStartedDesc') })
    uninstallVisible.value = false
    await refreshStatus()
  }
  catch (e: any) {
    toast.error(i18n.global.t('vpn.toast.uninstallFailed'), { description: e?.message })
  }
  finally {
    uninstalling.value = false
  }
}

// ---------- 对端与路由 ----------

const peers = ref<VpnTable | null>(null)
const routes = ref<VpnTable | null>(null)
const runtimeLoading = ref(false)

async function refreshRuntime() {
  runtimeLoading.value = true
  try {
    const [p, r] = await Promise.all([vpnApi.peers(), vpnApi.routes()])
    peers.value = p
    routes.value = r
  }
  catch (e: any) {
    toast.error(i18n.global.t('vpn.toast.peersLoadFailed'), { description: e?.message })
  }
  finally {
    runtimeLoading.value = false
  }
}

// ---------- 日志 ----------

const logsText = ref('')
const logsLoading = ref(false)

async function refreshLogs() {
  const token = useAppAccountStore().token
  if (!token || !status.value?.project) {
    return
  }
  logsLoading.value = true
  try {
    const url = apiCompose.logsURL(status.value.project, undefined, token, 500)
    const res = await fetch(url)
    logsText.value = await res.text()
  }
  catch (e: any) {
    logsText.value = i18n.global.t('vpn.toast.logsFailed', { msg: e?.message || e })
  }
  finally {
    logsLoading.value = false
  }
}

// ---------- Tab 副作用 ----------

let runtimeTimer: ReturnType<typeof setInterval> | null = null
let logsTimer: ReturnType<typeof setInterval> | null = null

function clearTimers() {
  if (runtimeTimer) {
    clearInterval(runtimeTimer)
    runtimeTimer = null
  }
  if (logsTimer) {
    clearInterval(logsTimer)
    logsTimer = null
  }
}

watch(activeTab, (tab) => {
  clearTimers()
  if (tab === 'runtime') {
    refreshRuntime()
    runtimeTimer = setInterval(refreshRuntime, 10000)
  }
  else if (tab === 'logs') {
    refreshLogs()
    logsTimer = setInterval(refreshLogs, 8000)
  }
})

function openTask() {
  taskCenter.open(installTaskId.value)
}

onMounted(async () => {
  await refreshStatus(false)
  if (status.value?.installed) {
    await loadConfig()
  }
  else {
    // 未安装：种子表单预填默认期望值
    try {
      const res = await vpnApi.getConfig()
      seed.networkName = res.config.networkName || seed.networkName
      seed.virtualIp = res.config.virtualIp || seed.virtualIp
    }
    catch {}
  }
})

onBeforeUnmount(clearTimers)
</script>

<template>
  <div class="flex flex-col gap-4 p-4">
    <!-- 未安装：空态 + 一键安装 -->
    <template v-if="!status?.installed">
      <div class="mx-auto mt-10 w-full max-w-xl rounded-xl border border-border bg-card p-8 text-center shadow-sm">
        <div class="mb-3 text-4xl">
          <span class="i-lucide-network mx-auto inline-block size-12 text-primary" />
        </div>
        <h2 class="mb-2 text-lg font-semibold">
          {{ $t('vpn.empty.title') }}
        </h2>
        <p class="text-muted-foreground mb-6 text-sm leading-6">
          {{ $t('vpn.empty.desc') }}
        </p>
        <div class="mb-4 grid grid-cols-1 gap-3 text-left sm:grid-cols-2">
          <label class="flex flex-col gap-1 text-sm">
            <span class="text-muted-foreground">{{ $t('vpn.field.networkName') }}</span>
            <input v-model="seed.networkName" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none" placeholder="ypanel-net">
          </label>
          <label class="flex flex-col gap-1 text-sm">
            <span class="text-muted-foreground">{{ $t('vpn.field.networkSecret') }}</span>
            <input v-model="seed.networkSecret" type="password" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none" :placeholder="$t('vpn.empty.secretPh')">
          </label>
          <label class="flex flex-col gap-1 text-sm">
            <span class="text-muted-foreground">{{ $t('vpn.field.virtualIp') }}</span>
            <input v-model="seed.virtualIp" :disabled="seed.dhcp" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none disabled:opacity-50" placeholder="10.144.144.1">
          </label>
          <label class="flex items-end gap-2 pb-1 text-sm">
            <FaSwitch v-model="seed.dhcp" />
            <span class="text-muted-foreground">{{ $t('vpn.field.dhcpAuto') }}</span>
          </label>
        </div>
        <div class="flex items-center justify-center gap-2">
          <FaButton :loading="installing" @click="startInstall">
            {{ $t('vpn.empty.install') }}
          </FaButton>
          <FaButton v-if="installTaskId" variant="outline" @click="openTask">
            {{ $t('vpn.empty.viewTask') }}
          </FaButton>
        </div>
        <p class="text-muted-foreground/70 mt-4 text-xs leading-5">
          {{ $t('vpn.empty.source') }}
        </p>
      </div>
    </template>

    <!-- 已安装 -->
    <template v-else>
      <!-- 状态条 -->
      <div class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-xl border border-border bg-card px-4 py-3 shadow-sm">
        <span class="flex items-center gap-1.5 text-sm font-medium">
          <span
            class="inline-block size-2 rounded-full"
            :class="status?.running ? 'bg-emerald-500' : 'bg-red-500'"
          />
          {{ status?.running ? $t('vpn.state.running') : $t('vpn.state.stopped') }}
        </span>
        <span v-if="status?.virtualIp" class="bg-primary/10 text-primary rounded-full px-2 py-0.5 text-xs">
          {{ $t('vpn.state.virtualIp', { ip: status.virtualIp }) }}
        </span>
        <span class="bg-accent text-muted-foreground rounded-full px-2 py-0.5 text-xs">
          {{ $t('vpn.state.network', { name: status?.networkName || '-' }) }}
        </span>
        <span class="bg-accent text-muted-foreground rounded-full px-2 py-0.5 text-xs">
          {{ $t('vpn.state.peers', { n: status?.peerCount ?? 0 }) }}
        </span>
        <span class="text-muted-foreground text-xs">v{{ status?.version }}</span>
        <div class="ml-auto flex items-center gap-2">
          <FaButton size="sm" variant="outline" :loading="statusLoading" @click="refreshStatus(false)">
            {{ $t('common.refresh') }}
          </FaButton>
          <FaButton v-if="!status?.running" size="sm" variant="outline" :loading="acting" @click="doAction('start')">
            {{ $t('common.start') }}
          </FaButton>
          <FaButton v-else size="sm" variant="outline" :loading="acting" @click="doAction('stop')">
            {{ $t('common.stop') }}
          </FaButton>
          <FaButton size="sm" variant="outline" :loading="acting" @click="doAction('restart')">
            {{ $t('common.restart') }}
          </FaButton>
          <FaButton size="sm" variant="outline" class="text-red-500" @click="uninstallVisible = true">
            {{ $t('vpn.action.uninstall') }}
          </FaButton>
        </div>
      </div>

      <!-- Tabs -->
      <div class="flex flex-wrap items-center gap-1">
        <button
          v-for="t in TABS"
          :key="t.key"
          class="rounded-md px-3 py-1.5 text-sm transition-colors"
          :class="activeTab === t.key ? 'bg-primary/10 text-primary font-medium' : 'text-muted-foreground hover:bg-accent/50'"
          @click="activeTab = t.key"
        >
          {{ $t(`vpn.tab.${t.key}`) }}
        </button>
      </div>

      <!-- 组网配置 -->
      <template v-if="activeTab === 'net'">
        <div class="rounded-xl border border-border bg-card p-5 shadow-sm">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <label class="flex flex-col gap-1 text-sm">
              <span class="text-muted-foreground">{{ $t('vpn.field.networkName') }}</span>
              <input v-model="config.networkName" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none">
            </label>
            <label class="flex flex-col gap-1 text-sm">
              <span class="text-muted-foreground">{{ $t('vpn.field.networkSecret') }}</span>
              <input v-model="config.networkSecret" type="password" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none" :placeholder="$t('vpn.net.secretPh')">
            </label>
            <label class="flex flex-col gap-1 text-sm">
              <span class="text-muted-foreground">{{ $t('vpn.field.virtualIp') }}</span>
              <input v-model="config.virtualIp" :disabled="config.dhcp" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none disabled:opacity-50" placeholder="10.144.144.1">
            </label>
            <label class="flex flex-col gap-1 text-sm">
              <span class="text-muted-foreground">{{ $t('vpn.net.hostname') }}</span>
              <input v-model="config.hostname" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none">
            </label>
            <label class="flex flex-col gap-1 text-sm md:col-span-2">
              <span class="text-muted-foreground">{{ $t('vpn.net.peersLabel') }}</span>
              <textarea v-model="peersText" rows="3" class="rounded-md border border-input bg-background px-2 py-1.5 font-mono text-sm outline-none" :placeholder="$t('vpn.net.peersPh')" />
            </label>
            <label class="flex flex-col gap-1 text-sm md:col-span-2">
              <span class="text-muted-foreground">{{ $t('vpn.net.listenersLabel') }}</span>
              <textarea v-model="listenersText" rows="2" class="rounded-md border border-input bg-background px-2 py-1.5 font-mono text-sm outline-none" />
            </label>
            <div class="flex items-center gap-2 text-sm">
              <FaSwitch v-model="config.dhcp" />
              <span class="text-muted-foreground">{{ $t('vpn.net.dhcpAuto') }}</span>
            </div>
            <div class="flex items-center gap-2 text-sm">
              <FaSwitch v-model="config.latencyFirst" />
              <span class="text-muted-foreground">{{ $t('vpn.net.latencyFirst') }}</span>
            </div>
            <div class="flex items-center gap-2 text-sm">
              <FaSwitch v-model="config.enableExitNode" />
              <span class="text-muted-foreground">{{ $t('vpn.net.exitNode') }}</span>
            </div>
            <label class="flex flex-col gap-1 text-sm">
              <span class="text-muted-foreground">{{ $t('vpn.net.exitNodesLabel') }}</span>
              <textarea v-model="exitNodesText" rows="2" class="rounded-md border border-input bg-background px-2 py-1.5 font-mono text-sm outline-none" :placeholder="$t('vpn.net.exitNodesPh')" />
            </label>
          </div>
          <div class="mt-5 flex items-center justify-end gap-2">
            <FaButton variant="outline" :disabled="saving" @click="loadConfig">
              {{ $t('common.reset') }}
            </FaButton>
            <FaButton :loading="saving" @click="saveConfig">
              {{ $t('vpn.action.saveApply') }}
            </FaButton>
          </div>
        </div>
      </template>

      <!-- 子网映射 -->
      <template v-else-if="activeTab === 'proxy'">
        <div class="rounded-xl border border-border bg-card p-5 shadow-sm">
          <div class="text-muted-foreground mb-4 text-sm leading-6">
            {{ $t('vpn.proxy.desc1') }}<b>{{ $t('vpn.proxy.descBold') }}</b>{{ $t('vpn.proxy.desc2') }}
          </div>
          <div class="mb-3 grid grid-cols-[1fr_1fr_auto_auto] items-center gap-2 px-1 text-xs text-muted-foreground">
            <span>{{ $t('vpn.proxy.cidr') }}</span>
            <span>{{ $t('vpn.proxy.remap') }}</span>
            <span>{{ $t('common.enabled') }}</span>
            <span class="w-14 text-right">{{ $t('common.operation') }}</span>
          </div>
          <div
            v-for="(pn, i) in config.proxyNetworks"
            :key="i"
            class="mb-2 grid grid-cols-[1fr_1fr_auto_auto] items-center gap-2"
          >
            <input v-model="pn.cidr" class="h-9 rounded-md border border-input bg-background px-2 font-mono text-sm outline-none" placeholder="192.168.1.0/24">
            <input v-model="pn.remap" class="h-9 rounded-md border border-input bg-background px-2 font-mono text-sm outline-none" :placeholder="$t('vpn.proxy.remapPh')">
            <FaSwitch v-model="pn.enabled" />
            <FaButton size="sm" variant="outline" class="text-red-500" @click="removeProxyNetwork(i)">
              {{ $t('common.delete') }}
            </FaButton>
          </div>
          <div class="mt-4 flex items-center justify-between">
            <FaButton size="sm" variant="outline" @click="addProxyNetwork">
              <span class="i-lucide-plus mr-1 inline-block size-3.5" /> {{ $t('vpn.proxy.add') }}
            </FaButton>
            <div class="flex items-center gap-2">
              <FaButton variant="outline" :disabled="saving" @click="loadConfig">
                {{ $t('common.reset') }}
              </FaButton>
              <FaButton :loading="saving" @click="saveConfig">
                {{ $t('vpn.action.saveApply') }}
              </FaButton>
            </div>
          </div>
        </div>
      </template>

      <!-- 对端与路由 -->
      <template v-else-if="activeTab === 'runtime'">
        <div class="flex flex-col gap-4">
          <div class="rounded-xl border border-border bg-card p-5 shadow-sm">
            <div class="mb-3 flex items-center justify-between">
              <h3 class="text-sm font-semibold">
                {{ $t('vpn.runtime.peers') }}
              </h3>
              <FaButton size="sm" variant="outline" :loading="runtimeLoading" @click="refreshRuntime">
                {{ $t('common.refresh') }}
              </FaButton>
            </div>
            <div v-if="peers?.rows?.length" class="overflow-x-auto">
              <table class="w-full text-left text-sm">
                <thead>
                  <tr class="text-muted-foreground border-b border-border text-xs">
                    <th v-for="c in peers.columns" :key="c" class="px-2 py-1.5 font-medium">
                      {{ c }}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, i) in peers.rows" :key="i" class="border-b border-border/50 last:border-0">
                    <td v-for="(cell, j) in row" :key="j" class="px-2 py-1.5 font-mono text-xs">
                      {{ cell }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="text-muted-foreground py-6 text-center text-sm">
              {{ $t('vpn.runtime.noPeers') }}
            </div>
          </div>
          <div class="rounded-xl border border-border bg-card p-5 shadow-sm">
            <h3 class="mb-3 text-sm font-semibold">
              {{ $t('vpn.runtime.routes') }}
            </h3>
            <div v-if="routes?.rows?.length" class="overflow-x-auto">
              <table class="w-full text-left text-sm">
                <thead>
                  <tr class="text-muted-foreground border-b border-border text-xs">
                    <th v-for="c in routes.columns" :key="c" class="px-2 py-1.5 font-medium">
                      {{ c }}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, i) in routes.rows" :key="i" class="border-b border-border/50 last:border-0">
                    <td v-for="(cell, j) in row" :key="j" class="px-2 py-1.5 font-mono text-xs">
                      {{ cell }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="text-muted-foreground py-6 text-center text-sm">
              {{ $t('vpn.runtime.noRoutes') }}
            </div>
          </div>
        </div>
      </template>

      <!-- 高级（TOML） -->
      <template v-else-if="activeTab === 'advanced'">
        <div class="rounded-xl border border-border bg-card p-5 shadow-sm">
          <div class="mb-3 flex flex-wrap items-center gap-3">
            <FaSwitch v-model="config.rawMode" />
            <span class="text-sm">{{ $t('vpn.advanced.rawMode') }}</span>
            <span class="text-muted-foreground text-xs">{{ $t('vpn.advanced.rawHint') }}</span>
            <FaButton v-if="config.rawMode" size="sm" variant="outline" class="ml-auto" @click="config.raw = rendered">
              {{ $t('vpn.advanced.importRendered') }}
            </FaButton>
          </div>
          <textarea
            v-model="config.raw"
            :disabled="!config.rawMode"
            rows="22"
            spellcheck="false"
            class="w-full rounded-md border border-input bg-background p-3 font-mono text-xs leading-5 outline-none disabled:opacity-60"
            :placeholder="rendered || $t('vpn.advanced.rawPh')"
          />
          <div class="mt-4 flex items-center justify-end gap-2">
            <FaButton variant="outline" :disabled="saving" @click="loadConfig">
              {{ $t('common.reset') }}
            </FaButton>
            <FaButton :loading="saving" @click="saveConfig">
              {{ $t('vpn.action.saveApply') }}
            </FaButton>
          </div>
        </div>
      </template>

      <!-- 日志 -->
      <template v-else>
        <div class="rounded-xl border border-border bg-card p-4 shadow-sm">
          <div class="mb-3 flex items-center justify-between">
            <h3 class="text-sm font-semibold">
              {{ $t('vpn.logs.title') }}
            </h3>
            <FaButton size="sm" variant="outline" :loading="logsLoading" @click="refreshLogs">
              {{ $t('common.refresh') }}
            </FaButton>
          </div>
          <YdLogViewer :logs="logsText" height="55vh" :loading="logsLoading" />
        </div>
      </template>
    </template>

    <!-- 卸载确认 -->
    <YdDangerDelete
      v-model:visible="uninstallVisible"
      :title="$t('vpn.uninstall.title', { project: status?.project || '' })"
      :name="status?.project || ''"
      :options="UNINSTALL_OPTS"
      :loading="uninstalling"
      @confirm="doUninstall"
    />
  </div>
</template>
