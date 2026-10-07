<script setup lang="ts">
// AI 供应商（智能/供应商）：LobeHub 风格预设网格 + 已配置列表 + 编辑弹窗。
import type { AIProvider } from '@/api/modules/ai'
import aiApi from '@/api/modules/ai'
import { providerPresets } from '@/api/modules/ai'

defineOptions({
  name: 'aiProviders',
})

const toast = useFaToast()

const providers = ref<AIProvider[]>([])
const provVisible = ref(false)
const provSaving = ref(false)
const provForm = ref<AIProvider>({ id: 0, name: '', apiType: 'openai', baseURL: '', apiKey: '', model: '', isDefault: false })

function iconFor(name: string): string | undefined {
  return providerPresets.find(p => p.name === name || name.includes(p.name.split(' ')[0]))?.icon
}

function openProvFromPreset(preset: AIProvider) {
  const existing = providers.value.find(x => x.name === preset.name)
  if (existing) {
    openProv(existing)
    return
  }
  provForm.value = { id: 0, name: preset.name, apiType: preset.apiType, baseURL: preset.baseURL, apiKey: '', model: preset.model, isDefault: providers.value.length === 0 }
  provVisible.value = true
}

function openProv(p?: AIProvider) {
  provForm.value = p ? { ...p, apiKey: '' } : { id: 0, name: '', apiType: 'openai', baseURL: '', apiKey: '', model: '', isDefault: false }
  provVisible.value = true
}

function applyPreset(preset: AIProvider) {
  provForm.value = { ...provForm.value, name: preset.name, apiType: preset.apiType, baseURL: preset.baseURL, model: preset.model }
}

async function saveProv() {
  provSaving.value = true
  try {
    providers.value = ((await aiApi.saveProvider(provForm.value)).data as any) || []
    provVisible.value = false
    toast.success('供应商已保存')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    provSaving.value = false
  }
}

async function removeProv(id: number) {
  try {
    await aiApi.removeProvider(id)
    providers.value = providers.value.filter(p => p.id !== id)
    toast.success('已删除')
  }
  catch (e: any) {
    toast.error('删除失败', { description: e?.message })
  }
}

async function loadProviders() {
  try {
    providers.value = await aiApi.providers()
  }
  catch {}
}

onMounted(loadProviders)
onActivated(loadProviders)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        AI 供应商
      </template>
      <template #description>
        内置多家预设与自定义接入，支持 OpenAI 兼容（智谱/DeepSeek/Ollama 等）与 Anthropic
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="space-y-5">
        <!-- 已配置 -->
        <div v-if="providers.length" class="space-y-2">
          <div class="text-sm font-medium">
            已配置（{{ providers.length }}）
          </div>
          <div v-for="p in providers" :key="p.id" class="flex items-center justify-between rounded-lg border p-3">
            <div class="flex items-center gap-3">
              <img v-if="iconFor(p.name)" :src="iconFor(p.name)" class="size-6">
              <FaIcon v-else name="i-lucide:box" class="size-5 text-muted-foreground" />
              <div>
                <div class="flex items-center gap-2 text-sm font-medium">
                  {{ p.name }}
                  <span v-if="p.isDefault" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">默认</span>
                </div>
                <div class="font-mono text-xs text-muted-foreground">
                  {{ p.baseURL }} · {{ p.model }}
                </div>
              </div>
            </div>
            <div class="flex gap-1">
              <FaButton variant="ghost" size="sm" @click="openProv(p)">
                编辑
              </FaButton>
              <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeProv(p.id)">
                删除
              </FaButton>
            </div>
          </div>
        </div>

        <!-- 可添加的供应商（LobeHub 风格网格卡片） -->
        <div class="space-y-2">
          <div class="text-sm font-medium">
            添加供应商
          </div>
          <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            <button
              v-for="preset in providerPresets"
              :key="preset.name"
              type="button"
              class="flex cursor-pointer items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:border-primary hover:bg-accent/30"
              :class="providers.some(x => x.name === preset.name) ? 'op-50' : ''"
              @click="openProvFromPreset(preset)"
            >
              <img v-if="preset.icon" :src="preset.icon" class="size-7 shrink-0">
              <span v-else class="flex size-7 items-center justify-center rounded bg-muted">
                <FaIcon name="i-lucide:box" class="text-sm" />
              </span>
              <span class="min-w-0">
                <span class="block truncate text-sm">{{ preset.name }}</span>
                <span class="block truncate font-mono text-[11px] text-muted-foreground">{{ preset.model }}</span>
              </span>
              <span v-if="providers.some(x => x.name === preset.name)" class="ml-auto shrink-0 rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[10px] text-emerald-600">
                已配置
              </span>
            </button>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 供应商弹窗 -->
    <FaModal v-model="provVisible" title="AI 供应商" class="max-w-2xl!" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="mr-1 text-xs text-muted-foreground">内置预设</span>
          <button
            v-for="p in providerPresets"
            :key="p.name"
            type="button"
            class="cursor-pointer rounded-full border px-2.5 py-0.5 text-xs text-muted-foreground hover:bg-accent/50"
            @click="applyPreset(p)"
          >
            {{ p.name }}
          </button>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">名称</span>
          <FaInput v-model="provForm.name" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">API 类型</span>
          <select v-model="provForm.apiType" class="h-9 flex-1 rounded-md border bg-background px-2 outline-none">
            <option value="openai">OpenAI Chat Completions（兼容智谱/DeepSeek/Ollama 等）</option>
            <option value="anthropic">Anthropic Messages</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">Base URL</span>
          <FaInput v-model="provForm.baseURL" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">模型</span>
          <FaInput v-model="provForm.model" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">API Key</span>
          <FaInput v-model="provForm.apiKey" type="password" :placeholder="provForm.id ? '留空不修改' : ''" class="flex-1" />
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="provForm.isDefault" type="checkbox"> 设为默认
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="provVisible = false">
          取消
        </FaButton>
        <FaButton :loading="provSaving" @click="saveProv">
          保存
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
