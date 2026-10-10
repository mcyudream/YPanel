<script setup lang="ts">
import type { UserInfo } from '@/api/modules/user'
import apiUser from '@/api/modules/user'
import { useFaModal } from '@fantastic-admin/components'
import { i18n } from '@/locales'

/**
 * YdOwnerDialog — 资源属主分配弹窗（M54-P3）：
 * 公共（0）+ 用户单选；confirm(ownerId) 由调用方调各自域的 setOwner 接口。
 */
defineOptions({ name: 'YdOwnerDialog' })

const props = defineProps<{
  title: string
  currentOwnerId: number
}>()

const visible = defineModel<boolean>({ default: false })

const emit = defineEmits<{
  confirm: [ownerId: number]
}>()

const users = ref<UserInfo[]>([])
const loading = ref(false)
const picked = ref(0)

watch(visible, (v) => {
  if (v) {
    picked.value = props.currentOwnerId
    load()
  }
})

async function load() {
  loading.value = true
  try {
    const res = await apiUser.list(1, 200)
    users.value = res.items
  }
  catch {}
  finally {
    loading.value = false
  }
}

function ownerName(uid: number) {
  const u = users.value.find(x => x.id === uid)
  return u ? `${u.nickname || u.username}（${u.username}）` : `#${uid}`
}

const modal = useFaModal()

function onPick(uid: number) {
  if (uid === props.currentOwnerId) {
    return
  }
  modal.confirm({
    title: props.title,
    content: uid === 0
      ? i18n.global.t('common.owner.toPublicConfirm')
      : i18n.global.t('common.owner.toUserConfirm', { name: ownerName(uid) }),
    onConfirm: async () => {
      emit('confirm', uid)
      visible.value = false
    },
  })
}
</script>

<template>
  <FaModal v-model="visible" :title="title" :destroy-on-close="true">
    <div class="flex flex-col gap-1.5">
      <div v-if="loading" class="py-6 text-center text-sm text-muted-foreground">
        {{ $t('common.loading') }}
      </div>
      <template v-else>
        <button
          class="flex items-center gap-2 rounded-md px-2 py-2 text-left text-sm transition-colors hover:bg-accent/40"
          :class="picked === 0 ? 'bg-primary/10' : ''"
          @click="onPick(0)"
        >
          <FaIcon name="i-lucide:globe" class="text-muted-foreground" />
          {{ $t('common.owner.public') }}
          <span v-if="picked === 0" class="ml-auto text-xs text-primary">{{ $t('common.owner.current') }}</span>
        </button>
        <button
          v-for="u in users"
          :key="u.id"
          class="flex items-center gap-2 rounded-md px-2 py-2 text-left text-sm transition-colors hover:bg-accent/40"
          :class="picked === u.id ? 'bg-primary/10' : ''"
          @click="onPick(u.id)"
        >
          <FaIcon name="i-lucide:user" class="text-muted-foreground" />
          {{ u.nickname || u.username }}
          <span class="font-mono text-xs text-muted-foreground">{{ u.username }}</span>
          <span v-if="picked === u.id" class="ml-auto text-xs text-primary">{{ $t('common.owner.current') }}</span>
        </button>
      </template>
    </div>
    <template #footer>
      <FaButton variant="outline" @click="visible = false">
        {{ $t('common.cancel') }}
      </FaButton>
    </template>
  </FaModal>
</template>
