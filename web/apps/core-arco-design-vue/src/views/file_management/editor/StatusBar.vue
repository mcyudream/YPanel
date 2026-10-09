<script setup lang="ts">
import { FILE_ENCODINGS, encodingLabel } from '@/composables/useTextEncoding'
import HistoryDialog from './HistoryDialog.vue'
import { i18n } from '@/locales'

// 状态栏：主题 | 编码 | EOL | 历史版本 | 语言 | 自动换行 | 缩略图 | 光标位置
const appSettingsStore = useAppSettingsStore()
const store = useFileEditorStore()

const historyVisible = ref(false)

// 语言下拉（懒加载 monaco 后取常用列表）
const LANGS = [
  { id: 'plaintext', label: i18n.global.t('files.editor.plainText') },
  { id: 'markdown', label: 'Markdown' },
  { id: 'json', label: 'JSON' },
  { id: 'yaml', label: 'YAML' },
  { id: 'ini', label: 'INI/TOML' },
  { id: 'xml', label: 'XML' },
  { id: 'html', label: 'HTML/Vue' },
  { id: 'css', label: 'CSS' },
  { id: 'scss', label: 'SCSS' },
  { id: 'javascript', label: 'JavaScript' },
  { id: 'typescript', label: 'TypeScript' },
  { id: 'go', label: 'Go' },
  { id: 'shell', label: 'Shell' },
  { id: 'python', label: 'Python' },
  { id: 'php', label: 'PHP' },
  { id: 'sql', label: 'SQL' },
  { id: 'java', label: 'Java' },
  { id: 'c', label: 'C' },
  { id: 'cpp', label: 'C++' },
  { id: 'rust', label: 'Rust' },
  { id: 'dockerfile', label: 'Dockerfile' },
]

const currentLangLabel = computed(() => {
  if (store.activeTab?.langOverride) {
    return LANGS.find(l => l.id === store.activeTab?.langOverride)?.label ?? store.activeTab.langOverride
  }
  const id = store.activeTabId
  const model = id ? store.getModel(id) : null
  return model ? (LANGS.find(l => l.id === model.getLanguageId())?.label ?? model.getLanguageId()) : '—'
})

function isDark() {
  return appSettingsStore.settings.theme.colorScheme === 'dark'
}

function toggleTheme() {
  appSettingsStore.settings.theme.colorScheme = isDark() ? 'light' : 'dark'
}

function setEncodingWithConfirm(id: string) {
  const tab = store.activeTab
  if (!tab || tab.encoding === id) {
    return
  }
  const apply = () => tab.id && store.setEncoding(tab.id, id)
  if (tab.dirty) {
    useFaModal().confirm({
      title: i18n.global.t('files.editor.switchEncoding'),
      content: i18n.global.t('files.editor.switchEncodingContent'),
      onConfirm: apply,
    })
  }
  else {
    apply()
  }
}

function setEol(eol: 'lf' | 'crlf') {
  if (store.activeTabId) {
    store.setEol(store.activeTabId, eol)
  }
}

function setLang(id: string) {
  if (store.activeTabId) {
    store.setLanguage(store.activeTabId, id)
  }
}

const sbItem = 'inline-flex h-full cursor-pointer items-center gap-1 px-2 transition-colors hover:bg-accent/60'
</script>

<template>
  <div class="flex h-6 shrink-0 select-none items-center border-t bg-muted/60 text-xs text-muted-foreground">
    <!-- 左：节点 + 光标 + 脏计数 -->
    <span class="inline-flex h-full items-center gap-1 px-2" :title="$t('files.editor.currentNode')">
      <YdMorphIcon name="server" :size="12" />
      {{ store.currentNode }}
    </span>
    <span class="inline-flex h-full items-center px-2 tabular-nums" :title="$t('files.editor.cursorPos')">
      {{ $t('files.editor.cursor', { line: store.cursor.line, col: store.cursor.col }) }}<template v-if="store.cursor.selected">{{ $t('files.editor.selCount', { n: store.cursor.selected }) }}</template>
    </span>
    <span v-if="store.dirtyCount" class="inline-flex h-full items-center gap-1 px-2 text-amber-600 dark:text-amber-400">
      ● {{ $t('files.editor.dirtyCount', { n: store.dirtyCount }) }}
    </span>

    <div class="ml-auto flex h-full items-stretch">
      <button :title="isDark() ? $t('files.editor.toLight') : $t('files.editor.toDark')" :class="sbItem" @click="toggleTheme">
        <FaIcon :name="isDark() ? 'i-lucide:sun' : 'i-lucide:moon'" class="text-[13px]" />
      </button>

      <FaDropdown
        :items="[FILE_ENCODINGS.map(e => ({
          label: encodingLabel(e.id),
          disabled: !store.activeTab,
          handle: () => setEncodingWithConfirm(e.id),
        }))]"
      >
        <button :class="sbItem" :title="store.activeTab ? $t('files.editor.fileEncoding') : $t('files.editor.encodingNoFile')">
          {{ store.activeTab ? encodingLabel(store.activeTab.encoding) : $t('files.editor.encoding') }}
        </button>
      </FaDropdown>

      <FaDropdown
        :items="[[
          { label: 'LF（\\n）', handle: () => setEol('lf') },
          { label: 'CRLF（\\r\\n）', handle: () => setEol('crlf') },
        ]]"
      >
        <button :class="sbItem" :title="$t('files.editor.eol')">
          {{ (store.activeTab?.eol ?? 'lf').toUpperCase() }}
        </button>
      </FaDropdown>

      <button :class="sbItem" :disabled="!store.activeTab" @click="historyVisible = true">
        <FaIcon name="i-lucide:history" class="text-[13px]" />
        {{ $t('files.editor.history') }}
      </button>

      <FaDropdown
        :items="[LANGS.map(l => ({
          label: l.label,
          disabled: !store.activeTab,
          handle: () => setLang(l.id),
        }))]"
      >
        <button :class="sbItem" :title="$t('files.editor.langMode')">
          {{ currentLangLabel }}
        </button>
      </FaDropdown>

      <button :class="[sbItem, store.layout.wordWrap ? 'text-foreground' : '']" :title="$t('files.editor.wordWrap')" @click="store.toggleWrap">
        {{ $t('files.editor.wrap') }}
      </button>
      <button :class="[sbItem, store.layout.minimap ? 'text-foreground' : '']" :title="$t('files.editor.minimap')" @click="store.toggleMinimap">
        {{ $t('files.editor.minimap') }}
      </button>
    </div>

    <HistoryDialog v-model="historyVisible" :tab-id="store.activeTabId" />
  </div>
</template>
