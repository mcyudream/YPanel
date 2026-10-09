<script setup lang="ts">
// AI 供应商（智能/供应商）：LobeHub 风格预设网格 + 已配置列表 + 编辑弹窗。
import { i18n } from '@/locales'
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
const provForm = ref<AIProvider>({ id: 0, name: '', apiType: 'openai', baseURL: '', apiKey: '', model: '', models: '', isDefault: false })
const fetchingModels = ref(false)

async function fetchModels() {
  if (!provForm.value.id) {
    toast.warning(i18n.global.t('ai.providers.saveBeforeFetch'))
    return
  }
  fetchingModels.value = true
  try {
    const list = await aiApi.providerModels(provForm.value.id)
    if (!list.length) {
      toast.warning(i18n.global.t('ai.providers.noModelsReturned'))
      return
    }
    provForm.value.models = list.join(',')
    if (!provForm.value.model || !list.includes(provForm.value.model)) {
      provForm.value.model = list[0]
    }
    toast.success(i18n.global.t('ai.providers.fetchedModels', { n: list.length }))
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.providers.fetchFailed'), { description: e?.message })
  }
  finally {
    fetchingModels.value = false
  }
}

function modelsFor(p: AIProvider): string[] {
  const list = (p.models || '').split(',').map(m => m.trim()).filter(Boolean)
  if (!list.includes(p.model)) {
    list.unshift(p.model)
  }
  return list
}

function iconFor(name: string): string | undefined {
  return providerPresets.find(p => p.name === name || name.includes(p.name.split(' ')[0]))?.icon
}

// 同一供应商支持多实例（不同 key/别名）：预设卡每次都开新建表单，同名自动编号，用户可改
function openProvFromPreset(preset: AIProvider) {
  const count = providers.value.filter(x => x.name === preset.name).length
  provForm.value = {
    id: 0,
    name: count ? `${preset.name}-${count + 1}` : preset.name,
    apiType: preset.apiType,
    baseURL: preset.baseURL,
    apiKey: '',
    model: preset.model,
    models: '',
    isDefault: providers.value.length === 0,
  }
  provVisible.value = true
}

function presetCount(name: string) {
  return providers.value.filter(x => x.name === name).length
}

function openProv(p?: AIProvider) {
  provForm.value = p ? { ...p, apiKey: '' } : { id: 0, name: '', apiType: 'openai', baseURL: '', apiKey: '', model: '', models: '', isDefault: false }
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
    toast.success(i18n.global.t('ai.providers.saved'))
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.providers.saveFailed'), { description: e?.message })
  }
  finally {
    provSaving.value = false
  }
}

async function removeProv(id: number) {
  try {
    await aiApi.removeProvider(id)
    providers.value = providers.value.filter(p => p.id !== id)
    toast.success(i18n.global.t('ai.providers.deleted'))
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.providers.deleteFailed'), { description: e?.message })
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
    <FaPageMain>
      <div class="space-y-5">
        <!-- 已配置 -->
        <div v-if="providers.length" class="space-y-2">
          <div class="text-sm font-medium">
            {{ $t('ai.providers.configured', { n: providers.length }) }}
          </div>
          <div v-for="p in providers" :key="p.id" class="flex items-center justify-between rounded-lg border p-3">
            <div class="flex items-center gap-3">
              <img v-if="iconFor(p.name)" :src="iconFor(p.name)" class="size-6">
              <FaIcon v-else name="i-lucide:box" class="size-5 text-muted-foreground" />
              <div>
                <div class="flex items-center gap-2 text-sm font-medium">
                  {{ p.name }}
                  <span v-if="p.isDefault" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">{{ $t('common.default') }}</span>
                </div>
                <div class="font-mono text-xs text-muted-foreground">
                  {{ p.baseURL }} · {{ modelsFor(p).join(' / ') }}
                </div>
              </div>
            </div>
            <div class="flex gap-1">
              <FaButton variant="ghost" size="sm" @click="openProv(p)">
                {{ $t('common.edit') }}
              </FaButton>
              <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeProv(p.id)">
                {{ $t('common.delete') }}
              </FaButton>
            </div>
          </div>
        </div>

        <!-- 可添加的供应商（LobeHub 风格网格卡片） -->
        <div class="space-y-2">
          <div class="text-sm font-medium">
            {{ $t('ai.providers.addProvider') }}
          </div>
          <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            <button
              type="button"
              class="flex cursor-pointer items-center gap-3 rounded-lg border border-dashed p-3 text-left transition-colors hover:border-primary hover:bg-accent/30"
              @click="openProv()"
            >
              <span class="flex size-7 items-center justify-center rounded bg-muted">
                <FaIcon name="i-lucide:plus" class="text-sm" />
              </span>
              <span class="min-w-0">
                <span class="block text-sm">{{ $t('ai.providers.customProvider') }}</span>
                <span class="block truncate text-[11px] text-muted-foreground">{{ $t('ai.providers.customProviderDesc') }}</span>
              </span>
            </button>
            <button
              v-for="preset in providerPresets"
              :key="preset.name"
              type="button"
              class="flex cursor-pointer items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:border-primary hover:bg-accent/30"
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
              <span v-if="presetCount(preset.name)" class="ml-auto shrink-0 rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[10px] text-emerald-600">
                {{ $t('ai.providers.configuredCount', { n: presetCount(preset.name) }) }}
              </span>
            </button>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 供应商弹窗 -->
    <FaModal v-model="provVisible" :title="$t('ai.providers.modalTitle')" class="max-w-2xl!" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="mr-1 text-xs text-muted-foreground">{{ $t('ai.providers.builtinPresets') }}</span>
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
          <span class="w-24 shrink-0 text-muted-foreground">{{ $t('common.name') }}</span>
          <FaInput v-model="provForm.name" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">{{ $t('ai.providers.apiType') }}</span>
          <select v-model="provForm.apiType" class="h-9 flex-1 rounded-md border bg-background px-2 outline-none">
            <option value="openai">{{ $t('ai.providers.apiTypeOpenai') }}</option>
            <option value="anthropic">{{ $t('ai.providers.apiTypeAnthropic') }}</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">{{ $t('ai.providers.baseURL') }}</span>
          <FaInput v-model="provForm.baseURL" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">{{ $t('ai.providers.defaultModel') }}</span>
          <FaInput v-model="provForm.model" class="flex-1" />
        </div>
        <div class="flex items-start gap-3">
          <span class="w-24 shrink-0 pt-1.5 text-muted-foreground">{{ $t('ai.providers.availableModels') }}</span>
          <div class="min-w-0 flex-1 space-y-1">
            <FaInput
              v-model="provForm.models"
              :placeholder="$t('ai.providers.modelsPlaceholder')"
            />
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
              <span>{{ $t('ai.providers.modelsHint') }}</span>
              <FaButton
                variant="ghost"
                size="sm"
                :loading="fetchingModels"
                :title="provForm.id ? $t('ai.providers.fetchTitle') : $t('ai.providers.fetchTitleNeedSave')"
                @click="fetchModels"
              >
                <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('ai.providers.fetchFromApi') }}
              </FaButton>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">{{ $t('ai.providers.apiKey') }}</span>
          <FaInput v-model="provForm.apiKey" type="password" :placeholder="provForm.id ? $t('ai.providers.apiKeyPlaceholder') : ''" class="flex-1" />
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="provForm.isDefault" type="checkbox"> {{ $t('ai.providers.setDefault') }}
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="provVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="provSaving" @click="saveProv">
          {{ $t('common.save') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
