import { ref, onMounted, onUnmounted } from 'vue';

export function useAutoRefresh(fetchFn, intervalMs = 30000) {
  const lastRefreshed = ref(null);
  const isRefreshing = ref(false);
  const refreshError = ref(false);
  let timer;
  let controller;
  let disposed = false;
  const refreshNow = async () => {
    if (disposed || isRefreshing.value) return;
    isRefreshing.value = true;
    try {
      controller = new AbortController();
      const result = await fetchFn({ signal: controller.signal });
      if (disposed) return;
      refreshError.value = result === false;
      if (result !== false) lastRefreshed.value = new Date();
    } catch {
      if (!disposed) refreshError.value = true;
    } finally {
      if (!disposed) isRefreshing.value = false;
    }
  };
  const resume = () => { if (!document.hidden && navigator.onLine) refreshNow(); };
  onMounted(() => {
    document.addEventListener('visibilitychange', resume);
    window.addEventListener('online', resume);
    timer = setInterval(resume, intervalMs);
  });
  onUnmounted(() => {
    disposed = true;
    controller?.abort();
    clearInterval(timer);
    document.removeEventListener('visibilitychange', resume);
    window.removeEventListener('online', resume);
  });
  return { lastRefreshed, isRefreshing, refreshError, refreshNow };
}
