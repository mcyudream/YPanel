<script setup lang="ts">
import type { RedisKeyDetail, RedisKeyItem } from '@/api/modules/dbadmin'
import apiDBA from '@/api/modules/dbadmin'
import { i18n } from '@/locales'

// Redis 工作台：SCAN key 浏览（模式过滤+游标续页）→ 类型化详情（查看=结构化表格，编辑=行编辑）+ 命令控制台。
const props = defineProps<{
  instanceId: number
  db: string
}>()

const toast = useFaToast()
const modal = useFaModal()

// ---- key 浏览 ----
const pattern = ref('*')
const keys = ref<RedisKeyItem[]>([])
const cursor = ref('0')
const scanning = ref(false)
const selected = ref('')

async function scan(reset = true) {
  scanning.value = true
  try {
    const r = await apiDBA.redisKeys(props.instanceId, props.db, pattern.value || '*', reset ? '0' : cursor.value)
    keys.value = reset ? r.keys : [...keys.value, ...r.keys]
    cursor.value = r.cursor
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.redis.scanFailed'), { description: e?.message })
  }
  finally {
    scanning.value = false
  }
}

watch(() => [props.instanceId, props.db] as const, () => {
  keys.value = []
  selected.value = ''
  detail.value = null
  scan(true)
}, { immediate: true })

// ---- key 详情 ----
const detail = ref<RedisKeyDetail | null>(null)
const detailLoading = ref(false)

const TYPE_COLOR: Record<string, string> = {
  string: 'bg-sky-500/15 text-sky-600',
  hash: 'bg-emerald-500/15 text-emerald-600',
  list: 'bg-violet-500/15 text-violet-600',
  set: 'bg-amber-500/15 text-amber-600',
  zset: 'bg-rose-500/15 text-rose-600',
  stream: 'bg-teal-500/15 text-teal-600',
}

async function selectKey(name: string) {
  selected.value = name
  detailLoading.value = true
  editing.value = false
  try {
    detail.value = await apiDBA.redisKey(props.instanceId, props.db, name)
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.redis.keyLoadFailed'), { description: e?.message })
    detail.value = null
  }
  finally {
    detailLoading.value = false
  }
}

// ---- 编辑（结构化行编辑；保存=整体替换并审计） ----
const editing = ref(false)
const editString = ref('')
// 通用编辑行：hash(k+v) / zset(v=member+s=score) / list、set(v)
const editRows = ref<{ k: string, v: string, s: string }[]>([])
const editTTL = ref('')

function buildEditRows() {
  const d = detail.value
  if (!d) {
    return
  }
  editTTL.value = d.ttl >= 0 ? String(d.ttl) : ''
  if (d.type === 'string') {
    editString.value = d.string ?? ''
    return
  }
  const rows: { k: string, v: string, s: string }[] = []
  if (d.type === 'hash') {
    for (const [k, v] of Object.entries(d.hash ?? {})) {
      rows.push({ k, v, s: '' })
    }
  }
  else if (d.type === 'zset') {
    for (const z of d.zset ?? []) {
      rows.push({ k: '', v: z.member, s: String(z.score) })
    }
  }
  else {
    const src = d.type === 'list' ? d.list : d.set
    for (const v of src ?? []) {
      rows.push({ k: '', v, s: '' })
    }
  }
  editRows.value = rows
}

const editableType = computed(() => !!detail.value && ['string', 'hash', 'list', 'set', 'zset'].includes(detail.value.type))

function startEdit() {
  buildEditRows()
  editing.value = true
}

function cancelEdit() {
  editing.value = false
}

function addRow() {
  editRows.value.push({ k: '', v: '', s: '0' })
}

function removeRow(i: number) {
  editRows.value.splice(i, 1)
}

async function saveKey() {
  const d = detail.value
  if (!d) {
    return
  }
  let value: unknown
  if (d.type === 'string') {
    value = editString.value
  }
  else if (d.type === 'hash') {
    value = Object.fromEntries(editRows.value.filter(r => r.k.trim() !== '').map(r => [r.k.trim(), r.v]))
  }
  else if (d.type === 'list' || d.type === 'set') {
    value = editRows.value.map(r => r.v).filter(v => v !== '')
  }
  else {
    value = editRows.value
      .filter(r => r.v.trim() !== '')
      .map(r => ({ member: r.v.trim(), score: Number(r.s) || 0 }))
  }
  const ok = await modal.confirm({
    title: i18n.global.t('dbadmin.redis.writeTitle'),
    content: i18n.global.t('dbadmin.redis.writeConfirm', { db: props.db, name: d.name }),
  })
  if (!ok) {
    return
  }
  try {
    await apiDBA.redisKeyWrite(props.instanceId, { db: props.db, name: d.name, type: d.type, value })
    toast.success(i18n.global.t('dbadmin.redis.written'))
    editing.value = false
    selectKey(d.name)
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.redis.writeFailed'), { description: e?.message })
  }
}

async function delKey() {
  const d = detail.value
  if (!d) {
    return
  }
  const ok = await modal.confirm({ title: i18n.global.t('dbadmin.redis.deleteTitle'), content: i18n.global.t('dbadmin.redis.deleteConfirm', { db: props.db, name: d.name }) })
  if (!ok) {
    return
  }
  try {
    await apiDBA.redisKeyDelete(props.instanceId, { db: props.db, name: d.name })
    toast.success(i18n.global.t('dbadmin.common.deleted'))
    selected.value = ''
    detail.value = null
    scan(true)
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.common.deleteFailed'), { description: e?.message })
  }
}

async function saveTTL() {
  const d = detail.value
  if (!d) {
    return
  }
  const text = editTTL.value.trim()
  const ttl = text === '' ? -1 : Number(text)
  if (text !== '' && (Number.isNaN(ttl) || ttl < -1)) {
    toast.warning(i18n.global.t('dbadmin.redis.ttlInvalid'))
    return
  }
  try {
    await apiDBA.redisKeyTTL(props.instanceId, { db: props.db, name: d.name, ttl })
    toast.success(ttl < 0 ? i18n.global.t('dbadmin.redis.ttlPermanent') : i18n.global.t('dbadmin.redis.ttlSet', { n: ttl }))
    selectKey(d.name)
  }
  catch (e: any) {
    toast.error(i18n.global.t('dbadmin.redis.setFailed'), { description: e?.message })
  }
}

// ---- 命令控制台 ----
const consoleInput = ref('')
const consoleLines = ref<{ cmd: string, out: string, ok: boolean }[]>([])
const consoleBusy = ref(false)

async function execCommand() {
  const cmd = consoleInput.value.trim()
  if (!cmd || consoleBusy.value) {
    return
  }
  consoleBusy.value = true
  try {
    const out = await apiDBA.redisExec(props.instanceId, props.db, cmd)
    consoleLines.value.push({ cmd, out, ok: !out.startsWith('ERR') })
  }
  catch (e: any) {
    consoleLines.value.push({ cmd, out: e?.message || i18n.global.t('dbadmin.redis.execFailed'), ok: false })
  }
  finally {
    consoleBusy.value = false
    consoleInput.value = ''
    // 命令可能增删 key，从头重扫（追加模式会导致 key 重复出现）
    scan(true)
  }
}
</script>

<template>
  <div class="flex size-full min-h-0 gap-3">
    <!-- key 列表 -->
    <div class="flex w-64 shrink-0 flex-col gap-2">
      <div class="flex items-center gap-1">
        <input
          v-model="pattern"
          class="flex-1 rounded-md border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
          :placeholder="$t('dbadmin.redis.patternPlaceholder')"
          @keydown.enter="scan(true)"
        >
        <FaButton size="sm" :loading="scanning" @click="scan(true)">{{ $t('dbadmin.redis.scan') }}</FaButton>
      </div>
      <div class="min-h-0 flex-1 overflow-auto rounded-md border">
        <button
          v-for="k in keys"
          :key="k.name"
          type="button"
          class="flex w-full items-center gap-1.5 px-2 py-1 text-left font-mono text-xs transition-colors hover:bg-accent/50"
          :class="{ 'bg-primary/10': selected === k.name }"
          :title="k.name"
          @click="selectKey(k.name)"
        >
          <span class="shrink-0 rounded px-1 text-[9px]" :class="TYPE_COLOR[k.type] ?? 'bg-muted text-muted-foreground'">{{ k.type }}</span>
          <span class="truncate">{{ k.name }}</span>
        </button>
        <div v-if="!keys.length && !scanning" class="px-2 py-4 text-center text-xs text-muted-foreground">{{ $t('dbadmin.redis.noKeys') }}</div>
      </div>
      <button
        v-if="cursor !== '0'"
        type="button"
        class="rounded border px-2 py-1 text-xs hover:bg-accent/50"
        :disabled="scanning"
        @click="scan(false)"
      >
        {{ $t('dbadmin.redis.loadMore', { cursor }) }}
      </button>
    </div>

    <!-- 详情：查看（结构化）/ 编辑（行编辑） -->
    <div class="flex min-h-0 flex-1 flex-col rounded-md border">
      <div class="flex flex-wrap items-center gap-2 border-b px-3 py-2 text-xs">
        <template v-if="detail">
          <span class="font-mono text-sm font-medium">{{ detail.name }}</span>
          <span class="rounded px-1.5 py-0.5 text-[10px]" :class="TYPE_COLOR[detail.type] ?? 'bg-muted'">{{ detail.type }}</span>
          <span class="text-muted-foreground">{{ detail.length }} {{ detail.type === 'string' ? 'bytes' : $t('dbadmin.redis.items') }}<span v-if="detail.preview">{{ $t('dbadmin.redis.previewTruncated') }}</span></span>
          <span class="text-muted-foreground">
            {{ $t('dbadmin.redis.ttlLabel') }}<span :class="detail.ttl >= 0 ? 'text-amber-600' : ''">{{ detail.ttl >= 0 ? `${detail.ttl}s` : $t('dbadmin.redis.neverExpires') }}</span>
          </span>
          <span class="flex-1" />
          <input
            v-model="editTTL"
            class="w-28 rounded border border-input bg-background px-2 py-1 text-xs outline-none focus:ring-1 focus:ring-primary"
            :placeholder="$t('dbadmin.redis.ttlPlaceholder')"
          >
          <FaButton size="sm" @click="saveTTL">{{ $t('dbadmin.redis.setTtl') }}</FaButton>
          <template v-if="editableType">
            <template v-if="!editing">
              <FaButton size="sm" type="primary" @click="startEdit">{{ $t('common.edit') }}</FaButton>
            </template>
            <template v-else>
              <FaButton size="sm" @click="cancelEdit">{{ $t('common.cancel') }}</FaButton>
              <FaButton size="sm" type="primary" @click="saveKey">{{ $t('dbadmin.common.saveChanges') }}</FaButton>
            </template>
          </template>
          <FaButton size="sm" status="danger" @click="delKey">{{ $t('common.delete') }}</FaButton>
        </template>
      </div>

      <div class="min-h-0 flex-1 overflow-auto p-3">
        <div v-if="!selected" class="flex size-full items-center justify-center text-sm text-muted-foreground">
          {{ $t('dbadmin.redis.selectKeyHint') }}
        </div>
        <div v-else-if="detailLoading" class="flex size-full items-center justify-center text-sm text-muted-foreground">
          {{ $t('common.loading') }}
        </div>

        <!-- 编辑态 -->
        <template v-else-if="detail && editing">
          <textarea v-if="detail.type === 'string'" v-model="editString" class="h-72 w-full resize-y rounded-md border border-input bg-background p-2 font-mono text-xs outline-none focus:ring-1 focus:ring-primary" spellcheck="false" />
          <div v-else class="space-y-1.5">
            <div v-for="(row, i) in editRows" :key="i" class="flex items-center gap-1.5">
              <input
                v-if="detail.type === 'hash'"
                v-model="row.k"
                class="w-40 rounded border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
                placeholder="field"
              >
              <input
                v-model="row.v"
                class="flex-1 rounded border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
                :placeholder="detail.type === 'zset' ? 'member' : 'value'"
              >
              <input
                v-if="detail.type === 'zset'"
                v-model="row.s"
                class="w-24 rounded border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
                placeholder="score"
              >
              <button type="button" class="rounded p-1 text-muted-foreground hover:text-red-500" :title="$t('dbadmin.common.deleteRow')" @click="removeRow(i)">
                <YdMorphIcon name="trash" :size="13" />
              </button>
            </div>
            <FaButton v-if="detail.type !== 'string'" size="sm" variant="outline" @click="addRow">
              <YdMorphIcon name="plus" :size="12" /> {{ $t('dbadmin.redis.addRow') }}
            </FaButton>
            <p class="text-xs text-muted-foreground">{{ $t('dbadmin.redis.editHint') }}</p>
          </div>
        </template>

        <!-- 查看态（结构化） -->
        <template v-else-if="detail">
          <pre v-if="detail.type === 'string'" class="max-h-full overflow-auto whitespace-pre-wrap rounded-md bg-muted/30 p-3 font-mono text-xs leading-relaxed">{{ detail.binary ? detail.string : (detail.string || $t('dbadmin.redis.emptyString')) }}</pre>

          <table v-else-if="detail.type === 'hash'" class="w-full text-xs">
            <thead class="bg-muted/50 text-left text-muted-foreground"><tr><th class="px-2 py-1.5">field</th><th class="px-2 py-1.5">value</th></tr></thead>
            <tbody>
              <tr v-for="(v, k) in detail.hash" :key="k" class="border-t hover:bg-accent/30">
                <td class="px-2 py-1 font-mono text-emerald-700">{{ k }}</td>
                <td class="break-all px-2 py-1 font-mono">{{ v }}</td>
              </tr>
            </tbody>
          </table>

          <table v-else-if="detail.type === 'zset'" class="w-full text-xs">
            <thead class="bg-muted/50 text-left text-muted-foreground"><tr><th class="w-10 px-2 py-1.5">#</th><th class="px-2 py-1.5">member</th><th class="px-2 py-1.5 text-right">score</th></tr></thead>
            <tbody>
              <tr v-for="(z, i) in detail.zset" :key="i" class="border-t hover:bg-accent/30">
                <td class="px-2 py-1 tabular-nums text-muted-foreground">{{ i + 1 }}</td>
                <td class="break-all px-2 py-1 font-mono">{{ z.member }}</td>
                <td class="px-2 py-1 text-right font-mono tabular-nums text-rose-600">{{ z.score }}</td>
              </tr>
            </tbody>
          </table>

          <table v-else-if="detail.type === 'list'" class="w-full text-xs">
            <tbody>
              <tr v-for="(v, i) in detail.list" :key="i" class="border-t hover:bg-accent/30">
                <td class="w-10 px-2 py-1 text-right tabular-nums text-muted-foreground">{{ i }}</td>
                <td class="break-all px-2 py-1 font-mono">{{ v }}</td>
              </tr>
            </tbody>
          </table>

          <div v-else-if="detail.type === 'set'" class="flex flex-wrap gap-1.5">
            <span v-for="(v, i) in detail.set" :key="i" class="break-all rounded-md bg-amber-500/10 px-2 py-1 font-mono text-xs">{{ v }}</span>
          </div>

          <div v-else-if="detail.type === 'stream'" class="space-y-1.5">
            <div v-for="e in detail.stream" :key="e.id" class="rounded border p-2 font-mono text-xs">
              <div class="text-muted-foreground">{{ e.id }}</div>
              <div v-for="(v, k) in e.fields" :key="k">{{ k }}: {{ v }}</div>
            </div>
          </div>

          <div v-if="detail.preview" class="mt-2 rounded bg-amber-500/10 px-2 py-1 text-xs text-amber-600">
            {{ $t('dbadmin.redis.largeValue', { n: detail.length }) }}
          </div>
        </template>
      </div>
    </div>

    <!-- 命令控制台 -->
    <div class="flex w-80 shrink-0 flex-col rounded-md border">
      <div class="border-b px-2 py-1.5 text-xs font-medium text-muted-foreground">{{ $t('dbadmin.redis.consoleTitle') }}</div>
      <div class="min-h-0 flex-1 overflow-auto p-2 font-mono text-xs">
        <div v-for="(l, i) in consoleLines" :key="i" class="mb-1.5">
          <div class="text-primary">&gt; {{ l.cmd }}</div>
          <pre class="whitespace-pre-wrap" :class="l.ok ? '' : 'text-red-500'">{{ l.out }}</pre>
        </div>
        <div v-if="!consoleLines.length" class="text-muted-foreground">{{ $t('dbadmin.redis.consoleHint') }}</div>
      </div>
      <div class="border-t p-2">
        <input
          v-model="consoleInput"
          class="w-full rounded border border-input bg-background px-2 py-1 font-mono text-xs outline-none focus:ring-1 focus:ring-primary"
          placeholder="GET foo"
          :disabled="consoleBusy"
          @keydown.enter="execCommand"
        >
      </div>
    </div>
  </div>
</template>
