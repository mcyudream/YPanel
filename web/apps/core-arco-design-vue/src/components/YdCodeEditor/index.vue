<script setup lang="ts">
import type * as Monaco from 'monaco-editor'
import { loadMonaco, monacoThemeName } from '@/utils/monacoLoader'

// Monaco 编辑器封装：单栏 / diff（传 diffOriginal 时）两种模式。
// model 由外部（fileEditor store）管理，同文件多组共享同一 model。
const props = defineProps<{
  model: Monaco.editor.ITextModel | null
  readOnly?: boolean
  wordWrap?: boolean
  minimap?: boolean
  fontSize?: number
  /** diff 模式的原始文本（历史版本预览用） */
  diffOriginal?: string
}>()

const emit = defineEmits<{
  cursor: [cursor: { line: number, col: number, selected: number }]
  ready: [editor: Monaco.editor.IStandaloneCodeEditor]
}>()

defineOptions({
  name: 'YdCodeEditor',
})

const appSettingsStore = useAppSettingsStore()
const containerRef = useTemplateRef<HTMLDivElement>('container')

let editor: Monaco.editor.IStandaloneCodeEditor | null = null
let diffEditor: Monaco.editor.IStandaloneDiffEditor | null = null
let originalModel: Monaco.editor.ITextModel | null = null

// settle 布局脉冲：FaModal 开启动画/懒加载分栏场景下，monaco 首次量到 0 高后
// automaticLayout 可能长期不再触发（实测 >15s 不自愈），在关键时间点主动补 layout
let layoutTimers: ReturnType<typeof setTimeout>[] = []

function scheduleSettleLayout() {
  layoutTimers.forEach(clearTimeout)
  layoutTimers = [50, 300, 1000, 2500, 5000, 12000].map(ms =>
    setTimeout(() => {
      editor?.layout()
      diffEditor?.layout()
    }, ms),
  )
}

// 交互兜底：点击编辑区 / 窗口尺寸变化时立即补 layout
let pokeListeners: (() => void)[] = []

function bindPokeLayout(el: HTMLElement) {
  const poke = () => {
    editor?.layout()
    diffEditor?.layout()
  }
  el.addEventListener('pointerdown', poke, { passive: true })
  window.addEventListener('resize', poke)
  pokeListeners = [
    () => el.removeEventListener('pointerdown', poke),
    () => window.removeEventListener('resize', poke),
  ]
}

function baseOptions(): Monaco.editor.IStandaloneEditorConstructionOptions {
  return {
    theme: monacoThemeName(appSettingsStore.settings.theme.colorScheme),
    readOnly: props.readOnly,
    fontSize: props.fontSize ?? 13,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    minimap: { enabled: props.minimap ?? true },
    wordWrap: props.wordWrap ? 'on' : 'off',
    scrollBeyondLastLine: false,
    smoothScrolling: true,
    cursorBlinking: 'smooth',
    renderWhitespace: 'selection',
    padding: { top: 8 },
    // monaco 自带 ResizeObserver 布局恢复：容器曾在 v-show 隐藏下挂载时自动纠正 0 尺寸
    automaticLayout: true,
  }
}

onMounted(async () => {
  const m = await loadMonaco()
  if (!containerRef.value) {
    return
  }
  bindPokeLayout(containerRef.value)
  if (props.diffOriginal !== undefined) {
    diffEditor = m.editor.createDiffEditor(containerRef.value, {
      ...baseOptions(),
      renderSideBySide: true,
      readOnly: true,
    })
    originalModel = m.editor.createModel(props.diffOriginal)
    diffEditor.setModel({ original: originalModel!, modified: props.model! })
  }
  else {
    editor = m.editor.create(containerRef.value, {
      ...baseOptions(),
      model: props.model ?? undefined,
    })
    editor.onDidChangeCursorPosition((e) => {
      const sel = editor?.getSelection()
      const selected = sel ? editor?.getModel()?.getValueInRange(sel).length ?? 0 : 0
      emit('cursor', { line: e.position.lineNumber, col: e.position.column, selected })
    })
    scheduleSettleLayout()
    emit('ready', editor)
  }
})

// flush: 'post' 关键——layout 必须在 v-show 容器补丁到 DOM 之后执行，
// 否则在 display:none 下量到 0 高，monaco 视图保持 0 行（automaticLayout 因尺寸不再变化也不会再触发）
watch(() => props.model, (model) => {
  if (editor) {
    if (model && editor.getModel() !== model) {
      const st = editor.saveViewState()
      editor.setModel(model)
      editor.restoreViewState(st)
    }
    editor.layout()
    scheduleSettleLayout()
  }
  if (diffEditor) {
    diffEditor.setModel({ original: originalModel!, modified: model! })
  }
}, { immediate: true, flush: 'post' })

watch(() => appSettingsStore.settings.theme.colorScheme, (scheme) => {
  loadMonaco().then(m => m.editor.setTheme(monacoThemeName(scheme)))
})

watch(() => props.readOnly, v => editor?.updateOptions({ readOnly: v }))
watch(() => props.wordWrap, v => editor?.updateOptions({ wordWrap: v ? 'on' : 'off' }))
watch(() => props.minimap, v => editor?.updateOptions({ minimap: { enabled: v } }))

onBeforeUnmount(() => {
  layoutTimers.forEach(clearTimeout)
  layoutTimers = []
  pokeListeners.forEach(fn => fn())
  pokeListeners = []
  editor?.dispose()
  diffEditor?.dispose()
  originalModel?.dispose()
  editor = null
  diffEditor = null
  originalModel = null
})

defineExpose({
  getEditor: () => editor,
})
</script>

<template>
  <div ref="container" class="size-full min-h-0 overflow-hidden" />
</template>
