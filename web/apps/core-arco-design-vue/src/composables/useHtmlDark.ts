import type { Ref } from 'vue'
import { onBeforeUnmount, onMounted, ref } from 'vue'

/**
 * 跟随实际生效的暗色状态（html.dark）。
 * 经典面板由 fa settings 驱动、桌面工作台由 webos 设置驱动——两套状态都切换 html.dark，
 * 统一监听 class 变化即可两模式通用（避免组件各自读自家 store 导致脱钩）。
 */
export function useHtmlDark(): Ref<boolean> {
  const isDark = ref(document.documentElement.classList.contains('dark'))
  let observer: MutationObserver | null = null

  onMounted(() => {
    observer = new MutationObserver(() => {
      isDark.value = document.documentElement.classList.contains('dark')
    })
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  })

  onBeforeUnmount(() => {
    observer?.disconnect()
  })

  return isDark
}
