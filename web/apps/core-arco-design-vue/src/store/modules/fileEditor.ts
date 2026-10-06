// 文件编辑工作台 store：编辑器组/tab/monaco model 注册表/布局偏好。
// monaco 命名空间与 model 不进响应式系统（避免 Proxy 化导致性能问题）。
import type * as Monaco from 'monaco-editor'
import type { MonacoNamespace } from '@/utils/monacoLoader'
import { defineStore } from 'pinia'
import apiFile from '@/api/modules/file'
import apiCFile from '@/api/modules/cfile'
import apiNode from '@/api/modules/node'
import type { NodeItem } from '@/api/modules/node'
import { pushFileHistory } from '@/composables/useFileHistory'
import { b64ToBytes, decodeWith, detectEol, detectEncoding, languageOf } from '@/composables/useTextEncoding'
import { loadMonaco } from '@/utils/monacoLoader'

export interface FileEditorTab {
  /** 宿主：`${node}::${path}`；容器：`c:${containerId}::${node}::${path}` */
  id: string
  path: string
  node: string
  /** 容器来源（存在时读写走容器文件 API，暂仅 UTF-8） */
  containerId?: string
  name: string
  encoding: string
  eol: 'lf' | 'crlf'
  truncated: boolean
  dirty: boolean
  /** 打开时的原始字节（base64），切换编码本地重解码用 */
  rawB64: string
  size: number
  saving: boolean
  /** 用户手动指定的语言（覆盖扩展名推断） */
  langOverride?: string
}

export interface FileEditorGroup {
  id: number
  tabIds: string[]
  activeTabId: string | null
}

export interface FileEditorLayout {
  sidebarVisible: boolean
  sidebarSide: 'left' | 'right'
  sidebarWidth: number
  terminalVisible: boolean
  terminalSide: 'bottom' | 'right'
  terminalSize: number
  wordWrap: boolean
  minimap: boolean
}

export interface FileEditorCursor {
  line: number
  col: number
  selected: number
}

const LAYOUT_KEY = 'ypanel-file-editor-layout'

function defaultLayout(): FileEditorLayout {
  return {
    sidebarVisible: true,
    sidebarSide: 'left',
    sidebarWidth: 20,
    terminalVisible: true,
    terminalSide: 'bottom',
    terminalSize: 30,
    wordWrap: false,
    minimap: true,
  }
}

function loadLayout(): FileEditorLayout {
  const base = defaultLayout()
  try {
    const raw = localStorage.getItem(LAYOUT_KEY)
    if (!raw) {
      return base
    }
    return { ...base, ...(JSON.parse(raw) as Partial<FileEditorLayout>) }
  }
  catch {
    return base
  }
}

// —— 非响应式注册表 ——
let monaco: MonacoNamespace | null = null
const modelRegistry = new Map<string, Monaco.editor.ITextModel>()
const editorRegistry = new Map<number, Monaco.editor.IStandaloneCodeEditor>()

async function ensureMonaco(): Promise<MonacoNamespace> {
  if (!monaco) {
    monaco = await loadMonaco()
  }
  return monaco
}

export const useFileEditorStore = defineStore('fileEditor', () => {
  // ---- 弹窗与节点 ----
  const visible = ref(false)
  /** 弹窗动画结束（opened 或可见后 350ms 兜底）后置 true：monaco 等需要真实容器尺寸的子组件此时才挂载 */
  const editorOpened = ref(false)
  watch(visible, (v) => {
    if (!v) {
      editorOpened.value = false
      return
    }
    // FaModal 定制尺寸下 @opened 可能不触发，用动画时长兜底
    setTimeout(() => {
      if (visible.value) {
        editorOpened.value = true
      }
    }, 350)
  })
  const nodes = ref<NodeItem[]>([])
  const currentNode = ref('local')
  /** 容器来源（openWorkspace 传入）：非空时编辑器打开/保存走容器文件 API */
  const currentContainer = ref('')

  // ---- tab / 组 ----
  const tabs = ref<Record<string, FileEditorTab>>({})
  const groups = ref<FileEditorGroup[]>([])
  const activeGroupId = ref(0)
  let groupSeq = 0

  // ---- 光标（状态栏展示） ----
  const cursor = ref<FileEditorCursor>({ line: 1, col: 1, selected: 0 })

  // ---- 布局（localStorage 持久化） ----
  const layout = ref<FileEditorLayout>(loadLayout())
  function persistLayout() {
    try {
      localStorage.setItem(LAYOUT_KEY, JSON.stringify(layout.value))
    }
    catch {}
  }
  watch(layout, persistLayout, { deep: true })

  // ---- getters ----
  const activeGroup = computed(() => groups.value.find(g => g.id === activeGroupId.value) ?? groups.value[0])
  const activeTabId = computed(() => activeGroup.value?.activeTabId ?? null)
  const activeTab = computed(() => (activeTabId.value ? tabs.value[activeTabId.value] : undefined))
  const dirtyTabs = computed(() => Object.values(tabs.value).filter(t => t.dirty))
  const dirtyCount = computed(() => dirtyTabs.value.length)

  // ---- model 管理 ----
  function getModel(tabId: string) {
    return modelRegistry.get(tabId) ?? null
  }

  function registerEditor(groupId: number, editor: Monaco.editor.IStandaloneCodeEditor | null) {
    if (editor) {
      editorRegistry.set(groupId, editor)
    }
    else {
      editorRegistry.delete(groupId)
    }
  }

  function activeEditor() {
    const gid = activeGroupId.value
    return gid ? editorRegistry.get(gid) : undefined
  }

  // 工具栏动作直达当前编辑器
  function runActiveAction(actionId: string) {
    activeEditor()?.getAction(actionId)?.run()
  }

  // ---- 组 ----
  function ensureGroup(): FileEditorGroup {
    let g = activeGroup.value
    if (!g) {
      groupSeq++
      g = { id: groupSeq, tabIds: [], activeTabId: null }
      groups.value.push(g)
    }
    activeGroupId.value = g.id
    return g
  }

  function activateTab(groupId: number, tabId: string) {
    const g = groups.value.find(x => x.id === groupId)
    if (!g || !g.tabIds.includes(tabId)) {
      return
    }
    activeGroupId.value = groupId
    g.activeTabId = tabId
  }

  // ---- 打开文件 ----
  async function open(path: string, node?: string, containerId?: string) {
    const nodeId = node ?? currentNode.value
    const cid = containerId ?? currentContainer.value
    const id = cid ? `c:${cid}::${nodeId}::${path}` : `${nodeId}::${path}`
    // 已打开 → 激活
    for (const g of groups.value) {
      if (g.tabIds.includes(id)) {
        activateTab(g.id, id)
        return
      }
    }
    const name = path.slice(path.lastIndexOf('/') + 1)
    try {
      const res = cid
        ? await apiCFile.read(cid, path, { raw: true })
        : await apiFile.read(path, nodeId, { raw: true })
      if (res.isBinary) {
        useFaToast().warning('二进制文件不支持编辑', { description: path })
        return
      }
      if (!res.contentB64) {
        useFaToast().error('读取失败', { description: '未收到文件内容' })
        return
      }
      const bytes = b64ToBytes(res.contentB64)
      const { encoding, text } = detectEncoding(bytes)
      const eol = detectEol(text)
      const lang = languageOf(path)

      const m = await ensureMonaco()
      let model = modelRegistry.get(id)
      if (!model) {
        const uri = m.Uri.parse(`ypanel:///${cid ? `c-${cid}` : nodeId}${path}`)
        model = m.editor.getModel(uri) ?? m.editor.createModel(text, lang, uri)
        modelRegistry.set(id, model)
        model.onDidChangeContent(() => {
          const t = tabs.value[id]
          if (t) {
            t.dirty = true
          }
        })
      }
      model.setValue(text)
      model.setEOL(eol === 'crlf' ? m.editor.EndOfLineSequence.CRLF : m.editor.EndOfLineSequence.LF)

      tabs.value[id] = {
        id,
        path,
        node: nodeId,
        containerId: cid || undefined,
        name,
        encoding,
        eol,
        truncated: res.truncated,
        dirty: false,
        rawB64: res.contentB64,
        size: res.size,
        saving: false,
      }
      const g = ensureGroup()
      g.tabIds.push(id)
      g.activeTabId = id
      activeGroupId.value = g.id
    }
    catch (e: unknown) {
      ;(window as unknown as Record<string, unknown>).__openErr = String(e)
      useFaToast().error('打开文件失败', { description: errMsg(e) })
    }
  }

  // ---- 关闭 ----
  /**
   * 关闭某组中的 tab；dirty 且未 force 时返回 'confirm' 由 UI 走确认流程。
   * 同文件可在多组打开，全部关闭后才释放 model。
   */
  function closeTab(tabId: string, groupId: number, force = false): 'confirm' | 'closed' {
    const tab = tabs.value[tabId]
    if (tab && tab.dirty && !force) {
      return 'confirm'
    }
    const stillUsed = groups.value.filter(g => g.id !== groupId && g.tabIds.includes(tabId))
    groups.value = groups.value.map((g) => {
      if (g.id !== groupId) {
        return g
      }
      const idx = g.tabIds.indexOf(tabId)
      const rest = g.tabIds.filter(t => t !== tabId)
      let nextActive = g.activeTabId
      if (g.activeTabId === tabId) {
        nextActive = rest[Math.max(0, idx - 1)] ?? rest[0] ?? null
      }
      return { ...g, tabIds: rest, activeTabId: nextActive }
    })
    // 组空且非唯一组 → 移除该组
    if (groups.value.length > 1) {
      groups.value = groups.value.filter((g) => {
        if (g.tabIds.length === 0 && g.id !== groups.value[0]?.id) {
          if (activeGroupId.value === g.id) {
            activeGroupId.value = groups.value[0].id
          }
          return false
        }
        return true
      })
    }
    if (!activeGroup.value && groups.value.length) {
      activeGroupId.value = groups.value[0].id
    }
    // 无任何组引用 → 删 tab 与 model
    if (!stillUsed.length && !groups.value.some(g => g.tabIds.includes(tabId))) {
      delete tabs.value[tabId]
      modelRegistry.get(tabId)?.dispose()
      modelRegistry.delete(tabId)
    }
    return 'closed'
  }

  // ---- 保存 ----
  async function save(tabId: string) {
    const tab = tabs.value[tabId]
    const model = modelRegistry.get(tabId)
    if (!tab || !model || tab.saving) {
      return
    }
    tab.saving = true
    try {
      const content = model.getValue()
      if (tab.containerId) {
        // 容器文件：直存 UTF-8（后端 tar 写回，暂不支持转码）
        if (tab.encoding !== 'utf-8') {
          useFaToast().error('容器文件暂仅支持 UTF-8 编码保存')
          return
        }
        await apiCFile.write(tab.containerId, tab.path, content)
      }
      else {
        await apiFile.write(tab.path, content, tab.node, tab.encoding === 'utf-8' ? undefined : tab.encoding)
      }
      tab.dirty = false
      await pushFileHistory({ key: tabId, content, encoding: tab.encoding, eol: tab.eol, size: content.length })
      useFaToast().success(tab.encoding === 'utf-8' ? '已保存' : `已以 ${tab.encoding} 编码保存`)
    }
    catch (e: unknown) {
      useFaToast().error('保存失败', { description: errMsg(e) })
    }
    finally {
      tab.saving = false
    }
  }

  async function saveAll() {
    for (const t of dirtyTabs.value) {
      await save(t.id)
    }
  }

  // ---- 编码 / EOL / 语言 ----
  /** 切换编码：本地用原始字节重解码（dirty 时由 UI 先确认放弃修改）。 */
  function setEncoding(tabId: string, encoding: string) {
    const tab = tabs.value[tabId]
    const model = modelRegistry.get(tabId)
    if (!tab || !model || tab.encoding === encoding) {
      return
    }
    try {
      const text = decodeWith(b64ToBytes(tab.rawB64), encoding)
      tab.encoding = encoding
      tab.eol = detectEol(text)
      model.setValue(text)
      const m = monaco
      if (m) {
        model.setEOL(tab.eol === 'crlf' ? m.editor.EndOfLineSequence.CRLF : m.editor.EndOfLineSequence.LF)
      }
    }
    catch (e: unknown) {
      useFaToast().error('编码切换失败', { description: errMsg(e) })
    }
  }

  function setEol(tabId: string, eol: 'lf' | 'crlf') {
    const tab = tabs.value[tabId]
    const model = modelRegistry.get(tabId)
    if (!tab || !model || tab.eol === eol || !monaco) {
      return
    }
    tab.eol = eol
    model.setEOL(eol === 'crlf' ? monaco.editor.EndOfLineSequence.CRLF : monaco.editor.EndOfLineSequence.LF)
  }

  function setLanguage(tabId: string, lang: string) {
    const tab = tabs.value[tabId]
    const model = modelRegistry.get(tabId)
    if (!tab || !model || !monaco) {
      return
    }
    tab.langOverride = lang
    monaco.editor.setModelLanguage(model, lang)
  }

  // ---- 跨组移动 / 切分 ----
  function moveTabToGroup(tabId: string, targetGroupId: number, index?: number) {
    const source = groups.value.find(g => g.tabIds.includes(tabId))
    const target = groups.value.find(g => g.id === targetGroupId)
    if (!source || !target || source.id === target.id) {
      return
    }
    source.tabIds = source.tabIds.filter(t => t !== tabId)
    if (source.activeTabId === tabId) {
      source.activeTabId = source.tabIds[source.tabIds.length - 1] ?? null
    }
    const at = index ?? target.tabIds.length
    target.tabIds.splice(at, 0, tabId)
    activeGroupId.value = target.id
    target.activeTabId = tabId
    // 源组空且非唯一组 → 移除
    if (source.tabIds.length === 0 && groups.value.length > 1) {
      groups.value = groups.value.filter(g => g.id !== source.id)
      if (activeGroupId.value === source.id) {
        activeGroupId.value = target.id
      }
    }
  }

  /** 切分：以 tab 新建一组 */
  function splitFromTab(tabId: string) {
    groupSeq++
    const g: FileEditorGroup = { id: groupSeq, tabIds: [tabId], activeTabId: tabId }
    groups.value.push(g)
    activeGroupId.value = g.id
  }

  function closeGroup(groupId: number) {
    if (groups.value.length <= 1) {
      return
    }
    const g = groups.value.find(x => x.id === groupId)
    if (!g) {
      return
    }
    for (const tabId of [...g.tabIds]) {
      closeTab(tabId, groupId, true)
    }
  }

  // ---- 布局 ----
  function toggleSidebar() {
    layout.value.sidebarVisible = !layout.value.sidebarVisible
  }
  function toggleTerminal() {
    layout.value.terminalVisible = !layout.value.terminalVisible
  }
  function toggleWrap() {
    layout.value.wordWrap = !layout.value.wordWrap
  }
  function toggleMinimap() {
    layout.value.minimap = !layout.value.minimap
  }

  // ---- 工作台开关 ----
  async function openWorkspace(path?: string, node?: string, containerId?: string) {
    visible.value = true
    currentContainer.value = containerId ?? ''
    if (node) {
      currentNode.value = node
    }
    if (!nodes.value.length) {
      await loadNodes()
    }
    if (path) {
      await open(path, node, containerId)
    }
  }

  /** 关闭工作台：有未保存内容时返回 'confirm' 由 UI 走确认。 */
  function requestClose(): 'confirm' | 'closed' {
    if (dirtyCount.value > 0) {
      return 'confirm'
    }
    visible.value = false
    editorOpened.value = false
    return 'closed'
  }

  async function loadNodes() {
    try {
      nodes.value = await apiNode.list()
      if (!nodes.value.some(n => n.id === currentNode.value)) {
        currentNode.value = 'local'
      }
    }
    catch {}
  }

  return {
    visible,
    editorOpened,
    nodes,
    currentNode,
    currentContainer,
    tabs,
    groups,
    activeGroupId,
    activeGroup,
    activeTabId,
    activeTab,
    dirtyTabs,
    dirtyCount,
    cursor,
    layout,
    open,
    closeTab,
    save,
    saveAll,
    setEncoding,
    setEol,
    setLanguage,
    moveTabToGroup,
    splitFromTab,
    closeGroup,
    activateTab,
    ensureGroup,
    toggleSidebar,
    toggleTerminal,
    toggleWrap,
    toggleMinimap,
    openWorkspace,
    requestClose,
    loadNodes,
    getModel,
    registerEditor,
    runActiveAction,
  }
})

function errMsg(e: unknown) {
  if (e && typeof e === 'object' && 'message' in e && typeof (e as { message: unknown }).message === 'string') {
    return (e as { message: string }).message
  }
  return String(e)
}
