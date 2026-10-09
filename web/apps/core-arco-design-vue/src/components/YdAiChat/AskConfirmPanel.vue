<script setup lang="ts">
// YdAiAskConfirmPanel：危险操作挂起时替换 Sender 输入框的批准面板（ZCode 式）。
// 工具名 + 风险徽标 + 参数滚动区 + 拒绝/批准；回执由 aiAskStore.settle 转发后端。
import { computed } from 'vue'
import { useAiAskStore } from '@/store/modules/aiAsk'
import { i18n } from '@/locales'
import { toolMetaOf } from '@/components/YdAiChat/toolMeta'

const store = useAiAskStore()

const payload = computed(() => store.payload)

const prettyArgs = computed(() => {
  const raw = store.payload?.args || ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  }
  catch {
    return raw || i18n.global.t('components.ydAiChat.noArgs')
  }
})

function resolve(approve: boolean) {
  store.settle(approve)
}
</script>

<template>
  <div
    v-if="payload"
    class="rounded-xl border p-3"
    :class="payload.risk === 'danger' ? 'border-red-500/40 bg-red-500/5' : 'border-amber-500/40 bg-amber-500/5'"
  >
    <div class="flex items-start gap-2.5">
      <span
        class="flex size-9 shrink-0 items-center justify-center rounded-lg"
        :class="payload.risk === 'danger' ? 'bg-red-500/10 text-red-500' : 'bg-amber-500/10 text-amber-600'"
      >
        <FaIcon :name="payload.risk === 'danger' ? 'i-lucide:triangle-alert' : 'i-lucide:pen-line'" class="text-lg" />
      </span>
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-1.5 text-sm font-medium">
          <span>{{ toolMetaOf(payload.tool).label }}</span>
          <span
            class="rounded-full px-2 py-0.5 text-[11px]"
            :class="payload.risk === 'danger' ? 'bg-red-500/10 text-red-500' : 'bg-amber-500/10 text-amber-600'"
          >
            {{ payload.risk === 'danger' ? $t('components.ydAiChat.riskDanger') : $t('components.ydAiChat.riskWrite') }}
          </span>
        </div>
        <div class="mt-0.5 text-xs text-muted-foreground">
          {{ $t('components.ydAiChat.confirmPrefix') }}{{ payload.risk === 'danger' ? $t('components.ydAiChat.confirmDangerInfix') : '' }}{{ $t('components.ydAiChat.confirmSuffix') }}
        </div>
      </div>
    </div>
    <pre class="mt-2 max-h-40 overflow-y-auto whitespace-pre-wrap break-words rounded-md bg-muted/60 px-3 py-2 font-mono text-[11px] leading-relaxed">{{ prettyArgs }}</pre>
    <div class="mt-2.5 flex justify-end gap-2">
      <FaButton variant="outline" @click="resolve(false)">
        {{ $t('components.ydAiChat.reject') }}
      </FaButton>
      <FaButton :variant="payload.risk === 'danger' ? 'destructive' : 'default'" @click="resolve(true)">
        <FaIcon name="i-lucide:check" class="mr-1" />
        {{ $t('components.ydAiChat.approve') }}
      </FaButton>
    </div>
  </div>
</template>
