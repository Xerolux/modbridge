<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { RouterLink, useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { useAuthStore } from '../stores/auth';
import { useAppStore } from '../stores/appStore';
import BrandMark from './BrandMark.vue';
import LanguageSelector from './LanguageSelector.vue';
import ThemeSettings from './ThemeSettings.vue';

const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const auth = useAuthStore();
const app = useAppStore();
const mobileMenuOpen = ref(false);
const mobileMenuButton = ref(null);
const mobileDrawer = ref(null);
const mobileCloseButton = ref(null);
let previousBodyOverflow = '';

const item = (labelKey, icon, path, permission) => ({ labelKey, icon, path, permission });
const navGroups = [
  {
    labelKey: 'nav.groupProxies',
    items: [
      item('nav.dashboard', 'pi pi-home', '/', null),
      item('nav.control', 'pi pi-sliders-h', '/control', 'proxy:view'),
      item('nav.devices', 'pi pi-desktop', '/devices', 'device:view')
    ]
  },
  {
    labelKey: 'nav.groupSecurity',
    items: [
      item('nav.users', 'pi pi-users', '/users', 'user:view'),
      item('nav.audit', 'pi pi-history', '/audit', 'audit:view')
    ]
  },
  {
    labelKey: 'nav.groupSystem',
    items: [
      item('nav.settings', 'pi pi-cog', '/config', 'config:view'),
      item('nav.system', 'pi pi-info-circle', '/system', 'system:view'),
      item('nav.logs', 'pi pi-list', '/logs', 'logs:view')
    ]
  }
];

const itemVisible = (entry) => !entry.permission || auth.isAdmin || auth.hasPermission(entry.permission);
const visibleGroups = computed(() => navGroups
  .map(group => ({ ...group, items: group.items.filter(itemVisible) }))
  .filter(group => group.items.length));
const allVisibleItems = computed(() => visibleGroups.value.flatMap(group => group.items));
const currentItem = computed(() => allVisibleItems.value.find(entry => isActiveRoute(entry.path)) || allVisibleItems.value[0]);
const proxyCount = computed(() => app.proxies?.length ?? 0);
const runningProxyCount = computed(() => app.proxies?.filter(proxy => proxy.status === 'Running').length ?? 0);

const isActiveRoute = (path) => path === '/' ? route.path === '/' : route.path.startsWith(path);

const logout = async () => {
  await auth.logout();
  router.push('/login');
};

const closeMobileMenu = () => { mobileMenuOpen.value = false; };
const handleKeydown = (event) => {
  if (event.key === 'Escape') closeMobileMenu();
  if (event.key !== 'Tab' || !mobileMenuOpen.value || !mobileDrawer.value) return;

  const focusable = [...mobileDrawer.value.querySelectorAll(
    'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
  )];
  if (!focusable.length) return;

  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
};

watch(mobileMenuOpen, async (open) => {
  if (open) {
    previousBodyOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    await nextTick();
    mobileCloseButton.value?.focus();
  } else {
    document.body.style.overflow = previousBodyOverflow;
    mobileMenuButton.value?.focus();
  }
});

watch(() => route.path, closeMobileMenu);

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
  app.fetchProxies();
});
onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
  document.body.style.overflow = previousBodyOverflow;
});
</script>

<template>
  <div class="app-layout">
    <aside class="sidebar hidden lg:flex" aria-label="Hauptnavigation">
      <button type="button" class="brand" @click="router.push('/')" aria-label="ModBridge Dashboard">
        <BrandMark />
        <span class="min-w-0 text-left">
          <strong class="block truncate text-[0.95rem] tracking-tight">ModBridge</strong>
          <small class="block truncate text-[0.64rem] uppercase tracking-[0.18em] text-[var(--text-muted)]">Proxy Manager</small>
        </span>
      </button>

      <div class="system-summary" aria-live="polite">
        <span class="system-summary__icon"><i class="pi pi-bolt"></i></span>
        <span class="min-w-0">
          <strong>{{ runningProxyCount }} / {{ proxyCount }}</strong>
          <small>{{ t('common.running') }}</small>
        </span>
        <span class="status-dot ml-auto" :class="runningProxyCount > 0 ? 'status-dot--running' : 'status-dot--unknown'"></span>
      </div>

      <nav class="sidebar-nav">
        <section v-for="group in visibleGroups" :key="group.labelKey" class="nav-section">
          <h2>{{ t(group.labelKey) }}</h2>
          <RouterLink
            v-for="entry in group.items"
            :key="entry.path"
            :to="entry.path"
            class="sidebar-link"
            :class="{ 'sidebar-link--active': isActiveRoute(entry.path) }"
            :aria-current="isActiveRoute(entry.path) ? 'page' : undefined"
          >
            <span class="sidebar-link__icon"><i :class="entry.icon"></i></span>
            <span>{{ t(entry.labelKey) }}</span>
            <i v-if="isActiveRoute(entry.path)" class="pi pi-chevron-right ml-auto text-[0.62rem]"></i>
          </RouterLink>
        </section>
      </nav>

      <div class="sidebar-footer">
        <div class="sidebar-tools">
          <ThemeSettings />
          <LanguageSelector />
        </div>
        <div class="user-card">
          <span class="user-avatar">{{ auth.user.username?.slice(0, 1).toUpperCase() || 'U' }}</span>
          <span class="min-w-0 flex-1">
            <strong class="block truncate text-xs">{{ auth.user.username }}</strong>
            <small class="block truncate text-[0.68rem] text-[var(--text-muted)]">{{ auth.user.role }}</small>
          </span>
          <button type="button" class="icon-button icon-button--danger" @click="logout" :title="t('nav.logout')" :aria-label="t('nav.logout')">
            <i class="pi pi-power-off"></i>
          </button>
        </div>
      </div>
    </aside>

    <div class="main-column">
      <header class="mobile-header flex lg:hidden">
        <button type="button" class="brand" @click="router.push('/')" aria-label="ModBridge Dashboard">
          <BrandMark />
          <span class="font-bold tracking-tight">ModBridge</span>
        </button>
        <div class="flex items-center gap-2">
          <span class="mobile-status"><span class="status-dot" :class="runningProxyCount > 0 ? 'status-dot--running' : 'status-dot--unknown'"></span>{{ runningProxyCount }}/{{ proxyCount }}</span>
          <button ref="mobileMenuButton" type="button" class="icon-button" @click="mobileMenuOpen = true" :aria-label="t('nav.openNavigation')" :aria-expanded="mobileMenuOpen">
            <i class="pi pi-bars"></i>
          </button>
        </div>
      </header>

      <div class="page-context hidden lg:flex">
        <div>
          <span class="page-context__eyebrow">ModBridge</span>
          <strong>{{ currentItem ? t(currentItem.labelKey) : 'Dashboard' }}</strong>
        </div>
        <div class="page-context__state">
          <span class="status-dot" :class="proxyCount > 0 ? 'status-dot--running' : 'status-dot--unknown'"></span>
          {{ proxyCount }} {{ t('nav.proxiesCount') }}
        </div>
      </div>

      <main class="page-content">
        <router-view />
      </main>
    </div>

    <Transition name="fade">
      <button v-if="mobileMenuOpen" type="button" class="mobile-backdrop lg:hidden" aria-label="Navigation schließen" @click="closeMobileMenu"></button>
    </Transition>
    <Transition name="slide">
      <aside ref="mobileDrawer" v-if="mobileMenuOpen" class="mobile-drawer lg:hidden" role="dialog" aria-modal="true" aria-label="Navigation">
        <div class="drawer-header">
          <div class="brand"><BrandMark /><strong>ModBridge</strong></div>
          <button ref="mobileCloseButton" type="button" class="icon-button" @click="closeMobileMenu" :aria-label="t('nav.closeNavigation')"><i class="pi pi-times"></i></button>
        </div>
        <nav class="sidebar-nav">
          <section v-for="group in visibleGroups" :key="group.labelKey" class="nav-section">
            <h2>{{ t(group.labelKey) }}</h2>
            <RouterLink v-for="entry in group.items" :key="entry.path" :to="entry.path" class="sidebar-link" :class="{ 'sidebar-link--active': isActiveRoute(entry.path) }" @click="closeMobileMenu">
              <span class="sidebar-link__icon"><i :class="entry.icon"></i></span>
              <span>{{ t(entry.labelKey) }}</span>
            </RouterLink>
          </section>
        </nav>
        <div class="sidebar-footer mt-auto">
          <div class="sidebar-tools"><ThemeSettings /><LanguageSelector /></div>
          <button type="button" class="logout-button" @click="logout"><i class="pi pi-power-off"></i>{{ t('nav.logout') }}</button>
        </div>
      </aside>
    </Transition>
  </div>
</template>

<style scoped>
.app-layout { display: flex; min-height: 100vh; }
.sidebar {
  position: sticky; top: 0; width: 16.5rem; height: 100vh; flex: 0 0 16.5rem; flex-direction: column;
  padding: 1rem; border-right: 1px solid var(--border-subtle); background: color-mix(in srgb, var(--bg-surface-strong) 88%, transparent);
  backdrop-filter: var(--glass-blur); -webkit-backdrop-filter: var(--glass-blur); z-index: 40;
}
.brand { display: flex; align-items: center; gap: .7rem; min-width: 0; color: var(--text-primary); background: transparent; border: 0; cursor: pointer; }
.system-summary { display: flex; align-items: center; gap: .7rem; margin: 1rem 0 1.1rem; padding: .75rem; border: 1px solid var(--border-subtle); border-radius: 1rem; background: var(--bg-panel-item); }
.system-summary__icon { display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: .7rem; color: var(--accent); background: var(--accent-tint); }
.system-summary strong, .system-summary small { display: block; }
.system-summary strong { font-size: .82rem; color: var(--text-primary); }
.system-summary small { margin-top: .08rem; font-size: .66rem; color: var(--text-muted); }
.sidebar-nav { min-height: 0; flex: 1; overflow-y: auto; }
.nav-section + .nav-section { margin-top: 1.15rem; }
.nav-section h2 { margin: 0 0 .35rem .65rem; font: 700 .62rem/1.4 ui-sans-serif, system-ui, sans-serif; letter-spacing: .16em; text-transform: uppercase; color: var(--text-muted); }
.sidebar-link { position: relative; display: flex; align-items: center; gap: .7rem; min-height: 2.55rem; padding: .42rem .55rem; border-radius: .85rem; color: var(--text-secondary); text-decoration: none; font-size: .82rem; font-weight: 550; transition: background .16s ease, color .16s ease, transform .16s ease; }
.sidebar-link:hover { color: var(--text-primary); background: var(--bg-soft); transform: translateX(2px); }
.sidebar-link--active { color: var(--accent); background: var(--accent-tint); }
.sidebar-link--active::before { content: ''; position: absolute; left: -.35rem; width: 3px; height: 1.15rem; border-radius: 9px; background: var(--accent); box-shadow: 0 0 12px color-mix(in srgb, var(--accent) 55%, transparent); }
.sidebar-link__icon { display: grid; place-items: center; width: 1.8rem; height: 1.8rem; border-radius: .65rem; background: var(--bg-panel-item); }
.sidebar-footer { padding-top: .85rem; border-top: 1px solid var(--border-subtle); }
.sidebar-tools { display: flex; align-items: center; justify-content: space-between; gap: .5rem; padding: 0 .25rem .75rem; }
.user-card { display: flex; align-items: center; gap: .65rem; padding: .65rem; border: 1px solid var(--border-subtle); border-radius: 1rem; background: var(--bg-panel-item); }
.user-avatar { display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: .7rem; color: var(--accent); background: var(--accent-tint); font-size: .75rem; font-weight: 800; }
.icon-button { display: grid; place-items: center; width: 2.35rem; height: 2.35rem; flex: 0 0 auto; border: 0; border-radius: .8rem; color: var(--text-secondary); background: var(--bg-panel-item); cursor: pointer; transition: background .16s, color .16s; }
.icon-button:hover { color: var(--text-primary); background: var(--bg-soft); }
.icon-button--danger:hover { color: var(--danger); background: color-mix(in srgb, var(--danger) 12%, transparent); }
.main-column { min-width: 0; flex: 1; }
.page-context { align-items: center; justify-content: space-between; min-height: 4.4rem; padding: .75rem 1.5rem; border-bottom: 1px solid var(--border-subtle); }
.page-context__eyebrow { display: block; margin-bottom: .18rem; font-size: .62rem; font-weight: 700; letter-spacing: .16em; text-transform: uppercase; color: var(--text-muted); }
.page-context strong { font-size: .95rem; color: var(--text-primary); }
.page-context__state, .mobile-status { display: flex; align-items: center; gap: .5rem; padding: .45rem .7rem; border: 1px solid var(--border-subtle); border-radius: 999px; background: var(--bg-panel-item); color: var(--text-muted); font-size: .72rem; }
.page-content { width: 100%; max-width: 100rem; margin: 0 auto; padding: .75rem 1rem 1.5rem; }
.mobile-header { position: sticky; top: 0; z-index: 50; align-items: center; justify-content: space-between; padding: calc(.65rem + env(safe-area-inset-top)) calc(.8rem + env(safe-area-inset-right)) .65rem calc(.8rem + env(safe-area-inset-left)); border-bottom: 1px solid var(--border-subtle); background: color-mix(in srgb, var(--bg-surface-strong) 88%, transparent); backdrop-filter: var(--glass-blur); }
.mobile-backdrop { position: fixed; inset: 0; z-index: 80; border: 0; background: rgba(2, 6, 23, .56); backdrop-filter: blur(3px); }
.mobile-drawer { position: fixed; inset: 0 auto 0 0; z-index: 90; display: flex; width: min(20rem, 88vw); flex-direction: column; padding: calc(1rem + env(safe-area-inset-top)) 1rem calc(1rem + env(safe-area-inset-bottom)) calc(1rem + env(safe-area-inset-left)); border-right: 1px solid var(--border-soft); background: var(--bg-surface-strong); box-shadow: var(--shadow-strong); }
.drawer-header { display: flex; align-items: center; justify-content: space-between; padding-bottom: 1rem; margin-bottom: 1rem; border-bottom: 1px solid var(--border-subtle); }
.logout-button { display: flex; align-items: center; justify-content: center; gap: .55rem; width: 100%; min-height: 2.6rem; border: 1px solid color-mix(in srgb, var(--danger) 28%, var(--border-subtle)); border-radius: .85rem; color: var(--danger); background: transparent; cursor: pointer; }
.fade-enter-active, .fade-leave-active, .slide-enter-active, .slide-leave-active { transition: opacity .2s ease, transform .2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
.slide-enter-from, .slide-leave-to { transform: translateX(-100%); }
@media (max-width: 640px) { .page-content { padding: .5rem max(.35rem, env(safe-area-inset-right)) calc(1rem + env(safe-area-inset-bottom)) max(.35rem, env(safe-area-inset-left)); } .mobile-status { padding-inline: .55rem; } }
@media (min-width: 1024px) { .mobile-drawer { display: none !important; } }
</style>
