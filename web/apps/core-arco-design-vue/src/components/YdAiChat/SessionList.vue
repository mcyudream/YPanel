<script setup lang="ts">
import type { ConversationMeta } from '@/api/modules/aiChat'
import { useI18n } from 'vue-i18n'

export type AiSessionItem = ConversationMeta
const props = defineProps<{
  sessions: ConversationMeta[]
  activeId?: number
  loading?: boolean
  more?: boolean
  archived?: boolean
}>()
const emit = defineEmits<{
  select: [
    number,
  ]
  create: [
  ]
  search: [
    string,
  ]
  more: [
  ]
  archiveView: [
  ]
  rename: [
    ConversationMeta,
    string,
  ]
  remove: [
    number,
  ]
  patch: [
    number,
    {
      pinned?: boolean
      archived?: boolean
    },
  ]
}>()
const keyword = ref('')
const editing = ref<ConversationMeta>()
const title = ref('')
const { t } = useI18n()
let debounce: ReturnType<typeof setTimeout>
watch(keyword, (v) => {
  clearTimeout(debounce)
  debounce = setTimeout(emit, 250, 'search', v)
})
onBeforeUnmount(() => clearTimeout(debounce))
function menu(s: ConversationMeta) {
  return [[
    { label: t('common.rename'), handle: () => {
      editing.value = s
      title.value = s.title
    } },
    { label: t(s.pinned ? 'aiAssistant.unpin' : 'aiAssistant.pin'), handle: () => emit('patch', s.id, { pinned: !s.pinned }) },
    { label: t(s.archived ? 'aiAssistant.restore' : 'aiAssistant.archive'), handle: () => emit('patch', s.id, { archived: !s.archived }) },
    { label: t('common.delete'), handle: () => emit('remove', s.id) },
  ]]
}
function group(s: ConversationMeta) {
  return s.pinned ? t('aiAssistant.pin') : new Date(s.updatedAt).toDateString() === new Date().toDateString() ? t('aiAssistant.today') : t('aiAssistant.previous')
}
</script>

<template>
  <div class="flex flex-col h-full min-h-0">
    <div class="p-3 gap-3 grid">
      <FaButton class="w-full" @click="emit('create')">
        <FaIcon name="i-lucide:plus" class="mr-2" />{{ $t('aiAssistant.new') }}
      </FaButton>
      <FaInput v-model="keyword" class="w-full" :placeholder="$t('aiAssistant.search')">
        <template #start>
          <FaIcon name="i-lucide:search" />
        </template>
      </FaInput>
      <FaButton size="sm" variant="ghost" class="text-muted-foreground justify-start" @click="emit('archiveView')">
        <FaIcon name="i-lucide:archive" class="mr-2" />{{ $t(archived ? 'aiAssistant.archived' : 'aiAssistant.recent') }}
      </FaButton>
    </div>
    <div class="px-2 pb-3 flex-1 min-h-0 overflow-y-auto">
      <div v-if="!sessions.length" class="text-xs text-muted-foreground p-5 text-center">
        {{ loading ? $t('common.loading') : $t('ai.chat.noSessions') }}
      </div>
      <template v-for="(s, i) in props.sessions" :key="s.id">
        <div v-if="i === 0 || group(s) !== group(props.sessions[i - 1])" class="text-[10px] text-muted-foreground px-3 pb-1 pt-4">
          {{ group(s) }}
        </div>
        <div class="group p-1.5 rounded-xl flex gap-1 items-center" :class="s.id === activeId ? 'bg-primary/10' : 'hover:bg-accent/60'">
          <button type="button" class="text-[13px] p-1 text-left flex flex-1 gap-2 min-w-0 items-center" @click="emit('select', s.id)">
            <FaIcon :name="s.runStatus ? 'i-lucide:loader-circle' : s.pinned ? 'i-lucide:pin' : 'i-lucide:message-square'" class="text-muted-foreground shrink-0" :class="s.runStatus ? 'animate-spin text-primary' : ''" /><span class="truncate">{{ s.title }}</span>
          </button>
          <FaDropdown :items="menu(s)">
            <FaButton variant="ghost" size="icon-sm" :title="$t('common.operation')" class="opacity-50 shrink-0 group-hover:opacity-100">
              <FaIcon name="i-lucide:ellipsis" />
            </FaButton>
          </FaDropdown>
        </div>
      </template>
      <FaButton v-if="more" variant="ghost" size="sm" class="mt-3 w-full" :loading="loading" @click="emit('more')">
        {{ $t('aiAssistant.loadMore') }}
      </FaButton>
    </div>
    <FaModal
      :model-value="!!editing" :title="$t('common.rename')" @update:model-value="v => { if (!v) editing = undefined }" @confirm="editing && emit('rename', editing, title);
                                                                                                                                 editing = undefined"
    >
      <FaInput v-model="title" class="w-full" />
    </FaModal>
  </div>
</template>
