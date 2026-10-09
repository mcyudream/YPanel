<script setup lang="ts">
import type { DockerVolume } from '@/api/modules/dockerext'
import { dockerExtApi } from '@/api/modules/dockerext'
import { i18n } from '@/locales'

// 卷管理（自 Docker 管理页迁入，M23）。
const toast = useFaToast()

const volumes = ref<DockerVolume[]>([])
const loading = ref(false)

// ---- 分页 ----
const page = ref(1)
const size = ref(20)
const paged = computed(() => volumes.value.slice((page.value - 1) * size.value, page.value * size.value))

async function load() {
  loading.value = true
  try {
    volumes.value = await dockerExtApi.volumes()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.volumes.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

onMounted(load)

const visible = ref(false)
const volName = ref('')

async function doCreate() {
  try {
    await dockerExtApi.createVolume(volName.value)
    toast.success(i18n.global.t('container.volumes.created'))
    visible.value = false
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.createFailed'), { description: e?.message })
  }
}

function remove(v: DockerVolume) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.volumes.deleteTitle'),
    content: i18n.global.t('container.volumes.deleteConfirm', { name: v.name }),
    onConfirm: async () => {
      try {
        await dockerExtApi.removeVolume(v.name)
        toast.success(i18n.global.t('container.common.deletedDone'))
        await load()
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.common.deleteFailed'), { description: e?.message })
      }
    },
  })
}

async function prune() {
  try {
    const out = await dockerExtApi.pruneVolumes()
    toast.success(out)
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.pruneFailed'), { description: e?.message })
  }
}
</script>

<template>
  <div>
    <FaPageMain>
  <div>
    <div class="mb-3 flex items-center gap-2">
      <FaButton variant="outline" size="sm" @click="prune">
        {{ $t('container.volumes.pruneUnused') }}
      </FaButton>
      <FaButton size="sm" @click="visible = true">
        <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('container.volumes.create') }}
      </FaButton>
      <FaButton variant="outline" size="icon-sm" :title="$t('common.refresh')" class="ml-auto" @click="load()">
        <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
      </FaButton>
    </div>

    <div class="overflow-x-auto rounded-lg border">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
          <tr>
            <th class="px-3 py-2">{{ $t('container.volumes.nameCol') }}</th>
            <th class="px-3 py-2">{{ $t('container.common.driver') }}</th>
            <th class="hidden px-3 py-2 md:table-cell">{{ $t('container.volumes.mountpoint') }}</th>
            <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !volumes.length">
            <td colspan="4" class="px-3 py-10 text-center text-muted-foreground">
              {{ $t('common.loading') }}
            </td>
          </tr>
          <tr v-for="v in paged" :key="v.name" class="border-t transition-colors hover:bg-accent/30">
            <td class="px-3 py-1.5">
              <div class="flex items-center gap-2">
                <FaIcon name="i-lucide:hard-drive" class="text-sm text-primary opacity-60" />
                <span class="font-mono text-[13px]">{{ v.name }}</span>
              </div>
            </td>
            <td class="px-3 py-1.5 text-xs text-muted-foreground">
              {{ v.driver }}
            </td>
            <td class="hidden max-w-96 truncate px-3 py-1.5 font-mono text-xs text-muted-foreground md:table-cell" :title="v.mountpoint">
              {{ v.mountpoint }}
            </td>
            <td class="px-3 py-1.5 text-right">
              <FaButton variant="outline" size="sm" @click="remove(v)">
                {{ $t('common.delete') }}
              </FaButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <FaPagination v-model:page="page" v-model:size="size" :total="volumes.length" class="mt-3" />

    <FaModal v-model="visible" :title="$t('container.volumes.create')" :destroy-on-close="true">
      <div class="flex items-center gap-3">
        <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('container.volumes.nameCol') }}</span>
        <FaInput v-model="volName" :placeholder="$t('container.volumes.namePlaceholder')" class="flex-1" @keyup.enter="doCreate" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="visible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton @click="doCreate">
          {{ $t('common.create') }}
        </FaButton>
      </template>
    </FaModal>
  </div>
    </FaPageMain>
  </div>
</template>
