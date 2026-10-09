<script setup lang="ts">
import type * as Monaco from 'monaco-editor'
import { loadMonaco } from '@/utils/monacoLoader'
import YdCodeEditor from '@/components/YdCodeEditor/index.vue'
import { i18n } from '@/locales'

// SQL 编辑器：monaco sql 高亮 + 表/字段补全 + Ctrl+Enter 执行。
// 补全提示词由外部 setHints 注入（模块级共享，provider 全局只注册一次）。
const props = defineProps<{
  readOnly?: boolean
}>()

const emit = defineEmits<{
  run: []
}>()

const model = shallowRef<Monaco.editor.ITextModel | null>(null)

// 补全提示（SqlWorkbench 更新）
const hints = { tables: [] as string[], columns: [] as string[], views: [] as string[] }
let providerRegistered = false

async function registerProvider() {
  if (providerRegistered) {
    return
  }
  providerRegistered = true
  const m = await loadMonaco()
  m.languages.registerCompletionItemProvider('sql', {
    provideCompletionItems(model, position) {
      const word = model.getWordUntilPosition(position)
      const range = new m.Range(position.lineNumber, word.startColumn, position.lineNumber, word.endColumn)
      const mk = (label: string, kind: Monaco.languages.CompletionItemKind, detail: string): Monaco.languages.CompletionItem => ({
        label, kind, detail, insertText: label, range,
      })
      const items: Monaco.languages.CompletionItem[] = []
      for (const t of hints.tables) {
        items.push(mk(t, m.languages.CompletionItemKind.Class, i18n.global.t('dbadmin.editor.hintTable')))
      }
      for (const t of hints.views) {
        items.push(mk(t, m.languages.CompletionItemKind.Interface, i18n.global.t('dbadmin.editor.hintView')))
      }
      for (const c of hints.columns) {
        items.push(mk(c, m.languages.CompletionItemKind.Field, i18n.global.t('dbadmin.editor.hintColumn')))
      }
      return { suggestions: items }
    },
  })
}

function setHints(tables: string[], views: string[], columns: string[]) {
  hints.tables = tables
  hints.views = views
  hints.columns = columns
}

async function init() {
  await registerProvider()
  const m = await loadMonaco()
  model.value = m.editor.createModel('', 'sql')
}

function onReady(ed: Monaco.editor.IStandaloneCodeEditor) {
  loadMonaco().then((m) => {
    ed.addCommand(m.KeyMod.CtrlCmd | m.KeyCode.Enter, () => {
      emit('run')
    })
  })
}

onMounted(() => {
  init()
})

function getValue() {
  return model.value?.getValue() ?? ''
}

function setValue(text: string) {
  model.value?.setValue(text)
}

onBeforeUnmount(() => {
  model.value?.dispose()
  model.value = null
})

defineExpose({ getValue, setValue, setHints })
</script>

<template>
  <div class="relative size-full min-h-0">
    <YdCodeEditor
      v-if="model"
      :model="model"
      :read-only="props.readOnly"
      :minimap="false"
      :font-size="13"
      @ready="onReady"
    />
    <div
      v-else
      class="flex size-full items-center justify-center text-sm text-muted-foreground"
    >
      {{ $t('dbadmin.editor.loading') }}
    </div>
  </div>
</template>
