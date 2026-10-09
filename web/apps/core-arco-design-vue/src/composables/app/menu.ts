import { i18n } from '@/locales'

export function useAppMenu() {
  const router = useRouter()

  const appSettingsStore = useAppSettingsStore()
  const appMenuStore = useAppMenuStore()

  function generateTitle(title?: string | (() => any)) {
    if (typeof title === 'function') {
      return title()
    }
    // meta.title 存 i18n key（menu.*）：存在词条则翻译，普通文本原样透传
    if (typeof title === 'string' && title && i18n.global.te(title)) {
      return i18n.global.t(title)
    }
    return title ?? i18n.global.t('layout.untitled')
  }

  function switchTo(index: number) {
    appMenuStore.setActived(index)
    if (
      appSettingsStore.settings.menu.mainMenuClickMode === 'jump'
      || (appSettingsStore.settings.menu.mainMenuClickMode === 'smart' && appMenuStore.sidebarMenusHasOnlyMenu)
    ) {
      router.push(appMenuStore.sidebarMenusFirstDeepestPath)
    }
  }

  return {
    generateTitle,
    switchTo,
  }
}
