import type { AiAskQuestion } from './ai'
import { useAppAccountStore } from '@/store/modules/app/account'
import api, { rawApi } from '../index'

export interface ChatAttachment {
  id: string
  name: string
  mime: string
  size: number
  kind: string
}
export interface ChatSegment {
  type: 'text' | 'tool' | 'question'
  id?: string
  text?: string
  name?: string
  args?: string
  summary?: string
  outputId?: string
  output?: string
  status?: string
  risk?: string
  askId?: string
  questions?: AiAskQuestion[]
  done?: boolean
}
export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  reasoning?: string
  steps: Array<{
    type: string
    name?: string
    detail?: string
    done?: boolean
  }>
  segments?: ChatSegment[]
  knowledge?: Array<{
    title: string
    body?: string
  }>
  attachments?: ChatAttachment[]
  pending?: boolean
  error?: boolean
  interrupted?: boolean
  durationMs?: number
  createdAt?: string
}
export interface ChatSettings {
  providerId: number
  model: string
  mode: 'read_only' | 'confirm' | 'auto_edit' | 'full'
  maxRounds: number
  contextLimit: number
  maxOutputTokens: number
  autoCompact: boolean
  compactThreshold: number
  retainTurns: number
  summaryProviderId: number
  summaryModel: string
  systemPrompt: string
  temperature?: number
  skills: string[]
  memory: boolean
  knowledge: boolean
  tools: boolean
}
export function defaultChatSettings(): ChatSettings {
  return { providerId: 0, model: '', mode: 'confirm', maxRounds: 100, contextLimit: 32768, maxOutputTokens: 4096, autoCompact: true, compactThreshold: 0.8, retainTurns: 4, summaryProviderId: 0, summaryModel: '', systemPrompt: '', skills: [], memory: true, knowledge: true, tools: true }
}
export interface ChatContext {
  tokens: number
  limit: number
  outputReserve: number
  system: number
  history: number
  tools: number
  attachments: number
  estimated: boolean
  inputTokens?: number
  outputTokens?: number
  summary?: string
  summaryThrough?: number
}
export interface ChatSnapshot {
  conversationId: number
  title: string
  version: number
  runId: string
  status: string
  lastEventId: number
  messages: ChatMessage[]
  total: number
  hasMore: boolean
  settings: ChatSettings
  context: ChatContext
}
export interface ConversationMeta {
  id: number
  title: string
  version: number
  pinned: boolean
  archived: boolean
  updatedAt: string
  runStatus?: string
}
export interface ConversationPage {
  items: ConversationMeta[]
  total: number
  page: number
  pageSize: number
}
export interface ChatStart {
  conversationId: number
  requestId: string
  version?: number
  content: string
  attachmentIds: string[]
  settings: ChatSettings
  scene: string
  focus: string
}
export const aiChatApi = {
  defaults: async () => (await api.get('api/v1/ai/chat/defaults', { silent: true })).data as ChatSettings,
  saveDefaults: (settings: ChatSettings) => api.put('api/v1/ai/chat/defaults', settings),
  start: async (body: ChatStart) => (await api.post('api/v1/ai/chat/start', body, { silent: true })).data as ChatSnapshot,
  stop: (id: number, runId: string) => api.post('api/v1/ai/chat/stop', { conversationId: id, runId }, { silent: true }),
  list: async (params: {
    search?: string
    page?: number
    pageSize?: number
    archived?: boolean
  } = {}) => (await api.get('api/v1/ai/conversations', { params, silent: true })).data as ConversationPage,
  snapshot: async (id: number, before = 0) => (await api.get(`api/v1/ai/conversations/${id}`, { params: { before, limit: 50 }, silent: true })).data as ChatSnapshot,
  update: (id: number, data: {
    title?: string
    pinned?: boolean
    archived?: boolean
    settings?: ChatSettings
  }) => api.patch(`api/v1/ai/conversations/${id}`, data),
  remove: (id: number) => api.delete(`api/v1/ai/conversations/${id}`),
  fork: async (id: number, through: string, include = true) => (await api.post(`api/v1/ai/conversations/${id}/fork`, { through, include })).data as {
    id: number
  },
  compact: async (id: number) => (await api.post(`api/v1/ai/conversations/${id}/compact`, {}, { timeout: 180000, silent: true })).data as ChatSnapshot,
  upload: async (file: File, onProgress: (n: number) => void) => {
    const body = new FormData()
    body.append('file', file)
    return (await api.post('api/v1/ai/attachments', body, { timeout: 120000, silent: true, onUploadProgress: e => onProgress(Math.round((e.progress ?? 0) * 95)) })).data as ChatAttachment
  },
  removeAttachment: (id: string) => api.delete(`api/v1/ai/attachments/${encodeURIComponent(id)}`, { silent: true }),
  attachment: async (id: string) => (await rawApi.get(`api/v1/ai/attachments/${encodeURIComponent(id)}`, { responseType: 'blob' })).data as Blob,
}
/** Single authenticated SSE request. Cursor belongs to the server, never to the reader. */
export async function readChatStream(id: number, runId: string, offset: number, signal: AbortSignal, receive: (event: Record<string, any>) => void) {
  const url = new URL(`${api.defaults.baseURL?.replace(/\/$/, '') || ''}/api/v1/ai/chat/stream`, window.location.origin)
  url.search = new URLSearchParams({ conversationId: String(id), runId, offset: String(offset) }).toString()
  const response = await fetch(url, { headers: { Authorization: `Bearer ${useAppAccountStore().token}` }, signal, credentials: 'same-origin' })
  if (!response.ok || !response.headers.get('content-type')?.includes('text/event-stream') || !response.body) {
    const body = await response.json().catch(() => ({}))
    if (body.code === 2001) {
      useAppAccountStore().requestLogout()
    }
    throw new Error(body.message || `HTTP ${response.status}`)
  }
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  try {
    for (;;) {
      const { value, done } = await reader.read()
      if (done) {
        break
      }
      buffer += decoder.decode(value, { stream: true }).replace(/\r/g, '')
      if (buffer.length > 16 * 1024 * 1024) {
        throw new Error('SSE frame exceeds limit')
      }
      let end: number
      for (;;) {
        end = buffer.indexOf('\n\n')
        if (end < 0) {
          break
        }
        const frame = buffer.slice(0, end)
        buffer = buffer.slice(end + 2)
        const data = frame.split('\n').filter(line => line.startsWith('data:')).map(line => line.slice(5).trimStart()).join('\n')
        if (data && data !== '[DONE]') {
          receive(JSON.parse(data))
        }
      }
    }
  }
  finally {
    await reader.cancel().catch(() => {

    })
    reader.releaseLock()
  }
}
