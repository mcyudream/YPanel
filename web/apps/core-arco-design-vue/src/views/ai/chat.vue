<script setup lang="ts">
import Panel from '@/components/YdAiChat/Panel.vue'
import { useYwEmbed } from '@/views/desktop/embed'
import { focusHintOf, lastFocusedWin, scenePathOfApp } from '@/views/desktop/focus-scene'

defineOptions({ name: 'AiChat' })
const embed = useYwEmbed()
const route = useRoute()
</script>

<template>
  <FaPageMain :main-class="embed ? 'min-h-0 flex-1 p-0' : 'p-0'">
    <div class="ai-page" :class="{ 'ai-page--embed': embed }">
      <Panel :initial-id="Number(route.query.conversation) || undefined" :scene="embed ? scenePathOfApp(lastFocusedWin?.appId || '') : '/'" :focus="embed && lastFocusedWin ? focusHintOf(lastFocusedWin) : ''" />
    </div>
  </FaPageMain>
</template>

<style scoped>
.ai-page {
  height: calc(100dvh - var(--g-slots-layout-top-height, 0px) - var(--g-header-actual-height, 60px) - var(--g-topbar-actual-height, 0px) - var(--g-main-container-padding-top, 0px) - var(--g-slots-layout-bottom-height, 0px) - var(--g-main-container-padding-bottom, 0px) - 20px);
  min-height: 0;
  overflow: hidden;
}

.ai-page--embed {
  height: 100%;
}
</style>
