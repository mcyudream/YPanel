<script setup lang="ts">
import type { UserInfo } from '@/api/modules/user'
import apiUser from '@/api/modules/user'
import { useFaModal } from '@fantastic-admin/components'

defineOptions({
  name: 'ManageUser',
})

const users = ref<UserInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await apiUser.list(page.value, pageSize)
    users.value = res.items
    total.value = res.total
  }
  finally {
    loading.value = false
  }
}

// 创建
const createVisible = ref(false)
const createForm = ref({ username: '', password: '', nickname: '', role: 'user' as 'admin' | 'user' })

async function doCreate() {
  if (!createForm.value.username || !createForm.value.password) {
    useFaToast().warning('请填写用户名与密码')
    return
  }
  try {
    await apiUser.create(createForm.value)
    useFaToast().success('用户已创建')
    createVisible.value = false
    createForm.value = { username: '', password: '', nickname: '', role: 'user' }
    load()
  }
  catch (e: any) {
    useFaToast().error('创建失败', { description: e?.message })
  }
}

// 编辑
const editVisible = ref(false)
const editTarget = ref<UserInfo | null>(null)
const editForm = ref<{ nickname: string, role: 'admin' | 'user', status: 0 | 1, password: string }>({ nickname: '', role: 'user', status: 1, password: '' })

function openEdit(u: UserInfo) {
  editTarget.value = u
  editForm.value = { nickname: u.nickname, role: u.role, status: 1, password: '' }
  // status 不在列表返回中，默认按启用处理（后端 update 为零值跳过）
  editVisible.value = true
}

async function doEdit() {
  if (!editTarget.value) {
    return
  }
  const data: Record<string, unknown> = {
    nickname: editForm.value.nickname,
    role: editForm.value.role,
  }
  if (editForm.value.password) {
    data.password = editForm.value.password
  }
  try {
    await apiUser.update(editTarget.value.id, data as any)
    useFaToast().success('已保存')
    editVisible.value = false
    load()
  }
  catch (e: any) {
    useFaToast().error('保存失败', { description: e?.message })
  }
}

// 删除
function doDelete(u: UserInfo) {
  const modal = useFaModal()
  modal.confirm({
    title: '删除用户',
    content: `确认删除用户 ${u.username}？`,
    onConfirm: async () => {
      try {
        await apiUser.remove(u.id)
        useFaToast().success('已删除')
        load()
      }
      catch (e: any) {
        useFaToast().error('删除失败', { description: e?.message })
      }
    },
  })
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="users-round" :size="24" />
          <span>用户管理</span>
        </div>
      </template>
      <template #description>
        <span>面板账号与角色（admin / user）</span>
      </template>
      <FaButton size="sm" @click="createVisible = true">
        <FaIcon name="i-lucide:user-plus" class="mr-1" /> 新建用户
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">用户名</th>
              <th class="px-3 py-2">昵称</th>
              <th class="px-3 py-2">角色</th>
              <th class="hidden px-3 py-2 md:table-cell">最近登录</th>
              <th class="px-3 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !users.length">
              <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
                加载中…
              </td>
            </tr>
            <tr v-for="u in users" :key="u.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2 font-mono text-[13px]">
                {{ u.username }}
              </td>
              <td class="px-3 py-2">
                {{ u.nickname || '—' }}
              </td>
              <td class="px-3 py-2">
                <span
                  class="rounded-full px-2 py-0.5 text-xs"
                  :class="u.role === 'admin' ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'"
                >
                  {{ u.role }}
                </span>
              </td>
              <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground md:table-cell">
                {{ u.lastLoginAt ? new Date(u.lastLoginAt).toLocaleString('zh-CN', { hour12: false }) : '从未登录' }}
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton variant="outline" size="sm" @click="openEdit(u)">
                    编辑
                  </FaButton>
                  <FaButton variant="outline" size="sm" @click="doDelete(u)">
                    删除
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-3 flex justify-end">
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <FaButton variant="outline" size="sm" :disabled="page <= 1" @click="page--; load()">
            上一页
          </FaButton>
          <span>{{ page }} / {{ Math.max(1, Math.ceil(total / pageSize)) }}（共 {{ total }} 人）</span>
          <FaButton variant="outline" size="sm" :disabled="page >= Math.ceil(total / pageSize)" @click="page++; load()">
            下一页
          </FaButton>
        </div>
      </div>
    </FaPageMain>

    <!-- 创建 -->
    <FaModal v-model="createVisible" title="新建用户" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">用户名</span>
          <FaInput v-model="createForm.username" placeholder="3-32 位" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">密码</span>
          <FaInput v-model="createForm.password" type="password" placeholder="6-64 位" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">昵称</span>
          <FaInput v-model="createForm.nickname" placeholder="选填" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">角色</span>
          <select v-model="createForm.role" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="user">
              user（普通用户）
            </option>
            <option value="admin">
              admin（管理员）
            </option>
          </select>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">
          取消
        </FaButton>
        <FaButton @click="doCreate">
          创建
        </FaButton>
      </template>
    </FaModal>

    <!-- 编辑 -->
    <FaModal v-model="editVisible" :title="`编辑：${editTarget?.username || ''}`" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">昵称</span>
          <FaInput v-model="editForm.nickname" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">角色</span>
          <select v-model="editForm.role" class="h-9 flex-1 rounded-md border border-input bg-background px-2 text-sm outline-none">
            <option value="user">
              user
            </option>
            <option value="admin">
              admin
            </option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">新密码</span>
          <FaInput v-model="editForm.password" type="password" placeholder="留空不修改" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="editVisible = false">
          取消
        </FaButton>
        <FaButton @click="doEdit">
          保存
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
