<script setup>
import { onMounted } from 'vue';
import { useAuthStore } from '../stores/auth';
import { useRouter } from 'vue-router';
import { useToast } from 'primevue/usetoast';

const auth = useAuthStore();
const router = useRouter();
const toast = useToast();

onMounted(async () => {
  const valid = await auth.checkAuth();
  if (!valid) {
    toast.add({
      severity: 'warn',
      summary: 'Session expired',
      detail: 'Please login again',
      life: 3000,
    });
    router.push('/login');
  }
});
</script>

<template>
  <div></div>
</template>
