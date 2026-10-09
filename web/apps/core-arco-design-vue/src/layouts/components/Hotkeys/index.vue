<script setup lang="ts">
import { hotkeyBindings } from '@/hotkeys'
import eventBus from '@/utils/eventBus'
import { tr } from '@/locales'

defineOptions({
  name: 'HotkeysIntro',
})

const appSettingsStore = useAppSettingsStore()
const hotkeyContext = {
  settings: appSettingsStore.settings,
}

const helpGroupOrder: Record<string, number> = {
  global: 10,
  nav: 20,
  tabbar: 30,
}

const isShow = ref(false)

const helpGroups = computed(() => {
  const groups = new Map<string, {
    id: string
    title: string
    order: number
    items: {
      id: string
      title: string
      order: number
      displayKeys: string[]
    }[]
  }>()

  hotkeyBindings.forEach((binding) => {
    const help = binding.help
    if (!help) {
      return
    }
    if (help.visible && !help.visible(hotkeyContext)) {
      return
    }

    const group = groups.get(help.group) ?? {
      id: help.group,
      title: help.groupTitleKey
        ? tr(`layout.hotkeys.${help.groupTitleKey}`, help.groupTitleKey)
        : tr(`layout.hotkeys.group.${help.group}`, help.group),
      order: helpGroupOrder[help.group] ?? 100,
      items: [],
    }

    group.items.push({
      id: binding.id,
      title: tr(`layout.hotkeys.${help.titleKey}`, help.titleKey),
      order: help.order ?? 0,
      displayKeys: appSettingsStore.os === 'mac' && help.displayKeys.mac
        ? help.displayKeys.mac
        : help.displayKeys.default,
    })
    groups.set(help.group, group)
  })

  return Array.from(groups.values())
    .map(group => ({
      ...group,
      items: [...group.items].sort((a, b) => a.order - b.order),
    }))
    .sort((a, b) => a.order - b.order)
})

onMounted(() => {
  eventBus.on('global-hotkeys-intro-toggle', () => {
    isShow.value = !isShow.value
  })
})

onUnmounted(() => {
  eventBus.off('global-hotkeys-intro-toggle')
})
</script>

<template>
  <FaModal v-model="isShow" :title="$t('layout.hotkeys.title')" :footer="false">
    <div class="px-4">
      <div class="gap-4 grid sm-grid-cols-2">
        <div v-for="group in helpGroups" :key="group.id">
          <h2 class="text-lg font-bold m-0">
            {{ group.title }}
          </h2>
          <ul class="text-sm ps-2 pt-2 list-none">
            <li v-for="item in group.items" :key="item.id" class="py-1 flex-baseline gap-2">
              <FaKbdGroup>
                <FaKbd v-for="key in item.displayKeys" :key="key">
                  {{ key }}
                </FaKbd>
              </FaKbdGroup>
              {{ item.title }}
            </li>
          </ul>
        </div>
      </div>
    </div>
  </FaModal>
</template>
