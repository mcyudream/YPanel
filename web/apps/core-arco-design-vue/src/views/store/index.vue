<script setup lang="ts">
import type { StoreAppItem, StoreFormField, StoreListQuery, StoreSource, StoreTag, StoreVersion } from '@/api/modules/store'
import type { AppTask } from '@/api/modules/task'
import { marked } from 'marked'
import { isPortField, storeApi } from '@/api/modules/store'
import { taskApi } from '@/api/modules/task'
import YdLogViewer from '@/components/YdLogViewer/index.vue'

defineOptions({
  name: 'StoreIndex',
})

type TabKey = 'all' | 'installed' | 'notInstalled' | 'upgradable' | 'sources'

const TABS: { key: TabKey, label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'installed', label: '已安装' },
  { key: 'notInstalled', label: '未安装' },
  { key: 'upgradable', label: '可升级' },
  { key: 'sources', label: '源管理' },
]

const KIND_LABEL: Record<string, string> = { app: '应用', service: '环境', middleware: '中间件' }
const TYPE_LABEL: Record<string, string> = { onepanel: '1Panel', 'yp-url': 'YP 远程', 'yp-git': 'YP Git' }

const activeTab = ref<TabKey>('all')
const search = ref('')
const activeTag = ref('')
const sourceFilter = ref<number | 0>(0)
const orderBy = ref<'name' | 'lastModified' | ''>('')
const page = ref(1)
const size = ref(20)
const total = ref(0)
const items = ref<StoreAppItem[]>([])
const tags = ref<StoreTag[]>([])
const sources = ref<StoreSource[]>([])
const loading = ref(false)
const syncing = ref(false)
const upgradableCount = ref(0)

async function load() {
  if (activeTab.value === 'sources') {
    await loadSources()
    return
  }
  loading.value = true
  try {
    const q: StoreListQuery = {
      search: search.value,
      tag: activeTag.value,
      sourceId: sourceFilter.value || undefined,
      status: activeTab.value,
      orderBy: orderBy.value || undefined,
      order: orderBy.value === 'name' ? 'asc' : 'desc',
      page: page.value,
      pageSize: size.value,
    }
    const out = await storeApi.list(q)
    items.value = out.items
    total.value = out.total
  }
  catch (e: any) {
    useFaToast().error('应用列表加载失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
  refreshUpgradableCount()
}

// 角标数 = 全量可升级数（借分页接口 total，pageSize=1 最轻）
async function refreshUpgradableCount() {
  try {
    const out = await storeApi.list({ status: 'upgradable', page: 1, pageSize: 1 })
    upgradableCount.value = out.total
  }
  catch {
    // 角标失败不打扰主流程
  }
}

async function loadTags() {
  try {
    tags.value = await storeApi.tags()
  }
  catch {}
}

async function loadSources() {
  try {
    sources.value = await storeApi.sources()
  }
  catch (e: any) {
    useFaToast().error('源列表加载失败', { description: e?.message })
  }
}

async function syncAll(force = false) {
  syncing.value = true
  try {
    const out = await storeApi.sync(force)
    const parts = Object.entries(out || {}).map(([k, v]) => `${k}: ${v}`)
    useFaToast().success(`同步完成（${parts.length} 个源）`, { description: parts.join('，') })
    await Promise.all([load(), loadTags()])
  }
  catch (e: any) {
    useFaToast().error('同步失败', { description: e?.message })
  }
  finally {
    syncing.value = false
  }
}

watch([search, activeTag, sourceFilter, orderBy], () => {
  page.value = 1
  load()
})
watch(activeTab, () => {
  page.value = 1
  load()
})
watch([page, size], () => load())

onMounted(() => {
  load()
  loadTags()
  loadSources()
})

// ---------- 详情抽屉 ----------
const detailVisible = ref(false)
const detailLoading = ref(false)
const detailApp = ref<StoreAppItem | null>(null)
const detailVersions = ref<StoreVersion[]>([])
const detailReadme = ref('')
const detailInstall = ref<{ version: string, composeProject: string } | null>(null)

function sourceName(id: number) {
  return sources.value.find(s => s.id === id)?.name || `源 #${id}`
}

async function openDetail(item: StoreAppItem) {
  detailApp.value = item
  detailVersions.value = []
  detailReadme.value = ''
  detailInstall.value = null
  detailVisible.value = true
  detailLoading.value = true
  try {
    const out = await storeApi.get(item.sourceId, item.key)
    detailVersions.value = out.versions
    detailInstall.value = out.installed ? { version: out.install.version, composeProject: out.install.composeProject } : null
    detailReadme.value = out.app.readMe
      ? (marked.parse(out.app.readMe, { async: false, breaks: true }) as string)
      : ''
  }
  catch (e: any) {
    useFaToast().error('读取应用详情失败', { description: e?.message })
  }
  finally {
    detailLoading.value = false
  }
}

// ---------- 安装向导 ----------
const installVisible = ref(false)
const installing = ref(false)
const installTarget = ref<StoreAppItem | null>(null)
const installForm = ref({ version: '', name: '', domain: '', params: {} as Record<string, string> })
const installFields = ref<StoreFormField[]>([])
const installProxyEnv = ref('')
const installVersions = ref<StoreVersion[]>([])
// 任务化安装：提交后切换为日志视图
const installTaskId = ref(0)
const installTask = ref<AppTask | null>(null)
const installLogLoading = ref(false)
let installPollTimer: ReturnType<typeof setInterval> | null = null

function fieldLabel(f: { label: Record<string, string>, envKey?: string }) {
  return f.label?.zh || f.label?.en || f.envKey || ''
}

function applyInstallVersion(versions: StoreVersion[], versionId: string) {
  const target = versions.find(v => v.id === versionId) || versions[0]
  if (!target) {
    return
  }
  installForm.value.version = target.id
  installFields.value = (target.formFields || []).filter(f => f.envKey)
  const params: Record<string, string> = {}
  for (const f of installFields.value) {
    params[f.envKey] = fieldDefault(f)
  }
  installForm.value.params = params
}

function stopInstallPolling() {
  if (installPollTimer) {
    clearInterval(installPollTimer)
    installPollTimer = null
  }
}

async function refreshInstallTask() {
  if (!installTaskId.value) {
    return
  }
  installLogLoading.value = true
  try {
    installTask.value = await taskApi.get(installTaskId.value)
    if (installTask.value.status !== 'running') {
      stopInstallPolling()
      if (installTask.value.status === 'success') {
        useFaToast().success(`应用 ${installTask.value.ref} 安装完成`)
      }
      else {
        useFaToast().error('安装失败', { description: installTask.value.error })
      }
      await Promise.all([load(), loadTags()])
    }
  }
  catch (e: any) {
    useFaToast().error('任务状态读取失败', { description: e?.message })
    stopInstallPolling()
  }
  finally {
    installLogLoading.value = false
  }
}

function openInstall(item: StoreAppItem, versionId = '') {
  installTarget.value = item
  installProxyEnv.value = item.reverseProxy
  installForm.value = { version: versionId || item.latestVersion || '', name: item.key, domain: '', params: {} }
  installTaskId.value = 0
  installTask.value = null
  // 优先复用详情抽屉已加载的版本；否则异步拉取
  if (detailApp.value?.sourceId === item.sourceId && detailApp.value?.key === item.key && detailVersions.value.length) {
    installVersions.value = detailVersions.value
    applyInstallVersion(detailVersions.value, installForm.value.version)
  }
  else {
    installVersions.value = []
    installFields.value = []
    storeApi.get(item.sourceId, item.key).then((out) => {
      installVersions.value = out.versions
      applyInstallVersion(out.versions, installForm.value.version)
    }).catch(() => {})
  }
  installVisible.value = true
}

async function doInstall() {
  const target = installTarget.value
  if (!target) {
    return
  }
  installing.value = true
  try {
    const out = await storeApi.install({
      sourceId: target.sourceId,
      key: target.key,
      version: installForm.value.version,
      name: installForm.value.name,
      params: installForm.value.params,
      domain: installForm.value.domain || undefined,
    })
    // 任务化：切换到日志视图并轮询
    installTaskId.value = out.taskId
    installTask.value = null
    useFaToast().success('安装任务已创建')
    void refreshInstallTask()
    stopInstallPolling()
    installPollTimer = setInterval(() => {
      if (!installTask.value || installTask.value.status === 'running') {
        void refreshInstallTask()
      }
      else {
        stopInstallPolling()
      }
    }, 2000)
  }
  catch (e: any) {
    useFaToast().error('创建安装任务失败', { description: e?.message })
  }
  finally {
    installing.value = false
  }
}

watch(installVisible, (v) => {
  if (!v) {
    stopInstallPolling()
  }
})

function uninstall(p: string) {
  const modal = useFaModal()
  modal.confirm({
    title: '卸载应用',
    content: `确认卸载 ${p}？容器与 compose 目录将被移除（数据卷保留在项目目录）。可在"任务中心"查看进度。`,
    onConfirm: async () => {
      try {
        await storeApi.uninstall(p)
        useFaToast().success('卸载任务已创建，可在「任务中心」查看进度')
        await Promise.all([load(), loadTags()])
      }
      catch (e: any) {
        useFaToast().error('卸载失败', { description: e?.message })
      }
    },
  })
}

async function upgrade(item: StoreAppItem) {
  openInstall(item, item.latestVer)
}

// ---------- 安装表单控件辅助 ----------

const CHARS = 'abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789'

function randomPassword(len = 16) {
  let s = ''
  for (let i = 0; i < len; i++) {
    s += CHARS[Math.floor(Math.random() * CHARS.length)]
  }
  return s
}

function randomPort() {
  return String(32768 + Math.floor(Math.random() * 28231))
}

function fieldIsEditable(f: StoreFormField) {
  return f.edit !== false && !f.disabled && f.type !== 'service' && f.type !== 'apps'
}

// 按字段语义生成默认值（密码随机 / 端口随机高位，对齐 1Panel 行为）
function fieldDefault(f: StoreFormField) {
  if (f.type === 'password' && f.random !== false && f.default === undefined) {
    return randomPassword()
  }
  if (isPortField(f) && f.type === 'number') {
    return randomPort()
  }
  if (f.default !== undefined && f.default !== null) {
    return String(f.default)
  }
  return ''
}

function randomizeField(f: StoreFormField) {
  if (f.type === 'password') {
    installForm.value.params[f.envKey] = randomPassword()
  }
  else if (isPortField(f)) {
    installForm.value.params[f.envKey] = randomPort()
  }
}

// ---------- 源管理 ----------
const sourceModalVisible = ref(false)
const sourceSaving = ref(false)
const sourceEditing = ref<StoreSource | null>(null)
const sourceForm = ref({ name: '', type: 'yp-git', url: '', branch: 'main', authToken: '', remark: '' })

function openSourceModal(src?: StoreSource) {
  sourceEditing.value = src || null
  sourceForm.value = src
    ? { name: src.name, type: src.type, url: src.url, branch: src.branch, authToken: '', remark: src.remark }
    : { name: '', type: 'yp-git', url: '', branch: 'main', authToken: '', remark: '' }
  sourceModalVisible.value = true
}

async function saveSource() {
  sourceSaving.value = true
  try {
    if (sourceEditing.value) {
      await storeApi.updateSource(sourceEditing.value.id, sourceForm.value)
      useFaToast().success('源已更新')
    }
    else {
      await storeApi.createSource(sourceForm.value)
      useFaToast().success('源已添加')
    }
    sourceModalVisible.value = false
    await loadSources()
  }
  catch (e: any) {
    useFaToast().error('保存失败', { description: e?.message })
  }
  finally {
    sourceSaving.value = false
  }
}

async function toggleSource(src: StoreSource) {
  try {
    await storeApi.setSourceEnabled(src.id, !src.enabled)
    await loadSources()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
  }
}

function removeSource(src: StoreSource) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除源',
    content: `确认删除源「${src.name}」？其同步的应用条目将一并移除（不影响已安装应用）。`,
    onConfirm: async () => {
      try {
        await storeApi.deleteSource(src.id)
        useFaToast().success('源已删除')
        await Promise.all([loadSources(), loadTags()])
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

async function syncOne(src: StoreSource) {
  try {
    const out = await storeApi.syncSource(src.id, true)
    useFaToast().success(`「${out.source}」同步完成：${out.total} 个应用`)
    await Promise.all([loadSources(), loadTags()])
  }
  catch (e: any) {
    useFaToast().error('同步失败', { description: e?.message })
    await loadSources()
  }
}

function statusText(s: StoreSource) {
  if (!s.url && s.type === 'yp-git') {
    return '未配置'
  }
  switch (s.status) {
    case 'ok': return '正常'
    case 'error': return `异常：${s.message}`
    default: return '待同步'
  }
}
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="package" :size="24" />
          <span>应用商店</span>
        </div>
      </template>
      <template #description>
        <span>多源驱动（YPanel 源 / 1Panel 源 / 自定义源）：一键安装为容器编排项目，支持环境服务与可复用中间件，可一键反代</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" :loading="syncing" @click="syncAll(true)">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> 同步全部源
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 状态 Tab -->
      <div class="mb-3 flex flex-wrap items-center gap-1 border-b border-border pb-2">
        <button
          v-for="t in TABS"
          :key="t.key"
          type="button"
          class="cursor-pointer rounded-md px-3 py-1.5 text-sm transition-colors"
          :class="activeTab === t.key ? 'bg-primary/10 font-medium text-primary' : 'text-muted-foreground hover:bg-accent/50'"
          @click="activeTab = t.key"
        >
          {{ t.label }}
          <span
            v-if="t.key === 'upgradable' && upgradableCount > 0"
            class="ml-1 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] leading-none text-white"
          >
            {{ upgradableCount > 99 ? '99+' : upgradableCount }}
          </span>
        </button>
      </div>

      <!-- 应用列表 -->
      <template v-if="activeTab !== 'sources'">
        <div class="mb-4 flex flex-wrap items-center gap-2">
          <div class="flex flex-wrap gap-1.5">
            <button
              type="button"
              class="cursor-pointer rounded-md border px-2.5 py-1 text-sm transition-colors"
              :class="activeTag === '' ? 'border-primary bg-primary/10 text-primary' : 'border-border hover:bg-accent/50'"
              @click="activeTag = ''"
            >
              全部
            </button>
            <button
              v-for="t in tags"
              :key="t.name"
              type="button"
              class="cursor-pointer rounded-md border px-2.5 py-1 text-sm transition-colors"
              :class="activeTag === t.name ? 'border-primary bg-primary/10 text-primary' : 'border-border hover:bg-accent/50'"
              @click="activeTag = activeTag === t.name ? '' : t.name"
            >
              {{ t.name }} <span class="text-xs opacity-60">{{ t.count }}</span>
            </button>
          </div>
          <div class="ml-auto flex items-center gap-2">
            <select v-model="sourceFilter" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none">
              <option :value="0">全部来源</option>
              <option v-for="s in sources.filter(x => x.enabled)" :key="s.id" :value="s.id">{{ s.name }}</option>
            </select>
            <select v-model="orderBy" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none">
              <option value="">默认排序</option>
              <option value="name">名称</option>
              <option value="lastModified">最近更新</option>
            </select>
            <FaInput v-model="search" placeholder="搜索应用…" class="w-52">
              <template #start>
                <FaIcon name="i-lucide:search" class="text-muted-foreground" />
              </template>
            </FaInput>
          </div>
        </div>

        <div v-if="loading && !items.length" class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div v-for="i in 8" :key="i" class="h-28 animate-pulse rounded-lg border bg-muted/30" />
        </div>
        <div v-else-if="!items.length" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
          {{ activeTab === 'upgradable' ? '所有应用均为最新版本' : activeTab === 'installed' ? '尚未安装任何应用' : '未找到匹配应用（点击右上角「同步全部源」拉取）' }}
        </div>
        <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div
            v-for="a in items"
            :key="`${a.sourceId}-${a.key}`"
            class="flex cursor-pointer flex-col rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
            @click="openDetail(a)"
          >
            <div class="flex items-start gap-3">
              <img
                :src="a.iconUrl || storeApi.iconUrl(a.sourceId, a.key)"
                class="size-10 shrink-0 rounded-lg object-contain"
                loading="lazy"
                @error="($event.target as HTMLImageElement).style.opacity = '0.2'"
              >
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1.5">
                  <span class="truncate font-medium">{{ a.name }}</span>
                  <span v-if="a.installed" class="shrink-0 rounded bg-green-500/15 px-1.5 py-0.5 text-xs text-green-600">已安装</span>
                  <span v-else-if="a.upgradable" class="shrink-0 rounded bg-orange-500/15 px-1.5 py-0.5 text-xs text-orange-600">可升级</span>
                </div>
                <div class="truncate text-xs text-muted-foreground" :title="a.title">{{ a.title }}</div>
              </div>
            </div>
            <div class="mt-2 line-clamp-2 min-h-10 flex-1 text-xs text-muted-foreground">{{ a.description }}</div>
            <div class="mt-3 flex items-center justify-between border-t pt-2" @click.stop>
              <div class="flex gap-1">
                <span class="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">{{ KIND_LABEL[a.kind] || a.kind }}</span>
                <span v-for="t in (a.tags || '').split(',').filter(Boolean).slice(0, 2)" :key="t" class="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">{{ t }}</span>
                <span class="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground" :title="sourceName(a.sourceId)">{{ sourceName(a.sourceId) }}</span>
              </div>
              <div class="flex gap-1.5">
                <FaButton v-if="a.upgradable" size="sm" variant="outline" @click="upgrade(a)">升级 {{ a.latestVer }}</FaButton>
                <FaButton v-if="a.installed" size="sm" variant="outline" @click="uninstall(a.installInfo?.composeProject || '')">卸载</FaButton>
                <FaButton v-else size="sm" @click="openInstall(a)">安装</FaButton>
              </div>
            </div>
          </div>
        </div>

        <div v-if="total > 0" class="mt-4 flex justify-end">
          <FaPagination v-model:page="page" v-model:size="size" :total="total" @page-change="load" @size-change="load" />
        </div>
      </template>

      <!-- 源管理 -->
      <template v-else>
        <div class="mb-4 flex items-center justify-between">
          <div class="text-sm text-muted-foreground">
            支持三类源：YP Git 仓库（github/gitee/gitlab/自建）、YP 远程 index.json、1Panel 1panel.json.zip
          </div>
          <FaButton size="sm" @click="openSourceModal()">
            <FaIcon name="i-lucide:plus" class="mr-1" /> 添加源
          </FaButton>
        </div>
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/40 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-2.5 font-medium">名称</th>
                <th class="px-4 py-2.5 font-medium">类型</th>
                <th class="px-4 py-2.5 font-medium">地址</th>
                <th class="px-4 py-2.5 font-medium">应用数</th>
                <th class="px-4 py-2.5 font-medium">状态</th>
                <th class="px-4 py-2.5 font-medium">最近同步</th>
                <th class="px-4 py-2.5 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in sources" :key="s.id" class="border-t border-border">
                <td class="px-4 py-2.5">
                  {{ s.name }}
                  <span v-if="s.builtin" class="ml-1 rounded bg-primary/10 px-1.5 py-0.5 text-xs text-primary">内置</span>
                  <span v-if="!s.enabled" class="ml-1 rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">已停用</span>
                </td>
                <td class="px-4 py-2.5 text-muted-foreground">{{ TYPE_LABEL[s.type] || s.type }}{{ s.branch ? ` · ${s.branch}` : '' }}</td>
                <td class="max-w-60 truncate px-4 py-2.5 font-mono text-xs text-muted-foreground" :title="s.url">{{ s.url || '—' }}</td>
                <td class="px-4 py-2.5">{{ s.appCount }}</td>
                <td class="max-w-52 truncate px-4 py-2.5" :class="s.status === 'error' ? 'text-red-500' : 'text-muted-foreground'" :title="statusText(s)">
                  {{ statusText(s) }}
                </td>
                <td class="px-4 py-2.5 text-xs text-muted-foreground">{{ s.lastSyncAt ? new Date(s.lastSyncAt).toLocaleString() : '—' }}</td>
                <td class="px-4 py-2.5">
                  <div class="flex justify-end gap-1">
                    <FaButton size="sm" variant="ghost" @click="syncOne(s)">同步</FaButton>
                    <FaButton size="sm" variant="ghost" @click="openSourceModal(s)">编辑</FaButton>
                    <FaButton size="sm" variant="ghost" @click="toggleSource(s)">{{ s.enabled ? '停用' : '启用' }}</FaButton>
                    <FaButton v-if="!s.builtin" size="sm" variant="ghost" class="text-red-500!" @click="removeSource(s)">删除</FaButton>
                  </div>
                </td>
              </tr>
              <tr v-if="!sources.length">
                <td colspan="7" class="px-4 py-8 text-center text-muted-foreground">暂无源</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </FaPageMain>

    <!-- 详情抽屉 -->
    <FaDrawer v-model="detailVisible" :title="`应用详情：${detailApp?.name || ''}`" class="max-w-2xl!">
      <div v-if="detailApp" class="flex flex-col gap-4">
        <div class="flex items-start gap-4">
          <img
            :src="detailApp.iconUrl || storeApi.iconUrl(detailApp.sourceId, detailApp.key)"
            class="size-16 shrink-0 rounded-xl border object-contain p-1"
            @error="($event.target as HTMLImageElement).style.opacity = '0.2'"
          >
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-lg font-medium">{{ detailApp.name }}</span>
              <span v-if="detailInstall" class="rounded bg-green-500/15 px-2 py-0.5 text-xs text-green-600">已安装 {{ detailInstall.version }}</span>
              <span v-if="detailApp.upgradable" class="rounded bg-orange-500/15 px-2 py-0.5 text-xs text-orange-600">可升级至 {{ detailApp.latestVer }}</span>
            </div>
            <div class="mt-1 text-sm text-muted-foreground">{{ detailApp.title || detailApp.description }}</div>
            <div class="mt-2 flex flex-wrap gap-1.5 text-xs text-muted-foreground">
              <span class="rounded bg-muted px-1.5 py-0.5">来源：{{ sourceName(detailApp.sourceId) }}</span>
              <span class="rounded bg-muted px-1.5 py-0.5">{{ KIND_LABEL[detailApp.kind] || detailApp.kind }}</span>
              <span v-if="detailApp.author" class="rounded bg-muted px-1.5 py-0.5">作者：{{ detailApp.author }}</span>
              <span v-if="detailApp.arch" class="rounded bg-muted px-1.5 py-0.5">架构：{{ detailApp.arch }}</span>
            </div>
          </div>
        </div>

        <div v-if="detailInstall" class="flex items-center justify-between rounded-lg border border-green-500/30 bg-green-500/5 px-3 py-2 text-sm">
          <span>运行中项目：<span class="font-mono">{{ detailInstall.composeProject }}</span></span>
          <div class="flex gap-2">
            <FaButton size="sm" variant="outline" @click="uninstall(detailInstall?.composeProject || '')">卸载</FaButton>
          </div>
        </div>

        <div>
          <div class="mb-1.5 text-sm font-medium">版本</div>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="v in detailVersions"
              :key="v.id"
              class="rounded-md border px-2 py-1 text-xs"
              :class="v.id === (detailInstall?.version || detailApp.latestVersion) ? 'border-primary bg-primary/10 text-primary' : 'border-border text-muted-foreground'"
            >
              {{ v.id }}
            </span>
            <span v-if="!detailVersions.length && detailLoading" class="text-xs text-muted-foreground">加载中…</span>
          </div>
        </div>

        <div v-if="detailApp.description" class="rounded-lg border bg-muted/20 p-3 text-sm text-muted-foreground">
          {{ detailApp.description }}
        </div>

        <div v-if="detailReadme">
          <div class="mb-1.5 text-sm font-medium">应用介绍</div>
          <div class="max-h-[50vh] overflow-auto rounded-lg border p-4 text-sm leading-6" v-html="detailReadme" />
        </div>

        <div class="flex justify-end border-t pt-3">
          <FaButton v-if="!detailInstall" size="sm" :disabled="detailLoading" @click="openInstall(detailApp, detailApp.latestVersion)">
            安装 {{ detailApp.latestVersion }}
          </FaButton>
          <FaButton v-else size="sm" variant="outline" @click="openInstall(detailApp, detailApp.latestVer)">
            升级至 {{ detailApp.latestVer }}
          </FaButton>
        </div>
      </div>
    </FaDrawer>

    <!-- 安装向导 -->
    <FaModal v-model="installVisible" :title="installTaskId ? `安装进行中：${installTarget?.name || ''}` : `安装：${installTarget?.name || ''}`" class="max-w-2xl!" :destroy-on-close="true" :close-on-click-modal="false">
      <!-- 阶段一：参数 -->
      <div v-if="!installTaskId" class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-28 shrink-0 text-sm text-muted-foreground">应用名</span>
          <FaInput v-model="installForm.name" placeholder="小写字母/数字/中划线" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-28 shrink-0 text-sm text-muted-foreground">版本</span>
          <select v-model="installForm.version" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option v-for="v in installVersions" :key="v.id" :value="v.id">{{ v.name || v.id }}</option>
          </select>
        </div>
        <template v-for="f in installFields" :key="f.envKey">
          <!-- 关联服务 / 复合字段（一期只读说明） -->
          <div v-if="f.type === 'service' || f.type === 'apps'" class="flex items-center gap-3">
            <span class="w-28 shrink-0 text-sm text-muted-foreground">{{ fieldLabel(f) }}</span>
            <span class="flex-1 rounded-md border border-dashed px-2 py-1.5 text-xs text-muted-foreground">
              {{ f.default ? String(f.default) : '自动关联已安装服务' }}{{ f.description ? `（${f.description}）` : '' }}
            </span>
          </div>
          <div v-else class="flex items-center gap-3">
            <span class="w-28 shrink-0 text-sm text-muted-foreground">
              {{ fieldLabel(f) }}<span v-if="f.required" class="text-red-500">*</span>
            </span>
            <!-- 枚举：下拉选择 -->
            <select
              v-if="f.type === 'select' && f.values?.length"
              v-model="installForm.params[f.envKey]"
              :disabled="!fieldIsEditable(f)"
              class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none"
            >
              <option v-for="o in f.values" :key="o.value" :value="o.value">{{ o.label || o.value }}</option>
            </select>
            <!-- 密码 / 端口：输入 + 随机 -->
            <div v-else class="flex flex-1 items-center gap-1.5">
              <FaInput
                v-model="installForm.params[f.envKey]"
                :type="f.type === 'password' ? 'password' : f.type === 'number' ? 'number' : 'text'"
                :disabled="!fieldIsEditable(f)"
                :placeholder="f.required ? '必填' : '可选'"
                class="flex-1"
              />
              <FaButton
                v-if="fieldIsEditable(f) && (f.type === 'password' || isPortField(f))"
                variant="outline" size="icon-sm" title="随机生成"
                @click="randomizeField(f)"
              >
                <FaIcon name="i-lucide:dices" class="text-sm" />
              </FaButton>
            </div>
          </div>
          <div v-if="f.description" class="-mt-2 pl-31 text-xs text-muted-foreground">{{ f.description }}</div>
        </template>
        <div v-if="installProxyEnv" class="flex items-center gap-3">
          <span class="w-28 shrink-0 text-sm text-muted-foreground">一键反代</span>
          <FaInput v-model="installForm.domain" placeholder="选填：安装后自动创建反代站点域名，如 app.example.com" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          安装为 compose 项目 app-&lt;应用名&gt;，容器接入 1panel-network；端口已随机避开占用（可手动修改，安装前会做占用预检）；卸载保留数据卷
        </div>
      </div>
      <!-- 阶段二：任务日志 -->
      <div v-else class="flex flex-col gap-3">
        <div class="flex items-center gap-3 text-sm">
          <span class="rounded-full px-2 py-0.5 text-xs" :class="installTask?.status === 'success' ? 'bg-emerald-500/10 text-emerald-600' : installTask?.status === 'failed' ? 'bg-red-500/10 text-red-600' : 'bg-blue-500/10 text-blue-600'">
            {{ installTask?.status === 'success' ? '安装成功' : installTask?.status === 'failed' ? '安装失败' : '进行中' }}
          </span>
          <span class="text-xs text-muted-foreground">任务 #{{ installTaskId }} · 也可在「任务中心」查看</span>
        </div>
        <YdLogViewer :logs="installTask?.logText || ''" height="320px" :loading="installLogLoading" />
      </div>
      <template #footer>
        <template v-if="!installTaskId">
          <FaButton variant="outline" @click="installVisible = false">取消</FaButton>
          <FaButton :loading="installing" @click="doInstall">创建安装任务</FaButton>
        </template>
        <template v-else>
          <FaButton variant="outline" @click="installVisible = false">
            {{ installTask?.status === 'running' ? '后台运行' : '关闭' }}
          </FaButton>
          <FaButton v-if="installTask?.status === 'failed'" @click="installTaskId = 0">返回修改参数</FaButton>
        </template>
      </template>
    </FaModal>

    <!-- 添加/编辑源 -->
    <FaModal v-model="sourceModalVisible" :title="sourceEditing ? '编辑源' : '添加源'" class="max-w-xl!">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">名称</span>
          <FaInput v-model="sourceForm.name" :disabled="!!sourceEditing?.builtin" placeholder="如：我的私有源" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">类型</span>
          <select v-model="sourceForm.type" :disabled="!!sourceEditing?.builtin" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="yp-git">YP Git 仓库（推荐）</option>
            <option value="yp-url">YP 远程 index.json</option>
            <option value="onepanel">1Panel（1panel.json.zip）</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">地址</span>
          <FaInput
            v-model="sourceForm.url"
            :placeholder="sourceForm.type === 'yp-git' ? 'https://github.com/you/yp-apps.git' : sourceForm.type === 'onepanel' ? 'https://apps-assets.fit2cloud.com/dev/1panel.json.zip' : 'https://example.com/index.json'"
            class="flex-1"
          />
        </div>
        <div v-if="sourceForm.type === 'yp-git'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">分支</span>
          <FaInput v-model="sourceForm.branch" placeholder="main" class="flex-1" />
        </div>
        <div v-if="sourceForm.type === 'yp-git'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">Token</span>
          <FaInput v-model="sourceForm.authToken" type="password" placeholder="私有仓库访问 token（可选）" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">备注</span>
          <FaInput v-model="sourceForm.remark" placeholder="选填" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="sourceModalVisible = false">取消</FaButton>
        <FaButton :loading="sourceSaving" @click="saveSource">保存</FaButton>
      </template>
    </FaModal>
  </div>
</template>
