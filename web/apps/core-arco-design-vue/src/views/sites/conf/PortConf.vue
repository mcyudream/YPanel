<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SitePortConf } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SitePortConf | null>(null)
const port = ref(80)
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteConfApi.getPort(props.site.id)
    port.value = conf.value.port
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.port.loadFailed'), { description: e?.message })
  }
}

async function save() {
  const v = Number(port.value)
  if (!Number.isInteger(v) || v < 1 || v > 65535) {
    toast.error(i18n.global.t('sites.conf.port.invalid'))
    return
  }
  saving.value = true
  try {
    conf.value = await siteConfApi.updatePort(props.site.id, v)
    port.value = conf.value.port
    toast.success(i18n.global.t('sites.conf.port.saved'))
    emit('changed')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.shared.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

const dirty = computed(() => conf.value ? Number(port.value) !== conf.value.port : false)

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="space-y-4 rounded-lg border p-4">
      <div>
        <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.port.listen') }}</div>
        <FaInput v-model.number="port" type="number" class="w-40" />
        <span v-if="conf" class="ml-3 inline-flex items-center gap-1 text-xs" :class="conf.applied ? 'text-emerald-600' : 'text-amber-600'">
          {{ conf.applied ? $t('sites.conf.port.applied') : $t('sites.conf.port.pending') }}
        </span>
      </div>
      <p class="text-xs text-muted-foreground">
        {{ conf?.mode === 'host' ? $t('sites.conf.port.hostHint') : $t('sites.conf.port.containerHint') }}
      </p>
      <p class="text-xs text-muted-foreground">{{ $t('sites.conf.port.sharedHint') }}</p>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
