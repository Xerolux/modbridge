<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import { RouterLink, useRouter, useRoute } from 'vue-router';
import { useAuthStore } from "../stores/auth";
import { useI18n } from 'vue-i18n';
import LanguageSelector from './LanguageSelector.vue';
import ThemeSettings from './ThemeSettings.vue';

const { t } = useI18n();

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();

const mobileMenuOpen = ref(false);

const allItems = [
  { labelKey: 'nav.dashboard', icon: 'pi pi-home',         path: '/',        permission: null },
  { labelKey: 'nav.control',   icon: 'pi pi-sliders-h',    path: '/control', permission: 'proxy:view' },
  { labelKey: 'nav.devices',   icon: 'pi pi-desktop',      path: '/devices', permission: 'device:view' },
  { labelKey: 'nav.logs',      icon: 'pi pi-list',         path: '/logs',    permission: 'logs:view' },
  { labelKey: 'nav.system',    icon: 'pi pi-info-circle',  path: '/system',  permission: 'system:view' },
  { labelKey: 'nav.settings',  icon: 'pi pi-cog',          path: '/config',  permission: 'config:view' },
  { labelKey: 'nav.users',     icon: 'pi pi-users',        path: '/users',   permission: 'user:view' },
  { labelKey: 'nav.audit',     icon: 'pi pi-history',      path: '/audit',   permission: 'audit:view' },
];

const items = computed(() =>
  allItems.filter(item => !item.permission || auth.isAdmin || auth.hasPermission(item.permission))
);

const logout = async () => {
  await auth.logout();
  router.push('/login');
};

const isActiveRoute = (path) => {
  if (path === '/') return route.path === '/';
  return route.path.startsWith(path);
};

const handleKeydown = (e) => {
  if (e.key === 'Escape') mobileMenuOpen.value = false;
};

onMounted(() => document.addEventListener('keydown', handleKeydown));
onUnmounted(() => document.removeEventListener('keydown', handleKeydown));
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <!-- ── Top Navigation ─────────────────────────────────────────── -->
    <header class="sticky top-0 z-50 px-3 py-2.5">
      <nav class="glass-card rounded-2xl flex items-center px-3 py-1.5 gap-2">

        <!-- Logo -->
        <button
          type="button"
          class="nav-logo"
          @click="router.push('/')"
          aria-label="Dashboard"
        >
          <img src="../assets/logo.png" alt="ModBridge" class="w-8 h-8 object-contain" />
          <span class="font-bold text-base tracking-tight hidden sm:block">ModBridge</span>
        </button>

        <div class="nav-divider hidden md:block"></div>

        <!-- Desktop nav links -->
        <div class="hidden md:flex items-center gap-0.5 flex-1 overflow-x-auto">
          <RouterLink
            v-for="item in items"
            :key="item.path"
            :to="item.path"
            class="nav-link"
            :class="{ 'nav-link--active': isActiveRoute(item.path) }"
            active-class=""
            exact-active-class=""
            :aria-current="isActiveRoute(item.path) ? 'page' : undefined"
          >
            <i :class="item.icon" class="text-sm shrink-0"></i>
            <span class="whitespace-nowrap">{{ t(item.labelKey) }}</span>
          </RouterLink>
        </div>

        <!-- Right controls -->
        <div class="flex items-center gap-1.5 ml-auto pl-1">
          <ThemeSettings />

          <LanguageSelector class="hidden sm:flex" />

          <div
            v-if="auth.user.username"
            class="hidden lg:flex items-center gap-1.5 px-2.5 py-1 rounded-xl border border-[var(--border-subtle)] bg-[var(--bg-panel-item)] text-sm select-none"
          >
            <i class="pi pi-user text-xs text-[var(--text-muted)]"></i>
            <span class="text-[var(--text-secondary)] max-w-[12rem] truncate">{{ auth.user.username }}</span>
            <span class="text-xs text-[var(--text-muted)] hidden xl:inline">({{ auth.user.role }})</span>
          </div>

          <button
            type="button"
            class="nav-icon-btn nav-icon-btn--danger hidden sm:flex"
            @click="logout"
            :title="t('nav.logout')"
            :aria-label="t('nav.logout')"
          >
            <i class="pi pi-power-off text-sm"></i>
          </button>

          <!-- Mobile hamburger -->
          <button
            type="button"
            class="nav-icon-btn md:hidden"
            @click="mobileMenuOpen = true"
            :aria-label="t('nav.openNavigation')"
          >
            <i class="pi pi-bars text-sm"></i>
          </button>
        </div>
      </nav>
    </header>

    <!-- ── Mobile backdrop ────────────────────────────────────────── -->
    <Transition name="fade">
      <div
        v-if="mobileMenuOpen"
        class="fixed inset-0 z-[9998] md:hidden bg-black/50 backdrop-blur-[2px]"
        @click="mobileMenuOpen = false"
        aria-hidden="true"
      ></div>
    </Transition>

    <!-- ── Mobile drawer ──────────────────────────────────────────── -->
    <Transition name="slide">
      <div
        v-if="mobileMenuOpen"
        class="drawer fixed left-0 top-0 h-full z-[9999] md:hidden w-72 flex flex-col"
        role="dialog"
        aria-modal="true"
        aria-label="Navigation"
      >
        <!-- Header -->
        <div class="flex items-center justify-between px-5 py-4 border-b border-[var(--border-soft)]">
          <div class="flex items-center gap-2.5">
            <img src="../assets/logo.png" alt="ModBridge" class="w-9 h-9 object-contain" />
            <span class="font-bold text-[17px] tracking-tight">ModBridge</span>
          </div>
          <button type="button" class="nav-icon-btn" @click="mobileMenuOpen = false" :aria-label="t('nav.closeNavigation')">
            <i class="pi pi-times text-sm"></i>
          </button>
        </div>

        <!-- Links -->
        <nav class="flex flex-col gap-1 p-4 flex-1 overflow-y-auto">
          <RouterLink
            v-for="item in items"
            :key="item.path"
            :to="item.path"
            class="mobile-nav-link"
            :class="{ 'mobile-nav-link--active': isActiveRoute(item.path) }"
            active-class=""
            exact-active-class=""
            @click="mobileMenuOpen = false"
          >
            <i :class="item.icon" class="text-base w-5 text-center shrink-0"></i>
            <span>{{ t(item.labelKey) }}</span>
          </RouterLink>
        </nav>

        <!-- Footer -->
        <div class="border-t border-[var(--border-soft)] p-4 flex flex-col gap-3">
          <div
            v-if="auth.user.username"
            class="flex items-center gap-2 px-3 py-2 rounded-xl bg-[var(--bg-panel-item)] border border-[var(--border-subtle)]"
          >
            <i class="pi pi-user text-sm text-[var(--text-muted)]"></i>
            <span class="text-sm text-[var(--text-secondary)] truncate">{{ auth.user.username }}</span>
            <span class="text-xs text-[var(--text-muted)] ml-auto shrink-0">({{ auth.user.role }})</span>
          </div>

          <div class="flex items-center justify-between px-1 py-1">
            <span class="text-sm text-[var(--text-secondary)]">{{ t('nav.theme') }}</span>
            <ThemeSettings />
          </div>

          <LanguageSelector />

          <button
            type="button"
            class="flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm text-[var(--danger)] hover:bg-[rgba(251,113,133,0.1)] transition-colors w-full"
            @click="logout"
          >
            <i class="pi pi-power-off"></i>
            <span>{{ t('nav.logout') }}</span>
          </button>
        </div>
      </div>
    </Transition>

    <!-- ── Page content ───────────────────────────────────────────── -->
    <main class="flex-grow w-full max-w-7xl mx-auto p-3 sm:p-4 pt-0">
      <router-view :key="route.fullPath" />
    </main>
  </div>
</template>

<style scoped>
/* ── Nav base ──────────────────────────────────────────────────────── */
.nav-logo {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px 4px 4px;
  border-radius: 14px;
  border: none;
  cursor: pointer;
  background: transparent;
  color: var(--text-primary);
  transition: background 0.15s;
  flex-shrink: 0;
}
.nav-logo:hover { background: var(--bg-soft); }

.nav-divider {
  width: 1px;
  height: 1.4rem;
  background: var(--border-subtle);
  margin: 0 8px;
  flex-shrink: 0;
}

.nav-link {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: 12px;
  font-size: 0.8rem;
  cursor: pointer;
  color: var(--text-muted);
  text-decoration: none;
  transition: background 0.15s ease, color 0.15s ease;
  flex-shrink: 0;
}
.nav-link:hover { background: var(--bg-soft); color: var(--text-primary); }
.nav-link--active {
  background: var(--accent-tint);
  color: var(--accent);
  font-weight: 600;
}

.nav-icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.1rem;
  height: 2.1rem;
  border-radius: 12px;
  border: none;
  cursor: pointer;
  background: transparent;
  color: var(--text-secondary);
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}
.nav-icon-btn:hover { background: var(--bg-soft); color: var(--text-primary); }
.nav-icon-btn--danger:hover {
  background: rgba(251, 113, 133, 0.15);
  color: var(--danger);
}

/* ── Mobile nav ────────────────────────────────────────────────────── */
.mobile-nav-link {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: 14px;
  font-size: 0.9rem;
  cursor: pointer;
  color: var(--text-secondary);
  text-decoration: none;
  transition: background 0.15s, color 0.15s;
}
.mobile-nav-link:hover { background: var(--bg-soft); color: var(--text-primary); }
.mobile-nav-link--active {
  background: var(--accent-tint);
  color: var(--accent);
  font-weight: 600;
}

/* ── Drawer ────────────────────────────────────────────────────────── */
.drawer {
  background: var(--bg-surface-strong);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border-right: 1px solid var(--border-soft);
  box-shadow: var(--shadow-strong);
}

/* Backdrop transition */
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* Drawer slide transition */
.slide-enter-active { transition: transform 0.26s cubic-bezier(0.4, 0, 0.2, 1); }
.slide-leave-active { transition: transform 0.2s ease; }
.slide-enter-from, .slide-leave-to { transform: translateX(-100%); }
</style>
