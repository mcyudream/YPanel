import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/ai',
  component: Layout,
  name: 'ai',
  meta: {
    title: '智能',
    icon: 'yd:sparkles',
  },
  children: [
    {
      path: '',
      name: 'aiIndex',
      redirect: '/ai/chat',
      meta: { menu: false },
    },
    {
      path: 'chat',
      name: 'aiChat',
      component: () => import('@/views/ai/chat.vue'),
      meta: {
        title: '对话',
        icon: 'i-lucide:message-circle',
      },
    },
    {
      path: 'providers',
      name: 'aiProviders',
      component: () => import('@/views/ai/providers.vue'),
      meta: {
        title: '供应商',
        icon: 'i-lucide:plug',
      },
    },
    {
      path: 'knowledge',
      name: 'aiKnowledge',
      component: () => import('@/views/ai/knowledge.vue'),
      meta: {
        title: '知识库',
        icon: 'i-lucide:book-open',
      },
    },
    {
      path: 'workspace',
      name: 'aiWorkspace',
      component: () => import('@/views/ai/workspace.vue'),
      meta: {
        title: '工作空间',
        icon: 'i-lucide:folder-code',
      },
    },
    {
      path: 'memory',
      name: 'aiMemory',
      component: () => import('@/views/ai/memory.vue'),
      meta: {
        title: '记忆',
        icon: 'i-lucide:brain',
      },
    },
    {
      path: 'tools',
      name: 'aiTools',
      component: () => import('@/views/ai/tools.vue'),
      meta: {
        title: '系统工具',
        icon: 'i-lucide:wrench',
      },
    },
    {
      path: 'skills',
      name: 'aiSkills',
      component: () => import('@/views/ai/skills.vue'),
      meta: {
        title: '技能',
        icon: 'i-lucide:puzzle',
      },
    },
    {
      path: 'mcp',
      name: 'aiMcp',
      component: () => import('@/views/ai/mcp.vue'),
      meta: {
        title: 'MCP',
        icon: 'i-lucide:plug-zap',
      },
    },
  ],
}

export default routes
