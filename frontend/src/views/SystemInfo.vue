<template>
    <div class="p-4 flex flex-col gap-4">
        <h1 class="text-2xl font-bold mb-4 text-gray-200">System Information</h1>

        <div v-if="loading" class="flex justify-center">
            <i class="pi pi-spin pi-spinner text-4xl text-blue-500"></i>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <Card class="bg-gray-800 text-white">
                <template #title>System</template>
                <template #content>
                    <div class="space-y-2">
                        <div class="flex justify-between">
                            <span class="text-gray-400">Uptime:</span>
                            <span class="font-semibold">{{ systemInfo.uptime_human }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Go Version:</span>
                            <span class="font-semibold">{{ systemInfo.go_version }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">OS:</span>
                            <span class="font-semibold">{{ systemInfo.os }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Architecture:</span>
                            <span class="font-semibold">{{ systemInfo.arch }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">CPU Cores:</span>
                            <span class="font-semibold">{{ systemInfo.num_cpu }}</span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white">
                <template #title>Memory</template>
                <template #content>
                    <div class="space-y-2">
                        <div class="flex justify-between">
                            <span class="text-gray-400">Allocated:</span>
                            <span class="font-semibold">{{ systemInfo.memory_alloc_mb }} MB</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">System:</span>
                            <span class="font-semibold">{{ systemInfo.memory_sys_mb }} MB</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Next GC:</span>
                            <span class="font-semibold">{{ systemInfo.memory_gc_mb }} MB</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Goroutines:</span>
                            <span class="font-semibold">{{ systemInfo.goroutines }}</span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white">
                <template #title>Proxies</template>
                <template #content>
                    <div class="space-y-2">
                        <div class="flex justify-between">
                            <span class="text-gray-400">Total:</span>
                            <span class="font-semibold">{{ systemInfo.total_proxies }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Running:</span>
                            <span class="font-semibold text-green-400">{{ systemInfo.running_proxies }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Stopped:</span>
                            <span class="font-semibold text-red-400">{{ systemInfo.total_proxies - systemInfo.running_proxies }}</span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white">
                <template #title>Configuration</template>
                <template #content>
                    <div class="space-y-2">
                        <div class="flex justify-between">
                            <span class="text-gray-400">Log Level:</span>
                            <span class="font-semibold">{{ config.log_level }}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Debug Mode:</span>
                            <span class="font-semibold" :class="config.debug_mode ? 'text-green-400' : 'text-red-400'">
                                {{ config.debug_mode ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Metrics:</span>
                            <span class="font-semibold" :class="config.metrics_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.metrics_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">TLS:</span>
                            <span class="font-semibold" :class="config.tls_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.tls_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white">
                <template #title>Security</template>
                <template #content>
                    <div class="space-y-2">
                        <div class="flex justify-between">
                            <span class="text-gray-400">Rate Limiting:</span>
                            <span class="font-semibold" :class="config.rate_limit_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.rate_limit_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">IP Whitelist:</span>
                            <span class="font-semibold" :class="config.ip_whitelist_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.ip_whitelist_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">IP Blacklist:</span>
                            <span class="font-semibold" :class="config.ip_blacklist_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.ip_blacklist_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-400">Email Alerts:</span>
                            <span class="font-semibold" :class="config.email_enabled ? 'text-green-400' : 'text-red-400'">
                                {{ config.email_enabled ? 'Enabled' : 'Disabled' }}
                            </span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white">
                <template #title>Server Control</template>
                <template #content>
                    <div class="flex flex-col gap-2">
                        <Button @click="refreshInfo" label="Refresh" icon="pi pi-refresh" class="w-full" />
                        <Button @click="restartSystem" label="Restart System" icon="pi pi-power-off" severity="warning" class="w-full" />
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white">
                <template #title>Software Update</template>
                <template #content>
                    <div class="space-y-3">
                        <div class="flex justify-between items-center">
                            <div>
                                <p class="font-semibold">Current Version</p>
                                <p class="text-gray-400">{{ systemInfo.app_version || 'Unknown' }}</p>
                            </div>
                            <div class="text-right">
                                <p class="font-semibold">Latest Version</p>
                                <p class="text-gray-400">{{ updateInfo.latest_version || 'Checking...' }}</p>
                            </div>
                        </div>

                        <div v-if="updateInfo.status === 'checking'" class="flex justify-center">
                            <i class="pi pi-spin pi-spinner text-2xl text-blue-500"></i>
                        </div>

                        <div v-else-if="updateInfo.status === 'up_to_date'" class="text-center">
                            <span class="text-green-400 font-semibold">You are using the latest version</span>
                        </div>

                        <div v-else-if="updateInfo.status === 'available'" class="text-center">
                            <span class="text-yellow-400 font-semibold">Update available</span>
                        </div>

                        <div class="flex flex-col gap-2">
                            <Button @click="checkForUpdate" label="Check for Updates" icon="pi pi-search" class="w-full" :loading="updateInfo.loading" />
                            <Button v-if="updateInfo.status === 'available'" @click="performUpdate" label="Update Now" icon="pi pi-download" severity="success" class="w-full" :loading="updateInfo.updating" />
                        </div>

                        <div v-if="updateInfo.message" class="text-sm p-3 bg-gray-700 rounded">
                            {{ updateInfo.message }}
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="bg-gray-800 text-white">
                <template #title>Proxy Control</template>
                <template #content>
                    <div class="flex flex-col gap-2">
                        <Button @click="startAllProxies" label="Start All Proxies" icon="pi pi-play" severity="success" class="w-full" />
                        <Button @click="stopAllProxies" label="Stop All Proxies" icon="pi pi-stop" severity="danger" class="w-full" />
                        <Button @click="restartAllProxies" label="Restart All Proxies" icon="pi pi-refresh" severity="warning" class="w-full" />
                        <Button @click="downloadLogs" label="Download Logs" icon="pi pi-download" severity="secondary" class="w-full" />
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
     arch: '',
     app_version: ''
 });

 const updateInfo = ref({
     status: 'checking',
     current_version: '',
     latest_version: '',
     message: '',
     loading: false,
     updating: false
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

 const checkForUpdate = async () => {
     updateInfo.value.loading = true;
     try {
         const res = await axios.get('/api/update/check');
         updateInfo.value = res.data;
         updateInfo.value.current_version = systemInfo.value.app_version || 'unknown';
     } catch (e) {
         toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to check for updates', life: 3000 });
         console.error('Failed to check for updates', e);
     } finally {
         updateInfo.value.loading = false;
     }
 };

 const performUpdate = () => {
     confirm.require({
         message: 'Are you sure you want to update the software? The system will restart during the update process.',
         header: 'Confirm Update',
         icon: 'pi pi-exclamation-triangle',
         accept: async () => {
             updateInfo.value.updating = true;
             try {
                 const res = await axios.post('/api/update/perform');
                 updateInfo.value = res.data;
                 toast.add({ severity: 'info', summary: 'Updating', detail: res.data.message, life: 5000 });

                 // Auto-refresh update status after 5 seconds
                 setTimeout(() => {
                     checkForUpdate();
                 }, 5000);
             } catch (e) {
                 toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to start update', life: 3000 });
                 console.error('Failed to perform update', e);
             } finally {
                 updateInfo.value.updating = false;
             }
         }
     });
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

 onMounted(() => {
     fetchInfo();
     loading.value = false;
     refreshInterval = setInterval(fetchInfo, 5000);
     checkForUpdate();
 });

 onUnmounted(() => {
     if (refreshInterval) {
         clearInterval(refreshInterval);
     }
 });
 </script>
