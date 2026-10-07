// useAiChat：AI 对话状态管理（对接 /api/v1/ai/chat SSE，含工具调用过程展示；每轮结束自动持久化会话）。
import { ref } from 'vue'
import { conversationApi } from '@/api/modules/ai'

export interface AiChatStep {
  type: string
  name?: string
  detail?: string
  /** 结果状态：完成/失败（空 = 未结束） */
  status?: string
  /** 结果摘要 */
  summary?: string
  done?: boolean
}

export interface AiChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  pending?: boolean
  error?: boolean
  /** 深度思考（reasoning 流式拼接） */
  reasoning?: string
  /** 工具/步骤链 */
  steps: AiChatStep[]
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

export function useAiChat(options?: {
  scenePath?: () => string
  providerId?: () => number | undefined
  autoSave?: boolean
  /** 每轮自动保存成功后的回调（如刷新会话列表） */
  onSaved?: () => void
}) {
  const messages = ref<AiChatMessage[]>([])
  const streaming = ref(false)
  const abort = ref<AbortController | null>(null)
  /** 当前会话 ID：首轮发送自动创建会话后回填；0 = 尚未持久化。clear() 归零 → 下一轮开新会话 */
  const conversationId = ref(0)

  /** 自动持久化：有可展示的 assistant 回复才保存（abort 半截也算，报错轮不落库）；含思考/步骤 */
  async function persist() {
    if (options?.autoSave === false) {
      return
    }
    if (!messages.value.some(m => m.role === 'assistant' && m.content && !m.error)) {
      return
    }
    const msgs = messages.value.filter(m => m.content).map(m => ({
      role: m.role,
      content: m.content,
      reasoning: m.reasoning || '',
      steps: m.steps || [],
    }))
    try {
      const res = await conversationApi.save(conversationId.value, '', msgs as any)
      conversationId.value = (res.data as any)?.id || conversationId.value
      options?.onSaved?.()
    }
    catch {
      // 会话可能已在别处（如另一标签/会话管理）被删除：归零转新建重试一次
      if (conversationId.value > 0) {
        conversationId.value = 0
        try {
          const res = await conversationApi.save(0, '', msgs as any)
          conversationId.value = (res.data as any)?.id || 0
          options?.onSaved?.()
        }
        catch (e) {
          console.warn('[ai] 会话自动保存失败', e)
        }
      }
      else {
        console.warn('[ai] 会话自动保存失败')
      }
    }
  }

  /** 发送一轮对话：流式更新最后一条 assistant（通过 reactive proxy 操作确保响应式）。 */
  async function send(text: string) {
    if (streaming.value) {
      return
    }
    if (text.trim()) {
      messages.value.push({ id: nextId(), role: 'user', content: text.trim(), steps: [] })
    }
    const lastUser = [...messages.value].reverse().find(m => m.role === 'user')
    if (!lastUser) {
      return
    }
    // 新建 assistant 占位消息，并经 reactive 数组索引取 proxy 引用（后续流式修改触发更新）。
    // 注意：不能取 length-1 复用 user 消息——role 必须是 assistant，思考块/步骤时间线/Markdown 都按它分支
    messages.value.push({ id: nextId(), role: 'assistant', content: '', pending: true, steps: [] })
    const assistant = messages.value[messages.value.length - 1]
    streaming.value = true
    const controller = new AbortController()
    abort.value = controller
    const token = localStorage.getItem('token') || ''
    const history = messages.value
      .filter(m => !m.pending && m.content && !m.error)
      .slice(-12)
      .map(m => ({ role: m.role, content: m.content }))

    try {
      const resp = await fetch(`api/v1/ai/chat?scene=${encodeURIComponent(options?.scenePath?.() || '/')}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        signal: controller.signal,
        body: JSON.stringify({
          providerId: options?.providerId?.(),
          messages: history,
        }),
      })
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
            assistant.steps.push({ type: 'scene', name: '读取页面数据', detail: ev.scene.page || '', done: true })
          }
          if (ev.reasoning) {
            assistant.reasoning = (assistant.reasoning || '') + ev.reasoning
          }
          if (ev.step) {
            assistant.steps.push({ type: ev.step.type, name: ev.step.name, detail: ev.step.detail })
          }
          if (ev.step_result) {
            // 把同名未完成步骤置为完成/失败并附结果摘要
            const st = [...assistant.steps].reverse().find(s => s.type === 'tool' && s.name === ev.step_result.name && !s.done)
            if (st) {
              st.done = true
              st.status = ev.step_result.status || ''
              st.summary = ev.step_result.detail || ''
            }
            else {
              assistant.steps.push({ type: 'tool_result', name: ev.step_result.name, detail: ev.step_result.detail, status: ev.step_result.status, done: true })
            }
          }
          if (ev.tool) {
            assistant.steps.push({ type: 'action', name: typeof ev.tool === 'string' ? ev.tool : ev.tool.name || '', done: true })
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
      await persist()
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
    conversationId.value = 0
  }

  return { messages, streaming, send, stop, regenerate, clear, conversationId }
}

/** 场景感知 SSE 请求需要 scene 路径时，随 body.scene 一起传给后端 */
export function aiSceneBody(scenePath?: string, extra?: Record<string, any>) {
  return { scene: scenePath, ...extra }
}
