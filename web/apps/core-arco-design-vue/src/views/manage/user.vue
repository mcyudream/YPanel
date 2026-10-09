<script setup lang="ts">
import type { RoleItem } from '@/api/modules/role'
import type { UserInfo } from '@/api/modules/user'
import apiRole from '@/api/modules/role'
import apiUser from '@/api/modules/user'
import { useFaModal } from '@fantastic-admin/components'
import { i18n } from '@/locales'

defineOptions({
  name: 'ManageUser',
})

const users = ref<UserInfo[]>([])
const roles = ref<RoleItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)

async function loadRoles() {
  try {
    roles.value = await apiRole.list()
  }
  catch {}
}

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
const createForm = ref({ username: '', password: '', nickname: '', roleId: 0 })

async function doCreate() {
  if (!createForm.value.username || !createForm.value.password) {
    useFaToast().warning(i18n.global.t('manage.user.fillRequired'))
    return
  }
  try {
    await apiUser.create({ ...createForm.value, roleId: createForm.value.roleId || defaultRoleId() })
    useFaToast().success(i18n.global.t('manage.user.created'))
    createVisible.value = false
    createForm.value = { username: '', password: '', nickname: '', roleId: 0 }
    load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('manage.createFailed'), { description: e?.message })
  }
}

// 编辑
const editVisible = ref(false)
const editTarget = ref<UserInfo | null>(null)
const editForm = ref<{ nickname: string, roleId: number, status: 0 | 1, password: string }>({ nickname: '', roleId: 0, status: 1, password: '' })

// 默认角色：operator（与后端兜底一致）
function defaultRoleId() {
  return roles.value.find(r => r.key === 'operator')?.id ?? roles.value[0]?.id ?? 0
}

const roleOptions = computed(() => roles.value.map(r => ({ label: `${r.name}（${r.key}）`, value: r.id })))

function openCreate() {
  createForm.value.roleId = defaultRoleId() // 预选运维用户，与既有「默认普通角色」习惯一致
  createVisible.value = true
}

function openEdit(u: UserInfo) {
  editTarget.value = u
  editForm.value = { nickname: u.nickname, roleId: u.roleId || defaultRoleId(), status: 1, password: '' }
  // status 不在列表返回中，默认按启用处理（后端 update 为零值跳过）
  editVisible.value = true
}

async function doEdit() {
  if (!editTarget.value) {
    return
  }
  const data: Record<string, unknown> = {
    nickname: editForm.value.nickname,
    roleId: editForm.value.roleId,
  }
  if (editForm.value.password) {
    data.password = editForm.value.password
  }
  try {
    await apiUser.update(editTarget.value.id, data as any)
    useFaToast().success(i18n.global.t('manage.saved'))
    editVisible.value = false
    load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('manage.saveFailed'), { description: e?.message })
  }
}

// 删除
function doDelete(u: UserInfo) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('manage.user.deleteTitle'),
    content: i18n.global.t('manage.user.deleteConfirm', { name: u.username }),
    onConfirm: async () => {
      try {
        await apiUser.remove(u.id)
        useFaToast().success(i18n.global.t('manage.deleted'))
        load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('manage.deleteFailed'), { description: e?.message })
      }
    },
  })
}

function roleName(key: string) {
  return roles.value.find(r => r.key === key)?.name ?? key
}

onMounted(() => {
  loadRoles()
  load()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="users-round" :size="24" />
          <span>{{ $t('manage.user.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('manage.user.desc') }}</span>
      </template>
      <FaButton size="sm" @click="openCreate">
        <FaIcon name="i-lucide:user-plus" class="mr-1" /> {{ $t('manage.user.createUser') }}
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('manage.user.username') }}</th>
              <th class="px-3 py-2">{{ $t('manage.user.nickname') }}</th>
              <th class="px-3 py-2">{{ $t('manage.user.role') }}</th>
              <th class="hidden px-3 py-2 md:table-cell">{{ $t('manage.user.lastLogin') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !users.length">
              <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('common.loading') }}
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
                  :class="u.roleKey === 'super-admin' ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'"
                >
                  {{ roleName(u.roleKey) }}
                </span>
              </td>
              <td class="hidden px-3 py-2 text-xs tabular-nums text-muted-foreground md:table-cell">
                {{ u.lastLoginAt ? new Date(u.lastLoginAt).toLocaleString('zh-CN', { hour12: false }) : $t('manage.user.neverLoggedIn') }}
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton variant="outline" size="sm" @click="openEdit(u)">
                    {{ $t('common.edit') }}
                  </FaButton>
                  <FaButton variant="outline" size="sm" @click="doDelete(u)">
                    {{ $t('common.delete') }}
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
            {{ $t('manage.audit.prev') }}
          </FaButton>
          <span>{{ $t('manage.user.pageInfo', { page, pages: Math.max(1, Math.ceil(total / pageSize)), total }) }}</span>
          <FaButton variant="outline" size="sm" :disabled="page >= Math.ceil(total / pageSize)" @click="page++; load()">
            {{ $t('manage.audit.next') }}
          </FaButton>
        </div>
      </div>
    </FaPageMain>

    <!-- 创建 -->
    <FaModal v-model="createVisible" :title="$t('manage.user.createUser')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">{{ $t('manage.user.username') }}</span>
          <FaInput v-model="createForm.username" :placeholder="$t('manage.user.usernamePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">{{ $t('manage.user.password') }}</span>
          <FaInput v-model="createForm.password" type="password" :placeholder="$t('manage.user.passwordPlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">{{ $t('manage.user.nickname') }}</span>
          <FaInput v-model="createForm.nickname" :placeholder="$t('manage.user.nicknamePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 shrink-0 text-sm text-muted-foreground">{{ $t('manage.user.role') }}</span>
          <YdSelect v-model="createForm.roleId" :options="roleOptions" size="default" button-class="w-full" class="min-w-0 flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="createVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton @click="doCreate">
          {{ $t('common.create') }}
        </FaButton>
      </template>
    </FaModal>

    <!-- 编辑 -->
    <FaModal v-model="editVisible" :title="$t('manage.user.editTitle', { name: editTarget?.username || '' })" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">{{ $t('manage.user.nickname') }}</span>
          <FaInput v-model="editForm.nickname" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 shrink-0 text-sm text-muted-foreground">{{ $t('manage.user.role') }}</span>
          <YdSelect v-model="editForm.roleId" :options="roleOptions" size="default" button-class="w-full" class="min-w-0 flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 text-sm text-muted-foreground">{{ $t('manage.user.newPassword') }}</span>
          <FaInput v-model="editForm.password" type="password" :placeholder="$t('manage.user.newPasswordPlaceholder')" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="editVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton @click="doEdit">
          {{ $t('common.save') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
