<script setup lang="ts">
import type { AppDefinition } from '@yudream/yudream-webos-core'
import { computed, onMounted, ref } from 'vue'
import { useAppsStore } from '../../stores/apps'
import { useWindowsStore } from '../../stores/windows'
import YwIconTile from '../icon-tile/index.vue'

const props = withDefaults(defineProps<{
  /** 最大结果数 */
  limit?: number
}>(), {
  limit: 9,
})

const emit = defineEmits<{
  close: []
}>()

const appsStore = useAppsStore()
const windowsStore = useWindowsStore()

const keyword = ref('')
const activeIndex = ref(0)

/** 最近使用（localStorage） */
const RECENT_KEY = 'yw.quicklaunch.recent'
const recent = ref<string[]>(JSON.parse(localStorage.getItem(RECENT_KEY) ?? '[]'))

function pushRecent(key: string) {
  recent.value = [key, ...recent.value.filter(k => k !== key)].slice(0, 8)
  localStorage.setItem(RECENT_KEY, JSON.stringify(recent.value))
}

/** 过滤匹配（title + keywords，小写包含） */
const filtered = computed<AppDefinition[]>(() => {
  const kw = keyword.value.trim().toLowerCase()
  const pool = kw
    ? appsStore.apps.filter(a =>
        a.name.toLowerCase().includes(kw) || (a.keywords ?? '').toLowerCase().includes(kw))
    : [...appsStore.apps].sort((a, b) => {
        const ra = recent.value.indexOf(a.id)
        const rb = recent.value.indexOf(b.id)
        return (ra === -1 ? 999 : ra) - (rb === -1 ? 999 : rb)
      })
  return pool.slice(0, props.limit)
})

function open(app: AppDefinition) {
  pushRecent(app.id)
  appsStore.openApp(app.id)
  emit('close')
}

function onKeydown(ev: KeyboardEvent) {
  if (ev.key === 'ArrowDown') {
    ev.preventDefault()
    activeIndex.value = Math.min(activeIndex.value + 1, filtered.value.length - 1)
  }
  else if (ev.key === 'ArrowUp') {
    ev.preventDefault()
    activeIndex.value = Math.max(activeIndex.value - 1, 0)
  }
  else if (ev.key === 'Enter') {
    const app = filtered.value[activeIndex.value]
    if (app) {
      open(app)
    }
  }
  else if (ev.key === 'Escape') {
    emit('close')
  }
}

const inputEl = ref<HTMLInputElement | null>(null)
onMounted(() => inputEl.value?.focus())

/** 应用是否运行中 */
const running = computed(() => new Set(windowsStore.runningApps))
</script>

<template>
  <div class="yw-quicklaunch-overlay" @pointerdown.self="emit('close')">
    <div class="yw-quicklaunch" role="dialog" aria-label="快速启动">
      <input
        ref="inputEl"
        v-model="keyword"
        class="yw-quicklaunch-input"
        placeholder="搜索应用（名称 / 拼音首字母）…"
        @keydown="onKeydown"
      >

      <div v-if="filtered.length" class="yw-quicklaunch-list">
        <button
          v-for="(app, i) in filtered"
          :key="app.id"
          class="yw-quicklaunch-item"
          :class="{ 'is-active': i === activeIndex }"
          @mouseenter="activeIndex = i"
          @click="open(app)"
        >
          <YwIconTile :app-key="app.id" :icon="app.icon" :icon-bg="app.iconBg" :size="30" />
          <span class="yw-quicklaunch-title">{{ app.name }}</span>
          <span v-if="running.has(app.id)" class="yw-quicklaunch-running">运行中</span>
          <span v-if="recent.includes(app.id) && !keyword" class="yw-quicklaunch-recent">最近</span>
        </button>
      </div>

      <p v-else class="yw-quicklaunch-empty">
        没有匹配「{{ keyword }}」的应用
      </p>

      <footer class="yw-quicklaunch-hint">
        ↑↓ 选择 · Enter 打开 · Esc 关闭
      </footer>
    </div>
  </div>
</template>

<style scoped>
.yw-quicklaunch-overlay {
  position: fixed;
  inset: 0;
  z-index: 9500;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 14vh;
  background: rgb(0 0 0 / 30%);
  backdrop-filter: blur(6px);
  animation: yw-ql-in 0.14s ease;
}

@keyframes yw-ql-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

.yw-quicklaunch {
  width: min(560px, 92vw);
  overflow: hidden;
  background: oklch(var(--yw-popover) / 92%);
  border: 1px solid oklch(var(--yw-border));
  border-radius: var(--yw-radius-lg);
  box-shadow: 0 30px 80px -16px rgb(0 0 0 / 55%);
  backdrop-filter: blur(24px) saturate(1.4);
}

.yw-quicklaunch-input {
  box-sizing: border-box;
  width: 100%;
  padding: 16px 20px;
  font-size: 16px;
  color: oklch(var(--yw-popover-foreground));
  outline: none;
  background: transparent;
  border: none;
  border-bottom: 1px solid oklch(var(--yw-border));
}

.yw-quicklaunch-list {
  max-height: 46vh;
  padding: 8px;
  overflow: auto;
}

.yw-quicklaunch-item {
  display: flex;
  gap: 12px;
  align-items: center;
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  color: oklch(var(--yw-popover-foreground));
  text-align: left;
  cursor: default;
  background: transparent;
  border: none;
  border-radius: var(--yw-radius-md);
}

.yw-quicklaunch-item.is-active {
  background: oklch(var(--yw-accent) / 70%);
}

.yw-quicklaunch-title {
  flex: 1;
}

.yw-quicklaunch-running,
.yw-quicklaunch-recent {
  padding: 2px 8px;
  font-size: 11px;
  color: oklch(var(--yw-primary));
  border: 1px solid oklch(var(--yw-primary) / 50%);
  border-radius: 999px;
}

.yw-quicklaunch-recent {
  color: oklch(var(--yw-muted-foreground));
  border-color: oklch(var(--yw-border));
}

.yw-quicklaunch-empty {
  padding: 32px;
  margin: 0;
  font-size: 13px;
  color: oklch(var(--yw-muted-foreground));
  text-align: center;
}

.yw-quicklaunch-hint {
  padding: 10px 16px;
  font-size: 12px;
  color: oklch(var(--yw-muted-foreground));
  border-top: 1px solid oklch(var(--yw-border));
}
</style>
