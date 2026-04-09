<script setup>
const props = defineProps({
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

const statusClass = {
  Running: 'widget-status widget-status--running',
  Error: 'widget-status widget-status--error',
  Stopped: 'widget-status widget-status--stopped',
  Unknown: 'widget-status widget-status--unknown'
};

const statusDotClass = {
  Running: 'status-dot status-dot--running',
  Error: 'status-dot status-dot--error',
  Stopped: 'status-dot status-dot--stopped',
  Unknown: 'status-dot status-dot--unknown'
};
</script>

<template>
  <div class="widget-shell h-full w-full">
    <div class="widget-noise"></div>

    <div class="relative z-[1] flex h-full flex-col justify-between p-4 sm:p-5">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <div class="text-[0.7rem] font-semibold uppercase tracking-[0.28em] text-[var(--text-muted)]">Proxy Widget</div>
          <div class="mt-2 text-lg font-bold text-[var(--text-primary)] truncate" :title="title">{{ title }}</div>
        </div>

        <div :class="statusClass[status] || statusClass.Unknown">
          <span :class="statusDotClass[status] || statusDotClass.Unknown"></span>
          {{ status }}
        </div>
      </div>

      <div class="space-y-3">
        <div class="text-2xl font-extrabold tracking-tight text-[var(--text-primary)] sm:text-[2rem] break-words" :title="String(value)">
          {{ value }}
          <span v-if="unit" class="ml-1 text-sm font-medium text-[var(--text-muted)]">{{ unit }}</span>
        </div>

        <div class="widget-footer">
          <div v-if="activeConnections !== null" class="flex items-center gap-2 text-sm text-[var(--text-secondary)]">
            <span :class="statusDotClass[status] || statusDotClass.Unknown"></span>
            <span>{{ activeConnections }} {{ activeConnections === 1 ? 'Client' : 'Clients' }}</span>
          </div>

          <div class="widget-footer-pill">
            <i class="pi pi-arrows-alt text-xs"></i>
            Drag to move
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.widget-shell {
  position: relative;
  height: 100%;
  border-radius: 20px;
  overflow-y: auto;
  overflow-x: hidden;
}

.widget-noise {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at top right, rgba(125, 211, 252, 0.18), transparent 35%),
    radial-gradient(circle at bottom left, rgba(192, 132, 252, 0.16), transparent 38%);
  opacity: 0.9;
}

.widget-status {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  border-radius: 999px;
  padding: 0.4rem 0.75rem;
  font-size: 0.72rem;
  font-weight: 700;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(15, 23, 42, 0.35);
  color: var(--text-secondary);
}

.widget-status--running {
  color: #bbf7d0;
}

.widget-status--stopped {
  color: #fde68a;
}

.widget-status--error {
  color: #fecdd3;
}

.widget-status--unknown {
  color: var(--text-secondary);
}

.widget-footer {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  align-items: center;
  flex-wrap: wrap;
}

.widget-footer-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  padding: 0.45rem 0.7rem;
  font-size: 0.72rem;
  color: var(--text-muted);
}
</style>
