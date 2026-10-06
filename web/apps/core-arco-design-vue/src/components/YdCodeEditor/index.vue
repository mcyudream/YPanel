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
let resizeObserver: ResizeObserver | null = null

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
    automaticLayout: false,
  }
}

onMounted(async () => {
  const m = await loadMonaco()
  if (!containerRef.value) {
    return
  }
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
    emit('ready', editor)
  }

  resizeObserver = new ResizeObserver(() => {
    editor?.layout()
    diffEditor?.layout()
  })
  resizeObserver.observe(containerRef.value)
})

watch(() => props.model, (model) => {
  if (editor && editor.getModel() !== model) {
    editor.setModel(model)
  }
  if (diffEditor) {
    diffEditor.setModel({ original: originalModel!, modified: model! })
  }
})

watch(() => appSettingsStore.settings.theme.colorScheme, (scheme) => {
  loadMonaco().then(m => m.editor.setTheme(monacoThemeName(scheme)))
})

watch(() => props.readOnly, v => editor?.updateOptions({ readOnly: v }))
watch(() => props.wordWrap, v => editor?.updateOptions({ wordWrap: v ? 'on' : 'off' }))
watch(() => props.minimap, v => editor?.updateOptions({ minimap: { enabled: v } }))

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
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
