<script setup>
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useToast } from 'primevue/usetoast';
import { useAuthStore } from '../stores/auth';
import { useI18n } from 'vue-i18n';
import InputText from 'primevue/inputtext';
import Button from 'primevue/button';
import axios from '../axios.js';

const router = useRouter();
const toast = useToast();
const auth = useAuthStore();
const { t } = useI18n();

const currentPassword = ref('');
const newPassword = ref('');
const confirmPassword = ref('');
const loading = ref(false);
const error = ref('');

const submit = async () => {
  error.value = '';
  if (newPassword.value !== confirmPassword.value) {
    error.value = t('changePassword.mismatch');
    return;
  }
  if (newPassword.value.length < 8) {
    error.value = t('changePassword.tooShort');
    return;
  }
  loading.value = true;
  try {
    await axios.post('/api/config/password', {
      current_password: currentPassword.value,
      new_password: newPassword.value,
    });
    toast.add({ severity: 'success', summary: t('changePassword.success'), life: 3000 });
    // Sessions are invalidated server-side on password change, so force re-login.
    await auth.logout();
    router.push('/login');
  } catch (err) {
    const msg = err?.response?.data || err?.message || t('changePassword.error');
    error.value = typeof msg === 'string' ? msg : t('changePassword.error');
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="login-stage flex items-center justify-center min-h-[90vh] px-4 py-8">
    <div class="w-full max-w-[380px] flex flex-col gap-8">

      <div class="flex flex-col items-center gap-4">
        <div class="brand-ring">
          <img src="../assets/logo.png" alt="ModBridge" class="w-12 h-12 object-contain" />
        </div>
        <div class="text-center">
          <h1 class="text-2xl font-bold tracking-tight text-[var(--text-primary)]">ModBridge</h1>
          <p class="text-sm text-[var(--text-muted)] mt-1">{{ t('changePassword.subtitle') }}</p>
        </div>
      </div>

      <div class="login-card flex flex-col gap-5">
        <div class="flex flex-col gap-2">
          <label for="cp-current" class="login-label">{{ t('changePassword.current') }}</label>
          <InputText
            id="cp-current"
            v-model="currentPassword"
            type="password"
            class="w-full"
            autocomplete="current-password"
          />
        </div>

        <div class="flex flex-col gap-2">
          <label for="cp-new" class="login-label">{{ t('changePassword.new') }}</label>
          <InputText
            id="cp-new"
            v-model="newPassword"
            type="password"
            class="w-full"
            autocomplete="new-password"
          />
        </div>

        <div class="flex flex-col gap-2">
          <label for="cp-confirm" class="login-label">{{ t('changePassword.confirm') }}</label>
          <InputText
            id="cp-confirm"
            v-model="confirmPassword"
            type="password"
            class="w-full"
            autocomplete="new-password"
            @keyup.enter="submit"
          />
        </div>

        <div v-if="error" class="login-error" role="alert">
          <i class="pi pi-exclamation-circle shrink-0 text-sm"></i>
          <span>{{ error }}</span>
        </div>

        <Button
          :label="t('changePassword.submit')"
          icon="pi pi-key"
          :loading="loading"
          @click="submit"
          class="w-full"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-stage {
  background: transparent;
}
.brand-ring {
  position: relative;
  width: 5rem;
  height: 5rem;
  border-radius: 999px;
  background: var(--bg-panel-item);
  border: 1px solid var(--border-soft);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow:
    0 0 0 7px var(--bg-canvas),
    0 0 0 8px var(--border-subtle),
    var(--shadow-soft);
}
.login-card {
  background: var(--bg-surface-strong);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border: 1px solid var(--border-soft);
  border-radius: 28px;
  padding: 2rem;
  box-shadow: var(--shadow-strong);
}
.login-label {
  font-size: 0.78rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--text-muted);
}
.login-error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  border-radius: 16px;
  background: rgba(251, 113, 133, 0.1);
  border: 1px solid rgba(251, 113, 133, 0.22);
  color: var(--danger);
  font-size: 0.875rem;
}
</style>
