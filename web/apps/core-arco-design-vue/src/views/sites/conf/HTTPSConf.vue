<script setup lang="ts">
import type { SiteItem } from '@/api/modules/site'
import type { SiteHTTPSConf } from '@/api/modules/siteconf'
import { siteConfApi } from '@/api/modules/siteconf'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()

const toast = useFaToast()

const conf = ref<SiteHTTPSConf | null>(null)
const loading = ref(false)
const busy = ref(false)

async function load() {
  loading.value = true
  try {
    conf.value = await siteConfApi.getHTTPS(props.site.id)
  }
  catch (e: any) {
    toast.error('读取 HTTPS 配置失败', { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

function enable() {
  const modal = useFaModal()
  modal.confirm({
    title: '启用 HTTPS（自签证书）',
    content: `将为 ${props.site.domain} 生成自签证书并监听 443，80 端口强制跳转 HTTPS。浏览器会提示不受信任，需手动信任或忽略；ACME 证书就绪后可替换。`,
    onConfirm: async () => {
      busy.value = true
      try {
        conf.value = await siteConfApi.enableHTTPS(props.site.id)
        toast.success('HTTPS 已启用')
        emit('changed')
      }
      catch (e: any) {
        toast.error('启用失败', { description: e?.message })
      }
      finally {
        busy.value = false
      }
    },
  })
}

function disable() {
  const modal = useFaModal()
  modal.confirm({
    title: '停用 HTTPS',
    content: '停用后 443 将不再监听，HTTPS 访问会失败，确认停用？',
    onConfirm: async () => {
      busy.value = true
      try {
        conf.value = await siteConfApi.disableHTTPS(props.site.id)
        toast.success('HTTPS 已停用')
        emit('changed')
      }
      catch (e: any) {
        toast.error('停用失败', { description: e?.message })
      }
      finally {
        busy.value = false
      }
    },
  })
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="flex items-center justify-between">
        <div>
          <div class="flex items-center gap-2 text-sm font-medium">
            HTTPS 状态
            <span
              class="rounded-full px-2 py-0.5 text-xs"
              :class="conf?.enable ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
            >
              {{ conf?.enable ? '已启用' : '未启用' }}
            </span>
          </div>
          <p class="mt-1 text-xs text-muted-foreground">
            {{ conf?.enable ? `证书域名 ${conf.certDomain} · HTTP 强制跳转 HTTPS（301）` : '启用后监听 443 并强制 HTTP 跳转' }}
          </p>
        </div>
        <FaButton v-if="!conf?.enable" size="sm" :loading="busy" @click="enable">
          启用 HTTPS
        </FaButton>
        <FaButton v-else variant="outline" size="sm" :loading="busy" class="text-red-500!" @click="disable">
          停用 HTTPS
        </FaButton>
      </div>
    </div>

    <div class="rounded-lg border p-4 text-sm">
      <div class="mb-2 font-medium">访问地址</div>
      <div class="space-y-1 font-mono text-[13px] text-muted-foreground">
        <div v-if="conf?.enable">
          <span class="text-emerald-600">https://</span>{{ site.domain }} <span class="text-xs">（80 自动 301 跳转）</span>
        </div>
        <div v-else>
          <span class="text-blue-600">http://</span>{{ site.domain }}
        </div>
      </div>
    </div>

    <div class="rounded-lg border border-dashed p-4 text-xs text-muted-foreground">
      <div class="mb-1 font-medium text-foreground">关于证书来源</div>
      当前支持自签证书（浏览器需信任）；Let's Encrypt（ACME）自动签发需公网环境或 DNS API 凭据，已列入后续批次。
    </div>
  </div>
</template>
