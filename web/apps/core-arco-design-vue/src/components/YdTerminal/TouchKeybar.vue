<script setup lang="ts">
import { useLocalStorage } from '@vueuse/core'

// 终端触屏按键条：mobile 模式显示，补物理键盘缺失的 Ctrl/Esc/Tab/方向键。
// 按键在 pointerdown 发送并 preventDefault——不夺走 xterm 隐藏 textarea 的焦点，软键盘不收起。
defineOptions({
  name: 'YdTerminalTouchKeybar',
})

const props = defineProps<{
  /** 发送原始字节（由宿主做协议封装：host JSON 帧 / exec 裸帧） */
  send: (data: string) => void
  /** 发送后兜底拉回终端焦点（软键盘被收起时补救） */
  refocus?: () => void
}>()

const appSettings = useAppSettingsStore()
const visible = computed(() => appSettings.mode === 'mobile')

// 收起状态全局持久化（桌面窄窗模式下同样不想看到就都别看）
const collapsed = useLocalStorage('ypanel.terminal.keybar.hidden', false)

const KEYS: Array<{ label: string, data: string, title?: string }> = [
  { label: 'Esc', data: '\x1b' },
  { label: 'Tab', data: '\t' },
  { label: '↑', data: '\x1b[A', title: 'Up' },
  { label: '↓', data: '\x1b[B', title: 'Down' },
  { label: '^C', data: '\x03', title: 'Ctrl+C' },
  { label: '^D', data: '\x04', title: 'Ctrl+D (EOF)' },
  { label: '^L', data: '\x0c', title: 'Ctrl+L (clear)' },
  { label: '|', data: '|' },
  { label: '~', data: '~' },
  { label: '/', data: '/' },
]

function press(k: { data: string }) {
  props.send(k.data)
  props.refocus?.()
}
</script>

<template>
  <div v-if="visible && !collapsed" class="flex h-8 shrink-0 items-stretch gap-px border-t bg-muted/40">
    <button
      v-for="k in KEYS"
      :key="k.label"
      type="button"
      class="min-w-0 flex-1 cursor-pointer select-none text-[11px] text-muted-foreground transition-colors active:bg-primary/15 active:text-foreground"
      :title="k.title"
      @pointerdown.prevent="press(k)"
    >
      {{ k.label }}
    </button>
    <button
      type="button"
      class="w-7 shrink-0 cursor-pointer text-muted-foreground transition-colors active:text-foreground"
      :title="$t('components.ydTerminal.keybarHide')"
      @pointerdown.prevent="collapsed = true"
    >
      <FaIcon name="i-lucide:chevron-down" class="text-xs" />
    </button>
  </div>
  <button
    v-else-if="visible && collapsed"
    type="button"
    class="flex h-4.5 shrink-0 cursor-pointer items-center justify-center border-t bg-muted/40 text-muted-foreground transition-colors active:text-foreground"
    :title="$t('components.ydTerminal.keybarShow')"
    @pointerdown.prevent="collapsed = false"
  >
    <FaIcon name="i-lucide:chevron-up" class="text-[10px]" />
  </button>
</template>
