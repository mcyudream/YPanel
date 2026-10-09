import { defineStore } from 'pinia'

import type { AiAskQuestion } from '@/api/modules/ai'
import aiApi from '@/api/modules/ai'

// AI 交互提问全局状态（ask_user 工具）：
// 挂起时 Sender 输入框位置替换为 QuestionPanel（多题导航 + 统一提交），回答后恢复输入框。
// seg 引用由 useAiChat 传入（消息流分段对象），提交/取消时同步段状态。
export const useAiQuestionStore = defineStore('aiQuestion', {
  state: () => ({
    id: '',
    questions: [] as AiAskQuestion[],
    seg: null as any,
    resolver: null as null | ((answerText: string) => void),
  }),
  actions: {
    /** SSE ask_user 事件入口：挂起等待用户作答 */
    request(id: string, questions: AiAskQuestion[], seg?: any) {
      if (this.resolver) {
        this.resolver('（有新提问进入，未作答的问题按未回答处理）')
        this.resolver = null
      }
      this.id = id
      this.questions = questions || []
      this.seg = seg || null
      return new Promise<string>((resolve) => {
        this.resolver = resolve
      })
    },
    /** 用户提交作答：回执后端、更新段状态并解决挂起，返回格式化答案文本 */
    async submit(answers: { header?: string, question?: string, selected: string[], text?: string }[]) {
      const id = this.id
      const qs = this.questions
      const seg = this.seg
      this.id = ''
      this.questions = []
      this.seg = null
      const resolver = this.resolver
      this.resolver = null
      try {
        await aiApi.resolveAskQuestion(id, answers)
      }
      catch {}
      const lines: string[] = []
      answers.forEach((a, i) => {
        const q = a.question || qs[i]?.question || qs[i]?.header || `问题${i + 1}`
        const parts: string[] = []
        if (a.selected?.length) {
          parts.push(`选择：${a.selected.join('、')}`)
        }
        if (a.text?.trim()) {
          parts.push(`补充：${a.text.trim()}`)
        }
        lines.push(`${i + 1}. ${q} → ${parts.length ? parts.join('；') : '（未作答）'}`)
      })
      const text = `用户回答：\n${lines.join('\n')}`
      if (seg) {
        seg.done = true
        seg.status = '已回答'
        seg.summary = text.replaceAll('\n', ' ')
      }
      resolver?.(text)
      return text
    },
    /** 会话中断取消 */
    cancelled(id: string) {
      if (this.id === id) {
        const seg = this.seg
        this.id = ''
        this.questions = []
        this.seg = null
        if (seg) {
          seg.done = true
          seg.status = '已取消'
        }
        this.resolver?.('用户没有回答（会话已中断）。')
        this.resolver = null
      }
    },
  },
})
