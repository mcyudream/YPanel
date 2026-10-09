<script setup lang="ts">
import apiContainer from '@/api/modules/container'
import { i18n } from '@/locales'

// 容器日志查看器：流式拉取（跟踪/行数/时间戳）+ YdLogViewer 统一渲染（级别筛选/着色/关键字过滤）。
const props = defineProps<{
  containerId: string
  containerName?: string
  active?: boolean
}>()

const appAccountStore = useAppAccountStore()

const content = ref('')
const loading = ref(false)
const following = ref(false)
const tail = ref(500)
const showTs = ref(true)
let abort: AbortController | null = null

function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}

function buildUrl(follow: boolean) {
  return `${wsBase()}/${apiContainer.logsURL(props.containerId, appAccountStore.token, tail.value, follow, showTs.value)}`
}

async function load(follow: boolean) {
  abort?.abort()
  abort = new AbortController()
  following.value = follow
  loading.value = true
  if (!follow) {
    content.value = ''
  }
  try {
    const url = buildUrl(follow)
    const resp = await fetch(url, {
      headers: { Authorization: `Bearer ${appAccountStore.token}` },
      signal: abort.signal,
    })
    if (!resp.ok || !resp.body) {
      throw new Error(`HTTP ${resp.status}`)
    }
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    for (;;) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }
      content.value += decoder.decode(value, { stream: true })
      if (content.value.length > 4_000_000) {
        content.value = content.value.slice(-2_000_000)
      }
    }
  }
  catch (e: any) {
    if (e?.name !== 'AbortError') {
      content.value += `\n${i18n.global.t('container.common.logStreamError', { msg: e?.message || e })}`
    }
  }
  finally {
    following.value = false
    loading.value = false
  }
}

function stop() {
  abort?.abort()
  following.value = false
}

function download() {
  const blob = new Blob([content.value], { type: 'text/plain;charset=utf-8' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `${props.containerName || props.containerId}-logs.txt`
  a.click()
  URL.revokeObjectURL(a.href)
}

function clearView() {
  content.value = ''
}

watch(() => props.containerId, () => {
  content.value = ''
  load(false)
})

onMounted(() => {
  load(false)
})

onBeforeUnmount(() => {
  abort?.abort()
})
</script>

<template>
  <div class="flex h-full flex-col gap-2">
    <div class="flex flex-wrap items-center gap-2">
      <FaButton size="sm" :variant="following ? 'default' : 'outline'" @click="following ? stop() : load(true)">
        <FaIcon name="i-lucide:radio" class="mr-1" :class="following ? 'animate-pulse' : ''" />
        {{ following ? $t('container.common.following') : $t('container.common.followLogs') }}
      </FaButton>
      <FaButton size="sm" variant="outline" @click="load(false)">
        <FaIcon name="i-lucide:refresh-cw" class="mr-1" />
        {{ $t('common.refresh') }}
      </FaButton>
      <label class="flex items-center gap-1 text-xs text-muted-foreground">
        {{ $t('container.logViewer.lines') }}
        <YdSelect
          v-model="tail"
          :options="[100, 500, 1000, 5000, 10000]"
          size="sm"
          @update:model-value="load(false)"
        />
      </label>
      <label class="flex items-center gap-1 text-xs text-muted-foreground">
        <input v-model="showTs" type="checkbox" class="accent-[var(--primary)]" @change="load(false)">
        {{ $t('container.logViewer.timestamps') }}
      </label>
      <div class="ml-auto flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="download">
          <FaIcon name="i-lucide:download" class="mr-1" /> {{ $t('common.download') }}
        </FaButton>
        <FaButton variant="outline" size="icon-sm" :title="$t('container.logViewer.clearView')" @click="clearView">
          <FaIcon name="i-lucide:eraser" class="text-sm" />
        </FaButton>
      </div>
    </div>
    <YdLogViewer :logs="content" height="100%" :loading="loading" class="min-h-0 flex-1" />
  </div>
</template>
