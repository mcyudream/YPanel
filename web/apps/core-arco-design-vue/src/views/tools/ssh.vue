<script setup lang="ts">
import type { NodeItem } from '@/api/modules/node'
import type { SshConfig, SshKey } from '@/api/modules/ssh'
import nodeApi from '@/api/modules/node'
// M53 SSH 管理（系统分组，节点维度）：sshd 配置 + 密钥统一管理（生成/导入/分发/删除）
import api from '@/api/modules/ssh'
import { i18n } from '@/locales'

defineOptions({
  name: 'ToolsSsh',
})

const toast = useFaToast()

// ---- 节点 ----
const nodes = ref<NodeItem[]>([])
const nodeId = ref('local')
const nodeOptions = computed(() => nodes.value.map(n => ({
  label: `${n.name}${n.remote ? '' : i18n.global.t('tools.localNode')}${n.online ? '' : i18n.global.t('tools.offlineNode')}`,
  value: n.id,
  disabled: !n.online,
})))

async function loadNodes() {
  try {
    nodes.value = await nodeApi.list()
  }
  catch {}
}

// ---- sshd 配置 ----
const sshCfg = ref<SshConfig | null>(null)
const sshBusy = ref(false)
const sshPort = ref(22)

async function loadSsh() {
  sshCfg.value = null
  try {
    sshCfg.value = await api.getConfig(nodeId.value)
    sshPort.value = sshCfg.value?.port || 22
  }
  catch (e: any) {
    toast.error(i18n.global.t('tools.ssh.readFail'), { description: e?.message })
  }
}

async function setSsh(patch: Partial<SshConfig>, msg: string) {
  sshBusy.value = true
  try {
    sshCfg.value = await api.setConfig(patch, nodeId.value)
    sshPort.value = sshCfg.value?.port || 22
    toast.success(msg)
  }
  catch (e: any) {
    toast.error(i18n.global.t('tools.ssh.updateFail'), { description: e?.message })
    await loadSsh() // 失败回弹开关状态
  }
  finally {
    sshBusy.value = false
  }
}

function applyPort() {
  const p = Number(sshPort.value)
  if (!p || p < 1 || p > 65535) {
    toast.warning(i18n.global.t('tools.ssh.portInvalid'))
    return
  }
  setSsh({ port: p }, i18n.global.t('tools.ssh.portUpdated'))
}

// ---- 密钥管理 ----
const keys = ref<SshKey[]>([])
const keysBusy = ref(false)

async function loadKeys() {
  keysBusy.value = true
  try {
    keys.value = await api.listKeys(nodeId.value)
  }
  catch {}
  finally {
    keysBusy.value = false
  }
}

// 生成
const genVisible = ref(false)
const genBusy = ref(false)
const genForm = ref({ type: 'ed25519', name: '', comment: '' })

function openGenerate() {
  genForm.value = { type: 'ed25519', name: '', comment: '' }
  genVisible.value = true
}

async function doGenerate() {
  genBusy.value = true
  try {
    await api.generateKey({ nodeId: nodeId.value, type: genForm.value.type, name: genForm.value.name, comment: genForm.value.comment })
    toast.success(i18n.global.t('tools.ssh.genDone'))
    genVisible.value = false
    await loadKeys()
  }
  catch (e: any) {
    toast.error(i18n.global.t('tools.ssh.genFail'), { description: e?.message })
  }
  finally {
    genBusy.value = false
  }
}

// 导入
const importVisible = ref(false)
const importBusy = ref(false)
const importForm = ref({ name: '', publicKey: '' })

function openImport() {
  importForm.value = { name: '', publicKey: '' }
  importVisible.value = true
}

async function doImport() {
  if (!importForm.value.publicKey.trim()) {
    toast.warning(i18n.global.t('tools.ssh.pubkeyRequired'))
    return
  }
  importBusy.value = true
  try {
    await api.importKey({ nodeId: nodeId.value, name: importForm.value.name, publicKey: importForm.value.publicKey })
    toast.success(i18n.global.t('tools.ssh.importDone'))
    importVisible.value = false
    await loadKeys()
  }
  catch (e: any) {
    toast.error(i18n.global.t('tools.ssh.importFail'), { description: e?.message })
  }
  finally {
    importBusy.value = false
  }
}

// 分发
const deployVisible = ref(false)
const deployBusy = ref(false)
const deployKey = ref<SshKey | null>(null)
const deployTarget = ref('')

const deployTargetOptions = computed(() => nodes.value
  .filter(n => n.id !== nodeId.value)
  .map(n => ({
    label: `${n.name}${n.online ? '' : i18n.global.t('tools.offlineNode')}`,
    value: n.id,
    disabled: !n.online,
  })))

function openDeploy(k: SshKey) {
  deployKey.value = k
  deployTarget.value = deployTargetOptions.value.find(o => !o.disabled)?.value || ''
  deployVisible.value = true
}

async function doDeploy() {
  if (!deployKey.value || !deployTarget.value) {
    return
  }
  deployBusy.value = true
  try {
    const out = await api.deployKey({ keyNode: nodeId.value, name: deployKey.value.name, targetNode: deployTarget.value })
    toast.success(out.result === 'exists' ? i18n.global.t('tools.ssh.deployExists') : i18n.global.t('tools.ssh.deployDone'))
    deployVisible.value = false
    await loadKeys()
  }
  catch (e: any) {
    toast.error(i18n.global.t('tools.ssh.deployFail'), { description: e?.message })
  }
  finally {
    deployBusy.value = false
  }
}

// 删除
const delVisible = ref(false)
const delBusy = ref(false)
const delKey = ref<SshKey | null>(null)
const delWithPrivate = ref(false)

function openDelete(k: SshKey) {
  delKey.value = k
  delWithPrivate.value = false
  delVisible.value = true
}

async function doDelete() {
  if (!delKey.value) {
    return
  }
  delBusy.value = true
  try {
    await api.deleteKey({ nodeId: nodeId.value, name: delKey.value.name, withPrivate: delWithPrivate.value })
    toast.success(i18n.global.t('tools.ssh.deleteDone'))
    delVisible.value = false
    await loadKeys()
  }
  catch (e: any) {
    toast.error(i18n.global.t('tools.ssh.deleteFail'), { description: e?.message })
  }
  finally {
    delBusy.value = false
  }
}

async function copyPubkey(k: SshKey) {
  try {
    await navigator.clipboard.writeText(k.publicKey)
    toast.success(i18n.global.t('tools.ssh.copied'))
  }
  catch {
    toast.error(i18n.global.t('tools.ssh.copyFail'))
  }
}

function deployedLabel(k: SshKey) {
  const on = (k.deployedOn || []).filter(d => d.deployed)
  if (!on.length) {
    return i18n.global.t('tools.ssh.notDeployed')
  }
  return on.map(d => d.nodeName).join('、')
}

async function loadAll() {
  await Promise.all([loadSsh(), loadKeys()])
}

watch(nodeId, loadAll)

onMounted(async () => {
  await loadNodes()
  await loadAll()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex gap-2 items-center">
          <FaIcon name="i-lucide:terminal" :size="22" />
          <span>{{ $t('tools.ssh.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('tools.ssh.desc') }}</span>
      </template>
      <div class="flex gap-2 items-center">
        <FaSelect v-model="nodeId" :options="nodeOptions" class="w-48" />
        <FaButton variant="outline" size="sm" :loading="keysBusy" @click="loadAll">
          <FaIcon name="i-lucide:refresh-cw" class="mr-1" /> {{ $t('common.refresh') }}
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <div class="space-y-4">
        <!-- sshd 配置 -->
        <section class="p-4 border rounded-lg bg-background">
          <div class="flex flex-wrap gap-2 items-center">
            <FaIcon name="i-lucide:settings-2" class="text-base text-primary opacity-70" />
            <span class="font-medium">{{ $t('tools.ssh.configTitle') }}</span>
            <span v-if="sshCfg" class="text-xs text-muted-foreground px-2 py-0.5 rounded-full bg-muted">{{ $t('tools.ssh.portLabel') }} {{ sshCfg.port }}</span>
          </div>
          <div v-if="sshCfg" class="mt-3 gap-3 grid grid-cols-1 lg:grid-cols-4 md:grid-cols-2">
            <div class="px-3 py-2 border rounded-md flex items-center justify-between">
              <span class="text-xs">{{ $t('tools.ssh.passwordAuth') }}</span>
              <FaSwitch :model-value="sshCfg.passwordAuth" :disabled="sshBusy" @update:model-value="(v: any) => setSsh({ passwordAuth: v }, $t('tools.ssh.passwordAuthUpdated'))" />
            </div>
            <div class="px-3 py-2 border rounded-md flex items-center justify-between">
              <span class="text-xs">{{ $t('tools.ssh.pubkeyAuth') }}</span>
              <FaSwitch :model-value="sshCfg.pubkeyAuth" :disabled="sshBusy" @update:model-value="(v: any) => setSsh({ pubkeyAuth: v }, $t('tools.ssh.pubkeyAuthUpdated'))" />
            </div>
            <div class="px-3 py-2 border rounded-md flex items-center justify-between">
              <span class="text-xs">{{ $t('tools.ssh.rootLogin') }}</span>
              <FaSelect
                :model-value="sshCfg.permitRootLogin" :disabled="sshBusy" class="w-28!"
                :options="[
                  { label: $t('tools.ssh.rootYes'), value: 'yes' },
                  { label: $t('tools.ssh.rootProhibit'), value: 'prohibit-password' },
                  { label: $t('tools.ssh.rootNo'), value: 'no' },
                ]"
                @update:model-value="(v: any) => setSsh({ permitRootLogin: v }, $t('tools.ssh.rootUpdated'))"
              />
            </div>
            <div class="px-3 py-2 border rounded-md flex items-center justify-between">
              <span class="text-xs">{{ $t('tools.ssh.portLabel') }}</span>
              <div class="flex gap-1 items-center">
                <FaInput v-model="sshPort" type="number" class="w-20" />
                <FaButton size="sm" variant="outline" :disabled="sshBusy" @click="applyPort">
                  {{ $t('tools.ssh.apply') }}
                </FaButton>
              </div>
            </div>
          </div>
          <div v-else class="text-xs text-muted-foreground mt-3">
            {{ $t('tools.ssh.reading') }}
          </div>
          <div class="text-xs text-muted-foreground mt-2">
            {{ $t('tools.ssh.configHint') }}
          </div>
        </section>

        <!-- 密钥管理 -->
        <section class="p-4 border rounded-lg bg-background">
          <div class="flex flex-wrap gap-2 items-center">
            <FaIcon name="i-lucide:key-round" class="text-base text-primary opacity-70" />
            <span class="font-medium">{{ $t('tools.ssh.keysTitle') }}</span>
            <span class="text-xs text-muted-foreground">{{ $t('tools.ssh.keysDesc') }}</span>
            <div class="ml-auto flex gap-2 items-center">
              <FaButton size="sm" @click="openGenerate">
                <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('tools.ssh.generate') }}
              </FaButton>
              <FaButton variant="outline" size="sm" @click="openImport">
                <FaIcon name="i-lucide:upload" class="mr-1" /> {{ $t('tools.ssh.import') }}
              </FaButton>
            </div>
          </div>
          <div v-if="keys.length" class="mt-3 space-y-2">
            <div v-for="k in keys" :key="k.name" class="px-3 py-2 border rounded-md">
              <div class="text-xs flex flex-wrap gap-2 items-center">
                <span class="font-medium font-mono">{{ k.name }}</span>
                <span class="font-mono px-1.5 py-0.5 rounded bg-muted">{{ k.type || '—' }}</span>
                <span class="text-muted-foreground font-mono" :title="k.comment">{{ k.fingerprint }}</span>
                <span class="px-2 py-0.5 rounded-full" :class="deployedLabel(k) === $t('tools.ssh.notDeployed') ? 'bg-muted text-muted-foreground' : 'bg-emerald-500/10 text-emerald-600'">
                  {{ $t('tools.ssh.deployedOn') }}{{ deployedLabel(k) }}
                </span>
                <div class="ml-auto flex gap-1 items-center">
                  <FaButton variant="outline" size="sm" @click="openDeploy(k)">
                    {{ $t('tools.ssh.deploy') }}
                  </FaButton>
                  <FaButton variant="ghost" size="sm" @click="copyPubkey(k)">
                    {{ $t('tools.ssh.copy') }}
                  </FaButton>
                  <FaButton variant="ghost" size="sm" class="text-red-500!" @click="openDelete(k)">
                    {{ $t('common.delete') }}
                  </FaButton>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="text-xs text-muted-foreground mt-3">
            {{ $t('tools.ssh.noKeys') }}
          </div>
        </section>
      </div>
    </FaPageMain>

    <!-- 生成密钥 -->
    <FaModal v-model="genVisible" :title="$t('tools.ssh.generate')" :destroy-on-close="true">
      <div class="text-sm space-y-3">
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-20">{{ $t('tools.ssh.keyType') }}</span>
          <FaSelect v-model="genForm.type" :options="[{ label: 'Ed25519（推荐）', value: 'ed25519' }, { label: 'RSA 4096', value: 'rsa' }]" class="flex-1" />
        </div>
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-20">{{ $t('tools.ssh.keyName') }}</span>
          <FaInput v-model="genForm.name" :placeholder="$t('tools.ssh.keyNamePlaceholder')" class="flex-1" />
        </div>
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-20">{{ $t('tools.ssh.keyComment') }}</span>
          <FaInput v-model="genForm.comment" :placeholder="$t('tools.ssh.keyCommentPlaceholder')" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          {{ $t('tools.ssh.genHint') }}
        </div>
      </div>
      <template #footer>
        <div class="flex gap-2 justify-end">
          <FaButton variant="outline" size="sm" @click="genVisible = false">
            {{ $t('common.cancel') }}
          </FaButton>
          <FaButton size="sm" :loading="genBusy" @click="doGenerate">
            {{ $t('common.confirm') }}
          </FaButton>
        </div>
      </template>
    </FaModal>

    <!-- 导入公钥 -->
    <FaModal v-model="importVisible" :title="$t('tools.ssh.import')" :destroy-on-close="true">
      <div class="text-sm space-y-3">
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-20">{{ $t('tools.ssh.keyName') }}</span>
          <FaInput v-model="importForm.name" :placeholder="$t('tools.ssh.importNamePlaceholder')" class="flex-1" />
        </div>
        <div class="flex gap-3 items-start">
          <span class="text-muted-foreground pt-2 shrink-0 w-20">{{ $t('tools.ssh.publicKey') }}</span>
          <textarea v-model="importForm.publicKey" :placeholder="$t('tools.ssh.publicKeyPlaceholder')" class="text-xs font-mono p-2 outline-none border rounded-md bg-background flex-1 h-28" />
        </div>
      </div>
      <template #footer>
        <div class="flex gap-2 justify-end">
          <FaButton variant="outline" size="sm" @click="importVisible = false">
            {{ $t('common.cancel') }}
          </FaButton>
          <FaButton size="sm" :loading="importBusy" @click="doImport">
            {{ $t('common.confirm') }}
          </FaButton>
        </div>
      </template>
    </FaModal>

    <!-- 分发 -->
    <FaModal v-model="deployVisible" :title="$t('tools.ssh.deployTitle', { name: deployKey?.name || '' })" :destroy-on-close="true">
      <div class="text-sm space-y-3">
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-24">{{ $t('tools.ssh.targetNode') }}</span>
          <FaSelect v-model="deployTarget" :options="deployTargetOptions" class="flex-1" />
        </div>
        <div class="text-[11px] text-muted-foreground font-mono p-2 border rounded-md bg-muted/30 break-all">
          {{ deployKey?.fingerprint }}
        </div>
        <div class="text-xs text-muted-foreground">
          {{ $t('tools.ssh.deployHint') }}
        </div>
      </div>
      <template #footer>
        <div class="flex gap-2 justify-end">
          <FaButton variant="outline" size="sm" @click="deployVisible = false">
            {{ $t('common.cancel') }}
          </FaButton>
          <FaButton size="sm" :loading="deployBusy" :disabled="!deployTarget" @click="doDeploy">
            {{ $t('tools.ssh.deploy') }}
          </FaButton>
        </div>
      </template>
    </FaModal>

    <!-- 删除 -->
    <FaModal v-model="delVisible" :title="$t('tools.ssh.deleteTitle', { name: delKey?.name || '' })" :destroy-on-close="true">
      <div class="text-sm space-y-3">
        <div class="text-muted-foreground">
          {{ $t('tools.ssh.deleteHint') }}
        </div>
        <label class="flex gap-2 items-center">
          <input v-model="delWithPrivate" type="checkbox" class="accent-primary">
          <span>{{ $t('tools.ssh.deleteWithPrivate') }}</span>
        </label>
      </div>
      <template #footer>
        <div class="flex gap-2 justify-end">
          <FaButton variant="outline" size="sm" @click="delVisible = false">
            {{ $t('common.cancel') }}
          </FaButton>
          <FaButton size="sm" class="text-white! bg-red-500!" :loading="delBusy" @click="doDelete">
            {{ $t('common.delete') }}
          </FaButton>
        </div>
      </template>
    </FaModal>
  </div>
</template>
