<script setup lang="ts">
import { ref } from 'vue'

defineOptions({
  name: 'IconsIndex',
})

// 图标选择器演示 + morph 动效演示
const picked = ref<string>('')
const morphA = ref('folder')
const morphB = ref('folder-open')

// 常用图标速览
const quickPicks = [
  'server', 'hard-drive', 'cpu', 'container', 'folder', 'file', 'terminal', 'database',
  'shield-check', 'lock', 'key-round', 'globe', 'network', 'activity', 'gauge', 'bell',
  'settings', 'power', 'refresh-cw', 'download', 'upload', 'play', 'pause', 'trash',
]
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="shapes" :size="24" />
          <span>{{ $t('icons.page.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('icons.page.desc') }}</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <!-- 选择器演示 -->
      <div class="rounded-lg border bg-background p-5">
        <div class="mb-3 flex items-center gap-2 text-sm font-medium">
          <YdMorphIcon name="text-cursor-input" :size="16" />
          {{ $t('icons.picker.title') }}
        </div>
        <div class="flex flex-wrap items-center gap-4">
          <YdIconPicker v-model="picked" />
          <span class="text-sm text-muted-foreground">
            {{ $t('icons.picker.current') }}<code class="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">{{ picked || $t('icons.picker.none') }}</code>
          </span>
        </div>
      </div>

      <!-- morph 动效演示 -->
      <div class="mt-4 rounded-lg border bg-background p-5">
        <div class="mb-3 flex items-center gap-2 text-sm font-medium">
          <YdMorphIcon name="wand" :size="16" />
          {{ $t('icons.morph.title') }}
        </div>
        <div class="flex flex-wrap items-center gap-6">
          <button
            type="button"
            class="flex cursor-pointer flex-col items-center gap-2 rounded-lg border p-6 transition-colors hover:bg-accent/40"
            :title="$t('icons.morph.tip')"
            @click="[morphA, morphB] = [morphB, morphA]"
          >
            <YdMorphIcon :name="morphA === morphB ? morphA : morphA" :size="48" :stroke-width="1.6" />
            <code class="text-xs text-muted-foreground">{{ morphA }}</code>
          </button>
          <div class="text-muted-foreground">
            <FaIcon name="i-lucide:arrow-right" class="size-6" />
          </div>
          <div class="flex flex-col items-center gap-2 rounded-lg border border-dashed p-6">
            <YdMorphIcon :name="morphB" :size="48" :stroke-width="1.6" />
            <code class="text-xs text-muted-foreground">{{ morphB }}</code>
          </div>
          <div class="flex flex-wrap gap-1.5">
            <button
              v-for="name in ['folder-open', 'folder', 'lock', 'lock-open', 'bell', 'bell-off', 'eye', 'eye-off', 'sun', 'moon', 'play', 'pause', 'heart', 'star']"
              :key="name"
              type="button"
              class="cursor-pointer rounded-md border px-2 py-1 font-mono text-xs text-muted-foreground transition-colors hover:bg-accent/50"
              @click="morphB = name"
            >
              {{ name }}
            </button>
          </div>
        </div>
      </div>

      <!-- 常用图标速览 -->
      <div class="mt-4 rounded-lg border bg-background p-5">
        <div class="mb-3 flex items-center gap-2 text-sm font-medium">
          <YdMorphIcon name="layout-grid" :size="16" />
          {{ $t('icons.quick.title') }}
        </div>
        <div class="grid grid-cols-6 gap-2 sm:grid-cols-8 md:grid-cols-12">
          <div
            v-for="name in quickPicks"
            :key="name"
            class="flex flex-col items-center gap-1 rounded-md border p-2 transition-colors hover:bg-accent/40"
            :title="name"
          >
            <YdMorphIcon :name="name" :size="22" />
            <span class="w-full truncate text-center text-[10px] text-muted-foreground">{{ name }}</span>
          </div>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
