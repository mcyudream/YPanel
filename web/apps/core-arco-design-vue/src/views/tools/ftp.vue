<script setup lang="ts">
import type { FtpStatus } from '@/api/modules/ftp'
// M53 FTP 服务管理（vsftpd，工具分组）——自 system.vue 迁出
import ftpApi from '@/api/modules/ftp'
import { i18n } from '@/locales'

defineOptions({
  name: 'ToolsFtp',
})

const toast = useFaToast()

const ftp = ref<FtpStatus | null>(null)
const ftpBusy = ref(false)
const ftpPort = ref(21)
const ftpPasv = ref('40000-40100')

async function loadFtp() {
  try {
    ftp.value = await ftpApi.status()
    ftpPort.value = ftp.value?.port || 21
    ftpPasv.value = `${ftp.value?.pasvMin || 40000}-${ftp.value?.pasvMax || 40100}`
  }
  catch {}
}

async function ftpInstall() {
  ftpBusy.value = true
  toast.info(i18n.global.t('tools.ftp.installing'))
  try {
    await ftpApi.install()
    toast.success(i18n.global.t('tools.ftp.installDone'))
    await loadFtp()
  }
  catch (e: any) {
    toast.error(i18n.global.t('tools.ftp.installFail'), { description: e?.message })
  }
  finally {
    ftpBusy.value = false
  }
}

async function ftpPower(action: 'start' | 'stop' | 'restart') {
  ftpBusy.value = true
  try {
    await ftpApi.power(action)
    toast.success(i18n.global.t('tools.ftp.powerDone', { action }))
    await loadFtp()
  }
  catch (e: any) {
    toast.error(i18n.global.t('common.opFailed'), { description: e?.message })
  }
  finally {
    ftpBusy.value = false
  }
}

async function ftpSetPort() {
  const pm = ftpPasv.value.split('-')
  ftpBusy.value = true
  try {
    await ftpApi.setPort(Number(ftpPort.value), Number(pm[0]) || 40000, Number(pm[1]) || 40100)
    toast.success(i18n.global.t('tools.ftp.portDone'))
    await loadFtp()
  }
  catch (e: any) {
    toast.error(i18n.global.t('common.opFailed'), { description: e?.message })
  }
  finally {
    ftpBusy.value = false
  }
}

onMounted(loadFtp)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex gap-2 items-center">
          <FaIcon name="i-lucide:folder-sync" :size="22" />
          <span>{{ $t('tools.ftp.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('tools.ftp.desc') }}</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <section class="mx-auto p-4 border rounded-lg bg-background max-w-3xl">
        <div class="flex flex-wrap gap-2 items-center">
          <span class="font-medium">vsftpd</span>
          <span class="text-xs px-2 py-0.5 rounded-full" :class="ftp?.running ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
            {{ ftp?.running ? `${$t('tools.ftp.running')} · ${$t('tools.ftp.portLabel')} ${ftp?.port || 21}` : ftp?.installed ? $t('tools.ftp.installedStopped') : $t('tools.ftp.notInstalled') }}
          </span>
        </div>
        <div class="mt-3 flex flex-wrap gap-2 items-center">
          <template v-if="ftp?.installed">
            <FaButton variant="outline" size="sm" :loading="ftpBusy" @click="ftpPower('start')">
              {{ $t('tools.ftp.start') }}
            </FaButton>
            <FaButton variant="outline" size="sm" :disabled="ftpBusy" @click="ftpPower('restart')">
              {{ $t('tools.ftp.restart') }}
            </FaButton>
            <FaButton variant="outline" size="sm" class="text-red-500!" :disabled="ftpBusy || !ftp?.running" @click="ftpPower('stop')">
              {{ $t('tools.ftp.stop') }}
            </FaButton>
          </template>
          <FaButton v-else size="sm" :loading="ftpBusy" @click="ftpInstall">
            {{ $t('tools.ftp.install') }}
          </FaButton>
        </div>
        <template v-if="ftp?.installed">
          <div class="text-sm mt-4 flex flex-wrap gap-2 items-center">
            <span class="text-muted-foreground">{{ $t('tools.ftp.portLabel') }}</span>
            <FaInput v-model="ftpPort" type="number" class="w-24" />
            <span class="text-muted-foreground">{{ $t('tools.ftp.pasvRange') }}</span>
            <FaInput v-model="ftpPasv" class="w-32" />
            <FaButton size="sm" :loading="ftpBusy" @click="ftpSetPort">
              {{ $t('tools.ftp.applyPort') }}
            </FaButton>
          </div>
          <div class="text-xs text-muted-foreground mt-2">
            {{ $t('tools.ftp.portHint') }}
          </div>
        </template>
      </section>
    </FaPageMain>
  </div>
</template>
