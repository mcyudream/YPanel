<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteLimitConn } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteLimitConn | null>(null)
const edit = ref<SiteLimitConn>({ enable: false, connPerIP: 20 })
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteExtraApi.getLimitConn(props.site.id)
    edit.value = { ...conf.value }
  }
  catch (e: any) {
    toast.error('读取连接限制失败', { description: e?.message })
  }
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteExtraApi.updateLimitConn(props.site.id, edit.value)
    edit.value = { ...conf.value }
    toast.success('连接限制已保存并重载 nginx')
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
          <div class="text-sm font-medium">启用连接数限制</div>
          <p class="mt-0.5 text-xs text-muted-foreground">限制单 IP 并发连接数，抑制下载占用与简单 CC；请求频率限制在 WAF 中配置</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.enable" type="checkbox"> 启用
        </label>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">单 IP 最大并发连接</div>
      <FaInput v-model="edit.connPerIP" type="number" placeholder="20" class="w-40" />
      <p class="mt-2 text-xs text-muted-foreground">超限请求返回 503；浏览器正常浏览通常同时 6-10 个连接</p>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        保存并生效
      </FaButton>
    </div>
  </div>
</template>
