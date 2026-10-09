<script setup lang="ts">
import { useSystemSettings } from '@yudream/yudream-webos-vue'
import { computed } from 'vue'

/**
 * YwControlCenter — 控制中心（菜单栏右侧弹出）。
 * 只保留对 Web 管理面板真实生效的项：深色模式 / 强调色 / 壁纸（随系统设置持久化，scope: system）。
 * Wi-Fi / 蓝牙 / 亮度 / 音量等「伪 OS」演示件已移除——面板不是桌面操作系统。
 * 壁纸：宿主传入 wallpapers 清单后渲染缩略图行，点选即换并持久化。
 */
const props = defineProps<{
  /** 可选：宿主提供的壁纸清单（src 为图片 URL 或渐变串）；传入后展示壁纸选择区 */
  wallpapers?: Array<{ id: string, name?: string, src: string }>
  /** 可选：宿主当前壁纸 src（高亮用；缺省读系统设置） */
  currentWallpaper?: string | null
}>()

const emit = defineEmits<{
  close: []
}>()

const { settings, setMode, setAccent, setWallpaper } = useSystemSettings()

/** 强调色候选（写入 settings.accent，全主题 --yw-primary/--yw-ring 跟随并持久化） */
const ACCENTS = [
  { id: 'blue', value: '#0a84ff' },
  { id: 'purple', value: '#bf5af2' },
  { id: 'pink', value: '#ff375f' },
  { id: 'orange', value: '#ff9f0a' },
  { id: 'green', value: '#32d74b' },
  { id: 'graphite', value: '#8e8e93' },
]

/** 深色生效态（system 模式下按系统暗色折算） */
const isDarkEff = computed(() =>
  settings.mode === 'system' ? settings.systemDark : settings.mode === 'dark',
)

const currentAccent = computed(() => settings.accent ?? null)

function toggleDark() {
  setMode(settings.mode === 'dark' ? 'light' : 'dark')
}

function isCurrentWallpaper(src: string): boolean {
  const cur = props.currentWallpaper ?? settings.wallpaper?.src ?? null
  return cur === src
}

function pickWallpaper(src: string | null) {
  setWallpaper(src === null ? null : { src })
}
</script>

<template>
  <div class="yw-cc-overlay" @pointerdown.self="emit('close')">
    <div class="yw-cc" role="dialog" aria-label="控制中心">
      <!-- 深色模式：行式滑动开关 -->
      <div class="yw-cc-row">
        <span class="yw-cc-row-icon" :class="{ 'is-on': isDarkEff }">
          <i class="i-lucide-moon" />
        </span>
        <div class="yw-cc-row-text">
          <b>深色模式</b>
          <small>{{ isDarkEff ? '深色界面' : '浅色界面' }}</small>
        </div>
        <button
          class="yw-cc-switch"
          :class="{ 'is-on': isDarkEff }"
          role="switch"
          :aria-checked="isDarkEff"
          aria-label="深色模式"
          @click="toggleDark"
        >
          <span class="yw-cc-switch-knob" />
        </button>
      </div>

      <!-- 强调色（真实生效：全主题主色跟随并持久化） -->
      <div class="yw-cc-section">
        <div class="yw-cc-section-head">
          <i class="i-lucide-palette" />
          <span>强调色</span>
        </div>
        <div class="yw-cc-swatch-row">
          <button
            class="yw-cc-swatch"
            :class="{ 'is-active': currentAccent === null }"
            title="默认"
            @click="setAccent(null)"
          >
            <i class="i-lucide-rotate-ccw" />
          </button>
          <button
            v-for="a in ACCENTS"
            :key="a.id"
            class="yw-cc-swatch"
            :class="{ 'is-active': currentAccent?.toLowerCase() === a.value.toLowerCase() }"
            :title="a.id"
            :style="{ background: a.value }"
            @click="setAccent(a.value)"
          />
        </div>
      </div>

      <!-- 壁纸（宿主传入清单才显示）：缩略图点选即换，随系统设置持久化 -->
      <div v-if="wallpapers?.length" class="yw-cc-section">
        <div class="yw-cc-section-head">
          <i class="i-lucide-image" />
          <span>壁纸</span>
        </div>
        <div class="yw-cc-wallpaper-row">
          <button
            class="yw-cc-wallpaper-thumb is-default"
            :class="{ 'is-active': (currentWallpaper ?? settings.wallpaper?.src ?? null) === null }"
            title="默认渐变"
            @click="pickWallpaper(null)"
          />
          <button
            v-for="w in wallpapers"
            :key="w.id"
            class="yw-cc-wallpaper-thumb"
            :class="{ 'is-active': isCurrentWallpaper(w.src) }"
            :title="w.name ?? w.id"
            :style="{ backgroundImage: `url(${w.src})` }"
            @click="pickWallpaper(w.src)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yw-cc-overlay {
  position: fixed;
  inset: 0;
  z-index: 9350;
}

.yw-cc {
  position: absolute;
  top: calc(var(--yw-menubar-h) + 6px);
  right: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 320px;
  padding: 12px;
  background: var(--yw-glass-panel-bg);
  border-radius: var(--yw-radius-cc-panel);
  box-shadow: var(--yw-shadow-menu);
  backdrop-filter: blur(var(--yw-blur-strong)) saturate(180%);
  animation: yw-cc-in 0.25s var(--yw-ease-spring);
}

@keyframes yw-cc-in {
  from {
    opacity: 0;
    transform: translateY(-8px) scale(0.97);
  }

  to {
    opacity: 1;
    transform: none;
  }
}

.yw-cc-row {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 10px 12px;
  background: var(--yw-label-4);
  border-radius: var(--yw-radius-cc-tile);
}

.yw-cc-row-icon {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  font-size: 15px;
  color: var(--yw-label-secondary);
  background: var(--yw-label-4);
  border-radius: 50%;
  transition:
    background var(--yw-dur-ui) var(--yw-ease-out),
    color var(--yw-dur-ui) var(--yw-ease-out);
}

.yw-cc-row-icon.is-on {
  color: #fff;
  background: oklch(var(--yw-primary));
}

.yw-cc-row-text {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 1px;
}

.yw-cc-row-text b {
  font-size: 13px;
  color: oklch(var(--yw-foreground));
}

.yw-cc-row-text small {
  font-size: 11px;
  color: var(--yw-label-secondary);
}

.yw-cc-switch {
  position: relative;
  flex-shrink: 0;
  width: 40px;
  height: 22px;
  padding: 0;
  cursor: pointer;
  background: var(--yw-label-4);
  border: none;
  border-radius: 11px;
  transition: background var(--yw-dur-ui) var(--yw-ease-out);
}

.yw-cc-switch-knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  background: #fff;
  border-radius: 50%;
  box-shadow: 0 1px 3px rgb(0 0 0 / 30%);
  transition: transform var(--yw-dur-ui) var(--yw-ease-spring);
}

.yw-cc-switch.is-on {
  background: oklch(var(--yw-primary));
}

.yw-cc-switch.is-on .yw-cc-switch-knob {
  transform: translateX(18px);
}

.yw-cc-tile {
  display: grid;
  flex: 1;
  gap: 8px;
  padding: 12px;
  color: oklch(var(--yw-foreground));
  text-align: left;
  cursor: default;
  background: var(--yw-label-4);
  border: none;
  border-radius: var(--yw-radius-cc-tile);
}

.yw-cc-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.yw-cc-section-head {
  display: flex;
  gap: 6px;
  align-items: center;
  font-size: 12px;
  font-weight: 600;
  color: oklch(var(--yw-foreground));
}

.yw-cc-section-head i {
  font-size: 13px;
  color: oklch(var(--yw-primary));
}

.yw-cc-swatch-row {
  display: flex;
  gap: 8px;
}

.yw-cc-swatch {
  width: 32px;
  height: 32px;
  cursor: pointer;
  background: var(--yw-label-4);
  border: 2px solid transparent;
  border-radius: 50%;
  transition: border-color var(--yw-dur-ui) var(--yw-ease-out);
}

.yw-cc-swatch i {
  font-size: 13px;
  color: oklch(var(--yw-foreground) / 60%);
}

.yw-cc-swatch.is-active {
  border-color: oklch(var(--yw-foreground));
  box-shadow: 0 0 0 2px oklch(var(--yw-foreground) / 30%);
}

.yw-cc-wallpaper-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.yw-cc-wallpaper-thumb {
  height: 44px;
  cursor: pointer;
  background-color: var(--yw-label-4);
  background-position: center;
  background-size: cover;
  border: 2px solid transparent;
  border-radius: 8px;
  transition: border-color var(--yw-dur-ui) var(--yw-ease-out);
}

.yw-cc-wallpaper-thumb.is-default {
  background: linear-gradient(160deg, #0f172a 0%, #1e293b 45%, #334155 100%);
}

.yw-cc-wallpaper-thumb.is-active {
  border-color: oklch(var(--yw-primary));
  box-shadow: 0 0 0 2px oklch(var(--yw-primary) / 35%);
}
</style>
