<script setup lang="ts">
import { computed } from 'vue'

/**
 * YwCard — WebOS 卡片：标题/操作/内容/页脚四槽位 + widget 模式。
 * 桌面 widget、应用内信息卡、磁贴内容容器共用。
 */
const props = withDefaults(defineProps<{
  title?: string
  /** 无边框透明模式（widget 场景） */
  glass?: boolean
  /** 内边距紧凑模式 */
  compact?: boolean
  /** 磁贴内容（裁剪溢出） */
  tile?: boolean
}>(), {
  title: '',
  glass: false,
  compact: false,
  tile: false,
})

const cls = computed(() => [
  'yw-card',
  props.glass && 'is-glass',
  props.compact && 'is-compact',
  props.tile && 'is-tile',
])
</script>

<template>
  <section :class="cls">
    <header v-if="title || $slots.actions" class="yw-card-header">
      <h3 class="yw-card-title">
        {{ title }}
      </h3>
      <div class="yw-card-actions">
        <slot name="actions" />
      </div>
    </header>

    <div class="yw-card-body">
      <slot />
    </div>

    <footer v-if="$slots.footer" class="yw-card-footer">
      <slot name="footer" />
    </footer>
  </section>
</template>

<style scoped>
.yw-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  color: oklch(var(--yw-card-foreground));
  background: oklch(var(--yw-card));
  border: 1px solid oklch(var(--yw-border));
  border-radius: var(--yw-radius-lg);
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 12%),
    0 0 0 0.5px rgb(255 255 255 / 6%),
    0 6px 24px -8px rgb(0 0 0 / 22%);
}

.yw-card.is-glass {
  background: oklch(var(--yw-glass) / 58%);
  border-color: oklch(var(--yw-border) / 50%);
  backdrop-filter: blur(20px) saturate(1.4);
}

.yw-card.is-compact .yw-card-body {
  padding: 10px 14px;
}

.yw-card.is-tile {
  border-radius: var(--yw-radius-md);
}

.yw-card.is-tile .yw-card-body {
  padding: 12px;
  overflow: hidden;
}

.yw-card-header {
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 14px 16px 0;
}

.yw-card-title {
  flex: 1;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
}

.yw-card-actions {
  display: flex;
  gap: 6px;
}

.yw-card-body {
  flex: 1;
  padding: 12px 16px;
  overflow: auto;
}

.yw-card-footer {
  padding: 10px 16px;
  font-size: 12px;
  color: oklch(var(--yw-muted-foreground));
  border-top: 1px solid oklch(var(--yw-border) / 70%);
}
</style>
