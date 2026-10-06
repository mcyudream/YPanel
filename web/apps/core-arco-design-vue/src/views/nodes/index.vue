<script setup lang="ts">
import type { NodeItem } from '@/api/modules/node'
import apiNode from '@/api/modules/node'

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
      <FaButton size="sm" @click="genCode">
        <YdMorphIcon name="key-round" :size="14" class="mr-1" /> 生成配对码
      </FaButton>
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
