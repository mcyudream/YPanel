// i18n（B26 → B26-full）：中英双语；缺失键回落 zh-CN。
// 语言包按域拆分在 lang/<locale>/*.ts；语言持久化在后端 panel.language + localStorage。
// 切换入口：顶栏 LocaleSwitch；启动回读见 App.vue（登录后拉取 settings 注入 initLocale）。
import { createI18n } from 'vue-i18n'
import zhCN from './lang/zh-CN'
import enUS from './lang/en-US'

export const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackLocale: 'zh-CN',
  missingWarn: false,
  fallbackWarn: false,
  messages: { 'zh-CN': zhCN, 'en-US': enUS },
})

type Locale = 'zh-CN' | 'en-US'

// SetLocale 切换并持久化（后端 + localStorage）。
export async function setLocale(locale: Locale, persist = true) {
  i18n.global.locale.value = locale
  localStorage.setItem('ypanel.locale', locale)
  if (persist) {
    try {
      const { default: api } = await import('@/api/index')
      await api.put('api/v1/settings', { 'panel.language': locale === 'en-US' ? 'en' : 'zh' })
    }
    catch {}
  }
}

// InitLocale 启动时恢复（localStorage 优先，其次后端 panel.language 由 App.vue 登录后注入）。
export function initLocale(saved?: string) {
  const v = localStorage.getItem('ypanel.locale') || (saved === 'en' ? 'en-US' : saved === 'zh' ? 'zh-CN' : '') || 'zh-CN'
  i18n.global.locale.value = v as Locale
}

// tr 动态键兜底：词条存在则翻译，否则原样返回 fallback（用于后端枚举值等不可控键）。
export function tr(key: string, fallback?: string) {
  return i18n.global.te(key) ? i18n.global.t(key) : (fallback ?? key)
}
