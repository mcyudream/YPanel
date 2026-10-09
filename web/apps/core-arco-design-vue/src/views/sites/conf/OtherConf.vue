<script setup lang="ts">
import type { SiteItem, SiteExtConfig } from '@/api/modules/site'
import { extApi } from '@/api/modules/site'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteExtConfig | null>(null)
const locs = ref<{ comment: string, content: string }[]>([])
const saving = ref(false)

async function load() {
  try {
    conf.value = await extApi.getExt(props.site.id)
    locs.value = JSON.parse(JSON.stringify(conf.value.customLocations || []))
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.other.loadFailed'), { description: e?.message })
  }
}

function addLoc() {
  locs.value.push({ comment: '', content: 'location /path {\n    return 200 "ok";\n}' })
}

function removeLoc(i: number) {
  locs.value.splice(i, 1)
}

async function save() {
  saving.value = true
  try {
    const next = { ...(conf.value as SiteExtConfig), customLocations: locs.value }
    await extApi.updateExt(props.site.id, next)
    conf.value = next
    toast.success(i18n.global.t('sites.conf.other.saved'))
    emit('changed')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.shared.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

const dirty = computed(() => conf.value ? JSON.stringify(locs.value) !== JSON.stringify(conf.value.customLocations || []) : false)

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="mb-3 flex items-center justify-between">
        <div>
          <span class="text-sm font-medium">{{ $t('sites.conf.other.title') }}</span>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ $t('sites.conf.other.desc') }}</p>
        </div>
        <FaButton variant="outline" size="sm" @click="addLoc">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('sites.conf.other.addLoc') }}
        </FaButton>
      </div>
      <div v-if="!locs.length" class="py-6 text-center text-sm text-muted-foreground">
        {{ $t('sites.conf.other.none') }}
      </div>
      <div v-for="(l, i) in locs" :key="i" class="mb-4 space-y-2 rounded-md border p-3">
        <div class="flex items-center gap-2">
          <FaInput v-model="l.comment" :placeholder="$t('sites.conf.other.commentPlaceholder')" class="flex-1" />
          <FaButton variant="ghost" size="icon-sm" class="text-red-500!" :title="$t('common.delete')" @click="removeLoc(i)">
            <FaIcon name="i-lucide:trash-2" class="text-sm" />
          </FaButton>
        </div>
        <textarea
          v-model="l.content"
          class="h-28 w-full resize-y rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary"
          placeholder="location /healthz { return 200 'ok'; }"
          spellcheck="false"
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
