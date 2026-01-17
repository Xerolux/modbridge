<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { useEventSource } from '../utils/eventSource';

const { data, isConnected, disconnect } = useEventSource('/api/logs/stream');

onUnmounted(() => {
  disconnect();
});
</script>

<template>
  <div class="p-4 flex flex-col gap-4">
    <div class="flex justify-between items-center mb-4">
      <h1 class="text-2xl font-bold">System Logs</h1>
      <div class="flex items-center gap-2">
        <div 
          class="w-2 h-2 rounded-full"
          :class="isConnected ? 'bg-green-500' : 'bg-red-500'"
        ></div>
        <span class="text-sm text-gray-400">
          {{ isConnected ? 'Connected' : 'Disconnected' }}
        </span>
      </div>
    </div>

    <div v-if="!data" class="flex justify-center items-center h-[600px]">
      <div class="text-center">
        <i class="pi pi-spin pi-spinner text-4xl text-blue-500"></i>
        <p class="mt-4 text-gray-400">Connecting to log stream...</p>
      </div>
    </div>

    <div 
      v-else 
      class="bg-gray-800 rounded-lg p-4 font-mono text-sm h-[600px] overflow-y-auto flex flex-col-reverse"
    >
      <div 
        v-for="(log, index) in data" 
        :key="index" 
        class="mb-1 border-b border-gray-700 pb-1"
      >
        <span class="text-gray-400">[{{ formatDate(log.time) }}]</span>
        <span :class="getLevelColor(log.level)" class="mx-2 font-bold">{{ log.level }}</span>
        <span class="text-blue-300">{{ log.component }}:</span>
        <span class="text-white ml-2">{{ log.message }}</span>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  methods: {
    formatDate(dateStr) {
      const date = new Date(dateStr);
      return date.toLocaleString('de-DE', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    },
    getLevelColor(level) {
      switch(level) {
        case 'INFO': return 'text-green-400';
        case 'WARN': return 'text-yellow-400';
        case 'ERROR': return 'text-red-400';
        case 'FATAL': return 'text-red-600';
        default: return 'text-gray-400';
      }
    },
  },
};
</script>
