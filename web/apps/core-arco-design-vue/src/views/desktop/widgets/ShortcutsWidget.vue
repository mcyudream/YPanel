<script setup lang="ts">
// 快捷入口小组件（small/medium）：高频应用 2×2 磁贴，点击直接开窗（经桌面嵌入上下文）。
// 磁贴视觉对齐桌面图标：渐变底 + 白色图标 + 名称，hover 提亮。
import { useYwEmbed } from '@/views/desktop/embed'
import { i18n } from '@/locales'
import { computed } from 'vue'

const embed = useYwEmbed()

const shortcuts = [
  { id: 'terminal', name: i18n.global.t('desktop.shortcuts.terminal'), icon: 'i-lucide-square-terminal', bg: 'linear-gradient(135deg, #0ea5e9 0%, #2563eb 100%)' },
  { id: 'file', name: i18n.global.t('desktop.shortcuts.file'), icon: 'i-lucide-folder-open', bg: 'linear-gradient(135deg, #34d399 0%, #059669 100%)' },
  { id: 'container', name: i18n.global.t('desktop.shortcuts.container'), icon: 'i-lucide-container', bg: 'linear-gradient(135deg, #60a5fa 0%, #4f46e5 100%)' },
  { id: 'sites', name: i18n.global.t('desktop.shortcuts.sites'), icon: 'i-lucide-globe', bg: 'linear-gradient(135deg, #a78bfa 0%, #7c3aed 100%)' },
] as const

const tiles = computed(() => shortcuts)

function open(id: string) {
  if (embed) {
    const app = tiles.value.find(t => t.id === id)
    embed.openApp(id, { title: app?.name })
  }
}
</script>

<template>
  <div class="quick-body">
    <div class="grid">
      <button
        v-for="t in tiles"
        :key="t.id"
        class="tile"
        :title="$t('desktop.shortcuts.open', { n: t.name })"
        @click="open(t.id)"
      >
        <span class="tile-icon" :style="{ background: t.bg }">
          <i :class="t.icon" />
        </span>
        <span class="tile-label">{{ t.name }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.quick-body {
  height: 100%;
}

.grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 6px;
  height: 100%;
}

.tile {
  display: flex;
  flex-direction: column;
  gap: 5px;
  align-items: center;
  justify-content: center;
  padding: 6px 2px;
  text-align: center;
  cursor: pointer;
  background: oklch(var(--yw-foreground) / 5%);
  border: none;
  border-radius: 10px;
  transition:
    background var(--yw-dur-ui, 0.15s),
    transform var(--yw-dur-ui, 0.15s);
}

.tile:hover {
  background: oklch(var(--yw-foreground) / 12%);
}

.tile:active {
  transform: scale(0.96);
}

.tile-icon {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  font-size: 17px;
  color: #fff;
  border-radius: 9px;
  box-shadow: 0 2px 6px rgb(0 0 0 / 18%);
}

.tile-label {
  font-size: 10px;
  color: var(--yw-label);
}
</style>
