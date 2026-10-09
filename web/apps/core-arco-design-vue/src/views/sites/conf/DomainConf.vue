<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteDomainConf } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const conf = ref<SiteDomainConf | null>(null)
const loading = ref(false)
const saving = ref(false)
// 编辑副本：每行一个域名；主域名切换走本地交换预览，保存时统一提交
const rows = ref<string[]>([])
const newDomain = ref('')
const localPrimary = ref('')

async function load() {
  loading.value = true
  try {
    conf.value = await siteConfApi.getDomain(props.site.id)
    rows.value = [...(conf.value.domains || [])]
    localPrimary.value = conf.value.domain
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.domain.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

function add() {
  const d = newDomain.value.trim().toLowerCase()
  if (!d) {
    return
  }
  if (d === localPrimary.value) {
    toast.error(i18n.global.t('sites.conf.domain.dupPrimary'))
    return
  }
  if (rows.value.includes(d)) {
    toast.error(i18n.global.t('sites.conf.domain.dupExists'))
    return
  }
  rows.value.push(d)
  newDomain.value = ''
}

function removeAt(i: number) {
  rows.value.splice(i, 1)
}

// 把附加域名提升为主域名：本地交换预览（原主域名落到该行位置），保存后生效
function makePrimary(i: number) {
  const d = rows.value[i]
  if (!d || d === localPrimary.value) {
    return
  }
  rows.value[i] = localPrimary.value
  localPrimary.value = d
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteConfApi.updateDomain(props.site.id, rows.value, localPrimary.value)
    rows.value = [...(conf.value.domains || [])]
    localPrimary.value = conf.value.domain
    if (conf.value.certNotice) {
      toast.warning(conf.value.certNotice)
    }
    toast.success(i18n.global.t('sites.conf.domain.saved'))
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
  ? (JSON.stringify(rows.value) !== JSON.stringify(conf.value.domains || []) || localPrimary.value !== conf.value.domain)
  : false)

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="mb-2 text-sm font-medium">{{ $t('sites.conf.domain.primary') }}</div>
      <div class="flex items-center gap-2">
        <code class="rounded bg-muted px-2 py-1 font-mono text-[13px]">{{ localPrimary }}</code>
        <span class="text-xs text-muted-foreground">{{ $t('sites.conf.domain.primaryTip', { cert: conf?.certDomain || $t('sites.shared.unset') }) }}</span>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-2 flex items-center justify-between">
        <span class="text-sm font-medium">{{ $t('sites.conf.domain.extra') }}</span>
        <span class="text-xs text-muted-foreground">{{ $t('sites.conf.domain.extraTip') }}</span>
      </div>
      <div class="mb-3 flex items-center gap-2">
        <FaInput v-model="newDomain" :placeholder="$t('sites.conf.domain.placeholder')" class="flex-1" @keyup.enter="add" />
        <FaButton variant="outline" size="sm" @click="add">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('common.add') }}
        </FaButton>
      </div>
      <div v-if="!rows.length" class="py-4 text-center text-sm text-muted-foreground">
        {{ $t('sites.conf.domain.none') }}
      </div>
      <div v-for="(d, i) in rows" :key="d" class="flex items-center justify-between border-b py-2 text-sm last:border-b-0">
        <code class="font-mono text-[13px]">{{ d }}</code>
        <div class="flex items-center gap-1">
          <FaButton variant="ghost" size="sm" class="text-muted-foreground" :title="$t('sites.conf.domain.makePrimary')" @click="makePrimary(i)">
            <FaIcon name="i-lucide:crown" class="mr-1 text-sm" /> {{ $t('sites.conf.domain.makePrimary') }}
          </FaButton>
          <FaButton variant="ghost" size="icon-sm" class="text-red-500!" :title="$t('common.remove')" @click="removeAt(i)">
            <FaIcon name="i-lucide:x" class="text-sm" />
          </FaButton>
        </div>
      </div>
      <div class="mt-4 flex justify-end">
        <FaButton :loading="saving" :disabled="!dirty" @click="save">
          {{ $t('sites.shared.saveApply') }}
        </FaButton>
      </div>
    </div>
  </div>
</template>
