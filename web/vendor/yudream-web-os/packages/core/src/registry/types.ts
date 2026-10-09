import type { Size } from '@yudream/yudream-webos-shared'
import type { YwBusEvents, YwEventBus } from '../events/bus'
import type { MenuContribution, MenuItem } from '../menu/menu-model'
import type { WidgetDefinition } from '../widgets/widget-model'

/** 图标：图片 URL / data URI / iconify 类名（Vue 组件形式由 apps 层承载） */
export type AppIcon = string

/** 应用定义（内置应用与宿主应用走同一条注册通道） */
export interface AppDefinition {
  /** 全局唯一 id，如 'finder'、'my-project.crm'（kebab-case） */
  id: string
  name: string
  icon: AppIcon
  /** 图标底座背景（CSS background 值）；缺省按 id 从色板稳定取色 */
  iconBg?: string
  version?: string
  /** 启动台分类 */
  category?: string
  keywords?: string
  /** 应用主组件：由 ui 层承载（core 保持框架无关，这里为任意载荷） */
  component?: unknown

  // ---- 窗口行为 ----
  singleton?: boolean
  /** 非单例时允许多实例（窗口标题追加 #2 #3…） */
  multiInstance?: boolean
  defaultSize?: Size
  minSize?: Size
  maxSize?: Size
  /** 默认 true */
  resizable?: boolean
  /** 无系统标题栏（应用自绘） */
  frameless?: boolean
  defaultPosition?: { x: number, y: number } | 'center' | 'cascade'

  // ---- 桌面生态挂载点 ----
  /** 应用聚焦时的顶部菜单 */
  menus?: MenuContribution
  dock?: {
    /** 默认 true */
    showInDock?: boolean
    contextMenu?: MenuItem[]
  }
  launchpad?: { show?: boolean, order?: number }
  /** 应用附带的小组件 */
  widgets?: WidgetDefinition[]
  /** 声明可打开的文件类型（Finder 双击路由），如 ['txt', 'md'] */
  fileHandlers?: string[]

  /** 桌面图标挂载点（Dock/启动台不受此控，分别看 dock.showInDock / launchpad.show） */
  desktop?: {
    /** 默认 true：播种到桌面图标网格；false = 仅经 Dock/启动台/搜索可达（适合低频应用） */
    show?: boolean
    contextMenu?: MenuItem[]
  }

  // ---- 生命周期（ui/宿主侧注入，core 仅存储） ----
  onLaunch?: (ctx: AppLaunchContext) => void
  onTerminate?: () => void
}

/** 应用启动上下文 */
export interface AppLaunchContext {
  appId: string
  /** 打开参数（如待打开文件路径） */
  launchOptions?: Record<string, unknown>
  bus: YwEventBus
  events: YwBusEvents
}
