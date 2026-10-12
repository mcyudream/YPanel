<script setup lang="ts">
import type { SiteItem, DiscoveredSite } from '@/api/modules/site'
import apiSite, { siteDiscoveryApi } from '@/api/modules/site'
import api from '@/api'
import { siteBatchApi } from '@/api/modules/site'
import apiRuntime from '@/api/modules/runtime'
import YdDangerDelete from '@/components/YdDangerDelete/index.vue'
import YdOwnerDialog from '@/components/YdOwnerDialog/index.vue'
import YdNodeSelect from '@/components/YdNodeSelect/index.vue'
import YdDataTable from '@/components/YdDataTable/index.vue'
import type { YdDataTableColumn } from '@/components/YdDataTable/index.vue'
import { siteGroupApi } from '@/api/modules/cert'
import type { SiteGroup } from '@/api/modules/cert'
import { useYwEmbed } from '@/views/desktop/embed'
import { i18n, tr } from '@/locales'

defineOptions({
  name: 'SitesIndex',
})

function typeLabel(t: string) {
  return tr(`sites.type.${t}`, t)
}

// ---- M54-P3 属主分配 ----
const ownerVisible = ref(false)
const ownerTarget = ref<{ id: number, ownerId: number }>({ id: 0, ownerId: 0 })

function openOwner(id: number, ownerId: number) {
  ownerTarget.value = { id, ownerId: ownerId ?? 0 }
  ownerVisible.value = true
}

async function doSetOwner(uid: number) {
  try {
    await apiSite.setOwner(ownerTarget.value.id, uid)
    useFaToast().success(i18n.global.t('common.owner.saved'))
    load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('common.owner.saveFailed'), { description: e?.message })
  }
}

const router = useRouter()

// webos 桌面承载时：站点配置开新窗（router.push 会顶掉 /desktop 路由）；经典模式保持路由跳转
const ywEmbed = useYwEmbed()

function openSiteConfig(id: number, domain: string) {
  if (ywEmbed) {
    ywEmbed.openApp('site-detail', { title: domain || i18n.global.t('sites.list.untitled', { n: id }), launchOptions: { id } })
    return
  }
  router.push(`/sites/${id}`)
}

const status = ref<{ installed: boolean, running: boolean, sites: number, mode?: 'container' | 'host' }>()
const sites = ref<SiteItem[]>([]) // 当前页行（M67 后端分页）
const siteTotal = ref(0)
const sitePage = ref(1)
const sitePageSize = ref(20)
const loading = ref(false)
const installing = ref(false)

// ---- 筛选 / 排序（B23 对齐 1Panel 列表）----
const filterType = ref('')
const filterGroup = ref<number | ''>('')
const keyword = ref('')
const sortKey = ref<'name' | 'type' | 'enabled' | ''>('')
const sortOrder = ref<1 | -1>(1)

const groups = ref<SiteGroup[]>([])

async function loadGroups() {
  try {
    groups.value = await siteGroupApi.list()
  }
  catch {}
}

function setSort(key: 'name' | 'type' | 'enabled') {
  if (sortKey.value === key) {
    sortOrder.value = sortOrder.value === 1 ? -1 : 1
  }
  else {
    sortKey.value = key
    sortOrder.value = 1
  }
  sitePage.value = 1
  load()
}

// 过滤/排序/分页全部后端承载（M67）；filterType/filterGroup/keyword 变化经 watch 防抖回查
async function load() {
  loading.value = true
  try {
    status.value = await apiSite.status()
    if (!status.value.installed) {
      return
    }
    const out = await apiSite.listPaged({
      page: sitePage.value,
      pageSize: sitePageSize.value,
      keyword: keyword.value.trim() || undefined,
      groupId: filterGroup.value === '' ? undefined : filterGroup.value,
      type: filterType.value || undefined,
      sort: sortKey.value || undefined,
      order: sortKey.value ? (sortOrder.value === 1 ? 'asc' : 'desc') : undefined,
    })
    sites.value = out.items
    siteTotal.value = out.total
    // 删除/过滤收缩后当前页越界回退
    if (!out.items.length && out.total > 0 && sitePage.value > 1) {
      sitePage.value = Math.ceil(out.total / sitePageSize.value)
      await load()
    }
  }
  finally {
    loading.value = false
  }
}

let kwTimer: ReturnType<typeof setTimeout> | null = null
watch([filterType, filterGroup], () => {
  sitePage.value = 1
  load()
})
watch(keyword, () => {
  if (kwTimer) {
    clearTimeout(kwTimer)
  }
  kwTimer = setTimeout(() => {
    sitePage.value = 1
    load()
  }, 300)
})

async function install() {
  installing.value = true
  useFaToast().info(i18n.global.t('sites.list.toast.installing'))
  try {
    await apiSite.install()
    useFaToast().success(i18n.global.t('sites.list.toast.ready'))
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.installFailed'), { description: e?.message })
  }
  finally {
    installing.value = false
  }
}

const nginxAdopting = ref(false)

async function adoptHost() {
  nginxAdopting.value = true
  try {
    const out = await apiSite.adoptHost()
    useFaToast().success(i18n.global.t('sites.list.toast.adoptedHost'), { description: out.hint })
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.adoptFailed'), { description: e?.message })
  }
  finally {
    nginxAdopting.value = false
  }
}

// 创建
const createVisible = ref(false)
const form = ref({ name: '', type: 'static', domain: '', extraDomains: '', port: 80, proxyPass: '', indexFiles: 'index.html', runtimeId: 0, groupId: 0, remark: '', nodeId: '' })
const siteNodes = ref<{ id: string, name: string, hostname?: string }[]>([])
const proxyRules = ref<{ prefix: string, target: string, ws?: boolean }[]>([{ prefix: '/api', target: '' }])
const creating = ref(false)
const runtimes = ref<{ id: number, name: string, type: string, version: string, origin: string, fcgiAddr: string, containerName: string, running: boolean }[]>([])

const runtimeOptions = computed(() => [
  ...(!runtimes.value.length ? [{ label: i18n.global.t('sites.dialogs.create.noRuntime'), value: 0, disabled: true }] : []),
  ...runtimes.value.map(r => ({
    label: `${r.origin === 'external' ? `${r.name}（${r.fcgiAddr}）` : `${r.containerName || `php-${r.name}`}（${r.version}）`}${r.running ? ` · ${i18n.global.t('sites.list.running')}` : ` · ${i18n.global.t('sites.list.stopped')}`}`,
    value: r.id,
  })),
])

function openCreate() {
  const defGroup = groups.value.find(g => g.isDefault)
  form.value = { name: '', type: 'static', domain: '', extraDomains: '', port: 80, proxyPass: '', indexFiles: 'index.html', runtimeId: 0, groupId: defGroup?.id || 0, remark: '', nodeId: '' }
  proxyRules.value = [{ prefix: '/api', target: '' }]
  createVisible.value = true
  apiRuntime.list().then((list) => {
    // PHP 站点只能绑定 PHP 运行环境（代码运行时走反代站点）
    runtimes.value = list.filter(r => r.type === 'php')
    if (runtimes.value.length && !form.value.runtimeId) {
      form.value.runtimeId = runtimes.value[0].id
    }
  }).catch(() => {})
}

async function doCreate() {
  creating.value = true
  try {
    const extraDomains = form.value.extraDomains.split(/[\n,]/).map(x => x.trim()).filter(Boolean)
    const rules = proxyRules.value
      .map(r => ({ prefix: r.prefix || '/', target: r.target.trim(), ws: true }))
      .filter(r => r.target)
    await apiSite.create({
      name: form.value.name,
      type: form.value.type,
      domain: form.value.domain,
      extraDomains,
      port: Number(form.value.port) || 80,
      nodeId: form.value.nodeId,
      proxyPass: form.value.proxyPass,
      proxyRules: form.value.type === 'proxy' ? rules : undefined,
      indexFiles: form.value.indexFiles,
      runtimeId: form.value.type === 'php' ? form.value.runtimeId : undefined,
      groupId: form.value.groupId || undefined,
      remark: form.value.remark || undefined,
    })
    useFaToast().success(i18n.global.t('sites.list.toast.created'))
    createVisible.value = false
    await load()
    await loadGroups()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.createFailed'), { description: e?.message })
  }
  finally {
    creating.value = false
  }
}

async function toggle(s: SiteItem) {
  try {
    if (s.enabled) {
      await apiSite.disable(s.id)
      useFaToast().success(i18n.global.t('sites.list.toast.disabled'))
    }
    else {
      await apiSite.enable(s.id)
      useFaToast().success(i18n.global.t('sites.list.toast.enabled'))
    }
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.actionFailed'), { description: e?.message })
  }
}

// 危险删除（选项 + 域名确认）
const delVisible = ref(false)
const delTarget = ref<SiteItem | null>(null)
const deleting = ref(false)
const SITE_DEL_OPTS = computed(() => [
  { key: 'files', label: i18n.global.t('sites.dialogs.delete.optFiles'), desc: i18n.global.t('sites.dialogs.delete.optFilesDesc') },
  { key: 'backups', label: i18n.global.t('sites.dialogs.delete.optBackups'), desc: i18n.global.t('sites.dialogs.delete.optBackupsDesc') },
])

function remove(s: SiteItem) {
  delTarget.value = s
  delVisible.value = true
}

async function doDelete(checked: Record<string, boolean>) {
  if (!delTarget.value) {
    return
  }
  deleting.value = true
  try {
    await apiSite.remove(delTarget.value.id, !!checked.files, !!checked.backups)
    useFaToast().success(i18n.global.t('sites.list.toast.deleted'))
    delVisible.value = false
    await load()
    await loadGroups()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.deleteFailed'), { description: e?.message })
  }
  finally {
    deleting.value = false
  }
}

// ---- 分组 / 备注（B23）----

const groupModalVisible = ref(false)
const newGroupName = ref('')
const groupBusy = ref(false)
const editingGroupName = ref(0)
const editingGroupNameVal = ref('')

function openGroups() {
  groupModalVisible.value = true
  loadGroups()
}

async function doCreateGroup() {
  if (!newGroupName.value.trim())
    return
  groupBusy.value = true
  try {
    await siteGroupApi.create(newGroupName.value.trim())
    newGroupName.value = ''
    await loadGroups()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.groupCreateFailed'), { description: e?.message })
  }
  finally {
    groupBusy.value = false
  }
}

async function doGroupAction(g: SiteGroup, action: 'remove' | 'default' | 'rename') {
  try {
    if (action === 'remove') {
      await siteGroupApi.remove(g.id)
    }
    else if (action === 'default') {
      await siteGroupApi.setDefault(g.id)
    }
    else if (action === 'rename') {
      await siteGroupApi.rename(g.id, editingGroupNameVal.value)
      editingGroupName.value = 0
    }
    await loadGroups()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.actionFailed'), { description: e?.message })
  }
}

function groupNameOf(s: SiteItem) {
  return s.groupName || i18n.global.t('common.default')
}

// 备注行内编辑
const remarkEditing = ref(0)
const remarkEditingVal = ref('')

function startEditRemark(s: SiteItem) {
  remarkEditing.value = s.id
  remarkEditingVal.value = s.remark || ''
}

async function saveRemark(s: SiteItem) {
  try {
    await apiSite.updateMeta(s.id, { remark: remarkEditingVal.value })
    s.remark = remarkEditingVal.value
    remarkEditing.value = 0
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.remarkSaveFailed'), { description: e?.message })
  }
}

// ---- 站点识别（扫描未接管配置与独立容器）----

const scanVisible = ref(false)
const scanning = ref(false)
const scanResult = ref<{ sites: DiscoveredSite[], containers: { name: string, image: string, ports: string }[] }>()
const adopting = ref('')

async function openScan() {
  scanVisible.value = true
  scanning.value = true
  try {
    scanResult.value = await siteDiscoveryApi.scan()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.scanFailed'), { description: e?.message })
  }
  finally {
    scanning.value = false
  }
}

async function adopt(d: DiscoveredSite) {
  adopting.value = d.file
  try {
    await siteDiscoveryApi.adopt({ file: d.file, domain: d.domain, type: d.type, proxyPass: d.proxyPass })
    useFaToast().success(i18n.global.t('sites.list.toast.adoptedSite', { name: d.domain }))
    scanResult.value!.sites = scanResult.value!.sites.filter(x => x.file !== d.file)
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.adoptFailed'), { description: e?.message })
  }
  finally {
    adopting.value = ''
  }
}

// ---- M39：批量 / 默认站点 / 到期 ----
const siteSelected = ref<Set<number>>(new Set())

function toggleSiteSelect(id: number) {
  const n = new Set(siteSelected.value)
  if (n.has(id)) n.delete(id)
  else n.add(id)
  siteSelected.value = n
}

async function batchSites(action: 'enable' | 'disable' | 'delete') {
  const ids = [...siteSelected.value]
  if (!ids.length) return
  const doIt = async () => {
    try {
      await siteBatchApi.batch(ids, action, { purgeFiles: action === 'delete' })
      const resultKey = action === 'enable' ? 'sites.list.toast.batchEnabled' : action === 'disable' ? 'sites.list.toast.batchDisabled' : 'sites.list.toast.batchDeleted'
      useFaToast().success(i18n.global.t(resultKey, { n: ids.length }))
      siteSelected.value = new Set()
      await load()
    } catch (e: any) {
      useFaToast().error(i18n.global.t('sites.list.toast.batchFailed'), { description: e?.message })
    }
  }
  if (action === 'delete') {
    useFaModal().confirm({ title: i18n.global.t('sites.list.batchDelete'), content: i18n.global.t('sites.dialogs.delete.batchConfirm', { n: ids.length }), onConfirm: doIt })
  } else {
    await doIt()
  }
}

async function setDefault(s: SiteItem) {
  try {
    await siteBatchApi.setDefault(s.id)
    useFaToast().success(i18n.global.t('sites.list.toast.setDefault', { name: s.name }))
    await load()
  } catch (e: any) {
    useFaToast().error(i18n.global.t('sites.list.toast.setFailed'), { description: e?.message })
  }
}

const expireVisible = ref(false)
const expireTarget = ref<SiteItem | null>(null)
const expireVal = ref('')

function openExpire(s: SiteItem) {
  expireTarget.value = s
  expireVal.value = s.expireAt ? String(s.expireAt).slice(0, 10) : ''
  expireVisible.value = true
}

async function saveExpire() {
  if (!expireTarget.value) return
  try {
    await siteBatchApi.setExpire(expireTarget.value.id, expireVal.value || null)
    useFaToast().success(i18n.global.t('sites.dialogs.expire.saved'))
    expireVisible.value = false
    await load()
  } catch (e: any) {
    useFaToast().error(i18n.global.t('sites.shared.saveFailed'), { description: e?.message })
  }
}

// ---- 表格列定义（M67 YdDataTable）：勾选+名称左固定，操作右固定，中间列可拖拽换序（记忆）----
const siteColumns = computed<YdDataTableColumn[]>(() => [
  { key: 'select', label: '', width: 36, fixed: 'left' },
  { key: 'name', label: i18n.global.t('common.name'), width: 200, sortable: true, fixed: 'left' },
  { key: 'type', label: i18n.global.t('common.type'), width: 90, sortable: true },
  { key: 'group', label: i18n.global.t('sites.list.group'), hidden: 'md' },
  { key: 'domain', label: i18n.global.t('sites.list.domain') },
  { key: 'protocol', label: i18n.global.t('sites.list.protocol'), width: 90 },
  { key: 'expire', label: i18n.global.t('sites.list.certExpire'), width: 110 },
  { key: 'enabled', label: i18n.global.t('common.status'), width: 84, sortable: true },
  { key: 'remark', label: i18n.global.t('common.remark'), hidden: 'lg' },
  { key: 'actions', label: i18n.global.t('common.operation'), width: 330, fixed: 'right', align: 'right' },
])

const allPageSelected = computed(() => sites.value.length > 0 && sites.value.every(s => siteSelected.value.has(s.id)))

function toggleAllPage() {
  siteSelected.value = allPageSelected.value ? new Set() : new Set(sites.value.map(s => s.id))
}

function onSort(key: string) {
  if (key === 'name' || key === 'type' || key === 'enabled') {
    setSort(key)
  }
}

onMounted(() => {
  api.get('api/v1/nodes', { silent: true }).then((r) => {
    siteNodes.value = (r.data as any[]).map((x: any) => ({ id: x.id, name: x.name, hostname: x.hostname }))
  }).catch(() => {})
  load()
  loadGroups()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="globe" :size="24" />
          <span>{{ $t('sites.list.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('sites.list.description') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <span v-if="status?.installed" class="mr-2 inline-flex items-center gap-1.5 text-xs text-muted-foreground">
          <span class="inline-block size-1.5 rounded-full" :class="status.running ? 'bg-emerald-500' : 'bg-red-500'" />
          nginx（{{ status.mode === 'host' ? $t('sites.list.modeHost') : $t('sites.list.modeContainer') }}）{{ status.running ? $t('sites.list.running') : $t('sites.list.stopped') }} · {{ $t('sites.list.siteCount', { n: status.sites }) }}
        </span>
        <FaButton v-if="!status?.installed" v-auth="['site:write']" size="sm" :loading="installing" @click="install">
          <YdMorphIcon name="download" :size="14" class="mr-1" /> {{ $t('sites.list.installNginx') }}
        </FaButton>
        <FaButton v-if="!status?.installed" v-auth="['site:write']" variant="outline" size="sm" :loading="nginxAdopting" @click="adoptHost">
          <FaIcon name="i-lucide:plug-zap" class="mr-1" /> {{ $t('sites.list.adoptHostNginx') }}
        </FaButton>
        <FaButton v-if="status?.installed" variant="outline" size="sm" @click="openScan">
          <YdMorphIcon name="search" :size="14" class="mr-1" /> {{ $t('sites.list.scan') }}
        </FaButton>
        <FaButton v-if="status?.installed" v-auth="['site:write']" variant="outline" size="sm" @click="openGroups">
          <FaIcon name="i-lucide:folder" class="mr-1" /> {{ $t('sites.list.groups') }}
        </FaButton>
        <FaButton v-if="status?.installed" v-auth="['site:write']" size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('sites.list.createSite') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="status && !status.installed" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
        {{ $t('sites.list.notInstalled') }}
      </div>

      <div v-else>
        <!-- 筛选栏（B23：类型 / 分组 / 搜索） -->
        <div class="mb-3 flex flex-wrap items-center gap-2">
          <YdSelect
            v-model="filterType"
            :options="[
              { label: $t('sites.list.allTypes'), value: '' },
              { label: $t('sites.type.static'), value: 'static' },
              { label: $t('sites.type.proxy'), value: 'proxy' },
              { label: 'PHP', value: 'php' },
            ]"
          />
          <YdSelect
            v-model="filterGroup"
            :options="[{ label: $t('sites.list.allGroups'), value: '' }, ...groups.map(g => ({ label: `${g.name}（${g.sites}）`, value: g.id }))]"
          />
          <FaInput v-model="keyword" :placeholder="$t('sites.list.searchPlaceholder')" class="w-56!" />
        </div>

        <YdDataTable
          :columns="siteColumns"
          :rows="sites"
          storage-key="sites"
          :loading="loading && !sites.length"
          :sort-key="sortKey"
          :sort-order="sortOrder === 1 ? 'asc' : 'desc'"
          v-model:page="sitePage"
          v-model:size="sitePageSize"
          :total="siteTotal"
          :empty-text="$t('sites.list.noSites')"
          @sort="onSort"
          @update:page="load"
          @update:size="sitePage = 1; load()"
        >
          <template #head-select>
            <input type="checkbox" :checked="allPageSelected" @change="toggleAllPage">
          </template>
          <template #cell-select="{ item }">
            <input type="checkbox" :checked="siteSelected.has(item.id)" @change="toggleSiteSelect(item.id)">
          </template>
          <template #cell-name="{ item }">
            <span class="font-medium">{{ item.name }}</span>
            <span v-if="item.isDefault" class="ml-1 rounded-full bg-sky-500/10 px-1.5 py-0.5 text-[10px] text-sky-600">{{ $t('common.default') }}</span>
            <a v-if="item.enabled" :href="`http://${item.domain}`" target="_blank" rel="noopener" class="ml-1 text-primary opacity-60" :title="$t('sites.list.visit')">↗</a>
          </template>
          <template #cell-type="{ item }">
            <span class="rounded-full px-2 py-0.5 text-xs" :class="{ static: 'bg-blue-500/10 text-blue-600', proxy: 'bg-purple-500/10 text-purple-600', php: 'bg-emerald-500/10 text-emerald-600' }[item.type] || 'bg-muted text-muted-foreground'">
              {{ typeLabel(item.type) }}
            </span>
          </template>
          <template #cell-group="{ item }">
            <span class="text-xs text-muted-foreground">{{ groupNameOf(item) }}</span>
          </template>
          <template #cell-domain="{ item }">
            <span class="font-mono text-[13px]">{{ item.domain }}</span>
            <span v-if="item.domains?.length" class="text-xs text-muted-foreground"> +{{ item.domains.length }}</span>
          </template>
          <template #cell-protocol="{ item }">
            <span v-if="item.certDomain" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">HTTPS</span>
            <span v-else class="text-xs text-muted-foreground">HTTP</span>
          </template>
          <template #cell-expire="{ item }">
            <template v-if="item.certDomain && item.certNotAfter">
              <span class="text-xs" :class="(new Date(item.certNotAfter).getTime() - Date.now()) < 0 ? 'text-red-500' : (new Date(item.certNotAfter).getTime() - Date.now()) < 15 * 86400000 ? 'text-amber-500' : 'text-muted-foreground'">
                {{ new Date(item.certNotAfter).toISOString().slice(0, 10) }}
              </span>
            </template>
            <span v-else class="text-xs text-muted-foreground">—</span>
          </template>
          <template #cell-enabled="{ item }">
            <button
              type="button"
              class="cursor-pointer rounded-full px-2 py-0.5 text-xs transition-colors"
              :class="item.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
              @click="toggle(item)"
            >
              {{ item.enabled ? $t('sites.list.started') : $t('sites.list.stopped') }}
            </button>
          </template>
          <template #cell-remark="{ item }">
            <input
              v-if="remarkEditing === item.id"
              v-model="remarkEditingVal"
              class="h-7 w-full rounded border border-input bg-background px-1.5 text-xs outline-none focus:border-primary"
              @keyup.enter="saveRemark(item)"
              @blur="saveRemark(item)"
            >
            <span
              v-else
              class="block max-w-40 cursor-pointer truncate text-xs" :class="item.remark ? 'text-foreground' : 'text-muted-foreground/50'"
              :title="$t('sites.list.clickEditRemark')"
              @click="startEditRemark(item)"
            >
              {{ item.remark || $t('sites.list.remarkPlaceholder') }}
            </span>
          </template>
          <template #cell-actions="{ item }">
            <div class="flex items-center justify-end gap-1">
              <FaButton size="sm" @click="openSiteConfig(item.id, item.domain)">{{ $t('sites.list.config') }}</FaButton>
              <FaButton v-if="!item.isDefault" variant="ghost" size="sm" :title="$t('sites.list.setDefaultTip')" @click="setDefault(item)">{{ $t('sites.list.setDefault') }}</FaButton>
              <FaButton variant="ghost" size="sm" :title="$t('sites.list.setExpireTip')" @click="openExpire(item)">{{ $t('sites.list.expire') }}</FaButton>
              <FaButton variant="outline" size="sm" @click="toggle(item)">{{ item.enabled ? $t('sites.list.disable') : $t('common.enabled') }}</FaButton>
              <FaButton variant="ghost" size="sm" :title="$t('common.owner.title')" @click="openOwner(item.id, item.ownerId)">{{ $t('common.owner.short') }}</FaButton>
              <FaButton variant="outline" size="sm" class="text-red-500!" @click="remove(item)">{{ $t('common.delete') }}</FaButton>
            </div>
          </template>
          <template #footerLeft>
            <template v-if="siteSelected.size">
              <span class="text-xs text-primary">{{ $t('sites.list.selectedCount', { n: siteSelected.size }) }}</span>
              <FaButton variant="outline" size="sm" @click="batchSites('enable')">{{ $t('sites.list.batchEnable') }}</FaButton>
              <FaButton variant="outline" size="sm" @click="batchSites('disable')">{{ $t('sites.list.batchDisable') }}</FaButton>
              <FaButton variant="outline" size="sm" class="text-red-500!" @click="batchSites('delete')">{{ $t('sites.list.batchDelete') }}</FaButton>
            </template>
          </template>
        </YdDataTable>

        <!-- 扫描识别结果 -->
        <div v-if="scanVisible" class="mt-4 rounded-lg border bg-background p-4">
          <div class="mb-3 flex items-center justify-between">
            <div class="flex items-center gap-2 text-sm font-medium">
              <YdMorphIcon name="search" :size="16" />
              {{ $t('sites.list.unadoptedCount', { n: scanResult?.sites.length || 0 }) }}
            </div>
            <FaButton variant="ghost" size="icon-sm" @click="scanVisible = false">
              <FaIcon name="i-lucide:x" class="text-sm" />
            </FaButton>
          </div>
          <div v-if="scanning" class="py-6 text-center text-sm text-muted-foreground">{{ $t('sites.list.scanning') }}</div>
          <template v-else>
            <div v-if="!scanResult?.sites.length" class="pb-3 text-sm text-muted-foreground">{{ $t('sites.list.noUnadopted') }}</div>
            <div v-for="d in scanResult?.sites || []" :key="d.file" class="mb-2 flex flex-wrap items-center gap-3 rounded-md border px-3 py-2">
              <span class="font-mono text-xs">{{ d.file }}</span>
              <span class="rounded-full bg-blue-500/10 px-2 py-0.5 text-xs text-blue-600">{{ typeLabel(d.type === 'proxy' ? 'proxy' : 'static') }}</span>
              <span class="font-mono text-xs text-muted-foreground">{{ d.domain }}</span>
              <span v-if="d.proxyPass" class="font-mono text-xs text-muted-foreground">→ {{ d.proxyPass }}</span>
              <FaButton class="ml-auto" size="sm" :loading="adopting === d.file" @click="adopt(d)">{{ $t('sites.list.adoptOne') }}</FaButton>
            </div>
            <div v-if="scanResult?.containers.length" class="mt-3 border-t pt-3">
              <div class="mb-2 text-xs text-muted-foreground">{{ $t('sites.list.detectedContainers') }}</div>
              <div v-for="c in scanResult.containers" :key="c.name" class="mb-1 font-mono text-xs text-muted-foreground">
                {{ c.name }} · {{ c.image }} · {{ c.ports }}
              </div>
            </div>
          </template>
        </div>
      </div>
    </FaPageMain>

    <!-- 创建 -->
    <FaModal v-model="createVisible" :title="$t('sites.dialogs.create.title')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.type') }}</span>
          <div class="flex flex-1 gap-1.5">
            <button
              v-for="t in [{ v: 'static', l: $t('sites.type.staticSite') }, { v: 'proxy', l: $t('sites.type.proxySite') }, { v: 'php', l: $t('sites.type.phpSite') }]"
              :key="t.v"
              type="button"
              class="flex-1 cursor-pointer rounded-md border px-2 py-1.5 text-sm transition-colors"
              :class="form.type === t.v ? 'border-primary bg-primary/10' : 'border-border hover:bg-accent/50'"
              @click="form.type = t.v"
            >
              {{ t.l }}
            </button>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('sites.list.group') }}</span>
          <YdSelect
            v-model="form.groupId"
            :options="[{ label: $t('common.default'), value: 0 }, ...groups.filter(x => !x.isDefault).map(g => ({ label: g.name, value: g.id }))]"
            size="default"
            button-class="flex-1"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('sites.dialogs.create.siteName') }}</span>
          <FaInput v-model="form.name" :placeholder="$t('sites.dialogs.create.namePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('nodes.targetNode') }}</span>
          <YdNodeSelect
            v-model="form.nodeId"
            local-value=""
            :nodes="siteNodes"
            size="default"
            button-class="min-w-0 flex-1"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('sites.dialogs.create.primaryDomain') }}</span>
          <FaInput v-model="form.domain" :placeholder="$t('sites.dialogs.create.domainPlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('sites.dialogs.create.port') }}</span>
          <FaInput v-model.number="form.port" type="number" class="w-40" />
          <span class="text-xs text-muted-foreground">{{ $t('sites.dialogs.create.portHint') }}</span>
        </div>
        <div class="flex items-start gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('sites.dialogs.create.extraDomains') }}</span>
          <textarea v-model="form.extraDomains" class="h-14 flex-1 rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" :placeholder="$t('sites.dialogs.create.extraPlaceholder')" />
        </div>
        <div v-if="form.type === 'php'" class="rounded-md border p-3">
          <div class="mb-2 text-sm font-medium">{{ $t('sites.dialogs.create.phpRuntime') }}</div>
          <YdSelect v-model="form.runtimeId" :options="runtimeOptions" size="default" button-class="w-full" />
          <div class="mt-1 text-xs text-muted-foreground">{{ $t('sites.dialogs.create.phpHint') }}</div>
        </div>
        <div v-if="form.type === 'proxy'" class="rounded-md border p-3">
          <div class="mb-2 flex items-center justify-between">
            <span class="text-sm font-medium">{{ $t('sites.dialogs.create.proxyRules') }}</span>
            <FaButton variant="outline" size="sm" @click="proxyRules.push({ prefix: '/api', target: '' })">
              <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('sites.dialogs.create.addRule') }}
            </FaButton>
          </div>
          <div v-for="(r, idx) in proxyRules" :key="idx" class="mb-2 flex items-center gap-2">
            <FaInput v-model="r.prefix" :placeholder="$t('sites.dialogs.create.prefixPlaceholder')" class="w-32" />
            <span class="text-muted-foreground">→</span>
            <FaInput v-model="r.target" placeholder="http://host.docker.internal:3000" class="flex-1" />
            <label class="flex shrink-0 cursor-pointer items-center gap-1 text-xs">
              <input v-model="r.ws" type="checkbox"> WS
            </label>
            <FaButton v-if="proxyRules.length > 1" variant="ghost" size="icon-sm" @click="proxyRules.splice(idx, 1)">
              <FaIcon name="i-lucide:trash" class="text-sm" />
            </FaButton>
          </div>
          <div class="text-xs text-muted-foreground">{{ $t('sites.dialogs.create.wsHint') }}</div>
        </div>
        <div v-if="form.type === 'static' || form.type === 'php'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('sites.dialogs.create.defaultDocs') }}</span>
          <FaInput v-model="form.indexFiles" :placeholder="form.type === 'php' ? 'index.php' : 'index.html'" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="form.remark" :placeholder="$t('sites.dialogs.create.remarkPlaceholder')" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          {{ $t('sites.dialogs.create.staticHint') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="creating" @click="doCreate">{{ $t('sites.shared.create') }}</FaButton>
      </template>
    </FaModal>

    <!-- 分组管理（B23 对齐 1Panel） -->
    <FaModal v-model="groupModalVisible" :title="$t('sites.list.groups')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-2">
          <FaInput v-model="newGroupName" :placeholder="$t('sites.dialogs.group.namePlaceholder')" class="flex-1" @keyup.enter="doCreateGroup" />
          <FaButton size="sm" :loading="groupBusy" @click="doCreateGroup">{{ $t('sites.dialogs.group.create') }}</FaButton>
        </div>
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('common.name') }}</th>
                <th class="px-3 py-2">{{ $t('sites.dialogs.group.siteCount') }}</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="g in groups" :key="g.id" class="border-t">
                <td class="px-3 py-2">
                  <template v-if="editingGroupName === g.id">
                    <input
                      v-model="editingGroupNameVal"
                      class="h-7 rounded border border-input bg-background px-1.5 text-xs outline-none focus:border-primary"
                      @keyup.enter="doGroupAction(g, 'rename')"
                    >
                  </template>
                  <template v-else>
                    {{ g.name }}
                    <span v-if="g.isDefault" class="ml-1 text-xs text-muted-foreground">{{ $t('sites.dialogs.group.defaultSuffix') }}</span>
                  </template>
                </td>
                <td class="px-3 py-2 text-xs text-muted-foreground">{{ g.sites }}</td>
                <td class="px-3 py-2 text-right">
                  <template v-if="editingGroupName !== g.id">
                    <FaButton variant="ghost" size="sm" @click="editingGroupName = g.id; editingGroupNameVal = g.name">{{ $t('common.edit') }}</FaButton>
                    <FaButton v-if="!g.isDefault" variant="ghost" size="sm" @click="doGroupAction(g, 'default')">{{ $t('sites.list.setDefault') }}</FaButton>
                    <FaButton v-if="!g.isDefault" variant="ghost" size="sm" class="text-red-500!" @click="doGroupAction(g, 'remove')">{{ $t('common.delete') }}</FaButton>
                  </template>
                  <template v-else>
                    <FaButton variant="ghost" size="sm" @click="doGroupAction(g, 'rename')">{{ $t('common.save') }}</FaButton>
                    <FaButton variant="ghost" size="sm" @click="editingGroupName = 0">{{ $t('common.cancel') }}</FaButton>
                  </template>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="text-xs text-muted-foreground">{{ $t('sites.dialogs.group.hint') }}</div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="groupModalVisible = false">{{ $t('common.close') }}</FaButton>
      </template>
    </FaModal>

    <!-- 删除站点（选项 + 域名确认） -->
    <!-- M39：到期时间 -->
    <FaModal v-model="expireVisible" :title="$t('sites.dialogs.expire.title', { name: expireTarget?.name || '' })" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <input v-model="expireVal" type="date" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none focus:border-primary">
        <p class="text-xs text-muted-foreground">{{ $t('sites.dialogs.expire.hint') }}</p>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="expireVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton @click="saveExpire">{{ $t('common.save') }}</FaButton>
      </template>
    </FaModal>
    <YdDangerDelete
      v-model:visible="delVisible"
      :title="$t('sites.dialogs.delete.title', { name: delTarget?.domain || '' })"
      :name="delTarget?.domain || ''"
      :options="SITE_DEL_OPTS"
      :loading="deleting"
      @confirm="doDelete"
    />
  
  <!-- M54-P3 属主分配 -->
  <YdOwnerDialog v-model="ownerVisible" :title="$t('common.owner.siteTitle')" :current-owner-id="ownerTarget.ownerId" @confirm="doSetOwner" />
</div>
</template>
