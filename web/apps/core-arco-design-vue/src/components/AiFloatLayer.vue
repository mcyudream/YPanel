<script setup lang="ts">
// AiFloatLayer：全局 AI 浮层（B18）——任何页面右下角悬浮球，点开对话窗口；
// 场景感知：自动携带当前页面路径，后端注入该页实时数据摘要。
import { useRoute } from 'vue-router'
import ChatWindow from '@/components/YdAiChat/ChatWindow.vue'
import MessageList from '@/components/YdAiChat/MessageList.vue'
import Sender from '@/components/YdAiChat/Sender.vue'
import YdAiBubble from '@/components/YdAiBubble/index.vue'
import { useAiChat } from '@/composables/useAiChat'

defineOptions({ name: 'AiFloatLayer' })

const route = useRoute()
const open = ref(false)

const scenePath = ref(route.fullPath)
watch(() => route.fullPath, (v) => { scenePath.value = v })
const { messages, streaming, send, stop, clear } = useAiChat({ scenePath: () => scenePath.value })

const suggestions = computed(() => {
  switch (true) {
    case route.path.startsWith('/container'):
      return ['容器占用过高怎么排查？', '帮我重启异常的容器']
    case route.path.startsWith('/sites'):
      return ['给站点配一个泛域名证书', '检查站点配置是否有问题']
    case route.path.startsWith('/database'):
      return ['数据库备份策略怎么定？', '远程访问有安全风险吗']
    default:
      return ['当前服务器状态如何？', '磁盘占用偏高怎么处理？', '写一个清理 Docker 悬空镜像的脚本']
  }
})

</script>

<template>
  <div>
    <!-- 悬浮球 -->
    <button
      v-if="!open"
      type="button"
      class="fixed bottom-6 right-6 z-1600 flex size-12 cursor-pointer items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-transform hover:scale-105"
      title="AI 助手"
      @click="open = true"
    >
      <FaIcon name="i-ri:sparkling-2-line" class="text-lg" />
    </button>

    <!-- 对话窗口 -->
    <ChatWindow
      v-if="open"
      title="AI 助手"
      :width="520"
      :height="720"
      :expand-only="false"
      @close="open = false"
      @expand="() => {}"
    >
      <MessageList :messages="messages">
        <!-- 欢迎页 -->
        <div v-if="!messages.length" class="space-y-3 py-8 text-center">
          <div class="text-4xl">
            ✨
          </div>
          <div class="text-sm font-medium">
            YPanel AI 助手
          </div>
          <div class="text-xs text-muted-foreground">
            已感知当前页面：{{ route.meta.title || route.path }}
          </div>
          <div class="flex flex-wrap justify-center gap-1.5 px-6 pt-2">
            <button
              v-for="s in suggestions"
              :key="s"
              type="button"
              class="cursor-pointer rounded-full border px-3 py-1 text-xs text-muted-foreground hover:bg-accent/50"
              @click="send(s)"
            >
              {{ s }}
            </button>
          </div>
        </div>

        <!-- 消息气泡（YDBubble 风格） -->
        <template v-else>
          <div v-for="m in messages" :key="m.id" class="space-y-1">
            <YdAiBubble
              :role="m.role"
              :content="m.content"
              :pending="m.pending"
              :reasoning="m.reasoning || ''"
              :steps="m.steps || []"
              show-actions
              @regenerate="() => {}"
            />
          </div>
        </template>
      </MessageList>

      <Sender
        :loading="streaming"
        :suggestions="!messages.length ? suggestions : []"
        :placeholder="`已感知当前页面「${route.meta.title || route.path}」，尽管问…`"
        @send="send"
        @stop="stop"
        @suggestion-click="send"
      >
        <template #actions>
          <button
            type="button"
            class="text-xs text-muted-foreground hover:text-red-500"
            @click="clear()"
          >
            清空对话
          </button>
        </template>
      </Sender>
    </ChatWindow>
  </div>
</template>
