<script setup lang="ts">
import type { StoreApp, StoreVersion } from '@/api/modules/store'
import { storeApi } from '@/api/modules/store'

defineOptions({
  name: 'StoreIndex',
})

const apps = ref<StoreApp[]>([])
const loading = ref(false)
const syncing = ref(false)
const search = ref('')
const activeTag = ref('全部')
const installedProjects = ref<string[]>([])

const TAGS = ['全部', 'AI', '建站', '数据库', '缓存', '监控', '网关', '开发工具', '运维', '网盘', '媒体', '通信', '安全']

async function load() {
  loading.value = true
  try {
    apps.value = await storeApi.list(search.value, activeTag.value === '全部' ? '' : activeTag.value)
    const installed = await storeApi.installed()
    installedProjects.value = installed.map(i => i.composeProject)
  }
  finally {
    loading.value = false
  }
}

async function sync(force = false) {
  syncing.value = true
  try {
    const out = await storeApi.sync(force)
    if (out.skipped) {
      useFaToast().info('源近期已同步')
    }
    else {
      useFaToast().success(`同步完成：源共 ${out.total} 个应用`)
    }
    await load()
  }
  catch (e: any) {
    useFaToast().error('同步失败', { description: e?.message })
  }
  finally {
    syncing.value = false
  }
}

// 安装向导
const installVisible = ref(false)
const installApp = ref<StoreApp | null>(null)
const installVersions = ref<StoreVersion[]>([])
const installForm = ref({ version: '', name: '', params: {} as Record<string, string> })
const installFields = ref<{ envKey: string, label: Record<string, string>, default?: unknown, required?: boolean, type?: string }[]>([])
const installing = ref(false)
const installLogs = ref('')
const installProject = ref('')

async function openInstall(app: StoreApp) {
  installApp.value = app
  installForm.value = { version: '', name: '', params: {} }
  installLogs.value = ''
  installProject.value = ''
  try {
    const detail = await storeApi.get(app.key)
    installVersions.value = detail.versions
    const latest = detail.versions[detail.versions.length - 1]
    if (latest) {
      installForm.value.version = latest.id
      installFields.value = latest.formFields.filter(f => f.envKey)
      const params: Record<string, string> = {}
      for (const f of latest.formFields) {
        if (f.default !== undefined && f.default !== null) {
          params[f.envKey] = String(f.default)
        }
      }
      installForm.value.params = params
      installForm.value.name = app.key
    }
    installVisible.value = true
  }
  catch (e: any) {
    useFaToast().error('读取详情失败', { description: e?.message })
  }
}

async function doInstall() {
  if (!installApp.value) {
    return
  }
  installing.value = true
  installLogs.value = '下载应用包并启动中…'
  try {
    const out = await storeApi.install({
      key: installApp.value.key,
      version: installForm.value.version,
      name: installForm.value.name,
      params: installForm.value.params,
    })
    installProject.value = out.project
    installLogs.value = out.logs || '完成'
    useFaToast().success(`应用 ${out.project} 已安装`)
    await load()
  }
  catch (e: any) {
    useFaToast().error('安装失败', { description: e?.message })
    if (e?.message) {
      installLogs.value = e.message
    }
  }
  finally {
    installing.value = false
  }
}

function uninstall(p: string) {
  const modal = useFaModal()
  modal.confirm({
    title: '卸载应用',
    content: `确认卸载 ${p}？容器与 compose 目录将被移除（数据卷保留在项目目录）。`,
    onConfirm: async () => {
      try {
        await storeApi.uninstall(p)
        useFaToast().success('已卸载')
        await load()
      }
      catch (e: any) {
        useFaToast().error('卸载失败', { description: e?.message })
      }
    },
  })
}

// 参数 label 取中文
function fieldLabel(f: { label: Record<string, string>, envKey?: string }) {
  return f.label?.zh || f.label?.en || f.envKey || ''
}

watch([search, activeTag], () => load())

onMounted(() => sync())
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
        <span>已接 1Panel 官方应用源（265+ 应用）：一键安装为容器编排项目；安装前备份参数，卸载保留数据卷</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" :loading="syncing" @click="sync(true)">
          <FaIcon name="i-lucide:cloud-download" class="mr-1" /> 同步源
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 分类 + 搜索 -->
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="t in TAGS"
            :key="t"
            type="button"
            class="cursor-pointer rounded-md border px-2.5 py-1 text-sm transition-colors"
            :class="activeTag === t ? 'border-primary bg-primary/10 text-primary' : 'border-border hover:bg-accent/50'"
            @click="activeTag = t"
          >
            {{ t }}
          </button>
        </div>
        <FaInput v-model="search" placeholder="搜索应用…" class="ml-auto w-56">
          <template #start>
            <FaIcon name="i-lucide:search" class="text-muted-foreground" />
          </template>
        </FaInput>
      </div>

      <!-- 应用卡片网格 -->
      <div v-if="loading && !apps.length" class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <div v-for="i in 8" :key="i" class="h-28 animate-pulse rounded-lg border bg-muted/30" />
      </div>
      <div v-else-if="!apps.length" class="rounded-lg border p-10 text-center text-sm text-muted-foreground">
        未找到匹配应用（源未同步时点击右上角"同步源"）
      </div>
      <!-- 已安装 -->
      <div v-if="installedProjects.length" class="mb-4 rounded-lg border bg-background p-4">
        <div class="mb-2 flex items-center gap-2 text-sm font-medium">
          <YdMorphIcon name="check-circle" :size="16" />
          已安装（{{ installedProjects.length }}）
        </div>
        <div class="flex flex-wrap gap-2">
          <span
            v-for="p in installedProjects"
            :key="p"
            class="inline-flex items-center gap-2 rounded-md border px-2.5 py-1 font-mono text-xs"
          >
            {{ p }}
            <button type="button" class="cursor-pointer text-red-500 opacity-70 hover:opacity-100" title="卸载" @click="uninstall(p)">✕</button>
          </span>
        </div>
      </div>

      <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <div v-for="a in apps" :key="a.key" class="flex flex-col rounded-lg border bg-background p-4 transition-shadow hover:shadow-md">
          <div class="flex items-start gap-3">
            <img :src="a.iconUrl" class="size-10 shrink-0 rounded-lg object-contain" loading="lazy" @error="($event.target as HTMLImageElement).style.opacity = '0.2'">
            <div class="min-w-0">
              <div class="truncate font-medium">{{ a.name }}</div>
              <div class="truncate text-xs text-muted-foreground" :title="a.title">{{ a.title }}</div>
            </div>
          </div>
          <div class="mt-2 line-clamp-2 min-h-10 flex-1 text-xs text-muted-foreground">{{ a.description }}</div>
          <div class="mt-3 flex items-center justify-between border-t pt-2">
            <div class="flex gap-1">
              <span v-for="t in (a.tags || '').split(',').filter(Boolean).slice(0, 2)" :key="t" class="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">{{ t }}</span>
            </div>
            <FaButton size="sm" @click="openInstall(a)">安装</FaButton>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 安装向导 -->
    <FaModal v-model="installVisible" :title="`安装：${installApp?.name || ''}`" class="max-w-2xl!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">应用名</span>
          <FaInput v-model="installForm.name" placeholder="小写字母/数字/中划线" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">版本</span>
          <select v-model="installForm.version" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option v-for="v in installVersions" :key="v.id" :value="v.id">{{ v.name }}</option>
          </select>
        </div>
        <div v-for="f in installFields" :key="f.envKey" class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ fieldLabel(f) }}</span>
          <FaInput
            v-model="installForm.params[f.envKey]"
            :type="f.type === 'number' ? 'number' : 'text'"
            :placeholder="f.required ? '必填' : '可选'"
            class="flex-1"
          />
        </div>
        <div class="text-xs text-muted-foreground">
          安装为 compose 项目 app-&lt;应用名&gt;，容器接入 1panel-network；卸载保留数据卷
        </div>
        <pre v-if="installLogs" class="max-h-40 overflow-auto whitespace-pre-wrap rounded-md bg-muted/50 p-2 font-mono text-xs">{{ installLogs }}</pre>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="installVisible = false">取消</FaButton>
        <FaButton :loading="installing" @click="doInstall">安装</FaButton>
      </template>
    </FaModal>
  </div>
</template>
