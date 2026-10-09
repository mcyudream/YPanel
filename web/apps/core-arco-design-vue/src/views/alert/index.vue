<script setup lang="ts">
import type { AlertRule } from '@/api/modules/alert'
import apiAlert from '@/api/modules/alert'
import { i18n } from '@/locales'

defineOptions({
  name: 'AlertIndex',
})

const rules = ref<AlertRule[]>([])
const loading = ref(false)
const createVisible = ref(false)
const form = ref({ name: '', metric: 'cpu', threshold: 80, webhookUrl: '', webhookType: 'feishu', silentStart: '', silentEnd: '', query: '', windowMin: 5 })
const creating = ref(false)

// M37：指标阈值量纲提示
const metricMeta: Record<string, { label: string, unit: string, def: number }> = {
  cpu: { label: i18n.global.t('alert.metricCpu'), unit: '%', def: 80 },
  memory: { label: i18n.global.t('alert.metricMem'), unit: '%', def: 85 },
  disk: { label: i18n.global.t('alert.metricDisk'), unit: '%', def: 90 },
  load: { label: i18n.global.t('alert.metricLoad'), unit: '', def: 4 },
  network: { label: i18n.global.t('alert.metricNetwork'), unit: ' MB/s', def: 80 },
  cert_expiry: { label: i18n.global.t('alert.metricCertExpiry'), unit: i18n.global.t('alert.unitDay'), def: 14 },
  log: { label: i18n.global.t('alert.metricLog'), unit: i18n.global.t('alert.unitCount'), def: 100 },
}
// 条件列的量纲后缀（含未列入 metricMeta 的 site_expiry）
function metricUnit(metric: string) {
  if (metric === 'load') return ''
  if (metric === 'network') return ' MB/s'
  if (metric === 'cert_expiry' || metric === 'site_expiry') return i18n.global.t('alert.unitDay')
  return '%'
}
const isLogRule = computed(() => form.value.metric === 'log')
watch(() => form.value.metric, (m) => {
  if (metricMeta[m]) {
    form.value.threshold = metricMeta[m].def
  }
})

// M37：邮件渠道不需要 URL
const needUrl = computed(() => form.value.webhookType !== 'email')

async function load() {
  loading.value = true
  try {
    rules.value = await apiAlert.list()
  }
  finally {
    loading.value = false
  }
}

async function doCreate() {
  creating.value = true
  try {
    const payload: any = { ...form.value }
    if (isLogRule.value) {
      payload.windowSec = (form.value.windowMin || 5) * 60
      if (!payload.query?.trim()) {
        useFaToast().warning(i18n.global.t('alert.queryRequired'))
        return
      }
    }
    await apiAlert.create(payload)
    useFaToast().success(i18n.global.t('alert.ruleCreated'))
    createVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('alert.createFailed'), { description: e?.message })
  }
  finally {
    creating.value = false
  }
}

async function toggle(r: AlertRule) {
  try {
    await apiAlert.update(r.id, { enabled: !r.enabled })
    await load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('alert.opFailed'), { description: e?.message })
  }
}

function remove(r: AlertRule) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('alert.deleteRuleTitle'),
    content: i18n.global.t('alert.deleteRuleConfirm', { name: r.name }),
    onConfirm: async () => {
      try {
        await apiAlert.remove(r.id)
        useFaToast().success(i18n.global.t('alert.deleted'))
        await load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('alert.deleteFailed'), { description: e?.message })
      }
    },
  })
}

// ---- M37：SMTP 邮件通道全局设置 ----
const smtp = ref({ host: '', port: 465, user: '', pass: '', from: '', ssl: true, to: '', hasPass: false })
const smtpSaving = ref(false)
const smtpTesting = ref(false)

async function loadSmtp() {
  try {
    const res = await apiAlert.getSmtp()
    smtp.value = { ...smtp.value, ...res, pass: '' }
  }
  catch {}
}

async function saveSmtp() {
  smtpSaving.value = true
  try {
    await apiAlert.putSmtp(smtp.value)
    useFaToast().success(i18n.global.t('alert.smtpSaved'))
    await loadSmtp()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('alert.saveFailed'), { description: e?.message })
  }
  finally {
    smtpSaving.value = false
  }
}

async function testSmtp() {
  smtpTesting.value = true
  try {
    await apiAlert.testSmtp(smtp.value.to)
    useFaToast().success(i18n.global.t('alert.testSentTo', { to: smtp.value.to }))
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('alert.sendFailed'), { description: e?.message })
  }
  finally {
    smtpTesting.value = false
  }
}

function thresholdHint() {
  if (isLogRule.value) {
    return i18n.global.t('alert.unitLogHits')
  }
  const m = metricMeta[form.value.metric]
  return m ? i18n.global.t('alert.thresholdUnit', { u: m.unit || i18n.global.t('alert.unitNone') }) : ''
}

onMounted(() => {
  load()
  loadSmtp()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="bell" :size="24" />
          <span>{{ $t('alert.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('alert.description') }}</span>
      </template>
      <FaButton size="sm" @click="createVisible = true">
        <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('alert.newRule') }}
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <!-- M37：邮件通道全局设置 -->
      <section class="mb-4 rounded-lg border bg-background p-4">
        <div class="flex flex-wrap items-center gap-2">
          <FaIcon name="i-lucide:mail" class="text-base text-primary opacity-70" />
          <span class="font-medium">{{ $t('alert.smtp') }}</span>
          <span class="text-xs text-muted-foreground">{{ $t('alert.smtpNote') }}</span>
          <div class="ml-auto flex items-center gap-2">
            <FaButton variant="outline" size="sm" :loading="smtpTesting" :disabled="!smtp.host || !smtp.to" @click="testSmtp">{{ $t('alert.testSend') }}</FaButton>
            <FaButton size="sm" :loading="smtpSaving" @click="saveSmtp">{{ $t('common.save') }}</FaButton>
          </div>
        </div>
        <div class="mt-3 grid grid-cols-2 gap-3 md:grid-cols-4">
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('alert.smtpHost') }}</span>
            <FaInput v-model="smtp.host" placeholder="smtp.example.com" class="w-full" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('alert.smtpPort') }}</span>
            <FaInput v-model="smtp.port" type="number" class="w-full" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('alert.smtpUser') }}</span>
            <FaInput v-model="smtp.user" class="w-full" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('alert.smtpPass') }}{{ smtp.hasPass ? $t('alert.passSet') : '' }}</span>
            <FaInput v-model="smtp.pass" type="password" class="w-full" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('alert.smtpFrom') }}</span>
            <FaInput v-model="smtp.from" placeholder="alert@example.com" class="w-full" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('alert.smtpTo') }}</span>
            <FaInput v-model="smtp.to" placeholder="ops@example.com" class="w-full" />
          </label>
          <label class="flex items-end gap-2 pb-2">
            <input v-model="smtp.ssl" type="checkbox" class="accent-[var(--primary)]">
            <span class="text-xs text-muted-foreground">{{ $t('alert.implicitSsl') }}</span>
          </label>
        </div>
      </section>

      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('alert.rule') }}</th>
              <th class="px-3 py-2">{{ $t('alert.condition') }}</th>
              <th class="hidden px-3 py-2 lg:table-cell">{{ $t('alert.channel') }}</th>
              <th class="hidden px-3 py-2 xl:table-cell">{{ $t('alert.silent') }}</th>
              <th class="px-3 py-2">{{ $t('common.status') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !rules.length">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">{{ $t('common.loading') }}</td>
            </tr>
            <tr v-else-if="!rules.length">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">{{ $t('alert.noRules') }}</td>
            </tr>
            <tr v-for="r in rules" :key="r.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2 font-medium">{{ r.name }}</td>
              <td class="px-3 py-2 text-xs">
                <template v-if="r.metric === 'log'">
                  {{ $t('alert.logCondition', { n: r.threshold, m: Math.round((r.windowSec || 300) / 60) }) }}
                  <div class="mt-0.5 max-w-56 truncate font-mono text-[10px] text-muted-foreground" :title="r.query">{{ r.query }}</div>
                </template>
                <template v-else>
                  {{ r.metric.toUpperCase() }} &gt; {{ r.threshold }}{{ metricUnit(r.metric) }}
                </template>
              </td>
              <td class="hidden max-w-72 truncate px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell" :title="r.webhookUrl">
                [{{ r.webhookType }}] {{ r.webhookType === 'email' ? $t('alert.globalSmtp') : r.webhookType === 'bark' ? 'bark' : r.webhookUrl }}
              </td>
              <td class="hidden px-3 py-2 text-xs text-muted-foreground xl:table-cell">
                {{ r.silentStart && r.silentEnd ? `${r.silentStart} ~ ${r.silentEnd}` : '—' }}
              </td>
              <td class="px-3 py-2">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="r.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  {{ r.enabled ? $t('common.enabled') : $t('common.disabled') }}
                </span>
              </td>
              <td class="px-3 py-2 text-right">
                <FaButton variant="outline" size="sm" @click="toggle(r)">{{ r.enabled ? $t('common.disabled') : $t('common.enabled') }}</FaButton>
                <FaButton variant="outline" size="sm" class="text-red-500!" @click="remove(r)">{{ $t('common.delete') }}</FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <FaModal v-model="createVisible" :title="$t('alert.createTitle')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.name') }}</span>
          <FaInput v-model="form.name" :placeholder="$t('alert.namePh')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('alert.metric') }}</span>
          <select v-model="form.metric" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option v-for="(m, k) in metricMeta" :key="k" :value="k">{{ m.label }}</option>
          </select>
        </div>
        <div v-if="isLogRule" class="flex items-start gap-3">
          <span class="w-20 shrink-0 pt-2 text-sm text-muted-foreground">LogsQL</span>
          <textarea
            v-model="form.query" rows="2" spellcheck="false"
            placeholder='如：{container_name="app-xxx"}（统计窗口内命中条数）'
            class="min-w-0 flex-1 rounded-md border border-input bg-background px-2 py-1.5 font-mono text-xs outline-none focus:ring-1 focus:ring-[rgb(var(--primary))]"
          />
        </div>
        <div v-if="isLogRule" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('alert.window') }}</span>
          <select v-model="form.windowMin" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option :value="1">{{ $t('alert.lastNMin', { n: 1 }) }}</option>
            <option :value="5">{{ $t('alert.lastNMin', { n: 5 }) }}</option>
            <option :value="15">{{ $t('alert.lastNMin', { n: 15 }) }}</option>
            <option :value="30">{{ $t('alert.lastNMin', { n: 30 }) }}</option>
            <option :value="60">{{ $t('alert.lastNHour', { n: 1 }) }}</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('alert.threshold') }}</span>
          <FaInput v-model="form.threshold" type="number" class="w-32" />
          <span class="text-xs text-muted-foreground">{{ thresholdHint() }}</span>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('alert.channelType') }}</span>
          <select v-model="form.webhookType" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="feishu">{{ $t('alert.chFeishu') }}</option>
            <option value="dingtalk">{{ $t('alert.chDingtalk') }}</option>
            <option value="wecom">{{ $t('alert.chWecom') }}</option>
            <option value="telegram">Telegram Bot</option>
            <option value="bark">{{ $t('alert.chBark') }}</option>
            <option value="email">{{ $t('alert.chEmail') }}</option>
            <option value="generic">{{ $t('alert.chGeneric') }}</option>
          </select>
        </div>
        <div v-if="needUrl" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">Webhook</span>
          <FaInput v-model="form.webhookUrl" :placeholder="form.webhookType === 'bark' ? $t('alert.barkPh') : 'https://...'" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('alert.silent') }}</span>
          <FaInput v-model="form.silentStart" :placeholder="$t('alert.silentStartPh')" class="flex-1" />
          <span class="text-xs text-muted-foreground">{{ $t('alert.silentTo') }}</span>
          <FaInput v-model="form.silentEnd" :placeholder="$t('alert.silentEndPh')" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">{{ $t('common.cancel') }}</FaButton>
        <FaButton :loading="creating" @click="doCreate">{{ $t('alert.create') }}</FaButton>
      </template>
    </FaModal>
  </div>
</template>
