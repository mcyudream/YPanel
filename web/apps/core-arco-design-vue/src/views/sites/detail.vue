<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import apiSite from '@/api/modules/site'
import DomainConf from './conf/DomainConf.vue'
import DefaultsConf from './conf/DefaultsConf.vue'
import ProxyConf from './conf/ProxyConf.vue'
import RewriteConf from './conf/RewriteConf.vue'
import HTTPSConf from './conf/HTTPSConf.vue'

defineOptions({
  name: 'SitesDetail',
})

const route = useRoute()
const router = useRouter()
const toast = useFaToast()

const siteId = computed(() => Number(route.params.id) || 0)
const site = ref<SiteItem | null>(null)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const list = await apiSite.list()
    site.value = list.find(s => s.id === siteId.value) ?? null
  }
  catch (e: any) {
    toast.error('加载站点失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

const typeLabel: Record<string, string> = { static: '静态', proxy: '反代', php: 'PHP' }

const tabs = computed(() => {
  const base = [
    { key: 'domain', label: '域名', icon: 'i-lucide:globe', comp: DomainConf },
    { key: 'defaults', label: '默认文档', icon: 'i-lucide:file-text', comp: DefaultsConf },
    { key: 'rewrite', label: '伪静态', icon: 'i-lucide:repeat', comp: RewriteConf },
  ]
  if (site.value?.type === 'proxy') {
    base.splice(2, 0, { key: 'proxy', label: '反向代理', icon: 'i-lucide:arrow-left-right', comp: ProxyConf })
  }
  base.push({ key: 'https', label: 'HTTPS', icon: 'i-lucide:lock', comp: HTTPSConf })
  return base
})
const activeTab = ref('domain')
const activeComp = computed(() => tabs.value.find(t => t.key === activeTab.value)?.comp)

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="globe" :size="24" />
          <span>站点配置：{{ site?.name || '…' }}</span>
          <span
            v-if="site"
            class="rounded-full px-2 py-0.5 text-xs"
            :class="{ static: 'bg-blue-500/10 text-blue-600', proxy: 'bg-purple-500/10 text-purple-600', php: 'bg-emerald-500/10 text-emerald-600' }[site.type] || 'bg-muted text-muted-foreground'"
          >
            {{ typeLabel[site.type] || site.type }}
          </span>
        </div>
      </template>
      <template #description>
        <span v-if="site">{{ site.domain }}<span v-if="site.enabled" class="ml-2 text-emerald-600">· 运行中</span><span v-else class="ml-2 text-muted-foreground">· 已停用</span></span>
        <span v-else>加载中…</span>
      </template>
      <FaButton variant="outline" size="sm" @click="router.push('/sites')">
        <FaIcon name="i-lucide:arrow-left" class="mr-1" /> 返回列表
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="loading && !site" class="py-16 text-center text-sm text-muted-foreground">
        加载中…
      </div>
      <div v-else-if="!site" class="py-16 text-center text-sm text-muted-foreground">
        站点不存在或已删除
      </div>
      <div v-else class="flex flex-col gap-4 md:flex-row">
        <!-- 左侧 tab 导航（1P 风格） -->
        <nav class="flex shrink-0 gap-1 overflow-x-auto md:w-44 md:flex-col">
          <button
            v-for="t in tabs"
            :key="t.key"
            type="button"
            class="flex shrink-0 cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors md:w-full"
            :class="activeTab === t.key ? 'bg-primary/10 font-medium text-foreground' : 'text-muted-foreground hover:bg-accent/50'"
            @click="activeTab = t.key"
          >
            <FaIcon :name="t.icon" class="text-base" />
            {{ t.label }}
          </button>
        </nav>
        <!-- 子页 -->
        <section class="min-w-0 flex-1">
          <component :is="activeComp" :key="activeTab" :site="site" @changed="load()" />
        </section>
      </div>
    </FaPageMain>
  </div>
</template>
