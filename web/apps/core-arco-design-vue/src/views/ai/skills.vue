<script setup lang="ts">
// 技能包（智能/技能）：SKILL.md 格式技能管理，启用的技能注入对话上下文。
import { i18n } from '@/locales'
import type { AISkill } from '@/api/modules/ai'
import { skillApi } from '@/api/modules/ai'

const zipInput = ref<HTMLInputElement>()
const zipUploading = ref(false)

async function onZipChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) {
    return
  }
  zipUploading.value = true
  try {
    const res = await skillApi.uploadZip(file)
    toast.success(i18n.global.t('ai.skills.zipImported', { name: (res.data as any)?.name || file.name }))
    await loadSkills()
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.skills.importFailed'), { description: e?.message })
  }
  finally {
    zipUploading.value = false
  }
}

defineOptions({
  name: 'aiSkills',
})

const toast = useFaToast()

const skills = ref<AISkill[]>([])
const skillVisible = ref(false)
const skillSaving = ref(false)
const skillForm = ref<AISkill>({ name: '', description: '', body: '', enabled: false })

async function loadSkills() {
  try {
    skills.value = await skillApi.list()
  }
  catch {}
}

function openSkill(k?: AISkill) {
  skillForm.value = k ? { ...k } : {
    name: '',
    description: '',
    body: '---\nname: my-skill\ndescription: 技能描述\n---\n\n技能正文（提示词与执行步骤）',
    enabled: false,
  }
  skillVisible.value = true
}

async function saveSkill() {
  skillSaving.value = true
  try {
    await skillApi.save({
      name: skillForm.value.name,
      description: skillForm.value.description,
      body: skillForm.value.body,
    })
    if (skillForm.value.enabled) {
      await skillApi.setEnabled(skillForm.value.name, true)
    }
    skillVisible.value = false
    toast.success(i18n.global.t('ai.skills.saved'))
    await loadSkills()
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.skills.saveFailed'), { description: e?.message })
  }
  finally {
    skillSaving.value = false
  }
}

async function toggleSkill(k: AISkill) {
  try {
    await skillApi.setEnabled(k.name, !k.enabled)
    await loadSkills()
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.skills.operationFailed'), { description: e?.message })
  }
}

async function removeSkill(k: AISkill) {
  try {
    await skillApi.remove(k.name)
    await loadSkills()
    toast.success(i18n.global.t('ai.skills.deleted'))
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.skills.deleteFailed'), { description: e?.message })
  }
}

onMounted(loadSkills)
onActivated(loadSkills)
</script>

<template>
  <div>
    <FaPageMain>
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <p class="text-xs text-muted-foreground">
            {{ $t('ai.skills.scpHint') }}
          </p>
          <div class="flex items-center gap-2">
            <FaButton size="sm" :loading="zipUploading" @click="zipInput?.click()">
              <FaIcon name="i-lucide:upload" class="mr-1" /> {{ $t('ai.skills.uploadZip') }}
            </FaButton>
            <FaButton size="sm" @click="openSkill()">
              <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('ai.skills.add') }}
            </FaButton>
          </div>
        </div>
        <div v-if="!skills.length" class="rounded-lg border p-8 text-center text-sm text-muted-foreground">
          {{ $t('ai.skills.empty') }}
        </div>
        <div v-for="k in skills" :key="k.name" class="rounded-lg border p-4">
          <div class="flex items-center justify-between">
            <div class="min-w-0">
              <div class="flex items-center gap-2 text-sm font-medium">
                {{ k.name }}
                <span
                  class="rounded-full px-2 py-0.5 text-xs"
                  :class="k.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
                >
                  {{ k.enabled ? $t('ai.skills.enabled') : $t('ai.skills.notEnabled') }}
                </span>
              </div>
              <div class="mt-1 line-clamp-2 text-xs text-muted-foreground">
                {{ k.description || k.body.slice(0, 100) }}
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <label class="flex cursor-pointer items-center gap-1.5 text-xs">
                <input type="checkbox" :checked="k.enabled" @change="toggleSkill(k)"> {{ $t('common.enabled') }}
              </label>
              <FaButton variant="ghost" size="sm" @click="openSkill(k)">
                {{ $t('common.edit') }}
              </FaButton>
              <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeSkill(k)">
                {{ $t('common.delete') }}
              </FaButton>
            </div>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 技能编辑 -->
    <FaModal v-model="skillVisible" :title="skillForm.name ? $t('ai.skills.editTitle', { name: skillForm.name }) : $t('ai.skills.add')" class="max-w-3xl!" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-muted-foreground">{{ $t('ai.skills.nameLabel') }}</span>
          <FaInput v-model="skillForm.name" :placeholder="$t('ai.skills.namePlaceholder')" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-muted-foreground">{{ $t('ai.skills.descLabel') }}</span>
          <FaInput v-model="skillForm.description" :placeholder="$t('ai.skills.descPlaceholder')" class="flex-1" />
        </div>
        <textarea
          v-model="skillForm.body"
          rows="10"
          class="w-full resize-y rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary"
          :placeholder="$t('ai.skills.bodyPlaceholder')"
        />
        <label class="flex cursor-pointer items-center gap-2 text-xs">
          <input v-model="skillForm.enabled" type="checkbox"> {{ $t('ai.skills.enableAfterSave') }}
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="skillVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="skillSaving" @click="saveSkill">
          {{ $t('common.save') }}
        </FaButton>
      </template>
    </FaModal>

    <input ref="zipInput" type="file" accept=".zip" class="hidden" @change="onZipChange">
  </div>
</template>
