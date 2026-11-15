<script setup lang="ts">
import type { ChatSegment } from '@/api/modules/aiChat'
import aiApi from '@/api/modules/ai'

const props = defineProps<{
  segment: ChatSegment
}>()
const emit = defineEmits<{
  resolved: [
  ]
}>()
const forms = ref<Record<number, {
  selected: string[]
  text: string
}>>({})
const busy = ref(false)
const error = ref('')
watch(() => props.segment.askId, () => {
  forms.value = {}
  error.value = ''
})
function form(i: number) {
  return forms.value[i] ||= { selected: [], text: '' }
}
function toggle(i: number, label: string, multi?: boolean) {
  const f = form(i)
  f.selected = multi ? f.selected.includes(label) ? f.selected.filter(v => v !== label) : [...f.selected, label] : [label]
}
const ready = computed(() => (props.segment.questions || []).every((_, i) => form(i).selected.length || form(i).text.trim()))
async function resolve(approved: boolean) {
  if (!props.segment.askId) {
    return
  }
  busy.value = true
  error.value = ''
  try {
    if (props.segment.type === 'question') {
      await aiApi.resolveAskQuestion(props.segment.askId, approved ? (props.segment.questions || []).map((q, i) => ({ question: q.question, header: q.header, ...form(i) })) : [])
    }
    else {
      await aiApi.resolveAsk(props.segment.askId, approved)
    }
    emit('resolved')
  }
  catch (e: any) {
    error.value = String(e?.message || e)
  }
  finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="mx-auto p-4 border border-amber-500/40 rounded-2xl bg-amber-500/5 max-w-[920px] w-full">
    <div class="text-sm font-medium mb-3 flex gap-2 items-center">
      <FaIcon :name="segment.type === 'question' ? 'i-lucide:circle-help' : 'i-lucide:shield-alert'" class="text-amber-600" />{{ $t(segment.type === 'question' ? 'aiAssistant.answer' : 'aiAssistant.confirm') }}<span class="text-xs text-muted-foreground font-mono">{{ segment.name }}</span>
    </div>
    <template v-if="segment.type === 'question'">
      <div v-for="(q, i) in segment.questions" :key="i" class="mb-3 space-y-2">
        <div class="text-sm">
          {{ q.question }}
        </div><div class="flex flex-wrap gap-2">
          <FaButton v-for="o in q.options" :key="o.label" size="sm" :variant="form(i).selected.includes(o.label) ? 'default' : 'outline'" :title="o.description" @click="toggle(i, o.label, q.multiSelect)">
            {{ o.label }}
          </FaButton>
        </div><FaInput v-model="form(i).text" class="w-full" :placeholder="$t('components.ydAiChat.supplementPlaceholder')" />
      </div>
    </template>
    <template v-else>
      <p v-if="segment.name?.startsWith('mcp_')" class="text-xs text-muted-foreground mb-2">
        {{ $t('aiAssistant.unknownRisk') }}
      </p><pre class="text-xs p-3 rounded-lg bg-muted/60 max-h-40 whitespace-pre-wrap break-all overflow-auto">{{ segment.args }}</pre>
    </template>
    <div v-if="error" class="text-xs text-red-500 mt-2">
      {{ error }}
    </div>
    <div class="mt-3 flex gap-2 justify-end">
      <FaButton variant="outline" :disabled="busy" @click="resolve(false)">
        {{ $t('aiAssistant.reject') }}
      </FaButton><FaButton :variant="segment.risk === 'danger' ? 'destructive' : 'default'" :loading="busy" :disabled="segment.type === 'question' && !ready" @click="resolve(true)">
        {{ $t(segment.type === 'question' ? 'aiAssistant.send' : 'aiAssistant.approve') }}
      </FaButton>
    </div>
  </div>
</template>
