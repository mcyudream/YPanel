<script setup lang="ts">
// YdAiQuestionPanel：ask_user 挂起时替换 Sender 输入框的提问面板（ZCode 式）。
// 多题上下切换导航、每题选项卡片（单选/多选）+ 补充填写，统一提交；提交后由 store 恢复。
import { computed, ref } from 'vue'
import { useAiQuestionStore } from '@/store/modules/aiQuestion'

const store = useAiQuestionStore()

const idx = ref(0)
const forms = ref<Record<number, { selected: string[], text: string }>>({})

const total = computed(() => store.questions.length)
const current = computed(() => store.questions[idx.value] || store.questions[0])

function form(i: number) {
  if (!forms.value[i]) {
    forms.value[i] = { selected: [], text: '' }
  }
  return forms.value[i]
}

function toggle(label: string, multi: boolean) {
  const f = form(idx.value)
  if (multi) {
    const i = f.selected.indexOf(label)
    if (i >= 0) {
      f.selected.splice(i, 1)
    }
    else {
      f.selected.push(label)
    }
  }
  else {
    f.selected = [label]
  }
}

const answered = computed(() => {
  const flags: boolean[] = []
  for (let i = 0; i < total.value; i++) {
    const f = form(i)
    flags.push(f.selected.length > 0 || f.text.trim().length > 0)
  }
  return flags
})

const canSubmit = computed(() => answered.value.every(Boolean))

async function submit() {
  if (!canSubmit.value) {
    return
  }
  const answers = store.questions.map((q, i) => ({
    header: q.header,
    question: q.question,
    selected: [...form(i).selected],
    text: form(i).text,
  }))
  await store.submit(answers)
}
</script>

<template>
  <div class="rounded-xl border border-primary/40 bg-background p-3">
    <!-- 顶部：进度导航 -->
    <div class="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
      <FaIcon name="i-lucide:help-circle" class="text-primary" />
      <span>{{ $t('components.ydAiChat.needYourInput') }}</span>
      <template v-if="total > 1">
        <span class="ml-auto flex items-center gap-1">
          <button
            v-for="i in total" :key="i" type="button"
            class="size-5 cursor-pointer rounded-full text-[10px]"
            :class="idx === i - 1 ? 'bg-primary text-primary-foreground' : answered[i - 1] ? 'bg-emerald-500/20 text-emerald-600' : 'bg-muted text-muted-foreground'"
            @click="idx = i - 1"
          >
            {{ i }}
          </button>
        </span>
      </template>
    </div>
    <!-- 当前题 -->
    <div v-if="current" class="space-y-2">
      <div class="text-sm font-medium">
        <span v-if="current.header" class="mr-1.5 rounded bg-primary/10 px-1.5 py-0.5 text-[11px] text-primary">{{ current.header }}</span>
        {{ current.question }}
      </div>
      <div class="space-y-1.5">
        <button
          v-for="o in current.options" :key="o.label"
          type="button"
          class="flex w-full cursor-pointer items-start gap-2 rounded-lg border px-3 py-2 text-left text-sm transition-colors hover:bg-accent/40"
          :class="form(idx).selected.includes(o.label) ? 'border-primary/60 bg-primary/10' : 'border-border/60'"
          @click="toggle(o.label, !!current.multiSelect)"
        >
          <FaIcon
            :name="form(idx).selected.includes(o.label) ? (current.multiSelect ? 'i-lucide:check-square' : 'i-lucide:check-circle-2') : (current.multiSelect ? 'i-lucide:square' : 'i-lucide:circle')"
            class="mt-0.5 shrink-0 text-sm"
            :class="form(idx).selected.includes(o.label) ? 'text-primary' : 'text-muted-foreground/40'"
          />
          <span class="min-w-0">
            <span class="font-medium">{{ o.label }}</span>
            <span v-if="o.description" class="ml-1.5 text-xs text-muted-foreground">{{ o.description }}</span>
          </span>
        </button>
      </div>
      <FaInput v-model="form(idx).text" :placeholder="$t('components.ydAiChat.supplementPlaceholder')" class="w-full" />
    </div>
    <!-- 底部：导航 + 统一提交 -->
    <div class="mt-3 flex items-center justify-between gap-2">
      <div class="flex gap-1.5 text-xs text-muted-foreground">
        <FaButton v-if="idx > 0" size="sm" variant="outline" @click="idx--">
          {{ $t('components.ydAiChat.prevQuestion') }}
        </FaButton>
        <FaButton v-if="idx < total - 1" size="sm" variant="outline" @click="idx++">
          {{ $t('components.ydAiChat.nextQuestion') }}
        </FaButton>
        <span v-if="total > 1" class="flex items-center gap-1">
          <template v-for="(a, i) in answered" :key="i">
            <span class="size-1.5 rounded-full" :class="a ? 'bg-emerald-500' : 'bg-muted-foreground/30'" />
          </template>
        </span>
      </div>
      <div class="flex items-center gap-2">
        <FaButton v-if="idx < total - 1" size="sm" :disabled="!answered[idx]" @click="idx++">
          {{ $t('components.ydAiChat.nextQuestion') }}
        </FaButton>
        <FaButton size="sm" :variant="canSubmit ? 'default' : 'outline'" :disabled="!canSubmit" @click="submit">
          {{ $t('components.ydAiChat.submitAnswers') }}
        </FaButton>
      </div>
    </div>
  </div>
</template>
