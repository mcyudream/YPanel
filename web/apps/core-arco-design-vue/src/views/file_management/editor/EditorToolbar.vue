<script setup lang="ts">
// 顶部工具栏：撤销重做/查找/格式化 | 保存/全存 | 切分/侧栏/底栏 | 最小化/关闭（节点选择在文件树头部）
import { i18n } from '@/locales'

const store = useFileEditorStore()

function closeWorkspace() {
  if (store.requestClose() === 'confirm') {
    const reasons: string[] = []
    if (store.dirtyCount > 0) {
      reasons.push(i18n.global.t('files.editor.unsavedAllContent', { n: store.dirtyCount }))
    }
    if (store.terminalSessions > 0) {
      reasons.push(i18n.global.t('files.editor.closeConfirmTerminals', { n: store.terminalSessions }))
    }
    useFaModal().confirm({
      title: i18n.global.t('files.editor.closeConfirmTitle'),
      content: reasons.join(' '),
      onConfirm: async () => {
        await store.saveAll()
        store.closeAll()
      },
    })
  }
}

const tbtn = 'size-7!'
</script>

<template>
  <div class="flex h-9 shrink-0 items-center gap-1 border-b bg-muted/30 px-2">
    <!-- 编辑动作 -->
    <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="$t('files.editor.undo')" :disabled="!store.activeTab" @click="store.runActiveAction('undo')">
      <FaIcon name="i-lucide:undo-2" class="text-sm" />
    </FaButton>
    <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="$t('files.editor.redo')" :disabled="!store.activeTab" @click="store.runActiveAction('redo')">
      <FaIcon name="i-lucide:redo-2" class="text-sm" />
    </FaButton>
    <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="$t('files.editor.find')" :disabled="!store.activeTab" @click="store.runActiveAction('actions.find')">
      <FaIcon name="i-lucide:search" class="text-sm" />
    </FaButton>
    <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="$t('files.editor.format')" :disabled="!store.activeTab" @click="store.runActiveAction('editor.action.formatDocument')">
      <FaIcon name="i-lucide:braces" class="text-sm" />
    </FaButton>

    <div class="mx-1 h-5 w-px bg-border" />

    <!-- 保存 -->
    <FaButton variant="ghost" size="sm" class="h-7! gap-1 px-2 text-xs" :disabled="!store.activeTab" :loading="store.activeTab?.saving" :title="$t('files.editor.saveTitle')" @click="store.activeTabId && store.save(store.activeTabId)">
      <FaIcon name="i-lucide:save" class="text-sm" />
      {{ $t('common.save') }}
    </FaButton>
    <FaButton variant="ghost" size="sm" class="h-7! gap-1 px-2 text-xs" :disabled="!store.dirtyCount" :title="$t('files.editor.saveAllTitle')" @click="store.saveAll">
      <FaIcon name="i-lucide:save-all" class="text-sm" />
      {{ $t('files.editor.saveAll') }}<span v-if="store.dirtyCount" class="text-primary">（{{ store.dirtyCount }}）</span>
    </FaButton>

    <!-- 布局 -->
    <div class="ml-auto flex items-center gap-1">
      <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="$t('files.editor.split')" :disabled="!store.activeTabId" @click="store.activeTabId && store.splitFromTab(store.activeTabId)">
        <FaIcon name="i-lucide:columns-2" class="text-sm" />
      </FaButton>
      <div class="mx-0.5 h-5 w-px bg-border" />
      <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="store.layout.sidebarVisible ? $t('files.editor.hideSidebar') : $t('files.editor.showSidebar')" @click="store.toggleSidebar">
        <FaIcon name="i-lucide:panel-left" class="text-sm" :class="store.layout.sidebarVisible ? 'text-primary' : ''" />
      </FaButton>
      <FaButton
        v-if="store.layout.sidebarVisible"
        variant="ghost"
        size="icon-sm"
        :class="tbtn"
        :title="store.layout.sidebarSide === 'left' ? $t('files.editor.sidebarToRight') : $t('files.editor.sidebarToLeft')"
        @click="store.layout.sidebarSide = store.layout.sidebarSide === 'left' ? 'right' : 'left'"
      >
        <FaIcon :name="store.layout.sidebarSide === 'left' ? 'i-lucide:arrow-right-to-line' : 'i-lucide:arrow-left-to-line'" class="text-sm" />
      </FaButton>
      <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="store.layout.terminalVisible ? $t('files.editor.hideTerminal') : $t('files.editor.showTerminal')" @click="store.toggleTerminal">
        <FaIcon name="i-lucide:square-terminal" class="text-sm" :class="store.layout.terminalVisible ? 'text-primary' : ''" />
      </FaButton>
      <div class="mx-0.5 h-5 w-px bg-border" />
      <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="$t('files.editor.minimize')" @click="store.requestMinimize">
        <FaIcon name="i-lucide:chevrons-right" class="text-sm" />
      </FaButton>
      <FaButton variant="ghost" size="icon-sm" :class="tbtn" :title="$t('files.editor.closeWorkspace')" @click="closeWorkspace">
        <FaIcon name="i-lucide:x" class="text-sm" />
      </FaButton>
    </div>
  </div>
</template>
