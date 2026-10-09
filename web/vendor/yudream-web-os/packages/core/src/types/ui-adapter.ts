/**
 * UI 适配器接口（类型定义在 core，框架无关）。
 * vue 层提供内置轻量实现；arco 包提供 Arco Design Vue 实现；宿主可自定义。
 * 所有方法为命令式 API，不依赖任何框架组件类型。
 */

export interface YwMenuItem {
  separator?: boolean
  label: string
  icon?: string
  danger?: boolean
  disabled?: boolean
  divider?: boolean
  onSelect?: () => void
  children?: YwMenuItem[]
}

export interface YwMenuOptions {
  x: number
  y: number
  items: YwMenuItem[]
}

export interface YwMenuHandle {
  close: () => void
}

export interface YwConfirmOptions {
  title: string
  content?: string
  okText?: string
  cancelText?: string
  danger?: boolean
}

export interface YwAlertOptions {
  title: string
  content?: string
  okText?: string
}

export type YwMessageType = 'success' | 'error' | 'warning' | 'info'

export interface YwUiAdapter {
  name: string
  /** 右键/上下文菜单 */
  menu: (options: YwMenuOptions) => YwMenuHandle
  /** 确认对话框，resolve 用户选择 */
  confirm: (options: YwConfirmOptions) => Promise<boolean>
  /** 警告/信息对话框 */
  alert: (options: YwAlertOptions) => Promise<void>
  /** 轻提示 */
  message: (type: YwMessageType, content: string) => void
}
