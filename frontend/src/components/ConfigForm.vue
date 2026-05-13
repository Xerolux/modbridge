<template>
  <section class="config-shell space-y-6">
    <div class="glass-hero rounded-[28px] p-5 sm:p-6">
      <div class="relative z-[1] flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div class="space-y-3">
          <div class="inline-flex items-center gap-2 rounded-full border border-gray-300 dark:border-white/10 bg-gray-100 dark:bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.3em] text-[var(--text-muted)]">
            <i class="pi pi-sparkles"></i>
            Proxy Studio
          </div>
          <div class="space-y-2">
            <h2 class="text-2xl sm:text-3xl font-bold text-gradient">Glass WebUI mit Drag and Drop</h2>
            <p class="max-w-2xl text-sm sm:text-base text-[var(--text-secondary)]">
              Reordne Proxies per Drag-and-Drop, bearbeite Parameter direkt in Karten und speichere nur die Einträge,
              die sich wirklich geändert haben.
            </p>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div class="hero-stat">
            <span class="hero-stat-label">Proxies</span>
            <strong class="hero-stat-value">{{ store.proxies.length }}</strong>
          </div>
          <div class="hero-stat">
            <span class="hero-stat-label">Unsaved</span>
            <strong class="hero-stat-value">{{ dirtyCount }}</strong>
          </div>
          <div class="hero-stat">
            <span class="hero-stat-label">Running</span>
            <strong class="hero-stat-value">{{ runningCount }}</strong>
          </div>
          <div class="hero-stat">
            <span class="hero-stat-label">Port</span>
            <strong class="hero-stat-value">{{ store.webPort || ':8080' }}</strong>
          </div>
        </div>
      </div>
    </div>

    <div class="glass-panel rounded-[28px] p-5 sm:p-6">
      <div class="relative z-[1] grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_360px]">
        <div class="space-y-4">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-xl font-bold text-[var(--text-primary)]">Proxy-Liste</h3>
              <p class="text-sm text-[var(--text-muted)]">
                Ziehe die Karten an der Griffleiste, um deine Arbeitsreihenfolge visuell zu organisieren.
              </p>
            </div>
            <Button
              label="Proxy hinzufügen"
              icon="pi pi-plus"
              @click="addProxy"
              class="w-full sm:w-auto"
            />
          </div>

          <div v-if="store.proxies.length === 0" class="empty-state rounded-[24px] border border-dashed border-gray-300 dark:border-white/15 p-8 text-center">
            <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-gray-100 dark:bg-white/5">
              <i class="pi pi-inbox text-2xl text-[var(--text-secondary)]"></i>
            </div>
            <h4 class="text-lg font-semibold text-[var(--text-primary)]">Noch keine Proxies angelegt</h4>
            <p class="mx-auto mt-2 max-w-md text-sm text-[var(--text-muted)]">
              Lege deinen ersten Proxy an und verwalte danach Reihenfolge, Status und Zeitlimits direkt in dieser Oberfläche.
            </p>
          </div>

          <VueDraggable
            v-else
            v-model="store.proxies"
            :animation="180"
            handle=".proxy-drag-handle"
            ghostClass="proxy-ghost"
            chosenClass="proxy-chosen"
            dragClass="proxy-drag"
            class="space-y-4"
            @end="onReorder"
          >
            <article
              v-for="(proxy, index) in store.proxies"
              :key="getProxyKey(proxy, index)"
              class="proxy-card rounded-[24px] p-4 sm:p-5"
              :class="{ 'proxy-card--dirty': isProxyDirty(proxy), 'proxy-card--selected': activeProxyKey === getProxyKey(proxy, index) }"
              @click="activeProxyKey = getProxyKey(proxy, index)"
            >
              <div class="flex flex-col gap-4">
                <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                  <div class="flex items-start gap-3">
                    <button
                      type="button"
                       class="proxy-drag-handle mt-1 flex h-11 w-11 items-center justify-center rounded-2xl border border-gray-300 dark:border-white/10 bg-gray-100 dark:bg-white/5 text-[var(--text-secondary)] transition hover:border-gray-400 dark:hover:border-white/20 hover:text-[var(--text-primary)]"
                      title="Proxy verschieben"
                    >
                      <GripVerticalIcon class="h-5 w-5" />
                    </button>

                    <div class="space-y-2">
                      <div class="flex flex-wrap items-center gap-2">
                        <span class="inline-flex items-center gap-2 rounded-full border border-gray-300 dark:border-white/10 bg-gray-100 dark:bg-white/5 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-[var(--text-muted)]">
                          <span class="status-dot" :class="statusDotClass(proxy.status)"></span>
                          {{ proxy.status || 'Draft' }}
                        </span>
                        <span v-if="proxy._isNew" class="proxy-pill proxy-pill--info">Neu</span>
                        <span v-if="proxy._isDirty" class="proxy-pill proxy-pill--warning">Geändert</span>
                        <span class="proxy-pill">{{ normalizeTags(proxy).length }} Tags</span>
                      </div>
                      <div>
                        <h4 class="text-lg font-semibold text-[var(--text-primary)]">
                          {{ proxy.name || `Proxy ${index + 1}` }}
                        </h4>
                        <p class="text-sm text-[var(--text-muted)]">
                          {{ proxy.listen_addr || ':5020' }} -> {{ proxy.target_addr || '127.0.0.1:502' }}
                        </p>
                      </div>
                    </div>
                  </div>

                  <div class="flex flex-wrap items-center gap-2">
                    <Button
                      v-if="proxy._isDirty || proxy._isNew"
                      :label="proxy._isNew ? 'Erstellen' : 'Speichern'"
                      icon="pi pi-save"
                      @click.stop="saveProxy(proxy, index)"
                      :loading="store.isLoading && activeSaveKey === getProxyKey(proxy, index)"
                      size="small"
                    />
                    <Button
                      :label="proxy._showAdvanced ? 'Weniger' : 'Mehr'"
                      :icon="proxy._showAdvanced ? 'pi pi-chevron-up' : 'pi pi-chevron-down'"
                      severity="secondary"
                      text
                      @click.stop="proxy._showAdvanced = !proxy._showAdvanced"
                      size="small"
                    />
                    <Button
                      icon="pi pi-trash"
                      severity="danger"
                      text
                      rounded
                      @click.stop="removeProxy(proxy.id, index)"
                      size="small"
                    />
                  </div>
                </div>

                <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-12">
                  <div class="field-group xl:col-span-4">
                    <label>Name</label>
                    <input v-model="proxy.name" type="text" placeholder="Factory Line A" @input="markDirty(proxy, index)" />
                    <small v-if="getFieldError(proxy, index, 'name')" class="field-error">{{ getFieldError(proxy, index, 'name') }}</small>
                  </div>

                  <div class="field-group xl:col-span-4">
                    <label>Listen Addr</label>
                    <input v-model="proxy.listen_addr" type="text" placeholder=":5020" @input="markDirty(proxy, index)" />
                    <small v-if="getFieldError(proxy, index, 'listen_addr')" class="field-error">{{ getFieldError(proxy, index, 'listen_addr') }}</small>
                  </div>

                  <div class="field-group xl:col-span-4">
                    <label>Target Addr</label>
                    <input v-model="proxy.target_addr" type="text" placeholder="192.168.1.100:502" @input="markDirty(proxy, index)" />
                    <small v-if="getFieldError(proxy, index, 'target_addr')" class="field-error">{{ getFieldError(proxy, index, 'target_addr') }}</small>
                  </div>

                  <div class="field-group xl:col-span-6">
                    <label>Beschreibung</label>
                    <input v-model="proxy.description" type="text" placeholder="Optionaler Hinweis zur Anlage" @input="markDirty(proxy, index)" />
                  </div>

                  <div class="field-group xl:col-span-6">
                    <label>Tags</label>
                    <input
                      v-model="proxy.tags"
                      type="text"
                      placeholder="production, critical, plc"
                      @input="markDirty(proxy, index)"
                    />
                  </div>
                </div>

                <div class="flex flex-col gap-3 rounded-[20px] border border-gray-200 dark:border-white/10 bg-gray-50 dark:bg-black/10 p-4 lg:flex-row lg:items-center lg:justify-between">
                  <div class="flex flex-wrap gap-3">
                    <label class="toggle-chip">
                      <input type="checkbox" v-model="proxy.enabled" @change="markDirty(proxy, index)" />
                      <span>Aktiviert</span>
                    </label>
                    <label class="toggle-chip">
                      <input type="checkbox" v-model="proxy.paused" @change="markDirty(proxy, index)" />
                      <span>Pausiert</span>
                    </label>
                  </div>

                  <div class="flex flex-wrap gap-2">
                    <span v-for="tag in normalizeTags(proxy)" :key="`${getProxyKey(proxy, index)}-${tag}`" class="proxy-pill">
                      {{ tag }}
                    </span>
                    <span v-if="normalizeTags(proxy).length === 0" class="text-xs text-[var(--text-muted)]">
                      Keine Tags gesetzt
                    </span>
                  </div>
                </div>

                <div v-if="proxy._showAdvanced" class="advanced-grid rounded-[20px] border border-gray-200 dark:border-white/10 bg-gray-50 dark:bg-black/10 p-4">
                  <div class="field-group">
                    <label>Conn Timeout (s)</label>
                    <input v-model.number="proxy.connection_timeout" type="number" min="1" max="300" @input="markDirty(proxy, index)" />
                    <small v-if="getFieldError(proxy, index, 'connection_timeout')" class="field-error">{{ getFieldError(proxy, index, 'connection_timeout') }}</small>
                  </div>
                  <div class="field-group">
                    <label>Read Timeout (s)</label>
                    <input v-model.number="proxy.read_timeout" type="number" min="1" max="300" @input="markDirty(proxy, index)" />
                    <small v-if="getFieldError(proxy, index, 'read_timeout')" class="field-error">{{ getFieldError(proxy, index, 'read_timeout') }}</small>
                  </div>
                  <div class="field-group">
                    <label>Max Retries</label>
                    <input v-model.number="proxy.max_retries" type="number" min="0" max="10" @input="markDirty(proxy, index)" />
                    <small v-if="getFieldError(proxy, index, 'max_retries')" class="field-error">{{ getFieldError(proxy, index, 'max_retries') }}</small>
                  </div>
                  <div class="field-group">
                    <label>Max Read Size</label>
                    <input v-model.number="proxy.max_read_size" type="number" min="0" max="65535" @input="markDirty(proxy, index)" />
                    <small v-if="getFieldError(proxy, index, 'max_read_size')" class="field-error">{{ getFieldError(proxy, index, 'max_read_size') }}</small>
                  </div>
                </div>
              </div>
            </article>
          </VueDraggable>
        </div>

        <aside class="space-y-4">
          <div class="side-card rounded-[24px] p-5">
            <div class="space-y-3">
              <div>
                <p class="text-xs uppercase tracking-[0.24em] text-[var(--text-muted)]">Web Interface</p>                <h3 class="mt-1 text-xl font-bold text-[var(--text-primary)]">Port und Zugriff</h3>
              </div>
              <div class="field-group">
                <label>Web Interface Address</label>
                <div class="flex gap-2">
                  <input
                    v-model="store.webPort"
                    type="text"
                    class="flex-1"
                    placeholder=":8080"
                  >
                  <Button
                    label="Speichern"
                    icon="pi pi-check"
                    @click="savePort"
                    :loading="store.isLoading"
                  />
                </div>
              </div>
              <p class="text-sm text-[var(--text-muted)]">
                Eine Port-Änderung benötigt einen Neustart des Dienstes. Die Eingabe akzeptiert `:8080` oder `host:8080`.
              </p>
            </div>
          </div>

          <div class="side-card rounded-[24px] p-5">
            <div class="space-y-3">
              <h3 class="text-xl font-bold text-[var(--text-primary)]">Workflow</h3>
              <ol class="space-y-3 text-sm text-[var(--text-secondary)]">
                <li class="workflow-step">
                  <span class="workflow-badge">1</span>
                  Karten verschieben, um deine bevorzugte Arbeitsreihenfolge zu setzen.
                </li>
                <li class="workflow-step">
                  <span class="workflow-badge">2</span>
                  Änderungen pro Karte prüfen und nur betroffene Einträge speichern.
                </li>
                <li class="workflow-step">
                  <span class="workflow-badge">3</span>
                  Erweiterte Timeout- und Retry-Werte bei Bedarf ausklappen.
                </li>
              </ol>
            </div>
          </div>
        </aside>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, ref } from 'vue';
import Button from 'primevue/button';
import { VueDraggable } from 'vue-draggable-plus';
import GripVerticalIcon from './icons/GripVertical.vue';
import { useAppStore } from '../stores/appStore';
import validators from '../utils/validators';

const store = useAppStore();

const validationErrors = ref({});
const activeProxyKey = ref(null);
const activeSaveKey = ref(null);

const dirtyCount = computed(() => store.proxies.filter(proxy => proxy._isDirty || proxy._isNew).length);
const runningCount = computed(() => store.proxies.filter(proxy => proxy.status === 'Running').length);

const getProxyKey = (proxy, index) => proxy.id || proxy._tempId || `proxy-${index}`;

const getValidationKey = (proxy, index) => getProxyKey(proxy, index);

const isProxyDirty = (proxy) => Boolean(proxy._isDirty || proxy._isNew);

const normalizeTags = (proxy) => {
  if (Array.isArray(proxy.tags)) return proxy.tags.filter(Boolean);
  if (typeof proxy.tags !== 'string') return [];
  return proxy.tags.split(',').map(tag => tag.trim()).filter(Boolean);
};

const getFieldError = (proxy, index, field) => {
  return validationErrors.value[getValidationKey(proxy, index)]?.[field] || '';
};

const statusDotClass = (status) => {
  switch (status) {
    case 'Running':
      return 'status-dot--running';
    case 'Stopped':
      return 'status-dot--stopped';
    case 'Error':
      return 'status-dot--error';
    default:
      return 'status-dot--unknown';
  }
};

const addProxy = () => {
  const tempId = `tmp_${Date.now()}`;
  store.proxies.unshift({
    id: '',
    _tempId: tempId,
    name: 'New Proxy',
    listen_addr: ':5020',
    target_addr: '127.0.0.1:502',
    enabled: true,
    paused: false,
    connection_timeout: 10,
    read_timeout: 30,
    max_retries: 3,
    max_read_size: 0,
    description: '',
    tags: '',
    _isNew: true,
    _showAdvanced: true
  });
  activeProxyKey.value = tempId;
};

const removeProxy = async (id, index) => {
  const proxy = store.proxies[index];
  if (!id) {
    store.proxies.splice(index, 1);
    delete validationErrors.value[getValidationKey(proxy, index)];
    return;
  }

  if (confirm('Moechtest du diesen Proxy wirklich entfernen?')) {
    await store.deleteProxy(id);
  }
};

const saveProxy = async (proxy, index) => {
  const errors = validators.validateProxyConfig(proxy);
  const validationKey = getValidationKey(proxy, index);

  if (errors) {
    validationErrors.value[validationKey] = errors;
    activeProxyKey.value = validationKey;
    return;
  }

  delete validationErrors.value[validationKey];
  activeSaveKey.value = validationKey;

  const { _isNew, _isDirty, _showAdvanced, _tempId, ...proxyData } = proxy;
  proxyData.tags = normalizeTags(proxy);

  let success = false;
  if (_isNew) {
    success = await store.addProxy(proxyData);
  } else {
    success = await store.updateProxy(proxyData);
  }

  if (success) {
    activeProxyKey.value = null;
  }

  activeSaveKey.value = null;
};

const savePort = async () => {
  if (confirm('Eine Port-Aenderung erfordert einen Neustart. Fortfahren?')) {
    await store.saveWebPort(store.webPort);
  }
};

const markDirty = (proxy, index) => {
  if (!proxy._isNew) {
    proxy._isDirty = true;
  }
  activeProxyKey.value = getProxyKey(proxy, index);
  delete validationErrors.value[getValidationKey(proxy, index)];
};

const onReorder = () => {
  activeProxyKey.value = null;
};
</script>

<style scoped>
.config-shell {
  color: var(--text-primary);
}

.hero-stat,
.side-card,
.proxy-card,
.empty-state {
  position: relative;
  overflow: hidden;
  background: var(--bg-panel-item);
  border: 1px solid var(--border-subtle);
  box-shadow: var(--shadow-soft);
  backdrop-filter: blur(18px);
  -webkit-backdrop-filter: blur(18px);
}

.hero-stat {
  min-width: 0;
  border-radius: 20px;
  padding: 0.9rem 1rem;
}

.hero-stat-label {
  display: block;
  color: var(--text-muted);
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.2em;
}

.hero-stat-value {
  display: block;
  margin-top: 0.45rem;
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--text-primary);
}

.proxy-card {
  transition: transform 0.22s ease, border-color 0.22s ease, box-shadow 0.22s ease;
}

.proxy-card:hover {
  transform: translateY(-2px);
  border-color: var(--border-strong);
}

.proxy-card--dirty {
  border-color: rgba(125, 211, 252, 0.32);
  box-shadow: 0 20px 50px rgba(56, 189, 248, 0.12);
}

.proxy-card--selected {
  border-color: rgba(192, 132, 252, 0.34);
}

.proxy-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border-radius: 999px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-panel-item);
  padding: 0.3rem 0.65rem;
  font-size: 0.72rem;
  color: var(--text-secondary);
}

.proxy-pill--info {
  color: #c4b5fd;
}

.proxy-pill--warning {
  color: #fde68a;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.field-group label {
  font-size: 0.78rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.16em;
  color: var(--text-muted);
}

.field-group input {
  width: 100%;
  border-radius: 16px;
  border: 1px solid var(--border-soft);
  background: var(--bg-input);
  color: var(--text-primary);
  padding: 0.9rem 1rem;
  outline: none;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.field-group input:focus {
  border-color: rgba(125, 211, 252, 0.4);
  box-shadow: 0 0 0 4px rgba(56, 189, 248, 0.12);
}

.field-error {
  color: #dc2626;
}
:root:not(.light) .field-error {
  color: #fda4af;
}

.toggle-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  border-radius: 999px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-panel-item);
  padding: 0.65rem 0.95rem;
  color: var(--text-secondary);
  cursor: pointer;
}

.toggle-chip input {
  accent-color: var(--accent-strong);
}

.advanced-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}

.workflow-step {
  display: flex;
  gap: 0.85rem;
  align-items: flex-start;
}

.workflow-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.8rem;
  height: 1.8rem;
  border-radius: 999px;
  background: var(--bg-panel-item);
  color: var(--text-primary);
  font-weight: 800;
  flex-shrink: 0;
}

:deep(.proxy-ghost) {
  opacity: 0.4;
}

:deep(.proxy-chosen) {
  transform: scale(1.01);
}

:deep(.proxy-drag) {
  cursor: grabbing;
}

@media (max-width: 768px) {
  .hero-stat {
    padding: 0.8rem 0.9rem;
  }
}
</style>
