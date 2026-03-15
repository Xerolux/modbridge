<template>
  <div class="p-4 sm:p-6 flex flex-col gap-4 sm:gap-6">
    <h1 class="text-xl sm:text-2xl font-bold text-gray-200">Dashboard</h1>

    <!-- Stats Cards -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-6">
      <div class="bg-gray-800 rounded-lg shadow p-4 sm:p-6 border border-gray-700">
        <div class="text-xs sm:text-sm font-medium text-gray-400">Total Proxies</div>
        <div class="mt-2 text-2xl sm:text-3xl font-semibold text-white">{{ stats.totalProxies }}</div>
        <div class="mt-1 text-xs sm:text-sm text-green-400">{{ stats.activeProxies }} active</div>
      </div>

      <div class="bg-gray-800 rounded-lg shadow p-4 sm:p-6 border border-gray-700">
        <div class="text-xs sm:text-sm font-medium text-gray-400">Connected Devices</div>
        <div class="mt-2 text-2xl sm:text-3xl font-semibold text-white">{{ stats.totalDevices }}</div>
        <div class="mt-1 text-xs sm:text-sm text-blue-400">{{ stats.totalRequests }} requests</div>
      </div>

      <div class="bg-gray-800 rounded-lg shadow p-4 sm:p-6 border border-gray-700">
        <div class="text-xs sm:text-sm font-medium text-gray-400">System Uptime</div>
        <div class="mt-2 text-2xl sm:text-3xl font-semibold text-white">{{ stats.uptime }}</div>
        <div class="mt-1 text-xs sm:text-sm text-gray-500">since {{ stats.startTime }}</div>
      </div>

      <div class="bg-gray-800 rounded-lg shadow p-4 sm:p-6 border border-gray-700">
        <div class="text-xs sm:text-sm font-medium text-gray-400">Memory Usage</div>
        <div class="mt-2 text-2xl sm:text-3xl font-semibold text-white">{{ stats.memoryUsage }}</div>
        <div class="mt-1 text-xs sm:text-sm text-yellow-400">{{ stats.memoryPercent }}%</div>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 sm:gap-6">
      <!-- Throughput Chart -->
      <div class="bg-gray-800 rounded-lg shadow p-4 sm:p-6 border border-gray-700">
        <h3 class="text-base sm:text-lg font-semibold mb-4 text-gray-200">Requests per Second</h3>
        <div class="h-48 sm:h-64 bg-gray-900 rounded flex items-center justify-center text-gray-500 border border-gray-700">
          Chart: Real-time throughput
        </div>
      </div>

      <!-- Proxy Status -->
      <div class="bg-gray-800 rounded-lg shadow p-4 sm:p-6 border border-gray-700">
        <h3 class="text-base sm:text-lg font-semibold mb-4 text-gray-200">Proxy Status</h3>
        <div class="space-y-3">
          <div v-for="proxy in proxies" :key="proxy.id" class="flex items-center justify-between">
            <span class="text-sm font-medium text-gray-300">{{ proxy.name }}</span>
            <span class="px-2 py-1 text-xs rounded" :class="getProxyStatusClass(proxy)">
              {{ proxy.enabled ? (proxy.paused ? 'Paused' : 'Running') : 'Disabled' }}
            </span>
          </div>
          <div v-if="proxies.length === 0" class="text-sm text-gray-500 text-center py-4">No proxies configured</div>
        </div>
      </div>
    </div>

    <!-- Recent Activity -->
    <div class="bg-gray-800 rounded-lg shadow p-4 sm:p-6 border border-gray-700">
      <h3 class="text-base sm:text-lg font-semibold mb-4 text-gray-200">Recent Activity</h3>
      <div class="space-y-3">
        <div v-for="activity in recentActivity" :key="activity.id" class="flex items-start">
          <div class="flex-shrink-0">
            <span class="text-2xl">{{ getActivityIcon(activity.type) }}</span>
          </div>
          <div class="ml-3">
            <p class="text-sm font-medium text-gray-200">{{ activity.title }}</p>
            <p class="text-sm text-gray-400">{{ activity.description }}</p>
            <p class="text-xs text-gray-500 mt-1">{{ formatTimestamp(activity.timestamp) }}</p>
          </div>
        </div>
        <div v-if="recentActivity.length === 0" class="text-sm text-gray-500 text-center py-4">No recent activity</div>
      </div>
    </div>
  </div>
</template>

<script>
import axios from '../axios';

export default {
  name: 'Dashboard',
  data() {
    return {
      stats: {
        totalProxies: 0,
        activeProxies: 0,
        totalDevices: 0,
        totalRequests: 0,
        uptime: '0d 0h',
        startTime: null,
        memoryUsage: '0 MB',
        memoryPercent: 0
      },
      proxies: [],
      recentActivity: []
    };
  },
  async mounted() {
    await this.loadDashboardData();
    // Refresh every 30 seconds
    this.refreshInterval = setInterval(() => this.loadDashboardData(), 30000);
  },
  beforeUnmount() {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
    }
  },
  methods: {
    async loadDashboardData() {
      try {
        const [statusResp, devicesResp] = await Promise.all([
          axios.get('/api/status'),
          axios.get('/api/devices')
        ]);

        const config = statusResp.data.config || {};
        this.proxies = config.proxies || [];

        const sysData = statusResp.data;
        this.stats = {
          totalProxies: this.proxies.length,
          activeProxies: this.proxies.filter(p => p.enabled).length,
          totalDevices: devicesResp.data.length || 0,
          totalRequests: devicesResp.data.reduce((sum, d) => sum + (d.request_count || 0), 0),
          uptime: sysData.uptime_human || sysData.uptime || '—',
          startTime: sysData.start_time ? new Date(sysData.start_time).toLocaleDateString() : new Date().toLocaleDateString(),
          memoryUsage: sysData.memory_alloc_mb ? sysData.memory_alloc_mb + ' MB' : '—',
          memoryPercent: sysData.memory_percent || 0
        };
      } catch (error) {
        console.error('Failed to load dashboard data:', error);
      }
    },
    getProxyStatusClass(proxy) {
      if (!proxy.enabled) return 'bg-gray-700 text-gray-300';
      if (proxy.paused) return 'bg-yellow-900/50 text-yellow-400';
      return 'bg-green-900/50 text-green-400';
    },
    getActivityIcon(type) {
      const icons = {
        login: '🔐',
        config: '⚙️',
        proxy: '🔌',
        alert: '⚠️',
        device: '📟'
      };
      return icons[type] || '📌';
    },
    formatTimestamp(ts) {
      return new Date(ts).toLocaleString();
    },
    calculateUptime() {
      return '—';
    }
  }
};
</script>
