<script setup>
import { useI18n } from 'vue-i18n';
const model = defineModel({ type: String, default: '' });
const { t } = useI18n();
</script>
<template><input v-model="model" type="search" class="workspace-search" :placeholder="t('workspace.search')" :aria-label="t('workspace.search')" /></template>
