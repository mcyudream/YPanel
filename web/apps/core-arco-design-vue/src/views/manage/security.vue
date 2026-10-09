<script setup lang="ts">
import type { SecuritySettings } from '@/api/modules/security'
import { securityApi } from '@/api/modules/security'
import { i18n } from '@/locales'

defineOptions({
  name: 'ManageSecurity',
})

const toast = useFaToast()

const loading = ref(false)
const saving = ref(false)
const data = ref<SecuritySettings>({
  twoFaEnabled: false,
  safeEntry: '',
  allowedIps: '',
  sessionHours: 24,
  minPasswordLen: 0,
  dangerLock: false,
})

async function load() {
  loading.value = true
  try {
    data.value = await securityApi.get()
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.security.loadFail'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    if (!data.value.safeEntry || data.value.safeEntry.replace(/^\/+|\/+$/g, '').length < 6) {
      toast.error('安全入口（强制开启，≥6 位）强制开启且长度不得小于 6 位')
      return
    }
    data.value = await securityApi.update({
      safeEntry: data.value.safeEntry.trim(),
      allowedIps: data.value.allowedIps.trim(),
      sessionHours: Number(data.value.sessionHours) || 24,
      minPasswordLen: Number(data.value.minPasswordLen) || 0,
      dangerLock: data.value.dangerLock ?? false,
    })
    toast.success(i18n.global.t('manage.security.saved'))
    if (data.value.safeEntry) {
      localStorage.setItem('login_entry', data.value.safeEntry)
    }
    else {
      localStorage.removeItem('login_entry')
    }
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

// ---- 2FA ----
const twofaModal = ref(false)
const twofaSetup = ref<{ secret: string, otpauthUri: string } | null>(null)
const twofaBusy = ref(false)

async function start2FA() {
  twofaBusy.value = true
  try {
    twofaSetup.value = await securityApi.twoFASetup()
    twofaModal.value = true
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.security.setupFail'), { description: e?.message })
  }
  finally {
    twofaBusy.value = false
  }
}

async function disable2FA() {
  twofaBusy.value = true
  try {
    await securityApi.twoFADisable()
    toast.success(i18n.global.t('manage.security.disabled'))
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('manage.opFailed'), { description: e?.message })
  }
  finally {
    twofaBusy.value = false
  }
}

function copyText(text: string) {
  navigator.clipboard.writeText(text).then(() => toast.success(i18n.global.t('common.copied')))
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="shield" :size="24" />
          <span>{{ $t('manage.security.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('manage.security.desc') }}</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="mx-auto max-w-3xl space-y-6">
        <!-- 2FA -->
        <div class="rounded-lg border p-5">
          <div class="flex items-center justify-between">
            <div>
              <div class="flex items-center gap-2 text-sm font-medium">
                <FaIcon name="i-lucide:key-round" class="text-base" />
                {{ $t('manage.security.twofa') }}
              </div>
              <p class="mt-1 text-xs text-muted-foreground">
                {{ $t('manage.security.twofaDesc') }}
              </p>
            </div>
            <span
              class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs"
              :class="data.twoFaEnabled ? 'text-emerald-600 bg-emerald-500/10' : 'text-muted-foreground bg-muted'"
            >
              <span class="inline-block size-1.5 rounded-full bg-current" :class="data.twoFaEnabled ? 'animate-pulse' : ''" />
              {{ data.twoFaEnabled ? $t('manage.security.twofaOn') : $t('manage.security.twofaOff') }}
            </span>
          </div>
          <div class="mt-3 flex gap-2">
            <FaButton v-if="!data.twoFaEnabled" size="sm" :loading="twofaBusy" @click="start2FA">
              {{ $t('manage.security.twofaEnable') }}
            </FaButton>
            <FaButton v-else variant="outline" size="sm" :loading="twofaBusy" @click="disable2FA">
              {{ $t('manage.security.twofaDisable') }}
            </FaButton>
          </div>
        </div>

        <!-- 入口/白名单/超时 -->
        <div class="space-y-5 rounded-lg border p-5">
          <div>
            <div class="flex items-center gap-2 text-sm font-medium">
              <FaIcon name="i-lucide:door-closed" class="text-base" />
              {{ $t('manage.security.entry') }}
            </div>
            <p class="mt-1 text-xs text-muted-foreground">
              {{ $t('manage.security.entryDescPre') }} <code>{{ $t('manage.security.entryExample') }}</code> {{ $t('manage.security.entryDescPost') }}
            </p>
            <FaInput v-model="data.safeEntry" :placeholder="$t('manage.security.entryPlaceholder')" class="mt-2 w-full" />
            <div class="mt-1 text-xs text-amber-600">安全入口强制开启（≥6 位）：保存后仅可通过 http://服务器地址:端口/{{ data.safeEntry }} 访问面板，根路径显示 404；请收藏该入口链接</div>
          </div>
          <div>
            <div class="flex items-center gap-2 text-sm font-medium">
              <FaIcon name="i-lucide:network" class="text-base" />
              {{ $t('manage.security.ipWhitelist') }}
            </div>
            <p class="mt-1 text-xs text-muted-foreground">
              {{ $t('manage.security.ipDesc') }}
            </p>
            <FaInput v-model="data.allowedIps" :placeholder="$t('manage.security.ipPlaceholder')" class="mt-2 w-full" />
          </div>
          <div>
            <div class="flex items-center gap-2 text-sm font-medium">
              <FaIcon name="i-lucide:timer" class="text-base" />
              {{ $t('manage.security.sessionTimeout') }}
            </div>
            <p class="mt-1 text-xs text-muted-foreground">
              {{ $t('manage.security.sessionDesc') }}
            </p>
            <FaInput v-model="data.sessionHours" type="number" class="mt-2 w-40" />
          <span class="text-xs text-muted-foreground">{{ $t('manage.security.minPasswordLen') }}</span>
            <FaInput v-model="data.minPasswordLen" type="number" class="mt-2 w-40" />
          </div>
          <div class="flex justify-end">
            <FaButton :loading="saving" @click="save">
              {{ $t('manage.security.saveSettings') }}
            </FaButton>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 2FA 绑定弹窗 -->
    <FaModal
      v-model="twofaModal"
      :title="$t('manage.security.bindTitle')"
      class="max-w-xl!"
      :destroy-on-close="true"
    >
      <div v-if="twofaSetup" class="space-y-4 text-sm">
        <div class="rounded-md border border-amber-300 bg-amber-50 p-3 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
          {{ $t('manage.security.bindWarn') }}
        </div>
        <div>
          <div class="mb-1 text-xs text-muted-foreground">
            {{ $t('manage.security.method1') }}
          </div>
          <div class="flex items-center gap-2">
            <code class="flex-1 truncate rounded bg-muted px-2 py-1.5 font-mono text-xs">{{ twofaSetup.otpauthUri }}</code>
            <FaButton variant="outline" size="icon-sm" :title="$t('manage.security.copyLink')" @click="copyText(twofaSetup.otpauthUri)">
              <FaIcon name="i-lucide:copy" class="text-sm" />
            </FaButton>
          </div>
        </div>
        <div>
          <div class="mb-1 text-xs text-muted-foreground">
            {{ $t('manage.security.method2') }}
          </div>
          <div class="flex items-center gap-2">
            <code class="flex-1 rounded bg-muted px-2 py-1.5 font-mono text-sm tracking-widest">{{ twofaSetup.secret }}</code>
            <FaButton variant="outline" size="icon-sm" :title="$t('manage.security.copySecret')" @click="copyText(twofaSetup.secret)">
              <FaIcon name="i-lucide:copy" class="text-sm" />
            </FaButton>
          </div>
        </div>
      </div>
      <template #footer>
        <FaButton @click="twofaModal = false">
          {{ $t('manage.security.keySaved') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
