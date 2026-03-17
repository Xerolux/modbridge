import { ref, onScopeDispose } from 'vue';

export function useEventSource(url, options = {}) {
  const data = ref(null);
  const error = ref(null);
  const isConnected = ref(false);

  const eventSource = ref(null);
  let reconnectAttempts = 0;
  let reconnectTimeout = null;

  const connect = () => {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }

    if (eventSource.value) {
      eventSource.value.close();
      eventSource.value = null;
    }

    try {
      eventSource.value = new EventSource(url, {
        withCredentials: true,
        ...options,
      });

      eventSource.value.onopen = () => {
        isConnected.value = true;
        error.value = null;
        reconnectAttempts = 0;
      };

      eventSource.value.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data);
          data.value = parsed;
        } catch {
          data.value = event.data;
        }
      };

      eventSource.value.onerror = () => {
        isConnected.value = false;

        if (eventSource.value?.readyState === EventSource.CLOSED) {
          return;
        }

        const backoffDelay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
        reconnectAttempts++;

        reconnectTimeout = setTimeout(() => {
          connect();
        }, backoffDelay);
      };
    } catch (err) {
      error.value = err;
      isConnected.value = false;
    }
  };

  const disconnect = () => {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }
    if (eventSource.value) {
      eventSource.value.close();
      eventSource.value = null;
    }
    isConnected.value = false;
    reconnectAttempts = 0;
  };

  connect();

  if (typeof onScopeDispose === 'function') {
    onScopeDispose(() => {
      disconnect();
    });
  }

  return {
    data,
    error,
    isConnected,
    disconnect,
    reconnect: connect,
  };
}
