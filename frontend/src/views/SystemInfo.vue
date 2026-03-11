<template>
    <div class="p-2 sm:p-4 flex flex-col gap-4 w-full">
        <h1 class="text-xl sm:text-2xl font-bold mb-2 sm:mb-4 text-gray-200">System Information</h1>

        <div v-if="loading" class="flex justify-center">
            <i class="pi pi-spin pi-spinner text-3xl sm:text-4xl text-blue-500"></i>
        </div>

        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
            <Card class="bg-gray-800 text-white shadow-md">
                <template #title><div class="text-lg sm:text-xl">System</div></template>
                <template #content>
                    <div class="space-y-2 text-sm sm:text-base">
                        <div class="flex justify-between">
                            <span class="text-gray-400">Uptime:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.uptime_human }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Go Version:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.go_version }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">OS:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.os }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Architecture:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.arch }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">CPU Cores:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.num_cpu }}</span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white shadow-md">
                <template #title><div class="text-lg sm:text-xl">Memory</div></template>
                <template #content>
                    <div class="space-y-2 text-sm sm:text-base">
                        <div class="flex justify-between">
                            <span class="text-gray-400">Allocated:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.memory_alloc_mb }} MB</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">System:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.memory_sys_mb }} MB</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Next GC:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.memory_gc_mb }} MB</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Goroutines:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.goroutines }}</span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white shadow-md">
                <template #title><div class="text-lg sm:text-xl">Proxies</div></template>
                <template #content>
                    <div class="space-y-2 text-sm sm:text-base">
                        <div class="flex justify-between">
                            <span class="text-gray-400">Total:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.total_proxies }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Running:</span>
                            <span class="font-semibold text-green-400 text-right pl-2">{{ systemInfo.running_proxies }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Stopped:</span>
                            <span class="font-semibold text-red-400 text-right pl-2">{{ systemInfo.total_proxies - systemInfo.running_proxies }}</span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white shadow-md">
                <template #title><div class="text-lg sm:text-xl">Configuration</div></template>
                <template #content>
                    <div class="space-y-2 text-sm sm:text-base">
                        <div class="flex justify-between">
                            <span class="text-gray-400">Log Level:</span>
                            <span class="font-semibold text-right pl-2">{{ config.log_level }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Debug Mode:</span>
                            <span class="font-semibold text-right pl-2" :class="config.debug_mode ? 'text-green-400' : 'text-red-400'">
                                {{ config.debug_mode ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Metrics:</span>
                            <span class="font-semibold text-right pl-2" :class="config.metrics_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.metrics_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">TLS:</span>
                            <span class="font-semibold text-right pl-2" :class="config.tls_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.tls_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white shadow-md">
                <template #title><div class="text-lg sm:text-xl">Security</div></template>
                <template #content>
                    <div class="space-y-2 text-sm sm:text-base">
                        <div class="flex justify-between">
                            <span class="text-gray-400">Rate Limiting:</span>
                            <span class="font-semibold text-right pl-2" :class="config.rate_limit_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.rate_limit_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">IP Whitelist:</span>
                            <span class="font-semibold text-right pl-2" :class="config.ip_whitelist_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.ip_whitelist_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">IP Blacklist:</span>
                            <span class="font-semibold text-right pl-2" :class="config.ip_blacklist_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.ip_blacklist_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Email Alerts:</span>
                            <span class="font-semibold text-right pl-2" :class="config.email_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.email_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white shadow-md">
                <template #title><div class="text-lg sm:text-xl">Server Control</div></template>
                <template #content>
                    <div class="flex flex-col gap-3">
                        <Button @click="refreshInfo" label="Refresh" icon="pi pi-refresh" class="w-full p-3 sm:p-2" />
                        <Button @click="restartSystem" label="Restart System" icon="pi pi-power-off" severity="warning" class="w-full p-3 sm:p-2" />
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white shadow-md">
                <template #title><div class="text-lg sm:text-xl">Proxy Control</div></template>
                <template #content>
                    <div class="flex flex-col gap-3">
                        <Button @click="startAllProxies" label="Start All Proxies" icon="pi pi-play" severity="success" class="w-full p-3 sm:p-2" />
                        <Button @click="stopAllProxies" label="Stop All Proxies" icon="pi pi-stop" severity="danger" class="w-full p-3 sm:p-2" />
                        <Button @click="restartAllProxies" label="Restart All Proxies" icon="pi pi-refresh" severity="warning" class="w-full p-3 sm:p-2" />
                        <Button @click="downloadLogs" label="Download Logs" icon="pi pi-download" severity="secondary" class="w-full p-3 sm:p-2" />
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white shadow-md">
                <template #title><div class="text-lg sm:text-xl">Port Management</div></template>
                <template #content>
                    <div class="flex flex-col gap-3">
                        <Button @click="checkPorts" label="Check Ports" icon="pi pi-search" severity="info" class="w-full p-3 sm:p-2" :loading="portCheckLoading" />

                        <div v-if="portStatus.summary" class="grid grid-cols-3 gap-2 text-sm sm:text-base">
                            <div class="text-center p-2 bg-gray-700 rounded">
                                <div class="text-gray-400">Total</div>
                                <div class="font-bold">{{ portStatus.summary.total }}</div>
                            </div>
                            <div class="text-center p-2 bg-green-900 rounded">
                                <div class="text-gray-400">Free</div>
                                <div class="font-bold text-green-400">{{ portStatus.summary.free }}</div>
                            </div>
                            <div class="text-center p-2 bg-red-900 rounded">
                                <div class="text-gray-400">In Use</div>
                                <div class="font-bold text-red-400">{{ portStatus.summary.in_use }}</div>
                            </div>
                        </div>

                        <div v-if="blockedPorts.length > 0" class="mt-2 p-2 bg-red-900 rounded text-sm">
                            <div class="font-bold text-red-400 mb-2">⚠️ Blocked Ports:</div>
                            <div v-for="(port, idx) in blockedPorts" :key="idx" class="mb-2 p-2 bg-gray-700 rounded">
                                <div class="flex justify-between items-center mb-1">
                                    <span class="font-semibold">Port {{ port.port }}</span>
                                    <span class="text-xs text-gray-400">PID: {{ port.pid }}</span>
                                </div>
                                <div class="text-xs text-gray-300 mb-1">
                                    <div>Process: {{ port.process }} (User: {{ port.user }})</div>
                                    <div class="truncate text-gray-500">{{ port.command }}</div>
                                </div>
                                <Button @click="releasePort(port)" label="Release Port" icon="pi pi-trash" severity="danger" size="small" class="w-full p-2" />
                            </div>
                        </div>

                        <div v-else-if="portStatus.summary" class="text-center p-2 bg-green-900 rounded text-green-400">
                            ✓ All ports are free
                        </div>
                    </div>
                </template>
            </Card>
        </div>

        <Toast />
        <ConfirmDialog />
    </div>
</template>

<script setup>
 import { ref, onMounted, onUnmounted } from 'vue';
 import axios from '../axios.js';
 import Card from 'primevue/card';
 import Button from 'primevue/button';
 import Toast from 'primevue/toast';
 import ConfirmDialog from 'primevue/confirmdialog';
 import { useToast } from 'primevue/usetoast';
 import { useConfirm } from 'primevue/useconfirm';

 const loading = ref(true);
 const toast = useToast();
 const confirm = useConfirm();

 const systemInfo = ref({
     uptime_seconds: 0,
     uptime_human: '',
     goroutines: 0,
     memory_alloc_mb: 0,
     memory_sys_mb: 0,
     memory_gc_mb: 0,
     num_cpu: 0,
     total_proxies: 0,
     running_proxies: 0,
     go_version: '',
     os: '',
     arch: ''
 });

 const config = ref({
     log_level: 'INFO',
     debug_mode: false,
     metrics_enabled: true,
     tls_enabled: false,
     rate_limit_enabled: true,
     ip_whitelist_enabled: false,
     ip_blacklist_enabled: false,
     email_enabled: false
 });

 const portCheckLoading = ref(false);
 const portStatus = ref({});
 const blockedPorts = ref([]);

 let refreshInterval = null;

 const fetchInfo = async () => {
     try {
         const [infoRes, configRes] = await Promise.all([
             axios.get('/api/system/info'),
             axios.get('/api/config/system')
         ]);
         systemInfo.value = infoRes.data;
         config.value = configRes.data;
     } catch (e) {
         console.error('Failed to fetch system info', e);
     }
 };

 const refreshInfo = () => {
     fetchInfo();
 };

 const downloadLogs = async () => {
     try {
         const res = await axios.get('/api/logs/download', { responseType: 'blob' });
         const url = window.URL.createObjectURL(new Blob([res.data]));
         const link = document.createElement('a');
         link.href = url;
         link.setAttribute('download', 'proxy.log');
         document.body.appendChild(link);
         link.click();
         link.remove();
         toast.add({ severity: 'success', summary: 'Success', detail: 'Logs downloaded', life: 3000 });
     } catch (e) {
         toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to download logs', life: 5000 });
     }
 };

 const restartSystem = () => {
     confirm.require({
         message: 'Are you sure you want to restart the system?',
         header: 'Confirm Restart',
         icon: 'pi pi-exclamation-triangle',
         accept: async () => {
             try {
                 await axios.post('/api/system/restart');
                 toast.add({ severity: 'info', summary: 'Restarting', detail: 'System is restarting...', life: 5000 });
             } catch (e) {
                 toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to restart', life: 3000 });
             }
         }
     });
 };

const startAllProxies = async () => {
    try {
        await axios.post('/api/proxies/control', { action: 'start_all' });
        toast.add({ severity: 'success', summary: 'Success', detail: 'All proxies started', life: 3000 });
        await fetchInfo();
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to start proxies', life: 3000 });
    }
};

const stopAllProxies = async () => {
    confirm.require({
        message: 'Are you sure you want to stop all proxies?',
        header: 'Confirm Stop',
        icon: 'pi pi-exclamation-triangle',
        accept: async () => {
            try {
                await axios.post('/api/proxies/control', { action: 'stop_all' });
                toast.add({ severity: 'success', summary: 'Success', detail: 'All proxies stopped', life: 3000 });
                await fetchInfo();
            } catch (e) {
                toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to stop proxies', life: 3000 });
            }
        }
    });
};

const restartAllProxies = async () => {
    try {
        await axios.post('/api/proxies/control', { action: 'restart_all' });
        toast.add({ severity: 'success', summary: 'Success', detail: 'All proxies restarted', life: 3000 });
        await fetchInfo();
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to restart proxies', life: 3000 });
    }
};

const checkPorts = async () => {
    portCheckLoading.value = true;
    try {
        const res = await axios.get('/api/system/ports/check');
        portStatus.value = res.data;
        blockedPorts.value = Object.values(res.data.ports || {}).filter(p => !p.is_open);

        if (blockedPorts.value.length > 0) {
            toast.add({
                severity: 'warn',
                summary: 'Blocked Ports',
                detail: `${blockedPorts.value.length} port(s) in use`,
                life: 5000
            });
        } else {
            toast.add({
                severity: 'success',
                summary: 'All Clear',
                detail: 'All ports are free',
                life: 3000
            });
        }
    } catch (e) {
        toast.add({
            severity: 'error',
            summary: 'Error',
            detail: 'Failed to check ports',
            life: 3000
        });
    } finally {
        portCheckLoading.value = false;
    }
};

const releasePort = (portInfo) => {
    confirm.require({
        message: `Kill process ${portInfo.process} (PID: ${portInfo.pid}) on port ${portInfo.port}?`,
        header: 'Confirm Port Release',
        icon: 'pi pi-exclamation-triangle',
        accept: async () => {
            try {
                await axios.post('/api/system/ports/release', {
                    port: portInfo.port,
                    pid: portInfo.pid
                });
                toast.add({
                    severity: 'success',
                    summary: 'Success',
                    detail: `Process ${portInfo.pid} killed, port ${portInfo.port} released`,
                    life: 3000
                });
                await checkPorts();
            } catch (e) {
                toast.add({
                    severity: 'error',
                    summary: 'Error',
                    detail: e.response?.data || 'Failed to release port',
                    life: 3000
                });
            }
        }
    });
};

 onMounted(() => {
     fetchInfo();
     loading.value = false;
     refreshInterval = setInterval(fetchInfo, 5000);
 });

 onUnmounted(() => {
     if (refreshInterval) {
         clearInterval(refreshInterval);
     }
 });
 </script>
