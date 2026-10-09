<script setup lang="ts">
import type { NodeItem } from '@/api/modules/node'
import type { CatalogGroup, RoleItem } from '@/api/modules/role'
import apiNode from '@/api/modules/node'
import apiRole from '@/api/modules/role'
import { useFaModal } from '@fantastic-admin/components'
import { i18n } from '@/locales'

defineOptions({
  name: 'ManageRole',
})

const roles = ref<RoleItem[]>([])
const catalog = ref<CatalogGroup[]>([])
const nodes = ref<NodeItem[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    roles.value = await apiRole.list()
    if (!catalog.value.length) {
      catalog.value = await apiRole.catalog()
    }
    if (!nodes.value.length) {
      nodes.value = await apiNode.list().catch(() => [])
    }
  }
  finally {
    loading.value = false
  }
}

// 权限点标题：i18n 键 = rbac.perm.<key 的 : . 替换为 _>，无词条回落原始 key
function permTitle(key: string) {
  const k = `rbac.perm.${key.replace(/[:.]/g, '_')}`
  return i18n.global.te(k) ? i18n.global.t(k) : key
}

function groupTitle(key: string) {
  return i18n.global.t(`rbac.group.${key}`)
}

// ---- 创建 / 编辑 / 复制 / 删除 ----
const editVisible = ref(false)
const editTarget = ref<RoleItem | null>(null) // null = 创建/复制
const editForm = ref({ key: '', name: '', remark: '', perms: [] as string[], scopeAllNodes: true, nodeIds: [] as string[], dataScope: 'all' as 'all' | 'assigned' })

function openCreate() {
  editTarget.value = null
  editForm.value = { key: '', name: '', remark: '', perms: [], scopeAllNodes: true, nodeIds: [], dataScope: 'all' }
  editVisible.value = true
}

function openCopy(r: RoleItem) {
  editTarget.value = null
  editForm.value = { key: `${r.key}-copy`, name: `${r.name} 副本`, remark: r.remark, perms: [...r.perms], scopeAllNodes: r.scopeAllNodes, nodeIds: [...(r.nodeIds ?? [])], dataScope: (r.dataScope as 'all' | 'assigned') ?? 'all' }
  editVisible.value = true
}

function openEdit(r: RoleItem) {
  editTarget.value = r
  editForm.value = { key: r.key, name: r.name, remark: r.remark, perms: [...r.perms], scopeAllNodes: r.scopeAllNodes, nodeIds: [...(r.nodeIds ?? [])], dataScope: (r.dataScope as 'all' | 'assigned') ?? 'all' }
  editVisible.value = true
}

async function doSave() {
  if (!editForm.value.name) {
    useFaToast().warning(i18n.global.t('manage.role.fillRequired'))
    return
  }
  try {
    const scope = {
      scopeAllNodes: editForm.value.scopeAllNodes,
      nodeIds: editForm.value.scopeAllNodes ? [] : editForm.value.nodeIds,
      dataScope: editForm.value.dataScope,
    }
    if (editTarget.value) {
      await apiRole.update(editTarget.value.id, { name: editForm.value.name, remark: editForm.value.remark, perms: editForm.value.perms, ...scope })
    }
    else {
      if (!editForm.value.key) {
        useFaToast().warning(i18n.global.t('manage.role.fillRequired'))
        return
      }
      await apiRole.create({ key: editForm.value.key, name: editForm.value.name, remark: editForm.value.remark, perms: editForm.value.perms, ...scope })
    }
    useFaToast().success(i18n.global.t('manage.saved'))
    editVisible.value = false
    load()
  }
  catch (e: any) {
    useFaToast().error(i18n.global.t('manage.saveFailed'), { description: e?.message })
  }
}

function doDelete(r: RoleItem) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('manage.role.deleteTitle'),
    content: i18n.global.t('manage.role.deleteConfirm', { name: r.name }),
    onConfirm: async () => {
      try {
        await apiRole.remove(r.id)
        useFaToast().success(i18n.global.t('manage.deleted'))
        load()
      }
      catch (e: any) {
        useFaToast().error(i18n.global.t('manage.deleteFailed'), { description: e?.message })
      }
    },
  })
}

// 分组勾选
function groupPerms(g: CatalogGroup) {
  return g.perms.map(p => p.key)
}

function isGroupAll(g: CatalogGroup) {
  return groupPerms(g).every(k => editForm.value.perms.includes(k))
}

function toggleGroup(g: CatalogGroup, val: boolean) {
  const keys = groupPerms(g)
  if (val) {
    const s = new Set([...editForm.value.perms, ...keys])
    editForm.value.perms = [...s]
  }
  else {
    editForm.value.perms = editForm.value.perms.filter(k => !keys.includes(k))
  }
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="key-round" :size="24" />
          <span>{{ $t('manage.role.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('manage.role.desc') }}</span>
      </template>
      <FaButton size="sm" @click="openCreate">
        <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('manage.role.createRole') }}
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div class="overflow-hidden rounded-lg border">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('manage.role.name') }}</th>
              <th class="px-3 py-2">{{ $t('manage.role.key') }}</th>
              <th class="hidden px-3 py-2 md:table-cell">{{ $t('manage.role.permCount') }}</th>
              <th class="hidden px-3 py-2 lg:table-cell">{{ $t('manage.role.remark') }}</th>
              <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !roles.length">
              <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('common.loading') }}
              </td>
            </tr>
            <tr v-for="r in roles" :key="r.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2">
                <div class="flex items-center gap-2">
                  {{ r.name }}
                  <span v-if="r.builtin" class="rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
                    {{ $t('manage.role.builtin') }}
                  </span>
                </div>
              </td>
              <td class="px-3 py-2 font-mono text-[13px]">
                {{ r.key }}
              </td>
              <td class="hidden px-3 py-2 tabular-nums md:table-cell">
                {{ r.perms.includes('*') ? $t('manage.role.allPerms') : r.perms.length }}
              </td>
              <td class="hidden px-3 py-2 text-xs text-muted-foreground lg:table-cell">
                {{ r.remark || '—' }}
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center justify-end gap-1">
                  <FaButton v-if="!r.builtin" variant="outline" size="sm" @click="openEdit(r)">
                    {{ $t('common.edit') }}
                  </FaButton>
                  <FaButton variant="outline" size="sm" @click="openCopy(r)">
                    {{ $t('manage.role.copy') }}
                  </FaButton>
                  <FaButton v-if="!r.builtin" variant="outline" size="sm" @click="doDelete(r)">
                    {{ $t('common.delete') }}
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaPageMain>

    <!-- 创建/编辑 -->
    <FaModal v-model="editVisible" :title="editTarget ? $t('manage.role.editTitle', { name: editTarget.name }) : $t('manage.role.createRole')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div v-if="!editTarget" class="flex items-center gap-3">
          <span class="w-16 shrink-0 text-sm text-muted-foreground">{{ $t('manage.role.key') }}</span>
          <FaInput v-model="editForm.key" :placeholder="$t('manage.role.keyPlaceholder')" class="w-full!" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 shrink-0 text-sm text-muted-foreground">{{ $t('manage.role.name') }}</span>
          <FaInput v-model="editForm.name" class="w-full!" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 shrink-0 text-sm text-muted-foreground">{{ $t('manage.role.remark') }}</span>
          <FaInput v-model="editForm.remark" class="w-full!" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 shrink-0 text-sm text-muted-foreground">{{ $t('manage.role.nodeScope') }}</span>
          <div class="flex flex-1 flex-col gap-1.5">
            <label class="flex cursor-pointer items-center gap-2 text-sm">
              <input v-model="editForm.scopeAllNodes" type="checkbox" class="accent-[var(--primary)]">
              {{ $t('manage.role.allNodes') }}
            </label>
            <div v-if="!editForm.scopeAllNodes" class="flex flex-wrap gap-x-4 gap-y-1.5 pl-6">
              <span v-if="!nodes.length" class="text-xs text-muted-foreground">{{ $t('manage.role.noNodes') }}</span>
              <label v-for="n in nodes" :key="n.id" class="flex cursor-pointer items-center gap-1.5 text-xs text-muted-foreground">
                <input v-model="editForm.nodeIds" type="checkbox" :value="n.id" class="accent-[var(--primary)]">
                {{ n.name }}（{{ n.id === 'local' ? 'local' : `#${n.id}` }}）
              </label>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-16 shrink-0 text-sm text-muted-foreground">{{ $t('manage.role.dataScope') }}</span>
          <YdSelect
            v-model="editForm.dataScope"
            :options="[
              { label: $t('manage.role.dataScopeAll'), value: 'all' },
              { label: $t('manage.role.dataScopeAssigned'), value: 'assigned' },
            ]"
            size="default"
            button-class="w-full"
            class="min-w-0 flex-1"
          />
        </div>
        <div class="max-h-[40vh] overflow-y-auto rounded-md border p-3">
          <div v-for="g in catalog" :key="g.key" class="mb-3 last:mb-0">
            <label class="flex cursor-pointer items-center gap-2 text-sm font-medium">
              <input
                type="checkbox"
                class="accent-[var(--primary)]"
                :checked="isGroupAll(g)"
                @change="toggleGroup(g, ($event.target as HTMLInputElement).checked)"
              >
              {{ groupTitle(g.key) }}
            </label>
            <div class="mt-1.5 flex flex-wrap gap-x-4 gap-y-1.5 pl-6">
              <label v-for="p in g.perms" :key="p.key" class="flex cursor-pointer items-center gap-1.5 text-xs text-muted-foreground">
                <input v-model="editForm.perms" type="checkbox" :value="p.key" class="accent-[var(--primary)]">
                {{ permTitle(p.key) }}
              </label>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="editVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton @click="doSave">
          {{ $t('common.save') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
