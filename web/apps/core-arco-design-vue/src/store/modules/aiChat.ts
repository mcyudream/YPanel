import type { AIProvider } from '@/api/modules/ai'
import type { ChatAttachment, ChatContext, ChatMessage, ChatSettings, ChatSnapshot, ChatStart } from '@/api/modules/aiChat'
import { defineStore } from 'pinia'
import { markRaw, reactive, ref, watch } from 'vue'
import aiApi from '@/api/modules/ai'
import { aiChatApi, defaultChatSettings, readChatStream } from '@/api/modules/aiChat'
import { useAppAccountStore } from '@/store/modules/app/account'

export interface DraftAttachment {
  key: string
  file?: File
  attachment?: ChatAttachment
  progress: number
  error?: string
  preview?: string
}
export interface ChatEntry {
  id: number
  title: string
  version: number
  runId: string
  cursor: number
  status: string
  messages: ChatMessage[]
  total: number
  hasMore: boolean
  settings: ChatSettings
  context?: ChatContext
  draft: string
  uploads: DraftAttachment[]
  loading: boolean
  sending: boolean
  connected: boolean
  error: string
  notice: string
  revision: number
  request?: ChatStart
}
const errorText = (e: any) => String(e?.message || e)
const clone = <T>(v: T): T => JSON.parse(JSON.stringify(v))
export const useAiChatStore = defineStore('aiChat', () => {
  const entries = reactive<Record<number, ChatEntry>>({})
  const providers = ref<AIProvider[]>([])
  const defaults = ref(defaultChatSettings())
  const listRevision = ref(0)
  let draftSequence = 0
  let epoch = 0
  let initializing: Promise<void> | undefined
  const viewers = new Map<ChatEntry, number>()
  const connections = new Map<ChatEntry, AbortController>()
  const refreshes = new Map<ChatEntry, Promise<void>>()
  function entry(id: number) {
    return entries[id] ||= { id, title: '', version: 0, runId: '', cursor: 0, status: 'idle', messages: [], total: 0, hasMore: false, settings: clone(defaults.value), draft: '', uploads: [], loading: false, sending: false, connected: false, error: '', notice: '', revision: 0 }
  }
  function create() {
    return entry(--draftSequence).id
  }
  async function initialize() {
    const generation = epoch
    initializing ||= Promise.all([aiApi.providers(), aiChatApi.defaults()]).then(([ps, settings]) => {
      if (generation !== epoch) {
        return
      }
      providers.value = ps
      defaults.value = settings
      const provider = ps.find(p => p.id === settings.providerId) || ps.find(p => p.isDefault) || ps[0]
      if (provider && !settings.providerId) {
        Object.assign(defaults.value, { providerId: provider.id, model: provider.model })
      }
      for (const e of Object.values(entries)) {
        if (e.id < 0 && !e.settings.providerId) {
          e.settings = clone(defaults.value)
        }
      }
    }).catch((err) => {
      initializing = undefined
      throw err
    })
    return initializing
  }
  async function reloadProviders() {
    const generation = epoch
    const ps = await aiApi.providers()
    if (generation === epoch) {
      providers.value = ps
    }
  }
  function applySnapshot(e: ChatEntry, s: ChatSnapshot) {
    if (s.version < e.version) {
      return
    }
    if (s.runId === e.runId && s.status === 'running' && s.lastEventId < e.cursor) {
      return
    }
    const tailIDs = new Set((s.messages || []).map(m => m.id))
    const start = e.messages.findIndex(m => tailIDs.has(m.id))
    const prefix = start > 0 ? e.messages.slice(0, start) : []
    e.messages = [...prefix, ...(s.messages || [])].map(m => ({ ...m, steps: m.steps || [], segments: m.segments || [] }))
    Object.assign(e, { id: s.conversationId, title: s.title, version: s.version, runId: s.runId || '', cursor: s.lastEventId || 0, status: s.status, total: s.total, hasMore: s.total > e.messages.length, settings: s.settings, context: s.context })
    e.revision++
  }
  function applyEvent(e: ChatEntry, ev: Record<string, any>) {
    if (ev.sync) {
      applySnapshot(e, ev.sync)
      return
    }
    if (ev.runId !== e.runId) {
      throw new Error('SSE run changed')
    }
    if (ev.eventId <= e.cursor) {
      return
    }
    if (ev.eventId !== e.cursor + 1) {
      throw new Error('SSE cursor gap')
    }
    e.cursor = ev.eventId
    e.error = ''
    const m = e.messages.at(-1)
    if (!m || m.role !== 'assistant') {
      throw new Error('SSE assistant missing')
    }
    const segs = m.segments ||= []
    if (ev.content) {
      m.content += ev.content
      let tail = segs.at(-1)
      if (tail?.type !== 'text') {
        tail = { type: 'text', text: '' }
        segs.push(tail)
      }
      tail.text = (tail.text || '') + ev.content
    }
    if (ev.reasoning) {
      m.reasoning = (m.reasoning || '') + ev.reasoning
    }
    if (ev.step?.type === 'tool') {
      segs.push({ type: 'tool', id: ev.step.id, name: ev.step.name, args: ev.step.detail, status: 'running' })
    }
    if (ev.step_result) {
      const r = ev.step_result
      const seg = [...segs].reverse().find(s => s.type === 'tool' && s.name === r.name && (!r.id || s.id === r.id))
      if (seg) {
        Object.assign(seg, { summary: r.detail, output: r.output, outputId: r.outputId, status: r.status, done: true })
      }
    }
    if (ev.ask) {
      const seg = [...segs].reverse().find(s => s.type === 'tool' && !s.done)
      if (seg) {
        Object.assign(seg, { askId: ev.ask.id, risk: ev.ask.risk, status: 'waiting' })
      }
    }
    if (ev.ask_result) {
      const seg = segs.find(s => s.askId === ev.ask_result.id)
      if (seg) {
        Object.assign(seg, { done: !ev.ask_result.approved, status: ev.ask_result.approved ? 'running' : '已拒绝' })
      }
    }
    if (ev.ask_user) {
      segs.push({ type: 'question', askId: ev.ask_user.id, questions: ev.ask_user.questions, status: 'waiting' })
    }
    if (ev.ask_user_result) {
      const seg = segs.find(s => s.askId === ev.ask_user_result.id)
      if (seg) {
        Object.assign(seg, { done: true, status: ev.ask_user_result.cancelled ? '已取消' : '已回答', summary: ev.ask_user_result.answer })
      }
    }
    if (ev.knowledge) {
      m.knowledge = ev.knowledge
    }
    if (ev.context) {
      e.context = ev.context
    }
    if (ev.compact_error) {
      e.notice = ev.compact_error
    }
    if (ev.error) {
      m.error = true
      m.content += `\n\n${ev.error}`
      segs.push({ type: 'text', text: ev.error })
    }
    if (ev.done) {
      e.status = ev.status || 'done'
      m.pending = false
      m.durationMs = ev.durationMs
      for (const seg of segs) {
        if (seg.type !== 'text' && !seg.done) {
          Object.assign(seg, { done: true, status: '已中断' })
        }
      }
      listRevision.value++
    }
    e.revision++
  }
  function connect(e: ChatEntry) {
    if (e.id <= 0 || e.status !== 'running' || !viewers.get(e) || connections.has(e)) {
      return
    }
    const controller = markRaw(new AbortController())
    connections.set(e, controller)
    const generation = epoch
    void (async () => {
      let failures = 0
      while (!controller.signal.aborted && e.status === 'running') {
        if (generation !== epoch) {
          break
        }
        let queue: Record<string, any>[] = []
        let timer: ReturnType<typeof setTimeout> | undefined
        let gap = false
        const streamController = new AbortController()
        const abortStream = () => streamController.abort()
        controller.signal.addEventListener('abort', abortStream, { once: true })
        const flush = () => {
          clearTimeout(timer)
          timer = undefined
          if (controller.signal.aborted || generation !== epoch) {
            queue = []
            return
          }
          for (const ev of queue) {
            try {
              applyEvent(e, ev)
            }
            catch {
              gap = true
              streamController.abort()
              break
            }
          }
          queue = []
        }
        try {
          e.connected = true
          await readChatStream(e.id, e.runId, e.cursor, streamController.signal, (ev) => {
            queue.push(ev)
            // Timers flush hidden tabs too; rAF alone would stall background rendering.
            timer ||= setTimeout(flush, 32)
          })
          flush()
          failures = 0
        }
        catch (err) {
          flush()
          if (controller.signal.aborted) {
            break
          }
          e.error = errorText(err)
          failures++
        }
        finally {
          controller.signal.removeEventListener('abort', abortStream)
          clearTimeout(timer)
          e.connected = false
        }
        if (controller.signal.aborted || e.status !== 'running') {
          break
        }
        if (gap || failures > 0) {
          try {
            const snapshot = await aiChatApi.snapshot(e.id)
            if (controller.signal.aborted || generation !== epoch) {
              break
            }
            applySnapshot(e, snapshot)
          }
          catch (err) {
            e.error = errorText(err)
          }
        }
        if (failures >= 5) {
          e.error ||= '连接已断开，请重新连接'
          break
        }
        await new Promise<void>((resolve) => {
          const delay = setTimeout(resolve, Math.min(8000, 600 * 2 ** failures))
          controller.signal.addEventListener('abort', () => {
            clearTimeout(delay)
            resolve()
          }, { once: true })
        })
      }
    })().finally(() => {
      if (connections.get(e) === controller) {
        connections.delete(e)
        e.connected = false
      }
    })
  }
  async function refresh(e: ChatEntry) {
    if (e.id <= 0) {
      return
    }
    const existing = refreshes.get(e)
    if (existing) {
      return existing
    }
    const generation = epoch
    e.loading = !e.messages.length
    const promise = (async () => {
      try {
        const s = await aiChatApi.snapshot(e.id)
        if (generation !== epoch) {
          return
        }
        if (!connections.has(e)) {
          applySnapshot(e, s)
        }
        e.error = ''
        connect(e)
      }
      catch (err) {
        e.error = errorText(err)
      }
      finally {
        e.loading = false
        refreshes.delete(e)
      }
    })()
    refreshes.set(e, promise)
    return promise
  }
  function observe(e: ChatEntry) {
    viewers.set(e, (viewers.get(e) || 0) + 1)
    void refresh(e)
    return () => {
      const n = (viewers.get(e) || 1) - 1
      if (n > 0) {
        viewers.set(e, n)
      }
      else {
        viewers.delete(e)
        connections.get(e)?.abort()
        connections.delete(e)
      }
    }
  }
  async function send(e: ChatEntry, scene = '/', focus = '') {
    if (e.sending || e.status === 'running' || (!e.draft.trim() && !e.uploads.length)) {
      return e.id
    }
    if (e.uploads.some(u => !u.attachment || u.error)) {
      throw new Error('附件尚未就绪')
    }
    const body: ChatStart = { conversationId: Math.max(0, e.id), requestId: crypto.randomUUID(), version: e.id > 0 ? e.version : undefined, content: e.draft, attachmentIds: e.uploads.map(u => u.attachment!.id), settings: clone(e.settings), scene, focus }
    if (e.request && (e.request.content !== body.content || JSON.stringify(e.request.attachmentIds) !== JSON.stringify(body.attachmentIds))) {
      throw new Error('请先重试上一次发送以确认结果')
    }
    const generation = epoch
    e.sending = true
    e.error = ''
    e.request ||= body
    try {
      const s = await aiChatApi.start(e.request)
      if (generation !== epoch) {
        throw new Error('Account changed during send')
      }
      const oldID = e.id
      applySnapshot(e, s)
      entries[s.conversationId] = e
      // Keep the draft alias until every selecting view has followed its assigned ID.
      if (oldID < 0) {
        entries[oldID] = e
      }
      e.draft = ''
      for (const u of e.uploads) {
        if (u.preview) {
          URL.revokeObjectURL(u.preview)
        }
      }
      e.uploads = []
      e.request = undefined
      listRevision.value++
      connect(e)
      return s.conversationId
    }
    catch (err) {
      e.error = errorText(err)
      if (err && typeof err === 'object' && 'code' in err) {
        e.request = undefined
        await refresh(e)
      }
      throw err
    }
    finally {
      e.sending = false
    }
  }
  async function retryUpload(e: ChatEntry, key: string) {
    const u = e.uploads.find(u => u.key === key)
    if (!u?.file) {
      return
    }
    const generation = epoch
    u.error = undefined
    u.progress = 0
    try {
      const attachment = await aiChatApi.upload(u.file, n => u.progress = n)
      if (generation !== epoch) {
        return
      }
      if (!e.uploads.includes(u)) {
        await aiChatApi.removeAttachment(attachment.id)
        return
      }
      u.attachment = attachment
      u.progress = 100
    }
    catch (err) {
      u.error = errorText(err)
    }
  }
  function upload(e: ChatEntry, files: File[]) {
    for (const file of files.slice(0, Math.max(0, 10 - e.uploads.length))) {
      const item: DraftAttachment = { key: crypto.randomUUID(), file: markRaw(file), progress: 0, preview: file.type.startsWith('image/') ? URL.createObjectURL(file) : undefined }
      e.uploads.push(item)
      void retryUpload(e, item.key)
    }
  }
  function removeUpload(e: ChatEntry, key: string) {
    const u = e.uploads.find(u => u.key === key)
    if (!u) {
      return
    }
    e.uploads = e.uploads.filter(u => u.key !== key)
    if (u.preview) {
      URL.revokeObjectURL(u.preview)
    }
    if (u.attachment) {
      void aiChatApi.removeAttachment(u.attachment.id).catch(() => {

      })
    }
  }
  async function older(e: ChatEntry) {
    if (!e.hasMore || e.loading) {
      return
    }
    e.loading = true
    try {
      const s = await aiChatApi.snapshot(e.id, e.total - e.messages.length)
      const ids = new Set(e.messages.map(m => m.id))
      e.messages.unshift(...s.messages.filter(m => !ids.has(m.id)))
      e.hasMore = e.messages.length < e.total
    }
    catch (err) {
      e.error = errorText(err)
    }
    finally {
      e.loading = false
    }
  }
  async function stop(e: ChatEntry) {
    try {
      await aiChatApi.stop(e.id, e.runId)
    }
    catch (err) {
      e.error = errorText(err)
    }
  }
  function reset() {
    epoch++
    for (const c of connections.values()) {
      c.abort()
    }
    for (const e of Object.values(entries)) {
      for (const u of e.uploads) {
        if (u.preview) {
          URL.revokeObjectURL(u.preview)
        }
      }
    }
    connections.clear()
    viewers.clear()
    refreshes.clear()
    for (const id of Object.keys(entries)) {
      delete entries[Number(id)]
    }
    providers.value = []
    initializing = undefined
    defaults.value = defaultChatSettings()
  }
  watch(() => useAppAccountStore().token, (v, old) => {
    if (v !== old) {
      reset()
    }
  })
  return { entries, providers, defaults, listRevision, entry, create, initialize, reloadProviders, observe, refresh, send, upload, retryUpload, removeUpload, older, stop, applySnapshot }
})
