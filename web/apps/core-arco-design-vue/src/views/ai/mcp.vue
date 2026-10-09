<script setup lang="ts">
// MCP 服务器（智能/MCP）：外部工具服务器接入（stdio / streamable-http），工具自动并入对话。
import { i18n, tr } from '@/locales'
import type { MCPServer } from '@/api/modules/ai'
import { mcpApi, mcpOpsApi } from '@/api/modules/ai'
import type { MCPOperation } from '@/api/modules/ai'

defineOptions({
  name: 'aiMcp',
})

const toast = useFaToast()

const mcpServers = ref<MCPServer[]>([])
const mcpVisible = ref(false)
const mcpSaving = ref(false)
const mcpTesting = ref('')
const mcpForm = ref<MCPServer>({ name: '', transport: 'stdio', command: '', args: [], url: '', enabled: true })
const mcpTools = ref<string[]>([])
const mcpTestName = ref('')

async function loadMCP() {
  try {
    mcpServers.value = await mcpApi.list()
  }
  catch {}
}

const mcpArgsStr = computed({
  get: () => (mcpForm.value.args || []).join(' '),
  set: (v: string) => {
    mcpForm.value.args = v.split(/\s+/).filter(Boolean)
  },
})

function openMCP(s?: MCPServer) {
  mcpForm.value = s ? JSON.parse(JSON.stringify(s)) : { name: '', transport: 'stdio', command: '', args: [], url: '', enabled: true }
  mcpVisible.value = true
}

async function saveMCP() {
  mcpSaving.value = true
  try {
    mcpServers.value = ((await mcpApi.saveAll(mcpServers.value)).data as any) || []
    mcpVisible.value = false
    toast.success(i18n.global.t('ai.mcp.saved'))
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.mcp.saveFailed'), { description: e?.message })
  }
  finally {
    mcpSaving.value = false
  }
}

async function testMCP(s: MCPServer) {
  mcpTesting.value = s.name
  try {
    const res = await mcpApi.test(s)
    mcpTools.value = res.tools || []
    mcpTestName.value = s.name
    toast.success(i18n.global.t('ai.mcp.testOk', { n: res.tools.length }))
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.mcp.testFailed'), { description: e?.message })
  }
  finally {
    mcpTesting.value = ''
  }
}

// ---- M45：MCP 写操作审批 ----
const ops = ref<MCPOperation[]>([])
const opsLoading = ref(false)
const opsState = ref('pending')
let opsTimer: ReturnType<typeof setInterval> | null = null

async function loadOps() {
  opsLoading.value = true
  try {
    ops.value = await mcpOpsApi.list(opsState.value)
  }
  finally {
    opsLoading.value = false
  }
}

async function decideOp(op: MCPOperation, approve: boolean) {
  await mcpOpsApi.approve(op.id, approve)
  toast.success(i18n.global.t(approve ? 'ai.mcp.approved' : 'ai.mcp.rejected'))
  await loadOps()
}

function fmtState(st: string) {
  return tr(`ai.mcp.state.${st}`, st)
}

function stateCls(st: string) {
  if (st === 'pending') return 'bg-amber-500/10 text-amber-600'
  if (st === 'succeeded') return 'bg-emerald-500/10 text-emerald-600'
  if (st === 'failed' || st === 'rejected') return 'bg-red-500/10 text-red-600'
  return 'bg-muted text-muted-foreground'
}

watch(opsState, loadOps)

function toggleOpsPolling(on: boolean) {
  if (opsTimer) { clearInterval(opsTimer); opsTimer = null }
  if (on) { void loadOps(); opsTimer = setInterval(loadOps, 5000) }
}

onMounted(() => { loadMCP() })
onActivated(() => { loadMCP(); toggleOpsPolling(opsState.value === 'pending') })
onDeactivated(() => toggleOpsPolling(false))
</script>

<template>
  <div>
    <FaPageMain>
      <!-- M45：写操作审批 -->
      <div class="mb-6 rounded-lg border bg-background p-4">
        <div class="flex flex-wrap items-center gap-2">
          <FaIcon name="i-lucide:shield-check" class="text-base text-primary opacity-70" />
          <span class="text-sm font-medium">{{ $t('ai.mcp.opsTitle') }}</span>
          <select v-model="opsState" class="h-8 rounded-md border bg-background px-2 text-xs outline-none focus:border-primary">
            <option value="">{{ $t('common.all') }}</option>
            <option value="pending">{{ $t('ai.mcp.state.pending') }}</option>
            <option value="succeeded">{{ $t('common.success') }}</option>
            <option value="failed">{{ $t('common.failed') }}</option>
            <option value="expired">{{ $t('ai.mcp.state.expired') }}</option>
          </select>
          <FaButton variant="ghost" size="sm" @click="loadOps">{{ $t('common.refresh') }}</FaButton>
        </div>
        <div v-if="!ops.length" class="mt-2 text-xs text-muted-foreground">{{ $t('ai.mcp.opsEmpty') }}</div>
        <div v-for="op in ops" :key="op.id" class="mt-2 flex flex-wrap items-center gap-2 rounded-md border px-3 py-2 text-xs">
          <span class="rounded-full px-2 py-0.5" :class="stateCls(op.state)">{{ fmtState(op.state) }}</span>
          <span class="font-mono font-medium">{{ op.tool }}</span>
          <span class="max-w-64 truncate font-mono text-muted-foreground" :title="op.args">{{ op.args }}</span>
          <span class="text-muted-foreground">{{ new Date(op.createdAt).toLocaleTimeString('zh-CN', { hour12: false }) }}</span>
          <span v-if="op.approvedBy" class="text-muted-foreground">by {{ op.approvedBy }}</span>
          <div v-if="op.state === 'pending'" class="ml-auto flex gap-1">
            <FaButton variant="outline" size="sm" @click="decideOp(op, true)">{{ $t('ai.mcp.approve') }}</FaButton>
            <FaButton variant="outline" size="sm" class="text-red-500!" @click="decideOp(op, false)">{{ $t('ai.mcp.deny') }}</FaButton>
          </div>
        </div>
      </div>

      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <span class="text-xs text-muted-foreground">
            {{ $t('ai.mcp.testHint') }}
          </span>
          <FaButton size="sm" @click="openMCP()">
            <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('ai.mcp.addServer') }}
          </FaButton>
        </div>
        <div v-if="!mcpServers.length" class="rounded-lg border p-8 text-center text-sm text-muted-foreground">
          {{ $t('ai.mcp.empty') }}
        </div>
        <div v-for="sv in mcpServers" :key="sv.name" class="rounded-lg border p-4">
          <div class="flex items-center justify-between">
            <div class="min-w-0">
              <div class="flex items-center gap-2 text-sm font-medium">
                {{ sv.name }}
                <span class="rounded-full bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">{{ sv.transport }}</span>
                <span v-if="sv.enabled" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">{{ $t('common.enabled') }}</span>
              </div>
              <div class="mt-1 truncate font-mono text-xs text-muted-foreground">
                {{ sv.transport === 'stdio' ? `${sv.command} ${sv.args?.join(' ') || ''}` : sv.url }}
              </div>
            </div>
            <div class="flex shrink-0 gap-1">
              <FaButton variant="ghost" size="sm" :loading="mcpTesting === sv.name" @click="testMCP(sv)">
                {{ $t('common.test') }}
              </FaButton>
              <FaButton variant="ghost" size="sm" @click="openMCP(sv)">
                {{ $t('common.edit') }}
              </FaButton>
              <FaButton variant="ghost" size="sm" class="text-red-500!" @click="() => { mcpServers = mcpServers.filter(x => x.name !== sv.name) }">
                {{ $t('common.remove') }}
              </FaButton>
            </div>
          </div>
          <div v-if="mcpTestName === sv.name && mcpTools.length" class="mt-2 flex flex-wrap gap-1">
            <span v-for="t in mcpTools" :key="t" class="rounded bg-muted px-1.5 py-0.5 font-mono text-[11px]">
              {{ t }}
            </span>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- MCP 编辑 -->
    <FaModal v-model="mcpVisible" :title="mcpForm.name ? $t('ai.mcp.editTitle', { name: mcpForm.name }) : $t('ai.mcp.addTitle')" class="max-w-2xl!" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">{{ $t('common.name') }}</span>
          <FaInput v-model="mcpForm.name" :placeholder="$t('ai.mcp.namePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">{{ $t('ai.mcp.transport') }}</span>
          <select v-model="mcpForm.transport" class="h-9 flex-1 rounded-md border bg-background px-2 outline-none">
            <option value="stdio">{{ $t('ai.mcp.transportStdio') }}</option>
            <option value="streamable-http">{{ $t('ai.mcp.transportHttp') }}</option>
          </select>
        </div>
        <div v-if="mcpForm.transport === 'stdio'" class="space-y-3">
          <div class="flex items-center gap-3">
            <span class="w-24 shrink-0 text-muted-foreground">{{ $t('ai.mcp.command') }}</span>
            <FaInput v-model="mcpForm.command" placeholder="npx" class="flex-1" />
          </div>
          <div class="flex items-center gap-3">
            <span class="w-24 shrink-0 text-muted-foreground">{{ $t('ai.mcp.args') }}</span>
            <FaInput v-model="mcpArgsStr" placeholder="-y @modelcontextprotocol/server-filesystem /tmp" class="flex-1" />
          </div>
        </div>
        <div v-else class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">{{ $t('ai.mcp.endpoint') }}</span>
          <FaInput v-model="mcpForm.url" placeholder="http://127.0.0.1:3001/mcp" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">{{ $t('ai.mcp.envVars') }}</span>
          <FaInput v-model="mcpArgsStr" :placeholder="$t('ai.mcp.envPlaceholder')" disabled class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="mcpVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="mcpSaving" @click="saveMCP">
          {{ $t('common.save') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
