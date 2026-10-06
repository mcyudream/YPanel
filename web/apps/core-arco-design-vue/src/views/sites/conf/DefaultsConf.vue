<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteDefaultsConf } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'

// B23：默认文档列表化（对齐 1Panel——逐条增删、顺序即 index 检查优先级）。

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const conf = ref<SiteDefaultsConf | null>(null)
const indexList = ref<string[]>([])
const errorPage404 = ref('')
const newItem = ref('')
const loading = ref(false)
const saving = ref(false)

function parseList(raw: string) {
  return raw.split(/[\s,]+/).map(x => x.trim()).filter(Boolean)
}

async function load() {
  loading.value = true
  try {
    conf.value = await siteConfApi.getDefaults(props.site.id)
    indexList.value = parseList(conf.value.indexFiles)
    errorPage404.value = conf.value.errorPage404
  }
  catch (e: any) {
    toast.error('读取默认文档失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

function addItem() {
  const v = newItem.value.trim().replace(/[\\/]/g, '')
  if (!v)
    return
  if (indexList.value.includes(v)) {
    toast.warning('已存在同名默认文档')
    return
  }
  indexList.value.push(v)
  newItem.value = ''
}

function removeItem(i: number) {
  indexList.value.splice(i, 1)
}

function move(i: number, dir: -1 | 1) {
  const j = i + dir
  if (j < 0 || j >= indexList.value.length)
    return
  const arr = [...indexList.value]
  ;[arr[i], arr[j]] = [arr[j], arr[i]]
  indexList.value = arr
}

const dirty = computed(() => {
  if (!conf.value)
    return false
  return indexList.value.join(' ') !== parseList(conf.value.indexFiles).join(' ') || errorPage404.value !== conf.value.errorPage404
})

async function save() {
  if (!indexList.value.length) {
    toast.warning('至少保留一个默认文档')
    return
  }
  saving.value = true
  try {
    conf.value = await siteConfApi.updateDefaults(props.site.id, {
      indexFiles: indexList.value.join(' '),
      errorPage404: errorPage404.value,
    } as SiteDefaultsConf)
    indexList.value = parseList(conf.value.indexFiles)
    errorPage404.value = conf.value.errorPage404
    toast.success('默认文档已保存并重载 nginx')
    emit('changed')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">默认文档（index 检查顺序）</div>
      <p class="mb-3 text-xs text-muted-foreground">
        按从上到下的顺序匹配，仅允许文件名（不含路径）
      </p>
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <span
          v-for="(f, i) in indexList"
          :key="f"
          class="inline-flex items-center gap-1 rounded-full bg-muted px-2.5 py-1 text-xs"
        >
          <span class="text-muted-foreground">{{ i + 1 }}.</span>
          <span class="font-mono">{{ f }}</span>
          <button type="button" class="ml-0.5 cursor-pointer text-muted-foreground transition-colors hover:text-red-500" title="删除" @click="removeItem(i)">
            <FaIcon name="i-lucide:x" class="text-[11px]" />
          </button>
          <button type="button" class="cursor-pointer text-muted-foreground transition-colors hover:text-primary" :disabled="i === 0" title="上移（提高优先级）" @click="move(i, -1)">
            <FaIcon name="i-lucide:arrow-up" class="text-[11px]" />
          </button>
          <button type="button" class="cursor-pointer text-muted-foreground transition-colors hover:text-primary" :disabled="i === indexList.length - 1" title="下移" @click="move(i, 1)">
            <FaIcon name="i-lucide:arrow-down" class="text-[11px]" />
          </button>
        </span>
        <span v-if="!indexList.length" class="text-xs text-muted-foreground">至少添加一个默认文档</span>
      </div>
      <div class="flex items-center gap-2">
        <FaInput v-model="newItem" placeholder="如 index.php，回车添加" class="w-64!" @keyup.enter="addItem" />
        <FaButton variant="outline" size="sm" @click="addItem">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 添加
        </FaButton>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">自定义 404 页面</div>
      <p class="mb-3 text-xs text-muted-foreground">
        相对站点根目录，如 /404.html；留空使用 nginx 默认
      </p>
      <FaInput v-model="errorPage404" placeholder="/404.html" class="w-full" />
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        保存并生效
      </FaButton>
    </div>
  </div>
</template>
