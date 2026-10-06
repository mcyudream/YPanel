<script setup lang="ts">
import type { FileEntry } from '@/api/modules/file'
import apiFile from '@/api/modules/file'
import type { FileTreeApi, FileTreeMenuItem, FileTreeNode } from './types'
import YdFileTreeNode from './TreeNode.vue'

// 通用懒加载文件树（默认全盘 /）：目录点开懒加载，右键常用操作，文件点击 emit open-file。
const props = withDefaults(defineProps<{
  /** 节点 id（默认 local） */
  node?: string
  /** 头部标题（默认"文件"） */
  title?: string
  /** 隐藏过滤输入框（窄面板用） */
  hideFilter?: boolean
}>(), {
  node: 'local',
  title: '文件',
  hideFilter: false,
})

const emits = defineEmits<{
  openFile: [entry: FileEntry]
}>()

const appAccountStore = useAppAccountStore()

const roots = ref<FileTreeNode[]>([])
const rootLoading = ref(false)
const filter = ref('')

function toNode(entry: FileEntry): FileTreeNode {
  return { entry, children: entry.isDir ? null : [], expanded: false, loading: false }
}

function sortEntries(ents: FileEntry[]) {
  return [...ents].sort((a, b) => (a.isDir === b.isDir ? a.name.localeCompare(b.name) : a.isDir ? -1 : 1))
}

async function loadChildren(node: FileTreeNode) {
  node.loading = true
  try {
    const res = await apiFile.list(node.entry.path, props.node)
    node.children = sortEntries(res.entries).map(toNode)
  }
  catch (e: unknown) {
    node.children = []
    useFaToast().error('目录读取失败', { description: msg(e) })
  }
  finally {
    node.loading = false
  }
}

async function loadRoots() {
  rootLoading.value = true
  try {
    const res = await apiFile.list('/', props.node)
    roots.value = sortEntries(res.entries).map(toNode)
  }
  catch (e: unknown) {
    roots.value = []
    useFaToast().error('目录读取失败', { description: msg(e) })
  }
  finally {
    rootLoading.value = false
  }
}

watch(() => props.node, loadRoots, { immediate: true })

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
          { label: '刷新', icon: 'i-lucide:refresh-cw', handle: () => { node.expanded = true; loadChildren(node) } },
          { label: '新建目录', icon: 'i-lucide:folder-plus', handle: () => promptMkdir(entry.path) },
        ]
      : [
          { label: '打开', icon: 'i-lucide:external-link', handle: () => treeApi.openFile(node) },
          { label: '下载', icon: 'i-lucide:download', handle: () => download(entry) },
        ]),
    { label: '重命名', icon: 'i-lucide:text-cursor-input', handle: () => promptRename(entry) },
    { label: '删除', icon: 'i-lucide:trash', variant: 'destructive' as const, handle: () => confirmDelete(entry) },
  ]]
}

// ---- 通用输入弹窗（新建目录/重命名共用） ----
const prompt = reactive({
  visible: false,
  title: '',
  label: '',
  value: '',
  okText: '确定',
  onOk: null as null | (() => Promise<void> | void),
})

function openPrompt(title: string, label: string, value: string, onOk: () => Promise<void> | void, okText = '确定') {
  prompt.title = title
  prompt.label = label
  prompt.value = value
  prompt.onOk = onOk
  prompt.okText = okText
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
  openPrompt('新建目录', `位置：${dir}`, '', async () => {
    try {
      await apiFile.mkdir(join(dir, prompt.value.trim()), props.node)
      await refreshDir(dir)
    }
    catch (e: unknown) {
      useFaToast().error('创建失败', { description: msg(e) })
    }
  }, '创建')
}

function promptRename(entry: FileEntry) {
  openPrompt('重命名', entry.path, entry.name, async () => {
    const to = join(parentOf(entry.path), prompt.value.trim())
    try {
      await apiFile.rename(entry.path, to, props.node)
      await refreshDir(parentOf(entry.path))
    }
    catch (e: unknown) {
      useFaToast().error('重命名失败', { description: msg(e) })
    }
  }, '重命名')
}

function confirmDelete(entry: FileEntry) {
  useFaModal().confirm({
    title: '删除确认',
    content: `确认删除 ${entry.path}？${entry.isDir ? '目录将递归删除，' : ''}不可恢复。`,
    onConfirm: async () => {
      try {
        await apiFile.delete([entry.path], props.node)
        await refreshDir(parentOf(entry.path))
      }
      catch (e: unknown) {
        useFaToast().error('删除失败', { description: msg(e) })
      }
    },
  })
}

function download(entry: FileEntry) {
  window.open(apiFile.downloadURL(entry.path, appAccountStore.token, props.node))
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
      <span class="truncate text-xs font-medium text-muted-foreground" :title="props.node">{{ title }}</span>
      <div class="ml-auto flex items-center gap-0.5">
        <FaButton variant="ghost" size="icon-sm" class="size-5!" title="刷新" @click="loadRoots">
          <FaIcon name="i-lucide:refresh-cw" class="text-xs" />
        </FaButton>
      </div>
    </div>

    <!-- 过滤 -->
    <div v-if="!props.hideFilter" class="shrink-0 p-2 pb-1">
      <FaInput v-model="filter" placeholder="过滤已加载文件…" class="h-7! text-xs" />
    </div>

    <!-- 树 -->
    <div class="min-h-0 flex-1 overflow-auto px-1 pb-2">
      <div v-if="rootLoading" class="px-2 py-4 text-center text-xs text-muted-foreground">
        加载中…
      </div>
      <template v-else>
        <YdFileTreeNode
          v-for="n in roots"
          :key="n.entry.path"
          :node="n"
          :depth="0"
          :filter="filter"
        />
        <div v-if="!roots.length" class="px-2 py-4 text-center text-xs text-muted-foreground">
          目录为空
        </div>
      </template>
    </div>

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
          取消
        </FaButton>
        <FaButton @click="submitPrompt">
          {{ prompt.okText }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
