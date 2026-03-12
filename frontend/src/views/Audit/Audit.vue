<template>
  <div class="p-6">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-900">Audit Log</h1>
      <button @click="exportLogs" class="bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700">
        Export JSON
      </button>
    </div>

    <div class="bg-white rounded-lg shadow overflow-x-auto">
      <table class="min-w-full">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Timestamp</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Action</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Resource</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Details</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="log in auditLogs" :key="log.id">
            <td class="px-6 py-4 whitespace-nowrap text-sm">{{ formatTimestamp(log.timestamp) }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm">{{ log.username || log.user_id }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">{{ log.action }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm">
              {{ log.resource_type }}{{ log.resource_id ? ':' + log.resource_id : '' }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm">
              <span v-if="log.success" class="text-green-600">✓ Success</span>
              <span v-else class="text-red-600">✗ Failed</span>
            </td>
            <td class="px-6 py-4 text-sm text-gray-500 max-w-xs truncate">{{ log.details }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="mt-4 flex justify-center space-x-2">
      <button @click="prevPage" :disabled="offset === 0" class="px-4 py-2 border rounded disabled:opacity-50">Previous</button>
      <span class="px-4 py-2">Page {{ Math.floor(offset / limit) + 1 }}</span>
      <button @click="nextPage" class="px-4 py-2 border rounded">Next</button>
    </div>
  </div>
</template>

<script>
import axios from '../axios';

export default {
  name: 'Audit',
  data() {
    return {
      auditLogs: [],
      limit: 50,
      offset: 0
    };
  },
  async mounted() {
    await this.loadLogs();
  },
  methods: {
    async loadLogs() {
      try {
        const response = await axios.get(`/api/audit/logs?limit=${this.limit}&offset=${this.offset}`);
        this.auditLogs = response.data;
      } catch (error) {
        console.error('Failed to load audit logs:', error);
      }
    },
    async exportLogs() {
      try {
        const response = await axios.get('/api/audit/logs/export');
        const blob = new Blob([JSON.stringify(response.data, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'audit_logs_' + new Date().toISOString() + '.json';
        a.click();
      } catch (error) {
        console.error('Failed to export logs:', error);
      }
    },
    prevPage() {
      if (this.offset >= this.limit) {
        this.offset -= this.limit;
        this.loadLogs();
      }
    },
    nextPage() {
      this.offset += this.limit;
      this.loadLogs();
    },
    formatTimestamp(ts) {
      return new Date(ts).toLocaleString();
    }
  }
};
</script>
