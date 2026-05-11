import router from '@/router'
import { useAuthStore } from '@/stores/auth'
import type { ChatMessage } from '@/types/api'

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message)
  }
}

function messageForStatus(status: number): string {
  if (status === 401) return 'Сессия истекла, войдите снова'
  if (status === 404) return 'Не найдено'
  if (status === 429) return 'Слишком много запросов, подождите немного'
  if (status >= 500) return 'Сервер недоступен, попробуйте позже'
  return `Ошибка запроса (${status})`
}

export function useApi() {
  const auth = useAuthStore()

  async function request(path: string, init: RequestInit = {}): Promise<Response> {
    const send = () =>
      fetch(path, {
        ...init,
        headers: { ...init.headers, Authorization: `Bearer ${auth.accessToken ?? ''}` },
      })

    let res = await send()
    if (res.status === 401) {
      if (!(await auth.refresh())) {
        await router.replace({ name: 'login' })
        throw new ApiError(401, messageForStatus(401))
      }
      res = await send()
    }
    if (!res.ok) throw new ApiError(res.status, messageForStatus(res.status))
    return res
  }

  async function get<T>(path: string): Promise<T> {
    const res = await request(path)
    return (await res.json()) as T
  }

  async function post<T = void>(path: string, body: unknown): Promise<T> {
    const res = await request(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (res.status === 204) return undefined as T
    return (await res.json()) as T
  }

  // Yields text chunks of the tutor's answer as they arrive over SSE.
  async function* streamChat(lessonId: number, history: ChatMessage[]): AsyncGenerator<string> {
    const res = await request(`/api/v1/lessons/${lessonId}/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ history }),
    })
    if (!res.body) throw new Error('Пустой ответ сервера')

    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buf = ''
    let event = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) return
      buf += decoder.decode(value, { stream: true })
      const lines = buf.split('\n')
      buf = lines.pop() ?? ''

      for (const line of lines) {
        if (line.startsWith('event: ')) {
          event = line.slice('event: '.length)
          continue
        }
        if (!line.startsWith('data: ')) continue
        if (event === 'done') return
        if (event === 'error') throw new Error('Тутор сейчас недоступен, попробуйте позже')

        const data = JSON.parse(line.slice('data: '.length)) as { text?: string }
        if (data.text) yield data.text
      }
    }
  }

  return { get, post, streamChat }
}
