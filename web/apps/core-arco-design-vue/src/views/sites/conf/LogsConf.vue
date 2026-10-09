<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import apiSite from '@/api/modules/site'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()

const logsType = ref<'access' | 'error'>('access')
const logsContent = ref('')
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    logsContent.value = await apiSite.siteLogs(props.site.id, logsType.value, 200) || i18n.global.t('sites.conf.logs.empty')
  }
  catch (e: any) {
    logsContent.value = i18n.global.t('sites.conf.logs.readFailed', { msg: e?.message || '' })
  }
  finally {
    loading.value = false
  }
}

function switchType(t: 'access' | 'error') {
  logsType.value = t
  load()
}

onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center gap-2">
      <button
        v-for="t in [{ v: 'access', l: $t('sites.conf.logs.access') }, { v: 'error', l: $t('sites.conf.logs.error') }]"
        :key="t.v"
        type="button"
        class="cursor-pointer rounded-md border px-3 py-1 text-sm transition-colors"
        :class="logsType === t.v ? 'border-primary bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-accent/50'"
        @click="switchType(t.v as 'access' | 'error')"
      >
        {{ t.l }}
      </button>
      <FaButton variant="outline" size="sm" class="ml-auto" :loading="loading" @click="load">
        <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('common.refresh') }}
      </FaButton>
    </div>
    <pre class="h-96 overflow-auto rounded-lg border bg-muted/40 p-3 font-mono text-xs leading-relaxed">{{ logsContent }}</pre>
    <p class="text-xs text-muted-foreground">
      {{ $t('sites.conf.logs.hint') }}
    </p>
  </div>
</template>
