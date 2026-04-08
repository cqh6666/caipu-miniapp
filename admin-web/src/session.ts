import { ref } from 'vue'
import * as adminApi from '@/api/admin'
import { ApiError } from '@/api/http'

export const currentUsername = ref('')
export const sessionReady = ref(false)

export async function bootstrapSession(force = false) {
  if (sessionReady.value && !force) {
    return currentUsername.value
  }

  try {
    const data = await adminApi.getMe()
    currentUsername.value = data.username
    sessionReady.value = true
    return currentUsername.value
  } catch (error) {
    currentUsername.value = ''
    if (error instanceof ApiError && error.status === 401) {
      sessionReady.value = true
      return ''
    }
    throw error
  }
}

export function markLoggedIn(username: string) {
  currentUsername.value = username
  sessionReady.value = true
}

export function markLoggedOut() {
  currentUsername.value = ''
  sessionReady.value = true
}
