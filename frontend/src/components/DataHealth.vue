<script setup>
import { useI18n } from 'vue-i18n';
defineProps({ failed: Boolean, busy: Boolean, lastSuccess: Date, live: { default: undefined } });
defineEmits(['refresh']);
const { t, locale } = useI18n();
</script>
<template>
  <div class="data-health" :class="{ 'data-health--stale': failed || live === false }" role="status">
    <i :class="busy ? 'pi pi-spin pi-spinner' : failed || live === false ? 'pi pi-exclamation-circle' : 'pi pi-check-circle'" aria-hidden="true" />
    <span>{{ t(busy ? 'workspace.refreshing' : failed ? 'workspace.stale' : live === false ? 'workspace.reconnecting' : 'workspace.ready') }}</span>
    <time v-if="lastSuccess" :datetime="lastSuccess.toISOString()">{{ t('workspace.lastSuccess') }} {{ lastSuccess.toLocaleTimeString(locale) }}</time>
    <button type="button" class="workspace-action" :disabled="busy" @click="$emit('refresh')">{{ t('workspace.retry') }}</button>
  </div>
</template>
