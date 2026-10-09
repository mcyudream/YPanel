<script setup lang="ts">
import type { FileEntry } from '@/api/modules/file'
import apiFile from '@/api/modules/file'
import { i18n } from '@/locales'

// YdDirPicker 轻量目录选择器弹窗：面包屑导航 + 仅目录列表 + 选择当前目录。
// 组件自持 FaModal 与 visible（defineModel），随打开重建加载（FaModal 插槽 defineModel 断链规避，见 exp/frontend.md）。
const visible = defineModel<boolean>('visible', { default: false })

const props = withDefaults(defineProps<{
  /** 宿主节点 id（默认 local） */
  node?: string
  /** 初始目录 */
  initialPath?: string
  title?: string
}>(), {
  node: 'local',
  initialPath: '/',
  title: undefined,
})

const emits = defineEmits<{
  select: [path: string]
}>()

const current = ref(props.initialPath || '/')
const dirs = ref<FileEntry[]>([])
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const resp = await apiFile.list(current.value, props.node)
    dirs.value = (resp.entries || []).filter(e => e.isDir).sort((a, b) => a.name.localeCompare(b.name))
  }
  catch (e: any) {
    error.value = e?.message || i18n.global.t('components.ydDirPicker.loadFailed')
    dirs.value = []
  }
  finally {
    loading.value = false
  }
}

function enter(path: string) {
  current.value = path
  load()
}

function up() {
  const p = current.value.replace(/\/+$/, '')
  if (p === '') {
    current.value = '/'
  }
  else {
    const i = p.lastIndexOf('/')
    current.value = i <= 0 ? '/' : p.slice(0, i)
  }
  load()
}

/** 面包屑分段（/ 为根） */
const crumbs = computed(() => {
  const out = [{ name: '/', path: '/' }]
  let acc = ''
  for (const seg of current.value.split('/').filter(Boolean)) {
    acc += `/${seg}`
    out.push({ name: seg, path: acc })
  }
  return out
})

function confirm() {
  emits('select', current.value)
  visible.value = false
}

watch(visible, (v) => {
  if (v) {
    current.value = props.initialPath || '/'
    load()
  }
})
</script>

<template>
  <FaModal v-model="visible" :title="title ?? $t('components.ydDirPicker.title')" :destroy-on-close="true">
    <div class="flex flex-col gap-2">
      <div class="flex items-center gap-1 overflow-x-auto rounded-md border bg-muted/40 px-2 py-1.5 text-xs whitespace-nowrap">
        <button
          v-for="(c, i) in crumbs" :key="c.path" type="button"
          class="font-mono hover:text-primary" :class="i === crumbs.length - 1 ? 'text-primary' : 'text-muted-foreground'"
          @click="enter(c.path)"
        >
          {{ c.name }}<span v-if="i < crumbs.length - 1" class="mx-0.5 text-muted-foreground">/</span>
        </button>
      </div>
      <div class="max-h-72 min-h-48 overflow-y-auto rounded-md border">
        <button
          type="button"
          class="flex w-full items-center gap-2 border-b px-3 py-2 text-left text-sm last:border-b-0 hover:bg-muted/50"
          :disabled="current === '/'"
          @click="up"
        >
          <FaIcon name="i-lucide:corner-left-up" class="text-muted-foreground" /> ..
        </button>
        <div v-if="loading" class="px-3 py-6 text-center text-xs text-muted-foreground">{{ $t('common.loading') }}</div>
        <div v-else-if="error" class="px-3 py-6 text-center text-xs text-red-500">{{ error }}</div>
        <div v-else-if="!dirs.length" class="px-3 py-6 text-center text-xs text-muted-foreground">{{ $t('components.ydDirPicker.emptyDir') }}</div>
        <button
          v-for="d in dirs" :key="d.path" type="button"
          class="flex w-full items-center gap-2 border-b px-3 py-2 text-left text-sm last:border-b-0 hover:bg-muted/50"
          @click="enter(d.path)"
        >
          <FaIcon name="i-lucide:folder" class="text-sky-500" />
          <span class="truncate font-mono text-xs">{{ d.name }}</span>
        </button>
      </div>
      <div class="text-xs text-muted-foreground">{{ $t('components.ydDirPicker.currentSelection') }}<span class="font-mono">{{ current }}</span></div>
    </div>
    <template #footer>
      <FaButton variant="outline" @click="visible = false">{{ $t('common.cancel') }}</FaButton>
      <FaButton @click="confirm">{{ $t('components.ydDirPicker.selectCurrent') }}</FaButton>
    </template>
  </FaModal>
</template>
