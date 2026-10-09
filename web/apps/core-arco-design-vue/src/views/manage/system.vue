<script setup lang="ts">
import api from '@/api/index'
import { i18n } from '@/locales'

defineOptions({
  name: 'SystemManage',
})

const toast = useFaToast()
const modal = useFaModal()

// ---- swap ----
const swap = ref<{ files: { file: string, size: string }[], totalBytes: number, usedBytes: number, on: boolean } | null>(null)
const swapBusy = ref(false)

async function loadSwap() {
  try {
    swap.value = (await api.get('api/v1/system/swap', { silent: true })).data
  }
  catch (e: any) {
    toast.error(i18n.global.t('system.swapReadFail'), { description: e?.message })
  }
}

function applySwap(sizeGB: number) {
  const label = sizeGB === 0 ? i18n.global.t('system.swapDisableLabel') : i18n.global.t('system.swapResizeLabel', { size: sizeGB })
  modal.confirm({
    title: i18n.global.t('system.swapAdjustTitle'),
    content: i18n.global.t('system.swapConfirm', { label }),
    onConfirm: async () => {
      swapBusy.value = true
      try {
        await api.post('api/v1/system/swap', { sizeGB })
        toast.success(i18n.global.t('system.applied'))
        await loadSwap()
      }
      catch (e: any) {
        toast.error(i18n.global.t('system.opFailed'), { description: e?.message })
      }
      finally {
        swapBusy.value = false
      }
    },
  })
}

// ---- BBR ----
const bbr = ref<{ algo: string, qdisc: string, bbr: boolean } | null>(null)
const bbrBusy = ref(false)

async function loadBBR() {
  try {
    bbr.value = (await api.get('api/v1/system/bbr', { silent: true })).data
  }
  catch (e: any) {
    toast.error(i18n.global.t('system.bbrReadFail'), { description: e?.message })
  }
}

async function applyBBR(enable: boolean) {
  bbrBusy.value = true
  try {
    const out = (await api.post('api/v1/system/bbr', { enable })).data
    toast.success(enable ? i18n.global.t('system.bbrOn', { algo: out.algo }) : i18n.global.t('system.bbrOff'))
    await loadBBR()
  }
  catch (e: any) {
    toast.error(i18n.global.t('system.opFailed'), { description: e?.message })
  }
  finally {
    bbrBusy.value = false
  }
}

// ---- M49：FTP / SSH / 暴力破解防护 ----
const ftp = ref<{ installed: boolean, running: boolean, port: number, pasvMin: number, pasvMax: number } | null>(null)
const ftpBusy = ref(false)
const ftpPort = ref(21)
const ftpPasv = ref('40000-40100')

async function loadFtp() {
  try {
    const res = await api.get('api/v1/ftp/status', { silent: true })
    ftp.value = res.data
    ftpPort.value = res.data?.port || 21
    ftpPasv.value = `${res.data?.pasvMin || 40000}-${res.data?.pasvMax || 40100}`
  }
  catch {}
}

async function ftpInstall() {
  ftpBusy.value = true
  toast.info('安装 vsftpd 中…')
  try {
    await api.post('api/v1/ftp/install')
    toast.success('vsftpd 已安装并启动')
    await loadFtp()
  }
  catch (e: any) {
    toast.error('安装失败', { description: e?.message })
  }
  finally { ftpBusy.value = false }
}

async function ftpPower(action: string) {
  ftpBusy.value = true
  try {
    await api.post('api/v1/ftp/power', { action })
    toast.success(`FTP ${action} 完成`)
    await loadFtp()
  }
  catch (e: any) {
    toast.error('操作失败', { description: e?.message })
  }
  finally { ftpBusy.value = false }
}

async function ftpSetPort() {
  const pm = ftpPasv.value.split('-')
  ftpBusy.value = true
  try {
    await api.post('api/v1/ftp/port', { port: Number(ftpPort.value), pasvMin: Number(pm[0]) || 40000, pasvMax: Number(pm[1]) || 40100 })
    toast.success('FTP 端口已生效（注意防火墙放行该端口与被动范围）')
    await loadFtp()
  }
  catch (e: any) {
    toast.error('设置失败', { description: e?.message })
  }
  finally { ftpBusy.value = false }
}

const sshCfg = ref<{ port: number, passwordAuth: boolean, pubkeyAuth: boolean, permitRootLogin: string } | null>(null)
const sshBusy = ref(false)
const sshKeys = ref<{ name: string, type: string, fingerprint: string, comment: string }[]>([])

async function loadSsh() {
  try {
    sshCfg.value = (await api.get('api/v1/ssh/config', { silent: true })).data
    sshKeys.value = (await api.get('api/v1/ssh/keys', { silent: true })).data || []
  }
  catch {}
}

async function setSsh(patch: Record<string, unknown>, msg: string) {
  sshBusy.value = true
  try {
    const res = await api.put('api/v1/ssh/config', patch)
    sshCfg.value = res.data
    toast.success(msg)
  }
  catch (e: any) {
    toast.error('修改失败', { description: e?.message })
  }
  finally { sshBusy.value = false }
}

const attempts = ref<{ ip: string, count: number, banned: boolean }[]>([])
const attemptsBusy = ref(false)
const banningIp = ref('')

async function loadAttempts() {
  attemptsBusy.value = true
  try {
    const res = await api.get('api/v1/ssh/attempts?limit=20', { silent: true })
    attempts.value = res.data || []
  }
  catch {}
  finally { attemptsBusy.value = false }
}

async function banIp(ip: string) {
  banningIp.value = ip
  try {
    await api.post('api/v1/fail2ban/ban', { jail: 'sshd', ip })
    toast.success(`已封禁 ${ip}`)
    await loadAttempts()
  }
  catch (e: any) {
    toast.error('封禁失败', { description: e?.message })
  }
  finally { banningIp.value = '' }
}

// ---- 系统清理 ----
const cleanBusy = ref(false)
const cleanOut = ref('')

async function doClean() {
  modal.confirm({
    title: i18n.global.t('system.clean'),
    content: i18n.global.t('system.cleanConfirm'),
    onConfirm: async () => {
      cleanBusy.value = true
      try {
        const out = (await api.post('api/v1/system/clean')).data
        cleanOut.value = out.output || i18n.global.t('system.cleanNoOutput')
        toast.success(i18n.global.t('system.cleanDone'))
      }
      catch (e: any) {
        toast.error(i18n.global.t('system.cleanFail'), { description: e?.message })
      }
      finally {
        cleanBusy.value = false
      }
    },
  })
}

onMounted(() => {
  loadSwap()
  loadBBR()
  loadFtp()
  loadSsh()
  loadAttempts()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <FaIcon name="i-lucide:wrench" :size="22" />
          <span>{{ $t('system.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('system.desc') }}</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <!-- Swap -->
        <section class="rounded-lg border bg-background p-4">
          <div class="flex items-center gap-2">
            <FaIcon name="i-lucide:hard-drive" class="text-base text-primary opacity-70" />
            <span class="font-medium">{{ $t('system.swap') }}</span>
            <span class="rounded-full px-2 py-0.5 text-xs" :class="swap?.on ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ swap?.on ? $t('system.swapOn') + `（${((swap.totalBytes || 0) / 1024 / 1024 / 1024).toFixed(1)}G）` : $t('system.swapOff') }}
            </span>
          </div>
          <div v-if="swap?.files?.length" class="mt-2 font-mono text-xs text-muted-foreground">
            {{ swap.files.map(f => `${f.file}(${f.size})`).join('  ') }}
          </div>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <FaButton variant="outline" size="sm" :disabled="swapBusy" @click="applySwap(1)">1G</FaButton>
            <FaButton variant="outline" size="sm" :disabled="swapBusy" @click="applySwap(2)">2G</FaButton>
            <FaButton variant="outline" size="sm" :disabled="swapBusy" @click="applySwap(4)">4G</FaButton>
            <FaButton variant="outline" size="sm" :disabled="swapBusy" @click="applySwap(8)">8G</FaButton>
            <FaButton variant="outline" size="sm" class="text-red-500!" :disabled="swapBusy || !swap?.on" @click="applySwap(0)">{{ $t('system.swapClose') }}</FaButton>
          </div>
        </section>

        <!-- BBR -->
        <section class="rounded-lg border bg-background p-4">
          <div class="flex items-center gap-2">
            <FaIcon name="i-lucide:gauge" class="text-base text-primary opacity-70" />
            <span class="font-medium">{{ $t('system.bbr') }}</span>
            <span class="rounded-full px-2 py-0.5 text-xs" :class="bbr?.bbr ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ bbr?.algo || '—' }}
            </span>
          </div>
          <div class="mt-2 text-xs text-muted-foreground">
            {{ $t('system.bbrDesc', { qdisc: bbr?.qdisc || '—' }) }}
          </div>
          <div class="mt-3 flex items-center gap-2">
            <FaButton v-if="!bbr?.bbr" size="sm" :loading="bbrBusy" @click="applyBBR(true)">{{ $t('system.bbrEnable') }}</FaButton>
            <FaButton v-else variant="outline" size="sm" class="text-red-500!" :loading="bbrBusy" @click="applyBBR(false)">{{ $t('system.bbrDisable') }}</FaButton>
          </div>
        </section>

        <!-- M49：FTP -->
        <section class="rounded-lg border bg-background p-4">
          <div class="flex flex-wrap items-center gap-2">
            <FaIcon name="i-lucide:folder-sync" class="text-base text-primary opacity-70" />
            <span class="font-medium">FTP 服务（vsftpd）</span>
            <span class="rounded-full px-2 py-0.5 text-xs" :class="ftp?.running ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ ftp?.running ? '运行中 · 端口 ' + (ftp?.port || 21) : ftp?.installed ? '已安装未运行' : '未安装' }}
            </span>
          </div>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <template v-if="ftp?.installed">
              <FaButton variant="outline" size="sm" :disabled="ftpBusy" @click="ftpPower('restart')">重启</FaButton>
              <FaButton variant="outline" size="sm" class="text-red-500!" :disabled="ftpBusy || !ftp?.running" @click="ftpPower('stop')">停止</FaButton>
              <span class="text-xs text-muted-foreground">端口</span>
              <FaInput v-model="ftpPort" type="number" class="w-20" />
              <span class="text-xs text-muted-foreground">被动</span>
              <FaInput v-model="ftpPasv" class="w-28" />
              <FaButton size="sm" :disabled="ftpBusy" @click="ftpSetPort">应用端口</FaButton>
            </template>
            <FaButton v-else size="sm" :loading="ftpBusy" @click="ftpInstall">安装 vsftpd</FaButton>
          </div>
        </section>

        <!-- M49：SSH 管理 -->
        <section class="rounded-lg border bg-background p-4">
          <div class="flex flex-wrap items-center gap-2">
            <FaIcon name="i-lucide:terminal" class="text-base text-primary opacity-70" />
            <span class="font-medium">SSH 管理</span>
            <span v-if="sshCfg" class="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">端口 {{ sshCfg.port }}</span>
          </div>
          <div v-if="sshCfg" class="mt-3 grid grid-cols-2 gap-3 md:grid-cols-3">
            <div class="flex items-center justify-between rounded-md border px-3 py-2">
              <span class="text-xs">密码认证</span>
              <FaSwitch :model-value="sshCfg.passwordAuth" @update:model-value="(v: any) => setSsh({ passwordAuth: v }, '密码认证已更新（reload 不中断现有连接）')" />
            </div>
            <div class="flex items-center justify-between rounded-md border px-3 py-2">
              <span class="text-xs">密钥认证</span>
              <FaSwitch :model-value="sshCfg.pubkeyAuth" @update:model-value="(v: any) => setSsh({ pubkeyAuth: v }, '密钥认证已更新')" />
            </div>
            <div class="flex items-center justify-between rounded-md border px-3 py-2">
              <span class="text-xs">root 登录</span>
              <select
                :value="sshCfg.permitRootLogin" class="h-7 rounded border bg-background px-1 text-xs outline-none"
                @change="setSsh({ permitRootLogin: ($event.target as HTMLSelectElement).value }, 'root 策略已更新')"
              >
                <option value="yes">允许</option>
                <option value="prohibit-password">仅密钥</option>
                <option value="no">禁止</option>
              </select>
            </div>
          </div>
          <div v-if="sshKeys.length" class="mt-2 flex flex-wrap gap-1">
            <span v-for="k in sshKeys" :key="k.name" class="rounded bg-muted px-1.5 py-0.5 font-mono text-[11px]" :title="k.fingerprint">
              {{ k.name }} ({{ k.type }})
            </span>
          </div>
          <div class="mt-2 text-xs text-muted-foreground">修改经 sshd -t 校验后 reload，不中断现有连接；改端口请同步放行防火墙</div>
        </section>

        <!-- M49：SSH 暴力破解防护 -->
        <section class="rounded-lg border bg-background p-4 lg:col-span-2">
          <div class="flex flex-wrap items-center gap-2">
            <FaIcon name="i-lucide:shield-ban" class="text-base text-primary opacity-70" />
            <span class="font-medium">SSH 暴力破解防护</span>
            <span class="text-xs text-muted-foreground">近 7 天失败登录聚合；封禁联动 fail2ban（sshd jail）</span>
            <FaButton variant="ghost" size="sm" class="ml-auto" :loading="attemptsBusy" @click="loadAttempts">刷新</FaButton>
          </div>
          <div v-if="attempts.length" class="mt-2 space-y-1">
            <div v-for="a in attempts" :key="a.ip" class="flex items-center gap-3 rounded-md border px-3 py-1.5 text-xs">
              <span class="w-40 font-mono">{{ a.ip }}</span>
              <span class="text-red-500">{{ a.count }} 次失败</span>
              <span v-if="a.banned" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-emerald-600">已封禁</span>
              <FaButton v-else variant="outline" size="sm" class="ml-auto" :loading="banningIp === a.ip" @click="banIp(a.ip)">封禁</FaButton>
            </div>
          </div>
          <div v-else class="mt-2 text-xs text-muted-foreground">近 7 天无失败登录记录</div>
        </section>

        <!-- 清理 -->
        <section class="rounded-lg border bg-background p-4 lg:col-span-2">
          <div class="flex flex-wrap items-center gap-2">
            <FaIcon name="i-lucide:brush" class="text-base text-primary opacity-70" />
            <span class="font-medium">{{ $t('system.clean') }}</span>
            <span class="text-xs text-muted-foreground">{{ $t('system.cleanDesc') }}</span>
            <FaButton variant="outline" size="sm" class="ml-auto" :loading="cleanBusy" @click="doClean">{{ $t('system.cleanRun') }}</FaButton>
          </div>
          <pre v-if="cleanOut" class="mt-3 max-h-56 overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs leading-relaxed">{{ cleanOut }}</pre>
        </section>
      </div>
    </FaPageMain>
  </div>
</template>
