<script setup lang="ts">
import { computed } from 'vue'
import { useWindowsStore } from '../../stores/windows'

const emit = defineEmits<{
  showLaunchpad: []
  showQuicklaunch: []
}>()

const windowsStore = useWindowsStore()

const tasks = computed(() => windowsStore.windows.filter(w => w.state !== 'minimized'))

const now = computed(() => new Date())

function fmt(d: Date) {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${p(d.getHours())}:${p(d.getMinutes())}`
}
</script>

<template>
  <footer class="yw-taskbar">
    <button class="yw-taskbar-start" title="启动台" @click="emit('showLaunchpad')">
      <i class="i-lucide-grid-3x3" />
    </button>

    <div class="yw-taskbar-tasks">
      <button
        v-for="win in tasks"
        :key="win.id"
        class="yw-taskbar-task"
        :class="{ 'is-focused': win.id === windowsStore.focusedId }"
        @click="windowsStore.focus(win.id)"
      >
        <i class="i-lucide-app-window" />
        <span>{{ win.title }}</span>
      </button>
    </div>

    <div class="yw-taskbar-tray">
      <button class="yw-taskbar-search" title="快速启动" @click="emit('showQuicklaunch')">
        <i class="i-lucide-search" />
      </button>
      <span class="yw-taskbar-clock">{{ fmt(now) }}</span>
    </div>
  </footer>
</template>

<style scoped>
.yw-taskbar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 9000;
  display: flex;
  gap: 8px;
  align-items: center;
  height: 40px;
  padding: 0 10px;
  background: oklch(var(--yw-glass) / 78%);
  border-top: 1px solid oklch(var(--yw-border) / 60%);
  backdrop-filter: blur(20px) saturate(1.4);
}

.yw-taskbar-start,
.yw-taskbar-search {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  font-size: 17px;
  color: oklch(var(--yw-glass-foreground));
  cursor: pointer;
  background: transparent;
  border: none;
  border-radius: 8px;
}

.yw-taskbar-start:hover,
.yw-taskbar-search:hover {
  background: oklch(var(--yw-accent) / 60%);
}

.yw-taskbar-tasks {
  display: flex;
  flex: 1;
  gap: 4px;
  overflow: hidden;
}

.yw-taskbar-task {
  display: flex;
  gap: 6px;
  align-items: center;
  max-width: 180px;
  padding: 5px 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  color: oklch(var(--yw-glass-foreground));
  white-space: nowrap;
  cursor: pointer;
  background: transparent;
  border: none;
  border-radius: 8px;
}

.yw-taskbar-task:hover {
  background: oklch(var(--yw-accent) / 60%);
}

.yw-taskbar-task.is-focused {
  background: oklch(var(--yw-accent) / 85%);
  box-shadow: inset 0 -2px 0 oklch(var(--yw-primary));
}

.yw-taskbar-tray {
  display: flex;
  gap: 10px;
  align-items: center;
  font-size: 12px;
  color: oklch(var(--yw-glass-foreground));
}
</style>
