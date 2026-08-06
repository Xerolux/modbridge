<script setup>
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import InputText from 'primevue/inputtext';
import Password from 'primevue/password';
import Button from 'primevue/button';
import axios from '../axios.js';
import BrandMark from '../components/BrandMark.vue';

const { t } = useI18n();
const router = useRouter();
const available = ref(false);
const checking = ref(true);
const loading = ref(false);
const success = ref(false);
const error = ref('');
const token = ref('');
const username = ref('');
const password = ref('');
const confirmPassword = ref('');

onMounted(async () => {
  try {
    const response = await axios.get('/api/account-recovery', { skipAuth: true });
    available.value = response.data.available === true;
  } catch {
    available.value = false;
  } finally {
    checking.value = false;
  }
});

const recover = async () => {
  error.value = '';
  if (password.value !== confirmPassword.value) {
    error.value = t('accountRecovery.mismatch');
    return;
  }
  loading.value = true;
  try {
    await axios.post('/api/account-recovery', {
      token: token.value.trim(),
      new_username: username.value.trim(),
      new_password: password.value,
    }, { skipAuth: true });
    success.value = true;
    available.value = false;
  } catch (requestError) {
    error.value = requestError.response?.data?.trim() || t('accountRecovery.error');
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="recovery-stage flex items-center justify-center min-h-[90vh] px-4 py-8">
    <div class="w-full max-w-[440px] flex flex-col gap-7">
      <div class="flex flex-col items-center gap-3">
        <div class="brand-mark"><BrandMark /></div>
        <div class="text-center">
          <h1 class="text-2xl font-bold text-[var(--text-primary)]">{{ t('accountRecovery.title') }}</h1>
          <p class="text-sm text-[var(--text-muted)] mt-1">{{ t('accountRecovery.subtitle') }}</p>
        </div>
      </div>

      <div class="recovery-card flex flex-col gap-5">
        <div v-if="checking" class="text-center text-sm text-[var(--text-muted)]">
          {{ t('accountRecovery.checking') }}
        </div>

        <template v-else-if="success">
          <div class="message success" role="status">
            <i class="pi pi-check-circle"></i>
            <span>{{ t('accountRecovery.success') }}</span>
          </div>
          <Button :label="t('accountRecovery.toLogin')" icon="pi pi-sign-in" @click="router.replace('/login')" />
        </template>

        <template v-else-if="!available">
          <div class="message info">
            <i class="pi pi-info-circle"></i>
            <span>{{ t('accountRecovery.notEnabled') }}</span>
          </div>
          <code class="command">./modbridge --enable-account-recovery</code>
          <p class="text-xs leading-5 text-[var(--text-muted)]">{{ t('accountRecovery.commandHint') }}</p>
          <Button :label="t('accountRecovery.checkAgain')" icon="pi pi-refresh" severity="secondary" @click="router.go(0)" />
          <Button :label="t('accountRecovery.back')" text @click="router.replace('/login')" />
        </template>

        <template v-else>
          <div class="message info">
            <i class="pi pi-shield"></i>
            <span>{{ t('accountRecovery.enabled') }}</span>
          </div>

          <div class="field">
            <label for="recovery-token">{{ t('accountRecovery.code') }}</label>
            <InputText id="recovery-token" v-model="token" class="w-full" autocomplete="one-time-code" />
          </div>
          <div class="field">
            <label for="recovery-username">{{ t('accountRecovery.newUsername') }}</label>
            <InputText id="recovery-username" v-model="username" class="w-full" autocomplete="username" />
          </div>
          <div class="field">
            <label for="recovery-password">{{ t('accountRecovery.newPassword') }}</label>
            <Password id="recovery-password" v-model="password" toggleMask class="w-full" inputClass="w-full" autocomplete="new-password" />
          </div>
          <div class="field">
            <label for="recovery-confirm">{{ t('accountRecovery.confirmPassword') }}</label>
            <Password id="recovery-confirm" v-model="confirmPassword" :feedback="false" toggleMask class="w-full" inputClass="w-full" autocomplete="new-password" />
          </div>

          <div v-if="error" class="message error" role="alert">
            <i class="pi pi-exclamation-circle"></i>
            <span>{{ error }}</span>
          </div>

          <Button
            :label="t('accountRecovery.submit')"
            icon="pi pi-key"
            :loading="loading"
            :disabled="!token.trim() || !username.trim() || !password || !confirmPassword"
            @click="recover"
          />
          <Button :label="t('accountRecovery.back')" text @click="router.replace('/login')" />
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.recovery-stage { background: transparent; }
.brand-mark { width: 4rem; height: 4rem; display: grid; place-items: center; }
.recovery-card {
  background: var(--bg-surface-strong);
  border: 1px solid var(--border-soft);
  border-radius: 28px;
  padding: 2rem;
  box-shadow: var(--shadow-strong);
}
.field { display: flex; flex-direction: column; gap: 0.5rem; }
.field label { font-size: 0.78rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.1em; color: var(--text-muted); }
.message { display: flex; align-items: flex-start; gap: 0.65rem; padding: 0.85rem 1rem; border-radius: 16px; font-size: 0.875rem; }
.message.info { background: rgba(59, 130, 246, 0.1); color: var(--text-secondary); }
.message.success { background: rgba(34, 197, 94, 0.1); color: var(--success); }
.message.error { background: rgba(251, 113, 133, 0.1); color: var(--danger); }
.command { display: block; overflow-wrap: anywhere; padding: 0.85rem 1rem; border-radius: 14px; background: var(--bg-panel-item); color: var(--text-primary); font-size: 0.8rem; }
</style>
