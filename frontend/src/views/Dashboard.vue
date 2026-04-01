<template>
  <div class="dashboard-container p-2 sm:p-4 flex flex-col gap-4 w-full overflow-x-hidden">
    <section class="glass-hero dashboard-hero rounded-[28px] p-5 sm:p-6">
      <div class="relative z-[1] flex flex-col gap-5 xl:flex-row xl:items-end xl:justify-between">
        <div class="space-y-3">
          <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.28em] text-[var(--text-muted)]">
            <i class="pi pi-th-large"></i>
            Live Dashboard
          </div>
          <div>
            <h1 class="text-2xl sm:text-3xl font-bold text-gradient">Glass Dashboard</h1>
            <p class="mt-2 max-w-2xl text-sm sm:text-base text-[var(--text-secondary)]">
              Widgets lassen sich frei anordnen. Ziehe Karten im Raster, speichere dein Layout lokal und blende Batch-Konfiguration bei Bedarf direkt daneben ein.
            </p>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div class="dashboard-stat">
            <span class="dashboard-stat-label">Widgets</span>
            <strong class="dashboard-stat-value">{{ widgets.length }}</strong>
          </div>
          <div class="dashboard-stat">
            <span class="dashboard-stat-label">Proxies</span>
            <strong class="dashboard-stat-value">{{ proxies.length }}</strong>
          </div>
          <div class="dashboard-stat">
            <span class="dashboard-stat-label">Running</span>
            <strong class="dashboard-stat-value">{{ runningProxyCount }}</strong>
          </div>
          <div class="dashboard-stat">
            <span class="dashboard-stat-label">Layout</span>
            <strong class="dashboard-stat-value">{{ isMobileLayout ? 'Locked' : 'Drag' }}</strong>
          </div>
        </div>
      </div>
    </section>

    <div class="dashboard-header flex flex-col lg:flex-row justify-between items-start lg:items-center gap-4">
      <div class="space-y-1">
        <h2 class="text-xl font-bold text-[var(--text-primary)]">Workspace</h2>
        <p class="text-sm text-[var(--text-muted)]">
          {{ isMobileLayout ? 'Auf kleinen Displays bleibt das Layout statisch.' : 'Widgets koennen per Drag-and-Drop neu angeordnet werden.' }}
        </p>
      </div>

      <div class="flex flex-wrap gap-2 sm:gap-3 w-full lg:w-auto">
        <Button
          icon="pi pi-cog"
          @click="showConfigPanel = true"
          class="flex-1 sm:flex-none"
          severity="secondary"
          v-tooltip.bottom="'Proxy-Konfiguration'"
        />
        <Button label="Widget hinzufügen" icon="pi pi-plus" @click="openAddWidget" class="flex-1 sm:flex-none" />
        <Button label="Reset" icon="pi pi-refresh" @click="resetLayout" severity="secondary" class="flex-1 sm:flex-none" />
      </div>
    </div>

    <div v-if="loading" class="glass-panel rounded-[28px] p-10 sm:p-12">
      <div class="relative z-[1] flex min-h-[360px] flex-col items-center justify-center text-center">
        <div class="loading-spinner">
          <i class="pi pi-spin pi-spinner text-5xl"></i>
        </div>
        <p class="mt-5 text-[var(--text-secondary)]">Lade Dashboard-Daten und stelle das Grid zusammen...</p>
      </div>
    </div>

    <div v-else-if="error" class="glass-panel rounded-[28px] p-6 sm:p-8">
      <div class="relative z-[1] flex min-h-[360px] flex-col items-center justify-center text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-red-500/15 text-red-300">
          <i class="pi pi-exclamation-triangle text-3xl"></i>
        </div>
        <h3 class="text-xl font-bold text-[var(--text-primary)]">Dashboard konnte nicht geladen werden</h3>
        <p class="mt-2 max-w-lg text-sm text-[var(--text-muted)]">{{ errorMessage }}</p>
        <Button @click="fetchData(true)" label="Erneut versuchen" class="mt-6" />
      </div>
    </div>

    <section v-else class="dashboard-stage glass-panel rounded-[28px] p-3 sm:p-4">
      <div class="relative z-[1]">
        <div class="mb-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex items-center gap-2 text-sm text-[var(--text-muted)]">
            <span class="status-dot" :class="isMobileLayout ? 'status-dot--unknown' : 'status-dot--running'"></span>
            <span>{{ isMobileLayout ? 'Mobile-Layout aktiv' : 'Drag-and-drop aktiv' }}</span>
          </div>
          <div class="text-xs uppercase tracking-[0.22em] text-[var(--text-muted)]">
            {{ layoutHint }}
          </div>
        </div>

        <div
          v-if="widgets.length === 0 && !grid"
          class="empty-grid rounded-[24px] border border-dashed border-white/15 p-10 text-center"
        >
          <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-white/5">
            <i class="pi pi-plus-circle text-2xl text-[var(--text-secondary)]"></i>
          </div>
          <h3 class="text-xl font-semibold text-[var(--text-primary)]">Noch keine Widgets auf dem Board</h3>
          <p class="mx-auto mt-2 max-w-md text-sm text-[var(--text-muted)]">
            Fuege einen Proxy als Widget hinzu und ziehe die Karten anschliessend an die gewuenschte Position.
          </p>
        </div>

        <div
          class="grid-stack-dashboard grid-stack min-h-[60vh] sm:min-h-[520px] rounded-[24px] border border-white/10"
          :class="{ 'grid-stack-dashboard--editing': layoutEditing, 'hidden': widgets.length === 0 }"
        >
          <div v-if="layoutEditing" class="layout-edit-banner">
            <i class="pi pi-arrows-alt"></i>
            Layout wird neu angeordnet
          </div>

          <Teleport v-for="widget in widgets" :key="widget.id" :to="'#mount_' + widget.id">
            <div class="relative h-full w-full p-2 sm:p-3 flex">
              <DashboardWidget
                :title="widget.title"
                :value="getWidgetValue(widget)"
                :unit="widget.unit"
                :status="getWidgetStatus(widget)"
                :active-connections="getWidgetConnections(widget)"
              />
              <button
                type="button"
                class="widget-remove"
                @click="removeWidget(widget.id)"
                aria-label="Widget entfernen"
                title="Widget entfernen"
              >
                <i class="pi pi-times text-sm"></i>
              </button>
            </div>
          </Teleport>
        </div>
      </div>
    </section>

    <Dialog v-model:visible="showAddWidget" header="Widget hinzufügen" :modal="true" class="w-11/12 sm:w-full max-w-[440px]">
      <div class="flex flex-col gap-4">
        <p class="text-sm text-[var(--text-muted)]">
          Verfuegbare Proxies koennen als Widgets auf das Board gelegt und danach frei positioniert werden.
        </p>
        <Dropdown
          v-model="selectedProxy"
          :options="availableProxyOptions"
          optionLabel="name"
          optionValue="id"
          placeholder="Waehle einen Proxy"
          filter
          class="w-full"
        />
        <Button label="Hinzufügen" @click="confirmAddWidget" :disabled="!selectedProxy" class="w-full" />
      </div>
    </Dialog>

    <ProxyConfigPanel
      :visible="showConfigPanel"
      :proxies="proxies"
      @close="showConfigPanel = false"
      @refresh="fetchData"
    />
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { GridStack } from 'gridstack';
import 'gridstack/dist/gridstack.min.css';
import axios from '../axios.js';
import Button from 'primevue/button';
import Dialog from 'primevue/dialog';
import Dropdown from 'primevue/dropdown';
import DashboardWidget from '../components/DashboardWidget.vue';
import ProxyConfigPanel from '../components/ProxyConfigPanel.vue';
import { useEventSource } from '../utils/eventSource';
import { BREAKPOINTS, DASHBOARD_CONFIG, GRID_CONFIG } from '../utils/constants';
import { debounce, formatNumber } from '../utils/helpers';

const STORAGE_KEY = 'dashboard_layout_v2';

const grid = ref(null);
const proxies = ref([]);
const widgets = ref([]);
const showAddWidget = ref(false);
const showConfigPanel = ref(false);
const selectedProxy = ref(null);
const loading = ref(true);
const error = ref(null);
const errorMessage = ref('');
const layoutEditing = ref(false);
const isMobileLayout = ref(window.innerWidth <= BREAKPOINTS.MOBILE);
let sseDisconnect = null;

const runningProxyCount = computed(() => proxies.value.filter(proxy => proxy.status === 'Running').length);
const availableProxyOptions = computed(() => {
  const usedIds = new Set(widgets.value.map(widget => widget.proxy_id));
  return proxies.value
    .filter(proxy => !usedIds.has(proxy.id))
    .map(proxy => ({ name: proxy.name, id: proxy.id }));
});
const layoutHint = computed(() => {
  if (isMobileLayout.value) return 'Touch first';
  if (layoutEditing.value) return 'Drop zone open';
  return 'Drag widgets';
});

const buildDefaultLayout = (proxyList) => proxyList.map((proxy, index) => ({
  x: (index % 3) * 2,
  y: Math.floor(index / 3) * 2,
  w: 2,
  h: 2,
  id: `w_default_${proxy.id}`,
  proxy_id: proxy.id,
  title: proxy.name,
  unit: 'req',
  status: proxy.status
}));

const getStoredLayout = () => {
  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) return [];

  try {
    return JSON.parse(raw);
  } catch (parseError) {
    console.error(parseError);
    return [];
  }
};

const syncGridInteractivity = () => {
  if (!grid.value) return;

  if (isMobileLayout.value) {
    grid.value.enableMove(false);
    grid.value.enableResize(false);
    grid.value.setStatic(true);
  } else {
    grid.value.setStatic(false);
    grid.value.enableMove(true);
    grid.value.enableResize(true);
  }
};

const initializeGrid = () => {
  grid.value = GridStack.init({
    float: DASHBOARD_CONFIG.FLOATING,
    cellHeight: DASHBOARD_CONFIG.MIN_WIDGET_WIDTH,
    minRow: GRID_CONFIG.MIN_ROW,
    margin: GRID_CONFIG.MARGIN,
    column: 6,
    disableDrag: isMobileLayout.value,
    disableResize: isMobileLayout.value,
    columnOpts: {
      breakpoints: [
        { w: 640, c: 1 },
        { w: 768, c: 2 },
        { w: 1024, c: 3 },
        { w: 1280, c: 4 },
        { w: 1536, c: 6 }
      ]
    }
  }, '.grid-stack-dashboard');

  if (!grid.value) return;

  syncGridInteractivity();

  grid.value.on('change', saveLayout);
  grid.value.on('dragstart', () => {
    layoutEditing.value = true;
  });
  grid.value.on('dragstop', () => {
    layoutEditing.value = false;
    saveLayout();
  });
  grid.value.on('resizestart', () => {
    layoutEditing.value = true;
  });
  grid.value.on('resizestop', () => {
    layoutEditing.value = false;
    saveLayout();
  });

  const currentWidgets = [...widgets.value];
  widgets.value = [];
  currentWidgets.forEach(item => addWidgetToGrid(item));
};

onMounted(async () => {
  try {
    await fetchData(true);

    let layoutToLoad = getStoredLayout();
    if (!layoutToLoad.length) {
      layoutToLoad = buildDefaultLayout(proxies.value);
    }

    loadGrid(layoutToLoad);
    await nextTick();
    initializeGrid();
    window.addEventListener('resize', handleResize);

    const { data, disconnect } = useEventSource('/api/proxies/stream');
    sseDisconnect = disconnect;

    watch(data, (eventData) => {
      if (!eventData) return;

      const proxyData = eventData.proxy;

      switch (eventData.type) {
        case 'proxy_added':
        case 'proxy_updated':
        case 'proxy_started':
        case 'proxy_stopped':
          if (!proxyData) return;
          updateProxyCollection(proxyData);
          break;
        case 'proxy_removed':
          if (!eventData.proxy_id) return;
          proxies.value = proxies.value.filter(proxy => proxy.id !== eventData.proxy_id);
          widgets.value = widgets.value.filter(widget => widget.proxy_id !== eventData.proxy_id);
          saveLayout();
          break;
      }
    });
  } catch (err) {
    error.value = true;
    errorMessage.value = err.message || 'Fehler beim Initialisieren des Dashboards';
    loading.value = false;
  }
});

const handleResize = debounce(() => {
  isMobileLayout.value = window.innerWidth <= BREAKPOINTS.MOBILE;
  syncGridInteractivity();
}, 150);

onUnmounted(() => {
  window.removeEventListener('resize', handleResize);
  if (grid.value) {
    grid.value.destroy(false);
  }
  if (sseDisconnect) {
    sseDisconnect();
  }
});

const updateProxyCollection = (proxyData) => {
  const index = proxies.value.findIndex(proxy => proxy.id === proxyData.id);
  if (index >= 0) {
    proxies.value[index] = proxyData;
    return;
  }

  proxies.value.push(proxyData);
};

const loadGrid = (layout) => {
  widgets.value = [];
  if (grid.value) {
    grid.value.removeAll();
  }
  layout.forEach(item => {
    const id = item.id || `w_${item.proxy_id || Date.now()}`;
    widgets.value.push({ ...item, id });
  });
};

const addWidgetToGrid = (item) => {
  if (!grid.value) return;

  const id = item.id || `w_${item.proxy_id || Date.now()}`;
  const contentHtml = `<div id="mount_${id}" class="h-full w-full relative"></div>`;
  const el = grid.value.addWidget({
    x: item.x ?? 0,
    y: item.y ?? 0,
    w: item.w ?? 2,
    h: item.h ?? 2,
    id,
    content: ''
  });

  if (el) {
    const contentEl = el.querySelector('.grid-stack-item-content');
    if (contentEl) {
      contentEl.innerHTML = contentHtml;
    }
  }

  nextTick(() => {
    widgets.value.push({ ...item, id });
  });
};

const saveLayout = () => {
  if (!grid.value) return;

  const items = grid.value.getGridItems();
  const layout = items
    .filter((item) => item.gridstackNode)
    .map((item) => {
      const widget = widgets.value.find(entry => entry.id === item.gridstackNode.id);
      return {
        x: item.gridstackNode.x,
        y: item.gridstackNode.y,
        w: item.gridstackNode.w,
        h: item.gridstackNode.h,
        id: item.gridstackNode.id,
        proxy_id: widget?.proxy_id || '',
        title: widget?.title || '',
        unit: widget?.unit || '',
        status: widget?.status || ''
      };
    });

  localStorage.setItem(STORAGE_KEY, JSON.stringify(layout));
};

const fetchData = async (isInitial = false) => {
  try {
    if (isInitial) loading.value = true;
    error.value = null;
    errorMessage.value = '';
    const res = await axios.get('/api/proxies');
    proxies.value = res.data;
    if (isInitial) loading.value = false;
  } catch (requestError) {
    const errorData = requestError.response?.data;
    error.value = true;
    errorMessage.value = typeof errorData === 'string' ? errorData : requestError.message || 'Unbekannter Fehler';
    if (isInitial) loading.value = false;
    throw requestError;
  }
};

const getWidgetValue = (widget) => {
  const proxy = proxies.value.find(entry => entry.id === widget.proxy_id);
  if (!proxy) return 'Unbekannt';
  if (proxy.status === 'Running') {
    return `${formatNumber(proxy.requests || 0)} Anfragen`;
  }
  return proxy.status;
};

const getWidgetConnections = (widget) => {
  const proxy = proxies.value.find(entry => entry.id === widget.proxy_id);
  return proxy ? proxy.active_connections ?? 0 : null;
};

const getWidgetStatus = (widget) => {
  const proxy = proxies.value.find(entry => entry.id === widget.proxy_id);
  return proxy ? proxy.status : widget.status || 'Unbekannt';
};

const openAddWidget = () => {
  selectedProxy.value = availableProxyOptions.value[0]?.id || null;
  showAddWidget.value = true;
};

const confirmAddWidget = () => {
  if (!selectedProxy.value) return;

  const existing = widgets.value.find(widget => widget.proxy_id === selectedProxy.value);
  if (existing) {
    showAddWidget.value = false;
    selectedProxy.value = null;
    return;
  }

  const proxy = proxies.value.find(entry => entry.id === selectedProxy.value);
  if (!proxy) return;

  addWidgetToGrid({
    x: 0,
    y: 0,
    w: 2,
    h: 2,
    proxy_id: selectedProxy.value,
    title: proxy.name,
    unit: 'req',
    status: proxy.status
  });

  showAddWidget.value = false;
  selectedProxy.value = null;
  saveLayout();
};

const removeWidget = (id) => {
  const mount = document.getElementById(`mount_${id}`);
  if (mount) {
    const gridItem = mount.closest('.grid-stack-item');
    if (gridItem && grid.value) {
      grid.value.removeWidget(gridItem);
    }
  }

  widgets.value = widgets.value.filter(widget => widget.id !== id);
  saveLayout();
};

const resetLayout = () => {
  localStorage.removeItem(STORAGE_KEY);
  loadGrid(buildDefaultLayout(proxies.value));
  saveLayout();
};
</script>

<style scoped>
.dashboard-container {
  background: transparent;
  min-height: 100vh;
}

.dashboard-hero,
.dashboard-stage,
.dashboard-stat {
  position: relative;
  overflow: hidden;
}

.dashboard-stat {
  border-radius: 20px;
  padding: 0.9rem 1rem;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.dashboard-stat-label {
  display: block;
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.2em;
  color: var(--text-muted);
}

.dashboard-stat-value {
  display: block;
  margin-top: 0.45rem;
  font-size: 1.2rem;
  font-weight: 800;
  color: var(--text-primary);
}

.loading-spinner {
  position: relative;
  display: flex;
  height: 5.5rem;
  width: 5.5rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
}

.loading-spinner::before {
  content: '';
  position: absolute;
  inset: -6px;
  border-radius: 999px;
  border: 1px solid rgba(125, 211, 252, 0.25);
}

.grid-stack-dashboard {
  position: relative;
  padding: 0.35rem;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), rgba(255, 255, 255, 0.015)),
    rgba(15, 23, 42, 0.18);
}

.grid-stack-dashboard--editing {
  border-color: rgba(125, 211, 252, 0.28);
}

.layout-edit-banner {
  position: absolute;
  top: 0.9rem;
  right: 0.9rem;
  z-index: 30;
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  border-radius: 999px;
  border: 1px solid rgba(125, 211, 252, 0.18);
  background: rgba(15, 23, 42, 0.7);
  padding: 0.45rem 0.8rem;
  font-size: 0.78rem;
  color: var(--text-secondary);
}

.widget-remove {
  position: absolute;
  top: 0.8rem;
  right: 0.8rem;
  z-index: 20;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border: 0;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.72);
  color: var(--text-secondary);
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease, transform 0.2s ease;
}

.widget-remove:hover {
  background: rgba(244, 63, 94, 0.22);
  color: #fecdd3;
  transform: scale(1.05);
}

.empty-grid {
  background: rgba(255, 255, 255, 0.03);
}

.grid-stack-item-content {
  height: 100%;
  border-radius: 24px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.1), rgba(255, 255, 255, 0.03)),
    rgba(15, 23, 42, 0.55);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 18px 45px rgba(2, 6, 23, 0.35);
  overflow: hidden;
  transition: transform 0.25s ease, box-shadow 0.25s ease, border-color 0.25s ease;
}

.grid-stack-item:hover .grid-stack-item-content {
  transform: translateY(-2px);
  border-color: rgba(125, 211, 252, 0.22);
  box-shadow: 0 28px 60px rgba(2, 6, 23, 0.42);
}

@media (max-width: 640px) {
  .grid-stack-item-content {
    border-radius: 18px;
  }

  .layout-edit-banner {
    left: 0.75rem;
    right: 0.75rem;
    justify-content: center;
  }
}
</style>
