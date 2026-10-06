<script setup lang="ts">
import type { FileEntry } from '@/api/modules/file'
import apiFile from '@/api/modules/file'
import { fmtBytes } from '@/utils/format'
import { useFaModal } from '@fantastic-admin/components'
import FileEditorWorkspace from './editor/Workspace.vue'

defineOptions({
  name: 'FileManagementIndex',
})

const appAccountStore = useAppAccountStore()
const fileEditorStore = useFileEditorStore()

const cwd = ref('/')
const entries = ref<FileEntry[]>([])
const loading = ref(false)
const selected = ref<Set<string>>(new Set())
const errorMsg = ref('')

// ---- 格式化 ----


function fmtTime(iso: string) {
  return new Date(iso).toLocaleString('zh-CN', { hour12: false })
}

// 面包屑：/a/b/c → [{name:'/',path:'/'},{name:'a',path:'/a'}...]
const crumbs = computed(() => {
  const parts = cwd.value.split('/').filter(Boolean)
  const out = [{ name: '/', path: '/' }]
  let acc = ''
  for (const p of parts) {
    acc += `/${p}`
    out.push({ name: p, path: acc })
  }
  return out
})

async function load(path = cwd.value) {
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await apiFile.list(path)
    cwd.value = res.path
    entries.value = res.entries
    selected.value = new Set()
  }
  catch (e: any) {
    errorMsg.value = e?.message || '目录读取失败'
  }
  finally {
    loading.value = false
  }
}

function open(entry: FileEntry) {
  if (entry.isDir) {
    load(entry.path)
  }
  else {
    openEditor(entry)
  }
}

// ---- 选择 ----
function toggleSelect(entry: FileEntry) {
  const next = new Set(selected.value)
  if (next.has(entry.path)) {
    next.delete(entry.path)
  }
  else {
    next.add(entry.path)
  }
  selected.value = next
}

const allSelected = computed(() => entries.value.length > 0 && selected.value.size === entries.value.length)

function toggleAll() {
  selected.value = allSelected.value ? new Set() : new Set(entries.value.map(e => e.path))
}

// ---- 新建 / 重命名 ----
const mkdirVisible = ref(false)
const mkdirName = ref('')

async function doMkdir() {
  if (!mkdirName.value.trim()) {
    return
  }
  const target = joinPath(cwd.value, mkdirName.value.trim())
  await apiFile.mkdir(target)
  mkdirVisible.value = false
  mkdirName.value = ''
  useFaToast().success('目录已创建')
  load()
}

const renameVisible = ref(false)
const renameTarget = ref<FileEntry | null>(null)
const renameName = ref('')

function openRename(entry: FileEntry) {
  renameTarget.value = entry
  renameName.value = entry.name
  renameVisible.value = true
}

async function doRename() {
  if (!renameTarget.value || !renameName.value.trim()) {
    return
  }
  await apiFile.rename(renameTarget.value.path, joinPath(cwd.value, renameName.value.trim()))
  renameVisible.value = false
  useFaToast().success('已重命名')
  load()
}

async function doDelete() {
  if (!selected.value.size) {
    return
  }
  const paths = [...selected.value]
  const modal = useFaModal()
  modal.confirm({
    title: '删除确认',
    content: `确认删除选中的 ${paths.length} 项？目录将递归删除，不可恢复。`,
    onConfirm: async () => {
      await apiFile.delete(paths)
      useFaToast().success('已删除')
      load()
    },
  })
}

// ---- 编辑器（VS Code 式工作台弹窗） ----
async function openEditor(entry: FileEntry) {
  await fileEditorStore.openWorkspace(entry.path)
}

// ---- 上传 / 下载 ----
const uploadInputRef = useTemplateRef<HTMLInputElement>('uploadInput')
const uploading = ref(false)
const uploadPercent = ref(0)

function pickUpload() {
  uploadInputRef.value?.click()
}

async function onUploadChange(ev: Event) {
  const input = ev.target as HTMLInputElement
  const files = input.files
  if (!files?.length) {
    return
  }
  uploading.value = true
  try {
    for (const f of Array.from(files)) {
      await apiFile.upload(cwd.value, f, (p) => {
        uploadPercent.value = p
      })
    }
    useFaToast().success('上传完成')
    load()
  }
  catch (e: any) {
    useFaToast().error('上传失败', { description: e?.message })
  }
  finally {
    uploading.value = false
    uploadPercent.value = 0
    input.value = ''
  }
}

function downloadURL(entry: FileEntry) {
  return apiFile.downloadURL(entry.path, appAccountStore.token)
}

function openDownload(entry: FileEntry) {
  window.open(downloadURL(entry))
}

// ---- 权限 ----
const chmodVisible = ref(false)
const chmodTarget = ref<FileEntry | null>(null)
const chmodMode = ref('')

function openChmod(entry: FileEntry) {
  chmodTarget.value = entry
  // modeOct 形如 0755，取后三位
  chmodMode.value = entry.modeOct.slice(-3) || '644'
  chmodVisible.value = true
}

async function doChmod() {
  if (!chmodTarget.value || !/^[0-7]{3}$/.test(chmodMode.value)) {
    useFaToast().error('权限格式错误（3 位八进制，如 755）')
    return
  }
  try {
    await apiFile.chmod(chmodTarget.value.path, chmodMode.value)
    chmodVisible.value = false
    useFaToast().success('权限已修改')
    load()
  }
  catch (e: any) {
    useFaToast().error('修改失败', { description: e?.message })
  }
}

// ---- 压缩 / 解压 ----
const compressVisible = ref(false)
const compressTarget = ref<FileEntry | null>(null)
const compressDest = ref('')
const compressing = ref(false)

function openCompress(entry: FileEntry) {
  compressTarget.value = entry
  compressDest.value = joinPath(cwd.value, `${entry.name}.tar.gz`)
  compressVisible.value = true
}

async function doCompress() {
  if (!compressTarget.value || !compressDest.value.trim()) {
    return
  }
  compressing.value = true
  try {
    await apiFile.compress(compressTarget.value.path, compressDest.value.trim())
    compressVisible.value = false
    useFaToast().success('压缩完成')
    load()
  }
  catch (e: any) {
    useFaToast().error('压缩失败', { description: e?.message })
  }
  finally {
    compressing.value = false
  }
}

const archivePattern = /\.(tar\.gz|tgz|tar|zip)$/i

function isArchive(entry: FileEntry) {
  return !entry.isDir && archivePattern.test(entry.name)
}

const decompressVisible = ref(false)
const decompressTarget = ref<FileEntry | null>(null)
const decompressDest = ref('')
const decompressing = ref(false)

function openDecompress(entry: FileEntry) {
  decompressTarget.value = entry
  decompressDest.value = cwd.value
  decompressVisible.value = true
}

async function doDecompress() {
  if (!decompressTarget.value || !decompressDest.value.trim()) {
    return
  }
  decompressing.value = true
  try {
    await apiFile.decompress(decompressTarget.value.path, decompressDest.value.trim())
    decompressVisible.value = false
    useFaToast().success('解压完成')
    load()
  }
  catch (e: any) {
    useFaToast().error('解压失败', { description: e?.message })
  }
  finally {
    decompressing.value = false
  }
}

// ---- 搜索 ----
const searchKeyword = ref('')
const searching = ref(false)
const searchVisible = ref(false)
const searchResults = ref<FileEntry[]>([])

async function doSearch() {
  const kw = searchKeyword.value.trim()
  if (!kw) {
    return
  }
  searching.value = true
  try {
    searchResults.value = await apiFile.search(cwd.value, kw)
    searchVisible.value = true
  }
  catch (e: any) {
    useFaToast().error('搜索失败', { description: e?.message })
  }
  finally {
    searching.value = false
  }
}

function joinPath(dir: string, name: string) {
  if (dir.endsWith('/')) {
    return dir + name
  }
  return `${dir}/${name}`
}

onMounted(() => load('/'))
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="folder-open" :size="24" />
          <span>文件管理</span>
        </div>
      </template>
      <template #description>
        <span>浏览、编辑与管理服务器文件（默认根目录为全盘）</span>
      </template>
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-1">
          <FaInput
            v-model="searchKeyword"
            placeholder="在当前目录下搜索…"
            class="w-44!"
            @keyup.enter="doSearch"
          />
          <FaButton variant="outline" size="sm" :loading="searching" @click="doSearch">
            <FaIcon name="i-lucide:search" class="mr-1" /> 搜索
          </FaButton>
        </div>
        <FaButton variant="outline" size="sm" @click="mkdirVisible = true">
          <FaIcon name="i-lucide:folder-plus" class="mr-1" /> 新建目录
        </FaButton>
        <FaButton variant="outline" size="sm" :disabled="uploading" @click="pickUpload">
          <FaIcon name="i-lucide:upload" class="mr-1" /> {{ uploading ? `上传中 ${uploadPercent}%` : '上传文件' }}
        </FaButton>
        <FaButton variant="outline" size="sm" :disabled="!selected.size" @click="doDelete">
          <FaIcon name="i-lucide:trash-2" class="mr-1" /> 删除 ({{ selected.size }})
        </FaButton>
        <FaButton variant="outline" size="icon-sm" title="刷新" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" />
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 面包屑 -->
      <div class="mb-3 flex flex-wrap items-center gap-1 text-sm">
        <template v-for="(c, i) in crumbs" :key="c.path">
          <span v-if="i" class="text-muted-foreground">/</span>
          <button
            type="button"
            class="cursor-pointer rounded px-1 py-0.5 font-mono transition-colors hover:bg-accent/60"
            :class="i === crumbs.length - 1 ? 'text-foreground font-medium' : 'text-muted-foreground'"
            @click="load(c.path)"
          >
            {{ c.name }}
          </button>
        </template>
        <span class="ml-auto text-xs text-muted-foreground">共 {{ entries.length }} 项</span>
      </div>

      <div v-if="errorMsg" class="mb-3 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
        {{ errorMsg }}
      </div>

      <!-- 文件表格 -->
      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="w-10 px-3 py-2">
                <input type="checkbox" :checked="allSelected" @change="toggleAll">
              </th>
              <th class="px-3 py-2">名称</th>
              <th class="hidden w-40 px-3 py-2 md:table-cell">大小</th>
              <th class="hidden w-56 px-3 py-2 lg:table-cell">属主 / 权限</th>
              <th class="hidden w-44 px-3 py-2 sm:table-cell">修改时间</th>
              <th class="w-48 px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                加载中…
              </td>
            </tr>
            <tr v-else-if="!cwd || cwd !== '/'">
              <td colspan="6" class="px-3 py-1">
                <button
                  type="button"
                  class="flex w-full cursor-pointer items-center gap-2 rounded px-1 py-1.5 hover:bg-accent/50"
                  @click="load(crumbs.length > 2 ? crumbs[crumbs.length - 2].path : '/')"
                >
                  <YdMorphIcon name="corner-left-up" :size="16" class="text-muted-foreground" />
                  <span>..</span>
                </button>
              </td>
            </tr>
            <tr
              v-for="e in entries"
              :key="e.path"
              class="border-t transition-colors hover:bg-accent/30"
              :class="selected.has(e.path) ? 'bg-primary/5' : ''"
            >
              <td class="px-3 py-1.5">
                <input type="checkbox" :checked="selected.has(e.path)" @change="toggleSelect(e)">
              </td>
              <td class="max-w-0 px-3 py-1.5">
                <button type="button" class="flex w-full cursor-pointer items-center gap-2" :title="e.path" @click="open(e)">
                  <YdMorphIcon
                    :name="e.isDir ? 'folder' : 'file'"
                    :size="16"
                    :class="e.isDir ? 'text-amber-500' : 'text-muted-foreground'"
                  />
                  <span class="truncate font-mono text-[13px]">{{ e.name }}</span>
                  <span v-if="e.target" class="truncate text-xs text-muted-foreground" :title="e.target">→ {{ e.target }}</span>
                </button>
              </td>
              <td class="hidden px-3 py-1.5 text-xs tabular-nums text-muted-foreground md:table-cell">
                {{ e.isDir ? '—' : fmtBytes(e.size) }}
              </td>
              <td class="hidden px-3 py-1.5 text-xs text-muted-foreground lg:table-cell">
                {{ e.owner }}:{{ e.group }} · {{ e.modeOct }}
              </td>
              <td class="hidden px-3 py-1.5 text-xs tabular-nums text-muted-foreground sm:table-cell">
                {{ fmtTime(e.modTime) }}
              </td>
              <td class="px-3 py-1.5 text-right">
                <div class="inline-flex items-center gap-0.5">
                  <FaButton v-if="!e.isDir" variant="ghost" size="icon-sm" title="编辑" @click="openEditor(e)">
                    <FaIcon name="i-lucide:pen-line" class="text-sm" />
                  </FaButton>
                  <FaButton v-if="!e.isDir" variant="ghost" size="icon-sm" title="下载" @click="openDownload(e)">
                    <FaIcon name="i-lucide:download" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" title="权限" @click="openChmod(e)">
                    <FaIcon name="i-lucide:lock" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" title="压缩" @click="openCompress(e)">
                    <FaIcon name="i-lucide:package" class="text-sm" />
                  </FaButton>
                  <FaButton v-if="isArchive(e)" variant="ghost" size="icon-sm" title="解压" @click="openDecompress(e)">
                    <FaIcon name="i-lucide:package-open" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" title="重命名" @click="openRename(e)">
                    <FaIcon name="i-lucide:text-cursor-input" class="text-sm" />
                  </FaButton>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && !entries.length && (!cwd || cwd === '/')">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                目录为空
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <!-- 新建目录 -->
    <FaModal v-model="mkdirVisible" title="新建目录" :destroy-on-close="true">
      <FaInput v-model="mkdirName" placeholder="目录名" class="w-full" @keyup.enter="doMkdir" />
      <template #footer>
        <FaButton variant="outline" @click="mkdirVisible = false">
          取消
        </FaButton>
        <FaButton @click="doMkdir">
          创建
        </FaButton>
      </template>
    </FaModal>

    <!-- 重命名 -->
    <FaModal v-model="renameVisible" title="重命名" :destroy-on-close="true">
      <FaInput v-model="renameName" placeholder="新名称" class="w-full" @keyup.enter="doRename" />
      <template #footer>
        <FaButton variant="outline" @click="renameVisible = false">
          取消
        </FaButton>
        <FaButton @click="doRename">
          确认
        </FaButton>
      </template>
    </FaModal>

    <!-- 文件编辑工作台（VS Code 式弹窗） -->
    <FileEditorWorkspace />

    <!-- 隐藏上传控件 -->
    <input ref="uploadInput" type="file" multiple class="hidden" @change="onUploadChange">

    <!-- 权限 -->
    <FaModal v-model="chmodVisible" title="修改权限" :destroy-on-close="true">
      <div class="space-y-2 text-sm">
        <div class="text-xs text-muted-foreground">
          {{ chmodTarget?.path }}
        </div>
        <FaInput v-model="chmodMode" placeholder="如 755 / 644" class="w-40" @keyup.enter="doChmod" />
        <div class="text-xs text-muted-foreground">
          递归修改请用终端 <code>chmod -R</code>；此处仅修改该项自身。
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="chmodVisible = false">
          取消
        </FaButton>
        <FaButton @click="doChmod">
          确认
        </FaButton>
      </template>
    </FaModal>

    <!-- 压缩 -->
    <FaModal v-model="compressVisible" title="压缩为 tar.gz" :destroy-on-close="true">
      <div class="space-y-2 text-sm">
        <div class="text-xs text-muted-foreground">
          {{ compressTarget?.path }}
        </div>
        <FaInput v-model="compressDest" placeholder="目标 .tar.gz 路径" class="w-full" @keyup.enter="doCompress" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="compressVisible = false">
          取消
        </FaButton>
        <FaButton :loading="compressing" @click="doCompress">
          开始压缩
        </FaButton>
      </template>
    </FaModal>

    <!-- 解压 -->
    <FaModal v-model="decompressVisible" title="解压" :destroy-on-close="true">
      <div class="space-y-2 text-sm">
        <div class="text-xs text-muted-foreground">
          {{ decompressTarget?.path }}
        </div>
        <FaInput v-model="decompressDest" placeholder="解压目标目录" class="w-full" @keyup.enter="doDecompress" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="decompressVisible = false">
          取消
        </FaButton>
        <FaButton :loading="decompressing" @click="doDecompress">
          开始解压
        </FaButton>
      </template>
    </FaModal>

    <!-- 搜索结果 -->
    <FaModal
      v-model="searchVisible"
      :title="`搜索结果：${searchKeyword}（${searchResults.length} 项）`"
      class="max-w-2xl!"
      :destroy-on-close="true"
    >
      <div class="max-h-80 overflow-auto rounded-md border">
        <button
          v-for="e in searchResults"
          :key="e.path"
          type="button"
          class="flex w-full cursor-pointer items-center gap-2 border-b px-3 py-2 text-left text-sm transition-colors last:border-b-0 hover:bg-accent/50"
          @click="e.isDir ? load(e.path) : load(e.path.slice(0, e.path.lastIndexOf('/')) || '/')"
        >
          <YdMorphIcon :name="e.isDir ? 'folder' : 'file'" :size="15" :class="e.isDir ? 'text-amber-500' : 'text-muted-foreground'" />
          <span class="truncate font-mono text-xs">{{ e.path }}</span>
        </button>
        <div v-if="!searchResults.length" class="px-3 py-8 text-center text-sm text-muted-foreground">
          无匹配结果
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="searchVisible = false">
          关闭
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
