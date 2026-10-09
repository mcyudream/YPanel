<script setup lang="ts">
import api from '@/api/index'
import { i18n } from '@/locales'

defineOptions({
  name: 'SystemManage',
})

const toast = useFaToast()
const modal = useFaModal()

// ---- swap ----
const swap = ref<{ files: { file: string, size: string }[], totalBytes: number, usedBytes: number, on: boolean } | null>(null)
const swapBusy = ref(false)

async function loadSwap() {
  try {
    swap.value = (await api.get('api/v1/system/swap', { silent: true })).data
  }
  catch (e: any) {
    toast.error(i18n.global.t('system.swapReadFail'), { description: e?.message })
  }
}

function applySwap(sizeGB: number) {
  const label = sizeGB === 0 ? i18n.global.t('system.swapDisableLabel') : i18n.global.t('system.swapResizeLabel', { size: sizeGB })
  modal.confirm({
    title: i18n.global.t('system.swapAdjustTitle'),
    content: i18n.global.t('system.swapConfirm', { label }),
    onConfirm: async () => {
      swapBusy.value = true
      try {
        await api.post('api/v1/system/swap', { sizeGB })
        toast.success(i18n.global.t('system.applied'))
        await loadSwap()
      }
      catch (e: any) {
        toast.error(i18n.global.t('system.opFailed'), { description: e?.message })
      }
      finally {
        swapBusy.value = false
      }
    },
  })
}

// ---- BBR ----
const bbr = ref<{ algo: string, qdisc: string, bbr: boolean } | null>(null)
const bbrBusy = ref(false)

async function loadBBR() {
  try {
    bbr.value = (await api.get('api/v1/system/bbr', { silent: true })).data
  }
  catch (e: any) {
    toast.error(i18n.global.t('system.bbrReadFail'), { description: e?.message })
  }
}

async function applyBBR(enable: boolean) {
  bbrBusy.value = true
  try {
    const out = (await api.post('api/v1/system/bbr', { enable })).data
    toast.success(enable ? i18n.global.t('system.bbrOn', { algo: out.algo }) : i18n.global.t('system.bbrOff'))
    await loadBBR()
  }
  catch (e: any) {
    toast.error(i18n.global.t('system.opFailed'), { description: e?.message })
  }
  finally {
    bbrBusy.value = false
  }
}

// ---- 系统清理 ----
const cleanBusy = ref(false)
const cleanOut = ref('')

async function doClean() {
  modal.confirm({
    title: i18n.global.t('system.clean'),
    content: i18n.global.t('system.cleanConfirm'),
    onConfirm: async () => {
      cleanBusy.value = true
      try {
        const out = (await api.post('api/v1/system/clean')).data
        cleanOut.value = out.output || i18n.global.t('system.cleanNoOutput')
        toast.success(i18n.global.t('system.cleanDone'))
      }
      catch (e: any) {
        toast.error(i18n.global.t('system.cleanFail'), { description: e?.message })
      }
      finally {
        cleanBusy.value = false
      }
    },
  })
}

onMounted(() => {
  loadSwap()
  loadBBR()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex gap-2 items-center">
          <FaIcon name="i-lucide:wrench" :size="22" />
          <span>{{ $t('system.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('system.desc') }}</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="gap-4 grid grid-cols-1 lg:grid-cols-2">
        <!-- Swap -->
        <section class="p-4 border rounded-lg bg-background">
          <div class="flex gap-2 items-center">
            <FaIcon name="i-lucide:hard-drive" class="text-base text-primary opacity-70" />
            <span class="font-medium">{{ $t('system.swap') }}</span>
            <span class="text-xs px-2 py-0.5 rounded-full" :class="swap?.on ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ swap?.on ? `${$t('system.swapOn')}（${((swap.totalBytes || 0) / 1024 / 1024 / 1024).toFixed(1)}G）` : $t('system.swapOff') }}
            </span>
          </div>
          <div v-if="swap?.files?.length" class="text-xs text-muted-foreground font-mono mt-2">
            {{ swap.files.map(f => `${f.file}(${f.size})`).join('  ') }}
          </div>
          <div class="mt-3 flex flex-wrap gap-2 items-center">
            <FaButton variant="outline" size="sm" :disabled="swapBusy" @click="applySwap(1)">
              1G
            </FaButton>
            <FaButton variant="outline" size="sm" :disabled="swapBusy" @click="applySwap(2)">
              2G
            </FaButton>
            <FaButton variant="outline" size="sm" :disabled="swapBusy" @click="applySwap(4)">
              4G
            </FaButton>
            <FaButton variant="outline" size="sm" :disabled="swapBusy" @click="applySwap(8)">
              8G
            </FaButton>
            <FaButton variant="outline" size="sm" class="text-red-500!" :disabled="swapBusy || !swap?.on" @click="applySwap(0)">
              {{ $t('system.swapClose') }}
            </FaButton>
          </div>
        </section>

        <!-- BBR -->
        <section class="p-4 border rounded-lg bg-background">
          <div class="flex gap-2 items-center">
            <FaIcon name="i-lucide:gauge" class="text-base text-primary opacity-70" />
            <span class="font-medium">{{ $t('system.bbr') }}</span>
            <span class="text-xs px-2 py-0.5 rounded-full" :class="bbr?.bbr ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ bbr?.algo || '—' }}
            </span>
          </div>
          <div class="text-xs text-muted-foreground mt-2">
            {{ $t('system.bbrDesc', { qdisc: bbr?.qdisc || '—' }) }}
          </div>
          <div class="mt-3 flex gap-2 items-center">
            <FaButton v-if="!bbr?.bbr" size="sm" :loading="bbrBusy" @click="applyBBR(true)">
              {{ $t('system.bbrEnable') }}
            </FaButton>
            <FaButton v-else variant="outline" size="sm" class="text-red-500!" :loading="bbrBusy" @click="applyBBR(false)">
              {{ $t('system.bbrDisable') }}
            </FaButton>
          </div>
        </section>

        <!-- 清理 -->
        <section class="p-4 border rounded-lg bg-background lg:col-span-2">
          <div class="flex flex-wrap gap-2 items-center">
            <FaIcon name="i-lucide:brush" class="text-base text-primary opacity-70" />
            <span class="font-medium">{{ $t('system.clean') }}</span>
            <span class="text-xs text-muted-foreground">{{ $t('system.cleanDesc') }}</span>
            <FaButton variant="outline" size="sm" class="ml-auto" :loading="cleanBusy" @click="doClean">
              {{ $t('system.cleanRun') }}
            </FaButton>
          </div>
          <pre v-if="cleanOut" class="text-xs leading-relaxed font-mono mt-3 p-3 border rounded-md bg-muted/30 max-h-56 overflow-auto">{{ cleanOut }}</pre>
        </section>
      </div>
    </FaPageMain>
  </div>
</template>
