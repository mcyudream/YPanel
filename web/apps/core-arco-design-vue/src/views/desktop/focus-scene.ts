// 桌面工作台 AI 场景感知：跟踪最近聚焦的工作窗口，翻译成经典面板场景路径 + 聚焦对象。
// 后端按 ?scene= 路径前缀注入实时数据摘要与场景工具模块；桌面模式窗口不换路由，
// 链路在此接回（desktop 壳写、chat.vue embed 态读）。经典模式不经此文件。
import { ref } from 'vue'

export interface FocusSceneInfo {
  windowId: string
  appId: string
  title: string
  launchOptions: Record<string, unknown> | undefined
}

/** 最近聚焦的工作窗口（AI/设置窗除外）；焦点落到桌面时保留——切走前看的窗口仍是对话最佳上下文 */
export const lastFocusedWin = ref<FocusSceneInfo | null>(null)

/** 应用 id → 场景路径（与后端 SceneContext/sceneAIModules 的路径前缀约定对齐；未知域回退概览） */
export function scenePathOfApp(appId: string): string {
  switch (appId) {
    case 'overview':
      return '/'
    case 'container':
    case 'container-list':
    case 'container-detail':
    case 'container-app-detail':
      return '/container'
    case 'database':
      return '/database'
    case 'sites':
    case 'site-detail':
      return '/sites'
    case 'certs':
      return '/certs'
    case 'runtimes':
      return '/runtimes'
    case 'docker-images':
    case 'docker-networks':
    case 'docker-volumes':
    case 'docker-settings':
      return '/docker'
    case 'file':
    case 'terminal':
    case 'processes':
      return '/system'
    case 'store':
      return '/store'
    case 'cron':
      return '/tools'
    case 'monitor':
      return '/monitor'
    default:
      return '/'
  }
}

/** 焦点窗携带的实体提示（容器名/编排项目/目录等），无则空串 */
export function focusHintOf(win: FocusSceneInfo): string {
  const lo = win.launchOptions || {}
  for (const key of ['name', 'project', 'dir']) {
    const v = lo[key]
    if (typeof v === 'string' && v.trim()) {
      return v.trim()
    }
  }
  return ''
}
