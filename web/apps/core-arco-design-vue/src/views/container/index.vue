<script setup lang="ts">
// M23 容器中心：应用（默认）/容器/镜像/网络/卷/配置 六区合一（对标 Portainer 的环境多 tab 结构）。
// 旧 /docker、/compose 路由重定向到此页对应 tab。
import AppsTab from './tabs/AppsTab.vue'
import ImagesTab from './tabs/ImagesTab.vue'
import ListTab from './tabs/ListTab.vue'
import NetworksTab from './tabs/NetworksTab.vue'
import SettingsTab from './tabs/SettingsTab.vue'
import VolumesTab from './tabs/VolumesTab.vue'

defineOptions({
  name: 'ContainerIndex',
})

const route = useRoute()
const router = useRouter()

const TABS: { label: string, value: string, icon: string }[] = [
  { label: '应用', value: 'apps', icon: 'i-lucide:layout-grid' },
  { label: '容器', value: 'list', icon: 'i-lucide:boxes' },
  { label: '镜像', value: 'images', icon: 'i-lucide:disc' },
  { label: '网络', value: 'networks', icon: 'i-lucide:network' },
  { label: '卷', value: 'volumes', icon: 'i-lucide:hard-drive' },
  { label: '配置', value: 'settings', icon: 'i-lucide:settings' },
]

const activeTab = computed<string>({
  get() {
    const t = String(route.query.tab || 'apps')
    return TABS.some(x => x.value === t) ? t : 'apps'
  },
  set(v) {
    router.replace({ query: { ...route.query, tab: v === 'apps' ? undefined : v } })
  },
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2.5">
          <FaIcon name="i-tabler:brand-docker" class="text-2xl" />
          <span>容器</span>
        </div>
      </template>
      <template #description>
        <span>应用编排 / 容器 / 镜像 / 网络 / 卷 / Docker 配置的统一入口</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <FaTabs v-model="activeTab" :list="TABS" />

      <div class="mt-3">
        <AppsTab v-if="activeTab === 'apps'" />
        <ListTab v-else-if="activeTab === 'list'" />
        <ImagesTab v-else-if="activeTab === 'images'" />
        <NetworksTab v-else-if="activeTab === 'networks'" />
        <VolumesTab v-else-if="activeTab === 'volumes'" />
        <SettingsTab v-else-if="activeTab === 'settings'" />
      </div>
    </FaPageMain>
  </div>
</template>
