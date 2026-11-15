<script setup lang="ts">
import api from '@/api'
import apiFile from '@/api/modules/file'
// AI 工作空间（智能/工作空间）：宿主工作目录文件列表 + 手动执行命令 + 上传/新建文件。
// 宿主工作目录与 core 侧 service/aibase.go workspaceDir 保持一致。
import { i18n } from '@/locales'

defineOptions({
  name: 'AiWorkspace',
})
const WS_DIR = '/opt/ypanel/ai-workspace'
interface WorkspaceFile {
  name: string
  size: number
}
const wsFiles = ref<WorkspaceFile[]>([])
const wsCommand = ref('')
const wsOutput = ref('')
const wsRunning = ref(false)
const wsError = ref('')
const wsLoading = ref(false)
const toast = useFaToast()
// 空态可一键填入的示例命令
const exampleCommands = [
  'echo "hello ypanel" > hello.txt && cat hello.txt',
  'uname -a > sysinfo.txt && cat sysinfo.txt',
]
async function loadWorkspace() {
  wsLoading.value = true
  wsError.value = ''
  try {
    const res = await api.get('api/v1/ai/workspace', { silent: true })
    const entries = (res.data as any)?.entries || []
    wsFiles.value = entries.filter((e: any) => !e.isDir).map((e: any) => ({ name: e.name, size: e.size }))
  }
  catch (e: any) {
    wsError.value = e?.message || i18n.global.t('ai.workspace.loadFailed')
    wsFiles.value = []
  }
  finally {
    wsLoading.value = false
  }
}
function useExample(cmd: string) {
  wsCommand.value = cmd
}
async function wsRun() {
  if (!wsCommand.value.trim()) {
    return
  }
  wsRunning.value = true
  wsOutput.value = ''
  try {
    const res = await api.post('api/v1/ai/workspace/run', { command: wsCommand.value })
    wsOutput.value = (res.data as any)?.output || i18n.global.t('ai.workspace.noOutput')
    await loadWorkspace()
  }
  catch (e: any) {
    wsOutput.value = i18n.global.t('ai.workspace.runFailed', { msg: e?.message || '' })
  }
  finally {
    wsRunning.value = false
  }
}
// ---- 上传文件 ----
const fileInput = ref<HTMLInputElement>()
const uploading = ref(false)
function pickUpload() {
  fileInput.value?.click()
}
async function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) {
    return
  }
  uploading.value = true
  try {
    await apiFile.upload(WS_DIR, file)
    toast.success(i18n.global.t('ai.workspace.uploaded', { name: file.name }))
    await loadWorkspace()
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.workspace.uploadFailed'), { description: e?.message })
  }
  finally {
    uploading.value = false
  }
}
// ---- 新建文件 ----
const createVisible = ref(false)
const createName = ref('')
const createContent = ref('')
const creating = ref(false)
function openCreate() {
  createName.value = ''
  createContent.value = ''
  createVisible.value = true
}
async function createFile() {
  const name = createName.value.trim()
  if (!name) {
    toast.error(i18n.global.t('ai.workspace.nameRequired'))
    return
  }
  if (/[/\\]|\.\./.test(name)) {
    toast.error(i18n.global.t('ai.workspace.nameInvalid'))
    return
  }
  creating.value = true
  try {
    await apiFile.write(`${WS_DIR}/${name}`, createContent.value)
    toast.success(i18n.global.t('ai.workspace.created', { name }))
    createVisible.value = false
    await loadWorkspace()
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.workspace.createFailed'), { description: e?.message })
  }
  finally {
    creating.value = false
  }
}
onMounted(loadWorkspace)
onActivated(loadWorkspace)
</script>

<template>
  <div>
    <FaPageMain>
      <div class="space-y-4">
        <div class="p-4 border rounded-lg">
          <div class="mb-2 flex items-center justify-between">
            <div class="text-sm font-medium">
              {{ $t('ai.workspace.fileList') }}
              <span class="text-xs text-muted-foreground font-mono font-normal ml-1">{{ WS_DIR }}</span>
            </div>
            <div class="flex gap-2">
              <FaButton size="sm" variant="outline" @click="openCreate">
                <FaIcon name="i-lucide:file-plus" class="mr-1" /> {{ $t('ai.workspace.createFile') }}
              </FaButton>
              <FaButton size="sm" variant="outline" :loading="uploading" @click="pickUpload">
                <FaIcon name="i-lucide:upload" class="mr-1" /> {{ $t('ai.workspace.uploadFile') }}
              </FaButton>
            </div>
          </div>

          <!-- 加载失败：显示错误与重试，不误报"目录为空" -->
          <div v-if="wsError" class="text-xs text-red-600 p-3 border border-red-300/60 rounded-md bg-red-500/5 dark:text-red-400 dark:border-red-500/40">
            {{ $t('ai.workspace.loadFailedWith', { msg: wsError }) }}
            <FaButton size="sm" variant="ghost" class="ml-2!" @click="loadWorkspace">
              {{ $t('ai.workspace.retry') }}
            </FaButton>
          </div>

          <!-- 空态引导：说明目录用途与文件来源 -->
          <div v-else-if="!wsFiles.length" class="text-xs text-muted-foreground py-4 text-center">
            <div class="mb-2">
              <FaIcon name="i-lucide:folder-code" class="text-lg mr-1" />{{ $t('ai.workspace.emptyDir') }}
            </div>
            <p class="leading-5 mx-auto mb-3 max-w-md">
              {{ $t('ai.workspace.emptyDesc') }}
            </p>
            <div class="mx-auto text-left max-w-md space-y-1.5">
              <div
                v-for="cmd in exampleCommands"
                :key="cmd"
                class="text-xs font-mono px-2.5 py-1.5 border rounded-md bg-muted/40 flex gap-2 cursor-pointer transition-colors items-center justify-between hover:text-foreground hover:border-primary/50"
                :title="$t('ai.workspace.fillHint')"
                @click="useExample(cmd)"
              >
                <span class="truncate">{{ cmd }}</span>
                <FaIcon name="i-lucide:corner-down-left" class="opacity-60 shrink-0" />
              </div>
            </div>
          </div>

          <template v-else>
            <div v-for="f in wsFiles" :key="f.name" class="text-xs font-mono py-1.5 border-b last:border-b-0">
              {{ f.name }} <span class="text-muted-foreground">({{ f.size }} B)</span>
            </div>
          </template>
        </div>
        <div class="p-4 border rounded-lg">
          <div class="text-sm font-medium mb-2">
            {{ $t('aiAssistant.hostExecution') }}
          </div>
          <div class="flex gap-2">
            <FaInput v-model="wsCommand" :placeholder="$t('ai.workspace.runPlaceholder')" class="flex-1" @keyup.enter="wsRun" />
            <FaButton :loading="wsRunning" @click="wsRun">
              {{ $t('ai.workspace.run') }}
            </FaButton>
          </div>
          <pre v-if="wsOutput" class="text-xs font-mono mt-3 p-3 rounded-md bg-muted/60 max-h-60 overflow-auto">{{ wsOutput }}</pre>
        </div>
      </div>
    </FaPageMain>

    <!-- 新建文件弹窗 -->
    <FaModal v-model="createVisible" :title="$t('ai.workspace.createFile')" class="max-w-xl!" :destroy-on-close="true">
      <div class="text-sm space-y-3">
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-20">{{ $t('ai.workspace.fileName') }}</span>
          <FaInput v-model="createName" :placeholder="$t('ai.workspace.namePlaceholder')" class="flex-1" @keyup.enter="createFile" />
        </div>
        <div class="flex gap-3 items-start">
          <span class="text-muted-foreground pt-2 shrink-0 w-20">{{ $t('ai.workspace.content') }}</span>
          <textarea
            v-model="createContent"
            rows="8"
            class="text-xs font-mono p-2 outline-none border rounded-md bg-background w-full resize-y focus:border-primary"
            :placeholder="$t('ai.workspace.contentPlaceholder')"
          />
        </div>
        <p class="text-xs text-muted-foreground pl-23">
          {{ $t('ai.workspace.willCreate', { path: WS_DIR }) }}
        </p>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="creating" @click="createFile">
          {{ $t('ai.workspace.create') }}
        </FaButton>
      </template>
    </FaModal>

    <input ref="fileInput" type="file" class="hidden" @change="onFileChange">
  </div>
</template>
