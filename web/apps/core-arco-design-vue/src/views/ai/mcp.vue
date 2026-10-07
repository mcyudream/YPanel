<script setup lang="ts">
// MCP 服务器（智能/MCP）：外部工具服务器接入（stdio / streamable-http），工具自动并入对话。
import type { MCPServer } from '@/api/modules/ai'
import { mcpApi } from '@/api/modules/ai'

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
    toast.success('MCP 服务器配置已保存')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
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
    toast.success(`连接成功，发现 ${res.tools.length} 个工具`)
  }
  catch (e: any) {
    toast.error('连接失败', { description: e?.message })
  }
  finally {
    mcpTesting.value = ''
  }
}

onMounted(loadMCP)
onActivated(loadMCP)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        MCP 服务器
      </template>
      <template #description>
        Model Context Protocol：接入外部工具服务器（stdio 常驻进程 / streamable-http 端点），其工具自动并入对话
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <span class="text-xs text-muted-foreground">
            保存后按「测试」验证连通性并列出工具
          </span>
          <FaButton size="sm" @click="openMCP()">
            <FaIcon name="i-lucide:plus" class="mr-1" /> 添加服务器
          </FaButton>
        </div>
        <div v-if="!mcpServers.length" class="rounded-lg border p-8 text-center text-sm text-muted-foreground">
          暂未接入 MCP 服务器
        </div>
        <div v-for="sv in mcpServers" :key="sv.name" class="rounded-lg border p-4">
          <div class="flex items-center justify-between">
            <div class="min-w-0">
              <div class="flex items-center gap-2 text-sm font-medium">
                {{ sv.name }}
                <span class="rounded-full bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">{{ sv.transport }}</span>
                <span v-if="sv.enabled" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">启用</span>
              </div>
              <div class="mt-1 truncate font-mono text-xs text-muted-foreground">
                {{ sv.transport === 'stdio' ? `${sv.command} ${sv.args?.join(' ') || ''}` : sv.url }}
              </div>
            </div>
            <div class="flex shrink-0 gap-1">
              <FaButton variant="ghost" size="sm" :loading="mcpTesting === sv.name" @click="testMCP(sv)">
                测试
              </FaButton>
              <FaButton variant="ghost" size="sm" @click="openMCP(sv)">
                编辑
              </FaButton>
              <FaButton variant="ghost" size="sm" class="text-red-500!" @click="() => { mcpServers = mcpServers.filter(x => x.name !== sv.name) }">
                移除
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
    <FaModal v-model="mcpVisible" :title="mcpForm.name ? `编辑 MCP：${mcpForm.name}` : '添加 MCP 服务器'" class="max-w-2xl!" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">名称</span>
          <FaInput v-model="mcpForm.name" placeholder="如 filesystem" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">传输类型</span>
          <select v-model="mcpForm.transport" class="h-9 flex-1 rounded-md border bg-background px-2 outline-none">
            <option value="stdio">stdio（本地子进程）</option>
            <option value="streamable-http">streamable-http（远程端点）</option>
          </select>
        </div>
        <div v-if="mcpForm.transport === 'stdio'" class="space-y-3">
          <div class="flex items-center gap-3">
            <span class="w-24 shrink-0 text-muted-foreground">启动命令</span>
            <FaInput v-model="mcpForm.command" placeholder="npx" class="flex-1" />
          </div>
          <div class="flex items-center gap-3">
            <span class="w-24 shrink-0 text-muted-foreground">参数</span>
            <FaInput v-model="mcpArgsStr" placeholder="-y @modelcontextprotocol/server-filesystem /tmp" class="flex-1" />
          </div>
        </div>
        <div v-else class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">端点 URL</span>
          <FaInput v-model="mcpForm.url" placeholder="http://127.0.0.1:3001/mcp" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">环境变量</span>
          <FaInput v-model="mcpArgsStr" placeholder="KEY=VALUE（暂不支持）" disabled class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="mcpVisible = false">
          取消
        </FaButton>
        <FaButton :loading="mcpSaving" @click="saveMCP">
          保存
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
