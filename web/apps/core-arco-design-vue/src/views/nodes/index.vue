<script setup lang="ts">
import type { NodeItem } from '@/api/modules/node'
import apiNode from '@/api/modules/node'
import type { NodeExecResult } from '@/api/modules/nodeexec'
import { nodeExecApi } from '@/api/modules/nodeexec'

defineOptions({
  name: 'NodesIndex',
})

const nodes = ref<NodeItem[]>([])
const loading = ref(false)
const pairVisible = ref(false)
const pairCode = ref('')
const pairCommand = ref('')
let timer: ReturnType<typeof setInterval> | null = null

async function load() {
  loading.value = true
  try {
    nodes.value = await apiNode.list()
  }
  finally {
    loading.value = false
  }
}

async function genCode() {
  try {
    const res = await apiNode.pairingCode()
    pairCode.value = res.code
    pairCommand.value = `ypagent -core ${location.origin} -code ${res.code}`
    pairVisible.value = true
  }
  catch (e: any) {
    useFaToast().error('生成失败', { description: e?.message })
  }
}

// ---- 批量命令 ----
const execVisible = ref(false)
const execSelected = ref<string[]>(['local'])
const execCommand = ref('')
const execRunning = ref(false)
const execResults = ref<NodeExecResult[]>([])

function openExec() {
  execSelected.value = nodes.value.filter(n => n.online).map(n => n.id)
  execCommand.value = ''
  execResults.value = []
  execVisible.value = true
}

async function doExec() {
  if (!execSelected.value.length || !execCommand.value.trim()) {
    useFaToast().warning('请选择节点并输入命令')
    return
  }
  execRunning.value = true
  try {
    execResults.value = await nodeExecApi.exec(execSelected.value, execCommand.value)
  }
  catch (e: any) {
    useFaToast().error('执行失败', { description: e?.message })
  }
  finally {
    execRunning.value = false
  }
}

function remove(n: NodeItem) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除节点',
    content: `确认删除节点 ${n.name}？agent 需重新配对才能接入。`,
    onConfirm: async () => {
      try {
        await apiNode.remove(n.id)
        useFaToast().success('已删除')
        await load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

onMounted(() => {
  load()
  timer = setInterval(load, 10000)
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
          <YdMorphIcon name="network" :size="24" />
          <span>节点管理</span>
        </div>
      </template>
      <template #description>
        <span>多节点：目标机执行 ypagent 配对命令即可接入（心跳 30s，90s 无心跳视为离线）</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="openExec">
          <YdMorphIcon name="terminal-square" :size="14" class="mr-1" /> 批量命令
        </FaButton>
        <FaButton size="sm" @click="genCode">
          <YdMorphIcon name="key-round" :size="14" class="mr-1" /> 生成配对码
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="n in nodes"
          :key="n.id"
          class="rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
        >
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-2">
              <YdMorphIcon name="server" :size="22" class="text-primary opacity-80" />
              <div>
                <div class="flex items-center gap-2 font-medium">
                  {{ n.name }}
                  <span v-if="!n.remote" class="rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">本机</span>
                </div>
                <div v-if="n.remote" class="font-mono text-xs text-muted-foreground">{{ n.addr }}</div>
              </div>
            </div>
            <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="n.online ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'">
              <span class="inline-block size-1.5 rounded-full" :class="n.online ? 'animate-pulse bg-current' : 'bg-current'" />
              {{ n.online ? '在线' : '离线' }}
            </span>
          </div>
          <div v-if="n.remote" class="mt-3 flex items-center justify-between border-t pt-3">
            <span class="text-xs text-muted-foreground">
              {{ [n.os, n.arch, n.version].filter(Boolean).join(' · ') || '—' }}
            </span>
            <FaButton variant="outline" size="sm" @click="remove(n)">删除</FaButton>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 批量命令 -->
    <FaModal v-model="execVisible" title="批量命令" class="max-w-3xl!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex flex-wrap gap-1.5">
          <label
            v-for="n in nodes.filter(x => x.online)"
            :key="n.id"
            class="flex cursor-pointer items-center gap-1.5 rounded-md border px-2 py-1 text-xs transition-colors"
            :class="execSelected.includes(n.id) ? 'border-primary bg-primary/10' : 'border-border'"
          >
            <input v-model="execSelected" type="checkbox" :value="n.id" class="hidden">
            {{ n.name }}
          </label>
        </div>
        <textarea
          v-model="execCommand"
          class="h-20 w-full resize-y rounded-md border border-input bg-background p-2 font-mono text-[13px] outline-none focus:ring-1 focus:ring-primary"
          placeholder="将在所选节点并发执行的命令"
          spellcheck="false"
        />
        <FaButton :loading="execRunning" @click="doExec">执行</FaButton>
        <div v-for="r in execResults" :key="r.nodeId" class="rounded-md border p-2">
          <div class="mb-1 flex items-center gap-2 text-xs">
            <span class="rounded-full px-2 py-0.5" :class="r.ok ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'">
              {{ r.name || r.nodeId }} · {{ r.ok ? '成功' : '失败' }}
            </span>
            <span v-if="r.error" class="text-red-500">{{ r.error }}</span>
          </div>
          <pre v-if="r.output" class="max-h-40 overflow-auto whitespace-pre-wrap bg-muted/50 p-2 font-mono text-xs">{{ r.output }}</pre>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="execVisible = false">关闭</FaButton>
      </template>
    </FaModal>

    <!-- 配对码 -->
    <FaModal v-model="pairVisible" title="节点配对" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="text-sm text-muted-foreground">
          在目标节点机器上执行以下命令（需已下载 ypagent，core 地址以实际部署为准）：
        </div>
        <div class="rounded-md bg-muted/60 p-3 font-mono text-[13px] break-all select-all">
          {{ pairCommand }}
        </div>
        <div class="text-xs text-muted-foreground">
          配对码 <code class="rounded bg-muted px-1">{{ pairCode }}</code> 10 分钟内有效且只能使用一次；agent 将监听 0.0.0.0:9527 并保持心跳。
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="pairVisible = false">关闭</FaButton>
      </template>
    </FaModal>
  </div>
</template>
