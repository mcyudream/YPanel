<script setup lang="ts">
import type { RewriteTemplate, SiteItem } from '@/api/modules/site'
import { extApi } from '@/api/modules/site'
import type { SiteRewriteConf } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const conf = ref<SiteRewriteConf | null>(null)
const edit = ref<SiteRewriteConf>({ rewriteName: '', rewriteContent: '' })
const templates = ref<RewriteTemplate[]>([])
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    conf.value = await siteConfApi.getRewrite(props.site.id)
    edit.value = { ...conf.value }
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.rewrite.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function loadTemplates() {
  try {
    templates.value = await extApi.rewriteTemplates()
  }
  catch {}
}

function applyTemplate(t: RewriteTemplate) {
  edit.value.rewriteName = t.name
  edit.value.rewriteContent = t.content
}

function clearRewrite() {
  edit.value.rewriteName = ''
  edit.value.rewriteContent = ''
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteConfApi.updateRewrite(props.site.id, edit.value)
    edit.value = { ...conf.value }
    toast.success(i18n.global.t('sites.conf.rewrite.saved'))
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

onMounted(() => {
  load()
  loadTemplates()
})
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="mb-3 flex flex-wrap items-center gap-1.5">
        <span class="mr-1 text-sm font-medium">{{ $t('sites.conf.rewrite.templates') }}</span>
        <button
          v-for="t in templates"
          :key="t.name"
          type="button"
          class="cursor-pointer rounded-full border px-2.5 py-0.5 text-xs transition-colors"
          :class="edit.rewriteName === t.name ? 'border-primary bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-accent/50'"
          @click="applyTemplate(t)"
        >
          {{ t.name }}
        </button>
        <button
          type="button"
          class="ml-auto cursor-pointer rounded-full border px-2.5 py-0.5 text-xs text-red-500 transition-colors hover:bg-red-500/10"
          @click="clearRewrite"
        >
          {{ $t('common.clear') }}
        </button>
      </div>
      <textarea
        v-model="edit.rewriteContent"
        class="h-64 w-full resize-y rounded-md border border-input bg-background p-3 font-mono text-[13px] leading-relaxed outline-none focus:ring-1 focus:ring-primary"
        placeholder="location / { try_files $uri $uri/ /index.php?$query_string; }"
        spellcheck="false"
      />
      <p class="mt-2 text-xs text-muted-foreground">
        {{ $t('sites.conf.rewrite.hint') }}
      </p>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
