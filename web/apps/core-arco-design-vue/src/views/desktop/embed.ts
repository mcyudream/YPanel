// 桌面工作台嵌入上下文：经典页面组件被 webos 窗口承载时注入。
// 页面据此切换交互形态（router.push 跳详情 → 开新窗；返回 → 关自己窗），经典模式保持原行为。
import type { InjectionKey } from 'vue'
import { inject } from 'vue'

export interface YwEmbedContext {
  /** 打开一个应用窗口（launchOptions 随窗口实例传递） */
  openApp: (appId: string, opts?: { title?: string, launchOptions?: Record<string, unknown> }) => void
  /** 关闭指定窗口（配合自身窗承载：返回按钮=关自己） */
  closeWindow: (windowId: string) => void
}

export const YwEmbedKey: InjectionKey<YwEmbedContext> = Symbol('ypanel-webos-embed')

/** 页面内取嵌入上下文；经典模式返回 null */
export function useYwEmbed(): YwEmbedContext | null {
  return inject(YwEmbedKey, null)
}

/**
 * 嵌入态组件定位「自己所在窗口」：向上找最近的 .yw-window 取 data-window-id。
 * 供承载组件（App wrapper）把自身窗 id 传给业务页的返回/关闭逻辑。
 */
export function closestWindowId(el: HTMLElement | undefined | null): string | null {
  return el?.closest?.('.yw-window')?.getAttribute('data-window-id') ?? null
}
