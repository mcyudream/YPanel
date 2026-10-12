<script setup lang="ts">
// M68 S3 对象工作台：桶切换 + 前缀目录浏览 + 上传/下载/删除 + 建桶/删桶。
import { ref, watch, onMounted, computed } from 'vue'
const toast = useFaToast()
import apiDBA from '@/api/modules/dbadmin'
import dbApi from '@/api/modules/database'
import YdSelect from '@/components/YdSelect/index.vue'

const props = defineProps<{ instanceId: number, db: string }>()

const buckets = ref<string[]>([])
const bucket = ref(props.db || '')
const prefix = ref('')
const objects = ref<{ key: string, size: number, lastModified?: string, isDir: boolean }[]>([])
const loading = ref(false)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

const crumbs = computed(() => {
  const parts = prefix.value.split('/').filter(Boolean)
  const list: { name: string, prefix: string }[] = [{ name: bucket.value || '/', prefix: '' }]
  let acc = ''
  for (const p of parts) {
    acc += `${p}/`
    list.push({ name: p, prefix: acc })
  }
  return list
})

async function loadBuckets() {
  try {
    const dbs = await apiDBA.databases(props.instanceId)
    buckets.value = dbs.map(d => d.name)
    if (!bucket.value || !buckets.value.includes(bucket.value))
      bucket.value = buckets.value[0] || ''
  }
  catch (e: any) {
    toast.error(e?.message || '加载桶清单失败')
  }
}

async function loadObjects() {
  if (!bucket.value)
    return
  loading.value = true
  try {
    const r = await apiDBA.s3Objects(props.instanceId, bucket.value, prefix.value)
    objects.value = r.objects || []
  }
  catch (e: any) {
    toast.error(e?.message || '加载对象失败')
  }
  finally {
    loading.value = false
  }
}

onMounted(loadBuckets)
watch(bucket, () => {
  prefix.value = ''
  loadObjects()
})
watch(() => [props.instanceId, props.db], () => {
  bucket.value = props.db || ''
  loadBuckets()
})

function enterDir(obj: { key: string }) {
  prefix.value = obj.key
}

function fmtSize(n: number) {
  if (n < 1024)
    return `${n} B`
  if (n < 1024 * 1024)
    return `${(n / 1024).toFixed(1)} KB`
  if (n < 1024 * 1024 * 1024)
    return `${(n / 1024 / 1024).toFixed(1)} MB`
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`
}

async function doDownload(obj: { key: string }) {
  try {
    await apiDBA.s3ObjectDownload(props.instanceId, bucket.value, obj.key)
  }
  catch (e: any) {
    toast.error(e?.message || '下载失败')
  }
}

async function doDelete(obj: { key: string }) {
  try {
    await apiDBA.s3ObjectDelete(props.instanceId, bucket.value, obj.key)
    toast.success('已删除')
    await loadObjects()
  }
  catch (e: any) {
    toast.error(e?.message || '删除失败')
  }
}

function pickUpload() {
  fileInput.value?.click()
}

async function onFileChosen(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file)
    return
  uploading.value = true
  try {
    await apiDBA.s3ObjectUpload(props.instanceId, bucket.value, prefix.value + file.name, file)
    toast.success(`已上传 ${file.name}`)
    await loadObjects()
  }
  catch (e: any) {
    toast.error(e?.message || '上传失败')
  }
  finally {
    uploading.value = false
  }
}

// 建桶
const createBucketOpen = ref(false)
const newBucket = ref('')

async function doCreateBucket() {
  if (!newBucket.value.trim())
    return
  try {
    await dbApi.createDatabase(props.instanceId, newBucket.value.trim())
    toast.success('桶已创建')
    createBucketOpen.value = false
    newBucket.value = ''
    await loadBuckets()
  }
  catch (e: any) {
    toast.error(e?.message || '创建失败')
  }
}

async function doDropBucket() {
  try {
    await dbApi.dropDatabase(props.instanceId, bucket.value)
    toast.success('桶已删除')
    bucket.value = ''
    await loadBuckets()
  }
  catch (e: any) {
    toast.error(e?.message || '删除失败（桶需为空）')
  }
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-3">
    <div class="flex flex-wrap items-center gap-2">
      <YdSelect
        v-model="bucket"
        class="w-56"
        :options="buckets.map(b => ({ label: b, value: b }))"
        placeholder="选择桶"
      />
      <FaButton variant="outline" size="sm" @click="createBucketOpen = true">
        新建桶
      </FaButton>
      <FaButton v-if="bucket" variant="ghost" size="sm" class="text-destructive!" @click="doDropBucket">
        删除空桶
      </FaButton>
      <span class="flex-1" />
      <input ref="fileInput" type="file" class="hidden" @change="onFileChosen">
      <FaButton size="sm" :loading="uploading" :disabled="!bucket" @click="pickUpload">
        上传对象
      </FaButton>
    </div>

    <!-- 面包屑 -->
    <div class="flex items-center gap-1 text-xs text-muted-foreground">
      <template v-for="(c, i) in crumbs" :key="c.prefix">
        <span v-if="i" class="opacity-50">/</span>
        <button
          type="button"
          class="rounded px-1 py-0.5 hover:bg-accent/50 hover:text-foreground"
          :class="{ 'font-medium text-foreground': i === crumbs.length - 1 }"
          @click="prefix = c.prefix; loadObjects()"
        >
          {{ c.name }}
        </button>
      </template>
    </div>

    <FaCard :padding="false" class="min-h-0 flex-1 overflow-hidden">
      <div class="max-h-full overflow-auto">
        <table class="w-full text-sm">
          <thead class="sticky top-0 bg-muted/60 text-left text-xs text-muted-foreground">
            <tr>
              <th class="px-3 py-2 font-medium">
                名称
              </th>
              <th class="px-3 py-2 font-medium">
                大小
              </th>
              <th class="hidden px-3 py-2 font-medium md:table-cell">
                修改时间
              </th>
              <th class="px-3 py-2 text-right font-medium">
                操作
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="4" class="px-3 py-10 text-center text-muted-foreground">
                加载中…
              </td>
            </tr>
            <tr v-else-if="!objects.length">
              <td colspan="4" class="px-3 py-10 text-center text-muted-foreground">
                {{ bucket ? '目录为空' : '选择或新建桶开始浏览' }}
              </td>
            </tr>
            <tr v-for="obj in objects" v-else :key="obj.key" class="border-t border-border/60 hover:bg-muted/30">
              <td class="px-3 py-2">
                <button
                  v-if="obj.isDir" type="button"
                  class="font-mono text-xs text-primary hover:underline"
                  @click="enterDir(obj)"
                >
                  {{ obj.key.slice(prefix.length) }}
                </button>
                <span v-else class="font-mono text-xs">{{ obj.key.slice(prefix.length) }}</span>
              </td>
              <td class="px-3 py-2 text-xs text-muted-foreground">
                {{ obj.isDir ? '—' : fmtSize(obj.size) }}
              </td>
              <td class="hidden px-3 py-2 text-xs text-muted-foreground md:table-cell">
                {{ obj.lastModified ? new Date(obj.lastModified).toLocaleString() : '' }}
              </td>
              <td class="px-3 py-2 text-right">
                <div v-if="!obj.isDir" class="inline-flex gap-1">
                  <FaButton variant="ghost" size="sm" @click="doDownload(obj)">
                    下载
                  </FaButton>
                  <FaButton variant="ghost" size="sm" class="text-destructive!" @click="doDelete(obj)">
                    删除
                  </FaButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FaCard>

    <FaModal v-model:open="createBucketOpen" title="新建桶" @ok="doCreateBucket" @cancel="createBucketOpen = false">
      <label class="flex flex-col gap-1 text-sm">
        <span>桶名（3-63 位小写字母/数字/中划线/点）<span class="text-destructive">*</span></span>
        <FaInput v-model="newBucket" placeholder="如 my-app-bucket" />
      </label>
    </FaModal>
  </div>
</template>
