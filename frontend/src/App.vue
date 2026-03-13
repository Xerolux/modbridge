<script setup>
import { onMounted } from 'vue';
import { useAppStore } from './stores/appStore';

const store = useAppStore();

onMounted(() => {
  // Apply theme on mount
  if (store.darkMode) {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
});

// Watch for dark mode changes
import { watch } from 'vue';
watch(() => store.darkMode, (isDark) => {
  if (isDark) {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
});
</script>

<template>
  <div :class="store.darkMode ? 'dark' : 'light'" class="app-container">
    <!-- Animated Background -->
    <div class="cyber-grid"></div>
    <div class="scanlines"></div>
    <div class="ambient-glow"></div>

    <!-- Main Content -->
    <div class="content-wrapper">
      <router-view></router-view>
    </div>
  </div>
</template>

<style>
/* ========================================
   NEON INDUSTRIAL THEME
   ======================================== */

/* Custom Font Imports via Google Fonts */
@import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@300;400;500;600;700&family=Rajdhani:wght@300;400;500;600;700&family=Orbitron:wght@400;500;600;700;800;900&display=swap');

/* CSS Variables - Neon Industrial Palette */
:root {
  /* Primary Colors - Neon Cyberpunk */
  --bg-deep: #0a0e17;
  --bg-dark: #111827;
  --bg-mid: #1f2937;
  --bg-light: #374151;

  /* Neon Accents */
  --neon-cyan: #00fff5;
  --neon-magenta: #ff00ff;
  --neon-amber: #ffaa00;
  --neon-green: #00ff88;
  --neon-blue: #00d4ff;

  /* Text Colors */
  --text-primary: #f0f4f8;
  --text-secondary: #94a3b8;
  --text-muted: #64748b;

  /* Status Colors */
  --status-running: var(--neon-green);
  --status-stopped: #ef4444;
  --status-paused: var(--neon-amber);
  --status-error: #ff0055;

  /* Gradients */
  --gradient-primary: linear-gradient(135deg, var(--neon-cyan) 0%, var(--neon-blue) 100%);
  --gradient-secondary: linear-gradient(135deg, var(--neon-magenta) 0%, #8b5cf6 100%);
  --gradient-accent: linear-gradient(135deg, var(--neon-green) 0%, var(--neon-cyan) 100%);
  --gradient-danger: linear-gradient(135deg, #ff0055 0%, #ff4444 100%);

  /* Effects */
  --glow-cyan: 0 0 20px rgba(0, 255, 245, 0.6), 0 0 40px rgba(0, 255, 245, 0.3);
  --glow-magenta: 0 0 20px rgba(255, 0, 255, 0.6), 0 0 40px rgba(255, 0, 255, 0.3);
  --glow-green: 0 0 20px rgba(0, 255, 136, 0.6), 0 0 40px rgba(0, 255, 136, 0.3);

  /* Glassmorphism */
  --glass-bg: rgba(31, 41, 55, 0.7);
  --glass-border: rgba(255, 255, 255, 0.1);
  --glass-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

/* Dark Theme Overrides */
.dark {
  --bg-deep: #05080f;
  --bg-dark: #0a0f1a;
  --bg-mid: #151b2e;
  --bg-light: #1e273a;
}

/* Light Theme (Professional) */
.light {
  --bg-deep: #f8fafc;
  --bg-dark: #f1f5f9;
  --bg-mid: #e2e8f0;
  --bg-light: #cbd5e1;
  --text-primary: #0f172a;
  --text-secondary: #475569;
  --text-muted: #64748b;
}

/* ========================================
   BASE STYLES
   ======================================== */

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

/* Fix iOS auto-zoom by enforcing 16px minimum font size for all inputs */
@media screen and (max-width: 768px) {
  input, select, textarea, .p-inputtext, .p-dropdown, .p-inputnumber-input, .p-password-input {
    font-size: 16px !important;
  }
}

body {
  font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  background-color: var(--bg-deep);
  color: var(--text-primary);
  overflow-x: hidden;
  line-height: 1.6;
}

/* App Container */
.app-container {
  min-height: 100vh;
  position: relative;
  overflow-x: hidden;
}

/* Content Wrapper */
.content-wrapper {
  position: relative;
  z-index: 10;
}

/* ========================================
   ANIMATED BACKGROUNDS
   ======================================== */

/* Cyber Grid Background */
.cyber-grid {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 1;
  background-image:
    linear-gradient(rgba(0, 255, 245, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(0, 255, 245, 0.03) 1px, transparent 1px);
  background-size: 50px 50px;
  animation: gridMove 20s linear infinite;
}

@keyframes gridMove {
  0% {
    transform: perspective(500px) rotateX(60deg) translateY(0);
  }
  100% {
    transform: perspective(500px) rotateX(60deg) translateY(50px);
  }
}

/* Scanlines Overlay */
.scanlines {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 2;
  background: repeating-linear-gradient(
    0deg,
    rgba(0, 0, 0, 0.1) 0px,
    rgba(0, 0, 0, 0.1) 1px,
    transparent 1px,
    transparent 2px
  );
  opacity: 0.3;
}

/* Ambient Glow */
.ambient-glow {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 3;
  background:
    radial-gradient(circle at 20% 80%, rgba(0, 255, 245, 0.08) 0%, transparent 40%),
    radial-gradient(circle at 80% 20%, rgba(255, 0, 255, 0.06) 0%, transparent 40%),
    radial-gradient(circle at 50% 50%, rgba(0, 212, 255, 0.04) 0%, transparent 50%);
  animation: ambientPulse 8s ease-in-out infinite;
}

@keyframes ambientPulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}

/* ========================================
   TYPOGRAPHY
   ======================================== */

h1, h2, h3, h4, h5, h6 {
  font-family: 'Orbitron', 'Rajdhani', sans-serif;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

h1 {
  font-size: 2.5rem;
  background: var(--gradient-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  text-shadow: 0 0 30px rgba(0, 255, 245, 0.5);
}

h2 {
  font-size: 1.75rem;
  color: var(--text-primary);
}

h3 {
  font-size: 1.25rem;
  color: var(--text-secondary);
}

/* ========================================
   UTILITY CLASSES
   ======================================== */

/* Neon Text Effects */
.text-neon-cyan {
  color: var(--neon-cyan);
  text-shadow: 0 0 10px rgba(0, 255, 245, 0.8), 0 0 20px rgba(0, 255, 245, 0.4);
}

.text-neon-magenta {
  color: var(--neon-magenta);
  text-shadow: 0 0 10px rgba(255, 0, 255, 0.8), 0 0 20px rgba(255, 0, 255, 0.4);
}

.text-neon-green {
  color: var(--neon-green);
  text-shadow: 0 0 10px rgba(0, 255, 136, 0.8), 0 0 20px rgba(0, 255, 136, 0.4);
}

/* Glass Effect Card */
.glass-card {
  background: var(--glass-bg);
  backdrop-filter: blur(16px);
  border: 1px solid var(--glass-border);
  border-radius: 16px;
  box-shadow: var(--glass-shadow);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.glass-card:hover {
  border-color: rgba(0, 255, 245, 0.3);
  box-shadow:
    var(--glass-shadow),
    0 0 30px rgba(0, 255, 245, 0.2);
  transform: translateY(-2px);
}

/* Neon Border */
.border-neon {
  border: 1px solid transparent;
  background-clip: padding-box;
  position: relative;
}

.border-neon::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  padding: 1px;
  background: var(--gradient-primary);
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  opacity: 0.5;
}

/* Status Indicators */
.status-running {
  color: var(--status-running);
  text-shadow: 0 0 10px rgba(0, 255, 136, 0.8);
  animation: statusPulse 2s ease-in-out infinite;
}

.status-stopped {
  color: var(--status-stopped);
  text-shadow: 0 0 10px rgba(239, 68, 68, 0.8);
}

.status-paused {
  color: var(--status-paused);
  text-shadow: 0 0 10px rgba(255, 170, 0, 0.8);
}

@keyframes statusPulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}

/* ========================================
   BUTTONS
   ======================================== */

.btn-neon {
  position: relative;
  padding: 12px 24px;
  font-family: 'Rajdhani', sans-serif;
  font-weight: 600;
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  border: none;
  border-radius: 8px;
  background: var(--gradient-primary);
  color: var(--bg-deep);
  cursor: pointer;
  overflow: hidden;
  transition: all 0.3s ease;
  box-shadow: 0 0 20px rgba(0, 255, 245, 0.3);
}

.btn-neon::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.3), transparent);
  transition: left 0.5s ease;
}

.btn-neon:hover::before {
  left: 100%;
}

.btn-neon:hover {
  box-shadow: var(--glow-cyan);
  transform: translateY(-2px);
}

.btn-neon:active {
  transform: translateY(0);
}

/* ========================================
   ANIMATIONS
   ======================================== */

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes slideInRight {
  from {
    opacity: 0;
    transform: translateX(-50px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes neonFlicker {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.8;
  }
  52% {
    opacity: 0.4;
  }
  54% {
    opacity: 0.8;
  }
}

.animate-fade-in-up {
  animation: fadeInUp 0.6s ease-out;
}

.animate-slide-in-right {
  animation: slideInRight 0.6s ease-out;
}

.animate-neon-flicker {
  animation: neonFlicker 3s ease-in-out infinite;
}

/* ========================================
   SCROLLBAR
   ======================================== */

::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--bg-dark);
}

::-webkit-scrollbar-thumb {
  background: var(--gradient-primary);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--neon-cyan);
}

/* ========================================
   RESPONSIVE
   ======================================== */

@media (max-width: 768px) {
  h1 {
    font-size: 1.75rem;
  }

  h2 {
    font-size: 1.25rem;
  }
}
</style>
