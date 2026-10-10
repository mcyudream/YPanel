<script setup lang="ts">
// YPanel 内网浏览器（M42）：地址栏输入内网地址，经面板第二端口会话式反代渲染。
// 请求全部以面板网络环境出站（loopback/私网目标），公网目标默认被后端拒绝。
// 会话 8h 惰性过期：组件每次挂载为地址创建新会话（成本极低），关窗不强制销毁。
import { computed, onMounted, ref } from 'vue'
import webgwApi, { webgwTargetsApi } from '@/api/modules/webgw'
import nodeApi from '@/api/modules/node'

const props = defineProps<{
  /** 初始地址（可选）：有则挂载即导航，无则显示引导空态 */
  url?: string
}>()

const address = ref(props.url ?? '')
const target = ref('')
/** 输入地址中 origin 之后的部分（/health?x=1），会话指向 origin 根，首跳落在这里 */
const initialPath = ref('/')
const sid = ref('')
const busy = ref(false)
const pageLoading = ref(false)
const error = ref('')
const iframeKey = ref(0)

const isDev = import.meta.env.DEV

/** 归一化输入：无 scheme 补 http://，拆出 origin 与路径部分（与后端 Create 解析口径一致） */
function splitInput(raw: string): { origin: string, rest: string } | null {
  let u = raw.trim()
  if (!u) {
    return null
  }
  if (!u.includes('://')) {
    u = `http://${u}`
  }
  try {
    const parsed = new URL(u)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      return null
    }
    const rest = `${parsed.pathname === '/' ? '' : parsed.pathname}${parsed.search}`
    return { origin: parsed.origin, rest: rest || '/' }
  }
  catch {
    return null
  }
}

// 网关 = 面板主端口 + 1（与后端 config.GwPort 默认值一致）；dev 端口推导不适用
const gwBase = computed(() => {
  if (isDev) {
    return ''
  }
  const base = Number(location.port || (location.protocol === 'https:' ? 443 : 80))
  if (!base || Number.isNaN(base)) {
    return ''
  }
  return `${location.protocol}//${location.hostname}:${base + 1}`
})

const frameSrc = computed(() => (sid.value && gwBase.value ? `${gwBase.value}/s/${sid.value}${initialPath.value}` : ''))

async function navigate(raw?: string) {
  const u = (raw ?? address.value).trim()
  const parts = splitInput(u)
  if (!parts || busy.value) {
    if (!parts && u) {
      error.value = '仅支持 http(s) 地址'
    }
    return
  }
  error.value = ''
  busy.value = true
  try {
    const ses = await webgwApi.create(parts.origin)
    sid.value = ses.sid
    target.value = parts.origin
    initialPath.value = parts.rest
    address.value = u
    pageLoading.value = true
    iframeKey.value++
  }
  catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
    sid.value = ''
  }
  finally {
    busy.value = false
  }
}

function reload() {
  if (!sid.value) {
    return
  }
  pageLoading.value = true
  iframeKey.value++
}

function openExternal() {
  if (target.value) {
    window.open(target.value, '_blank', 'noopener')
  }
}

function onFrameLoad() {
  pageLoading.value = false
}

// ---- M50：节点切换 + 目标发现 ----
const nodeSel = ref('local')
const nodes = ref<{ id: string, name: string }[]>([])
const showTargets = ref(false)
const targetsLoading = ref(false)
const targets = ref<{ name: string, url: string, hostPort: string, reachable: boolean }[]>([])
const targetsIp = ref('')

async function loadNodes() {
  try {
    const res = await nodeApi.list()
    nodes.value = (res || []).map((n: any) => ({ id: n.id === 0 || n.id === 'local' ? 'local' : String(n.id), name: n.name }))
  }
  catch {}
}

async function loadTargets() {
  targetsLoading.value = true
  try {
    const out = await webgwTargetsApi.list(nodeSel.value)
    targets.value = out.targets || []
    targetsIp.value = out.ip || ''
  }
  catch {
    targets.value = []
  }
  finally {
    targetsLoading.value = false
  }
}

function toggleTargets() {
  showTargets.value = !showTargets.value
  if (showTargets.value) {
    void loadTargets()
  }
}

function openTarget(t: { url: string }) {
  showTargets.value = false
  address.value = t.url
  void navigate(t.url)
}

onMounted(() => {
  void loadNodes()
  if (props.url) {
    void navigate(props.url)
  }
})

defineExpose({ navigate })
</script>

<template>
  <div class="ydb-browser">
    <!-- 工具条：地址栏 / 刷新 / 外部打开 -->
    <div class="ydb-bar">
      <YdSelect
        v-model="nodeSel"
        :options="nodes.map(n => ({ label: n.name, value: n.id }))"
        button-class="ydb-btn ydb-node"
        title="节点"
        @update:model-value="showTargets && loadTargets()"
      />
      <button class="ydb-btn" title="发现该节点可浏览目标" @click="toggleTargets()">
        <i class="i-lucide-list-ends" />
      </button>
      <input
        v-model="address"
        class="ydb-input"
        :placeholder="$t('desktop.webgw.placeholder')"
        @keyup.enter="navigate()"
      >
      <button class="ydb-btn" :disabled="busy || !address.trim()" :title="$t('desktop.webgw.go')" @click="navigate()">
        <i class="i-lucide-arrow-right" />
      </button>
      <button class="ydb-btn" :disabled="!sid" :title="$t('desktop.webgw.reload')" @click="reload()">
        <i class="i-lucide-rotate-cw" />
      </button>
      <button class="ydb-btn" :disabled="!target" :title="$t('desktop.webgw.openExternal')" @click="openExternal()">
        <i class="i-lucide-external-link" />
      </button>
    </div>

    <!-- M50：节点目标列表 -->
    <div v-if="showTargets" class="ydb-error">
      <template v-if="targetsLoading">
        <i class="i-lucide-loader-circle animate-spin" /> 加载中…
      </template>
      <template v-else-if="!targets.length">
        <i class="i-lucide-circle-alert" /> 未发现该节点上的可浏览目标（运行中容器的发布端口）
      </template>
      <template v-else>
        <div class="mb-1 text-[11px] opacity-70">节点 {{ targetsIp }} 上的目标（点击打开）：</div>
        <div
          v-for="t in targets" :key="t.url"
          class="flex cursor-pointer items-center gap-2 rounded px-1 py-0.5 hover:bg-white/10"
          @click="openTarget(t)"
        >
          <i :class="t.reachable ? 'i-lucide-circle-check text-emerald-400' : 'i-lucide-circle-dashed text-amber-400'" />
          <span class="font-mono">{{ t.name }}</span>
          <span class="opacity-60">{{ t.url }}</span>
          <span v-if="!t.reachable" class="text-[11px] text-amber-400">不可达</span>
        </div>
      </template>
    </div>

    <!-- 错误条（创建会话失败：SSRF 拒绝/解析失败等） -->
    <div v-if="error" class="ydb-error">
      <i class="i-lucide-circle-alert" />
      <span>{{ error }}</span>
    </div>

    <!-- dev 提示（端口推导不适用） -->
    <div v-if="isDev && !error" class="ydb-error">
      <i class="i-lucide-info" />
      <span>{{ $t('desktop.webgw.devOnly') }}</span>
    </div>

    <!-- 页面区 -->
    <div class="ydb-body">
      <iframe
        v-if="frameSrc"
        :key="iframeKey"
        :src="frameSrc"
        class="ydb-frame"
        referrerpolicy="no-referrer-when-downgrade"
        @load="onFrameLoad"
      />
      <div v-else class="ydb-empty">
        <i class="i-lucide-globe" />
        <p>{{ $t('desktop.webgw.empty') }}</p>
        <p class="ydb-hint">{{ $t('desktop.webgw.hint') }}</p>
      </div>
      <div v-if="pageLoading && frameSrc" class="ydb-loading">
        <i class="i-lucide-loader-circle" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.ydb-browser {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background: var(--yw-window-bg);
}

.ydb-bar {
  display: flex;
  gap: 6px;
  align-items: center;
  padding: 6px 8px;
  border-bottom: 1px solid var(--yw-separator);
  flex-shrink: 0;
}

.ydb-input {
  flex: 1;
  min-width: 0;
  padding: 5px 10px;
  font-size: 12px;
  color: oklch(var(--yw-foreground));
  background: oklch(var(--yw-foreground) / 5%);
  border: 1px solid transparent;
  border-radius: 8px;
  outline: none;
}

.ydb-input:focus {
  border-color: oklch(var(--yw-primary));
  background: oklch(var(--yw-foreground) / 8%);
}

.ydb-node {
  width: auto;
  height: 28px;
  padding: 0 6px;
  font-size: 12px;
}

.ydb-btn {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  font-size: 13px;
  color: oklch(var(--yw-foreground));
  cursor: pointer;
  background: oklch(var(--yw-foreground) / 5%);
  border: none;
  border-radius: 8px;
  transition: background var(--yw-dur-ui, 0.15s);
  flex-shrink: 0;
}

.ydb-btn:hover:not(:disabled) {
  background: oklch(var(--yw-foreground) / 12%);
}

.ydb-btn:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.ydb-error {
  display: flex;
  gap: 6px;
  align-items: center;
  padding: 6px 12px;
  font-size: 12px;
  color: #e11d48;
  background: rgb(225 29 72 / 8%);
  flex-shrink: 0;
}

.ydb-body {
  position: relative;
  flex: 1;
  min-height: 0;
}

.ydb-frame {
  display: block;
  width: 100%;
  height: 100%;
  border: 0;
  background: #fff;
}

.ydb-empty {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 13px;
  color: oklch(var(--yw-muted-foreground));
}

.ydb-empty > i {
  font-size: 34px;
  opacity: 0.5;
}

.ydb-hint {
  margin: 0;
  font-size: 11px;
  opacity: 0.7;
}

.ydb-loading {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  font-size: 22px;
  color: oklch(var(--yw-foreground) / 60%);
  background: var(--yw-window-bg);
}

.ydb-loading > i {
  animation: ydb-spin 0.9s linear infinite;
}

@keyframes ydb-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
