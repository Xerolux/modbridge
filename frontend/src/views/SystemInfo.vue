<template>
    <div class="p-2 sm:p-4 flex flex-col gap-4 w-full">
        <div class="flex items-center gap-3 mb-2 sm:mb-4">
          <h1 class="text-xl sm:text-2xl font-bold text-gray-800 dark:text-gray-200">{{ t('system.title') }}</h1>
          <div v-if="lastRefreshed" class="flex items-center gap-1.5 text-xs text-gray-400 dark:text-gray-500">
            <i class="pi pi-refresh text-[10px]" :class="{ 'pi-spin': isRefreshing }"></i>
            <span>{{ t('common.lastRefreshed') }}: {{ timeAgo }}</span>
          </div>
        </div>

        <div v-if="loading" class="flex justify-center">
            <i class="pi pi-spin pi-spinner text-3xl sm:text-4xl text-blue-500"></i>
        </div>

        <div v-else-if="loadError && !systemInfo.hostname" class="glass-card rounded-3xl border border-red-300 dark:border-red-500/30 p-6 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
            <div class="flex items-center gap-3">
                <i class="pi pi-exclamation-triangle text-red-500"></i>
                <span class="text-sm">{{ t('systemInfo.loadError') }}</span>
            </div>
            <button @click="fetchInfo" class="px-3 py-1.5 rounded-lg bg-gray-900 text-white text-sm hover:bg-gray-700 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200">{{ t('common.retry') }}</button>
        </div>

        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
            <Card class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10">
                <template #title><div class="text-lg sm:text-xl">{{ t('system.system') }}</div></template>
                <template #content>
                    <div class="space-y-2 text-sm sm:text-base">
                        <div class="flex justify-between">
                             <span class="text-gray-500 dark:text-gray-400">{{ t('system.uptime') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.uptime_human }}</span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.goVersion') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.go_version }}</span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.os') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.os }}</span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.arch') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.arch }}</span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.cpuCount') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.num_cpu }}</span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10">
                <template #title><div class="text-lg sm:text-xl">{{ t('system.memory') }}</div></template>
                <template #content>
                    <div class="space-y-2 text-sm sm:text-base">
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.memoryAlloc') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.memory_alloc_mb }} MB</span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.memorySys') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.memory_sys_mb }} MB</span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.memoryGc') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.memory_gc_mb }} MB</span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.goroutines') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.goroutines }}</span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10">
                <template #title><div class="text-lg sm:text-xl">{{ t('system.proxies') }}</div></template>
                <template #content>
                    <div class="space-y-2 text-sm sm:text-base">
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.totalProxies') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ systemInfo.total_proxies }}</span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.runningProxies') }}:</span>
                             <span class="font-semibold text-green-600 dark:text-green-400 text-right pl-2">{{ systemInfo.running_proxies }}</span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.stoppedProxies') }}:</span>
                             <span class="font-semibold text-red-600 dark:text-red-400 text-right pl-2">{{ systemInfo.total_proxies - systemInfo.running_proxies }}</span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10">
                <template #title><div class="text-lg sm:text-xl">{{ t('system.configuration') }}</div></template>
                <template #content>
                    <div class="space-y-2 text-sm sm:text-base">
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.logLevel') }}:</span>
                            <span class="font-semibold text-right pl-2">{{ config.log_level }}</span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.debugMode') }}:</span>
                            <span class="font-semibold text-right pl-2" :class="config.debug_mode ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">
                                {{ config.debug_mode ? t('common.enabled') : t('common.disabled') }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.metrics') }}:</span>
                            <span class="font-semibold text-right pl-2" :class="config.metrics_enabled ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">
                                {{ config.metrics_enabled ? t('common.enabled') : t('common.disabled') }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.tls') }}:</span>
                            <span class="font-semibold text-right pl-2" :class="config.tls_enabled ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">
                                {{ config.tls_enabled ? t('common.enabled') : t('common.disabled') }}
                            </span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10">
                <template #title><div class="text-lg sm:text-xl">{{ t('system.security') }}</div></template>
                <template #content>
                    <div class="space-y-2 text-sm sm:text-base">
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.rateLimiting') }}:</span>
                            <span class="font-semibold text-right pl-2" :class="config.rate_limit_enabled ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">
                                {{ config.rate_limit_enabled ? t('common.enabled') : t('common.disabled') }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.ipWhitelist') }}:</span>
                            <span class="font-semibold text-right pl-2" :class="config.ip_whitelist_enabled ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">
                                {{ config.ip_whitelist_enabled ? t('common.enabled') : t('common.disabled') }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.ipBlacklist') }}:</span>
                            <span class="font-semibold text-right pl-2" :class="config.ip_blacklist_enabled ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">
                                {{ config.ip_blacklist_enabled ? t('common.enabled') : t('common.disabled') }}
                            </span>
                        </div>
                        <div class="flex justify-between">
                                                         <span class="text-gray-500 dark:text-gray-400">{{ t('system.emailAlerts') }}:</span>
                            <span class="font-semibold text-right pl-2" :class="config.email_enabled ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">
                                {{ config.email_enabled ? t('common.enabled') : t('common.disabled') }}
                            </span>
                        </div>
                    </div>
                </template>
            </Card>

            <Card class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10">
                <template #title><div class="text-lg sm:text-xl">{{ t('system.serverControl') }}</div></template>
                <template #content>
                    <div class="flex flex-col gap-3">
                        <Button @click="refreshNow" :label="t('system.refresh')" icon="pi pi-refresh" :loading="isRefreshing" class="w-full p-3 sm:p-2" />
                        <Button v-if="auth.hasPermission('system:restart')" @click="restartSystem" :label="t('system.restart')" icon="pi pi-power-off" severity="warning" class="w-full p-3 sm:p-2" />
                    </div>
                </template>
            </Card>

            <Card class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10">
                <template #title><div class="text-lg sm:text-xl">{{ t('system.proxyControl') }}</div></template>
                <template #content>
                    <div class="flex flex-col gap-3">
                        <Button v-if="auth.hasPermission('proxy:control')" @click="startAllProxies" :label="t('system.startAllProxies')" icon="pi pi-play" severity="success" class="w-full p-3 sm:p-2" />
                        <Button v-if="auth.hasPermission('proxy:control')" @click="stopAllProxies" :label="t('system.stopAllProxies')" icon="pi pi-stop" severity="danger" class="w-full p-3 sm:p-2" />
                        <Button v-if="auth.hasPermission('proxy:control')" @click="restartAllProxies" :label="t('system.restartAllProxies')" icon="pi pi-refresh" severity="warning" class="w-full p-3 sm:p-2" />
                        <Button @click="downloadLogs" :label="t('system.downloadLogs')" icon="pi pi-download" severity="secondary" class="w-full p-3 sm:p-2" />
                    </div>
                </template>
            </Card>

            <Card class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10">
                <template #title><div class="text-lg sm:text-xl">{{ t('system.portManagement') }}</div></template>
                <template #content>
                    <div class="flex flex-col gap-3">
                        <Button @click="checkPorts" :label="t('system.checkPorts')" icon="pi pi-search" severity="info" class="w-full p-3 sm:p-2" :loading="portCheckLoading" />

                        <div v-if="portStatus.summary" class="grid grid-cols-3 gap-2 text-sm sm:text-base">
                            <div class="text-center p-2 bg-gray-200 dark:bg-gray-700 rounded">
                                <div class="text-gray-500 dark:text-gray-400">{{ t('system.total') }}</div>
                                <div class="font-bold">{{ portStatus.summary.total }}</div>
                            </div>
                            <div class="text-center p-2 bg-green-100 dark:bg-green-900 rounded">
                                <div class="text-gray-500 dark:text-gray-400">{{ t('system.free') }}</div>
                                <div class="font-bold text-green-600 dark:text-green-400">{{ portStatus.summary.free }}</div>
                            </div>
                            <div class="text-center p-2 bg-red-100 dark:bg-red-900 rounded">
                                <div class="text-gray-500 dark:text-gray-400">{{ t('system.inUse') }}</div>
                                <div class="font-bold text-red-600 dark:text-red-400">{{ portStatus.summary.in_use }}</div>
                            </div>
                        </div>

                        <div v-if="blockedPorts.length > 0" class="mt-2 p-2 bg-red-100 dark:bg-red-900 rounded text-sm">
                            <div class="font-bold text-red-600 dark:text-red-400 mb-2">⚠️ {{ t('system.blockedPorts') }}:</div>
                            <div v-for="(port, idx) in blockedPorts" :key="idx" class="mb-2 p-2 bg-gray-200 dark:bg-gray-700 rounded">
                                <div class="flex justify-between items-center mb-1">
                                    <span class="font-semibold">Port {{ port.port }}</span>
                                    <span class="text-xs text-gray-500 dark:text-gray-400">PID: {{ port.pid }}</span>
                                </div>
                                <div class="text-xs text-gray-600 dark:text-gray-300 mb-1">
                                    <div>Process: {{ port.process }} (User: {{ port.user }})</div>
                                    <div class="truncate text-gray-500 dark:text-gray-500">{{ port.command }}</div>
                                </div>
                                <Button v-if="auth.hasPermission('system:manage')" @click="releasePort(port)" :label="t('system.releasePort')" icon="pi pi-trash" severity="danger" size="small" class="w-full p-2" />
                            </div>
                        </div>

                        <div v-else-if="portStatus.summary" class="text-center p-2 bg-green-100 dark:bg-green-900 rounded text-green-600 dark:text-green-400">
                            ✓ {{ t('system.allPortsFree') }}
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
  import { downloadBlob } from '../utils/helpers';
  import { useAuthStore } from '../stores/auth';
  import { useI18n } from 'vue-i18n';
  import { useAutoRefresh } from '../utils/useAutoRefresh';
  import { REFRESH_INTERVALS } from '../utils/constants';

  const auth = useAuthStore();
  const { t } = useI18n();

  const loading = ref(true);
  const loadError = ref(null);
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

  const fetchInfo = async () => {
      try {
          loadError.value = null;
          const [infoRes, configRes] = await Promise.all([
              axios.get('/api/system/info'),
              axios.get('/api/config/system')
          ]);
          systemInfo.value = infoRes.data;
          config.value = configRes.data;
      } catch (e) {
          // Surface the error instead of silently leaving zeros/empty fields.
          loadError.value = e.response?.data || e.message || 'Failed to fetch system info';
      }
  };

  const { lastRefreshed, isRefreshing, refreshNow } = useAutoRefresh(fetchInfo, REFRESH_INTERVALS.SYSTEM_INFO);

  const timeAgo = ref('');
  let timeAgoTimer = null;

  const updateTimeAgo = () => {
    if (!lastRefreshed.value) { timeAgo.value = ''; return; }
    const diff = Math.floor((Date.now() - lastRefreshed.value.getTime()) / 1000);
    if (diff < 5) { timeAgo.value = t('common.justNow'); return; }
    if (diff < 60) { timeAgo.value = t('common.secondsAgo', { n: diff }); return; }
    if (diff < 120) { timeAgo.value = t('common.minuteAgo'); return; }
    timeAgo.value = t('common.minutesAgo', { n: Math.floor(diff / 60) });
  };

  const refreshInfo = () => {
      refreshNow();
  };

  const downloadLogs = async () => {
      try {
          const res = await axios.get('/api/logs/download', { responseType: 'blob' });
          downloadBlob(res.data, 'proxy.log');
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
    confirm.require({
        message: t('control.startAllConfirm'),
        header: t('common.confirm'),
        icon: 'pi pi-exclamation-triangle',
        accept: async () => {
            try {
                await axios.post('/api/proxies/control', { action: 'start_all' });
                toast.add({ severity: 'success', summary: 'Success', detail: 'All proxies started', life: 3000 });
                await fetchInfo();
            } catch (e) {
                toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to start proxies', life: 3000 });
            }
        }
    });
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
    confirm.require({
        message: t('control.stopAllConfirm'),
        header: t('common.confirm'),
        icon: 'pi pi-exclamation-triangle',
        accept: async () => {
            try {
                await axios.post('/api/proxies/control', { action: 'restart_all' });
                toast.add({ severity: 'success', summary: 'Success', detail: 'All proxies restarted', life: 3000 });
                await fetchInfo();
            } catch (e) {
                toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to restart proxies', life: 3000 });
            }
        }
    });
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

  onMounted(async () => {
      await fetchInfo();
      loading.value = false;
      timeAgoTimer = setInterval(updateTimeAgo, 5000);
  });

  onUnmounted(() => {
      if (timeAgoTimer) clearInterval(timeAgoTimer);
  });
  </script>
