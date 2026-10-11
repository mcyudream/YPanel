// useAiChat：AI 对话状态管理（M62 后台任务化：POST start 启动任务 + GET stream 订阅事件流）。
// 任务在后端运行（不随连接断开终止）；断线自动重连（offset 续传）；换窗口/强刷新经 attachLive 无感介入；
// 停止 = POST stop 终止后端任务。含工具调用/危险操作确认过程展示；任务结束由后端统一落库。
import { computed, ref } from 'vue'
import { i18n } from '@/locales'
import { useAiAskStore } from '@/store/modules/aiAsk'
import { useAiQuestionStore } from '@/store/modules/aiQuestion'
import { useAppAccountStore } from '@/store/modules/app/account'
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
  /** 流异常中断（连接断开且未收到 done 事件）：气泡下灰字提示（M60） */
  interrupted?: boolean
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
  /** 场景内聚焦对象（容器名/编排项目/目录等），随请求传给后端注入场景摘要 */
  sceneFocus?: () => string
  providerId?: () => number | undefined
  /** 模型覆盖（同一供应商多模型） */
  model?: () => string | undefined
  autoSave?: boolean
  /** 任务收尾（完成/停止/出错）后的回调（如刷新会话列表） */
  onSaved?: () => void
  /** 权限档位：read_only / confirm / auto_edit / full，随请求传后端 */
  mode?: () => string
  /** 上下文窗口上限（按当前模型），供用量估算；默认 128000 */
  contextLimit?: () => number
}) {
  const messages = ref<AiChatMessage[]>([])
  const streaming = ref(false)
  const abort = ref<AbortController | null>(null)
  /** 当前会话 ID：首轮发送后端创建会话后回填 */
  const conversationId = ref(0)

  function authHeaders() {
    // 与 axios 拦截器同源（account store）：裸 localStorage 在 fa 包装下取不到（M60 修复）
    const token = useAppAccountStore().token || ''
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }
  }

  /** consumeStream 订阅任务事件流并驱动渲染（回放 + 实时；断线自动重连 offset 续传）。
   *  assistant 为占位消息引用；返回时任务流已收尾（done/中断/用户停止）。 */
  async function consumeStream(
    controller: AbortController,
    convoId: number,
    startOffset: number,
    attempt: number,
    assistant: AiChatMessage,
  ): Promise<void> {
    const resp = await fetch(`api/v1/ai/chat/stream?conversationId=${convoId}&offset=${startOffset}`, {
      headers: { Authorization: `Bearer ${useAppAccountStore().token || ''}` },
      signal: controller.signal,
    })
    if (!resp.headers.get('content-type')?.includes('event-stream')) {
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
    let received = startOffset
    let sawDone = false
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
        // 心跳注释帧（: ping）不计数
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
        received++

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
          const seg = lastRunningToolSeg()
          if (seg && seg.name === ev.step_result.name) {
            seg.summary = ev.step_result.detail || ''
            seg.output = ev.step_result.output || ''
            seg.status = ev.step_result.status || '完成'
            seg.done = true
          }
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
          const seg = lastRunningToolSeg()
          if (seg && seg.type === 'tool') {
            seg.askId = ev.ask.id
            seg.risk = ev.ask.risk
          }
          assistant.steps.push({ type: 'ask', name: ev.ask.tool, detail: ev.ask.args, risk: ev.ask.risk, askId: ev.ask.id })
          useAiAskStore().request(ev.ask, seg && seg.type === 'tool' ? seg : undefined)
        }
        if (ev.ask_user) {
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
        if (ev.done) {
          sawDone = true
        }
        if (ev.error) {
          assistant.error = true
          assistant.content += `\n\n${ev.error}`
          sawDone = true
        }
      }
    }
    if (sawDone) {
      return
    }
    // 连接异常断开（未收 done 且非用户停止）：offset 续传重连，任务在后端不受影响
    if (attempt < 3 && !controller.signal.aborted) {
      await new Promise(r => setTimeout(r, 800 * (attempt + 1)))
      if (controller.signal.aborted) {
        return
      }
      await consumeStream(controller, convoId, received, attempt + 1, assistant)
      return
    }
    if (!controller.signal.aborted) {
      assistant.interrupted = true
    }
  }

  /** 发送一轮对话：POST start 启动后台任务 → 订阅事件流。 */
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
    // 新建 assistant 占位消息（reactive 数组索引取 proxy 引用保证流式响应式）
    messages.value.push({ id: nextId(), role: 'assistant', content: '', pending: true, steps: [] })
    const assistant = messages.value[messages.value.length - 1]
    streaming.value = true
    const controller = new AbortController()
    abort.value = controller
    const history = messages.value
      .filter(m => !m.pending && m.content && !m.error)
      .slice(-20)
      .map(m => ({ role: m.role, content: m.content }))
    try {
      const resp = await fetch('api/v1/ai/chat/start', {
        method: 'POST',
        headers: authHeaders(),
        signal: controller.signal,
        body: JSON.stringify({
          conversationId: conversationId.value || 0,
          providerId: options?.providerId?.() || undefined,
          model: options?.model?.() || undefined,
          mode: options?.mode?.() || undefined,
          scene: options?.scenePath?.() || '/',
          focus: options?.sceneFocus?.() || undefined,
          messages: history,
        }),
      })
      if (!resp.headers.get('content-type')?.includes('json')) {
        throw new Error(`HTTP ${resp.status}`)
      }
      const j = await resp.json()
      if (j.code !== 0) {
        throw new Error(j.message || `HTTP ${resp.status}`)
      }
      conversationId.value = j.data.conversationId
      await consumeStream(controller, conversationId.value, 0, 0, assistant)
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
      options?.onSaved?.()
    }
  }

  /** attachLive 探测并接入进行中的任务（换窗口/强刷新无感介入，M62）。
   *  返回运行状态字符串（running/done/stopped/error）或 null=无任务（调用方回退静态历史）。
   *  base 为该会话的历史消息（attach 后自动追加本轮占位并回放重建）。 */
  async function attachLive(id: number, base: AiChatMessage[]): Promise<string | null> {
    const controller = new AbortController()
    let resp: Response
    try {
      resp = await fetch(`api/v1/ai/chat/stream?conversationId=${id}&offset=0`, {
        headers: { Authorization: `Bearer ${useAppAccountStore().token || ''}` },
        signal: controller.signal,
      })
    }
    catch {
      return null
    }
    const runStatus = resp.headers.get('X-Ai-Run') || 'none'
    if (runStatus === 'none') {
      controller.abort()
      return null
    }
    messages.value = [...base]
    messages.value.push({ id: nextId(), role: 'assistant', content: '', pending: true, steps: [] })
    const assistant = messages.value[messages.value.length - 1]
    conversationId.value = id
    streaming.value = true
    abort.value = controller
    try {
      await consumeStream(controller, id, 0, 0, assistant)
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
      options?.onSaved?.()
    }
    return runStatus
  }

  /** 停止：终止后端任务（fail-closed）+ 关闭本地订阅。 */
  function stop() {
    const convo = conversationId.value
    if (streaming.value && convo) {
      void fetch('api/v1/ai/chat/stop', {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify({ conversationId: convo }),
      }).catch(() => {})
    }
    abort.value?.abort()
  }

  function clear() {
    messages.value = []
    conversationId.value = 0
  }

  function regenerate() {
    const lastUser = [...messages.value].reverse().find(m => m.role === 'user')
    if (!lastUser || streaming.value) {
      return
    }
    for (let i = messages.value.length - 1; i >= 0; i--) {
      if (messages.value[i].role === 'assistant') {
        messages.value.splice(i, 1)
        break
      }
    }
    return send('')
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

  return { messages, streaming, send, stop, regenerate, clear, attachLive, conversationId, contextUsage }
}

