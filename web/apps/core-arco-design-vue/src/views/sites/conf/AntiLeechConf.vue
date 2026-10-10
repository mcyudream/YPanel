<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteAntiLeech } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteAntiLeech | null>(null)
const edit = ref<SiteAntiLeech>({ enable: false, validReferers: [], allowNone: true, allowBlocked: true, returnCode: 403 })
const referersRaw = ref('')
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteExtraApi.getAntiLeech(props.site.id)
    edit.value = { ...conf.value }
    referersRaw.value = (conf.value.validReferers || []).join('\n')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.antileech.loadFailed'), { description: e?.message })
  }
}

async function save() {
  saving.value = true
  try {
    edit.value.validReferers = referersRaw.value.split('\n').map(x => x.trim()).filter(Boolean)
    conf.value = await siteExtraApi.updateAntiLeech(props.site.id, edit.value)
    edit.value = { ...conf.value }
    toast.success(i18n.global.t('sites.conf.antileech.saved'))
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
  ? edit.value.enable !== conf.value.enable
    || edit.value.returnCode !== conf.value.returnCode
    || edit.value.allowNone !== conf.value.allowNone
    || edit.value.allowBlocked !== conf.value.allowBlocked
    || JSON.stringify(edit.value.validReferers) !== JSON.stringify(conf.value.validReferers || [])
  : false)

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-sm font-medium">{{ $t('sites.conf.antileech.enable') }}</div>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ $t('sites.conf.antileech.enableDesc') }}</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.enable" type="checkbox"> {{ $t('common.enabled') }}
        </label>
      </div>
    </div>

    <div class="space-y-4 rounded-lg border p-4">
      <div>
        <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.antileech.whitelist') }}</div>
        <p class="mb-2 text-xs text-muted-foreground">{{ $t('sites.conf.antileech.whitelistDesc') }}</p>
        <textarea v-model="referersRaw" rows="4" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="*.example.com&#10;friend-site.com" />
      </div>
      <div class="flex flex-wrap gap-5 text-sm">
        <label class="flex cursor-pointer items-center gap-2">
          <input v-model="edit.allowNone" type="checkbox"> {{ $t('sites.conf.antileech.allowNone') }}
        </label>
        <label class="flex cursor-pointer items-center gap-2">
          <input v-model="edit.allowBlocked" type="checkbox"> {{ $t('sites.conf.antileech.allowBlocked') }}
        </label>
        <label class="flex items-center gap-2">
          {{ $t('sites.conf.antileech.returnCode') }}
          <YdSelect v-model="edit.returnCode" :options="[403, 404]" />
        </label>
      </div>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
