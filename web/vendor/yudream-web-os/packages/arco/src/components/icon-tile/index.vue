<script setup lang="ts">
import { computed } from 'vue'
import { tileBackground } from './colors'

/**
 * YwIconTile — macOS 应用图标风格的底座容器。
 * 拟真质感三层：彩色渐变底 + 顶部光泽 overlay（::before）+ hairline 内描边；
 * 线条图标加粗描边（2.5）+ 落影，接近原生应用图标的分量感。
 */
const props = withDefaults(defineProps<{
  appKey: string
  icon: string
  /** 底座边长 px */
  size?: number
  /** manifest 显式底色（CSS background 值） */
  iconBg?: string
}>(), {
  size: 48,
})

const isImage = computed(() => /^(?:https?:|data:|\/|\.)/.test(props.icon))

const bg = computed(() => tileBackground(props.appKey, props.iconBg))

const iconFontSize = computed(() => `${Math.round(props.size * 0.5)}px`)

const style = computed(() => ({
  width: `${props.size}px`,
  height: `${props.size}px`,
  background: bg.value,
  borderRadius: '22.37%',
  boxShadow: `inset 0 1px 0 rgba(255, 255, 255, 0.4),
    inset 0 0 0 0.5px rgba(255, 255, 255, 0.25),
    inset 0 -6px 12px rgba(0, 0, 0, 0.18),
    0 3px 8px rgba(0, 0, 0, 0.3),
    0 1px 3px rgba(0, 0, 0, 0.2)`,
}))
</script>

<template>
  <div class="yw-icon-tile" :style="style">
    <img v-if="isImage" :src="icon" alt="" draggable="false">
    <template v-else>
      <span class="yw-icon-tile-gloss" />
      <i :class="icon" />
    </template>
  </div>
</template>

<style scoped>
.yw-icon-tile {
  position: relative;
  display: grid;
  flex-shrink: 0;
  place-items: center;
  overflow: hidden;
  user-select: none;
}

/* 顶部光泽：macOS 图标的标志性高光弧面 */
.yw-icon-tile-gloss {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(180deg, rgb(255 255 255 / 30%) 0%, rgb(255 255 255 / 10%) 42%, rgb(255 255 255 / 0%) 46%, rgb(0 0 0 / 6%) 100%);
}

.yw-icon-tile img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.yw-icon-tile i {
  position: relative;
  font-size: v-bind(iconFontSize);
  color: #fff;
  filter: drop-shadow(0 1.5px 2px rgb(0 0 0 / 35%));
}

.yw-icon-tile i svg {
  stroke-width: 2.5;
}
</style>
