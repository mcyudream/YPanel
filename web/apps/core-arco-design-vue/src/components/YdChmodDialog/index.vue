<script setup lang="ts">
import type { FileEntry, FileOwnersResp } from '@/api/modules/file'
import apiFile from '@/api/modules/file'
import apiCFile from '@/api/modules/cfile'
import { i18n } from '@/locales'

// 权限/属主编辑弹窗（宿主与容器双模式）：读写执行矩阵 ↔ 八进制双向联动 + 常用预设 +
// 递归选项 + 属主修改（宿主=系统用户/组下拉，容器=自由输入，走容器内 exec）。
// 组件自持 FaModal 与 visible（defineModel），随打开重建（FaModal 插槽 defineModel 断链规避，见 exp/frontend.md）。
const visible = defineModel<boolean>('visible', { default: false })

const props = withDefaults(defineProps<{
  /** 目标路径（1 条或多条批量） */
  paths: string[]
  /** 目标条目（用于回填当前权限/属主；与 paths 一一对应，可只传首个） */
  entries?: FileEntry[]
  /** 宿主模式的节点 id（默认 local） */
  node?: string
  /** 容器模式：容器 ID/名称（存在即容器模式，node 失效） */
  containerId?: string
}>(), {
  entries: () => [],
  node: 'local',
  containerId: '',
})

const emits = defineEmits<{
  /** 任一项修改成功后触发（调用方刷新列表） */
  done: []
}>()

const UNCHANGED = '__unchanged__'

const modeInput = ref('644')
const recursive = ref(false)
const ownerSel = ref(UNCHANGED)
const groupSel = ref(UNCHANGED)
const owners = ref<FileOwnersResp>({ users: [], groups: [] })
const ownersLoaded = ref(false)
const applying = ref(false)

const isContainer = computed(() => !!props.containerId)

// ---- 权限矩阵（状态源是矩阵，八进制输入与之双向同步；4 位时最高位为特殊位仅输入框可改） ----
const perms = reactive({
  owner: { r: true, w: true, x: false },
  group: { r: true, w: false, x: false },
  other: { r: true, w: false, x: false },
})

function digitOf(o: { r: boolean, w: boolean, x: boolean }) {
  return (o.r ? 4 : 0) + (o.w ? 2 : 0) + (o.x ? 1 : 0)
}

function specialBit() {
  const m = modeInput.value.trim()
  return /^[0-7]$/.test(m.slice(0, 1)) && m.length === 4 ? m.slice(0, 1) : ''
}

function syncInputFromPerms() {
  modeInput.value = specialBit() + `${digitOf(perms.owner)}${digitOf(perms.group)}${digitOf(perms.other)}`
}

function applyToPerms(tri: string) {
  const d = (n: string) => ({
    r: (Number(n) & 4) !== 0,
    w: (Number(n) & 2) !== 0,
    x: (Number(n) & 1) !== 0,
  })
  Object.assign(perms.owner, d(tri[0]))
  Object.assign(perms.group, d(tri[1]))
  Object.assign(perms.other, d(tri[2]))
}

function syncPermsFromInput() {
  const m = modeInput.value.trim()
  if (/^[0-7]{3,4}$/.test(m)) {
    applyToPerms(m.slice(-3))
  }
}

function applyPreset(v: string) {
  modeInput.value = v
  syncPermsFromInput()
}

/** 权限串解读（如 rwxr-xr-x） */
const modeText = computed(() => {
  const m = modeInput.value.trim()
  if (!/^[0-7]{3,4}$/.test(m)) {
    return '—'
  }
  const name = (n: number) => `${n & 4 ? 'r' : '-'}${n & 2 ? 'w' : '-'}${n & 1 ? 'x' : '-'}`
  return m.slice(-3).split('').map(c => name(Number(c))).join('')
})

// ---- 目标展示 ----
const targetLabel = computed(() => {
  if (props.paths.length === 1) {
    return props.paths[0]
  }
  const names = (props.entries || []).slice(0, 3).map(e => e.name)
  return i18n.global.t('components.ydChmodDialog.selectedItems', { n: props.paths.length, names: names.join(i18n.global.t('components.ydChmodDialog.nameSep')) }) + (props.paths.length > 3 ? ' …' : '')
})

const showRecursive = computed(() => props.paths.length > 1 || !!props.entries?.[0]?.isDir)

// ---- 打开时初始化 ----
watch(visible, (v) => {
  if (!v) {
    return
  }
  const first = props.entries?.[0]
  const oct = first?.modeOct?.replace(/^0(?=[0-7]{3}$)/, '') || '644'
  modeInput.value = /^[0-7]{3,4}$/.test(oct) ? oct : '644'
  syncPermsFromInput()
  recursive.value = false
  ownerSel.value = isContainer.value ? '' : UNCHANGED
  groupSel.value = isContainer.value ? '' : UNCHANGED
  owners.value = { users: [], groups: [] }
  ownersLoaded.value = false
  if (!isContainer.value) {
    loadOwners(first)
  }
})

async function loadOwners(first?: FileEntry) {
  try {
    owners.value = await apiFile.owners(props.node)
  }
  catch {
    // 枚举失败不阻塞 chmod；属主保持"不更改"（仍可 chmod）
  }
  ownersLoaded.value = true
  // 当前属主在枚举结果中时回填选中（便于确认不改也看得见现状）
  if (first?.owner && owners.value.users.some(u => u.name === first.owner || u.id === first.owner)) {
    ownerSel.value = first.owner
  }
  if (first?.group && owners.value.groups.some(g => g.name === first.group || g.id === first.group)) {
    groupSel.value = first.group
  }
}

// ---- 执行 ----
function doChmod(path: string, mode: string) {
  return isContainer.value
    ? apiCFile.chmod(props.containerId, path, mode, recursive.value)
    : apiFile.chmod(path, mode, props.node, recursive.value)
}

function doChown(path: string) {
  const owner = ownerSel.value === UNCHANGED ? '' : ownerSel.value.trim()
  const group = groupSel.value === UNCHANGED ? '' : groupSel.value.trim()
  if (!owner && !group) {
    return null
  }
  return isContainer.value
    ? apiCFile.chown(props.containerId, path, owner, group, recursive.value)
    : apiFile.chown(path, owner, group, props.node, recursive.value)
}

async function apply() {
  const mode = modeInput.value.trim()
  if (!/^[0-7]{3,4}$/.test(mode)) {
    useFaToast().error(i18n.global.t('components.ydChmodDialog.invalidMode'))
    return
  }
  applying.value = true
  let ok = 0
  let fail = 0
  let firstErr = ''
  for (const p of props.paths) {
    try {
      await doChmod(p, mode)
      await doChown(p)
      ok++
    }
    catch (e: unknown) {
      fail++
      if (!firstErr) {
        firstErr = e && typeof e === 'object' && 'message' in e ? String((e as { message: unknown }).message) : String(e)
      }
    }
  }
  applying.value = false
  if (ok && !fail) {
    useFaToast().success(i18n.global.t('components.ydChmodDialog.modified', { n: ok }))
  }
  else if (ok && fail) {
    useFaToast().warning(i18n.global.t('components.ydChmodDialog.partialFailed', { ok, fail }), { description: firstErr })
  }
  else {
    useFaToast().error(i18n.global.t('components.ydChmodDialog.applyFailed'), { description: firstErr })
  }
  if (ok) {
    visible.value = false
    emits('done')
  }
}
</script>

<template>
  <FaModal v-model="visible" :title="$t('components.ydChmodDialog.title')" class="max-w-xl!" :destroy-on-close="true">
    <div class="space-y-4 text-sm">
      <div class="truncate rounded-md bg-muted/50 px-2.5 py-1.5 font-mono text-xs text-muted-foreground" :title="paths.join('\n')">
        {{ targetLabel }}
      </div>

      <!-- 权限矩阵 -->
      <div class="grid grid-cols-[3.5rem_repeat(3,1fr)] items-center gap-x-3 gap-y-2">
        <span class="text-muted-foreground">{{ $t('components.ydChmodDialog.owner') }}</span>
        <label class="flex cursor-pointer items-center gap-1.5"><input v-model="perms.owner.r" type="checkbox" @change="syncInputFromPerms"> {{ $t('components.ydChmodDialog.read') }}</label>
        <label class="flex cursor-pointer items-center gap-1.5"><input v-model="perms.owner.w" type="checkbox" @change="syncInputFromPerms"> {{ $t('components.ydChmodDialog.write') }}</label>
        <label class="flex cursor-pointer items-center gap-1.5"><input v-model="perms.owner.x" type="checkbox" @change="syncInputFromPerms"> {{ $t('components.ydChmodDialog.exec') }}</label>

        <span class="text-muted-foreground">{{ $t('components.ydChmodDialog.group') }}</span>
        <label class="flex cursor-pointer items-center gap-1.5"><input v-model="perms.group.r" type="checkbox" @change="syncInputFromPerms"> {{ $t('components.ydChmodDialog.read') }}</label>
        <label class="flex cursor-pointer items-center gap-1.5"><input v-model="perms.group.w" type="checkbox" @change="syncInputFromPerms"> {{ $t('components.ydChmodDialog.write') }}</label>
        <label class="flex cursor-pointer items-center gap-1.5"><input v-model="perms.group.x" type="checkbox" @change="syncInputFromPerms"> {{ $t('components.ydChmodDialog.exec') }}</label>

        <span class="text-muted-foreground">{{ $t('components.ydChmodDialog.others') }}</span>
        <label class="flex cursor-pointer items-center gap-1.5"><input v-model="perms.other.r" type="checkbox" @change="syncInputFromPerms"> {{ $t('components.ydChmodDialog.read') }}</label>
        <label class="flex cursor-pointer items-center gap-1.5"><input v-model="perms.other.w" type="checkbox" @change="syncInputFromPerms"> {{ $t('components.ydChmodDialog.write') }}</label>
        <label class="flex cursor-pointer items-center gap-1.5"><input v-model="perms.other.x" type="checkbox" @change="syncInputFromPerms"> {{ $t('components.ydChmodDialog.exec') }}</label>
      </div>

      <!-- 八进制 + 解读 + 预设 -->
      <div class="flex flex-wrap items-center gap-2">
        <span class="w-14 shrink-0 text-muted-foreground">{{ $t('components.ydChmodDialog.perm') }}</span>
        <FaInput
          v-model="modeInput"
          :placeholder="$t('components.ydChmodDialog.permPlaceholder')"
          class="w-28!"
          @input="syncPermsFromInput"
        />
        <span class="font-mono text-xs text-muted-foreground">{{ modeText }}</span>
        <div class="ml-auto flex items-center gap-1">
          <FaButton v-for="p in ['600', '644', '700', '755']" :key="p" variant="outline" size="icon-sm" class="font-mono text-xs!" @click="applyPreset(p)">
            {{ p }}
          </FaButton>
        </div>
      </div>

      <!-- 属主 -->
      <div class="grid grid-cols-[3.5rem_1fr] items-center gap-x-3 gap-y-2">
        <span class="text-muted-foreground">{{ $t('components.ydChmodDialog.user') }}</span>
        <YdSelect v-if="!isContainer" v-model="ownerSel" size="sm" :options="[{ label: $t('components.ydChmodDialog.unchanged'), value: UNCHANGED }, ...owners.users.map(u => ({ label: $t('components.ydChmodDialog.ownerEntry', { name: u.name, id: u.id }), value: u.name }))]" button-class="w-full" />
        <FaInput v-else v-model="ownerSel" :placeholder="$t('components.ydChmodDialog.ownerInputPlaceholder')" class="w-full" />

        <span class="text-muted-foreground">{{ $t('components.ydChmodDialog.group') }}</span>
        <YdSelect v-if="!isContainer" v-model="groupSel" size="sm" :options="[{ label: $t('components.ydChmodDialog.unchanged'), value: UNCHANGED }, ...owners.groups.map(g => ({ label: $t('components.ydChmodDialog.ownerEntry', { name: g.name, id: g.id }), value: g.name }))]" button-class="w-full" />
        <FaInput v-else v-model="groupSel" :placeholder="$t('components.ydChmodDialog.groupInputPlaceholder')" class="w-full" />
      </div>

      <!-- 递归 -->
      <label v-if="showRecursive" class="flex cursor-pointer items-center gap-2">
        <input v-model="recursive" type="checkbox">
        <span>{{ $t('components.ydChmodDialog.recursive') }}</span>
      </label>
      <div v-if="isContainer" class="text-xs text-muted-foreground">
        {{ $t('components.ydChmodDialog.containerHint') }}
      </div>
    </div>

    <template #footer>
      <FaButton variant="outline" @click="visible = false">
        {{ $t('common.cancel') }}
      </FaButton>
      <FaButton :loading="applying" @click="apply">
        {{ $t('common.apply') }}{{ paths.length > 1 ? $t('components.ydChmodDialog.itemCount', { n: paths.length }) : '' }}
      </FaButton>
    </template>
  </FaModal>
</template>
