<script setup>
import { ref, onMounted } from 'vue';
import { useAuthStore } from '../stores/auth';
import { useRouter } from 'vue-router';
import Card from 'primevue/card';
import InputText from 'primevue/inputtext';
import Button from 'primevue/button';
import Message from 'primevue/message';
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
    if (multiUser.value) {
        payload.username = username.value;
    }
    const result = await auth.login(payload);
    loading.value = false;
    if (result.success) {
        router.push('/');
    } else {
        error.value = result.message || 'Invalid credentials';
    }
};
</script>

<template>
    <div class="flex items-center justify-center min-h-[80vh] px-4 py-8">
        <Card class="w-full max-w-md glass-card border border-white/10 shadow-2xl overflow-hidden relative">
            <template #title>
                <div class="text-2xl font-semibold tracking-tight text-surface-900 dark:text-white mb-2">Welcome Back</div>
                <div class="text-sm font-normal text-surface-400">
                    <span v-if="multiUser">Enter your credentials to continue</span>
                    <span v-else>Enter your password to continue</span>
                </div>
            </template>
            <template #content>
                <div class="flex flex-col gap-5 mt-4">
                    <div v-if="multiUser" class="flex flex-col gap-2">
                        <label for="username" class="text-sm font-medium text-surface-200">Username</label>
                        <InputText id="username" v-model="username" @keyup.enter="handleLogin" class="p-3 w-full bg-surface-800/50 border-surface-700/50 focus:border-primary-500 transition-colors rounded-xl" placeholder="Username" />
                    </div>
                    <div class="flex flex-col gap-2">
                        <label for="password" class="text-sm font-medium text-surface-200">Password</label>
                        <InputText id="password" v-model="password" type="password" @keyup.enter="handleLogin" class="p-3 w-full bg-surface-800/50 border-surface-700/50 focus:border-primary-500 transition-colors rounded-xl" placeholder="••••••••" />
                    </div>
                    <Message v-if="error" severity="error" class="text-sm rounded-xl">{{ error }}</Message>
                    <Button label="Login" @click="handleLogin" :loading="loading" class="btn-neon w-full p-3 font-semibold mt-2 rounded-xl" />
                </div>
            </template>
        </Card>
    </div>
</template>

<style scoped>
:deep(.p-card) {
    background: rgba(17, 24, 39, 0.4) !important;
    backdrop-filter: blur(24px) !important;
    -webkit-backdrop-filter: blur(24px) !important;
    border-radius: 24px;
}
:deep(.p-card-body) {
    padding: 2.5rem;
}
:deep(.p-inputtext) {
    background: rgba(31, 41, 55, 0.5);
    border: 1px solid rgba(75, 85, 99, 0.4);
}
:deep(.p-inputtext:focus) {
    background: rgba(31, 41, 55, 0.8);
    border-color: var(--p-primary-500);
    box-shadow: 0 0 0 2px rgba(168, 85, 247, 0.2);
}
</style>
