<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteDefaultsConf } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const conf = ref<SiteDefaultsConf | null>(null)
const edit = ref<SiteDefaultsConf>({ indexFiles: '', errorPage404: '' })
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    conf.value = await siteConfApi.getDefaults(props.site.id)
    edit.value = { ...conf.value }
  }
  catch (e: any) {
    toast.error('读取默认文档失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteConfApi.updateDefaults(props.site.id, edit.value)
    edit.value = { ...conf.value }
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

const dirty = computed(() => conf.value ? JSON.stringify(edit.value) !== JSON.stringify(conf.value) : false)

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">默认文档（index 检查顺序）</div>
      <p class="mb-3 text-xs text-muted-foreground">
        逗号分隔，仅文件名不含路径，如 index.html,index.php
      </p>
      <FaInput v-model="edit.indexFiles" placeholder="index.html" class="w-full" />
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">自定义 404 页面</div>
      <p class="mb-3 text-xs text-muted-foreground">
        相对站点根目录，如 /404.html；留空使用 nginx 默认
      </p>
      <FaInput v-model="edit.errorPage404" placeholder="/404.html" class="w-full" />
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        保存并生效
      </FaButton>
    </div>
  </div>
</template>
