<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useApi } from '@/composables/useApi'
import type { Chapter, CourseDetail } from '@/types/api'
import { courseAccent, depthLabel, plural } from '@/utils/format'

const router = useRouter()
const route = useRoute()
const api = useApi()

const courseId = Number(route.params.courseId)
const accent = courseAccent(courseId)
const DONE_COLOR = '#0F9D5E'

const course = ref<CourseDetail | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const chapters = computed(() =>
  (course.value?.chapters ?? []).map((ch) => {
    const total = ch.lessons.length
    const done = ch.lessons.filter((l) => l.status === 'done').length
    return { ...ch, total, done, percent: total > 0 ? (done / total) * 100 : 0 }
  }),
)

const totalLessons = computed(() => chapters.value.reduce((s, ch) => s + ch.total, 0))
const doneLessons = computed(() => chapters.value.reduce((s, ch) => s + ch.done, 0))
const progress = computed(() =>
  totalLessons.value > 0 ? Math.round((doneLessons.value / totalLessons.value) * 100) : 0,
)

function chapterColor(ch: Chapter) {
  if (ch.status === 'done') return DONE_COLOR
  if (ch.status === 'in_progress') return accent
  return '#8b8270'
}

onMounted(async () => {
  try {
    course.value = await api.get<CourseDetail>(`/api/v1/courses/${courseId}`)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    loading.value = false
  }
})

function goToChapter(chapterId: number) {
  router.push({ name: 'chapter', params: { courseId, chapterId } })
}
</script>

<template>
  <div class="page">
    <header class="topnav">
      <button class="back-btn" @click="router.push({ name: 'courses' })">‹ Курсы</button>
      <span v-if="course" class="domain-dot" :style="{ background: accent }"></span>
      <span class="topnav__title">{{ course?.title ?? '…' }}</span>
      <span v-if="course" class="topnav__meta">
        {{ depthLabel(course.goal_depth) }} · {{ doneLessons }}/{{ totalLessons }}
        {{ plural(totalLessons, ['урок', 'урока', 'уроков']) }}
      </span>
    </header>

    <div class="body">
      <div v-if="loading" class="state-msg">Загружаем курс…</div>
      <div v-else-if="error" class="state-msg state-msg--err">{{ error }}</div>

      <template v-else-if="course">
        <div class="progress-row">
          <div class="section-title">Главы</div>
          <div class="progress-bar-wrap">
            <div class="progress-bar">
              <div class="progress-bar__fill" :style="{ width: `${progress}%`, background: accent }"></div>
            </div>
            <span class="progress-pct">{{ progress }}%</span>
          </div>
        </div>

        <div class="chapters">
          <button
            v-for="ch in chapters"
            :key="ch.id"
            class="chapter-row"
            :class="{
              'chapter-row--active': ch.status === 'in_progress',
              'chapter-row--pending': ch.status === 'pending',
            }"
            @click="goToChapter(ch.id)"
          >
            <span class="chapter-num">{{ ch.number }}</span>
            <div class="chapter-info">
              <div class="chapter-title" :class="{ 'chapter-title--dim': ch.status === 'pending' }">
                {{ ch.title }}
              </div>
              <div class="chapter-sub">
                {{ ch.total }} {{ plural(ch.total, ['урок', 'урока', 'уроков']) }}
              </div>
            </div>
            <div v-if="ch.total > 0" class="chapter-progress">
              <div class="mini-bar">
                <div
                  class="mini-bar__fill"
                  :style="{ width: `${ch.percent}%`, background: ch.status === 'done' ? DONE_COLOR : accent }"
                ></div>
              </div>
              <span class="chapter-progress-label" :style="{ color: chapterColor(ch) }">
                {{ ch.done }}/{{ ch.total }}
              </span>
            </div>
            <span class="chevron">›</span>
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.page {
  min-height: 100dvh;
  background: #FBF7EE;
  font-family: 'Hanken Grotesk', sans-serif;
  color: #2E2B26;
}

.topnav {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 22px;
  border-bottom: 1.6px solid #2E2B26;
  background: #F1EADC;
}

.back-btn {
  font: 600 15px 'Hanken Grotesk', sans-serif;
  color: #2E2B26;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}

.domain-dot {
  width: 11px;
  height: 11px;
  border-radius: 50%;
  flex: none;
  margin-left: 6px;
}

.topnav__title {
  font: 500 italic 22px 'Newsreader', serif;
  color: #2E2B26;
}

.topnav__meta {
  margin-left: auto;
  font: 400 13px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
}

.body {
  padding: 24px 28px 32px;
}

.state-msg {
  padding: 24px 0;
  font: 400 15px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
}

.state-msg--err { color: #C0392B; }

.progress-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18px;
}

.section-title {
  font: 500 italic 26px 'Newsreader', serif;
  color: #2E2B26;
}

.progress-bar-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 300px;
}

.progress-bar {
  flex: 1;
  height: 9px;
  border-radius: 5px;
  background: #E7E0D0;
  overflow: hidden;
}

.progress-bar__fill {
  height: 100%;
  transition: width .3s;
}

.progress-pct {
  font: 600 12px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
}

.chapters {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.chapter-row {
  display: flex;
  align-items: center;
  gap: 18px;
  background: #FFFEF9;
  border: 1.7px solid #2E2B26;
  border-radius: 12px;
  padding: 16px 20px;
  width: 100%;
  text-align: left;
  cursor: pointer;
  transition: box-shadow .12s;
}

.chapter-row:hover { box-shadow: 3px 4px 0 rgba(46,43,38,.09); }

.chapter-row--active {
  border-width: 2px;
  box-shadow: 3px 4px 0 rgba(46,43,38,.09);
}

.chapter-row--pending {
  background: rgba(255,255,255,.45);
  border-style: dashed;
  border-color: #bdb49f;
}

.chapter-num {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 9px;
  border: 1.7px solid #2E2B26;
  background: #FBF3E2;
  display: flex;
  align-items: center;
  justify-content: center;
  font: 500 20px 'Newsreader', serif;
  color: #2E2B26;
}

.chapter-row--pending .chapter-num {
  border-style: dashed;
  border-color: #bdb49f;
  background: #F4EEDF;
  color: #b3aa96;
}

.chapter-info { flex: 1; }

.chapter-title {
  font: 500 19px 'Newsreader', serif;
  color: #2E2B26;
}

.chapter-title--dim { color: #8b8270; }

.chapter-sub {
  font: 400 12px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
  margin-top: 3px;
}

.chapter-progress {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 160px;
}

.mini-bar {
  flex: 1;
  height: 7px;
  border-radius: 4px;
  background: #E7E0D0;
  overflow: hidden;
}

.mini-bar__fill {
  height: 100%;
  transition: width .3s;
}

.chapter-progress-label {
  font: 600 12px 'Hanken Grotesk', sans-serif;
}

.chevron {
  font: 600 15px 'Hanken Grotesk', sans-serif;
  color: #2E2B26;
}

.chapter-row--pending .chevron { color: #bdb49f; }

@media (max-width: 600px) {
  .progress-bar-wrap { width: 160px; }
  .chapter-progress { display: none; }
}
</style>
