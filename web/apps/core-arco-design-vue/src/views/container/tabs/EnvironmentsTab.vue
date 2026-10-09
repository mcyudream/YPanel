<script setup lang="ts">
import type { DockerEnvironment, EnvContainer, EnvImage } from '@/api/modules/dockerenv'
import { dockerEnvApi } from '@/api/modules/dockerenv'
import { i18n, tr } from '@/locales'

defineOptions({
  name: 'ContainerEnvironments',
})

const toast = useFaToast()
const modal = useFaModal()

const envs = ref<DockerEnvironment[]>([])
const loading = ref(false)
const testingId = ref(0)
const activeId = ref<number>(0)
const envTab = ref<'containers' | 'images'>('containers')
const envContainers = ref<EnvContainer[]>([])
const envImages = ref<EnvImage[]>([])
const envLoading = ref(false)
const acting = ref('')

async function load() {
  loading.value = true
  try {
    envs.value = await dockerEnvApi.list()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.envs.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

// ---- 环境 CRUD ----
const dialogVisible = ref(false)
const editing = ref<DockerEnvironment | null>(null)
const saving = ref(false)

type EnvInput = {
  name: string
  type: 'local' | 'tcp'
  endpoint: string
  tlsEnable: boolean
  tlsCa: string
  tlsCert: string
  tlsKey: string
  remark: string
}

const form = ref<EnvInput>({ name: '', type: 'tcp', endpoint: '', tlsEnable: false, tlsCa: '', tlsCert: '', tlsKey: '', remark: '' })

function openCreate() {
  editing.value = null
  form.value = { name: '', type: 'tcp', endpoint: '', tlsEnable: false, tlsCa: '', tlsCert: '', tlsKey: '', remark: '' }
  dialogVisible.value = true
}

function openEdit(e: DockerEnvironment) {
  editing.value = e
  form.value = { name: e.name, type: e.type, endpoint: e.endpoint, tlsEnable: e.tlsEnable, tlsCa: '', tlsCert: '', tlsKey: '', remark: e.remark }
  dialogVisible.value = true
}

async function save() {
  saving.value = true
  try {
    if (editing.value) {
      await dockerEnvApi.update(editing.value.id, form.value)
      toast.success(i18n.global.t('container.envs.saved'))
    }
    else {
      await dockerEnvApi.create(form.value)
      toast.success(i18n.global.t('container.envs.created'))
    }
    dialogVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.saveFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

async function testEnv(e: DockerEnvironment) {
  testingId.value = e.id
  try {
    const out = await dockerEnvApi.test(e.id)
    const detail = `${out.latencyMs}ms${out.apiVersion ? ` · API ${out.apiVersion}` : ''}`
    toast.success(i18n.global.t('container.envs.testOk', { detail }))
  }
  catch (err: any) {
    toast.error(i18n.global.t('container.envs.testFailed'), { description: err?.message })
  }
  finally {
    testingId.value = 0
  }
}

function removeEnv(e: DockerEnvironment) {
  modal.confirm({
    title: i18n.global.t('container.envs.deleteTitle'),
    content: i18n.global.t('container.envs.deleteConfirm', { name: e.name }),
    onConfirm: async () => {
      await dockerEnvApi.remove(e.id)
      toast.success(i18n.global.t('container.common.deletedDone'))
      if (activeId.value === e.id) {
        activeId.value = 0
      }
      await load()
    },
  })
}

// ---- 环境资源 ----
async function selectEnv(e: DockerEnvironment) {
  if (e.type === 'local') {
    toast.info(i18n.global.t('container.envs.localHint'))
    return
  }
  activeId.value = e.id
  await loadEnvResources()
}

async function loadEnvResources() {
  if (!activeId.value) {
    return
  }
  envLoading.value = true
  try {
    if (envTab.value === 'containers') {
      envContainers.value = await dockerEnvApi.containers(activeId.value)
    }
    else {
      envImages.value = await dockerEnvApi.images(activeId.value)
    }
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.envs.resLoadFailed'), { description: e?.message })
  }
  finally {
    envLoading.value = false
  }
}

watch(envTab, () => loadEnvResources())

async function envAction(c: EnvContainer, action: string) {
  acting.value = c.id + action
  try {
    await dockerEnvApi.containerAction(activeId.value, c.name, action)
    const msgKey = action === 'start' ? 'container.common.started' : action === 'stop' ? 'container.common.stopped' : action === 'restart' ? 'container.common.restarted' : 'container.common.deletedName'
    toast.success(i18n.global.t(msgKey, { name: c.name }))
    await loadEnvResources()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.opFailed'), { description: e?.message })
  }
  finally {
    acting.value = ''
  }
}

function removeImage(img: EnvImage) {
  modal.confirm({
    title: i18n.global.t('container.common.deleteImageTitle'),
    content: i18n.global.t('container.envs.deleteImageConfirm', { name: img.tags[0] || img.id }),
    onConfirm: async () => {
      try {
        await dockerEnvApi.imageRemove(activeId.value, img.tags[0] || img.id)
        toast.success(i18n.global.t('container.common.deletedDone'))
        await loadEnvResources()
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.common.deleteFailed'), { description: e?.message })
      }
    },
  })
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <FaIcon name="i-lucide:server-cog" :size="22" />
          <span>{{ $t('env.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('env.desc') }}</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('env.connect') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="e in envs" :key="e.id"
          class="cursor-pointer rounded-lg border p-4 transition-colors hover:bg-accent/30"
          :class="activeId === e.id ? 'border-primary/50 bg-primary/5' : ''"
          @click="selectEnv(e)"
        >
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-2">
              <span class="inline-flex size-8 items-center justify-center rounded-md bg-primary/10 text-primary">
                <FaIcon name="i-lucide:server" class="text-sm" />
              </span>
              <div>
                <div class="font-medium">{{ e.name }}</div>
                <div class="font-mono text-xs text-muted-foreground">
                  {{ e.type === 'local' ? $t('container.envs.localAgent') : `${e.tlsEnable ? 'tcps' : 'tcp'}://${e.endpoint}` }}
                </div>
              </div>
            </div>
            <span class="rounded-full px-2 py-0.5 text-xs" :class="e.type === 'local' ? 'bg-emerald-500/10 text-emerald-600' : 'bg-sky-500/10 text-sky-600'">
              {{ e.type === 'local' ? $t('env.local') : $t('env.remote') }}
            </span>
          </div>
          <div v-if="e.remark" class="mt-2 text-xs text-muted-foreground">
            {{ e.remark }}
          </div>
          <div v-if="e.type !== 'local'" class="mt-3 flex items-center gap-1 border-t pt-2" @click.stop>
            <FaButton variant="ghost" size="sm" :loading="testingId === e.id" @click="testEnv(e)">
              {{ $t('common.test') }}
            </FaButton>
            <FaButton variant="ghost" size="sm" @click="openEdit(e)">
              {{ $t('common.edit') }}
            </FaButton>
            <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeEnv(e)">
              {{ $t('common.delete') }}
            </FaButton>
          </div>
        </div>
      </div>

      <!-- 远程环境资源视图 -->
      <div v-if="activeId" class="mt-5">
        <div class="mb-2 flex items-center gap-2">
          <FaTabs
            v-model="envTab" :list="[
              { label: $t('container.common.containers'), value: 'containers' },
              { label: $t('container.common.images'), value: 'images' },
            ]"
          />
          <FaButton variant="ghost" size="icon-sm" :title="$t('common.refresh')" @click="loadEnvResources">
            <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="envLoading ? 'animate-spin' : ''" />
          </FaButton>
        </div>
        <div class="overflow-x-auto rounded-lg border">
          <table v-if="envTab === 'containers'" class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">{{ $t('container.common.containers') }}</th>
                <th class="px-3 py-2">{{ $t('container.common.image') }}</th>
                <th class="px-3 py-2">{{ $t('common.status') }}</th>
                <th class="hidden px-3 py-2 lg:table-cell">{{ $t('container.common.ports') }}</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="envLoading && !envContainers.length">
                <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
                  {{ $t('common.loading') }}
                </td>
              </tr>
              <tr v-else-if="!envContainers.length">
                <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
                  {{ $t('container.common.noContainers') }}
                </td>
              </tr>
              <tr v-for="c in envContainers" :key="c.id" class="border-t hover:bg-accent/30">
                <td class="px-3 py-2 font-mono text-[13px]">
                  {{ c.name }}
                </td>
                <td class="max-w-48 truncate px-3 py-2 font-mono text-xs text-muted-foreground" :title="c.image">
                  {{ c.image }}
                </td>
                <td class="px-3 py-2">
                  <span class="rounded-full px-2 py-0.5 text-xs" :class="c.state === 'running' ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                    {{ tr(`container.state.${c.state}`) }}
                  </span>
                </td>
                <td class="hidden px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell">
                  {{ c.ports.join(' ') || '—' }}
                </td>
                <td class="px-3 py-2 text-right">
                  <FaButton v-if="c.state !== 'running'" variant="outline" size="sm" @click="envAction(c, 'start')">
                    {{ $t('common.start') }}
                  </FaButton>
                  <template v-else>
                    <FaButton variant="outline" size="sm" @click="envAction(c, 'stop')">
                      {{ $t('common.stop') }}
                    </FaButton>
                    <FaButton variant="ghost" size="sm" @click="envAction(c, 'restart')">
                      {{ $t('common.restart') }}
                    </FaButton>
                  </template>
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="envAction(c, 'remove')">
                    {{ $t('common.delete') }}
                  </FaButton>
                </td>
              </tr>
            </tbody>
          </table>
          <table v-else class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">TAG</th>
                <th class="px-3 py-2">ID</th>
                <th class="px-3 py-2">{{ $t('common.size') }}</th>
                <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="envLoading && !envImages.length">
                <td colspan="4" class="px-3 py-8 text-center text-muted-foreground">
                  {{ $t('common.loading') }}
                </td>
              </tr>
              <tr v-else-if="!envImages.length">
                <td colspan="4" class="px-3 py-8 text-center text-muted-foreground">
                  {{ $t('container.common.noImages') }}
                </td>
              </tr>
              <tr v-for="img in envImages" :key="img.id" class="border-t hover:bg-accent/30">
                <td class="px-3 py-2 font-mono text-[13px]">
                  {{ img.tags[0] || '<none>' }}
                </td>
                <td class="px-3 py-2 font-mono text-xs text-muted-foreground">
                  {{ img.id }}
                </td>
                <td class="px-3 py-2 text-xs tabular-nums">
                  {{ img.sizeMb.toFixed(0) }} MB
                </td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeImage(img)">
                    {{ $t('common.delete') }}
                  </FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </FaPageMain>

    <!-- 接入/编辑环境 -->
    <FaModal v-model="dialogVisible" :title="editing ? $t('container.envs.editTitle', { name: editing.name }) : $t('container.envs.createTitle')" :destroy-on-close="true">
      <div class="space-y-3">
        <div class="grid grid-cols-2 gap-3">
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('common.name') }}</span>
            <FaInput v-model="form.name" :placeholder="$t('container.envs.namePlaceholder')" class="w-full" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.envs.endpointLabel') }}</span>
            <FaInput v-model="form.endpoint" placeholder="192.168.1.10:2375" class="w-full" />
          </label>
        </div>
        <label class="flex items-center gap-2">
          <input v-model="form.tlsEnable" type="checkbox" class="accent-[var(--primary)]">
          <span class="text-sm">{{ $t('container.envs.tlsLabel') }}</span>
        </label>
        <template v-if="form.tlsEnable">
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.envs.caLabel') }}</span>
            <textarea v-model="form.tlsCa" class="h-20 w-full resize-y rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" spellcheck="false" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.envs.certLabel') }}{{ editing ? $t('container.envs.keepOriginal') : '' }}</span>
            <textarea v-model="form.tlsCert" class="h-20 w-full resize-y rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" spellcheck="false" />
          </label>
          <label class="block space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.envs.keyLabel') }}{{ editing ? $t('container.envs.keepOriginal') : '' }}</span>
            <textarea v-model="form.tlsKey" class="h-20 w-full resize-y rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" spellcheck="false" />
          </label>
        </template>
        <label class="block space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('common.remark') }}</span>
          <FaInput v-model="form.remark" class="w-full" />
        </label>
        <div class="text-xs text-muted-foreground">
          {{ $t('container.envs.tcpHint') }}
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="dialogVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="saving" @click="save">
          {{ $t('common.save') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
