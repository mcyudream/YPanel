// useAiChat：AI 对话状态管理（对接 /api/v1/ai/chat SSE，含工具调用/危险操作确认过程展示；每轮结束自动持久化会话）。
import { computed, ref } from 'vue'
import { conversationApi } from '@/api/modules/ai'
import { i18n } from '@/locales'
import { useAiAskStore } from '@/store/modules/aiAsk'
import { useAiQuestionStore } from '@/store/modules/aiQuestion'
import type { AiAskQuestion } from '@/api/modules/ai'

export interface AiChatStep {
  type: string
  name?: string
  detail?: string
  /** 结果状态：完成/失败/已确认/已拒绝/已超时（空 = 未结束） */
  status?: string
  /** 结果摘要 */
  summary?: string
  done?: boolean
  /** 危险操作确认卡片：后端 ask ID */
  askId?: string
  /** 风险分级（ask 卡片用） */
  risk?: string
  /** 交互提问卡片：问题集（ask_user 工具） */
  questions?: AiAskQuestion[]
  /** 工具原始输出（截断），卡片展开查看 */
  output?: string
}

export interface AiKnowledgeRef {
  title: string
  body?: string
}

export interface AiChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  pending?: boolean
  error?: boolean
  /** 深度思考（reasoning 流式拼接） */
  reasoning?: string
  /** 工具/步骤链（旧版渲染路径，保留兼容历史会话） */
  steps: AiChatStep[]
  /** 引用的知识库条目 */
  knowledge?: AiKnowledgeRef[]
  /** ZCode 式分段序列（M32）：文本与工具块按时间序交错，工具输出为流内一等块 */
  segments?: AiSegment[]
}

/** 消息分段：文本段（Markdown）/ 工具块（输出直接可见）/ 交互提问块 */
export type AiSegment =
  | { type: 'text', text: string }
  | {
    type: 'tool'
    name: string
    args: string
    /** 可读摘要（后端渲染） */
    summary: string
    /** 原始输出（截断） */
    output: string
    status: string
    risk?: string
    /** 危险操作确认：挂起的 ask ID */
    askId?: string
    done?: boolean
  }
  | { type: 'question', askId: string, questions: AiAskQuestion[], done?: boolean, answer?: string }

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
  /** 场景内聚焦对象（容器名/编排项目/目录等），随 ?focus= 传给后端注入场景摘要 */
  sceneFocus?: () => string
  providerId?: () => number | undefined
  /** 模型覆盖（同一供应商多模型） */
  model?: () => string | undefined
  autoSave?: boolean
  /** 每轮自动保存成功后的回调（如刷新会话列表） */
  onSaved?: () => void
  /** 权限模式（M32 仿 ZCode）：read_only / standard / auto，随请求传后端 */
  mode?: () => string
  /** 上下文窗口上限（按当前模型），供用量估算；默认 128000 */
  contextLimit?: () => number
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
    const msgs = messages.value.filter(m => m.content || m.segments?.length).map(m => ({
      role: m.role,
      content: m.content,
      reasoning: m.reasoning || '',
      steps: m.steps || [],
      knowledge: m.knowledge || [],
      segments: m.segments || [],
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
      const focus = options?.sceneFocus?.() || ''
      const resp = await fetch(`api/v1/ai/chat?scene=${encodeURIComponent(options?.scenePath?.() || '/')}${focus ? `&focus=${encodeURIComponent(focus)}` : ''}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        signal: controller.signal,
        body: JSON.stringify({
          providerId: options?.providerId?.(),
          model: options?.model?.() || undefined,
          mode: options?.mode?.() || undefined,
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
        throw new Error(i18n.global.t('components.useAiChat.noResponseBody'))
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
          // ---- 段模型辅助：尾段操作 ----
          const segs = (assistant.segments = assistant.segments || [])
          function lastTextSeg() {
            const last = segs[segs.length - 1]
            if (last && last.type === 'text') {
              return last
            }
            const seg = { type: 'text' as const, text: '' }
            segs.push(seg)
            return seg
          }
          function lastRunningToolSeg() {
            for (let i = segs.length - 1; i >= 0; i--) {
              const seg = segs[i]
              if (seg.type === 'tool' && !seg.done) {
                return seg
              }
            }
            return undefined
          }

          if (ev.scene) {
            assistant.steps.push({ type: 'scene', name: '读取页面数据', detail: ev.scene.page || '', done: true })
          }
          if (ev.reasoning) {
            assistant.reasoning = (assistant.reasoning || '') + ev.reasoning
          }
          if (ev.step && ev.step.type === 'tool') {
            segs.push({ type: 'tool', name: ev.step.name || '', args: ev.step.detail || '', summary: '', output: '', status: 'running' })
            assistant.steps.push({ type: ev.step.type, name: ev.step.name, detail: ev.step.detail })
          }
          if (ev.step_result) {
            // 段模型：填充最近的运行中工具块
            const seg = lastRunningToolSeg()
            if (seg && seg.name === ev.step_result.name) {
              seg.summary = ev.step_result.detail || ''
              seg.output = ev.step_result.output || ''
              seg.status = ev.step_result.status || '完成'
              seg.done = true
            }
            // 兼容 steps（旧渲染/持久化）
            const st = [...assistant.steps].reverse().find(s => s.type === 'tool' && s.name === ev.step_result.name && !s.done)
            if (st) {
              st.done = true
              st.status = ev.step_result.status || ''
              st.summary = ev.step_result.detail || ''
              st.output = ev.step_result.output || ''
            }
            else {
              assistant.steps.push({ type: 'tool_result', name: ev.step_result.name, detail: ev.step_result.detail, status: ev.step_result.status, done: true, output: ev.step_result.output })
            }
          }
          if (ev.tool) {
            assistant.steps.push({ type: 'action', name: typeof ev.tool === 'string' ? ev.tool : ev.tool.name || '', done: true })
          }
          if (ev.ask) {
            // 危险操作确认：挂到最近运行中的工具段（内联批准/拒绝），同时通知 store
            const seg = lastRunningToolSeg()
            if (seg && seg.type === 'tool') {
              seg.askId = ev.ask.id
              seg.risk = ev.ask.risk
            }
            assistant.steps.push({ type: 'ask', name: ev.ask.tool, detail: ev.ask.args, risk: ev.ask.risk, askId: ev.ask.id })
            useAiAskStore().request(ev.ask, seg && seg.type === 'tool' ? seg : undefined)
          }
          if (ev.ask_user) {
            // 交互提问：以 ask_user 工具块形态进调用链（问题清单在摘要，答案回填 summary）
            const qList = (ev.ask_user.questions || []).map((q: AiAskQuestion) => q.question).join('；')
            segs.push({ type: 'tool', name: 'ask_user', args: JSON.stringify(ev.ask_user.questions || []), summary: i18n.global.t('components.useAiChat.awaitingAnswers', { list: qList }), output: '', status: 'running' })
            const qseg = segs[segs.length - 1]
            assistant.steps.push({
              type: 'question',
              name: '询问用户',
              detail: qList,
              questions: ev.ask_user.questions || [],
              askId: ev.ask_user.id,
            })
            useAiQuestionStore().request(ev.ask_user.id, ev.ask_user.questions || [], qseg)
          }
          if (ev.ask_user_result) {
            const st = [...assistant.steps].reverse().find(x => x.type === 'question' && x.askId === ev.ask_user_result.id)
            if (st) {
              st.done = true
              st.status = '已取消'
            }
            useAiQuestionStore().cancelled(ev.ask_user_result.id)
          }
          if (ev.ask_result) {
            // 确认终结：把对应 ask 卡片置为已确认/已拒绝/已超时；弹窗若仍挂起则强制关闭
            const st = [...assistant.steps].reverse().find(x => x.type === 'ask' && x.askId === ev.ask_result.id)
            if (st) {
              st.done = true
              st.status = ev.ask_result.approved ? '已确认' : (ev.ask_result.reason === 'timeout' ? '已超时' : '已拒绝')
            }
            useAiAskStore().settled(ev.ask_result.id)
          }
          if (ev.knowledge) {
            assistant.knowledge = ev.knowledge
          }
          if (ev.content) {
            assistant.content += ev.content
            lastTextSeg().text += ev.content
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
        assistant.content = i18n.global.t('components.useAiChat.requestFailed', { message: String(e?.message || e) })
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

  // 上下文用量估算（langchaingo 不透出 usage）：按会话字符数近似折算 tokens
  const contextUsage = computed(() => {
    let chars = 0
    for (const m of messages.value) {
      chars += (m.content?.length || 0) + (m.reasoning?.length || 0)
      for (const st of m.steps || []) {
        chars += (st.detail?.length || 0) + (st.summary?.length || 0)
      }
    }
    const limit = options?.contextLimit?.() || 128000
    const tokens = Math.min(limit, Math.round(chars / 2.2) + 1500)
    return { tokens, limit, pct: Math.round((tokens / limit) * 100) }
  })

  return { messages, streaming, send, stop, regenerate, clear, conversationId, contextUsage }
}

/** 场景感知 SSE 请求需要 scene 路径时，随 body.scene 一起传给后端 */
export function aiSceneBody(scenePath?: string, extra?: Record<string, any>) {
  return { scene: scenePath, ...extra }
}
