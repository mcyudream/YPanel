import type { GridPosition } from '@yudream/yudream-webos-shared'
import { shortId } from '@yudream/yudream-webos-shared'

/** 小组件尺寸：2x2 / 4x2 / 4x4 网格 */
export type WidgetSize = 'small' | 'medium' | 'large'

/** 小组件放置：桌面右侧栏（macOS 15）或嵌入桌面网格 */
export type WidgetPlacement = 'sidebar' | 'desktop'

/** 小组件定义（随应用注册） */
export interface WidgetDefinition {
  id: string
  name: string
  sizes: WidgetSize[]
  /** 组件由 ui 层承载（core 框架无关） */
  component?: unknown
  /** 配置项 schema，设置面板据此自动渲染表单 */
  configSchema?: WidgetConfigField[]
}

export interface WidgetConfigField {
  key: string
  label: string
  type: 'text' | 'number' | 'boolean' | 'select'
  default?: unknown
  options?: Array<{ label: string, value: unknown }>
}

/** 小组件实例 */
export interface WidgetInstance {
  instanceId: string
  widgetId: string
  size: WidgetSize
  position: GridPosition
  config: Record<string, unknown>
}

/** 小组件注册表 + 实例布局（编辑模式状态由 core 管，动画在 ui 层） */
export class WidgetStore {
  private definitions = new Map<string, WidgetDefinition>()
  private instances: WidgetInstance[] = []
  private listeners = new Set<(reason: string) => void>()

  editing = false
  placement: WidgetPlacement = 'sidebar'

  constructor(instances?: WidgetInstance[]) {
    this.instances = [...(instances ?? [])]
  }

  onChange(fn: (reason: string) => void): () => void {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  register(def: WidgetDefinition): void {
    this.definitions.set(def.id, def)
  }

  getDefinition(widgetId: string): WidgetDefinition | undefined {
    return this.definitions.get(widgetId)
  }

  listDefinitions(): WidgetDefinition[] {
    return [...this.definitions.values()]
  }

  listInstances(): WidgetInstance[] {
    return [...this.instances]
  }

  /** 持久化恢复：整体替换实例布局 */
  loadInstances(instances: WidgetInstance[]): void {
    this.instances = instances.map(x => ({ ...x, config: { ...x.config } }))
    this.notify('restore')
  }

  /** addInstance 别名（composables 便捷入口） */
  add(widgetId: string, size: WidgetSize, position: GridPosition, config?: Record<string, unknown>): WidgetInstance | null {
    return this.addInstance(widgetId, size, position, config)
  }

  addInstance(widgetId: string, size: WidgetSize, position: GridPosition, config: Record<string, unknown> = {}): WidgetInstance | null {
    const def = this.definitions.get(widgetId)
    if (!def || !def.sizes.includes(size)) {
      return null
    }
    const inst: WidgetInstance = { instanceId: shortId('widget'), widgetId, size, position, config }
    this.instances.push(inst)
    this.notify('add')
    return inst
  }

  removeInstance(instanceId: string): void {
    const idx = this.instances.findIndex(x => x.instanceId === instanceId)
    if (idx > -1) {
      this.instances.splice(idx, 1)
      this.notify('remove')
    }
  }

  moveInstance(instanceId: string, position: GridPosition): void {
    const inst = this.instances.find(x => x.instanceId === instanceId)
    if (inst) {
      inst.position = { ...position }
      this.notify('move')
    }
  }

  resizeInstance(instanceId: string, size: WidgetSize): void {
    const inst = this.instances.find(x => x.instanceId === instanceId)
    const def = inst && this.definitions.get(inst.widgetId)
    if (inst && def?.sizes.includes(size)) {
      inst.size = size
      this.notify('resize')
    }
  }

  setConfig(instanceId: string, config: Record<string, unknown>): void {
    const inst = this.instances.find(x => x.instanceId === instanceId)
    if (inst) {
      inst.config = { ...inst.config, ...config }
      this.notify('config')
    }
  }

  setEditing(editing: boolean): void {
    this.editing = editing
    this.notify('editing')
  }

  private notify(reason: string) {
    this.listeners.forEach(fn => fn(reason))
  }
}
