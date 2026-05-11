<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import { useApi } from '@/composables/useApi'

const router = useRouter()
const api = useApi()

const topic = ref('')
const description = ref('')
const depth = ref(2)
const submitting = ref(false)
const error = ref<string | null>(null)

const DEPTHS = [
  { n: 1, label: 'Поверхностно', hint: 'Мало глав, 3–4 обзорных урока' },
  { n: 2, label: 'Рабочий', hint: 'Средний объём и детальность' },
  { n: 3, label: 'Эксперт', hint: 'Больше глав, 6–10 глубоких уроков' },
]

const ready = computed(() => topic.value.trim().length >= 3)

const depthCards = computed(() =>
  DEPTHS.map((d) => {
    const active = depth.value === d.n
    const dotColor = active ? '#fdfbf6' : '#2b2723'
    return {
      ...d,
      active,
      dotBorder: dotColor,
      dots: [0, 1, 2].map((i) => (i < d.n ? dotColor : 'transparent')),
    }
  }),
)

const footnote = computed(() =>
  ready.value ? 'Gemini соберёт оглавление за пару секунд' : 'введите тему курса',
)

async function onSubmit() {
  if (!ready.value || submitting.value) return
  submitting.value = true
  error.value = null
  try {
    const trimmedTopic = topic.value.trim()
    const motivation = description.value.trim() || trimmedTopic

    const course = await api.post<{ id: number }>('/api/v1/courses', {
      title: trimmedTopic,
      motivation,
      goal_depth: depth.value,
    })
    await router.replace({ name: 'course-toc', params: { courseId: course.id } })
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Не удалось создать курс'
    submitting.value = false
  }
}
</script>

<template>
  <div
    class="min-h-dvh bg-cream bg-[radial-gradient(rgba(120,100,68,0.28)_1.4px,transparent_1.4px)] bg-[length:26px_26px] bg-[position:-8px_-8px] px-8 py-10 pb-18 font-sans text-ink"
  >
    <div class="mx-auto max-w-[760px]">
      <div class="mb-[30px] flex items-center justify-between">
        <div class="flex items-baseline gap-3">
          <span class="font-serif-display text-[30px] font-bold italic tracking-[-0.5px]">Learning</span>
          <span class="font-hand text-[21px] text-muted">учебник, который отвечает</span>
        </div>
        <RouterLink
          :to="{ name: 'courses' }"
          class="inline-flex h-[42px] items-center gap-2 rounded-[13px] border-[2.5px] border-ink bg-paper px-4 text-sm font-bold text-ink no-underline hover:bg-[#f1ece0]"
        >
          ← К курсам
        </RouterLink>
      </div>

      <div
        class="rounded-[26px] border-[3px] border-ink bg-paper px-10 pt-10 pb-[34px] shadow-[7px_8px_0_rgba(43,39,35,0.12)]"
      >
        <h1 class="m-0 mb-1.5 font-serif-display text-[40px] font-bold italic tracking-[-1px]">
          Новый курс
        </h1>
        <p class="m-0 mb-[30px] text-[15px] font-medium text-muted">
          Задайте тему и глубину — план глав соберётся автоматически.
        </p>

        <div class="mb-7">
          <div class="mb-[13px] flex items-center gap-2.5">
            <span
              class="inline-flex h-6 w-6 items-center justify-center rounded-[7px] border-2 border-ink bg-accent text-xs font-extrabold"
              >1</span
            >
            <span class="text-sm font-extrabold tracking-[0.2px] text-muted-dark">ТЕМА КУРСА</span>
          </div>
          <InputText
            v-model="topic"
            placeholder="напр. Конкурентность в Go: горутины и каналы"
            class="h-[60px] w-full rounded-[15px] border-[2.5px] border-ink bg-accent px-5 font-sans text-[17px] font-semibold text-ink outline-none"
          />
        </div>

        <div class="mb-7">
          <div class="mb-[13px] flex items-center gap-2.5">
            <span
              class="inline-flex h-6 w-6 items-center justify-center rounded-[7px] border-2 border-ink bg-accent text-xs font-extrabold"
              >2</span
            >
            <span class="text-sm font-extrabold tracking-[0.2px] text-muted-dark">ОПИСАНИЕ ТЕМЫ</span>
            <span class="text-[12.5px] font-semibold text-muted-light">необязательно</span>
          </div>
          <Textarea
            v-model="description"
            :rows="4"
            placeholder="Что именно хотите разобрать, зачем и с каким уклоном — чем точнее, тем лучше план глав."
            class="min-h-[108px] w-full resize-y rounded-[15px] border-[2.5px] border-ink bg-accent px-5 py-4 font-sans text-[15px] leading-normal font-semibold text-ink outline-none"
          />
        </div>

        <div class="mb-[34px]">
          <div class="mb-[13px] flex items-center gap-2.5">
            <span
              class="inline-flex h-6 w-6 items-center justify-center rounded-[7px] border-2 border-ink bg-accent text-xs font-extrabold"
              >3</span
            >
            <span class="text-sm font-extrabold tracking-[0.2px] text-muted-dark">ГЛУБИНА</span>
          </div>
          <div class="grid grid-cols-3 gap-3">
            <button
              v-for="dp in depthCards"
              :key="dp.n"
              type="button"
              class="flex flex-col gap-2 rounded-[15px] border-[2.5px] border-ink px-4 pt-4 pb-[15px] text-left font-sans cursor-pointer"
              :class="dp.active ? 'bg-ink text-paper' : 'bg-depth-inactive text-ink'"
              @click="depth = dp.n"
            >
              <span class="inline-flex gap-[5px]">
                <span
                  v-for="(dot, i) in dp.dots"
                  :key="i"
                  class="h-2 w-2 rounded-full border-[1.5px]"
                  :style="{ borderColor: dp.dotBorder, background: dot }"
                ></span>
              </span>
              <span class="text-base font-extrabold">{{ dp.label }}</span>
              <span class="text-[12.5px] leading-snug font-semibold opacity-[0.72]">{{ dp.hint }}</span>
            </button>
          </div>
        </div>

        <button
          type="button"
          :disabled="!ready || submitting"
          class="flex h-[62px] w-full items-center justify-center gap-2.5 rounded-2xl border-[2.5px] border-ink font-sans text-lg font-extrabold transition-[box-shadow,transform] duration-100"
          :class="
            ready && !submitting
              ? 'cursor-pointer bg-ink text-paper shadow-[4px_4px_0_rgba(43,39,35,0.18)] hover:translate-x-px hover:translate-y-px hover:shadow-[3px_3px_0_rgba(43,39,35,0.18)]'
              : 'cursor-not-allowed bg-disabled text-muted'
          "
          @click="onSubmit"
        >
          {{ submitting ? 'Собираем план глав…' : 'Создать курс →' }}
        </button>

        <p v-if="error" class="mt-3 mb-0 text-center text-sm font-medium text-[#C0392B]">
          {{ error }}
        </p>

        <p class="mt-3.5 mb-0 text-center font-hand text-[19px] text-muted-light">{{ footnote }}</p>
      </div>
    </div>
  </div>
</template>
