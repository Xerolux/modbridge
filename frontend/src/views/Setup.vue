<script setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import axios from '../axios';
import { useAuthStore } from '../stores/auth';
import { useAppStore } from '../stores/appStore';
import PageState from '../components/PageState.vue';
const { t } = useI18n();
const auth = useAuthStore();
const store = useAppStore();
const router = useRouter();
const step = ref(1);
const form = ref({ name: '', listen_addr: ':5020', target_addr: '', enabled: false, paused: false, connection_timeout: 5, read_timeout: 30, max_retries: 3 });
const busy = ref(false);
const failed = ref(false);
const createdId = ref('');
const diagnosis = ref(null);
const validAddress = (address, allowEmptyHost) => {
  const match = address.trim().match(/^(\[[0-9a-fA-F:]+\]|[a-zA-Z0-9.-]*):([0-9]+)$/);
  return !!match && (allowEmptyHost || !!match[1]) && Number(match[2]) > 0 && Number(match[2]) <= 65535;
};
const valid = computed(() => form.value.name.trim().length > 0 && form.value.name.trim().length <= 100 && validAddress(form.value.listen_addr, true) && validAddress(form.value.target_addr, false));
const save = async () => {
  if (!valid.value || busy.value || createdId.value) return;
  busy.value = true;
  failed.value = false;
  try {
    const response = await axios.post('/api/proxies', { ...form.value, name: form.value.name.trim(), listen_addr: form.value.listen_addr.trim(), target_addr: form.value.target_addr.trim() });
    createdId.value = response.data.id;
    step.value = 3;
    await store.fetchProxies();
  } catch { failed.value = true; }
  finally { busy.value = false; }
};
const diagnose = async () => {
  if (busy.value || !createdId.value) return;
  busy.value = true;
  failed.value = false;
  diagnosis.value = null;
  try {
    const response = await axios.get('/api/system/diagnostics/connectivity', { params: { proxy_id: createdId.value } });
    diagnosis.value = response.data[createdId.value];
  } catch { failed.value = true; }
  finally { busy.value = false; }
};
</script>
<template>
  <div class="setup-page p-2 sm:p-4 space-y-5">
    <h1 class="text-2xl font-bold">{{ t('setup.title') }}</h1>
    <ol class="setup-steps" :aria-label="t('setup.progress')">
      <li v-for="n in 3" :key="n" :aria-current="step === n ? 'step' : undefined">{{ n }} · {{ t(`setup.step${n}`) }}</li>
    </ol>
    <section class="glass-panel rounded-2xl space-y-5">
      <form v-if="step === 1" class="setup-fields" @submit.prevent="step = 2">
        <label for="setup-name">{{ t('setup.name') }}</label>
        <input id="setup-name" v-model="form.name" class="workspace-search" required maxlength="100" autocomplete="off" />
        <label for="setup-listen">{{ t('setup.listen') }}</label>
        <input id="setup-listen" v-model="form.listen_addr" class="workspace-search" required placeholder=":5020" aria-describedby="setup-address-help" />
        <label for="setup-target">{{ t('setup.target') }}</label>
        <input id="setup-target" v-model="form.target_addr" class="workspace-search" required placeholder="192.168.1.100:502" aria-describedby="setup-address-help" />
        <p id="setup-address-help" class="text-sm text-[var(--text-secondary)]">{{ t('setup.addressHint') }}</p>
        <button type="submit" class="workspace-action" :disabled="!valid">{{ t('setup.next') }}</button>
      </form>
      <template v-else-if="step === 2">
        <h2 class="text-xl font-semibold">{{ t('setup.review') }}</h2>
        <dl class="setup-review"><dt>{{ t('setup.name') }}</dt><dd>{{ form.name }}</dd><dt>{{ t('setup.listen') }}</dt><dd>{{ form.listen_addr }}</dd><dt>{{ t('setup.target') }}</dt><dd>{{ form.target_addr }}</dd></dl>
        <p>{{ t('setup.disabledHint') }}</p>
        <div class="flex flex-wrap gap-3"><button class="workspace-action" type="button" :disabled="busy" @click="step = 1">{{ t('setup.back') }}</button><button class="workspace-action" type="button" :disabled="busy" @click="save">{{ t('setup.save') }}</button></div>
      </template>
      <template v-else>
        <h2 class="text-xl font-semibold">{{ t('setup.saved') }}</h2>
        <p>{{ t('setup.savedHint') }}</p>
        <p>{{ t('setup.probeHint') }}</p>
        <div class="flex flex-wrap gap-3">
          <button v-if="auth.hasPermission('system:view')" type="button" class="workspace-action" :disabled="busy" @click="diagnose">{{ t('setup.diagnose') }}</button>
          <button type="button" class="workspace-action" @click="router.push('/control')">{{ t('setup.finish') }}</button>
        </div>
        <div v-if="diagnosis" class="data-health" role="status" :class="{ 'data-health--stale': !diagnosis.reachable }">{{ t(diagnosis.reachable ? 'setup.reachable' : 'setup.unreachable') }}<p v-if="!diagnosis.reachable">{{ t('setup.diagnosisHint') }}</p></div>
      </template>
      <PageState :loading="busy" :error="failed" @retry="step === 3 ? diagnose() : save()" />
    </section>
  </div>
</template>
<style scoped>
.setup-page { max-width: 58rem; margin: auto; }
.setup-steps { display: flex; flex-wrap: wrap; gap: .7rem; }
.setup-steps li { padding: .7rem 1rem; border-radius: .7rem; background: var(--bg-soft); }
.setup-steps [aria-current] { color: var(--accent); border: 1px solid var(--accent); }
.setup-fields { display: grid; gap: .7rem; }
.setup-fields input { width: 100%; }
.setup-fields label { font-weight: 600; }
.setup-review { display: grid; grid-template-columns: minmax(6rem, 1fr) 2fr; gap: .7rem; }
.setup-review dd { overflow-wrap: anywhere; }
</style>
