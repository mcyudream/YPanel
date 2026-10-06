<script setup lang="ts">
import type { MarketApp } from '@/api/modules/market'
import apiMarket from '@/api/modules/market'

defineOptions({
  name: 'MarketIndex',
})

const apps = ref<MarketApp[]>([])
const loading = ref(false)
const installing = ref('')
const paramsModalVisible = ref(false)
const paramsApp = ref<MarketApp | null>(null)
const paramsForm = ref<Record<string, string>>({})

const categories = ref<string[]>([])

async function load() {
  loading.value = true
  try {
    apps.value = await apiMarket.list()
    categories.value = [...new Set(apps.value.map(a => a.category))]
  }
  finally {
    loading.value = false
  }
}

function openInstall(app: MarketApp) {
  paramsApp.value = app
  paramsForm.value = Object.fromEntries(app.params.map(p => [p.key, p.default]))
  paramsModalVisible.value = true
}

async function doInstall() {
  if (!paramsApp.value) {
    return
  }
  installing.value = paramsApp.value.id
  try {
    await apiMarket.install(paramsApp.value.id, paramsForm.value)
    useFaToast().success(`应用 ${paramsApp.value.name} 安装完成（首次拉取镜像可能较慢）`)
    paramsModalVisible.value = false
  }
  catch (e: any) {
    useFaToast().error('安装失败', { description: e?.message })
  }
  finally {
    installing.value = ''
  }
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="package" :size="24" />
          <span>应用市场</span>
        </div>
      </template>
      <template #description>
        <span>内置精选应用一键安装；兼容 1Panel 应用目录格式（/opt/ypanel/apps/&lt;app&gt;/&lt;version&gt;/）</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <template v-for="cat in categories" :key="cat">
        <div class="mb-2 mt-4 flex items-center gap-2 text-sm font-medium first:mt-0">
          <YdMorphIcon name="tag" :size="14" />
          {{ cat }}
        </div>
        <div class="mb-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <div v-for="a in apps.filter(x => x.category === cat)" :key="a.id" class="rounded-lg border bg-background p-4">
            <div class="flex items-start justify-between">
              <div>
                <div class="font-medium">{{ a.name }}</div>
                <div class="mt-1 line-clamp-2 min-h-10 text-xs text-muted-foreground">{{ a.description }}</div>
              </div>
              <span v-if="a.source === '1panel'" class="shrink-0 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">1P</span>
            </div>
            <div class="mt-3 flex items-center justify-between border-t pt-3">
              <span class="font-mono text-xs text-muted-foreground">{{ a.id }}</span>
              <FaButton size="sm" @click="openInstall(a)">安装</FaButton>
            </div>
          </div>
        </div>
      </template>
    </FaPageMain>

    <FaModal v-model="paramsModalVisible" :title="`安装：${paramsApp?.name || ''}`" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div v-for="p in paramsApp?.params || []" :key="p.key" class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">{{ p.label }}</span>
          <FaInput v-model="paramsForm[p.key]" :type="p.isPassword ? 'password' : 'text'" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="paramsModalVisible = false">取消</FaButton>
        <FaButton :loading="installing === paramsApp?.id" @click="doInstall">安装并启动</FaButton>
      </template>
    </FaModal>
  </div>
</template>
