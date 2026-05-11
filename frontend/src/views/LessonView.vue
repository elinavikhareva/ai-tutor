<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useApi } from '@/composables/useApi'
import type { ChatMessage, Lesson } from '@/types/api'

const router = useRouter()
const route = useRoute()
const api = useApi()

const lessonId = Number(route.params.lessonId)
const courseId = route.query.courseId ? Number(route.query.courseId) : null
const chapterId = route.query.chapterId ? Number(route.query.chapterId) : null

const lesson = ref<Lesson | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const messages = ref<ChatMessage[]>([])
const inputText = ref('')
const streaming = ref(false)
const streamingText = ref('')
const chatEl = ref<HTMLElement | null>(null)

const activeTab = ref<'lecture' | 'tutor'>('lecture')
const chatCollapsed = ref(false)
const completing = ref(false)

const QUICK_REPLIES = ['Поясни проще', 'Пример', 'Дай задачу']

const contentParagraphs = computed(() => {
  if (!lesson.value?.content) return []
  return lesson.value.content.split('\n').filter((p) => p.trim())
})

onMounted(async () => {
  try {
    lesson.value = await api.get<Lesson>(`/api/v1/lessons/${lessonId}`)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    loading.value = false
  }
})

async function sendMessage(text: string) {
  if (!text.trim() || streaming.value) return
  messages.value.push({ role: 'user', text: text.trim() })
  inputText.value = ''
  streaming.value = true
  streamingText.value = ''
  scrollChat()

  try {
    for await (const chunk of api.streamChat(lessonId, [...messages.value])) {
      streamingText.value += chunk
      scrollChat()
    }
    if (streamingText.value) {
      messages.value.push({ role: 'assistant', text: streamingText.value })
    }
  } catch (e) {
    const reason = e instanceof Error ? e.message : 'Ошибка соединения с тутором'
    messages.value.push({ role: 'assistant', text: reason })
  } finally {
    streaming.value = false
    streamingText.value = ''
    scrollChat()
  }
}

function scrollChat() {
  nextTick(() => {
    if (chatEl.value) {
      chatEl.value.scrollTop = chatEl.value.scrollHeight
    }
  })
}

async function completeLesson() {
  if (completing.value) return
  completing.value = true
  try {
    await api.post(`/api/v1/lessons/${lessonId}/complete`, { history: messages.value })
    goBack()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Не удалось завершить урок'
    completing.value = false
  }
}

function goBack() {
  if (courseId && chapterId) {
    router.push({ name: 'chapter', params: { courseId, chapterId } })
  } else {
    router.back()
  }
}
</script>

<template>
  <div class="page">
    <div v-if="loading" class="state-msg">Загружаем урок…</div>
    <div v-else-if="error" class="state-msg state-msg--err">{{ error }}</div>

    <template v-else-if="lesson">
      <header class="topnav">
        <button class="back-btn" @click="goBack">‹ Глава</button>
        <span class="topnav__num">{{ lesson.position }}</span>
        <span class="topnav__title">{{ lesson.title }}</span>
        <button class="complete-btn" :disabled="completing" @click="completeLesson">
          {{ completing ? '…' : 'Завершить' }}
        </button>
      </header>

      <div class="mobile-tabs">
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'lecture' }"
          @click="activeTab = 'lecture'"
        >Лекция</button>
        <button
          class="tab-btn tab-btn--tutor"
          :class="{ 'tab-btn--active': activeTab === 'tutor' }"
          @click="activeTab = 'tutor'"
        >
          <span class="tutor-icon"></span>
          Тутор
        </button>
      </div>

      <div class="body">
        <div
          class="lecture"
          :class="{ 'lecture--hidden-mobile': activeTab === 'tutor' }"
        >
          <div class="lecture__inner">
            <h1 class="lecture__title">{{ lesson.title }}</h1>
            <p class="lecture__objective">{{ lesson.objective }}</p>

            <div v-if="contentParagraphs.length > 0" class="lecture__content">
              <p v-for="(p, i) in contentParagraphs" :key="i" class="lecture__para">{{ p }}</p>
            </div>
            <div v-else class="lecture__placeholder">
              <div class="skeleton-line" style="width:97%"></div>
              <div class="skeleton-line" style="width:92%"></div>
              <div class="skeleton-line" style="width:84%"></div>
              <div class="skeleton-line" style="width:64%"></div>
            </div>
          </div>

          <div class="collapse-handle" @click="chatCollapsed = !chatCollapsed">
            <span class="collapse-handle__bar"></span>
          </div>
        </div>

        <div v-if="chatCollapsed" class="chat-rail-collapsed" @click="chatCollapsed = false">
          <span class="tutor-icon tutor-icon--lg"></span>
          <span class="rail-label">Тутор</span>
          <span class="rail-expand">‹</span>
        </div>

        <div
          v-if="!chatCollapsed"
          class="chat"
          :class="{ 'chat--hidden-mobile': activeTab === 'lecture' }"
        >
          <div class="chat__header">
            <span class="tutor-icon"></span>
            <span class="chat__label">Тутор</span>
            <span class="chat__sub">по этому уроку</span>
            <button class="chat__collapse-btn" @click="chatCollapsed = true">⤢</button>
          </div>

          <div ref="chatEl" class="chat__messages">
            <div v-if="messages.length === 0" class="chat__empty">
              Задайте вопрос по уроку — тутор ответит.
            </div>
            <div
              v-for="(m, i) in messages"
              :key="i"
              class="bubble"
              :class="m.role === 'user' ? 'bubble--user' : 'bubble--tutor'"
            >{{ m.text }}</div>
            <div v-if="streaming" class="bubble bubble--tutor">
              {{ streamingText }}<span v-if="streamingText" class="cursor"></span>
              <span v-else class="typing-dots">
                <span></span><span></span><span></span>
              </span>
            </div>
          </div>

          <div class="quick-replies">
            <button
              v-for="r in QUICK_REPLIES"
              :key="r"
              class="quick-reply"
              @click="sendMessage(r)"
            >{{ r }}</button>
          </div>

          <div class="chat__input-row">
            <input
              v-model="inputText"
              class="chat__input"
              type="text"
              placeholder="Спросите…"
              :disabled="streaming"
              @keydown.enter="sendMessage(inputText)"
            />
            <button
              class="chat__send"
              :disabled="streaming || !inputText.trim()"
              @click="sendMessage(inputText)"
            >↑</button>
          </div>
        </div>
      </div>

      <div class="mobile-finish">
        <button class="mobile-finish__btn" :disabled="completing" @click="completeLesson">
          {{ completing ? '…' : 'Завершить урок' }}
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.page {
  height: 100dvh;
  display: flex;
  flex-direction: column;
  background: #FBF7EE;
  font-family: 'Hanken Grotesk', sans-serif;
  color: #2E2B26;
}

.state-msg {
  padding: 40px 24px;
  font: 400 15px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
}

.state-msg--err { color: #C0392B; }

.topnav {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 13px 18px;
  border-bottom: 1.6px solid #2E2B26;
  background: #F1EADC;
  flex: none;
}

.back-btn {
  font: 600 14px 'Hanken Grotesk', sans-serif;
  color: #2E2B26;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}

.topnav__num {
  font: 600 12px 'Hanken Grotesk', sans-serif;
  color: #8b8270;
}

.topnav__title {
  font: 500 italic 17px 'Newsreader', serif;
  color: #2E2B26;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.complete-btn {
  height: 30px;
  padding: 0 11px;
  border: none;
  border-radius: 7px;
  background: #2E2B26;
  color: #FBF7EE;
  font: 600 12px 'Hanken Grotesk', sans-serif;
  cursor: pointer;
  flex: none;
}

.complete-btn:disabled { opacity: .6; cursor: default; }

.mobile-tabs {
  display: none;
  gap: 6px;
  margin: 12px 14px 0;
  padding: 4px;
  background: #EDE7D9;
  border: 1.5px solid #2E2B26;
  border-radius: 11px;
}

.tab-btn {
  flex: 1;
  text-align: center;
  padding: 7px 0;
  border-radius: 8px;
  border: none;
  background: none;
  color: #6E6A60;
  font: 600 13px 'Hanken Grotesk', sans-serif;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
}

.tab-btn--active {
  background: #2E2B26;
  color: #FBF7EE;
}

.tutor-icon {
  display: inline-block;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: #0E93AD;
}

.tutor-icon--lg {
  width: 30px;
  height: 30px;
}

.body {
  flex: 1;
  display: flex;
  overflow: hidden;
  background: #FBF7EE;
}

.lecture {
  flex: 1;
  position: relative;
  overflow-y: auto;
  min-width: 0;
}

.lecture__inner {
  padding: 30px 34px 50px;
  max-width: 720px;
}

.lecture__title {
  font: 500 italic 27px/1.2 'Newsreader', serif;
  color: #2E2B26;
  margin: 0 0 10px;
}

.lecture__objective {
  font: 400 15px 'Hanken Grotesk', sans-serif;
  color: #6E6A60;
  margin: 0 0 20px;
}

.lecture__para {
  font: 400 17px/1.7 'Newsreader', serif;
  color: #33312C;
  margin: 0 0 14px;
}

.lecture__placeholder {
  display: flex;
  flex-direction: column;
  gap: 9px;
  margin-top: 16px;
}

.skeleton-line {
  height: 9px;
  border-radius: 5px;
  background: #ECE6D8;
}

.collapse-handle {
  position: absolute;
  top: 50%;
  right: -1px;
  transform: translateY(-50%);
  width: 12px;
  height: 46px;
  cursor: col-resize;
  display: flex;
  align-items: center;
  justify-content: center;
}

.collapse-handle__bar {
  width: 6px;
  height: 46px;
  border-radius: 4px;
  background: #cfc7b5;
}

.chat-rail-collapsed {
  flex: none;
  width: 56px;
  border-left: 1.6px solid #2E2B26;
  background: #F1EADC;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 18px;
  gap: 14px;
  cursor: pointer;
}

.rail-label {
  writing-mode: vertical-rl;
  transform: rotate(180deg);
  font: 600 12px 'Hanken Grotesk', sans-serif;
  letter-spacing: .16em;
  text-transform: uppercase;
  color: #2E2B26;
}

.rail-expand {
  margin-top: auto;
  margin-bottom: 18px;
  font: 600 17px 'Hanken Grotesk', sans-serif;
  color: #2E2B26;
  border: 1.6px solid #2E2B26;
  border-radius: 8px;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.chat {
  flex: none;
  width: 336px;
  border-left: 1.6px solid #2E2B26;
  background: #FFFEF9;
  display: flex;
  flex-direction: column;
}

.chat__header {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 13px 16px;
  border-bottom: 1.4px dashed #e0d9c8;
  flex: none;
}

.chat__label {
  font: 600 14px 'Hanken Grotesk', sans-serif;
  color: #2E2B26;
}

.chat__sub {
  font: 400 15px 'Caveat', cursive;
  color: #8b8270;
}

.chat__collapse-btn {
  margin-left: auto;
  background: none;
  border: none;
  font: 500 18px 'Hanken Grotesk', sans-serif;
  color: #b3aa96;
  cursor: pointer;
}

.chat__messages {
  flex: 1;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 11px;
  overflow-y: auto;
}

.chat__empty {
  font: 400 14px 'Hanken Grotesk', sans-serif;
  color: #a59c88;
  text-align: center;
  margin-top: 20px;
}

.bubble {
  max-width: 88%;
  padding: 9px 13px;
  font: 400 15px/1.5 'Newsreader', serif;
  word-break: break-word;
}

.bubble--user {
  align-self: flex-end;
  background: #2E2B26;
  color: #FBF7EE;
  border-radius: 14px 14px 4px 14px;
}

.bubble--tutor {
  align-self: flex-start;
  background: #F4EEDF;
  border: 1.4px solid #e6dfce;
  color: #33312C;
  border-radius: 14px 14px 14px 4px;
}

.cursor {
  display: inline-block;
  width: 8px;
  height: 17px;
  background: #0E93AD;
  margin-left: 2px;
  vertical-align: -3px;
}

.typing-dots {
  display: inline-flex;
  gap: 4px;
  align-items: center;
}

.typing-dots span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #b3aa96;
  animation: blink 1.2s infinite;
}

.typing-dots span:nth-child(2) { animation-delay: .2s; }
.typing-dots span:nth-child(3) { animation-delay: .4s; }

@keyframes blink {
  0%, 80%, 100% { opacity: .3; }
  40% { opacity: 1; }
}

.quick-replies {
  padding: 0 14px 8px;
  display: flex;
  gap: 7px;
  flex-wrap: wrap;
  flex: none;
}

.quick-reply {
  font: 500 12px 'Hanken Grotesk', sans-serif;
  color: #6E6A60;
  border: 1.4px solid #cfc7b5;
  border-radius: 14px;
  padding: 5px 10px;
  background: #fff;
  cursor: pointer;
  transition: background .1s;
}

.quick-reply:hover { background: #F4EEDF; }

.chat__input-row {
  padding: 0 14px 14px;
  display: flex;
  gap: 9px;
  flex: none;
}

.chat__input {
  flex: 1;
  height: 44px;
  border: 1.7px solid #2E2B26;
  border-radius: 22px;
  background: #fff;
  padding: 0 16px;
  font: 400 15px 'Newsreader', serif;
  color: #3a3730;
  outline: none;
}

.chat__input::placeholder { color: #b3aa96; }
.chat__input:disabled { opacity: .6; }

.chat__send {
  width: 44px;
  height: 44px;
  border: 1.8px solid #2E2B26;
  border-radius: 50%;
  background: #2E2B26;
  color: #FBF7EE;
  font-size: 18px;
  cursor: pointer;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
}

.chat__send:disabled { opacity: .5; cursor: default; }

.mobile-finish {
  display: none;
  padding: 10px 14px 14px;
  flex: none;
}

.mobile-finish__btn {
  width: 100%;
  height: 42px;
  border: 1.7px solid #2E2B26;
  border-radius: 10px;
  background: #2E2B26;
  color: #FBF7EE;
  font: 600 14px 'Hanken Grotesk', sans-serif;
  cursor: pointer;
}

@media (max-width: 768px) {
  .mobile-tabs { display: flex; }
  .mobile-finish { display: block; }
  .collapse-handle { display: none; }

  .chat {
    flex: 1;
    width: 100%;
    border-left: none;
  }

  .chat--hidden-mobile { display: none; }
  .lecture--hidden-mobile { display: none; }
  .lecture { width: 100%; }
  .chat-rail-collapsed { display: none; }

  .complete-btn { display: none; }
}
</style>
