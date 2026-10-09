import { defineStore } from 'pinia'

import type { AiAskPayload } from '@/api/modules/ai'
import aiApi from '@/api/modules/ai'

// AI 危险操作确认全局弹窗状态（M31 ask 机制，任务中心同款单例模式）：
// useAiChat 收到 SSE ask 事件即调 request()，Layout 挂载的全局 YdAiAskModal 展示；
// 任何对话面（对话页/AI 浮层/桌面窗口）自动可用，无需各自接线。
export const useAiAskStore = defineStore('aiAsk', {
  state: () => ({
    payload: null as AiAskPayload | null,
    seg: null as any, // 消息流工具段引用（确认后同步段状态）
    resolver: null as ((approve: boolean) => void) | null,
  }),
  actions: {
    /** SSE ask 事件入口：挂起 Promise 等用户选择 */
    request(payload: AiAskPayload, seg?: any): Promise<boolean> {
      // 兜底：上一个 ask 仍挂着（后端串行执行理论不会发生），按拒绝先结算旧的
      if (this.resolver) {
        this.settle(false)
      }
      this.seg = seg || null
      return new Promise((resolve) => {
        this.payload = payload
        this.resolver = resolve
      })
    },
    /** 用户回执（批准/拒绝）：解决挂起、更新段状态并转发后端 */
    settle(approve: boolean) {
      const p = this.payload
      const seg = this.seg
      this.payload = null
      this.seg = null
      this.resolver?.(approve)
      this.resolver = null
      if (seg) {
        seg.done = true
        seg.status = approve ? '已批准' : '已拒绝'
      }
      if (p) {
        aiApi.resolveAsk(p.id, approve).catch(() => {})
      }
    },
    /** 后端超时/中断回执：仅收起弹窗（后端已有结论，不再发请求） */
    settled(id: string) {
      if (this.payload?.id === id) {
        this.payload = null
        this.resolver?.(false)
        this.resolver = null
      }
    },
  },
})
