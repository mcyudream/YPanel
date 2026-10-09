<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useWindowsStore } from '../../stores/windows'

withDefaults(defineProps<{
  /** 应用名（加粗显示，macOS 惯例） */
  title?: string
  showLogo?: boolean
  /** 显示内置实时时钟（右侧托盘） */
  showClock?: boolean
  /** 无 menus 插槽时显示的占位菜单名 */
  fallbackMenus?: string[]
}>(), {
  title: '',
  showLogo: true,
  showClock: true,
  fallbackMenus: () => ['文件', '编辑', '显示', '窗口', '帮助'],
})

const emit = defineEmits<{
  logoclick: []
  menuclick: [name: string]
}>()

const windowsStore = useWindowsStore()

/** 有窗口时菜单栏变实底毛玻璃（macOS：壁纸直出透明，窗口内容滚到下方时起雾） */
const isSolid = computed(() => windowsStore.windows.length > 0)

/* 实时时钟（30s 粒度足够分钟级显示） */
const now = ref(new Date())
let clockTimer: ReturnType<typeof setInterval> | null = null

const clockText = computed(() => {
  const d = now.value
  const week = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][d.getDay()]
  const p = (n: number) => String(n).padStart(2, '0')
  return `${week} ${d.getMonth() + 1}月${d.getDate()}日 ${p(d.getHours())}:${p(d.getMinutes())}`
})

onMounted(() => {
  clockTimer = setInterval(() => {
    now.value = new Date()
  }, 30_000)
})

onBeforeUnmount(() => {
  if (clockTimer) {
    clearInterval(clockTimer)
  }
})
</script>

<template>
  <header class="yw-menubar" :class="{ 'is-solid': isSolid }">
    <button v-if="showLogo" class="yw-menubar-logo" title="系统菜单" @click="emit('logoclick')">
      <i class="i-lucide-command" />
    </button>
    <span v-if="title" class="yw-menubar-title">{{ title }}</span>

    <div class="yw-menubar-menus">
      <slot name="menus">
        <button
          v-for="name in fallbackMenus"
          :key="name"
          class="yw-menubar-item"
          @click="emit('menuclick', name)"
        >
          {{ name }}
        </button>
      </slot>
    </div>

    <div class="yw-menubar-tray">
      <slot name="tray">
        <i class="yw-menubar-tray-icon i-lucide-wifi" title="Wi-Fi" />
        <i class="yw-menubar-tray-icon i-lucide-battery-medium" title="电池" />
        <i class="yw-menubar-tray-icon i-lucide-search" title="聚焦搜索" />
      </slot>
      <span v-if="showClock" class="yw-menubar-clock">{{ clockText }}</span>
    </div>
  </header>
</template>

<style scoped>
.yw-menubar {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 9600;
  display: flex;
  align-items: center;
  height: var(--yw-menubar-h);
  padding: 0 10px;
  font-size: 13px;
  font-weight: 500;
  color: #fff;
  text-shadow: 0 0 3px rgb(0 0 0 / 35%);
  background: linear-gradient(180deg, rgb(0 0 0 / 22%), rgb(0 0 0 / 8%));
  backdrop-filter: blur(var(--yw-blur-light));
  transition:
    background var(--yw-dur-ui),
    backdrop-filter var(--yw-dur-ui),
    color var(--yw-dur-ui),
    text-shadow var(--yw-dur-ui);
}

/* 有窗口内容时起雾 */
.yw-menubar.is-solid {
  color: oklch(var(--yw-glass-foreground));
  text-shadow: none;
  background: var(--yw-glass-menubar-solid);
  backdrop-filter: blur(var(--yw-blur-medium)) saturate(150%);
}

.yw-menubar-logo {
  display: flex;
  align-items: center;
  padding: 2px 10px;
  font-size: 16px;
  color: inherit;
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 4px;
}

.yw-menubar-title {
  margin-right: 10px;
  font-weight: 700;
}

.yw-menubar-menus {
  display: flex;
  flex: 1;
  gap: 2px;
  align-items: stretch;
}

.yw-menubar-item {
  display: inline-flex;
  align-items: center;
  height: 100%;
  padding: 0 12px;
  font-size: inherit;
  font-weight: inherit;
  color: inherit;
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 4px;
}

.yw-menubar-logo:hover,
.yw-menubar-item:hover {
  background: rgb(255 255 255 / 22%);
}

.yw-menubar.is-solid .yw-menubar-logo:hover,
.yw-menubar.is-solid .yw-menubar-item:hover {
  background: oklch(var(--yw-foreground) / 12%);
}

.yw-menubar-tray {
  display: flex;
  gap: 14px;
  align-items: center;
  margin-left: auto;
  font-size: var(--yw-fs-small);
  font-weight: 500;
}

.yw-menubar-tray-icon {
  font-size: 14px;
  opacity: 0.9;
}

.yw-menubar-clock {
  min-width: 118px;
  font-variant-numeric: tabular-nums;
  text-align: right;
}
</style>
