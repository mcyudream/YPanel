<script setup lang="ts">
import type { SecuritySettings } from '@/api/modules/security'
import { securityApi } from '@/api/modules/security'

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
})

async function load() {
  loading.value = true
  try {
    data.value = await securityApi.get()
  }
  catch (e: any) {
    toast.error('加载安全设置失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    data.value = await securityApi.update({
      safeEntry: data.value.safeEntry.trim(),
      allowedIps: data.value.allowedIps.trim(),
      sessionHours: Number(data.value.sessionHours) || 24,
    })
    toast.success('安全设置已保存')
    if (data.value.safeEntry) {
      localStorage.setItem('login_entry', data.value.safeEntry)
    }
    else {
      localStorage.removeItem('login_entry')
    }
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
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
    toast.error('生成 2FA 密钥失败', { description: e?.message })
  }
  finally {
    twofaBusy.value = false
  }
}

async function disable2FA() {
  twofaBusy.value = true
  try {
    await securityApi.twoFADisable()
    toast.success('两步验证已关闭')
    await load()
  }
  catch (e: any) {
    toast.error('操作失败', { description: e?.message })
  }
  finally {
    twofaBusy.value = false
  }
}

function copyText(text: string) {
  navigator.clipboard.writeText(text).then(() => toast.success('已复制'))
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="shield" :size="24" />
          <span>安全设置</span>
        </div>
      </template>
      <template #description>
        <span>两步验证、安全入口、IP 白名单与会话超时</span>
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
                两步验证（TOTP）
              </div>
              <p class="mt-1 text-xs text-muted-foreground">
                启用后登录需输入验证器 App 中的 6 位动态码
              </p>
            </div>
            <span
              class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs"
              :class="data.twoFaEnabled ? 'text-emerald-600 bg-emerald-500/10' : 'text-muted-foreground bg-muted'"
            >
              <span class="inline-block size-1.5 rounded-full bg-current" :class="data.twoFaEnabled ? 'animate-pulse' : ''" />
              {{ data.twoFaEnabled ? '已启用' : '未启用' }}
            </span>
          </div>
          <div class="mt-3 flex gap-2">
            <FaButton v-if="!data.twoFaEnabled" size="sm" :loading="twofaBusy" @click="start2FA">
              启用两步验证
            </FaButton>
            <FaButton v-else variant="outline" size="sm" :loading="twofaBusy" @click="disable2FA">
              关闭两步验证
            </FaButton>
          </div>
        </div>

        <!-- 入口/白名单/超时 -->
        <div class="space-y-5 rounded-lg border p-5">
          <div>
            <div class="flex items-center gap-2 text-sm font-medium">
              <FaIcon name="i-lucide:door-closed" class="text-base" />
              安全入口
            </div>
            <p class="mt-1 text-xs text-muted-foreground">
              设置后登录页需带 ?entry=入口段 访问（如 <code>/login?entry=my-panel</code>），留空关闭
            </p>
            <FaInput v-model="data.safeEntry" placeholder="如 my-panel（字母/数字/中划线）" class="mt-2 w-full" />
          </div>
          <div>
            <div class="flex items-center gap-2 text-sm font-medium">
              <FaIcon name="i-lucide:network" class="text-base" />
              IP 白名单
            </div>
            <p class="mt-1 text-xs text-muted-foreground">
              逗号分隔，支持 192.168.1.10 或 10.0.0.* 通配；留空不限制。注意：保存前请确认当前 IP 在列，否则可能被锁在门外
            </p>
            <FaInput v-model="data.allowedIps" placeholder="如 192.168.100.0/24 格式或 192.168.100.* 通配" class="mt-2 w-full" />
          </div>
          <div>
            <div class="flex items-center gap-2 text-sm font-medium">
              <FaIcon name="i-lucide:timer" class="text-base" />
              会话超时（小时）
            </div>
            <p class="mt-1 text-xs text-muted-foreground">
              登录态有效期，1-720 小时，修改后新登录生效
            </p>
            <FaInput v-model="data.sessionHours" type="number" class="mt-2 w-40" />
          </div>
          <div class="flex justify-end">
            <FaButton :loading="saving" @click="save">
              保存设置
            </FaButton>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 2FA 绑定弹窗 -->
    <FaModal
      v-model="twofaModal"
      title="绑定两步验证"
      class="max-w-xl!"
      :destroy-on-close="true"
    >
      <div v-if="twofaSetup" class="space-y-4 text-sm">
        <div class="rounded-md border border-amber-300 bg-amber-50 p-3 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
          密钥已生成并即刻生效。请立即在验证器 App（Google Authenticator / 1Password 等）中添加，否则下次登录将无法通过验证。
        </div>
        <div>
          <div class="mb-1 text-xs text-muted-foreground">
            方式一：App 扫描 otpauth 链接（选择「扫二维码」时粘贴此链接或手工输入密钥）
          </div>
          <div class="flex items-center gap-2">
            <code class="flex-1 truncate rounded bg-muted px-2 py-1.5 font-mono text-xs">{{ twofaSetup.otpauthUri }}</code>
            <FaButton variant="outline" size="icon-sm" title="复制链接" @click="copyText(twofaSetup.otpauthUri)">
              <FaIcon name="i-lucide:copy" class="text-sm" />
            </FaButton>
          </div>
        </div>
        <div>
          <div class="mb-1 text-xs text-muted-foreground">
            方式二：手工输入密钥（账户名 admin，类型「基于时间」）
          </div>
          <div class="flex items-center gap-2">
            <code class="flex-1 rounded bg-muted px-2 py-1.5 font-mono text-sm tracking-widest">{{ twofaSetup.secret }}</code>
            <FaButton variant="outline" size="icon-sm" title="复制密钥" @click="copyText(twofaSetup.secret)">
              <FaIcon name="i-lucide:copy" class="text-sm" />
            </FaButton>
          </div>
        </div>
      </div>
      <template #footer>
        <FaButton @click="twofaModal = false">
          我已保存密钥
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
