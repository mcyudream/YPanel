// FTP 账号管理卡片（M69）——追加为 ftp.vue 的独立组件文件
<script setup lang="ts">
import { ref, computed } from 'vue'
import { useNodeModule } from '@/store/modules/node'
import { toast } from '@/utils/toast'
import api from '@/api/modules/ftp'
import type { FtpAccount } from '@/api/modules/ftp'

const emit = defineEmits<{ (e: 'change'): void }>()

const accounts = ref<FtpAccount[]>([])
const loading = ref(false)
const keyword = ref('')

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw)
    return accounts.value
  return accounts.value.filter(a => `${a.name} ${a.directory} ${a.remark || ''}`.toLowerCase().includes(kw))
})

async function load() {
  loading.value = true
  try {
    accounts.value = (await api.accounts.list()) || []
  }
  catch (e: any) {
    toast.error(e?.message || '加载失败')
  }
  finally {
    loading.value = false
  }
}

load()

// ---- 创建/编辑 ----
const editOpen = ref(false)
const editing = ref<FtpAccount | null>(null)
const form = ref({ name: '', directory: '', password: '', readOnly: false, enabled: true, remark: '' })
const saving = ref(false)

function openCreate() {
  editing.value = null
  form.value = { name: '', directory: '', password: '', readOnly: false, enabled: true, remark: '' }
  editOpen.value = true
}

function openEdit(row: FtpAccount) {
  editing.value = row
  form.value = { name: row.name, directory: row.directory, password: '', readOnly: row.readOnly, enabled: row.enabled, remark: row.remark || '' }
  editOpen.value = true
}

async function doSave() {
  if (!form.value.name.trim() || !form.value.directory.trim()) {
    toast.warning('账号名与根目录必填')
    return
  }
  if (!editing.value && form.value.password.length < 8) {
    toast.warning('密码至少 8 位')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await api.accounts.update(editing.value.id, {
        directory: form.value.directory,
        password: form.value.password || undefined,
        readOnly: form.value.readOnly,
        enabled: form.value.enabled,
        remark: form.value.remark,
      })
      toast.success('已保存并重载 vsftpd')
    }
    else {
      await api.accounts.create({ name: form.value.name, directory: form.value.directory, password: form.value.password, readOnly: form.value.readOnly, remark: form.value.remark })
      toast.success('账号已创建（vsftpd 已重载）')
    }
    editOpen.value = false
    load()
    emit('change')
  }
  catch (e: any) {
    toast.error(e?.message || '保存失败')
  }
  finally {
    saving.value = false
  }
}

// ---- 测试登录（一次性密码） ----
const testOpen = ref(false)
const testTarget = ref<FtpAccount | null>(null)
const testPassword = ref('')
const testing = ref(false)
const testResult = ref('')

function openTest(row: FtpAccount) {
  testTarget.value = row
  testPassword.value = ''
  testResult.value = ''
  testOpen.value = true
}

async function doTest() {
  if (!testTarget.value || !testPassword.value) {
    toast.warning('输入当前密码')
    return
  }
  testing.value = true
  try {
    const r = await api.accounts.test(testTarget.value.id, testPassword.value)
    const ok = (r.data as any)?.ok
    testResult.value = ok ? `登录成功\n${(r.data as any)?.listing || '(根目录为空)'}` : `登录失败（凭据或 vsftpd 状态异常）`
    toast[ok ? 'success' : 'error'](ok ? 'FTP 登录成功' : 'FTP 登录失败')
  }
  catch (e: any) {
    toast.error(e?.message || '测试失败')
  }
  finally {
    testing.value = false
  }
}

// ---- 删除 ----
const deleteOpen = ref(false)
const deleteTarget = ref<FtpAccount | null>(null)
const purgeDir = ref(false)
const deleting = ref(false)

function openDelete(row: FtpAccount) {
  deleteTarget.value = row
  purgeDir.value = false
  deleteOpen.value = true
}

async function doDelete() {
  if (!deleteTarget.value)
    return
  deleting.value = true
  try {
    await api.accounts.remove(deleteTarget.value.id, purgeDir.value)
    toast.success('已删除（vsftpd 已重载）')
    deleteOpen.value = false
    load()
    emit('change')
  }
  catch (e: any) {
    toast.error(e?.message || '删除失败')
  }
  finally {
    deleting.value = false
  }
}

// ---- 启停 ----
async function toggleEnabled(row: FtpAccount) {
  try {
    await api.accounts.update(row.id, { enabled: !row.enabled })
    toast.success(row.enabled ? '已禁用（白名单摘除，登录即拒）' : '已启用')
    load()
  }
  catch (e: any) {
    toast.error(e?.message || '操作失败')
  }
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex flex-wrap items-center gap-2">
      <FaInput v-model="keyword" placeholder="搜索账号/目录/备注" class="w-56!" />
      <span class="ml-auto text-xs text-muted-foreground">{{ $t('common.total', { n: filtered.length }) }}</span>
      <FaButton size="sm" @click="openCreate()">
        <FaIcon name="i-lucide:user-plus" class="mr-1" /> 新建账号
      </FaButton>
    </div>

    <FaCard :padding="false" class="overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2 font-medium">
                账号
              </th>
              <th class="px-3 py-2 font-medium">
                根目录
              </th>
              <th class="px-3 py-2 font-medium">
                权限
              </th>
              <th class="px-3 py-2 font-medium">
                状态
              </th>
              <th class="hidden px-3 py-2 font-medium md:table-cell">
                备注
              </th>
              <th class="px-3 py-2 text-right font-medium">
                操作
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !accounts.length">
              <td colspan="6" class="px-3 py-8 text-center text-muted-foreground">
                加载中…
              </td>
            </tr>
            <tr v-else-if="!filtered.length">
              <td colspan="6" class="px-3 py-8 text-center text-muted-foreground">
                暂无账号——新建账号会创建同名的 nologin 系统用户并加入 vsftpd 白名单
              </td>
            </tr>
            <tr v-for="row in filtered" :key="row.id" class="border-t border-border/60 hover:bg-muted/30">
              <td class="px-3 py-2">
                <div class="font-mono text-xs font-medium">{{ row.name }}</div>
                <div v-if="row.systemUserExists === false" class="text-[10px] text-destructive">
                  系统用户缺失
                </div>
              </td>
              <td class="max-w-[240px] truncate px-3 py-2 font-mono text-xs" :title="row.directory">
                {{ row.directory }}
              </td>
              <td class="px-3 py-2">
                <span class="rounded px-1.5 py-0.5 text-xs" :class="row.readOnly ? 'bg-amber-500/10 text-amber-600' : 'bg-emerald-500/10 text-emerald-600'">
                  {{ row.readOnly ? '只读' : '读写' }}
                </span>
              </td>
              <td class="px-3 py-2">
                <span class="rounded px-1.5 py-0.5 text-xs" :class="row.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'">
                  {{ row.enabled ? '启用' : '禁用' }}
                </span>
              </td>
              <td class="hidden max-w-[200px] truncate px-3 py-2 text-xs text-muted-foreground md:table-cell">
                {{ row.remark }}
              </td>
              <td class="px-3 py-2 text-right">
                <div class="inline-flex gap-1">
                  <FaButton variant="ghost" size="sm" @click="openEdit(row)">
                    编辑
                  </FaButton>
                  <FaButton variant="ghost" size="sm" @click="openTest(row)">
                    测试
                  </FaButton>
                  <FaButton variant="ghost" size="sm" @click="toggleEnabled(row)">
                    {{ row.enabled ? '禁用' : '启用' }}
                  </FaButton>
                  <FaButton variant="ghost" size="sm" class="text-destructive!" @click="openDelete(row)">
                    删除
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaCard>

    <!-- 创建/编辑 -->
    <FaModal v-model="editOpen" :title="editing ? `编辑账号 ${editing.name}` : '新建 FTP 账号'" :confirm-loading="saving" @ok="doSave" @cancel="editOpen = false">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="flex flex-col gap-1 text-sm">
          <span>账号名 <span class="text-destructive">*</span></span>
          <FaInput v-model="form.name" :disabled="!!editing" placeholder="小写字母开头，3-32 位" />
        </label>
        <label class="flex flex-col gap-1 text-sm">
          <span>密码 {{ editing ? '（留空=不改）' : '' }} <span v-if="!editing" class="text-destructive">*</span></span>
          <FaInput v-model="form.password" type="password" :placeholder="editing ? '留空保持不变' : '8-64 位'" />
        </label>
        <label class="flex flex-col gap-1 text-sm sm:col-span-2">
          <span>根目录（FTP 登录后锁定的主目录）<span class="text-destructive">*</span></span>
          <FaInput v-model="form.directory" placeholder="/www/ftp/share" />
        </label>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="form.readOnly" type="checkbox" class="accent-[var(--primary)]">
          只读（禁止上传/删除/改名）
        </label>
        <label v-if="editing" class="flex items-center gap-2 text-sm">
          <input v-model="form.enabled" type="checkbox" class="accent-[var(--primary)]">
          启用（禁用即从白名单摘除，登录被拒）
        </label>
        <label class="flex flex-col gap-1 text-sm sm:col-span-2">
          <span>备注</span>
          <FaInput v-model="form.remark" />
        </label>
      </div>
      <p class="mt-3 text-xs text-muted-foreground">
        实现为同名的 nologin 系统用户，根目录即 home 并 chroot；账号写入 vsftpd 白名单（user_list），非白名单系统用户无法走 FTP。
      </p>
    </FaModal>

    <!-- 测试登录 -->
    <FaModal v-model="testOpen" :title="`FTP 登录测试：${testTarget?.name}`" :confirm-loading="testing" @ok="doTest" @cancel="testOpen = false">
      <label class="flex flex-col gap-1 text-sm">
        <span>当前密码</span>
        <FaInput v-model="testPassword" type="password" />
      </label>
      <pre v-if="testResult" class="mt-3 max-h-40 overflow-auto whitespace-pre-wrap rounded-md bg-muted/50 p-2 font-mono text-xs">{{ testResult }}</pre>
    </FaModal>

    <!-- 删除 -->
    <FaModal v-model="deleteOpen" title="删除 FTP 账号" :confirm-loading="deleting" @ok="doDelete" @cancel="deleteOpen = false">
      <p class="text-sm">
        将删除系统用户 <code class="rounded bg-muted px-1 font-mono">{{ deleteTarget?.name }}</code> 并从 vsftpd 白名单摘除。
      </p>
      <label class="mt-3 flex items-center gap-2 text-sm text-destructive">
        <input v-model="purgeDir" type="checkbox" class="accent-[var(--primary)]">
        同时删除根目录及其全部数据（不勾选则保留目录）
      </label>
    </FaModal>
  </div>
</template>
