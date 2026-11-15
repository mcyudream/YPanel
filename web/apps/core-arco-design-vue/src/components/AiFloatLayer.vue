<script setup lang="ts">
import ChatWindow from '@/components/YdAiChat/ChatWindow.vue'
import Panel from '@/components/YdAiChat/Panel.vue'

const route = useRoute()
const router = useRouter()
const open = ref(false)
const panel = ref<InstanceType<typeof Panel>>()
const floatConversation = ref<number>()
function close() {
  floatConversation.value = panel.value?.conversationId
  open.value = false
}
function expand() {
  const id = panel.value?.conversationId
  close()
  void router.push({ path: '/ai/chat', query: id ? { conversation: id } : {} })
}
</script>

<template>
  <FaButton v-if="!open && !route.path.startsWith('/ai/chat')" size="icon" class="rounded-full size-12 shadow-lg bottom-6 right-6 fixed z-1600" :title="$t('components.ydAiChat.assistantTitle')" @click="open = true">
    <FaIcon name="i-ri:sparkling-2-line" />
  </FaButton>
  <ChatWindow v-if="open" :title="$t('components.ydAiChat.assistantTitle')" :width="560" :height="760" :expand-only="true" @close="close" @expand="expand">
    <Panel ref="panel" compact :initial-id="floatConversation" :scene="route.path" />
  </ChatWindow>
</template>
