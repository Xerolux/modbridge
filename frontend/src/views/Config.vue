<template>
    <div class="p-2 sm:p-4 flex flex-col gap-4">
        <h1 class="text-xl sm:text-2xl font-bold mb-2 sm:mb-4 text-gray-800 dark:text-gray-200">{{ t('config.title') }}</h1>

        <div v-if="loading" class="flex justify-center">
            <i class="pi pi-spin pi-spinner text-4xl text-blue-500"></i>
        </div>

        <div v-else class="flex flex-col gap-6">
            <Tabs value="0">
                <TabList class="glass-card rounded-t-3xl text-gray-800 dark:text-gray-200 overflow-x-auto flex-nowrap whitespace-nowrap hide-scrollbar border border-gray-200 dark:border-white/10 border-b-0">
                    <Tab value="0" class="shrink-0">{{ t('config.proxies') }}</Tab>
                    <Tab value="1" class="shrink-0">{{ t('config.logging') }}</Tab>
                    <Tab v-if="auth.hasPermission('config:edit')" value="2" class="shrink-0">{{ t('config.security') }}</Tab>
                    <Tab v-if="auth.hasPermission('config:edit')" value="3" class="shrink-0">{{ t('config.email') }}</Tab>
                    <Tab v-if="auth.hasPermission('config:edit')" value="4" class="shrink-0">{{ t('config.backup') }}</Tab>
                    <Tab v-if="auth.hasPermission('config:edit')" value="5" class="shrink-0">{{ t('config.advanced') }}</Tab>
                </TabList>

                <TabPanels class="glass-card rounded-b-3xl text-surface-900 dark:text-white p-2 sm:p-4 border border-gray-200 dark:border-white/10 border-t-0">
                    <TabPanel value="0">
                        <ConfigForm />
                    </TabPanel>

                    <TabPanel value="1">
                        <div class="space-y-4">
                            <h3 class="text-lg font-semibold">{{ t('config.loggingConfig') }}</h3>

                            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.logLevel') }}</label>
                                    <Dropdown v-model="config.log_level" :options="logLevels" optionLabel="label" optionValue="value" class="w-full" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.logMaxSize') }}</label>
                                    <InputNumber v-model="config.log_max_size" :min="1" class="w-full" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.logMaxFiles') }}</label>
                                    <InputNumber v-model="config.log_max_files" :min="1" class="w-full" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.logMaxAgeDays') }}</label>
                                    <InputNumber v-model="config.log_max_age_days" :min="1" class="w-full" />
                                </div>
                            </div>

                            <Button @click="saveConfig" :disabled="!configLoaded" :label="t('config.saveLogging')" icon="pi pi-save" />
                        </div>
                    </TabPanel>

                    <TabPanel value="2">
                        <div class="space-y-6">
                            <div>
                                <h3 class="text-lg font-semibold mb-4">SSL/TLS</h3>
                                <div class="grid grid-cols-1 gap-4">
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.enableTLS') }}</label>
                                        <ToggleSwitch v-model="config.tls_enabled" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.certFile') }}</label>
                                        <InputText v-model="config.tls_cert_file" class="w-full" placeholder="/path/to/cert.pem" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.keyFile') }}</label>
                                        <InputText v-model="config.tls_key_file" class="w-full" placeholder="/path/to/key.pem" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.sessionTimeout') }}</label>
                                        <InputNumber v-model="config.session_timeout" :min="1" class="w-full" />
                                    </div>
                                </div>
                            </div>

                            <div>
                                <h3 class="text-lg font-semibold mb-4">CORS</h3>
                                <div class="grid grid-cols-1 gap-4">
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.corsOrigins') }}</label>
                                        <Chips v-model="config.cors_allowed_origins" class="w-full" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.corsMethods') }}</label>
                                        <Chips v-model="config.cors_allowed_methods" class="w-full" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.corsHeaders') }}</label>
                                        <Chips v-model="config.cors_allowed_headers" class="w-full" />
                                    </div>
                                </div>
                            </div>

                            <div>
                                <h3 class="text-lg font-semibold mb-4">{{ t('config.rateLimiting') }}</h3>
                                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.rateLimitEnabled') }}</label>
                                        <ToggleSwitch v-model="config.rate_limit_enabled" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.rateLimitRequests') }}</label>
                                        <InputNumber v-model="config.rate_limit_requests" :min="1" class="w-full" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.rateLimitBurst') }}</label>
                                        <InputNumber v-model="config.rate_limit_burst" :min="1" class="w-full" />
                                    </div>
                                </div>
                            </div>

                            <div>
                                <h3 class="text-lg font-semibold mb-4">{{ t('config.ipFiltering') }}</h3>
                                <div class="grid grid-cols-1 gap-4">
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.ipWhitelistEnabled') }}</label>
                                        <ToggleSwitch v-model="config.ip_whitelist_enabled" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.ipWhitelist') }}</label>
                                        <Chips v-model="config.ip_whitelist" class="w-full" placeholder="192.168.1.0/24" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.ipBlacklistEnabled') }}</label>
                                        <ToggleSwitch v-model="config.ip_blacklist_enabled" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.ipBlacklist') }}</label>
                                        <Chips v-model="config.ip_blacklist" class="w-full" placeholder="10.0.0.0/8" />
                                    </div>
                                </div>
                            </div>

                            <Button @click="saveConfig" :disabled="!configLoaded" :label="t('config.saveSecurity')" icon="pi pi-shield" />
                        </div>
                    </TabPanel>

                    <TabPanel value="3">
                        <div class="space-y-4">
                            <h3 class="text-lg font-semibold">{{ t('config.emailConfig') }}</h3>

                            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.emailEnabled') }}</label>
                                    <ToggleSwitch v-model="config.email_enabled" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.smtpServer') }}</label>
                                    <InputText v-model="config.email_smtp_server" class="w-full" placeholder="smtp.gmail.com" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.smtpPort') }}</label>
                                    <InputNumber v-model="config.email_smtp_port" :min="1" :max="65535" class="w-full" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.emailFrom') }}</label>
                                    <InputText v-model="config.email_from" class="w-full" placeholder="noreply@example.com" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.emailTo') }}</label>
                                    <InputText v-model="config.email_to" class="w-full" placeholder="admin@example.com" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.emailUsername') }}</label>
                                    <InputText v-model="config.email_username" class="w-full" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.emailPassword') }}</label>
                                    <Password v-model="config.email_password" :feedback="false" toggleMask class="w-full" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.alertOnError') }}</label>
                                    <ToggleSwitch v-model="config.email_alert_on_error" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.alertOnWarning') }}</label>
                                    <ToggleSwitch v-model="config.email_alert_on_warning" />
                                </div>
                            </div>

                            <Button @click="saveConfig" :disabled="!configLoaded" :label="t('config.saveEmail')" icon="pi pi-envelope" />
                        </div>
                    </TabPanel>

                    <TabPanel value="4">
                        <div class="space-y-4">
                            <h3 class="text-lg font-semibold">{{ t('config.backupSection') }}</h3>

                            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.backupEnabled') }}</label>
                                    <ToggleSwitch v-model="config.backup_enabled" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.backupInterval') }}</label>
                                    <Dropdown v-model="config.backup_interval" :options="backupIntervals" optionLabel="label" optionValue="value" class="w-full" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.backupRetention') }}</label>
                                    <InputNumber v-model="config.backup_retention" :min="1" class="w-full" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.backupPath') }}</label>
                                    <InputText v-model="config.backup_path" class="w-full" placeholder="./backups" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.backupDatabase') }}</label>
                                    <ToggleSwitch v-model="config.backup_database" />
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.backupConfig') }}</label>
                                    <ToggleSwitch v-model="config.backup_config" />
                                </div>
                            </div>

                            <Button @click="saveConfig" :disabled="!configLoaded" :label="t('config.saveBackup')" icon="pi pi-download" />
                        </div>
                    </TabPanel>

                    <TabPanel value="5">
                        <div class="space-y-6">
                            <div>
                                <h3 class="text-lg font-semibold mb-4">{{ t('config.advancedConfig') }}</h3>
                                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.debugMode') }}</label>
                                        <ToggleSwitch v-model="config.debug_mode" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.maxConnections') }}</label>
                                        <InputNumber v-model="config.max_connections" :min="1" class="w-full" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.metricsEnabled') }}</label>
                                        <ToggleSwitch v-model="config.metrics_enabled" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('config.metricsPort') }}</label>
                                        <InputText v-model="config.metrics_port" class="w-full" placeholder=":9090" />
                                    </div>
                                </div>
                            </div>

                            <div v-if="auth.hasPermission('config:edit')">
                                <h3 class="text-lg font-semibold mb-4">{{ t('config.passwordSection') }}</h3>
                                <div class="grid grid-cols-1 gap-4">
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('login.currentPassword') }}</label>
                                        <Password v-model="passwordForm.current_password" :feedback="false" toggleMask class="w-full" />
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">{{ t('login.newPassword') }}</label>
                                        <Password v-model="passwordForm.new_password" toggleMask class="w-full" />
                                        <div class="mt-2 p-3 bg-blue-500/10 border border-blue-500/30 rounded-lg">
                                            <p class="text-xs text-blue-600 dark:text-blue-300 font-medium mb-1">{{ t('login.passwordRequirements') }}:</p>
                                            <ul class="text-xs text-gray-500 dark:text-gray-400 space-y-1 ml-4 list-disc">
                                                <li>{{ t('login.passwordMinLength') }}</li>
                                                <li>{{ t('login.passwordComplexity') }}</li>
                                                <li>{{ t('login.passwordNotCommon') }}</li>
                                            </ul>
                                        </div>
                                    </div>
                                    <Button :label="t('login.changePassword')" icon="pi pi-key" @click="changePassword" />
                                </div>
                            </div>

                            <div v-if="auth.hasPermission('config:export')">
                                <h3 class="text-lg font-semibold mb-4">{{ t('config.configBackup') }}</h3>
                                <div class="flex gap-4">
                                    <Button :label="t('config.exportConfig')" icon="pi pi-download" @click="exportConfig" />
                                    <Button :label="t('config.importConfig')" icon="pi pi-upload" severity="secondary" @click="triggerImport" />
                                    <input type="file" ref="importFile" accept=".json" @change="importConfig" style="display: none" />
                                </div>
                            </div>

                            <div v-if="auth.hasPermission('system:restart')">
                                <h3 class="text-lg font-semibold mb-4">{{ t('config.systemActions') }}</h3>
                                <div class="flex gap-4">
                                    <Button :label="t('config.restartSystem')" icon="pi pi-refresh" severity="danger" @click="confirmRestart" />
                                </div>
                            </div>

                            <Button @click="saveConfig" :disabled="!configLoaded" :label="t('config.saveAdvanced')" icon="pi pi-cog" />
                        </div>
                    </TabPanel>
                </TabPanels>
            </Tabs>
        </div>
    </div>
</template>

<script setup>
  import { ref, computed, onMounted } from 'vue';
  import { useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import axios from '../axios.js';
  import Button from 'primevue/button';
  import Password from 'primevue/password';
  import Tabs from 'primevue/tabs';
  import TabList from 'primevue/tablist';
  import Tab from 'primevue/tab';
  import TabPanels from 'primevue/tabpanels';
  import TabPanel from 'primevue/tabpanel';
  import Dropdown from 'primevue/select';
  import InputNumber from 'primevue/inputnumber';
  import InputText from 'primevue/inputtext';
  import ToggleSwitch from 'primevue/toggleswitch';
  import Chips from 'primevue/inputtags';
  import { useToast } from 'primevue/usetoast';
  import { useConfirm } from 'primevue/useconfirm';
  import ConfigForm from '../components/ConfigForm.vue';
  import { useAppStore } from '../stores/appStore';
  import { useAuthStore } from '../stores/auth';

  const auth = useAuthStore();
  const router = useRouter();

 const loading = ref(true);
 const toast = useToast();
const { t } = useI18n();
 const confirm = useConfirm();
 const store = useAppStore();

 const passwordForm = ref({
     current_password: '',
     new_password: ''
 });

 const config = ref({
     log_level: 'INFO',
     log_max_size: 100,
     log_max_files: 10,
     log_max_age_days: 30,
     tls_enabled: false,
     tls_cert_file: '',
     tls_key_file: '',
     session_timeout: 24,
     cors_allowed_origins: ['http://localhost:8080', 'http://localhost:3000'],
     cors_allowed_methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
     cors_allowed_headers: ['Content-Type', 'Authorization', 'X-CSRF-Token'],
     rate_limit_enabled: true,
     rate_limit_requests: 60,
     rate_limit_burst: 100,
     ip_whitelist_enabled: false,
     ip_whitelist: [],
     ip_blacklist_enabled: false,
     ip_blacklist: [],
     email_enabled: false,
     email_smtp_server: '',
     email_smtp_port: 587,
     email_from: '',
     email_to: '',
     email_username: '',
     email_password: '',
     email_alert_on_error: true,
     email_alert_on_warning: false,
     backup_enabled: true,
     backup_interval: 'daily',
     backup_retention: 7,
     backup_path: './backups',
     backup_database: true,
     backup_config: true,
     metrics_enabled: true,
     metrics_port: ':9090',
     debug_mode: false,
     max_connections: 1000
 });

 const logLevels = [
     { label: 'DEBUG', value: 'DEBUG' },
     { label: 'INFO', value: 'INFO' },
     { label: 'WARN', value: 'WARN' },
     { label: 'ERROR', value: 'ERROR' }
 ];

 const backupIntervals = computed(() => [
     { label: t('config.intervalHourly'), value: 'hourly' },
     { label: t('config.intervalDaily'), value: 'daily' },
     { label: t('config.intervalWeekly'), value: 'weekly' }
 ]);

 const importFile = ref(null);

 // Until the server config has been read once, the values above are only
 // placeholders. Saving them would overwrite the real configuration — the CORS
 // origins in particular, which can lock the operator out of the UI.
 const configLoaded = ref(false);

 const fetchConfig = async () => {
     try {
         const res = await axios.get('/api/config/system');
         config.value = { ...config.value, ...res.data };
         configLoaded.value = true;
     } catch (e) {
         configLoaded.value = false;
         toast.add({ severity: 'error', summary: t('common.error'), detail: t('config.loadError'), life: 5000 });
     }
 };

 onMounted(async () => {
     await Promise.all([
         store.fetchWebPort(),
         store.fetchProxies(),
         fetchConfig()
     ]);
     loading.value = false;
 });

 const saveConfig = async () => {
     if (!configLoaded.value) {
         toast.add({ severity: 'warn', summary: t('common.error'), detail: t('config.loadError'), life: 5000 });
         return;
     }
     try {
         await axios.put('/api/config/system', config.value);
         toast.add({ severity: 'success', summary: t('common.success'), detail: t('config.saved'), life: 3000 });
     } catch (e) {
         const detail = typeof e.response?.data === 'string' ? e.response.data : e.message;
         toast.add({ severity: 'error', summary: t('common.error'), detail: t('config.saveError'), life: 5000 });
     }
 };

 const changePassword = async () => {
     try {
         await axios.post('/api/config/password', passwordForm.value);
         toast.add({ severity: 'success', summary: t('common.success'), detail: t('config.passwordChanged'), life: 3000 });
         passwordForm.value = { current_password: '', new_password: '' };
         await auth.logout();
         await router.replace('/login');
     } catch (e) {
         let errorMsg = typeof e.response?.data === 'string' ? e.response.data : e.message;
         // Provide user-friendly error messages for common password validation errors
         if (typeof errorMsg === 'string') {
             if (errorMsg.includes('at least 8 characters')) {
                 errorMsg = t('config.passwordErrorMinLength');
             } else if (errorMsg.includes('at least 3 of')) {
                 errorMsg = t('config.passwordErrorComplexity');
             } else if (errorMsg.includes('too common')) {
                 errorMsg = t('config.passwordErrorTooCommon');
             }
         }
         toast.add({ severity: 'error', summary: t('common.error'), detail: errorMsg, life: 5000 });
     }
 };

 const exportConfig = async () => {
     try {
         const res = await axios.get('/api/config/export', { responseType: 'blob' });
         const url = window.URL.createObjectURL(new Blob([res.data]));
         const link = document.createElement('a');
         link.href = url;
         link.setAttribute('download', 'config.json');
         document.body.appendChild(link);
         link.click();
         link.remove();
         toast.add({ severity: 'success', summary: t('common.success'), detail: t('config.exported'), life: 3000 });
     } catch (e) {
         toast.add({ severity: 'error', summary: t('common.error'), detail: t('config.exportError'), life: 5000 });
     }
 };

 const triggerImport = () => {
     importFile.value.click();
 };

 const importConfig = async (event) => {
     const file = event.target.files[0];
     if (!file) return;

     try {
         const formData = new FormData();
         formData.append('file', file);
         await axios.post('/api/config/import', formData, {
             headers: { 'Content-Type': 'multipart/form-data' }
         });
         toast.add({ severity: 'success', summary: t('common.success'), detail: t('config.imported'), life: 3000 });
         await store.fetchProxies();
         await fetchConfig();
     } catch (e) {
         toast.add({ severity: 'error', summary: t('common.error'), detail: t('config.importError'), life: 5000 });
     }
     event.target.value = '';
 };

 const confirmRestart = () => {
     confirm.require({
         message: t('config.confirmRestartMessage'),
         header: t('common.confirm'),
         icon: 'pi pi-exclamation-triangle',
         accept: async () => {
             try {
                 await axios.post('/api/system/restart');
                 toast.add({ severity: 'info', summary: t('common.info'), detail: t('config.restarting'), life: 3000 });
             } catch (e) {
                 toast.add({ severity: 'error', summary: t('common.error'), detail: t('config.restartFailed'), life: 3000 });
             }
         }
     });
 };
 </script>
