<script setup lang="ts">
import type { FileEntry } from '@/api/modules/file'
import apiFile, { fileExtApi } from '@/api/modules/file'
import type { TrashItem, FileFavorite, FileShareRow } from '@/api/modules/file'
import apiNode from '@/api/modules/node'
import { fmtBytes } from '@/utils/format'
import { useFaModal } from '@fantastic-admin/components'
import YdChmodDialog from '@/components/YdChmodDialog/index.vue'
import YdDirPicker from '@/components/YdDirPicker/index.vue'
import FileEditorWorkspace from './editor/Workspace.vue'
import { useYwEmbed } from '@/views/desktop/embed'
import { i18n } from '@/locales'

defineOptions({
  name: 'FileManagementIndex',
})

// 桌面工作台承载时经 props 传入初始目录/节点（launchOptions），经典模式走路由参数
const props = defineProps<{
  /** 初始目录（webos 窗口承载时由 launchOptions 注入） */
  initialDir?: string
  /** 初始节点（webos 窗口承载时由 launchOptions 注入） */
  initialNode?: string
}>()

const appAccountStore = useAppAccountStore()
const fileEditorStore = useFileEditorStore()

const route = useRoute()
const router = useRouter()
/** 跨窗口拖拽的实例标识：drop 回自己时忽略；移动完成通知源窗口刷新 */
const instanceId = `fm-${Math.random().toString(36).slice(2, 8)}`
const embed = useYwEmbed()

const cwd = ref(props.initialDir ?? '/')
const entries = ref<FileEntry[]>([])
const loading = ref(false)
const selected = ref<Set<string>>(new Set())
const errorMsg = ref('')

// ---- 节点选择（?node= 进入指定节点，默认本机） ----
const nodeId = ref<string>(props.initialNode ?? ((route.query.node as string) || 'local'))
const nodes = ref<{ id: string, name: string, online: boolean }[]>([])

async function loadNodes() {
  try {
    const list = await apiNode.list()
    nodes.value = list.map(n => ({ id: n.id, name: n.online ? n.name : `${n.name}${i18n.global.t('files.common.offline')}`, online: n.online }))
  }
  catch {}
}

const currentNodeName = computed(() => nodes.value.find(n => n.id === nodeId.value)?.name || i18n.global.t('files.common.local'))

/** 节点下拉（FaDropdown：原生 select 展开层不可主题化，深色窗口里突兀） */
const nodeMenuItems = computed(() => [nodes.value.map(n => ({
  label: n.id === 'local' ? i18n.global.t('files.common.localNode', { name: n.name }) : n.name,
  disabled: !n.online,
  handle: () => pickNode(n.id),
}))])

function pickNode(id: string) {
  nodeId.value = id
  router.replace({ query: { ...route.query, node: id === 'local' ? undefined : id } })
  load('/')
}

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
    const res = await apiFile.list(path, nodeId.value)
    cwd.value = res.path
    entries.value = res.entries
    selected.value = new Set()
  }
  catch (e: any) {
    errorMsg.value = e?.message || i18n.global.t('files.common.readFailed')
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
  await apiFile.mkdir(target, nodeId.value)
  mkdirVisible.value = false
  mkdirName.value = ''
  useFaToast().success(i18n.global.t('files.list.dirCreated'))
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
  await apiFile.rename(renameTarget.value.path, joinPath(cwd.value, renameName.value.trim()), nodeId.value)
  renameVisible.value = false
  useFaToast().success(i18n.global.t('files.list.renamed'))
  load()
}

async function doDelete() {
  if (!selected.value.size) {
    return
  }
  const paths = [...selected.value]
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('files.common.deleteConfirmTitle'),
    content: i18n.global.t('files.list.deleteSelectedContent', { n: paths.length }),
    onConfirm: async () => {
      if (nodeId.value === 'local') {
        await fileExtApi.trash(paths)
      }
      else {
        await apiFile.delete(paths, nodeId.value) // 远程节点暂无回收站通道
      }
      useFaToast().success(i18n.global.t('files.list.movedToTrash'))
      load()
    },
  })
}

// ---- M38：回收站 / 收藏 / 分享 / 远程下载 / 预览 ----
const trashVisible = ref(false)
const trashItems = ref<TrashItem[]>([])
const trashLoading = ref(false)

async function openTrash() {
  trashVisible.value = true
  await loadTrash()
}

async function loadTrash() {
  trashLoading.value = true
  try {
    trashItems.value = await fileExtApi.trashList()
  }
  finally {
    trashLoading.value = false
  }
}

async function trashRestore(n: string) {
  await fileExtApi.trashRestore([n])
  useFaToast().success(i18n.global.t('files.list.restored'))
  await loadTrash()
}

function trashPurge(n: string) {
  useFaModal().confirm({
    title: i18n.global.t('files.dialogs.purgeTitle'),
    content: i18n.global.t('files.dialogs.purgeContent', { name: n }),
    onConfirm: async () => {
      await fileExtApi.trashPurge([n])
      await loadTrash()
    },
  })
}

function trashClear() {
  useFaModal().confirm({
    title: i18n.global.t('files.dialogs.clearTitle'),
    content: i18n.global.t('files.dialogs.clearContent'),
    onConfirm: async () => {
      const out = await fileExtApi.trashClear()
      useFaToast().success(i18n.global.t('files.dialogs.cleared', { n: out.count }))
      await loadTrash()
    },
  })
}

const favorites = ref<FileFavorite[]>([])

async function loadFavorites() {
  try {
    favorites.value = await fileExtApi.favorites()
  }
  catch {}
}

function favOf(p: string) {
  return favorites.value.find(f => f.path === p)
}

async function toggleFav(e: FileEntry) {
  const f = favOf(e.path)
  if (f) {
    await fileExtApi.removeFavorite(f.id)
    useFaToast().success(i18n.global.t('files.list.unfavDone'))
  }
  else {
    await fileExtApi.addFavorite(e.path)
    useFaToast().success(i18n.global.t('files.list.favDone'))
  }
  await loadFavorites()
}

async function removeFav(f: FileFavorite) {
  await fileExtApi.removeFavorite(f.id)
  await loadFavorites()
}

const shareVisible = ref(false)
const shares = ref<FileShareRow[]>([])
const shareForm = ref({ path: '', days: 7 })
const lastShareLink = ref('')

async function openShares() {
  shareVisible.value = true
  await loadShares()
}

async function loadShares() {
  shares.value = await fileExtApi.shares()
}

function quickShare(e: FileEntry) {
  shareForm.value = { path: e.path, days: 7 }
  lastShareLink.value = ''
  openShares()
}

async function createShare() {
  if (!shareForm.value.path) {
    useFaToast().warning(i18n.global.t('files.list.shareNeedPath'))
    return
  }
  try {
    const out = await fileExtApi.createShare(shareForm.value.path, shareForm.value.days)
    lastShareLink.value = `${location.origin}/api/v1/s/${out.token}`
    await loadShares()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('files.list.shareCreateFailed'), { description: e?.message })
  }
}

async function revokeShare(id: number) {
  await fileExtApi.revokeShare(id)
  useFaToast().success(i18n.global.t('files.list.shareRevoked'))
  await loadShares()
}

function copyShareLink(link: string) {
  void navigator.clipboard.writeText(link)
  useFaToast().success(i18n.global.t('files.list.linkCopied'))
}

const remoteVisible = ref(false)
const remoteForm = ref({ url: '', destDir: '' })
const remoteBusy = ref(false)

function openRemote() {
  remoteForm.value = { url: '', destDir: cwd.value }
  remoteVisible.value = true
}

async function doRemoteDownload() {
  if (!remoteForm.value.url || !remoteForm.value.destDir) {
    useFaToast().warning(i18n.global.t('files.list.remoteNeedUrl'))
    return
  }
  remoteBusy.value = true
  useFaToast().info(i18n.global.t('files.list.downloading'))
  try {
    const out = await fileExtApi.remoteDownload(remoteForm.value.url, remoteForm.value.destDir)
    useFaToast().success(i18n.global.t('files.list.downloadDone', { path: `${out.dir}/${out.file}` }))
    remoteVisible.value = false
    load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('files.list.downloadFailed'), { description: e?.message })
  }
  finally {
    remoteBusy.value = false
  }
}

const moreMenuItems = computed(() => [[
  { label: i18n.global.t('files.trash'), icon: 'i-lucide:trash-2', handle: () => openTrash() },
  { label: i18n.global.t('files.remoteDownload'), icon: 'i-lucide:cloud-download', handle: () => openRemote() },
  { label: i18n.global.t('files.share'), icon: 'i-lucide:share-2', handle: () => { shareForm.value = { path: cwd.value, days: 7 }; lastShareLink.value = ''; openShares() } },
]])

// 预览
const previewVisible = ref(false)
const previewEntry = ref<FileEntry | null>(null)

function previewKind(name: string): '' | 'image' | 'video' | 'audio' | 'pdf' {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(ext)) return 'image'
  if (['mp4', 'webm', 'mkv', 'mov'].includes(ext)) return 'video'
  if (['mp3', 'wav', 'ogg', 'flac', 'm4a'].includes(ext)) return 'audio'
  if (ext === 'pdf') return 'pdf'
  return ''
}

function openPreview(e: FileEntry) {
  previewEntry.value = e
  previewVisible.value = true
}

// ---- 编辑器（VS Code 式工作台弹窗） ----
// 显式传当前节点，防止工作台残留的节点选择影响打开目标
async function openEditor(entry: FileEntry) {
  await fileEditorStore.openWorkspace(entry.path, nodeId.value)
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
      }, nodeId.value)
    }
    useFaToast().success(i18n.global.t('files.list.uploadDone'))
    load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('files.list.uploadFailed'), { description: e?.message })
  }
  finally {
    uploading.value = false
    uploadPercent.value = 0
    input.value = ''
  }
}

function downloadURL(entry: FileEntry) {
  return apiFile.downloadURL(entry.path, appAccountStore.token, nodeId.value)
}

function openDownload(entry: FileEntry) {
  window.open(downloadURL(entry))
}

// ---- 批量操作（移动/复制/压缩/权限/删除） ----
const selectedEntries = computed(() => entries.value.filter(e => selected.value.has(e.path)))

const batchMenuItems = computed(() => [[
  { label: i18n.global.t('files.list.moveTo'), icon: 'i-lucide:folder-input', handle: () => openBatchTransfer('move') },
  { label: i18n.global.t('files.list.copyTo'), icon: 'i-lucide:folder-output', handle: () => openBatchTransfer('copy') },
  { label: i18n.global.t('files.list.copyToNode'), icon: 'i-lucide:copy-plus', handle: () => openCopyAcross() },
  { label: i18n.global.t('files.list.compressMenu'), icon: 'i-lucide:package', handle: () => openCompress([...selected.value]) },
  { label: i18n.global.t('files.list.permMenu'), icon: 'i-lucide:lock', handle: () => openChmod([...selected.value], selectedEntries.value) },
  { label: i18n.global.t('common.delete'), icon: 'i-lucide:trash', variant: 'destructive' as const, handle: () => doDelete() },
]])

// ---- M56 复制到节点（core 中转流式，任务化） ----
const crossVisible = ref(false)
const crossForm = ref({ dstNode: '', dstDir: '', overwrite: false })
const crossing = ref(false)
const allNodes = ref<{ id: string, name: string }[]>([])

function openCopyAcross() {
  if (!selected.value.size) {
    useFaToast().warning(i18n.global.t('files.list.selectFirst'))
    return
  }
  crossForm.value = { dstNode: '', dstDir: cwd.value, overwrite: false }
  allNodes.value = (nodes.value as any[]).filter((x: any) => x.id !== 'local' && x.id !== nodeId.value && x.online)
  crossVisible.value = true
}

async function doCopyAcross() {
  if (!crossForm.value.dstNode) {
    useFaToast().warning(i18n.global.t('files.list.pickTargetNode'))
    return
  }
  crossing.value = true
  try {
    const items = selectedEntries.value.map(e => ({ path: e.path, name: e.name, isDir: e.isDir }))
    const res = await apiFile.copyAcross({
      srcNode: nodeId.value, srcPath: items[0]?.path || cwd.value,
      dstNode: crossForm.value.dstNode, dstDir: crossForm.value.dstDir,
      overwrite: crossForm.value.overwrite, items,
    })
    const taskId = (res.data as any)?.taskId
    useFaToast().info(i18n.global.t('files.list.crossTaskCreated'))
    // 轮询任务终态
    if (taskId) {
      for (let i = 0; i < 300; i++) {
        await new Promise(r => setTimeout(r, 2000))
        try {
          const t = await apiFile.taskGet(taskId)
          if (t.status === 'success') {
            useFaToast().success(i18n.global.t('files.list.crossDone'))
            break
          }
          if (t.status === 'failed') {
            useFaToast().error(i18n.global.t('files.list.crossFailed'), { description: t.error || '' })
            break
          }
        }
        catch {}
      }
    }
    crossVisible.value = false
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('common.opFailed'), { description: e?.message })
  }
  finally {
    crossing.value = false
  }
}

// ---- 权限编辑（单项/批量共用 YdChmodDialog） ----
const chmodDialogVisible = ref(false)
const chmodPaths = ref<string[]>([])
const chmodEntries = ref<FileEntry[]>([])

function openChmod(paths: string[], ents: FileEntry[]) {
  chmodPaths.value = paths
  chmodEntries.value = ents
  chmodDialogVisible.value = true
}

const dirPickerVisible = ref(false)
const batchTransfer = reactive({
  visible: false,
  mode: 'move' as 'move' | 'copy',
  target: '',
  overwrite: false,
  running: false,
})

function openBatchTransfer(mode: 'move' | 'copy') {
  batchTransfer.mode = mode
  dirPickerVisible.value = true
}

function onBatchDirPicked(path: string) {
  batchTransfer.target = path
  batchTransfer.overwrite = false
  batchTransfer.visible = true
}

async function runBatchTransfer() {
  const move = batchTransfer.mode === 'move'
  const targetDir = batchTransfer.target
  const list = selectedEntries.value
  // 目标不能是选中项自身或其内部（防自环）
  for (const e of list) {
    if (targetDir === e.path || targetDir.startsWith(`${e.path}/`)) {
      useFaToast().error(i18n.global.t(move ? 'files.list.transferFailedMove' : 'files.list.transferFailedCopy'), { description: i18n.global.t('files.list.targetInside', { name: e.name }) })
      return
    }
  }
  batchTransfer.running = true
  let done = 0
  let skipped = 0
  try {
    // 预读目标目录同名集合（失败不阻塞，逐项再验）
    let names = new Set<string>()
    try {
      names = new Set((await apiFile.list(targetDir, nodeId.value)).entries.map(x => x.name))
    }
    catch {}
    for (const e of list) {
      const to = joinPath(targetDir, e.name)
      try {
        if (e.path === to) {
          skipped++
          continue
        }
        if (names.has(e.name) && !batchTransfer.overwrite) {
          skipped++
          continue
        }
        if (move) {
          await apiFile.rename(e.path, to, nodeId.value)
        }
        else {
          await apiFile.copy(e.path, to, nodeId.value, batchTransfer.overwrite)
        }
        names.add(e.name)
        done++
      }
      catch {
        skipped++
      }
    }
  }
  finally {
    batchTransfer.running = false
    batchTransfer.visible = false
  }
  if (done || skipped) {
    useFaToast().success(i18n.global.t(move ? 'files.list.transferDoneMove' : 'files.list.transferDoneCopy', { n: done }) + (skipped ? i18n.global.t('files.list.skippedSuffix', { n: skipped }) : ''))
  }
  load()
}

// ---- 批量/单项压缩（多源打成一个 tar.gz） ----
const compressVisible = ref(false)
const compressSrcs = ref<string[]>([])
const compressDest = ref('')
const compressing = ref(false)

function openCompress(srcs: string[]) {
  compressSrcs.value = srcs
  // 默认名：单项取条目名，多项 archive.tar.gz；与当前目录重名时递增后缀
  const base = srcs.length === 1 ? srcs[0].split('/').filter(Boolean).pop() || 'archive' : 'archive'
  const names = new Set(entries.value.map(e => e.name))
  let name = `${base}.tar.gz`
  for (let i = 2; names.has(name); i++) {
    name = `${base}-${i}.tar.gz`
  }
  compressDest.value = joinPath(cwd.value, name)
  compressVisible.value = true
}

async function doCompress() {
  if (!compressSrcs.value.length || !compressDest.value.trim()) {
    return
  }
  compressing.value = true
  try {
    await apiFile.compress(compressSrcs.value, compressDest.value.trim(), nodeId.value)
    compressVisible.value = false
    useFaToast().success(i18n.global.t('files.common.compressDone'))
    load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('files.common.compressFailed'), { description: e?.message })
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
    await apiFile.decompress(decompressTarget.value.path, decompressDest.value.trim(), nodeId.value)
    decompressVisible.value = false
    useFaToast().success(i18n.global.t('files.common.decompressDone'))
    load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('files.common.decompressFailed'), { description: e?.message })
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
    searchResults.value = await apiFile.search(cwd.value, kw, nodeId.value)
    searchVisible.value = true
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('files.common.searchFailed'), { description: e?.message })
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

// ---- 跨窗口拖拽：文件行可拖出（携带路径/节点），其它文件窗口拖入即移动到当前目录 ----
interface DragPayload {
  srcId: string
  node: string
  fromCwd: string
  items: Array<{ path: string, name: string, isDir: boolean }>
}
const DRAG_MIME = 'application/x-ypanel-files'

function onFileDragStart(entry: FileEntry, e: DragEvent) {
  // 拖动的行若在多选集里则携带整个选区，否则只带该行
  const items = selected.value.has(entry.path) && selected.value.size > 1
    ? entries.value.filter(x => selected.value.has(x.path)).map(x => ({ path: x.path, name: x.name, isDir: x.isDir }))
    : [{ path: entry.path, name: entry.name, isDir: entry.isDir }]
  const payload: DragPayload = { srcId: instanceId, node: nodeId.value, fromCwd: cwd.value, items }
  e.dataTransfer?.setData(DRAG_MIME, JSON.stringify(payload))
  e.dataTransfer?.setData('text/plain', items.map(x => x.path).join('\n'))
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'copyMove'
  }
}

function onDropToFiles(e: DragEvent) {
  const raw = e.dataTransfer?.getData(DRAG_MIME)
  if (!raw) {
    return
  }
  let payload: DragPayload
  try {
    payload = JSON.parse(raw) as DragPayload
  }
  catch {
    return
  }
  // 拖回原窗口不处理；跨节点走 core 中转流式拷贝（M56）
  if (payload.node !== nodeId.value) {
    const items = payload.items.map(x => ({ path: x.path, name: x.name, isDir: x.isDir }))
    apiFile.copyAcross({
      srcNode: payload.node, srcPath: items[0]?.path || payload.fromCwd,
      dstNode: nodeId.value, dstDir: cwd.value, items,
    }).then((res: any) => {
      useFaToast().info(i18n.global.t('files.list.crossTaskCreated'))
      const taskId = (res.data as any)?.taskId
      if (taskId) {
        const poll = setInterval(async () => {
          try {
            const t = await apiFile.taskGet(taskId)
            if (t.status !== 'running') {
              clearInterval(poll)
              if (t.status === 'success') {
                useFaToast().success(i18n.global.t('files.list.crossDone'))
                load()
              } else {
                useFaToast().error(i18n.global.t('files.list.crossFailed'), { description: t.error || '' })
              }
            }
          } catch {}
        }, 2000)
      }
    }).catch((e: any) => {
      useFaToast().error(i18n.global.t('common.opFailed'), { description: e?.message })
    })
    return
  }
  if (payload.srcId === instanceId) {
    return
  }
  const targets = payload.items
    .filter(x => x.path !== joinPath(cwd.value, x.name))
    .map(x => ({ from: x.path, to: joinPath(cwd.value, x.name), name: x.name }))
  if (!targets.length) {
    return
  }
  // 默认复制（安全语义）；按住 Shift/Alt 拖放为移动
  const move = e.shiftKey || e.altKey
  void transferInto(targets, move, payload.fromCwd)
}

async function transferInto(targets: Array<{ from: string, to: string, name: string }>, move: boolean, fromCwd: string) {
  let done = 0
  let skipped = 0
  for (const t of targets) {
    try {
      // 目标同名已存在则跳过（覆盖有风险，宁跳过）
      const exist = await apiFile.list(cwd.value, nodeId.value)
      if (exist.entries.some(x => x.name === t.name)) {
        skipped++
        continue
      }
      if (move) {
        await apiFile.rename(t.from, t.to, nodeId.value)
      }
      else {
        await apiFile.copy(t.from, t.to, nodeId.value)
      }
      done++
    }
    catch {
      skipped++
    }
  }
  if (done) {
    load()
  }
  if (done || skipped) {
    useFaToast().success(i18n.global.t(move ? 'files.list.transferDoneMove' : 'files.list.transferDoneCopy', { n: done }) + (skipped ? i18n.global.t('files.list.skippedSuffixReason', { n: skipped }) : ''))
  }
  // 移动改变源目录内容——通知源窗口刷新；复制不影响源
  if (move) {
    window.dispatchEvent(new CustomEvent('ypanel:files:moved', { detail: { srcId: instanceId, node: nodeId.value, fromCwd } }))
  }
}

function onSourceMoved(e: Event) {
  const detail = (e as CustomEvent).detail as { srcId: string, node: string, fromCwd: string }
  if (detail.srcId !== instanceId && detail.node === nodeId.value && detail.fromCwd === cwd.value) {
    load()
  }
}

/** 目录在新窗口打开（桌面工作台承载时） */
function openDirInNewWindow(entry: FileEntry) {
  embed?.openApp('file', { title: entry.name, launchOptions: { dir: entry.path, node: nodeId.value } })
}

onMounted(() => {
  loadNodes()
  load(cwd.value)
  window.addEventListener('ypanel:files:moved', onSourceMoved)
})

onBeforeUnmount(() => {
  window.removeEventListener('ypanel:files:moved', onSourceMoved)
})
</script>

<template>
  <div @dragover.prevent @drop.prevent="onDropToFiles">
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="folder-open" :size="24" />
          <span>{{ $t('files.list.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('files.list.description') }}</span>
      </template>
      <div class="flex flex-wrap items-center gap-2">
        <FaDropdown v-if="nodes.some(n => n.id !== 'local')" :items="nodeMenuItems">
          <FaButton variant="outline" size="sm" class="h-8">
            {{ nodeId === 'local' ? $t('files.common.localNode', { name: currentNodeName }) : currentNodeName }}
            <FaIcon name="i-lucide:chevron-down" class="ml-1 text-xs text-muted-foreground" />
          </FaButton>
        </FaDropdown>
        <div class="flex items-center gap-1">
          <FaInput
            v-model="searchKeyword"
            :placeholder="$t('files.list.searchPlaceholder')"
            class="w-44!"
            @keyup.enter="doSearch"
          />
          <FaButton variant="outline" size="sm" :loading="searching" @click="doSearch">
            <FaIcon name="i-lucide:search" class="mr-1" /> {{ $t('common.search') }}
          </FaButton>
        </div>
        <FaButton v-auth="['file:write']" variant="outline" size="sm" @click="mkdirVisible = true">
          <FaIcon name="i-lucide:folder-plus" class="mr-1" /> {{ $t('files.common.newDir') }}
        </FaButton>
        <FaButton v-auth="['file:write']" variant="outline" size="sm" :disabled="uploading" @click="pickUpload">
          <FaIcon name="i-lucide:upload" class="mr-1" /> {{ uploading ? $t('files.list.uploading', { n: uploadPercent }) : $t('files.list.uploadFile') }}
        </FaButton>
        <FaDropdown :items="batchMenuItems">
          <FaButton variant="outline" size="sm" :disabled="!selected.size">
            <FaIcon name="i-lucide:list-checks" class="mr-1" /> {{ $t('files.list.batch') }} ({{ selected.size }})
          </FaButton>
        </FaDropdown>
        <FaDropdown :items="moreMenuItems">
          <FaButton variant="outline" size="sm">
            {{ $t('files.more') }} <FaIcon name="i-lucide:chevron-down" class="ml-1 text-xs text-muted-foreground" />
          </FaButton>
        </FaDropdown>
        <FaButton variant="outline" size="icon-sm" :title="$t('common.refresh')" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" />
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 面包屑 -->
      <div class="mb-3 flex flex-wrap items-center gap-1 text-sm">
        <span v-if="nodeId !== 'local'" class="mr-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
          {{ currentNodeName }}
        </span>
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
        <span class="ml-auto text-xs text-muted-foreground">{{ $t('files.list.totalItems', { n: entries.length }) }}</span>
      </div>

      <!-- M38：收藏栏 -->
      <div v-if="favorites.length" class="mb-3 flex flex-wrap items-center gap-1.5">
        <FaIcon name="i-lucide:star" class="text-xs text-amber-500" />
        <span
          v-for="f in favorites" :key="f.id"
          class="group inline-flex cursor-pointer items-center gap-1 rounded-full border bg-muted/30 px-2 py-0.5 text-xs transition-colors hover:bg-accent/60"
          :title="f.path"
          @click="load(f.path)"
        >
          {{ f.name }}
          <button type="button" class="opacity-40 group-hover:opacity-100" @click.stop="removeFav(f)">×</button>
        </span>
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
              <th class="px-3 py-2">{{ $t('common.name') }}</th>
              <th class="hidden w-40 px-3 py-2 md:table-cell">{{ $t('common.size') }}</th>
              <th class="hidden w-56 px-3 py-2 lg:table-cell">{{ $t('files.list.ownerPerm') }}</th>
              <th class="hidden w-44 px-3 py-2 sm:table-cell">{{ $t('files.list.modifiedAt') }}</th>
              <th class="w-48 px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('common.loading') }}
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
              draggable="true"
              @dragstart="onFileDragStart(e, $event)"
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
                  <FaButton v-if="!e.isDir" variant="ghost" size="icon-sm" :title="$t('common.edit')" @click="openEditor(e)">
                    <FaIcon name="i-lucide:pen-line" class="text-sm" />
                  </FaButton>
                  <FaButton v-if="!e.isDir && previewKind(e.name)" variant="ghost" size="icon-sm" :title="$t('files.preview')" @click="openPreview(e)">
                    <FaIcon name="i-lucide:eye" class="text-sm" />
                  </FaButton>
                  <FaButton
                    variant="ghost" size="icon-sm" :class="favOf(e.path) ? 'text-amber-500!' : ''"
                    :title="favOf(e.path) ? $t('files.list.unfav') : $t('files.fav')" @click="toggleFav(e)"
                  >
                    <FaIcon name="i-lucide:star" class="text-sm" />
                  </FaButton>
                  <FaButton v-if="!e.isDir" variant="ghost" size="icon-sm" :title="$t('files.shared')" @click="quickShare(e)">
                    <FaIcon name="i-lucide:share-2" class="text-sm" />
                  </FaButton>
                  <FaButton v-if="!e.isDir" variant="ghost" size="icon-sm" :title="$t('common.download')" @click="openDownload(e)">
                    <FaIcon name="i-lucide:download" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" :title="$t('files.common.permissions')" @click="openChmod([e.path], [e])">
                    <FaIcon name="i-lucide:lock" class="text-sm" />
                  </FaButton>
                  <FaButton v-if="e.isDir && embed" variant="ghost" size="icon-sm" :title="$t('files.list.openInNewWindow')" @click="openDirInNewWindow(e)">
                    <FaIcon name="i-lucide:app-window" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" :title="$t('files.list.compress')" @click="openCompress([e.path])">
                    <FaIcon name="i-lucide:package" class="text-sm" />
                  </FaButton>
                  <FaButton v-if="isArchive(e)" variant="ghost" size="icon-sm" :title="$t('files.list.decompress')" @click="openDecompress(e)">
                    <FaIcon name="i-lucide:package-open" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" :title="$t('files.common.rename')" @click="openRename(e)">
                    <FaIcon name="i-lucide:text-cursor-input" class="text-sm" />
                  </FaButton>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && !entries.length && (!cwd || cwd === '/')">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('files.common.emptyDir') }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <!-- 新建目录 -->
    <FaModal v-model="mkdirVisible" :title="$t('files.common.newDir')" :destroy-on-close="true">
      <FaInput v-model="mkdirName" :placeholder="$t('files.dialogs.dirNamePlaceholder')" class="w-full" @keyup.enter="doMkdir" />
      <template #footer>
        <FaButton variant="outline" @click="mkdirVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton @click="doMkdir">
          {{ $t('files.common.create') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 重命名 -->
    <FaModal v-model="renameVisible" :title="$t('files.common.rename')" :destroy-on-close="true">
      <FaInput v-model="renameName" :placeholder="$t('files.dialogs.newNamePlaceholder')" class="w-full" @keyup.enter="doRename" />
      <template #footer>
        <FaButton variant="outline" @click="renameVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton @click="doRename">
          {{ $t('common.confirm') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 文件编辑工作台（VS Code 式弹窗） -->
    <FileEditorWorkspace />

    <!-- M38：回收站 -->
    <FaModal v-model="trashVisible" :title="$t('files.trash')" class="max-w-3xl!" :destroy-on-close="true">
      <div class="mb-2 flex items-center gap-2">
        <span class="text-xs text-muted-foreground">{{ $t('files.dialogs.trashHint') }}</span>
        <FaButton variant="outline" size="sm" class="ml-auto text-red-500!" :disabled="!trashItems.length" @click="trashClear">{{ $t('common.clear') }}</FaButton>
      </div>
      <div class="max-h-96 overflow-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('files.dialogs.originalPath') }}</th>
              <th class="hidden w-44 px-3 py-2 md:table-cell">{{ $t('files.dialogs.deletedAt') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="trashLoading && !trashItems.length">
              <td colspan="3" class="px-3 py-8 text-center text-muted-foreground">{{ $t('common.loading') }}</td>
            </tr>
            <tr v-else-if="!trashItems.length">
              <td colspan="3" class="px-3 py-8 text-center text-muted-foreground">{{ $t('files.dialogs.trashEmpty') }}</td>
            </tr>
            <tr v-for="t in trashItems" :key="t.name" class="border-t hover:bg-accent/30">
              <td class="px-3 py-2 font-mono text-xs break-all">{{ t.original }}</td>
              <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground md:table-cell">{{ new Date(t.trashedAt).toLocaleString('zh-CN', { hour12: false }) }}</td>
              <td class="px-3 py-2 text-right">
                <FaButton variant="ghost" size="sm" @click="trashRestore(t.name)">{{ $t('files.dialogs.restore') }}</FaButton>
                <FaButton variant="ghost" size="sm" class="text-red-500!" @click="trashPurge(t.name)">{{ $t('files.dialogs.purge') }}</FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaModal>

    <!-- M38：分享管理 -->
    <FaModal v-model="shareVisible" :title="$t('files.dialogs.shareTitle')" class="max-w-2xl!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-end gap-2">
          <label class="flex-1 space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('files.dialogs.filePath') }}</span>
            <FaInput v-model="shareForm.path" placeholder="/path/to/file" class="w-full" />
          </label>
          <label class="w-24 space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('files.dialogs.validDays') }}</span>
            <FaInput v-model="shareForm.days" type="number" class="w-full" />
          </label>
          <FaButton size="sm" @click="createShare">{{ $t('files.dialogs.createShare') }}</FaButton>
        </div>
        <div v-if="lastShareLink" class="flex items-center gap-2 rounded-md border border-emerald-500/40 bg-emerald-500/5 p-2">
          <span class="flex-1 truncate font-mono text-xs">{{ lastShareLink }}</span>
          <FaButton variant="outline" size="sm" @click="copyShareLink(lastShareLink)">{{ $t('common.copy') }}</FaButton>
        </div>
        <div class="rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('files.dialogs.shareFile') }}</th>
                <th class="hidden w-40 px-3 py-2 md:table-cell">{{ $t('files.dialogs.expireAt') }}</th>
                <th class="w-16 px-3 py-2">{{ $t('common.status') }}</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!shares.length">
                <td colspan="4" class="px-3 py-6 text-center text-muted-foreground">{{ $t('files.dialogs.noShares') }}</td>
              </tr>
              <tr v-for="sh in shares" :key="sh.id" class="border-t hover:bg-accent/30">
                <td class="max-w-0 truncate px-3 py-2 font-mono text-xs" :title="sh.path">{{ sh.path }}</td>
                <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground md:table-cell">{{ sh.expireAt ? new Date(sh.expireAt).toLocaleString('zh-CN', { hour12: false }) : $t('files.dialogs.forever') }}</td>
                <td class="px-3 py-2">
                  <span class="rounded-full px-2 py-0.5 text-xs" :class="sh.valid ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">{{ sh.valid ? $t('files.dialogs.valid') : $t('files.dialogs.invalid') }}</span>
                </td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="revokeShare(sh.id)">{{ $t('files.dialogs.revoke') }}</FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="text-xs text-muted-foreground">{{ $t('files.dialogs.shareHint') }}</p>
      </div>
    </FaModal>

    <!-- M38：远程下载 -->
    <FaModal v-model="remoteVisible" :title="$t('files.dialogs.remoteTitle')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <label class="block space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('files.dialogs.remoteUrlLabel') }}</span>
          <FaInput v-model="remoteForm.url" placeholder="https://example.com/file.tar.gz" class="w-full" />
        </label>
        <label class="block space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('files.dialogs.saveDir') }}</span>
          <FaInput v-model="remoteForm.destDir" class="w-full" />
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="remoteVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="remoteBusy" @click="doRemoteDownload">{{ $t('files.dialogs.startDownload') }}</FaButton>
      </template>
    </FaModal>

    <!-- M38：预览 -->
    <FaModal v-model="previewVisible" :title="previewEntry?.name || $t('files.preview')" class="max-w-4xl!" :destroy-on-close="true">
      <div class="flex max-h-[70vh] items-center justify-center overflow-auto rounded-lg bg-black/5 p-2 dark:bg-black/30">
        <img v-if="previewKind(previewEntry?.name || '') === 'image'" :src="downloadURL(previewEntry!)" class="max-h-[65vh] max-w-full rounded object-contain" :alt="previewEntry?.name">
        <video v-else-if="previewKind(previewEntry?.name || '') === 'video'" :src="downloadURL(previewEntry!)" controls class="max-h-[65vh] max-w-full rounded" />
        <audio v-else-if="previewKind(previewEntry?.name || '') === 'audio'" :src="downloadURL(previewEntry!)" controls class="w-96" />
        <iframe v-else-if="previewKind(previewEntry?.name || '') === 'pdf'" :src="downloadURL(previewEntry!)" class="h-[65vh] w-full rounded border-0" />
        <span v-else class="p-8 text-sm text-muted-foreground">{{ $t('files.dialogs.unsupportedPreview') }}</span>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="openDownload(previewEntry!)">{{ $t('common.download') }}</FaButton>
        <FaButton @click="previewVisible = false">{{ $t('common.close') }}</FaButton>
      </template>
    </FaModal>

    <!-- 隐藏上传控件 -->
    <input ref="uploadInput" type="file" multiple class="hidden" @change="onUploadChange">

    <!-- 权限（单项/批量共用，宿主模式） -->
    <YdChmodDialog v-model:visible="chmodDialogVisible" :paths="chmodPaths" :entries="chmodEntries" :node="nodeId" @done="load()" />

    <!-- 批量移动/复制：选目标目录 -->
    <YdDirPicker v-model:visible="dirPickerVisible" :node="nodeId" :initial-path="cwd" :title="batchTransfer.mode === 'move' ? $t('files.list.moveTo') : $t('files.list.copyTo')" @select="onBatchDirPicked" />

    <!-- 批量移动/复制：执行确认 -->
    <FaModal v-model="batchTransfer.visible" :title="batchTransfer.mode === 'move' ? $t('files.dialogs.batchConfirmMove') : $t('files.dialogs.batchConfirmCopy')" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div>
          {{ $t(batchTransfer.mode === 'move' ? 'files.dialogs.batchMoveLabel' : 'files.dialogs.batchCopyLabel') }} <span class="font-medium">{{ selected.size }}</span> {{ $t('files.dialogs.itemsTo') }}
          <div class="mt-1 rounded-md bg-muted/50 px-2.5 py-1.5 font-mono text-xs text-muted-foreground">
            {{ batchTransfer.target }}
          </div>
        </div>
        <label class="flex cursor-pointer items-center gap-2">
          <input v-model="batchTransfer.overwrite" type="checkbox">
          <span>{{ $t('files.dialogs.overwriteHint') }}</span>
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="batchTransfer.visible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="batchTransfer.running" @click="runBatchTransfer">
          {{ $t(batchTransfer.mode === 'move' ? 'files.list.moveVerb' : 'files.list.copyVerb') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 压缩（单项/批量共用） -->
    <FaModal v-model="compressVisible" :title="compressSrcs.length > 1 ? $t('files.dialogs.compressTitleMulti', { n: compressSrcs.length }) : $t('files.dialogs.compressTitle')" :destroy-on-close="true">
      <div class="space-y-2 text-sm">
        <div class="truncate text-xs text-muted-foreground" :title="compressSrcs.join('\n')">
          {{ compressSrcs.length > 1 ? $t('files.dialogs.selectedN', { n: compressSrcs.length }) : compressSrcs[0] }}
        </div>
        <FaInput v-model="compressDest" :placeholder="$t('files.dialogs.compressDestPlaceholder')" class="w-full" @keyup.enter="doCompress" />
        <div class="text-xs text-muted-foreground">
          {{ $t('files.dialogs.overwriteNote') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="compressVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="compressing" @click="doCompress">
          {{ $t('files.dialogs.startCompress') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 解压 -->
    <FaModal v-model="decompressVisible" :title="$t('files.dialogs.decompressTitle')" :destroy-on-close="true">
      <div class="space-y-2 text-sm">
        <div class="text-xs text-muted-foreground">
          {{ decompressTarget?.path }}
        </div>
        <FaInput v-model="decompressDest" :placeholder="$t('files.dialogs.decompressDestPlaceholder')" class="w-full" @keyup.enter="doDecompress" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="decompressVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="decompressing" @click="doDecompress">
          {{ $t('files.dialogs.startDecompress') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 搜索结果 -->
    <FaModal
      v-model="searchVisible"
      :title="$t('files.dialogs.searchResultTitle', { kw: searchKeyword, n: searchResults.length })"
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
          {{ $t('files.dialogs.noResults') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="searchVisible = false">
          {{ $t('common.close') }}
        </FaButton>
      </template>
    </FaModal>
    <!-- M56 复制到节点 -->
    <FaModal v-model="crossVisible" :title="$t('files.list.copyToNodeTitle')" class="max-w-md!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <label class="space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('files.list.targetNode') }}</span>
          <YdSelect
            v-model="crossForm.dstNode"
            :placeholder="$t('files.list.pickTargetNode')"
            :options="[
              { label: $t('files.list.pickTargetNode'), value: '', disabled: true },
              ...allNodes.map(n => ({ label: n.name, value: n.id })),
            ]"
            size="default"
            button-class="w-full"
          />
        </label>
        <label class="space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('files.list.targetDir') }}</span>
          <FaInput v-model="crossForm.dstDir" class="w-full" placeholder="/opt/data" />
        </label>
        <label class="flex items-center gap-2 text-xs text-muted-foreground">
          <input v-model="crossForm.overwrite" type="checkbox" class="accent-(--primary)">
          {{ $t('files.list.overwriteExisting') }}
        </label>
        <p class="text-[11px] text-muted-foreground">{{ $t('files.list.crossHint') }}</p>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="crossVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="crossing" @click="doCopyAcross">{{ $t('common.start') }}</FaButton>
      </template>
    </FaModal>

  </div>
</template>