<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteAntiLeech } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteAntiLeech | null>(null)
const edit = ref<SiteAntiLeech>({ enable: false, validReferers: [], allowNone: true, allowBlocked: true, returnCode: 403 })
const referersRaw = ref('')
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteExtraApi.getAntiLeech(props.site.id)
    edit.value = { ...conf.value }
    referersRaw.value = (conf.value.validReferers || []).join('\n')
  }
  catch (e: any) {
    toast.error('读取防盗链配置失败', { description: e?.message })
  }
}

async function save() {
  saving.value = true
  try {
    edit.value.validReferers = referersRaw.value.split('\n').map(x => x.trim()).filter(Boolean)
    conf.value = await siteExtraApi.updateAntiLeech(props.site.id, edit.value)
    edit.value = { ...conf.value }
    toast.success('防盗链已保存并重载 nginx')
    emit('changed')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

const dirty = computed(() => conf.value
  ? edit.value.enable !== conf.value.enable
    || edit.value.returnCode !== conf.value.returnCode
    || edit.value.allowNone !== conf.value.allowNone
    || edit.value.allowBlocked !== conf.value.allowBlocked
    || JSON.stringify(edit.value.validReferers) !== JSON.stringify(conf.value.validReferers || [])
  : false)

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-sm font-medium">启用防盗链</div>
          <p class="mt-0.5 text-xs text-muted-foreground">校验 Referer，白名单外来源访问静态资源将被拦截</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.enable" type="checkbox"> 启用
        </label>
      </div>
    </div>

    <div class="space-y-4 rounded-lg border p-4">
      <div>
        <div class="mb-1 text-sm font-medium">Referer 白名单（每行一个）</div>
        <p class="mb-2 text-xs text-muted-foreground">域名或通配如 *.example.com；server_names（本站域名）始终放行</p>
        <textarea v-model="referersRaw" rows="4" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="*.example.com&#10;friend-site.com" />
      </div>
      <div class="flex flex-wrap gap-5 text-sm">
        <label class="flex cursor-pointer items-center gap-2">
          <input v-model="edit.allowNone" type="checkbox"> 允许空 Referer（直接打开）
        </label>
        <label class="flex cursor-pointer items-center gap-2">
          <input v-model="edit.allowBlocked" type="checkbox"> 允许被遮蔽的 Referer
        </label>
        <label class="flex items-center gap-2">
          拦截返回码
          <select v-model.number="edit.returnCode" class="h-8 rounded-md border bg-background px-2 text-sm outline-none">
            <option :value="403">403</option>
            <option :value="404">404</option>
          </select>
        </label>
      </div>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        保存并生效
      </FaButton>
    </div>
  </div>
</template>
