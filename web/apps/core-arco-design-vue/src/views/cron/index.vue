<script setup lang="ts">
import type { CronTask, CronTaskLog } from '@/api/modules/cron'
import apiCron from '@/api/modules/cron'

defineOptions({
  name: 'CronIndex',
})

const tasks = ref<CronTask[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await apiCron.list(1, 100)
    tasks.value = res.items
  }
  finally {
    loading.value = false
  }
}

// ---- 创建/编辑 ----
const editorVisible = ref(false)
const isCreate = ref(false)
const editorId = ref<number>(0)
const form = ref({ name: '', cron: '*/5 * * * *', command: '', timeoutSecs: 300 })
const saving = ref(false)

function openCreate() {
  isCreate.value = true
  editorId.value = 0
  form.value = { name: '', cron: '*/5 * * * *', command: '', timeoutSecs: 300 }
  editorVisible.value = true
}

function openEdit(t: CronTask) {
  isCreate.value = false
  editorId.value = t.id
  form.value = { name: t.name, cron: t.cron, command: t.command, timeoutSecs: t.timeoutSecs }
  editorVisible.value = true
}

async function save() {
  if (!form.value.name || !form.value.cron || !form.value.command) {
    useFaToast().warning('请填写完整：名称 / cron 表达式 / 命令')
    return
  }
  saving.value = true
  try {
    if (isCreate.value) {
      await apiCron.create(form.value)
      useFaToast().success('任务已创建')
    }
    else {
      await apiCron.update(editorId.value, form.value)
      useFaToast().success('任务已保存')
    }
    editorVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error('保存失败', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

async function toggle(t: CronTask) {
  try {
    await apiCron.update(t.id, { enabled: !t.enabled })
    await load()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
  }
}

async function runNow(t: CronTask) {
  try {
    await apiCron.run(t.id)
    useFaToast().success('已触发执行')
    setTimeout(load, 800)
  }
  catch (e: any) {
    useFaToast().error('触发失败', { description: e?.message })
  }
}

function remove(t: CronTask) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除任务',
    content: `确认删除任务 ${t.name}？执行记录将保留。`,
    onConfirm: async () => {
      try {
        await apiCron.remove(t.id)
        useFaToast().success('已删除')
        await load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

// ---- 日志 ----
const logsVisible = ref(false)
const logsTaskId = ref<number | undefined>(undefined)
const logs = ref<CronTaskLog[]>([])
const logsTotal = ref(0)
const logsPage = ref(1)
const logsLoading = ref(false)
const expanded = ref<CronTaskLog | null>(null)

async function openLogs(taskId?: number) {
  logsTaskId.value = taskId
  logsPage.value = 1
  logsVisible.value = true
  await loadLogs()
}

async function loadLogs() {
  logsLoading.value = true
  try {
    const res = await apiCron.logs(logsTaskId.value, logsPage.value, 20)
    logs.value = res.items
    logsTotal.value = res.total
  }
  finally {
    logsLoading.value = false
  }
}

function fmtTime(iso?: string | null) {
  return iso ? new Date(iso).toLocaleString('zh-CN', { hour12: false }) : '—'
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="calendar-clock" :size="24" />
          <span>计划任务</span>
        </div>
      </template>
      <template #description>
        <span>cron 调度 shell 命令（在服务器上经受控通道执行），支持手动触发与执行记录</span>
      </template>
      <div class="flex items-center gap-2">
        <FaButton variant="outline" size="sm" @click="openLogs()">
          <FaIcon name="i-lucide:scroll-text" class="mr-1" /> 全部执行记录
        </FaButton>
        <FaButton size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 新建任务
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">任务</th>
              <th class="px-3 py-2">cron</th>
              <th class="hidden px-3 py-2 lg:table-cell">命令</th>
              <th class="px-3 py-2">状态</th>
              <th class="hidden px-3 py-2 xl:table-cell">最近执行</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !tasks.length">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                加载中…
              </td>
            </tr>
            <tr v-else-if="!tasks.length">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                暂无计划任务
              </td>
            </tr>
            <tr v-for="t in tasks" :key="t.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2">
                <div class="font-medium">{{ t.name }}</div>
                <div class="text-xs text-muted-foreground">超时 {{ t.timeoutSecs }}s</div>
              </td>
              <td class="px-3 py-2 font-mono text-xs">
                {{ t.cron }}
              </td>
              <td class="hidden max-w-72 truncate px-3 py-2 font-mono text-xs text-muted-foreground lg:table-cell" :title="t.command">
                {{ t.command }}
              </td>
              <td class="px-3 py-2">
                <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="t.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  <span class="inline-block size-1.5 rounded-full" :class="t.enabled ? 'animate-pulse bg-current' : 'bg-current'" />
                  {{ t.enabled ? '启用' : '停用' }}
                </span>
                <span
                  v-if="t.lastSuccess !== null && t.lastSuccess !== undefined"
                  class="ml-1.5 rounded-full px-2 py-0.5 text-xs"
                  :class="t.lastSuccess ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'"
                >
                  {{ t.lastSuccess ? '成功' : '失败' }}
                </span>
              </td>
              <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground xl:table-cell">
                {{ fmtTime(t.lastRunAt) }}
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton variant="outline" size="sm" @click="runNow(t)">
                    运行
                  </FaButton>
                  <FaButton variant="ghost" size="sm" @click="openLogs(t.id)">
                    记录
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" :title="t.enabled ? '停用' : '启用'" @click="toggle(t)">
                    <FaIcon :name="t.enabled ? 'i-lucide:pause' : 'i-lucide:play'" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" title="编辑" @click="openEdit(t)">
                    <FaIcon name="i-lucide:pen-line" class="text-sm" />
                  </FaButton>
                  <FaButton variant="ghost" size="icon-sm" title="删除" @click="remove(t)">
                    <FaIcon name="i-lucide:trash" class="text-sm" />
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <!-- 创建/编辑 -->
    <FaModal v-model="editorVisible" :title="isCreate ? '新建计划任务' : `编辑：${form.name}`" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">任务名称</span>
          <FaInput v-model="form.name" placeholder="如：日志清理" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">cron 表达式</span>
          <FaInput v-model="form.cron" placeholder="分 时 日 月 周，如 */5 * * * *" class="flex-1" />
        </div>
        <div class="flex gap-1.5">
          <button
            v-for="preset in ['*/1 * * * *', '*/5 * * * *', '0 * * * *', '0 3 * * *', '0 3 * * 1']"
            :key="preset"
            type="button"
            class="cursor-pointer rounded-md border px-2 py-0.5 font-mono text-xs text-muted-foreground transition-colors hover:bg-accent/50"
            @click="form.cron = preset"
          >
            {{ preset }}
          </button>
        </div>
        <div class="flex items-start gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">命令</span>
          <textarea
            v-model="form.command"
            class="h-24 w-full flex-1 resize-y rounded-md border border-input bg-background p-2 font-mono text-[13px] outline-none focus:ring-1 focus:ring-primary"
            placeholder="sh 命令，如：find /var/log -name '*.log' -mtime +7 -delete"
            spellcheck="false"
          />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-sm text-muted-foreground">超时（秒）</span>
          <FaInput v-model="form.timeoutSecs" type="number" class="w-32" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="editorVisible = false">
          取消
        </FaButton>
        <FaButton :loading="saving" @click="save">
          保存
        </FaButton>
      </template>
    </FaModal>

    <!-- 执行记录 -->
    <FaModal v-model="logsVisible" :title="logsTaskId ? `执行记录：任务 #${logsTaskId}` : '全部执行记录'" class="max-w-4xl!" :destroy-on-close="true">
      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">开始时间</th>
              <th class="px-3 py-2">任务</th>
              <th class="px-3 py-2">触发</th>
              <th class="px-3 py-2">结果</th>
              <th class="px-3 py-2 text-right">耗时</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="logsLoading && !logs.length">
              <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
                加载中…
              </td>
            </tr>
            <tr v-else-if="!logs.length">
              <td colspan="5" class="px-3 py-8 text-center text-muted-foreground">
                暂无记录
              </td>
            </tr>
            <template v-for="l in logs" :key="l.id">
              <tr class="cursor-pointer border-t hover:bg-accent/30" @click="expanded = expanded?.id === l.id ? null : l">
                <td class="px-3 py-2 text-xs tabular-nums text-muted-foreground">
                  {{ fmtTime(l.startAt) }}
                </td>
                <td class="px-3 py-2">
                  {{ l.taskName }}
                </td>
                <td class="px-3 py-2 text-xs text-muted-foreground">
                  {{ l.trigger === 'cron' ? '调度' : '手动' }}
                </td>
                <td class="px-3 py-2">
                  <span class="rounded-full px-2 py-0.5 text-xs" :class="l.success ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'">
                    {{ l.success ? '成功' : '失败' }}
                  </span>
                </td>
                <td class="px-3 py-2 text-right text-xs tabular-nums text-muted-foreground">
                  {{ (l.durationMs / 1000).toFixed(1) }}s
                </td>
              </tr>
              <tr v-if="expanded?.id === l.id">
                <td colspan="5" class="bg-muted/30 p-0">
                  <pre class="max-h-64 overflow-auto p-3 font-mono text-xs leading-relaxed">{{ l.output || '（无输出）' }}</pre>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
      <div class="mt-3 flex justify-end">
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <FaButton variant="outline" size="sm" :disabled="logsPage <= 1" @click="logsPage--; loadLogs()">
            上一页
          </FaButton>
          <span>{{ logsPage }} / {{ Math.max(1, Math.ceil(logsTotal / 20)) }}（共 {{ logsTotal }} 条）</span>
          <FaButton variant="outline" size="sm" :disabled="logsPage >= Math.ceil(logsTotal / 20)" @click="logsPage++; loadLogs()">
            下一页
          </FaButton>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="logsVisible = false">
          关闭
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
