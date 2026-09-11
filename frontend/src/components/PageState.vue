<script setup>
import { useI18n } from 'vue-i18n';
defineProps({ loading: Boolean, error: Boolean, empty: Boolean });
defineEmits(['retry']);
const { t } = useI18n();
</script>
<template>
  <section v-if="loading || error || empty" class="page-state glass-panel" :role="error ? 'alert' : 'status'" :aria-busy="loading">
    <i :class="loading ? 'pi pi-spin pi-spinner' : error ? 'pi pi-exclamation-circle' : 'pi pi-inbox'" aria-hidden="true" />
    <strong>{{ t(loading ? 'workspace.loading' : error ? 'workspace.failed' : 'workspace.empty') }}</strong>
    <p>{{ t(error ? 'workspace.failureHint' : empty ? 'workspace.emptyHint' : 'workspace.loadingHint') }}</p>
    <button v-if="error" type="button" class="workspace-action" @click="$emit('retry')">{{ t('workspace.retry') }}</button>
  </section>
</template>
