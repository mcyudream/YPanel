<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteRealIP } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteRealIP | null>(null)
const edit = ref<SiteRealIP>({ enable: false, trustedProxies: [], header: 'X-Forwarded-For' })
const proxiesRaw = ref('')
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteExtraApi.getRealIP(props.site.id)
    edit.value = { ...conf.value }
    proxiesRaw.value = (conf.value.trustedProxies || []).join('\n')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.realip.loadFailed'), { description: e?.message })
  }
}

async function save() {
  saving.value = true
  try {
    edit.value.trustedProxies = proxiesRaw.value.split('\n').map(x => x.trim()).filter(Boolean)
    conf.value = await siteExtraApi.updateRealIP(props.site.id, edit.value)
    edit.value = { ...conf.value }
    proxiesRaw.value = (conf.value.trustedProxies || []).join('\n')
    toast.success(i18n.global.t('sites.conf.realip.saved'))
    emit('changed')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.shared.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

const dirty = computed(() => conf.value
  ? edit.value.enable !== conf.value.enable || edit.value.header !== conf.value.header
    || JSON.stringify(edit.value.trustedProxies) !== JSON.stringify(conf.value.trustedProxies || [])
  : false)

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-sm font-medium">{{ $t('sites.conf.realip.enable') }}</div>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ $t('sites.conf.realip.enableDesc') }}</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.enable" type="checkbox"> {{ $t('common.enabled') }}
        </label>
      </div>
    </div>

    <div class="space-y-4 rounded-lg border p-4">
      <div>
        <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.realip.proxies') }}</div>
        <p class="mb-2 text-xs text-muted-foreground">{{ $t('sites.conf.realip.proxiesDesc') }}</p>
        <textarea v-model="proxiesRaw" rows="4" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="173.245.48.0/20&#10;172.17.0.0/16" />
      </div>
      <div>
        <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.realip.header') }}</div>
        <select v-model="edit.header" class="h-9 w-full rounded-md border bg-background px-2 text-sm outline-none md:w-72">
          <option value="X-Forwarded-For">{{ $t('sites.conf.realip.headerXFF') }}</option>
          <option value="X-Real-IP">{{ $t('sites.conf.realip.headerXRealIP') }}</option>
          <option value="CF-Connecting-IP">{{ $t('sites.conf.realip.headerCF') }}</option>
        </select>
      </div>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
