<script setup lang="ts">
import apiFile from '@/api/modules/file'
import { dockerExtApi } from '@/api/modules/dockerext'
import FileEditorWorkspace from '@/views/file_management/editor/Workspace.vue'

// Docker 配置（M23）：daemon.json 统一走文件工作台编辑（保存自动存版本），此处负责重启生效与仓库管理。
const toast = useFaToast()
const fileEditorStore = useFileEditorStore()

const DAEMON_PATH = '/etc/docker/daemon.json'

const daemonContent = ref('')
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    daemonContent.value = await dockerExtApi.daemonConfig()
  }
  catch (e: any) {
    toast.error('daemon.json 读取失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  loadRegistries()
})

function openEdit() {
  fileEditorStore.openWorkspace(DAEMON_PATH, 'local')
}

// 工作台保存仅写盘（会自动存版本快照）；配置要生效需重启 Docker：
// 用「原内容写回 + 重启」的既有接口触发（PUT daemon-config = 保存 + systemctl restart docker）
const restarting = ref(false)

function restartDocker() {
  const modal = useFaModal()
  modal.confirm({
    title: '重启 Docker',
    content: '重启 Docker 守护进程会使全部容器短暂中断（配置自动重启策略的容器会自动拉起）。确认继续？',
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
        toast.success('Docker 正在重启，稍候刷新页面查看状态')
      }
      catch (e: any) {
        toast.error('重启失败', { description: e?.message })
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
    toast.success('仓库凭据已保存')
    regVisible.value = false
    regForm.value = { registry: '', username: '', password: '' }
    await loadRegistries()
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
}

function removeRegistry(reg: string) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除仓库凭据',
    content: `确认删除 ${reg} 的登录凭据？`,
    onConfirm: async () => {
      try {
        await dockerExtApi.removeRegistry(reg)
        toast.success('已删除')
        await loadRegistries()
      }
      catch (e: any) {
        toast.error('删除失败', { description: e?.message })
      }
    },
  })
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <!-- daemon.json -->
    <section class="rounded-lg border bg-background p-4">
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-2">
          <FaIcon name="i-lucide:file-cog" class="text-base text-primary opacity-70" />
          <span class="font-medium">Docker 守护进程配置</span>
          <span class="font-mono text-xs text-muted-foreground">{{ DAEMON_PATH }}</span>
        </div>
        <div class="ml-auto flex items-center gap-2">
          <FaButton variant="outline" size="sm" @click="load()">
            <FaIcon name="i-lucide:refresh-cw" class="mr-1" :class="loading ? 'animate-spin' : ''" /> 重新读取
          </FaButton>
          <FaButton variant="outline" size="sm" @click="restartDocker" :loading="restarting">
            <FaIcon name="i-lucide:rotate-cw" class="mr-1" /> 重启 Docker 生效
          </FaButton>
          <FaButton size="sm" @click="openEdit">
            <FaIcon name="i-lucide:pen-line" class="mr-1" /> 编辑配置
          </FaButton>
        </div>
      </div>
      <div class="mt-3 rounded-md bg-muted/50 p-3 text-xs leading-relaxed text-muted-foreground">
        点击「编辑配置」在文件工作台打开 daemon.json（JSON 高亮 + 保存自动存版本快照）。
        工作台保存只写盘，<b>需点击「重启 Docker 生效」</b>重启守护进程后配置才应用（全部容器会短暂中断）。
        镜像加速源在 registry-mirrors 字段内编辑。
      </div>
      <pre class="mt-3 max-h-56 overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs">{{ daemonContent || '（空）' }}</pre>
    </section>

    <!-- 镜像仓库 -->
    <section class="rounded-lg border bg-background p-4">
      <div class="flex items-center gap-2">
        <FaIcon name="i-lucide:server" class="text-base text-primary opacity-70" />
        <span class="font-medium">镜像仓库凭据</span>
        <span class="text-xs text-muted-foreground">拉取私有镜像时使用（写入目标机 ~/.docker/config.json）</span>
        <FaButton class="ml-auto" size="sm" @click="regVisible = true">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 添加仓库
        </FaButton>
      </div>
      <div class="mt-3 overflow-hidden rounded-md border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">仓库地址</th>
              <th class="px-3 py-2">用户名</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!registries.length">
              <td colspan="3" class="px-3 py-6 text-center text-muted-foreground">
                未配置私有仓库（公共镜像无需配置）
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
                  删除
                </FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- 添加仓库 -->
    <FaModal v-model="regVisible" title="添加镜像仓库" class="max-w-lg!" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <label class="space-y-1">
          <span class="text-xs text-muted-foreground">仓库地址</span>
          <FaInput v-model="regForm.registry" placeholder="registry.example.com" class="w-full" />
        </label>
        <label class="space-y-1">
          <span class="text-xs text-muted-foreground">用户名</span>
          <FaInput v-model="regForm.username" class="w-full" />
        </label>
        <label class="space-y-1">
          <span class="text-xs text-muted-foreground">密码 / 访问令牌</span>
          <FaInput v-model="regForm.password" type="password" class="w-full" />
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="regVisible = false">
          取消
        </FaButton>
        <FaButton @click="saveRegistry">
          保存
        </FaButton>
      </template>
    </FaModal>

    <FileEditorWorkspace />
  </div>
</template>
