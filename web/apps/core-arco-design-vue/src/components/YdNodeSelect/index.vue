<script setup lang="ts">
/**
 * YdNodeSelect — 统一节点选择器（M63）：全站「选节点」的唯一入口。
 * 身份显示口径收敛于此处维护：name（配对唯一名）为主 + hostname 副文本
 * （hostname 可重复，禁止作主 label）；本地节点显示本地词条；离线节点禁用并标注。
 * 节点列表默认自拉（silent），也可经 nodes 透传复用页面已有数据。
 */
import api from '@/api'
import type { NodeItem } from '@/api/modules/node'
import { nodeDisplayName } from '@/api/modules/node'
import { i18n } from '@/locales'
import { FaButton, FaDropdown, FaIcon } from '@fantastic-admin/components'

defineOptions({ name: 'YdNodeSelect' })

/** 调用方透传的最小节点形状（完整 NodeItem 或页面瘦身形状均可） */
export interface YdNodeItem {
  id: string
  name: string
  hostname?: string
  /** 缺省视为在线 */
  online?: boolean
}

const props = withDefaults(defineProps<{
  /** 透传节点列表（不传则自拉 api/v1/nodes） */
  nodes?: YdNodeItem[]
  /** 节点 id；'' / undefined 视为 local */
  modelValue?: string
  /** local 在调用方值域中的表示（表单常用 ''，默认 'local'） */
  localValue?: string
  disabled?: boolean
  size?: 'sm' | 'default' | 'icon-sm'
  /** 触发按钮附加类（宽度等） */
  buttonClass?: string
  /** 自定义副文本（覆盖默认的 hostname·离线口径，如安装向导标注架构） */
  describe?: (n: YdNodeItem) => string
}>(), {
  nodes: undefined,
  modelValue: '',
  localValue: 'local',
  disabled: false,
  size: 'sm',
  buttonClass: '',
  describe: undefined,
})

const emit = defineEmits<{
  'update:modelValue': [id: string]
  /** 选中变化，携带完整节点对象 */
  change: [node: YdNodeItem | undefined]
  /** 列表就绪（自拉或透传首次同步），供调用方做合法性校验 */
  loaded: [nodes: YdNodeItem[]]
}>()

const fetched = ref<NodeItem[]>([])
const list = computed<YdNodeItem[]>(() => props.nodes ?? fetched.value)

const currentId = computed(() => (props.modelValue || 'local') === props.localValue ? 'local' : (props.modelValue || 'local'))
const currentNode = computed(() => list.value.find(n => n.id === currentId.value))

/** 选中项展示名：local 用本地词条，远程走 nodeDisplayName 唯一口径 */
const currentLabel = computed(() => {
  if (currentId.value === 'local') {
    return i18n.global.t('components.ydNodeSelect.local')
  }
  return currentNode.value ? nodeDisplayName(currentNode.value) : currentId.value
})

/** 副文本：默认 hostname 仅在与 name 不同时展示 + 离线标注；可经 describe 覆盖 */
function itemDescription(n: YdNodeItem) {
  if (props.describe) {
    return props.describe(n)
  }
  const parts: string[] = []
  if (n.hostname && n.hostname !== n.name) {
    parts.push(n.hostname)
  }
  if (n.id !== 'local' && n.online === false) {
    parts.push(i18n.global.t('components.ydNodeSelect.offline'))
  }
  return parts.join(' · ')
}

const menuItems = computed(() => [list.value.map(n => ({
  label: n.id === 'local' ? i18n.global.t('components.ydNodeSelect.local') : nodeDisplayName(n),
  description: itemDescription(n),
  disabled: n.id !== 'local' && n.online === false,
  handle: () => pick(n.id),
}))])

onMounted(async () => {
  if (props.nodes) {
    emit('loaded', props.nodes)
    return
  }
  try {
    const res = await api.get('api/v1/nodes', { silent: true })
    fetched.value = (res.data as NodeItem[]) || []
    emit('loaded', fetched.value)
  }
  catch {}
})

watch(() => props.nodes, (v) => {
  if (v) {
    emit('loaded', v)
  }
})

function pick(id: string) {
  if (id === currentId.value) {
    return
  }
  emit('update:modelValue', id === 'local' ? props.localValue : id)
  emit('change', list.value.find(n => n.id === id))
}

const sizeCls = computed(() => ({ default: 'h-9 text-sm', sm: 'h-8 text-sm', 'icon-sm': 'h-7 text-xs' }[props.size] ?? 'h-8'))
</script>

<template>
  <FaDropdown :items="menuItems">
    <FaButton
      variant="outline"
      :size="size"
      :disabled="disabled"
      :class="[sizeCls, buttonClass]"
      class="w-36 max-w-full justify-start gap-1 font-normal"
      :title="currentLabel"
    >
      <FaIcon name="i-lucide:server" class="shrink-0 text-xs text-muted-foreground" />
      <span class="truncate">{{ currentLabel }}</span>
      <FaIcon name="i-lucide:chevron-down" class="ml-auto shrink-0 text-xs text-muted-foreground" />
    </FaButton>
  </FaDropdown>
</template>
