<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteRedirect } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteRedirect | null>(null)
const edit = ref<SiteRedirect>({ enable: false, target: '', code: 301 })
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteExtraApi.getRedirect(props.site.id)
    edit.value = { ...conf.value }
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.redirect.loadFailed'), { description: e?.message })
  }
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteExtraApi.updateRedirect(props.site.id, edit.value)
    edit.value = { ...conf.value }
    toast.success(i18n.global.t('sites.conf.redirect.saved'))
    emit('changed')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.shared.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

const dirty = computed(() => conf.value ? JSON.stringify(edit.value) !== JSON.stringify(conf.value) : false)

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-sm font-medium">{{ $t('sites.conf.redirect.enable') }}</div>
          <p class="mt-0.5 text-xs text-muted-foreground">
            {{ site.certDomain ? $t('sites.conf.redirect.enableDescHttps') : $t('sites.conf.redirect.enableDesc') }}
          </p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm" :class="site.certDomain ? 'pointer-events-none opacity-50' : ''">
          <input v-model="edit.enable" type="checkbox"> {{ $t('common.enabled') }}
        </label>
      </div>
    </div>

    <div class="space-y-4 rounded-lg border p-4">
      <div>
        <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.redirect.target') }}</div>
        <FaInput v-model="edit.target" placeholder="https://new-domain.com" class="w-full" />
      </div>
      <div>
        <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.redirect.code') }}</div>
        <YdSelect
          v-model="edit.code"
          :options="[
            { label: $t('sites.conf.redirect.code301'), value: 301 },
            { label: $t('sites.conf.redirect.code302'), value: 302 },
            { label: $t('sites.conf.redirect.code307'), value: 307 },
            { label: $t('sites.conf.redirect.code308'), value: 308 },
          ]"
          size="default"
          button-class="w-full md:w-64"
        />
      </div>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
