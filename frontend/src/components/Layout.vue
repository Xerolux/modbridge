<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { RouterLink, useRouter, useRoute } from 'vue-router';
import { useAuthStore } from "../stores/auth";
import { useAppStore } from "../stores/appStore";
import { useI18n } from 'vue-i18n';
import LanguageSelector from './LanguageSelector.vue';
import ThemeSettings from './ThemeSettings.vue';

const { t } = useI18n();

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();
const app = useAppStore();

const mobileMenuOpen = ref(false);

// Navigation is structured into collapsible groups (SLZB-style) so the top bar
// stays compact even with many destinations. Each group toggles open/closed;
// the first group defaults open.
const item = (labelKey, icon, path, permission) => ({ labelKey, icon, path, permission });
const navGroups = [
  {
    labelKey: 'nav.groupProxies',
    icon: 'pi pi-sitemap',
    open: ref(true),
    items: [
      item('nav.dashboard', 'pi pi-home',      '/',        null),
      item('nav.control',   'pi pi-sliders-h', '/control', 'proxy:view'),
    ],
  },
  {
    labelKey: 'nav.devices',
    icon: 'pi pi-desktop',
    flat: true, // single-item group renders as a direct link
    items: [ item('nav.devices', 'pi pi-desktop', '/devices', 'device:view') ],
  },
  {
    labelKey: 'nav.groupSecurity',
    icon: 'pi pi-shield',
    open: ref(false),
    items: [
      item('nav.users', 'pi pi-users',   '/users', 'user:view'),
      item('nav.audit', 'pi pi-history', '/audit', 'audit:view'),
    ],
  },
  {
    labelKey: 'nav.groupSystem',
    icon: 'pi pi-cog',
    open: ref(false),
    items: [
      item('nav.settings', 'pi pi-cog',          '/config', 'config:view'),
      item('nav.system',   'pi pi-info-circle',  '/system', 'system:view'),
      item('nav.logs',     'pi pi-list',         '/logs',   'logs:view'),
    ],
  },
];

// Flatten for permission filtering + active-group detection
const groupVisible = (g) => g.items.some(i => !i.permission || auth.isAdmin || auth.hasPermission(i.permission));
const visibleGroups = computed(() => navGroups.filter(groupVisible));
const itemVisible = (i) => !i.permission || auth.isAdmin || auth.hasPermission(i.permission);

const proxyCount = computed(() => app.proxies?.length ?? 0);

const toggleGroup = (g) => { g.open.value = !g.open.value; };

const logout = async () => {
  await auth.logout();
  router.push('/login');
};

const isActiveRoute = (path) => {
  if (path === '/') return route.path === '/';
  return route.path.startsWith(path);
};

// Auto-expand a group when the active route is one of its children, so the
// current location is always visible without the user having to hunt for it.
watch(() => route.path, (p) => {
  for (const g of navGroups) {
    if (g.flat || !g.open) continue;
    if (g.items.some(i => isActiveRoute(i.path))) g.open.value = true;
  }
}, { immediate: true });

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

        <!-- Status indicator (proxy count) -->
        <div class="hidden lg:flex items-center gap-1.5 px-2.5 py-1 rounded-xl text-xs text-[var(--text-muted)]">
          <span class="status-pill" :class="proxyCount > 0 ? 'status-pill--on' : 'status-pill--off'"></span>
          <span>{{ proxyCount }} {{ t('nav.proxiesCount') }}</span>
        </div>

        <!-- Desktop nav: collapsible groups (SLZB-style) -->
        <div class="hidden md:flex items-center gap-0.5 flex-1 overflow-x-auto">
          <template v-for="g in visibleGroups" :key="g.labelKey">
            <!-- Single-item group renders as a direct link -->
            <RouterLink
              v-if="g.flat"
              v-for="i in g.items.filter(itemVisible)"
              :key="i.path"
              :to="i.path"
              class="nav-link"
              :class="{ 'nav-link--active': isActiveRoute(i.path) }"
              active-class=""
              exact-active-class=""
              :aria-current="isActiveRoute(i.path) ? 'page' : undefined"
            >
              <i :class="i.icon" class="text-sm shrink-0"></i>
              <span class="whitespace-nowrap">{{ t(g.labelKey) }}</span>
            </RouterLink>

            <!-- Multi-item group: hover dropdown -->
            <div v-else class="nav-group">
              <button
                type="button"
                class="nav-link"
                :class="{ 'nav-link--active': g.items.some(i => isActiveRoute(i.path)) }"
                @click="toggleGroup(g)"
              >
                <i :class="g.icon" class="text-sm shrink-0"></i>
                <span class="whitespace-nowrap">{{ t(g.labelKey) }}</span>
                <i class="pi pi-chevron-down text-[0.65rem] shrink-0 nav-group-caret" :class="{ 'nav-group-caret--open': g.open.value }"></i>
              </button>
              <Transition name="dropdown">
                <div v-if="g.open.value" class="nav-group-menu">
                  <RouterLink
                    v-for="i in g.items.filter(itemVisible)"
                    :key="i.path"
                    :to="i.path"
                    class="nav-group-item"
                    :class="{ 'nav-group-item--active': isActiveRoute(i.path) }"
                    active-class=""
                    exact-active-class=""
                    @click="g.open.value = false"
                  >
                    <i :class="i.icon" class="text-sm shrink-0"></i>
                    <span class="whitespace-nowrap">{{ t(i.labelKey) }}</span>
                  </RouterLink>
                </div>
              </Transition>
            </div>
          </template>
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
          <template v-for="g in visibleGroups" :key="g.labelKey">
            <template v-for="i in g.items.filter(itemVisible)" :key="i.path">
              <RouterLink
                :to="i.path"
                class="mobile-nav-link"
                :class="{ 'mobile-nav-link--active': isActiveRoute(i.path) }"
                active-class=""
                exact-active-class=""
                @click="mobileMenuOpen = false"
              >
                <i :class="i.icon" class="text-base w-5 text-center shrink-0"></i>
                <span>{{ t(g.flat ? g.labelKey : i.labelKey) }}</span>
              </RouterLink>
            </template>
          </template>
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
      <router-view />
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

/* ── Collapsible nav groups ────────────────────────────────────────── */
.nav-group {
  position: relative;
}
.nav-group-caret {
  transition: transform 0.15s ease;
  opacity: 0.7;
}
.nav-group-caret--open { transform: rotate(180deg); }

.nav-group-menu {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  min-width: 200px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px;
  border-radius: 14px;
  background: var(--bg-surface-strong);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border: 1px solid var(--border-soft);
  box-shadow: var(--shadow-strong);
  z-index: 60;
}
.nav-group-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 10px;
  font-size: 0.8rem;
  color: var(--text-secondary);
  text-decoration: none;
  transition: background 0.15s, color 0.15s;
}
.nav-group-item:hover { background: var(--bg-soft); color: var(--text-primary); }
.nav-group-item--active { background: var(--accent-tint); color: var(--accent); font-weight: 600; }

.dropdown-enter-active { transition: opacity 0.15s ease, transform 0.15s ease; }
.dropdown-leave-active { transition: opacity 0.12s ease, transform 0.12s ease; }
.dropdown-enter-from, .dropdown-leave-to { opacity: 0; transform: translateY(-4px); }

/* ── Status pill (proxy count) ─────────────────────────────────────── */
.status-pill {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  display: inline-block;
}
.status-pill--on { background: #10b981; box-shadow: 0 0 0 3px rgba(16,185,129,0.18); }
.status-pill--off { background: var(--text-muted); opacity: 0.5; }

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
