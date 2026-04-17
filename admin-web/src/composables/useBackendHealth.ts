import { onBeforeUnmount, onMounted, readonly, ref } from 'vue'
import * as adminApi from '@/api/admin'
import { ApiError } from '@/api/http'
import type { ServerHealthStatus } from '@/types'

export type BackendHealthState =
  | 'unknown'
  | 'online'
  | 'degraded'
  | 'critical'
  | 'offline'

const POLL_INTERVAL_MS = 30_000

const state = ref<BackendHealthState>('unknown')
const lastCheckedAt = ref<string>('')

let subscriberCount = 0
let timerId: ReturnType<typeof setInterval> | null = null
let inflight: Promise<void> | null = null

function mapStatus(status: ServerHealthStatus | undefined): BackendHealthState {
  switch (status) {
    case 'healthy':
      return 'online'
    case 'warning':
      return 'degraded'
    case 'critical':
      return 'critical'
    default:
      return 'unknown'
  }
}

async function probe() {
  if (inflight) {
    return inflight
  }
  inflight = (async () => {
    try {
      const { overview } = await adminApi.getServerHealthOverview()
      state.value = mapStatus(overview.summary.status)
      lastCheckedAt.value = overview.generatedAt || new Date().toISOString()
    } catch (error) {
      // 401 仅代表会话失效，路由守卫会接手跳转；这里退到 unknown，避免误报“故障”。
      if (error instanceof ApiError && error.status === 401) {
        state.value = 'unknown'
      } else {
        state.value = 'offline'
      }
      lastCheckedAt.value = new Date().toISOString()
    } finally {
      inflight = null
    }
  })()
  return inflight
}

function start() {
  if (typeof window === 'undefined' || timerId !== null) {
    return
  }
  void probe()
  timerId = setInterval(() => {
    void probe()
  }, POLL_INTERVAL_MS)
}

function stop() {
  if (timerId !== null) {
    clearInterval(timerId)
    timerId = null
  }
}

export function useBackendHealth() {
  onMounted(() => {
    subscriberCount += 1
    if (subscriberCount === 1) {
      start()
    }
  })

  onBeforeUnmount(() => {
    subscriberCount = Math.max(subscriberCount - 1, 0)
    if (subscriberCount === 0) {
      stop()
    }
  })

  return {
    state: readonly(state),
    lastCheckedAt: readonly(lastCheckedAt),
    refresh: () => probe()
  }
}
