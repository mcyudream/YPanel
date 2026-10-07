<script setup lang="ts">
// AI 工作空间（智能/工作空间）：沙箱目录文件列表 + 手动执行命令。
import api from '@/api'

defineOptions({
  name: 'aiWorkspace',
})

interface WorkspaceFile { name: string, size: number }
const wsFiles = ref<WorkspaceFile[]>([])
const wsCommand = ref('')
const wsOutput = ref('')
const wsRunning = ref(false)

async function loadWorkspace() {
  try {
    const res = await api.get('api/v1/ai/workspace', { silent: true })
    const entries = (res.data as any)?.entries || []
    wsFiles.value = entries.filter((e: any) => !e.isDir).map((e: any) => ({ name: e.name, size: e.size }))
  }
  catch {}
}

async function wsRun() {
  if (!wsCommand.value.trim()) {
    return
  }
  wsRunning.value = true
  wsOutput.value = ''
  try {
    const res = await api.post('api/v1/ai/workspace/run', { command: wsCommand.value })
    wsOutput.value = (res.data as any)?.output || '(无输出)'
    await loadWorkspace()
  }
  catch (e: any) {
    wsOutput.value = '执行失败：' + (e?.message || '')
  }
  finally {
    wsRunning.value = false
  }
}

onMounted(loadWorkspace)
onActivated(loadWorkspace)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        AI 工作空间
      </template>
      <template #description>
        沙箱目录 /opt/ypanel/ai-workspace：AI 通过工具在此执行脚本与代码，也可手动运行命令
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="space-y-4">
        <div class="rounded-lg border p-4">
          <div class="mb-2 text-sm font-medium">
            文件列表
          </div>
          <div v-if="!wsFiles.length" class="py-3 text-center text-xs text-muted-foreground">
            目录为空
          </div>
          <div v-for="f in wsFiles" :key="f.name" class="border-b py-1.5 font-mono text-xs last:border-b-0">
            {{ f.name }} <span class="text-muted-foreground">({{ f.size }} B)</span>
          </div>
        </div>
        <div class="rounded-lg border p-4">
          <div class="mb-2 text-sm font-medium">
            在沙箱中执行命令
          </div>
          <div class="flex gap-2">
            <FaInput v-model="wsCommand" placeholder="如：echo hello > test.txt && cat test.txt" class="flex-1" @keyup.enter="wsRun" />
            <FaButton :loading="wsRunning" @click="wsRun">
              执行
            </FaButton>
          </div>
          <pre v-if="wsOutput" class="mt-3 max-h-60 overflow-auto rounded-md bg-muted/60 p-3 font-mono text-xs">{{ wsOutput }}</pre>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
