<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteLoadBalance } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteLoadBalance | null>(null)
const edit = ref<SiteLoadBalance>({ enable: false, strategy: 'round-robin', upstreams: [] })
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteExtraApi.getLoadBalance(props.site.id)
    edit.value = JSON.parse(JSON.stringify(conf.value))
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.loadbalance.loadFailed'), { description: e?.message })
  }
}

function addUpstream() {
  edit.value.upstreams.push({ address: '', weight: 1 })
}

function removeUpstream(i: number) {
  edit.value.upstreams.splice(i, 1)
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteExtraApi.updateLoadBalance(props.site.id, edit.value)
    edit.value = JSON.parse(JSON.stringify(conf.value))
    toast.success(i18n.global.t('sites.conf.loadbalance.saved'))
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
  if (!edit.value.upstreams.length) {
    addUpstream()
    addUpstream()
  }
})
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-sm font-medium">{{ $t('sites.conf.loadbalance.enable') }}</div>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ $t('sites.conf.loadbalance.enableDesc') }}</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.enable" type="checkbox"> {{ $t('common.enabled') }}
        </label>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.loadbalance.strategy') }}</div>
      <select v-model="edit.strategy" class="mb-4 h-9 rounded-md border bg-background px-2 text-sm outline-none md:w-64">
        <option value="round-robin">{{ $t('sites.conf.loadbalance.stratRR') }}</option>
        <option value="least_conn">{{ $t('sites.conf.loadbalance.stratLC') }}</option>
        <option value="ip_hash">{{ $t('sites.conf.loadbalance.stratIPHash') }}</option>
      </select>

      <div class="mb-3 flex items-center justify-between">
        <span class="text-sm font-medium">{{ $t('sites.conf.loadbalance.upstreams') }}</span>
        <FaButton variant="outline" size="sm" @click="addUpstream">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('sites.conf.loadbalance.addNode') }}
        </FaButton>
      </div>
      <div class="mb-2 hidden gap-2 text-xs text-muted-foreground md:flex">
        <span class="flex-1">{{ $t('sites.conf.loadbalance.colAddress') }}</span>
        <span class="w-24">{{ $t('sites.conf.loadbalance.colWeight') }}</span>
        <span class="w-8" />
      </div>
      <div v-for="(u, i) in edit.upstreams" :key="i" class="mb-2 flex items-center gap-2">
        <FaInput v-model="u.address" placeholder="172.17.0.1:3000" class="flex-1" />
        <FaInput v-model="u.weight" type="number" placeholder="1" class="w-24" />
        <FaButton variant="ghost" size="icon-sm" class="text-red-500!" :title="$t('common.delete')" @click="removeUpstream(i)">
          <FaIcon name="i-lucide:trash-2" class="text-sm" />
        </FaButton>
      </div>
      <p class="mt-2 text-xs text-muted-foreground">
        {{ $t('sites.conf.loadbalance.weightHint') }}
      </p>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
