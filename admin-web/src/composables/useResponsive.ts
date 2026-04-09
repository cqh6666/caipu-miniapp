import { computed, onBeforeUnmount, onMounted, readonly, ref } from 'vue'

const viewportWidth = ref(typeof window === 'undefined' ? 1440 : window.innerWidth)

let subscriberCount = 0

function updateViewportWidth() {
  if (typeof window === 'undefined') {
    return
  }
  viewportWidth.value = window.innerWidth
}

function startViewportTracking() {
  if (typeof window === 'undefined' || subscriberCount !== 1) {
    return
  }
  updateViewportWidth()
  window.addEventListener('resize', updateViewportWidth, { passive: true })
}

function stopViewportTracking() {
  if (typeof window === 'undefined' || subscriberCount !== 0) {
    return
  }
  window.removeEventListener('resize', updateViewportWidth)
}

export function useResponsive() {
  onMounted(() => {
    subscriberCount += 1
    startViewportTracking()
  })

  onBeforeUnmount(() => {
    subscriberCount = Math.max(subscriberCount - 1, 0)
    stopViewportTracking()
  })

  const width = readonly(viewportWidth)

  return {
    width,
    isCompactLayout: computed(() => width.value <= 992),
    isSingleColumnContent: computed(() => width.value <= 1200),
    isMobile: computed(() => width.value <= 768)
  }
}
