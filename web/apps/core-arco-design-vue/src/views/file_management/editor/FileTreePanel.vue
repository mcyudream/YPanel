<script setup lang="ts">
import apiFile from '@/api/modules/file'
import apiCFile from '@/api/modules/cfile'
import type { FileEntry } from '@/api/modules/file'
import type { FileTreeApi, FileTreeMenuItem, FileTreeNode } from './types'
import { fmtBytes } from '@/utils/format'
import { Pane } from 'splitpanes'
import FileTreeRow from './FileTreeRow.vue'

// 文件树侧栏：懒加载目录树 + 右键操作 + 子树过滤 + 全局搜索。
// currentContainer 非空时切换为容器内文件系统（docker tar 归档 API；压缩/解压/搜索不可用）。
const appAccountStore = useAppAccountStore()
const store = useFileEditorStore()

const isContainer = computed(() => !!store.currentContainer)
const containerLabel = computed(() => `容器 · ${store.currentContainer.slice(0, 12)}`)

const roots = ref<FileTreeNode[]>([])
const rootLoading = ref(false)
const filter = ref('')
const searching = ref(false)
const searchKeyword = ref('')
const searchResults = ref<FileEntry[]>([])

const nodeOptions = computed(() => store.nodes.map(n => ({
  label: n.hostname ? `${n.hostname}${n.online ? '' : '（离线）'}` : n.id,
  value: n.id,
  disabled: !n.online,
})))

function toNode(entry: FileEntry): FileTreeNode {
  return { entry, children: entry.isDir ? null : [], expanded: false, loading: false }
}

function sortEntries(ents: FileEntry[]) {
  return [...ents].sort((a, b) => (a.isDir === b.isDir ? a.name.localeCompare(b.name) : a.isDir ? -1 : 1))
}

// ---- 来源分派（宿主 / 容器） ----
function listDir(dir: string) {
  return isContainer.value
    ? apiCFile.list(store.currentContainer, dir)
    : apiFile.list(dir, store.currentNode)
}
function downloadURL(entry: FileEntry) {
  return isContainer.value
    ? apiCFile.downloadURL(store.currentContainer, entry.path, appAccountStore.token)
    : apiFile.downloadURL(entry.path, appAccountStore.token, store.currentNode)
}

async function loadChildren(node: FileTreeNode) {
  node.loading = true
  try {
    const res = await listDir(node.entry.path)
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
    const res = await listDir('/')
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

watch(() => [store.currentNode, store.currentContainer], loadRoots, { immediate: true })

const treeApi: FileTreeApi = {
  toggle(node) {
    if (!node.expanded && !node.children) {
      loadChildren(node)
    }
    node.expanded = !node.expanded
  },
  openFile(node) {
    store.open(node.entry.path, store.currentNode)
  },
  menuItems: node => menuItems(node),
}
provide('fileTreeApi', treeApi)

// ---- 右键菜单 ----
function menuItems(node: FileTreeNode): FileTreeMenuItem[][] {
  const { entry } = node
  const items: FileTreeMenuItem[] = [
    ...(entry.isDir
      ? [
          { label: '新建文件', icon: 'i-lucide:file-plus', handle: () => promptNewFile(entry.path) },
          { label: '新建目录', icon: 'i-lucide:folder-plus', handle: () => promptMkdir(entry.path) },
        ]
      : [
          { label: '打开', icon: 'i-lucide:external-link', handle: () => treeApi.openFile(node) },
          { label: '下载', icon: 'i-lucide:download', handle: () => download(entry) },
        ]),
    { label: '重命名', icon: 'i-lucide:text-cursor-input', handle: () => promptRename(entry) },
    { label: '权限', icon: 'i-lucide:lock', handle: () => promptChmod(entry) },
    // 容器模式无压缩/解压能力，隐藏
    ...(!isContainer.value
      ? [
          { label: '压缩为 tar.gz', icon: 'i-lucide:package', handle: () => promptCompress(entry) },
          ...(/\.(tar\.gz|tgz|tar|zip)$/i.test(entry.name)
            ? [{ label: '解压到此处', icon: 'i-lucide:package-open', handle: () => decompressHere(entry) }]
            : []),
        ]
      : []),
    { label: '删除', icon: 'i-lucide:trash', variant: 'destructive' as const, handle: () => confirmDelete(entry) },
  ]
  return [items]
}

// ---- 通用输入弹窗 ----
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

// 在树中找到目录节点（或根）
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
  else if (dir === '/') {
    await loadRoots()
  }
}

// ---- 各操作 ----
function promptNewFile(dir: string) {
  openPrompt('新建文件', `位置：${dir}`, '', async () => {
    const path = join(dir, prompt.value.trim())
    try {
      if (isContainer.value) {
        await apiCFile.write(store.currentContainer, path, '')
      }
      else {
        await apiFile.write(path, '', store.currentNode)
      }
      await refreshDir(dir)
      store.open(path, store.currentNode)
    }
    catch (e: unknown) {
      useFaToast().error('创建失败', { description: msg(e) })
    }
  }, '创建')
}

function promptMkdir(dir: string) {
  openPrompt('新建目录', `位置：${dir}`, '', async () => {
    try {
      const path = join(dir, prompt.value.trim())
      if (isContainer.value) {
        await apiCFile.mkdir(store.currentContainer, path)
      }
      else {
        await apiFile.mkdir(path, store.currentNode)
      }
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
      if (isContainer.value) {
        await apiCFile.rename(store.currentContainer, entry.path, to)
      }
      else {
        await apiFile.rename(entry.path, to, store.currentNode)
      }
      await refreshDir(parentOf(entry.path))
      // 已打开的 tab：未保存直接换路径重开；有修改则提示
      const id = isContainer.value
        ? `c:${store.currentContainer}::${store.currentNode}::${entry.path}`
        : `${store.currentNode}::${entry.path}`
      if (store.tabs[id]) {
        if (!store.tabs[id].dirty) {
          for (const g of store.groups.filter(g => g.tabIds.includes(id))) {
            store.closeTab(id, g.id, true)
          }
          store.open(to, store.currentNode)
        }
        else {
          useFaToast().info('文件已重命名，当前标签内容未保存，保存前请注意路径变化')
        }
      }
    }
    catch (e: unknown) {
      useFaToast().error('重命名失败', { description: msg(e) })
    }
  }, '重命名')
}

function promptChmod(entry: FileEntry) {
  openPrompt('修改权限', `${entry.path}（3 位八进制，如 755；仅该项自身）`, entry.modeOct.slice(-3) || '644', async () => {
    if (!/^[0-7]{3}$/.test(prompt.value.trim())) {
      useFaToast().error('权限格式错误（3 位八进制，如 755）')
      return
    }
    try {
      if (isContainer.value) {
        await apiCFile.chmod(store.currentContainer, entry.path, prompt.value.trim())
      }
      else {
        await apiFile.chmod(entry.path, prompt.value.trim(), store.currentNode)
      }
      await refreshDir(parentOf(entry.path))
    }
    catch (e: unknown) {
      useFaToast().error('修改失败', { description: msg(e) })
    }
  })
}

function promptCompress(entry: FileEntry) {
  openPrompt('压缩为 tar.gz', entry.path, `${entry.path}.tar.gz`, async () => {
    try {
      await apiFile.compress(entry.path, prompt.value.trim(), store.currentNode)
      useFaToast().success('压缩完成')
      await refreshDir(parentOf(prompt.value.trim()))
    }
    catch (e: unknown) {
      useFaToast().error('压缩失败', { description: msg(e) })
    }
  }, '开始压缩')
}

function decompressHere(entry: FileEntry) {
  const dest = parentOf(entry.path)
  apiFile.decompress(entry.path, dest, store.currentNode)
    .then(() => {
      useFaToast().success('解压完成')
      return refreshDir(dest)
    })
    .catch((e: unknown) => useFaToast().error('解压失败', { description: msg(e) }))
}

function confirmDelete(entry: FileEntry) {
  useFaModal().confirm({
    title: '删除确认',
    content: `确认删除 ${entry.path}？${entry.isDir ? '目录将递归删除，' : ''}不可恢复。`,
    onConfirm: async () => {
      try {
        if (isContainer.value) {
          await apiCFile.delete(store.currentContainer, [entry.path])
        }
        else {
          await apiFile.delete([entry.path], store.currentNode)
        }
        await refreshDir(parentOf(entry.path))
        // 关掉已打开且未保存的 tab（容器 tab id 为三段形态，此处 host 形态查不到自然跳过）
        const id = `${store.currentNode}::${entry.path}`
        for (const g of store.groups.filter(g => g.tabIds.includes(id))) {
          if (!store.tabs[id]?.dirty) {
            store.closeTab(id, g.id, true)
          }
        }
      }
      catch (e: unknown) {
        useFaToast().error('删除失败', { description: msg(e) })
      }
    },
  })
}

function download(entry: FileEntry) {
  window.open(downloadURL(entry))
}

// ---- 全局搜索 ----
async function doSearch() {
  const kw = searchKeyword.value.trim()
  if (!kw) {
    return
  }
  searching.value = true
  try {
    searchResults.value = await apiFile.search('/', kw, store.currentNode)
  }
  catch (e: unknown) {
    useFaToast().error('搜索失败', { description: msg(e) })
  }
  finally {
    searching.value = false
  }
}

function msg(e: unknown) {
  if (e && typeof e === 'object' && 'message' in e && typeof (e as { message: unknown }).message === 'string') {
    return (e as { message: string }).message
  }
  return String(e)
}
</script>

<template>
  <Pane
    :size="store.layout.sidebarWidth"
    min-size="8"
    max-size="45"
    class="flex min-h-0 min-w-0 flex-col overflow-hidden border-r bg-muted/20"
  >
    <!-- 头部：节点选择（宿主）/ 容器标识 + 刷新 -->
    <div class="flex h-8 shrink-0 items-center gap-1 border-b bg-muted/40 px-2 text-[13px]">
      <YdMorphIcon name="folder-tree" :size="14" class="shrink-0 text-muted-foreground" />
      <template v-if="isContainer">
        <span class="min-w-0 flex-1 truncate text-xs text-muted-foreground" :title="store.currentContainer">
          {{ containerLabel }}
        </span>
      </template>
      <FaSelect
        v-else
        v-model="store.currentNode"
        :options="nodeOptions"
        class="h-6! min-w-0 flex-1! text-xs!"
        :disabled="!nodeOptions.length"
      />
      <div class="ml-auto flex shrink-0 items-center gap-0.5">
        <FaButton variant="ghost" size="icon-sm" class="size-5!" title="刷新" @click="loadRoots">
          <FaIcon name="i-lucide:refresh-cw" class="text-xs" />
        </FaButton>
      </div>
    </div>

    <!-- 过滤 -->
    <div class="shrink-0 space-y-1 p-2">
      <FaInput v-model="filter" placeholder="过滤已加载文件…" class="h-7! text-xs" />
      <div v-if="!isContainer" class="flex gap-1">
        <FaInput
          v-model="searchKeyword"
          placeholder="全盘搜索文件名，回车执行"
          class="h-7! flex-1 text-xs"
          @keyup.enter="doSearch"
        />
        <FaButton variant="outline" size="icon-sm" class="size-7!" :loading="searching" title="搜索" @click="doSearch">
          <FaIcon name="i-lucide:search" class="text-xs" />
        </FaButton>
      </div>
    </div>

    <!-- 树 -->
    <div class="min-h-0 flex-1 overflow-auto px-1 pb-2">
      <div v-if="rootLoading" class="px-2 py-4 text-center text-xs text-muted-foreground">
        加载中…
      </div>
      <template v-else>
        <FileTreeRow
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

    <!-- 搜索结果 -->
    <div v-if="searchResults.length" class="max-h-48 shrink-0 overflow-auto border-t bg-muted/20">
      <div class="flex items-center justify-between px-2 py-1 text-xs text-muted-foreground">
        <span>搜索结果（{{ searchResults.length }}）</span>
        <button type="button" class="cursor-pointer hover:text-foreground" @click="searchResults = []">
          清除
        </button>
      </div>
      <button
        v-for="e in searchResults"
        :key="e.path"
        type="button"
        class="flex w-full cursor-pointer items-center gap-1.5 px-2 py-1 text-left text-xs transition-colors hover:bg-accent/50"
        :title="`${e.path}（${fmtBytes(e.size)}）`"
        @click="!e.isDir && store.open(e.path, store.currentNode)"
      >
        <YdMorphIcon :name="e.isDir ? 'folder' : 'file'" :size="12" :class="e.isDir ? 'text-amber-500' : 'text-muted-foreground'" />
        <span class="truncate font-mono">{{ e.path }}</span>
      </button>
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
  </Pane>
</template>
