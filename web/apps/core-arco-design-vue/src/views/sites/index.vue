<script setup lang="ts">
import type { SiteItem, DiscoveredSite } from '@/api/modules/site'
import apiSite, { siteDiscoveryApi, wafApi, extApi } from '@/api/modules/site'
import type { SiteWaf, SiteExtConfig, RewriteTemplate } from '@/api/modules/site'

defineOptions({
  name: 'SitesIndex',
})

const status = ref<{ installed: boolean, running: boolean, sites: number }>()
const sites = ref<SiteItem[]>([])
const loading = ref(false)
const installing = ref(false)

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

// 创建
const createVisible = ref(false)
const form = ref({ name: '', type: 'static', domain: '', extraDomains: '', port: 80, proxyPass: '', indexFiles: 'index.html' })
const proxyRules = ref<{ prefix: string, target: string, ws?: boolean }[]>([{ prefix: '/api', target: '' }])
const creating = ref(false)

function openCreate() {
  form.value = { name: '', type: 'static', domain: '', extraDomains: '', port: 80, proxyPass: '', indexFiles: 'index.html' }
  proxyRules.value = [{ prefix: '/api', target: '' }]
  createVisible.value = true
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
    })
    useFaToast().success('站点已创建')
    createVisible.value = false
    await load()
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

// ---- 高级配置（伪静态/自定义 location/404/缓存）----
const extVisible = ref(false)
const extTarget = ref<SiteItem | null>(null)
const extForm = ref<SiteExtConfig>({ rewriteName: '', rewriteContent: '', customLocations: [], errorPage404: '', cacheEnable: false, cacheDuration: '12h' })
const rewriteTemplates = ref<RewriteTemplate[]>([])
const extSaving = ref(false)
const customLocs = ref<{ comment: string, content: string }[]>([])

async function openExt(s: SiteItem) {
  extTarget.value = s
  try {
    if (!rewriteTemplates.value.length) {
      rewriteTemplates.value = await extApi.rewriteTemplates()
    }
    extForm.value = await extApi.getExt(s.id)
    customLocs.value = extForm.value.customLocations?.length ? [...extForm.value.customLocations] : [{ comment: '', content: '' }]
    extVisible.value = true
  }
  catch (e: any) {
    useFaToast().error('读取高级配置失败', { description: e?.message })
  }
}

function onRewriteTemplate() {
  const t = rewriteTemplates.value.find(x => x.name === extForm.value.rewriteName)
  if (t) {
    extForm.value.rewriteContent = t.content
  }
  if (extForm.value.rewriteName === 'custom') {
    extForm.value.rewriteContent = ''
  }
}

function addCustomLoc() {
  customLocs.value.push({ comment: '', content: 'location /path {\n    return 200 "ok";\n}' })
}

async function saveExt() {
  if (!extTarget.value) {
    return
  }
  extSaving.value = true
  try {
    extForm.value.customLocations = customLocs.value.filter(c => c.content.trim())
    await extApi.updateExt(extTarget.value.id, extForm.value)
    useFaToast().success('高级配置已保存并重载 nginx')
    extVisible.value = false
  }
  catch (e: any) {
    useFaToast().error('保存失败（已回滚）', { description: e?.message })
  }
  finally {
    extSaving.value = false
  }
}

// ---- 站点日志 ----
const logsVisible = ref(false)
const logsTarget = ref<SiteItem | null>(null)
const logsType = ref<'access' | 'error'>('access')
const logsContent = ref('')
const logsLoading = ref(false)

async function openLogs(s: SiteItem) {
  logsTarget.value = s
  logsType.value = 'access'
  logsContent.value = ''
  logsVisible.value = true
  await loadLogs()
}

async function loadLogs() {
  if (!logsTarget.value) {
    return
  }
  logsLoading.value = true
  try {
    logsContent.value = await apiSite.siteLogs(logsTarget.value.id, logsType.value, 200)
  }
  catch (e: any) {
    logsContent.value = '读取失败：' + (e?.message || '')
  }
  finally {
    logsLoading.value = false
  }
}

// ---- WAF 配置 ----
const wafVisible = ref(false)
const wafTarget = ref<SiteItem | null>(null)
const wafForm = ref<SiteWaf>({ denyIps: [], allowIps: [], denyUAs: [], rateEnable: false, rate: 10, burst: 20 })
const wafSaving = ref(false)
const wafDenyIps = ref('')
const wafAllowIps = ref('')
const wafDenyUAs = ref('')

async function openWaf(s: SiteItem) {
  wafTarget.value = s
  try {
    const w = await wafApi.get(s.id)
    wafForm.value = { ...w, denyIps: w.denyIps || [], allowIps: w.allowIps || [], denyUAs: w.denyUAs || [] }
    wafDenyIps.value = (w.denyIps || []).join(', ')
    wafAllowIps.value = (w.allowIps || []).join(', ')
    wafDenyUAs.value = (w.denyUAs || []).join(', ')
    wafVisible.value = true
  }
  catch (e: any) {
    useFaToast().error('读取 WAF 配置失败', { description: e?.message })
  }
}

async function saveWaf() {
  if (!wafTarget.value) {
    return
  }
  const w: SiteWaf = {
    denyIps: wafDenyIps.value.split(/[,\n]/).map(x => x.trim()).filter(Boolean),
    allowIps: wafAllowIps.value.split(/[,\n]/).map(x => x.trim()).filter(Boolean),
    denyUAs: wafDenyUAs.value.split(/[,\n]/).map(x => x.trim()).filter(Boolean),
    rateEnable: wafForm.value.rateEnable,
    rate: Number(wafForm.value.rate) || 10,
    burst: Number(wafForm.value.burst) || 20,
  }
  wafSaving.value = true
  try {
    await wafApi.update(wafTarget.value.id, w)
    useFaToast().success('WAF 规则已保存并重载 nginx')
    wafVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error('保存失败（已回滚）', { description: e?.message })
  }
  finally {
    wafSaving.value = false
  }
}

async function issueCert(s: SiteItem) {
  try {
    await apiSite.issueSelfSigned(s.id)
    useFaToast().success('自签证书已签发并启用 443（浏览器需信任/忽略告警）')
    await load()
  }
  catch (e: any) {
    useFaToast().error('签发失败', { description: e?.message })
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
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
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

// 配置编辑
const editorVisible = ref(false)
const editorId = ref(0)
const editorName = ref('')
const editorContent = ref('')
const editorSaving = ref(false)

async function openEditor(s: SiteItem) {
  editorId.value = s.id
  editorName.value = s.name
  try {
    editorContent.value = await apiSite.config(s.id)
    editorVisible.value = true
  }
  catch (e: any) {
    useFaToast().error('读取配置失败', { description: e?.message })
  }
}

async function saveEditor() {
  editorSaving.value = true
  try {
    await apiSite.updateConfig(editorId.value, editorContent.value)
    useFaToast().success('配置已保存并重载')
    editorVisible.value = false
  }
  catch (e: any) {
    useFaToast().error('保存失败（配置校验不通过会自动回滚）', { description: e?.message })
  }
  finally {
    editorSaving.value = false
  }
}

onMounted(load)
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
        <span>容器化 nginx：静态站 / 反向代理 / 域名绑定 / 自签证书（ACME 预留）</span>
      </template>
      <div class="flex items-center gap-2">
        <span v-if="status?.installed" class="mr-2 inline-flex items-center gap-1.5 text-xs text-muted-foreground">
          <span class="inline-block size-1.5 rounded-full" :class="status.running ? 'bg-emerald-500' : 'bg-red-500'" />
          nginx {{ status.running ? '运行中' : '已停止' }} · {{ status.sites }} 个站点
        </span>
        <FaButton v-if="!status?.installed" size="sm" :loading="installing" @click="install">
          <YdMorphIcon name="download" :size="14" class="mr-1" /> 安装 nginx
        </FaButton>
        <FaButton v-if="status?.installed" variant="outline" size="sm" @click="openScan">
          <YdMorphIcon name="search" :size="14" class="mr-1" /> 扫描识别
        </FaButton>
        <FaButton v-if="status?.installed" size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 创建站点
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="status && !status.installed" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
        尚未安装 nginx（容器化，端口 80/443）。点击右上角"安装 nginx"开始。
      </div>

      <div v-else class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">站点</th>
              <th class="px-3 py-2">域名</th>
              <th class="px-3 py-2">类型</th>
              <th class="hidden px-3 py-2 lg:table-cell">后端 / 目录</th>
              <th class="px-3 py-2">SSL</th>
              <th class="px-3 py-2">状态</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !sites.length">
              <td colspan="7" class="px-3 py-10 text-center text-muted-foreground">加载中…</td>
            </tr>
            <tr v-else-if="!sites.length">
              <td colspan="7" class="px-3 py-10 text-center text-muted-foreground">暂无站点</td>
            </tr>
            <tr v-for="s in sites" :key="s.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2 font-medium">{{ s.name }}</td>
              <td class="px-3 py-2 font-mono text-[13px]">
                {{ s.domain }}
                <span v-if="s.domains?.length" class="text-muted-foreground"> +{{ s.domains.length }}</span>
                <a v-if="s.enabled" :href="`http://${s.domain}`" target="_blank" rel="noopener" class="ml-1 text-primary opacity-60" title="访问">↗</a>
              </td>
              <td class="px-3 py-2">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="s.type === 'static' ? 'bg-blue-500/10 text-blue-600' : 'bg-purple-500/10 text-purple-600'">
                  {{ s.type === 'static' ? '静态' : '反代' }}
                </span>
              </td>
              <td class="hidden max-w-56 truncate px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell">
                {{ s.type === 'proxy' ? s.proxyPass : `/var/www/sites/${s.name}` }}
              </td>
              <td class="px-3 py-2">
                <span v-if="s.certDomain" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">HTTPS</span>
                <span v-else class="text-xs text-muted-foreground">—</span>
              </td>
              <td class="px-3 py-2">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="s.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  {{ s.enabled ? '启用' : '禁用' }}
                </span>
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton variant="ghost" size="sm" @click="openEditor(s)">配置</FaButton>
                  <FaButton variant="ghost" size="sm" @click="openLogs(s)">日志</FaButton>
                  <FaButton variant="ghost" size="sm" @click="openWaf(s)">WAF</FaButton>
                  <FaButton variant="ghost" size="sm" @click="openExt(s)">高级</FaButton>
                  <FaButton v-if="!s.certDomain" variant="ghost" size="sm" @click="issueCert(s)">证书</FaButton>
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
    </FaPageMain>

    <!-- 创建 -->
    <FaModal v-model="createVisible" title="创建站点" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">类型</span>
          <div class="flex flex-1 gap-1.5">
            <button
              v-for="t in [{ v: 'static', l: '静态站' }, { v: 'proxy', l: '反向代理' }]"
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
        <div v-if="form.type === 'static'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">默认文档</span>
          <FaInput v-model="form.indexFiles" placeholder="index.html" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          静态站根目录 /var/www/sites/&lt;站点名&gt;（自动生成欢迎页）；域名解析到服务器 IP 后即可访问
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">取消</FaButton>
        <FaButton :loading="creating" @click="doCreate">创建</FaButton>
      </template>
    </FaModal>

    <!-- 高级配置 -->
    <FaModal v-model="extVisible" :title="`高级配置：${extTarget?.name || ''}`" class="max-w-3xl!" :destroy-on-close="true">
      <div class="flex flex-col gap-4">
        <div class="rounded-md border p-3">
          <div class="mb-2 text-sm font-medium">伪静态</div>
          <select v-model="extForm.rewriteName" class="mb-2 h-9 w-48 rounded-md border border-input bg-background px-2 text-sm outline-none" @change="onRewriteTemplate">
            <option value="">无</option>
            <option value="custom">自定义</option>
            <option v-for="t in rewriteTemplates" :key="t.name" :value="t.name">{{ t.name }}</option>
          </select>
          <textarea
            v-if="extForm.rewriteName === 'custom'"
            v-model="extForm.rewriteContent"
            class="h-24 w-full rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
            spellcheck="false"
            placeholder="location / { ... }"
          />
          <pre v-else-if="extForm.rewriteContent" class="max-h-32 overflow-auto rounded-md bg-muted/50 p-2 font-mono text-xs">{{ extForm.rewriteContent }}</pre>
        </div>

        <div class="rounded-md border p-3">
          <div class="mb-2 flex items-center justify-between">
            <span class="text-sm font-medium">自定义 location</span>
            <FaButton variant="outline" size="sm" @click="addCustomLoc">
              <FaIcon name="i-lucide:plus" class="mr-1" /> 加一条
            </FaButton>
          </div>
          <div v-for="(cl, idx) in customLocs" :key="idx" class="mb-2">
            <div class="mb-1 flex items-center gap-2">
              <FaInput v-model="cl.comment" placeholder="备注" class="w-40" />
              <FaButton v-if="customLocs.length > 1" variant="ghost" size="icon-sm" @click="customLocs.splice(idx, 1)">
                <FaIcon name="i-lucide:trash" class="text-sm" />
              </FaButton>
            </div>
            <textarea
              v-model="cl.content"
              class="h-20 w-full rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
              spellcheck="false"
              placeholder="location /healthz { return 200 'ok'; }"
            />
          </div>
        </div>

        <div class="rounded-md border p-3">
          <div class="mb-2 text-sm font-medium">自定义 404 页面</div>
          <FaInput v-model="extForm.errorPage404" placeholder="如 /404.html（相对站点根目录，仅静态站）" class="w-full" />
        </div>

        <div class="rounded-md border p-3">
          <label class="flex cursor-pointer items-center gap-2 text-sm font-medium">
            <input v-model="extForm.cacheEnable" type="checkbox">
            反代缓存（proxy_cache）
          </label>
          <div v-if="extForm.cacheEnable" class="mt-2 flex items-center gap-3">
            <span class="w-24 text-sm text-muted-foreground">缓存有效期</span>
            <select v-model="extForm.cacheDuration" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none">
              <option value="1h">1 小时</option>
              <option value="12h">12 小时</option>
              <option value="1d">1 天</option>
              <option value="7d">7 天</option>
            </select>
            <span class="text-xs text-muted-foreground">响应带 X-Cache-Status 头（MISS/HIT）</span>
          </div>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="extVisible = false">取消</FaButton>
        <FaButton :loading="extSaving" @click="saveExt">保存并重载</FaButton>
      </template>
    </FaModal>

    <!-- WAF 配置 -->
    <FaModal v-model="wafVisible" :title="`WAF：${wafTarget?.name || ''}`" class="max-w-2xl!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div>
          <div class="mb-1 text-sm font-medium">IP 黑名单（逗号分隔，命中返回 403）</div>
          <textarea v-model="wafDenyIps" class="h-16 w-full rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" placeholder="如 1.2.3.4, 5.6.7.8" />
        </div>
        <div>
          <div class="mb-1 text-sm font-medium">IP 白名单（逗号分隔；非空 = 仅白名单可访问）</div>
          <textarea v-model="wafAllowIps" class="h-16 w-full rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" />
        </div>
        <div>
          <div class="mb-1 text-sm font-medium">UA 拦截关键字（逗号分隔，命中返回 403）</div>
          <FaInput v-model="wafDenyUAs" placeholder="如 curl, bot, scanner" class="w-full" />
        </div>
        <div class="rounded-md border p-3">
          <label class="flex cursor-pointer items-center gap-2 text-sm font-medium">
            <input v-model="wafForm.rateEnable" type="checkbox">
            启用限流（nginx limit_req）
          </label>
          <div v-if="wafForm.rateEnable" class="mt-2 flex items-center gap-3">
            <span class="w-24 text-sm text-muted-foreground">速率（次/秒）</span>
            <FaInput v-model="wafForm.rate" type="number" class="w-28" />
            <span class="w-16 text-sm text-muted-foreground">突发</span>
            <FaInput v-model="wafForm.burst" type="number" class="w-28" />
          </div>
          <div class="mt-1 text-xs text-muted-foreground">超限返回 503；保存走 nginx -t 校验，失败自动回滚</div>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="wafVisible = false">取消</FaButton>
        <FaButton :loading="wafSaving" @click="saveWaf">保存并重载</FaButton>
      </template>
    </FaModal>

    <!-- 站点日志 -->
    <FaModal v-model="logsVisible" :title="`日志：${logsTarget?.name || ''}`" class="max-w-4xl!" :destroy-on-close="true">
      <div class="mb-2 flex items-center gap-2">
        <FaTabs
          v-model="logsType" :list="[
            { label: '访问日志', value: 'access' },
            { label: '错误日志', value: 'error' },
          ]" @change="loadLogs"
        />
        <FaButton variant="outline" size="sm" class="ml-auto" @click="loadLogs">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" :class="logsLoading ? 'animate-spin' : ''" /> 刷新
        </FaButton>
      </div>
      <pre class="h-96 overflow-auto rounded-md bg-muted/60 p-3 font-mono text-xs leading-relaxed">{{ logsContent || '暂无日志' }}</pre>
      <template #footer>
        <FaButton variant="outline" @click="logsVisible = false">关闭</FaButton>
      </template>
    </FaModal>

    <!-- 配置编辑 -->
    <FaModal
      v-model="editorVisible"
      :title="`配置：${editorName}.conf`"
      class="max-w-3xl!"
      :destroy-on-close="true"
    >
      <div class="mb-2 text-xs text-muted-foreground">保存时自动 nginx -t 校验，失败将回滚本次修改</div>
      <textarea
        v-model="editorContent"
        class="h-96 w-full resize-y rounded-md border border-input bg-background p-3 font-mono text-[13px] leading-relaxed outline-none focus:ring-1 focus:ring-primary"
        spellcheck="false"
      />
      <template #footer>
        <FaButton variant="outline" @click="editorVisible = false">取消</FaButton>
        <FaButton :loading="editorSaving" @click="saveEditor">保存并重载</FaButton>
      </template>
    </FaModal>
  </div>
</template>
