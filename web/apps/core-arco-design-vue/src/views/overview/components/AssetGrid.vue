<script setup lang="ts">
import type { Dashboard } from '@/api/modules/system'
import { i18n } from '@/locales'

// M44 管理对象卡行：网站 / 数据库 / 容器 / 镜像 / 证书 / 计划任务 / 节点 / 应用商店。
const props = defineProps<{
  dashboard: Dashboard | null
}>()

const router = useRouter()
const t = (key: string, args?: Record<string, unknown>) => i18n.global.t(key, args as any)

interface CardDef {
  key: string
  icon: string
  path: string
  docker?: boolean
}

const cards: CardDef[] = [
  { key: 'sites', icon: 'globe', path: '/sites' },
  { key: 'databases', icon: 'database', path: '/database' },
  { key: 'containers', icon: 'box', path: '/container/list', docker: true },
  { key: 'images', icon: 'image', path: '/container/images', docker: true },
  { key: 'certs', icon: 'shield-check', path: '/certs' },
  { key: 'cron', icon: 'calendar-clock', path: '/cron' },
  { key: 'nodes', icon: 'network', path: '/nodes' },
  { key: 'store', icon: 'package', path: '/store' },
]

function valueOf(card: CardDef): string {
  const d = props.dashboard
  if (!d) {
    return '—'
  }
  if (card.docker && !d.dockerAvailable) {
    return '—'
  }
  switch (card.key) {
    case 'sites':
      return String(d.sites)
    case 'databases':
      return String(d.databases)
    case 'containers':
      return `${d.containersRunning}/${d.containersTotal}`
    case 'images':
      return String(d.images)
    case 'certs':
      return String(d.certsOK + d.certsExpiring + d.certsExpired)
    case 'cron':
      return String(d.cronTasks)
    case 'nodes':
      return `${d.nodesOnline}/${d.nodesTotal}`
    case 'store':
      return t('overview.assets.enter')
    default:
      return '—'
  }
}

function subOf(card: CardDef): { text: string, tone: 'ok' | 'warn' | 'danger' | 'muted' } | null {
  const d = props.dashboard
  if (!d) {
    return null
  }
  if (card.docker && !d.dockerAvailable) {
    return { text: t('overview.assets.dockerDown'), tone: 'muted' }
  }
  switch (card.key) {
    case 'containers':
      return { text: t('overview.assets.running'), tone: 'ok' }
    case 'certs':
      if (d.certsExpired > 0) {
        return { text: t('overview.assets.expired', { n: d.certsExpired }), tone: 'danger' }
      }
      if (d.certsExpiring > 0) {
        return { text: t('overview.assets.expiring', { n: d.certsExpiring }), tone: 'warn' }
      }
      return { text: t('overview.assets.allOk'), tone: 'ok' }
    case 'nodes':
      return { text: t('overview.assets.online'), tone: d.nodesOnline < d.nodesTotal ? 'warn' : 'ok' }
    default:
      return null
  }
}

const toneClass = {
  ok: 'text-emerald-600 dark:text-emerald-400',
  warn: 'text-amber-600 dark:text-amber-400',
  danger: 'text-red-500',
  muted: 'text-muted-foreground',
} as const

function go(card: CardDef) {
  const d = props.dashboard
  if (card.docker && d && !d.dockerAvailable) {
    return
  }
  router.push(card.path)
}
</script>

<template>
  <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
    <button
      v-for="card in cards"
      :key="card.key"
      type="button"
      class="group flex cursor-pointer items-center gap-3 rounded-xl border bg-background p-4 text-left transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
      :class="card.docker && dashboard && !dashboard.dockerAvailable ? 'opacity-60' : ''"
      @click="go(card)"
    >
      <div class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
        <YdMorphIcon :name="card.icon" :size="20" />
      </div>
      <div class="min-w-0">
        <div class="text-xl leading-7 font-bold tabular-nums">
          {{ valueOf(card) }}
        </div>
        <div class="truncate text-xs text-muted-foreground">
          {{ $t(`overview.assets.${card.key}`) }}
        </div>
        <div v-if="subOf(card)" class="truncate text-[11px]" :class="toneClass[subOf(card)!.tone]">
          {{ subOf(card)!.text }}
        </div>
      </div>
    </button>
  </div>
</template>
