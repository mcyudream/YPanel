<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteAuthBasic } from '@/api/modules/siteconf'
import { siteExtraApi } from '@/api/modules/siteconf'
import { i18n } from '@/locales'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteAuthBasic | null>(null)
const edit = ref<SiteAuthBasic>({ enable: false, realm: 'Restricted', users: [] })
const saving = ref(false)

async function load() {
  try {
    conf.value = await siteExtraApi.getAuthBasic(props.site.id)
    edit.value = JSON.parse(JSON.stringify(conf.value))
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.authbasic.loadFailed'), { description: e?.message })
  }
}

function addUser() {
  edit.value.users.push({ user: '', password: '' })
}

function removeUser(i: number) {
  edit.value.users.splice(i, 1)
}

async function save() {
  saving.value = true
  try {
    conf.value = await siteExtraApi.updateAuthBasic(props.site.id, edit.value)
    edit.value = JSON.parse(JSON.stringify(conf.value))
    toast.success(i18n.global.t('sites.conf.authbasic.saved'))
    emit('changed')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.shared.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

function toggleEnable() {
  if (edit.value.enable && !edit.value.users.length) {
    addUser()
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
          <div class="text-sm font-medium">{{ $t('sites.conf.authbasic.enable') }}</div>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ $t('sites.conf.authbasic.enableDesc') }}</p>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="edit.enable" type="checkbox" @change="toggleEnable"> {{ $t('common.enabled') }}
        </label>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-3 flex items-center justify-between">
        <span class="text-sm font-medium">{{ $t('sites.conf.authbasic.users') }}</span>
        <FaButton variant="outline" size="sm" @click="addUser">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('sites.conf.authbasic.addUser') }}
        </FaButton>
      </div>
      <div class="mb-2 hidden gap-2 text-xs text-muted-foreground md:flex">
        <span class="w-44">{{ $t('sites.conf.authbasic.colUser') }}</span>
        <span class="flex-1">{{ $t('sites.conf.authbasic.colPassword') }}</span>
        <span class="w-8" />
      </div>
      <div v-for="(u, i) in edit.users" :key="i" class="mb-2 flex items-center gap-2">
        <FaInput v-model="u.user" placeholder="user" class="w-44" />
        <FaInput v-model="u.password" type="password" placeholder="••••••" class="flex-1" />
        <FaButton variant="ghost" size="icon-sm" class="text-red-500!" :title="$t('common.delete')" @click="removeUser(i)">
          <FaIcon name="i-lucide:trash-2" class="text-sm" />
        </FaButton>
      </div>
      <div v-if="!edit.users.length" class="py-4 text-center text-sm text-muted-foreground">
        {{ $t('sites.conf.authbasic.none') }}
      </div>
      <label v-if="edit.users.length" class="mt-2 flex items-center gap-3 text-sm">
        <span class="w-24 shrink-0 text-muted-foreground">{{ $t('sites.conf.authbasic.realm') }}</span>
        <FaInput v-model="edit.realm" placeholder="Restricted" class="w-56" />
      </label>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        {{ $t('sites.shared.saveApply') }}
      </FaButton>
    </div>
  </div>
</template>
