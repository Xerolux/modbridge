import { ref, onScopeDispose } from 'vue';
import { EVENT_SOURCE_CONFIG } from './constants';

export function useEventSource(url, options = {}) {
  const data = ref(null);
  const error = ref(null);
  const isConnected = ref(false);

  const eventSource = ref(null);
  let reconnectAttempts = 0;
  let reconnectTimeout = null;
  let manualClose = false; // true when disconnect() is called explicitly

  const connect = () => {
    manualClose = false;

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

        // Don't reconnect if the user explicitly disconnected
        if (manualClose) return;

        // Always reconnect — including when server closes (CLOSED state after 30min timeout)
        const backoffDelay = Math.min(
          EVENT_SOURCE_CONFIG.INITIAL_DELAY * Math.pow(2, reconnectAttempts),
          EVENT_SOURCE_CONFIG.MAX_DELAY
        );
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
    manualClose = true;
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
