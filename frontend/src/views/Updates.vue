<template><div class="p-2 sm:p-4 space-y-5"><h1 class="text-2xl font-bold">{{ t('update.title') }}</h1><p>{{ t('onlineUpdate.hint') }}</p><div v-if="verified" role="status" class="data-health">{{ t('onlineUpdate.verified') }}</div><div v-if="installError" role="alert" class="data-health data-health--stale">{{ installError }}</div>        <!-- ── Update Section ──────────────────────────────────── -->
        <Card v-if="auth.hasPermission('system:restart')" class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden transition-all duration-300 hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/10">
            <template #title>
              <div class="text-lg sm:text-xl flex items-center justify-between">
                <span class="flex items-center gap-2"><i class="pi pi-cloud-download"></i> {{ t('update.title') }}</span>
                <Badge
                  v-if="checked && !checkError"
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
                <Button v-if="updateData.update_available && !updateData.asset_unavailable" :label="t('update.install')" icon="pi pi-download" @click="confirmInstall" :disabled="updating || checking || checkError" size="small" />
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
            <Button :label="t('update.install')" icon="pi pi-download" @click="doInstall" :disabled="updating" size="small" />
          </div>
        </Dialog></div></template>
<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useAuthStore } from '../stores/auth';
import axios from '../axios';
import Card from 'primevue/card';
import Badge from 'primevue/badge';
import Button from 'primevue/button';
import Dialog from 'primevue/dialog';
import ProgressBar from 'primevue/progressbar';
const { t, locale } = useI18n();
const auth = useAuthStore();
const updateData = ref({});
const updateStatus = ref({state:'idle',progress:0});
const checking = ref(false);
const checked = ref(false);
const checkError = ref(false);
const updating = ref(false);
const showUpdateDialog = ref(false);
const installError = ref('');
const verified = ref(sessionStorage.getItem('modbridge_update_verified') === 'true');
sessionStorage.removeItem('modbridge_update_verified');
let disposed = false;
let pollTimer;
let targetVersion = sessionStorage.getItem('modbridge_update_target') || '';
let deadline = Number(sessionStorage.getItem('modbridge_update_deadline')) || 0;
const formatReleaseDate = iso => new Date(iso).toLocaleDateString(locale.value);
const checkUpdate = async () => {
  if (checking.value) return;
  checking.value = true;
  checkError.value = false;
  try { const response = await axios.get('/api/update/check'); if (!disposed) { updateData.value = response.data; checked.value = true; } }
  catch { if (!disposed) checkError.value = true; }
  finally { if (!disposed) checking.value = false; }
};
const clearPending = () => { sessionStorage.removeItem('modbridge_update_target'); sessionStorage.removeItem('modbridge_update_deadline'); };
const pollStatus = async () => {
  if (disposed || !updating.value) return;
  try {
    const health = await fetch('/api/health', { cache:'no-store', signal:AbortSignal.timeout(5000) });
    const body = health.ok ? await health.json() : {};
    if (disposed) return;
    if (String(body.version || '').replace(/^v/,'') === targetVersion.replace(/^v/,'')) {
      clearPending(); sessionStorage.setItem('modbridge_update_verified','true'); window.location.reload(); return;
    }
    const response = await axios.get('/api/update/status', { timeout:5000, skipAuth:true });
    if (disposed) return;
    updateStatus.value = response.data;
    if (response.data.state === 'error') {
      updating.value = false; clearPending(); installError.value = t('update.installFailed', { error:response.data.error || response.data.message }); return;
    }
  } catch { /* A restarting service is briefly unavailable; verify the version on the next attempt. */ }
  if (disposed) return;
  if (Date.now() > deadline) { updating.value = false; clearPending(); installError.value = t('onlineUpdate.timeout'); return; }
  pollTimer = setTimeout(pollStatus, 1500);
};
const confirmInstall = () => { if (!updating.value && !checkError.value) showUpdateDialog.value = true; };
const doInstall = async () => {
  if (updating.value) return;
  showUpdateDialog.value = false;
  updating.value = true;
  installError.value = '';
  targetVersion = updateData.value.latest_version;
  deadline = Date.now() + 11 * 60 * 1000;
  sessionStorage.setItem('modbridge_update_target',targetVersion);
  sessionStorage.setItem('modbridge_update_deadline',String(deadline));
  try { await axios.post('/api/update/perform'); if (!disposed) pollStatus(); }
  catch (error) {
    if (disposed) return;
    if (!error.response || error.response.status === 409) { pollStatus(); return; }
    updating.value = false; clearPending(); installError.value = t('update.installFailed', { error:typeof error.response.data === 'string' ? error.response.data : error.message });
  }
};
onMounted(() => { checkUpdate(); if (targetVersion && deadline > Date.now()) { updating.value = true; pollStatus(); } });
onUnmounted(() => { disposed = true; clearTimeout(pollTimer); });
</script>
