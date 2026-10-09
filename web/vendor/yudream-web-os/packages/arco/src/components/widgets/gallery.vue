<script setup lang="ts">
import { useWidgets } from '@yudream/yudream-webos-vue'
import { computed } from 'vue'

/**
 * YwWidgetGallery — 小组件画廊（编辑模式下选择可添加的小组件与尺寸）。
 */
const emit = defineEmits<{
  close: []
}>()

const { widgets, add, setEditing } = useWidgets()

const definitions = computed(() => widgets.listDefinitions())

function addWidget(widgetId: string, size: 'small' | 'medium' | 'large') {
  add(widgetId, size, { col: 0, row: 0 })
  setEditing(true)
  emit('close')
}
</script>

<template>
  <div class="yw-widget-gallery-overlay" @pointerdown.self="emit('close')">
    <div class="yw-widget-gallery" role="dialog" aria-label="小组件画廊">
      <h3>小组件</h3>
      <div class="yw-widget-gallery-list">
        <div v-for="def in definitions" :key="def.id" class="yw-widget-gallery-item">
          <span class="yw-widget-gallery-name">{{ def.name }}</span>
          <span class="yw-widget-gallery-sizes">
            <button
              v-for="size in def.sizes"
              :key="size"
              class="yw-widget-size-btn"
              @click="addWidget(def.id, size)"
            >
              {{ size === 'small' ? '小' : size === 'medium' ? '中' : '大' }}
            </button>
          </span>
        </div>
        <p v-if="!definitions.length" class="yw-widget-gallery-empty">
          尚无可添加的小组件
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yw-widget-gallery-overlay {
  position: fixed;
  inset: 0;
  z-index: 9450;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 14vh;
  background: rgb(0 0 0 / 30%);
  backdrop-filter: blur(6px);
}

.yw-widget-gallery {
  width: min(460px, 92vw);
  padding: 20px;
  background: oklch(var(--yw-popover) / 92%);
  border: 1px solid oklch(var(--yw-border));
  border-radius: var(--yw-radius-spotlight);
  box-shadow: var(--yw-shadow-window);
}

.yw-widget-gallery h3 {
  margin: 0 0 14px;
  font-size: 16px;
  color: oklch(var(--yw-popover-foreground));
}

.yw-widget-gallery-item {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  padding: 10px 4px;
  border-bottom: 0.5px solid oklch(var(--yw-border) / 60%);
}

.yw-widget-gallery-name {
  font-size: 13px;
  color: oklch(var(--yw-popover-foreground));
}

.yw-widget-gallery-sizes {
  display: flex;
  gap: 6px;
}

.yw-widget-size-btn {
  padding: 4px 14px;
  font-size: 12px;
  color: oklch(var(--yw-primary));
  cursor: default;
  background: transparent;
  border: 1px solid oklch(var(--yw-primary) / 45%);
  border-radius: var(--yw-radius-capsule);
}

.yw-widget-size-btn:hover {
  color: #fff;
  background: oklch(var(--yw-primary));
}

.yw-widget-gallery-empty {
  padding: 24px;
  font-size: 13px;
  color: oklch(var(--yw-muted-foreground));
  text-align: center;
}
</style>
