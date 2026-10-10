<script setup lang="ts">
import type { FirewallStatus } from '@/api/modules/firewall'
import apiFW, { fail2banApi } from '@/api/modules/firewall'
import { i18n } from '@/locales'

defineOptions({
  name: 'FirewallIndex',
})

const fw = ref<FirewallStatus>()
const loading = ref(false)
const allowVisible = ref(false)
const allowForm = ref({ port: '', proto: 'tcp' })

// fail2ban
const f2b = ref<{ available: boolean, hint?: string, jails?: { name: string, banned: string[], total: number }[] }>()
const banModalVisible = ref(false)
const banForm = ref({ jail: '', ip: '' })

async function loadF2B() {
  try {
    f2b.value = await fail2banApi.status()
  }
  catch {}
}

async function doUnban(jail: string, ip: string) {
  try {
    await fail2banApi.unban(jail, ip)
    useFaToast().success(i18n.global.t('firewall.unbanned', { ip }))
    await loadF2B()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('firewall.unbanFailed'), { description: e?.message })
  }
}

function doBan() {
  if (!banForm.value.ip || !banForm.value.jail) {
    useFaToast().warning(i18n.global.t('firewall.requireJailIp'))
    return
  }
  fail2banApi.ban(banForm.value.jail, banForm.value.ip).then(() => {
    useFaToast().success(i18n.global.t('firewall.banned', { ip: banForm.value.ip }))
    banModalVisible.value = false
    loadF2B()
  }).catch((e: any) => useFaToast().error(i18n.global.t('firewall.banFailed'), { description: e?.message }))
}

async function load() {
  loading.value = true
  try {
    fw.value = await apiFW.status()
  }
  finally {
    loading.value = false
  }
}

async function doAllow() {
  try {
    await apiFW.allow(allowForm.value.port, allowForm.value.proto)
    useFaToast().success(i18n.global.t('firewall.allowed', { port: allowForm.value.port, proto: allowForm.value.proto }))
    allowVisible.value = false
    allowForm.value = { port: '', proto: 'tcp' }
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('firewall.allowFailed'), { description: e?.message })
  }
}

async function toggleRun() {
  if (!fw.value) {
    return
  }
  const modal = useFaModal()
  const enable = !(fw.value.enabled ?? fw.value.running)
  if (enable) {
    modal.confirm({
      title: i18n.global.t('firewall.enable'),
      content: i18n.global.t('firewall.enableConfirm'),
      onConfirm: async () => {
        try {
          await apiFW.enable()
          useFaToast().success(i18n.global.t('firewall.enabledDone'))
          await load()
        }
        catch (e: any) {
          useFaToast().error(i18n.global.t('firewall.opFailed'), { description: e?.message })
        }
      },
    })
  }
  else {
    try {
      await apiFW.disable()
      useFaToast().success(i18n.global.t('firewall.disabledDone'))
      await load()
    }
    catch (e: any) {
      useFaToast().error(i18n.global.t('firewall.opFailed'), { description: e?.message })
    }
  }
}

async function deleteRule(raw: string) {
  const m = raw.match(/\[\s*(\d+)\]/)
  if (!m) {
    return
  }
  confirmDelete(Number(m[1]), raw)
}

// firewalld 后端行内即端口号（后端 number=port）
function deletePort(p: { port: string, proto: string, number: number }) {
  confirmDelete(p.number, `${p.port}/${p.proto}`)
}

function confirmDelete(number: number, desc: string) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('firewall.deleteRuleTitle'),
    content: desc,
    onConfirm: async () => {
      try {
        await apiFW.deleteRule(number)
        useFaToast().success(i18n.global.t('firewall.deleted'))
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('firewall.deleteFailed'), { description: e?.message })
      }
    },
  })
}

onMounted(() => {
  load()
  loadF2B()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="shield" :size="24" />
          <span>{{ $t('firewall.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('firewall.desc') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="load">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" :class="loading ? 'animate-spin' : ''" /> {{ $t('common.refresh') }}
        </FaButton>
        <FaButton v-if="fw?.available" size="sm" :variant="(fw.enabled ?? fw.running) ? 'outline' : 'default'" @click="toggleRun">
          {{ (fw.enabled ?? fw.running) ? $t('firewall.disable') : $t('firewall.enable') }}
        </FaButton>
        <FaButton v-if="fw?.available" size="sm" @click="allowVisible = true">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('firewall.allowPort') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
		<div class="flex justify-end mb-3">
			<YdHostNodeSelect />
		</div>
      <!-- fail2ban 入侵防护 -->
      <div class="mb-4 rounded-lg border bg-background p-4">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2 text-sm font-medium">
            <YdMorphIcon name="siren" :size="16" />
            {{ $t('firewall.f2bTitle') }}
          </div>
          <FaButton variant="outline" size="sm" @click="banModalVisible = true">
            <FaIcon name="i-lucide:shield-ban" class="mr-1" /> {{ $t('firewall.banTitle') }}
          </FaButton>
        </div>
        <div v-if="f2b && !f2b.available" class="text-xs text-muted-foreground">{{ f2b.hint }}</div>
        <template v-else>
          <div v-for="j in f2b?.jails || []" :key="j.name" class="mb-2">
            <div class="mb-1 text-xs text-muted-foreground">
              {{ $t('firewall.jailLabel') }}<span class="font-mono">{{ j.name }}</span> · {{ $t('firewall.bannedCount', { n: j.total }) }}
            </div>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="ip in j.banned"
                :key="ip"
                class="inline-flex items-center gap-1 rounded-md bg-red-500/10 px-2 py-0.5 font-mono text-xs text-red-600"
              >
                {{ ip }}
                <button type="button" class="cursor-pointer opacity-60 hover:opacity-100" :title="$t('firewall.unban')" @click="doUnban(j.name, ip)">✕</button>
              </span>
              <span v-if="!j.banned.length" class="text-xs text-muted-foreground">{{ $t('firewall.noBanned') }}</span>
            </div>
          </div>
        </template>
      </div>

      <div v-if="fw && !fw.available" class="rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-700 dark:bg-amber-950/30 dark:text-amber-400">
        {{ fw.hint }}
      </div>
      <template v-else-if="fw">
        <div class="mb-4 flex items-center gap-2">
          <span class="text-sm text-muted-foreground">{{ $t('firewall.statusLabel') }}</span>
          <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="(fw.enabled ?? fw.running) ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
            <span class="inline-block size-1.5 rounded-full" :class="(fw.enabled ?? fw.running) ? 'animate-pulse bg-current' : 'bg-current'" />
            {{ (fw.enabled ?? fw.running) ? $t('firewall.running') : $t('firewall.stopped') }}
          </span>
        </div>
        <!-- firewalld 后端：端口列表 -->
        <div v-if="fw.backend === 'firewalld'" class="overflow-x-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('firewall.portCol') }}</th>
                <th class="px-3 py-2">proto</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!fw.ports?.length">
                <td colspan="3" class="px-3 py-8 text-center text-muted-foreground">{{ $t('firewall.noRules') }}</td>
              </tr>
              <tr v-for="p in fw.ports" :key="`${p.port}/${p.proto}`" class="border-t hover:bg-accent/30">
                <td class="px-3 py-2 font-mono text-xs">{{ p.port }}</td>
                <td class="px-3 py-2 font-mono text-xs">{{ p.proto }}</td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="outline" size="sm" @click="deletePort(p)">{{ $t('common.delete') }}</FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <!-- ufw 后端：编号规则列表 -->
        <div v-else class="overflow-x-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('firewall.ruleCol') }}</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!fw.rules?.length">
                <td colspan="2" class="px-3 py-8 text-center text-muted-foreground">{{ $t('firewall.noRules') }}</td>
              </tr>
              <tr v-for="r in fw.rules" :key="r.raw" class="border-t hover:bg-accent/30">
                <td class="px-3 py-2 font-mono text-xs">{{ r.raw }}</td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="outline" size="sm" @click="deleteRule(r.raw)">{{ $t('common.delete') }}</FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <!-- 站点托管放行（M50：由站点监听端口自动维护，站点侧回收，此处只读展示） -->
        <div v-if="fw.siteManaged?.length" class="mt-4 rounded-lg border p-4">
          <div class="mb-1 text-sm font-medium">{{ $t('firewall.managedTitle') }}</div>
          <p class="mb-3 text-xs text-muted-foreground">{{ $t('firewall.managedHint') }}</p>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="m in fw.siteManaged"
              :key="`${m.port}/${m.proto}`"
              class="inline-flex items-center gap-1 rounded-md bg-primary/10 px-2 py-0.5 font-mono text-xs text-primary"
              :title="m.sites"
            >
              {{ m.port }}/{{ m.proto }}
              <span class="font-sans opacity-70">{{ m.sites }}</span>
            </span>
          </div>
        </div>
      </template>
    </FaPageMain>

    <FaModal v-model="banModalVisible" :title="$t('firewall.banTitle')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">jail</span>
          <YdSelect
            v-model="banForm.jail"
            :options="(f2b?.jails || []).map(j => ({ label: j.name, value: j.name }))"
            size="default"
            button-class="flex-1"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">IP</span>
          <FaInput v-model="banForm.ip" :placeholder="$t('firewall.banIpPlaceholder')" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="banModalVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton @click="doBan">{{ $t('firewall.ban') }}</FaButton>
      </template>
    </FaModal>

    <FaModal v-model="allowVisible" :title="$t('firewall.allowPort')" :destroy-on-close="true">
      <div class="flex items-center gap-3">
        <FaInput v-model="allowForm.port" :placeholder="$t('firewall.allowPortPlaceholder')" class="w-40" />
        <YdSelect v-model="allowForm.proto" :options="['tcp', 'udp']" size="default" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="allowVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton @click="doAllow">{{ $t('firewall.allow') }}</FaButton>
      </template>
    </FaModal>
  </div>
</template>
