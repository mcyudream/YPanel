<script setup lang="ts">
import type { AIMemory } from '@/api/modules/ai'
import { memoryApi } from '@/api/modules/ai'
import { useAppAccountStore } from '@/store/modules/app/account'

defineOptions({ name: 'AiMemory' })
const items = ref<AIMemory[]>([])
const search = ref('')
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const visible = ref(false)
const form = ref({ id: 0, content: '', shared: false })
const error = ref('')
const admin = computed(() => useAppAccountStore().permissions.includes('*'))
let timer: ReturnType<typeof setTimeout>
let generation = 0
async function load(more = false) {
  const request = ++generation
  loading.value = true
  try {
    const result = await memoryApi.list(search.value, more ? page.value + 1 : 1)
    if (request !== generation) {
      return
    }
    items.value = more ? [...items.value, ...result.items] : result.items
    total.value = result.total
    page.value = more ? page.value + 1 : 1
  }
  catch (err: any) {
    error.value = String(err?.message || err)
  }
  finally {
    loading.value = false
  }
}
async function save() {
  try {
    await memoryApi.save(form.value)
    visible.value = false
    await load()
  }
  catch (err: any) {
    error.value = String(err?.message || err)
  }
}
async function remove(id: number) {
  await memoryApi.remove(id)
  await load()
}
watch(search, () => {
  clearTimeout(timer)
  timer = setTimeout(load, 250)
})
onBeforeUnmount(() => clearTimeout(timer))
onMounted(() => load())
</script>

<template>
  <FaPageMain>
    <div class="space-y-4">
      <div class="flex gap-3 items-center">
        <FaInput v-model="search" :placeholder="$t('aiAssistant.search')" class="flex-1 max-w-sm" /><span class="text-xs text-muted-foreground">{{ $t('common.total', { n: total }) }}</span><FaButton
          class="ml-auto" @click="form = { id: 0, content: '', shared: false };
                                  visible = true"
        >
          <FaIcon name="i-lucide:plus" />
        </FaButton>
      </div>
      <p class="text-xs text-muted-foreground">
        {{ $t('aiAssistant.scopeHelp') }}
      </p>
      <div v-if="error" class="text-sm text-red-500">
        {{ error }}
      </div>
      <div class="border rounded-xl divide-y">
        <div v-for="m in items" :key="m.id" class="p-4 flex gap-3 items-start">
          <FaIcon :name="m.shared ? 'i-lucide:users' : 'i-lucide:brain'" class="text-primary mt-1 shrink-0" /><p class="text-sm leading-6 flex-1 whitespace-pre-wrap">
            {{ m.content }}
          </p><FaButton
            size="icon-sm" variant="ghost" :title="$t('aiAssistant.edit')" @click="form = { id: m.id, content: m.content, shared: m.shared };
                                                                                   visible = true"
          >
            <FaIcon name="i-lucide:pencil" />
          </FaButton><FaButton size="icon-sm" variant="ghost" :title="$t('common.delete')" @click="remove(m.id)">
            <FaIcon name="i-lucide:trash-2" />
          </FaButton>
        </div><div v-if="!items.length" class="text-sm text-muted-foreground p-8 text-center">
          {{ loading ? $t('common.loading') : $t('ai.memory.empty') }}
        </div>
      </div>
      <FaButton v-if="items.length < total" variant="outline" :loading="loading" @click="load(true)">
        {{ $t('aiAssistant.loadMore') }}
      </FaButton>
    </div>
    <FaModal v-model="visible" :title="$t('aiAssistant.memory')" :footer="false">
      <textarea v-model="form.content" rows="6" class="p-3 border rounded-xl bg-background w-full" /><label v-if="admin" class="text-sm my-3 flex items-center justify-between"><span>{{ $t('aiAssistant.shared') }}</span><FaSwitch v-model="form.shared" /></label><div class="mt-3 flex justify-end">
        <FaButton @click="save">
          {{ $t('common.save') }}
        </FaButton>
      </div>
    </FaModal>
  </FaPageMain>
</template>
