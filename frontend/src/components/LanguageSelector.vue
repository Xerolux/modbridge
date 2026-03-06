<template>
    <div class="language-selector">
        <Button
            :icon="currentIcon"
            @click="toggleMenu"
            rounded
            severity="secondary"
            v-tooltip.bottom="$t('common.language')"
            class="lang-btn"
        />
        <Menu ref="menu" :model="items" :popup="true" />
    </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import Button from 'primevue/button';
import Menu from 'primevue/menu';
import { saveLanguage } from '../i18n';

const { locale } = useI18n();
const menu = ref();

const currentIcon = computed(() => {
    return locale.value === 'de' ? 'pi pi-flag' : 'pi pi-globe';
});

const items = ref([
    {
        label: 'Deutsch',
        icon: 'pi pi-flag',
        command: () => setLanguage('de')
    },
    {
        label: 'English',
        icon: 'pi pi-globe',
        command: () => setLanguage('en')
    }
]);

const toggleMenu = (event) => {
    menu.value.toggle(event);
};

const setLanguage = (lang) => {
    locale.value = lang;
    saveLanguage(lang);
};
</script>

<style scoped>
.language-selector {
    display: inline-block;
}

.lang-btn {
    width: 44px;
    height: 44px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.lang-btn:hover {
    transform: scale(1.1);
    box-shadow: 0 0 20px rgba(59, 130, 246, 0.4);
}
</style>
