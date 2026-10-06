<script setup lang="ts">
import * as z from 'zod'
import type { ContainerCreateReq, ContainerItem } from '@/api/modules/container'
import apiContainer from '@/api/modules/container'
import YdTerminalModal from '@/components/YdTerminal/Modal.vue'

defineOptions({
  name: 'ContainerIndex',
})

const appAccountStore = useAppAccountStore()
const toast = useFaToast()

const containers = ref<ContainerItem[]>([])
// F1：列表搜索三件套
const search = ref('')
const filteredContainers = computed(() => {
  const kw = search.value.trim().toLowerCase()
  if (!kw) {
    return containers.value
  }
  return containers.value.filter(c =>
    c.name.toLowerCase().includes(kw)
    || c.image.toLowerCase().includes(kw)
    || c.id.toLowerCase().includes(kw))
})
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
    toast.success(`已${act === 'start' ? '启动' : act === 'stop' ? '停止' : '重启'} ${c.name}`)
    await load()
  }
  catch (e: any) {
    toast.error('操作失败', { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

// ---- 删除（二次确认） ----
function remove(c: ContainerItem) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除容器',
    content: `确认删除容器 ${c.name}？容器可写层数据将一并删除，不可恢复。`,
    onConfirm: async () => {
      acting.value = c.id + 'rm'
      try {
        await apiContainer.remove(c.id, true)
        toast.success(`已删除 ${c.name}`)
        await load()
      }
      catch (e: any) {
        toast.error('删除失败', { description: e?.message })
      }
      finally {
        acting.value = ''
      }
    },
  })
}

// ---- 创建容器 ----
const createVisible = ref(false)
const creating = ref(false)
const createForm = ref<ContainerCreateReq & { cmdRaw: string, envRaw: string, portsRaw: string, mountsRaw: string }>({
  name: '',
  image: '',
  cmdRaw: '',
  envRaw: '',
  portsRaw: '',
  mountsRaw: '',
  restart: 'no',
  network: '',
})

function openCreate() {
  createForm.value = { name: '', image: '', cmdRaw: '', envRaw: '', portsRaw: '', mountsRaw: '', restart: 'no', network: '' }
  createVisible.value = true
}

function lines(raw: string): string[] {
  return raw.split('\n').map(s => s.trim()).filter(Boolean)
}

// F2：创建表单 zod 校验（vee-validate 体系内的 schema，先覆盖创建类表单）
const createSchema = z.object({
  name: z.string().regex(/^[a-z0-9][a-z0-9_-]*$/, '容器名需小写字母/数字开头，含 - _'),
  image: z.string().min(1, '镜像必填'),
})

async function submitCreate() {
  const f = createForm.value
  const parsed = createSchema.safeParse({ name: f.name, image: f.image })
  if (!parsed.success) {
    toast.error(parsed.error.issues[0]?.message ?? '表单校验失败')
    return
  }
  creating.value = true
  try {
    const req: ContainerCreateReq = {
      name: f.name,
      image: f.image,
      cmd: f.cmdRaw.trim() ? f.cmdRaw.trim().split(/\s+/) : undefined,
      env: f.envRaw.trim() ? lines(f.envRaw) : undefined,
      ports: f.portsRaw.trim()
        ? lines(f.portsRaw).map((l) => {
            const [hp, rest] = l.split(':')
            const [cp, proto] = rest.split('/')
            return { host: hp, container: cp, proto: proto || 'tcp' }
          })
        : undefined,
      mounts: f.mountsRaw.trim() ? lines(f.mountsRaw) : undefined,
      restart: f.restart === 'no' ? '' : f.restart,
      network: f.network?.trim() || undefined,
    }
    const id = await apiContainer.create(req)
    toast.success(`容器已创建：${id.slice(0, 12)}`)
    createVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error('创建失败', { description: e?.message })
  }
  finally {
    creating.value = false
  }
}

// ---- exec 终端（YdTerminalModal 承载） ----
const execVisible = ref(false)
const execTarget = ref<ContainerItem | null>(null)

function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}

function openExec(c: ContainerItem) {
  execTarget.value = c
  execVisible.value = true
}

// ---- 详情 ----
const inspectVisible = ref(false)
const inspectTarget = ref<ContainerItem | null>(null)
const inspectData = ref<Record<string, any> | null>(null)
const inspectStats = ref<Record<string, any> | null>(null)

function statsCPU(s: Record<string, any> | null): string {
  if (!s) {
    return '—'
  }
  // 后端返回 docker stats 原始 JSON：cpu_delta/system_delta
  try {
    const cpuDelta = (s.cpu_stats?.cpu_usage?.total_usage ?? 0) - (s.precpu_stats?.cpu_usage?.total_usage ?? 0)
    const sysDelta = (s.cpu_stats?.system_cpu_usage ?? 0) - (s.precpu_stats?.system_cpu_usage ?? 0)
    const ncpu = s.cpu_stats?.online_cpus || 1
    const cpuPct = sysDelta > 0 ? (cpuDelta / sysDelta) * ncpu * 100 : 0
    const memUsed = (s.memory_stats?.usage ?? 0) - (s.memory_stats?.stats?.cache ?? 0)
    const memLimit = s.memory_stats?.limit ?? 0
    return `${cpuPct.toFixed(1)}% / ${memLimit ? (memUsed / 1048576).toFixed(1) : 0}MB`
  }
  catch {
    return '—'
  }
}

async function openInspect(c: ContainerItem) {
  inspectTarget.value = c
  inspectData.value = null
  inspectStats.value = null
  inspectVisible.value = true
  try {
    const [d, s] = await Promise.all([
      apiContainer.inspect(c.id),
      c.state === 'running' ? apiContainer.stats(c.id).catch(() => null) : Promise.resolve(null),
    ])
    inspectData.value = d
    inspectStats.value = s
  }
  catch (e: any) {
    toast.error('获取详情失败', { description: e?.message })
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
        <span>Docker 容器列表、电源操作、终端与详情</span>
      </template>
      <FaInput v-model="search" placeholder="搜索名称/镜像/ID…" class="w-52!" />
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" title="刷新" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 创建容器
        </FaButton>
      </div>
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
            <tr v-else-if="!filteredContainers.length && !disabled">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                暂无容器
              </td>
            </tr>
            <tr v-for="c in filteredContainers" :key="c.id" class="border-t transition-colors hover:bg-accent/30">
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
                  <FaButton v-if="c.state === 'running'" variant="ghost" size="icon-sm" title="终端" @click="openExec(c)">
                    <FaIcon name="i-lucide:square-terminal" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" title="日志" @click="openLogs(c)">
                    <FaIcon name="i-lucide:scroll-text" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" title="详情" @click="openInspect(c)">
                    <FaIcon name="i-lucide:info" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" title="删除" class="text-red-500!" :disabled="acting === c.id + 'rm'" @click="remove(c)">
                    <FaIcon name="i-lucide:trash-2" class="text-sm" />
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

    <!-- 创建容器 -->
    <FaModal
      v-model="createVisible"
      title="创建容器"
      class="max-w-2xl!"
      :destroy-on-close="true"
    >
      <div class="grid gap-3 text-sm">
        <div class="grid grid-cols-2 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">容器名（小写字母/数字/中划线）</span>
            <FaInput v-model="createForm.name" placeholder="my-container" class="w-full" />
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">镜像</span>
            <FaInput v-model="createForm.image" placeholder="nginx:latest" class="w-full" />
          </label>
        </div>
        <label class="space-y-1">
          <span class="text-xs text-muted-foreground">启动命令（可选，空格分隔）</span>
          <FaInput v-model="createForm.cmdRaw" placeholder="nginx -g daemon off;" class="w-full" />
        </label>
        <div class="grid grid-cols-2 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">端口映射（每行 宿主:容器[/协议]）</span>
            <textarea
              v-model="createForm.portsRaw"
              rows="2"
              placeholder="8080:80&#10;53:53/udp"
              class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary"
            />
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">目录挂载（每行 宿主:容器[:模式]）</span>
            <textarea
              v-model="createForm.mountsRaw"
              rows="2"
              placeholder="/srv/www:/usr/share/nginx/html&#10;/etc/localtime:/etc/localtime:ro"
              class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary"
            />
          </label>
        </div>
        <label class="space-y-1">
          <span class="text-xs text-muted-foreground">环境变量（每行 KEY=VALUE）</span>
          <textarea
            v-model="createForm.envRaw"
            rows="2"
            placeholder="TZ=Asia/Shanghai&#10;MYSQL_ROOT_PASSWORD=secret"
            class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary"
          />
        </label>
        <div class="grid grid-cols-2 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">重启策略</span>
            <select v-model="createForm.restart" class="h-9 w-full rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
              <option value="no">不重启</option>
              <option value="always">always</option>
              <option value="unless-stopped">unless-stopped</option>
              <option value="on-failure">on-failure</option>
            </select>
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">网络（可选，默认 bridge）</span>
            <FaInput v-model="createForm.network" placeholder="bridge / host / 自定义网络名" class="w-full" />
          </label>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">
          取消
        </FaButton>
        <FaButton :loading="creating" @click="submitCreate">
          创建并启动
        </FaButton>
      </template>
    </FaModal>

    <!-- exec 终端（YdTerminalModal：双引擎，默认 xterm） -->
    <YdTerminalModal
      v-model:open="execVisible"
      :title="`容器终端：${execTarget?.name || ''}`"
      :container-id="execTarget?.id || ''"
    />

    <!-- 详情 -->
    <FaModal
      v-model="inspectVisible"
      :title="`容器详情：${inspectTarget?.name || ''}`"
      class="max-w-4xl!"
      :destroy-on-close="true"
    >
      <div v-if="inspectData" class="space-y-3 text-sm">
        <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">状态</div>
            <div class="mt-1 font-mono text-xs">{{ inspectData.State?.Status || '—' }}</div>
          </div>
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">CPU / 内存</div>
            <div class="mt-1 font-mono text-xs">{{ statsCPU(inspectStats) }}</div>
          </div>
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">重启策略</div>
            <div class="mt-1 font-mono text-xs">{{ inspectData.HostConfig?.RestartPolicy?.Name || 'no' }}</div>
          </div>
          <div class="rounded-md border p-3">
            <div class="text-xs text-muted-foreground">网络</div>
            <div class="mt-1 truncate font-mono text-xs" :title="Object.keys(inspectData.NetworkSettings?.Networks || {}).join(',')">
              {{ Object.keys(inspectData.NetworkSettings?.Networks || {}).join(',') || '—' }}
            </div>
          </div>
        </div>
        <div>
          <div class="mb-1 text-xs text-muted-foreground">挂载</div>
          <div class="max-h-32 space-y-1 overflow-auto rounded-md border p-2 font-mono text-xs">
            <div v-for="(m, i) in inspectData.Mounts || []" :key="i" class="truncate">
              {{ m.Source || m.Name }} → {{ m.Destination }}{{ m.RW === false ? ' (ro)' : '' }}
            </div>
            <div v-if="!(inspectData.Mounts || []).length" class="text-muted-foreground">
              无挂载
            </div>
          </div>
        </div>
        <details>
          <summary class="cursor-pointer text-xs text-muted-foreground">
            完整 Inspect JSON
          </summary>
          <pre class="mt-1 max-h-72 overflow-auto rounded-md bg-muted/60 p-3 font-mono text-xs leading-relaxed">{{ JSON.stringify(inspectData, null, 2) }}</pre>
        </details>
      </div>
      <div v-else class="py-10 text-center text-sm text-muted-foreground">
        加载中…
      </div>
      <template #footer>
        <FaButton variant="outline" @click="inspectVisible = false">
          关闭
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
