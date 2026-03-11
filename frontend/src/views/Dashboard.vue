<template>
    <div class="dashboard-container p-2 sm:p-4 flex flex-col gap-4 w-full overflow-x-hidden">
        <!-- Header -->
        <div class="dashboard-header flex flex-col lg:flex-row justify-between items-start lg:items-center mb-2 sm:mb-6 gap-4 lg:gap-0">
             <div class="flex items-center gap-3 sm:gap-4">
                 <div class="header-icon">
                     <i class="pi pi-th-large text-xl sm:text-2xl"></i>
                 </div>
                 <h1 class="text-lg sm:text-2xl lg:text-3xl font-bold bg-gradient-to-r from-blue-400 to-purple-500 bg-clip-text text-transparent">
                     Dashboard
                 </h1>
             </div>
             <div class="flex flex-wrap sm:flex-nowrap gap-2 sm:gap-3 w-full lg:w-auto">
                  <!-- Settings Button -->
                  <Button
                    icon="pi pi-cog"
                    @click="showConfigPanel = true"
                    class="flex-1 sm:flex-none animate-glow p-button-sm sm:p-button-md"
                    v-tooltip.bottom="'Proxy-Konfiguration'"
                    severity="secondary"
                  />
                  <Button label="Hinzufügen" icon="pi pi-plus" @click="openAddWidget" class="flex-1 sm:flex-none p-button-sm sm:p-button-md" />
                  <Button label="Reset" icon="pi pi-refresh" @click="resetLayout" severity="secondary" class="flex-1 sm:flex-none p-button-sm sm:p-button-md" />
             </div>
        </div>

        <!-- Loading State -->
        <div v-if="loading" class="flex justify-center items-center min-h-[500px]">
             <div class="text-center loading-container">
                  <div class="loading-spinner">
                      <i class="pi pi-spin pi-spinner text-5xl"></i>
                  </div>
                  <p class="mt-4 text-gray-400 animate-pulse">Lade Dashboard...</p>
             </div>
        </div>

        <!-- Error State -->
        <div v-else-if="error" class="flex justify-center items-center min-h-[500px]">
             <div class="text-center error-container glass-effect rounded-2xl p-4 sm:p-8 mx-2 sm:mx-0 max-w-[95vw] sm:max-w-lg">
                  <div class="error-icon mb-4">
                      <i class="pi pi-exclamation-triangle text-5xl text-red-500 animate-shake"></i>
                  </div>
                  <p class="mt-4 text-red-400 font-semibold text-xs sm:text-base break-words whitespace-normal overflow-hidden">
                      Fehler beim Laden: <span class="block text-gray-300 text-xs mt-2">{{ errorMessage }}</span>
                  </p>
                  <Button @click="fetchData" label="Erneut versuchen" class="mt-6 w-full sm:w-auto p-button-sm sm:p-button-md" />
             </div>
        </div>

        <!-- Dashboard Grid -->
        <div v-else class="grid-stack-dashboard grid-stack glass-effect rounded-xl sm:rounded-2xl min-h-[60vh] sm:min-h-[500px] border border-gray-700/50 relative overflow-hidden">
            <Teleport v-for="widget in widgets" :key="widget.id" :to="'#mount_' + widget.id">
                <div class="relative h-full w-full p-2 sm:p-3 flex flex-col justify-between animate-fade-in">
                    <DashboardWidget
                        :title="widget.title"
                        :value="getWidgetValue(widget)"
                        :unit="widget.unit"
                        :status="widget.status"
                    />
                     <button
                        type="button"
                        class="absolute top-2 right-2 cursor-pointer text-gray-400 hover:text-red-400 z-10 transition-colors p-1 rounded hover:bg-red-500/20 bg-gray-800 shadow-sm"
                        @click="removeWidget(widget.id)"
                        aria-label="Remove widget"
                        title="Widget entfernen"
                    >
                        <i class="pi pi-times text-sm"></i>
                    </button>
                </div>
            </Teleport>
        </div>

        <Dialog v-model:visible="showAddWidget" header="Widget hinzufügen" :modal="true" class="w-11/12 sm:w-full max-w-[400px]">
            <div class="flex flex-col gap-4">
                <label class="text-sm font-medium">Proxy wählen</label>
                <Dropdown
                    v-model="selectedProxy"
                    :options="proxyOptions"
                    optionLabel="name"
                    optionValue="id"
                    placeholder="Wähle einen Proxy"
                    filter
                    class="w-full p-inputtext-sm sm:p-inputtext-md"
                />
                <Button label="Hinzufügen" @click="confirmAddWidget" :disabled="!selectedProxy" class="w-full p-button-sm sm:p-button-md" />
            </div>
        </Dialog>

        <!-- Proxy Configuration Panel -->
        <ProxyConfigPanel
            :visible="showConfigPanel"
            :proxies="proxies"
            @close="showConfigPanel = false"
            @refresh="fetchData"
        />
    </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue';
import { GridStack } from 'gridstack';
import 'gridstack/dist/gridstack.min.css';
import axios from '../axios.js';
import Button from 'primevue/button';
import Dialog from 'primevue/dialog';
import Dropdown from 'primevue/dropdown';
import DashboardWidget from '../components/DashboardWidget.vue';
import ProxyConfigPanel from '../components/ProxyConfigPanel.vue';
import { useEventSource } from '../utils/eventSource';

const grid = ref(null);
const proxies = ref([]);
const widgets = ref([]);
const showAddWidget = ref(false);
const showConfigPanel = ref(false);
const selectedProxy = ref(null);
const proxyOptions = ref([]);
const loading = ref(true);
const error = ref(null);
const errorMessage = ref('');
let sseDisconnect = null;
let refreshInterval = null;

// Auto-refresh every 10 seconds
const AUTO_REFRESH_INTERVAL = 10000;

onMounted(async () => {
    try {
        await fetchData();

        const isMobile = window.innerWidth <= 640;

        grid.value = GridStack.init({
            float: true,
            cellHeight: 80,
            minRow: 1,
            margin: 3,
            column: 6,
            disableOneColumnMode: false,
            oneColumnModeDomSort: true,
            oneColumnModeWidth: 640,
            disableDrag: isMobile,
            disableResize: isMobile,
            breakpointForNColumn: {
                1: { width: 640, column: 1 },
                2: { width: 768, column: 2 },
                3: { width: 1024, column: 3 },
                4: { width: 1280, column: 4 },
                6: { width: 1536, column: 6 }
            }
        });

        // Add resize listener to update drag/resize state
        window.addEventListener('resize', handleResize);

        const savedLayout = localStorage.getItem('dashboard_layout');
        let layoutToLoad = [];

        if (savedLayout) {
            try {
                layoutToLoad = JSON.parse(savedLayout);
            } catch (e) {
                console.error(e);
            }
        }

        if (!layoutToLoad || layoutToLoad.length === 0) {
            const defaults = proxies.value.map((p, index) => ({
                x: (index % 3) * 2,
                y: Math.floor(index / 3) * 2,
                w: 2,
                h: 2,
                id: `w_default_${p.id}`,
                proxy_id: p.id,
                title: p.name,
                unit: 'req',
                status: p.status
            }));
            layoutToLoad = defaults;
        }

        loadGrid(layoutToLoad);

        grid.value.on('change', saveLayout);
        grid.value.on('dragstop', saveLayout);
        grid.value.on('resizestop', saveLayout);

        const { data, disconnect } = useEventSource('/api/proxies/stream');
        sseDisconnect = disconnect;

        watch(data, (eventData) => {
            if (!eventData) return;

            const eventType = eventData.type;
            const proxyData = eventData.proxy;

            switch (eventType) {
                case 'proxy_added':
                case 'proxy_updated':
                case 'proxy_started':
                case 'proxy_stopped':
                    if (proxyData) {
                        const index = proxies.value.findIndex(p => p.id === proxyData.id);
                        if (index !== -1) {
                            proxies.value[index] = proxyData;
                        } else {
                            proxies.value.push(proxyData);
                            proxyOptions.value.push({ name: proxyData.name, id: proxyData.id });
                        }
                    }
                    break;
                case 'proxy_removed':
                    if (eventData.proxy_id) {
                        proxies.value = proxies.value.filter(p => p.id !== eventData.proxy_id);
                        proxyOptions.value = proxyOptions.value.filter(p => p.id !== eventData.proxy_id);
                    }
                    break;
            }
        });

        // Set up auto-refresh interval as fallback
        refreshInterval = setInterval(async () => {
            await fetchData();
        }, AUTO_REFRESH_INTERVAL);

    } catch (err) {
        error.value = true;
        errorMessage.value = err.message || 'Fehler beim Initialisieren des Dashboards';
        loading.value = false;
    }
});

const handleResize = () => {
    if (grid.value) {
        const isMobile = window.innerWidth <= 640;
        grid.value.setStatic(isMobile);
    }
};

onUnmounted(() => {
    window.removeEventListener('resize', handleResize);
    if (grid.value) {
        grid.value.destroy();
    }
    if (sseDisconnect) {
        sseDisconnect();
    }
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
});

const loadGrid = (layout) => {
    grid.value.removeAll();
    widgets.value = [];

    layout.forEach(item => {
        addWidgetToGrid(item);
    });
};

const addWidgetToGrid = (item) => {
    const id = item.id || `w_${Date.now()}`;
    const contentHtml = `<div id="mount_${id}" class="h-full w-full relative"></div>`;

    const el = grid.value.addWidget({
        x: item.x,
        y: item.y,
        w: item.w,
        h: item.h,
        id: id,
        content: ''
    });

    if (el) {
        const contentEl = el.querySelector('.grid-stack-item-content');
        if (contentEl) {
            contentEl.innerHTML = contentHtml;
        }
    }

    nextTick(() => {
        widgets.value.push({
            ...item,
            id: id
        });
    });
};

const saveLayout = () => {
    const items = grid.value.getGridItems();
    const layout = items.map(item => {
        const w = widgets.value.find(x => x.id == item.gridstackNode.id);
        return {
            x: item.gridstackNode.x,
            y: item.gridstackNode.y,
            w: item.gridstackNode.w,
            h: item.gridstackNode.h,
            id: item.gridstackNode.id,
            proxy_id: w ? w.proxy_id : '',
            title: w ? w.title : '',
            unit: w ? w.unit : '',
            status: w ? w.status : ''
        };
    });
    localStorage.setItem('dashboard_layout', JSON.stringify(layout));
};

const fetchData = async () => {
    try {
        loading.value = true;
        error.value = null;
        errorMessage.value = '';
        const res = await axios.get('/api/proxies');
        proxies.value = res.data;
        proxyOptions.value = res.data.map(p => ({ name: p.name, id: p.id }));
        loading.value = false;
    } catch (e) {
        const errorData = e.response?.data;
        let msg = 'Unbekannter Fehler';

        if (typeof errorData === 'string') {
            msg = errorData;
        } else if (e.message) {
            msg = e.message;
        }

        error.value = true;
        errorMessage.value = msg;
        loading.value = false;
    }
};

const getWidgetValue = (widget) => {
    const p = proxies.value.find(x => x.id === widget.proxy_id);
    if (!p) return 'Unknown';
    if (p.status === 'Running') {
        return `${p.requests || 0} Anfragen`;
    }
    return p.status;
};

const openAddWidget = () => {
    showAddWidget.value = true;
};

const confirmAddWidget = () => {
    if (selectedProxy.value) {
        const p = proxies.value.find(x => x.id === selectedProxy.value);
        addWidgetToGrid({
            x: 0, y: 0, w: 2, h: 2,
            proxy_id: selectedProxy.value,
            title: p.name,
            unit: 'req',
            status: p.status
        });
        showAddWidget.value = false;
        selectedProxy.value = null;
        saveLayout();
    }
};

const removeWidget = (id) => {
    const el = document.getElementById(`mount_${id}`);
    if (el) {
        const gridItem = el.closest('.grid-stack-item');
        if (gridItem) {
            grid.value.removeWidget(gridItem);
        }
    }

    widgets.value = widgets.value.filter(w => w.id !== id);
    saveLayout();
};

const resetLayout = () => {
    localStorage.removeItem('dashboard_layout');
    location.reload();
};
</script>

<style scoped>
/* Dashboard Container */
.dashboard-container {
    background: linear-gradient(135deg, rgba(17, 24, 39, 0.9) 0%, rgba(31, 41, 55, 0.9) 100%);
    min-height: 100vh;
}

/* Header Styles */
.dashboard-header {
    position: relative;
    z-index: 10;
}

.header-icon {
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.2) 0%, rgba(139, 92, 246, 0.2) 100%);
    border-radius: 12px;
    border: 1px solid rgba(59, 130, 246, 0.3);
    color: #60a5fa;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.2);
}

/* Glass Effect */
.glass-effect {
    background: rgba(31, 41, 55, 0.6);
    backdrop-filter: blur(12px);
    border: 1px solid rgba(75, 85, 99, 0.3);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

/* Loading Animation */
.loading-container {
    padding: 40px;
}

.loading-spinner {
    position: relative;
    width: 80px;
    height: 80px;
    margin: 0 auto;
}

.loading-spinner::before {
    content: '';
    position: absolute;
    inset: -8px;
    border-radius: 50%;
    background: conic-gradient(from 0deg, transparent, #3b82f6, transparent);
    animation: spin 2s linear infinite;
    opacity: 0.3;
}

.loading-spinner i {
    color: #3b82f6;
    filter: drop-shadow(0 0 12px rgba(59, 130, 246, 0.6));
}

/* Error Animation */
.error-container {
    max-width: 400px;
    margin: 0 auto;
}

.error-icon {
    display: inline-block;
    animation: shake 0.5s ease-in-out;
}

/* Grid Stack */
.grid-stack-dashboard {
    position: relative;
}

.grid-stack-dashboard::before {
    content: '';
    position: absolute;
    inset: 0;
    background: radial-gradient(circle at 50% 50%, rgba(59, 130, 246, 0.05) 0%, transparent 50%);
    pointer-events: none;
}

.grid-stack-item-content {
    background: linear-gradient(135deg, rgba(31, 41, 55, 0.8) 0%, rgba(17, 24, 39, 0.8) 100%);
    backdrop-filter: blur(16px);
    color: white;
    border-radius: 16px;
    box-shadow:
        0 4px 6px -1px rgba(0, 0, 0, 0.3),
        0 2px 4px -1px rgba(0, 0, 0, 0.06),
        inset 0 1px 0 rgba(255, 255, 255, 0.05);
    overflow: hidden;
    transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
    border: 1px solid rgba(75, 85, 99, 0.4);
    position: relative;
}

.grid-stack-item-content::before {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(135deg, rgba(255, 255, 255, 0.05) 0%, transparent 50%);
    pointer-events: none;
}

.grid-stack-item:hover .grid-stack-item-content {
    box-shadow:
        0 20px 25px -5px rgba(0, 0, 0, 0.5),
        0 10px 10px -5px rgba(0, 0, 0, 0.04),
        0 0 0 1px rgba(59, 130, 246, 0.5),
        0 0 20px rgba(59, 130, 246, 0.2);
    border-color: rgba(59, 130, 246, 0.6);
    transform: translateY(-4px) scale(1.02);
    z-index: 10 !important;
}

/* Animations */
.animate-fade-in {
    animation: fadeIn 0.4s ease-out;
}

.animate-pulse {
    animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

.animate-shake {
    animation: shake 0.5s ease-in-out;
}

.animate-glow {
    animation: glow 2s ease-in-out infinite;
}

.animate-spin-slow {
    animation: spin 3s linear infinite;
}

@keyframes fadeIn {
    from {
        opacity: 0;
        transform: translateY(20px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

@keyframes pulse {
    0%, 100% {
        opacity: 1;
    }
    50% {
        opacity: 0.5;
    }
}

@keyframes shake {
    0%, 100% {
        transform: translateX(0);
    }
    10%, 30%, 50%, 70%, 90% {
        transform: translateX(-4px);
    }
    20%, 40%, 60%, 80% {
        transform: translateX(4px);
    }
}

@keyframes glow {
    0%, 100% {
        box-shadow: 0 0 5px rgba(59, 130, 246, 0.3);
    }
    50% {
        box-shadow: 0 0 20px rgba(59, 130, 246, 0.6), 0 0 30px rgba(59, 130, 246, 0.3);
    }
}

@keyframes spin {
    from {
        transform: rotate(0deg);
    }
    to {
        transform: rotate(360deg);
    }
}

/* Button Enhancements */
:deep(.p-button-rounded) {
    width: 44px;
    height: 44px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

:deep(.p-button-rounded:hover) {
    transform: rotate(90deg) scale(1.1);
    box-shadow: 0 0 20px rgba(59, 130, 246, 0.4);
}

/* Dialog Enhancements */
:deep(.p-dialog) {
    backdrop-filter: blur(20px);
    background: rgba(31, 41, 55, 0.95);
    border: 1px solid rgba(75, 85, 99, 0.5);
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
}

:deep(.p-dialog-header) {
    border-bottom: 1px solid rgba(75, 85, 99, 0.3);
}

/* Responsive Design */
@media (max-width: 640px) {
    .dashboard-header {
        gap: 12px;
    }

    .header-icon {
        width: 40px;
        height: 40px;
    }

    .grid-stack-item-content {
        border-radius: 12px;
    }
}
</style>
