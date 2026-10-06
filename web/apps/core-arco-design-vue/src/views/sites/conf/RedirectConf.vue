<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteRedirect } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteRedirect | null>(null)
const edit = ref<SiteRedirect>({ enable: false, target: '', code: 301 })
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteExtraApi.getRedirect(props.site.id)
    edit.value = { ...conf.value }
  }
  catch (e: any) {
    toast.error('读取重定向配置失败', { description: e?.message })
  }
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteExtraApi.updateRedirect(props.site.id, edit.value)
    edit.value = { ...conf.value }
    toast.success('重定向已保存并重载 nginx')
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
      <div class="flex items-center justify-between">
        <div>
          <div class="text-sm font-medium">启用整站重定向</div>
          <p class="mt-0.5 text-xs text-muted-foreground">
            {{ site.certDomain ? '注意：当前站点已启用 HTTPS，80 端口已固定 301 跳转 HTTPS，重定向域不生效' : '启用后所有请求按状态码跳转到目标地址' }}
          </p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm" :class="site.certDomain ? 'pointer-events-none opacity-50' : ''">
          <input v-model="edit.enable" type="checkbox"> 启用
        </label>
      </div>
    </div>

    <div class="space-y-4 rounded-lg border p-4">
      <div>
        <div class="mb-1 text-sm font-medium">目标地址</div>
        <FaInput v-model="edit.target" placeholder="https://new-domain.com" class="w-full" />
      </div>
      <div>
        <div class="mb-1 text-sm font-medium">重定向状态码</div>
        <select v-model.number="edit.code" class="h-9 w-full rounded-md border bg-background px-2 text-sm outline-none md:w-64">
          <option :value="301">301 永久（SEO 传递权重）</option>
          <option :value="302">302 临时</option>
          <option :value="307">307 临时（保持方法）</option>
          <option :value="308">308 永久（保持方法）</option>
        </select>
      </div>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        保存并生效
      </FaButton>
    </div>
  </div>
</template>
