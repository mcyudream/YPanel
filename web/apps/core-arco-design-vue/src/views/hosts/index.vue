<script setup lang="ts">
import type { HostNodeState, HostNodeStatus, HostRecord } from '@/api/modules/hosts'
import apiHosts from '@/api/modules/hosts'
import nodeApi, { type NodeItem } from '@/api/modules/node'
import { i18n, tr } from '@/locales'

defineOptions({
  name: 'HostsIndex',
})

interface HostForm {
  ip: string
  hostnames: string
  comment: string
  enabled: boolean
  sort: number
}

function emptyForm(): HostForm {
  return { ip: '', hostnames: '', comment: '', enabled: true, sort: 0 }
}

const nodes = ref<NodeItem[]>([])
const records = ref<HostRecord[]>([])
const loading = ref(false)
const statuses = ref<HostNodeStatus[]>([])
const statusLoading = ref(false)

const recordModal = ref(false)
const editingId = ref(0)
const saving = ref(false)
const form = ref<HostForm>(emptyForm())

const applyModal = ref(false)
const applying = ref(false)
const selectedNodes = ref<string[]>([])

const nodeName = (id: string) => nodes.value.find(n => n.id === id)?.name ?? id

const nodeOptions = computed(() => nodes.value.map(n => ({
  label: `${n.name}${n.remote ? '' : i18n.global.t('hosts.localNode')}${n.online ? '' : i18n.global.t('hosts.offlineNode')}`,
  value: n.id,
  disabled: !n.online,
})))

const stateCls: Record<HostNodeState, string> = {
  match: 'bg-emerald-500/10 text-emerald-600',
  notApplied: 'bg-amber-500/10 text-amber-600',
  drift: 'bg-red-500/10 text-red-600',
  offline: 'bg-muted text-muted-foreground',
  error: 'bg-red-500/10 text-red-600',
}

async function loadNodes() {
  try {
    nodes.value = await nodeApi.list()
  }
  catch {}
}

async function load() {
  loading.value = true
  try {
    records.value = await apiHosts.list()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('hosts.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function loadStatus() {
  statusLoading.value = true
  try {
    statuses.value = await apiHosts.status()
  }
  catch {}
  finally {
    statusLoading.value = false
  }
}

function openCreate() {
  editingId.value = 0
  form.value = emptyForm()
  recordModal.value = true
}

function openEdit(r: HostRecord) {
  editingId.value = r.id
  form.value = { ip: r.ip, hostnames: r.hostnames, comment: r.comment, enabled: r.enabled, sort: r.sort }
  recordModal.value = true
}

function formError(): string {
  if (!form.value.ip.trim()) {
    return i18n.global.t('hosts.requireIp')
  }
  if (!form.value.hostnames.trim()) {
    return i18n.global.t('hosts.requireHostnames')
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
    await apiHosts.save({
      id: editingId.value || undefined,
      ip: form.value.ip.trim(),
      hostnames: form.value.hostnames.trim(),
      comment: form.value.comment.trim(),
      enabled: form.value.enabled,
      sort: Number(form.value.sort) || 0,
    })
    useFaToast().success(editingId.value ? i18n.global.t('hosts.updated') : i18n.global.t('hosts.created'))
    recordModal.value = false
    await Promise.all([load(), loadStatus()])
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('hosts.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

async function toggle(r: HostRecord) {
  try {
    await (r.enabled ? apiHosts.disable(r.id) : apiHosts.enable(r.id))
    useFaToast().success(r.enabled ? i18n.global.t('hosts.disabledDone') : i18n.global.t('hosts.enabledDone'))
    await Promise.all([load(), loadStatus()])
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('hosts.opFailed'), { description: e?.message })
    await Promise.all([load(), loadStatus()])
  }
}

function remove(r: HostRecord) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('hosts.deleteTitle'),
    content: i18n.global.t('hosts.deleteConfirm', { hostnames: r.hostnames, ip: r.ip }),
    onConfirm: async () => {
      try {
        await apiHosts.remove(r.id)
        useFaToast().success(i18n.global.t('hosts.deleted'))
        await Promise.all([load(), loadStatus()])
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('hosts.deleteFailed'), { description: e?.message })
        await load()
      }
    },
  })
}

function openApply() {
  selectedNodes.value = []
  applyModal.value = true
}

function toggleNode(id: string) {
  const i = selectedNodes.value.indexOf(id)
  if (i >= 0) {
    selectedNodes.value.splice(i, 1)
  }
  else {
    selectedNodes.value.push(id)
  }
}

async function doApply() {
  if (!selectedNodes.value.length) {
    useFaToast().warning(i18n.global.t('hosts.requireNodes'))
    return
  }
  applying.value = true
  try {
    const results = await apiHosts.apply(selectedNodes.value)
    const fails = results.filter(r => !r.ok)
    if (fails.length) {
      useFaToast().warning(i18n.global.t('hosts.partialFailed', { f: fails.length, t: results.length }), { description: fails.map(f => i18n.global.t('hosts.failLine', { node: nodeName(f.nodeId), message: f.message })).join('\n'), duration: 10000 })
    }
    else {
      useFaToast().success(i18n.global.t('hosts.appliedTo', { n: results.length }))
    }
    applyModal.value = false
    await loadStatus()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('hosts.applyFailed'), { description: e?.message })
  }
  finally {
    applying.value = false
  }
}

function reapply(nodeId: string) {
  applying.value = true
  apiHosts.apply([nodeId]).then((results) => {
    if (results[0]?.ok) {
      useFaToast().success(i18n.global.t('hosts.reapplied'))
    }
    else {
      useFaToast().error(i18n.global.t('hosts.applyFailed'), { description: results[0]?.message })
    }
  }).catch((e: any) => {
    useFaToast().error(i18n.global.t('hosts.applyFailed'), { description: e?.message })
  }).finally(async () => {
    applying.value = false
    await loadStatus()
  })
}

function removeTarget(st: HostNodeStatus) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('hosts.unmanageTitle'),
    content: i18n.global.t('hosts.unmanageConfirm', { node: nodeName(st.nodeId) }),
    onConfirm: async () => {
      try {
        await apiHosts.removeNode(st.nodeId)
        useFaToast().success(i18n.global.t('hosts.unmanaged'))
        await loadStatus()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('hosts.unmanageFailed'), { description: e?.message })
      }
    },
  })
}

onMounted(() => {
  loadNodes()
  load()
  loadStatus()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="book-user" :size="24" />
          <span>Hosts</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('hosts.desc') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" :loading="loading" @click="load">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('common.refresh') }}
        </FaButton>
        <FaButton size="sm" @click="openApply">
          <FaIcon name="i-lucide:send" class="mr-1" /> {{ $t('hosts.applyToNodes') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('hosts.createRecord') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 托管节点状态 -->
      <div class="rounded-lg border p-3">
        <div class="mb-2 flex items-center justify-between">
          <div class="text-sm font-medium">
            {{ $t('hosts.managedNodes') }}
          </div>
          <FaButton variant="ghost" size="icon-sm" :loading="statusLoading" :title="$t('hosts.refreshStatus')" @click="loadStatus">
            <FaIcon name="i-lucide:refresh-cw" class="text-sm" />
          </FaButton>
        </div>
        <div v-if="!statuses.length" class="text-xs text-muted-foreground">
          {{ $t('hosts.noManaged') }}
        </div>
        <div v-else class="flex flex-wrap gap-2">
          <div
            v-for="st in statuses"
            :key="st.nodeId"
            class="flex items-center gap-2 rounded-full border px-3 py-1 text-xs"
            :title="st.message || ''"
          >
            <span class="font-medium">{{ nodeName(st.nodeId) }}</span>
            <span class="rounded-full px-2 py-0.5" :class="stateCls[st.state]">{{ tr(`hosts.state.${st.state}`, st.state) }}</span>
            <button
              v-if="st.state !== 'match' && st.state !== 'offline'"
              class="text-blue-600 hover:underline"
              :disabled="applying"
              @click="reapply(st.nodeId)"
            >
              {{ $t('hosts.reapply') }}
            </button>
            <button class="text-red-600 hover:underline" @click="removeTarget(st)">
              {{ $t('common.remove') }}
            </button>
          </div>
        </div>
      </div>

      <!-- 解析记录表 -->
      <div class="mt-4 overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">IP</th>
              <th class="px-3 py-2">{{ $t('hosts.hostnameCol') }}</th>
              <th class="px-3 py-2">{{ $t('common.status') }}</th>
              <th class="px-3 py-2">{{ $t('common.remark') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!records.length">
              <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('hosts.noRecords') }}
              </td>
            </tr>
            <tr v-for="r in records" :key="r.id" class="border-t hover:bg-accent/30">
              <td class="px-3 py-2 font-mono text-xs">
                {{ r.ip }}
              </td>
              <td class="px-3 py-2">
                <span class="font-mono text-xs">{{ r.hostnames }}</span>
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
      <div class="mt-3 text-xs text-muted-foreground">
        {{ $t('hosts.footNote') }}
      </div>
    </FaPageMain>

    <!-- 记录编辑弹窗 -->
    <FaModal v-model="recordModal" :title="editingId ? $t('hosts.editRecord') : $t('hosts.createRecord')" :destroy-on-close="true" class="lg:w-[560px]">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">IP</span>
          <FaInput v-model="form.ip" :placeholder="$t('hosts.ipPlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-start gap-3">
          <span class="w-20 shrink-0 pt-2 text-sm text-muted-foreground">{{ $t('hosts.hostnameCol') }}</span>
          <div class="flex flex-1 flex-col gap-1">
            <FaInput v-model="form.hostnames" :placeholder="$t('hosts.hostnamesPlaceholder')" class="flex-1" />
            <span class="text-xs text-muted-foreground">{{ $t('hosts.hostnamesHint') }}</span>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="form.comment" :placeholder="$t('hosts.optional')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('hosts.sortStatus') }}</span>
          <FaInput v-model="form.sort" placeholder="0" class="w-20" />
          <FaButton :variant="form.enabled ? 'default' : 'outline'" size="sm" @click="form.enabled = !form.enabled">
            {{ form.enabled ? $t('common.enabled') : $t('common.disabled') }}
          </FaButton>
          <span class="text-xs text-muted-foreground">{{ $t('hosts.smallFirst') }}</span>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="recordModal = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="saving" @click="doSave">
          {{ editingId ? $t('hosts.saveApply') : $t('hosts.createApply') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 应用到节点弹窗 -->
    <FaModal v-model="applyModal" :title="$t('hosts.applyToNodes')" :destroy-on-close="true" class="lg:w-[460px]">
      <div class="flex flex-col gap-3">
        <p class="text-sm text-muted-foreground">
          {{ $t('hosts.applyDesc') }}
        </p>
        <div class="flex flex-wrap gap-2">
          <FaButton
            v-for="n in nodeOptions"
            :key="n.value"
            size="sm"
            :variant="selectedNodes.includes(n.value) ? 'default' : 'outline'"
            :disabled="n.disabled"
            @click="toggleNode(n.value)"
          >
            {{ n.label }}
          </FaButton>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="applyModal = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="applying" @click="doApply">
          {{ $t('hosts.applyNodes', { n: selectedNodes.length }) }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
