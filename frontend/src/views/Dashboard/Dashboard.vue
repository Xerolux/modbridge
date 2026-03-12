<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-gray-900 mb-6">Dashboard</h1>

    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-6 mb-6">
      <div class="bg-white rounded-lg shadow p-6">
        <div class="text-sm font-medium text-gray-500">Total Proxies</div>
        <div class="mt-2 text-3xl font-semibold text-gray-900">{{ stats.totalProxies }}</div>
        <div class="mt-1 text-sm text-green-600">{{ stats.activeProxies }} active</div>
      </div>

      <div class="bg-white rounded-lg shadow p-6">
        <div class="text-sm font-medium text-gray-500">Connected Devices</div>
        <div class="mt-2 text-3xl font-semibold text-gray-900">{{ stats.totalDevices }}</div>
        <div class="mt-1 text-sm text-blue-600">{{ stats.totalRequests }} requests</div>
      </div>

      <div class="bg-white rounded-lg shadow p-6">
        <div class="text-sm font-medium text-gray-500">System Uptime</div>
        <div class="mt-2 text-3xl font-semibold text-gray-900">{{ stats.uptime }}</div>
        <div class="mt-1 text-sm text-gray-500">since {{ stats.startTime }}</div>
      </div>

      <div class="bg-white rounded-lg shadow p-6">
        <div class="text-sm font-medium text-gray-500">Memory Usage</div>
        <div class="mt-2 text-3xl font-semibold text-gray-900">{{ stats.memoryUsage }}</div>
        <div class="mt-1 text-sm text-yellow-600">{{ stats.memoryPercent }}%</div>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
      <!-- Throughput Chart -->
      <div class="bg-white rounded-lg shadow p-6">
        <h3 class="text-lg font-semibold mb-4">Requests per Second</h3>
        <div class="h-64 bg-gray-50 rounded flex items-center justify-center text-gray-400">
          Chart: Real-time throughput
        </div>
      </div>

      <!-- Proxy Status -->
      <div class="bg-white rounded-lg shadow p-6">
        <h3 class="text-lg font-semibold mb-4">Proxy Status</h3>
        <div class="space-y-3">
          <div v-for="proxy in proxies" :key="proxy.id" class="flex items-center justify-between">
            <span class="text-sm font-medium">{{ proxy.name }}</span>
            <span class="px-2 py-1 text-xs rounded" :class="getProxyStatusClass(proxy)">
              {{ proxy.enabled ? (proxy.paused ? 'Paused' : 'Running') : 'Disabled' }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Recent Activity -->
    <div class="bg-white rounded-lg shadow p-6">
      <h3 class="text-lg font-semibold mb-4">Recent Activity</h3>
      <div class="space-y-3">
        <div v-for="activity in recentActivity" :key="activity.id" class="flex items-start">
          <div class="flex-shrink-0">
            <span class="text-2xl">{{ getActivityIcon(activity.type) }}</span>
          </div>
          <div class="ml-3">
            <p class="text-sm font-medium text-gray-900">{{ activity.title }}</p>
            <p class="text-sm text-gray-500">{{ activity.description }}</p>
            <p class="text-xs text-gray-400 mt-1">{{ formatTimestamp(activity.timestamp) }}</p>
          </div>
        </div>
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

        this.stats = {
          totalProxies: this.proxies.length,
          activeProxies: this.proxies.filter(p => p.enabled).length,
          totalDevices: devicesResp.data.length || 0,
          totalRequests: devicesResp.data.reduce((sum, d) => sum + (d.request_count || 0), 0),
          uptime: this.calculateUptime(),
          startTime: new Date().toLocaleDateString(),
          memoryUsage: '15 MB',
          memoryPercent: 2
        };
      } catch (error) {
        console.error('Failed to load dashboard data:', error);
      }
    },
    getProxyStatusClass(proxy) {
      if (!proxy.enabled) return 'bg-gray-100 text-gray-800';
      if (proxy.paused) return 'bg-yellow-100 text-yellow-800';
      return 'bg-green-100 text-green-800';
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
      // Simple uptime calculation
      return Math.floor(process.uptime() / 86400) + 'd ' + Math.floor((process.uptime() % 86400) / 3600) + 'h';
    }
  }
};
</script>
