<script setup lang="ts">
import apiFile from '@/api/modules/file'
import { dockerExtApi } from '@/api/modules/dockerext'
import { dockerInstallApi } from '@/api/modules/dockerinstall'
import type { DockerPrecheck } from '@/api/modules/dockerinstall'
import { taskApi } from '@/api/modules/task'
import DockerInstallWizard from '../components/DockerInstallWizard.vue'
import { dockerImgApi } from '@/api/modules/dockerenv'
import { i18n } from '@/locales'

// Docker 配置（M23）：daemon.json 统一走文件工作台编辑（保存自动存版本），此处负责重启生效与仓库管理。
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()

const DAEMON_PATH = '/etc/docker/daemon.json'

// M35：Swarm 状态
const swarm = ref<{ state: string, controlAvailable: boolean, clusterId?: string }>({ state: 'unknown', controlAvailable: false })
onMounted(() => {
  dockerImgApi.swarmStatus().then((s) => {
    swarm.value = s
  }).catch(() => {})
})

const daemonContent = ref('')
const loading = ref(false)

// ---- M52：安装状态卡 ----
const installVisible = ref(false)
const envInfo = ref<DockerPrecheck | null>(null)

async function loadEnv() {
  try {
    const res = await dockerInstallApi.precheck()
    envInfo.value = res.precheck
  }
  catch {}
}

function onInstalled() {
  loadEnv()
}

// ---- M52：镜像加速器卡 ----
const PRESET_MIRRORS = [
  { url: 'https://docker.mirrors.ustc.edu.cn', label: '中科大' },
  { url: 'https://hub-mirror.c.163.com', label: '网易' },
  { url: 'https://mirror.baidubce.com', label: '百度' },
]
const mirrors = ref<string[]>([])
const checkedMirrors = ref<string[]>([])
const customMirror = ref('')
const savingMirrors = ref(false)

async function loadMirrors() {
  try {
    mirrors.value = await dockerInstallApi.mirrors()
    // 现有值回填：预设勾选 + 剩余进自定义框
    checkedMirrors.value = mirrors.value.filter(m => PRESET_MIRRORS.some(p => p.url === m))
    const custom = mirrors.value.filter(m => !PRESET_MIRRORS.some(p => p.url === m))
    customMirror.value = custom.join(',')
  }
  catch {}
}

// 保存走任务中心：重启可达分钟级，立即返回 taskId，轮询到终态再提示结果
let mirrorTaskTimer: ReturnType<typeof setInterval> | null = null
onUnmounted(() => {
  if (mirrorTaskTimer) {
    clearInterval(mirrorTaskTimer)
  }
})

function pollMirrorTask(taskId: number) {
  if (mirrorTaskTimer) {
    clearInterval(mirrorTaskTimer)
  }
  mirrorTaskTimer = setInterval(async () => {
    let t: Awaited<ReturnType<typeof taskApi.get>>
    try {
      t = await taskApi.get(taskId)
    }
    catch {
      return // 单次轮询失败下一轮重试
    }
    if (t.status === 'running') {
      return
    }
    clearInterval(mirrorTaskTimer!)
    mirrorTaskTimer = null
    savingMirrors.value = false
    if (t.status === 'success') {
      toast.success(i18n.global.t('container.settings.mirrorApplied'))
      await loadMirrors()
      loadEnv()
    }
    else {
      toast.error(i18n.global.t('container.settings.mirrorApplyFailed'), { description: t.error })
    }
  }, 2000)
}

function saveMirrors() {
  const list = [...checkedMirrors.value]
  for (const part of customMirror.value.split(/[\s,]+/).map(x => x.trim()).filter(Boolean)) {
    if (!list.includes(part)) {
      list.push(part)
    }
  }
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.settings.mirrorSaveTitle'),
    content: i18n.global.t('container.settings.mirrorSaveConfirm'),
    onConfirm: async () => {
      savingMirrors.value = true
      try {
        const { taskId } = await dockerInstallApi.setMirrors(list)
        toast.info(i18n.global.t('container.settings.mirrorSubmitted'))
        pollMirrorTask(taskId)
      }
      catch (e: any) {
        savingMirrors.value = false
        toast.error(i18n.global.t('container.common.saveFailed'), { description: e?.message })
      }
    },
  })
}

async function load() {
  loading.value = true
  try {
    daemonContent.value = await dockerExtApi.daemonConfig()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.settings.daemonReadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  loadRegistries()
  loadEnv()
  loadMirrors()
})

function openEdit() {
  fileEditorStore.openWorkspace(DAEMON_PATH, 'local', undefined, '/etc/docker')
}

// 工作台保存仅写盘（会自动存版本快照）；配置要生效需重启 Docker：
// 用「原内容写回 + 重启」的既有接口触发（PUT daemon-config = 保存 + systemctl restart docker）
const restarting = ref(false)

function restartDocker() {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.settings.restartTitle'),
    content: i18n.global.t('container.settings.restartConfirm'),
    onConfirm: async () => {
      restarting.value = true
      try {
        // 重读盘上最新内容（可能刚在工作台保存过）再写回，确保重启时用的是新配置
        let content = daemonContent.value
        try {
          const r = await apiFile.read(DAEMON_PATH, 'local')
          if (r.content) {
            content = r.content
          }
        }
        catch {}
        await dockerExtApi.updateDaemonConfig(content)
        toast.success(i18n.global.t('container.settings.restarting'))
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.settings.restartFailed'), { description: e?.message })
      }
      finally {
        restarting.value = false
      }
    },
  })
}

// ---- 镜像仓库 ----
const registries = ref<{ registry: string, username: string }[]>([])
const regVisible = ref(false)
const regForm = ref({ registry: '', username: '', password: '' })

async function loadRegistries() {
  try {
    registries.value = await dockerExtApi.registries()
  }
  catch {}
}

async function saveRegistry() {
  try {
    await dockerExtApi.setRegistry(regForm.value.registry.trim(), regForm.value.username.trim(), regForm.value.password)
    toast.success(i18n.global.t('container.settings.registrySaved'))
    regVisible.value = false
    regForm.value = { registry: '', username: '', password: '' }
    await loadRegistries()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.saveFailed'), { description: e?.message })
  }
}

function removeRegistry(reg: string) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.settings.deleteRegTitle'),
    content: i18n.global.t('container.settings.deleteRegConfirm', { name: reg }),
    onConfirm: async () => {
      try {
        await dockerExtApi.removeRegistry(reg)
        toast.success(i18n.global.t('container.common.deletedDone'))
        await loadRegistries()
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.common.deleteFailed'), { description: e?.message })
      }
    },
  })
}
</script>

<template>
  <div>
    <FaPageMain>
  <div class="flex flex-col gap-6">
    <!-- M52：安装状态 -->
    <section class="rounded-lg border bg-background p-4">
      <div class="flex flex-wrap items-center gap-2">
        <FaIcon name="i-lucide:container" class="text-base text-primary opacity-70" />
        <span class="font-medium">{{ $t('container.settings.installTitle') }}</span>
        <template v-if="envInfo?.dockerInstalled">
          <span class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">{{ $t('container.settings.installed') }}</span>
          <span v-if="envInfo.dockerVersion" class="font-mono text-xs text-muted-foreground">{{ envInfo.dockerVersion }}</span>
          <span class="rounded-full px-2 py-0.5 text-xs" :class="envInfo.dockerRunning ? 'bg-emerald-500/10 text-emerald-600' : 'bg-amber-500/10 text-amber-600'">
            {{ envInfo.dockerRunning ? $t('container.install.running') : $t('container.install.stopped') }}
          </span>
          <span v-if="!envInfo.composeInstalled" class="rounded-full bg-amber-500/10 px-2 py-0.5 text-xs text-amber-600">{{ $t('container.settings.composeMissing') }}</span>
        </template>
        <span v-else class="rounded-full bg-amber-500/10 px-2 py-0.5 text-xs text-amber-600">{{ $t('container.settings.notInstalled') }}</span>
        <FaButton v-if="envInfo && (!envInfo.dockerInstalled || !envInfo.composeInstalled)" size="sm" class="ml-auto" @click="installVisible = true">
          <FaIcon name="i-lucide:download" class="mr-1" /> {{ $t('container.install.openWizard') }}
        </FaButton>
        <FaButton v-else variant="outline" size="sm" class="ml-auto" @click="loadEnv()">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('container.install.recheck') }}
        </FaButton>
      </div>
      <div v-if="envInfo && !envInfo.supported" class="mt-2 text-xs text-muted-foreground">{{ envInfo.reason }}</div>
    </section>

    <!-- M52：镜像加速器 -->
    <section class="rounded-lg border bg-background p-4">
      <div class="flex flex-wrap items-center gap-2">
        <FaIcon name="i-lucide:gauge" class="text-base text-primary opacity-70" />
        <span class="font-medium">{{ $t('container.settings.mirrorTitle') }}</span>
        <span class="text-xs text-muted-foreground">{{ $t('container.settings.mirrorHint') }}</span>
        <FaButton class="ml-auto" size="sm" :loading="savingMirrors" @click="saveMirrors">
          <FaIcon name="i-lucide:save" class="mr-1" /> {{ $t('common.save') }}
        </FaButton>
      </div>
      <div class="mt-3 flex flex-col gap-2">
        <div class="flex flex-wrap gap-4 text-xs">
          <label v-for="m in PRESET_MIRRORS" :key="m.url" class="flex items-center gap-1.5">
            <input v-model="checkedMirrors" type="checkbox" :value="m.url" class="accent-(--primary)">
            {{ m.label }} <span class="font-mono text-muted-foreground">{{ m.url }}</span>
          </label>
        </div>
        <FaInput v-model="customMirror" :placeholder="$t('container.settings.mirrorCustomPh')" class="w-full text-xs" />
        <p class="text-[11px] text-muted-foreground">{{ $t('container.settings.mirrorSaveHint') }}</p>
      </div>
    </section>

    <!-- M35：Swarm 状态 -->
    <section class="rounded-lg border bg-background p-4">
      <div class="flex items-center gap-2">
        <FaIcon name="i-lucide:network" class="text-base text-primary opacity-70" />
        <span class="font-medium">Swarm</span>
        <span class="rounded-full px-2 py-0.5 text-xs" :class="swarm.state === 'active' ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
          {{ swarm.state === 'unknown' ? $t('common.unknown') : swarm.state }}
        </span>
        <span v-if="swarm.controlAvailable" class="rounded-full bg-sky-500/10 px-2 py-0.5 text-xs text-sky-600">{{ $t('container.settings.managerNode') }}</span>
      </div>
      <div class="mt-2 text-xs text-muted-foreground">
        {{ $t('container.settings.swarmDesc', { state: swarm.state, cluster: swarm.clusterId ? $t('container.settings.swarmCluster', { id: String(swarm.clusterId).slice(0, 12) }) : '' }) }}
      </div>
    </section>

    <!-- daemon.json -->
    <section class="rounded-lg border bg-background p-4">
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-2">
          <FaIcon name="i-lucide:file-cog" class="text-base text-primary opacity-70" />
          <span class="font-medium">{{ $t('container.settings.daemonTitle') }}</span>
          <span class="font-mono text-xs text-muted-foreground">{{ DAEMON_PATH }}</span>
        </div>
        <div class="ml-auto flex items-center gap-2">
          <FaButton variant="outline" size="sm" @click="load()">
            <FaIcon name="i-lucide:refresh-cw" class="mr-1" :class="loading ? 'animate-spin' : ''" /> {{ $t('container.settings.reread') }}
          </FaButton>
          <FaButton variant="outline" size="sm" @click="restartDocker" :loading="restarting">
            <FaIcon name="i-lucide:rotate-cw" class="mr-1" /> {{ $t('container.settings.restartToApply') }}
          </FaButton>
          <FaButton size="sm" @click="openEdit">
            <FaIcon name="i-lucide:pen-line" class="mr-1" /> {{ $t('container.common.editConfig') }}
          </FaButton>
        </div>
      </div>
      <div class="mt-3 rounded-md bg-muted/50 p-3 text-xs leading-relaxed text-muted-foreground">
        {{ $t('container.settings.daemonHint1') }}
        {{ $t('container.settings.daemonHint2a') }}<b>{{ $t('container.settings.daemonHint2b') }}</b>{{ $t('container.settings.daemonHint2c') }}
        {{ $t('container.settings.daemonHint3') }}
      </div>
      <pre class="mt-3 max-h-56 overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs">{{ daemonContent || $t('container.settings.empty') }}</pre>
    </section>

    <!-- 镜像仓库 -->
    <section class="rounded-lg border bg-background p-4">
      <div class="flex items-center gap-2">
        <FaIcon name="i-lucide:server" class="text-base text-primary opacity-70" />
        <span class="font-medium">{{ $t('container.settings.registryTitle') }}</span>
        <span class="text-xs text-muted-foreground">{{ $t('container.settings.registryHint') }}</span>
        <FaButton class="ml-auto" size="sm" @click="regVisible = true">
          <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('container.settings.addRegistry') }}
        </FaButton>
      </div>
      <div class="mt-3 overflow-hidden rounded-md border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('container.settings.registryAddr') }}</th>
              <th class="px-3 py-2">{{ $t('container.settings.username') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!registries.length">
              <td colspan="3" class="px-3 py-6 text-center text-muted-foreground">
                {{ $t('container.settings.noRegistries') }}
              </td>
            </tr>
            <tr v-for="r in registries" :key="r.registry" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-1.5 font-mono text-[13px]">
                {{ r.registry }}
              </td>
              <td class="px-3 py-1.5 font-mono text-xs text-muted-foreground">
                {{ r.username || '—' }}
              </td>
              <td class="px-3 py-1.5 text-right">
                <FaButton variant="outline" size="sm" @click="removeRegistry(r.registry)">
                  {{ $t('common.delete') }}
                </FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- 添加仓库 -->
    <FaModal v-model="regVisible" :title="$t('container.settings.addRegistryTitle')" class="max-w-lg!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <label class="space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('container.settings.registryAddr') }}</span>
          <FaInput v-model="regForm.registry" placeholder="registry.example.com" class="w-full" />
        </label>
        <label class="space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('container.settings.username') }}</span>
          <FaInput v-model="regForm.username" class="w-full" />
        </label>
        <label class="space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('container.settings.passwordLabel') }}</span>
          <FaInput v-model="regForm.password" type="password" class="w-full" />
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="regVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton @click="saveRegistry">
          {{ $t('common.save') }}
        </FaButton>
      </template>
    </FaModal>

    <DockerInstallWizard v-model="installVisible" @installed="onInstalled" />
  </div>
    </FaPageMain>
  </div>
</template>
