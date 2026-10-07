<script setup lang="ts">
// 系统工具（智能/系统工具）：内置工具表格（可排序）+ 启用/停用开关；停用后对话不可调用该工具。
import aiApi from '@/api/modules/ai'
import type { AiToolInfo } from '@/api/modules/ai'
import { toolMetaOf } from '@/components/YdAiChat/toolMeta'

defineOptions({
  name: 'aiTools',
})

const toast = useFaToast()

const tools = ref<AiToolInfo[]>([])
const savingName = ref('')

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
    toast.success(enabled ? `已启用：${toolMetaOf(t.name).label}` : `已停用：${toolMetaOf(t.name).label}`)
  }
  catch (e: any) {
    t.enabled = !enabled
    toast.error('操作失败', { description: e?.message })
  }
  finally {
    savingName.value = ''
  }
}

// ---- 排序 ----
const sortKey = ref<'label' | 'enabled'>('label')
const sortAsc = ref(true)

function toggleSort(key: 'label' | 'enabled') {
  if (sortKey.value === key) {
    sortAsc.value = !sortAsc.value
  }
  else {
    sortKey.value = key
    sortAsc.value = true
  }
}

function sortMark(key: 'label' | 'enabled') {
  if (sortKey.value !== key) {
    return 'i-lucide:chevrons-up-down'
  }
  return sortAsc.value ? 'i-lucide:chevron-up' : 'i-lucide:chevron-down'
}

const sortedTools = computed(() => {
  const arr = [...tools.value]
  arr.sort((a, b) => {
    if (sortKey.value === 'enabled') {
      return sortAsc.value ? Number(a.enabled) - Number(b.enabled) : Number(b.enabled) - Number(a.enabled)
    }
    return sortAsc.value
      ? toolMetaOf(a.name).label.localeCompare(toolMetaOf(b.name).label, 'zh')
      : toolMetaOf(b.name).label.localeCompare(toolMetaOf(a.name).label, 'zh')
  })
  return arr
})

const enabledCount = computed(() => tools.value.filter(t => t.enabled).length)

onMounted(loadTools)
onActivated(loadTools)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        系统工具
      </template>
      <template #description>
        AI 对话可调用的内置工具；停用后对话不再提供该工具（点击对话中的流程卡片可查看每次调用的入参与结果）
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="space-y-3">
        <div class="text-sm text-muted-foreground">
          共 {{ tools.length }} 个工具，已启用 <span class="font-medium text-foreground">{{ enabledCount }}</span> 个
        </div>
        <div class="overflow-x-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b bg-muted/40 text-left text-xs text-muted-foreground">
                <th class="cursor-pointer select-none px-3 py-2.5" @click="toggleSort('label')">
                  <span class="inline-flex items-center gap-1">
                    工具
                    <FaIcon :name="sortMark('label')" class="text-[11px]" :class="sortKey === 'label' ? 'text-foreground' : ''" />
                  </span>
                </th>
                <th class="px-3 py-2.5">
                  说明
                </th>
                <th class="cursor-pointer select-none px-3 py-2.5" @click="toggleSort('enabled')">
                  <span class="inline-flex items-center gap-1">
                    状态
                    <FaIcon :name="sortMark('enabled')" class="text-[11px]" :class="sortKey === 'enabled' ? 'text-foreground' : ''" />
                  </span>
                </th>
                <th class="w-24 px-3 py-2.5 text-right">
                  启用
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!sortedTools.length">
                <td colspan="4" class="px-3 py-8 text-center text-muted-foreground">
                  加载中…
                </td>
              </tr>
              <tr v-for="t in sortedTools" :key="t.name" class="border-b transition-colors last:border-b-0 hover:bg-accent/30">
                <td class="px-3 py-2.5">
                  <div class="flex items-center gap-2.5">
                    <span class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted">
                      <FaIcon :name="toolMetaOf(t.name).icon" class="text-sm text-muted-foreground" />
                    </span>
                    <div class="min-w-0">
                      <div class="flex items-center gap-1.5 font-medium">
                        {{ toolMetaOf(t.name).label }}
                      </div>
                      <div class="font-mono text-[11px] text-muted-foreground">
                        {{ t.name }}
                      </div>
                    </div>
                  </div>
                </td>
                <td class="max-w-0 px-3 py-2.5">
                  <div class="truncate text-muted-foreground" :title="toolMetaOf(t.name).desc || t.description">
                    {{ toolMetaOf(t.name).desc || t.description }}
                  </div>
                  <div class="truncate font-mono text-[11px] text-muted-foreground/70" :title="t.description">
                    {{ t.description }}
                  </div>
                </td>
                <td class="px-3 py-2.5">
                  <span
                    class="rounded-full px-2 py-0.5 text-xs"
                    :class="t.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
                  >
                    {{ t.enabled ? '已启用' : '已停用' }}
                  </span>
                </td>
                <td class="px-3 py-2.5 text-right">
                  <!-- FaSwitch=reka Switch：受控 prop 是 modelValue（v-model），checked 只是内部计算值 -->
                  <FaSwitch
                    v-model="t.enabled"
                    :disabled="savingName === t.name"
                    @update:model-value="(v: boolean) => setFlag(t, v)"
                  />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
