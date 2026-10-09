<script setup lang="ts">
import type { FileEntry } from '@/api/modules/file'
import apiFile from '@/api/modules/file'
import apiCFile from '@/api/modules/cfile'
import YdChmodDialog from '@/components/YdChmodDialog/index.vue'
import { i18n } from '@/locales'
import type { FileTreeApi, FileTreeMenuItem, FileTreeNode } from './types'
import YdFileTreeNode from './TreeNode.vue'

// 通用懒加载文件树（默认宿主全盘 /）：目录点开懒加载，右键常用操作，文件点击 emit open-file。
// containerId 存在时切换为容器内文件系统（docker tar 归档 API）。
const props = withDefaults(defineProps<{
  /** 宿主模式的节点 id（默认 local） */
  node?: string
  /** 容器模式：容器 ID/名称（存在即容器模式，node 失效） */
  containerId?: string
  /** 头部标题（宿主默认"文件"，容器默认"容器文件"） */
  title?: string
  /** 隐藏过滤输入框（窄面板用） */
  hideFilter?: boolean
  /** 跟随的当前目录（非空时逐级展开定位并高亮；随终端 cd 更新） */
  cwd?: string
  /** 根目录路径（默认 /；宿主模式可传容器可写层等宿主路径作为树根） */
  initialPath?: string
}>(), {
  node: 'local',
  containerId: '',
  title: '',
  hideFilter: false,
  cwd: '',
  initialPath: '',
})

const emits = defineEmits<{
  openFile: [entry: FileEntry]
}>()

const appAccountStore = useAppAccountStore()

const isContainer = computed(() => !!props.containerId)
const displayTitle = computed(() => props.title || i18n.global.t(isContainer.value ? 'components.ydFileTree.containerFiles' : 'components.ydFileTree.files'))

const roots = ref<FileTreeNode[]>([])
const rootLoading = ref(false)
const filter = ref('')

// ---- 来源分派（宿主 / 容器） ----
function listDir(dir: string) {
  return isContainer.value ? apiCFile.list(props.containerId, dir) : apiFile.list(dir, props.node)
}
function doMkdirAt(dir: string, name: string) {
  const p = join(dir, name)
  return isContainer.value ? apiCFile.mkdir(props.containerId, p) : apiFile.mkdir(p, props.node)
}
function doRename(entry: FileEntry, name: string) {
  const to = join(parentOf(entry.path), name)
  return isContainer.value ? apiCFile.rename(props.containerId, entry.path, to) : apiFile.rename(entry.path, to, props.node)
}
function doDelete(entry: FileEntry) {
  return isContainer.value ? apiCFile.delete(props.containerId, [entry.path]) : apiFile.delete([entry.path], props.node)
}
function downloadURL(entry: FileEntry) {
  return isContainer.value
    ? apiCFile.downloadURL(props.containerId, entry.path, appAccountStore.token)
    : apiFile.downloadURL(entry.path, appAccountStore.token, props.node)
}

function toNode(entry: FileEntry): FileTreeNode {
  return { entry, children: entry.isDir ? null : [], expanded: false, loading: false }
}

function sortEntries(ents: FileEntry[]) {
  return [...ents].sort((a, b) => (a.isDir === b.isDir ? a.name.localeCompare(b.name) : a.isDir ? -1 : 1))
}

async function loadChildren(node: FileTreeNode) {
  node.loading = true
  try {
    const res = await listDir(node.entry.path)
    node.children = sortEntries(res.entries).map(toNode)
  }
  catch (e: unknown) {
    node.children = []
    useFaToast().error(i18n.global.t('components.ydFileTree.loadFailed'), { description: msg(e) })
  }
  finally {
    node.loading = false
  }
}

const rootPath = computed(() => props.initialPath || '/')

async function loadRoots() {
  rootLoading.value = true
  try {
    const res = await listDir(rootPath.value)
    roots.value = sortEntries(res.entries).map(toNode)
  }
  catch (e: unknown) {
    roots.value = []
    useFaToast().error(i18n.global.t('components.ydFileTree.loadFailed'), { description: msg(e) })
  }
  finally {
    rootLoading.value = false
  }
}

watch(() => [props.node, props.containerId, props.initialPath], loadRoots, { immediate: true })

// 在树中定位目录节点（'/' 表示根）
function findDirNode(dir: string): FileTreeNode | null {
  if (dir === '/') {
    return null
  }
  const stack = [...roots.value]
  while (stack.length) {
    const n = stack.pop()!
    if (n.entry.path === dir) {
      return n
    }
    if (n.children) {
      stack.push(...n.children)
    }
  }
  return null
}

async function refreshDir(dir: string) {
  const n = findDirNode(dir)
  if (n) {
    await loadChildren(n)
  }
  else {
    await loadRoots()
  }
}

// ---- 目录跟随（终端 cd → 树定位展开高亮） ----
const highlightPath = ref('')
let locateSeq = 0

async function locate(path: string) {
  const seq = ++locateSeq
  if (!path || !path.startsWith('/')) {
    highlightPath.value = ''
    return
  }
  // 根列表尚在加载则等它完成，否则逐级找不到节点
  while (rootLoading.value) {
    await new Promise(r => setTimeout(r, 100))
    if (seq !== locateSeq) {
      return // 已被更新的定位请求取代
    }
  }
  if (seq !== locateSeq) {
    return
  }
  const parts = path.split('/').filter(Boolean)
  let acc = ''
  for (const part of parts) {
    acc += `/${part}`
    const n = findDirNode(acc)
    if (!n || !n.entry.isDir) {
      return // 树中不存在（非目录/未挂载等），停在上一级
    }
    if (!n.children) {
      await loadChildren(n)
      if (seq !== locateSeq) {
        return
      }
    }
    n.expanded = true
  }
  highlightPath.value = acc
}

watch(() => props.cwd, p => locate(p))

const treeApi: FileTreeApi = {
  toggle(node) {
    if (!node.expanded && !node.children) {
      loadChildren(node)
    }
    node.expanded = !node.expanded
  },
  openFile(node) {
    emits('openFile', node.entry)
  },
  menuItems: node => menuItems(node),
}
provide('ydFileTreeApi', treeApi)

function menuItems(node: FileTreeNode): FileTreeMenuItem[][] {
  const { entry } = node
  return [[
    ...(entry.isDir
      ? [
          { label: i18n.global.t('common.refresh'), icon: 'i-lucide:refresh-cw', handle: () => { node.expanded = true; loadChildren(node) } },
          { label: i18n.global.t('components.ydFileTree.newDir'), icon: 'i-lucide:folder-plus', handle: () => promptMkdir(entry.path) },
          { label: i18n.global.t('components.ydFileTree.uploadFile'), icon: 'i-lucide:upload', handle: () => pickUpload(entry.path) },
        ]
      : [
          { label: i18n.global.t('components.ydFileTree.open'), icon: 'i-lucide:external-link', handle: () => treeApi.openFile(node) },
          { label: i18n.global.t('common.download'), icon: 'i-lucide:download', handle: () => download(entry) },
        ]),
    { label: i18n.global.t('components.ydFileTree.permissions'), icon: 'i-lucide:lock', handle: () => openChmodDialog(entry) },
    { label: i18n.global.t('components.ydFileTree.rename'), icon: 'i-lucide:text-cursor-input', handle: () => promptRename(entry) },
    { label: i18n.global.t('common.delete'), icon: 'i-lucide:trash', variant: 'destructive' as const, handle: () => confirmDelete(entry) },
  ]]
}

// ---- 上传（目录右键，单文件直传） ----
const uploadInputRef = useTemplateRef<HTMLInputElement>('uploadInput')
let uploadTargetDir = '/'

function pickUpload(dir: string) {
  uploadTargetDir = dir
  uploadInputRef.value?.click()
}

async function onUploadChange(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) {
    return
  }
  try {
    if (isContainer.value) {
      await apiCFile.upload(props.containerId, uploadTargetDir, file)
    }
    else {
      await apiFile.upload(uploadTargetDir, file, undefined, props.node)
    }
    useFaToast().success(i18n.global.t('components.ydFileTree.uploadDone'))
    await refreshDir(uploadTargetDir)
  }
  catch (e: unknown) {
    useFaToast().error(i18n.global.t('components.ydFileTree.uploadFailed'), { description: msg(e) })
  }
  finally {
    input.value = ''
  }
}

// ---- 通用输入弹窗（新建目录/重命名/权限共用） ----
const prompt = reactive({
  visible: false,
  title: '',
  label: '',
  value: '',
  okText: '',
  onOk: null as null | (() => Promise<void> | void),
})

function openPrompt(title: string, label: string, value: string, onOk: () => Promise<void> | void, okText?: string) {
  prompt.title = title
  prompt.label = label
  prompt.value = value
  prompt.onOk = onOk
  prompt.okText = okText ?? i18n.global.t('common.confirm')
  prompt.visible = true
}

async function submitPrompt() {
  if (!prompt.value.trim()) {
    return
  }
  await prompt.onOk?.()
  prompt.visible = false
}

function join(dir: string, name: string) {
  return dir.endsWith('/') ? dir + name : `${dir}/${name}`
}

function parentOf(path: string) {
  return path.slice(0, path.lastIndexOf('/')) || '/'
}

function promptMkdir(dir: string) {
  openPrompt(i18n.global.t('components.ydFileTree.newDir'), i18n.global.t('components.ydFileTree.location', { dir }), '', async () => {
    try {
      await doMkdirAt(dir, prompt.value.trim())
      await refreshDir(dir)
    }
    catch (e: unknown) {
      useFaToast().error(i18n.global.t('components.ydFileTree.createFailed'), { description: msg(e) })
    }
  }, i18n.global.t('common.create'))
}

function promptRename(entry: FileEntry) {
  openPrompt(i18n.global.t('components.ydFileTree.rename'), entry.path, entry.name, async () => {
    try {
      await doRename(entry, prompt.value.trim())
      await refreshDir(parentOf(entry.path))
    }
    catch (e: unknown) {
      useFaToast().error(i18n.global.t('components.ydFileTree.renameFailed'), { description: msg(e) })
    }
  }, i18n.global.t('components.ydFileTree.rename'))
}

// ---- 权限编辑（YdChmodDialog：矩阵 + 属主 + 递归，宿主/容器分派在组件内） ----
const chmodVisible = ref(false)
const chmodPaths = ref<string[]>([])
const chmodEntries = ref<FileEntry[]>([])
const chmodParent = ref('/')

function openChmodDialog(entry: FileEntry) {
  chmodPaths.value = [entry.path]
  chmodEntries.value = [entry]
  chmodParent.value = parentOf(entry.path)
  chmodVisible.value = true
}

async function onChmodDone() {
  await refreshDir(chmodParent.value)
}

function confirmDelete(entry: FileEntry) {
  useFaModal().confirm({
    title: i18n.global.t('components.ydFileTree.deleteConfirmTitle'),
    content: i18n.global.t('components.ydFileTree.deleteConfirmPrefix', { path: entry.path }) + (entry.isDir ? i18n.global.t('components.ydFileTree.deleteConfirmRecursive') : '') + i18n.global.t('components.ydFileTree.deleteConfirmSuffix'),
    onConfirm: async () => {
      try {
        await doDelete(entry)
        await refreshDir(parentOf(entry.path))
      }
      catch (e: unknown) {
        useFaToast().error(i18n.global.t('components.ydFileTree.deleteFailed'), { description: msg(e) })
      }
    },
  })
}

function download(entry: FileEntry) {
  window.open(downloadURL(entry))
}

function msg(e: unknown) {
  if (e && typeof e === 'object' && 'message' in e && typeof (e as { message: unknown }).message === 'string') {
    return (e as { message: string }).message
  }
  return String(e)
}

defineExpose({ reload: loadRoots })
</script>

<template>
  <div class="flex size-full min-h-0 min-w-0 flex-col overflow-hidden">
    <!-- 头部 -->
    <div class="flex h-8 shrink-0 items-center gap-1 border-b bg-muted/40 px-2 text-[13px]">
      <YdMorphIcon name="folder-tree" :size="14" class="text-muted-foreground" />
      <span class="truncate text-xs font-medium text-muted-foreground">{{ displayTitle }}</span>
      <div class="ml-auto flex items-center gap-0.5">
        <FaButton variant="ghost" size="icon-sm" class="size-5!" :title="$t('common.refresh')" @click="loadRoots">
          <FaIcon name="i-lucide:refresh-cw" class="text-xs" />
        </FaButton>
      </div>
    </div>

    <!-- 过滤 -->
    <div v-if="!props.hideFilter" class="shrink-0 p-2 pb-1">
      <FaInput v-model="filter" :placeholder="$t('components.ydFileTree.filterPlaceholder')" class="h-7! text-xs" />
    </div>

    <!-- 树 -->
    <div class="min-h-0 flex-1 overflow-auto px-1 pb-2">
      <div v-if="rootLoading" class="px-2 py-4 text-center text-xs text-muted-foreground">
        {{ $t('common.loading') }}
      </div>
      <template v-else>
        <YdFileTreeNode
          v-for="n in roots"
          :key="n.entry.path"
          :node="n"
          :depth="0"
          :filter="filter"
          :highlight="highlightPath"
        />
        <div v-if="!roots.length" class="px-2 py-4 text-center text-xs text-muted-foreground">
          {{ $t('components.ydFileTree.emptyDir') }}
        </div>
      </template>
    </div>

    <!-- 隐藏上传控件 -->
    <input ref="uploadInput" type="file" class="hidden" @change="onUploadChange">

    <!-- 权限编辑弹窗（宿主/容器双模式） -->
    <YdChmodDialog
      v-model:visible="chmodVisible"
      :paths="chmodPaths"
      :entries="chmodEntries"
      :node="props.node"
      :container-id="props.containerId"
      @done="onChmodDone"
    />

    <!-- 通用输入弹窗 -->
    <FaModal v-model="prompt.visible" :title="prompt.title" class="max-w-lg!" :destroy-on-close="true">
      <div class="space-y-2">
        <div class="truncate text-xs text-muted-foreground">
          {{ prompt.label }}
        </div>
        <FaInput v-model="prompt.value" class="w-full" @keyup.enter="submitPrompt" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="prompt.visible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton @click="submitPrompt">
          {{ prompt.okText }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
