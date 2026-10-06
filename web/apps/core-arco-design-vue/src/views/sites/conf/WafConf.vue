<script setup lang="ts">
import type { SiteItem, SiteWaf } from '@/api/modules/site'
import { wafApi } from '@/api/modules/site'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const form = ref<SiteWaf>({ denyIps: [], allowIps: [], denyUAs: [], rateEnable: false, rate: 10, burst: 20 })
const denyIps = ref('')
const allowIps = ref('')
const denyUAs = ref('')
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const w = await wafApi.get(props.site.id)
    form.value = { ...w, denyIps: w.denyIps || [], allowIps: w.allowIps || [], denyUAs: w.denyUAs || [] }
    denyIps.value = (w.denyIps || []).join(', ')
    allowIps.value = (w.allowIps || []).join(', ')
    denyUAs.value = (w.denyUAs || []).join(', ')
  }
  catch (e: any) {
    toast.error('读取 WAF 配置失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const w: SiteWaf = {
      denyIps: denyIps.value.split(/[,\n]/).map(x => x.trim()).filter(Boolean),
      allowIps: allowIps.value.split(/[,\n]/).map(x => x.trim()).filter(Boolean),
      denyUAs: denyUAs.value.split(/[,\n]/).map(x => x.trim()).filter(Boolean),
      rateEnable: form.value.rateEnable,
      rate: Number(form.value.rate) || 10,
      burst: Number(form.value.burst) || 20,
    }
    await wafApi.update(props.site.id, w)
    toast.success('WAF 规则已保存并重载 nginx')
    emit('changed')
    await load()
  }
  catch (e: any) {
    toast.error('保存失败（已回滚）', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="space-y-4 rounded-lg border p-4">
      <div class="text-sm font-medium">IP 黑名单（拒绝访问）</div>
      <textarea v-model="denyIps" rows="2" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="1.2.3.4, 5.6.7.0/24" />
      <div class="text-sm font-medium">IP 白名单（永远放行）</div>
      <textarea v-model="allowIps" rows="2" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="192.168.1.0/24" />
      <div class="text-sm font-medium">UA 拦截（每行一条，正则转义后匹配）</div>
      <textarea v-model="denyUAs" rows="2" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="curl&#10;sqlmap" />
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-3 flex items-center justify-between">
        <div>
          <div class="text-sm font-medium">请求频率限制（limit_req）</div>
          <p class="mt-0.5 text-xs text-muted-foreground">超限请求返回 503；连接数限制在「连接限制」子页</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="form.rateEnable" type="checkbox"> 启用
        </label>
      </div>
      <div v-if="form.rateEnable" class="flex flex-wrap items-center gap-4 text-sm">
        <label class="flex items-center gap-2">
          阈值（req/s）
          <FaInput v-model="form.rate" type="number" class="w-24" />
        </label>
        <label class="flex items-center gap-2">
          突发容量
          <FaInput v-model="form.burst" type="number" class="w-24" />
        </label>
      </div>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" @click="save">
        保存并生效
      </FaButton>
    </div>
  </div>
</template>
