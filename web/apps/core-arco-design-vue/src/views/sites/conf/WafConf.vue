<script setup lang="ts">
import type { SiteItem, SiteWaf } from '@/api/modules/site'
import { wafApi } from '@/api/modules/site'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const form = ref<SiteWaf>({ denyIps: [], allowIps: [], denyUAs: [], rateEnable: false, rate: 10, burst: 20 })
const denyIps = ref('')
const allowIps = ref('')
const denyUAs = ref('')
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const w = await wafApi.get(props.site.id)
    form.value = { ...w, denyIps: w.denyIps || [], allowIps: w.allowIps || [], denyUAs: w.denyUAs || [] }
    denyIps.value = (w.denyIps || []).join(', ')
    allowIps.value = (w.allowIps || []).join(', ')
    denyUAs.value = (w.denyUAs || []).join(', ')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.waf.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const w: SiteWaf = {
      denyIps: denyIps.value.split(/[,\n]/).map(x => x.trim()).filter(Boolean),
      allowIps: allowIps.value.split(/[,\n]/).map(x => x.trim()).filter(Boolean),
      denyUAs: denyUAs.value.split(/[,\n]/).map(x => x.trim()).filter(Boolean),
      rateEnable: form.value.rateEnable,
      rate: Number(form.value.rate) || 10,
      burst: Number(form.value.burst) || 20,
    }
    await wafApi.update(props.site.id, w)
    toast.success(i18n.global.t('sites.conf.waf.saved'))
    emit('changed')
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.waf.saveFailedRollback'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="space-y-4 rounded-lg border p-4">
      <div class="text-sm font-medium">{{ $t('sites.conf.waf.denyIps') }}</div>
      <textarea v-model="denyIps" rows="2" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="1.2.3.4, 5.6.7.0/24" />
      <div class="text-sm font-medium">{{ $t('sites.conf.waf.allowIps') }}</div>
      <textarea v-model="allowIps" rows="2" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="192.168.1.0/24" />
      <div class="text-sm font-medium">{{ $t('sites.conf.waf.denyUAs') }}</div>
      <textarea v-model="denyUAs" rows="2" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="curl&#10;sqlmap" />
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-3 flex items-center justify-between">
        <div>
          <div class="text-sm font-medium">{{ $t('sites.conf.waf.rateTitle') }}</div>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ $t('sites.conf.waf.rateDesc') }}</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="form.rateEnable" type="checkbox"> {{ $t('common.enabled') }}
        </label>
      </div>
      <div v-if="form.rateEnable" class="flex flex-wrap items-center gap-4 text-sm">
        <label class="flex items-center gap-2">
          {{ $t('sites.conf.waf.rate') }}
          <FaInput v-model="form.rate" type="number" class="w-24" />
        </label>
        <label class="flex items-center gap-2">
          {{ $t('sites.conf.waf.burst') }}
          <FaInput v-model="form.burst" type="number" class="w-24" />
        </label>
      </div>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
