<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import apiSite from '@/api/modules/site'
import DomainConf from './conf/DomainConf.vue'
import PortConf from './conf/PortConf.vue'
import DirConf from './conf/DirConf.vue'
import DefaultsConf from './conf/DefaultsConf.vue'
import ProxyConf from './conf/ProxyConf.vue'
import RewriteConf from './conf/RewriteConf.vue'
import HTTPSConf from './conf/HTTPSConf.vue'
import AntiLeechConf from './conf/AntiLeechConf.vue'
import AuthBasicConf from './conf/AuthBasicConf.vue'
import CorsConf from './conf/CorsConf.vue'
import RedirectConf from './conf/RedirectConf.vue'
import RealIPConf from './conf/RealIPConf.vue'
import LimitConnConf from './conf/LimitConnConf.vue'
import LoadBalanceConf from './conf/LoadBalanceConf.vue'
import OtherConf from './conf/OtherConf.vue'
import LogsConf from './conf/LogsConf.vue'
import WafConf from './conf/WafConf.vue'
import FileConf from './conf/FileConf.vue'
import { i18n } from '@/locales'
import { closestWindowId, useYwEmbed } from '@/views/desktop/embed'

defineOptions({
  name: 'SitesDetail',
})

// 桌面工作台承载时经 props 传入（launchOptions），经典模式走路由参数
const props = defineProps<{
  /** 站点 ID（webos 窗口承载时由 launchOptions 注入，优先于路由参数） */
  id?: number
}>()

const route = useRoute()
const router = useRouter()
const toast = useFaToast()

// 桌面承载：返回=关自己窗（列表窗仍在）；经典模式保持路由。
// 根元素 ref 定位自身窗 id（getCurrentInstance 在点击期求值拿不到实例，见 docs/exp/frontend.md）
const ywEmbed = useYwEmbed()
const rootRef = ref<HTMLElement | null>(null)

function onBack() {
  if (ywEmbed) {
    const winId = closestWindowId(rootRef.value)
    if (winId) {
      ywEmbed.closeWindow(winId)
      return
    }
  }
  router.push('/sites')
}

const siteId = computed(() => props.id ?? (Number(route.params.id) || 0))
const site = ref<SiteItem | null>(null)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    // F8：单条端点，免拉全量列表
    site.value = await apiSite.getOne(siteId.value)
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.detail.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

const typeLabel = computed<Record<string, string>>(() => ({
  static: i18n.global.t('sites.type.static'),
  proxy: i18n.global.t('sites.type.proxy'),
  php: i18n.global.t('sites.type.php'),
}))

const tabs = computed(() => {
  const t = (key: string) => i18n.global.t(`sites.detail.tabs.${key}`)
  const base = [
    { key: 'domain', label: t('domain'), icon: 'i-lucide:globe', comp: DomainConf },
    { key: 'port', label: t('port'), icon: 'i-lucide:plug', comp: PortConf },
    { key: 'dir', label: t('dir'), icon: 'i-lucide:folder', comp: DirConf },
    { key: 'defaults', label: t('defaults'), icon: 'i-lucide:file-text', comp: DefaultsConf },
    { key: 'rewrite', label: t('rewrite'), icon: 'i-lucide:repeat', comp: RewriteConf },
    { key: 'redirect', label: t('redirect'), icon: 'i-lucide:corner-up-right', comp: RedirectConf },
    { key: 'https', label: t('https'), icon: 'i-lucide:lock', comp: HTTPSConf },
    { key: 'antileech', label: t('antileech'), icon: 'i-lucide:shield-off', comp: AntiLeechConf },
    { key: 'authbasic', label: t('authbasic'), icon: 'i-lucide:key-round', comp: AuthBasicConf },
    { key: 'cors', label: t('cors'), icon: 'i-lucide:shuffle', comp: CorsConf },
    { key: 'realip', label: t('realip'), icon: 'i-lucide:locate', comp: RealIPConf },
    { key: 'limitconn', label: t('limitconn'), icon: 'i-lucide:gauge', comp: LimitConnConf },
    { key: 'logs', label: t('logs'), icon: 'i-lucide:scroll-text', comp: LogsConf },
    { key: 'waf', label: t('waf'), icon: 'i-lucide:shield-alert', comp: WafConf },
    { key: 'file', label: t('file'), icon: 'i-lucide:file-code', comp: FileConf },
    { key: 'other', label: t('other'), icon: 'i-lucide:code', comp: OtherConf },
  ]
  if (site.value?.type === 'proxy') {
    base.splice(3, 0, { key: 'proxy', label: t('proxy'), icon: 'i-lucide:arrow-left-right', comp: ProxyConf })
    base.splice(4, 0, { key: 'loadbalance', label: t('loadbalance'), icon: 'i-lucide:network', comp: LoadBalanceConf })
  }
  return base
})
const activeTab = ref('domain')
const activeComp = computed(() => tabs.value.find(t => t.key === activeTab.value)?.comp)

onMounted(load)
</script>

<template>
  <div ref="rootRef">
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="globe" :size="24" />
          <span>{{ $t('sites.detail.title', { name: site?.name || '…' }) }}</span>
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
        <span v-if="site">{{ site.domain }}<span v-if="site.enabled" class="ml-2 text-emerald-600">· {{ $t('sites.detail.running') }}</span><span v-else class="ml-2 text-muted-foreground">· {{ $t('sites.detail.stopped') }}</span></span>
        <span v-else>{{ $t('common.loading') }}</span>
      </template>
      <FaButton variant="outline" size="sm" @click="onBack">
        <FaIcon name="i-lucide:arrow-left" class="mr-1" /> {{ $t('sites.detail.back') }}
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="loading && !site" class="py-16 text-center text-sm text-muted-foreground">
        {{ $t('common.loading') }}
      </div>
      <div v-else-if="!site" class="py-16 text-center text-sm text-muted-foreground">
        {{ $t('sites.detail.notFound') }}
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
