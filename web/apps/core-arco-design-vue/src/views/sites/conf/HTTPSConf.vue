<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteHTTPSConf, SiteHTTPSUpdate } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'
import { certApi } from '@/api/modules/cert'
import type { Certificate } from '@/api/modules/cert'
import { i18n } from '@/locales'

// B23 HTTPS 完整设置（对齐 1Panel）：证书库选择 / HTTP 模式 / HSTS / TLS 版本 / 加密算法 / HTTP2。

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const conf = ref<SiteHTTPSConf | null>(null)
const certs = ref<Certificate[]>([])
const loading = ref(false)
const saving = ref(false)

// 编辑态
const selCertId = ref(0)
const httpMode = ref<'redirect' | 'both' | 'deny'>('redirect')
const hsts = ref(false)
const hstsSubdomain = ref(false)
const http2 = ref(true)
const tlsVersions = ref<string[]>(['1.3', '1.2'])
const ciphers = ref('')

const tlsOptions = computed(() => [
  { v: '1.3', label: 'TLS 1.3' },
  { v: '1.2', label: 'TLS 1.2' },
  { v: '1.1', label: i18n.global.t('sites.conf.https.tls11Unsafe') },
  { v: '1.0', label: i18n.global.t('sites.conf.https.tls10Unsafe') },
])

const httpModeOptions = computed(() => [
  { v: 'redirect', label: i18n.global.t('sites.conf.https.modeRedirect') },
  { v: 'both', label: i18n.global.t('sites.conf.https.modeBoth') },
  { v: 'deny', label: i18n.global.t('sites.conf.https.modeDeny') },
])

function applyConf(c: SiteHTTPSConf) {
  conf.value = c
  selCertId.value = c.certId || 0
  httpMode.value = c.httpMode || 'redirect'
  hsts.value = c.hsts
  hstsSubdomain.value = c.hstsSubdomain
  http2.value = c.http2
  tlsVersions.value = c.tlsVersions?.length ? [...c.tlsVersions] : ['1.3', '1.2']
  ciphers.value = c.ciphers || ''
}

async function load() {
  loading.value = true
  try {
    const [c, list] = await Promise.all([
      siteConfApi.getHTTPS(props.site.id),
      certApi.list().catch(() => [] as Certificate[]),
    ])
    certs.value = list
    applyConf(c)
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.conf.https.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

function toggleTLS(v: string) {
  const i = tlsVersions.value.indexOf(v)
  if (i >= 0)
    tlsVersions.value.splice(i, 1)
  else
    tlsVersions.value.push(v)
}

async function save() {
  if (!tlsVersions.value.length) {
    toast.warning(i18n.global.t('sites.conf.https.needTls'))
    return
  }
  saving.value = true
  try {
    const body: SiteHTTPSUpdate = {
      httpMode: httpMode.value,
      hsts: hsts.value,
      hstsSubdomain: hstsSubdomain.value,
      tlsVersions: ['1.3', '1.2', '1.1', '1.0'].filter(v => tlsVersions.value.includes(v)),
      ciphers: ciphers.value,
      http2: http2.value,
    }
    if (selCertId.value && selCertId.value !== conf.value?.certId) {
      body.certId = selCertId.value
    }
    else if (!conf.value?.enable) {
      toast.warning(i18n.global.t('sites.conf.https.needCert'))
      return
    }
    const res = await siteConfApi.updateHTTPS(props.site.id, body)
    applyConf(res)
    toast.success(i18n.global.t('sites.conf.https.saved'))
    emit('changed')
  }
  catch (e: any) {
    toast.error(i18n.global.t('sites.shared.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

function disable() {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('sites.conf.https.disableTitle'),
    content: i18n.global.t('sites.conf.https.disableConfirm'),
    onConfirm: async () => {
      saving.value = true
      try {
        const res = await siteConfApi.updateHTTPS(props.site.id, {
          disable: true,
          httpMode: httpMode.value,
          hsts: hsts.value,
          hstsSubdomain: hstsSubdomain.value,
          tlsVersions: tlsVersions.value,
          ciphers: ciphers.value,
          http2: http2.value,
        })
        applyConf(res)
        toast.success(i18n.global.t('sites.conf.https.disabled'))
        emit('changed')
      }
      catch (e: any) {
        toast.error(i18n.global.t('sites.conf.https.disableFailed'), { description: e?.message })
      }
      finally {
        saving.value = false
      }
    },
  })
}

const selCert = computed(() => certs.value.find(c => c.id === selCertId.value))

function notAfterText(c: Certificate): { date: string, cls: string } | undefined {
  if (!c.notAfter)
    return undefined
  const d = new Date(c.notAfter)
  const days = Math.floor((d.getTime() - Date.now()) / 86400000)
  const cls = days < 0 ? 'text-red-500' : days < 15 ? 'text-amber-500' : 'text-muted-foreground'
  return { date: d.toISOString().slice(0, 10), cls }
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2 text-sm font-medium">
          {{ $t('sites.conf.https.status') }}
          <span
            class="rounded-full px-2 py-0.5 text-xs"
            :class="conf?.enable ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
          >
            {{ conf?.enable ? $t('common.enabled') : $t('sites.conf.https.notEnabled') }}
          </span>
          <span v-if="conf?.enable" class="font-mono text-xs text-muted-foreground">{{ conf.certDomain }}</span>
        </div>
        <FaButton v-if="conf?.enable" variant="outline" size="sm" :loading="saving" class="text-red-500!" @click="disable">
          {{ $t('sites.conf.https.disableBtn') }}
        </FaButton>
      </div>
      <p class="mt-1 text-xs text-muted-foreground">
        {{ $t('sites.conf.https.statusHint') }}
      </p>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.https.certSource') }}</div>
      <p class="mb-3 text-xs text-muted-foreground">
        {{ $t('sites.conf.https.certSourceDesc', { domain: site.domain }) }}
      </p>
      <select v-model.number="selCertId" class="h-9 w-96 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
        <option :value="0" disabled>{{ $t('sites.conf.https.pickCert') }}</option>
        <option v-for="c in certs" :key="c.id" :value="c.id">
          {{ c.domain }}{{ c.altDomains?.length ? ` (+${c.altDomains.length})` : '' }} · {{ c.issuer || c.provider }}
        </option>
      </select>
      <div v-if="selCert" class="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
        <span>{{ $t('sites.conf.https.issuer', { org: selCert.issuer || '—' }) }}</span>
        <span v-if="selCert.notAfter">{{ $t('sites.conf.https.expires') }}
          <span :class="notAfterText(selCert)?.cls">{{ notAfterText(selCert)?.date }}</span>
        </span>
        <span>{{ $t('sites.conf.https.autoRenew') }}{{ selCert.autoRenew ? $t('common.on') : $t('common.off') }}</span>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-3 text-sm font-medium">{{ $t('sites.conf.https.httpOptions') }}</div>
      <select v-model="httpMode" class="h-9 w-96 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
        <option v-for="o in httpModeOptions" :key="o.v" :value="o.v">{{ o.label }}</option>
      </select>
      <div class="mt-4 space-y-3">
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="http2" type="checkbox"> HTTP/2
          <span class="text-xs text-muted-foreground">{{ $t('sites.conf.https.http2Hint') }}</span>
        </label>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="hsts" type="checkbox"> HSTS
          <span class="text-xs text-muted-foreground">{{ $t('sites.conf.https.hstsHint') }}</span>
        </label>
        <label v-if="hsts" class="flex cursor-pointer items-center gap-2 pl-6 text-sm">
          <input v-model="hstsSubdomain" type="checkbox"> {{ $t('sites.conf.https.hstsSub') }}
          <span class="text-xs text-muted-foreground">{{ $t('sites.conf.https.hstsSubHint') }}</span>
        </label>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">{{ $t('sites.conf.https.sslProtocols') }}</div>
      <div class="mb-1 text-xs text-muted-foreground">{{ $t('sites.conf.https.supportedVersions') }}</div>
      <div class="mb-3 flex flex-wrap gap-4">
        <label v-for="o in tlsOptions" :key="o.v" class="flex cursor-pointer items-center gap-1.5 text-sm">
          <input type="checkbox" :checked="tlsVersions.includes(o.v)" @change="toggleTLS(o.v)"> {{ o.label }}
        </label>
      </div>
      <div class="mb-1 text-xs text-muted-foreground">{{ $t('sites.conf.https.ciphers') }}</div>
      <textarea
        v-model="ciphers"
        class="h-20 w-full rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
        :placeholder="$t('sites.conf.https.ciphersPlaceholder')"
      />
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" @click="save">{{ $t('sites.shared.saveReload') }}</FaButton>
    </div>
  </div>
</template>
