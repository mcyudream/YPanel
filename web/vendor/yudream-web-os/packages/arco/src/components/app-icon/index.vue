<script setup lang="ts">
import { computed } from 'vue'
import YwIconTile from '../icon-tile/index.vue'

const props = withDefaults(defineProps<{
  appKey: string
  icon: string
  title: string
  size?: number
  selected?: boolean
  iconBg?: string
  /** 标签文字颜色（缺省取 --yw-desktop-icon-label，适配壁纸） */
  labelColor?: string
}>(), {
  size: 48,
  selected: false,
})

const emit = defineEmits<{
  click: [ev: MouseEvent]
  dblclick: [ev: MouseEvent]
  contextmenu: [ev: MouseEvent]
}>()

const labelColorStyle = computed(() => props.labelColor ?? undefined)
</script>

<template>
  <div
    class="yw-app-icon"
    :class="{ 'is-selected': selected }"
    @click="emit('click', $event)"
    @dblclick="emit('dblclick', $event)"
    @contextmenu="emit('contextmenu', $event)"
  >
    <YwIconTile :app-key="appKey" :icon="icon" :size="size" :icon-bg="iconBg" />
    <span class="yw-app-icon-label" :style="labelColorStyle">{{ title }}</span>
  </div>
</template>

<style scoped>
.yw-app-icon {
  display: flex;
  flex-direction: column;
  gap: 7px;
  align-items: center;
  width: var(--yw-deskicon-cell-w, 84px);
  padding: 10px 4px;
  cursor: default;
  user-select: none;
  border-radius: var(--yw-radius-menu, 8px);
  transition: background-color var(--yw-dur-fast) ease;
}

.yw-app-icon:hover .yw-app-icon-label {
  text-shadow:
    0 0 8px rgb(255 255 255 / 35%),
    0 1px 3px rgb(0 0 0 / 65%);
}

.yw-app-icon.is-selected :deep(.yw-icon-tile) {
  filter: brightness(0.82);
}

.yw-app-icon-label {
  max-width: 92px;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.35;
  color: oklch(var(--yw-desktop-icon-label));
  white-space: nowrap;
  text-shadow: 0 1px 3px rgb(0 0 0 / 65%), 0 0 6px rgb(0 0 0 / 25%);
}
</style>
