<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteLoadBalance } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteLoadBalance | null>(null)
const edit = ref<SiteLoadBalance>({ enable: false, strategy: 'round-robin', upstreams: [] })
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteExtraApi.getLoadBalance(props.site.id)
    edit.value = JSON.parse(JSON.stringify(conf.value))
  }
  catch (e: any) {
    toast.error('读取负载均衡配置失败', { description: e?.message })
  }
}

function addUpstream() {
  edit.value.upstreams.push({ address: '', weight: 1 })
}

function removeUpstream(i: number) {
  edit.value.upstreams.splice(i, 1)
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteExtraApi.updateLoadBalance(props.site.id, edit.value)
    edit.value = JSON.parse(JSON.stringify(conf.value))
    toast.success('负载均衡已保存并重载 nginx')
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

onMounted(() => {
  load()
  if (!edit.value.upstreams.length) {
    addUpstream()
    addUpstream()
  }
})
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-sm font-medium">启用负载均衡</div>
          <p class="mt-0.5 text-xs text-muted-foreground">启用后所有转发规则的流量按策略分发到上游组（自动生成 upstream 块）</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.enable" type="checkbox"> 启用
        </label>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">均衡策略</div>
      <select v-model="edit.strategy" class="mb-4 h-9 rounded-md border bg-background px-2 text-sm outline-none md:w-64">
        <option value="round-robin">轮询（round-robin）</option>
        <option value="least_conn">最少连接（least_conn）</option>
        <option value="ip_hash">IP 哈希（会话保持）</option>
      </select>

      <div class="mb-3 flex items-center justify-between">
        <span class="text-sm font-medium">上游节点</span>
        <FaButton variant="outline" size="sm" @click="addUpstream">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 加节点
        </FaButton>
      </div>
      <div class="mb-2 hidden gap-2 text-xs text-muted-foreground md:flex">
        <span class="flex-1">地址（host:port）</span>
        <span class="w-24">权重</span>
        <span class="w-8" />
      </div>
      <div v-for="(u, i) in edit.upstreams" :key="i" class="mb-2 flex items-center gap-2">
        <FaInput v-model="u.address" placeholder="172.17.0.1:3000" class="flex-1" />
        <FaInput v-model="u.weight" type="number" placeholder="1" class="w-24" />
        <FaButton variant="ghost" size="icon-sm" class="text-red-500!" title="删除" @click="removeUpstream(i)">
          <FaIcon name="i-lucide:trash-2" class="text-sm" />
        </FaButton>
      </div>
      <p class="mt-2 text-xs text-muted-foreground">
        权重 0 或留空 = 默认权重；至少两个节点才能启用
      </p>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        保存并生效
      </FaButton>
    </div>
  </div>
</template>
