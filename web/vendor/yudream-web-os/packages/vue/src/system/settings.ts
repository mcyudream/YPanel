import type { YwBusEvents } from '@yudream/yudream-webos-core'
import { darkTheme, hexToOklchChannels, lightTheme, mergeTheme } from '@yudream/yudream-webos-core'
import { reactive } from 'vue'

/** 系统设置响应式状态（壁纸/主题/强调色） */
export interface YwSettingsState {
  mode: 'light' | 'dark' | 'system'
  systemDark: boolean
  accent: string | null
  wallpaper: { src: string, fit?: 'cover' | 'contain' | 'tile' } | null
}

export function createSettingsState(): YwSettingsState {
  return reactive({
    mode: 'system',
    systemDark: false,
    accent: null,
    wallpaper: null,
  })
}

let styleEl: HTMLStyleElement | null = null

function ensureStyleEl(): HTMLStyleElement {
  if (!styleEl) {
    styleEl = document.createElement('style')
    styleEl.setAttribute('data-yw-theme', '')
    document.head.appendChild(styleEl)
  }
  return styleEl
}

/** 主题 token 注入（:root 亮 / .dark 暗） */
export function injectThemeTokens(light: Record<string, string>, dark: Record<string, string>) {
  const css = (tokens: Record<string, string>) =>
    Object.entries(tokens)
      .map(([k, v]) => `${k}: ${v}`)
      .join('; ')
  ensureStyleEl().textContent = `:root{${css(light)}} .dark{${css(dark)}}`
}

/** 应用设置到 DOM（dark class / colorScheme / accent） */
export function applySettings(state: YwSettingsState) {
  injectThemeTokens(mergeTheme(lightTheme), mergeTheme(darkTheme))
  const root = document.documentElement
  const dark = state.mode === 'system' ? state.systemDark : state.mode === 'dark'
  root.classList.toggle('dark', dark)
  root.style.colorScheme = dark ? 'dark' : 'light'
  if (state.accent) {
    const channels = state.accent.startsWith('#') ? hexToOklchChannels(state.accent) : state.accent
    root.style.setProperty('--yw-primary', channels)
    root.style.setProperty('--yw-ring', channels)
  }
  else {
    root.style.removeProperty('--yw-primary')
    root.style.removeProperty('--yw-ring')
  }
}

/** 设置变更 → 总线（theme-change / wallpaper-change） */
export function settingsEventShim() {
  void (0 as unknown as YwBusEvents)
}
