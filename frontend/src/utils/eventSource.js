import { ref, computed } from 'vue';
import axios from 'axios';

export function useEventSource(url, options = {}) {
  const data = ref(null);
  const error = ref(null);
  const isConnected = ref(false);

  const eventSource = ref(null);

  const connect = () => {
    try {
      eventSource.value = new EventSource(url, {
        withCredentials: true,
        ...options,
      });

      eventSource.value.onopen = () => {
        isConnected.value = true;
        error.value = null;
      };

      eventSource.value.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data);
          data.value = parsed;
        } catch (e) {
          data.value = event.data;
        }
      };

      eventSource.value.onerror = (err) => {
        isConnected.value = false;
        error.value = err;
        
        if (eventSource.value.readyState === EventSource.CLOSED) {
          return;
        }

        setTimeout(() => {
          connect();
        }, 5000);
      };
    } catch (err) {
      error.value = err;
      isConnected.value = false;
    }
  };

  const disconnect = () => {
    if (eventSource.value) {
      eventSource.value.close();
      eventSource.value = null;
      isConnected.value = false;
    }
  };

  connect();

  return {
    data,
    error,
    isConnected,
    disconnect,
    reconnect: connect,
  };
}
