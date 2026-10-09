<script setup lang="ts">
import Workspace from '@/components/YdTerminal/Workspace.vue'
import { useYwEmbed } from '@/views/desktop/embed'

defineOptions({
  name: 'TerminalIndex',
})

// 桌面工作台窗口内：页头被隐藏，高度直接撑满窗口（经典模式的 100vh-页头 扣除不适用）
const embed = useYwEmbed()

// Dock 拖文件到终端图标：新窗携带待粘贴路径，经 Workspace→Panel 在会话就绪后填入
const props = defineProps<{
  initialPaste?: string
}>()
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <!-- 桌面工作台：页头/卡片包装全免，工作台直接撑满窗口 -->
    <template v-if="embed">
      <Workspace class="min-h-0 flex-1" v-model:initial-paste="props.initialPaste" />
    </template>

    <!-- 经典面板：页头 + 卡片容器（100vh-250px ≈ 面包屑+页头+页边） -->
    <template v-else>
      <FaPageHeader>
        <template #title>
          <div class="flex items-center gap-2">
            <YdMorphIcon name="square-terminal" :size="24" />
            <span>{{ $t('terminal.page.title') }}</span>
          </div>
        </template>
        <template #description>
          <span>{{ $t('terminal.page.desc') }}</span>
        </template>
      </FaPageHeader>

      <FaPageMain class="min-h-0 flex-1!">
        <div class="h-[calc(100vh-250px)] min-h-100">
          <Workspace v-model:initial-paste="props.initialPaste" />
        </div>
      </FaPageMain>
    </template>
  </div>
</template>
