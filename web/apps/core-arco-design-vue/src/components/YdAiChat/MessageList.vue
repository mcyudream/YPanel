<script setup lang="ts">
// YdAiAiMessageList：消息列表容器（ZCode 式自动滚动：新内容贴底跟随；用户上翻时暂停并显示「回到底部」）。
import type { AiChatMessage } from '@/composables/useAiChat'
import { nextTick, onUpdated, ref, watch } from 'vue'

const props = defineProps<{
  messages: AiChatMessage[]
}>()

const listRef = ref<HTMLElement>()
// 距底 < 48px 视为贴底（自动滚动跟随）；用户上翻则暂停，滚回底部自动恢复
const stickBottom = ref(true)
const showJump = ref(false)

function onScroll() {
  const el = listRef.value
  if (!el) {
    return
  }
  const gap = el.scrollHeight - el.scrollTop - el.clientHeight
  stickBottom.value = gap < 48
  showJump.value = !stickBottom.value
}

async function scrollToBottom(force = false) {
  await nextTick()
  const el = listRef.value
  if (!el || (!stickBottom.value && !force)) {
    return
  }
  el.scrollTop = el.scrollHeight
  stickBottom.value = true
  showJump.value = false
}

// 流式内容高频更新：deep watch 覆盖 content/segments/步骤等全部嵌套变化
watch(() => props.messages, () => {
  void scrollToBottom()
}, { deep: true, immediate: true })

// 嵌套组件（工具块展开等）引起的 DOM 高度变化兜底
onUpdated(() => {
  if (stickBottom.value) {
    const el = listRef.value
    if (el) {
      el.scrollTop = el.scrollHeight
    }
  }
})

function jump() {
  void scrollToBottom(true)
}
</script>

<template>
  <div class="relative min-h-0 flex-1">
    <div ref="listRef" class="h-full overflow-y-auto px-4 py-4" @scroll="onScroll">
      <!-- 内容宽度约束 920px 居中（参考 YuDream-Admin） -->
      <div class="mx-auto w-full max-w-[920px] space-y-5">
        <slot />
      </div>
    </div>
    <!-- 回到底部：右下角圆形悬浮钮 -->
    <button
      v-if="showJump"
      type="button"
      class="absolute bottom-4 right-4 flex size-9 cursor-pointer items-center justify-center rounded-full border bg-background text-muted-foreground shadow-md transition-all hover:text-foreground hover:shadow-lg"
      :title="$t('components.ydAiChat.jumpToBottom')"
      @click="jump"
    >
      <FaIcon name="i-lucide:arrow-down-to-line" class="text-sm" />
    </button>
  </div>
</template>
