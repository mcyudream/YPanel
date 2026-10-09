<script setup lang="ts">
import type { DockerNetwork } from '@/api/modules/dockerext'
import { dockerExtApi } from '@/api/modules/dockerext'
import { i18n } from '@/locales'

// 网络管理（自 Docker 管理页迁入，M23）。
const toast = useFaToast()

const networks = ref<DockerNetwork[]>([])
const loading = ref(false)

// ---- 分页 ----
const page = ref(1)
const size = ref(20)
const paged = computed(() => networks.value.slice((page.value - 1) * size.value, page.value * size.value))

async function load() {
  loading.value = true
  try {
    networks.value = await dockerExtApi.networks()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.networks.loadFailed'), { description: e?.message })
  }
  finally {
    loading.value = false
  }
}

onMounted(load)

const visible = ref(false)
const form = ref({ name: '', driver: 'bridge' })

async function doCreate() {
  try {
    await dockerExtApi.createNetwork(form.value.name, form.value.driver)
    toast.success(i18n.global.t('container.networks.created'))
    visible.value = false
    await load()
  }
  catch (e: any) {
    toast.error(i18n.global.t('container.common.createFailed'), { description: e?.message })
  }
}

function remove(n: DockerNetwork) {
  const modal = useFaModal()
  modal.confirm({
    title: i18n.global.t('container.networks.deleteTitle'),
    content: i18n.global.t('container.networks.deleteConfirm', { name: n.name }),
    onConfirm: async () => {
      try {
        await dockerExtApi.removeNetwork(n.name)
        toast.success(i18n.global.t('container.common.deletedDone'))
        await load()
      }
      catch (e: any) {
        toast.error(i18n.global.t('container.common.deleteFailed'), { description: e?.message })
      }
    },
  })
}
</script>

<template>
  <div>
    <FaPageMain>
  <div>
		<YdDockerNodeSelect />
    <div class="mb-3 flex items-center gap-2">
      <FaButton class="ml-auto" size="sm" @click="visible = true">
        <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('container.networks.create') }}
      </FaButton>
      <FaButton variant="outline" size="icon-sm" :title="$t('common.refresh')" @click="load()">
        <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
      </FaButton>
    </div>

    <div class="overflow-x-auto rounded-lg border">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
          <tr>
            <th class="px-3 py-2">{{ $t('common.name') }}</th>
            <th class="px-3 py-2">{{ $t('container.common.driver') }}</th>
            <th class="hidden px-3 py-2 md:table-cell">{{ $t('container.networks.subnet') }}</th>
            <th class="hidden px-3 py-2 lg:table-cell">ID</th>
            <th class="px-3 py-2 text-right">{{ $t('common.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !networks.length">
            <td colspan="5" class="px-3 py-10 text-center text-muted-foreground">
              {{ $t('common.loading') }}
            </td>
          </tr>
          <tr v-for="n in paged" :key="n.id" class="border-t transition-colors hover:bg-accent/30">
            <td class="px-3 py-1.5">
              <div class="flex items-center gap-2">
                <FaIcon name="i-lucide:network" class="text-sm text-primary opacity-60" />
                <span class="font-mono text-[13px]">{{ n.name }}</span>
              </div>
            </td>
            <td class="px-3 py-1.5 text-xs">
              {{ n.driver }}
            </td>
            <td class="hidden px-3 py-1.5 font-mono text-xs text-muted-foreground md:table-cell">
              {{ n.subnet || '—' }}
            </td>
            <td class="hidden px-3 py-1.5 font-mono text-xs text-muted-foreground lg:table-cell">
              {{ n.id.slice(0, 12) }}
            </td>
            <td class="px-3 py-1.5 text-right">
              <FaButton
                variant="outline" size="sm"
                :disabled="n.name === 'bridge' || n.name === 'host' || n.name === 'none'"
                @click="remove(n)"
              >
                {{ $t('common.delete') }}
              </FaButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <FaPagination v-model:page="page" v-model:size="size" :total="networks.length" class="mt-3" />

    <FaModal v-model="visible" :title="$t('container.networks.create')" :destroy-on-close="true">
      <div class="flex flex-col gap-3">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('common.name') }}</span>
          <FaInput v-model="form.name" :placeholder="$t('container.networks.namePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-sm text-muted-foreground">{{ $t('container.common.driver') }}</span>
          <select v-model="form.driver" class="h-9 flex-1 rounded-md border bg-background px-2 text-sm outline-none">
            <option value="bridge">bridge</option>
            <option value="host">host</option>
            <option value="overlay">overlay</option>
          </select>
        </div>
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
