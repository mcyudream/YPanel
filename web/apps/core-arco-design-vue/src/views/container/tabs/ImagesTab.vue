<script setup lang="ts">
import type { DockerImage } from '@/api/modules/dockerext'
import { dockerExtApi } from '@/api/modules/dockerext'

// 镜像管理（自 Docker 管理页迁入，M23）：列表 + 拉取 + 删除 + 清理悬空。
const toast = useFaToast()

const images = ref<DockerImage[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    images.value = await dockerExtApi.images()
  }
  catch (e: any) {
    toast.error('镜像列表加载失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

onMounted(load)

const search = ref('')
const filtered = computed(() => {
  const kw = search.value.trim().toLowerCase()
  if (!kw) {
    return images.value
  }
  return images.value.filter(img => img.tags.some(t => t.toLowerCase().includes(kw)) || img.id.toLowerCase().includes(kw))
})

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
    toast.success('拉取完成')
    pullVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error('拉取失败', { description: e?.message })
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
        toast.success('已删除')
        await load()
      }
      catch (e: any) {
        toast.error('删除失败', { description: e?.message })
      }
    },
  })
}

async function pruneImages() {
  try {
    const out = await dockerExtApi.pruneImages()
    toast.success(out)
    await load()
  }
  catch (e: any) {
    toast.error('清理失败', { description: e?.message })
  }
}

function fmtSize(mb: number) {
  return mb >= 1024 ? `${(mb / 1024).toFixed(2)} GB` : `${mb.toFixed(0)} MB`
}

function fmtTime(unix: number) {
  return new Date(unix * 1000).toLocaleString('zh-CN', { hour12: false })
}
</script>

<template>
  <div>
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <FaInput v-model="search" placeholder="搜索 TAG/ID…" class="h-8 w-52!" />
      <div class="ml-auto flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" title="刷新" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton variant="outline" size="sm" @click="pruneImages">
          清理悬空镜像
        </FaButton>
        <FaButton size="sm" @click="pullVisible = true">
          <FaIcon name="i-lucide:download" class="mr-1" /> 拉取镜像
        </FaButton>
      </div>
    </div>

    <div class="overflow-x-auto rounded-lg border">
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
          <tr v-if="loading && !filtered.length">
            <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
              加载中…
            </td>
          </tr>
          <tr v-else-if="!filtered.length">
            <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
              暂无镜像
            </td>
          </tr>
          <tr v-for="img in filtered" :key="img.id" class="border-t transition-colors hover:bg-accent/30">
            <td class="px-3 py-1.5 font-mono text-[13px]">
              <div class="flex items-center gap-2">
                <YdAppIcon :name="img.tags[0] || img.id" :size="18" />
                <div>
                  <div v-for="t in img.tags" :key="t">
                    {{ t }}
                  </div>
                  <span v-if="!img.tags.length" class="text-muted-foreground">&lt;none&gt;</span>
                </div>
              </div>
            </td>
            <td class="px-3 py-1.5 font-mono text-xs text-muted-foreground">
              {{ img.id.slice(0, 12) }}
            </td>
            <td class="px-3 py-1.5 text-xs tabular-nums">
              {{ fmtSize(img.sizeMb) }}
            </td>
            <td class="hidden px-3 py-1.5 text-xs tabular-nums text-muted-foreground md:table-cell">
              {{ fmtTime(img.createdAt) }}
            </td>
            <td class="px-3 py-1.5 text-right">
              <FaButton variant="outline" size="sm" @click="removeImage(img)">
                删除
              </FaButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <FaModal v-model="pullVisible" title="拉取镜像" :destroy-on-close="true">
      <FaInput v-model="pullRef" placeholder="如 redis:alpine 或 registry/example/img:tag" class="w-full" @keyup.enter="doPull" />
      <pre class="mt-2 max-h-40 overflow-auto rounded-md bg-muted/50 p-2 font-mono text-xs whitespace-pre-wrap">{{ pullOutput }}</pre>
      <template #footer>
        <FaButton variant="outline" @click="pullVisible = false">
          取消
        </FaButton>
        <FaButton :loading="pulling" @click="doPull">
          拉取
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
