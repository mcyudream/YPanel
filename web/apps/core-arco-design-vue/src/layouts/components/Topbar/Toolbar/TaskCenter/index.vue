<script setup lang="ts">
// 顶栏任务中心入口：有运行中任务时显示红点（8s 轮询，轻量 pageSize=1）。
import { taskApi } from '@/api/modules/task'

defineOptions({
  name: 'ToolbarTaskCenter',
})

const appSettingsStore = useAppSettingsStore()
const router = useRouter()

const running = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

async function refresh() {
  try {
    const out = await taskApi.list({ status: 'running', pageSize: 1 })
    running.value = out.total
  }
  catch {
    // 静默：角标轮询失败不打扰
  }
}

function open() {
  router.push('/tasks')
  // 立即刷新一次，离开页面返回时红点及时消退
  void refresh()
}

onMounted(() => {
  void refresh()
  timer = setInterval(refresh, 8000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<template>
  <FaButton
    v-if="appSettingsStore.mode === 'pc'"
    variant="ghost"
    size="icon-sm"
    title="任务中心"
    class="relative!"
    @click="open"
  >
    <FaIcon name="i-lucide:list-checks" class="size-4" />
    <span
      v-if="running > 0"
      class="absolute right-0.5 top-0.5 inline-flex size-2 rounded-full bg-red-500 ring-2 ring-background"
    />
  </FaButton>
</template>
