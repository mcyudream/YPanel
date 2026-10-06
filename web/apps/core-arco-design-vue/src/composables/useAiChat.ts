// useAiChat：AI 对话状态管理（对接 /api/v1/ai/chat SSE，含工具调用过程展示）。
import { computed, ref } from 'vue'

export interface AiChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  pending?: boolean
  error?: boolean
  /** 工具调用过程行（ChatProcess 风格） */
  steps: string[]
}

export interface AiSceneData {
  page?: string
  [key: string]: any
}

let seq = 0
function nextId() {
  seq++
  return `ai-${Date.now()}-${seq}`
}

export function useAiChat(options?: { scenePath?: () => string, providerId?: () => number | undefined }) {
  const messages = ref<AiChatMessage[]>([])
  const streaming = ref(false)
  const abort = ref<AbortController | null>(null)

  const canSend = computed(() => !streaming.value && messages.value.some(m => m.role === 'user' ? false : true) || messages.value.length === 0 || true)

  function addUser(text: string) {
    messages.value.push({ id: nextId(), role: 'user', content: text, steps: [] })
  }

  function addAssistant(): AiChatMessage {
    const m: AiChatMessage = { id: nextId(), role: 'assistant', content: '', pending: true, steps: [] }
    messages.value.push(m)
    return m
  }

  /** 发送一轮对话：取最后一条用户消息为输入，流式更新最后一条 assistant。 */
  async function send(text: string) {
    if (streaming.value) {
      return
    }
    if (text.trim()) {
      addUser(text.trim())
    }
    const lastUser = [...messages.value].reverse().find(m => m.role === 'user')
    if (!lastUser) {
      return
    }
    const assistant = addAssistant()
    streaming.value = true
    abort.value = new AbortController()
    const token = localStorage.getItem('token') || ''
    const history = messages.value
      .filter(m => !m.pending && m.content && !m.error)
      .slice(-12)
      .map(m => ({ role: m.role, content: m.content }))

    try {
      const resp = await fetch(`api/v1/ai/chat?scene=${encodeURIComponent(options?.scenePath?.() || '/')}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        signal: abort.value.signal,
        body: JSON.stringify({
          providerId: options?.providerId?.(),
          messages: history,
        }),
      })
      // 后端业务错误（非 SSE）
      if (!resp.ok && !resp.headers.get('content-type')?.includes('event-stream')) {
        let msg = `HTTP ${resp.status}`
        try {
          const j = await resp.json()
          msg = j.message || msg
        }
        catch {}
        throw new Error(msg)
      }
      if (!resp.body) {
        throw new Error('无响应体')
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
        for (const raw of lines) {
          const line = raw.trim()
          if (!line.startsWith('data:')) {
            continue
          }
          const payload = line.slice(5).trim()
          if (!payload || payload === '[DONE]') {
            continue
          }
          let ev: any
          try {
            ev = JSON.parse(payload)
          }
          catch {
            continue
          }
          if (ev.scene) {
            assistant.steps.push(`已读取场景数据：${ev.scene.page || ''}`)
          }
          if (ev.tool) {
            assistant.steps.push(`调用工具：${typeof ev.tool === 'string' ? ev.tool : ev.tool.name || ''}`)
          }
          if (ev.content) {
            assistant.content += ev.content
          }
          if (ev.error) {
            assistant.error = true
            assistant.content += `\n\n${ev.error}`
          }
        }
      }
    }
    catch (e: any) {
      if (e?.name !== 'AbortError') {
        assistant.error = true
        assistant.content = `请求失败：${e?.message || e}`
      }
    }
    finally {
      assistant.pending = false
      streaming.value = false
      abort.value = null
    }
  }

  function stop() {
    abort.value?.abort()
  }

  function regenerate() {
    const lastUser = [...messages.value].reverse().find(m => m.role === 'user')
    if (!lastUser || streaming.value) {
      return
    }
    // 移除最后的 assistant 消息后重发
    for (let i = messages.value.length - 1; i >= 0; i--) {
      if (messages.value[i].role === 'assistant') {
        messages.value.splice(i, 1)
        break
      }
    }
    return send('')
  }

  function clear() {
    messages.value = []
  }

  void canSend
  return { messages, streaming, send, stop, regenerate, clear }
}

/** 场景感知 SSE 请求需要 scene 路径时，随 body.scene 一起传给后端 */
export function aiSceneBody(scenePath?: string, extra?: Record<string, any>) {
  return { scene: scenePath, ...extra }
}

