<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const username = ref('')
const password = ref('')
const showPassword = ref(false)
const error = ref<string | null>(null)
const loading = ref(false)

async function onSubmit() {
  if (loading.value) return
  error.value = null
  loading.value = true
  try {
    await auth.login(username.value.trim(), password.value)
    await router.replace({ name: 'courses' })
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Не удалось войти'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="page">
    <div class="card">
      <div class="brand">Learning</div>
      <div class="brand-sub">учебник, который отвечает</div>

      <form class="form" novalidate @submit.prevent="onSubmit">
        <div class="field">
          <label class="label" for="username">Логин</label>
          <div class="input-wrap">
            <input
              id="username"
              v-model="username"
              class="input"
              type="text"
              autocomplete="username"
              placeholder="имя пользователя"
              :disabled="loading"
            />
          </div>
        </div>

        <div class="field">
          <label class="label" for="password">Пароль</label>
          <div class="input-wrap">
            <input
              id="password"
              v-model="password"
              class="input"
              :type="showPassword ? 'text' : 'password'"
              autocomplete="current-password"
              placeholder="••••••••"
              :disabled="loading"
            />
            <button
              type="button"
              class="show-btn"
              tabindex="-1"
              @click="showPassword = !showPassword"
            >{{ showPassword ? 'скрыть' : 'показать' }}</button>
          </div>
        </div>

        <div v-if="error" class="error-row" role="alert">
          <span class="error-dot"></span>
          {{ error }} · 5 попыток/мин
        </div>

        <button class="submit" type="submit" :disabled="loading">
          {{ loading ? 'Входим…' : 'Войти →' }}
        </button>
      </form>
    </div>
  </main>
</template>

<style scoped>
.page {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #FBF7EE;
  background-image: radial-gradient(#d3cdbe 1px, transparent 1px);
  background-size: 26px 26px;
  font-family: 'Hanken Grotesk', sans-serif;
}

.card {
  width: 420px;
  background: #FFFEF9;
  border: 1.8px solid #2E2B26;
  border-radius: 18px;
  box-shadow: 6px 7px 0 rgba(46,43,38,.1);
  padding: 42px 40px 36px;
  text-align: center;
  position: relative;
}

.brand {
  font: 500 italic 40px 'Newsreader', serif;
  color: #2E2B26;
}

.brand-sub {
  font: 400 20px 'Caveat', cursive;
  color: #8b8270;
  margin-top: 4px;
}

.form {
  margin-top: 30px;
  text-align: left;
}

.field + .field {
  margin-top: 16px;
}

.label {
  display: block;
  font: 600 13px 'Hanken Grotesk', sans-serif;
  color: #6E6A60;
  margin-bottom: 7px;
}

.input-wrap {
  position: relative;
}

.input {
  width: 100%;
  height: 48px;
  border: 1.7px solid #2E2B26;
  border-radius: 11px 7px 12px 8px;
  background: #fff;
  padding: 0 16px;
  font: 400 16px 'Newsreader', serif;
  color: #3a3730;
  box-sizing: border-box;
  outline: none;
  transition: box-shadow .15s;
}

.input:focus {
  box-shadow: 0 0 0 3px rgba(46,43,38,.08);
}

.input:disabled { opacity: .6; }

.show-btn {
  position: absolute;
  right: 14px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  cursor: pointer;
  font: 400 16px 'Caveat', cursive;
  color: #b3aa96;
  padding: 0;
}

.error-row {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 14px;
  font: 400 13px 'Hanken Grotesk', sans-serif;
  color: #a99f49;
}

.error-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #C0392B;
  flex: none;
}

.submit {
  margin-top: 22px;
  width: 100%;
  height: 50px;
  border: 1.8px solid #2E2B26;
  border-radius: 11px 7px 12px 8px;
  background: #2E2B26;
  color: #FBF7EE;
  font: 600 16px 'Hanken Grotesk', sans-serif;
  cursor: pointer;
  transition: opacity .15s;
}

.submit:hover:not(:disabled) { opacity: .9; }
.submit:disabled { opacity: .7; cursor: default; }

@media (max-width: 500px) {
  .card {
    width: 100%;
    min-height: 100dvh;
    border-radius: 0;
    border: none;
    box-shadow: none;
    padding: 60px 28px 40px;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
}
</style>
