<script setup lang="ts">
import { useWebOS, useWebOSEvents } from '@yudream/yudream-webos-vue'
import { onMounted, onUnmounted, ref } from 'vue'

/**
 * YwNotificationCenter — 通知中心（菜单栏时间点击弹出）。
 * 数据源：webos:* 事件聚合（window/app/vfs 事件转通知演示）。
 */
const emit = defineEmits<{
  close: []
}>()

const { ui } = useWebOS()

interface Notice {
  id: number
  title: string
  body: string
  time: string
}

const notices = ref<Notice[]>([])
let seq = 1

function push(title: string, body: string) {
  notices.value.unshift({
    id: seq++,
    title,
    body,
    time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
  })
  if (notices.value.length > 20) {
    notices.value.pop()
  }
}

const { on } = useWebOSEvents()
let offs: Array<() => void> = []

onMounted(() => {
  offs = [
    on('webos:window:open', ({ appId }) => push('应用启动', `${appId} 已打开`)),
    on('webos:app:relaunch', ({ appId }) => push('应用聚焦', `${appId} 已在运行`)),
    on('webos:vfs:change', ({ path, type }) => push('文件系统', `${type}: ${path}`)),
  ]
  push('欢迎', '通知中心已就绪')
})

onUnmounted(() => {
  for (const off of offs) {
    off()
  }
})

function clearAll() {
  notices.value = []
  void ui
}
</script>

<template>
  <div class="yw-nc-overlay" @pointerdown.self="emit('close')">
    <div class="yw-nc" role="dialog" aria-label="通知中心">
      <header class="yw-nc-header">
        <b>通知中心</b>
        <button class="yw-nc-clear" @click="clearAll">
          全部清除
        </button>
      </header>
      <div class="yw-nc-list">
        <div v-for="n in notices" :key="n.id" class="yw-nc-item">
          <div class="yw-nc-item-head">
            <b>{{ n.title }}</b>
            <span>{{ n.time }}</span>
          </div>
          <p>{{ n.body }}</p>
        </div>
        <p v-if="!notices.length" class="yw-nc-empty">
          暂无通知
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yw-nc-overlay {
  position: fixed;
  inset: 0;
  z-index: 9400;
}

.yw-nc {
  position: absolute;
  top: calc(var(--yw-menubar-h) + 6px);
  right: 12px;
  display: flex;
  flex-direction: column;
  width: 360px;
  max-height: 70vh;
  overflow: hidden;
  background: var(--yw-glass-panel-bg);
  border-radius: var(--yw-radius-notification);
  box-shadow: var(--yw-shadow-menu);
  backdrop-filter: blur(var(--yw-blur-strong)) saturate(180%);
  animation: yw-cc-in 0.25s var(--yw-ease-spring);
}

.yw-nc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px 8px;
  font-size: 14px;
  color: oklch(var(--yw-foreground));
}

.yw-nc-clear {
  padding: 2px 10px;
  font-size: 12px;
  color: oklch(var(--yw-primary));
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 6px;
}

.yw-nc-clear:hover {
  background: var(--yw-label-4);
}

.yw-nc-list {
  flex: 1;
  padding: 0 10px 10px;
  overflow-y: auto;
}

.yw-nc-item {
  padding: 10px 12px;
  margin-bottom: 8px;
  background: var(--yw-control-bg);
  border-radius: var(--yw-radius-notification);
  box-shadow: 0 0 0 0.5px var(--yw-separator);
}

.yw-nc-item-head {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
}

.yw-nc-item-head span {
  color: var(--yw-label-2);
}

.yw-nc-item p {
  margin: 4px 0 0;
  font-size: 12px;
  color: oklch(var(--yw-foreground));
}

.yw-nc-empty {
  padding: 30px;
  font-size: 12px;
  color: var(--yw-label-3);
  text-align: center;
}
</style>
