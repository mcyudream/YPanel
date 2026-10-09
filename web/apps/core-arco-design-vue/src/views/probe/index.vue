<script setup lang="ts">
import type { ContainerOption, MonitorProbe, MonitorProbeInput, SiteOption } from '@/api/modules/probe'

import containerApi from '@/api/modules/container'
import siteApi from '@/api/modules/site'
import { probeApi } from '@/api/modules/probe'
import YdDangerDelete from '@/components/YdDangerDelete/index.vue'
import { i18n, tr } from '@/locales'

defineOptions({
  name: 'ProbeIndex',
})

type ProbeType = MonitorProbeInput['targetType']

const TYPE_OPTIONS: { key: ProbeType, label: string, desc: string }[] = [
  { key: 'site', label: 'monitor.probe.types.site', desc: 'monitor.probe.typeDescs.site' },
  { key: 'container', label: 'monitor.probe.types.container', desc: 'monitor.probe.typeDescs.container' },
  { key: 'http', label: 'monitor.probe.types.http', desc: 'monitor.probe.typeDescs.http' },
  { key: 'tcp', label: 'monitor.probe.types.tcp', desc: 'monitor.probe.typeDescs.tcp' },
]

function typeLabel(v: string) {
  return tr(`monitor.probe.typeLabels.${v}`, v)
}

const toast = useFaToast()

const probes = ref<MonitorProbe[]>([])
const loading = ref(false)
const sites = ref<SiteOption[]>([])
const containers = ref<ContainerOption[]>([])

const stats = computed(() => {
  const up = probes.value.filter(p => p.enabled && p.status === 'up').length
  const down = probes.value.filter(p => p.enabled && p.status === 'down').length
  const paused = probes.value.filter(p => !p.enabled).length
  return { total: probes.value.length, up, down, paused }
})

async function refresh(silent = true) {
  if (!silent) {
    loading.value = true
  }
  try {
    probes.value = await probeApi.list()
  }
  catch (e: any) {
    if (!silent) {
      toast.error(i18n.global.t('monitor.probe.loadFailed'), { description: e?.message })
    }
  }
  finally {
    loading.value = false
  }
}

// ---------- 新建 / 编辑 ----------

const modalVisible = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)
const testing = ref(false)

const form = reactive<Required<MonitorProbeInput>>({
  name: '',
  targetType: 'site',
  targetRef: '',
  method: 'GET',
  expectStatus: '2xx',
  keyword: '',
  intervalSec: 60,
  timeoutSec: 5,
  retries: 3,
  webhookUrl: '',
  webhookType: 'generic',
})

// 各类型默认模板：开启即带一套可直接用的探测参数
function applyTemplate(t: ProbeType) {
  form.targetType = t
  form.targetRef = ''
  form.method = 'GET'
  form.expectStatus = '2xx'
  form.keyword = ''
  form.intervalSec = 60
  form.timeoutSec = 5
  form.retries = 3
}

async function openCreate() {
  editingId.value = null
  Object.assign(form, {
    name: '', targetType: 'site', targetRef: '', method: 'GET', expectStatus: '2xx',
    keyword: '', intervalSec: 60, timeoutSec: 5, retries: 3, webhookUrl: '', webhookType: 'generic',
  })
  await loadOptions()
  modalVisible.value = true
}

async function openEdit(row: MonitorProbe) {
  editingId.value = row.id
  Object.assign(form, {
    name: row.name, targetType: row.targetType, targetRef: row.targetRef,
    method: row.method || 'GET', expectStatus: row.expectStatus || '2xx', keyword: row.keyword || '',
    intervalSec: row.intervalSec, timeoutSec: row.timeoutSec, retries: row.retries,
    webhookUrl: row.webhookUrl || '', webhookType: row.webhookType || 'generic',
  })
  await loadOptions()
  modalVisible.value = true
}

async function loadOptions() {
  try {
    const [s, c] = await Promise.all([siteApi.list(), containerApi.list()])
    sites.value = (s as unknown as SiteOption[]) || []
    containers.value = (c as unknown as ContainerOption[]) || []
  }
  catch {
    // 下拉数据加载失败不阻塞表单（自定义类型不受影响）
  }
}

function formPayload(): MonitorProbeInput {
  return {
    name: form.name.trim(),
    targetType: form.targetType,
    targetRef: form.targetRef.trim(),
    method: form.method,
    expectStatus: form.expectStatus.trim(),
    keyword: form.keyword.trim(),
    intervalSec: form.intervalSec,
    timeoutSec: form.timeoutSec,
    retries: form.retries,
    webhookUrl: form.webhookUrl.trim(),
    webhookType: 'generic',
  }
}

async function save(enable: boolean) {
  if (!form.name.trim()) {
    toast.error(i18n.global.t('monitor.probe.nameRequired'))
    return
  }
  if (!form.targetRef.trim()) {
    toast.error(form.targetType === 'site' ? i18n.global.t('monitor.probe.targetRequiredSite') : form.targetType === 'container' ? i18n.global.t('monitor.probe.targetRequiredContainer') : i18n.global.t('monitor.probe.targetRequiredRef'))
    return
  }
  saving.value = true
  try {
    let id: number
    if (editingId.value) {
      const row = await probeApi.update(editingId.value, formPayload())
      id = row.id
    }
    else {
      const row = await probeApi.create(formPayload())
      id = row.id
    }
    if (enable) {
      await probeApi.setEnabled(id, true)
    }
    toast.success(editingId.value ? i18n.global.t('monitor.probe.updated') : enable ? i18n.global.t('monitor.probe.createdEnabled') : i18n.global.t('monitor.probe.createdPaused'), { description: enable ? i18n.global.t('monitor.probe.descNextCycle') : i18n.global.t('monitor.probe.descEnableAnytime') })
    modalVisible.value = false
    await refresh()
  }
  catch (e: any) {
    toast.error(i18n.global.t('monitor.probe.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

async function runTest() {
  testing.value = true
  try {
    const res = await probeApi.test(formPayload())
    if (res.ok) {
      toast.success(i18n.global.t('monitor.probe.testOk'), { description: res.detail })
    }
    else {
      toast.error(i18n.global.t('monitor.probe.testFailed'), { description: res.detail })
    }
  }
  catch (e: any) {
    toast.error(i18n.global.t('monitor.probe.testFailed'), { description: e?.message })
  }
  finally {
    testing.value = false
  }
}

// ---------- 列表操作 ----------

async function toggleRow(row: MonitorProbe, enabled: boolean) {
  try {
    await probeApi.setEnabled(row.id, enabled)
    row.enabled = enabled
    row.status = enabled ? row.status : 'paused'
    toast.success(enabled ? i18n.global.t('monitor.probe.enabledToast', { name: row.name }) : i18n.global.t('monitor.probe.disabledToast', { name: row.name }))
  }
  catch (e: any) {
    toast.error(i18n.global.t('monitor.probe.operateFailed'), { description: e?.message })
  }
}

async function testRow(row: MonitorProbe) {
  testing.value = true
  try {
    const res = await probeApi.test({
      name: row.name, targetType: row.targetType, targetRef: row.targetRef,
      method: row.method, expectStatus: row.expectStatus, keyword: row.keyword,
      intervalSec: row.intervalSec, timeoutSec: row.timeoutSec, retries: row.retries,
    })
    if (res.ok) {
      toast.success(i18n.global.t('monitor.probe.testOkNamed', { name: row.name }), { description: res.detail })
    }
    else {
      toast.error(i18n.global.t('monitor.probe.testFailedNamed', { name: row.name }), { description: res.detail })
    }
  }
  catch (e: any) {
    toast.error(i18n.global.t('monitor.probe.testFailed'), { description: e?.message })
  }
  finally {
    testing.value = false
  }
}

const deleteVisible = ref(false)
const deleteTarget = ref<MonitorProbe | null>(null)
const deleting = ref(false)

function askDelete(row: MonitorProbe) {
  deleteTarget.value = row
  deleteVisible.value = true
}

async function doDelete() {
  if (!deleteTarget.value) {
    return
  }
  deleting.value = true
  try {
    await probeApi.remove(deleteTarget.value.id)
    toast.success(i18n.global.t('monitor.probe.deleted'))
    deleteVisible.value = false
    await refresh()
  }
  catch (e: any) {
    toast.error(i18n.global.t('monitor.probe.deleteFailed'), { description: e?.message })
  }
  finally {
    deleting.value = false
  }
}

// ---------- 展示辅助 ----------

function statusClass(status: string) {
  return status === 'up' ? 'bg-emerald-500' : status === 'down' ? 'bg-red-500' : 'bg-muted-foreground/40'
}

function statusLabel(row: MonitorProbe) {
  if (!row.enabled) {
    return i18n.global.t('monitor.probe.statusPaused')
  }
  return row.status === 'up' ? i18n.global.t('monitor.probe.statusUp') : row.status === 'down' ? i18n.global.t('monitor.probe.statusDown') : i18n.global.t('monitor.probe.statusPending')
}

function fmtTime(t: string | null) {
  if (!t) {
    return '—'
  }
  return new Date(t).toLocaleString()
}

function siteDomain(name: string) {
  return sites.value.find(s => s.name === name)?.domain || ''
}

let timer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  refresh(false)
  timer = setInterval(refresh, 30000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<template>
  <div class="flex flex-col gap-4 p-4">
    <!-- 概览条 -->
    <div class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-xl border border-border bg-card px-4 py-3 shadow-sm">
      <span class="flex items-center gap-1.5 text-sm font-medium">
        <span class="i-lucide-radar inline-block size-4 text-primary" /> {{ $t('monitor.probe.title') }}
      </span>
      <span class="bg-accent text-muted-foreground rounded-full px-2 py-0.5 text-xs">{{ $t('monitor.probe.totalCount', { n: stats.total }) }}</span>
      <span class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">{{ $t('monitor.probe.upCount', { n: stats.up }) }}</span>
      <span class="rounded-full bg-red-500/10 px-2 py-0.5 text-xs text-red-600">{{ $t('monitor.probe.downCount', { n: stats.down }) }}</span>
      <span class="bg-accent text-muted-foreground rounded-full px-2 py-0.5 text-xs">{{ $t('monitor.probe.pausedCount', { n: stats.paused }) }}</span>
      <span class="text-muted-foreground ml-auto text-xs">{{ $t('monitor.probe.hint') }}</span>
      <FaButton size="sm" :loading="loading" @click="refresh(false)">
        <span class="i-lucide-refresh-cw mr-1 inline-block size-3.5" /> {{ $t('common.refresh') }}
      </FaButton>
      <FaButton size="sm" @click="openCreate">
        <span class="i-lucide-plus mr-1 inline-block size-3.5" /> {{ $t('monitor.probe.create') }}
      </FaButton>
    </div>

    <!-- 列表 -->
    <div class="rounded-xl border border-border bg-card p-5 shadow-sm">
      <div v-if="probes.length" class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead>
            <tr class="text-muted-foreground border-b border-border text-xs">
              <th class="px-2 py-1.5 font-medium">{{ $t('common.status') }}</th>
              <th class="px-2 py-1.5 font-medium">{{ $t('common.name') }}</th>
              <th class="px-2 py-1.5 font-medium">{{ $t('common.type') }}</th>
              <th class="px-2 py-1.5 font-medium">{{ $t('monitor.probe.target') }}</th>
              <th class="px-2 py-1.5 font-medium">{{ $t('monitor.probe.interval') }}</th>
              <th class="px-2 py-1.5 font-medium">{{ $t('monitor.probe.lastCheck') }}</th>
              <th class="px-2 py-1.5 font-medium">{{ $t('monitor.probe.lastError') }}</th>
              <th class="px-2 py-1.5 text-right font-medium">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in probes" :key="row.id" class="border-b border-border/50 last:border-0">
              <td class="px-2 py-2">
                <span class="flex items-center gap-1.5 whitespace-nowrap text-xs">
                  <span class="inline-block size-2 rounded-full" :class="statusClass(row.enabled ? row.status : 'paused')" />
                  {{ statusLabel(row) }}
                </span>
              </td>
              <td class="px-2 py-2 font-medium">{{ row.name }}</td>
              <td class="px-2 py-2">
                <span class="bg-primary/10 text-primary rounded-full px-2 py-0.5 text-xs">{{ typeLabel(row.targetType) }}</span>
              </td>
              <td class="max-w-64 truncate px-2 py-2 font-mono text-xs" :title="row.targetType === 'site' ? (siteDomain(row.targetRef) || row.targetRef) : row.targetRef">
                {{ row.targetType === 'site' && siteDomain(row.targetRef) ? `${row.targetRef}（${siteDomain(row.targetRef)}）` : row.targetRef }}
              </td>
              <td class="text-muted-foreground whitespace-nowrap px-2 py-2 text-xs">{{ row.intervalSec }}s</td>
              <td class="text-muted-foreground whitespace-nowrap px-2 py-2 text-xs">{{ fmtTime(row.lastCheckedAt) }}</td>
              <td class="max-w-48 truncate px-2 py-2 text-xs text-red-500" :title="row.lastError">{{ row.lastError || '—' }}</td>
              <td class="whitespace-nowrap px-2 py-2 text-right">
                <FaSwitch
                  :model-value="row.enabled"
                  @update:model-value="(v: boolean | undefined) => toggleRow(row, !!v)"
                />
                <FaButton size="sm" variant="outline" class="ml-2" :disabled="testing" :title="$t('monitor.probe.testRunTitle')" @click="testRow(row)">
                  {{ $t('monitor.probe.testRun') }}
                </FaButton>
                <FaButton size="sm" variant="outline" class="ml-1" @click="openEdit(row)">
                  {{ $t('common.edit') }}
                </FaButton>
                <FaButton size="sm" variant="outline" class="ml-1 text-red-500" @click="askDelete(row)">
                  {{ $t('common.delete') }}
                </FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="text-muted-foreground flex flex-col items-center gap-2 py-10 text-sm">
        <span class="i-lucide-radar inline-block size-8 opacity-40" />
        <span>{{ $t('monitor.probe.empty') }}</span>
        <FaButton size="sm" variant="outline" @click="openCreate">
          {{ $t('monitor.probe.createFirst') }}
        </FaButton>
      </div>
    </div>

    <!-- 新建 / 编辑弹窗 -->
    <FaModal v-model="modalVisible" :title="editingId ? $t('monitor.probe.editTitle') : $t('monitor.probe.create')" :destroy-on-close="true" class="w-[560px]">
      <div class="flex flex-col gap-3">
        <div class="grid grid-cols-2 gap-2">
          <button
            v-for="t in TYPE_OPTIONS"
            :key="t.key"
            class="rounded-lg border p-3 text-left transition-colors"
            :class="form.targetType === t.key ? 'border-primary bg-primary/5' : 'border-border hover:bg-accent/50'"
            @click="applyTemplate(t.key)"
          >
            <div class="text-sm font-medium" :class="form.targetType === t.key ? 'text-primary' : ''">
              {{ $t(t.label) }}
            </div>
            <div class="text-muted-foreground mt-0.5 text-xs leading-4">
              {{ $t(t.desc) }}
            </div>
          </button>
        </div>

        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.name') }}</span>
          <FaInput v-model="form.name" :placeholder="$t('monitor.probe.namePlaceholder')" class="flex-1" />
        </div>

        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('monitor.probe.target') }}</span>
          <template v-if="form.targetType === 'site'">
            <select v-model="form.targetRef" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
              <option value="" disabled>
                {{ $t('monitor.probe.selectSite') }}
              </option>
              <option v-for="s in sites" :key="s.name" :value="s.name">
                {{ s.name }}{{ s.domain ? `（${s.domain}）` : '' }}
              </option>
            </select>
          </template>
          <template v-else-if="form.targetType === 'container'">
            <select v-model="form.targetRef" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
              <option value="" disabled>
                {{ $t('monitor.probe.selectContainer') }}
              </option>
              <option v-for="c in containers" :key="c.id" :value="c.name">
                {{ c.name }}（{{ c.state }}）
              </option>
            </select>
          </template>
          <template v-else-if="form.targetType === 'http'">
            <FaInput v-model="form.targetRef" placeholder="https://example.com/health" class="flex-1" />
          </template>
          <template v-else>
            <FaInput v-model="form.targetRef" :placeholder="$t('monitor.probe.targetTcpPlaceholder')" class="flex-1" />
          </template>
        </div>

        <template v-if="form.targetType === 'http' || form.targetType === 'site'">
          <div class="flex items-center gap-3">
            <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('monitor.probe.methodExpect') }}</span>
            <select v-model="form.method" class="h-9 w-24 rounded-md border border-input bg-background px-2 text-sm outline-none">
              <option>GET</option>
              <option>HEAD</option>
              <option>POST</option>
            </select>
            <FaInput v-model="form.expectStatus" :placeholder="$t('monitor.probe.expectStatusPlaceholder')" class="flex-1" />
          </div>
          <div class="flex items-center gap-3">
            <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('monitor.probe.responseContains') }}</span>
            <FaInput v-model="form.keyword" :placeholder="$t('monitor.probe.keywordPlaceholder')" class="flex-1" />
          </div>
        </template>

        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('monitor.probe.checkParams') }}</span>
          <div class="flex flex-1 items-center gap-2 text-xs text-muted-foreground">
            <span>{{ $t('monitor.probe.interval') }}</span><FaInput v-model.number="form.intervalSec" class="w-20" />
            <span>{{ $t('monitor.probe.secTimeout') }}</span><FaInput v-model.number="form.timeoutSec" class="w-16" />
            <span>{{ $t('monitor.probe.secRetries') }}</span><FaInput v-model.number="form.retries" class="w-16" />
            <span>{{ $t('monitor.probe.timesDown') }}</span>
          </div>
        </div>

        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">Webhook</span>
          <FaInput v-model="form.webhookUrl" :placeholder="$t('monitor.probe.webhookPlaceholder')" class="flex-1" />
        </div>

        <div class="text-muted-foreground text-xs leading-5">
          {{ $t('monitor.probe.templateHint', { mode: $t(form.targetType === 'container' ? 'monitor.probe.tmplContainer' : 'monitor.probe.tmplHttp') }) }}
        </div>
      </div>
      <template #footer>
        <div class="flex items-center gap-2">
          <FaButton variant="outline" size="sm" :loading="testing" @click="runTest">
            {{ $t('monitor.probe.testOnce') }}
          </FaButton>
          <div class="flex-1" />
          <FaButton variant="outline" size="sm" :disabled="saving" @click="modalVisible = false">
            {{ $t('common.cancel') }}
          </FaButton>
          <FaButton variant="outline" size="sm" :loading="saving" @click="save(false)">
            {{ $t('monitor.probe.saveDisabled') }}
          </FaButton>
          <FaButton size="sm" :loading="saving" @click="save(true)">
            {{ $t('monitor.probe.saveEnable') }}
          </FaButton>
        </div>
      </template>
    </FaModal>

    <!-- 删除确认 -->
    <YdDangerDelete
      v-model:visible="deleteVisible"
      :title="$t('monitor.probe.deleteTitle', { name: deleteTarget?.name || '' })"
      :name="deleteTarget?.name || ''"
      :loading="deleting"
      @confirm="doDelete"
    />
  </div>
</template>
