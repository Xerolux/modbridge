import { defineStore } from 'pinia';
import { ref } from 'vue';
import axios from '../axios.js';

export const useAppStore = defineStore('app', () => {
  const proxies = ref([]);
  const webPort = ref('');
  const status = ref({
    setup_required: false,
    proxies: []
  });

  const isLoading = ref(false);
  const error = ref(null);
  const darkMode = ref(localStorage.getItem('theme') === 'dark' || (!localStorage.getItem('theme') && window.matchMedia('(prefers-color-scheme: dark)').matches));

  const toggleDarkMode = (value) => {
    // If value is provided, use it; otherwise toggle
    darkMode.value = value !== undefined ? value : !darkMode.value;
    localStorage.setItem('theme', darkMode.value ? 'dark' : 'light');
  };

  const fetchProxies = async () => {
    try {
      const res = await axios.get('/api/proxies');
      // Convert tags array to comma-separated string for editing
      proxies.value = res.data.map(proxy => ({
        ...proxy,
        tags: Array.isArray(proxy.tags) ? proxy.tags.join(', ') : proxy.tags || ''
      }));
    } catch (e) {
      error.value = e.response?.data || e.message;
    }
  };

  const addProxy = async (proxyData) => {
    isLoading.value = true;
    try {
      await axios.post('/api/proxies', proxyData);
      await fetchProxies();
      return true;
    } catch (e) {
      error.value = e.response?.data || e.message;
      return false;
    } finally {
      isLoading.value = false;
    }
  };

  const updateProxy = async (proxyData) => {
    isLoading.value = true;
    try {
      await axios.put('/api/proxies', proxyData);
      await fetchProxies();
      return true;
    } catch (e) {
      error.value = e.response?.data || e.message;
      return false;
    } finally {
      isLoading.value = false;
    }
  };

  const deleteProxy = async (id) => {
    isLoading.value = true;
    try {
      await axios.delete(`/api/proxies?id=${id}`);
      await fetchProxies();
      return true;
    } catch (e) {
      error.value = e.response?.data || e.message;
      return false;
    } finally {
      isLoading.value = false;
    }
  };

  const fetchWebPort = async () => {
    try {
      const res = await axios.get('/api/config/webport');
      webPort.value = res.data.web_port;
    } catch (e) {
      console.error('Failed to fetch web port', e);
    }
  };

  const saveWebPort = async (port) => {
    isLoading.value = true;
    try {
      await axios.put('/api/config/webport', { web_port: port });
      webPort.value = port;
      return true;
    } catch (e) {
      error.value = e.response?.data || e.message;
      return false;
    } finally {
      isLoading.value = false;
    }
  };

  const fetchStatus = async () => {
    try {
      const res = await axios.get('/api/status');
      status.value = res.data;
    } catch (e) {
      console.error('Status fetch failed', e);
    }
  };

  const restartSystem = async () => {
    try {
      await axios.post('/api/system/restart');
      return true;
    } catch (e) {
      error.value = e.response?.data || e.message;
      return false;
    }
  };

  const exportDeviceHistory = async (format = 'json') => {
    try {
      const res = await axios.get(`/api/devices/history?format=${format}`, {
        responseType: 'blob'
      });
      const url = window.URL.createObjectURL(res.data);
      const a = document.createElement('a');
      a.href = url;
      a.download = `device_history.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
      return true;
    } catch (e) {
      error.value = e.response?.data || e.message;
      return false;
    }
  };

  return {
     proxies,
     webPort,
     status,
     isLoading,
     error,
     darkMode,
     toggleDarkMode,
     fetchProxies,
     addProxy,
     updateProxy,
     deleteProxy,
     fetchWebPort,
     saveWebPort,
     fetchStatus,
     restartSystem,
     exportDeviceHistory
   };
});
