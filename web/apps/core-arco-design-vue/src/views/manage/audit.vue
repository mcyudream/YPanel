<script setup lang="ts">
import type { LoginLogItem } from '@/api/modules/user'
import apiUser from '@/api/modules/user'

defineOptions({
  name: 'ManageAudit',
})

const logs = ref<LoginLogItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await apiUser.loginLogs(page.value, pageSize)
    logs.value = res.items
    total.value = res.total
  }
  finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="scroll-text" :size="24" />
          <span>{{ $t('manage.audit.title') }}</span>
        </div>
      </template>
      <template #description>
        <span>{{ $t('manage.audit.desc') }}</span>
      </template>
      <FaButton variant="outline" size="icon-sm" :title="$t('common.refresh')" @click="load()">
        <FaIcon name="i-lucide:refresh-cw" class="text-sm" :class="loading ? 'animate-spin' : ''" />
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div class="overflow-x-auto rounded-lg border">
        <table class="w-full min-w-160 text-sm">
          <thead class="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2">{{ $t('common.time') }}</th>
              <th class="px-3 py-2">{{ $t('manage.audit.username') }}</th>
              <th class="px-3 py-2">{{ $t('manage.audit.result') }}</th>
              <th class="hidden px-3 py-2 md:table-cell">IP</th>
              <th class="hidden px-3 py-2 lg:table-cell">User-Agent</th>
              <th class="px-3 py-2">{{ $t('manage.audit.message') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !logs.length">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('common.loading') }}
              </td>
            </tr>
            <tr v-else-if="!logs.length">
              <td colspan="6" class="px-3 py-10 text-center text-muted-foreground">
                {{ $t('manage.audit.empty') }}
              </td>
            </tr>
            <tr v-for="l in logs" :key="l.id" class="border-t transition-colors hover:bg-accent/30">
              <td class="px-3 py-2 text-xs tabular-nums text-muted-foreground">
                {{ new Date(l.createdAt).toLocaleString('zh-CN', { hour12: false }) }}
              </td>
              <td class="px-3 py-2 font-mono text-[13px]">
                {{ l.username }}
              </td>
              <td class="px-3 py-2">
                <span
                  class="rounded-full px-2 py-0.5 text-xs"
                  :class="l.success ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'"
                >
                  {{ l.success ? $t('common.success') : $t('common.failed') }}
                </span>
              </td>
              <td class="hidden px-3 py-2 font-mono text-xs text-muted-foreground md:table-cell">
                {{ l.ip }}
              </td>
              <td class="hidden max-w-64 truncate px-3 py-2 text-xs text-muted-foreground lg:table-cell" :title="l.userAgent">
                {{ l.userAgent }}
              </td>
              <td class="px-3 py-2 text-xs text-muted-foreground">
                {{ l.message }}
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
          <span>{{ $t('manage.audit.pageInfo', { page, pages: Math.max(1, Math.ceil(total / pageSize)), total }) }}</span>
          <FaButton variant="outline" size="sm" :disabled="page >= Math.ceil(total / pageSize)" @click="page++; load()">
            {{ $t('manage.audit.next') }}
          </FaButton>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
