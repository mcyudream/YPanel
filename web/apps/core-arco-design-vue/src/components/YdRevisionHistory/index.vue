<script setup lang="ts">
import type * as Monaco from 'monaco-editor'
import type { ConfigRevisionMeta } from '@/api/modules/revision'
import apiFile from '@/api/modules/file'
import { revisionApi } from '@/api/modules/revision'
import { loadMonaco } from '@/utils/monacoLoader'

// 受管配置版本历史（M23 服务端快照）：列表 / 与当前盘上内容 diff / 一键回滚。
// 使用场景：应用详情、配置 tab 的「版本历史」按钮，以及文件工作台 HistoryDialog 的服务器版本页。
const props = defineProps<{
  node?: string
  path: string
}>()

const emit = defineEmits<{
  restored: []
}>()

const visible = defineModel<boolean>({ default: false })

const entries = ref<ConfigRevisionMeta[]>([])
const loading = ref(false)
const restoringId = ref(0)
const diffOpen = ref(false)
const diffOriginal = ref('')
const diffTitle = ref('')

// diff 用临时 model（组件自管，弹窗关闭时释放，不进 fileEditor store 注册表）
let diffModel: Monaco.editor.ITextModel | null = null
const diffModelRef = shallowRef<Monaco.editor.ITextModel | null>(null)

watch(visible, (v) => {
  if (v) {
    load()
  }
  else {
    diffOpen.value = false
    diffModel?.dispose()
    diffModel = null
    diffModelRef.value = null
  }
})

async function load() {
  loading.value = true
  try {
    entries.value = await revisionApi.list(props.node || 'local', props.path)
  }
  catch (e: any) {
    useFaToast().error('版本列表加载失败', { description: e?.message })
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
    diffTitle.value = `当前内容 vs 版本 #${rev.id}（${fmtTime(rev.createdAt)}）`
    diffOpen.value = true
  }
  catch (e: any) {
    useFaToast().error('对比失败', { description: e?.message })
  }
}

function rollback(rev: ConfigRevisionMeta) {
  const modal = useFaModal()
  modal.confirm({
    title: '回滚配置',
    content: `确认把 ${props.path} 回滚到 ${fmtTime(rev.createdAt)} 的版本？当前盘上内容会先自动存一条快照。`,
    onConfirm: async () => {
      restoringId.value = rev.id
      try {
        await revisionApi.restore(rev.id)
        useFaToast().success('已回滚，盘上内容已恢复')
        emit('restored')
        await load()
      }
      catch (e: any) {
        useFaToast().error('回滚失败', { description: e?.message })
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
  return t === 'rollback' ? '回滚' : '保存前快照'
}
</script>

<template>
  <div>
    <div class="text-xs text-muted-foreground">
      面板保存受管配置（compose 项目文件、daemon.json 等）时自动快照旧内容；每路径保留最近 50 份。
    </div>
    <div class="mt-2 overflow-auto rounded-md border" style="max-height: 320px;">
      <table class="w-full text-sm">
        <thead class="sticky top-0 bg-muted/60 text-left text-xs text-muted-foreground backdrop-blur">
          <tr>
            <th class="px-3 py-2">时间</th>
            <th class="px-3 py-2">来源</th>
            <th class="px-3 py-2">操作人</th>
            <th class="hidden px-3 py-2 md:table-cell">说明</th>
            <th class="px-3 py-2 text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !entries.length">
            <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
              加载中…
            </td>
          </tr>
          <tr v-else-if="!entries.length">
            <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
              暂无版本（首次通过面板保存后自动记录）
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
                对比当前
              </FaButton>
              <FaButton variant="outline" size="sm" :loading="restoringId === rev.id" @click="rollback(rev)">
                回滚
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
          关闭
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
