<script setup lang="ts">
import type { SiteItem, SiteExtConfig } from '@/api/modules/site'
import { extApi } from '@/api/modules/site'

const props = defineProps<{ site: SiteItem }>()
const emit = defineEmits<{ changed: [] }>()
const toast = useFaToast()

const conf = ref<SiteExtConfig | null>(null)
const locs = ref<{ comment: string, content: string }[]>([])
const saving = ref(false)

async function load() {
  try {
    conf.value = await extApi.getExt(props.site.id)
    locs.value = JSON.parse(JSON.stringify(conf.value.customLocations || []))
  }
  catch (e: any) {
    toast.error('读取自定义 location 失败', { description: e?.message })
  }
}

function addLoc() {
  locs.value.push({ comment: '', content: 'location /path {\n    return 200 "ok";\n}' })
}

function removeLoc(i: number) {
  locs.value.splice(i, 1)
}

async function save() {
  saving.value = true
  try {
    const next = { ...(conf.value as SiteExtConfig), customLocations: locs.value }
    await extApi.updateExt(props.site.id, next)
    conf.value = next
    toast.success('自定义 location 已保存并重载 nginx')
    emit('changed')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    saving.value = false
  }
}

const dirty = computed(() => conf.value ? JSON.stringify(locs.value) !== JSON.stringify(conf.value.customLocations || []) : false)

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="rounded-lg border p-4">
      <div class="mb-3 flex items-center justify-between">
        <div>
          <span class="text-sm font-medium">自定义 location</span>
          <p class="mt-0.5 text-xs text-muted-foreground">注入 server 块尾部的 location 指令，保存时经 nginx -t 校验（失败自动回滚）</p>
        </div>
        <FaButton variant="outline" size="sm" @click="addLoc">
          <FaIcon name="i-lucide:plus" class="mr-1" /> 加 location
        </FaButton>
      </div>
      <div v-if="!locs.length" class="py-6 text-center text-sm text-muted-foreground">
        暂无自定义 location
      </div>
      <div v-for="(l, i) in locs" :key="i" class="mb-4 space-y-2 rounded-md border p-3">
        <div class="flex items-center gap-2">
          <FaInput v-model="l.comment" placeholder="备注（可选）" class="flex-1" />
          <FaButton variant="ghost" size="icon-sm" class="text-red-500!" title="删除" @click="removeLoc(i)">
            <FaIcon name="i-lucide:trash-2" class="text-sm" />
          </FaButton>
        </div>
        <textarea
          v-model="l.content"
          class="h-28 w-full resize-y rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary"
          placeholder="location /healthz { return 200 'ok'; }"
          spellcheck="false"
        />
      </div>
    </div>

    <div class="flex justify-end">
      <FaButton :loading="saving" :disabled="!dirty" @click="save">
        保存并生效
      </FaButton>
    </div>
  </div>
</template>
