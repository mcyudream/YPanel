import api from '../index'

export interface AIProvider {
  id: number
  name: string
  apiType: 'openai' | 'anthropic' | 'response'
  baseURL: string
  apiKey?: string
  model: string
  isDefault: boolean
}

export interface ChatMsg {
  role: 'user' | 'assistant'
  content: string
}

export default {
  providers: async () => {
    const res = await api.get('api/v1/ai/providers', { silent: true })
    return res.data as AIProvider[]
  },
  presets: async () => {
    const res = await api.get('api/v1/ai/presets', { silent: true })
    return res.data as AIProvider[]
  },
  saveProvider: (p: AIProvider) => api.post('api/v1/ai/providers', p),
  removeProvider: (id: number) => api.delete(`api/v1/ai/providers/${id}`),

  // SSE 流式对话（fetch 流解析，返回增量回调）
  chatStream: async (
    providerId: number | undefined,
    messages: ChatMsg[],
    onDelta: (content: string) => void,
    onDone?: () => void,
  ) => {
    const token = localStorage.getItem('token') || ''
    const resp = await fetch('api/v1/ai/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ providerId, messages }),
    })
    if (!resp.ok || !resp.body) {
      throw new Error(`AI 服务异常: HTTP ${resp.status}`)
    }
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    let buf = ''
    for (;;) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }
      buf += decoder.decode(value, { stream: true })
      const lines = buf.split('\n')
      buf = lines.pop() || ''
      for (const line of lines) {
        const t = line.trim()
        if (!t.startsWith('data:')) {
          continue
        }
        const payload = t.slice(5).trim()
        if (payload === '[DONE]') {
          onDone?.()
          return
        }
        try {
          const ev = JSON.parse(payload)
          if (ev.content) {
            onDelta(ev.content)
          }
        }
        catch {}
      }
    }
    onDone?.()
  },
}
