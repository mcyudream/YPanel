<script setup lang="ts">
// 系统工具（智能/系统工具，M31 治理化）：
// 模块分组工具表（风险徽标 + 开关）+ 确认模式切换（strict / danger_only）+ AI 操作审计。
import { i18n, tr } from '@/locales'
import aiApi from '@/api/modules/ai'
import type { AiOperationLog, AiToolInfo } from '@/api/modules/ai'
import { RISK_META, toolMetaOf } from '@/components/YdAiChat/toolMeta'

defineOptions({
  name: 'aiTools',
})

const toast = useFaToast()

const tools = ref<AiToolInfo[]>([])
const savingName = ref('')
const keyword = ref('')

async function loadTools() {
  try {
    tools.value = await aiApi.tools()
  }
  catch {}
}

async function setFlag(t: AiToolInfo, enabled: boolean) {
  savingName.value = t.name
  try {
    tools.value = await aiApi.setToolFlag(t.name, enabled)
    toast.success(i18n.global.t(enabled ? 'ai.tools.enabledTool' : 'ai.tools.disabledTool', { name: t.name }))
  }
  catch (e: any) {
    t.enabled = !enabled
    toast.error(i18n.global.t('ai.tools.operationFailed'), { description: e?.message })
  }
  finally {
    savingName.value = ''
  }
}

// ---- 确认模式（ask 机制）：strict = 写入+危险都确认；danger_only = 仅危险操作确认 ----
const askMode = ref<'strict' | 'danger_only'>('strict')
const askModeSaving = ref(false)

async function loadAskMode() {
  try {
    askMode.value = await aiApi.askMode() || 'strict'
  }
  catch {}
}

async function changeAskMode(mode: 'strict' | 'danger_only') {
  const prev = askMode.value
  askMode.value = mode
  askModeSaving.value = true
  try {
    await aiApi.setAskMode(mode)
    toast.success(i18n.global.t(mode === 'strict' ? 'ai.tools.switchedStrict' : 'ai.tools.switchedDangerOnly'))
  }
  catch (e: any) {
    askMode.value = prev
    toast.error(i18n.global.t('ai.tools.switchFailed'), { description: e?.message })
  }
  finally {
    askModeSaving.value = false
  }
}

// ---- 模块分组（后端已按模块返回，顺序即注册表顺序） ----
const filteredTools = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) {
    return tools.value
  }
  return tools.value.filter(t =>
    t.name.toLowerCase().includes(kw)
    || t.description.toLowerCase().includes(kw)
    || t.moduleTitle.toLowerCase().includes(kw),
  )
})

// ---- 分页（先切片再分组：页内仍按模块分组展示，组头计数用全量口径） ----
const page = ref(1)
const size = ref(20)
const pagedTools = computed(() =>
  filteredTools.value.slice((page.value - 1) * size.value, page.value * size.value),
)

watch(keyword, () => {
  page.value = 1
})

const moduleTotals = computed(() => {
  const m = new Map<string, number>()
  for (const t of filteredTools.value) {
    m.set(t.module, (m.get(t.module) || 0) + 1)
  }
  return m
})

const grouped = computed(() => {
  const map = new Map<string, AiToolInfo[]>()
  for (const t of pagedTools.value) {
    const key = `${t.moduleTitle}|${t.module}`
    if (!map.has(key)) {
      map.set(key, [])
    }
    map.get(key)!.push(t)
  }
  return [...map.entries()].map(([key, list]) => ({ key, title: key.split('|')[0], module: key.split('|')[1], list }))
})

const enabledCount = computed(() => tools.value.filter(t => t.enabled).length)
const riskCount = computed(() => tools.value.filter(t => t.risk === 'danger').length)

// ---- 操作审计 ----
const logs = ref<AiOperationLog[]>([])
const logTotal = ref(0)
const logPage = ref(1)
const logSize = ref(10)
const logLoading = ref(false)

const ACTION_META: Record<string, { text: string, class: string }> = {
  executed: { text: '自动执行', class: 'bg-sky-500/10 text-sky-600' },
  approved: { text: '已批准', class: 'bg-emerald-500/10 text-emerald-600' },
  denied: { text: '已拒绝', class: 'bg-red-500/10 text-red-500' },
  timeout: { text: '已超时', class: 'bg-red-500/10 text-red-500' },
}

async function loadLogs() {
  logLoading.value = true
  try {
    const res = await aiApi.operationLogs(logPage.value, logSize.value)
    logs.value = res.items || []
    logTotal.value = res.total || 0
  }
  catch {}
  finally {
    logLoading.value = false
  }
}

watch(logPage, loadLogs)

function fmtTime(v: string) {
  return v ? new Date(v).toLocaleString('zh-CN', { hour12: false }) : '-'
}

onMounted(() => {
  loadTools()
  loadAskMode()
  loadLogs()
})
onActivated(() => {
  loadTools()
  loadAskMode()
})
</script>

<template>
  <div>
    <FaPageMain>
      <div class="space-y-4">
        <!-- 顶部：确认模式 + 统计 -->
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex items-center gap-2 text-sm">
            <span class="text-muted-foreground">{{ $t('ai.tools.askMode') }}</span>
            <YdSelect
              v-model="askMode"
              :options="[
                { value: 'strict', label: $t('ai.tools.strictLabel') },
                { value: 'danger_only', label: $t('ai.tools.dangerOnlyLabel') },
              ]"
              button-class="min-w-56"
              @update:model-value="v => changeAskMode(v as 'strict' | 'danger_only')"
            />
          </div>
          <div class="ml-auto text-xs text-muted-foreground">
            {{ $t('ai.tools.totalTools', { n: tools.length }) }} · {{ $t('ai.tools.enabledPrefix') }} <span class="font-medium text-foreground">{{ enabledCount }}</span> {{ $t('ai.tools.enabledSuffix') }} · {{ $t('ai.tools.dangerTools', { n: riskCount }) }}
          </div>
          <FaInput v-model="keyword" :placeholder="$t('ai.tools.searchPlaceholder')" class="w-52" />
        </div>

        <!-- 工具表：按模块分组 -->
        <div class="overflow-hidden rounded-lg border">
          <div v-if="!grouped.length" class="px-3 py-8 text-center text-sm text-muted-foreground">
            {{ $t('ai.tools.noMatch') }}
          </div>
          <div v-for="g in grouped" :key="g.key">
            <div class="flex items-center gap-2 border-b bg-muted/40 px-3 py-2 text-xs font-medium text-muted-foreground">
              <FaIcon name="i-lucide:layout-grid" class="text-[11px]" />
              {{ g.title }}
              <span class="font-mono text-[10px] text-muted-foreground/60">{{ g.module }}</span>
              <span class="ml-auto">{{ $t('ai.tools.toolCount', { n: moduleTotals.get(g.module) }) }}</span>
            </div>
            <table class="w-full text-sm">
              <tbody>
                <tr v-for="t in g.list" :key="t.name" class="border-b transition-colors last:border-b-0 hover:bg-accent/30">
                  <td class="w-56 px-3 py-2">
                    <div class="flex items-center gap-2.5">
                      <span class="flex size-7 shrink-0 items-center justify-center rounded-lg bg-muted">
                        <FaIcon :name="toolMetaOf(t.name).icon" class="text-xs text-muted-foreground" />
                      </span>
                      <div class="min-w-0">
                        <div class="truncate font-mono text-xs font-medium">
                          {{ t.name }}
                        </div>
                      </div>
                    </div>
                  </td>
                  <td class="max-w-0 px-3 py-2">
                    <div class="truncate text-muted-foreground" :title="t.description">
                      {{ t.description }}
                    </div>
                  </td>
                  <td class="w-16 px-3 py-2">
                    <span
                      class="rounded-full px-2 py-0.5 text-[11px]"
                      :class="(RISK_META[t.risk] || RISK_META.read).badgeClass"
                    >
                      {{ (RISK_META[t.risk] || RISK_META.read).label }}
                    </span>
                  </td>
                  <td class="w-20 px-3 py-2 text-right">
                    <FaSwitch
                      v-model="t.enabled"
                      :disabled="savingName === t.name"
                      @update:model-value="(v: boolean | undefined) => setFlag(t, v === true)"
                    />
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <FaPagination v-model:page="page" v-model:size="size" :total="filteredTools.length" />
      </div>
    </FaPageMain>

    <!-- 操作审计 -->
    <FaPageMain>
      <div class="space-y-3">
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium">{{ $t('ai.tools.auditTitle') }}</span>
          <span class="text-xs text-muted-foreground">{{ $t('ai.tools.auditDesc') }}</span>
          <FaButton size="icon-sm" variant="ghost" class="ml-auto" @click="loadLogs">
            <FaIcon name="i-lucide:refresh-cw" :class="logLoading ? 'animate-spin' : ''" />
          </FaButton>
        </div>
        <div class="overflow-x-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b bg-muted/40 text-left text-xs text-muted-foreground">
                <th class="w-40 px-3 py-2">
                  {{ $t('common.time') }}
                </th>
                <th class="w-48 px-3 py-2">
                  {{ $t('ai.tools.colTool') }}
                </th>
                <th class="w-24 px-3 py-2">
                  {{ $t('ai.tools.colRisk') }}
                </th>
                <th class="w-24 px-3 py-2">
                  {{ $t('ai.tools.colAction') }}
                </th>
                <th class="px-3 py-2">
                  {{ $t('ai.tools.colArgsResult') }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!logs.length">
                <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
                  {{ logLoading ? $t('common.loading') : $t('ai.tools.noLogs') }}
                </td>
              </tr>
              <tr v-for="log in logs" :key="log.id" class="border-b transition-colors last:border-b-0 hover:bg-accent/30">
                <td class="px-3 py-2 font-mono text-xs text-muted-foreground">
                  {{ fmtTime(log.createdAt) }}
                </td>
                <td class="px-3 py-2">
                  <div class="truncate font-mono text-xs font-medium" :title="log.tool">
                    {{ log.tool }}
                  </div>
                  <div class="text-[11px] text-muted-foreground">
                    {{ log.module }}
                  </div>
                </td>
                <td class="px-3 py-2">
                  <span class="rounded-full px-2 py-0.5 text-[11px]" :class="(RISK_META[log.risk] || RISK_META.read).badgeClass">
                    {{ (RISK_META[log.risk] || RISK_META.read).label }}
                  </span>
                </td>
                <td class="px-3 py-2">
                  <span
                    class="rounded-full px-2 py-0.5 text-[11px]"
                    :class="(ACTION_META[log.action] || ACTION_META.executed).class"
                  >
                    {{ tr(`ai.tools.actionState.${log.action || 'executed'}`, (ACTION_META[log.action] || ACTION_META.executed).text) }}
                  </span>
                </td>
                <td class="max-w-0 px-3 py-2">
                  <div class="truncate font-mono text-[11px] text-muted-foreground" :title="log.args">
                    {{ log.args || '-' }}
                  </div>
                  <div class="truncate text-[11px]" :class="log.success ? 'text-muted-foreground' : 'text-red-500'" :title="log.result">
                    {{ log.result || (log.success ? $t('common.success') : $t('common.failed')) }}
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <FaPagination v-model:page="logPage" v-model:size="logSize" :total="logTotal" />
      </div>
    </FaPageMain>
  </div>
</template>
