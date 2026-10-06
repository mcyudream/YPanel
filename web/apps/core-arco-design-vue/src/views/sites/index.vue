<script setup lang="ts">
import type { SiteItem, DiscoveredSite } from '@/api/modules/site'
import apiSite, { siteDiscoveryApi } from '@/api/modules/site'
import apiRuntime from '@/api/modules/runtime'
import { siteGroupApi } from '@/api/modules/cert'
import type { SiteGroup } from '@/api/modules/cert'

defineOptions({
  name: 'SitesIndex',
})

const router = useRouter()

const status = ref<{ installed: boolean, running: boolean, sites: number, mode?: 'container' | 'host' }>()
const sites = ref<SiteItem[]>([])
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
}

const filteredSites = computed(() => {
  let list = [...sites.value]
  if (filterType.value)
    list = list.filter(s => s.type === filterType.value)
  if (filterGroup.value !== '') {
    list = list.filter((s) => {
      const gid = s.groupId || groups.value.find(g => g.isDefault)?.id || 0
      return gid === filterGroup.value
    })
  }
  if (keyword.value.trim()) {
    const k = keyword.value.trim().toLowerCase()
    list = list.filter(s => s.name.includes(k) || s.domain.includes(k) || (s.remark || '').toLowerCase().includes(k))
  }
  if (sortKey.value) {
    list.sort((a, b) => {
      const va = String(a[sortKey.value as 'name' | 'type' | 'enabled'] ?? '')
      const vb = String(b[sortKey.value as 'name' | 'type' | 'enabled'] ?? '')
      return va.localeCompare(vb) * sortOrder.value
    })
  }
  return list
})

async function load() {
  loading.value = true
  try {
    status.value = await apiSite.status()
    if (status.value.installed) {
      sites.value = await apiSite.list()
    }
  }
  finally {
    loading.value = false
  }
}

async function install() {
  installing.value = true
  useFaToast().info('正在安装 nginx 容器…')
  try {
    await apiSite.install()
    useFaToast().success('nginx 已就绪')
    await load()
  }
  catch (e: any) {
    useFaToast().error('安装失败', { description: e?.message })
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
    useFaToast().success('已接管本机 nginx', { description: out.hint })
    await load()
  }
  catch (e: any) {
    useFaToast().error('接管失败', { description: e?.message })
  }
  finally {
    nginxAdopting.value = false
  }
}

// 创建
const createVisible = ref(false)
const form = ref({ name: '', type: 'static', domain: '', extraDomains: '', port: 80, proxyPass: '', indexFiles: 'index.html', runtimeId: 0, groupId: 0, remark: '' })
const proxyRules = ref<{ prefix: string, target: string, ws?: boolean }[]>([{ prefix: '/api', target: '' }])
const creating = ref(false)
const runtimes = ref<{ id: number, name: string, version: string, running: boolean }[]>([])

function openCreate() {
  const defGroup = groups.value.find(g => g.isDefault)
  form.value = { name: '', type: 'static', domain: '', extraDomains: '', port: 80, proxyPass: '', indexFiles: 'index.html', runtimeId: 0, groupId: defGroup?.id || 0, remark: '' }
  proxyRules.value = [{ prefix: '/api', target: '' }]
  createVisible.value = true
  apiRuntime.list().then((list) => {
    runtimes.value = list
    if (list.length && !form.value.runtimeId) {
      form.value.runtimeId = list[0].id
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
      port: form.value.port,
      proxyPass: form.value.proxyPass,
      proxyRules: form.value.type === 'proxy' ? rules : undefined,
      indexFiles: form.value.indexFiles,
      runtimeId: form.value.type === 'php' ? form.value.runtimeId : undefined,
      groupId: form.value.groupId || undefined,
      remark: form.value.remark || undefined,
    })
    useFaToast().success('站点已创建')
    createVisible.value = false
    await load()
    await loadGroups()
  }
  catch (e: any) {
    useFaToast().error('创建失败', { description: e?.message })
  }
  finally {
    creating.value = false
  }
}

async function toggle(s: SiteItem) {
  try {
    if (s.enabled) {
      await apiSite.disable(s.id)
      useFaToast().success('已禁用')
    }
    else {
      await apiSite.enable(s.id)
      useFaToast().success('已启用')
    }
    await load()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
  }
}

function remove(s: SiteItem) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除站点',
    content: `确认删除站点 ${s.name}（${s.domain}）？`,
    onConfirm: async () => {
      try {
        await apiSite.remove(s.id, true)
        useFaToast().success('已删除')
        await load()
        await loadGroups()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
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
    useFaToast().error('创建分组失败', { description: e?.message })
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
    useFaToast().error('操作失败', { description: e?.message })
  }
}

function groupNameOf(s: SiteItem) {
  return s.groupName || '默认'
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
    useFaToast().error('保存备注失败', { description: e?.message })
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
    useFaToast().error('扫描失败', { description: e?.message })
  }
  finally {
    scanning.value = false
  }
}

async function adopt(d: DiscoveredSite) {
  adopting.value = d.file
  try {
    await siteDiscoveryApi.adopt({ file: d.file, domain: d.domain, type: d.type, proxyPass: d.proxyPass })
    useFaToast().success(`已接管 ${d.domain}`)
    scanResult.value!.sites = scanResult.value!.sites.filter(x => x.file !== d.file)
    await load()
  }
  catch (e: any) {
    useFaToast().error('接管失败', { description: e?.message })
  }
  finally {
    adopting.value = ''
  }
}

onMounted(() => {
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
          <span>网站</span>
        </div>
      </template>
      <template #description>
        <span>nginx 环境（容器化 / 接管本机）：静态站 / 反向代理 / PHP / HTTPS 与证书管理</span>
      </template>
      <div class="flex items-center gap-2">
        <span v-if="status?.installed" class="mr-2 inline-flex items-center gap-1.5 text-xs text-muted-foreground">
          <span class="inline-block size-1.5 rounded-full" :class="status.running ? 'bg-emerald-500' : 'bg-red-500'" />
          nginx（{{ status.mode === 'host' ? '本机接管' : '容器' }}）{{ status.running ? '运行中' : '已停止' }} · {{ status.sites }} 个站点
        </span>
        <FaButton v-if="!status?.installed" size="sm" :loading="installing" @click="install">
          <YdMorphIcon name="download" :size="14" class="mr-1" /> 安装 nginx
        </FaButton>
        <FaButton v-if="!status?.installed" variant="outline" size="sm" :loading="nginxAdopting" @click="adoptHost">
          <FaIcon name="i-lucide:plug-zap" class="mr-1" /> 接管本机 nginx
        </FaButton>
        <FaButton v-if="status?.installed" variant="outline" size="sm" @click="openScan">
          <YdMorphIcon name="search" :size="14" class="mr-1" /> 扫描识别
        </FaButton>
        <FaButton v-if="status?.installed" variant="outline" size="sm" @click="openGroups">
          <FaIcon name="i-lucide:folder" class="mr-1" /> 分组
        </FaButton>
        <FaButton v-if="status?.installed" size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 创建站点
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="status && !status.installed" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
        尚未配置 nginx 环境：可"安装 nginx"（容器化，端口 80/443），或"接管本机 nginx"（systemd 管理的本机 nginx）。
      </div>

      <div v-else>
        <!-- 筛选栏（B23：类型 / 分组 / 搜索） -->
        <div class="mb-3 flex flex-wrap items-center gap-2">
          <select v-model="filterType" class="h-8 rounded-md border bg-background px-2 text-xs outline-none focus:border-primary">
            <option value="">全部类型</option>
            <option value="static">静态</option>
            <option value="proxy">反代</option>
            <option value="php">PHP</option>
          </select>
          <select v-model="filterGroup" class="h-8 rounded-md border bg-background px-2 text-xs outline-none focus:border-primary">
            <option value="">全部分组</option>
            <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}（{{ g.sites }}）</option>
          </select>
          <FaInput v-model="keyword" placeholder="搜索名称 / 域名 / 备注" class="w-56!" />
          <span class="ml-auto text-xs text-muted-foreground">共 {{ filteredSites.length }} 条</span>
        </div>

        <div class="overflow-x-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="cursor-pointer px-3 py-2 select-none" @click="setSort('name')">
                  名称
                  <FaIcon v-if="sortKey === 'name'" :name="sortOrder === 1 ? 'i-lucide:chevron-up' : 'i-lucide:chevron-down'" class="text-[10px]" />
                </th>
                <th class="cursor-pointer px-3 py-2 select-none" @click="setSort('type')">
                  类型
                  <FaIcon v-if="sortKey === 'type'" :name="sortOrder === 1 ? 'i-lucide:chevron-up' : 'i-lucide:chevron-down'" class="text-[10px]" />
                </th>
                <th class="hidden px-3 py-2 md:table-cell">分组</th>
                <th class="px-3 py-2">域名</th>
                <th class="px-3 py-2">协议</th>
                <th class="px-3 py-2">证书过期</th>
                <th class="cursor-pointer px-3 py-2 select-none" @click="setSort('enabled')">
                  状态
                  <FaIcon v-if="sortKey === 'enabled'" :name="sortOrder === 1 ? 'i-lucide:chevron-up' : 'i-lucide:chevron-down'" class="text-[10px]" />
                </th>
                <th class="hidden px-3 py-2 lg:table-cell">备注</th>
                <th class="px-3 py-2 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading && !sites.length">
                <td colspan="9" class="px-3 py-10 text-center text-muted-foreground">加载中…</td>
              </tr>
              <tr v-else-if="!filteredSites.length">
                <td colspan="9" class="px-3 py-10 text-center text-muted-foreground">暂无站点</td>
              </tr>
              <tr v-for="s in filteredSites" :key="s.id" class="border-t transition-colors hover:bg-accent/30">
                <td class="px-3 py-2 font-medium">
                  {{ s.name }}
                  <a v-if="s.enabled" :href="`http://${s.domain}`" target="_blank" rel="noopener" class="ml-1 text-primary opacity-60" title="访问">↗</a>
                </td>
                <td class="px-3 py-2">
                  <span class="rounded-full px-2 py-0.5 text-xs" :class="{ static: 'bg-blue-500/10 text-blue-600', proxy: 'bg-purple-500/10 text-purple-600', php: 'bg-emerald-500/10 text-emerald-600' }[s.type] || 'bg-muted text-muted-foreground'">
                    {{ { static: '静态', proxy: '反代', php: 'PHP' }[s.type] || s.type }}
                  </span>
                </td>
                <td class="hidden px-3 py-2 text-xs text-muted-foreground md:table-cell">{{ groupNameOf(s) }}</td>
                <td class="px-3 py-2 font-mono text-[13px]">
                  {{ s.domain }}
                  <span v-if="s.domains?.length" class="text-muted-foreground"> +{{ s.domains.length }}</span>
                </td>
                <td class="px-3 py-2">
                  <span v-if="s.certDomain" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">HTTPS</span>
                  <span v-else class="text-xs text-muted-foreground">HTTP</span>
                </td>
                <td class="px-3 py-2 text-xs">
                  <template v-if="s.certDomain && s.certNotAfter">
                    <span :class="(new Date(s.certNotAfter).getTime() - Date.now()) < 0 ? 'text-red-500' : (new Date(s.certNotAfter).getTime() - Date.now()) < 15 * 86400000 ? 'text-amber-500' : 'text-muted-foreground'">
                      {{ new Date(s.certNotAfter).toISOString().slice(0, 10) }}
                    </span>
                  </template>
                  <span v-else class="text-muted-foreground">—</span>
                </td>
                <td class="px-3 py-2">
                  <button
                    type="button"
                    class="cursor-pointer rounded-full px-2 py-0.5 text-xs transition-colors"
                    :class="s.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
                    @click="toggle(s)"
                  >
                    {{ s.enabled ? '已启动' : '已停止' }}
                  </button>
                </td>
                <td class="hidden max-w-40 px-3 py-2 lg:table-cell">
                  <template v-if="remarkEditing === s.id">
                    <input
                      v-model="remarkEditingVal"
                      class="h-7 w-full rounded border border-input bg-background px-1.5 text-xs outline-none focus:border-primary"
                      @keyup.enter="saveRemark(s)"
                      @blur="saveRemark(s)"
                    >
                  </template>
                  <template v-else>
                    <span
                      class="cursor-pointer truncate text-xs" :class="s.remark ? 'text-foreground' : 'text-muted-foreground/50'"
                      :title="'点击编辑备注'"
                      @click="startEditRemark(s)"
                    >
                      {{ s.remark || '备注…' }}
                    </span>
                  </template>
                </td>
                <td class="px-3 py-2">
                  <div class="flex items-center justify-end gap-1">
                    <FaButton size="sm" @click="router.push(`/sites/${s.id}`)">配置</FaButton>
                    <FaButton variant="outline" size="sm" @click="toggle(s)">{{ s.enabled ? '禁用' : '启用' }}</FaButton>
                    <FaButton variant="outline" size="sm" class="text-red-500!" @click="remove(s)">删除</FaButton>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 扫描识别结果 -->
        <div v-if="scanVisible" class="mt-4 rounded-lg border bg-background p-4">
          <div class="mb-3 flex items-center justify-between">
            <div class="flex items-center gap-2 text-sm font-medium">
              <YdMorphIcon name="search" :size="16" />
              未接管的站点配置（{{ scanResult?.sites.length || 0 }}）
            </div>
            <FaButton variant="ghost" size="icon-sm" @click="scanVisible = false">
              <FaIcon name="i-lucide:x" class="text-sm" />
            </FaButton>
          </div>
          <div v-if="scanning" class="py-6 text-center text-sm text-muted-foreground">扫描中…</div>
          <template v-else>
            <div v-if="!scanResult?.sites.length" class="pb-3 text-sm text-muted-foreground">没有发现未接管的配置</div>
            <div v-for="d in scanResult?.sites || []" :key="d.file" class="mb-2 flex flex-wrap items-center gap-3 rounded-md border px-3 py-2">
              <span class="font-mono text-xs">{{ d.file }}</span>
              <span class="rounded-full bg-blue-500/10 px-2 py-0.5 text-xs text-blue-600">{{ d.type === 'proxy' ? '反代' : '静态' }}</span>
              <span class="font-mono text-xs text-muted-foreground">{{ d.domain }}</span>
              <span v-if="d.proxyPass" class="font-mono text-xs text-muted-foreground">→ {{ d.proxyPass }}</span>
              <FaButton class="ml-auto" size="sm" :loading="adopting === d.file" @click="adopt(d)">一键接管</FaButton>
            </div>
            <div v-if="scanResult?.containers.length" class="mt-3 border-t pt-3">
              <div class="mb-2 text-xs text-muted-foreground">检测到独立 Web 服务器容器（未纳管，仅提示）：</div>
              <div v-for="c in scanResult.containers" :key="c.name" class="mb-1 font-mono text-xs text-muted-foreground">
                {{ c.name }} · {{ c.image }} · {{ c.ports }}
              </div>
            </div>
          </template>
        </div>
      </div>
    </FaPageMain>

    <!-- 创建 -->
    <FaModal v-model="createVisible" title="创建站点" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">类型</span>
          <div class="flex flex-1 gap-1.5">
            <button
              v-for="t in [{ v: 'static', l: '静态站' }, { v: 'proxy', l: '反向代理' }, { v: 'php', l: 'PHP 站点' }]"
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
          <span class="w-20 shrink-0 text-sm text-muted-foreground">分组</span>
          <select v-model.number="form.groupId" class="h-9 flex-1 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
            <option :value="0">默认</option>
            <option v-for="g in groups.filter(x => !x.isDefault)" :key="g.id" :value="g.id">{{ g.name }}</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">站点名</span>
          <FaInput v-model="form.name" placeholder="小写字母/数字/中划线" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">主域名</span>
          <FaInput v-model="form.domain" placeholder="如 demo.example.com（本地测试可配 hosts）" class="flex-1" />
        </div>
        <div class="flex items-start gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">附加域名</span>
          <textarea v-model="form.extraDomains" class="h-14 flex-1 rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" placeholder="每行一个，可选，如&#10;www.demo.example.com&#10;demo2.example.com" />
        </div>
        <div v-if="form.type === 'php'" class="rounded-md border p-3">
          <div class="mb-2 text-sm font-medium">PHP 运行环境</div>
          <select v-model.number="form.runtimeId" class="h-9 w-full rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
            <option v-if="!runtimes.length" :value="0" disabled>暂无运行环境，请先到「运行环境」创建 php-fpm 实例</option>
            <option v-for="r in runtimes" :key="r.id" :value="r.id">
              php-{{ r.name }}（{{ r.version }}）{{ r.running ? '· 运行中' : '· 已停止' }}
            </option>
          </select>
          <div class="mt-1 text-xs text-muted-foreground">站点根目录 /var/www/sites/站点名，PHP 文件经 fastcgi 转发到所选 php-fpm 容器</div>
        </div>
        <div v-if="form.type === 'proxy'" class="rounded-md border p-3">
          <div class="mb-2 flex items-center justify-between">
            <span class="text-sm font-medium">反向代理规则</span>
            <FaButton variant="outline" size="sm" @click="proxyRules.push({ prefix: '/api', target: '' })">
              <FaIcon name="i-lucide:plus" class="mr-1" /> 加规则
            </FaButton>
          </div>
          <div v-for="(r, idx) in proxyRules" :key="idx" class="mb-2 flex items-center gap-2">
            <FaInput v-model="r.prefix" placeholder="前缀 /api" class="w-32" />
            <span class="text-muted-foreground">→</span>
            <FaInput v-model="r.target" placeholder="http://172.17.0.1:3000" class="flex-1" />
            <label class="flex shrink-0 cursor-pointer items-center gap-1 text-xs">
              <input v-model="r.ws" type="checkbox"> WS
            </label>
            <FaButton v-if="proxyRules.length > 1" variant="ghost" size="icon-sm" @click="proxyRules.splice(idx, 1)">
              <FaIcon name="i-lucide:trash" class="text-sm" />
            </FaButton>
          </div>
          <div class="text-xs text-muted-foreground">规则自动启用 WebSocket 支持（Upgrade/Connection 头）</div>
        </div>
        <div v-if="form.type === 'static' || form.type === 'php'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">默认文档</span>
          <FaInput v-model="form.indexFiles" :placeholder="form.type === 'php' ? 'index.php' : 'index.html'" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">备注</span>
          <FaInput v-model="form.remark" placeholder="站点用途说明（可选）" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          静态站根目录 /var/www/sites/&lt;站点名&gt;（自动生成欢迎页）；PHP 框架可在站点「网站目录」中配置运行目录（如 /public）
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">取消</FaButton>
        <FaButton :loading="creating" @click="doCreate">创建</FaButton>
      </template>
    </FaModal>

    <!-- 分组管理（B23 对齐 1Panel） -->
    <FaModal v-model="groupModalVisible" title="分组" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-2">
          <FaInput v-model="newGroupName" placeholder="新分组名称" class="flex-1" @keyup.enter="doCreateGroup" />
          <FaButton size="sm" :loading="groupBusy" @click="doCreateGroup">创建分组</FaButton>
        </div>
        <div class="overflow-hidden rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">名称</th>
                <th class="px-3 py-2">站点数</th>
                <th class="px-3 py-2 text-right">操作</th>
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
                    <span v-if="g.isDefault" class="ml-1 text-xs text-muted-foreground">(默认)</span>
                  </template>
                </td>
                <td class="px-3 py-2 text-xs text-muted-foreground">{{ g.sites }}</td>
                <td class="px-3 py-2 text-right">
                  <template v-if="editingGroupName !== g.id">
                    <FaButton variant="ghost" size="sm" @click="editingGroupName = g.id; editingGroupNameVal = g.name">编辑</FaButton>
                    <FaButton v-if="!g.isDefault" variant="ghost" size="sm" @click="doGroupAction(g, 'default')">设为默认</FaButton>
                    <FaButton v-if="!g.isDefault" variant="ghost" size="sm" class="text-red-500!" @click="doGroupAction(g, 'remove')">删除</FaButton>
                  </template>
                  <template v-else>
                    <FaButton variant="ghost" size="sm" @click="doGroupAction(g, 'rename')">保存</FaButton>
                    <FaButton variant="ghost" size="sm" @click="editingGroupName = 0">取消</FaButton>
                  </template>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="text-xs text-muted-foreground">删除分组时，组内站点自动归入默认分组；默认分组不可删除。</div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="groupModalVisible = false">关闭</FaButton>
      </template>
    </FaModal>
  </div>
</template>
