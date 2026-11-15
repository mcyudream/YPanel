<script setup lang="ts">
const props = defineProps<{
  revision?: number
  identity?: number
}>()
const scroller = ref<HTMLElement>()
const content = ref<HTMLElement>()
const atBottom = ref(true)
let observer: ResizeObserver | undefined
let frame = 0
function scroll(force = false) {
  if (!force && !atBottom.value) {
    return
  }
  cancelAnimationFrame(frame)
  frame = requestAnimationFrame(() => {
    const el = scroller.value
    if (el) {
      el.scrollTop = el.scrollHeight
    }
  })
}
function onScroll() {
  const el = scroller.value
  if (el) {
    atBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 64
  }
}
watch(() => props.revision, () => nextTick(() => scroll()))
watch(() => props.identity, () => {
  atBottom.value = true
  nextTick(() => scroll(true))
})
onMounted(() => {
  observer = new ResizeObserver(() => scroll())
  if (content.value) {
    observer.observe(content.value)
  }
  scroll(true)
})
onBeforeUnmount(() => {
  observer?.disconnect()
  cancelAnimationFrame(frame)
})
async function preserve(action: () => Promise<void>) {
  const el = scroller.value
  if (!el) {
    return
  }
  const height = el.scrollHeight
  const top = el.scrollTop
  atBottom.value = false
  await action()
  await nextTick()
  el.scrollTop = top + el.scrollHeight - height
}
defineExpose({ preserve })
</script>

<template>
  <div class="flex-1 min-h-0 relative">
    <div ref="scroller" class="px-4 py-5 overscroll-contain h-full overflow-y-auto" @scroll.passive="onScroll">
      <div ref="content" class="mx-auto max-w-[920px] w-full space-y-6">
        <slot />
      </div>
    </div>
    <FaButton
      v-if="!atBottom" size="icon-sm" variant="outline" class="rounded-full bg-background shadow-md bottom-4 right-4 absolute" :title="$t('components.ydAiChat.jumpToBottom')" @click="atBottom = true;
                                                                                                                                                                                       scroll(true)"
    >
      <FaIcon name="i-lucide:arrow-down" />
    </FaButton>
  </div>
</template>
