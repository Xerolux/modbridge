<script setup>
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

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
  Running: 'widget-badge widget-badge--running',
  Error:   'widget-badge widget-badge--error',
  Stopped: 'widget-badge widget-badge--stopped',
  Unknown: 'widget-badge widget-badge--unknown'
};

const statusDotClass = {
  Running: 'status-dot status-dot--running',
  Error:   'status-dot status-dot--error',
  Stopped: 'status-dot status-dot--stopped',
  Unknown: 'status-dot status-dot--unknown'
};
</script>

<template>
  <div class="widget-shell h-full w-full">
    <div class="widget-bg"></div>

    <div class="relative z-[1] flex h-full flex-col justify-between p-4 sm:p-5">
      <!-- Header -->
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0 flex-1">
          <div class="text-[0.68rem] font-semibold uppercase tracking-[0.26em] text-[var(--text-muted)] mb-1.5">{{ t('widget.proxyLabel') }}</div>
          <div class="text-base font-bold text-[var(--text-primary)] truncate leading-tight" :title="title">{{ title }}</div>
        </div>
        <div :class="statusClass[status] || statusClass.Unknown" class="shrink-0">
          <span :class="statusDotClass[status] || statusDotClass.Unknown"></span>
          {{ status }}
        </div>
      </div>

      <!-- Value -->
      <div class="space-y-3 mt-4">
        <div class="text-2xl font-extrabold tracking-tight text-[var(--text-primary)] sm:text-[1.9rem] break-words" :title="String(value)">
          {{ value }}
          <span v-if="unit" class="ml-1 text-sm font-medium text-[var(--text-muted)]">{{ unit }}</span>
        </div>

        <!-- Footer -->
        <div class="flex items-center justify-between gap-2 flex-wrap">
          <div v-if="activeConnections !== null" class="flex items-center gap-1.5 text-xs text-[var(--text-secondary)]">
            <span :class="statusDotClass[status] || statusDotClass.Unknown"></span>
            <span>{{ activeConnections }} {{ activeConnections === 1 ? t('widget.client') : t('widget.clients') }}</span>
          </div>
          <div class="widget-drag-hint ml-auto">
            <i class="pi pi-arrows-alt text-xs"></i>
            <span>{{ t('widget.drag') }}</span>
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
  overflow: hidden;
}

.widget-bg {
  position: absolute;
  inset: 0;
  background: var(--hero-gradient);
  opacity: 0.7;
}

/* Status badge */
.widget-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  border-radius: 999px;
  padding: 0.35rem 0.65rem;
  font-size: 0.7rem;
  font-weight: 700;
  border: 1px solid var(--border-subtle);
  background: var(--bg-panel-item);
  color: var(--text-secondary);
}
.widget-badge--running { color: var(--success); }
.widget-badge--stopped { color: var(--warning); }
.widget-badge--error   { color: var(--danger); }
.widget-badge--unknown { color: var(--text-muted); }

/* Drag hint */
.widget-drag-hint {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  border-radius: 999px;
  background: var(--bg-panel-item);
  padding: 0.3rem 0.6rem;
  font-size: 0.68rem;
  color: var(--text-muted);
}
</style>
