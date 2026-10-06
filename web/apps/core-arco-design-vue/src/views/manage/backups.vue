<script setup lang="ts">
import { panelBackupApi } from '@/api/modules/m9'

defineOptions({
  name: 'ManageBackups',
})

const toast = useFaToast()

const backups = ref<{ name: string, sizeMb: number, modTime: string, path: string }[]>([])
const loading = ref(false)
const backupBusy = ref(false)
const restoreHint = ref('')

async function load() {
  loading.value = true
  try {
    backups.value = await panelBackupApi.list()
  }
  catch (e: any) {
    toast.error('加载备份列表失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function doBackup() {
  backupBusy.value = true
  try {
    const out = await panelBackupApi.create()
    toast.success(`面板备份完成：${out.file || ''}`)
    await load()
  }
  catch (e: any) {
    toast.error('备份失败', { description: e?.message })
  }
  finally {
    backupBusy.value = false
  }
}

function downloadBackup(b: { path: string }) {
  const token = localStorage.getItem('token') || ''
  window.open(panelBackupApi.downloadURL(b.path, token))
}

function removeBackup(b: { name: string }) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除备份',
    content: `确认删除 ${b.name}？删除后不可恢复。`,
    onConfirm: async () => {
      await panelBackupApi.remove(b.name)
      toast.success('已删除')
      await load()
    },
  })
}

async function showHint() {
  restoreHint.value = await panelBackupApi.restoreHint()
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="save" :size="24" />
          <span>面板备份</span>
        </div>
      </template>
      <template #description>
        <span>面板配置与数据（SQLite 快照）；恢复需在服务器手动操作，见恢复说明</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="showHint">恢复说明</FaButton>
        <FaButton size="sm" :loading="backupBusy" @click="doBackup">
          <YdMorphIcon name="save" :size="14" class="mr-1" /> 立即备份
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="restoreHint" class="mb-4 rounded-lg border border-amber-300 bg-amber-50 p-4 text-xs whitespace-pre-line text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
        {{ restoreHint }}
      </div>

      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2">备份文件</th>
              <th class="hidden w-36 px-4 py-2 sm:table-cell">大小</th>
              <th class="hidden w-52 px-4 py-2 md:table-cell">创建时间</th>
              <th class="w-40 px-4 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !backups.length">
              <td colspan="4" class="px-4 py-10 text-center text-muted-foreground">
                加载中…
              </td>
            </tr>
            <tr v-else-if="!backups.length">
              <td colspan="4" class="px-4 py-10 text-center text-muted-foreground">
                暂无备份，点击右上角「立即备份」创建首个快照
              </td>
            </tr>
            <tr v-for="b in backups" :key="b.name" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-4 py-2.5 font-mono text-[13px]">
                {{ b.name }}
              </td>
              <td class="hidden px-4 py-2.5 text-xs tabular-nums text-muted-foreground sm:table-cell">
                {{ b.sizeMb.toFixed(2) }} MB
              </td>
              <td class="hidden px-4 py-2.5 text-xs tabular-nums text-muted-foreground md:table-cell">
                {{ new Date(b.modTime).toLocaleString('zh-CN', { hour12: false }) }}
              </td>
              <td class="px-4 py-2.5 text-right">
                <FaButton variant="ghost" size="sm" @click="downloadBackup(b)">
                  下载
                </FaButton>
                <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeBackup(b)">
                  删除
                </FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>
  </div>
</template>
