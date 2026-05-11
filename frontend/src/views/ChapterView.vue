<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useApi } from '@/composables/useApi'
import type { Chapter, CourseDetail } from '@/types/api'
import { courseAccent, plural } from '@/utils/format'

const router = useRouter()
const route = useRoute()
const api = useApi()

const courseId = Number(route.params.courseId)
const chapterId = Number(route.params.chapterId)
const accent = courseAccent(courseId)

const course = ref<CourseDetail | null>(null)
const chapter = ref<Chapter | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const doneCount = computed(() => chapter.value?.lessons.filter((l) => l.status === 'done').length ?? 0)
const totalCount = computed(() => chapter.value?.lessons.length ?? 0)
const progress = computed(() =>
  totalCount.value > 0 ? Math.round((doneCount.value / totalCount.value) * 100) : 0,
)

onMounted(async () => {
  try {
    course.value = await api.get<CourseDetail>(`/api/v1/courses/${courseId}`)
    chapter.value = course.value.chapters.find((c) => c.id === chapterId) ?? null
    if (!chapter.value) error.value = 'Глава не найдена'
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    loading.value = false
  }
})

function goToLesson(lessonId: number) {
  router.push({
    name: 'lesson',
    params: { lessonId },
    query: { courseId, chapterId },
  })
}
</script>

<template>
  <div class="page">
    <header class="topnav">
      <button class="back-btn" @click="router.push({ name: 'course-toc', params: { courseId } })">
        ‹ {{ course?.title ?? 'Курс' }}
      </button>
      <span class="topnav__meta">
        Глава {{ chapter?.number }} · {{ totalCount }} {{ plural(totalCount, ['урок', 'урока', 'уроков']) }}
      </span>
    </header>

    <div class="body">
      <div v-if="loading" class="state-msg">Загружаем главу…</div>
      <div v-else-if="error" class="state-msg state-msg--err">{{ error }}</div>

      <template v-else-if="chapter">
        <div class="chapter-header">
          <div class="chapter-title">{{ chapter.number }} · {{ chapter.title }}</div>
          <div class="chapter-progress-wrap">
            <span class="chapter-progress-label">пройдено</span>
            <div class="progress-bar">
              <div class="progress-bar__fill" :style="{ width: `${progress}%`, background: accent }"></div>
            </div>
            <span class="progress-frac">{{ doneCount }}/{{ totalCount }}</span>
          </div>
        </div>

        <div class="lessons-label">Уроки главы</div>

        <div class="lessons">
          <button
            v-for="l in chapter.lessons"
            :key="l.id"
            class="lesson-row"
            :class="{ 'lesson-row--active': l.status === 'in_progress' }"
            @click="goToLesson(l.id)"
          >
            <span v-if="l.status === 'done'" class="status-icon status-icon--done">✓</span>
            <span
              v-else-if="l.status === 'in_progress'"
              class="status-icon status-icon--progress"
              :style="{ borderColor: accent }"
            ></span>
            <span v-else class="status-icon status-icon--empty"></span>

            <span class="lesson-num">{{ l.position }}</span>
            <div class="lesson-info">
              <span class="lesson-title">{{ l.title }}</span>
            </div>
            <span v-if="l.status === 'done'" class="lesson-tag lesson-tag--done">пройден</span>
            <span v-else-if="l.status === 'needs_review'" class="lesson-tag lesson-tag--review">
              повторить
            </span>
            <span v-else-if="l.status === 'in_progress'" class="lesson-tag">в процессе</span>
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

.topnav__meta {
  margin-left: auto;
  font: 400 13px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
}

.body {
  padding: 28px 32px;
}

.state-msg {
  padding: 24px 0;
  font: 400 15px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
}

.state-msg--err { color: #C0392B; }

.chapter-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 22px;
  flex-wrap: wrap;
  gap: 12px;
}

.chapter-title {
  font: 500 italic 30px 'Newsreader', serif;
  color: #2E2B26;
}

.chapter-progress-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
}

.chapter-progress-label {
  font: 400 12px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
}

.progress-bar {
  width: 160px;
  height: 8px;
  border-radius: 5px;
  background: #E7E0D0;
  overflow: hidden;
}

.progress-bar__fill {
  height: 100%;
  transition: width .3s;
}

.progress-frac {
  font: 600 12px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
}

.lessons-label {
  font: 600 12px 'Hanken Grotesk', sans-serif;
  letter-spacing: .14em;
  text-transform: uppercase;
  color: #9a917d;
  margin-bottom: 12px;
}

.lessons {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.lesson-row {
  display: flex;
  align-items: center;
  gap: 14px;
  background: #FFFEF9;
  border: 1.6px solid #2E2B26;
  border-radius: 11px;
  padding: 13px 16px;
  width: 100%;
  text-align: left;
  cursor: pointer;
  transition: box-shadow .12s;
}

.lesson-row:hover { box-shadow: 2px 3px 0 rgba(46,43,38,.08); }

.lesson-row--active {
  border-width: 2px;
  box-shadow: 3px 4px 0 rgba(46,43,38,.08);
}

.status-icon {
  flex: none;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.status-icon--done {
  background: #0F9D5E;
  color: #fff;
  font: 600 14px 'Hanken Grotesk', sans-serif;
}

.status-icon--progress {
  border: 2px solid;
  background: #fff;
}

.status-icon--empty {
  border: 2px solid #cfc7b5;
  background: #fff;
}

.lesson-num {
  font: 600 13px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
  width: 34px;
}

.lesson-info { flex: 1; }

.lesson-title {
  font: 500 17px 'Newsreader', serif;
  color: #2E2B26;
}

.lesson-tag {
  font: 500 12px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
}

.lesson-tag--done { color: #0F9D5E; }
.lesson-tag--review { color: #0E93AD; font-weight: 600; }

@media (max-width: 600px) {
  .chapter-header { flex-direction: column; align-items: flex-start; }
}
</style>
