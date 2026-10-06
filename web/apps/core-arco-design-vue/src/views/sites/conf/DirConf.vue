<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteRunDirConf } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'

// B23 网站目录（对齐 1Panel「网站目录」子页）：运行目录二级目录 + 保存并重载。
// PHP 框架（Laravel/ThinkPHP 等）入口常在子目录（如 /public），通过运行目录调整 root。

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const conf = ref<SiteRunDirConf | null>(null)
const runDir = ref('/')
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    conf.value = await siteConfApi.getRunDir(props.site.id)
    runDir.value = conf.value.runDir || '/'
  }
  catch (e: any) {
    toast.error('读取网站目录失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await siteConfApi.updateRunDir(props.site.id, runDir.value)
    toast.success('运行目录已保存并重载 nginx')
    await load()
    emit('changed')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

const dirty = computed(() => (conf.value ? runDir.value !== (conf.value.runDir || '/') : false))

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">网站根目录</div>
      <p class="mb-2 text-xs text-muted-foreground">
        站点代码存放目录（容器内 /var/www/sites/站点名，宿主机 {{ conf?.hostRoot || '…' }}），可在文件管理中维护
      </p>
      <div class="rounded-md bg-muted/50 px-3 py-2 font-mono text-[13px]">
        {{ conf?.root || '…' }}
      </div>
    </div>

    <div class="rounded-lg border p-4">
      <div class="mb-1 text-sm font-medium">运行目录<span class="ml-2 text-xs font-normal text-muted-foreground">PHP 框架常用</span></div>
      <p class="mb-3 text-xs text-muted-foreground">
        实际对外服务的二级目录（相对站点根目录），如 Laravel / ThinkPHP 填 /public；留 / 表示根目录。保存后自动校验并重载 nginx
      </p>
      <div class="flex items-center gap-2">
        <select v-model="runDir" class="h-9 w-56 rounded-md border bg-background px-2 text-sm outline-none focus:border-primary">
          <option value="/">/（根目录）</option>
          <option v-for="d in conf?.subdirs || []" :key="d" :value="d">{{ d }}</option>
          <option v-if="runDir !== '/' && !(conf?.subdirs || []).includes(runDir)" :value="runDir">{{ runDir }}</option>
        </select>
        <FaButton size="sm" :loading="saving" :disabled="!dirty" @click="save">
          保存并重载
        </FaButton>
      </div>
      <div v-if="loading" class="mt-2 text-xs text-muted-foreground">读取子目录中…</div>
    </div>

    <div class="rounded-lg border p-4 text-sm">
      <div class="mb-2 font-medium">站点主要文件夹</div>
      <div class="overflow-hidden rounded-md border">
        <table class="w-full text-[13px]">
          <tbody>
            <tr class="border-b">
              <td class="w-32 bg-muted/40 px-3 py-2 font-mono text-xs">sites/&lt;name&gt;</td>
              <td class="px-3 py-2 text-muted-foreground">网站 root 目录（PHP 运行环境 / 静态站代码存放目录）</td>
            </tr>
            <tr class="border-b">
              <td class="bg-muted/40 px-3 py-2 font-mono text-xs">conf.d/&lt;name&gt;.conf</td>
              <td class="px-3 py-2 text-muted-foreground">站点 nginx 配置（「配置文件」页可编辑）</td>
            </tr>
            <tr>
              <td class="bg-muted/40 px-3 py-2 font-mono text-xs">logs/&lt;name&gt;.log</td>
              <td class="px-3 py-2 text-muted-foreground">站点访问 / 错误日志（「日志」页查看）</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
