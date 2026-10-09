<script setup lang="ts">
import type { SystemOverview } from '@/api/modules/system'
import { i18n } from '@/locales'

// M44 欢迎横幅：问候 + 主机身份 + 实时时钟 + 负载摘要 + 桌面工作台入口。
const props = defineProps<{
  overview: SystemOverview | null
}>()

const router = useRouter()

const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6) {
    return i18n.global.t('overview.banner.night')
  }
  if (h < 12) {
    return i18n.global.t('overview.banner.morning')
  }
  if (h < 18) {
    return i18n.global.t('overview.banner.afternoon')
  }
  return i18n.global.t('overview.banner.evening')
})

const now = ref(new Date())
let clockTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  clockTimer = setInterval(() => {
    now.value = new Date()
  }, 1000)
})

onBeforeUnmount(() => {
  if (clockTimer) {
    clearInterval(clockTimer)
  }
})

const clockText = computed(() => now.value.toLocaleTimeString(i18n.global.locale.value === 'en-US' ? 'en-US' : 'zh-CN', { hour12: false }))
const dateText = computed(() => now.value.toLocaleDateString(i18n.global.locale.value === 'en-US' ? 'en-US' : 'zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' }))

const metaChips = computed(() => {
  const o = props.overview
  if (!o) {
    return []
  }
  return [
    `${o.platform} · ${o.arch}`,
    o.kernelVersion || '',
    `${o.cpu.modelName || i18n.global.t('overview.sys.unknownModel')} · ${i18n.global.t('overview.sys.cores', { n: o.cpu.logicalCount })}`,
  ].filter(Boolean)
})
</script>

<template>
  <div class="yp-banner relative overflow-hidden rounded-2xl p-5 text-white md:p-6">
    <div class="relative flex flex-col gap-5 md:flex-row md:items-center md:justify-between">
      <div class="min-w-0">
        <div class="text-sm opacity-85">
          {{ greeting }}，{{ overview?.hostname ?? '—' }}
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-2 text-xs">
          <span
            v-for="chip in metaChips"
            :key="chip"
            class="rounded-full bg-white/15 px-2.5 py-1 backdrop-blur-sm"
          >{{ chip }}</span>
        </div>
      </div>
      <div class="flex items-center gap-5 md:flex-col md:items-end md:gap-2">
        <div class="md:text-right">
          <div class="text-2xl font-bold tabular-nums md:text-3xl">
            {{ clockText }}
          </div>
          <div class="mt-0.5 text-xs opacity-80">
            {{ dateText }}
          </div>
        </div>
        <button
          type="button"
          class="inline-flex cursor-pointer items-center gap-1.5 rounded-full bg-white/15 px-3 py-1.5 text-xs backdrop-blur-sm transition-colors hover:bg-white/25"
          :title="$t('overview.desktopTip')"
          @click="router.push('/desktop')"
        >
          <YdMorphIcon name="layout-grid" :size="13" /> {{ $t('overview.desktop') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yp-banner {
  background: linear-gradient(115deg, #1d39c4 0%, #165dff 48%, #0fc6c2 115%);
}

html.dark .yp-banner {
  background: linear-gradient(115deg, #10205c 0%, #12359e 50%, #0a5c58 115%);
}
</style>
