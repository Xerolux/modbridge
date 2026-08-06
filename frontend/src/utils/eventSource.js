import { ref, onScopeDispose } from 'vue';
import { EVENT_SOURCE_CONFIG } from './constants';

export function useEventSource(url, options = {}) {
  const data = ref(null);
  const error = ref(null);
  const isConnected = ref(false);
  // Timestamp of the last received message; consumers use it (together with
  // isConnected) to detect stale connections and trigger a polling fallback.
  const lastMessageAt = ref(0);

  const eventSource = ref(null);
  let reconnectAttempts = 0;
  let reconnectTimeout = null;
  let manualClose = false; // true when disconnect() is called explicitly
  let reconnecting = false;
  const maxReconnectAttempts = EVENT_SOURCE_CONFIG.MAX_RECONNECT_ATTEMPTS;

  const connect = () => {
    manualClose = false;
    reconnecting = false;

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
        lastMessageAt.value = Date.now();
        error.value = null;
        reconnectAttempts = 0;
      };

      eventSource.value.addEventListener('heartbeat', () => {
        isConnected.value = true;
        lastMessageAt.value = Date.now();
      });

      eventSource.value.onmessage = (event) => {
        if (!isConnected.value) {
          isConnected.value = true;
        }
        lastMessageAt.value = Date.now();
        try {
          const parsed = JSON.parse(event.data);
          data.value = parsed;
        } catch {
          data.value = event.data;
        }
      };

      eventSource.value.onerror = (err) => {
        isConnected.value = false;

        // Don't reconnect if the user explicitly disconnected
        if (manualClose) return;
        if (reconnecting) return;

        // EventSource retries automatically unless it is closed. We own the
        // retry schedule below so only one connection attempt can be active.
        if (eventSource.value) {
          eventSource.value.close();
          eventSource.value = null;
        }

        // Stop retrying after max attempts to prevent infinite loops
        if (reconnectAttempts >= maxReconnectAttempts) {
          error.value = new Error(`SSE: Max reconnect attempts (${maxReconnectAttempts}) reached`);
          return;
        }

        // Give up on authentication/authorization errors; they will not recover by reconnecting
        if (err?.target?.readyState === EventSource.CLOSED && (err?.target?.status === 401 || err?.target?.status === 403)) {
          disconnect();
          return;
        }

        // Exponential backoff with ±30% jitter — prevents thundering herd
        // when many clients reconnect simultaneously after a server restart
        const base = Math.min(
          EVENT_SOURCE_CONFIG.INITIAL_DELAY * Math.pow(2, reconnectAttempts),
          EVENT_SOURCE_CONFIG.MAX_DELAY
        );
        const jitter = (Math.random() * 0.6 - 0.3) * base;
        const backoffDelay = Math.max(100, base + jitter);
        reconnectAttempts++;
        reconnecting = true;

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
    reconnecting = false;
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
    lastMessageAt,
    disconnect,
    reconnect: connect,
  };
}
