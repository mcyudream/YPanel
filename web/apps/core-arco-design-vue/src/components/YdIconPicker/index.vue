<script setup lang="ts">
// YdIconPicker 图标选择器（yd- 前缀自研组件，视觉与交互对齐 fa）。
// v-model: 图标名（yd: 前缀完整名，如 "yd:server"；空串表示未选择）
// 特性：名称/关键词搜索、分类过滤、虚拟滚动、morph 动效预览、清除。
import { computed, ref, watch } from 'vue'

defineOptions({
  name: 'YdIconPicker',
})

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  clearable?: boolean
  disabled?: boolean
}>(), {
  placeholder: '选择图标',
  clearable: true,
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'change': [value: string]
}>()

const visible = ref(false)
const keyword = ref('')
const activeCategory = ref('全部')

type IconMeta = { name: string, categories: string[], tags: string[] }
const allIcons = ref<IconMeta[]>([])
const loading = ref(false)

const CATEGORIES = ['全部', '服务器', '文件', '容器与部署', '系统操作', '安全', '状态', '导航', '编辑', '媒体', '时间']

watch(visible, async (v) => {
  if (!v || allIcons.value.length) {
    return
  }
  loading.value = true
  try {
    const mod = await import('@/ui/icons/data.json')
    const icons = (mod.default as any).icons ?? {}
    allIcons.value = Object.entries(icons).map(([name, meta]: [string, any]) => ({
      name,
      categories: meta.categories ?? [],
      tags: meta.tags ?? [],
    }))
  }
  finally {
    loading.value = false
  }
})

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return allIcons.value.filter((it) => {
    if (activeCategory.value !== '全部' && !it.categories.includes(activeCategory.value)) {
      return false
    }
    if (!kw) {
      return true
    }
    return it.name.includes(kw) || it.tags.some(t => t.includes(kw))
  })
})

// ---- 虚拟滚动 ----
const CELL = 64
const BUFFER_ROWS = 3
const gridRef = ref<HTMLElement>()
const scrollTop = ref(0)
const viewportHeight = ref(420)

function onScroll() {
  if (gridRef.value) {
    scrollTop.value = gridRef.value.scrollTop
  }
}

const columns = computed(() => {
  const w = gridRef.value?.clientWidth ?? 600
  return Math.max(4, Math.floor(w / CELL))
})

const totalRows = computed(() => Math.ceil(filtered.value.length / columns.value))
const startRow = computed(() => Math.max(0, Math.floor(scrollTop.value / CELL) - BUFFER_ROWS))
const endRow = computed(() => Math.min(totalRows.value, Math.ceil((scrollTop.value + viewportHeight.value) / CELL) + BUFFER_ROWS))
const visibleIcons = computed(() => {
  const out: { name: string, index: number }[] = []
  for (let row = startRow.value; row < endRow.value; row++) {
    for (let col = 0; col < columns.value; col++) {
      const index = row * columns.value + col
      if (index >= filtered.value.length) {
        break
      }
      out.push({ name: filtered.value[index].name, index })
    }
  }
  return out
})
const padTop = computed(() => startRow.value * CELL)

// ---- 选择 ----
const currentName = computed(() => props.modelValue?.replace(/^yd:/, '') || '')

function select(name: string) {
  const val = `yd:${name}`
  emit('update:modelValue', val)
  emit('change', val)
  visible.value = false
}

function clear() {
  emit('update:modelValue', '')
  emit('change', '')
}

watch(keyword, () => {
  scrollTop.value = 0
  if (gridRef.value) {
    gridRef.value.scrollTop = 0
  }
})
watch(activeCategory, () => {
  scrollTop.value = 0
  if (gridRef.value) {
    gridRef.value.scrollTop = 0
  }
})
</script>

<template>
  <div class="inline-flex items-center gap-2">
    <!-- 触发器：图标预览 + 名称，视觉对齐 fa 输入框 -->
    <button
      type="button"
      class="yicon-trigger inline-flex cursor-pointer items-center gap-2 rounded-md border border-input bg-background px-3 py-1.5 text-sm text-foreground transition-colors hover:bg-accent/50 disabled:cursor-not-allowed disabled:opacity-50"
      :disabled="disabled"
      @click="visible = true"
    >
      <YdMorphIcon v-if="currentName" :name="currentName" :size="18" />
      <span class="i-radix-icons:mixer-horizontal opacity-50" style="font-size: 16px;" />
      <span :class="currentName ? '' : 'opacity-50'">{{ currentName || placeholder }}</span>
    </button>
    <FaButton
      v-if="clearable && currentName && !disabled"
      variant="ghost"
      size="icon"
      class="size-7"
      title="清除"
      @click="clear"
    >
      <FaIcon name="i-ri:close-line" class="text-sm" />
    </FaButton>

    <FaModal
      v-model="visible"
      title="选择图标"
      class="lg:max-w-3xl"
      content-class="bg-[var(--g-main-area-bg)]"
    >
      <div class="flex flex-col gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <FaInput
            v-model="keyword"
            placeholder="搜索图标名或关键词，如 server / folder / lock"
            class="w-64"
          >
            <template #start>
              <FaIcon name="i-ri:search-line" class="text-muted-foreground" />
            </template>
          </FaInput>
          <div class="flex flex-1 flex-wrap gap-1.5">
            <button
              v-for="cat in CATEGORIES"
              :key="cat"
              type="button"
              class="cursor-pointer rounded-md border px-2 py-0.5 text-xs transition-colors"
              :class="activeCategory === cat
                ? 'border-primary bg-primary text-primary-foreground'
                : 'border-border text-muted-foreground hover:bg-accent/50'"
              @click="activeCategory = cat"
            >
              {{ cat }}
            </button>
          </div>
        </div>

        <div
          ref="gridRef"
          class="relative overflow-y-auto rounded-md border border-border"
          style="height: 420px;"
          @scroll.passive="onScroll"
        >
          <div v-if="loading" class="p-8 text-center text-muted-foreground text-sm">
            图标库加载中…
          </div>
          <div v-else-if="!filtered.length" class="p-8 text-center text-muted-foreground text-sm">
            未找到匹配图标
          </div>
          <div v-else class="relative" :style="{ height: `${totalRows * CELL}px` }">
            <div class="absolute inset-x-0" :style="{ top: `${padTop}px` }">
              <div class="grid" :style="{ gridTemplateColumns: `repeat(${columns}, ${CELL}px)` }">
                <button
                  v-for="it in visibleIcons"
                  :key="it.name"
                  type="button"
                  class="group m-0.5 flex cursor-pointer flex-col items-center justify-center gap-1 rounded-md border p-1.5 transition-colors"
                  :class="currentName === it.name
                    ? 'border-primary bg-primary/10'
                    : 'border-transparent hover:border-border hover:bg-accent/50'"
                  :title="it.name"
                  @click="select(it.name)"
                >
                  <YdMorphIcon :name="it.name" :size="22" />
                  <span class="w-full truncate text-center text-[10px] text-muted-foreground leading-none">{{ it.name }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="flex items-center justify-between text-muted-foreground text-xs">
          <span>共 {{ filtered.length }} 枚 · 选中：{{ currentName || '无' }}</span>
          <span>图标随构建本地打包（lucide / ISC）</span>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="visible = false">
          关闭
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
