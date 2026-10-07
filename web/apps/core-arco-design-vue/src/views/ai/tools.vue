<script setup lang="ts">
// 系统工具（智能/系统工具）：内置工具清单 + 启用/停用开关；停用后对话不可调用该工具。
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
  const prev = t.enabled
  t.enabled = enabled
  try {
    tools.value = await aiApi.setToolFlag(t.name, enabled)
    toast.success(enabled ? `已启用：${toolMetaOf(t.name).label}` : `已停用：${toolMetaOf(t.name).label}`)
  }
  catch (e: any) {
    t.enabled = prev
    toast.error('操作失败', { description: e?.message })
  }
  finally {
    savingName.value = ''
  }
}

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
        AI 对话可调用的内置工具；停用后对话不再提供该工具（点击流程卡片可查看每次调用的入参与结果）
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="space-y-4">
        <div class="text-sm text-muted-foreground">
          共 {{ tools.length }} 个工具，已启用 <span class="font-medium text-foreground">{{ enabledCount }}</span> 个
        </div>
        <div v-if="!tools.length" class="rounded-lg border p-8 text-center text-sm text-muted-foreground">
          加载中…
        </div>
        <div v-for="t in tools" :key="t.name" class="flex items-center justify-between rounded-lg border p-4">
          <div class="flex min-w-0 items-start gap-3">
            <span class="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted">
              <FaIcon :name="toolMetaOf(t.name).icon" class="text-base text-muted-foreground" />
            </span>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2 text-sm font-medium">
                {{ toolMetaOf(t.name).label }}
                <span class="rounded bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">{{ t.name }}</span>
                <span
                  class="rounded-full px-2 py-0.5 text-xs"
                  :class="t.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
                >
                  {{ t.enabled ? '已启用' : '已停用' }}
                </span>
              </div>
              <p class="mt-1 text-xs text-muted-foreground">
                {{ toolMetaOf(t.name).desc || t.description }}
              </p>
              <p class="mt-0.5 truncate font-mono text-[11px] text-muted-foreground/70" :title="t.description">
                {{ t.description }}
              </p>
            </div>
          </div>
          <!-- FaSwitch 基于 reka-ui Switch：受控 prop 是 checked（非 modelValue），事件是 update:checked -->
          <FaSwitch
            :checked="t.enabled"
            :disabled="savingName === t.name"
            @update:checked="(v: boolean) => setFlag(t, !!v)"
          />
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
