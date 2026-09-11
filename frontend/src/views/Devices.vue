<template>
  <div class="p-2 sm:p-4 flex flex-col gap-4 min-w-0">
    <DataHealth :failed="refreshError" :busy="isRefreshing" :last-success="lastRefreshed" @refresh="refreshNow" />
    <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 mb-4">
      <div class="flex flex-wrap items-center gap-3 min-w-0">
        <h1 class="text-2xl font-bold">{{ t('devices.title') }}</h1>
        <div v-if="lastRefreshed" class="flex items-center gap-1.5 text-xs text-gray-400 dark:text-gray-500">
          <i class="pi pi-refresh text-[10px]" :class="{ 'pi-spin': isRefreshing }"></i>
          <span>{{ t('common.lastRefreshed') }}: {{ timeAgo }}</span>
        </div>
      </div>
      <div class="flex flex-wrap gap-2">
        <Button
          :label="t('devices.exportCsv')"
          icon="pi pi-download"
          severity="secondary"
          @click="exportDevices"
        />
        <Button
          :label="t('devices.refresh')"
          icon="pi pi-refresh"
          :loading="isRefreshing"
          @click="refreshNow"
        />
      </div>
    </div>

    <PageState :loading="loading" :error="Boolean(error)" @retry="fetchDevices" />
    <div v-if="!loading && !error" class="flex flex-col gap-4">
      <div class="flex flex-col sm:flex-row gap-4 items-center w-full">
        <InputText
          v-model="searchTerm"
          :placeholder="t('devices.searchPlaceholder')"
          class="w-full sm:max-w-md"
        />
        <Dropdown
          v-model="selectedSort"
          :options="sortOptions"
          optionLabel="label"
          optionValue="value"
          :placeholder="t('devices.sortPlaceholder')"
          class="w-full sm:w-48"
        />
      </div>

      <DataTable
        :value="filteredDevices"
        :paginator="true"
        :rows="25"
        :rowsPerPageOptions="[10, 25, 50, 100]"
        :globalFilterFields="['name', 'ip', 'firstSeen']"
        :filters="filters"
        filterDisplay="row"
        responsiveLayout="scroll"
        stripedRows
        class="p-datatable-sm glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden"
      >
        <Column field="ip" :header="t('devices.ipAddress')" sortable></Column>
         <Column field="name" :header="t('devices.columnName')" sortable filterMatchMode="contains">
          <template #body="{ data }">
            <InputText
              v-model="data.name"
              @change="updateDeviceName(data)"
              class="w-full"
            />
          </template>
        </Column>
        <Column field="mac" :header="t('devices.macAddress')" sortable>
          <template #body="{ data }">
            <span
              :title="data.mac === 'unknown' ? t('devices.macUnknownTooltip') : data.mac"
              class="cursor-help"
            >
              {{ data.mac === 'unknown' ? 'N/A' : data.mac }}
            </span>
          </template>
        </Column>
        <Column field="firstSeen" :header="t('devices.firstSeen')" sortable>
          <template #body="{ data }">
            {{ formatDate(data.firstSeen) }}
          </template>
        </Column>
        <Column field="connectionCount" :header="t('devices.connections')" sortable>
          <template #body="{ data }">
            <Badge :value="data.connectionCount" :severity="getConnectionSeverity(data.connectionCount)" />
          </template>
        </Column>
        <Column :header="t('devices.actions')" :exportable="false">
          <template #body="{ data }">
            <div class="flex gap-2">
              <Button
                icon="pi pi-eye"
                size="small"
                text
                :title="t('devices.showDetails')"
                @click="showDeviceDetails(data)"
              />
              <Button
                icon="pi pi-history"
                size="small"
                text
                :title="t('devices.showHistory')"
                :loading="historyLoading && selectedDevice?.ip === data.ip"
                @click="showConnectionHistory(data.ip)"
              />
            </div>
          </template>
        </Column>
      </DataTable>
    </div>

    <Dialog v-model:visible="deviceDetailsVisible" :header="t('devices.deviceDetails')" class="w-full max-w-lg mx-4" modal>
      <div v-if="selectedDevice" class="flex flex-col gap-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="font-semibold">{{ t('devices.ipAddress') }}:</label>
            <p>{{ selectedDevice.ip }}</p>
          </div>
          <div>
            <label class="font-semibold">{{ t('devices.macAddress') }}:</label>
            <p>{{ selectedDevice.mac || 'N/A' }}</p>
          </div>
          <div>
            <label class="font-semibold">{{ t('devices.columnName') }}:</label>
            <p>{{ selectedDevice.name || 'N/A' }}</p>
          </div>
          <div>
            <label class="font-semibold">{{ t('devices.firstSeen') }}:</label>
            <p>{{ formatDate(selectedDevice.firstSeen) }}</p>
          </div>
          <div>
            <label class="font-semibold">{{ t('devices.lastSeen') }}</label>
            <p>{{ selectedDevice.lastSeen ? formatDate(selectedDevice.lastSeen) : 'N/A' }}</p>
          </div>
          <div>
            <label class="font-semibold">{{ t('devices.totalConnections') }}</label>
            <p>{{ selectedDevice.connectionCount }}</p>
          </div>
        </div>
      </div>
    </Dialog>

    <Dialog v-model:visible="historyVisible" :header="t('devices.connectionHistory')" class="w-full max-w-4xl mx-4" modal>
      <div class="flex flex-col gap-4">
        <div class="flex justify-between items-center">
          <h3>{{ selectedDevice?.ip }}</h3>
          <div class="flex gap-2">
            <Button
              :label="t('devices.exportCsv')"
              icon="pi pi-download"
              severity="secondary"
              size="small"
              @click="exportHistoryCSV"
            />
          </div>
        </div>
        <DataTable
          :value="connectionHistory"
          :paginator="true"
          :rows="10"
          :rowsPerPageOptions="[10, 25, 50]"
          stripedRows
          class="p-datatable-sm"
        >
          <Column field="proxyID" :header="t('devices.proxyId')" sortable></Column>
          <Column field="connectedAt" :header="t('devices.connectedAt')" sortable>
            <template #body="{ data }">
              {{ formatDateTime(data.connectedAt) }}
            </template>
          </Column>
          <Column field="requestCount" :header="t('devices.requestCount')" sortable></Column>
        </DataTable>
      </div>
    </Dialog>
  </div>
</template>

<script setup>
import PageState from '../components/PageState.vue';
import DataHealth from '../components/DataHealth.vue';
import { ref, onMounted, computed, onUnmounted } from 'vue';
import { useI18n } from 'vue-i18n';
import axios from '../axios.js';
import DataTable from 'primevue/datatable';
import Column from 'primevue/column';
import Button from 'primevue/button';
import InputText from 'primevue/inputtext';
import Dropdown from 'primevue/select';
import Badge from 'primevue/badge';
import Dialog from 'primevue/dialog';
import { useToast } from 'primevue/usetoast';
import { useAppStore } from '../stores/appStore';
import { formatDate, formatDateTime } from '../utils/helpers';
import { useAutoRefresh } from '../utils/useAutoRefresh';
import { REFRESH_INTERVALS } from '../utils/constants';

const { t } = useI18n();

const store = useAppStore();
const toast = useToast();

const devices = ref([]);
const loading = ref(true);
const error = ref(null);
const searchTerm = ref('');
const selectedSort = ref('name_asc');
const deviceDetailsVisible = ref(false);
const historyVisible = ref(false);
const historyLoading = ref(false);
const selectedDevice = ref(null);
const connectionHistory = ref([]);

// computed so the labels follow language switches without a remount
const sortOptions = computed(() => [
  { label: t('devices.sortNameAsc'), value: 'name_asc' },
  { label: t('devices.sortNameDesc'), value: 'name_desc' },
  { label: t('devices.sortIpAsc'), value: 'ip_asc' },
  { label: t('devices.sortConnectionsDesc'), value: 'connections_desc' },
  { label: t('devices.sortFirstSeenDesc'), value: 'firstSeen_desc' }
]);

const filters = ref({
  'ip': { value: null, matchMode: 'contains' },
  'name': { value: null, matchMode: 'contains' },
  'mac': { value: null, matchMode: 'contains' },
  'firstSeen': { value: null, matchMode: 'date' }
});

const filteredDevices = computed(() => {
  let result = [...devices.value];

  if (searchTerm.value) {
    const search = searchTerm.value.toLowerCase();
    result = result.filter(device =>
      device.ip.toLowerCase().includes(search) ||
      device.name?.toLowerCase().includes(search) ||
      device.mac?.toLowerCase().includes(search)
    );
  }

  switch (selectedSort.value) {
    case 'name_asc':
      result.sort((a, b) => (a.name || '').localeCompare(b.name || ''));
      break;
    case 'name_desc':
      result.sort((a, b) => (b.name || '').localeCompare(a.name || ''));
      break;
    case 'ip_asc':
      result.sort((a, b) => a.ip.localeCompare(b.ip));
      break;
    case 'connections_desc':
      result.sort((a, b) => b.connectionCount - a.connectionCount);
      break;
    case 'firstSeen_desc':
      result.sort((a, b) => new Date(b.firstSeen) - new Date(a.firstSeen));
      break;
  }

  return result;
});

const fetchDevices = async () => {
  loading.value = true;
  error.value = null;
  try {
    const res = await axios.get('/api/devices');
    devices.value = res.data.map(device => ({
      ...device,
      connectionCount: device.request_count || 0,
      firstSeen: device.first_seen,
      lastSeen: device.last_connect,
    }));
    loading.value = false;
  } catch (e) {
    error.value = e.response?.data?.error || e.message;
    toast.add({ severity: 'error', summary: t('common.error'), detail: t('devices.fetchError'), life: 3000 });
    loading.value = false;
  }
};

const updateDeviceName = async (device) => {
  try {
    await axios.put('/api/devices', {
      ip: device.ip,
      name: device.name
    });
    toast.add({ severity: 'success', summary: t('common.success'), detail: t('devices.nameUpdated'), life: 3000 });
  } catch (e) {
    toast.add({ severity: 'error', summary: t('common.error'), detail: t('devices.nameUpdateError'), life: 3000 });
  }
};

const showDeviceDetails = (device) => {
  selectedDevice.value = device;
  deviceDetailsVisible.value = true;
};

const showConnectionHistory = async (ip) => {
  selectedDevice.value = devices.value.find(d => d.ip === ip);
  historyLoading.value = true;
  try {
    const res = await axios.get(`/api/devices/history?device_ip=${ip}`);
    connectionHistory.value = res.data;
    historyVisible.value = true;
    historyLoading.value = false;
  } catch (e) {
    toast.add({ severity: 'error', summary: t('common.error'), detail: t('devices.fetchHistoryError'), life: 3000 });
    historyLoading.value = false;
  }
};

const exportHistoryCSV = async () => {
  // The store catches errors internally and returns false — check the result,
  // otherwise a failed export would still show the success toast.
  const ok = await store.exportDeviceHistory('csv');
  if (ok) {
    toast.add({ severity: 'success', summary: t('common.success'), detail: t('devices.exported'), life: 3000 });
  } else {
    toast.add({ severity: 'error', summary: t('common.error'), detail: t('devices.exportError'), life: 3000 });
  }
};

const exportDevices = async () => {
  const ok = await store.exportDeviceHistory('csv');
  if (ok) {
    toast.add({ severity: 'success', summary: t('common.success'), detail: t('devices.exported'), life: 3000 });
  } else {
    toast.add({ severity: 'error', summary: t('common.error'), detail: t('devices.exportError'), life: 3000 });
  }
};

const getConnectionSeverity = (count) => {
  if (count > 100) return 'danger';
  if (count > 50) return 'warn';
  return 'success';
};

const silentFetchDevices = async ({ signal } = {}) => {
  try {
    const res = await axios.get('/api/devices', { signal });
    devices.value = res.data.map(device => ({
      ...device,
      connectionCount: device.request_count || 0,
      firstSeen: device.first_seen,
      lastSeen: device.last_connect,
    }));
    error.value = null;
  } catch (e) {
    return false;
  }
};

const { lastRefreshed, isRefreshing, refreshError, refreshNow } = useAutoRefresh(silentFetchDevices, REFRESH_INTERVALS.DEVICES);

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

onMounted(() => {
  fetchDevices();
  timeAgoTimer = setInterval(updateTimeAgo, 5000);
});

onUnmounted(() => {
  if (timeAgoTimer) clearInterval(timeAgoTimer);
});
</script>
