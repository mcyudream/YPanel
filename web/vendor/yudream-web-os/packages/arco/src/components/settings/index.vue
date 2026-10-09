<script setup lang="ts">
import type { WindowInstance as WindowState } from '@yudream/yudream-webos-core'
import { YW_CORE_VERSION } from '@yudream/yudream-webos-core'
import { useWebOS } from '@yudream/yudream-webos-vue'
import { computed, ref } from 'vue'
import { useAppsStore } from '../../stores/apps'
import { useThemeStore } from '../../stores/theme'
import { useWindowsStore } from '../../stores/windows'

/**
 * YwSettingsApp — 内置系统设置应用。
 * 布局与视觉逐行对照 macOS System Settings（参考站规格迁移）：
 * 玻璃侧栏(216px, radius 12, margin 10px) + 640px 居中内容列 + control-bg 圆角卡片组 + 0.5px 分隔行。
 */
const props = withDefaults(defineProps<{
  win?: WindowState
  /** 启用的分区（按此顺序渲染侧栏） */
  sections?: Array<'appearance' | 'wallpaper' | 'dock' | 'about'>
  /** 侧栏用户卡 */
  user?: { name: string, subtitle?: string } | null
}>(), {
  sections: () => ['appearance', 'wallpaper', 'dock', 'about'],
  user: null,
})

const theme = useThemeStore()
const apps = useAppsStore()
const windows = useWindowsStore()

const SECTION_META = {
  appearance: { label: '外观', icon: 'i-lucide-paintbrush', from: '#3A3A3C', to: '#1C1C1E' },
  wallpaper: { label: '壁纸', icon: 'i-lucide-image', from: '#30B0C7', to: '#12688E' },
  dock: { label: '桌面与 Dock', icon: 'i-lucide-layout-grid', from: '#5FB3F9', to: '#1263E9' },
  about: { label: '关于', icon: 'i-lucide-info', from: '#8E8E93', to: '#48484A' },
} as const

const active = ref<'appearance' | 'wallpaper' | 'dock' | 'about'>(props.sections[0] ?? 'appearance')

/* 外观 */
const modeOptions = [
  { value: 'light', label: '浅色' },
  { value: 'dark', label: '深色' },
  { value: 'auto', label: '自动' },
] as const

/* 强调色（苹果系，22px 圆点） */
const accents = [
  { value: null, color: '#007AFF', label: '蓝色' },
  { value: '#A550A7', color: '#A550A7', label: '紫色' },
  { value: '#F74F9E', color: '#F74F9E', label: '粉色' },
  { value: '#FF5257', color: '#FF5257', label: '红色' },
  { value: '#F7821B', color: '#F7821B', label: '橙色' },
  { value: '#FFC600', color: '#FFC600', label: '黄色' },
  { value: '#62BA46', color: '#62BA46', label: '绿色' },
  { value: '#8E8E93', color: '#8E8E93', label: '石墨' },
]

/* 壁纸预设 */
const wallpapers = [
  { label: '海景', src: 'https://images.unsplash.com/photo-1439405326854-014607f694d7?w=2560&q=80', thumb: 'https://images.unsplash.com/photo-1439405326854-014607f694d7?w=320&q=60' },
  { label: '山脉', src: 'https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=2560&q=80', thumb: 'https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=320&q=60' },
  { label: '湖泊', src: 'https://images.unsplash.com/photo-1439066615861-d1af74d74000?w=2560&q=80', thumb: 'https://images.unsplash.com/photo-1439066615861-d1af74d74000?w=320&q=60' },
  { label: '极光', src: 'https://images.unsplash.com/photo-1483347756197-71ef80e95f73?w=2560&q=80', thumb: 'https://images.unsplash.com/photo-1483347756197-71ef80e95f73?w=320&q=60' },
  { label: '暮色', src: 'linear-gradient(160deg, #0b3b66 0%, #1265a8 38%, #2d8fd0 68%, #6db6e8 100%)', thumb: 'linear-gradient(160deg, #0b3b66 0%, #1265a8 38%, #2d8fd0 68%, #6db6e8 100%)' },
  { label: '无', src: '', thumb: '' },
]

const currentWallpaperSrc = computed(() => theme.wallpaper?.src ?? '')

function pickWallpaper(src: string) {
  theme.setWallpaper(src ? { src } : null)
}

/* Dock 管理 */
const { dock } = useWebOS()
const dockPinned = computed(() => dock.pinned.map((id: string) => apps.all[id]).filter((a): a is NonNullable<typeof a> => Boolean(a)))

function unpin(key: string) {
  dock.unpin(key)
}

function pin(key: string) {
  dock.pin(key)
}

const runningKeys = computed(() => new Set(windows.runningApps))
</script>

<template>
  <div class="yw-settings">
    <!-- 玻璃侧栏（216px 圆角浮动面板，规格照抄参考站 glass-sidebar） -->
    <aside class="yw-settings-sidebar">
      <div v-if="user" class="yw-settings-user" :class="{ 'is-active': active === 'about' }" @click="active = 'about'">
        <span class="yw-settings-avatar">{{ user.name.slice(0, 1) }}</span>
        <span class="yw-settings-user-meta">
          <b>{{ user.name }}</b>
          <small v-if="user.subtitle">{{ user.subtitle }}</small>
        </span>
      </div>

      <nav class="yw-settings-nav">
        <button
          v-for="s in sections"
          :key="s"
          class="yw-settings-nav-item"
          :class="{ 'is-active': active === s }"
          @click="active = s"
        >
          <span class="yw-settings-nav-icon" :style="{ background: `linear-gradient(180deg, ${SECTION_META[s].from}, ${SECTION_META[s].to})` }">
            <i :class="SECTION_META[s].icon" />
          </span>
          {{ SECTION_META[s].label }}
        </button>
      </nav>
    </aside>

    <!-- 内容：640px 居中列 -->
    <div class="yw-settings-body">
      <div class="yw-settings-pane">
        <!-- 外观 -->
        <template v-if="active === 'appearance'">
          <h1 class="yw-settings-h1">
            外观
          </h1>

          <div class="yw-settings-card">
            <div class="yw-settings-modes">
              <button
                v-for="m in modeOptions"
                :key="m.value"
                class="yw-settings-mode"
                @click="theme.setMode(m.value === 'auto' ? 'system' : m.value); theme.apply()"
              >
                <span class="yw-settings-mode-art" :class="[`is-${m.value}`, { 'is-active': theme.mode === m.value || (m.value === 'auto' && theme.mode === 'system') }]">
                  <span class="art-chrome"><i /><i /><i /></span>
                  <span class="art-pane art-pane--light">
                    <span class="art-doc">
                      <i class="art-line" style="width: 55%;" />
                      <i class="art-line" style="width: 70%;" />
                      <i class="art-line" style="width: 45%;" />
                    </span>
                  </span>
                  <span v-if="m.value !== 'light'" class="art-pane art-pane--dark">
                    <span class="art-doc">
                      <i class="art-line" style="width: 55%;" />
                      <i class="art-line" style="width: 70%;" />
                      <i class="art-line" style="width: 45%;" />
                    </span>
                  </span>
                </span>
                <span class="yw-settings-mode-label">{{ m.label }}</span>
              </button>
            </div>
          </div>

          <div class="yw-settings-label">
            强调色
          </div>
          <div class="yw-settings-card">
            <div class="yw-settings-accents">
              <button
                v-for="a in accents"
                :key="a.label"
                class="yw-settings-accent"
                :class="{ 'is-active': theme.accent === a.value }"
                :style="{ background: a.color }"
                :title="a.label"
                @click="theme.setAccent(a.value)"
              >
                <i class="i-lucide-check" />
              </button>
            </div>
          </div>
        </template>

        <!-- 壁纸 -->
        <template v-else-if="active === 'wallpaper'">
          <h1 class="yw-settings-h1">
            壁纸
          </h1>
          <div class="yw-settings-label">
            内置壁纸
          </div>
          <div class="yw-settings-card">
            <div class="yw-settings-wallpapers">
              <button
                v-for="w in wallpapers"
                :key="w.label"
                class="yw-settings-wallpaper"
                :class="{ 'is-active': currentWallpaperSrc === w.src }"
                @click="pickWallpaper(w.src)"
              >
                <span
                  class="yw-settings-wallpaper-thumb"
                  :style="w.thumb.startsWith('linear') ? { background: w.thumb } : { backgroundImage: `url(${w.thumb})` }"
                />
                <span>{{ w.label }}</span>
              </button>
            </div>
          </div>
        </template>

        <!-- Dock -->
        <template v-else-if="active === 'dock'">
          <h1 class="yw-settings-h1">
            桌面与 Dock
          </h1>
          <div class="yw-settings-label">
            固定在 Dock 上的应用
          </div>
          <div class="yw-settings-card">
            <div
              v-for="(app, i) in dockPinned"
              :key="app.id"
              class="yw-settings-row"
              :class="{ 'is-last': i === dockPinned.length - 1 }"
            >
              <span class="yw-settings-row-icon" :style="{ background: app.iconBg ?? 'linear-gradient(135deg, #5FB3F9, #1263E9)' }">
                <i :class="app.icon" />
              </span>
              <span class="yw-settings-row-main">
                <span class="row-label">{{ app.name }}</span>
                <span v-if="runningKeys.has(app.id)" class="row-sub">运行中</span>
              </span>
              <button class="yw-settings-row-btn" @click="unpin(app.id)">
                移除
              </button>
            </div>
            <p v-if="!dockPinned.length" class="yw-settings-empty">
              暂无固定应用
            </p>
          </div>

          <div class="yw-settings-label">
            未固定的应用
          </div>
          <div class="yw-settings-card">
            <div
              v-for="(app, i) in apps.apps.filter(a => !dock.pinned.includes(a.id))"
              :key="app.id"
              class="yw-settings-row"
              :class="{ 'is-last': i === apps.apps.filter(a => !dock.pinned.includes(a.id)).length - 1 }"
            >
              <span class="yw-settings-row-icon" :style="{ background: app.iconBg ?? 'linear-gradient(135deg, #5FB3F9, #1263E9)' }">
                <i :class="app.icon" />
              </span>
              <span class="yw-settings-row-main">
                <span class="row-label">{{ app.name }}</span>
              </span>
              <button class="yw-settings-row-btn" @click="pin(app.id)">
                固定
              </button>
            </div>
          </div>
        </template>

        <!-- 关于 -->
        <template v-else>
          <h1 class="yw-settings-h1">
            关于
          </h1>
          <div class="yw-settings-card">
            <div class="yw-settings-row is-last">
              <span class="yw-settings-row-main">
                <span class="row-label">YudreamWebOS</span>
                <span class="row-sub">核心版本 {{ YW_CORE_VERSION }} · Vue 3 渲染层</span>
              </span>
            </div>
            <div class="yw-settings-row is-last">
              <span class="yw-settings-row-main">
                <span class="row-label">许可</span>
                <span class="row-sub">MIT License · © 2026 YuDream</span>
              </span>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ═══════════ 布局（照抄参考站 SettingsApp.tsx 规格） ═══════════ */
.yw-settings {
  display: flex;
  height: 100%;
  overflow: hidden;
  font-family: var(--yw-font-ui, inherit);
  font-size: 13px;
  color: oklch(var(--yw-foreground));
}

/* 玻璃侧栏：w 216 / margin 10 / radius 12 / blur 30 */
.yw-settings-sidebar {
  display: flex;
  flex-shrink: 0;
  flex-direction: column;
  gap: 10px;
  width: 216px;
  padding: 8px 10px;
  margin: 10px 0 10px 10px;
  overflow: hidden auto;
  user-select: none;
  background: var(--yw-glass-panel-bg, color-mix(in srgb, var(--yw-window-bg) 70%, transparent));
  border-radius: var(--yw-radius-sidebar);
  box-shadow: var(--yw-shadow-menu);
  backdrop-filter: blur(var(--yw-blur-strong)) saturate(170%);
}

/* 搜索框占位（预留接口感） */
.yw-settings-user {
  display: flex;
  gap: 10px;
  align-items: center;
  width: 100%;
  padding: 6px 8px;
  text-align: left;
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 8px;
  transition: background var(--yw-dur-micro);
}

.yw-settings-user:hover,
.yw-settings-user.is-active {
  background: var(--yw-accent-selection);
}

.yw-settings-avatar {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 34px;
  height: 34px;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
  background: linear-gradient(180deg, #5eb3f8, #1463e8);
  border-radius: 50%;
}

.yw-settings-user-meta {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.25;
}

.yw-settings-user-meta b {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 13px;
  font-weight: 600;
  color: oklch(var(--yw-foreground));
  white-space: nowrap;
}

.yw-settings-user-meta small {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 11px;
  color: var(--yw-label-2);
  white-space: nowrap;
}

.yw-settings-nav {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 1px;
  overflow-y: auto;
}

.yw-settings-nav-item {
  display: flex;
  flex-shrink: 0;
  gap: 8px;
  align-items: center;
  height: 28px;
  padding: 0 8px;
  font-size: 13px;
  color: oklch(var(--yw-foreground));
  text-align: left;
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 7px;
  transition: background var(--yw-dur-micro);
}

.yw-settings-nav-item:hover:not(.is-active) {
  background: var(--yw-label-4);
}

.yw-settings-nav-item.is-active {
  color: oklch(var(--yw-foreground));
  background: var(--yw-accent-selection);
}

.yw-settings-nav-icon {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 20px;
  height: 20px;
  border-radius: 5px;
  box-shadow: inset 0 0 0 0.5px rgb(255 255 255 / 25%);
}

.yw-settings-nav-icon i {
  font-size: 12px;
  color: #fff;
}

/* 内容列：640px 居中 */
.yw-settings-body {
  flex: 1;
  overflow-y: auto;
  background: var(--yw-window-bg);
}

.yw-settings-pane {
  max-width: 640px;
  padding: 16px 24px 40px;
  margin: 0 auto;
}

.yw-settings-h1 {
  margin: 0 0 16px;
  font-size: 26px;
  font-weight: 700;
  color: oklch(var(--yw-foreground));
  letter-spacing: -0.01em;
}

.yw-settings-label {
  padding: 0 1px;
  margin: 16px 0 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--yw-label-2);
}

/* 卡片：control-bg + 0.5px 描边（We 组件规格） */
.yw-settings-card {
  overflow: hidden;
  background: var(--yw-control-bg);
  border-radius: 10px;
  box-shadow: 0 0 0 0.5px var(--yw-separator);
}

/* ── 外观模式选择（76x52 艺术图，精确复刻 n7e） ── */
.yw-settings-modes {
  display: flex;
  gap: 16px;
  padding: 14px 16px;
}

.yw-settings-mode {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  padding: 0;
  font-size: 12px;
  color: oklch(var(--yw-foreground));
  cursor: default;
  background: transparent;
  border: none;
}

.yw-settings-mode-art {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 76px;
  height: 52px;
  overflow: hidden;
  background: linear-gradient(180deg, #f2f2f7, #e4e4ea);
  border-radius: 8px;
  box-shadow:
    0 0 0 0.5px var(--yw-separator),
    0 1px 4px rgb(0 0 0 / 12%);
  transition: all 150ms ease;
}

.yw-settings-mode-art.is-active {
  box-shadow:
    0 0 0 2.5px oklch(var(--yw-primary)),
    0 2px 8px rgb(0 0 0 / 20%);
}

.art-chrome {
  position: relative;
  z-index: 1;
  display: flex;
  flex-shrink: 0;
  gap: 3px;
  align-items: center;
  height: 10px;
  padding: 0 5px;
  background: linear-gradient(180deg, #f0f0f4, #e2e2e8);
  border-bottom: 1px solid rgb(0 0 0 / 8%);
}

.art-chrome i {
  width: 4px;
  height: 4px;
  background: #c8c8ce;
  border-radius: 50%;
}

.is-dark .art-chrome,
.is-auto .art-chrome {
  background: linear-gradient(180deg, #3a3a3e, #2c2c30);
  border-bottom-color: rgb(255 255 255 / 10%);
}

.is-dark .art-chrome i,
.is-auto .art-chrome i {
  background: #55555c;
}

.art-pane {
  display: flex;
  flex: 1;
  min-height: 0;
}

.art-pane--light {
  width: 100%;
  background: linear-gradient(180deg, #f2f2f7, #e4e4ea);
}

.yw-settings-mode-art.is-dark .art-pane--light,
.yw-settings-mode-art.is-auto .art-pane--light {
  width: 50%;
}

.art-pane--dark {
  flex: 1;
  background: linear-gradient(180deg, #3a3a3e, #1e1e22);
}

.art-doc {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
  padding: 3px;
  margin: 6px;
  background: #fff;
  border-radius: 3px;
  box-shadow: 0 1px 2px rgb(0 0 0 / 18%);
}

.art-pane--dark .art-doc {
  background: #2c2c2e;
}

.art-line {
  display: block;
  height: 3px;
  background: #d1d1d6;
  border-radius: 999px;
}

.art-pane--dark .art-line {
  background: #636366;
}

.yw-settings-mode-label {
  color: oklch(var(--yw-foreground));
}

/* ── 强调色圆点（22px） ── */
.yw-settings-accents {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 12px 16px;
}

.yw-settings-accent {
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  padding: 0;
  cursor: default;
  border: none;
  border-radius: 50%;
  box-shadow: 0 1px 3px rgb(0 0 0 / 20%);
  transition: transform var(--yw-dur-micro);
}

.yw-settings-accent:active {
  transform: scale(0.9);
}

.yw-settings-accent i {
  font-size: 12px;
  color: #fff;
  opacity: 0;
  stroke-width: 3;
}

.yw-settings-accent.is-active i {
  opacity: 1;
}

/* ── 壁纸网格 ── */
.yw-settings-wallpapers {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 12px;
  padding: 12px;
}

.yw-settings-wallpaper {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  padding: 0;
  font-size: 12px;
  color: oklch(var(--yw-foreground));
  cursor: default;
  background: transparent;
  border: none;
}

.yw-settings-wallpaper-thumb {
  display: block;
  width: 100%;
  height: 68px;
  background: var(--yw-label-4) center / cover no-repeat;
  border-radius: 8px;
  box-shadow:
    0 0 0 0.5px var(--yw-separator),
    0 0 0 0 transparent;
  transition: box-shadow 150ms ease;
}

.yw-settings-wallpaper.is-active .yw-settings-wallpaper-thumb {
  box-shadow:
    0 0 0 2.5px oklch(var(--yw-primary)),
    0 2px 8px rgb(0 0 0 / 20%);
}

/* ── 行（tt 规格：min-h 38 / px-3 / 0.5px 分隔线） ── */
.yw-settings-row {
  display: flex;
  gap: 10px;
  align-items: center;
  min-height: 38px;
  padding: 7px 12px;
  border-bottom: 0.5px solid var(--yw-separator);
}

.yw-settings-row.is-last {
  border-bottom: none;
}

.yw-settings-row-icon {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  box-shadow: inset 0 0 0 0.5px rgb(255 255 255 / 22%);
}

.yw-settings-row-icon i {
  font-size: 13px;
  color: #fff;
}

.yw-settings-row-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

.row-label {
  font-size: 13px;
  line-height: 1.25;
  color: oklch(var(--yw-foreground));
}

.row-sub {
  margin-top: 1px;
  font-size: 11px;
  line-height: 1.25;
  color: var(--yw-label-2);
}

.yw-settings-row-btn {
  flex-shrink: 0;
  padding: 3px 12px;
  font-size: 12px;
  color: oklch(var(--yw-primary));
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 6px;
}

.yw-settings-row-btn:hover {
  background: var(--yw-label-4);
}

.yw-settings-empty {
  padding: 14px;
  font-size: 12px;
  color: var(--yw-label-3);
  text-align: center;
}
</style>
