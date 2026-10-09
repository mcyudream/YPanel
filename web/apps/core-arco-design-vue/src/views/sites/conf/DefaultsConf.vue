<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteDefaultsConf } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

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
    toast.error(i18n.global.t('sites.conf.defaults.loadFailed'), { description: e?.message })
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
    toast.warning(i18n.global.t('sites.conf.defaults.dup'))
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
    toast.warning(i18n.global.t('sites.conf.defaults.needOne'))
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
    toast.success(i18n.global.t('sites.conf.defaults.saved'))
    emit('changed')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.shared.saveFailed'), { description: e?.message })
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
      <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.defaults.title') }}</div>
      <p class="mb-3 text-xs text-muted-foreground">
        {{ $t('sites.conf.defaults.desc') }}
      </p>
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <span
          v-for="(f, i) in indexList"
          :key="f"
          class="inline-flex items-center gap-1 rounded-full bg-muted px-2.5 py-1 text-xs"
        >
          <span class="text-muted-foreground">{{ i + 1 }}.</span>
          <span class="font-mono">{{ f }}</span>
          <button type="button" class="ml-0.5 cursor-pointer text-muted-foreground transition-colors hover:text-red-500" :title="$t('common.delete')" @click="removeItem(i)">
            <FaIcon name="i-lucide:x" class="text-[11px]" />
          </button>
          <button type="button" class="cursor-pointer text-muted-foreground transition-colors hover:text-primary" :disabled="i === 0" :title="$t('sites.conf.defaults.moveUp')" @click="move(i, -1)">
            <FaIcon name="i-lucide:arrow-up" class="text-[11px]" />
          </button>
          <button type="button" class="cursor-pointer text-muted-foreground transition-colors hover:text-primary" :disabled="i === indexList.length - 1" :title="$t('sites.conf.defaults.moveDown')" @click="move(i, 1)">
            <FaIcon name="i-lucide:arrow-down" class="text-[11px]" />
          </button>
        </span>
        <span v-if="!indexList.length" class="text-xs text-muted-foreground">{{ $t('sites.conf.defaults.needAdd') }}</span>
      </div>
      <div class="flex items-center gap-2">
        <FaInput v-model="newItem" :placeholder="$t('sites.conf.defaults.placeholder')" class="w-64!" @keyup.enter="addItem" />
        <FaButton variant="outline" size="sm" @click="addItem">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('common.add') }}
        </FaButton>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.defaults.page404') }}</div>
      <p class="mb-3 text-xs text-muted-foreground">
        {{ $t('sites.conf.defaults.page404Desc') }}
      </p>
      <FaInput v-model="errorPage404" placeholder="/404.html" class="w-full" />
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
