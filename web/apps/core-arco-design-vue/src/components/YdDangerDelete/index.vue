<script setup lang="ts">
// YdDangerDelete 危险删除确认弹窗（对齐 1Panel）：可选项勾选 + 输入资源名称确认。
// 适用：网站/应用/容器/数据库实例等不可逆删除。
export interface DangerOption {
  key: string
  label: string
  desc?: string
}

defineOptions({
  name: 'YdDangerDelete',
})

const props = withDefaults(defineProps<{
  visible: boolean
  title: string
  /** 需要输入确认的名称；空串则不要求输入 */
  name?: string
  options?: DangerOption[]
  loading?: boolean
  /** 确认按钮文案 */
  confirmText?: string
}>(), {
  name: '',
  options: () => [],
  loading: false,
  confirmText: undefined,
})

const emit = defineEmits<{
  'update:visible': [v: boolean]
  confirm: [checked: Record<string, boolean>]
}>()

const checked = ref<Record<string, boolean>>({})
const typed = ref('')

const nameOk = computed(() => !props.name || typed.value === props.name)
const canConfirm = computed(() => nameOk.value && !props.loading)

watch(() => props.visible, (v) => {
  try { localStorage.setItem('ydd-debug', 'YDD visible=' + v + ' @' + Date.now()) } catch {}
  if (v) {
    checked.value = Object.fromEntries((props.options || []).map(o => [o.key, false]))
    typed.value = ''
  }
})

function onConfirm() {
  if (!canConfirm.value) {
    return
  }
  emit('confirm', { ...checked.value })
}

// FaModal 打开时也会回发 update:modelValue=true，无条件 close 会形成「开→回发→关」死循环
// （表现为点击删除/解除毫无反应）。标准受控：把新值如实上抛，由父层 v-model 收敛。
function onModalUpdate(v: boolean) {
  emit('update:visible', v)
}

function close() {
  emit('update:visible', false)
}
</script>

<template>
  <FaModal :model-value="visible" :title="title" class="max-w-md!" :close-on-click-modal="false" @update:model-value="onModalUpdate">
    <div class="flex flex-col gap-4">
      <div v-for="o in options" :key="o.key">
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="checked[o.key]" type="checkbox" class="accent-[rgb(var(--primary))]">
          {{ o.label }}
        </label>
        <div v-if="o.desc" class="mt-0.5 pl-6 text-xs text-muted-foreground">
          {{ o.desc }}
        </div>
      </div>

      <div v-if="name" class="text-sm">
        <div class="mb-1.5">
          {{ $t('components.ydDangerDelete.confirmHintPrefix') }} <span class="font-medium text-red-500">{{ name }}</span> {{ $t('components.ydDangerDelete.confirmHintSuffix') }}
        </div>
        <FaInput v-model="typed" :placeholder="name" class="w-full" @keyup.enter="onConfirm" />
      </div>
    </div>
    <template #footer>
      <FaButton variant="outline" @click="close">
        {{ $t('common.cancel') }}
      </FaButton>
      <FaButton :disabled="!canConfirm" :loading="loading" class="bg-red-500! text-white!" @click="onConfirm">
        {{ confirmText ?? $t('common.delete') }}
      </FaButton>
    </template>
  </FaModal>
</template>
