<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteCORS } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteCORS | null>(null)
const edit = ref<SiteCORS>({ enable: false, allowOrigins: [], allowMethods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'], allowHeaders: ['Content-Type', 'Authorization'], allowCredentials: false, maxAge: 86400 })
const originsRaw = ref('')
const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS']
const saving = ref(false)

const allowHeadersStr = computed({
  get: () => (edit.value.allowHeaders || []).join(', '),
  set: (v: string) => {
    edit.value.allowHeaders = v.split(',').map(x => x.trim()).filter(Boolean)
  },
})

async function load() {
  try {
    conf.value = await siteExtraApi.getCORS(props.site.id)
    edit.value = { ...conf.value }
    originsRaw.value = (conf.value.allowOrigins || []).join('\n')
  }
  catch (e: any) {
    toast.error('读取 CORS 配置失败', { description: e?.message })
  }
}

async function save() {
  saving.value = true
  try {
    edit.value.allowOrigins = originsRaw.value.split('\n').map(x => x.trim()).filter(Boolean)
    conf.value = await siteExtraApi.updateCORS(props.site.id, edit.value)
    edit.value = { ...conf.value }
    originsRaw.value = (conf.value.allowOrigins || []).join('\n')
    toast.success('CORS 已保存并重载 nginx')
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
          <div class="text-sm font-medium">启用跨域（CORS）</div>
          <p class="mt-0.5 text-xs text-muted-foreground">为 API/静态资源响应添加 Access-Control-* 头，预检请求返回 204</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.enable" type="checkbox"> 启用
        </label>
      </div>
    </div>

    <div class="space-y-4 rounded-lg border p-4">
      <div>
        <div class="mb-1 text-sm font-medium">允许来源（每行一个）</div>
        <p class="mb-2 text-xs text-muted-foreground">* 或 https://app.example.com；多来源时按请求头动态回显</p>
        <textarea v-model="originsRaw" rows="3" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="*&#10;https://app.example.com" />
      </div>
      <div>
        <div class="mb-1 text-sm font-medium">允许方法</div>
        <div class="flex flex-wrap gap-3 text-sm">
          <label v-for="m in methods" :key="m" class="flex cursor-pointer items-center gap-1.5">
            <input v-model="edit.allowMethods" type="checkbox" :value="m"> {{ m }}
          </label>
        </div>
      </div>
      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <div class="mb-1 text-sm font-medium">允许请求头</div>
          <FaInput v-model="allowHeadersStr" placeholder="Content-Type, Authorization" class="w-full" />
        </div>
        <div>
          <div class="mb-1 text-sm font-medium">预检缓存（秒）</div>
          <FaInput v-model="edit.maxAge" type="number" class="w-full" />
        </div>
      </div>
      <label class="flex cursor-pointer items-center gap-2 text-sm">
        <input v-model="edit.allowCredentials" type="checkbox"> 允许携带凭据（Allow-Credentials）
      </label>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        保存并生效
      </FaButton>
    </div>
  </div>
</template>
