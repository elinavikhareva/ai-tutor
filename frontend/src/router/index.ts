import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      redirect: '/courses',
    },
    {
      path: '/courses',
      name: 'courses',
      component: () => import('@/views/CoursesView.vue'),
    },
    {
      path: '/courses/new',
      name: 'courses-new',
      component: () => import('@/views/CreateCourseView.vue'),
    },
    {
      path: '/courses/:courseId',
      name: 'course-toc',
      component: () => import('@/views/CourseTOCView.vue'),
    },
    {
      path: '/courses/:courseId/chapters/:chapterId',
      name: 'chapter',
      component: () => import('@/views/ChapterView.vue'),
    },
    {
      path: '/lessons/:lessonId',
      name: 'lesson',
      component: () => import('@/views/LessonView.vue'),
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.isAuthenticated && auth.canRefresh) {
    await auth.refresh()
  }
  if (!to.meta.public && !auth.isAuthenticated) {
    return { name: 'login' }
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'courses' }
  }
})

export default router
