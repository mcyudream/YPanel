<script setup lang="ts">
import { computed } from 'vue'

// 容器/应用品牌图标：商店应用用 iconUrl，其余按镜像/项目名关键词映射品牌 logo。
// 注意：brandRules 中的 i-logos: class 必须以字面量出现在本文件，UnoCSS presetIcons 才会构建期内联。
const props = withDefaults(defineProps<{
  image?: string
  name?: string
  size?: number
}>(), { size: 20 })

const brandRules: [string, string][] = [
  ['redis', 'i-logos:redis'],
  ['nginx', 'i-logos:nginx'],
  ['mysql', 'i-logos:mysql'],
  ['mariadb', 'i-logos:mariadb'],
  ['postgres', 'i-logos:postgresql'],
  ['mongo', 'i-logos:mongodb'],
  ['grafana', 'i-logos:grafana'],
  ['prometheus', 'i-logos:prometheus'],
  ['rabbitmq', 'i-logos:rabbitmq'],
  ['memcached', 'i-logos:memcached'],
  ['node', 'i-logos:nodejs'],
  ['python', 'i-logos:python'],
  ['golang', 'i-logos:go'],
  ['openjdk', 'i-logos:java'],
  ['java', 'i-logos:java'],
  ['tomcat', 'i-logos:tomcat'],
  ['wordpress', 'i-logos:wordpress'],
  ['gitlab', 'i-logos:gitlab'],
  ['jenkins', 'i-logos:jenkins'],
  ['php', 'i-logos:php'],
  ['consul', 'i-logos:consul'],
  ['etcd', 'i-logos:etcd'],
  ['vault', 'i-logos:vault'],
  ['influxdb', 'i-logos:influxdb'],
  ['kafka', 'i-logos:kafka'],
  ['ruby', 'i-logos:ruby'],
  ['rust', 'i-logos:rust'],
  ['perl', 'i-logos:perl'],
  ['deno', 'i-logos:deno'],
  ['bun', 'i-logos:bun'],
  ['vite', 'i-logos:vite'],
  ['flask', 'i-logos:flask'],
  ['django', 'i-logos:django'],
  ['laravel', 'i-logos:laravel'],
  ['spring', 'i-logos:spring'],
  ['docker', 'i-logos:docker-icon'],
  ['portainer', 'i-logos:docker-icon'],
  ['dpanel', 'i-logos:docker-icon'],
  ['1panel', 'i-logos:docker-icon'],
]

const resolved = computed(() => {
  if (props.image) {
    return { type: 'img', value: props.image } as const
  }
  const s = (props.name || '').toLowerCase()
  for (const [kw, icon] of brandRules) {
    if (s.includes(kw)) {
      return { type: 'brand', value: icon } as const
    }
  }
  return { type: 'fallback' } as const
})
</script>

<template>
  <span
    class="inline-flex shrink-0 items-center justify-center"
    :style="{ width: `${size}px`, height: `${size}px`, fontSize: `${size}px` }"
  >
    <img v-if="resolved.type === 'img'" :src="resolved.value" alt="" class="h-full w-full rounded-sm object-contain">
    <FaIcon v-else-if="resolved.type === 'brand'" :name="resolved.value" class="size-inherit!" />
    <YdMorphIcon v-else name="container" :size="size" class="opacity-60" />
  </span>
</template>
