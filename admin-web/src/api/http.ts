import type { ApiEnvelope } from '@/types'

const API_BASE = import.meta.env.VITE_API_BASE ?? ''

export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export async function request<T>(url: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers || {})
  const isJsonBody = init.body && !(init.body instanceof FormData)
  if (isJsonBody && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(`${API_BASE}${url}`, {
    credentials: 'include',
    ...init,
    headers
  })

  const payload = (await response.json().catch(() => null)) as ApiEnvelope<T> | null
  if (!response.ok || !payload || payload.code !== 0) {
    throw new ApiError(payload?.message || '请求失败', response.status)
  }

  return payload.data
}
