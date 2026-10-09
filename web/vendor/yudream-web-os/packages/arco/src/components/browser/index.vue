<script setup lang="ts">
import type { WindowInstance as WindowState } from '@yudream/yudream-webos-core'
import { computed, onMounted, ref, watch } from 'vue'
import { resolveInput, useBrowserStore } from '../../stores/browser'

const props = defineProps<{
  win: WindowState
}>()

const browser = useBrowserStore()
const session = computed(() => browser.sessionOf(props.win.id))
const addressInput = ref('')
const editing = ref(false)

const opts = computed(() => browser.options)
const iframeSrc = computed(() => session.value.url)

// iframe key：url 变化即重建（含 reload 场景）
const iframeKey = computed(() => `${props.win.id}:${session.value.history.slice(0, session.value.index + 1).length}:${session.value.url}`)

watch(() => session.value.url, (url) => {
  if (!editing.value) {
    addressInput.value = url
  }
}, { immediate: true })

onMounted(() => {
  // 初始会话已有 url（openUrl 开窗场景）直接加载；否则停在起始页
  addressInput.value = session.value.url
})

function go() {
  const url = resolveInput(addressInput.value, opts.value.searchEngine ?? 'https://www.bing.com/search?q={q}')
  if (url) {
    browser.navigate(props.win.id, url)
  }
}

function newWindow() {
  if (session.value.url) {
    browser.openUrl(session.value.url, 'browser')
  }
}

function openExternal() {
  if (session.value.url) {
    browser.openUrl(session.value.url, 'external')
  }
}

function onIframeLoad() {
  browser.setLoading(props.win.id, false)
}
</script>

<template>
  <div class="yw-browser">
    <div class="yw-browser-bar">
      <button class="yw-browser-btn" :disabled="!browser.canBack(win.id)" title="后退" @click="browser.back(win.id)">
        <i class="i-lucide-arrow-left" />
      </button>
      <button class="yw-browser-btn" :disabled="!browser.canForward(win.id)" title="前进" @click="browser.forward(win.id)">
        <i class="i-lucide-arrow-right" />
      </button>
      <button class="yw-browser-btn" :disabled="!session.url" title="刷新" @click="browser.reload(win.id)">
        <i class="i-lucide-rotate-cw" />
      </button>

      <input
        v-model="addressInput"
        class="yw-browser-address"
        placeholder="输入网址或搜索词，回车访问"
        spellcheck="false"
        @focus="editing = true"
        @blur="editing = false"
        @keydown.enter="go(); ($event.target as HTMLInputElement).blur()"
      >

      <button class="yw-browser-btn" :disabled="!session.url" title="在新窗口打开" @click="newWindow">
        <i class="i-lucide-copy-plus" />
      </button>
      <button class="yw-browser-btn" :disabled="!session.url" title="系统浏览器打开" @click="openExternal">
        <i class="i-lucide-external-link" />
      </button>
    </div>

    <div class="yw-browser-body">
      <div v-if="!session.url" class="yw-browser-start">
        <i class="i-lucide-globe" />
        <p>在上方地址栏输入网址开始浏览</p>
      </div>

      <template v-else>
        <!-- iframe 常驻渲染（@load 才能触发结束 loading）；loading 为覆盖层 -->
        <iframe
          :key="iframeKey"
          :src="iframeSrc"
          class="yw-browser-frame"
          :sandbox="(opts.sandbox ?? ['allow-scripts', 'allow-same-origin', 'allow-forms', 'allow-popups-to-escape-sandbox']).join(' ')"
          :allow="opts.allow ? opts.allow.join('; ') : undefined"
          referrerpolicy="no-referrer"
          @load="onIframeLoad"
        />
        <div v-if="session.loading" class="yw-browser-loading">
          <i class="yw-browser-spin i-lucide-loader-circle" />
          <p>正在加载 {{ session.url }}</p>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.yw-browser {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: oklch(var(--yw-card));
}

.yw-browser-bar {
  display: flex;
  gap: 4px;
  align-items: center;
  padding: 8px 10px;
  background: oklch(var(--yw-muted));
  border-bottom: 1px solid oklch(var(--yw-border));
}

.yw-browser-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  font-size: 15px;
  color: oklch(var(--yw-foreground));
  cursor: pointer;
  background: transparent;
  border: none;
  border-radius: 8px;
}

.yw-browser-btn:hover:not(:disabled) {
  background: oklch(var(--yw-accent) / 70%);
}

.yw-browser-btn:disabled {
  cursor: default;
  opacity: 0.35;
}

.yw-browser-address {
  flex: 1;
  padding: 7px 14px;
  margin: 0 6px;
  font-size: 13px;
  color: oklch(var(--yw-foreground));
  outline: none;
  background: oklch(var(--yw-card));
  border: 1px solid oklch(var(--yw-border));
  border-radius: 999px;
}

.yw-browser-address:focus {
  border-color: oklch(var(--yw-primary));
}

.yw-browser-body {
  position: relative;
  flex: 1;
}

.yw-browser-frame {
  width: 100%;
  height: 100%;
  background: #fff;
  border: none;
}

.yw-browser-start,
.yw-browser-loading {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: oklch(var(--yw-muted-foreground));
}

.yw-browser-start i {
  font-size: 48px;
  opacity: 0.5;
}

.yw-browser-spin {
  font-size: 32px;
  animation: yw-spin 1s linear infinite;
}

@keyframes yw-spin {
  to { transform: rotate(360deg); }
}
</style>
