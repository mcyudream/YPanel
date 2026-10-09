<script setup lang="ts">
import type * as Monaco from 'monaco-editor'
import type { ConfigRevisionMeta } from '@/api/modules/revision'
import apiFile from '@/api/modules/file'
import { revisionApi } from '@/api/modules/revision'
import { i18n } from '@/locales'
import { loadMonaco } from '@/utils/monacoLoader'

// 版本历史面板（无弹窗壳）：列表 / 与当前盘上内容 diff / 一键回滚。
// 由 YdRevisionHistory（自含 FaModal）或工作台 HistoryDialog（bare 内嵌）承载。
const props = defineProps<{
  node?: string
  path: string
}>()

const emit = defineEmits<{
  restored: []
}>()

const entries = ref<ConfigRevisionMeta[]>([])
const loading = ref(false)
const loadError = ref('')
const restoringId = ref(0)
const diffOpen = ref(false)
const diffOriginal = ref('')
const diffTitle = ref('')

// diff 用临时 model（组件自管，弹窗关闭时释放，不进 fileEditor store 注册表）
let diffModel: Monaco.editor.ITextModel | null = null
const diffModelRef = shallowRef<Monaco.editor.ITextModel | null>(null)

// 承载方均为「打开时重建」（FaModal destroy-on-close / 内嵌挂载），onMounted 即加载
onMounted(load)

onBeforeUnmount(() => {
  diffModel?.dispose()
  diffModel = null
})

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    entries.value = await revisionApi.list(props.node || 'local', props.path)
  }
  catch (e: any) {
    loadError.value = e?.message || i18n.global.t('components.ydRevisionHistory.loadFailed')
  }
  finally {
    loading.value = false
  }
}

async function openDiff(rev: ConfigRevisionMeta) {
  try {
    const [revFull, current] = await Promise.all([
      revisionApi.get(rev.id),
      apiFile.read(props.path, props.node || 'local'),
    ])
    const monaco = await loadMonaco()
    if (diffModel) {
      diffModel.dispose()
    }
    diffModel = monaco.editor.createModel(current.content)
    diffModelRef.value = diffModel
    diffOriginal.value = revFull.content
    diffTitle.value = i18n.global.t('components.ydRevisionHistory.diffTitle', { id: rev.id, time: fmtTime(rev.createdAt) })
    diffOpen.value = true
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('components.ydRevisionHistory.diffFailed'), { description: e?.message })
  }
}

function rollback(rev: ConfigRevisionMeta) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('components.ydRevisionHistory.rollbackConfirmTitle'),
    content: i18n.global.t('components.ydRevisionHistory.rollbackConfirm', { path: props.path, time: fmtTime(rev.createdAt) }),
    onConfirm: async () => {
      restoringId.value = rev.id
      try {
        await revisionApi.restore(rev.id)
        useFaToast().success(i18n.global.t('components.ydRevisionHistory.rolledBack'))
        emit('restored')
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('components.ydRevisionHistory.rollbackFailed'), { description: e?.message })
      }
      finally {
        restoringId.value = 0
      }
    },
  })
}

function fmtTime(t: string | number) {
  return new Date(t).toLocaleString('zh-CN', { hour12: false })
}

function triggerLabel(t: string) {
  return t === 'rollback' ? i18n.global.t('components.ydRevisionHistory.triggerRollback') : i18n.global.t('components.ydRevisionHistory.triggerSnapshot')
}
</script>

<template>
  <div>
    <div class="text-xs text-muted-foreground">
      {{ $t('components.ydRevisionHistory.hint') }}{{ path }}
    </div>
    <div class="mt-2 overflow-auto rounded-md border" style="max-height: 320px;">
      <table class="w-full text-sm">
        <thead class="sticky top-0 bg-muted/60 text-left text-xs text-muted-foreground backdrop-blur">
          <tr>
            <th class="px-3 py-2">{{ $t('common.time') }}</th>
            <th class="px-3 py-2">{{ $t('components.ydRevisionHistory.colSource') }}</th>
            <th class="px-3 py-2">{{ $t('components.ydRevisionHistory.colAuthor') }}</th>
            <th class="hidden px-3 py-2 md:table-cell">{{ $t('components.ydRevisionHistory.colNote') }}</th>
            <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !entries.length">
            <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
              {{ $t('common.loading') }}
            </td>
          </tr>
          <tr v-else-if="loadError">
            <td colspan="5" class="px-3 py-6 text-center text-sm text-red-500">
              {{ loadError }}
            </td>
          </tr>
          <tr v-else-if="!entries.length">
            <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
              {{ $t('components.ydRevisionHistory.empty') }}
            </td>
          </tr>
          <tr v-for="rev in entries" :key="rev.id" class="border-t transition-colors hover:bg-accent/30">
            <td class="px-3 py-1.5 text-xs tabular-nums">
              #{{ rev.id }} · {{ fmtTime(rev.createdAt) }}
            </td>
            <td class="px-3 py-1.5">
              <span
                class="rounded-full px-2 py-0.5 text-xs"
                :class="rev.trigger === 'rollback' ? 'bg-amber-500/10 text-amber-600' : 'bg-primary/10 text-primary'"
              >{{ triggerLabel(rev.trigger) }}</span>
            </td>
            <td class="px-3 py-1.5 text-xs text-muted-foreground">
              {{ rev.author || '—' }}
            </td>
            <td class="hidden max-w-64 truncate px-3 py-1.5 text-xs text-muted-foreground md:table-cell" :title="rev.note">
              {{ rev.note }}
            </td>
            <td class="px-3 py-1.5 text-right">
              <FaButton variant="ghost" size="sm" @click="openDiff(rev)">
                {{ $t('components.ydRevisionHistory.diffWithCurrent') }}
              </FaButton>
              <FaButton variant="outline" size="sm" :loading="restoringId === rev.id" @click="rollback(rev)">
                {{ $t('components.ydRevisionHistory.rollback') }}
              </FaButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 与当前内容 diff -->
    <FaModal v-model="diffOpen" :title="diffTitle" class="max-w-5xl!" :destroy-on-close="true">
      <div class="h-[420px] overflow-hidden rounded-md border">
        <YdCodeEditor v-if="diffModelRef" :model="diffModelRef" :diff-original="diffOriginal" :read-only="true" class="h-full" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="diffOpen = false">
          {{ $t('common.close') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
