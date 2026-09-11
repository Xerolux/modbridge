<script setup>
import { onMounted, onUnmounted, watch, computed } from 'vue';
import { useAppStore } from './stores/appStore';
import { useToast } from 'primevue/usetoast';
import { useI18n } from 'vue-i18n';
import Toast from 'primevue/toast';
import ConfirmDialog from 'primevue/confirmdialog';
import { navigation } from './router/navigation';

const store = useAppStore();
const toast = useToast();
const { t } = useI18n();

const appShellClass = computed(() => [
  store.theme, // 'light' | 'dark' | 'bw'
  'app-shell'
]);

const reloadPage = () => {
  window.location.hash = `#${navigation.failedPath || '/'}`;
  window.location.reload();
};

// Apply the entire theme to <html> in one pass — class toggles, accent palette,
// density and motion preferences. Runs once on mount and whenever any value changes.
const applyTheme = () => {
  const html = document.documentElement;
  const t = store.theme;

  // PrimeVue uses the `.dark` selector for its dark preset — reuse it for both
  // the dark and the monochrome (bw) themes so component styling stays consistent.
  html.classList.toggle('dark', t === 'dark' || t === 'bw');
  html.classList.toggle('light', t === 'light');
  html.classList.toggle('bw', t === 'bw');

  html.setAttribute('data-accent', store.accent);
  html.setAttribute('data-density', store.density);
  html.classList.toggle('reduced-motion', store.reducedMotion);
};

// immediate: true handles initial application — no separate onMounted call needed
watch(
  () => [store.theme, store.accent, store.density, store.reducedMotion],
  applyTheme,
  { immediate: true, deep: true }
);

onMounted(() => {
  // Global 403 handler: axios dispatches 'app:forbidden' whenever a request is
  // rejected with Forbidden. Show a single toast instead of silent failures.
  const handleForbidden = () => {
    toast.add({ severity: 'warn', summary: t('common.forbidden'), life: 3000 });
  };
  window.addEventListener('app:forbidden', handleForbidden);
  onUnmounted(() => window.removeEventListener('app:forbidden', handleForbidden));
});
</script>

<template>
  <div :class="appShellClass">
    <Toast />
    <ConfirmDialog />
    <div v-if="navigation.loading" class="navigation-progress" role="status">
      <span>{{ t('navigation.loading') }}</span>
    </div>
    <div v-if="navigation.failedPath" class="navigation-error" role="alert">
      <strong>{{ t('navigation.failed') }}</strong>
      <span>{{ t('navigation.hint') }}</span>
      <button type="button" @click="reloadPage">{{ t('navigation.retry') }}</button>
    </div>

    <div class="content-wrapper">
      <router-view></router-view>
    </div>
  </div>
</template>

<style>
 :root {
   --bg-canvas: #09111f;
   --bg-surface: #111d2e;
   --bg-surface-strong: #111d2e;
   --bg-soft: rgba(148, 163, 184, 0.12);
   --bg-input: rgba(255, 255, 255, 0.05);
   --bg-panel-item: rgba(255, 255, 255, 0.04);
   --bg-dark-overlay: rgba(15, 23, 42, 0.55);
   --text-primary: #f3f7fb;
   --text-secondary: #c4d2e3;
   --text-muted: #8ba0b8;
   --accent: #7dd3fc;
   --accent-strong: #38bdf8;
   --accent-secondary: #c084fc;
   --success: #4ade80;
   --warning: #fbbf24;
   --danger: #fb7185;
   --border-soft: rgba(255, 255, 255, 0.12);
   --border-strong: rgba(255, 255, 255, 0.2);
   --border-subtle: rgba(255, 255, 255, 0.08);
   --shadow-soft: 0 20px 60px rgba(2, 6, 23, 0.35);
   --shadow-strong: 0 35px 80px rgba(2, 6, 23, 0.5);
   --glass-blur: none;
   --hero-gradient: linear-gradient(135deg, rgba(56, 189, 248, 0.2), rgba(192, 132, 252, 0.18));
   --panel-gradient: linear-gradient(180deg, rgba(255, 255, 255, 0.12), rgba(255, 255, 255, 0.04));
   --grid-line: rgba(255, 255, 255, 0.035);
   /* Density knobs — overridden by [data-density="compact"] */
   --space-card: 1.25rem;
   --radius-panel: 16px;
   --radius-hero: 18px;
   --control-h: 44px;
}

/* ── Accent palettes ───────────────────────────────────────────────
   Each accent sets --accent / --accent-strong / --accent-secondary plus
   matching tinted gradients. Driven by data-accent on <html>. */
:root[data-accent="sky"] {
  --accent: #7dd3fc;
  --accent-strong: #38bdf8;
  --accent-secondary: #c084fc;
  --hero-gradient: linear-gradient(135deg, rgba(56, 189, 248, 0.2), rgba(192, 132, 252, 0.18));
  --accent-tint: rgba(56, 189, 248, 0.12);
}
:root[data-accent="violet"] {
  --accent: #c4b5fd;
  --accent-strong: #a78bfa;
  --accent-secondary: #f0abfc;
  --hero-gradient: linear-gradient(135deg, rgba(167, 139, 250, 0.22), rgba(240, 171, 252, 0.16));
  --accent-tint: rgba(167, 139, 250, 0.12);
}
:root[data-accent="emerald"] {
  --accent: #6ee7b7;
  --accent-strong: #34d399;
  --accent-secondary: #5eead4;
  --hero-gradient: linear-gradient(135deg, rgba(52, 211, 153, 0.2), rgba(94, 234, 212, 0.16));
  --accent-tint: rgba(52, 211, 153, 0.12);
}
:root[data-accent="amber"] {
  --accent: #fcd34d;
  --accent-strong: #fbbf24;
  --accent-secondary: #fb923c;
  --hero-gradient: linear-gradient(135deg, rgba(251, 191, 36, 0.2), rgba(251, 146, 60, 0.16));
  --accent-tint: rgba(251, 191, 36, 0.12);
}
:root[data-accent="rose"] {
  --accent: #fda4af;
  --accent-strong: #fb7185;
  --accent-secondary: #f0abfc;
  --hero-gradient: linear-gradient(135deg, rgba(251, 113, 133, 0.2), rgba(240, 171, 252, 0.16));
  --accent-tint: rgba(251, 113, 133, 0.12);
}
/* Monochrome accent — pairs naturally with the bw theme but works everywhere */
:root[data-accent="mono"] {
  --accent: #cbd5e1;
  --accent-strong: #e2e8f0;
  --accent-secondary: #94a3b8;
  --hero-gradient: linear-gradient(135deg, rgba(203, 213, 225, 0.16), rgba(148, 163, 184, 0.12));
  --accent-tint: rgba(203, 213, 225, 0.1);
}

.light {
  --bg-canvas: #f4f6f9;
  --bg-surface: #ffffff;
  --bg-surface-strong: #ffffff;
  --bg-soft: rgba(15, 23, 42, 0.05);
  --bg-input: rgba(15, 23, 42, 0.06);
  --bg-panel-item: rgba(15, 23, 42, 0.04);
  --bg-dark-overlay: rgba(0, 0, 0, 0.04);
  --text-primary: #102038;
  --text-secondary: #334155;
  --text-muted: #526176;
  --border-soft: rgba(15, 23, 42, 0.08);
  --border-strong: rgba(56, 189, 248, 0.22);
  --border-subtle: rgba(15, 23, 42, 0.1);
  --shadow-soft: 0 2px 8px rgba(15, 23, 42, 0.04);
  --shadow-strong: 0 28px 60px rgba(148, 163, 184, 0.24);
  --hero-gradient: linear-gradient(135deg, rgba(56, 189, 248, 0.15), rgba(192, 132, 252, 0.12));
  --panel-gradient: linear-gradient(180deg, rgba(255, 255, 255, 0.6), rgba(255, 255, 255, 0.3));
  --grid-line: rgba(15, 23, 42, 0.06);
}
/* Light theme keeps its accent-specific hero gradient tint */
.light[data-accent="violet"]   { --hero-gradient: linear-gradient(135deg, rgba(167, 139, 250, 0.16), rgba(240, 171, 252, 0.12)); }
.light[data-accent="emerald"]  { --hero-gradient: linear-gradient(135deg, rgba(52, 211, 153, 0.16), rgba(94, 234, 212, 0.12)); }
.light[data-accent="amber"]    { --hero-gradient: linear-gradient(135deg, rgba(251, 191, 36, 0.16), rgba(251, 146, 60, 0.12)); }
.light[data-accent="rose"]     { --hero-gradient: linear-gradient(135deg, rgba(251, 113, 133, 0.16), rgba(240, 171, 252, 0.12)); }
.light[data-accent="mono"]     { --hero-gradient: linear-gradient(135deg, rgba(100, 116, 139, 0.12), rgba(148, 163, 184, 0.1)); }

/* ── Black & White theme ───────────────────────────────────────────
   High-contrast monochrome looking-glass surface. Color status signals
   (running/error) keep a faint semantic tint so state stays readable. */
.bw {
  --bg-canvas: #0a0a0a;
  --bg-surface: rgba(22, 22, 22, 0.78);
  --bg-surface-strong: rgba(14, 14, 14, 0.92);
  --bg-soft: rgba(255, 255, 255, 0.08);
  --bg-input: rgba(255, 255, 255, 0.05);
  --bg-panel-item: rgba(255, 255, 255, 0.04);
  --bg-dark-overlay: rgba(0, 0, 0, 0.5);
  --text-primary: #fafafa;
  --text-secondary: #d4d4d4;
  --text-muted: #8f8f8f;
  --accent: #ffffff;
  --accent-strong: #e5e5e5;
  --accent-secondary: #a3a3a3;
  --success: #e5e5e5;
  --warning: #a3a3a3;
  --danger: #f5f5f5;
  --border-soft: rgba(255, 255, 255, 0.14);
  --border-strong: rgba(255, 255, 255, 0.32);
  --border-subtle: rgba(255, 255, 255, 0.08);
  --shadow-soft: 0 20px 60px rgba(0, 0, 0, 0.6);
  --shadow-strong: 0 35px 80px rgba(0, 0, 0, 0.75);
  --glass-blur: none;
  --hero-gradient: linear-gradient(135deg, rgba(255, 255, 255, 0.08), rgba(255, 255, 255, 0.02));
  --panel-gradient: linear-gradient(180deg, rgba(255, 255, 255, 0.06), rgba(255, 255, 255, 0.01));
  --grid-line: rgba(255, 255, 255, 0.04);
  --accent-tint: rgba(255, 255, 255, 0.08);
}

/* In B&W, keep status dots legible via shape/brightness rather than hue.
   Running = bright, Stopped = mid, Error = bright ringed, Unknown = dim. */
.bw .status-dot--running { background: #fafafa; box-shadow: 0 0 0.6rem rgba(255, 255, 255, 0.6); }
.bw .status-dot--stopped { background: #737373; box-shadow: none; }
.bw .status-dot--error   { background: #fafafa; box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.8), 0 0 0 3px #fafafa; }
.bw .status-dot--unknown { background: #525252; }

/* ── Density: compact ──────────────────────────────────────────────
   Tighter spacing, smaller radii and controls — fits more on mobile and
   on dense dashboards without sacrificing the glass aesthetic. */
:root[data-density="compact"] {
  --space-card: 0.9rem;
  --radius-panel: 18px;
  --radius-hero: 22px;
  --control-h: 38px;
}
:root[data-density="compact"] .p-card,
:root[data-density="compact"] .p-dialog,
:root[data-density="compact"] .p-datatable,
:root[data-density="compact"] .p-tabpanels,
:root[data-density="compact"] .p-tablist,
:root[data-density="compact"] .p-menubar,
:root[data-density="compact"] .p-sidebar {
  border-radius: var(--radius-panel) !important;
}
:root[data-density="compact"] .p-button {
  min-height: var(--control-h) !important;
}
:root[data-density="compact"] .p-inputtext,
:root[data-density="compact"] .p-dropdown,
:root[data-density="compact"] .p-inputnumber-input,
:root[data-density="compact"] .p-password-input,
:root[data-density="compact"] .p-textarea,
:root[data-density="compact"] input[type="text"],
:root[data-density="compact"] input[type="number"],
:root[data-density="compact"] input[type="email"],
:root[data-density="compact"] input[type="password"],
:root[data-density="compact"] select,
:root[data-density="compact"] textarea {
  min-height: var(--control-h) !important;
}

/* ── Reduced motion ────────────────────────────────────────────────
   Honors user preference for less animation — better on low-end mobile
   and for accessibility. Disables orbs/grid float and button transforms. */
.reduced-motion .p-button,
.reduced-motion .proxy-card,
.reduced-motion .grid-stack-item-content {
  transition: none !important;
  transform: none !important;
}

* {
  box-sizing: border-box;
}

html,
body,
#app {
  min-height: 100%;
}

body {
  margin: 0;
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  background: var(--bg-canvas);
  color: var(--text-primary);
  overflow-x: hidden;
}


h1,
h2,
h3,
h4,
h5,
h6 {
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  letter-spacing: -0.03em;
  margin: 0;
}

a {
  color: inherit;
}

input,
select,
textarea,
.p-inputtext,
.p-dropdown,
.p-inputnumber-input,
.p-password-input {
  font-size: 16px !important;
}

.app-shell {
  min-height: 100vh;
  position: relative;
  isolation: isolate;
}

.content-wrapper {
  position: relative;
  z-index: 2;
}

.glass-card,
.glass-panel,
.glass-hero {
  position: relative;
  background: var(--bg-surface);
  border: 1px solid var(--border-soft);
  box-shadow: var(--shadow-soft);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
}

.glass-hero {
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.08), rgba(255, 255, 255, 0.02)),
    var(--hero-gradient),
    var(--bg-surface);
}

.text-gradient {
  background: linear-gradient(135deg, var(--text-primary), var(--accent));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.status-dot {
  width: 0.65rem;
  height: 0.65rem;
  border-radius: 999px;
  display: inline-block;
}

.status-dot--running {
  background: var(--success);
  box-shadow: 0 0 0.75rem rgba(74, 222, 128, 0.45);
}

.status-dot--stopped {
  background: var(--warning);
  box-shadow: 0 0 0.75rem rgba(251, 191, 36, 0.4);
}

.status-dot--error {
  background: var(--danger);
  box-shadow: 0 0 0.75rem rgba(251, 113, 133, 0.4);
}

.status-dot--unknown {
  background: rgba(148, 163, 184, 0.8);
}

::-webkit-scrollbar {
  width: 10px;
  height: 10px;
}

::-webkit-scrollbar-track {
  background: var(--bg-soft);
}

::-webkit-scrollbar-thumb {
  background: var(--border-strong);
  border-radius: 999px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--accent);
}


:focus-visible {
  outline: 2px solid var(--accent-strong);
  outline-offset: 2px;
}

.p-card,
.p-dialog,
.p-datatable,
.p-tabpanels,
.p-tablist,
.p-menubar,
.p-sidebar {
  border-radius: var(--radius-panel) !important;
}

.p-card,
.p-dialog,
.p-datatable,
.p-tabpanels,
.p-tablist,
.p-menubar,
.p-sidebar,
.p-dropdown-panel {
  background: var(--bg-surface-strong) !important;
  border: 1px solid var(--border-soft) !important;
  box-shadow: var(--shadow-soft) !important;
  backdrop-filter: var(--glass-blur) !important;
  -webkit-backdrop-filter: var(--glass-blur) !important;
}

.p-inputtext,
.p-dropdown,
.p-inputnumber-input,
.p-password-input,
.p-textarea,
.p-multiselect,
.p-chips-input-token input {
  background: var(--bg-input) !important;
  border: 1px solid var(--border-soft) !important;
  color: var(--text-primary) !important;
  border-radius: 10px !important;
  min-height: 44px;
}

.p-inputtext:enabled:focus,
.p-inputnumber-input:enabled:focus,
.p-password-input:enabled:focus,
.p-dropdown:not(.p-disabled).p-focus,
.p-multiselect:not(.p-disabled).p-focus {
  border-color: var(--accent-strong) !important;
  box-shadow: 0 0 0 4px var(--accent-tint) !important;
}

.p-button {
  border-radius: 10px !important;
  min-height: 44px;
  transition: transform 0.2s ease, box-shadow 0.2s ease !important;
}

.p-button:hover {
  filter: brightness(0.97);
}

.p-tab {
  border-radius: 14px 14px 0 0 !important;
}

.p-datatable-thead > tr > th {
  background: var(--bg-panel-item) !important;
  color: var(--text-secondary) !important;
}

.p-datatable-tbody > tr {
  background: transparent !important;
}

.p-datatable-tbody > tr:hover {
  background: var(--bg-panel-item) !important;
}

.p-tag {
  border-radius: 999px !important;
}

/* ── Global usability fixes ──────────────────────────────────────────
   Address: text truncation, table overflow, inconsistent card spacing,
   panel delineation, and mobile responsiveness across ALL views.      */

/* Tables: prevent content from being cut off invisibly — allow wrapping */
.p-datatable td,
.p-datatable th {
  overflow: hidden;
  text-overflow: ellipsis;
}
.p-datatable .p-datatable-tbody > tr > td {
  white-space: normal !important;
  word-break: break-word;
}

/* Long text in dropdowns / selects must not overflow */
.p-dropdown-label,
.p-multiselect-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Dialogs: scrollable body, never cut off content */
.p-dialog-content {
  overflow-y: auto !important;
  max-height: 70vh !important;
}

/* Cards / panels: consistent padding and clear borders */
.glass-panel,
.glass-card {
  padding: var(--space-card);
}

/* Truncation utility for tight spaces */
.truncate-safe {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Responsive font scaling: prevent text from being cut on small screens */
@media (max-width: 640px) {
  .text-2xl { font-size: 1.25rem !important; }
  .text-3xl { font-size: 1.5rem !important; }
  .text-xl  { font-size: 1.1rem !important; }
}

/* InputNumber: full width, no cut-off */
.p-inputnumber-input,
.p-inputnumber {
  width: 100%;
}

/* Toast: wider on desktop so messages aren't truncated */
.p-toast-message-text {
  max-width: 420px;
}
</style>
