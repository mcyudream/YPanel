<script setup lang="ts">
import type { FileEntry } from '@/api/modules/file'
import apiFile from '@/api/modules/file'
import { useFaModal } from '@fantastic-admin/components'

defineOptions({
  name: 'FileManagementIndex',
})

const appAccountStore = useAppAccountStore()

const cwd = ref('/')
const entries = ref<FileEntry[]>([])
const loading = ref(false)
const selected = ref<Set<string>>(new Set())
const errorMsg = ref('')

// ---- 格式化 ----
function fmtSize(n: number) {
  if (n < 1024) {
    return `${n} B`
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(v >= 100 || i === 0 ? 0 : 1)} ${units[i]}`
}

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

// ---- 编辑器 ----
const editorVisible = ref(false)
const editorPath = ref('')
const editorContent = ref('')
const editorTruncated = ref(false)
const editorSaving = ref(false)

async function openEditor(entry: FileEntry) {
  try {
    const res = await apiFile.read(entry.path)
    editorPath.value = res.path
    editorContent.value = res.content
    editorTruncated.value = res.truncated
    editorVisible.value = true
  }
  catch (e: any) {
    useFaToast().error('读取失败', { description: e?.message })
  }
}

async function saveEditor() {
  editorSaving.value = true
  try {
    await apiFile.write(editorPath.value, editorContent.value)
    useFaToast().success('已保存')
    editorVisible.value = false
    load()
  }
  catch (e: any) {
    useFaToast().error('保存失败', { description: e?.message })
  }
  finally {
    editorSaving.value = false
  }
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
              <th class="w-28 px-3 py-2 text-right">操作</th>
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
                {{ e.isDir ? '—' : fmtSize(e.size) }}
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

    <!-- 文本编辑器 -->
    <FaModal
      v-model="editorVisible"
      :title="`编辑：${editorPath}`"
      class="max-w-4xl!"
      :destroy-on-close="true"
    >
      <div v-if="editorTruncated" class="mb-2 rounded-md border border-amber-300 bg-amber-50 p-2 text-xs text-amber-700 dark:bg-amber-950/30">
        文件较大，仅显示前 1 MiB；保存将覆盖整个文件，请谨慎操作。
      </div>
      <textarea
        v-model="editorContent"
        class="h-96 w-full resize-y rounded-md border border-input bg-background p-3 font-mono text-[13px] leading-relaxed outline-none focus:ring-1 focus:ring-primary"
        spellcheck="false"
      />
      <template #footer>
        <FaButton variant="outline" @click="editorVisible = false">
          取消
        </FaButton>
        <FaButton :loading="editorSaving" @click="saveEditor">
          保存
        </FaButton>
      </template>
    </FaModal>

    <!-- 隐藏上传控件 -->
    <input ref="uploadInput" type="file" multiple class="hidden" @change="onUploadChange">
  </div>
</template>
