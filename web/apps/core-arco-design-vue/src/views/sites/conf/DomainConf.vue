<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteDomainConf } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const conf = ref<SiteDomainConf | null>(null)
const loading = ref(false)
const saving = ref(false)
// 编辑副本：每行一个域名
const rows = ref<string[]>([])
const newDomain = ref('')

async function load() {
  loading.value = true
  try {
    conf.value = await siteConfApi.getDomain(props.site.id)
    rows.value = [...(conf.value.domains || [])]
  }
  catch (e: any) {
    toast.error('读取域名配置失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

function add() {
  const d = newDomain.value.trim().toLowerCase()
  if (!d) {
    return
  }
  if (d === props.site.domain) {
    toast.error('不能与主域名重复')
    return
  }
  if (rows.value.includes(d)) {
    toast.error('该域名已存在')
    return
  }
  rows.value.push(d)
  newDomain.value = ''
}

function removeAt(i: number) {
  rows.value.splice(i, 1)
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteConfApi.updateDomain(props.site.id, rows.value)
    rows.value = [...(conf.value.domains || [])]
    toast.success('域名已保存并重载 nginx')
    emit('changed')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

const dirty = computed(() => conf.value ? JSON.stringify(rows.value) !== JSON.stringify(conf.value.domains || []) : false)

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="mb-2 text-sm font-medium">主域名</div>
      <div class="flex items-center gap-2">
        <code class="rounded bg-muted px-2 py-1 font-mono text-[13px]">{{ site.domain }}</code>
        <span class="text-xs text-muted-foreground">主域名创建后不可修改；证书域名：{{ conf?.certDomain || '未配置' }}</span>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-2 flex items-center justify-between">
        <span class="text-sm font-medium">附加域名</span>
        <span class="text-xs text-muted-foreground">多个域名指向同一站点，需自行将 DNS 解析到服务器</span>
      </div>
      <div class="mb-3 flex items-center gap-2">
        <FaInput v-model="newDomain" placeholder="如 www.example.com" class="flex-1" @keyup.enter="add" />
        <FaButton variant="outline" size="sm" @click="add">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 添加
        </FaButton>
      </div>
      <div v-if="!rows.length" class="py-4 text-center text-sm text-muted-foreground">
        暂无附加域名
      </div>
      <div v-for="(d, i) in rows" :key="d" class="flex items-center justify-between border-b py-2 text-sm last:border-b-0">
        <code class="font-mono text-[13px]">{{ d }}</code>
        <FaButton variant="ghost" size="icon-sm" class="text-red-500!" title="移除" @click="removeAt(i)">
          <FaIcon name="i-lucide:x" class="text-sm" />
        </FaButton>
      </div>
      <div class="mt-4 flex justify-end">
        <FaButton :loading="saving" :disabled="!dirty" @click="save">
          保存并生效
        </FaButton>
      </div>
    </div>
  </div>
</template>
