<script setup lang="ts">
import type { AppTask } from '@/api/modules/task'
import YdLogViewer from '@/components/YdLogViewer/index.vue'
import { taskApi } from '@/api/modules/task'

defineOptions({
  name: 'TasksIndex',
})

const TYPE_LABEL: Record<string, string> = {
  'store-install': '应用安装',
  'store-uninstall': '应用卸载',
  'image-pull': '镜像拉取',
}

const STATUS_META: Record<string, { label: string, cls: string }> = {
  running: { label: '运行中', cls: 'bg-blue-500/10 text-blue-600' },
  success: { label: '成功', cls: 'bg-emerald-500/10 text-emerald-600' },
  failed: { label: '失败', cls: 'bg-red-500/10 text-red-600' },
}

const items = ref<Omit<AppTask, 'logText' | 'error'>[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const statusFilter = ref('')
const loading = ref(false)
let timer: ReturnType<typeof setInterval> | null = null

async function load() {
  loading.value = true
  try {
    const out = await taskApi.list({ status: statusFilter.value, page: page.value, pageSize: size.value })
    items.value = out.items
    total.value = out.total
  }
  finally {
    loading.value = false
  }
}

watch([statusFilter], () => {
  page.value = 1
  load()
})
watch([page, size], () => load())

// ---- 日志抽屉 ----
const logVisible = ref(false)
const logTask = ref<AppTask | null>(null)
const logLoading = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

function openLog(id: number) {
  logVisible.value = true
  void refreshLog(id)
  if (pollTimer) {
    clearInterval(pollTimer)
  }
  pollTimer = setInterval(() => {
    if (!logVisible.value || !logTask.value || logTask.value.status !== 'running') {
      if (pollTimer) {
        clearInterval(pollTimer)
        pollTimer = null
      }
      return
    }
    void refreshLog(id)
  }, 2000)
}

async function refreshLog(id: number) {
  logLoading.value = true
  try {
    logTask.value = await taskApi.get(id)
  }
  catch (e: any) {
    useFaToast().error('任务读取失败', { description: e?.message })
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }
  finally {
    logLoading.value = false
  }
}

function removeTask(t: { id: number, title: string }) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除任务记录',
    content: `确认删除「${t.title}」的记录？`,
    onConfirm: async () => {
      try {
        await taskApi.remove(t.id)
        await load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

async function clearFinished() {
  try {
    const out = await taskApi.clear()
    useFaToast().success(`已清理 ${out.cleared} 条记录`)
    await load()
  }
  catch (e: any) {
    useFaToast().error('清理失败', { description: e?.message })
  }
}

onMounted(() => {
  load()
  timer = setInterval(() => {
    // 列表自动刷新：有运行中任务时加速
    load()
  }, 5000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
  if (pollTimer) {
    clearInterval(pollTimer)
  }
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="list-check" :size="24" />
          <span>任务中心</span>
        </div>
      </template>
      <template #description>
        <span>应用安装 / 卸载 / 镜像拉取等耗时任务统一记录：点击"日志"查看实时输出</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="clearFinished">
          <FaIcon name="i-lucide:eraser" class="mr-1" /> 清理已结束
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="mb-3 flex items-center gap-2">
        <select v-model="statusFilter" class="h-8 rounded-md border bg-background px-2 text-sm outline-none">
          <option value="">全部状态</option>
          <option value="running">运行中</option>
          <option value="success">成功</option>
          <option value="failed">失败</option>
        </select>
        <span class="ml-auto text-xs text-muted-foreground">共 {{ total }} 条</span>
      </div>

      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/40 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2.5 font-medium">任务</th>
              <th class="px-4 py-2.5 font-medium">类型</th>
              <th class="px-4 py-2.5 font-medium">状态</th>
              <th class="px-4 py-2.5 font-medium">创建时间</th>
              <th class="px-4 py-2.5 text-right font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in items" :key="t.id" class="border-t hover:bg-accent/30">
              <td class="px-4 py-2.5">
                <div>{{ t.title }}</div>
                <div class="font-mono text-xs text-muted-foreground">{{ t.ref }}</div>
              </td>
              <td class="px-4 py-2.5 text-muted-foreground">{{ TYPE_LABEL[t.type] || t.type }}</td>
              <td class="px-4 py-2.5">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="STATUS_META[t.status]?.cls">
                  {{ STATUS_META[t.status]?.label || t.status }}
                </span>
              </td>
              <td class="px-4 py-2.5 text-xs text-muted-foreground">{{ new Date(t.createdAt).toLocaleString() }}</td>
              <td class="px-4 py-2.5">
                <div class="flex justify-end gap-1">
                  <FaButton size="sm" variant="ghost" @click="openLog(t.id)">
                    日志
                    <span v-if="t.status === 'running'" class="ml-1 inline-block size-1.5 animate-pulse rounded-full bg-blue-500" />
                    <span v-else-if="t.status === 'failed'" class="ml-1 inline-block size-1.5 rounded-full bg-red-500" />
                  </FaButton>
                  <FaButton v-if="t.status !== 'running'" size="sm" variant="ghost" class="text-red-500!" @click="removeTask(t)">
                    删除
                  </FaButton>
                </div>
              </td>
            </tr>
            <tr v-if="!items.length && !loading">
              <td colspan="5" class="px-4 py-10 text-center text-muted-foreground">
                暂无任务（商店安装 / 镜像拉取等操作会记录在这里）
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="total > 0" class="mt-4 flex justify-end">
        <FaPagination v-model:page="page" v-model:size="size" :total="total" @page-change="load" @size-change="load" />
      </div>
    </FaPageMain>

    <!-- 日志抽屉 -->
    <FaDrawer v-model="logVisible" :title="`任务日志：${logTask?.title || ''}`" class="max-w-3xl!">
      <div v-if="logTask" class="flex flex-col gap-3">
        <div class="flex items-center gap-3 text-sm">
          <span class="rounded-full px-2 py-0.5 text-xs" :class="STATUS_META[logTask.status]?.cls">
            {{ STATUS_META[logTask.status]?.label || logTask.status }}
          </span>
          <span class="text-muted-foreground">{{ TYPE_LABEL[logTask.type] || logTask.type }} · {{ new Date(logTask.createdAt).toLocaleString() }}</span>
          <span v-if="logTask.status === 'running'" class="ml-auto inline-flex items-center gap-1 text-xs text-blue-500">
            <span class="inline-block size-1.5 animate-pulse rounded-full bg-blue-500" /> 实时刷新中…
          </span>
        </div>
        <div v-if="logTask.error" class="rounded-md border border-red-500/30 bg-red-500/5 p-2 text-xs text-red-500">
          {{ logTask.error }}
        </div>
        <YdLogViewer :logs="logTask.logText" height="55vh" :loading="logLoading" />
      </div>
    </FaDrawer>
  </div>
</template>
