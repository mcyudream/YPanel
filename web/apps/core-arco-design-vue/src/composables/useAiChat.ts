import { computed, onActivated, onDeactivated, onScopeDispose, ref, watch } from 'vue'
import { useAiChatStore } from '@/store/modules/aiChat'

export interface AiChatStep {
  type: string
  name?: string
  detail?: string
  status?: string
  summary?: string
  output?: string
  done?: boolean
  askId?: string
  risk?: string
}
export type { ChatMessage as AiChatMessage, ChatSegment as AiSegment } from '@/api/modules/aiChat'
/** Views own selection; the store owns conversation state and subscriptions. */
export function useAiChat() {
  const store = useAiChatStore()
  const selected = ref(store.create())
  const current = computed(() => store.entry(selected.value))
  let release: (() => void) | undefined
  let active = true
  watch(current, (e) => {
    release?.()
    release = active ? store.observe(e) : undefined
  }, { immediate: true })
  onDeactivated(() => {
    active = false
    release?.()
    release = undefined
  })
  onActivated(() => {
    active = true
    release ||= store.observe(current.value)
  })
  onScopeDispose(() => release?.())
  function open(id: number) {
    selected.value = id
  }
  function create() {
    selected.value = store.create()
  }
  async function send(scene = '/', focus = '') {
    const target = current.value
    const selectedID = selected.value
    const id = await store.send(target, scene, focus)
    if (selected.value === selectedID) {
      selected.value = id
    }
  }
  return { store, current, selected, open, create, send }
}
