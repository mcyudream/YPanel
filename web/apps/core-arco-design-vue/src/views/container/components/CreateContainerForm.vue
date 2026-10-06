<script setup lang="ts">
import type { ContainerCreateReq } from '@/api/modules/container'
import apiContainer from '@/api/modules/container'
import { dockerExtApi, type DockerNetwork, type DockerVolume } from '@/api/modules/dockerext'

// 结构化创建容器（M23 对标 Portainer Add container）：分组表单 + 行编辑器 + 高级折叠区。
const emit = defineEmits<{
  created: [id: string]
}>()

const visible = defineModel<boolean>({ default: false })

const toast = useFaToast()
const router = useRouter()

const saving = ref(false)
const networks = ref<DockerNetwork[]>([])
const volumes = ref<DockerVolume[]>([])

const form = ref(emptyForm())

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

watch(visible, async (v) => {
  if (v) {
    form.value = emptyForm()
    // 网络与卷下拉数据（失败不阻塞表单）
    dockerExtApi.networks().then(n => networks.value = n).catch(() => {})
    dockerExtApi.volumes().then(v2 => volumes.value = v2).catch(() => {})
  }
})

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

async function submit() {
  const f = form.value
  if (!f.name.trim() || !f.image.trim()) {
    toast.error('容器名与镜像必填')
    return
  }
  if (!/^[a-z0-9][a-z0-9_.-]{0,62}$/.test(f.name.trim())) {
    toast.error('容器名仅允许小写字母/数字，含 . _ -')
    return
  }
  for (const p of f.ports) {
    if (!p.host.trim() || !p.container.trim()) {
      toast.error('端口映射的宿主与容器端口都要填写')
      return
    }
  }
  for (const m of f.mounts) {
    if (!m.source.trim() || !m.target.trim()) {
      toast.error('挂载的来源与目标路径都要填写')
      return
    }
  }
  const req: ContainerCreateReq = {
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
  saving.value = true
  try {
    const id = await apiContainer.create(req)
    toast.success(`容器已创建：${id.slice(0, 12)}`)
    visible.value = false
    emit('created', id)
    router.push(`/container/detail/${id}`)
  }
  catch (e: any) {
    toast.error('创建失败', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}
</script>

<template>
  <FaModal v-model="visible" title="创建容器" class="max-w-3xl!" :destroy-on-close="true">
    <div class="flex flex-col gap-4 text-sm">
      <!-- 基础 -->
      <section class="space-y-3">
        <div class="text-xs font-medium text-muted-foreground">
          基础
        </div>
        <div class="grid grid-cols-2 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">容器名</span>
            <FaInput v-model="form.name" placeholder="my-container" class="w-full" />
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">镜像</span>
            <FaInput v-model="form.image" placeholder="redis:7-alpine" class="w-full" />
          </label>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">启动命令（可选，空格分隔）</span>
            <FaInput v-model="form.cmdRaw" placeholder="redis-server --appendonly yes" class="w-full" />
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">工作目录（可选）</span>
            <FaInput v-model="form.workdir" placeholder="/data" class="w-full" />
          </label>
        </div>
        <div class="grid grid-cols-3 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">网络</span>
            <select v-model="form.network" class="h-9 w-full rounded-md border bg-background px-2 text-sm outline-none">
              <option value="bridge">bridge</option>
              <option value="host">host</option>
              <option v-for="n in networks.filter(n => n.name !== 'bridge' && n.name !== 'host')" :key="n.id" :value="n.name">
                {{ n.name }}
              </option>
            </select>
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">重启策略</span>
            <select v-model="form.restart" class="h-9 w-full rounded-md border bg-background px-2 text-sm outline-none">
              <option value="no">不重启</option>
              <option value="always">always</option>
              <option value="unless-stopped">unless-stopped</option>
              <option value="on-failure">on-failure</option>
            </select>
          </label>
          <label class="flex items-center gap-2 pt-5 text-sm">
            <input v-model="form.tty" type="checkbox" class="accent-[var(--primary)]">
            分配 TTY（可直接 exec）
          </label>
        </div>
      </section>

      <!-- 端口映射 -->
      <section class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">端口映射</span>
          <FaButton variant="outline" size="sm" @click="addRow(form.ports, () => ({ host: '', container: '', proto: 'tcp' as const }))">
            <FaIcon name="i-lucide:plus" class="mr-1" /> 添加
          </FaButton>
        </div>
        <div v-for="(p, i) in form.ports" :key="i" class="flex items-center gap-2">
          <FaInput v-model="p.host" placeholder="宿主端口 8080" class="w-36" />
          <span class="text-muted-foreground">→</span>
          <FaInput v-model="p.container" placeholder="容器端口 80" class="w-36" />
          <select v-model="p.proto" class="h-9 rounded-md border bg-background px-2 text-sm outline-none">
            <option value="tcp">tcp</option>
            <option value="udp">udp</option>
          </select>
          <FaButton variant="ghost" size="icon-sm" class="text-red-500!" @click="form.ports.splice(i, 1)">
            <FaIcon name="i-lucide:trash-2" class="text-sm" />
          </FaButton>
        </div>
        <div v-if="!form.ports.length" class="text-xs text-muted-foreground">
          暂不映射端口
        </div>
      </section>

      <!-- 挂载 -->
      <section class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">存储挂载</span>
          <FaButton variant="outline" size="sm" @click="addRow(form.mounts, () => ({ type: 'bind' as const, source: '', target: '', readonly: false }))">
            <FaIcon name="i-lucide:plus" class="mr-1" /> 添加
          </FaButton>
        </div>
        <div v-for="(m, i) in form.mounts" :key="i" class="flex items-center gap-2">
          <select v-model="m.type" class="h-9 w-24 rounded-md border bg-background px-2 text-sm outline-none">
            <option value="bind">宿主目录</option>
            <option value="volume">卷</option>
          </select>
          <FaInput v-if="m.type === 'bind'" v-model="m.source" placeholder="/srv/data" class="w-56" />
          <select v-else v-model="m.source" class="h-9 w-56 rounded-md border bg-background px-2 text-sm outline-none">
            <option value="" disabled>
              选择卷…
            </option>
            <option v-for="v in volumes" :key="v.name" :value="v.name">
              {{ v.name }}
            </option>
          </select>
          <span class="text-muted-foreground">→</span>
          <FaInput v-model="m.target" placeholder="容器内路径 /data" class="w-56" />
          <label class="flex items-center gap-1 text-xs text-muted-foreground">
            <input v-model="m.readonly" type="checkbox" class="accent-[var(--primary)]"> 只读
          </label>
          <FaButton variant="ghost" size="icon-sm" class="text-red-500!" @click="form.mounts.splice(i, 1)">
            <FaIcon name="i-lucide:trash-2" class="text-sm" />
          </FaButton>
        </div>
        <div v-if="!form.mounts.length" class="text-xs text-muted-foreground">
          暂不挂载（容器数据存于可写层，删除即失）
        </div>
      </section>

      <!-- 环境变量 -->
      <section class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">环境变量</span>
          <div class="flex gap-1.5">
            <FaButton variant="outline" size="sm" @click="pasteEnvVisible = true">
              批量粘贴
            </FaButton>
            <FaButton variant="outline" size="sm" @click="addRow(form.env, () => ({ name: '', value: '' }))">
              <FaIcon name="i-lucide:plus" class="mr-1" /> 添加
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
          暂无环境变量
        </div>
      </section>

      <!-- 高级 -->
      <details class="rounded-md border p-3">
        <summary class="cursor-pointer text-xs font-medium text-muted-foreground">
          高级选项（资源限制 / 特权 / labels / entrypoint）
        </summary>
        <div class="mt-3 grid grid-cols-2 gap-3">
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">内存上限（MB，0=不限）</span>
            <FaInput v-model.number="form.memoryMB" type="number" :min="0" class="w-full" />
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">CPU 核数上限（0=不限）</span>
            <FaInput v-model.number="form.cpus" type="number" :min="0" :step="0.1" class="w-full" />
          </label>
        </div>
        <div class="mt-3 grid grid-cols-2 gap-3">
          <label class="flex items-center gap-2 text-sm">
            <input v-model="form.privileged" type="checkbox" class="accent-[var(--primary)]">
            特权模式（privileged）
          </label>
          <label class="space-y-1">
            <span class="text-xs text-muted-foreground">Entrypoint（可选，空格分隔）</span>
            <FaInput v-model="form.entrypointRaw" class="w-full" />
          </label>
        </div>
        <div class="mt-3 space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-xs text-muted-foreground">Labels</span>
            <FaButton variant="outline" size="sm" @click="addRow(form.labels, () => ({ name: '', value: '' }))">
              <FaIcon name="i-lucide:plus" class="mr-1" /> 添加
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
    </div>

    <template #footer>
      <FaButton variant="outline" @click="visible = false">
        取消
      </FaButton>
      <FaButton :loading="saving" @click="submit">
        创建并启动
      </FaButton>
    </template>
  </FaModal>

  <!-- env 批量粘贴 -->
  <FaModal v-model="pasteEnvVisible" title="批量粘贴环境变量" class="max-w-lg!" :destroy-on-close="true">
    <div class="text-xs text-muted-foreground">
      每行一条 KEY=VALUE，# 开头行忽略
    </div>
    <textarea
      v-model="pasteEnvRaw"
      rows="8"
      placeholder="TZ=Asia/Shanghai&#10;MYSQL_ROOT_PASSWORD=secret"
      class="mt-2 w-full rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary"
    />
    <template #footer>
      <FaButton variant="outline" @click="pasteEnvVisible = false">
        取消
      </FaButton>
      <FaButton @click="doPasteEnv">
        解析添加
      </FaButton>
    </template>
  </FaModal>
</template>
