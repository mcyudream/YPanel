<script setup lang="ts">
import type { ContainerItem } from '@/api/modules/container'
import { LabelComposeProject } from '@/api/modules/container'
import apiContainer from '@/api/modules/container'
import { dockerExtApi } from '@/api/modules/dockerext'
import { dockerImgApi } from '@/api/modules/dockerenv'
import FileEditorWorkspace from '@/views/file_management/editor/Workspace.vue'
import YdDangerDelete from '@/components/YdDangerDelete/index.vue'
import ContainerForm from '../components/ContainerForm.vue'
import DockerInstallWizard from '../components/DockerInstallWizard.vue'
import { i18n, tr } from '@/locales'
import { useYwEmbed } from '@/views/desktop/embed'
import { useVpnAccess } from '@/composables/useVpnAccess'

// 容器列表（M23）：批量操作 / 状态与项目过滤 / 品牌 logo / 快速操作。
const router = useRouter()
// 桌面承载时：详情下钻开新窗（router.push 会顶掉 /desktop 路由）；经典模式保持路由跳转
const ywEmbed = useYwEmbed()

function openContainerDetailWin(containerId: string, name?: string) {
  if (ywEmbed) {
    // launchOptions.name 供 AI 焦点窗口感知取用（title 同步用人名可读）
    ywEmbed.openApp('container-detail', { title: name ? i18n.global.t('container.common.containerTitle', { name }) : i18n.global.t('container.common.containerTitle', { name: containerId.slice(0, 12) }), launchOptions: { id: containerId, name: name || '' } })
    return
  }
  router.push(`/container/detail/${containerId}`)
}

function openAppDetailWin(projectName: string) {
  if (ywEmbed) {
    ywEmbed.openApp('container-app-detail', { title: projectName, launchOptions: { project: projectName } })
    return
  }
  router.push(`/container/app/${projectName}`)
}
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()

const containers = ref<ContainerItem[]>([])
const loading = ref(false)
const disabled = ref(false)
const disabledMsg = ref('')
// M52：Docker 未安装引导——一键安装向导
const installVisible = ref(false)
function onInstalled() {
  disabled.value = false
  disabledMsg.value = ''
  load(true)
}
const acting = ref('')
const autoRefresh = ref(true)
const search = ref('')
const stateFilter = ref<'all' | 'running' | 'stopped'>('all')
const projectFilter = ref('all')

const selected = ref<Set<string>>(new Set())

const composeProjects = computed(() => {
  const set = new Set<string>()
  for (const c of containers.value) {
    const p = c.labels?.[LabelComposeProject]
    if (p) {
      set.add(p)
    }
  }
  return [...set].sort()
})

const filtered = computed(() => {
  const kw = search.value.trim().toLowerCase()
  return containers.value.filter((c) => {
    if (stateFilter.value === 'running' && c.state !== 'running') {
      return false
    }
    if (stateFilter.value === 'stopped' && c.state === 'running') {
      return false
    }
    if (projectFilter.value !== 'all' && c.labels?.[LabelComposeProject] !== projectFilter.value) {
      return false
    }
    if (kw && !c.name.toLowerCase().includes(kw) && !c.image.toLowerCase().includes(kw) && !c.id.toLowerCase().includes(kw)) {
      return false
    }
    return true
  })
})

const allChecked = computed(() => filtered.value.length > 0 && filtered.value.every(c => selected.value.has(c.id)))

// ---- 分页（筛选/搜索变化自动回第 1 页） ----
const page = ref(1)
const size = ref(20)
const paged = computed(() => filtered.value.slice((page.value - 1) * size.value, page.value * size.value))
watch([search, stateFilter, projectFilter], () => {
  page.value = 1
})

function toggleAll() {
  if (allChecked.value) {
    selected.value.clear()
  }
  else {
    for (const c of filtered.value) {
      selected.value.add(c.id)
    }
  }
  selected.value = new Set(selected.value)
}

function toggle(id: string) {
  const next = new Set(selected.value)
  if (next.has(id)) {
    next.delete(id)
  }
  else {
    next.add(id)
  }
  selected.value = next
}

function projectOf(c: ContainerItem) {
  return c.labels?.[LabelComposeProject] || ''
}

const stateStyle: Record<string, string> = {
  running: 'text-emerald-600 bg-emerald-500/10',
  exited: 'text-muted-foreground bg-muted',
  paused: 'text-amber-600 bg-amber-500/10',
  created: 'text-blue-600 bg-blue-500/10',
  restarting: 'text-amber-600 bg-amber-500/10',
  dead: 'text-red-600 bg-red-500/10',
}

async function load(silent = false) {
  if (!silent) {
    loading.value = true
  }
  try {
    containers.value = await apiContainer.list()
    disabled.value = false
    // 清掉已消失的选中项
    const ids = new Set(containers.value.map(c => c.id))
    const next = new Set([...selected.value].filter(x => ids.has(x)))
    if (next.size !== selected.value.size) {
      selected.value = next
    }
  }
  catch (e: any) {
    if (e?.code === 5002) {
      disabled.value = true
      disabledMsg.value = e?.message || i18n.global.t('container.list.noDockerCurrent')
    }
    else {
      disabledMsg.value = e?.message || i18n.global.t('container.common.loadFailed')
    }
  }
  finally {
    loading.value = false
  }
}

async function action(c: ContainerItem, act: 'start' | 'stop' | 'restart') {
  acting.value = c.id + act
  try {
    await apiContainer.action(c.id, act)
    toast.success(i18n.global.t(act === 'start' ? 'container.common.started' : act === 'stop' ? 'container.common.stopped' : 'container.common.restarted', { name: c.name }))
    await load(true)
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.opFailed'), { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

// M35：提交为镜像（modal 输入镜像名）
const commitVisible = ref(false)
const commitBusy = ref(false)
const commitTarget = ref<ContainerItem | null>(null)
const commitImage = ref('')

function commitOne(c: ContainerItem) {
  commitTarget.value = c
  commitImage.value = `${c.name}:snapshot`
  commitVisible.value = true
}

async function doCommit() {
  if (!commitTarget.value || !commitImage.value) {
    return
  }
  commitBusy.value = true
  try {
    await dockerImgApi.commit(commitTarget.value.name, commitImage.value)
    toast.success(i18n.global.t('container.list.committedAsImage', { name: commitImage.value }))
    commitVisible.value = false
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.list.commitFailed'), { description: e?.message })
  }
  finally {
    commitBusy.value = false
  }
}

// M35：容器备份（named volumes 导出）
async function backupOne(c: ContainerItem) {
  acting.value = c.id + 'backup'
  try {
    const out = await dockerImgApi.containerBackup(c.name)
    if (out.volumes.length) {
      toast.success(i18n.global.t('container.list.backupDoneWithVolumes', { n: out.volumes.length, dir: out.dir }))
    }
    else {
      toast.success(i18n.global.t('container.list.backupDoneNoVolumes', { dir: out.dir }))
    }
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.list.backupFailed'), { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

// 危险删除（选项 + 名称确认）
const delVisible = ref(false)
const delTarget = ref<ContainerItem | null>(null)
const delOpts = computed(() => [
  { key: 'force', label: i18n.global.t('container.list.optForceLabel'), desc: i18n.global.t('container.list.optForceDesc') },
  { key: 'volumes', label: i18n.global.t('container.list.optVolumesLabel'), desc: i18n.global.t('container.list.optVolumesDesc') },
])

function removeOne(c: ContainerItem) {
  delTarget.value = c
  delVisible.value = true
}

async function doDelete(checked: Record<string, boolean>) {
  if (!delTarget.value) {
    return
  }
  try {
    await apiContainer.remove(delTarget.value.id, !!checked.force, !!checked.volumes)
    toast.success(i18n.global.t('container.list.deletedContainer', { name: delTarget.value.name }))
    delVisible.value = false
    await load(true)
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.deleteFailed'), { description: e?.message })
  }
}

const bulkRunning = ref(false)

async function bulkAction(act: 'start' | 'stop' | 'restart') {
  const list = containers.value.filter(c => selected.value.has(c.id))
  if (!list.length) {
    return
  }
  bulkRunning.value = true
  const results = await Promise.allSettled(list.map(c => apiContainer.action(c.id, act)))
  const ok = results.filter(r => r.status === 'fulfilled').length
  const failed = results.length - ok
  const label = i18n.global.t(act === 'start' ? 'common.start' : act === 'stop' ? 'common.stop' : 'common.restart')
  if (failed) {
    toast.warning(i18n.global.t('container.list.bulkPartial', { action: label, ok, failed }))
  }
  else {
    toast.success(i18n.global.t('container.list.bulkDone', { action: label, n: ok }))
  }
  await load(true)
  bulkRunning.value = false
}

async function batchRemove(list: ContainerItem[]) {
  bulkRunning.value = true
  try {
    const results = await Promise.allSettled(list.map(c => apiContainer.remove(c.id, true)))
    const ok = results.filter(r => r.status === 'fulfilled').length
    toast.success(i18n.global.t('container.list.bulkDeleted', { ok, total: list.length }))
    selected.value = new Set()
    await load(true)
  }
  finally {
    bulkRunning.value = false
  }
}

function bulkRemove() {
  const list = containers.value.filter(c => selected.value.has(c.id))
  if (!list.length) {
    return
  }
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.list.bulkDeleteTitle'),
    content: i18n.global.t('container.list.bulkDeleteConfirm', { n: list.length }),
    onConfirm: () => batchRemove(list),
  })
}

function openExec(c: ContainerItem) {
  fileEditorStore.openWorkspace(undefined, 'local', c.id)
  fileEditorStore.layout.terminalVisible = true
}

async function pruneContainers() {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.list.pruneStopped'),
    content: i18n.global.t('container.list.pruneConfirm'),
    onConfirm: async () => {
      try {
        const out = await dockerExtApi.pruneContainers()
        useFaToast().success(out)
        await load(true)
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.common.pruneFailed'), { description: e?.message })
      }
    },
  })
}

// 组网开启时：端口列给出经虚拟 IP 的一键跳转
const { portJumpUrl } = useVpnAccess()

function openPortJump(url: string) {
  if (url) {
    window.open(url, '_blank', 'noopener')
  }
}

const createVisible = ref(false)
// 编辑（删除重建式）：compose 管理的容器禁用入口
const editVisible = ref(false)
const editId = ref('')

function openEdit(c: ContainerItem) {
  editId.value = c.id
  editVisible.value = true
}

const isCompose = (c: ContainerItem) => !!c.labels?.[LabelComposeProject]

let timer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  load()
  timer = setInterval(() => {
    if (autoRefresh.value && !acting.value && !bulkRunning.value && !document.hidden) {
      load(true)
    }
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
    <FaPageMain>
  <div>
    <!-- 工具条 -->
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <YdDockerNodeSelect />
      <FaInput v-model="search" :placeholder="$t('container.list.searchPlaceholder')" class="h-8 w-52!" />
      <div class="flex overflow-hidden rounded-md border text-xs">
        <button
          v-for="s in [{ v: 'all', l: $t('common.all') }, { v: 'running', l: $t('container.state.running') }, { v: 'stopped', l: $t('container.list.stateStopped') }]" :key="s.v"
          type="button"
          class="px-2.5 py-1.5 transition-colors"
          :class="stateFilter === s.v ? 'bg-primary text-primary-foreground' : 'hover:bg-accent/50'"
          @click="stateFilter = s.v as any"
        >
          {{ s.l }}
        </button>
      </div>
      <YdSelect
        v-model="projectFilter"
        :options="[{ value: 'all', label: $t('container.list.allProjects') }, ...composeProjects.map(p => ({ value: p, label: p }))]"
        size="sm"
      />
      <label class="flex items-center gap-1 text-xs text-muted-foreground">
        <input v-model="autoRefresh" type="checkbox" class="accent-[var(--primary)]">
        {{ $t('container.list.autoRefresh') }}
      </label>
      <div class="ml-auto flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" :title="$t('common.refresh')" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton variant="outline" size="sm" @click="pruneContainers">
          {{ $t('container.list.pruneStopped') }}
        </FaButton>
        <FaButton v-auth="['docker:write']" size="sm" @click="createVisible = true">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('container.list.createContainer') }}
        </FaButton>
      </div>
    </div>

    <!-- 批量操作条 -->
    <div v-if="selected.size" class="mb-3 flex flex-wrap items-center gap-2 rounded-md border bg-primary/5 px-3 py-2">
      <span class="text-sm">{{ $t('container.list.selectedPrefix') }} <b class="tabular-nums">{{ selected.size }}</b> {{ $t('container.list.selectedSuffix') }}</span>
      <FaButton variant="outline" size="sm" :disabled="bulkRunning" @click="bulkAction('start')">
        {{ $t('container.list.bulkStart') }}
      </FaButton>
      <FaButton variant="outline" size="sm" :disabled="bulkRunning" @click="bulkAction('stop')">
        {{ $t('container.list.bulkStop') }}
      </FaButton>
      <FaButton variant="outline" size="sm" :disabled="bulkRunning" @click="bulkAction('restart')">
        {{ $t('container.list.bulkRestart') }}
      </FaButton>
      <FaButton variant="outline" size="sm" class="text-red-500!" :disabled="bulkRunning" @click="bulkRemove">
        {{ $t('container.list.bulkDelete') }}
      </FaButton>
      <FaButton variant="ghost" size="sm" @click="selected = new Set()">
        {{ $t('container.list.clearSelection') }}
      </FaButton>
    </div>

    <div v-if="disabled" class="mb-4 flex items-center gap-2 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
      <YdMorphIcon name="triangle-alert" :size="18" />
      {{ $t('container.list.noDockerDetail', { msg: disabledMsg }) }}
      <FaButton size="sm" class="ml-auto" @click="installVisible = true">
        <FaIcon name="i-lucide:download" class="mr-1" /> {{ $t('container.install.openWizard') }}
      </FaButton>
    </div>
    <div v-else-if="disabledMsg" class="mb-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
      {{ disabledMsg }}
    </div>
    <DockerInstallWizard v-model="installVisible" @installed="onInstalled" />

    <!-- 列表 -->
    <div class="overflow-x-auto rounded-lg border">
      <table class="w-full min-w-200 text-sm">
        <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
          <tr>
            <th class="w-9 px-3 py-2">
              <input type="checkbox" :checked="allChecked" class="accent-[var(--primary)]" @change="toggleAll">
            </th>
            <th class="px-3 py-2">{{ $t('common.name') }}</th>
            <th class="px-3 py-2">{{ $t('container.common.image') }}</th>
            <th class="px-3 py-2">{{ $t('common.status') }}</th>
            <th class="hidden px-3 py-2 lg:table-cell">{{ $t('container.common.ports') }}</th>
            <th class="hidden px-3 py-2 xl:table-cell">{{ $t('container.list.project') }}</th>
            <th class="px-3 py-2 text-right">{{ $t('container.list.quickActions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !filtered.length">
            <td colspan="7" class="px-3 py-10 text-center text-muted-foreground">
              {{ $t('common.loading') }}
            </td>
          </tr>
          <tr v-else-if="!filtered.length">
            <td colspan="7" class="px-3 py-10 text-center text-muted-foreground">
              {{ $t('container.common.noContainers') }}
            </td>
          </tr>
          <tr v-for="c in paged" :key="c.id" class="border-t transition-colors hover:bg-accent/30" :class="selected.has(c.id) ? 'bg-primary/5' : ''">
            <td class="px-3 py-2">
              <input type="checkbox" :checked="selected.has(c.id)" class="accent-[var(--primary)]" @change="toggle(c.id)">
            </td>
            <td class="max-w-64 px-3 py-2">
              <div class="flex items-center gap-2">
                <YdAppIcon :name="c.image" :size="20" />
                <button type="button" class="cursor-pointer truncate font-mono text-[13px] font-medium hover:text-primary" :title="c.name" @click="openContainerDetailWin(c.id, c.name)">
                  {{ c.name }}
                </button>
              </div>
            </td>
            <td class="max-w-48 truncate px-3 py-2 font-mono text-xs text-muted-foreground" :title="c.image">
              {{ c.image }}
            </td>
            <td class="px-3 py-2">
              <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs whitespace-nowrap" :class="stateStyle[c.state] || stateStyle.exited">
                <span class="inline-block size-1.5 rounded-full" :class="c.state === 'running' ? 'animate-pulse bg-current' : 'bg-current'" />
                {{ tr(`container.state.${c.state}`) }}<span v-if="c.status" class="hidden opacity-70 xl:inline">· {{ c.status }}</span>
              </span>
            </td>
            <td class="hidden px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell">
              <template v-if="c.ports?.some(p => p.hostPort)">
                <span v-for="(p, i) in c.ports.filter(x => x.hostPort)" :key="i" class="mr-2 inline-flex items-center gap-0.5 whitespace-nowrap">
                  {{ p.hostPort }}→{{ p.containerPort }}/{{ p.proto }}
                  <button
                    v-if="portJumpUrl(p.hostIp, p.hostPort, p.proto)" type="button"
                    class="cursor-pointer text-muted-foreground transition-colors hover:text-primary"
                    :title="$t('container.common.portJumpTip', { url: portJumpUrl(p.hostIp, p.hostPort, p.proto) })"
                    @click="openPortJump(portJumpUrl(p.hostIp, p.hostPort, p.proto))"
                  >
                    <FaIcon name="i-lucide:external-link" class="size-3 shrink-0" />
                  </button>
                </span>
              </template>
              <span v-else>—</span>
            </td>
            <td class="hidden px-3 py-2 xl:table-cell">
              <button
                v-if="projectOf(c)" type="button"
                class="cursor-pointer rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary hover:bg-primary/20"
                @click="openAppDetailWin(projectOf(c))"
              >
                {{ projectOf(c) }}
              </button>
              <span v-else class="text-xs text-muted-foreground">—</span>
            </td>
            <td class="px-3 py-2">
              <div class="flex items-center justify-end gap-1">
                <FaButton v-if="c.state !== 'running'" variant="outline" size="sm" :disabled="acting === c.id + 'start'" @click="action(c, 'start')">
                  {{ $t('common.start') }}
                </FaButton>
                <template v-else>
                  <FaButton variant="outline" size="sm" :disabled="acting === c.id + 'stop'" @click="action(c, 'stop')">
                    {{ $t('common.stop') }}
                  </FaButton>
                  <FaButton variant="outline" size="sm" :disabled="acting === c.id + 'restart'" @click="action(c, 'restart')">
                    {{ $t('common.restart') }}
                  </FaButton>
                </template>
                <FaButton v-if="c.state === 'running'" variant="ghost" size="icon-sm" :title="$t('container.list.terminalAndFiles')" @click="openExec(c)">
                  <FaIcon name="i-lucide:square-terminal" class="text-sm" />
                </FaButton>
                <FaButton variant="ghost" size="icon-sm" :title="$t('container.list.commitTitle')" :disabled="acting === c.id + 'commit'" @click="commitOne(c)">
                  <FaIcon name="i-lucide:camera" class="text-sm" />
                </FaButton>
                <FaButton variant="ghost" size="icon-sm" :title="$t('container.list.backupTitle')" :disabled="acting === c.id + 'backup'" @click="backupOne(c)">
                  <FaIcon name="i-lucide:package" class="text-sm" />
                </FaButton>
                <FaButton
                  variant="ghost" size="icon-sm"
                  :class="isCompose(c) ? 'cursor-not-allowed opacity-40' : ''"
                  :disabled="isCompose(c)" :title="isCompose(c) ? $t('container.common.managedByCompose') : $t('common.edit')"
                  @click="!isCompose(c) && openEdit(c)"
                >
                  <FaIcon name="i-lucide:pencil" class="text-sm" />
                </FaButton>
                <FaButton variant="ghost" size="icon-sm" :title="$t('common.detail')" @click="openContainerDetailWin(c.id, c.name)">
                  <FaIcon name="i-lucide:info" class="text-sm" />
                </FaButton>
                <FaButton variant="ghost" size="icon-sm" :title="$t('common.delete')" class="text-red-500!" :disabled="acting === c.id + 'rm'" @click="removeOne(c)">
                  <FaIcon name="i-lucide:trash" class="text-sm" />
                </FaButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <FaPagination v-model:page="page" v-model:size="size" :total="filtered.length" class="mt-3" />

    <ContainerForm v-model="createVisible" mode="create" @created="load(true)" />
    <ContainerForm v-model="editVisible" mode="edit" :container-id="editId" @saved="load(true)" />
    <FileEditorWorkspace />
    <!-- M35：提交为镜像 -->
    <FaModal v-model="commitVisible" :title="$t('container.list.commitModalTitle', { name: commitTarget?.name || '' })" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <FaInput v-model="commitImage" :placeholder="$t('container.list.commitPlaceholder')" class="w-full" @keyup.enter="doCommit" />
        <div class="text-xs text-muted-foreground">{{ $t('container.list.commitHint') }}</div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="commitVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="commitBusy" @click="doCommit">
          {{ $t('common.submit') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
    </FaPageMain>
      <YdDangerDelete
      v-model:visible="delVisible"
      :title="$t('container.list.deleteTitle', { name: delTarget?.name || '' })"
      :name="delTarget?.name || ''"
      :options="delOpts"
      :confirm-text="$t('common.delete')"
      @confirm="doDelete"
    />
</div>
</template>
