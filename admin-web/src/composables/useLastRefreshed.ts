import { computed, reactive } from 'vue'

const store = reactive<Record<string, number>>({})

export function useLastRefreshed(key: string) {
  const timestamp = computed(() => store[key] ?? 0)
  const display = computed(() => {
    const ts = timestamp.value
    if (!ts) return ''
    return new Date(ts).toLocaleTimeString('zh-CN', { hour12: false })
  })
  const mark = () => {
    store[key] = Date.now()
  }
  return { timestamp, display, mark }
}
