<script setup lang="ts">
import type { DockerImage } from '@/api/modules/dockerext'
import type { ImageUpdateCheck } from '@/api/modules/dockerenv'
import YdLogViewer from '@/components/YdLogViewer/index.vue'
import { dockerExtApi } from '@/api/modules/dockerext'
import { dockerImgApi } from '@/api/modules/dockerenv'
import { taskApi } from '@/api/modules/task'
import { i18n } from '@/locales'

// 镜像管理（自 Docker 管理页迁入，M23）：列表 + 拉取 + 删除 + 清理悬空。
// M35：构建 / 导入 / 导出 / 打标签 / 更新检查。
const toast = useFaToast()

const images = ref<DockerImage[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    images.value = await dockerExtApi.images()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.images.loadFailed'), { description: e?.message })
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

// ---- 分页（搜索变化自动回第 1 页） ----
const page = ref(1)
const size = ref(20)
const paged = computed(() => filtered.value.slice((page.value - 1) * size.value, page.value * size.value))
watch(search, () => {
  page.value = 1
})

const pullVisible = ref(false)
const pullRef = ref('')
const pulling = ref(false)
const pullTaskId = ref(0)
const pullTask = ref<{ id: number, status: string, logText: string, error: string } | null>(null)
let pullPollTimer: ReturnType<typeof setInterval> | null = null

async function refreshPullTask() {
  if (!pullTaskId.value) {
    return
  }
  try {
    const t = await taskApi.get(pullTaskId.value)
    pullTask.value = t
    if (t.status !== 'running') {
      if (pullPollTimer) {
        clearInterval(pullPollTimer)
        pullPollTimer = null
      }
      if (t.status === 'success') {
        toast.success(i18n.global.t('container.images.pullDone'))
        await load()
      }
      else {
        toast.error(i18n.global.t('container.images.pullFailed'), { description: t.error })
      }
    }
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.images.taskStatusFailed'), { description: e?.message })
    if (pullPollTimer) {
      clearInterval(pullPollTimer)
      pullPollTimer = null
    }
  }
}

async function doPull() {
  if (!pullRef.value.trim()) {
    return
  }
  pulling.value = true
  try {
    const out = await dockerExtApi.pull(pullRef.value.trim())
    pullTaskId.value = out.taskId
    pullTask.value = null
    void refreshPullTask()
    pullPollTimer = setInterval(() => {
      if (!pullTask.value || pullTask.value.status === 'running') {
        void refreshPullTask()
      }
      else if (pullPollTimer) {
        clearInterval(pullPollTimer)
        pullPollTimer = null
      }
    }, 2000)
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.images.pullTaskFailed'), { description: e?.message })
  }
  finally {
    pulling.value = false
  }
}

watch(pullVisible, (v) => {
  if (!v && pullPollTimer) {
    clearInterval(pullPollTimer)
    pullPollTimer = null
  }
})

function removeImage(img: DockerImage) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.common.deleteImageTitle'),
    content: i18n.global.t('container.images.deleteConfirm', { name: img.tags[0] || img.id.slice(0, 12) }),
    onConfirm: async () => {
      try {
        await dockerExtApi.removeImage(img.id)
        toast.success(i18n.global.t('container.common.deletedDone'))
        await load()
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.common.deleteFailed'), { description: e?.message })
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
    toast.error(i18n.global.t('container.common.pruneFailed'), { description: e?.message })
  }
}

function fmtSize(mb: number) {
  return mb >= 1024 ? `${(mb / 1024).toFixed(2)} GB` : `${mb.toFixed(0)} MB`
}

function fmtTime(unix: number) {
  return new Date(unix * 1000).toLocaleString('zh-CN', { hour12: false })
}

// ---- M35：构建 ----
const buildVisible = ref(false)
const buildBusy = ref(false)
const buildForm = ref({ contextDir: '', dockerfile: '', tag: '' })

async function doBuild() {
  if (!buildForm.value.contextDir || !buildForm.value.tag) {
    toast.warning(i18n.global.t('container.images.buildRequired'))
    return
  }
  buildBusy.value = true
  try {
    await dockerImgApi.build(buildForm.value)
    toast.success(i18n.global.t('container.images.buildDone', { name: buildForm.value.tag }))
    buildVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.images.buildFailed'), { description: e?.message })
  }
  finally {
    buildBusy.value = false
  }
}

// ---- M35：导入 ----
const importInputRef = ref<HTMLInputElement | null>(null)

async function doImport(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) {
    return
  }
  toast.info(i18n.global.t('container.images.importing'))
  try {
    await dockerImgApi.load(file)
    toast.success(i18n.global.t('container.images.importDone'))
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.images.importFailed'), { description: e?.message })
  }
  finally {
    if (importInputRef.value) {
      importInputRef.value.value = ''
    }
  }
}

// ---- M35：导出 ----
const saveBusy = ref('')

async function doSave(img: DockerImage) {
  const refName = img.tags[0] || img.id
  saveBusy.value = img.id
  try {
    const out = await dockerImgApi.save([refName])
    toast.success(i18n.global.t('container.images.exportDone', { file: out.file }))
    const token = localStorage.getItem('token') || ''
    window.open(`api/v1/files/download?path=${encodeURIComponent(out.file)}&token=${encodeURIComponent(token)}`)
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.images.exportFailed'), { description: e?.message })
  }
  finally {
    saveBusy.value = ''
  }
}

// ---- M35：打标签 ----
const tagVisible = ref(false)
const tagBusy = ref(false)
const tagForm = ref({ src: '', dst: '' })

function openTag(img: DockerImage) {
  tagForm.value = { src: img.tags[0] || img.id, dst: '' }
  tagVisible.value = true
}

async function doTag() {
  if (!tagForm.value.dst) {
    return
  }
  tagBusy.value = true
  try {
    await dockerImgApi.tag(tagForm.value.src, tagForm.value.dst)
    toast.success(i18n.global.t('container.images.tagDone'))
    tagVisible.value = false
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.images.tagFailed'), { description: e?.message })
  }
  finally {
    tagBusy.value = false
  }
}

// ---- M35：更新检查 ----
const updatesVisible = ref(false)
const updatesBusy = ref(false)
const updates = ref<ImageUpdateCheck[]>([])

async function doCheckUpdates() {
  // 取当前筛选下前 30 个带 tag 的镜像（registry 检查限流保护）
  const targets = filtered.value.filter(i => i.tags.length).slice(0, 30).map(i => i.tags[0])
  if (!targets.length) {
    toast.warning(i18n.global.t('container.images.noCheckTargets'))
    return
  }
  updatesBusy.value = true
  updatesVisible.value = true
  try {
    updates.value = await dockerImgApi.checkUpdates(targets)
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.images.checkFailed'), { description: e?.message })
    updatesVisible.value = false
  }
  finally {
    updatesBusy.value = false
  }
}
</script>

<template>
  <div>
    <FaPageMain>
  <div>
		<YdDockerNodeSelect />
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <FaInput v-model="search" :placeholder="$t('container.images.searchPlaceholder')" class="h-8 w-52!" />
      <div class="ml-auto flex items-center gap-2">
        <FaButton variant="outline" size="icon-sm" :title="$t('common.refresh')" @click="load()">
          <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
        </FaButton>
        <FaButton variant="outline" size="sm" @click="doCheckUpdates" :loading="updatesBusy">
          <FaIcon name="i-lucide:scan-search" class="mr-1" /> {{ $t('container.images.checkUpdates') }}
        </FaButton>
        <FaButton variant="outline" size="sm" @click="buildVisible = true">
          <FaIcon name="i-lucide:hammer" class="mr-1" /> {{ $t('container.images.build') }}
        </FaButton>
        <label class="cursor-pointer rounded-md border px-3 py-1.5 text-sm transition-colors hover:bg-accent/50">
          <FaIcon name="i-lucide:upload" class="mr-1" /> {{ $t('common.import') }}
          <input ref="importInputRef" type="file" class="hidden" accept=".tar,.tar.gz,.tgz" @change="doImport">
        </label>
        <FaButton variant="outline" size="sm" @click="pruneImages">
          {{ $t('container.images.pruneDangling') }}
        </FaButton>
        <FaButton v-auth="['docker:write']" size="sm" @click="pullVisible = true">
          <FaIcon name="i-lucide:download" class="mr-1" /> {{ $t('container.images.pull') }}
        </FaButton>
      </div>
    </div>

    <div class="overflow-x-auto rounded-lg border">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
          <tr>
            <th class="px-3 py-2">TAG</th>
            <th class="px-3 py-2">ID</th>
            <th class="px-3 py-2">{{ $t('common.size') }}</th>
            <th class="hidden px-3 py-2 md:table-cell">{{ $t('container.common.createdAt') }}</th>
            <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !filtered.length">
            <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
              {{ $t('common.loading') }}
            </td>
          </tr>
          <tr v-else-if="!filtered.length">
            <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
              {{ $t('container.common.noImages') }}
            </td>
          </tr>
          <tr v-for="img in paged" :key="img.id" class="border-t transition-colors hover:bg-accent/30">
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
              <FaButton variant="ghost" size="sm" :disabled="!img.tags.length" :title="$t('container.images.tagTitle')" @click="openTag(img)">
                {{ $t('container.images.tagBtn') }}
              </FaButton>
              <FaButton variant="ghost" size="sm" :loading="saveBusy === img.id" :disabled="!img.tags.length" :title="$t('container.images.exportTar')" @click="doSave(img)">
                {{ $t('common.export') }}
              </FaButton>
              <FaButton variant="outline" size="sm" @click="removeImage(img)">
                {{ $t('common.delete') }}
              </FaButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <FaPagination v-model:page="page" v-model:size="size" :total="filtered.length" class="mt-3" />

    <FaModal v-model="pullVisible" :title="pullTaskId ? $t('container.images.pullRunning', { name: pullRef }) : $t('container.images.pull')" :destroy-on-close="true" :close-on-click-modal="false">
      <div v-if="!pullTaskId" class="flex flex-col gap-2">
        <FaInput v-model="pullRef" :placeholder="$t('container.images.pullPlaceholder')" class="w-full" @keyup.enter="doPull" />
        <div class="text-xs text-muted-foreground">{{ $t('container.images.pullHint') }}</div>
      </div>
      <div v-else class="flex flex-col gap-2">
        <div class="text-xs text-muted-foreground">{{ $t('container.images.taskNo', { n: pullTaskId }) }} · {{ pullTask?.status === 'success' ? $t('container.images.statusDone') : pullTask?.status === 'failed' ? $t('common.failed') : $t('container.images.pulling') }}</div>
        <YdLogViewer :logs="pullTask?.logText || ''" height="280px" />
      </div>
      <template #footer>
        <template v-if="!pullTaskId">
          <FaButton variant="outline" @click="pullVisible = false">
            {{ $t('common.cancel') }}
          </FaButton>
          <FaButton :loading="pulling" @click="doPull">
            {{ $t('container.images.createPullTask') }}
          </FaButton>
        </template>
        <template v-else>
          <FaButton variant="outline" @click="pullVisible = false">
            {{ pullTask?.status === 'running' ? $t('container.images.backgroundRun') : $t('common.close') }}
          </FaButton>
        </template>
      </template>
    </FaModal>
    <!-- M35：构建 -->
    <FaModal v-model="buildVisible" :title="$t('container.images.buildTitle')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <label class="block space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('container.images.contextLabel') }}</span>
          <FaInput v-model="buildForm.contextDir" :placeholder="$t('container.images.contextPlaceholder')" class="w-full" />
        </label>
        <label class="block space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('container.images.dockerfileLabel') }}</span>
          <FaInput v-model="buildForm.dockerfile" placeholder="Dockerfile" class="w-full" />
        </label>
        <label class="block space-y-1">
          <span class="text-xs text-muted-foreground">{{ $t('container.images.tagLabel') }}</span>
          <FaInput v-model="buildForm.tag" :placeholder="$t('container.images.tagPlaceholder')" class="w-full" />
        </label>
        <div class="text-xs text-muted-foreground">{{ $t('container.images.buildHint') }}</div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="buildVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="buildBusy" @click="doBuild">
          {{ $t('container.images.startBuild') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- M35：打标签 -->
    <FaModal v-model="tagVisible" :title="$t('container.images.tagModalTitle')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="text-xs text-muted-foreground">{{ $t('container.images.source', { name: tagForm.src }) }}</div>
        <FaInput v-model="tagForm.dst" :placeholder="$t('container.images.newTagPlaceholder')" class="w-full" @keyup.enter="doTag" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="tagVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="tagBusy" @click="doTag">
          {{ $t('container.images.ok') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- M35：更新检查结果 -->
    <FaModal v-model="updatesVisible" :title="$t('container.images.updatesTitle')" class="max-w-3xl!" :destroy-on-close="true">
      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('container.common.image') }}</th>
              <th class="w-24 px-3 py-2">{{ $t('container.images.result') }}</th>
              <th class="hidden px-3 py-2 md:table-cell">{{ $t('container.images.remoteDigest') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in updates" :key="u.image" class="border-t">
              <td class="px-3 py-2 font-mono text-xs break-all">
                {{ u.image }}
                <div v-if="u.error" class="text-xs text-red-500">
                  {{ u.error }}
                </div>
              </td>
              <td class="px-3 py-2">
                <span v-if="!u.ok" class="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">{{ $t('common.unknown') }}</span>
                <span v-else-if="u.hasUpdate" class="rounded-full bg-amber-500/10 px-2 py-0.5 text-xs text-amber-600">{{ $t('container.images.hasUpdate') }}</span>
                <span v-else class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">{{ $t('container.images.upToDate') }}</span>
              </td>
              <td class="hidden px-3 py-2 font-mono text-xs text-muted-foreground md:table-cell">
                {{ u.remoteDigest?.slice(0, 24) || '—' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="updatesVisible = false">
          {{ $t('common.close') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
    </FaPageMain>
  </div>
</template>
