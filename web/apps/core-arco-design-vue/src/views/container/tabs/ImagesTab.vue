<script setup lang="ts">
import type { DockerImage } from '@/api/modules/dockerext'
import YdLogViewer from '@/components/YdLogViewer/index.vue'
import { dockerExtApi } from '@/api/modules/dockerext'
import { taskApi } from '@/api/modules/task'

// 镜像管理（自 Docker 管理页迁入，M23）：列表 + 拉取 + 删除 + 清理悬空。
const toast = useFaToast()

const images = ref<DockerImage[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    images.value = await dockerExtApi.images()
  }
  catch (e: any) {
    toast.error('镜像列表加载失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

onMounted(load)

const search = ref('')
const filtered = computed(() => {
  const kw = search.value.trim().toLowerCase()
  if (!kw) {
    return images.value
  }
  return images.value.filter(img => img.tags.some(t => t.toLowerCase().includes(kw)) || img.id.toLowerCase().includes(kw))
})

const pullVisible = ref(false)
const pullRef = ref('')
const pulling = ref(false)
const pullTaskId = ref(0)
const pullTask = ref<{ id: number, status: string, logText: string, error: string } | null>(null)
let pullPollTimer: ReturnType<typeof setInterval> | null = null

async function refreshPullTask() {
  if (!pullTaskId.value) {
    return
  }
  try {
    const t = await taskApi.get(pullTaskId.value)
    pullTask.value = t
    if (t.status !== 'running') {
      if (pullPollTimer) {
        clearInterval(pullPollTimer)
        pullPollTimer = null
      }
      if (t.status === 'success') {
        toast.success('镜像拉取完成')
        await load()
      }
      else {
        toast.error('拉取失败', { description: t.error })
      }
    }
  }
  catch (e: any) {
    toast.error('任务状态读取失败', { description: e?.message })
    if (pullPollTimer) {
      clearInterval(pullPollTimer)
      pullPollTimer = null
    }
  }
}

async function doPull() {
  if (!pullRef.value.trim()) {
    return
  }
  pulling.value = true
  try {
    const out = await dockerExtApi.pull(pullRef.value.trim())
    pullTaskId.value = out.taskId
    pullTask.value = null
    void refreshPullTask()
    pullPollTimer = setInterval(() => {
      if (!pullTask.value || pullTask.value.status === 'running') {
        void refreshPullTask()
      }
      else if (pullPollTimer) {
        clearInterval(pullPollTimer)
        pullPollTimer = null
      }
    }, 2000)
  }
  catch (e: any) {
    toast.error('创建拉取任务失败', { description: e?.message })
  }
  finally {
    pulling.value = false
  }
}

watch(pullVisible, (v) => {
  if (!v && pullPollTimer) {
    clearInterval(pullPollTimer)
    pullPollTimer = null
  }
})

function removeImage(img: DockerImage) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除镜像',
    content: `确认删除镜像 ${img.tags[0] || img.id.slice(0, 12)}？`,
    onConfirm: async () => {
      try {
        await dockerExtApi.removeImage(img.id)
        toast.success('已删除')
        await load()
      }
      catch (e: any) {
        toast.error('删除失败', { description: e?.message })
      }
    },
  })
}

async function pruneImages() {
  try {
    const out = await dockerExtApi.pruneImages()
    toast.success(out)
    await load()
  }
  catch (e: any) {
    toast.error('清理失败', { description: e?.message })
  }
}

function fmtSize(mb: number) {
  return mb >= 1024 ? `${(mb / 1024).toFixed(2)} GB` : `${mb.toFixed(0)} MB`
}

function fmtTime(unix: number) {
  return new Date(unix * 1000).toLocaleString('zh-CN', { hour12: false })
}
</script>

<template>
  <div>
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <FaInput v-model="search" placeholder="搜索 TAG/ID…" class="h-8 w-52!" />
      <div class="ml-auto flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" title="刷新" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton variant="outline" size="sm" @click="pruneImages">
          清理悬空镜像
        </FaButton>
        <FaButton size="sm" @click="pullVisible = true">
          <FaIcon name="i-lucide:download" class="mr-1" /> 拉取镜像
        </FaButton>
      </div>
    </div>

    <div class="overflow-x-auto rounded-lg border">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
          <tr>
            <th class="px-3 py-2">TAG</th>
            <th class="px-3 py-2">ID</th>
            <th class="px-3 py-2">大小</th>
            <th class="hidden px-3 py-2 md:table-cell">创建时间</th>
            <th class="px-3 py-2 text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !filtered.length">
            <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
              加载中…
            </td>
          </tr>
          <tr v-else-if="!filtered.length">
            <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
              暂无镜像
            </td>
          </tr>
          <tr v-for="img in filtered" :key="img.id" class="border-t transition-colors hover:bg-accent/30">
            <td class="px-3 py-1.5 font-mono text-[13px]">
              <div class="flex items-center gap-2">
                <YdAppIcon :name="img.tags[0] || img.id" :size="18" />
                <div>
                  <div v-for="t in img.tags" :key="t">
                    {{ t }}
                  </div>
                  <span v-if="!img.tags.length" class="text-muted-foreground">&lt;none&gt;</span>
                </div>
              </div>
            </td>
            <td class="px-3 py-1.5 font-mono text-xs text-muted-foreground">
              {{ img.id.slice(0, 12) }}
            </td>
            <td class="px-3 py-1.5 text-xs tabular-nums">
              {{ fmtSize(img.sizeMb) }}
            </td>
            <td class="hidden px-3 py-1.5 text-xs tabular-nums text-muted-foreground md:table-cell">
              {{ fmtTime(img.createdAt) }}
            </td>
            <td class="px-3 py-1.5 text-right">
              <FaButton variant="outline" size="sm" @click="removeImage(img)">
                删除
              </FaButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <FaModal v-model="pullVisible" :title="pullTaskId ? `拉取进行中：${pullRef}` : '拉取镜像'" :destroy-on-close="true" :close-on-click-modal="false">
      <div v-if="!pullTaskId" class="flex flex-col gap-2">
        <FaInput v-model="pullRef" placeholder="如 redis:alpine 或 registry/example/img:tag" class="w-full" @keyup.enter="doPull" />
        <div class="text-xs text-muted-foreground">拉取为后台任务，可在「任务中心」查看历史进度</div>
      </div>
      <div v-else class="flex flex-col gap-2">
        <div class="text-xs text-muted-foreground">任务 #{{ pullTaskId }} · {{ pullTask?.status === 'success' ? '完成' : pullTask?.status === 'failed' ? '失败' : '拉取中…' }}</div>
        <YdLogViewer :logs="pullTask?.logText || ''" height="280px" />
      </div>
      <template #footer>
        <template v-if="!pullTaskId">
          <FaButton variant="outline" @click="pullVisible = false">
            取消
          </FaButton>
          <FaButton :loading="pulling" @click="doPull">
            创建拉取任务
          </FaButton>
        </template>
        <template v-else>
          <FaButton variant="outline" @click="pullVisible = false">
            {{ pullTask?.status === 'running' ? '后台运行' : '关闭' }}
          </FaButton>
        </template>
      </template>
    </FaModal>
  </div>
</template>
