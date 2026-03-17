<template>
  <div class="p-2 sm:p-4 flex flex-col gap-4 w-full">
    <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 mb-2 sm:mb-4">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-200">Audit Log</h1>
      <div class="flex gap-2 w-full sm:w-auto">
        <Button
          label="Export JSON"
          icon="pi pi-download"
          severity="success"
          @click="exportLogs"
          class="flex-1 sm:flex-none"
        />
        <Button
          label="Refresh"
          icon="pi pi-refresh"
          @click="loadLogs"
          class="flex-1 sm:flex-none"
        />
      </div>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <i class="pi pi-spin pi-spinner text-4xl text-blue-500"></i>
    </div>

    <div v-else-if="error" class="flex flex-col items-center justify-center py-12">
      <i class="pi pi-exclamation-triangle text-4xl text-red-500 mb-4"></i>
      <p class="text-red-400">{{ error }}</p>
      <Button @click="loadLogs" label="Retry" class="mt-4" />
    </div>

    <div v-else class="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
      <DataTable
        :value="auditLogs"
        :paginator="auditLogs.length >= limit"
        :rows="limit"
        :rowsPerPageOptions="[25, 50, 100]"
        stripedRows
        responsiveLayout="scroll"
        class="p-datatable-sm"
        :globalFilterFields="['action', 'username', 'resource_type']"
      >
        <Column field="timestamp" header="Timestamp" sortable>
          <template #body="{ data }">
            <span class="text-gray-300 text-sm">{{ formatTimestamp(data.timestamp) }}</span>
          </template>
        </Column>
        <Column field="username" header="User" sortable>
          <template #body="{ data }">
            <div class="flex items-center gap-2">
              <i class="pi pi-user text-gray-400 text-sm"></i>
              <span class="text-gray-200">{{ data.username || data.user_id || 'System' }}</span>
            </div>
          </template>
        </Column>
        <Column field="action" header="Action" sortable>
          <template #body="{ data }">
            <Tag :value="data.action" :severity="getActionSeverity(data.action)" />
          </template>
        </Column>
        <Column field="resource" header="Resource">
          <template #body="{ data }">
            <span class="text-gray-300">
              {{ data.resource_type }}{{ data.resource_id ? ':' + data.resource_id : '' }}
            </span>
          </template>
        </Column>
        <Column field="success" header="Status" sortable>
          <template #body="{ data }">
            <Tag
              :value="data.success ? 'Success' : 'Failed'"
              :severity="data.success ? 'success' : 'danger'"
            />
          </template>
        </Column>
        <Column field="details" header="Details">
          <template #body="{ data }">
            <span class="text-gray-400 text-sm truncate block max-w-[200px]" :title="data.details">
              {{ data.details || '-' }}
            </span>
          </template>
        </Column>
        <template #empty>
          <div class="text-center py-8 text-gray-500">
            <i class="pi pi-history text-4xl mb-2 block"></i>
            <p>No audit logs found</p>
          </div>
        </template>
      </DataTable>
    </div>

    <Toast />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import axios from '../../axios.js';
import DataTable from 'primevue/datatable';
import Column from 'primevue/column';
import Button from 'primevue/button';
import Tag from 'primevue/tag';
import Toast from 'primevue/toast';
import { useToast } from 'primevue/usetoast';

const auditLogs = ref([]);
const loading = ref(true);
const error = ref(null);
const limit = ref(50);
const toast = useToast();

const loadLogs = async () => {
  loading.value = true;
  error.value = null;
  try {
    const response = await axios.get(`/api/audit/logs?limit=${limit.value}&offset=0`);
    auditLogs.value = response.data || [];
  } catch (e) {
    error.value = e.response?.data || 'Failed to load audit logs';
    console.error('Failed to load audit logs:', e);
  } finally {
    loading.value = false;
  }
};

const exportLogs = async () => {
  try {
    const response = await axios.get('/api/audit/logs/export');
    const blob = new Blob([JSON.stringify(response.data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `audit_logs_${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    toast.add({ severity: 'success', summary: 'Success', detail: 'Audit logs exported', life: 3000 });
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to export logs', life: 5000 });
  }
};

const formatTimestamp = (ts) => {
  if (!ts) return '-';
  return new Date(ts).toLocaleString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  });
};

const getActionSeverity = (action) => {
  if (!action) return 'secondary';
  const actionLower = action.toLowerCase();
  if (actionLower.includes('delete') || actionLower.includes('remove')) return 'danger';
  if (actionLower.includes('create') || actionLower.includes('add')) return 'success';
  if (actionLower.includes('update') || actionLower.includes('edit')) return 'info';
  if (actionLower.includes('login') || actionLower.includes('logout')) return 'warn';
  return 'secondary';
};

onMounted(() => {
  loadLogs();
});
</script>
