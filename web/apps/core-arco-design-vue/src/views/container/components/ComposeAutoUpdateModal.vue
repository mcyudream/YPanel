<script setup lang="ts">
import type { ComposeAutoUpdateConfig } from '@/api/modules/compose'
import { useIntervalFn } from '@vueuse/core'
import apiCompose from '@/api/modules/compose'
import { i18n } from '@/locales'
import { useTaskCenterStore } from '@/store/modules/taskCenter'

const props = defineProps<{ project: string, dir?: string, nodeId: string }>()
const visible = defineModel<boolean>({ default: false })
const toast = useFaToast()
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const refreshing = ref(false)
const statusError = ref('')
const advanced = ref(false)
const frequency = ref<string | number>(60)
const customInterval = ref<string | number>(60)
const config = ref<ComposeAutoUpdateConfig | null>(null)
const form = ref({ enabled: false, preHook: '', postHook: '', hookFailurePolicy: 'stop' as 'stop' | 'continue', remark: '' })
let requestVersion = 0

const intervalMin = computed(() => Number(frequency.value === 'custom' ? customInterval.value : frequency.value))
const intervalValid = computed(() => Number.isInteger(intervalMin.value) && intervalMin.value >= 5 && intervalMin.value <= 1440)
const frequencyOptions = computed(() => [
  { value: 60, label: i18n.global.t('container.apps.autoUpdate.hourly') },
  { value: 360, label: i18n.global.t('container.apps.autoUpdate.everySixHours') },
  { value: 1440, label: i18n.global.t('container.apps.autoUpdate.daily') },
  { value: 'custom', label: i18n.global.t('container.apps.autoUpdate.custom') },
])
const failureOptions = computed(() => [
  { value: 'stop', label: i18n.global.t('container.apps.autoUpdate.stopOnFailure') },
  { value: 'continue', label: i18n.global.t('container.apps.autoUpdate.continueOnFailure') },
])
const failurePolicy = computed<string | number>({
  get: () => form.value.hookFailurePolicy,
  set: (value) => {
    if (value === 'stop' || value === 'continue') {
      form.value.hookFailurePolicy = value
    }
  },
})

function formatTime(value?: string) {
  if (!value) {
    return i18n.global.t('container.apps.autoUpdate.notChecked')
  }
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString(i18n.global.locale.value)
}

const { pause, resume } = useIntervalFn(() => {
  void refreshStatus()
}, 3000, { immediate: false })

async function refreshStatus() {
  if (refreshing.value || loading.value || !visible.value) {
    return
  }
  const version = requestVersion
  refreshing.value = true
  try {
    const next = await apiCompose.autoUpdateGet(props.project, props.nodeId)
    if (version !== requestVersion || !visible.value) {
      return
    }
    config.value = next
    statusError.value = ''
    if (next.lastStatus === 'running') {
      resume()
    }
    else {
      pause()
    }
  }
  catch (e: any) {
    if (version === requestVersion && visible.value) {
      statusError.value = e?.message || i18n.global.t('container.apps.autoUpdate.loadFailed')
      pause()
    }
  }
  finally {
    if (version === requestVersion) {
      refreshing.value = false
    }
  }
}

async function load() {
  const version = ++requestVersion
  pause()
  loading.value = true
  loadError.value = ''
  statusError.value = ''
  refreshing.value = false
  config.value = null
  try {
    const next = await apiCompose.autoUpdateGet(props.project, props.nodeId)
    if (version !== requestVersion || !visible.value) {
      return
    }
    config.value = next
    form.value = {
      enabled: next.enabled,
      preHook: next.preHook,
      postHook: next.postHook,
      hookFailurePolicy: next.hookFailurePolicy,
      remark: next.remark,
    }
    customInterval.value = next.intervalMin
    frequency.value = [60, 360, 1440].includes(next.intervalMin) ? next.intervalMin : 'custom'
    advanced.value = frequency.value === 'custom'
    if (next.lastStatus === 'running') {
      resume()
    }
  }
  catch (e: any) {
    if (version === requestVersion && visible.value) {
      loadError.value = e?.message || i18n.global.t('container.apps.autoUpdate.loadFailed')
    }
  }
  finally {
    if (version === requestVersion) {
      loading.value = false
    }
  }
}

watch(() => [visible.value, props.project, props.nodeId], () => {
  if (visible.value) {
    void load()
  }
  else {
    requestVersion++
    pause()
  }
}, { immediate: true })

watch(frequency, (value) => {
  if (value === 'custom') {
    advanced.value = true
  }
})

async function save() {
  if (saving.value || loading.value || loadError.value || !config.value) {
    return
  }
  if (!intervalValid.value) {
    toast.error(i18n.global.t('container.apps.autoUpdate.intervalInvalid'))
    return
  }
  saving.value = true
  try {
    await apiCompose.autoUpdateSave({ ...form.value, project: props.project, dir: props.dir, nodeId: props.nodeId, intervalMin: intervalMin.value })
    toast.success(i18n.global.t(form.value.enabled ? 'container.apps.autoUpdate.enabledSaved' : 'container.apps.autoUpdate.disabledSaved'))
    visible.value = false
  }
  catch (e: any) {
    toast.error(e?.message || i18n.global.t('container.apps.autoUpdate.saveFailed'))
  }
  finally {
    saving.value = false
  }
}
</script>

<template>
  <FaModal
    v-model="visible" :title="$t('container.apps.autoUpdateTitle', { name: project })"
    class="max-w-2xl!" :closable="!saving" :close-on-click-overlay="false" :close-on-press-escape="!saving"
  >
    <div v-if="loading" class="text-sm text-muted-foreground py-12 flex gap-2 items-center justify-center">
      <FaIcon name="i-lucide:loader-circle" class="animate-spin" /> {{ $t('common.loading') }}
    </div>
    <div v-else-if="loadError" class="text-sm p-4 border border-destructive/30 rounded-lg bg-destructive/5 space-y-3">
      <p class="text-destructive break-words">
        {{ loadError }}
      </p>
      <FaButton variant="outline" size="sm" @click="load()">
        {{ $t('container.apps.autoUpdate.retry') }}
      </FaButton>
    </div>
    <div v-else-if="config" class="space-y-5">
      <div class="p-4 border rounded-lg bg-muted/30 flex gap-4 items-center justify-between">
        <div class="min-w-0">
          <p class="font-medium">
            {{ $t('container.apps.autoUpdate.enable') }}
          </p>
          <p class="text-xs text-muted-foreground leading-relaxed mt-1">
            {{ $t('container.apps.autoUpdate.enableHint') }}
          </p>
        </div>
        <FaSwitch v-model="form.enabled" :disabled="saving" :aria-label="$t('container.apps.autoUpdate.enable')" />
      </div>

      <div class="space-y-2">
        <p class="text-sm font-medium">
          {{ $t('container.apps.autoUpdate.frequency') }}
        </p>
        <YdSelect v-model="frequency" :options="frequencyOptions" :disabled="saving" size="default" button-class="w-full" />
        <p class="text-xs text-muted-foreground leading-relaxed">
          {{ $t('container.apps.autoUpdate.frequencyHint') }}
        </p>
      </div>

      <div class="border rounded-lg">
        <FaButton variant="ghost" class="rounded-lg w-full justify-between p-3! h-auto!" :aria-expanded="advanced" @click="advanced = !advanced">
          <span class="flex gap-2 items-center"><FaIcon name="i-lucide:sliders-horizontal" /> {{ $t('container.apps.autoUpdate.advanced') }}</span>
          <FaIcon name="i-lucide:chevron-down" class="transition-transform" :class="{ 'rotate-180': advanced }" />
        </FaButton>
        <div v-if="advanced" class="p-4 border-t space-y-4">
          <div v-if="frequency === 'custom'" class="space-y-2">
            <p class="text-sm font-medium">
              {{ $t('container.apps.autoUpdate.customInterval') }}
            </p>
            <FaInput v-model="customInterval" :aria-label="$t('container.apps.autoUpdate.customInterval')" type="number" min="5" max="1440" step="1" class="w-full!" :disabled="saving" />
            <p class="text-xs" :class="intervalValid ? 'text-muted-foreground' : 'text-destructive'">
              {{ $t('container.apps.autoUpdate.intervalInvalid') }}
            </p>
          </div>
          <div class="space-y-2">
            <p class="text-sm font-medium">
              {{ $t('container.apps.autoUpdate.preHook') }}
            </p>
            <FaTextarea v-model="form.preHook" :aria-label="$t('container.apps.autoUpdate.preHook')" rows="3" maxlength="8192" class="w-full" input-class="font-mono text-xs" :placeholder="$t('container.apps.autoUpdate.preHookPlaceholder')" :disabled="saving" />
            <p class="text-xs text-muted-foreground">
              {{ $t('container.apps.autoUpdate.preHookHint') }}
            </p>
          </div>
          <div class="space-y-2">
            <p class="text-sm font-medium">
              {{ $t('container.apps.autoUpdate.postHook') }}
            </p>
            <FaTextarea v-model="form.postHook" :aria-label="$t('container.apps.autoUpdate.postHook')" rows="3" maxlength="8192" class="w-full" input-class="font-mono text-xs" :placeholder="$t('container.apps.autoUpdate.postHookPlaceholder')" :disabled="saving" />
          </div>
          <div class="space-y-2">
            <p class="text-sm font-medium">
              {{ $t('container.apps.autoUpdate.failurePolicy') }}
            </p>
            <YdSelect v-model="failurePolicy" :options="failureOptions" :disabled="saving" size="default" button-class="w-full" />
            <p class="text-xs text-muted-foreground leading-relaxed">
              {{ $t('container.apps.autoUpdate.failureHint') }}
            </p>
          </div>
          <div class="space-y-2">
            <p class="text-sm font-medium">
              {{ $t('container.apps.autoUpdate.remark') }}
            </p>
            <FaInput v-model="form.remark" :aria-label="$t('container.apps.autoUpdate.remark')" maxlength="255" class="w-full!" :disabled="saving" />
          </div>
        </div>
      </div>

      <div class="text-sm p-4 border rounded-lg bg-muted/20 space-y-3">
        <div class="flex flex-wrap gap-2 items-center justify-between">
          <p class="font-medium">
            {{ $t('container.apps.autoUpdate.recentActivity') }}
          </p>
          <div class="flex gap-1 items-center">
            <FaButton variant="ghost" size="sm" :loading="refreshing" :disabled="saving" @click="refreshStatus()">
              <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('container.apps.autoUpdate.refresh') }}
            </FaButton>
            <FaButton v-if="config.lastTaskId" variant="outline" size="sm" @click="useTaskCenterStore().open(config.lastTaskId)">
              <FaIcon name="i-lucide:scroll-text" class="mr-1" /> {{ $t('container.apps.autoUpdate.viewLog') }}
            </FaButton>
          </div>
        </div>
        <p v-if="statusError" class="text-xs text-destructive break-words">
          {{ statusError }}
        </p>
        <dl class="gap-2 grid grid-cols-1 sm:gap-x-4 sm:grid-cols-[auto_1fr]">
          <dt class="text-muted-foreground">
            {{ $t('container.apps.autoUpdate.lastChecked') }}
          </dt>
          <dd>{{ formatTime(config.lastCheckedAt) }}</dd>
          <dt class="text-muted-foreground">
            {{ $t('container.apps.autoUpdate.nextCheck') }}
          </dt>
          <dd>{{ config.enabled ? (config.nextCheckAt ? formatTime(config.nextCheckAt) : $t('container.apps.autoUpdate.pending')) : $t('container.apps.autoUpdate.off') }}</dd>
          <dt class="text-muted-foreground">
            {{ $t('container.apps.autoUpdate.lastResult') }}
          </dt>
          <dd class="break-words" :class="{ 'text-destructive': config.lastStatus === 'failed', 'text-primary': config.lastStatus === 'running' }">
            {{ config.lastResult || $t('container.apps.autoUpdate.noHistory') }}
          </dd>
        </dl>
      </div>
    </div>
    <template #footer>
      <FaButton variant="outline" :disabled="saving" @click="visible = false">
        {{ $t('common.cancel') }}
      </FaButton>
      <FaButton :loading="saving" :disabled="loading || !!loadError || !config || !intervalValid" @click="save()">
        {{ $t('common.save') }}
      </FaButton>
    </template>
  </FaModal>
</template>
