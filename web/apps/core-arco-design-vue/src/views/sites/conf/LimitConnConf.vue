<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteLimitConn } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteLimitConn | null>(null)
const edit = ref<SiteLimitConn>({ enable: false, connPerIP: 20 })
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteExtraApi.getLimitConn(props.site.id)
    edit.value = { ...conf.value }
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.limitconn.loadFailed'), { description: e?.message })
  }
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteExtraApi.updateLimitConn(props.site.id, edit.value)
    edit.value = { ...conf.value }
    toast.success(i18n.global.t('sites.conf.limitconn.saved'))
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
          <div class="text-sm font-medium">{{ $t('sites.conf.limitconn.enable') }}</div>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ $t('sites.conf.limitconn.enableDesc') }}</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.enable" type="checkbox"> {{ $t('common.enabled') }}
        </label>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.limitconn.connPerIP') }}</div>
      <FaInput v-model="edit.connPerIP" type="number" placeholder="20" class="w-40" />
      <p class="mt-2 text-xs text-muted-foreground">{{ $t('sites.conf.limitconn.connDesc') }}</p>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
