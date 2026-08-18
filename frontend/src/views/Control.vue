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
                    :label="$t('control.addProxy')"
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
                            :model-value="group.proxies"
                            @update:model-value="onProxyReorder"
                            :disabled="!editMode"
                            :group="{ name: 'proxy-order', pull: false, put: false }"
                            handle=".drag-handle"
                            ghost-class="drag-ghost"
                            drag-class="drag-active"
                            :animation="180"
                            :delay="180"
                            :delay-on-touch-only="true"
                            :touch-start-threshold="4"
                            :fallback-tolerance="5"
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
                                            <button
                                                type="button"
                                                v-if="editMode"
                                                class="drag-handle shrink-0 cursor-grab active:cursor-grabbing flex items-center justify-center w-11 h-11 rounded-xl border-0 bg-transparent hover:bg-[var(--bg-soft)] transition-colors"
                                                :aria-label="$t('control.drag')"
                                                :title="$t('control.drag')"
                                            >
                                                <i class="pi pi-bars text-[var(--text-muted)] text-sm"></i>
                                            </button>
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
                                            :label="$t('control.start')"
                                            @click="controlProxy(proxy.id, 'start')"
                                            class="flex-1 min-h-[40px] rounded-2xl text-sm"
                                            size="small"
                                        />
                                        <Button
                                            v-if="auth.hasPermission('proxy:control') && proxy.status === 'Running'"
                                            icon="pi pi-stop"
                                            severity="danger"
                                            :label="$t('control.stop')"
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

         <Dialog v-model:visible="showProxyDialog" :header="isEditMode ? $t('control.editProxy') : $t('control.addProxy')" modal class="w-full max-w-lg">
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
                 <div>
                     <label class="block text-sm font-medium mb-1">{{ $t('control.form.deviceProfile') }}</label>
                     <Select
                         v-model="proxyForm.device_profile"
                         :options="deviceProfileOptions"
                         optionLabel="label"
                         optionValue="id"
                         optionGroupLabel="label"
                         optionGroupChildren="items"
                         filter
                         :filterPlaceholder="$t('control.form.deviceProfileFilter')"
                         :placeholder="$t('control.form.deviceProfilePlaceholder')"
                         class="w-full"
                         @change="applyDeviceProfile($event.value)"
                     />
                     <small class="block text-xs text-[var(--text-muted)]">{{ selectedProfileHint || $t('control.form.deviceProfileHint') }}</small>
                     <small class="block text-xs text-[var(--text-muted)] opacity-70">{{ $t('control.profiles.disclaimer') }}</small>
                 </div>
                 <div>
                     <label class="block text-sm font-medium mb-1">{{ $t('control.form.protocol') }}</label>
                     <Select
                         v-model="proxyForm.protocol"
                         :options="protocolOptions"
                         optionLabel="label"
                         optionValue="value"
                         class="w-full"
                     />
                     <small class="text-xs text-[var(--text-muted)]">{{ $t('control.form.protocolHint') }}</small>
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
                 <div class="flex flex-col sm:flex-row gap-4">
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">{{ $t('control.form.connectDelay') }}</label>
                         <InputNumber v-model="proxyForm.connect_delay_ms" :min="0" :max="60000" class="w-full" />
                     </div>
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">{{ $t('control.form.maxTargetConns') }}</label>
                         <InputNumber v-model="proxyForm.max_target_conns" :min="0" :max="100" class="w-full" />
                         <small class="text-xs text-[var(--text-muted)]">{{ $t('control.form.maxTargetConnsHint') }}</small>
                     </div>
                 </div>
                 <div class="flex flex-col sm:flex-row gap-4">
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">{{ $t('control.form.minRequestGap') }}</label>
                         <InputNumber v-model="proxyForm.min_request_gap_ms" :min="0" :max="10000" class="w-full" />
                         <small class="text-xs text-[var(--text-muted)]">{{ $t('control.form.minRequestGapHint') }}</small>
                     </div>
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">{{ $t('control.form.requestTimeout') }}</label>
                         <InputNumber v-model="proxyForm.request_timeout_ms" :min="0" :max="600000" class="w-full" />
                         <small class="text-xs text-[var(--text-muted)]">{{ $t('control.form.requestTimeoutHint') }}</small>
                     </div>
                 </div>
                 <div class="flex flex-col sm:flex-row gap-4">
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">{{ $t('control.form.cacheTtl') }}</label>
                         <InputNumber v-model="proxyForm.cache_ttl_ms" :min="0" :max="3600000" :disabled="!proxyForm.cache_enabled" class="w-full" />
                         <small class="text-xs text-[var(--text-muted)]">{{ $t('control.form.cacheTtlHint') }}</small>
                     </div>
                     <div class="flex-1">
                         <label class="block text-sm font-medium mb-1">{{ $t('control.form.pollInterval') }}</label>
                         <InputNumber v-model="proxyForm.poll_interval_ms" :min="0" :max="3600000" :disabled="!proxyForm.cache_enabled" class="w-full" />
                         <small class="text-xs text-[var(--text-muted)]">{{ $t('control.form.pollIntervalHint') }}</small>
                     </div>
                 </div>
                 <div v-if="isEditMode" class="rounded-xl border border-[var(--border-subtle)] p-3">
                     <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                         <div class="min-w-0">
                             <div class="text-sm font-medium">{{ $t('control.form.calibrate') }}</div>
                             <small class="block text-xs text-[var(--text-muted)]">{{ $t('control.form.calibrateHint') }}</small>
                             <small v-if="proxyForm.calibrated_at" class="block text-xs text-[var(--text-muted)]">
                                 {{ $t('control.form.calibratedAt', { when: new Date(proxyForm.calibrated_at).toLocaleString() }) }}
                             </small>
                             <Button
                                 v-if="proxyForm.last_calibration"
                                 :label="$t('control.form.calibrateShowLast')"
                                 icon="pi pi-history"
                                 severity="secondary"
                                 text
                                 size="small"
                                 class="px-0 mt-1"
                                 @click="showLastCalibration = true"
                             />
                         </div>
                         <Button
                             :label="$t('control.form.calibrateStart')"
                             icon="pi pi-gauge"
                             severity="secondary"
                             size="small"
                             class="w-full sm:w-auto shrink-0"
                             :loading="calibrating"
                             :disabled="calibrating"
                             @click="runCalibration"
                         />
                     </div>
                     <!-- A run takes tens of seconds and blocks every client for its
                          duration. Saying so, and counting, is the difference between
                          waiting and wondering whether the click registered at all. -->
                     <div v-if="calibrating" class="mt-3 space-y-2">
                         <ProgressBar :value="calibrationProgress" :show-value="false" style="height: 6px" />
                         <p class="text-xs text-[var(--text-muted)]">
                             {{ $t('control.form.calibrateRunning', { elapsed: calibrationElapsed, max: calibrationMaxSeconds }) }}
                         </p>
                     </div>
                     <div v-if="calibrationResult" ref="calibrationResultPanel" class="mt-3">
                         <CalibrationReport :report="calibrationResult" show-apply @apply="applyCalibration" />
                     </div>
                 </div>
                 <div class="flex items-center gap-4">
                     <div class="flex items-center gap-2">
                         <Checkbox v-model="proxyForm.cache_enabled" binary @change="onCacheToggle" />
                         <span class="text-sm">{{ $t('control.form.cacheEnabled') }}</span>
                     </div>
                 </div>
                 <small class="block text-xs text-[var(--text-muted)] -mt-2">{{ $t('control.form.cacheEnabledHint') }}</small>
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

         <Dialog v-model:visible="showLogsDialog" :header="$t('control.logsTitle', { name: currentProxy?.name })" modal class="w-full max-w-4xl">
              <div class="rounded-2xl p-4 font-mono text-sm h-[500px] overflow-y-auto bg-gray-100 dark:bg-black/40 border border-gray-200 dark:border-white/5">
                  <div v-if="proxyLogs.length === 0" class="text-gray-400 dark:text-gray-500">{{ $t('control.noLogs') }}</div>
                  <div v-else class="space-y-1">
                      <div v-for="(log, index) in proxyLogs" :key="index" class="border-b border-gray-200 dark:border-white/5 pb-1">
                           <span class="text-gray-500 dark:text-gray-400">[{{ formatTime(log.timestamp) }}]</span>
                         <span :class="getLogLevelColor(log.level)" class="mx-2 font-bold">{{ log.level }}</span>
                         <span class="text-surface-900 dark:text-white">{{ log.message }}</span>
                     </div>
                 </div>
             </div>
         </Dialog>
         <Dialog
             v-model:visible="showLastCalibration"
             :header="$t('control.form.calibrateLastTitle')"
             modal
             class="w-full max-w-lg"
         >
             <p v-if="proxyForm.calibrated_at" class="text-xs text-[var(--text-muted)] mb-3">
                 {{ $t('control.form.calibratedAt', { when: new Date(proxyForm.calibrated_at).toLocaleString() }) }}
             </p>
             <CalibrationReport :report="lastCalibrationReport" show-apply @apply="applyStoredCalibration" />
         </Dialog>
         <Menu ref="actionMenu" id="overlay_menu" :model="menuItems" :popup="true" />
         <Toast />
         <ConfirmDialog />
     </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, computed, nextTick } from 'vue';
import axios from '../axios.js';
import Button from 'primevue/button';
import Dialog from 'primevue/dialog';
import Menu from 'primevue/menu';
import InputText from 'primevue/inputtext';
import InputNumber from 'primevue/inputnumber';
import Checkbox from 'primevue/checkbox';
import ProgressBar from 'primevue/progressbar';
import CalibrationReport from '../components/CalibrationReport.vue';
import Select from 'primevue/select';
import Chips from 'primevue/inputtags';
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
import { deviceCategories, findDeviceProfile } from '../deviceProfiles';

const auth = useAuthStore();
const { t, te } = useI18n();
const proxies = ref([]);
const loading = ref(true);
const editMode = ref(false);
const searchQuery = ref('');
const sseConnected = ref(null);
const liveStale = ref(false);

const runningCount = computed(() => proxies.value.filter(p => p.status === 'Running').length);
const stoppedCount = computed(() => proxies.value.filter(p => p.status === 'Stopped').length);
const errorCount   = computed(() => proxies.value.filter(p => p.status === 'Error').length);
const toast = useToast();
const confirm = useConfirm();
let disconnectFn = null;
const pendingTimers = [];
let watchdogTimer = null;
let unwatchConnectedLive = null;

const testingProxy = ref(null);
const connectionStatus = ref({});

let unwatchConnected = null;
const actionMenu = ref();
const menuItems = ref([]);
const activeProxyForMenu = ref(null);

let unwatchData = null;
let pendingProxyUpdates = new Map();
let pendingProxyRemovals = new Set();
let proxyBatchFrame = null;

const normalizeProxy = (proxy) => ({
    ...proxy,
    tags: Array.isArray(proxy.tags)
        ? proxy.tags
        : String(proxy.tags || '').split(',').map(tag => tag.trim()).filter(Boolean)
});

const normalizeProxyList = (data) => Array.isArray(data) ? data.map(normalizeProxy) : [];

const defaultProxyForm = () => ({
    id: '',
    name: t('control.newProxyName'),
    listen_addr: ':5020',
    target_addr: '127.0.0.1:502',
    description: '',
    connection_timeout: 10,
    read_timeout: 30,
    max_retries: 3,
    max_read_size: 0,
    connect_delay_ms: 0,
    protocol: 'tcp',
    max_target_conns: 0,
    min_request_gap_ms: 0,
    request_timeout_ms: 0,
    device_profile: '',
    calibrated_at: '',
    cache_enabled: false,
    cache_ttl_ms: 0,
    poll_interval_ms: 0,
    enabled: true,
    paused: false,
    tags: []
});

const showProxyDialog = ref(false);
const isEditMode = ref(false);
const proxyForm = ref(defaultProxyForm());
const savingProxy = ref(false);

// Device profiles fill the form with settings known to work for a device class.
// The chosen profile is stored with the proxy so the dialog can show it again;
// the proxy's behaviour still follows the individual fields below it.
const deviceProfileOptions = computed(() =>
    deviceCategories.map((category) => ({
        label: t(`control.profiles.categories.${category.id}`),
        items: category.devices.map((device) => ({ id: device.id, label: device.label }))
    }))
);
const protocolOptions = computed(() => [
    { value: 'tcp', label: t('control.form.protocolTcp') },
    { value: 'rtu-tcp', label: t('control.form.protocolRtuTcp') }
]);
// The behaviour class carries the hint; a note is appended when the device has
// a quirk worth calling out.
const selectedProfileHint = computed(() => {
    const profile = findDeviceProfile(proxyForm.value.device_profile);
    if (!profile) return '';
    const classHint = t(`control.profiles.classes.${profile.class}`);
    return profile.note ? `${classHint} — ${t(`control.profiles.notes.${profile.note}`)}` : classHint;
});

// Calibration measures the device and reports; the numbers are only written
// into the form when the operator says so.
const calibrating = ref(false);
const calibrationResult = ref(null);
const calibrationResultPanel = ref(null);
const calibrationElapsed = ref(0);
let calibrationTimer = null;

// The server bounds a run at 90 seconds, so the bar is an honest fraction of
// that ceiling rather than an invented percentage. It stops just short of full:
// a run that finishes early should be announced by its result, not by a bar
// that sat at 100% for the last twenty seconds.
const calibrationMaxSeconds = 90;
const calibrationProgress = computed(() =>
    Math.min(97, Math.round((calibrationElapsed.value / calibrationMaxSeconds) * 100))
);

// A refused run is meant to be read by whoever pressed the button, so the
// server sends a code and its numbers and the sentence is built here. Older
// servers answer with plain text; that is shown as it comes.
const calibrationRefusal = (e) => {
    const payload = e.response?.data;
    const refusal = payload && typeof payload === 'object' ? payload.error : null;
    if (refusal?.code) {
        const key = `control.form.calibrateRefusals.${refusal.code}`;
        if (te(key)) return t(key, refusal.args || {});
    }
    return refusal?.text || (typeof payload === 'string' && payload) || e.message;
};

const runCalibration = async () => {
    calibrating.value = true;
    calibrationResult.value = null;
    calibrationElapsed.value = 0;
    clearInterval(calibrationTimer);
    calibrationTimer = setInterval(() => {
        calibrationElapsed.value += 1;
    }, 1000);
    try {
        const res = await axios.post(
            '/api/proxies/calibrate',
            { id: proxyForm.value.id },
            { timeout: 300000 }
        );
        calibrationResult.value = res.data;
        // The server files the report against the proxy; mirroring it here keeps
        // "show the last measurement" correct without a reload.
        proxyForm.value = {
            ...proxyForm.value,
            calibrated_at: new Date().toISOString(),
            last_calibration: res.data
        };
        // On a phone the result lands below the fold, which reads as nothing
        // having happened at all.
        await nextTick();
        calibrationResultPanel.value?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    } catch (e) {
        toast.add({
            severity: 'error',
            summary: t('common.error'),
            detail: calibrationRefusal(e),
            life: 8000
        });
    } finally {
        clearInterval(calibrationTimer);
        calibrationTimer = null;
        calibrating.value = false;
    }
};

const applyCalibration = (report) => {
    const recommended = (report || calibrationResult.value)?.recommended;
    if (!recommended) return;
    proxyForm.value = {
        ...proxyForm.value,
        min_request_gap_ms: recommended.min_request_gap_ms,
        max_target_conns: recommended.max_target_conns,
        read_timeout: recommended.read_timeout
    };
    toast.add({ severity: 'info', summary: t('control.form.calibrateApplied'), life: 3000 });
};

// The last run is kept with the proxy, so it can be read again without paying
// for another measurement. It is written by the server; a hand-edited config
// should not be able to break the dialog, hence the shape check.
const showLastCalibration = ref(false);
const lastCalibrationReport = computed(() => {
    const stored = proxyForm.value.last_calibration;
    return stored && typeof stored === 'object' ? stored : null;
});

const applyStoredCalibration = (report) => {
    applyCalibration(report);
    showLastCalibration.value = false;
};

const onCacheToggle = () => {
    if (!proxyForm.value.cache_enabled) {
        proxyForm.value.poll_interval_ms = 0;
    }
};

const applyDeviceProfile = (id) => {
    const profile = findDeviceProfile(id);
    if (!profile) return;
    proxyForm.value = { ...proxyForm.value, ...profile.values, device_profile: id };
    toast.add({
        severity: 'info',
        summary: t('control.profiles.applied'),
        detail: profile.label,
        life: 3000
    });
};

const showLogsDialog = ref(false);
const currentProxy = ref(null);
const proxyLogs = ref([]);

const groups = computed(() => {
    const groupMap = {};
    proxies.value.forEach(proxy => {
        let proxyGroups = [t('control.ungrouped')];
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
         return [{ name: t('control.allGroup'), proxies: proxies.value }];
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

const persistProxyOrder = () => {
    localStorage.setItem('proxy_order', JSON.stringify(proxies.value.map(proxy => proxy.id)));
};

// Group/search lists are derived arrays. Merge their new order back into the
// canonical proxy list so the order survives the next SSE event or refresh.
const onProxyReorder = (orderedGroup) => {
    const orderedIds = orderedGroup.map(proxy => proxy.id);
    const groupedIds = new Set(orderedIds);
    let nextIndex = 0;

    proxies.value = proxies.value.map(proxy => (
        groupedIds.has(proxy.id)
            ? orderedGroup[nextIndex++]
            : proxy
    ));
    persistProxyOrder();
};

const fetchProxies = async () => {
    try {
        const res = await axios.get('/api/proxies');
        proxies.value = applyProxyOrder(normalizeProxyList(res.data));
    } catch (e) {
        toast.add({ severity: 'error', summary: t('common.error'), detail: t('control.fetchProxiesFailed'), life: 5000 });
    }
};

const flushProxyUpdates = () => {
    if (pendingProxyRemovals.size > 0) {
        proxies.value = proxies.value.filter(p => !pendingProxyRemovals.has(p.id));
    }

    pendingProxyUpdates.forEach((proxyData) => {
        if (pendingProxyRemovals.has(proxyData.id)) return;
        const normalized = normalizeProxy(proxyData);
        const index = proxies.value.findIndex(p => p.id === normalized.id);
        if (index !== -1) {
            proxies.value[index] = normalized;
        } else {
            proxies.value.push(normalized);
        }
    });

    pendingProxyUpdates.clear();
    pendingProxyRemovals.clear();
    proxyBatchFrame = null;
};

const scheduleProxyFlush = () => {
    if (!proxyBatchFrame) proxyBatchFrame = requestAnimationFrame(flushProxyUpdates);
};

onMounted(async () => {
    await fetchProxies();
    loading.value = false;

    const { data, disconnect, isConnected, lastMessageAt } = useEventSource('/api/proxies/stream');
    disconnectFn = disconnect;

    // Polling fallback: if SSE drops or stops delivering data for >30s, poll
    // /api/proxies periodically so the control view never freezes on stale data.
    const LIVE_STALE_MS = 30000;
    const checkLiveness = () => {
        const now = Date.now();
        const stale = !isConnected.value || lastMessageAt.value === 0 || (now - lastMessageAt.value) > LIVE_STALE_MS;
        liveStale.value = stale;
        if (stale) {
            fetchProxies();
        }
    };
    watchdogTimer = setInterval(checkLiveness, 15000);
    // Also react immediately when connectivity toggles.
    unwatchConnectedLive = watch(isConnected, () => checkLiveness());

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
                    pendingProxyUpdates.set(proxyData.id, proxyData);
                    pendingProxyRemovals.delete(proxyData.id);
                    scheduleProxyFlush();
                }
                break;
            case 'proxy_removed':
                if (eventData.proxy_id) {
                    pendingProxyUpdates.delete(eventData.proxy_id);
                    pendingProxyRemovals.add(eventData.proxy_id);
                    scheduleProxyFlush();
                }
                break;
        }
    });
});

onUnmounted(() => {
    pendingTimers.forEach(clearTimeout);
    pendingTimers.length = 0;
    if (watchdogTimer) clearInterval(watchdogTimer);
    if (unwatchConnectedLive) unwatchConnectedLive();
    if (unwatchConnected) unwatchConnected();
    if (unwatchData) unwatchData();
    if (proxyBatchFrame) cancelAnimationFrame(proxyBatchFrame);
    if (disconnectFn) {
        disconnectFn();
    }
});

const openAddProxyDialog = () => {
    isEditMode.value = false;
    proxyForm.value = defaultProxyForm();
    calibrationResult.value = null;
    showProxyDialog.value = true;
};

const openEditProxyDialog = (proxy) => {
    isEditMode.value = true;
    proxyForm.value = { ...proxy, protocol: proxy.protocol || 'tcp', device_profile: proxy.device_profile || '' };
    calibrationResult.value = null;
    showProxyDialog.value = true;
};

const saveProxy = async () => {
    if (savingProxy.value) return;
    savingProxy.value = true;
    try {
        if (isEditMode.value) {
            await axios.put('/api/proxies', proxyForm.value);
            toast.add({ severity: 'success', summary: t('common.success'), detail: t('control.proxyUpdated'), life: 3000 });
        } else {
            await axios.post('/api/proxies', proxyForm.value);
            toast.add({ severity: 'success', summary: t('common.success'), detail: t('control.proxyCreated'), life: 3000 });
        }
        showProxyDialog.value = false;
        await fetchProxies();
    } catch (e) {
        toast.add({ severity: 'error', summary: t('common.error'), detail: e.response?.data || e.message, life: 5000 });
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
                toast.add({ severity: 'success', summary: t('common.success'), detail: t('control.proxyDeleted'), life: 3000 });
                await fetchProxies();
            } catch (e) {
                toast.add({ severity: 'error', summary: t('common.error'), detail: e.response?.data || e.message, life: 5000 });
            }
        }
    });
};

const toggleMenu = (event, proxy) => {
    activeProxyForMenu.value = proxy;

    const items = [];

    if (auth.hasPermission('proxy:control')) {
        const controlGroup = {
            label: t('control.controlGroup'),
            items: []
        };

        if (proxy.status !== 'Stopped' && proxy.status !== 'Error') {
            controlGroup.items.push({
                label: t('control.restart'),
                icon: 'pi pi-refresh',
                command: () => controlProxy(proxy.id, 'restart')
            });
        }

        if (!proxy.paused && proxy.status === 'Running') {
            controlGroup.items.push({
                label: t('control.pause'),
                icon: 'pi pi-pause',
                command: () => controlProxy(proxy.id, 'pause')
            });
        } else if (proxy.paused) {
            controlGroup.items.push({
                label: t('control.resume'),
                icon: 'pi pi-play',
                command: () => controlProxy(proxy.id, 'resume')
            });
        }

        if (controlGroup.items.length > 0) {
            items.push(controlGroup);
        }
    }

    const settingsGroup = {
        label: t('control.manageGroup'),
        items: []
    };

    settingsGroup.items.push({
        label: t('control.testConnection'),
        icon: 'pi pi-search',
        command: () => testConnectivity(proxy)
    });

    settingsGroup.items.push({
        label: t('control.viewLogs'),
        icon: 'pi pi-eye',
        command: () => openProxyLogs(proxy.id)
    });

    if (auth.hasPermission('proxy:edit')) {
        settingsGroup.items.push({
            label: t('control.edit'),
            icon: 'pi pi-pencil',
            command: () => openEditProxyDialog(proxy)
        });
    }

    if (auth.hasPermission('proxy:delete')) {
        settingsGroup.items.push({
            label: t('common.delete'),
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
        toast.add({ severity: 'success', summary: t('common.success'), detail: t('control.controlCommandSent', { action: t('control.' + action) }), life: 3000 });
        pendingTimers.push(setTimeout(fetchProxies, 500));
    } catch (e) {
        toast.add({ severity: 'error', summary: t('common.error'), detail: e.response?.data || e.message, life: 5000 });
    }
};

const controlAllProxies = async (action) => {
    const actionKey = action === 'start_all' ? 'startAll' : 'stopAll';
    const message = action === 'start_all' ? t('control.startAllConfirm') : t('control.stopAllConfirm');
    confirm.require({
        message,
        header: t('common.confirm'),
        icon: 'pi pi-exclamation-triangle',
        accept: async () => {
            try {
                await axios.post('/api/proxies/control', { action });
                toast.add({ severity: 'success', summary: t('common.success'), detail: t('control.allControlCommandSent', { action: t('control.' + actionKey) }), life: 3000 });
                pendingTimers.push(setTimeout(fetchProxies, 500));
            } catch (e) {
                toast.add({ severity: 'error', summary: t('common.error'), detail: e.response?.data || e.message, life: 5000 });
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
                    summary: t('control.connectionOk'),
                    detail: t('control.connectionOkDetail', { name: proxy.name, target: proxyConnStatus.target }),
                    life: 4000
                });
            } else {
                toast.add({
                    severity: 'error',
                    summary: t('control.connectionFailed'),
                    detail: t('control.connectionFailedDetail', { target: proxyConnStatus.target, error: proxyConnStatus.error }),
                    life: 5000
                });
            }
        }
    } catch (e) {
        toast.add({ severity: 'error', summary: t('control.diagnosticError'), detail: e.message, life: 3000 });
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
