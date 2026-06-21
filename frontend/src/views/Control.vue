<template>
    <div class="p-2 sm:p-4 flex flex-col gap-4 w-full min-w-0">

        <!-- ── Hero ────────────────────────────────────────────────── -->
        <section class="glass-hero rounded-[28px] p-5 sm:p-6">
            <div class="relative z-[1] flex flex-col gap-5 xl:flex-row xl:items-end xl:justify-between">
                <div class="space-y-3">
                    <div class="inline-flex items-center gap-3 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.28em] text-[var(--text-muted)]">
                        <i class="pi pi-sliders-h"></i>
                        {{ t('control.badge') }}
                        <span v-if="sseConnected !== null" class="flex items-center gap-1.5 ml-1">
                            <span class="status-dot" :class="sseConnected ? 'status-dot--running' : 'status-dot--error'"></span>
                            <span>{{ sseConnected ? t('common.connected') : t('common.disconnected') }}</span>
                        </span>
                    </div>
                    <div>
                        <h1 class="text-2xl sm:text-3xl font-bold text-[var(--text-primary)]">{{ $t('control.title') }}</h1>
                        <p class="mt-2 text-sm sm:text-base text-[var(--text-secondary)]">{{ $t('control.subtitle') }}</p>
                    </div>
                </div>
                <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
                    <div class="ctrl-stat">
                        <span class="ctrl-stat-label">{{ $t('control.total') }}</span>
                        <strong class="ctrl-stat-value">{{ proxies.length }}</strong>
                    </div>
                    <div class="ctrl-stat">
                        <span class="ctrl-stat-label">{{ $t('control.running') }}</span>
                        <strong class="ctrl-stat-value" style="color:var(--success)">{{ runningCount }}</strong>
                    </div>
                    <div class="ctrl-stat">
                        <span class="ctrl-stat-label">{{ $t('control.stopped') }}</span>
                        <strong class="ctrl-stat-value" style="color:var(--warning)">{{ stoppedCount }}</strong>
                    </div>
                    <div class="ctrl-stat">
                        <span class="ctrl-stat-label">{{ $t('control.error') }}</span>
                        <strong class="ctrl-stat-value" style="color:var(--danger)">{{ errorCount }}</strong>
                    </div>
                </div>
            </div>
        </section>

        <!-- ── Toolbar ─────────────────────────────────────────────── -->
        <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
            <div class="relative flex-1 min-w-0">
                <i class="pi pi-search absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)] text-sm pointer-events-none"></i>
                <input
                    v-model="searchQuery"
                    type="search"
                    :placeholder="$t('control.searchPlaceholder')"
                    class="ctrl-search w-full"
                />
            </div>
            <div class="flex flex-wrap gap-2">
                <Button
                    v-if="auth.hasPermission('proxy:edit')"
                    :icon="editMode ? 'pi pi-lock' : 'pi pi-pencil'"
                    :severity="editMode ? 'warn' : 'secondary'"
                    :label="editMode ? $t('control.lock') : $t('control.edit')"
                    @click="editMode = !editMode"
                    class="text-sm shrink-0"
                />
                <Button
                    v-if="auth.hasPermission('proxy:create')"
                    icon="pi pi-plus"
                    severity="info"
                    :label="$t('common.add') + ' Proxy'"
                    @click="openAddProxyDialog"
                    class="text-sm shrink-0"
                />
                <Button
                    v-if="auth.hasPermission('proxy:control')"
                    icon="pi pi-play"
                    severity="success"
                    :label="$t('control.startAll')"
                    @click="controlAllProxies('start_all')"
                    class="text-sm shrink-0"
                />
                <Button
                    v-if="auth.hasPermission('proxy:control')"
                    icon="pi pi-stop"
                    severity="danger"
                    :label="$t('control.stopAll')"
                    @click="controlAllProxies('stop_all')"
                    class="text-sm shrink-0"
                />
            </div>
        </div>

        <!-- ── Loading ─────────────────────────────────────────────── -->
        <div v-if="loading" class="glass-panel rounded-[28px] p-10">
            <div class="flex min-h-[320px] flex-col items-center justify-center text-center relative z-[1]">
                <div class="mb-5 flex h-20 w-20 items-center justify-center rounded-2xl bg-[var(--bg-panel-item)] border border-[var(--border-subtle)]">
                    <i class="pi pi-spin pi-spinner text-3xl text-[var(--accent)]"></i>
                </div>
                <p class="text-[var(--text-secondary)] text-sm">{{ $t('control.loading') }}</p>
            </div>
        </div>

        <!-- ── Empty state ─────────────────────────────────────────── -->
        <div v-else-if="proxies.length === 0" class="glass-panel rounded-[28px] p-10">
            <div class="flex min-h-[280px] flex-col items-center justify-center text-center relative z-[1]">
                <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-[var(--bg-panel-item)] border border-[var(--border-subtle)]">
                    <i class="pi pi-inbox text-2xl text-[var(--text-muted)]"></i>
                </div>
                <h3 class="text-lg font-semibold text-[var(--text-primary)]">{{ $t('control.noProxies') }}</h3>
                <p class="mt-2 text-sm text-[var(--text-muted)] max-w-sm">{{ $t('control.noProxiesHint') }}</p>
            </div>
        </div>

        <!-- ── Proxy grid ──────────────────────────────────────────── -->
        <div v-else>
            <!-- No search results -->
            <div v-if="filteredGroups.length === 0" class="glass-panel rounded-[28px] p-8 text-center relative z-[1]">
                <i class="pi pi-search text-2xl text-[var(--text-muted)] mb-3 block"></i>
                <p class="text-[var(--text-secondary)] text-sm">{{ $t('control.noResults', { query: searchQuery }) }}</p>
            </div>

            <Tabs v-else value="0">
                <TabList>
                    <Tab v-for="(group, index) in filteredGroups" :key="group.name" :value="String(index)">
                        {{ group.name }}
                        <span class="ml-1.5 text-xs text-[var(--text-muted)]">({{ group.proxies.length }})</span>
                    </Tab>
                </TabList>
                <TabPanels>
                    <TabPanel v-for="(group, index) in filteredGroups" :key="group.name" :value="String(index)">
                        <VueDraggable
                            :list="group.proxies"
                            :disabled="!editMode"
                            group="proxies"
                            handle=".drag-handle"
                            ghost-class="drag-ghost"
                            drag-class="drag-active"
                            class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4"
                        >
                            <div
                                v-for="proxy in group.proxies"
                                :key="proxy.id"
                                class="proxy-card"
                                :class="{ 'proxy-card--edit': editMode }"
                            >
                                <div class="p-5 flex flex-col gap-3">
                                    <!-- Card header -->
                                    <div class="flex items-start justify-between gap-3">
                                        <div class="flex items-center gap-2.5 min-w-0">
                                            <div
                                                v-if="editMode"
                                                class="drag-handle shrink-0 cursor-grab active:cursor-grabbing flex items-center justify-center w-8 h-8 rounded-lg hover:bg-[var(--bg-soft)] transition-colors"
                                            >
                                                <i class="pi pi-bars text-[var(--text-muted)] text-sm"></i>
                                            </div>
                                            <div class="min-w-0">
                                                <span class="block text-base font-semibold text-[var(--text-primary)] truncate" :title="proxy.name">{{ proxy.name }}</span>
                                                <span class="block text-xs text-[var(--text-muted)] mt-0.5 truncate">{{ proxy.description || '—' }}</span>
                                            </div>
                                        </div>
                                        <div class="proxy-status-badge shrink-0" :class="`proxy-status-badge--${proxy.status?.toLowerCase()}`">
                                            <span class="status-dot" :class="`status-dot--${proxy.status === 'Running' ? 'running' : proxy.status === 'Error' ? 'error' : proxy.status === 'Stopped' ? 'stopped' : 'unknown'}`"></span>
                                            {{ proxy.status }}
                                        </div>
                                    </div>

                                    <!-- Route line -->
                                    <div class="flex items-center gap-2 text-xs text-[var(--text-muted)] font-mono bg-[var(--bg-panel-item)] rounded-xl px-3 py-2 min-w-0">
                                        <span class="truncate text-[var(--accent)]" :title="proxy.listen_addr">{{ proxy.listen_addr }}</span>
                                        <i class="pi pi-arrow-right shrink-0 text-[var(--border-strong)]"></i>
                                        <span class="truncate" :title="proxy.target_addr">{{ proxy.target_addr }}</span>
                                    </div>

                                    <!-- Tags -->
                                    <div v-if="proxy.tags?.length" class="flex flex-wrap gap-1">
                                        <span v-for="tag in proxy.tags" :key="tag" class="proxy-tag">{{ tag }}</span>
                                    </div>

                                    <!-- Actions -->
                                    <div class="flex gap-2 mt-1">
                                        <Button
                                            v-if="auth.hasPermission('proxy:control') && proxy.status !== 'Running'"
                                            icon="pi pi-play"
                                            severity="success"
                                            label="Start"
                                            @click="controlProxy(proxy.id, 'start')"
                                            class="flex-1 min-h-[40px] rounded-2xl text-sm"
                                            size="small"
                                        />
                                        <Button
                                            v-if="auth.hasPermission('proxy:control') && proxy.status === 'Running'"
                                            icon="pi pi-stop"
                                            severity="danger"
                                            label="Stop"
                                            @click="controlProxy(proxy.id, 'stop')"
                                            class="flex-1 min-h-[40px] rounded-2xl text-sm"
                                            size="small"
                                        />
                                        <Button
                                            icon="pi pi-ellipsis-v"
                                            severity="secondary"
                                            @click="(e) => toggleMenu(e, proxy)"
                                            class="min-h-[40px] rounded-2xl w-10 shrink-0"
                                            size="small"
                                            aria-haspopup="true"
                                        />
                                    </div>

                                    <!-- Connection test result -->
                                    <div
                                        v-if="connectionStatus[proxy.id]"
                                        class="px-3 py-2.5 rounded-xl text-xs flex items-start gap-2"
                                        :class="connectionStatus[proxy.id].reachable
                                            ? 'bg-[rgba(74,222,128,0.08)] border border-[rgba(74,222,128,0.2)] text-[var(--success)]'
                                            : 'bg-[rgba(251,113,133,0.08)] border border-[rgba(251,113,133,0.2)] text-[var(--danger)]'"
                                    >
                                         <i :class="connectionStatus[proxy.id].reachable ? 'pi pi-check-circle' : 'pi pi-times-circle'" class="shrink-0 mt-0.5"></i>
                                         <div>
                                             <div class="font-semibold">{{ connectionStatus[proxy.id].reachable ? $t('control.reachable') : $t('control.notReachable') }}</div>
                                             <div v-if="!connectionStatus[proxy.id].reachable" class="mt-0.5 text-[var(--text-muted)]">{{ connectionStatus[proxy.id].error }}</div>
                                         </div>
                                    </div>
                                </div>
                            </div>
                        </VueDraggable>
                    </TabPanel>
                </TabPanels>
            </Tabs>
        </div>

         <Dialog v-model:visible="showProxyDialog" :header="isEditMode ? $t('common.edit') + ' Proxy' : $t('common.add') + ' Proxy'" modal class="w-full max-w-lg">
             <div class="flex flex-col gap-4">
                 <div>
                     <label class="block text-sm font-medium mb-1">{{ $t('control.form.name') }}</label>
                     <InputText v-model="proxyForm.name" class="w-full" />
                 </div>
                 <div>
                     <label class="block text-sm font-medium mb-1">{{ $t('control.form.listenAddr') }}</label>
                     <InputText v-model="proxyForm.listen_addr" placeholder=":5020" class="w-full" />
                 </div>
                 <div>
                     <label class="block text-sm font-medium mb-1">{{ $t('control.form.targetAddr') }}</label>
                     <InputText v-model="proxyForm.target_addr" placeholder="192.168.1.100:502" class="w-full" />
                 </div>
                 <div>
                     <label class="block text-sm font-medium mb-1">{{ $t('control.form.description') }}</label>
                     <InputText v-model="proxyForm.description" class="w-full" />
                 </div>
                 <div class="flex flex-col sm:flex-row gap-4">
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">{{ $t('control.form.connectionTimeout') }}</label>
                         <InputNumber v-model="proxyForm.connection_timeout" :min="1" class="w-full" />
                     </div>
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">{{ $t('control.form.readTimeout') }}</label>
                         <InputNumber v-model="proxyForm.read_timeout" :min="1" class="w-full" />
                     </div>
                 </div>
                 <div class="flex flex-col sm:flex-row gap-4">
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">{{ $t('control.form.maxRetries') }}</label>
                         <InputNumber v-model="proxyForm.max_retries" :min="0" class="w-full" />
                     </div>
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">{{ $t('control.form.maxReadSize') }}</label>
                         <InputNumber v-model="proxyForm.max_read_size" :min="0" class="w-full" />
                     </div>
                 </div>
                 <div class="flex items-center gap-4">
                     <div class="flex items-center gap-2">
                         <Checkbox v-model="proxyForm.enabled" binary />
                         <span class="text-sm">{{ $t('control.form.enabled') }}</span>
                     </div>
                     <div class="flex items-center gap-2">
                         <Checkbox v-model="proxyForm.paused" binary />
                         <span class="text-sm">{{ $t('control.form.paused') }}</span>
                     </div>
                 </div>
                 <div>
                     <label class="block text-sm font-medium mb-1">{{ $t('control.form.tags') }}</label>
                     <Chips v-model="proxyForm.tags" class="w-full" :placeholder="$t('control.form.tags')" />
                 </div>
             </div>
             <template #footer>
                 <Button :label="$t('common.cancel')" severity="secondary" @click="showProxyDialog = false" />
                 <Button :label="isEditMode ? $t('common.edit') : $t('common.add')" :loading="savingProxy" @click="saveProxy" />
             </template>
         </Dialog>

         <Dialog v-model:visible="showLogsDialog" :header="`Logs - ${currentProxy?.name}`" modal class="w-full max-w-4xl">
              <div class="rounded-2xl p-4 font-mono text-sm h-[500px] overflow-y-auto bg-gray-100 dark:bg-black/40 border border-gray-200 dark:border-white/5">
                  <div v-if="proxyLogs.length === 0" class="text-gray-400 dark:text-gray-500">No logs available</div>
                  <div v-else class="space-y-1">
                      <div v-for="(log, index) in proxyLogs" :key="index" class="border-b border-gray-200 dark:border-white/5 pb-1">
                           <span class="text-gray-500 dark:text-gray-400">[{{ formatTime(log.timestamp) }}]</span>
                         <span :class="getLogLevelColor(log.level)" class="mx-2 font-bold">{{ log.level }}</span>
                         <span class="text-surface-900 dark:text-white">{{ log.message }}</span>
                     </div>
                 </div>
             </div>
         </Dialog>
         <Menu ref="actionMenu" id="overlay_menu" :model="menuItems" :popup="true" />
         <Toast />
         <ConfirmDialog />
     </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, computed } from 'vue';
import axios from '../axios.js';
import Button from 'primevue/button';
import Dialog from 'primevue/dialog';
import Menu from 'primevue/menu';
import InputText from 'primevue/inputtext';
import InputNumber from 'primevue/inputnumber';
import Checkbox from 'primevue/checkbox';
import Chips from 'primevue/chips';
import Toast from 'primevue/toast';
import ConfirmDialog from 'primevue/confirmdialog';
import Tabs from 'primevue/tabs';
import TabList from 'primevue/tablist';
import Tab from 'primevue/tab';
import TabPanels from 'primevue/tabpanels';
import TabPanel from 'primevue/tabpanel';
import { useToast } from 'primevue/usetoast';
import { useConfirm } from 'primevue/useconfirm';
import { useI18n } from 'vue-i18n';
import { VueDraggable } from 'vue-draggable-plus';
import { useEventSource } from '../utils/eventSource';
import { getLogLevelColor, formatTime } from '../utils/helpers';
import { useAuthStore } from '../stores/auth';

const auth = useAuthStore();
const { t } = useI18n();
const proxies = ref([]);
const loading = ref(true);
const editMode = ref(false);
const searchQuery = ref('');
const sseConnected = ref(null);

const runningCount = computed(() => proxies.value.filter(p => p.status === 'Running').length);
const stoppedCount = computed(() => proxies.value.filter(p => p.status === 'Stopped').length);
const errorCount   = computed(() => proxies.value.filter(p => p.status === 'Error').length);
const toast = useToast();
const confirm = useConfirm();
let disconnectFn = null;
const pendingTimers = [];

const testingProxy = ref(null);
const connectionStatus = ref({});

let unwatchConnected = null;
const actionMenu = ref();
const menuItems = ref([]);
const activeProxyForMenu = ref(null);

let unwatchData = null;

const defaultProxyForm = () => ({
    id: '',
    name: 'New Proxy',
    listen_addr: ':5020',
    target_addr: '127.0.0.1:502',
    description: '',
    connection_timeout: 10,
    read_timeout: 30,
    max_retries: 3,
    max_read_size: 0,
    enabled: true,
    paused: false,
    tags: []
});

const showProxyDialog = ref(false);
const isEditMode = ref(false);
const proxyForm = ref(defaultProxyForm());
const savingProxy = ref(false);

const showLogsDialog = ref(false);
const currentProxy = ref(null);
const proxyLogs = ref([]);

const groups = computed(() => {
    const groupMap = {};
    proxies.value.forEach(proxy => {
        let proxyGroups = ['Ungrouped'];
        if (proxy.tags && proxy.tags.length > 0) {
            proxyGroups = proxy.tags;
        }
        proxyGroups.forEach(tag => {
            if (!groupMap[tag]) {
                groupMap[tag] = [];
            }
            groupMap[tag].push(proxy);
        });
    });

    const result = Object.keys(groupMap).sort().map(key => ({
        name: key,
        proxies: groupMap[key]
    }));

    if (result.length === 0) {
         return [{ name: 'All', proxies: proxies.value }];
    }

    return result;
});

const filteredGroups = computed(() => {
    const q = searchQuery.value.trim().toLowerCase();
    if (!q) return groups.value;
    return groups.value
        .map(group => ({
            ...group,
            proxies: group.proxies.filter(p =>
                p.name.toLowerCase().includes(q) ||
                p.listen_addr.toLowerCase().includes(q) ||
                p.target_addr.toLowerCase().includes(q) ||
                (p.description || '').toLowerCase().includes(q)
            )
        }))
        .filter(group => group.proxies.length > 0);
});

const applyProxyOrder = (data) => {
    const saved = localStorage.getItem('proxy_order');
    if (!saved) return data;
    try {
        const order = JSON.parse(saved);
        const ordered = [];
        const remaining = [...data];
        for (const id of order) {
            const idx = remaining.findIndex(p => p.id === id);
            if (idx !== -1) {
                ordered.push(remaining.splice(idx, 1)[0]);
            }
        }
        return [...ordered, ...remaining];
    } catch {
        return data;
    }
};

const fetchProxies = async () => {
    try {
        const res = await axios.get('/api/proxies');
        proxies.value = applyProxyOrder(res.data);
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to fetch proxies', life: 5000 });
    }
};

onMounted(async () => {
    await fetchProxies();
    loading.value = false;

    const { data, disconnect, isConnected } = useEventSource('/api/proxies/stream');
    disconnectFn = disconnect;

    unwatchConnected = watch(isConnected, (connected) => {
        sseConnected.value = connected;
        if (!connected) {
            console.warn('SSE connection lost');
        }
    });

    unwatchData = watch(data, (eventData) => {
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
                    }
                }
                break;
            case 'proxy_removed':
                if (eventData.proxy_id) {
                    proxies.value = proxies.value.filter(p => p.id !== eventData.proxy_id);
                }
                break;
        }
    });
});

onUnmounted(() => {
    pendingTimers.forEach(clearTimeout);
    pendingTimers.length = 0;
    if (unwatchConnected) unwatchConnected();
    if (unwatchData) unwatchData();
    if (disconnectFn) {
        disconnectFn();
    }
});

const openAddProxyDialog = () => {
    isEditMode.value = false;
    proxyForm.value = defaultProxyForm();
    showProxyDialog.value = true;
};

const openEditProxyDialog = (proxy) => {
    isEditMode.value = true;
    proxyForm.value = { ...proxy };
    showProxyDialog.value = true;
};

const saveProxy = async () => {
    if (savingProxy.value) return;
    savingProxy.value = true;
    try {
        if (isEditMode.value) {
            await axios.put('/api/proxies', proxyForm.value);
            toast.add({ severity: 'success', summary: 'Success', detail: 'Proxy updated', life: 3000 });
        } else {
            await axios.post('/api/proxies', proxyForm.value);
            toast.add({ severity: 'success', summary: 'Success', detail: 'Proxy created', life: 3000 });
        }
        showProxyDialog.value = false;
        await fetchProxies();
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: e.response?.data || e.message, life: 5000 });
    } finally {
        savingProxy.value = false;
    }
};

const confirmDeleteProxy = (id) => {
    confirm.require({
        message: t('control.deleteConfirm'),
        header: t('common.confirm'),
        icon: 'pi pi-exclamation-triangle',
        accept: async () => {
            try {
                await axios.delete(`/api/proxies?id=${id}`);
                toast.add({ severity: 'success', summary: 'Success', detail: 'Proxy deleted', life: 3000 });
                await fetchProxies();
            } catch (e) {
                toast.add({ severity: 'error', summary: 'Error', detail: e.response?.data || e.message, life: 5000 });
            }
        }
    });
};

const toggleMenu = (event, proxy) => {
    activeProxyForMenu.value = proxy;

    const items = [];

    if (auth.hasPermission('proxy:control')) {
        const controlGroup = {
            label: 'Control',
            items: []
        };

        if (proxy.status !== 'Stopped' && proxy.status !== 'Error') {
            controlGroup.items.push({
                label: 'Restart',
                icon: 'pi pi-refresh',
                command: () => controlProxy(proxy.id, 'restart')
            });
        }

        if (!proxy.paused && proxy.status === 'Running') {
            controlGroup.items.push({
                label: 'Pause',
                icon: 'pi pi-pause',
                command: () => controlProxy(proxy.id, 'pause')
            });
        } else if (proxy.paused) {
            controlGroup.items.push({
                label: 'Resume',
                icon: 'pi pi-play',
                command: () => controlProxy(proxy.id, 'resume')
            });
        }

        if (controlGroup.items.length > 0) {
            items.push(controlGroup);
        }
    }

    const settingsGroup = {
        label: 'Manage',
        items: []
    };

    settingsGroup.items.push({
        label: 'Test Connection',
        icon: 'pi pi-search',
        command: () => testConnectivity(proxy)
    });

    settingsGroup.items.push({
        label: 'View Logs',
        icon: 'pi pi-eye',
        command: () => openProxyLogs(proxy.id)
    });

    if (auth.hasPermission('proxy:edit')) {
        settingsGroup.items.push({
            label: 'Edit',
            icon: 'pi pi-pencil',
            command: () => openEditProxyDialog(proxy)
        });
    }

    if (auth.hasPermission('proxy:delete')) {
        settingsGroup.items.push({
            label: 'Delete',
            icon: 'pi pi-trash',
            class: 'text-red-400',
            command: () => confirmDeleteProxy(proxy.id)
        });
    }

    items.push(settingsGroup);
    menuItems.value = items;

    actionMenu.value.toggle(event);
};

const openProxyLogs = async (id) => {
    currentProxy.value = proxies.value.find(p => p.id === id);
    try {
        const res = await axios.get('/api/logs');
        proxyLogs.value = res.data.filter(log => log.proxy_id === id);
    } catch (e) {
        console.error("Failed to fetch logs", e);
        proxyLogs.value = [];
    }
    showLogsDialog.value = true;
};

const controlProxy = async (id, action) => {
    try {
        await axios.post('/api/proxies/control', { id, action });
        toast.add({ severity: 'success', summary: 'Success', detail: `Proxy ${action} command sent`, life: 3000 });
        pendingTimers.push(setTimeout(fetchProxies, 500));
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: e.response?.data || e.message, life: 5000 });
    }
};

const controlAllProxies = async (action) => {
    const message = action === 'start_all' ? t('control.startAllConfirm') : t('control.stopAllConfirm');
    confirm.require({
        message,
        header: t('common.confirm'),
        icon: 'pi pi-exclamation-triangle',
        accept: async () => {
            try {
                await axios.post('/api/proxies/control', { action });
                toast.add({ severity: 'success', summary: 'Success', detail: `All proxies ${actionLabel} command sent`, life: 3000 });
                pendingTimers.push(setTimeout(fetchProxies, 500));
            } catch (e) {
                toast.add({ severity: 'error', summary: 'Error', detail: e.response?.data || e.message, life: 5000 });
            }
        }
    });
};

const testConnectivity = async (proxy) => {
    testingProxy.value = proxy.id;
    try {
        const res = await axios.get('/api/system/diagnostics/connectivity');
        const proxyConnStatus = res.data[proxy.id];
        if (proxyConnStatus) {
            connectionStatus.value = { ...connectionStatus.value, [proxy.id]: proxyConnStatus };
            if (proxyConnStatus.reachable) {
                toast.add({
                    severity: 'success',
                    summary: 'Connection OK',
                    detail: `${proxy.name} can reach ${proxyConnStatus.target}`,
                    life: 4000
                });
            } else {
                toast.add({
                    severity: 'error',
                    summary: 'Connection Failed',
                    detail: `Cannot reach ${proxyConnStatus.target}: ${proxyConnStatus.error}`,
                    life: 5000
                });
            }
        }
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Diagnostic Error', detail: e.message, life: 3000 });
     } finally {
         testingProxy.value = null;
     }
};
</script>

<style scoped>
/* ── Hero stats ──────────────────────────────────────────────────── */
.ctrl-stat {
    border-radius: 20px;
    padding: 0.9rem 1rem;
    background: var(--bg-panel-item);
    border: 1px solid var(--border-subtle);
}
.ctrl-stat-label {
    display: block;
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.2em;
    color: var(--text-muted);
}
.ctrl-stat-value {
    display: block;
    margin-top: 0.4rem;
    font-size: 1.2rem;
    font-weight: 800;
    color: var(--text-primary);
}

/* ── Search input ───────────────────────────────────────────────── */
.ctrl-search {
    height: 2.75rem;
    border-radius: 16px;
    border: 1px solid var(--border-soft);
    background: var(--bg-input);
    color: var(--text-primary);
    padding: 0 1rem 0 2.5rem;
    outline: none;
    transition: border-color 0.2s, box-shadow 0.2s;
    font-size: 0.9rem;
}
.ctrl-search::placeholder { color: var(--text-muted); }
.ctrl-search:focus {
    border-color: var(--accent-strong);
    box-shadow: 0 0 0 4px var(--accent-tint);
}

/* ── Proxy card ─────────────────────────────────────────────────── */
.proxy-card {
    background: var(--bg-surface);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    border: 1px solid var(--border-soft);
    border-radius: 24px;
    box-shadow: var(--shadow-soft);
    transition: transform 0.22s ease, border-color 0.22s ease, box-shadow 0.22s ease;
}
.proxy-card:hover {
    transform: translateY(-2px);
    border-color: var(--border-strong);
    box-shadow: var(--shadow-strong);
}
.proxy-card--edit {
    border-color: var(--accent-tint);
    box-shadow: 0 0 0 2px var(--accent-tint);
}

/* ── Status badge ───────────────────────────────────────────────── */
.proxy-status-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.3rem 0.7rem;
    border-radius: 999px;
    font-size: 0.72rem;
    font-weight: 700;
    border: 1px solid var(--border-subtle);
    background: var(--bg-panel-item);
    color: var(--text-secondary);
}

/* ── Tag pill ────────────────────────────────────────────────────── */
.proxy-tag {
    display: inline-flex;
    align-items: center;
    padding: 0.2rem 0.55rem;
    border-radius: 999px;
    font-size: 0.7rem;
    background: var(--bg-panel-item);
    border: 1px solid var(--border-subtle);
    color: var(--text-muted);
}

/* ── Drag states ────────────────────────────────────────────────── */
.drag-ghost  { opacity: 0.4; border-radius: 24px !important; }
.drag-active { transform: rotate(1.5deg) scale(1.02); z-index: 1000; box-shadow: var(--shadow-strong) !important; }
</style>
