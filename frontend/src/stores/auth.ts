import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

interface TokenResponse {
  access_token: string
  csrf_token: string
}

// The access token lives only in memory. The CSRF token and the username are
// kept in localStorage so a page reload can restore the session through the
// httpOnly refresh cookie.
const CSRF_KEY = 'csrf_token'
const USERNAME_KEY = 'username'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const csrfToken = ref<string | null>(localStorage.getItem(CSRF_KEY))
  const username = ref<string | null>(localStorage.getItem(USERNAME_KEY))

  const isAuthenticated = computed(() => accessToken.value !== null)
  const canRefresh = computed(() => csrfToken.value !== null)

  function setTokens(data: TokenResponse) {
    accessToken.value = data.access_token
    csrfToken.value = data.csrf_token
    localStorage.setItem(CSRF_KEY, data.csrf_token)
  }

  function clear() {
    accessToken.value = null
    csrfToken.value = null
    username.value = null
    localStorage.removeItem(CSRF_KEY)
    localStorage.removeItem(USERNAME_KEY)
  }

  async function login(user: string, password: string): Promise<void> {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: user, password }),
    })
    if (!res.ok) {
      if (res.status === 401) throw new Error('Неверный логин или пароль')
      if (res.status === 429) throw new Error('Слишком много попыток. Подождите немного.')
      throw new Error('Не удалось войти. Попробуйте ещё раз.')
    }
    setTokens((await res.json()) as TokenResponse)
    username.value = user
    localStorage.setItem(USERNAME_KEY, user)
  }

  async function doRefresh(): Promise<boolean> {
    if (!csrfToken.value) return false
    const res = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'X-CSRF-Token': csrfToken.value },
    }).catch(() => null)
    if (!res?.ok) {
      clear()
      return false
    }
    setTokens((await res.json()) as TokenResponse)
    return true
  }

  // Refresh tokens are single-use, so concurrent 401s must share one request:
  // a second refresh with the same cookie would be treated as token reuse.
  let pending: Promise<boolean> | null = null
  function refresh(): Promise<boolean> {
    pending ??= doRefresh().finally(() => {
      pending = null
    })
    return pending
  }

  async function logout(): Promise<void> {
    await fetch('/api/v1/auth/logout', {
      method: 'POST',
      headers: csrfToken.value ? { 'X-CSRF-Token': csrfToken.value } : {},
    }).catch(() => {})
    clear()
  }

  return { accessToken, username, isAuthenticated, canRefresh, login, refresh, logout }
})
