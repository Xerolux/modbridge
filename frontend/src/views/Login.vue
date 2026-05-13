<script setup>
import { ref, onMounted } from 'vue';
import { useAuthStore } from '../stores/auth';
import { useRouter } from 'vue-router';
import InputText from 'primevue/inputtext';
import Button from 'primevue/button';
import axios from '../axios.js';

const username = ref('');
const password = ref('');
const error = ref('');
const auth = useAuthStore();
const router = useRouter();
const loading = ref(false);
const multiUser = ref(false);

onMounted(async () => {
  try {
    const res = await axios.get('/api/status', { skipAuth: true });
    multiUser.value = res.data.multi_user === true;
  } catch {
    multiUser.value = false;
  }
});

const handleLogin = async () => {
  loading.value = true;
  error.value = '';
  const payload = { password: password.value };
  if (multiUser.value) payload.username = username.value;
  const result = await auth.login(payload);
  loading.value = false;
  if (result.success) {
    router.push('/');
  } else {
    error.value = result.message || 'Ungültige Anmeldedaten';
  }
};
</script>

<template>
  <div class="login-stage flex items-center justify-center min-h-[90vh] px-4 py-8">
    <div class="w-full max-w-[360px] flex flex-col gap-8">

      <!-- Brand mark -->
      <div class="flex flex-col items-center gap-4">
        <div class="brand-ring">
          <img src="../assets/logo.png" alt="ModBridge" class="w-12 h-12 object-contain" />
        </div>
        <div class="text-center">
          <h1 class="text-2xl font-bold tracking-tight text-[var(--text-primary)]">ModBridge</h1>
          <p class="text-sm text-[var(--text-muted)] mt-1">Industrial Modbus Proxy Manager</p>
        </div>
      </div>

      <!-- Form card -->
      <div class="login-card flex flex-col gap-5">
        <p class="text-sm text-center text-[var(--text-secondary)]">
          {{ multiUser ? 'Mit Zugangsdaten anmelden' : 'Passwort eingeben um fortzufahren' }}
        </p>

        <div v-if="multiUser" class="flex flex-col gap-2">
          <label for="login-username" class="login-label">Benutzername</label>
          <InputText
            id="login-username"
            v-model="username"
            @keyup.enter="handleLogin"
            placeholder="benutzername"
            class="w-full"
            autocomplete="username"
          />
        </div>

        <div class="flex flex-col gap-2">
          <label for="login-password" class="login-label">Passwort</label>
          <InputText
            id="login-password"
            v-model="password"
            type="password"
            @keyup.enter="handleLogin"
            placeholder="••••••••"
            class="w-full"
            autocomplete="current-password"
          />
        </div>

        <div v-if="error" class="login-error" role="alert">
          <i class="pi pi-exclamation-circle shrink-0 text-sm"></i>
          <span>{{ error }}</span>
        </div>

        <Button
          label="Anmelden"
          icon="pi pi-sign-in"
          :loading="loading"
          @click="handleLogin"
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
