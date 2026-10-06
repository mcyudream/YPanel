<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import apiSite from '@/api/modules/site'

const props = defineProps<{ site: SiteItem }>()

const toast = useFaToast()

const content = ref('')
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    content.value = await apiSite.config(props.site.id)
  }
  catch (e: any) {
    toast.error('读取配置失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await apiSite.updateConfig(props.site.id, content.value)
    toast.success('配置已保存并重载')
  }
  catch (e: any) {
    toast.error('保存失败（配置校验不通过会自动回滚）', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <div class="rounded-lg border border-amber-300 bg-amber-50 p-3 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
      直接编辑 nginx 站点配置（conf.d/{{ site.name }}.conf）。保存时自动 nginx -t 校验，不通过会自动回滚；一般配置请优先使用左侧各子页。
    </div>
    <textarea
      v-model="content"
      class="h-[28rem] w-full resize-y rounded-md border bg-background p-3 font-mono text-[13px] leading-relaxed outline-none focus:ring-1 focus:ring-primary"
      spellcheck="false"
    />
    <div class="flex justify-end">
      <FaButton :loading="saving" @click="save">
        保存并重载
      </FaButton>
    </div>
  </div>
</template>
