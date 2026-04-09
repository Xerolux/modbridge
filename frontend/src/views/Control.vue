<template>
    <div class="p-4 flex flex-col gap-4">
        <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3 mb-4">
            <h1 class="text-xl sm:text-2xl font-bold text-white">Proxy Control</h1>
            <div class="flex flex-wrap gap-2">
                <Button
                    v-if="auth.hasPermission('proxy:edit')"
                    :icon="editMode ? 'pi pi-lock' : 'pi pi-pencil'"
                    :severity="editMode ? 'warn' : 'secondary'"
                    :label="editMode ? 'Lock' : 'Edit'"
                    @click="editMode = !editMode"
                    class="text-sm flex-1 sm:flex-none"
                />
                <Button
                    v-if="auth.hasPermission('proxy:create')"
                    icon="pi pi-plus"
                    severity="info"
                    label="Add Proxy"
                    @click="openAddProxyDialog"
                    class="text-sm flex-1 sm:flex-none"
                />
                <Button
                    v-if="auth.hasPermission('proxy:control')"
                    icon="pi pi-play"
                    severity="success"
                    label="Start All"
                    @click="controlAllProxies('start_all')"
                    class="text-sm flex-1 sm:flex-none"
                />
                <Button
                    v-if="auth.hasPermission('proxy:control')"
                    icon="pi pi-stop"
                    severity="danger"
                    label="Stop All"
                    @click="controlAllProxies('stop_all')"
                    class="text-sm flex-1 sm:flex-none"
                />
            </div>
        </div>
        <div v-if="unreachableCount > 0" class="glass-card rounded-2xl p-3 border border-red-400/30 text-red-200 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <span>{{ unreachableCount }} proxy endpoints are unreachable. Check diagnostics and logs.</span>
            <Button label="Open Logs" icon="pi pi-eye" severity="danger" outlined @click="goToLogs" />
        </div>

         <div v-if="loading" class="flex justify-center min-h-[400px] items-center">
             <div class="text-center">
                 <i class="pi pi-spin pi-spinner text-4xl text-purple-400"></i>
                 <p class="mt-4 text-gray-400">Loading proxies...</p>
             </div>
         </div>

         <div v-else-if="proxies.length === 0" class="flex justify-center items-center min-h-[300px]">
             <div class="text-center text-gray-400 glass-card p-12 rounded-3xl">
                 <i class="pi pi-inbox text-5xl mb-4 block text-purple-400/50"></i>
                 <p>No proxies configured. Click "Add Proxy" to get started.</p>
             </div>
         </div>

         <VueDraggable
             v-else
             v-model="proxies"
             :disabled="!editMode"
             group="proxies"
             item-key="id"
             ghost-class="drag-ghost"
             drag-class="drag-active"
             class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4"
             @end="onDragEnd"
         >
             <template #item="{ element: proxy }">
                 <div class="glass-card rounded-3xl border border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10 cursor-default"
                      :class="{ 'ring-2 ring-purple-500/50 cursor-grab active:cursor-grabbing': editMode }">
                     <div class="p-5">
                         <div class="flex justify-between items-center mb-4">
                             <div class="flex items-center gap-3">
                                 <div v-if="editMode" class="drag-handle cursor-grab active:cursor-grabbing p-1 rounded-lg hover:bg-white/10 transition-colors">
                                     <i class="pi pi-bars text-gray-400 text-sm"></i>
                                 </div>
                                 <span class="text-lg font-semibold text-white truncate" :title="proxy.name">{{ proxy.name }}</span>
                             </div>
                             <Tag :severity="getSeverity(proxy.status)" :value="proxy.status" class="rounded-xl" />
                         </div>

                         <div class="flex flex-col gap-3">
                             <div class="text-gray-400 text-sm">{{ proxy.description || 'No description' }}</div>
                             <div class="flex items-center gap-2 text-sm text-gray-300">
                                 <i class="pi pi-arrow-right-arrow-left text-purple-400 text-xs"></i>
                                 <span>{{ proxy.listen_addr }}</span>
                                 <i class="pi pi-arrow-right text-gray-500 text-xs"></i>
                                 <span>{{ proxy.target_addr }}</span>
                             </div>

                              <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 mt-2">
                                   <Button
                                      v-if="auth.hasPermission('proxy:control')"
                                      icon="pi pi-play"
                                      severity="success"
                                      label="Start"
                                      :disabled="proxy.status === 'Running'"
                                      @click="controlProxy(proxy.id, 'start')"
                                      class="text-base p-3 sm:p-2 min-h-[44px] rounded-2xl"
                                   />
                                   <Button
                                      v-if="auth.hasPermission('proxy:control')"
                                      icon="pi pi-stop"
                                      severity="danger"
                                      label="Stop"
                                      :disabled="proxy.status === 'Stopped' || proxy.status === 'Error'"
                                      @click="controlProxy(proxy.id, 'stop')"
                                      class="text-base p-3 sm:p-2 min-h-[44px] rounded-2xl"
                                   />
                                    <Button
                                      v-if="auth.hasPermission('proxy:control') && !proxy.paused && proxy.status === 'Running'"
                                      icon="pi pi-pause"
                                      severity="warning"
                                      label="Pause"
                                      @click="controlProxy(proxy.id, 'pause')"
                                      class="text-base p-3 sm:p-2 min-h-[44px] rounded-2xl"
                                   />
                                   <Button
                                      v-if="auth.hasPermission('proxy:control') && proxy.paused"
                                      icon="pi pi-play"
                                      severity="success"
                                      label="Resume"
                                      @click="controlProxy(proxy.id, 'resume')"
                                      class="text-base p-3 sm:p-2 min-h-[44px] rounded-2xl"
                                   />
                                   <Button
                                      v-if="auth.hasPermission('proxy:control')"
                                      icon="pi pi-refresh"
                                      severity="info"
                                      label="Restart"
                                      :disabled="proxy.status === 'Stopped'"
                                      @click="controlProxy(proxy.id, 'restart')"
                                      class="text-base p-3 sm:p-2 min-h-[44px] rounded-2xl"
                                   />
                                   <Button
                                      v-if="auth.hasPermission('proxy:edit')"
                                      icon="pi pi-pencil"
                                      severity="secondary"
                                      label="Edit"
                                      @click="openEditProxyDialog(proxy)"
                                      class="text-base p-3 sm:p-2 min-h-[44px] rounded-2xl"
                                   />
                              </div>
                              <div class="grid grid-cols-1 sm:grid-cols-3 gap-2 mt-2">
                                  <Button
                                      icon="pi pi-search"
                                      severity="info"
                                      label="Test"
                                      @click="testConnectivity(proxy)"
                                      :loading="testingProxy === proxy.id"
                                      class="text-base p-3 sm:p-2 min-h-[44px] rounded-2xl"
                                  />
                                  <Button
                                      icon="pi pi-eye"
                                      severity="secondary"
                                      label="View Logs"
                                      @click="openProxyLogs(proxy.id)"
                                      class="text-base p-3 sm:p-2 min-h-[44px] rounded-2xl"
                                  />
                                  <Button
                                      v-if="auth.hasPermission('proxy:delete')"
                                      icon="pi pi-trash"
                                      severity="danger"
                                      label="Delete"
                                      @click="confirmDeleteProxy(proxy.id)"
                                      class="text-base p-3 sm:p-2 min-h-[44px] rounded-2xl"
                                  />
                              </div>

                             <div v-if="connectionStatus[proxy.id]" class="mt-3 p-3 rounded-2xl" :class="connectionStatus[proxy.id].reachable ? 'glass-card border-green-500/30 text-green-300' : 'glass-card border-red-500/30 text-red-300'">
                                 <div class="flex items-center gap-2 mb-1">
                                     <i :class="connectionStatus[proxy.id].reachable ? 'pi pi-check-circle' : 'pi pi-times-circle'"></i>
                                     <span class="font-semibold text-sm">{{ connectionStatus[proxy.id].reachable ? 'Reachable' : 'Unreachable' }}</span>
                                 </div>
                                 <div class="text-xs text-gray-300">{{ connectionStatus[proxy.id].target }}</div>
                                 <div v-if="!connectionStatus[proxy.id].reachable" class="text-xs mt-1 text-yellow-300">{{ connectionStatus[proxy.id].error }}</div>
                             </div>
                         </div>
                     </div>
                 </div>
             </template>
         </VueDraggable>

         <Dialog v-model:visible="showProxyDialog" :header="isEditMode ? 'Edit Proxy' : 'Add Proxy'" modal class="w-full max-w-lg">
             <div class="flex flex-col gap-4">
                 <div>
                     <label class="block text-sm font-medium mb-1">Name</label>
                     <InputText v-model="proxyForm.name" class="w-full" />
                 </div>
                 <div>
                     <label class="block text-sm font-medium mb-1">Listen Address</label>
                     <InputText v-model="proxyForm.listen_addr" placeholder=":5020" class="w-full" />
                 </div>
                 <div>
                     <label class="block text-sm font-medium mb-1">Target Address</label>
                     <InputText v-model="proxyForm.target_addr" placeholder="192.168.1.100:502" class="w-full" />
                 </div>
                 <div>
                     <label class="block text-sm font-medium mb-1">Description</label>
                     <InputText v-model="proxyForm.description" class="w-full" />
                 </div>
                 <div class="flex gap-4">
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">Connection Timeout (s)</label>
                         <InputNumber v-model="proxyForm.connection_timeout" :min="1" class="w-full" />
                     </div>
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">Read Timeout (s)</label>
                         <InputNumber v-model="proxyForm.read_timeout" :min="1" class="w-full" />
                     </div>
                 </div>
                 <div class="flex gap-4">
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">Max Retries</label>
                         <InputNumber v-model="proxyForm.max_retries" :min="0" class="w-full" />
                     </div>
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">Max Read Size (0=unlimited)</label>
                         <InputNumber v-model="proxyForm.max_read_size" :min="0" class="w-full" />
                     </div>
                 </div>
                 <div class="flex items-center gap-4">
                     <div class="flex items-center gap-2">
                         <Checkbox v-model="proxyForm.enabled" binary />
                         <span class="text-sm">Enabled</span>
                     </div>
                     <div class="flex items-center gap-2">
                         <Checkbox v-model="proxyForm.paused" binary />
                         <span class="text-sm">Paused</span>
                     </div>
                 </div>
                 <div>
                     <label class="block text-sm font-medium mb-1">Tags</label>
                     <Chips v-model="proxyForm.tags" class="w-full" placeholder="Add tags" />
                 </div>
             </div>
             <template #footer>
                 <Button label="Cancel" severity="secondary" @click="showProxyDialog = false" />
                 <Button :label="isEditMode ? 'Update' : 'Create'" @click="saveProxy" />
             </template>
         </Dialog>

         <Dialog v-model:visible="showLogsDialog" :header="`Logs - ${currentProxy?.name}`" modal class="w-full max-w-4xl">
             <div class="rounded-2xl p-4 font-mono text-sm h-[500px] overflow-y-auto bg-black/40 border border-white/5">
                 <div v-if="proxyLogs.length === 0" class="text-gray-500">No logs available</div>
                 <div v-else class="space-y-1">
                     <div v-for="(log, index) in proxyLogs" :key="index" class="border-b border-white/5 pb-1">
                          <span class="text-gray-400">[{{ formatTime(log.timestamp) }}]</span>
                         <span :class="getLogLevelColor(log.level)" class="mx-2 font-bold">{{ log.level }}</span>
                         <span class="text-white">{{ log.message }}</span>
                     </div>
                 </div>
             </div>
         </Dialog>
         <Toast />
         <ConfirmDialog />
     </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import axios from '../axios.js';
import Button from 'primevue/button';
import Tag from 'primevue/tag';
import Dialog from 'primevue/dialog';
import InputText from 'primevue/inputtext';
import InputNumber from 'primevue/inputnumber';
import Checkbox from 'primevue/checkbox';
import Chips from 'primevue/chips';
import Toast from 'primevue/toast';
import ConfirmDialog from 'primevue/confirmdialog';
import { useToast } from 'primevue/usetoast';
import { useConfirm } from 'primevue/useconfirm';
import { VueDraggable } from 'vue-draggable-plus';
import { useEventSource } from '../utils/eventSource';
import { getSeverity, getLogLevelColor, formatTime } from '../utils/helpers';
import { useAuthStore } from '../stores/auth';

const auth = useAuthStore();
const proxies = ref([]);
const loading = ref(true);
const editMode = ref(false);
const toast = useToast();
const confirm = useConfirm();
const router = useRouter();
let disconnectFn = null;

const testingProxy = ref(null);
const connectionStatus = ref({});
const unreachableCount = computed(() => Object.values(connectionStatus.value).filter((entry) => entry && !entry.reachable).length);

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

const showLogsDialog = ref(false);
const currentProxy = ref(null);
const proxyLogs = ref([]);

const onDragEnd = () => {
    const order = proxies.value.map(p => p.id);
    localStorage.setItem('proxy_order', JSON.stringify(order));
};

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

    watch(isConnected, (connected) => {
        if (!connected) {
            console.warn('SSE connection lost');
        }
    });

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
    }
};

const confirmDeleteProxy = (id) => {
    confirm.require({
        message: 'Are you sure you want to delete this proxy?',
        header: 'Confirm Delete',
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

const openProxyLogs = async (id) => {
    currentProxy.value = proxies.value.find(p => p.id === id);
    try {
        const res = await axios.get('/api/logs');
        const allLogs = res.data;
        proxyLogs.value = allLogs.filter(log => log.proxy_id === id);
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
        setTimeout(fetchProxies, 500);
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: e.response?.data || e.message, life: 5000 });
    }
};

const controlAllProxies = async (action) => {
    try {
        await axios.post('/api/proxies/control', { action });
        toast.add({ severity: 'success', summary: 'Success', detail: `All proxies ${action.replace('_all', '')} command sent`, life: 3000 });
        setTimeout(fetchProxies, 500);
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: e.response?.data || e.message, life: 5000 });
    }
};

const testConnectivity = async (proxy) => {
    testingProxy.value = proxy.id;
    try {
        const res = await axios.get('/api/system/diagnostics/connectivity');
        const proxyConnStatus = res.data[proxy.id];
        if (proxyConnStatus) {
            connectionStatus.value[proxy.id] = proxyConnStatus;
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

const goToLogs = () => {
    router.push('/logs');
};
</script>

<style scoped>
.drag-ghost {
    opacity: 0.5;
    border-radius: 24px !important;
}

.drag-active {
    transform: rotate(2deg) scale(1.02);
    z-index: 1000;
    box-shadow: 0 25px 50px rgba(168, 85, 247, 0.3) !important;
}

.glass-card {
    background: rgba(31, 41, 55, 0.5);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.glass-card:hover {
    border-color: rgba(255, 255, 255, 0.2);
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4), 0 0 20px rgba(168, 85, 247, 0.1);
    transform: translateY(-2px);
}
</style>
