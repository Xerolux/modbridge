<script setup>
defineProps({
    title: String,
    value: [String, Number],
    unit: String,
    status: {
        type: String,
        default: 'Unknown'
    },
    activeConnections: {
        type: Number,
        default: null
    }
});
</script>

<template>
    <div class="h-full flex flex-col justify-between p-2">
        <div class="flex justify-between items-start">
            <div class="text-gray-400 text-sm font-medium uppercase tracking-wider">{{ title }}</div>
            <div
                class="px-2 py-0.5 rounded-full text-xs font-medium"
                :class="{
                    'bg-green-500/20 text-green-400': status === 'Running',
                    'bg-red-500/20 text-red-400': status === 'Error',
                    'bg-yellow-500/20 text-yellow-400': status === 'Stopped',
                    'bg-gray-500/20 text-gray-400': status === 'Unknown'
                }"
            >
                {{ status }}
            </div>
        </div>
        <div class="text-2xl font-bold text-blue-400 my-1 truncate" :title="String(value)">
            {{ value }} <span v-if="unit" class="text-sm text-gray-500 ml-1">{{ unit }}</span>
        </div>
        <div v-if="activeConnections !== null" class="flex items-center gap-1 mt-1">
            <span class="inline-block w-2 h-2 rounded-full"
                :class="status === 'Running' ? 'bg-green-400' : 'bg-gray-500'"></span>
            <span class="text-xs text-gray-400">
                {{ activeConnections }} {{ activeConnections === 1 ? 'Client' : 'Clients' }}
            </span>
        </div>
    </div>
</template>
