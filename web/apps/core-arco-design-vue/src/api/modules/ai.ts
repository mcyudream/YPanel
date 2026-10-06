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

// LobeHub 风格供应商预设（B18）：icon 为 @lobehub/icons-static-svg 的 SVG 模块
export interface ProviderPreset extends AIProvider {
  icon?: string
}

// 静态 SVG 导入（vite 处理为 url）
import openaiSvg from '@lobehub/icons-static-svg/icons/openai.svg'
import anthropicSvg from '@lobehub/icons-static-svg/icons/anthropic.svg'
import geminiSvg from '@lobehub/icons-static-svg/icons/gemini.svg'
import deepseekSvg from '@lobehub/icons-static-svg/icons/deepseek.svg'
import zhipuSvg from '@lobehub/icons-static-svg/icons/zhipu.svg'
import moonshotSvg from '@lobehub/icons-static-svg/icons/moonshot.svg'
import qwenSvg from '@lobehub/icons-static-svg/icons/qwen.svg'
import minimaxSvg from '@lobehub/icons-static-svg/icons/minimax.svg'
import yiSvg from '@lobehub/icons-static-svg/icons/yi.svg'
import wenxinSvg from '@lobehub/icons-static-svg/icons/wenxin.svg'
import sparkSvg from '@lobehub/icons-static-svg/icons/spark.svg'
import siliconcloudSvg from '@lobehub/icons-static-svg/icons/siliconcloud.svg'
import grokSvg from '@lobehub/icons-static-svg/icons/grok.svg'
import mistralSvg from '@lobehub/icons-static-svg/icons/mistral.svg'
import groqSvg from '@lobehub/icons-static-svg/icons/groq.svg'
import togetherSvg from '@lobehub/icons-static-svg/icons/together.svg'
import openrouterSvg from '@lobehub/icons-static-svg/icons/openrouter.svg'
import baichuanSvg from '@lobehub/icons-static-svg/icons/baichuan.svg'
import cohereSvg from '@lobehub/icons-static-svg/icons/cohere.svg'
import ollamaSvg from '@lobehub/icons-static-svg/icons/ollama.svg'
import azureSvg from '@lobehub/icons-static-svg/icons/azure.svg'

export const providerPresets: ProviderPreset[] = [
  { id: 0, name: 'OpenAI', apiType: 'openai', baseURL: 'https://api.openai.com/v1', model: 'gpt-4o', isDefault: false, icon: openaiSvg },
  { id: 0, name: 'Anthropic Claude', apiType: 'anthropic', baseURL: 'https://api.anthropic.com', model: 'claude-sonnet-4-20250514', isDefault: false, icon: anthropicSvg },
  { id: 0, name: 'Google Gemini', apiType: 'openai', baseURL: 'https://generativelanguage.googleapis.com/v1beta/openai', model: 'gemini-2.0-flash', isDefault: false, icon: geminiSvg },
  { id: 0, name: 'DeepSeek', apiType: 'openai', baseURL: 'https://api.deepseek.com/v1', model: 'deepseek-chat', isDefault: false, icon: deepseekSvg },
  { id: 0, name: '智谱 GLM', apiType: 'openai', baseURL: 'https://open.bigmodel.cn/api/paas/v4', model: 'glm-4.6', isDefault: false, icon: zhipuSvg },
  { id: 0, name: 'Moonshot Kimi', apiType: 'openai', baseURL: 'https://api.moonshot.cn/v1', model: 'moonshot-v1-32k', isDefault: false, icon: moonshotSvg },
  { id: 0, name: '阿里通义 Qwen', apiType: 'openai', baseURL: 'https://dashscope.aliyuncs.com/compatible-mode/v1', model: 'qwen-plus', isDefault: false, icon: qwenSvg },
  { id: 0, name: '字节豆包（火山方舟）', apiType: 'openai', baseURL: 'https://ark.cn-beijing.volces.com/api/v3', model: 'doubao-pro-32k', isDefault: false },
  { id: 0, name: 'MiniMax', apiType: 'openai', baseURL: 'https://api.minimax.chat/v1', model: 'abab6.5s-chat', isDefault: false, icon: minimaxSvg },
  { id: 0, name: '零一万物 Yi', apiType: 'openai', baseURL: 'https://api.lingyiwanwu.com/v1', model: 'yi-large', isDefault: false, icon: yiSvg },
  { id: 0, name: '百度文心', apiType: 'openai', baseURL: 'https://qianfan.baidubce.com/v2', model: 'ernie-4.0-8k', isDefault: false, icon: wenxinSvg },
  { id: 0, name: '讯飞星火', apiType: 'openai', baseURL: 'https://spark-api-open.xf-yun.com/v1', model: 'generalv3.5', isDefault: false, icon: sparkSvg },
  { id: 0, name: '硅基流动 SiliconCloud', apiType: 'openai', baseURL: 'https://api.siliconflow.cn/v1', model: 'deepseek-ai/DeepSeek-V3', isDefault: false, icon: siliconcloudSvg },
  { id: 0, name: 'xAI Grok', apiType: 'openai', baseURL: 'https://api.x.ai/v1', model: 'grok-3', isDefault: false, icon: grokSvg },
  { id: 0, name: 'Mistral', apiType: 'openai', baseURL: 'https://api.mistral.ai/v1', model: 'mistral-large-latest', isDefault: false, icon: mistralSvg },
  { id: 0, name: 'Groq', apiType: 'openai', baseURL: 'https://api.groq.com/openai/v1', model: 'llama-3.3-70b-versatile', isDefault: false, icon: groqSvg },
  { id: 0, name: 'Together', apiType: 'openai', baseURL: 'https://api.together.xyz/v1', model: 'meta-llama/Llama-3.3-70B-Instruct-Turbo', isDefault: false, icon: togetherSvg },
  { id: 0, name: 'OpenRouter', apiType: 'openai', baseURL: 'https://openrouter.ai/api/v1', model: 'openai/gpt-4o', isDefault: false, icon: openrouterSvg },
  { id: 0, name: '百川 Baichuan', apiType: 'openai', baseURL: 'https://api.baichuan-ai.com/v1', model: 'Baichuan4', isDefault: false, icon: baichuanSvg },
  { id: 0, name: 'Cohere', apiType: 'openai', baseURL: 'https://api.cohere.com/compatibility/v1', model: 'command-r-plus', isDefault: false, icon: cohereSvg },
  { id: 0, name: 'Azure OpenAI', apiType: 'openai', baseURL: 'https://<resource>.openai.azure.com', model: 'gpt-4o', isDefault: false, icon: azureSvg },
  { id: 0, name: 'Ollama 本地', apiType: 'openai', baseURL: 'http://127.0.0.1:11434/v1', model: 'llama3.1', isDefault: false, icon: ollamaSvg },
]

/** 供应商名称 → 图标 URL（用于已配置项显示） */
export function providerIcon(name: string): string | undefined {
  return providerPresets.find(p => p.name === name || name.includes(p.name.split(' ')[0]))?.icon
}

// B18：会话持久化
export interface AIConversationMeta {
  id: number
  title: string
  updatedAt: string
}

export const conversationApi = {
  list: async () => {
    const res = await api.get('api/v1/ai/conversations', { silent: true })
    return res.data as AIConversationMeta[]
  },
  get: async (id: number) => {
    const res = await api.get(`api/v1/ai/conversations/${id}`, { silent: true })
    return res.data as { id: number, title: string, messages: ChatMsg[] }
  },
  save: (id: number, title: string, messages: ChatMsg[]) =>
    api.post('api/v1/ai/conversations', { id, title, messages }),
  remove: (id: number) => api.delete(`api/v1/ai/conversations/${id}`),
}

// B18：长期记忆
export const memoryApi = {
  list: async () => {
    const res = await api.get('api/v1/ai/memories', { silent: true })
    return res.data as { id: number, content: string, createdAt: string }[]
  },
  clear: () => api.delete('api/v1/ai/memories'),
}

// B18+：技能包（SKILL.md 文件格式）
export interface AISkill {
  name: string
  description: string
  body: string
  enabled: boolean
}

export const skillApi = {
  list: async () => {
    const res = await api.get('api/v1/ai/skills', { silent: true })
    return res.data as AISkill[]
  },
  save: (data: { name: string, description: string, body: string }) =>
    api.post('api/v1/ai/skills', data),
  setEnabled: (name: string, enabled: boolean) =>
    api.post(`api/v1/ai/skills/${encodeURIComponent(name)}/enable`, { enabled }),
  remove: (name: string) => api.delete(`api/v1/ai/skills/${encodeURIComponent(name)}`),
}

// B18+：MCP 服务器
export interface MCPServer {
  name: string
  transport: 'stdio' | 'streamable-http'
  command?: string
  args?: string[]
  url?: string
  env?: Record<string, string>
  enabled: boolean
}

export const mcpApi = {
  list: async () => {
    const res = await api.get('api/v1/ai/mcp/servers', { silent: true })
    return res.data as MCPServer[]
  },
  saveAll: (servers: MCPServer[]) => api.put('api/v1/ai/mcp/servers', { servers }),
  test: async (server: MCPServer) => {
    const res = await api.post('api/v1/ai/mcp/test', server, { timeout: 60000 })
    return res.data as { connected: boolean, tools: string[] }
  },
  close: (name: string) => api.delete(`api/v1/ai/mcp/${encodeURIComponent(name)}`),
}
