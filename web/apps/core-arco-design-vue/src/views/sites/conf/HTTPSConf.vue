<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteHTTPSConf, SiteHTTPSUpdate } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'
import { certApi } from '@/api/modules/cert'
import type { Certificate } from '@/api/modules/cert'

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

const tlsOptions = [
  { v: '1.3', label: 'TLS 1.3' },
  { v: '1.2', label: 'TLS 1.2' },
  { v: '1.1', label: 'TLS 1.1（不安全）' },
  { v: '1.0', label: 'TLS 1.0（不安全）' },
]

const httpModeOptions = [
  { v: 'redirect', label: '访问 HTTP 自动跳转到 HTTPS' },
  { v: 'both', label: 'HTTP 与 HTTPS 均可访问' },
  { v: 'deny', label: '停止 HTTP 访问' },
]

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
    toast.error('读取 HTTPS 配置失败', { description: e?.message })
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
    toast.warning('至少选择一个 TLS 协议版本')
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
      toast.warning('请先在「证书」页申请证书，或到站点 HTTPS 选择证书来源')
      return
    }
    const res = await siteConfApi.updateHTTPS(props.site.id, body)
    applyConf(res)
    toast.success('HTTPS 设置已保存并重载 nginx')
    emit('changed')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

function disable() {
  const modal = useFaModal()
  modal.confirm({
    title: '停用 HTTPS',
    content: '停用后 443 将不再监听，HTTPS 访问会失败，确认停用？',
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
        toast.success('HTTPS 已停用')
        emit('changed')
      }
      catch (e: any) {
        toast.error('停用失败', { description: e?.message })
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
          HTTPS 状态
          <span
            class="rounded-full px-2 py-0.5 text-xs"
            :class="conf?.enable ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
          >
            {{ conf?.enable ? '已启用' : '未启用' }}
          </span>
          <span v-if="conf?.enable" class="font-mono text-xs text-muted-foreground">{{ conf.certDomain }}</span>
        </div>
        <FaButton v-if="conf?.enable" variant="outline" size="sm" :loading="saving" class="text-red-500!" @click="disable">
          停用 HTTPS
        </FaButton>
      </div>
      <p class="mt-1 text-xs text-muted-foreground">
        HTTPS 端口 443；证书在「证书」页统一管理（申请 / 上传 / 续签）
      </p>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">证书来源</div>
      <p class="mb-3 text-xs text-muted-foreground">
        从证书库选择证书（主域名或其他域名需覆盖 {{ site.domain }}；缺少合适证书时请先到「网站 → 证书」申请）
      </p>
      <select v-model.number="selCertId" class="h-9 w-96 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
        <option :value="0" disabled>请选择证书</option>
        <option v-for="c in certs" :key="c.id" :value="c.id">
          {{ c.domain }}{{ c.altDomains?.length ? ` (+${c.altDomains.length})` : '' }} · {{ c.issuer || c.provider }}
        </option>
      </select>
      <div v-if="selCert" class="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
        <span>颁发组织：{{ selCert.issuer || '—' }}</span>
        <span v-if="selCert.notAfter">过期时间：
          <span :class="notAfterText(selCert)?.cls">{{ notAfterText(selCert)?.date }}</span>
        </span>
        <span>自动续签：{{ selCert.autoRenew ? '已开启' : '关闭' }}</span>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-3 text-sm font-medium">HTTP 选项</div>
      <select v-model="httpMode" class="h-9 w-96 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
        <option v-for="o in httpModeOptions" :key="o.v" :value="o.v">{{ o.label }}</option>
      </select>
      <div class="mt-4 space-y-3">
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="http2" type="checkbox"> HTTP/2
          <span class="text-xs text-muted-foreground">（HTTP/2 提供更快的连接复用，nginx http2 指令）</span>
        </label>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="hsts" type="checkbox"> HSTS
          <span class="text-xs text-muted-foreground">开启 HSTS 可以增加网站安全性（Strict-Transport-Security）</span>
        </label>
        <label v-if="hsts" class="flex cursor-pointer items-center gap-2 pl-6 text-sm">
          <input v-model="hstsSubdomain" type="checkbox"> HSTS 子域
          <span class="text-xs text-muted-foreground">启用后，HSTS 策略将应用于当前域名的所有子域名（includeSubDomains）</span>
        </label>
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">SSL 协议设置</div>
      <div class="mb-1 text-xs text-muted-foreground">支持的协议版本</div>
      <div class="mb-3 flex flex-wrap gap-4">
        <label v-for="o in tlsOptions" :key="o.v" class="flex cursor-pointer items-center gap-1.5 text-sm">
          <input type="checkbox" :checked="tlsVersions.includes(o.v)" @change="toggleTLS(o.v)"> {{ o.label }}
        </label>
      </div>
      <div class="mb-1 text-xs text-muted-foreground">加密算法（留空使用现代默认套件）</div>
      <textarea
        v-model="ciphers"
        class="h-20 w-full rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
        placeholder="ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:…（ssl_ciphers 指令）"
      />
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" @click="save">保存并重载</FaButton>
    </div>
  </div>
</template>
