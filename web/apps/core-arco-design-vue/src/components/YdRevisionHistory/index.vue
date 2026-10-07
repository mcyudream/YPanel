<script setup lang="ts">
import Panel from './panel.vue'

// 受管配置版本历史（M23 服务端快照）：自含 FaModal 弹窗；bare=true 时仅渲染面板（工作台 HistoryDialog 内嵌用）。
// 注意不可做成「外部弹窗 + 传入 visible 驱动加载」：FaModal 插槽内容在关闭态即挂载，defineModel 收不到
// 打开时的更新（reka 传递链），面板会停留在关闭态的加载结果——故面板组件随弹窗打开而重建（onMounted 加载）。
const props = withDefaults(defineProps<{
  node?: string
  path: string
  bare?: boolean
}>(), { bare: false })

const emit = defineEmits<{
  restored: []
}>()

const visible = defineModel<boolean>({ default: false })
</script>

<template>
  <FaModal v-if="!bare" v-model="visible" title="版本历史" class="max-w-3xl!" :destroy-on-close="true">
    <Panel :node="props.node" :path="props.path" @restored="emit('restored')" />
    <template #footer>
      <FaButton variant="outline" @click="visible = false">
        关闭
      </FaButton>
    </template>
  </FaModal>
  <Panel v-else :node="props.node" :path="props.path" @restored="emit('restored')" />
</template>
