<script setup lang="ts">
import api from '@/api'
import { rawApi } from '@/api'
import { i18n } from '@/locales'

import type { OnlineCheck } from '@/api/modules/selfupdate'
import { selfUpdateApi } from '@/api/modules/selfupdate'

interface UpdateFile {
  name: string
  sizeMb: number
  modTime: string
}

const router = useRouter()

defineOptions({
  name: 'SelfUpdateIndex',
})

const status = ref<{ currentVersion: string, channelDir: string, available: UpdateFile[] }>()
const loading = ref(false)
const applying = ref('')

// ---- 在线更新（GitHub/Gitee 双源） ----
const checking = ref(false)
const online = ref<OnlineCheck | null>(null)
const upgrading = ref(false)
const notesOpen = ref(false)

async function checkOnline() {
  checking.value = true
  online.value = null
  try {
    online.value = await selfUpdateApi.check()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('selfupdate.online.checkFailed'), { description: e?.message })
  }
  finally {
    checking.value = false
  }
}

function bestSource(): 'github' | 'gitee' {
  if (!online.value) {
    return 'gitee'
  }
  const ok = online.value.sources.filter(s => s.reachable && s.assetUrl)
  if (!ok.length) {
    return 'gitee'
  }
  // 版本相同优先 gitee（国内下载快），否则选版本更高的源
  const sorted = [...ok].sort((a, b) => {
    const cmp = compareV(b.version, a.version)
    if (cmp !== 0) {
      return cmp
    }
    return a.source === 'gitee' ? -1 : 1
  })
  return sorted[0].source
}

function compareV(a?: string, b?: string) {
  const pa = (a || '').replace(/^v/, '').split('.').map(x => Number.parseInt(x) || 0)
  const pb = (b || '').replace(/^v/, '').split('.').map(x => Number.parseInt(x) || 0)
  for (let i = 0; i < 3; i++) {
    if ((pa[i] || 0) !== (pb[i] || 0)) {
      return (pa[i] || 0) - (pb[i] || 0)
    }
  }
  return 0
}

function upgrade() {
  const source = bestSource()
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('selfupdate.online.upgradeTitle'),
    content: i18n.global.t('selfupdate.online.upgradeConfirm', {
      version: online.value?.latest || '',
      source: source === 'gitee' ? 'Gitee' : 'GitHub',
    }),
    onConfirm: async () => {
      upgrading.value = true
      try {
        await selfUpdateApi.upgrade(source)
        useFaToast().info(i18n.global.t('selfupdate.toast.updateStarted'))
        // 轮询 /health 直到恢复
        let recovered = false
        for (let i = 0; i < 30; i++) {
          await new Promise(r => setTimeout(r, 2000))
          try {
            const h = await rawApi.get('/health').then(r => r.data)
            if (h.status === 'ok' && h.version && h.version !== online.value?.currentVersion) {
              recovered = true
              useFaToast().success(i18n.global.t('selfupdate.toast.recovered', { version: h.version }))
              break
            }
          }
          catch {}
        }
        if (!recovered) {
          useFaToast().error(i18n.global.t('selfupdate.toast.notRecovered'))
        }
        await load()
        await checkOnline()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('selfupdate.toast.applyFailed'), { description: e?.message })
      }
      finally {
        upgrading.value = false
      }
    },
  })
}

async function load() {
  loading.value = true
  try {
    const res = await api.get('api/v1/system/update/status', { silent: true })
    status.value = res.data
  }
  finally {
    loading.value = false
  }
}

function apply(file: string) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('selfupdate.modal.applyTitle'),
    content: i18n.global.t('selfupdate.modal.applyConfirm', { file }),
    onConfirm: async () => {
      applying.value = file
      try {
        await api.post('api/v1/system/update/apply', { file })
        useFaToast().info(i18n.global.t('selfupdate.toast.updateStarted'))
        // 轮询 health 直到恢复
        let recovered = false
        for (let i = 0; i < 20; i++) {
          await new Promise(r => setTimeout(r, 1500))
          try {
            const h = await rawApi.get('/health').then(r => r.data)
            if (h.status === 'ok') {
              recovered = true
              useFaToast().success(i18n.global.t('selfupdate.toast.recovered', { version: h.version }))
              break
            }
          }
          catch {}
        }
        if (!recovered) {
          useFaToast().error(i18n.global.t('selfupdate.toast.notRecovered'))
        }
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('selfupdate.toast.applyFailed'), { description: e?.message })
      }
      finally {
        applying.value = ''
      }
    },
  })
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="refresh-cw" :size="24" />
          <span>{{ $t('selfupdate.page.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('selfupdate.page.desc', { dir: status?.channelDir || '/opt/ypanel/updates' }) }}</span>
      </template>
      <FaButton variant="outline" size="sm" @click="() => router.push('/manage/security')">
        <FaIcon name="i-lucide:shield-check" class="mr-1" /> {{ $t('selfupdate.page.security') }}
      </FaButton>
      <FaButton variant="outline" size="sm" @click="load">
        <FaIcon name="i-lucide:refresh-cw" class="mr-1" :class="loading ? 'animate-spin' : ''" /> {{ $t('common.refresh') }}
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div class="mb-4 flex items-center gap-2 rounded-lg border bg-background p-4">
        <YdMorphIcon name="info" :size="18" class="text-primary" />
        {{ $t('selfupdate.page.currentVersion') }}<code class="rounded bg-muted px-1.5 py-0.5 font-mono text-sm">{{ status?.currentVersion || '—' }}</code>
      </div>

      <!-- 在线更新（GitHub/Gitee 双源） -->
      <div class="mb-4 rounded-lg border bg-background p-4">
        <div class="flex flex-wrap items-center gap-2">
          <FaIcon name="i-lucide:cloud-download" class="text-base text-primary opacity-70" />
          <span class="font-medium">{{ $t('selfupdate.online.title') }}</span>
          <span class="text-xs text-muted-foreground">{{ $t('selfupdate.online.desc') }}</span>
          <FaButton class="ml-auto" size="sm" :loading="checking" @click="checkOnline">
            <FaIcon name="i-lucide:search-check" class="mr-1" /> {{ $t('selfupdate.online.check') }}
          </FaButton>
          <FaButton
            v-if="online?.updatable" size="sm" :loading="upgrading" @click="upgrade"
          >
            <FaIcon name="i-lucide:rocket" class="mr-1" /> {{ $t('selfupdate.online.upgradeTo', { version: online?.latest }) }}
          </FaButton>
        </div>
        <div v-if="online" class="mt-3 flex flex-col gap-2 text-sm">
          <div class="flex flex-wrap items-center gap-2 text-xs">
            <span>{{ $t('selfupdate.online.latest') }}: <code class="rounded bg-muted px-1.5 py-0.5 font-mono">{{ online.latest || '—' }}</code></span>
            <span
              v-for="src in online.sources" :key="src.source" class="rounded-full px-2 py-0.5"
              :class="src.reachable ? (src.assetUrl ? 'bg-emerald-500/10 text-emerald-600' : 'bg-amber-500/10 text-amber-600') : 'bg-muted text-muted-foreground'"
              :title="src.error || ''"
            >
              {{ src.source === 'gitee' ? 'Gitee' : 'GitHub' }}: {{ !src.reachable ? $t('selfupdate.online.unreachable') : src.assetUrl ? src.version : $t('selfupdate.online.noAsset') }}
            </span>
            <span v-if="online.currentIsDev" class="rounded-full bg-sky-500/10 px-2 py-0.5 text-sky-600">{{ $t('selfupdate.online.devVersion') }}</span>
            <span v-else-if="!online.updatable && online.latest" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-emerald-600">{{ $t('selfupdate.online.upToDate') }}</span>
          </div>
          <button v-if="online.sources.some(s => s.notes)" class="self-start text-xs text-primary" @click="notesOpen = !notesOpen">
            {{ notesOpen ? $t('selfupdate.online.hideNotes') : $t('selfupdate.online.showNotes') }}
          </button>
          <pre v-if="notesOpen" class="max-h-48 overflow-auto whitespace-pre-wrap rounded-md bg-muted/50 p-3 font-mono text-xs">{{ online.sources.find(s => s.notes)?.notes }}</pre>
        </div>
      </div>

      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('selfupdate.table.file') }}</th>
              <th class="px-3 py-2">{{ $t('common.size') }}</th>
              <th class="hidden px-3 py-2 md:table-cell">{{ $t('selfupdate.table.time') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!status?.available?.length && !loading">
              <td colspan="4" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('selfupdate.table.empty', { dir: status?.channelDir || '' }) }}
              </td>
            </tr>
            <tr v-for="f in status?.available || []" :key="f.name" class="border-t hover:bg-accent/30">
              <td class="px-3 py-2 font-mono text-[13px]">{{ f.name }}</td>
              <td class="px-3 py-2 text-xs tabular-nums">{{ f.sizeMb.toFixed(1) }} MB</td>
              <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground md:table-cell">
                {{ new Date(f.modTime).toLocaleString('zh-CN', { hour12: false }) }}
              </td>
              <td class="px-3 py-2 text-right">
                <FaButton size="sm" :disabled="applying === f.name" @click="apply(f.name)">
                  {{ applying === f.name ? $t('selfupdate.table.applying') : $t('selfupdate.table.applyRestart') }}
                </FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>
  </div>
</template>
