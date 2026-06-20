<script setup>
import { ref, computed } from 'vue';
import Popover from 'primevue/popover';
import ToggleSwitch from 'primevue/toggleswitch';
import SelectButton from 'primevue/selectbutton';
import { useAppStore } from '../stores/appStore';
import { useI18n } from 'vue-i18n';

const appStore = useAppStore();
const { t } = useI18n();
const popover = ref();

// Theme modes — segmented control
const themeOptions = computed(() => [
  { label: t('theme.light'), value: 'light', icon: 'pi pi-sun' },
  { label: t('theme.dark'), value: 'dark', icon: 'pi pi-moon' },
  { label: t('theme.bw'), value: 'bw', icon: 'pi pi-circle-fill' },
]);

// Accent swatches — value matches data-accent keys in App.vue
const accents = [
  { key: 'sky',     color: '#38bdf8' },
  { key: 'violet',  color: '#a78bfa' },
  { key: 'emerald', color: '#34d399' },
  { key: 'amber',   color: '#fbbf24' },
  { key: 'rose',    color: '#fb7185' },
  { key: 'mono',    color: '#cbd5e1' },
];

const densityOptions = computed(() => [
  { label: t('theme.comfortable'), value: 'comfortable' },
  { label: t('theme.compact'), value: 'compact' },
]);

const toggle = (event) => {
  popover.value.toggle(event);
};
</script>

<template>
  <button
    type="button"
    class="theme-trigger"
    @click="toggle"
    :aria-label="t('theme.appearance')"
    :title="t('theme.appearance')"
  >
    <i class="pi text-sm" :class="{
      'pi-sun': appStore.theme === 'light',
      'pi-moon': appStore.theme === 'dark',
      'pi-circle-fill': appStore.theme === 'bw',
    }"></i>
  </button>

  <Popover ref="popover" class="theme-popover">
    <div class="theme-panel">
      <!-- Mode -->
      <div class="theme-group">
        <div class="theme-group-label">{{ t('theme.mode') }}</div>
        <div class="theme-modes">
          <button
            v-for="opt in themeOptions"
            :key="opt.value"
            type="button"
            class="theme-mode-btn"
            :class="{ 'theme-mode-btn--active': appStore.theme === opt.value }"
            @click="appStore.setTheme(opt.value)"
          >
            <i :class="opt.icon" class="text-sm"></i>
            <span>{{ opt.label }}</span>
          </button>
        </div>
      </div>

      <!-- Accent -->
      <div class="theme-group">
        <div class="theme-group-label">{{ t('theme.accent') }}</div>
        <div class="theme-swatches">
          <button
            v-for="a in accents"
            :key="a.key"
            type="button"
            class="theme-swatch"
            :class="{ 'theme-swatch--active': appStore.accent === a.key }"
            :style="{ '--swatch': a.color }"
            :title="a.key"
            :aria-label="a.key"
            @click="appStore.setAccent(a.key)"
          >
            <i v-if="appStore.accent === a.key" class="pi pi-check text-xs"></i>
          </button>
        </div>
      </div>

      <!-- Density -->
      <div class="theme-group">
        <div class="theme-group-label">{{ t('theme.density') }}</div>
        <SelectButton
          :modelValue="appStore.density"
          @update:modelValue="appStore.setDensity($event)"
          :options="densityOptions"
          optionLabel="label"
          optionValue="value"
          :allowEmpty="false"
          size="small"
          class="theme-density"
        />
      </div>

      <!-- Reduced motion -->
      <div class="theme-row">
        <div class="theme-row-text">
          <div class="theme-row-title">{{ t('theme.reducedMotion') }}</div>
          <div class="theme-row-hint">{{ t('theme.reducedMotionHint') }}</div>
        </div>
        <ToggleSwitch
          :modelValue="appStore.reducedMotion"
          @update:modelValue="appStore.toggleReducedMotion($event)"
        />
      </div>
    </div>
  </Popover>
</template>

<style scoped>
.theme-trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.1rem;
  height: 2.1rem;
  border-radius: 12px;
  border: none;
  cursor: pointer;
  background: transparent;
  color: var(--text-secondary);
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}
.theme-trigger:hover {
  background: var(--bg-soft);
  color: var(--text-primary);
}

.theme-panel {
  display: flex;
  flex-direction: column;
  gap: 1.1rem;
  width: min(86vw, 300px);
  padding: 0.25rem 0.25rem;
}

.theme-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.theme-group-label {
  font-size: 0.66rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.18em;
  color: var(--text-muted);
}

/* Mode segmented buttons */
.theme-modes {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.4rem;
}
.theme-mode-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
  padding: 0.6rem 0.4rem;
  border-radius: 14px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-panel-item);
  color: var(--text-secondary);
  font-size: 0.72rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s, transform 0.15s;
}
.theme-mode-btn:hover {
  background: var(--bg-soft);
  color: var(--text-primary);
}
.theme-mode-btn--active {
  background: var(--accent-tint);
  border-color: var(--accent);
  color: var(--accent);
}

/* Accent swatches */
.theme-swatches {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
}
.theme-swatch {
  width: 1.8rem;
  height: 1.8rem;
  border-radius: 999px;
  border: 2px solid transparent;
  background: var(--swatch);
  color: #0b1220;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  transition: transform 0.15s, box-shadow 0.15s;
}
.theme-swatch:hover {
  transform: scale(1.12);
}
.theme-swatch--active {
  border-color: var(--text-primary);
  box-shadow: 0 0 0 2px var(--bg-surface-strong), 0 0 0 4px var(--swatch);
}

/* Density */
:deep(.theme-density) {
  width: 100%;
}
:deep(.theme-density .p-button) {
  flex: 1;
  min-height: 36px !important;
}

/* Reduced motion row */
.theme-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--border-subtle);
}
.theme-row-text {
  min-width: 0;
}
.theme-row-title {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--text-primary);
}
.theme-row-hint {
  font-size: 0.7rem;
  color: var(--text-muted);
  margin-top: 0.15rem;
}
</style>
