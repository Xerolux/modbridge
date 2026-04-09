<script setup>
import { onMounted, onUnmounted, watch } from 'vue';
import { useAppStore } from './stores/appStore';

const store = useAppStore();

const applyTheme = (isDark) => {
  document.documentElement.classList.toggle('dark', isDark);
  document.documentElement.classList.toggle('light', !isDark);
};

const applyEffectsPreference = (reduced) => {
  document.documentElement.classList.toggle('effects-reduced', reduced);
};

const onBeforeUnload = (event) => {
  if (!store.hasUnsavedChanges) return;
  event.preventDefault();
  event.returnValue = '';
};

onMounted(() => {
  applyTheme(store.darkMode);
  applyEffectsPreference(store.reducedEffects);
  window.addEventListener('beforeunload', onBeforeUnload);
});

onUnmounted(() => {
  window.removeEventListener('beforeunload', onBeforeUnload);
});

watch(() => store.darkMode, applyTheme, { immediate: true });
watch(() => store.reducedEffects, applyEffectsPreference, { immediate: true });
</script>

<template>
  <div
    :class="[
      store.darkMode ? 'dark' : 'light',
      store.reducedEffects ? 'effects-reduced' : ''
    ]"
    class="app-shell"
  >
    <div class="ambient-layer ambient-grid"></div>
    <div class="ambient-layer ambient-orb ambient-orb-a"></div>
    <div class="ambient-layer ambient-orb ambient-orb-b"></div>
    <div class="ambient-layer ambient-orb ambient-orb-c"></div>

    <div class="content-wrapper">
      <router-view></router-view>
    </div>
  </div>
</template>

<style>
@import url('https://fonts.googleapis.com/css2?family=Manrope:wght@400;500;600;700;800&family=Space+Grotesk:wght@500;700&display=swap');

:root {
  --bg-canvas: #09111f;
  --bg-surface: rgba(14, 22, 39, 0.72);
  --bg-surface-strong: rgba(11, 18, 32, 0.9);
  --bg-soft: rgba(148, 163, 184, 0.12);
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
  --shadow-soft: 0 20px 60px rgba(2, 6, 23, 0.35);
  --shadow-strong: 0 35px 80px rgba(2, 6, 23, 0.5);
  --glass-blur: blur(24px);
  --hero-gradient: linear-gradient(135deg, rgba(56, 189, 248, 0.2), rgba(192, 132, 252, 0.18));
  --panel-gradient: linear-gradient(180deg, rgba(255, 255, 255, 0.12), rgba(255, 255, 255, 0.04));
}

.light {
  --bg-canvas: #eef4fb;
  --bg-surface: rgba(255, 255, 255, 0.7);
  --bg-surface-strong: rgba(255, 255, 255, 0.92);
  --bg-soft: rgba(15, 23, 42, 0.05);
  --text-primary: #102038;
  --text-secondary: #334155;
  --text-muted: #64748b;
  --border-soft: rgba(15, 23, 42, 0.08);
  --border-strong: rgba(56, 189, 248, 0.22);
  --shadow-soft: 0 18px 45px rgba(148, 163, 184, 0.18);
  --shadow-strong: 0 28px 60px rgba(148, 163, 184, 0.24);
}

* { box-sizing: border-box; }

html,
body,
#app {
  min-height: 100%;
}

body {
  margin: 0;
  font-family: 'Manrope', sans-serif;
  background:
    radial-gradient(circle at top left, rgba(125, 211, 252, 0.14), transparent 32%),
    radial-gradient(circle at top right, rgba(192, 132, 252, 0.16), transparent 28%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.12), transparent 20%),
    var(--bg-canvas);
  color: var(--text-primary);
  overflow-x: hidden;
}

h1,h2,h3,h4,h5,h6 {
  font-family: 'Space Grotesk', sans-serif;
  letter-spacing: -0.03em;
  margin: 0;
}

a { color: inherit; }

.app-shell {
  min-height: 100vh;
  position: relative;
  isolation: isolate;
}

.content-wrapper { position: relative; z-index: 2; }

.ambient-layer {
  pointer-events: none;
  position: fixed;
  inset: 0;
}

.ambient-grid {
  opacity: 0.4;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.035) 1px, transparent 1px);
  background-size: 28px 28px;
  mask-image: radial-gradient(circle at center, black 30%, transparent 90%);
}

.ambient-orb {
  filter: blur(80px);
  opacity: 0.55;
  animation: floatOrb 18s ease-in-out infinite;
}

.ambient-orb-a {
  inset: auto auto 72% 6%;
  width: 26rem;
  height: 26rem;
  background: rgba(56, 189, 248, 0.22);
}

.ambient-orb-b {
  inset: 8% 4% auto auto;
  width: 24rem;
  height: 24rem;
  background: rgba(192, 132, 252, 0.18);
  animation-duration: 22s;
}

.ambient-orb-c {
  inset: auto 20% 10% auto;
  width: 20rem;
  height: 20rem;
  background: rgba(34, 197, 94, 0.12);
  animation-duration: 26s;
}

.effects-reduced .ambient-orb,
.effects-reduced .ambient-grid {
  display: none;
}

.effects-reduced .glass-card,
.effects-reduced .glass-panel,
.effects-reduced .glass-hero {
  backdrop-filter: none;
  -webkit-backdrop-filter: none;
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

.glass-card::before,
.glass-panel::before,
.glass-hero::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: var(--panel-gradient);
  opacity: 0.9;
  pointer-events: none;
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

.status-dot--running { background: var(--success); box-shadow: 0 0 0.75rem rgba(74, 222, 128, 0.45); }
.status-dot--stopped { background: var(--warning); box-shadow: 0 0 0.75rem rgba(251, 191, 36, 0.4); }
.status-dot--error { background: var(--danger); box-shadow: 0 0 0.75rem rgba(251, 113, 133, 0.4); }
.status-dot--unknown { background: rgba(148, 163, 184, 0.8); }

@keyframes floatOrb {
  0%, 100% { transform: translate3d(0, 0, 0) scale(1); }
  50% { transform: translate3d(1.5rem, -1.2rem, 0) scale(1.08); }
}

@media (prefers-reduced-motion: reduce) {
  .ambient-orb { animation: none !important; }
}
</style>
