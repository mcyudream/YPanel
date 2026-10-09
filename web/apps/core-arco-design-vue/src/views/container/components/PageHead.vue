<script setup lang="ts">
// 容器模块各二级页共享页头（tab 拆分到二级菜单后每页自带）。
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <FaPageHeader>
    <template #title>
      <div class="flex items-center gap-2.5">
        <FaIcon name="i-tabler:brand-docker" class="text-xl" />
        <span>{{ title }}</span>
      </div>
    </template>
    <template #description>
      <span>{{ description }}</span>
    </template>
  </FaPageHeader>
</template>
