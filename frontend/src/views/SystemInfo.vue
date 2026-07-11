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

        <!-- ── Update Section ──────────────────────────────────── -->
        <Card class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10">
            <template #title>
              <div class="text-lg sm:text-xl flex items-center justify-between">
                <span class="flex items-center gap-2"><i class="pi pi-cloud-download"></i> {{ t('update.title') }}</span>
                <Badge
                  :severity="updateData.update_available ? 'warn' : 'success'"
                  :value="updateData.update_available ? t('update.available') : t('update.upToDate')"
                />
              </div>
            </template>
            <template #content>
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-3">
                <div class="p-3 rounded-xl border border-gray-200 dark:border-white/10 bg-white/50 dark:bg-gray-800/50">
                  <div class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400 mb-1">{{ t('update.installed') }}</div>
                  <div class="text-lg font-bold text-gray-800 dark:text-gray-200">{{ updateData.current_version || '—' }}</div>
                  <div class="text-xs text-gray-400 mt-1">{{ updateData.os }}/{{ updateData.arch }} · {{ updateData.go_version }}</div>
                </div>
                <div class="p-3 rounded-xl border border-gray-200 dark:border-white/10 bg-white/50 dark:bg-gray-800/50">
                  <div class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400 mb-1">{{ t('update.latest') }}</div>
                  <div class="text-lg font-bold text-gray-800 dark:text-gray-200">{{ updateData.latest_version || '—' }}</div>
                  <div class="text-xs text-gray-400 mt-1" v-if="updateData.published_at">{{ formatReleaseDate(updateData.published_at) }}</div>
                </div>
              </div>

              <div v-if="updateData.asset_unavailable" class="mb-3 p-2 rounded-lg bg-amber-500/10 border border-amber-500/30 text-xs text-amber-600 dark:text-amber-400">
                {{ t('update.assetUnavailable') }}
              </div>

              <div v-if="checkError" class="mb-3 p-2 rounded-lg bg-red-500/10 border border-red-500/30 text-xs text-red-600 dark:text-red-400">
                {{ t('update.checkFailed') }}
              </div>

              <div v-if="updateData.release_notes" class="mb-3">
                <pre class="text-xs text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-900/50 p-3 rounded-xl border border-gray-200 dark:border-white/10 whitespace-pre-wrap max-h-48 overflow-y-auto">{{ updateData.release_notes }}</pre>
              </div>

              <div class="flex flex-wrap gap-2">
                <Button :label="t('update.checkAgain')" icon="pi pi-refresh" severity="secondary" @click="checkUpdate" :loading="checking" size="small" />
                <Button v-if="updateData.update_available && !updateData.asset_unavailable" :label="t('update.install')" icon="pi pi-download" @click="confirmInstall" :disabled="updating" size="small" />
                <a v-if="updateData.release_url" :href="updateData.release_url" target="_blank" rel="noopener" class="text-xs text-purple-600 dark:text-purple-400 hover:underline self-center ml-1">{{ t('update.viewOnGithub') }}</a>
              </div>

              <div v-if="updating || updateStatus.state === 'done'" class="mt-3">
                <ProgressBar :value="updateStatus.progress" />
                <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">{{ t(`update.state.${updateStatus.state}`) }}</p>
                <p v-if="updateStatus.message" class="text-[10px] text-gray-400 mt-1">{{ updateStatus.message }}</p>
              </div>
            </template>
        </Card>

        <Dialog v-model:visible="showUpdateDialog" :header="t('update.confirmTitle')" :modal="true" class="w-11/12 sm:w-full max-w-[440px]">
          <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('update.confirmMessage') }}</p>
          <div class="flex justify-end gap-2 mt-4">
            <Button :label="t('common.cancel')" severity="secondary" @click="showUpdateDialog = false" size="small" />
            <Button :label="t('update.install')" icon="pi pi-download" @click="doInstall" size="small" />
          </div>
        </Dialog>
    </div>
</template>

 <script setup>
  import { ref, onMounted, onUnmounted } from 'vue';
  import axios from '../axios.js';
  import Card from 'primevue/card';
  import Button from 'primevue/button';
  import Toast from 'primevue/toast';
  import ConfirmDialog from 'primevue/confirmdialog';
  import Badge from 'primevue/badge';
  import ProgressBar from 'primevue/progressbar';
  import Dialog from 'primevue/dialog';
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

  // ── Update module state ──────────────────────────────────────
  const updateData = ref({
    current_version: '', latest_version: '', update_available: false,
    asset_unavailable: false, release_notes: '', release_url: '',
    published_at: '', os: '', arch: '', go_version: '',
  });
  const updateStatus = ref({ state: 'idle', progress: 0, message: '' });
  const checking = ref(false);
  const updating = ref(false);
  const checkError = ref(false);
  const showUpdateDialog = ref(false);
  let statusPollTimer = null;

  const formatReleaseDate = (iso) => {
    try {
      return new Date(iso).toLocaleDateString('de-DE', { year: 'numeric', month: 'short', day: 'numeric' });
    } catch { return iso; }
  };

  const checkUpdate = async () => {
    checking.value = true;
    checkError.value = false;
    try {
      const res = await axios.get('/api/update/check');
      updateData.value = res.data;
    } catch {
      checkError.value = true;
    } finally {
      checking.value = false;
    }
  };

  const confirmInstall = () => { showUpdateDialog.value = true; };

  const doInstall = async () => {
    showUpdateDialog.value = false;
    updating.value = true;
    try {
      await axios.post('/api/update/perform');
      statusPollTimer = setInterval(pollStatus, 1500);
    } catch (err) {
      const msg = err.response?.status === 409
        ? t('update.alreadyRunning')
        : t('update.installFailed', { error: err.response?.data || err.message });
      toast.add({ severity: 'error', summary: t('update.title'), detail: msg, life: 5000 });
      updating.value = false;
    }
  };

  const pollStatus = async () => {
    try {
      const res = await axios.get('/api/update/status');
      updateStatus.value = res.data;
      if (res.data.state === 'done') {
        clearInterval(statusPollTimer);
        statusPollTimer = null;
        toast.add({ severity: 'success', summary: t('update.title'), detail: t('update.installSuccess'), life: 3000 });
        setTimeout(() => window.location.reload(), 4000);
      } else if (res.data.state === 'error') {
        clearInterval(statusPollTimer);
        statusPollTimer = null;
        updating.value = false;
        toast.add({ severity: 'error', summary: t('update.title'), detail: t('update.installFailed', { error: res.data.error }), life: 8000 });
      }
    } catch {
      // Network error during restart is expected — keep polling
    }
  };

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
      checkUpdate(); // auto-check for updates in background
  });

  onUnmounted(() => {
      if (timeAgoTimer) clearInterval(timeAgoTimer);
      if (statusPollTimer) clearInterval(statusPollTimer);
  });
  </script>
