<script setup lang="ts">
import api from '@/api'
import type { DockerImage, DockerNetwork, DockerVolume } from '@/api/modules/dockerext'
import { dockerExtApi } from '@/api/modules/dockerext'

defineOptions({
  name: 'DockerIndex',
})

const tab = ref<'containers' | 'images' | 'networks' | 'volumes' | 'config'>('containers')
const loading = ref(false)
const dockerAvailable = ref(true)
const dockerHint = ref('')

const containers = ref<any[]>([])
const images = ref<DockerImage[]>([])
const networks = ref<DockerNetwork[]>([])
const volumes = ref<DockerVolume[]>([])
const daemonContent = ref('')

let timer: ReturnType<typeof setInterval> | null = null

async function load() {
  loading.value = true
  try {
    if (tab.value === 'containers' || tab.value === 'config') {
      // 探测可用性走 containers 接口
      try {
        containers.value = await (await import('@/api')).default
          .get('api/v1/docker/containers', { silent: true }).then((r: any) => r.data)
        dockerAvailable.value = true
      }
      catch (e: any) {
        if (e?.code === 5002) {
          dockerAvailable.value = false
          dockerHint.value = '未检测到可用的 Docker 环境'
        }
      }
    }
    if (tab.value === 'images') {
      images.value = await dockerExtApi.images()
    }
    else if (tab.value === 'networks') {
      networks.value = await dockerExtApi.networks()
    }
    else if (tab.value === 'volumes') {
      volumes.value = await dockerExtApi.volumes()
    }
    else if (tab.value === 'config') {
      daemonContent.value = await dockerExtApi.daemonConfig()
    }
  }
  finally {
    loading.value = false
  }
}

async function refreshCurrent() {
  await load()
}

// 容器操作（沿用容器页逻辑的轻量版）
const containerActionBusy = ref('')

async function containerAction(c: any, action: string) {
  containerActionBusy.value = c.id + action
  try {
    await api.post(`api/v1/docker/containers/${c.id}/${action}`)
    await load()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
  }
  finally {
    containerActionBusy.value = ''
  }
}

// 镜像拉取
const pullVisible = ref(false)
const pullRef = ref('')
const pulling = ref(false)
const pullOutput = ref('')

async function doPull() {
  if (!pullRef.value.trim()) {
    return
  }
  pulling.value = true
  pullOutput.value = '拉取中…'
  try {
    pullOutput.value = await dockerExtApi.pull(pullRef.value.trim())
    useFaToast().success('拉取完成')
    pullVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error('拉取失败', { description: e?.message })
    pullOutput.value = e?.message || '拉取失败'
  }
  finally {
    pulling.value = false
  }
}

function removeImage(img: DockerImage) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除镜像',
    content: `确认删除镜像 ${img.tags[0] || img.id.slice(0, 12)}？`,
    onConfirm: async () => {
      try {
        await dockerExtApi.removeImage(img.id)
        useFaToast().success('已删除')
        await load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

async function pruneContainers() {
  const modal = useFaModal()
  modal.confirm({
    title: '清理停止容器',
    content: '确认清理全部已停止容器？',
    onConfirm: async () => {
      try {
        const out = await dockerExtApi.pruneContainers()
        useFaToast().success(out)
        await load()
      }
      catch (e: any) {
        useFaToast().error('清理失败', { description: e?.message })
      }
    },
  })
}

async function pruneImages() {
  const out = await dockerExtApi.pruneImages()
  useFaToast().success(out)
  await load()
}

// 网络
const netVisible = ref(false)
const netForm = ref({ name: '', driver: 'bridge' })

async function doCreateNetwork() {
  try {
    await dockerExtApi.createNetwork(netForm.value.name, netForm.value.driver)
    useFaToast().success('网络已创建')
    netVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error('创建失败', { description: e?.message })
  }
}

async function removeNetwork(n: DockerNetwork) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除网络',
    content: `确认删除网络 ${n.name}？`,
    onConfirm: async () => {
      try {
        await dockerExtApi.removeNetwork(n.name)
        useFaToast().success('已删除')
        await load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

// 卷
const volVisible = ref(false)
const volName = ref('')

async function doCreateVolume() {
  try {
    await dockerExtApi.createVolume(volName.value)
    useFaToast().success('卷已创建')
    volVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error('创建失败', { description: e?.message })
  }
}

async function removeVolume(v: DockerVolume) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除卷',
    content: `确认删除卷 ${v.name}？数据不可恢复。`,
    onConfirm: async () => {
      try {
        await dockerExtApi.removeVolume(v.name)
        useFaToast().success('已删除')
        await load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

// daemon.json 保存
const daemonSaving = ref(false)

async function saveDaemon() {
  daemonSaving.value = true
  try {
    await dockerExtApi.updateDaemonConfig(daemonContent.value)
    useFaToast().success('daemon.json 已保存，docker 重启中…')
  }
  catch (e: any) {
    useFaToast().error('保存失败', { description: e?.message })
  }
  finally {
    daemonSaving.value = false
  }
}

function fmtSize(mb: number) {
  return mb >= 1024 ? (mb / 1024).toFixed(2) + ' GB' : mb.toFixed(0) + ' MB'
}

function fmtTime(unix: number) {
  return new Date(unix * 1000).toLocaleString('zh-CN', { hour12: false })
}

watch(tab, load)

onMounted(() => {
  load()
  timer = setInterval(() => {
    if (tab.value === 'containers' && !containerActionBusy.value) {
      load()
    }
  }, 5000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="container" :size="24" />
          <span>Docker 管理</span>
        </div>
      </template>
      <template #description>
        <span>容器 / 镜像 / 网络 / 卷 / Docker 配置（daemon.json）</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" @click="refreshCurrent">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton v-if="tab === 'containers'" size="sm" @click="pruneContainers">清理停止容器</FaButton>
        <FaButton v-if="tab === 'images'" variant="outline" size="sm" @click="pruneImages">清理悬空镜像</FaButton>
        <FaButton v-if="tab === 'images'" size="sm" @click="pullVisible = true">
          <FaIcon name="i-lucide:download" class="mr-1" /> 拉取镜像
        </FaButton>
        <FaButton v-if="tab === 'networks'" size="sm" @click="netVisible = true">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 创建网络
        </FaButton>
        <FaButton v-if="tab === 'volumes'" size="sm" @click="volVisible = true">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 创建卷
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="!dockerAvailable" class="mb-4 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-700 dark:bg-amber-950/30 dark:text-amber-400">
        {{ dockerHint || '未检测到可用的 Docker 环境' }}
      </div>

      <FaTabs
        v-model="tab" :list="[
          { label: '容器', value: 'containers' },
          { label: '镜像', value: 'images' },
          { label: '网络', value: 'networks' },
          { label: '卷', value: 'volumes' },
          { label: 'Docker 配置', value: 'config' },
        ]" @change="refreshCurrent"
      />

      <!-- 容器 -->
      <div v-if="tab === 'containers' && dockerAvailable" class="mt-3 overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">名称</th>
              <th class="px-3 py-2">镜像</th>
              <th class="px-3 py-2">状态</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in containers" :key="c.id" class="border-t hover:bg-accent/30">
              <td class="px-3 py-1.5 font-mono text-[13px]">{{ c.name }}</td>
              <td class="max-w-48 truncate px-3 py-1.5 font-mono text-xs">{{ c.image }}</td>
              <td class="px-3 py-1.5">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="c.state === 'running' ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">{{ c.state }}</span>
              </td>
              <td class="px-3 py-1.5 text-right">
                <FaButton variant="outline" size="sm" @click="containerAction(c, 'start')">启动</FaButton>
                <FaButton variant="outline" size="sm" @click="containerAction(c, 'stop')">停止</FaButton>
                <FaButton variant="outline" size="sm" @click="containerAction(c, 'restart')">重启</FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 镜像 -->
      <div v-if="tab === 'images' && dockerAvailable" class="mt-3 overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">TAG</th>
              <th class="px-3 py-2">ID</th>
              <th class="px-3 py-2">大小</th>
              <th class="hidden px-3 py-2 md:table-cell">创建时间</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="img in images" :key="img.id" class="border-t hover:bg-accent/30">
              <td class="px-3 py-1.5 font-mono text-[13px]">
                <div v-for="t in img.tags" :key="t">{{ t }}</div>
                <span v-if="!img.tags.length" class="text-muted-foreground">&lt;none&gt;</span>
              </td>
              <td class="px-3 py-1.5 font-mono text-xs text-muted-foreground">{{ img.id.slice(0, 12) }}</td>
              <td class="px-3 py-1.5 text-xs tabular-nums">{{ fmtSize(img.sizeMb) }}</td>
              <td class="hidden px-3 py-1.5 text-xs tabular-nums text-muted-foreground md:table-cell">{{ fmtTime(img.createdAt) }}</td>
              <td class="px-3 py-1.5 text-right">
                <FaButton variant="outline" size="sm" @click="removeImage(img)">删除</FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 网络 -->
      <div v-if="tab === 'networks' && dockerAvailable" class="mt-3 overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">名称</th>
              <th class="px-3 py-2">驱动</th>
              <th class="px-3 py-2">子网</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="n in networks" :key="n.id" class="border-t hover:bg-accent/30">
              <td class="px-3 py-1.5 font-mono text-[13px]">{{ n.name }}</td>
              <td class="px-3 py-1.5 text-xs">{{ n.driver }}</td>
              <td class="px-3 py-1.5 font-mono text-xs text-muted-foreground">{{ n.subnet || '—' }}</td>
              <td class="px-3 py-1.5 text-right">
                <FaButton variant="outline" size="sm" @click="removeNetwork(n)">删除</FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 卷 -->
      <div v-if="tab === 'volumes' && dockerAvailable" class="mt-3 overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">卷名</th>
              <th class="px-3 py-2">驱动</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="v in volumes" :key="v.name" class="border-t hover:bg-accent/30">
              <td class="px-3 py-1.5 font-mono text-[13px]">{{ v.name }}</td>
              <td class="px-3 py-1.5 text-xs text-muted-foreground">{{ v.driver }}</td>
              <td class="px-3 py-1.5 text-right">
                <FaButton variant="outline" size="sm" @click="removeVolume(v)">删除</FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Docker 配置 -->
      <div v-if="tab === 'config' && dockerAvailable" class="mt-3 rounded-md border p-3">
        <div class="mb-2 text-sm font-medium">daemon.json（保存后自动重启 docker，全部容器短暂中断）</div>
        <textarea
          v-model="daemonContent"
          class="h-72 w-full rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
          spellcheck="false"
        />
        <FaButton class="mt-2" :loading="daemonSaving" @click="saveDaemon">保存并重启 Docker</FaButton>
      </div>
    </FaPageMain>

    <!-- 拉取镜像 -->
    <FaModal v-model="pullVisible" title="拉取镜像" :destroy-on-close="true">
      <FaInput v-model="pullRef" placeholder="如 redis:alpine 或 registry/example/img:tag" class="w-full" @keyup.enter="doPull" />
      <pre v-if="pullOutput" class="mt-2 max-h-40 overflow-auto whitespace-pre-wrap rounded-md bg-muted/50 p-2 font-mono text-xs">{{ pullOutput }}</pre>
      <template #footer>
        <FaButton variant="outline" @click="pullVisible = false">取消</FaButton>
        <FaButton :loading="pulling" @click="doPull">拉取</FaButton>
      </template>
    </FaModal>

    <!-- 创建网络 -->
    <FaModal v-model="netVisible" title="创建网络" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">名称</span>
          <FaInput v-model="netForm.name" placeholder="如 my-net" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">驱动</span>
          <select v-model="netForm.driver" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="bridge">bridge</option>
            <option value="host">host</option>
            <option value="overlay">overlay</option>
          </select>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="netVisible = false">取消</FaButton>
        <FaButton @click="doCreateNetwork">创建</FaButton>
      </template>
    </FaModal>

    <!-- 创建卷 -->
    <FaModal v-model="volVisible" title="创建卷" :destroy-on-close="true">
      <div class="flex items-center gap-3">
        <span class="w-20 shrink-0 text-sm text-muted-foreground">卷名</span>
        <FaInput v-model="volName" placeholder="如 my-data" class="flex-1" @keyup.enter="doCreateVolume" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="volVisible = false">取消</FaButton>
        <FaButton @click="doCreateVolume">创建</FaButton>
      </template>
    </FaModal>
  </div>
</template>
