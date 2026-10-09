<script setup lang="ts">
import JsonNode from './JsonNode.vue'
import { i18n } from '@/locales'

// JSON 可视化（树形折叠 + 类型着色 + 过滤 + 复制），用于容器 Inspect 等原始 JSON 展示。
const props = withDefaults(defineProps<{
  data: unknown
  defaultExpandLevel?: number
}>(), {
  defaultExpandLevel: 2,
})

const expandLevel = ref(props.defaultExpandLevel)
const expandSignal = ref(0)
const expandDirection = ref<'all' | 'none'>('all')
const keyword = ref('')
const copied = ref(false)

const dataText = computed(() => {
  try {
    return JSON.stringify(props.data, null, 2)
  }
  catch {
    return ''
  }
})

const isEmpty = computed(() => props.data === null || props.data === undefined)

function expandAll() {
  expandDirection.value = 'all'
  expandSignal.value++
}

function collapseAll() {
  expandDirection.value = 'none'
  expandSignal.value++
}

async function copy() {
  try {
    await navigator.clipboard.writeText(dataText.value)
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  }
  catch {
    useFaToast().error(i18n.global.t('components.ydJsonViewer.copyFailed'))
  }
}
</script>

<template>
  <div class="flex flex-col gap-2">
    <div class="flex flex-wrap items-center gap-2">
      <select v-model.number="expandLevel" class="h-8 rounded-md border bg-background px-2 text-xs outline-none" :title="$t('components.ydJsonViewer.defaultExpandLevel')">
        <option :value="1">
          {{ $t('components.ydJsonViewer.expandN', { n: 1 }) }}
        </option>
        <option :value="2">
          {{ $t('components.ydJsonViewer.expandN', { n: 2 }) }}
        </option>
        <option :value="3">
          {{ $t('components.ydJsonViewer.expandN', { n: 3 }) }}
        </option>
        <option :value="99">
          {{ $t('components.ydJsonViewer.expandAllOption') }}
        </option>
      </select>
      <FaButton variant="outline" size="sm" @click="expandAll">
        {{ $t('components.ydJsonViewer.expandAllBtn') }}
      </FaButton>
      <FaButton variant="outline" size="sm" @click="collapseAll">
        {{ $t('components.ydJsonViewer.collapse') }}
      </FaButton>
      <FaInput v-model="keyword" :placeholder="$t('components.ydJsonViewer.filterPlaceholder')" class="h-8 w-48!" />
      <FaButton variant="outline" size="sm" class="ml-auto" @click="copy">
        <FaIcon :name="copied ? 'i-lucide:check' : 'i-lucide:copy'" class="mr-1" />
        {{ copied ? $t('common.copied') : $t('components.ydJsonViewer.copyJson') }}
      </FaButton>
    </div>
    <div class="max-h-[calc(100vh-380px)] overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs leading-relaxed">
      <JsonNode
        v-if="!isEmpty" name="root" :value="data" :depth="0" :keyword="keyword.trim()"
        :expand-level="expandLevel" :expand-signal="expandSignal" :expand-direction="expandDirection" root
      />
      <div v-else class="text-muted-foreground">
        {{ $t('components.ydJsonViewer.empty') }}
      </div>
    </div>
  </div>
</template>
