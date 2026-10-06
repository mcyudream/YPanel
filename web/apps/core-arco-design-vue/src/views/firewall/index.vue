<script setup lang="ts">
import type { FirewallStatus } from '@/api/modules/firewall'
import apiFW from '@/api/modules/firewall'

defineOptions({
  name: 'FirewallIndex',
})

const fw = ref<FirewallStatus>()
const loading = ref(false)
const allowVisible = ref(false)
const allowForm = ref({ port: '', proto: 'tcp' })

async function load() {
  loading.value = true
  try {
    fw.value = await apiFW.status()
  }
  finally {
    loading.value = false
  }
}

async function doAllow() {
  try {
    await apiFW.allow(allowForm.value.port, allowForm.value.proto)
    useFaToast().success(`已放行 ${allowForm.value.port}/${allowForm.value.proto}`)
    allowVisible.value = false
    allowForm.value = { port: '', proto: 'tcp' }
    await load()
  }
  catch (e: any) {
    useFaToast().error('放行失败', { description: e?.message })
  }
}

async function toggleRun() {
  if (!fw.value) {
    return
  }
  const modal = useFaModal()
  const enable = !fw.value.enabled
  if (enable) {
    modal.confirm({
      title: '启用防火墙',
      content: '将自动放行 SSH(22/tcp) 与面板端口后再启用，防止连接中断。确认启用？',
      onConfirm: async () => {
        try {
          await apiFW.enable()
          useFaToast().success('防火墙已启用')
          await load()
        }
        catch (e: any) {
          useFaToast().error('操作失败', { description: e?.message })
        }
      },
    })
  }
  else {
    try {
      await apiFW.disable()
      useFaToast().success('防火墙已停用')
      await load()
    }
    catch (e: any) {
      useFaToast().error('操作失败', { description: e?.message })
    }
  }
}

async function deleteRule(raw: string) {
  const m = raw.match(/\[\s*(\d+)\]/)
  if (!m) {
    return
  }
  const modal = useFaModal()
  modal.confirm({
    title: '删除规则',
    content: raw,
    onConfirm: async () => {
      try {
        await apiFW.deleteRule(Number(m[1]))
        useFaToast().success('已删除')
        await load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
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
          <YdMorphIcon name="shield" :size="24" />
          <span>防火墙</span>
        </div>
      </template>
      <template #description>
        <span>ufw 形态：端口放行 / 规则管理（启用前自动放行 SSH 与面板端口防自锁）</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="load">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" :class="loading ? 'animate-spin' : ''" /> 刷新
        </FaButton>
        <FaButton v-if="fw?.available" size="sm" :variant="fw.enabled ? 'outline' : 'default'" @click="toggleRun">
          {{ fw.enabled ? '停用防火墙' : '启用防火墙' }}
        </FaButton>
        <FaButton v-if="fw?.available" size="sm" @click="allowVisible = true">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 放行端口
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div v-if="fw && !fw.available" class="rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-700 dark:bg-amber-950/30 dark:text-amber-400">
        {{ fw.hint }}
      </div>
      <template v-else-if="fw">
        <div class="mb-4 flex items-center gap-2">
          <span class="text-sm text-muted-foreground">状态：</span>
          <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="fw.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
            <span class="inline-block size-1.5 rounded-full" :class="fw.enabled ? 'animate-pulse bg-current' : 'bg-current'" />
            {{ fw.enabled ? '运行中' : '已停用' }}
          </span>
        </div>
        <div class="overflow-x-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th class="px-3 py-2">规则（ufw 编号）</th>
                <th class="px-3 py-2 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!fw.rules?.length">
                <td colspan="2" class="px-3 py-8 text-center text-muted-foreground">暂无规则</td>
              </tr>
              <tr v-for="r in fw.rules" :key="r.raw" class="border-t hover:bg-accent/30">
                <td class="px-3 py-2 font-mono text-xs">{{ r.raw }}</td>
                <td class="px-3 py-2 text-right">
                  <FaButton variant="outline" size="sm" @click="deleteRule(r.raw)">删除</FaButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </FaPageMain>

    <FaModal v-model="allowVisible" title="放行端口" :destroy-on-close="true">
      <div class="flex items-center gap-3">
        <FaInput v-model="allowForm.port" placeholder="端口，如 8080" class="w-40" />
        <select v-model="allowForm.proto" class="h-9 rounded-md border border-input bg-background px-2 text-sm outline-none">
          <option value="tcp">tcp</option>
          <option value="udp">udp</option>
        </select>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="allowVisible = false">取消</FaButton>
        <FaButton @click="doAllow">放行</FaButton>
      </template>
    </FaModal>
  </div>
</template>
