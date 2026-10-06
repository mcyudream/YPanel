<script setup lang="ts">
import type { ProcessItem, ServiceItem } from '@/api/modules/nodeexec'
import { procApi } from '@/api/modules/nodeexec'

defineOptions({
  name: 'ProcessesIndex',
})

const tab = ref<'processes' | 'services'>('processes')
const processes = ref<ProcessItem[]>([])
const services = ref<ServiceItem[]>([])
const loading = ref(false)
const keyword = ref('')
const acting = ref('')

let timer: ReturnType<typeof setInterval> | null = null

async function load() {
  loading.value = true
  try {
    if (tab.value === 'processes') {
      processes.value = await procApi.processes()
    }
    else {
      services.value = await procApi.services()
    }
  }
  catch (e: any) {
    useFaToast().error('加载失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

const filteredProcesses = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) {
    return processes.value
  }
  return processes.value.filter(p => p.name.toLowerCase().includes(kw) || p.cmdline.toLowerCase().includes(kw))
})

const filteredServices = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) {
    return services.value
  }
  return services.value.filter(s => s.name.toLowerCase().includes(kw) || s.desc.toLowerCase().includes(kw))
})

function fmtRss(n: number) {
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

async function kill(p: ProcessItem) {
  const modal = useFaModal()
  modal.confirm({
    title: '结束进程',
    content: `确认强制结束进程 ${p.name} (PID ${p.pid})？`,
    onConfirm: async () => {
      try {
        await procApi.kill(p.pid)
        useFaToast().success('已结束')
        await load()
      }
      catch (e: any) {
        useFaToast().error('操作失败', { description: e?.message })
      }
    },
  })
}

async function svcAction(s: ServiceItem, action: 'start' | 'stop' | 'restart') {
  acting.value = s.name + action
  try {
    await procApi.serviceAction(s.name, action)
    useFaToast().success(`已${action === 'start' ? '启动' : action === 'stop' ? '停止' : '重启'} ${s.name}`)
    await load()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

watch(tab, load)

onMounted(() => {
  load()
  timer = setInterval(() => {
    if (!acting.value) {
      load()
    }
  }, 5000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="cpu" :size="24" />
          <span>进程与服务</span>
        </div>
      </template>
      <template #description>
        <span>本机进程与 systemd 服务管理</span>
      </template>
      <div class="flex items-center gap-2">
        <FaInput v-model="keyword" placeholder="筛选…" class="w-48" />
        <FaButton variant="outline" size="icon-sm" @click="load">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <FaTabs
        v-model="tab" :list="[
          { label: `进程 (${filteredProcesses.length})`, value: 'processes' },
          { label: `服务 (${filteredServices.length})`, value: 'services' },
        ]"
      />

      <!-- 进程 -->
      <div v-if="tab === 'processes'" class="mt-3 overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">PID</th>
              <th class="px-3 py-2">名称</th>
              <th class="px-3 py-2">CPU%</th>
              <th class="px-3 py-2">内存%</th>
              <th class="hidden px-3 py-2 md:table-cell">RSS</th>
              <th class="hidden px-3 py-2 lg:table-cell">用户</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in filteredProcesses.slice(0, 200)" :key="p.pid" class="border-t hover:bg-accent/30">
              <td class="px-3 py-1.5 font-mono text-xs">{{ p.pid }}</td>
              <td class="max-w-40 truncate px-3 py-1.5 font-mono text-[13px]" :title="p.cmdline">{{ p.name }}</td>
              <td class="px-3 py-1.5 text-xs tabular-nums">{{ p.cpu.toFixed(1) }}</td>
              <td class="px-3 py-1.5 text-xs tabular-nums">{{ p.mem.toFixed(1) }}</td>
              <td class="hidden px-3 py-1.5 text-xs tabular-nums text-muted-foreground md:table-cell">{{ fmtRss(p.memRss) }}</td>
              <td class="hidden px-3 py-1.5 font-mono text-xs text-muted-foreground lg:table-cell">{{ p.user }}</td>
              <td class="px-3 py-1.5 text-right">
                <FaButton variant="outline" size="sm" @click="kill(p)">结束</FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 服务 -->
      <div v-else class="mt-3 overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">服务</th>
              <th class="hidden px-3 py-2 md:table-cell">加载</th>
              <th class="px-3 py-2">状态</th>
              <th class="hidden px-3 py-2 lg:table-cell">描述</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in filteredServices" :key="s.name" class="border-t hover:bg-accent/30">
              <td class="px-3 py-1.5 font-mono text-[13px]">{{ s.name }}</td>
              <td class="hidden px-3 py-1.5 text-xs text-muted-foreground md:table-cell">{{ s.load }}</td>
              <td class="px-3 py-1.5">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="s.active === 'active' ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  {{ s.active }}
                </span>
              </td>
              <td class="hidden max-w-64 truncate px-3 py-1.5 text-xs text-muted-foreground lg:table-cell" :title="s.desc">{{ s.desc }}</td>
              <td class="px-3 py-1.5 text-right">
                <div class="inline-flex items-center gap-1">
                  <FaButton v-if="s.active !== 'active'" variant="outline" size="sm" :disabled="acting === s.name + 'start'" @click="svcAction(s, 'start')">启动</FaButton>
                  <FaButton v-if="s.active === 'active'" variant="outline" size="sm" :disabled="acting === s.name + 'stop'" @click="svcAction(s, 'stop')">停止</FaButton>
                  <FaButton v-if="s.active === 'active'" variant="ghost" size="sm" :disabled="acting === s.name + 'restart'" @click="svcAction(s, 'restart')">重启</FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>
  </div>
</template>
