<script setup lang="ts">
// 桌面文本文件承载：便签级编辑器。内容存桌面项 content 字段——随 desktop.layout 持久化，
// 布局导出/导入（M44）与跨浏览器恢复（M45）自动带走。入口：桌面文件图标双击。
import type { WindowInstance } from '@yudream/yudream-webos-core'
import { useWebOS } from '@yudream/yudream-webos-vue'
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  win?: WindowInstance & { launchOptions?: Record<string, unknown> }
}>()

const os = useWebOS()
const desktopId = computed(() => String(props.win?.launchOptions?.desktopId ?? ''))
const item = computed(() => (desktopId.value ? os.desktop.get(desktopId.value) : undefined))
const text = ref('')

watch(item, (it) => {
  text.value = it?.content ?? ''
}, { immediate: true })

const dirty = computed(() => text.value !== (item.value?.content ?? ''))

function save() {
  if (!desktopId.value) {
    return
  }
  os.desktop.writeFile(desktopId.value, text.value)
  os.ui.message('success', '已保存')
}
</script>

<template>
  <div class="yp-text-editor">
    <div class="yp-text-editor-bar">
      <i class="i-lucide-file-text" />
      <span class="yp-text-editor-name">{{ item?.name ?? win?.launchOptions?.name ?? '文本文件' }}</span>
      <span v-if="dirty" class="yp-text-editor-dirty">未保存</span>
      <span class="flex-1" />
      <span class="yp-text-editor-count">{{ text.length }} 字符</span>
      <button class="yp-text-editor-save" :disabled="!dirty" @click="save">
        保存
      </button>
    </div>
    <textarea
      v-model="text"
      class="yp-text-editor-area"
      spellcheck="false"
      placeholder="输入文本…"
    />
  </div>
</template>

<style scoped>
.yp-text-editor {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background: oklch(var(--yw-background, 100% 0 0));
}

.yp-text-editor-bar {
  display: flex;
  flex-shrink: 0;
  gap: 8px;
  align-items: center;
  padding: 8px 12px;
  font-size: 12px;
  color: oklch(var(--yw-foreground, 0% 0 0));
  border-bottom: 1px solid rgb(120 120 128 / 16%);
}

.yp-text-editor-name {
  max-width: 50%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.yp-text-editor-dirty {
  padding: 1px 8px;
  font-size: 10px;
  color: #fff;
  background: oklch(var(--yw-primary, 60% 0.15 250));
  border-radius: 99px;
}

.yp-text-editor-count {
  font-size: 11px;
  color: oklch(var(--yw-muted-foreground, 50% 0 0));
}

.yp-text-editor-save {
  padding: 4px 14px;
  font-size: 12px;
  color: #fff;
  cursor: default;
  background: oklch(var(--yw-primary, 60% 0.15 250));
  border: none;
  border-radius: 7px;
}

.yp-text-editor-save:disabled {
  opacity: 0.45;
}

.yp-text-editor-area {
  flex: 1;
  width: 100%;
  padding: 12px 14px;
  font-size: 13px;
  line-height: 1.6;
  color: oklch(var(--yw-foreground, 0% 0 0));
  resize: none;
  outline: none;
  background: transparent;
  border: none;
}
</style>
