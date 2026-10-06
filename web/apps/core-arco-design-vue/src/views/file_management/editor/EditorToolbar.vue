<script setup lang="ts">
// 顶部工具栏：节点选择 | 撤销重做/查找/格式化 | 保存/全存 | 切分/侧栏/底栏 | 关闭
const store = useFileEditorStore()

const nodeOptions = computed(() => store.nodes.map(n => ({
  label: n.hostname ? `${n.hostname}${n.online ? '' : '（离线）'}` : n.id,
  value: n.id,
  disabled: !n.online,
})))

function closeWorkspace() {
  if (store.requestClose() === 'confirm') {
    useFaModal().confirm({
      title: '未保存的修改',
      content: `有 ${store.dirtyCount} 个文件未保存。确定要保存全部并关闭吗？（取消则留在工作台）`,
      onConfirm: async () => {
        await store.saveAll()
        store.visible = false
      },
    })
  }
}

const tbtn = 'size-7!'
</script>

<template>
  <div class="flex h-9 shrink-0 items-center gap-1 border-b bg-muted/30 px-2">
    <!-- 节点 -->
    <FaSelect
      v-model="store.currentNode"
      :options="nodeOptions"
      class="w-40!"
      :disabled="!nodeOptions.length"
    />

    <div class="mx-1 h-5 w-px bg-border" />

    <!-- 编辑动作 -->
    <FaButton variant="ghost" size="icon-sm" :class="tbtn" title="撤销" :disabled="!store.activeTab" @click="store.runActiveAction('undo')">
      <FaIcon name="i-lucide:undo-2" class="text-sm" />
    </FaButton>
    <FaButton variant="ghost" size="icon-sm" :class="tbtn" title="重做" :disabled="!store.activeTab" @click="store.runActiveAction('redo')">
      <FaIcon name="i-lucide:redo-2" class="text-sm" />
    </FaButton>
    <FaButton variant="ghost" size="icon-sm" :class="tbtn" title="查找 (Ctrl+F)" :disabled="!store.activeTab" @click="store.runActiveAction('actions.find')">
      <FaIcon name="i-lucide:search" class="text-sm" />
    </FaButton>
    <FaButton variant="ghost" size="icon-sm" :class="tbtn" title="格式化文档" :disabled="!store.activeTab" @click="store.runActiveAction('editor.action.formatDocument')">
      <FaIcon name="i-lucide:braces" class="text-sm" />
    </FaButton>

    <div class="mx-1 h-5 w-px bg-border" />

    <!-- 保存 -->
    <FaButton variant="ghost" size="sm" class="h-7! gap-1 px-2 text-xs" :disabled="!store.activeTab" :loading="store.activeTab?.saving" title="保存 (Ctrl+S)" @click="store.activeTabId && store.save(store.activeTabId)">
      <FaIcon name="i-lucide:save" class="text-sm" />
      保存
    </FaButton>
    <FaButton variant="ghost" size="sm" class="h-7! gap-1 px-2 text-xs" :disabled="!store.dirtyCount" title="全部保存 (Ctrl+Shift+S)" @click="store.saveAll">
      <FaIcon name="i-lucide:save-all" class="text-sm" />
      全部保存<span v-if="store.dirtyCount" class="text-primary">（{{ store.dirtyCount }}）</span>
    </FaButton>

    <!-- 布局 -->
    <div class="ml-auto flex items-center gap-1">
      <FaButton variant="ghost" size="icon-sm" :class="tbtn" title="切分编辑器 (Ctrl+\\)" :disabled="!store.activeTabId" @click="store.activeTabId && store.splitFromTab(store.activeTabId)">
        <FaIcon name="i-lucide:columns-2" class="text-sm" />
      </FaButton>
      <div class="mx-0.5 h-5 w-px bg-border" />
      <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="store.layout.sidebarVisible ? '隐藏侧栏 (Ctrl+B)' : '显示侧栏 (Ctrl+B)'" @click="store.toggleSidebar">
        <FaIcon name="i-lucide:panel-left" class="text-sm" :class="store.layout.sidebarVisible ? 'text-primary' : ''" />
      </FaButton>
      <FaButton
        v-if="store.layout.sidebarVisible"
        variant="ghost"
        size="icon-sm"
        :class="tbtn"
        :title="store.layout.sidebarSide === 'left' ? '侧栏移到右侧' : '侧栏移到左侧'"
        @click="store.layout.sidebarSide = store.layout.sidebarSide === 'left' ? 'right' : 'left'"
      >
        <FaIcon :name="store.layout.sidebarSide === 'left' ? 'i-lucide:arrow-right-to-line' : 'i-lucide:arrow-left-to-line'" class="text-sm" />
      </FaButton>
      <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="store.layout.terminalVisible ? '隐藏终端 (Ctrl+J)' : '显示终端 (Ctrl+J)'" @click="store.toggleTerminal">
        <FaIcon name="i-lucide:square-terminal" class="text-sm" :class="store.layout.terminalVisible ? 'text-primary' : ''" />
      </FaButton>
      <div class="mx-0.5 h-5 w-px bg-border" />
      <FaButton variant="ghost" size="icon-sm" :class="tbtn" title="关闭工作台" @click="closeWorkspace">
        <FaIcon name="i-lucide:x" class="text-sm" />
      </FaButton>
    </div>
  </div>
</template>
