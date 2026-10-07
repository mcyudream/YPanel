<script setup lang="ts">
// 顶栏任务中心：图标（运行中红点）+ 全局弹窗（任务列表 / 日志两态）。
// 弹窗组件全局唯一（挂在顶栏），任意页面经 useTaskCenterStore 唤起（含直接定位某任务日志）。
import type { AppTask } from '@/api/modules/task'
import YdLogViewer from '@/components/YdLogViewer/index.vue'
import { taskApi } from '@/api/modules/task'
import { useTaskCenterStore } from '@/store/modules/taskCenter'

defineOptions({
  name: 'ToolbarTaskCenter',
})

const appSettingsStore = useAppSettingsStore()
const taskCenter = useTaskCenterStore()

const running = ref(0)
let badgeTimer: ReturnType<typeof setInterval> | null = null

async function refreshBadge() {
  try {
    const out = await taskApi.list({ status: 'running', pageSize: 1 })
    running.value = out.total
  }
  catch {
    // 静默：角标轮询失败不打扰
  }
}

// ---------- 弹窗：任务列表 ----------
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
const listLoading = ref(false)
let listTimer: ReturnType<typeof setInterval> | null = null

async function loadList() {
  listLoading.value = true
  try {
    const out = await taskApi.list({ status: statusFilter.value, page: page.value, pageSize: size.value })
    items.value = out.items
    total.value = out.total
    running.value = items.value.filter(i => i.status === 'running').length
  }
  finally {
    listLoading.value = false
  }
}

// ---------- 弹窗：任务日志 ----------
const activeTask = ref<AppTask | null>(null)
const logLoading = ref(false)
let logTimer: ReturnType<typeof setInterval> | null = null

async function openLog(id: number) {
  await refreshLog(id)
  if (logTimer) {
    clearInterval(logTimer)
  }
  logTimer = setInterval(() => {
    if (activeTask.value?.status === 'running') {
      void refreshLog(id)
    }
    else if (logTimer) {
      clearInterval(logTimer)
      logTimer = null
    }
  }, 2000)
}

async function refreshLog(id: number) {
  logLoading.value = true
  try {
    activeTask.value = await taskApi.get(id)
  }
  catch (e: any) {
    useFaToast().error('任务读取失败', { description: e?.message })
    stopLogPolling()
  }
  finally {
    logLoading.value = false
  }
}

function stopLogPolling() {
  if (logTimer) {
    clearInterval(logTimer)
    logTimer = null
  }
}

function backToList() {
  stopLogPolling()
  taskCenter.activeTaskId = 0
  activeTask.value = null
  void loadList()
}

// ---------- 操作 ----------
function removeTask(t: { id: number, title: string }) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除任务记录',
    content: `确认删除「${t.title}」的记录？`,
    onConfirm: async () => {
      try {
        await taskApi.remove(t.id)
        if (activeTask.value?.id === t.id) {
          backToList()
        }
        else {
          await loadList()
        }
        void refreshBadge()
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
    await loadList()
    void refreshBadge()
  }
  catch (e: any) {
    useFaToast().error('清理失败', { description: e?.message })
  }
}

function fmtTime(t: string) {
  return new Date(t).toLocaleString('zh-CN', { hour12: false })
}

// ---------- 生命周期与联动 ----------
watch(statusFilter, () => {
  page.value = 1
  loadList()
})
watch([page, size], () => loadList())

// 打开弹窗：直接定位日志或进列表；关闭时停止轮询
watch(() => taskCenter.visible, (v) => {
  if (v) {
    if (taskCenter.activeTaskId > 0) {
      void openLog(taskCenter.activeTaskId)
    }
    else {
      void loadList()
    }
    if (listTimer) {
      clearInterval(listTimer)
    }
    listTimer = setInterval(() => {
      if (!taskCenter.activeTaskId) {
        void loadList()
      }
    }, 5000)
  }
  else {
    if (listTimer) {
      clearInterval(listTimer)
      listTimer = null
    }
    stopLogPolling()
    taskCenter.activeTaskId = 0
    activeTask.value = null
  }
})

// 外部（向导等）refreshTick 触发：弹窗开着则刷新
watch(() => taskCenter.refreshTick, () => {
  if (taskCenter.visible) {
    if (taskCenter.activeTaskId > 0) {
      void openLog(taskCenter.activeTaskId)
    }
    else {
      void loadList()
    }
  }
})

onMounted(() => {
  void refreshBadge()
  badgeTimer = setInterval(refreshBadge, 8000)
})

onBeforeUnmount(() => {
  if (badgeTimer) {
    clearInterval(badgeTimer)
  }
  if (listTimer) {
    clearInterval(listTimer)
  }
  stopLogPolling()
})
</script>

<template>
  <div v-if="appSettingsStore.mode === 'pc'" class="flex items-center">
    <FaButton variant="ghost" size="icon-sm" title="任务中心" class="relative!" @click="taskCenter.open()">
      <FaIcon name="i-lucide:list-checks" class="size-4" />
      <span
        v-if="running > 0"
        class="absolute right-0.5 top-0.5 inline-flex size-2 rounded-full bg-red-500 ring-2 ring-background"
      />
    </FaButton>

    <!-- 任务中心全局弹窗 -->
    <FaModal
      v-model="taskCenter.visible"
      :title="taskCenter.activeTaskId > 0 ? '任务日志' : '任务中心'"
      class="max-w-3xl!"
      :close-on-click-modal="false"
    >
      <!-- 列表态 -->
      <div v-if="!taskCenter.activeTaskId" class="flex flex-col gap-3">
        <div class="flex items-center gap-2">
          <select v-model="statusFilter" class="h-8 rounded-md border bg-background px-2 text-sm outline-none">
            <option value="">全部状态</option>
            <option value="running">运行中</option>
            <option value="success">成功</option>
            <option value="failed">失败</option>
          </select>
          <span class="ml-auto text-xs text-muted-foreground">共 {{ total }} 条</span>
          <FaButton variant="outline" size="sm" @click="clearFinished">
            <FaIcon name="i-lucide:eraser" class="mr-1" /> 清理已结束
          </FaButton>
        </div>
        <div class="max-h-[55vh] overflow-y-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead class="sticky top-0 bg-muted/60 text-left text-xs text-muted-foreground backdrop-blur">
              <tr>
                <th class="px-3 py-2 font-medium">任务</th>
                <th class="px-3 py-2 font-medium">类型</th>
                <th class="px-3 py-2 font-medium">状态</th>
                <th class="px-3 py-2 font-medium">创建时间</th>
                <th class="px-3 py-2 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="t in items" :key="t.id" class="border-t hover:bg-accent/30">
                <td class="px-3 py-2">
                  <div>{{ t.title }}</div>
                  <div class="font-mono text-xs text-muted-foreground">{{ t.ref }}</div>
                </td>
                <td class="px-3 py-2 text-muted-foreground">{{ TYPE_LABEL[t.type] || t.type }}</td>
                <td class="px-3 py-2">
                  <span class="rounded-full px-2 py-0.5 text-xs" :class="STATUS_META[t.status]?.cls">
                    {{ STATUS_META[t.status]?.label || t.status }}
                  </span>
                </td>
                <td class="px-3 py-2 text-xs text-muted-foreground">{{ fmtTime(t.createdAt) }}</td>
                <td class="px-3 py-2">
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
              <tr v-if="!items.length && !listLoading">
                <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
                  暂无任务（商店安装 / 镜像拉取等操作会记录在这里）
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="total > size" class="flex justify-end">
          <FaPagination v-model:page="page" v-model:size="size" :total="total" :layout="'pager, ->, jumper'" @page-change="loadList" @size-change="loadList" />
        </div>
      </div>

      <!-- 日志态 -->
      <div v-else-if="activeTask" class="flex flex-col gap-3">
        <div class="flex items-center gap-3 text-sm">
          <FaButton variant="outline" size="sm" @click="backToList">
            <FaIcon name="i-lucide:arrow-left" class="mr-1" /> 返回
          </FaButton>
          <span class="truncate font-medium">{{ activeTask.title }}</span>
          <span class="rounded-full px-2 py-0.5 text-xs" :class="STATUS_META[activeTask.status]?.cls">
            {{ STATUS_META[activeTask.status]?.label || activeTask.status }}
          </span>
          <span class="ml-auto text-xs text-muted-foreground">{{ TYPE_LABEL[activeTask.type] || activeTask.type }} · {{ fmtTime(activeTask.createdAt) }}</span>
          <span v-if="activeTask.status === 'running'" class="inline-flex items-center gap-1 text-xs text-blue-500">
            <span class="inline-block size-1.5 animate-pulse rounded-full bg-blue-500" /> 实时刷新中…
          </span>
        </div>
        <div v-if="activeTask.error" class="rounded-md border border-red-500/30 bg-red-500/5 p-2 text-xs text-red-500">
          {{ activeTask.error }}
        </div>
        <YdLogViewer :logs="activeTask.logText" height="50vh" :loading="logLoading" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="taskCenter.close()">
          {{ activeTask?.status === 'running' || (!taskCenter.activeTaskId && running > 0) ? '后台运行' : '关闭' }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
