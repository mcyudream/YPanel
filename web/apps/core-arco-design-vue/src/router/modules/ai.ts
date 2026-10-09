import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

// 智能域（M23）：八个功能区做成「单子页模块」（子页 menu:false → fa Menu 渲染为叶子项，
// 次侧栏直接平铺叶子、无分组头行，与容器域形态一致）。

function singleModule(path: string, name: string, title: string, icon: string, component: () => Promise<any>) {
  return {
    path,
    component: Layout,
    name,
    meta: {
      title,
      icon,
      auth: ['ai:use'],
    },
    children: [
      {
        path: '',
        name: `${name}Index`,
        component,
        meta: {
          title,
          icon,
          menu: false,
        },
      },
    ],
  }
}

export const AiChat = singleModule('/ai/chat', 'aiChat', 'menu.aiChat', 'i-lucide:message-circle', () => import('@/views/ai/chat.vue'))
export const AiProviders = singleModule('/ai/providers', 'aiProviders', 'menu.aiProviders', 'i-lucide:plug', () => import('@/views/ai/providers.vue'))
export const AiKnowledge = singleModule('/ai/knowledge', 'aiKnowledge', 'menu.aiKnowledge', 'i-lucide:book-open', () => import('@/views/ai/knowledge.vue'))
export const AiWorkspace = singleModule('/ai/workspace', 'aiWorkspace', 'menu.aiWorkspace', 'i-lucide:folder-code', () => import('@/views/ai/workspace.vue'))
export const AiMemory = singleModule('/ai/memory', 'aiMemory', 'menu.aiMemory', 'i-lucide:brain', () => import('@/views/ai/memory.vue'))
export const AiTools = singleModule('/ai/tools', 'aiTools', 'menu.aiTools', 'i-lucide:wrench', () => import('@/views/ai/tools.vue'))
export const AiSkills = singleModule('/ai/skills', 'aiSkills', 'menu.aiSkills', 'i-lucide:puzzle', () => import('@/views/ai/skills.vue'))
export const AiMcp = singleModule('/ai/mcp', 'aiMcp', 'menu.aiMcp', 'i-lucide:plug-zap', () => import('@/views/ai/mcp.vue'))

// /ai → /ai/chat 兼容重定向
export const AiRoot: RouteRecordRaw = {
  path: '/ai',
  name: 'aiRoot',
  redirect: { path: '/ai/chat' },
  meta: {
    title: 'menu.ai',
    menu: false,
  },
  children: [],
}
