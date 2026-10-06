<script setup lang="ts">
import api from '@/api'
import { rawApi } from '@/api'

interface UpdateFile {
  name: string
  sizeMb: number
  modTime: string
}

const router = useRouter()

defineOptions({
  name: 'SelfUpdateIndex',
})

const status = ref<{ currentVersion: string, channelDir: string, available: UpdateFile[] }>()
const loading = ref(false)
const applying = ref('')

async function load() {
  loading.value = true
  try {
    const res = await api.get('api/v1/system/update/status', { silent: true })
    status.value = res.data
  }
  finally {
    loading.value = false
  }
}

function apply(file: string) {
  const modal = useFaModal()
  modal.confirm({
    title: '应用更新',
    content: `将备份当前二进制后替换为 ${file} 并重启面板服务（约 5 秒中断）。确认执行？`,
    onConfirm: async () => {
      applying.value = file
      try {
        await api.post('api/v1/system/update/apply', { file })
        useFaToast().info('更新已启动，服务重启中…')
        // 轮询 health 直到恢复
        let recovered = false
        for (let i = 0; i < 20; i++) {
          await new Promise(r => setTimeout(r, 1500))
          try {
            const h = await rawApi.get('/health').then(r => r.data)
            if (h.status === 'ok') {
              recovered = true
              useFaToast().success(`服务已恢复，版本 ${h.version}`)
              break
            }
          }
          catch {}
        }
        if (!recovered) {
          useFaToast().error('服务未在 30 秒内恢复，请检查 journalctl -u ypanel')
        }
        await load()
      }
      catch (e: any) {
        useFaToast().error('更新失败', { description: e?.message })
      }
      finally {
        applying.value = ''
      }
    },
  })
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="refresh-cw" :size="24" />
          <span>面板设置</span>
        </div>
      </template>
      <template #description>
        <span>更新通道：将新版本二进制放入服务器 {{ status?.channelDir || '/opt/ypanel/updates' }}（ypanel 前缀命名），此处一键应用；更新前自动备份 ypanel.bak</span>
      </template>
      <FaButton variant="outline" size="sm" @click="() => router.push('/manage/security')">
        <FaIcon name="i-lucide:shield-check" class="mr-1" /> 安全设置
      </FaButton>
      <FaButton variant="outline" size="sm" @click="load">
        <FaIcon name="i-lucide:refresh-cw" class="mr-1" :class="loading ? 'animate-spin' : ''" /> 刷新
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div class="mb-4 flex items-center gap-2 rounded-lg border bg-background p-4">
        <YdMorphIcon name="info" :size="18" class="text-primary" />
        当前版本：<code class="rounded bg-muted px-1.5 py-0.5 font-mono text-sm">{{ status?.currentVersion || '—' }}</code>
      </div>

      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">更新文件</th>
              <th class="px-3 py-2">大小</th>
              <th class="hidden px-3 py-2 md:table-cell">放入时间</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!status?.available?.length && !loading">
              <td colspan="4" class="px-3 py-10 text-center text-muted-foreground">
                更新通道为空：将新版本二进制 scp 到服务器 {{ status?.channelDir }} 目录后刷新
              </td>
            </tr>
            <tr v-for="f in status?.available || []" :key="f.name" class="border-t hover:bg-accent/30">
              <td class="px-3 py-2 font-mono text-[13px]">{{ f.name }}</td>
              <td class="px-3 py-2 text-xs tabular-nums">{{ f.sizeMb.toFixed(1) }} MB</td>
              <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground md:table-cell">
                {{ new Date(f.modTime).toLocaleString('zh-CN', { hour12: false }) }}
              </td>
              <td class="px-3 py-2 text-right">
                <FaButton size="sm" :disabled="applying === f.name" @click="apply(f.name)">
                  {{ applying === f.name ? '更新中…' : '应用并重启' }}
                </FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>
  </div>
</template>
