<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteProxyConf } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const conf = ref<SiteProxyConf | null>(null)
const edit = ref<SiteProxyConf>({ rules: [], cacheEnable: false, cacheDuration: '12h' })
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    conf.value = await siteConfApi.getProxy(props.site.id)
    edit.value = JSON.parse(JSON.stringify(conf.value))
  }
  catch (e: any) {
    toast.error('读取反代配置失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

function addRule() {
  edit.value.rules.push({ prefix: '/', target: '', ws: true })
}

function removeRule(i: number) {
  edit.value.rules.splice(i, 1)
}

async function save() {
  if (edit.value.rules.some(r => !r.target.trim())) {
    toast.error('存在未填写转发目标的规则')
    return
  }
  saving.value = true
  try {
    conf.value = await siteConfApi.updateProxy(props.site.id, edit.value)
    edit.value = JSON.parse(JSON.stringify(conf.value))
    toast.success('反代规则已保存并重载 nginx')
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
      <div class="mb-3 flex items-center justify-between">
        <span class="text-sm font-medium">转发规则</span>
        <FaButton variant="outline" size="sm" @click="addRule">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 加规则
        </FaButton>
      </div>
      <div class="mb-2 hidden gap-2 text-xs text-muted-foreground md:flex">
        <span class="w-32">路径前缀</span>
        <span class="flex-1">转发目标（http(s)://）</span>
        <span class="w-14 text-center">WS</span>
        <span class="w-8" />
      </div>
      <div v-for="(r, i) in edit.rules" :key="i" class="mb-2 flex items-center gap-2">
        <FaInput v-model="r.prefix" placeholder="/" class="w-32" />
        <span class="shrink-0 text-muted-foreground">→</span>
        <FaInput v-model="r.target" placeholder="http://172.17.0.1:3000" class="flex-1" />
        <label class="flex w-14 shrink-0 cursor-pointer items-center justify-center gap-1 text-xs">
          <input v-model="r.ws" type="checkbox"> WS
        </label>
        <FaButton variant="ghost" size="icon-sm" class="text-red-500!" :disabled="edit.rules.length <= 1" title="删除" @click="removeRule(i)">
          <FaIcon name="i-lucide:trash-2" class="text-sm" />
        </FaButton>
      </div>
      <div class="text-xs text-muted-foreground">
        规则按顺序匹配；全部规则自动启用 WebSocket 支持（Upgrade/Connection 头）
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-3 flex items-center justify-between">
        <div>
          <div class="text-sm font-medium">反代缓存（proxy_cache）</div>
          <p class="mt-0.5 text-xs text-muted-foreground">缓存上游响应，静态化动态内容加速回源</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.cacheEnable" type="checkbox"> 启用
        </label>
      </div>
      <div v-if="edit.cacheEnable" class="flex items-center gap-3">
        <span class="w-24 shrink-0 text-sm text-muted-foreground">缓存有效期</span>
        <FaInput v-model="edit.cacheDuration" placeholder="12h" class="w-40" />
      </div>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        保存并生效
      </FaButton>
    </div>
  </div>
</template>
