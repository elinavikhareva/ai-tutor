<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useApi } from '@/composables/useApi'
import type { Course } from '@/types/api'
import { courseAccent, depthLabel, plural } from '@/utils/format'

const router = useRouter()
const auth = useAuthStore()
const api = useApi()

const courses = ref<Course[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

function depthDots(d: number) {
  return [0, 1, 2].map((i) => i < d)
}

const MONTHS = ['янв', 'фев', 'мар', 'апр', 'мая', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек']

function fmtDate(iso: string) {
  const d = new Date(iso)
  return `${d.getDate()} ${MONTHS[d.getMonth()]}`
}

const countLabel = computed(() => {
  const n = courses.value.length
  return `${n} ${plural(n, ['курс', 'курса', 'курсов'])}`
})

onMounted(async () => {
  try {
    courses.value = await api.get<Course[]>('/api/v1/courses')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    loading.value = false
  }
})

async function onLogout() {
  await auth.logout()
  await router.replace({ name: 'login' })
}
</script>

<template>
  <div class="page">
    <header class="topnav">
      <div class="topnav__brand-group">
        <span class="topnav__brand">Learning</span>
        <span class="topnav__tagline">учебник, который отвечает</span>
      </div>
      <div class="topnav__right">
        <div class="topnav__avatar">
          {{ auth.username?.[0]?.toUpperCase() ?? 'A' }}
        </div>
        <button class="topnav__logout" @click="onLogout">Выйти</button>
      </div>
    </header>

    <div class="page-content">
      <div class="heading-row">
        <div>
          <h1 class="heading-row__title">Мои курсы</h1>
          <p class="heading-row__count">{{ countLabel }}</p>
        </div>
        <RouterLink :to="{ name: 'courses-new' }" class="new-course-btn">
          <span class="new-course-btn__icon">+</span> Новый курс
        </RouterLink>
      </div>

      <div v-if="loading" class="state-msg">Загружаем курсы…</div>
      <div v-else-if="error" class="state-msg state-msg--err">{{ error }}</div>

      <div v-else class="grid">
        <RouterLink
          v-for="c in courses"
          :key="c.id"
          :to="{ name: 'course-toc', params: { courseId: c.id } }"
          class="course-card"
        >
          <div class="course-card__top">
            <span class="course-card__dot" :style="{ background: courseAccent(c.id) }"></span>
            <span class="course-card__date">{{ fmtDate(c.created_at) }}</span>
          </div>

          <div class="course-card__title">{{ c.title }}</div>

          <div class="course-card__depth-row">
            <span class="course-card__depth-label">Глубина</span>
            <span class="course-card__depth-dots">
              <span
                v-for="(filled, i) in depthDots(c.goal_depth)"
                :key="i"
                class="course-card__depth-dot"
                :class="{ 'course-card__depth-dot--filled': filled }"
              ></span>
            </span>
            <span class="course-card__depth-text">{{ depthLabel(c.goal_depth) }}</span>
          </div>

        </RouterLink>
      </div>

      <div v-if="!loading && !error && courses.length === 0" class="empty">
        <p class="empty__title">У вас пока нет курсов.</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page {
  min-height: 100dvh;
  background-color: #f7f1e4;
  background-image: radial-gradient(rgba(120, 100, 68, 0.28) 1.4px, transparent 1.4px);
  background-size: 26px 26px;
  background-position: -8px -8px;
  font-family: 'Manrope', sans-serif;
  color: #2b2723;
}

.topnav {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 22px;
  border-bottom: 2.5px solid #2b2723;
  background: #f1ece0;
}

.topnav__brand-group {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.topnav__brand {
  font: 700 italic 24px 'Playfair Display', serif;
  color: #2b2723;
}

.topnav__tagline {
  font: 500 15px 'Caveat', cursive;
  color: #8a7f6a;
}

.topnav__right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 12px;
}

.topnav__avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 2.5px solid;
  background: #e7ecfb;
  display: flex;
  align-items: center;
  justify-content: center;
  font: 700 italic 16px 'Playfair Display', serif;
  color: #2b2723;
}

.topnav__logout {
  font: 700 13px 'Manrope', sans-serif;
  color: #8a7f6a;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}

.topnav__logout:hover {
  color: #2b2723;
}

.page-content {
  max-width: 1120px;
  margin: 0 auto;
  padding: 32px 24px 60px;
}

.heading-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 28px;
}

.heading-row__title {
  font: 700 italic 38px 'Playfair Display', serif;
  color: #2b2723;
  margin: 0 0 4px;
  letter-spacing: -0.5px;
}

.heading-row__count {
  margin: 0;
  font: 500 14px 'Manrope', sans-serif;
  color: #8a7f6a;
}

.new-course-btn {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  height: 52px;
  padding: 0 24px;
  border: 2.5px solid #2b2723;
  border-radius: 15px;
  background: #2b2723;
  color: #fdfbf6;
  font: 700 16px 'Manrope', sans-serif;
  text-decoration: none;
  cursor: pointer;
  box-shadow: 4px 4px 0 rgba(43, 39, 35, 0.18);
  transition: box-shadow .12s, transform .12s;
  flex: none;
}

.new-course-btn:hover {
  box-shadow: 3px 3px 0 rgba(43, 39, 35, 0.18);
  transform: translate(1px, 1px);
}

.new-course-btn__icon {
  font-size: 22px;
  line-height: 0;
  margin-top: -2px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 18px;
}

.course-card {
  background: #fdfbf6;
  border: 2.5px solid #2b2723;
  border-radius: 22px;
  box-shadow: 5px 6px 0 rgba(43, 39, 35, 0.10);
  padding: 22px 22px 20px;
  text-decoration: none;
  color: inherit;
  display: flex;
  flex-direction: column;
  transition: box-shadow .12s, transform .12s;
}

.course-card:hover {
  box-shadow: 4px 5px 0 rgba(43, 39, 35, 0.10);
  transform: translate(1px, 1px);
}

.course-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.course-card__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: none;
}

.course-card__date {
  font: 600 12px 'Manrope', sans-serif;
  color: #a89a80;
}

.course-card__title {
  font: 600 23px/1.18 'Playfair Display', serif;
  color: #2b2723;
  margin: 0;
  letter-spacing: -0.3px;
}

.course-card__depth-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 18px 0 14px;
}

.course-card__depth-label {
  font: 700 12px 'Manrope', sans-serif;
  color: #8a7f6a;
}

.course-card__depth-dots {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.course-card__depth-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  border: 1.5px solid #2b2723;
  background: transparent;
}

.course-card__depth-dot--filled {
  background: #2b2723;
}

.course-card__depth-text {
  font: 700 12.5px 'Manrope', sans-serif;
  color: #2b2723;
}

.state-msg {
  padding: 24px 0;
  font: 400 15px 'Manrope', sans-serif;
  color: #8a7f6a;
}

.state-msg--err { color: #C0392B; }

.empty {
  padding: 60px 20px;
  text-align: center;
  color: #8a7f6a;
}

.empty__title {
  font: 500 26px 'Caveat', cursive;
  margin: 0;
}

@media (max-width: 640px) {
  .topnav__tagline { display: none; }
  .heading-row { flex-direction: column; align-items: flex-start; }
  .new-course-btn { width: 100%; justify-content: center; }
}
</style>
