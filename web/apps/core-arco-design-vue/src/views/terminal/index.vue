<script setup lang="ts">
import Workspace from '@/components/YdTerminal/Workspace.vue'

defineOptions({
  name: 'TerminalIndex',
})
</script>

<template>
  <div class="flex h-full flex-col">
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="square-terminal" :size="24" />
          <span>终端</span>
        </div>
      </template>
      <template #description>
        <span>服务器交互式终端工作台（文件树 · 多标签终端 · 实时监控）</span>
      </template>
    </FaPageHeader>

    <FaPageMain class="min-h-0 flex-1!">
      <!-- 全高工作台：撑满页头以下剩余空间 -->
      <div class="h-[calc(100vh-190px)] min-h-100">
        <Workspace />
      </div>
    </FaPageMain>
  </div>
</template>
