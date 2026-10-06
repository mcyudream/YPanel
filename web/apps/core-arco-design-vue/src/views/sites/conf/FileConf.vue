<script setup lang="ts">
import type * as Monaco from 'monaco-editor'
import type { SiteItem } from '@/api/modules/site'
import apiSite from '@/api/modules/site'
import YdCodeEditor from '@/components/YdCodeEditor/index.vue'
import { loadMonaco } from '@/utils/monacoLoader'

// B23：配置文件编辑器升级为 Monaco（复用系统文件编辑器封装），
// 保存链路不变（nginx -t 校验失败自动回滚）。

const props = defineProps<{ site: SiteItem }>()

const toast = useFaToast()

const model = shallowRef<Monaco.editor.ITextModel | null>(null)
const loading = ref(false)
const saving = ref(false)
let monaco: typeof Monaco | null = null

async function load() {
  loading.value = true
  try {
    const [m, content] = await Promise.all([
      loadMonaco(),
      apiSite.config(props.site.id),
    ])
    monaco = m
    const uri = m.Uri.parse(`ypanel:///site-${props.site.id}/nginx.conf`)
    const existing = m.editor.getModel(uri)
    model.value = existing ?? m.editor.createModel(content, 'ini', uri)
    if (existing)
      existing.setValue(content)
  }
  catch (e: any) {
    toast.error('读取配置失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function save() {
  if (!model.value)
    return
  saving.value = true
  try {
    await apiSite.updateConfig(props.site.id, model.value.getValue())
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

onBeforeUnmount(() => {
  // 仅释放本站点模型（uri 域内自管，不影响文件编辑器 store 的模型）
  if (monaco && model.value)
    monaco.editor.getModel(model.value.uri)?.dispose()
})
</script>

<template>
  <div class="space-y-4">
    <div class="rounded-lg border border-amber-300 bg-amber-50 p-3 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
      直接编辑 nginx 站点配置（conf.d/{{ site.name }}.conf）。保存时自动 nginx -t 校验，不通过会自动回滚；一般配置请优先使用左侧各子页。
    </div>
    <div class="h-[32rem] overflow-hidden rounded-md border">
      <div v-if="loading" class="flex h-full items-center justify-center text-sm text-muted-foreground">
        加载编辑器…
      </div>
      <YdCodeEditor v-else :model="model" :font-size="13" />
    </div>
    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="loading" @click="save">
        保存并重载
      </FaButton>
    </div>
  </div>
</template>
