<script setup lang="ts">
import type { IconNode } from 'morphicons'
import { MorphIcon } from 'morphicons/vue'
import { computed, ref, watch } from 'vue'

defineOptions({
  name: 'YdMorphIcon',
})

// YPanel 图标主通道：本地打包的 lucide 24×24 描边图标（morphicons 引擎渲染）。
// 兜底链：本地数据未命中 → fallback prop（任一 FaIcon 兼容的旧图标名）→ 占位图形。
// 数据经 scripts/gen-icons.mjs 生成并随构建本地打包，无任何 CDN 外链。

const props = defineProps<{
  /** 图标名（lucide 命名，如 server / folder-open） */
  name: string
  size?: number | string
  strokeWidth?: number | string
  color?: string
  /** 可访问性标签；缺省视为装饰性图标 */
  label?: string
  /** 兜底图标名（FaIcon 兼容格式，如 i-ep:setting） */
  fallback?: string
}>()

// 懒加载图标数据（独立 chunk，首次使用时拉取一次）
const registry = ref<Record<string, { nodes: IconNode }>>({})
let loadStarted = false

async function loadRegistry() {
  if (loadStarted) {
    return
  }
  loadStarted = true
  try {
    const mod = await import('@/ui/icons/data.json')
    registry.value = (mod.default as any).icons ?? {}
  }
  catch (e) {
    console.warn('[YdMorphIcon] 图标数据加载失败', e)
  }
}
loadRegistry()

const iconNode = computed<IconNode | undefined>(() => registry.value[props.name]?.nodes)

const placeholderVisible = computed(() => registry.value && Object.keys(registry.value).length > 0 && !iconNode.value)

watch(() => props.name, () => loadRegistry())
</script>

<template>
  <MorphIcon
    v-if="iconNode"
    :icon="iconNode"
    :size="size ?? 18"
    :stroke-width="strokeWidth ?? 2"
    :color="color"
    :label="label"
    class="yd-morph-icon shrink-0"
  />
  <FaIcon
    v-else-if="fallback"
    :name="fallback"
    :class="$attrs.class"
    :style="typeof size !== 'undefined' ? { fontSize: typeof size === 'number' ? `${size}px` : size } : undefined"
  />
  <span
    v-else-if="placeholderVisible"
    class="inline-block shrink-0 rounded-full border border-current opacity-40"
    :style="{ width: `${size ?? 18}px`, height: `${size ?? 18}px`, borderWidth: `${strokeWidth ?? 2}px` }"
    :title="`未知图标: ${name}`"
  />
  <span v-else class="inline-block shrink-0" :style="{ width: `${size ?? 18}px`, height: `${size ?? 18}px` }" />
</template>

<style scoped>
.yd-morph-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
</style>
