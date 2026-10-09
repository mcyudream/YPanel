<script setup lang="ts">
import type { YwSettingsState } from './system/settings'
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { provideUIRegistry, useUIRegistry } from './override/registry'
import { useWebOS } from './provider'
import { shortcuts } from './shortcuts'
import { applySettings } from './system/settings'

/**
 * WebOSProvider — 桌面环境根容器。
 * 职责：视口跟踪、主题应用（dark class/accent）、快捷键分发、uiRegistry 透传、默认插槽渲染完整桌面。
 */
const props = withDefaults(defineProps<{
  /** 主题模式初始值（settings state 已由 createWebOS 构造） */
  wallpaper?: { src: string, fit?: 'cover' | 'contain' | 'tile' } | null
}>(), {
  wallpaper: null,
})

const os = useWebOS()
const uiRegistryCtx = useUIRegistry()
// 合并全局 uiRegistry（provide 供 ui-arco 层解析）
provideUIRegistry(uiRegistryCtx.overrides)

const settings: YwSettingsState = os.settings

const effectiveDark = computed(() =>
  settings.mode === 'system' ? settings.systemDark : settings.mode === 'dark',
)

const wallpaperStyle = computed(() => {
  const w = props.wallpaper ?? settings.wallpaper
  if (!w) {
    return { background: 'linear-gradient(160deg, #0b3b66 0%, #1265a8 38%, #2d8fd0 68%, #6db6e8 100%)' }
  }
  if (w.src.startsWith('linear-gradient') || w.src.startsWith('radial-gradient')) {
    return { background: w.src }
  }
  return {
    backgroundImage: `url(${w.src})`,
    backgroundSize: w.fit === 'tile' ? 'auto' : (w.fit ?? 'cover'),
    backgroundRepeat: w.fit === 'tile' ? 'repeat' : 'no-repeat',
    backgroundPosition: 'center',
  }
})

let mediaQuery: MediaQueryList | null = null
let detachShortcuts: (() => void) | null = null

function apply() {
  applySettings(settings)
  os.bus.emit('webos:system:theme-change', { mode: effectiveDark.value ? 'dark' : 'light' })
}

function syncViewport() {
  os.wm.setViewport({ width: window.innerWidth, height: window.innerHeight, menubarHeight: 24 })
}

onMounted(() => {
  syncViewport()
  window.addEventListener('resize', syncViewport)
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  settings.systemDark = mediaQuery.matches
  const onMedia = (e: MediaQueryListEvent) => {
    settings.systemDark = e.matches
    apply()
  }
  mediaQuery.addEventListener('change', onMedia)
  apply()
  detachShortcuts = shortcuts.attach(window)
  // 会话恢复（可选，scope: windows.session）
  if (os.config.windows.sessionRestore) {
    void os.persist.get<unknown[]>(`windows.session`).then((snaps) => {
      if (snaps?.length) {
        os.wm.restoreSession(snaps as never)
      }
    })
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', syncViewport)
  mediaQuery?.removeEventListener('change', () => {})
  detachShortcuts?.()
})

defineExpose({ apply, os })
</script>

<template>
  <div class="yw-provider" :style="wallpaperStyle">
    <slot />
  </div>
</template>

<style scoped>
.yw-provider {
  position: fixed;
  inset: 0;
  overflow: hidden;
  font-family: var(--yw-font-ui, inherit);
}
</style>
