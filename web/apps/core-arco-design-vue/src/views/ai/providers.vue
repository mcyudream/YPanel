<script setup lang="ts">
import type { AIProvider } from '@/api/modules/ai'
import aiApi, { providerPresets } from '@/api/modules/ai'
// AI 供应商（智能/供应商）：LobeHub 风格预设网格 + 已配置列表 + 编辑弹窗。
import { i18n } from '@/locales'

defineOptions({
  name: 'AiProviders',
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
    allowPrivate: preset.allowPrivate,
    capabilities: preset.capabilities,
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
  provForm.value = { ...provForm.value, name: preset.name, apiType: preset.apiType, baseURL: preset.baseURL, model: preset.model, capabilities: preset.capabilities, allowPrivate: preset.allowPrivate }
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
  catch {

  }
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
          <div v-for="p in providers" :key="p.id" class="p-3 border rounded-lg flex items-center justify-between">
            <div class="flex gap-3 items-center">
              <img v-if="iconFor(p.name)" :src="iconFor(p.name)" class="size-6">
              <FaIcon v-else name="i-lucide:box" class="text-muted-foreground size-5" />
              <div>
                <div class="text-sm font-medium flex gap-2 items-center">
                  {{ p.name }}
                  <span v-if="p.isDefault" class="text-xs text-emerald-600 px-2 py-0.5 rounded-full bg-emerald-500/10">{{ $t('common.default') }}</span>
                </div>
                <div class="text-xs text-muted-foreground font-mono">
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
          <div class="gap-2 grid lg:grid-cols-3 sm:grid-cols-2">
            <button
              type="button"
              class="p-3 text-left border rounded-lg border-dashed flex gap-3 cursor-pointer transition-colors items-center hover:border-primary hover:bg-accent/30"
              @click="openProv()"
            >
              <span class="rounded bg-muted flex size-7 items-center justify-center">
                <FaIcon name="i-lucide:plus" class="text-sm" />
              </span>
              <span class="min-w-0">
                <span class="text-sm block">{{ $t('ai.providers.customProvider') }}</span>
                <span class="text-[11px] text-muted-foreground block truncate">{{ $t('ai.providers.customProviderDesc') }}</span>
              </span>
            </button>
            <button
              v-for="preset in providerPresets"
              :key="preset.name"
              type="button"
              class="p-3 text-left border rounded-lg flex gap-3 cursor-pointer transition-colors items-center hover:border-primary hover:bg-accent/30"
              @click="openProvFromPreset(preset)"
            >
              <img v-if="preset.icon" :src="preset.icon" class="shrink-0 size-7">
              <span v-else class="rounded bg-muted flex size-7 items-center justify-center">
                <FaIcon name="i-lucide:box" class="text-sm" />
              </span>
              <span class="min-w-0">
                <span class="text-sm block truncate">{{ preset.name }}</span>
                <span class="text-[11px] text-muted-foreground font-mono block truncate">{{ preset.model }}</span>
              </span>
              <span v-if="presetCount(preset.name)" class="text-[10px] text-emerald-600 ml-auto px-1.5 py-0.5 rounded-full bg-emerald-500/10 shrink-0">
                {{ $t('ai.providers.configuredCount', { n: presetCount(preset.name) }) }}
              </span>
            </button>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 供应商弹窗 -->
    <FaModal v-model="provVisible" :title="$t('ai.providers.modalTitle')" class="max-w-2xl!" :destroy-on-close="true">
      <div class="text-sm space-y-3">
        <div class="flex flex-wrap gap-1.5 items-center">
          <span class="text-xs text-muted-foreground mr-1">{{ $t('ai.providers.builtinPresets') }}</span>
          <button
            v-for="p in providerPresets"
            :key="p.name"
            type="button"
            class="text-xs text-muted-foreground px-2.5 py-0.5 border rounded-full cursor-pointer hover:bg-accent/50"
            @click="applyPreset(p)"
          >
            {{ p.name }}
          </button>
        </div>
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-24">{{ $t('common.name') }}</span>
          <FaInput v-model="provForm.name" class="flex-1" />
        </div>
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-24">{{ $t('ai.providers.apiType') }}</span>
          <YdSelect
            v-model="provForm.apiType"
            :options="[
              { label: $t('ai.providers.apiTypeOpenai'), value: 'openai' },
              { label: $t('ai.providers.apiTypeAnthropic'), value: 'anthropic' },
              { label: 'OpenAI Responses', value: 'responses' },
            ]"
            size="default"
            button-class="flex-1"
          />
        </div>
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-24">{{ $t('ai.providers.baseURL') }}</span>
          <FaInput v-model="provForm.baseURL" class="flex-1" />
        </div>
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-24">{{ $t('ai.providers.defaultModel') }}</span>
          <FaInput v-model="provForm.model" class="flex-1" />
        </div>
        <div class="flex gap-3 items-start">
          <span class="text-muted-foreground pt-1.5 shrink-0 w-24">{{ $t('ai.providers.availableModels') }}</span>
          <div class="flex-1 min-w-0 space-y-1">
            <FaInput
              v-model="provForm.models"
              :placeholder="$t('ai.providers.modelsPlaceholder')"
            />
            <div class="text-xs text-muted-foreground flex gap-2 items-center">
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
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-24">{{ $t('ai.providers.apiKey') }}</span>
          <FaInput v-model="provForm.apiKey" type="password" :placeholder="provForm.id ? $t('ai.providers.apiKeyPlaceholder') : ''" class="flex-1" />
        </div>
        <label class="text-sm flex gap-2 items-center justify-between"><span>{{ $t('aiAssistant.privateEndpoint') }}</span><FaSwitch v-model="provForm.allowPrivate" /></label>
        <div class="space-y-2">
          <label class="text-sm">{{ $t('aiAssistant.modelCapabilities') }}</label><textarea v-model="provForm.capabilities" rows="6" placeholder="{&quot;model-name&quot;:{&quot;vision&quot;:true,&quot;tools&quot;:true,&quot;contextLimit&quot;:32768,&quot;maxOutputTokens&quot;:4096,&quot;temperature&quot;:true}}" class="text-xs font-mono p-3 outline-none border rounded-lg bg-background w-full focus:border-primary/50" /><p class="text-xs text-muted-foreground">
            {{ $t('aiAssistant.capabilityHelp') }}
          </p>
        </div>
        <label class="text-sm flex gap-2 cursor-pointer items-center">
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
