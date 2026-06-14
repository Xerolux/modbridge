import { ref, onMounted, onUnmounted } from 'vue'

export function useAutoRefresh(fetchFn, intervalMs = 30000) {
  const lastRefreshed = ref(null)
  const isRefreshing = ref(false)
  let timer = null
  let isTabVisible = !document.hidden

  const refreshNow = async () => {
    if (isRefreshing.value) return
    isRefreshing.value = true
    try {
      await fetchFn()
      lastRefreshed.value = new Date()
    } catch {
      // errors handled by the fetchFn itself
    } finally {
      isRefreshing.value = false
    }
  }

  const onVisibilityChange = () => {
    isTabVisible = !document.hidden
    if (isTabVisible) refreshNow()
  }

  const start = () => {
    document.addEventListener('visibilitychange', onVisibilityChange)
    timer = setInterval(() => {
      if (isTabVisible) refreshNow()
    }, intervalMs)
  }

  const stop = () => {
    document.removeEventListener('visibilitychange', onVisibilityChange)
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  onMounted(() => start())
  onUnmounted(() => stop())

  return { lastRefreshed, isRefreshing, refreshNow }
}
