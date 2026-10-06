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
const extVisible = ref(false)
const extForm = ref({ name: '', version: '8.2', fcgiAddr: '127.0.0.1:9000', remark: '' })
const extSaving = ref(false)

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

async function doAttach() {
  extSaving.value = true
  try {
    await apiRT.attachExternal(extForm.value)
    useFaToast().success('已接管本机 PHP（fastcgi 直连）')
    extVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error('接管失败', { description: e?.message })
  }
  finally {
    extSaving.value = false
  }
}

function remove(r: RuntimeItem) {
  const external = r.origin === 'external'
  const modal = useFaModal()
  modal.confirm({
    title: external ? '解除接管' : '删除运行环境',
    content: external
      ? `确认解除对 ${r.name}（${r.fcgiAddr}）的接管？本机 php-fpm 不受影响。`
      : `确认删除 PHP ${r.version} 运行环境 ${r.name}？（站点数据保留在 www 目录）`,
    onConfirm: async () => {
      try {
        await apiRT.remove(r.id)
        useFaToast().success(external ? '已解除接管' : '已删除')
        await load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

async function toggle(r: RuntimeItem) {
  if (r.origin === 'external') {
    useFaToast().info('外部运行环境由其所在主机管理')
    return
  }
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
        <span>PHP-FPM 运行时（容器化 / 接管本机 php-fpm）：配合"网站"页创建 PHP 类型站点</span>
      </template>
      <div class="flex gap-2">
        <FaButton size="sm" variant="outline" @click="extVisible = true">
          <FaIcon name="i-lucide:plug-zap" class="mr-1" /> 接管本机 PHP
        </FaButton>
        <FaButton size="sm" @click="createVisible = true">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 创建运行环境
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div v-for="r in runtimes" :key="r.id" class="rounded-lg border bg-background p-4">
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-2">
              <YdMorphIcon name="file-code" :size="22" class="text-indigo-500" />
              <div>
                <div class="flex items-center gap-1.5">
                  <span class="font-medium">{{ r.name }}</span>
                  <span
                    class="rounded px-1.5 py-0.5 text-xs"
                    :class="r.origin === 'external' ? 'bg-blue-500/10 text-blue-600' : 'bg-muted text-muted-foreground'"
                  >
                    {{ r.origin === 'external' ? '外部接管' : '容器' }}
                  </span>
                </div>
                <div class="text-xs text-muted-foreground">
                  {{ r.origin === 'external' ? `PHP ${r.version} · ${r.fcgiAddr}` : `PHP ${r.version} · fpm-alpine` }}
                </div>
              </div>
            </div>
            <span class="rounded-full px-2 py-0.5 text-xs" :class="r.running ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ r.running ? '运行中' : '已停止' }}
            </span>
          </div>
          <div class="mt-3 flex items-center justify-between border-t pt-3">
            <span class="font-mono text-xs text-muted-foreground">{{ r.origin === 'external' ? 'fastcgi 直连' : r.composeProject }}</span>
            <div class="flex gap-1">
              <FaButton v-if="r.origin !== 'external'" variant="outline" size="sm" @click="toggle(r)">{{ r.running ? '停止' : '启动' }}</FaButton>
              <FaButton variant="outline" size="sm" class="text-red-500!" @click="remove(r)">{{ r.origin === 'external' ? '解除' : '删除' }}</FaButton>
            </div>
          </div>
        </div>
        <div v-if="!runtimes.length && !loading" class="rounded-lg border p-10 text-center text-sm text-muted-foreground md:col-span-2 xl:col-span-3">
          暂无运行环境
        </div>
      </div>
    </FaPageMain>

    <FaModal v-model="extVisible" title="接管本机 PHP" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">名称</span>
          <FaInput v-model="extForm.name" placeholder="小写字母/数字/中划线，如 host-php" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">版本</span>
          <select v-model="extForm.version" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="8.2">PHP 8.2</option>
            <option value="8.3">PHP 8.3</option>
            <option value="8.1">PHP 8.1</option>
            <option value="7.4">PHP 7.4</option>
            <option value="unknown">未知</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">FastCGI</span>
          <FaInput v-model="extForm.fcgiAddr" placeholder="127.0.0.1:9000 或 unix:/run/php/php-fpm.sock" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          直连本机已有 php-fpm（systemd 安装或手动部署）；注意 php-fpm 需监听 TCP 或可读权限的 unix socket，且站点 root 路径需与面板 www 目录一致（/opt/ypanel/nginx/www）
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="extVisible = false">取消</FaButton>
        <FaButton :loading="extSaving" @click="doAttach">接管</FaButton>
      </template>
    </FaModal>

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
