<script setup lang="ts">
import type { ContainerCreateReq } from '@/api/modules/container'
import apiContainer from '@/api/modules/container'
import { dockerExtApi, type DockerNetwork, type DockerVolume } from '@/api/modules/dockerext'
import { i18n } from '@/locales'
import { useYwEmbed } from '@/views/desktop/embed'

// 通用容器表单（create/edit 双模式，对标 Portainer duplicate/edit）：
// - create：结构化创建（原 CreateContainerForm 能力）
// - edit：从 inspect 预填；仅资源限制/重启策略变化时走 docker update 热更新（免重建），
//   其余字段变化走「删除并重建」（Docker 容器创建后 env/端口/网络/挂载不可变），保存前弹确认。
const props = withDefaults(defineProps<{
  mode?: 'create' | 'edit'
  containerId?: string
}>(), {
  mode: 'create',
  containerId: '',
})

const emit = defineEmits<{
  created: [id: string]
  saved: [id: string]
}>()

const visible = defineModel<boolean>({ default: false })

const toast = useFaToast()
const router = useRouter()
// 桌面承载：创建成功=新开应用详情窗（router.push 会顶掉 /desktop 路由）；经典模式保持跳转
const ywEmbed = useYwEmbed()

const isEdit = computed(() => props.mode === 'edit')

const saving = ref(false)
const preloading = ref(false)
const networks = ref<DockerNetwork[]>([])
const volumes = ref<DockerVolume[]>([])

// compose 编排管理的容器：重建后脱离 compose 追踪，禁止面板编辑（入口置灰，此处兜底提示）
const composeManaged = ref(false)

const form = ref(emptyForm())
// edit 预填后的基线（与提交同一 normalize 函数产物），用于变更分流
let baseline: ContainerCreateReq | null = null

interface PortRow { host: string, container: string, proto: 'tcp' | 'udp' }
interface MountRow { type: 'bind' | 'volume', source: string, target: string, readonly: boolean }
interface EnvRow { name: string, value: string }
interface LabelRow { name: string, value: string }

function emptyForm() {
  return {
    name: '',
    image: '',
    cmdRaw: '',
    workdir: '',
    tty: true,
    network: 'bridge',
    restart: 'unless-stopped',
    ports: [] as PortRow[],
    mounts: [] as MountRow[],
    env: [] as EnvRow[],
    // 高级
    memoryMB: 0,
    cpus: 0,
    privileged: false,
    entrypointRaw: '',
    labels: [] as LabelRow[],
  }
}

const netOptions = computed(() => networks.value.filter(n => n.name !== 'bridge' && n.name !== 'host').map(n => n.name))

watch(visible, async (v) => {
  if (!v) {
    return
  }
  form.value = emptyForm()
  baseline = null
  composeManaged.value = false
  // 网络与卷下拉数据（失败不阻塞表单）
  dockerExtApi.networks().then(n => networks.value = n).catch(() => {})
  dockerExtApi.volumes().then(v2 => volumes.value = v2).catch(() => {})
  if (isEdit.value && props.containerId) {
    await prefill(props.containerId)
  }
})

/** edit 模式：inspect 预填 + 记录基线 */
async function prefill(cid: string) {
  preloading.value = true
  try {
    const insp = await apiContainer.inspect(cid)
    const cfg = insp?.Config || {}
    const hc = insp?.HostConfig || {}
    composeManaged.value = !!cfg.Labels?.['com.docker.compose.project']

    const ports: PortRow[] = []
    for (const [key, binds] of Object.entries<any>(hc.PortBindings || {})) {
      const slash = key.indexOf('/')
      const cPort = slash > 0 ? key.slice(0, slash) : key
      const proto = slash > 0 ? key.slice(slash + 1) : 'tcp'
      for (const b of (binds || []).slice(0, 1)) {
        ports.push({ host: b.HostPort || '', container: cPort, proto: proto === 'udp' ? 'udp' : 'tcp' })
      }
    }

    const mounts: MountRow[] = (insp.Mounts || [])
      .filter((m: any) => m.Type === 'bind' || m.Type === 'volume')
      .map((m: any) => ({
        type: (m.Type === 'volume' ? 'volume' : 'bind') as 'bind' | 'volume',
        source: m.Type === 'volume' ? (m.Name || m.Source || '') : (m.Source || ''),
        target: m.Destination || '',
        readonly: m.RW === false,
      }))

    let network = ''
    for (const k of Object.keys(insp?.NetworkSettings?.Networks || {})) {
      network = k
      break
    }

    const env: EnvRow[] = []
    for (const e of (cfg.Env || []) as string[]) {
      const i = e.indexOf('=')
      if (i > 0) {
        env.push({ name: e.slice(0, i), value: e.slice(i + 1) })
      }
    }

    const labels: LabelRow[] = Object.entries(cfg.Labels || {}).map(([name, value]) => ({ name, value: String(value) }))

    form.value = {
      name: (insp?.Name || '').replace(/^\//, ''),
      image: cfg.Image || '',
      cmdRaw: (cfg.Cmd || []).join(' '),
      workdir: cfg.WorkingDir || '',
      tty: !!cfg.Tty,
      network,
      restart: hc.RestartPolicy?.Name && hc.RestartPolicy.Name !== 'no' ? hc.RestartPolicy.Name : 'no',
      ports,
      mounts,
      env,
      memoryMB: hc.Memory ? Math.round(hc.Memory / 1048576) : 0,
      cpus: hc.NanoCPUs ? Math.round(hc.NanoCPUs / 1e7) / 100 : 0,
      privileged: !!hc.Privileged,
      entrypointRaw: (cfg.Entrypoint || []).join(' '),
      labels,
    }
    baseline = buildReq()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.form.loadFailed'), { description: e?.message })
    visible.value = false
  }
  finally {
    preloading.value = false
  }
}

function buildReq(): ContainerCreateReq {
  const f = form.value
  return {
    name: f.name.trim(),
    image: f.image.trim(),
    cmd: f.cmdRaw.trim() ? f.cmdRaw.trim().split(/\s+/) : undefined,
    workdir: f.workdir.trim() || undefined,
    tty: f.tty,
    network: f.network.trim() || undefined,
    restart: f.restart === 'no' ? '' : f.restart,
    ports: f.ports.filter(p => p.host.trim() && p.container.trim())
      .map(p => ({ host: p.host.trim(), container: p.container.trim(), proto: p.proto })),
    mounts: f.mounts.filter(m => m.source.trim() && m.target.trim())
      .map((m) => {
        const src = m.type === 'volume' && !m.source.includes('/') ? m.source : m.source.trim()
        return `${src}:${m.target.trim()}${m.readonly ? ':ro' : ''}`
      }),
    env: f.env.filter(e => e.name.trim()).map(e => `${e.name.trim()}=${e.value}`),
    entrypoint: f.entrypointRaw.trim() ? f.entrypointRaw.trim().split(/\s+/) : undefined,
    labels: Object.fromEntries(f.labels.filter(l => l.name.trim()).map(l => [l.name.trim(), l.value])),
    privileged: f.privileged,
    memoryMB: f.memoryMB > 0 ? f.memoryMB : undefined,
    cpus: f.cpus > 0 ? f.cpus : undefined,
  }
}

function validate(f: ReturnType<typeof emptyForm>): string {
  if (!f.name.trim() || !f.image.trim()) {
    return i18n.global.t('container.form.vNameImage')
  }
  if (!/^[a-z0-9][a-z0-9_.-]{0,62}$/.test(f.name.trim())) {
    return i18n.global.t('container.form.vNameChars')
  }
  for (const p of f.ports) {
    if (!p.host.trim() || !p.container.trim()) {
      return i18n.global.t('container.form.vPort')
    }
  }
  for (const m of f.mounts) {
    if (!m.source.trim() || !m.target.trim()) {
      return i18n.global.t('container.form.vMount')
    }
  }
  return ''
}

// 变更分流签名：结构字段任一变化需重建；仅资源三项变化可热更新
function structSig(req: ContainerCreateReq) {
  return JSON.stringify([req.image, req.cmd, req.env, req.ports, req.mounts, req.network, req.entrypoint, req.workdir, req.tty, req.labels, req.privileged])
}
function resSig(req: ContainerCreateReq) {
  return JSON.stringify([req.memoryMB, req.cpus, req.restart])
}

async function submit() {
  if (isEdit.value) {
    await submitEdit()
    return
  }
  const f = form.value
  const err = validate(f)
  if (err) {
    toast.error(err)
    return
  }
  const req = buildReq()
  saving.value = true
  try {
    const id = await apiContainer.create(req)
    toast.success(i18n.global.t('container.form.created', { id: id.slice(0, 12) }))
    visible.value = false
    emit('created', id)
    if (ywEmbed) {
      ywEmbed.openApp('container-app-detail', { title: i18n.global.t('container.form.newContainer'), launchOptions: { id } })
      return
    }
    router.push(`/container/detail/${id}`)
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.createFailed'), { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

async function submitEdit() {
  if (composeManaged.value) {
    toast.error(i18n.global.t('container.form.composeManagedToast'))
    return
  }
  if (!props.containerId) {
    return
  }
  const f = form.value
  const err = validate(f)
  if (err) {
    toast.error(err)
    return
  }
  const req = buildReq()
  if (!baseline) {
    toast.error(i18n.global.t('container.form.baselineMissing'))
    return
  }
  const structChanged = structSig(req) !== structSig(baseline)
  const resChanged = resSig(req) !== resSig(baseline)

  if (!structChanged && !resChanged) {
    toast.info(i18n.global.t('container.form.noChanges'))
    return
  }

  // 仅内存/CPU/重启策略变化：docker update 免重建，立即生效
  if (!structChanged) {
    saving.value = true
    try {
      await apiContainer.updateResources(props.containerId, {
        memoryMB: req.memoryMB ?? 0,
        cpus: req.cpus ?? 0,
        restart: req.restart || 'no',
      })
      toast.success(i18n.global.t('container.form.hotUpdated'))
      visible.value = false
      emit('saved', props.containerId)
    }
    catch (e: any) {
      toast.error(i18n.global.t('container.form.hotUpdateFailed'), { description: e?.message })
    }
    finally {
      saving.value = false
    }
    return
  }

  // 结构字段变化：删除并重建
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.form.recreateTitle'),
    content: i18n.global.t('container.form.recreateConfirm', { name: f.name }),
    onConfirm: async () => {
      saving.value = true
      try {
        const newId = await apiContainer.recreate(props.containerId!, req)
        toast.success(i18n.global.t('container.form.recreated', { id: newId.slice(0, 12) }))
        visible.value = false
        emit('saved', newId)
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.form.recreateFailed'), { description: e?.message })
      }
      finally {
        saving.value = false
      }
    },
  })
}

function addRow<T>(arr: T[], factory: () => T) {
  arr.push(factory())
}

const pasteEnvVisible = ref(false)
const pasteEnvRaw = ref('')

function doPasteEnv() {
  const rows: EnvRow[] = []
  for (const line of pasteEnvRaw.value.split('\n')) {
    const s = line.trim()
    if (!s || s.startsWith('#')) {
      continue
    }
    const idx = s.indexOf('=')
    if (idx <= 0) {
      continue
    }
    rows.push({ name: s.slice(0, idx), value: s.slice(idx + 1) })
  }
  form.value.env.push(...rows)
  pasteEnvRaw.value = ''
  pasteEnvVisible.value = false
}
</script>

<template>
  <FaModal v-model="visible" :title="isEdit ? $t('container.form.editTitle') : $t('container.list.createContainer')" class="max-w-3xl!" :destroy-on-close="true">
    <div v-if="preloading" class="py-10 text-center text-sm text-muted-foreground">
      {{ $t('container.form.loading') }}
    </div>
    <div v-else class="flex flex-col gap-4 text-sm">
      <!-- compose 管理提示（编辑兜底） -->
      <div v-if="isEdit && composeManaged" class="flex items-center gap-2 rounded-md border border-amber-300 bg-amber-50 p-2.5 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
        <YdMorphIcon name="triangle-alert" :size="16" />
        {{ $t('container.form.composeBanner') }}
      </div>

      <!-- 基础 -->
      <section class="space-y-3">
        <div class="text-xs font-medium text-muted-foreground">
          {{ $t('container.form.sectionBasic') }}
        </div>
        <div class="grid grid-cols-2 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.form.nameLabel') }}{{ isEdit ? $t('container.form.nameLocked') : '' }}</span>
            <FaInput v-model="form.name" placeholder="my-container" class="w-full" :disabled="isEdit" />
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.form.imageLabel') }}{{ isEdit ? $t('container.form.imageEditable') : '' }}</span>
            <FaInput v-model="form.image" placeholder="redis:7-alpine" class="w-full" />
          </label>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.form.cmdLabel') }}</span>
            <FaInput v-model="form.cmdRaw" placeholder="redis-server --appendonly yes" class="w-full" />
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.form.workdirLabel') }}</span>
            <FaInput v-model="form.workdir" placeholder="/data" class="w-full" />
          </label>
        </div>
        <div class="grid grid-cols-3 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.form.networkLabel') }}{{ isEdit ? $t('container.form.networkEditNote') : '' }}</span>
            <select v-model="form.network" class="h-9 w-full rounded-md border bg-background px-2 text-sm outline-none">
              <option value="bridge">bridge</option>
              <option value="host">host</option>
              <option v-if="form.network && form.network !== 'bridge' && form.network !== 'host' && !netOptions.includes(form.network)" :value="form.network">
                {{ form.network }}
              </option>
              <option v-for="n in netOptions" :key="n" :value="n">
                {{ n }}
              </option>
            </select>
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.form.restartLabel') }}</span>
            <select v-model="form.restart" class="h-9 w-full rounded-md border bg-background px-2 text-sm outline-none">
              <option value="no">{{ $t('container.form.restartNo') }}</option>
              <option value="always">always</option>
              <option value="unless-stopped">unless-stopped</option>
              <option value="on-failure">on-failure</option>
            </select>
          </label>
          <label class="flex items-center gap-2 pt-5 text-sm">
            <input v-model="form.tty" type="checkbox" class="accent-[var(--primary)]">
            {{ $t('container.form.ttyLabel') }}
          </label>
        </div>
      </section>

      <!-- 端口映射 -->
      <section class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">{{ $t('container.common.portMappings') }}</span>
          <FaButton variant="outline" size="sm" @click="addRow(form.ports, () => ({ host: '', container: '', proto: 'tcp' as const }))">
            <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('common.add') }}
          </FaButton>
        </div>
        <div v-for="(p, i) in form.ports" :key="i" class="flex items-center gap-2">
          <FaInput v-model="p.host" :placeholder="$t('container.form.hostPortPlaceholder')" class="w-36" />
          <span class="text-muted-foreground">→</span>
          <FaInput v-model="p.container" :placeholder="$t('container.form.containerPortPlaceholder')" class="w-36" />
          <select v-model="p.proto" class="h-9 rounded-md border bg-background px-2 text-sm outline-none">
            <option value="tcp">tcp</option>
            <option value="udp">udp</option>
          </select>
          <FaButton variant="ghost" size="icon-sm" class="text-red-500!" @click="form.ports.splice(i, 1)">
            <FaIcon name="i-lucide:trash-2" class="text-sm" />
          </FaButton>
        </div>
        <div v-if="!form.ports.length" class="text-xs text-muted-foreground">
          {{ $t('container.form.noPorts') }}
        </div>
      </section>

      <!-- 挂载 -->
      <section class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">{{ $t('container.form.mountsTitle') }}</span>
          <FaButton variant="outline" size="sm" @click="addRow(form.mounts, () => ({ type: 'bind' as const, source: '', target: '', readonly: false }))">
            <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('common.add') }}
          </FaButton>
        </div>
        <div v-for="(m, i) in form.mounts" :key="i" class="flex items-center gap-2">
          <select v-model="m.type" class="h-9 w-24 rounded-md border bg-background px-2 text-sm outline-none">
            <option value="bind">{{ $t('container.form.bindOption') }}</option>
            <option value="volume">{{ $t('container.form.volumeOption') }}</option>
          </select>
          <FaInput v-if="m.type === 'bind'" v-model="m.source" placeholder="/srv/data" class="w-56" />
          <select v-else v-model="m.source" class="h-9 w-56 rounded-md border bg-background px-2 text-sm outline-none">
            <option value="" disabled>
              {{ $t('container.form.selectVolume') }}
            </option>
            <option v-for="v in volumes" :key="v.name" :value="v.name">
              {{ v.name }}
            </option>
          </select>
          <span class="text-muted-foreground">→</span>
          <FaInput v-model="m.target" :placeholder="$t('container.form.targetPlaceholder')" class="w-56" />
          <label class="flex items-center gap-1 text-xs text-muted-foreground">
            <input v-model="m.readonly" type="checkbox" class="accent-[var(--primary)]"> {{ $t('container.form.readonly') }}
          </label>
          <FaButton variant="ghost" size="icon-sm" class="text-red-500!" @click="form.mounts.splice(i, 1)">
            <FaIcon name="i-lucide:trash-2" class="text-sm" />
          </FaButton>
        </div>
        <div v-if="!form.mounts.length" class="text-xs text-muted-foreground">
          {{ $t('container.form.noMounts') }}
        </div>
      </section>

      <!-- 环境变量 -->
      <section class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">{{ $t('container.form.envTitle') }}</span>
          <div class="flex gap-1.5">
            <FaButton variant="outline" size="sm" @click="pasteEnvVisible = true">
              {{ $t('container.form.pasteBulk') }}
            </FaButton>
            <FaButton variant="outline" size="sm" @click="addRow(form.env, () => ({ name: '', value: '' }))">
              <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('common.add') }}
            </FaButton>
          </div>
        </div>
        <div v-for="(e, i) in form.env" :key="i" class="flex items-center gap-2">
          <FaInput v-model="e.name" placeholder="KEY" class="w-48" />
          <span class="text-muted-foreground">=</span>
          <FaInput v-model="e.value" placeholder="VALUE" class="flex-1" />
          <FaButton variant="ghost" size="icon-sm" class="text-red-500!" @click="form.env.splice(i, 1)">
            <FaIcon name="i-lucide:trash-2" class="text-sm" />
          </FaButton>
        </div>
        <div v-if="!form.env.length" class="text-xs text-muted-foreground">
          {{ $t('container.form.noEnv') }}
        </div>
      </section>

      <!-- 高级 -->
      <details class="rounded-md border p-3">
        <summary class="cursor-pointer text-xs font-medium text-muted-foreground">
          {{ $t('container.form.advanced') }}
        </summary>
        <div class="mt-3 grid grid-cols-2 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.form.memoryLabel') }}</span>
            <FaInput v-model.number="form.memoryMB" type="number" :min="0" class="w-full" />
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.form.cpuLabel') }}</span>
            <FaInput v-model.number="form.cpus" type="number" :min="0" :step="0.1" class="w-full" />
          </label>
        </div>
        <div class="mt-3 grid grid-cols-2 gap-3">
          <label class="flex items-center gap-2 text-sm">
            <input v-model="form.privileged" type="checkbox" class="accent-[var(--primary)]">
            {{ $t('container.form.privilegedLabel') }}
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">{{ $t('container.form.entrypointLabel') }}</span>
            <FaInput v-model="form.entrypointRaw" class="w-full" />
          </label>
        </div>
        <div class="mt-3 space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-xs text-muted-foreground">Labels</span>
            <FaButton variant="outline" size="sm" @click="addRow(form.labels, () => ({ name: '', value: '' }))">
              <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('common.add') }}
            </FaButton>
          </div>
          <div v-for="(l, i) in form.labels" :key="i" class="flex items-center gap-2">
            <FaInput v-model="l.name" placeholder="key" class="w-48" />
            <span class="text-muted-foreground">=</span>
            <FaInput v-model="l.value" placeholder="value" class="flex-1" />
            <FaButton variant="ghost" size="icon-sm" class="text-red-500!" @click="form.labels.splice(i, 1)">
              <FaIcon name="i-lucide:trash-2" class="text-sm" />
            </FaButton>
          </div>
        </div>
      </details>

      <div v-if="isEdit" class="rounded-md bg-muted/50 p-2.5 text-xs text-muted-foreground">
        {{ $t('container.form.editNote') }}
      </div>
    </div>

    <template #footer>
      <FaButton variant="outline" @click="visible = false">
        {{ $t('common.cancel') }}
      </FaButton>
      <FaButton :loading="saving" :disabled="preloading || (isEdit && composeManaged)" @click="submit">
        {{ isEdit ? $t('container.form.saveChanges') : $t('container.common.createAndStart') }}
      </FaButton>
    </template>
  </FaModal>

  <!-- env 批量粘贴 -->
  <FaModal v-model="pasteEnvVisible" :title="$t('container.form.pasteTitle')" class="max-w-lg!" :destroy-on-close="true">
    <div class="text-xs text-muted-foreground">
      {{ $t('container.form.pasteHint') }}
    </div>
    <textarea
      v-model="pasteEnvRaw"
      rows="8"
      placeholder="TZ=Asia/Shanghai&#10;MYSQL_ROOT_PASSWORD=secret"
      class="mt-2 w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary"
    />
    <template #footer>
      <FaButton variant="outline" @click="pasteEnvVisible = false">
        {{ $t('common.cancel') }}
      </FaButton>
      <FaButton @click="doPasteEnv">
        {{ $t('container.form.parseAdd') }}
      </FaButton>
    </template>
  </FaModal>
</template>
