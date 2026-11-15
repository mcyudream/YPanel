<script setup lang="ts">
import type { AIProvider, AISkill } from '@/api/modules/ai'
import type { ChatEntry } from '@/store/modules/aiChat'
import { skillApi } from '@/api/modules/ai'
import { aiChatApi } from '@/api/modules/aiChat'
import ModelSelect from './ModelSelect.vue'

const props = defineProps<{
  entry: ChatEntry
  providers: AIProvider[]
}>()
const emit = defineEmits<{
  saved: [settings: import('@/api/modules/aiChat').ChatSettings, asDefault: boolean]
  compact: [
  ]
}>()
const visible = defineModel<boolean>({ default: false })
const form = ref(structuredClone(toRaw(props.entry.settings)))
const skills = ref<AISkill[]>([])
const asDefault = ref(false)
const saving = ref(false)
const error = ref('')
watch(visible, async (v) => {
  if (!v) {
    return
  }
  form.value = JSON.parse(JSON.stringify(props.entry.settings))
  error.value = ''
  try {
    skills.value = await skillApi.list()
  }
  catch {

  }
})
const temperatureSupported = computed(() => {
  const p = props.providers.find(p => p.id === form.value.providerId)
  try {
    return JSON.parse(p?.capabilities || '{}')[form.value.model]?.temperature === true
  }
  catch {
    return false
  }
})
function skill(name: string, checked: boolean) {
  form.value.skills = checked ? [...form.value.skills, name] : form.value.skills.filter(s => s !== name)
}
async function save() {
  saving.value = true
  try {
    if (!temperatureSupported.value) {
      delete form.value.temperature
    }
    if (props.entry.id > 0) {
      await aiChatApi.update(props.entry.id, { settings: form.value })
    }
    if (asDefault.value) {
      await aiChatApi.saveDefaults(form.value)
    }
    visible.value = false
    emit('saved', JSON.parse(JSON.stringify(form.value)), asDefault.value)
  }
  catch (e: any) {
    error.value = String(e?.message || e)
  }
  finally {
    saving.value = false
  }
}
</script>

<template>
  <FaModal v-model="visible" :title="$t('aiAssistant.settings')" class="max-w-3xl!" :footer="false">
    <div class="text-sm space-y-5">
      <ModelSelect v-model:provider-id="form.providerId" v-model:model="form.model" :providers="providers" />
      <div class="gap-4 grid grid-cols-1 sm:grid-cols-2">
        <label v-for="field in (['contextLimit', 'maxOutputTokens', 'retainTurns', 'maxRounds'] as const)" :key="field" class="space-y-1.5"><span class="text-xs text-muted-foreground">{{ $t(`aiAssistant.${{ contextLimit: 'window', maxOutputTokens: 'output', retainTurns: 'retain', maxRounds: 'rounds' }[field]}`) }}</span><FaInput v-model.number="form[field]" type="number" class="w-full" /></label>
      </div>
      <label class="flex gap-3 items-center justify-between"><span>{{ $t('aiAssistant.autoCompact') }}</span><FaSwitch v-model="form.autoCompact" /></label>
      <label class="block space-y-2"><span class="text-xs text-muted-foreground">{{ $t('aiAssistant.threshold') }} · {{ Math.round(form.compactThreshold * 100) }}%</span><input v-model.number="form.compactThreshold" type="range" min="0.5" max="0.95" step="0.05" class="accent-primary w-full block"></label>
      <div class="p-3 border rounded-xl bg-muted/20">
        <div class="text-xs font-medium mb-2">
          {{ $t('aiAssistant.summaryModel') }}
        </div>
        <ModelSelect v-model:provider-id="form.summaryProviderId" v-model:model="form.summaryModel" :providers="providers" />
        <FaButton
          size="sm" variant="ghost" @click="form.summaryProviderId = 0;
                                            form.summaryModel = ''"
        >
          {{ $t('common.clear') }}
        </FaButton>
      </div>
      <label class="block space-y-2"><span class="text-xs text-muted-foreground">{{ $t('aiAssistant.prompt') }}</span><textarea v-model="form.systemPrompt" rows="4" class="p-3 outline-none border rounded-xl bg-background w-full focus:border-primary/50" /></label>
      <label v-if="temperatureSupported" class="block space-y-2"><span class="text-xs text-muted-foreground">{{ $t('aiAssistant.temperature') }}</span><FaInput :model-value="form.temperature ?? ''" type="number" min="0" max="2" step="0.1" @update:model-value="v => form.temperature = v === '' ? undefined : Number(v)" /></label>
      <div class="gap-3 grid grid-cols-1 sm:grid-cols-3">
        <label v-for="field in (['memory', 'knowledge', 'tools'] as const)" :key="field" class="text-xs p-3 border rounded-xl flex gap-2 items-center justify-between"><span>{{ $t(`aiAssistant.${field}`) }}</span><FaSwitch v-model="form[field]" /></label>
      </div>
      <div v-if="skills.length" class="space-y-2">
        <div class="text-xs text-muted-foreground">
          {{ $t('aiAssistant.skills') }}
        </div><label v-for="s in skills.filter(s => s.enabled)" :key="s.name" class="p-3 border rounded-xl flex gap-3 items-center justify-between"><span class="min-w-0"><span>{{ s.name }}</span><span class="text-xs text-muted-foreground mt-1 block">{{ s.description }}</span></span><FaSwitch :model-value="form.skills.includes(s.name)" @update:model-value="v => skill(s.name, !!v)" /></label>
      </div>
      <div class="p-3 border rounded-xl bg-muted/20 space-y-3">
        <div class="flex items-center justify-between">
          <span class="font-medium">{{ $t('aiAssistant.context') }}</span><FaButton size="sm" variant="outline" :disabled="entry.id <= 0 || entry.status === 'running'" @click="emit('compact')">
            {{ $t('aiAssistant.compact') }}
          </FaButton>
        </div>
        <div v-if="entry.context?.tokens" class="text-xs text-muted-foreground gap-2 grid grid-cols-2">
          <span>{{ $t('aiAssistant.estimated') }}: {{ entry.context.tokens.toLocaleString() }}</span><span>{{ $t('aiAssistant.reserved') }}: {{ entry.context.outputReserve }}</span><span>{{ $t('aiAssistant.system') }}: {{ entry.context.system }}</span><span>{{ $t('aiAssistant.history') }}: {{ entry.context.history }}</span><span>{{ $t('aiAssistant.toolSchemas') }}: {{ entry.context.tools }}</span><span>{{ $t('aiAssistant.attachments') }}: {{ entry.context.attachments }}</span><span v-if="entry.context.inputTokens">{{ $t('aiAssistant.actual') }}: {{ entry.context.inputTokens }} / {{ entry.context.outputTokens }}</span>
        </div>
        <details>
          <summary class="text-xs text-muted-foreground cursor-pointer">
            {{ $t('aiAssistant.summary') }}
          </summary><pre class="text-xs leading-6 mt-2 max-h-64 whitespace-pre-wrap overflow-auto">{{ entry.context?.summary || $t('aiAssistant.noSummary') }}</pre>
        </details>
      </div>
      <label class="text-xs flex gap-3 items-center justify-between"><span>{{ $t('aiAssistant.saveDefault') }}</span><FaSwitch v-model="asDefault" /></label>
      <div v-if="error" class="text-xs text-red-500">
        {{ error }}
      </div>
      <div class="flex gap-2 justify-end">
        <FaButton variant="outline" @click="visible = false">
          {{ $t('common.cancel') }}
        </FaButton><FaButton :loading="saving" :disabled="entry.status === 'running'" @click="save">
          {{ $t('common.save') }}
        </FaButton>
      </div>
    </div>
  </FaModal>
</template>
