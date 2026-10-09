<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteCORS } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteCORS | null>(null)
const edit = ref<SiteCORS>({ enable: false, allowOrigins: [], allowMethods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'], allowHeaders: ['Content-Type', 'Authorization'], allowCredentials: false, maxAge: 86400 })
const originsRaw = ref('')
const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS']
const saving = ref(false)

const allowHeadersStr = computed({
  get: () => (edit.value.allowHeaders || []).join(', '),
  set: (v: string) => {
    edit.value.allowHeaders = v.split(',').map(x => x.trim()).filter(Boolean)
  },
})

async function load() {
  try {
    conf.value = await siteExtraApi.getCORS(props.site.id)
    edit.value = { ...conf.value }
    originsRaw.value = (conf.value.allowOrigins || []).join('\n')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.cors.loadFailed'), { description: e?.message })
  }
}

async function save() {
  saving.value = true
  try {
    edit.value.allowOrigins = originsRaw.value.split('\n').map(x => x.trim()).filter(Boolean)
    conf.value = await siteExtraApi.updateCORS(props.site.id, edit.value)
    edit.value = { ...conf.value }
    originsRaw.value = (conf.value.allowOrigins || []).join('\n')
    toast.success(i18n.global.t('sites.conf.cors.saved'))
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
          <div class="text-sm font-medium">{{ $t('sites.conf.cors.enable') }}</div>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ $t('sites.conf.cors.enableDesc') }}</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.enable" type="checkbox"> {{ $t('common.enabled') }}
        </label>
      </div>
    </div>

    <div class="space-y-4 rounded-lg border p-4">
      <div>
        <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.cors.origins') }}</div>
        <p class="mb-2 text-xs text-muted-foreground">{{ $t('sites.conf.cors.originsDesc') }}</p>
        <textarea v-model="originsRaw" rows="3" class="w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary" placeholder="*&#10;https://app.example.com" />
      </div>
      <div>
        <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.cors.methods') }}</div>
        <div class="flex flex-wrap gap-3 text-sm">
          <label v-for="m in methods" :key="m" class="flex cursor-pointer items-center gap-1.5">
            <input v-model="edit.allowMethods" type="checkbox" :value="m"> {{ m }}
          </label>
        </div>
      </div>
      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.cors.headers') }}</div>
          <FaInput v-model="allowHeadersStr" placeholder="Content-Type, Authorization" class="w-full" />
        </div>
        <div>
          <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.cors.maxAge') }}</div>
          <FaInput v-model="edit.maxAge" type="number" class="w-full" />
        </div>
      </div>
      <label class="flex cursor-pointer items-center gap-2 text-sm">
        <input v-model="edit.allowCredentials" type="checkbox"> {{ $t('sites.conf.cors.credentials') }}
      </label>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
