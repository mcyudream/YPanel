<script setup lang="ts">
import type { FileHistoryEntry } from '@/composables/useFileHistory'
import { encodingLabel } from '@/composables/useTextEncoding'
import { fmtBytes } from '@/utils/format'
import { loadMonaco } from '@/utils/monacoLoader'

// 历史版本弹窗：本地快照（IndexedDB）+ 服务器版本（受管配置的服务端快照，M23）。
const props = defineProps<{
  tabId: string | null
}>()

const visible = defineModel<boolean>({ default: false })
const store = useFileEditorStore()

const entries = ref<FileHistoryEntry[]>([])
const selected = ref<FileHistoryEntry | null>(null)
const restoring = ref(false)
const source = ref<'local' | 'server'>('local')

const tab = computed(() => (props.tabId ? store.tabs[props.tabId] : undefined))
const model = computed(() => (props.tabId ? store.getModel(props.tabId) : null))

// 受管配置（与后端 service.ScopeFor 对齐）：compose 托管目录与 daemon.json，走服务端版本
const managed = computed(() => {
  const t = tab.value
  if (!t || t.containerId) {
    return false
  }
  return t.path === '/etc/docker/daemon.json' || t.path.startsWith('/opt/ypanel/compose/')
})

watch(visible, async (v) => {
  if (v && props.tabId) {
    selected.value = null
    source.value = 'local'
    entries.value = await listFileHistory(props.tabId)
  }
})

watch(source, (v) => {
  if (v === 'local' && props.tabId) {
    selected.value = null
    listFileHistory(props.tabId).then((e) => {
      entries.value = e
    })
  }
})

function fmtTime(ts: number) {
  return new Date(ts).toLocaleString('zh-CN', { hour12: false })
}

async function restore(entry: FileHistoryEntry) {
  const m = model.value
  if (!m || !tab.value) {
    return
  }
  restoring.value = true
  try {
    m.setValue(entry.content)
    const monaco = await loadMonaco()
    tab.value.eol = entry.eol
    m.setEOL(entry.eol === 'crlf' ? monaco.editor.EndOfLineSequence.CRLF : monaco.editor.EndOfLineSequence.LF)
    useFaToast().success('已恢复此版本（未保存，可 Ctrl+S 写盘）')
    visible.value = false
  }
  finally {
    restoring.value = false
  }
}
</script>

<template>
  <FaModal v-model="visible" title="历史版本" class="max-w-4xl!" :destroy-on-close="true">
    <div class="mb-2 flex items-center gap-2">
      <div class="flex overflow-hidden rounded-md border text-xs">
        <button
          type="button" class="px-2.5 py-1 transition-colors"
          :class="source === 'local' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent/50'"
          @click="source = 'local'"
        >
          本地快照
        </button>
        <button
          v-if="managed" type="button" class="px-2.5 py-1 transition-colors"
          :class="source === 'server' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent/50'"
          @click="source = 'server'"
        >
          服务器版本
        </button>
      </div>
      <span class="truncate text-xs text-muted-foreground">{{ tab?.path }}</span>
    </div>

    <!-- 本地快照（IndexedDB，仅本机） -->
    <div v-if="source === 'local'" class="text-xs text-muted-foreground">
      每次保存自动记录快照（本机 IndexedDB，按文件保留最近 30 份）。
    </div>
    <div v-if="source === 'local'" class="mt-2 flex gap-2">
      <!-- 列表 -->
      <div class="w-56 shrink-0 overflow-auto rounded-md border" style="height: 384px;">
        <button
          v-for="e in entries"
          :key="e.id"
          type="button"
          class="block w-full cursor-pointer border-b px-2 py-1.5 text-left text-xs transition-colors last:border-b-0 hover:bg-accent/50"
          :class="selected?.id === e.id ? 'bg-primary/10' : ''"
          @click="selected = e"
        >
          <div class="font-medium">{{ fmtTime(e.ts) }}</div>
          <div class="text-muted-foreground">{{ fmtBytes(e.size) }} · {{ encodingLabel(e.encoding) }}</div>
        </button>
        <div v-if="!entries.length" class="px-2 py-8 text-center text-muted-foreground">
          暂无快照，保存后自动记录
        </div>
      </div>
      <!-- diff 预览 -->
      <div class="min-w-0 flex-1 overflow-hidden rounded-md border" style="height: 384px;">
        <template v-if="selected">
          <YdCodeEditor :model="model" :diff-original="selected.content" class="h-full" />
        </template>
        <div v-else class="flex h-full items-center justify-center text-sm text-muted-foreground">
          选择左侧快照查看与当前内容的差异
        </div>
      </div>
    </div>

    <!-- 服务器版本（受管配置，跨设备可回滚） -->
    <div v-if="source === 'server' && managed && tab">
      <YdRevisionHistory :node="tab.node" :path="tab.path" @restored="store.reload(tab.id)" />
    </div>

    <template #footer>
      <FaButton variant="outline" @click="visible = false">
        关闭
      </FaButton>
      <FaButton v-if="source === 'local'" :disabled="!selected" :loading="restoring" @click="selected && restore(selected)">
        恢复此版本
      </FaButton>
    </template>
  </FaModal>
</template>
