import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import axios from '../axios.js';

export const THEMES = ['light', 'dark', 'bw'];
export const ACCENTS = ['sky', 'violet', 'emerald', 'amber', 'rose', 'mono'];

const readStoredTheme = () => {
  const stored = localStorage.getItem('modbridge_theme');
  if (stored === 'dark' || stored === 'light' || stored === 'bw') return stored;
  // Migrate legacy boolean key
  const legacy = localStorage.getItem('theme');
  if (legacy === 'dark') return 'dark';
  if (legacy === 'light') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
};

export const useAppStore = defineStore('app', () => {
  const proxies = ref([]);
  const webPort = ref('');
  const status = ref({
    setup_required: false,
    proxies: []
  });

  const isLoading = ref(false);
  const error = ref(null);

  // ── Theme system ────────────────────────────────────────────────
  // theme: 'light' | 'dark' | 'bw'  (bw = monochrome high-contrast)
  // accent: sky | violet | emerald | amber | rose | mono
  // density: 'comfortable' | 'compact'
  // reducedMotion: boolean — disables ambient animations for speed/battery
  const theme = ref(readStoredTheme());
  const accent = ref(localStorage.getItem('modbridge_accent') || 'sky');
  const density = ref(localStorage.getItem('modbridge_density') || 'comfortable');
  const reducedMotion = ref(
    localStorage.getItem('modbridge_reduced_motion') === 'true' ||
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  );

  // Backward-compatible boolean — true when in dark or bw mode (PrimeVue dark selector)
  const darkMode = computed(() => theme.value === 'dark' || theme.value === 'bw');

  const setTheme = (value) => {
    if (!THEMES.includes(value)) return;
    theme.value = value;
    localStorage.setItem('modbridge_theme', value);
    localStorage.removeItem('theme');
  };

  const setAccent = (value) => {
    if (!ACCENTS.includes(value)) return;
    accent.value = value;
    localStorage.setItem('modbridge_accent', value);
  };

  const setDensity = (value) => {
    if (value !== 'comfortable' && value !== 'compact') return;
    density.value = value;
    localStorage.setItem('modbridge_density', value);
  };

  const toggleReducedMotion = (value) => {
    reducedMotion.value = value !== undefined ? value : !reducedMotion.value;
    localStorage.setItem('modbridge_reduced_motion', String(reducedMotion.value));
  };

  // Legacy toggle kept for compatibility — cycles light/dark
  const toggleDarkMode = (value) => {
    if (value !== undefined) {
      setTheme(value ? 'dark' : 'light');
    } else {
      setTheme(theme.value === 'dark' ? 'light' : 'dark');
    }
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
      // Full refetch needed to get the server-assigned ID and canonical state
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
    const index = proxies.value.findIndex(p => p.id === proxyData.id);
    const snapshot = index >= 0 ? { ...proxies.value[index] } : null;
    // Optimistic update — apply immediately, rollback on error
    if (index >= 0) {
      proxies.value[index] = { ...proxies.value[index], ...proxyData };
    }
    try {
      await axios.put('/api/proxies', proxyData);
      return true;
    } catch (e) {
      if (index >= 0 && snapshot) proxies.value[index] = snapshot;
      error.value = e.response?.data || e.message;
      return false;
    } finally {
      isLoading.value = false;
    }
  };

  const deleteProxy = async (id) => {
    isLoading.value = true;
    const index = proxies.value.findIndex(p => p.id === id);
    const snapshot = index >= 0 ? { ...proxies.value[index] } : null;
    // Optimistic removal — remove immediately, rollback on error
    if (index >= 0) proxies.value.splice(index, 1);
    try {
      await axios.delete(`/api/proxies?id=${id}`);
      return true;
    } catch (e) {
      if (index >= 0 && snapshot) proxies.value.splice(index, 0, snapshot);
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
     theme,
     accent,
     density,
     reducedMotion,
     darkMode,
     toggleDarkMode,
     setTheme,
     setAccent,
     setDensity,
     toggleReducedMotion,
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
