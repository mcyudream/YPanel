<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'

defineOptions({
  name: 'LoginForm',
})

const props = defineProps<{
  account?: string
}>()

const emits = defineEmits<{
  onLogin: [account?: string]
}>()

const appAccountStore = useAppAccountStore()

const title = import.meta.env.VITE_APP_TITLE
const loading = ref(false)

interface LoginModel {
  account: string
  password: string
}

const model = ref<LoginModel>({
  account: props.account ?? localStorage.getItem('login_account') ?? '',
  password: '',
})

// 2FA：后端返回 2101 表示需要动态码
const needOtp = ref(false)
const otpCode = ref('')

const validationSchema = toTypedSchema(z.object({
  account: z.string().min(1, '请输入用户名'),
  password: z.string().min(1, '请输入密码'),
}))

function onSubmit(values: LoginModel) {
  loading.value = true
  appAccountStore.login({ ...values, otpCode: otpCode.value }).then(() => {
    localStorage.setItem('login_account', values.account)
    emits('onLogin', values.account)
  }).catch((err: any) => {
    if (err?.code === 2101) {
      needOtp.value = true
    }
    throw err
  }).finally(() => {
    loading.value = false
  })
}
</script>

<template>
  <div class="p-12 flex-col-stretch-center min-h-500px w-full">
    <div class="mb-6 space-y-2">
      <h3 class="text-4xl font-bold">
        欢迎回来 👋🏻
      </h3>
      <p class="text-sm text-muted-foreground lg:text-base">
        {{ title }} · 服务器管理面板
      </p>
    </div>
    <FaForm
      ref="formRef"
      :model="model"
      :validation-schema="validationSchema"
      @submit="onSubmit"
    >
      <FaFormItem name="account">
        <FaInput type="text" placeholder="用户名" class="w-full">
          <template #start>
            <FaIcon name="i-lucide:user" />
          </template>
        </FaInput>
      </FaFormItem>
      <FaFormItem name="password">
        <FaInput type="password" placeholder="密码" class="w-full">
          <template #start>
            <FaIcon name="i-lucide:lock" />
          </template>
        </FaInput>
      </FaFormItem>
      <FaFormItem v-if="needOtp" name="otpCode">
        <FaInput v-model="otpCode" type="text" inputmode="numeric" maxlength="6" placeholder="两步验证码（6 位）" class="w-full">
          <template #start>
            <FaIcon name="i-lucide:shield-check" />
          </template>
        </FaInput>
      </FaFormItem>
      <FaButton :loading="loading" size="lg" class="w-full" type="submit">
        登 录
      </FaButton>
      <div class="text-sm mt-4 text-center text-secondary-foreground op-50">
        忘记密码？请在服务器执行 <code>ypanel -reset-admin admin</code> 重置
      </div>
    </FaForm>
  </div>
</template>
