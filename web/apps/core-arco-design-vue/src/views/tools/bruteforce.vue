<script setup lang="ts">
import { fail2banApi } from '@/api/modules/firewall'
// M53 暴力破解防护（工具分组）：SSH 失败登录聚合 + 手动封禁/解封 + fail2ban 服务状态
import api from '@/api/modules/ssh'
import { i18n } from '@/locales'

defineOptions({
  name: 'ToolsBruteforce',
})

const toast = useFaToast()

const attempts = ref<{ ip: string, count: number, banned: boolean }[]>([])
const attemptsBusy = ref(false)
const banningIp = ref('')

async function loadAttempts() {
  attemptsBusy.value = true
  try {
    attempts.value = await api.attempts('local', 50)
  }
  catch {}
  finally {
    attemptsBusy.value = false
  }
}

async function banIp(ip: string) {
  banningIp.value = ip
  try {
    await fail2banApi.ban('sshd', ip)
    toast.success(i18n.global.t('tools.brute.banned', { ip }))
    await loadAttempts()
    await loadF2b()
  }
  catch (e: any) {
    toast.error(i18n.global.t('tools.brute.banFail'), { description: e?.message })
  }
  finally {
    banningIp.value = ''
  }
}

// ---- fail2ban 服务状态 ----
const f2b = ref<{ available: boolean, hint?: string, jails?: { name: string, banned: string[], total: number }[] }>()
const f2bBusy = ref(false)
const unbanning = ref('')

async function loadF2b() {
  f2bBusy.value = true
  try {
    f2b.value = await fail2banApi.status()
  }
  catch {}
  finally {
    f2bBusy.value = false
  }
}

// ---- fail2ban 一键安装 ----
const f2bInstalling = ref(false)

async function installF2b() {
  f2bInstalling.value = true
  toast.info(i18n.global.t('tools.brute.f2bInstalling'), { duration: 3000 })
  try {
    f2b.value = await fail2banApi.install()
    toast.success(i18n.global.t('tools.brute.f2bInstallDone'))
  }
  catch (e: any) {
    toast.error(i18n.global.t('tools.brute.f2bInstallFail'), { description: e?.message })
  }
  finally {
    f2bInstalling.value = false
  }
}

async function unbanIp(jail: string, ip: string) {
  unbanning.value = `${jail}:${ip}`
  try {
    await fail2banApi.unban(jail, ip)
    toast.success(i18n.global.t('tools.brute.unbanned', { ip }))
    await loadF2b()
  }
  catch (e: any) {
    toast.error(i18n.global.t('tools.brute.unbanFail'), { description: e?.message })
  }
  finally {
    unbanning.value = ''
  }
}

onMounted(() => {
  loadAttempts()
  loadF2b()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex gap-2 items-center">
          <FaIcon name="i-lucide:shield-ban" :size="22" />
          <span>{{ $t('tools.brute.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('tools.brute.desc') }}</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
		<div class="flex justify-end mb-3">
			<YdHostNodeSelect />
		</div>
      <div class="gap-4 grid grid-cols-1 xl:grid-cols-5">
        <!-- SSH 失败登录 -->
        <section class="p-4 border rounded-lg bg-background xl:col-span-3">
          <div class="flex flex-wrap gap-2 items-center">
            <span class="font-medium">{{ $t('tools.brute.attemptsTitle') }}</span>
            <span class="text-xs text-muted-foreground">{{ $t('tools.brute.attemptsDesc') }}</span>
            <FaButton variant="ghost" size="sm" class="ml-auto" :loading="attemptsBusy" @click="loadAttempts">
              {{ $t('common.refresh') }}
            </FaButton>
          </div>
          <div v-if="attempts.length" class="mt-3 space-y-1">
            <div v-for="a in attempts" :key="a.ip" class="text-xs px-3 py-1.5 border rounded-md flex gap-3 items-center">
              <span class="font-mono w-40">{{ a.ip }}</span>
              <span class="text-red-500">{{ $t('tools.brute.failCount', { count: a.count }) }}</span>
              <span v-if="a.banned" class="text-emerald-600 px-2 py-0.5 rounded-full bg-emerald-500/10">{{ $t('tools.brute.bannedLabel') }}</span>
              <FaButton v-else variant="outline" size="sm" class="ml-auto" :loading="banningIp === a.ip" @click="banIp(a.ip)">
                {{ $t('tools.brute.ban') }}
              </FaButton>
            </div>
          </div>
          <div v-else class="text-xs text-muted-foreground mt-3">
            {{ $t('tools.brute.noAttempts') }}
          </div>
        </section>

        <!-- fail2ban 服务 -->
        <section class="p-4 border rounded-lg bg-background xl:col-span-2">
          <div class="flex flex-wrap gap-2 items-center">
            <span class="font-medium">fail2ban</span>
            <span class="text-xs px-2 py-0.5 rounded-full" :class="f2b?.available ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ f2b?.available ? $t('tools.brute.f2bRunning') : $t('tools.brute.f2bUnavailable') }}
            </span>
            <FaButton variant="ghost" size="sm" class="ml-auto" :loading="f2bBusy" @click="loadF2b">
              {{ $t('common.refresh') }}
            </FaButton>
          </div>
          <div v-if="f2b && !f2b.available" class="mt-2 flex gap-2 items-center">
            <span class="text-xs text-muted-foreground">{{ f2b.hint }}</span>
            <FaButton variant="outline" size="sm" class="ml-auto" :loading="f2bInstalling" @click="installF2b">
              {{ $t('tools.brute.f2bInstall') }}
            </FaButton>
          </div>
          <div v-if="f2b?.jails?.length" class="mt-3 space-y-2">
            <div v-for="j in f2b.jails" :key="j.name" class="text-xs px-3 py-2 border rounded-md">
              <div class="flex gap-2 items-center">
                <span class="font-medium font-mono">{{ j.name }}</span>
                <span class="text-muted-foreground">{{ $t('tools.brute.bannedTotal', { total: j.total }) }}</span>
              </div>
              <div v-if="j.banned.length" class="mt-1 space-y-1">
                <div v-for="ip in j.banned" :key="ip" class="flex gap-2 items-center">
                  <span class="font-mono">{{ ip }}</span>
                  <FaButton variant="ghost" size="sm" class="ml-auto" :loading="unbanning === `${j.name}:${ip}`" @click="unbanIp(j.name, ip)">
                    {{ $t('tools.brute.unban') }}
                  </FaButton>
                </div>
              </div>
            </div>
          </div>
          <div v-else-if="f2b?.available" class="text-xs text-muted-foreground mt-3">
            {{ $t('tools.brute.noJails') }}
          </div>
        </section>
      </div>
    </FaPageMain>
  </div>
</template>
