<script setup lang="ts">
import type { RuntimeItem } from '@/api/modules/runtime'
import apiRT from '@/api/modules/runtime'

defineOptions({
  name: 'RuntimesIndex',
})

const runtimes = ref<RuntimeItem[]>([])
const loading = ref(false)
const createVisible = ref(false)
const form = ref({ name: '', version: '8.2' })
const creating = ref(false)

async function load() {
  loading.value = true
  try {
    runtimes.value = await apiRT.list()
  }
  finally {
    loading.value = false
  }
}

async function doCreate() {
  creating.value = true
  try {
    await apiRT.create(form.value)
    useFaToast().success('运行环境创建中（首次拉取镜像可能较慢）')
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

function remove(r: RuntimeItem) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除运行环境',
    content: `确认删除 PHP ${r.version} 运行环境 ${r.name}？（站点数据保留在 www 目录）`,
    onConfirm: async () => {
      try {
        await apiRT.remove(r.id)
        useFaToast().success('已删除')
        await load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

async function toggle(r: RuntimeItem) {
  try {
    if (r.running) {
      await apiRT.stop(r.id)
      useFaToast().success('已停止')
    }
    else {
      await apiRT.start(r.id)
      useFaToast().success('已启动')
    }
    await load()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
  }
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="file-code" :size="24" />
          <span>运行环境</span>
        </div>
      </template>
      <template #description>
        <span>PHP-FPM 容器化运行时：配合"网站"页创建 PHP 类型站点（伪静态模板可用）</span>
      </template>
      <FaButton size="sm" @click="createVisible = true">
        <FaIcon name="i-lucide:plus" class="mr-1" /> 创建运行环境
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div v-for="r in runtimes" :key="r.id" class="rounded-lg border bg-background p-4">
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-2">
              <YdMorphIcon name="file-code" :size="22" class="text-indigo-500" />
              <div>
                <div class="font-medium">{{ r.name }}</div>
                <div class="text-xs text-muted-foreground">PHP {{ r.version }} · fpm-alpine</div>
              </div>
            </div>
            <span class="rounded-full px-2 py-0.5 text-xs" :class="r.running ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ r.running ? '运行中' : '已停止' }}
            </span>
          </div>
          <div class="mt-3 flex items-center justify-between border-t pt-3">
            <span class="font-mono text-xs text-muted-foreground">{{ r.composeProject }}</span>
            <div class="flex gap-1">
              <FaButton variant="outline" size="sm" @click="toggle(r)">{{ r.running ? '停止' : '启动' }}</FaButton>
              <FaButton variant="outline" size="sm" class="text-red-500!" @click="remove(r)">删除</FaButton>
            </div>
          </div>
        </div>
        <div v-if="!runtimes.length && !loading" class="rounded-lg border p-10 text-center text-sm text-muted-foreground md:col-span-2 xl:col-span-3">
          暂无运行环境
        </div>
      </div>
    </FaPageMain>

    <FaModal v-model="createVisible" title="创建 PHP 运行环境" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">名称</span>
          <FaInput v-model="form.name" placeholder="小写字母/数字/中划线" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">版本</span>
          <select v-model="form.version" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="8.2">PHP 8.2 (fpm-alpine)</option>
            <option value="8.3">PHP 8.3 (fpm-alpine)</option>
          </select>
        </div>
        <div class="text-xs text-muted-foreground">共享 nginx www 卷（/opt/ypanel/nginx/www），PHP 站点的 PHP 文件即在此目录</div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">取消</FaButton>
        <FaButton :loading="creating" @click="doCreate">创建并启动</FaButton>
      </template>
    </FaModal>
  </div>
</template>
