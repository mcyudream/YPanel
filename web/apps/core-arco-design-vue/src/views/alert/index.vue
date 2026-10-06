<script setup lang="ts">
import type { AlertRule } from '@/api/modules/alert'
import apiAlert from '@/api/modules/alert'

defineOptions({
  name: 'AlertIndex',
})

const rules = ref<AlertRule[]>([])
const loading = ref(false)
const createVisible = ref(false)
const form = ref({ name: '', metric: 'cpu', threshold: 80, webhookUrl: '', webhookType: 'feishu' })
const creating = ref(false)

async function load() {
  loading.value = true
  try {
    rules.value = await apiAlert.list()
  }
  finally {
    loading.value = false
  }
}

async function doCreate() {
  creating.value = true
  try {
    await apiAlert.create(form.value)
    useFaToast().success('规则已创建')
    createVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error('创建失败', { description: e?.message })
  }
  finally {
    creating.value = false
  }
}

async function toggle(r: AlertRule) {
  try {
    await apiAlert.update(r.id, { enabled: !r.enabled })
    await load()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
  }
}

function remove(r: AlertRule) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除规则',
    content: `确认删除告警规则 ${r.name}？`,
    onConfirm: async () => {
      try {
        await apiAlert.remove(r.id)
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
          <YdMorphIcon name="bell" :size="24" />
          <span>告警通知</span>
        </div>
      </template>
      <template #description>
        <span>阈值告警（CPU/内存/磁盘）→ webhook 通知（飞书/钉钉/企微/通用），持续 2 周期确认触发，10 分钟去抖，恢复也通知</span>
      </template>
      <FaButton size="sm" @click="createVisible = true">
        <FaIcon name="i-lucide:plus" class="mr-1" /> 新建规则
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">规则</th>
              <th class="px-3 py-2">条件</th>
              <th class="hidden px-3 py-2 lg:table-cell">Webhook</th>
              <th class="px-3 py-2">状态</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !rules.length">
              <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">加载中…</td>
            </tr>
            <tr v-else-if="!rules.length">
              <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">暂无告警规则</td>
            </tr>
            <tr v-for="r in rules" :key="r.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2 font-medium">{{ r.name }}</td>
              <td class="px-3 py-2 text-xs">
                {{ r.metric.toUpperCase() }} &gt; {{ r.threshold }}%
              </td>
              <td class="hidden max-w-72 truncate px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell" :title="r.webhookUrl">
                [{{ r.webhookType }}] {{ r.webhookUrl }}
              </td>
              <td class="px-3 py-2">
                <span class="rounded-full px-2 py-0.5 text-xs" :class="r.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  {{ r.enabled ? '启用' : '停用' }}
                </span>
              </td>
              <td class="px-3 py-2 text-right">
                <FaButton variant="outline" size="sm" @click="toggle(r)">{{ r.enabled ? '停用' : '启用' }}</FaButton>
                <FaButton variant="outline" size="sm" class="text-red-500!" @click="remove(r)">删除</FaButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <FaModal v-model="createVisible" title="新建告警规则" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">名称</span>
          <FaInput v-model="form.name" placeholder="如：CPU 过高" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">指标</span>
          <select v-model="form.metric" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="cpu">CPU 使用率</option>
            <option value="memory">内存使用率</option>
            <option value="disk">磁盘使用率（首分区）</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">阈值 %</span>
          <FaInput v-model="form.threshold" type="number" class="w-32" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">渠道类型</span>
          <select v-model="form.webhookType" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="feishu">飞书机器人</option>
            <option value="dingtalk">钉钉机器人</option>
            <option value="wecom">企业微信机器人</option>
            <option value="generic">通用 Webhook</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">Webhook</span>
          <FaInput v-model="form.webhookUrl" placeholder="https://..." class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">取消</FaButton>
        <FaButton :loading="creating" @click="doCreate">创建</FaButton>
      </template>
    </FaModal>
  </div>
</template>
