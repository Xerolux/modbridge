<template>
  <span class="brand-mark" aria-hidden="true">
    <span class="brand-mark__bridge"></span>
    <span class="brand-mark__letters">MB</span>
  </span>
</template>

<style scoped>
.brand-mark {
  position: relative;
  display: inline-grid;
  place-items: center;
  width: 2.5rem;
  height: 2.5rem;
  flex: 0 0 auto;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--accent) 45%, var(--border-soft));
  border-radius: 0.85rem;
  background:
    linear-gradient(145deg, color-mix(in srgb, var(--accent) 22%, transparent), transparent 58%),
    var(--bg-surface-strong);
  box-shadow: inset 0 1px rgba(255, 255, 255, 0.16), 0 8px 20px color-mix(in srgb, var(--accent) 14%, transparent);
}

.brand-mark__bridge {
  position: absolute;
  top: 0.52rem;
  width: 1.55rem;
  height: 0.7rem;
  border: 2px solid var(--accent);
  border-bottom: 0;
  border-radius: 999px 999px 0 0;
  opacity: 0.9;
}

.brand-mark__letters {
  position: relative;
  margin-top: 0.48rem;
  font-size: 0.7rem;
  font-weight: 800;
  letter-spacing: -0.06em;
  color: var(--text-primary);
}
</style>
