<script setup lang="ts">
import type { ContainerItem } from '@/api/modules/container'
import apiContainer from '@/api/modules/container'

defineOptions({
  name: 'ContainerIndex',
})

const appAccountStore = useAppAccountStore()

const containers = ref<ContainerItem[]>([])
const loading = ref(false)
const disabled = ref(false) // docker 不可用
const disabledMsg = ref('')
const acting = ref<string>('')

let timer: ReturnType<typeof setInterval> | null = null

async function load() {
  loading.value = true
  try {
    containers.value = await apiContainer.list()
    disabled.value = false
  }
  catch (e: any) {
    if (e?.code === 5002) {
      disabled.value = true
      disabledMsg.value = e?.message || '当前节点未检测到 Docker'
    }
    else {
      disabledMsg.value = e?.message || '加载失败'
    }
  }
  finally {
    loading.value = false
  }
}

async function action(c: ContainerItem, act: 'start' | 'stop' | 'restart') {
  acting.value = c.id + act
  try {
    await apiContainer.action(c.id, act)
    useFaToast().success(`已${act === 'start' ? '启动' : act === 'stop' ? '停止' : '重启'} ${c.name}`)
    await load()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

// ---- 日志 ----
const logsVisible = ref(false)
const logsTarget = ref<ContainerItem | null>(null)
const logsContent = ref('')
const logsFollowing = ref(false)
let logsAbort: AbortController | null = null

async function openLogs(c: ContainerItem) {
  logsTarget.value = c
  logsContent.value = ''
  logsVisible.value = true
  loadLogs(c, false)
}

function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}

async function loadLogs(c: ContainerItem, follow: boolean) {
  logsAbort?.abort()
  logsAbort = new AbortController()
  logsFollowing.value = follow
  try {
    const url = `${wsBase()}/${apiContainer.logsURL(c.id, appAccountStore.token, 1000, follow)}`
    const resp = await fetch(url, {
      headers: { Authorization: `Bearer ${appAccountStore.token}` },
      signal: logsAbort.signal,
    })
    if (!resp.ok || !resp.body) {
      throw new Error(`HTTP ${resp.status}`)
    }
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    for (;;) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }
      logsContent.value += decoder.decode(value, { stream: true })
      // 限制长度防内存膨胀
      if (logsContent.value.length > 2_000_000) {
        logsContent.value = logsContent.value.slice(-1_000_000)
      }
      await nextTick()
      scrollLogsToBottom()
    }
  }
  catch (e: any) {
    if (e?.name !== 'AbortError') {
      logsContent.value += `\n[日志流错误] ${e?.message || e}`
    }
  }
  finally {
    logsFollowing.value = false
  }
}

const logsBoxRef = useTemplateRef<HTMLElement>('logsBox')

function scrollLogsToBottom() {
  const el = logsBoxRef.value
  if (el) {
    el.scrollTop = el.scrollHeight
  }
}

function closeLogs() {
  logsAbort?.abort()
  logsVisible.value = false
}

const stateStyle: Record<string, string> = {
  running: 'text-emerald-600 bg-emerald-500/10',
  exited: 'text-muted-foreground bg-muted',
  paused: 'text-amber-600 bg-amber-500/10',
  created: 'text-blue-600 bg-blue-500/10',
  restarting: 'text-amber-600 bg-amber-500/10',
  dead: 'text-red-600 bg-red-500/10',
}

function fmtPorts(c: ContainerItem) {
  if (!c.ports?.length) {
    return '—'
  }
  return c.ports.map(p => p.hostPort ? `${p.hostPort}->${p.containerPort}/${p.proto}` : `${p.containerPort}/${p.proto}`).join('  ')
}

onMounted(() => {
  load()
  timer = setInterval(() => {
    if (!disabled.value && !acting.value) {
      load()
    }
  }, 5000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
  logsAbort?.abort()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="container" :size="24" />
          <span>容器管理</span>
        </div>
      </template>
      <template #description>
        <span>Docker 容器列表与电源操作</span>
      </template>
      <FaButton variant="outline" size="icon-sm" title="刷新" @click="load()">
        <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="disabled" class="mb-4 flex items-center gap-2 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
        <YdMorphIcon name="triangle-alert" :size="18" />
        {{ disabledMsg }}：未检测到可用的 Docker 环境（/var/run/docker.sock）。安装 Docker 后即可管理容器。
      </div>
      <div v-else-if="disabledMsg" class="mb-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30">
        {{ disabledMsg }}
      </div>

      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full min-w-200 text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">名称</th>
              <th class="px-3 py-2">镜像</th>
              <th class="px-3 py-2">状态</th>
              <th class="hidden px-3 py-2 lg:table-cell">端口</th>
              <th class="hidden px-3 py-2 xl:table-cell">创建时间</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !containers.length">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                加载中…
              </td>
            </tr>
            <tr v-else-if="!containers.length && !disabled">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                暂无容器
              </td>
            </tr>
            <tr v-for="c in containers" :key="c.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2">
                <div class="font-mono text-[13px] font-medium">{{ c.name }}</div>
                <div class="font-mono text-xs text-muted-foreground">{{ c.id }}</div>
              </td>
              <td class="max-w-48 truncate px-3 py-2 font-mono text-xs" :title="c.image">
                {{ c.image }}
              </td>
              <td class="px-3 py-2">
                <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="stateStyle[c.state] || stateStyle.exited">
                  <span class="inline-block size-1.5 rounded-full" :class="c.state === 'running' ? 'animate-pulse bg-current' : 'bg-current'" />
                  {{ c.state }}<span v-if="c.status" class="opacity-70">· {{ c.status }}</span>
                </span>
              </td>
              <td class="hidden px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell">
                {{ fmtPorts(c) }}
              </td>
              <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground xl:table-cell">
                {{ new Date(c.created).toLocaleString('zh-CN', { hour12: false }) }}
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton v-if="c.state !== 'running'" variant="outline" size="sm" :disabled="acting === c.id + 'start'" @click="action(c, 'start')">
                    启动
                  </FaButton>
                  <template v-else>
                    <FaButton variant="outline" size="sm" :disabled="acting === c.id + 'stop'" @click="action(c, 'stop')">
                      停止
                    </FaButton>
                    <FaButton variant="outline" size="sm" :disabled="acting === c.id + 'restart'" @click="action(c, 'restart')">
                      重启
                    </FaButton>
                  </template>
                  <FaButton variant="ghost" size="icon-sm" title="日志" @click="openLogs(c)">
                    <FaIcon name="i-lucide:scroll-text" class="text-sm" />
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <!-- 日志 -->
    <FaModal
      v-model="logsVisible"
      :title="`容器日志：${logsTarget?.name || ''}`"
      class="max-w-5xl!"
      :destroy-on-close="true"
      @close="closeLogs"
    >
      <div class="mb-2 flex items-center gap-2">
        <FaButton size="sm" :variant="logsFollowing ? 'default' : 'outline'" @click="logsTarget && loadLogs(logsTarget, !logsFollowing)">
          <FaIcon name="i-lucide:radio" class="mr-1" :class="logsFollowing ? 'animate-pulse' : ''" />
          {{ logsFollowing ? '跟踪中（点击停止）' : '跟踪日志' }}
        </FaButton>
        <span class="text-xs text-muted-foreground">最近 1000 行</span>
      </div>
      <pre ref="logsBox" class="h-96 overflow-auto rounded-md bg-muted/60 p-3 font-mono text-xs leading-relaxed">{{ logsContent || '暂无日志' }}</pre>
      <template #footer>
        <FaButton variant="outline" @click="closeLogs">
          关闭
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
