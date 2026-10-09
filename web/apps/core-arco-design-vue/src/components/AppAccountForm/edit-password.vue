<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'
import { i18n } from '@/locales'

defineOptions({
  name: 'EditPasswordForm',
})

const appAccountStore = useAppAccountStore()

const loading = ref(false)

interface EditPasswordModel {
  password: string
  newPassword: string
  checkPassword: string
}

const model = ref<EditPasswordModel>({
  password: '',
  newPassword: '',
  checkPassword: '',
})

const validationSchema = toTypedSchema(
  z.object({
    password: z.string().min(1, i18n.global.t('login.pw.oldRequired')),
    newPassword: z.string().min(1, i18n.global.t('login.pw.newRequired')).min(6, i18n.global.t('login.pw.length')).max(18, i18n.global.t('login.pw.length')),
    checkPassword: z.string().min(1, i18n.global.t('login.pw.confirmRequired')),
  }).refine(data => data.newPassword === data.checkPassword, {
    message: i18n.global.t('login.pw.mismatch'),
    path: ['checkPassword'],
  }),
)

function onSubmit(values: EditPasswordModel) {
  loading.value = true
  appAccountStore.editPassword(values).then(async () => {
    useFaToast().success(i18n.global.t('login.pw.mockSuccess'))
    appAccountStore.logout()
  }).finally(() => {
    loading.value = false
  })
}
</script>

<template>
  <div class="flex-col-stretch-center w-full">
    <div class="mb-6 space-y-2">
      <h3 class="text-4xl font-bold">
        {{ $t('login.pw.title') }}
      </h3>
      <p class="text-sm text-muted-foreground lg:text-base">
        {{ $t('login.pw.desc') }}
      </p>
    </div>
    <FaForm :model="model" :validation-schema="validationSchema" @submit="onSubmit">
      <FaFormItem name="password">
        <FaInput type="password" :placeholder="$t('login.pw.oldPlaceholder')" class="w-full">
          <template #start>
            <FaIcon name="i-lucide:lock" />
          </template>
        </FaInput>
      </FaFormItem>
      <FaFormItem name="newPassword">
        <FaInput type="password" :placeholder="$t('login.pw.newPlaceholder')" class="w-full">
          <template #start>
            <FaIcon name="i-lucide:lock" />
          </template>
        </FaInput>
      </FaFormItem>
      <FaFormItem name="checkPassword">
        <FaInput type="password" :placeholder="$t('login.pw.confirmPlaceholder')" class="w-full">
          <template #start>
            <FaIcon name="i-lucide:lock" />
          </template>
        </FaInput>
      </FaFormItem>
      <FaButton :loading="loading" size="lg" class="mt-8 w-full" type="submit">
        {{ $t('common.save') }}
      </FaButton>
    </FaForm>
  </div>
</template>
