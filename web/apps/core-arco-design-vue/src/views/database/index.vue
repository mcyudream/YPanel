<script setup lang="ts">
import type { DbBackup, DbDatabase, DbInstance, DbUser } from '@/api/modules/database'
import apiDb from '@/api/modules/database'

defineOptions({
  name: 'DatabaseIndex',
})

const appAccountStore = useAppAccountStore()

const instances = ref<DbInstance[]>([])
const loading = ref(false)
let timer: ReturnType<typeof setInterval> | null = null

const TYPE_META: Record<string, { label: string, icon: string, color: string }> = {
  mysql: { label: 'MySQL', icon: 'database', color: '#00758f' },
  postgres: { label: 'PostgreSQL', icon: 'database-zap', color: '#336791' },
  redis: { label: 'Redis', icon: 'database-backup', color: '#dc382d' },
  mongo: { label: 'MongoDB', icon: 'database-zap', color: '#47a248' },
}

async function load() {
  loading.value = true
  try {
    instances.value = await apiDb.list()
  }
  finally {
    loading.value = false
  }
}

// ---- 创建 ----
const createVisible = ref(false)
const createForm = ref({ name: '', type: 'mysql', port: 0, password: '' })
const creating = ref(false)

function openCreate() {
  createForm.value = { name: '', type: 'mysql', port: 0, password: '' }
  createVisible.value = true
}

async function doCreate() {
  creating.value = true
  try {
    await apiDb.create(createForm.value)
    useFaToast().success('实例创建中，容器就绪后即可连接（首次拉取镜像可能较慢）')
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

function remove(inst: DbInstance) {
  const external = inst.origin === 'external'
  const modal = useFaModal()
  modal.confirm({
    title: external ? '解除纳管' : '删除实例',
    content: external
      ? `确认解除对 ${inst.name} 的纳管？远端实例不受影响。`
      : `确认删除 ${inst.name}？选择"清除"将同时移除容器数据与备份；否则仅摘除管理。`,
    onConfirm: async () => {
      await apiDb.remove(inst.id, false)
      useFaToast().success(external ? '已解除纳管' : '已删除（数据保留，可 purge 清理）')
      await load()
    },
  })
}

async function toggleRun(inst: DbInstance) {
  if (inst.origin === 'external') {
    useFaToast().info('外部实例由其所在主机管理')
    return
  }
  try {
    if (inst.running) {
      await apiDb.stop(inst.id)
      useFaToast().success('已停止')
    }
    else {
      await apiDb.start(inst.id)
      useFaToast().success('已启动')
    }
    await load()
  }
  catch (e: any) {
    useFaToast().error('操作失败', { description: e?.message })
  }
}

// ---- 接入外部实例 ----
const extVisible = ref(false)
const extForm = ref({ name: '', type: 'mysql', host: '127.0.0.1', port: 3306, user: 'root', password: '', remark: '' })
const extSaving = ref(false)

function openExternal() {
  extForm.value = { name: '', type: 'mysql', host: '127.0.0.1', port: 3306, user: 'root', password: '', remark: '' }
  extVisible.value = true
}

function extType(t: string) {
  const ports: Record<string, number> = { mysql: 3306, postgres: 5432, redis: 6379, mongo: 27017 }
  const users: Record<string, string> = { mysql: 'root', postgres: 'postgres', redis: 'default', mongo: 'root' }
  extForm.value.port = ports[t] || 3306
  extForm.value.user = users[t] || 'root'
}

async function doCreateExternal() {
  extSaving.value = true
  try {
    await apiDb.createExternal(extForm.value)
    useFaToast().success('外部实例已接入（直连纳管）')
    extVisible.value = false
    await load()
  }
  catch (e: any) {
    useFaToast().error('接入失败', { description: e?.message })
  }
  finally {
    extSaving.value = false
  }
}

// ---- 连接信息 ----
const connVisible = ref(false)
const conn = ref<{ user: string, password: string, host: string, port: number } | null>(null)

async function showConn(inst: DbInstance) {
  try {
    conn.value = await apiDb.reveal(inst.id)
    connVisible.value = true
  }
  catch (e: any) {
    useFaToast().error('获取失败', { description: e?.message })
  }
}

// ---- 管理面板 ----
const active = ref<DbInstance | null>(null)
const tab = ref<'databases' | 'users' | 'backups'>('databases')
const panelLoading = ref(false)
const databases = ref<DbDatabase[]>([])
const users = ref<DbUser[]>([])
const backups = ref<DbBackup[]>([])

function openPanel(inst: DbInstance) {
  active.value = inst
  tab.value = 'databases'
  refreshPanel()
}

async function refreshPanel() {
  if (!active.value) {
    return
  }
  panelLoading.value = true
  const id = active.value.id
  try {
    if (tab.value === 'databases') {
      databases.value = await apiDb.databases(id)
    }
    else if (tab.value === 'users') {
      users.value = await apiDb.users(id)
    }
    else {
      backups.value = await apiDb.backups(id)
    }
  }
  catch (e: any) {
    useFaToast().error('读取失败', { description: e?.message })
  }
  finally {
    panelLoading.value = false
  }
}

watch(tab, refreshPanel)

// 库操作
const dbModalVisible = ref(false)
const dbForm = ref({ name: '', charset: 'utf8mb4' })

function openCreateDb() {
  dbForm.value = { name: '', charset: 'utf8mb4' }
  dbModalVisible.value = true
}

async function doCreateDb() {
  if (!active.value) {
    return
  }
  try {
    await apiDb.createDatabase(active.value.id, dbForm.value.name, dbForm.value.charset)
    useFaToast().success('已创建')
    dbModalVisible.value = false
    refreshPanel()
  }
  catch (e: any) {
    useFaToast().error('创建失败', { description: e?.message })
  }
}

function dropDb(name: string) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除数据库',
    content: `确认删除数据库 ${name}？不可恢复。`,
    onConfirm: async () => {
      if (!active.value) {
        return
      }
      try {
        await apiDb.dropDatabase(active.value.id, name)
        useFaToast().success('已删除')
        refreshPanel()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

// 用户操作
const userModalVisible = ref(false)
const userForm = ref({ name: '', host: '%', password: '' })

function openCreateUser() {
  userForm.value = { name: '', host: '%', password: '' }
  userModalVisible.value = true
}

async function doCreateUser() {
  if (!active.value) {
    return
  }
  try {
    await apiDb.createUser(active.value.id, userForm.value)
    useFaToast().success('已创建')
    userModalVisible.value = false
    refreshPanel()
  }
  catch (e: any) {
    useFaToast().error('创建失败', { description: e?.message })
  }
}

function dropUser(u: DbUser) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除用户',
    content: `确认删除用户 ${u.name}${u.host ? `@${u.host}` : ''}？`,
    onConfirm: async () => {
      if (!active.value) {
        return
      }
      try {
        await apiDb.dropUser(active.value.id, u.name, u.host)
        useFaToast().success('已删除')
        refreshPanel()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

const pwdModalVisible = ref(false)
const pwdForm = ref({ name: '', host: '', password: '' })

function openChangePwd(u: DbUser) {
  pwdForm.value = { name: u.name, host: u.host, password: '' }
  pwdModalVisible.value = true
}

async function doChangePwd() {
  if (!active.value) {
    return
  }
  try {
    await apiDb.changeUserPassword(active.value.id, pwdForm.value.name, pwdForm.value.password, pwdForm.value.host)
    useFaToast().success('密码已修改')
    pwdModalVisible.value = false
  }
  catch (e: any) {
    useFaToast().error('修改失败', { description: e?.message })
  }
}

// 备份操作
const backupBusy = ref(false)

async function doBackup() {
  if (!active.value) {
    return
  }
  backupBusy.value = true
  useFaToast().info('备份执行中…')
  try {
    const out = await apiDb.createBackup(active.value.id)
    if (out.file) {
      useFaToast().success(`备份完成：${out.file}`)
    }
    refreshPanel()
  }
  catch (e: any) {
    useFaToast().error('备份失败', { description: e?.message })
  }
  finally {
    backupBusy.value = false
  }
}

function downloadBackup(b: DbBackup) {
  window.open(apiDb.backupDownloadURL(b.path, appAccountStore.token))
}

function removeBackup(b: DbBackup) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除备份',
    content: `确认删除备份文件 ${b.name}？`,
    onConfirm: async () => {
      if (!active.value) {
        return
      }
      try {
        await apiDb.deleteBackup(active.value.id, b.name)
        useFaToast().success('已删除')
        refreshPanel()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

function restoreBackup(b: DbBackup) {
  const modal = useFaModal()
  modal.confirm({
    title: '恢复备份',
    content: `确认从 ${b.name} 恢复？当前数据将被覆盖。`,
    onConfirm: async () => {
      if (!active.value) {
        return
      }
      try {
        await apiDb.restoreBackup(active.value.id, b.name)
        useFaToast().success('恢复完成')
      }
      catch (e: any) {
        useFaToast().error('恢复失败', { description: e?.message })
      }
    },
  })
}

onMounted(() => {
  load()
  timer = setInterval(load, 8000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="database" :size="24" />
          <span>数据库</span>
        </div>
      </template>
      <template #description>
        <span>MySQL / PostgreSQL / Redis / MongoDB 实例：容器化安装、库/用户管理、备份恢复</span>
      </template>
      <div class="flex gap-2">
        <FaButton size="sm" variant="outline" @click="openExternal">
          <FaIcon name="i-lucide:plug-zap" class="mr-1" /> 接入外部实例
        </FaButton>
        <FaButton size="sm" @click="openCreate">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 创建实例
        </FaButton>
      </div>
    </FaPageHeader>

    <FaPageMain>
      <!-- 实例卡片 -->
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="inst in instances"
          :key="inst.id"
          class="rounded-lg border bg-background p-4 transition-shadow hover:shadow-md"
        >
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-2">
              <YdMorphIcon :name="TYPE_META[inst.type]?.icon || 'database'" :size="22" :color="TYPE_META[inst.type]?.color" />
              <div>
                <div class="flex items-center gap-1.5">
                  <span class="font-medium">{{ inst.name }}</span>
                  <span
                    class="rounded px-1.5 py-0.5 text-xs"
                    :class="inst.origin === 'external' ? 'bg-blue-500/10 text-blue-600' : 'bg-muted text-muted-foreground'"
                  >
                    {{ inst.origin === 'external' ? '外部接管' : '容器' }}
                  </span>
                </div>
                <div class="text-xs text-muted-foreground">{{ TYPE_META[inst.type]?.label }} · {{ inst.host || '127.0.0.1' }}:{{ inst.port }}</div>
              </div>
            </div>
            <span class="rounded-full px-2 py-0.5 text-xs" :class="inst.running ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
              {{ inst.running ? '运行中' : '已停止' }}
            </span>
          </div>
          <div class="mt-3 flex flex-wrap items-center gap-1.5">
            <FaButton variant="outline" size="sm" @click="openPanel(inst)">
              管理
            </FaButton>
            <FaButton v-if="inst.origin !== 'external'" variant="ghost" size="sm" @click="toggleRun(inst)">
              {{ inst.running ? '停止' : '启动' }}
            </FaButton>
            <FaButton variant="ghost" size="sm" @click="showConn(inst)">
              连接信息
            </FaButton>
            <FaButton variant="ghost" size="sm" class="ml-auto text-red-500!" @click="remove(inst)">
              {{ inst.origin === 'external' ? '解除' : '删除' }}
            </FaButton>
          </div>
        </div>
        <div v-if="!instances.length && !loading" class="rounded-lg border p-10 text-center text-sm text-muted-foreground md:col-span-2 xl:col-span-3">
          暂无数据库实例，点击右上角"创建实例"
        </div>
      </div>

      <!-- 管理面板 -->
      <div v-if="active" class="mt-4 rounded-lg border bg-background">
        <div class="flex flex-wrap items-center gap-2 border-b px-4 py-3">
          <span class="font-medium">{{ active.name }} 管理</span>
          <FaTabs
            v-model="tab" :list="[
              { label: '数据库', value: 'databases' },
              { label: '用户', value: 'users' },
              { label: '备份', value: 'backups' },
            ]" class="ml-2"
          />
          <div class="ml-auto flex items-center gap-2">
            <template v-if="tab === 'databases'">
              <FaButton variant="outline" size="sm" @click="openCreateDb">新建数据库</FaButton>
            </template>
            <template v-else-if="tab === 'users'">
              <FaButton variant="outline" size="sm" @click="openCreateUser">新建用户</FaButton>
            </template>
            <template v-else>
              <FaButton variant="outline" size="sm" :loading="backupBusy" @click="doBackup">立即备份</FaButton>
            </template>
            <FaButton variant="ghost" size="icon-sm" title="刷新" @click="refreshPanel">
              <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="panelLoading ? 'animate-spin' : ''" />
            </FaButton>
            <FaButton variant="ghost" size="icon-sm" title="关闭" @click="active = null">
              <FaIcon name="i-lucide:x" class="text-sm" />
            </FaButton>
          </div>
        </div>

        <div v-if="active.type === 'redis' && tab === 'databases'" class="px-4 py-2 text-xs text-muted-foreground">
          Redis 为固定逻辑库 db0-db15，"大小"列为 key 数量；删除库 = FLUSHDB
        </div>
        <div v-if="active.type === 'redis' && tab === 'users'" class="px-4 py-2 text-xs text-muted-foreground">
          Redis 仅支持 default 用户（修改密码即 ACL 重置）
        </div>

        <!-- 数据库表 -->
        <table v-if="tab === 'databases'" class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2">名称</th>
              <th class="px-4 py-2">大小</th>
              <th class="hidden px-4 py-2 sm:table-cell">字符集</th>
              <th class="px-4 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in databases" :key="d.name" class="border-t hover:bg-accent/30">
              <td class="px-4 py-2 font-mono text-[13px]">{{ d.name }}</td>
              <td class="px-4 py-2 text-xs tabular-nums">{{ d.sizeMb.toFixed(2) }} {{ active.type === 'redis' ? 'keys' : 'MB' }}</td>
              <td class="hidden px-4 py-2 text-xs text-muted-foreground sm:table-cell">{{ d.charset || '—' }}</td>
              <td class="px-4 py-2 text-right">
                <FaButton variant="outline" size="sm" @click="dropDb(d.name)">删除</FaButton>
              </td>
            </tr>
            <tr v-if="!databases.length && !panelLoading">
              <td colspan="4" class="px-4 py-8 text-center text-muted-foreground">暂无数据库</td>
            </tr>
          </tbody>
        </table>

        <!-- 用户表 -->
        <table v-else-if="tab === 'users'" class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2">用户名</th>
              <th class="hidden px-4 py-2 md:table-cell">主机</th>
              <th class="hidden px-4 py-2 md:table-cell">说明</th>
              <th class="px-4 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in users" :key="u.name + u.host" class="border-t hover:bg-accent/30">
              <td class="px-4 py-2 font-mono text-[13px]">{{ u.name }}</td>
              <td class="hidden px-4 py-2 font-mono text-xs text-muted-foreground md:table-cell">{{ u.host || '—' }}</td>
              <td class="hidden px-4 py-2 font-mono text-xs text-muted-foreground md:table-cell">{{ u.extra }}</td>
              <td class="px-4 py-2 text-right">
                <FaButton variant="ghost" size="sm" @click="openChangePwd(u)">改密</FaButton>
                <FaButton v-if="u.name !== 'default' && u.name !== 'postgres'" variant="outline" size="sm" @click="dropUser(u)">删除</FaButton>
              </td>
            </tr>
            <tr v-if="!users.length && !panelLoading">
              <td colspan="4" class="px-4 py-8 text-center text-muted-foreground">暂无用户</td>
            </tr>
          </tbody>
        </table>

        <!-- 备份表 -->
        <table v-else class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-4 py-2">文件</th>
              <th class="px-4 py-2">大小</th>
              <th class="hidden px-4 py-2 md:table-cell">时间</th>
              <th class="px-4 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in backups" :key="b.name" class="border-t hover:bg-accent/30">
              <td class="px-4 py-2 font-mono text-[13px]">{{ b.name }}</td>
              <td class="px-4 py-2 text-xs tabular-nums">{{ b.sizeMb.toFixed(2) }} MB</td>
              <td class="hidden px-4 py-2 text-xs tabular-nums text-muted-foreground md:table-cell">{{ new Date(b.modTime).toLocaleString('zh-CN', { hour12: false }) }}</td>
              <td class="px-4 py-2 text-right">
                <FaButton variant="ghost" size="sm" @click="downloadBackup(b)">下载</FaButton>
                <FaButton variant="ghost" size="sm" @click="restoreBackup(b)">恢复</FaButton>
                <FaButton variant="outline" size="sm" @click="removeBackup(b)">删除</FaButton>
              </td>
            </tr>
            <tr v-if="!backups.length && !panelLoading">
              <td colspan="4" class="px-4 py-8 text-center text-muted-foreground">暂无备份</td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <!-- 接入外部实例 -->
    <FaModal v-model="extVisible" title="接入外部数据库实例" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">类型</span>
          <div class="flex flex-1 gap-1.5">
            <button
              v-for="(meta, t) in TYPE_META"
              :key="t"
              type="button"
              class="flex-1 cursor-pointer rounded-md border px-2 py-1.5 text-sm transition-colors"
              :class="extForm.type === t ? 'border-primary bg-primary/10' : 'border-border hover:bg-accent/50'"
              @click="extType(t as string); extForm.type = t"
            >
              {{ meta.label }}
            </button>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">实例名</span>
          <FaInput v-model="extForm.name" placeholder="如 local-mysql" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">主机</span>
          <FaInput v-model="extForm.host" placeholder="127.0.0.1 / 内网 IP / 域名" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">端口</span>
          <FaInput v-model="extForm.port" type="number" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">用户</span>
          <FaInput v-model="extForm.user" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">密码</span>
          <FaInput v-model="extForm.password" type="password" placeholder="必填" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">备注</span>
          <FaInput v-model="extForm.remark" placeholder="选填" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          直连纳管本机或远端已有实例（不做容器安装）；库/用户管理走 SQL 直连；备份需面板所在主机安装对应客户端工具
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="extVisible = false">取消</FaButton>
        <FaButton :loading="extSaving" @click="doCreateExternal">测试并接入</FaButton>
      </template>
    </FaModal>

    <!-- 创建实例 -->
    <FaModal v-model="createVisible" title="创建数据库实例" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">类型</span>
          <div class="flex flex-1 gap-1.5">
            <button
              v-for="(meta, t) in TYPE_META"
              :key="t"
              type="button"
              class="flex-1 cursor-pointer rounded-md border px-2 py-1.5 text-sm transition-colors"
              :class="createForm.type === t ? 'border-primary bg-primary/10' : 'border-border hover:bg-accent/50'"
              @click="createForm.type = t"
            >
              {{ meta.label }}
            </button>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">实例名</span>
          <FaInput v-model="createForm.name" placeholder="小写字母/数字/中划线，如 main-mysql" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">端口</span>
          <FaInput v-model="createForm.port" type="number" placeholder="0 = 默认端口" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">root 密码</span>
          <FaInput v-model="createForm.password" placeholder="留空自动生成" class="flex-1" />
        </div>
        <div class="text-xs text-muted-foreground">
          将在 /opt/ypanel/compose/db-&lt;名称&gt; 创建容器化实例；数据落本地卷，删除实例时可选保留
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">取消</FaButton>
        <FaButton :loading="creating" @click="doCreate">创建并启动</FaButton>
      </template>
    </FaModal>

    <!-- 连接信息 -->
    <FaModal v-model="connVisible" title="连接信息" :destroy-on-close="true">
      <div v-if="conn" class="flex flex-col gap-2 font-mono text-sm">
        <div>主机：<span class="rounded bg-muted px-1.5">{{ conn.host }}</span></div>
        <div>端口：<span class="rounded bg-muted px-1.5">{{ conn.port }}</span></div>
        <div>用户：<span class="rounded bg-muted px-1.5">{{ conn.user || '—' }}</span></div>
        <div>密码：<span class="rounded bg-muted px-1.5">{{ conn.password }}</span></div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="connVisible = false">关闭</FaButton>
      </template>
    </FaModal>

    <!-- 新建数据库 -->
    <FaModal v-model="dbModalVisible" title="新建数据库" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">名称</span>
          <FaInput v-model="dbForm.name" placeholder="字母/数字/下划线" class="flex-1" @keyup.enter="doCreateDb" />
        </div>
        <div v-if="active?.type === 'mysql'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">字符集</span>
          <select v-model="dbForm.charset" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="utf8mb4">utf8mb4</option>
            <option value="utf8mb3">utf8mb3</option>
            <option value="ascii">ascii</option>
          </select>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="dbModalVisible = false">取消</FaButton>
        <FaButton @click="doCreateDb">创建</FaButton>
      </template>
    </FaModal>

    <!-- 新建用户 -->
    <FaModal v-model="userModalVisible" title="新建用户" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">用户名</span>
          <FaInput v-model="userForm.name" placeholder="字母/数字/下划线" class="flex-1" />
        </div>
        <div v-if="active?.type === 'mysql'" class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">主机</span>
          <FaInput v-model="userForm.host" placeholder="%（任意主机）" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">密码</span>
          <FaInput v-model="userForm.password" placeholder="字母/数字/下划线/中划线，8-64 位" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="userModalVisible = false">取消</FaButton>
        <FaButton @click="doCreateUser">创建</FaButton>
      </template>
    </FaModal>

    <!-- 改密 -->
    <FaModal v-model="pwdModalVisible" :title="`修改密码：${pwdForm.name}`" :destroy-on-close="true">
      <FaInput v-model="pwdForm.password" placeholder="新密码（字母/数字/下划线/中划线，8-64 位）" class="w-full" />
      <template #footer>
        <FaButton variant="outline" @click="pwdModalVisible = false">取消</FaButton>
        <FaButton @click="doChangePwd">确认</FaButton>
      </template>
    </FaModal>
  </div>
</template>
