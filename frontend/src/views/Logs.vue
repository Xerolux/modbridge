<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue';
import { useEventSource } from '../utils/eventSource';
import axios from '../axios.js';
import { formatDateTime, getLogLevelColor } from '../utils/helpers';

const logs = ref([]);
const isConnected = ref(false);
const autoScroll = ref(localStorage.getItem('logsAutoScroll') !== 'false');
const logsContainer = ref(null);
const loadingInitial = ref(true);
let disconnectFn = null;
let unwatchData = null;
let unwatchConnected = null;
let pendingLogs = [];
let logBatchFrame = null;

const MAX_LOG_ENTRIES = 500;

const trimLogs = (items) => items.length > MAX_LOG_ENTRIES ? items.slice(-MAX_LOG_ENTRIES) : items;

const scheduleLogFlush = () => {
  if (logBatchFrame) return;
  logBatchFrame = requestAnimationFrame(() => {
    logs.value = trimLogs([...logs.value, ...pendingLogs]);
    pendingLogs = [];
    logBatchFrame = null;
  });
};

const toggleAutoScroll = () => {
  localStorage.setItem('logsAutoScroll', autoScroll.value.toString());
};

const fetchInitialLogs = async () => {
  loadingInitial.value = true;
  try {
    const res = await axios.get('/api/logs');
    logs.value = res.data || [];
  } catch (e) {
    console.error('Failed to fetch initial logs', e);
  } finally {
    loadingInitial.value = false;
  }
};

onMounted(async () => {
  await fetchInitialLogs();

  const { data, disconnect, isConnected: connected } = useEventSource('/api/logs/stream');
  disconnectFn = disconnect;

  unwatchConnected = watch(connected, (val) => { isConnected.value = val; });

  unwatchData = watch(data, (eventData) => {
    if (!eventData) return;
    if (Array.isArray(eventData)) {
      pendingLogs = [];
      logs.value = trimLogs(eventData);
    } else {
      pendingLogs.push(eventData);
      scheduleLogFlush();
    }
  });
});

onUnmounted(() => {
  if (unwatchData) unwatchData();
  if (unwatchConnected) unwatchConnected();
  if (logBatchFrame) cancelAnimationFrame(logBatchFrame);
  if (disconnectFn) disconnectFn();
});

watch(logs, (newVal) => {
  if (autoScroll.value && logsContainer.value && newVal.length > 0) {
    nextTick(() => {
      if (logsContainer.value) logsContainer.value.scrollTop = logsContainer.value.scrollHeight;
    });
  }
});
</script>

<template>
  <div class="p-2 sm:p-4 flex flex-col gap-4 w-full">

    <!-- ── Hero ─────────────────────────────────────────────────── -->
    <section class="glass-hero rounded-[28px] p-5 sm:p-6">
      <div class="relative z-[1] flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div class="space-y-3">
          <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.28em] text-[var(--text-muted)]">
            <i class="pi pi-list"></i>
            System Logs
          </div>
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
            <h1 class="text-2xl sm:text-3xl font-bold text-[var(--text-primary)]">Systemlogs</h1>
            <div class="flex items-center gap-1.5 shrink-0">
              <span class="status-dot" :class="isConnected ? 'status-dot--running' : 'status-dot--error'"></span>
              <span class="text-sm text-[var(--text-muted)]">{{ isConnected ? 'Live' : 'Getrennt' }}</span>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap gap-2">
          <!-- Auto-Scroll toggle -->
          <button
            type="button"
            class="logs-ctrl-btn"
            :class="{ 'logs-ctrl-btn--active': autoScroll }"
            @click="autoScroll = !autoScroll; toggleAutoScroll()"
            :title="autoScroll ? 'Auto-Scroll deaktivieren' : 'Auto-Scroll aktivieren'"
          >
            <i :class="autoScroll ? 'pi pi-lock' : 'pi pi-lock-open'" class="text-sm"></i>
            <span>Auto-Scroll</span>
            <span class="logs-ctrl-dot" :class="autoScroll ? 'logs-ctrl-dot--on' : 'logs-ctrl-dot--off'"></span>
          </button>

          <!-- Refresh -->
          <button
            type="button"
            class="logs-ctrl-btn"
            @click="fetchInitialLogs"
            title="Logs neu laden"
          >
            <i class="pi pi-refresh text-sm"></i>
            <span>Aktualisieren</span>
          </button>
        </div>
      </div>
    </section>

    <!-- ── Loading ───────────────────────────────────────────────── -->
    <div v-if="loadingInitial" class="glass-panel rounded-[28px] p-10">
      <div class="flex min-h-[320px] flex-col items-center justify-center text-center relative z-[1]">
        <div class="mb-5 flex h-20 w-20 items-center justify-center rounded-2xl bg-[var(--bg-panel-item)] border border-[var(--border-subtle)]">
          <i class="pi pi-spin pi-spinner text-3xl text-[var(--accent)]"></i>
        </div>
        <p class="text-[var(--text-secondary)] text-sm">Logs werden geladen…</p>
      </div>
    </div>

    <!-- ── Empty state ───────────────────────────────────────────── -->
    <div v-else-if="logs.length === 0" class="glass-panel rounded-[28px] p-10">
      <div class="flex min-h-[320px] flex-col items-center justify-center text-center relative z-[1]">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-[var(--bg-panel-item)] border border-[var(--border-subtle)]">
          <i class="pi pi-inbox text-2xl text-[var(--text-muted)]"></i>
        </div>
        <h3 class="text-lg font-semibold text-[var(--text-primary)]">Keine Logs vorhanden</h3>
        <p class="mt-2 text-sm text-[var(--text-muted)] max-w-sm">Es wurden noch keine Log-Einträge empfangen.</p>
      </div>
    </div>

    <!-- ── Log list ──────────────────────────────────────────────── -->
    <div
      v-else
      ref="logsContainer"
      class="glass-panel rounded-[28px] p-3 sm:p-4 font-mono text-sm h-[60vh] sm:h-[calc(100vh-280px)] overflow-y-auto"
    >
      <div
        v-for="(log, index) in logs"
        :key="index"
        class="log-row"
      >
        <span class="log-time">{{ formatDateTime(log.timestamp) }}</span>
        <span :class="getLogLevelColor(log.level)" class="log-level">{{ log.level }}</span>
        <span class="log-source">{{ log.proxy_id || 'SYSTEM' }}</span>
        <span class="log-msg">{{ log.message }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ── Header controls ─────────────────────────────────────────────── */
.logs-ctrl-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: 14px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-panel-item);
  color: var(--text-secondary);
  font-size: 0.84rem;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}
.logs-ctrl-btn:hover {
  background: var(--bg-soft);
  color: var(--text-primary);
  border-color: var(--border-soft);
}
.logs-ctrl-btn--active {
  border-color: var(--accent-tint);
  color: var(--accent);
}

.logs-ctrl-dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  flex-shrink: 0;
}
.logs-ctrl-dot--on  { background: var(--accent); }
.logs-ctrl-dot--off { background: var(--text-muted); }

/* ── Log row ─────────────────────────────────────────────────────── */
.log-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0 8px;
  padding: 4px 6px;
  border-radius: 8px;
  border-bottom: 1px solid var(--border-subtle);
  transition: background 0.1s;
  line-height: 1.5;
}
.log-row:hover { background: var(--bg-panel-item); }
.log-row:last-child { border-bottom: none; }

.log-time   { color: var(--text-muted); font-size: 0.78rem; white-space: nowrap; }
.log-level  { font-weight: 700; font-size: 0.78rem; white-space: nowrap; }
.log-source { color: var(--accent); white-space: nowrap; }
.log-msg    { color: var(--text-primary); word-break: break-word; flex: 1 1 100%; }

@media (min-width: 640px) {
  .log-msg { flex: 1 1 auto; }
}
</style>
