<script setup>
  import { ref, computed, onMounted, onUnmounted } from "vue";
  import { useRouter, useRoute } from 'vue-router'
  import { useAuthStore } from "../stores/auth";
  import { useAppStore } from "../stores/appStore";
  import Menubar from 'primevue/menubar';
  import Button from 'primevue/button';
  import Sidebar from 'primevue/sidebar';
  import InputSwitch from 'primevue/inputswitch';
  import LanguageSelector from './LanguageSelector.vue';
  import { debounce } from '../utils/helpers';

  const router = useRouter();
  const route = useRoute();
  const auth = useAuthStore();
  const appStore = useAppStore();

  const mobileMenuVisible = ref(false);
  const isMobile = ref(false);

  const navigate = (path) => {
      router.push(path);
      mobileMenuVisible.value = false;
  };

  const allItems = [
      {
          label: 'Dashboard',
          icon: 'pi pi-home',
          path: '/',
          permission: null,
          command: () => navigate('/')
      },
      {
          label: 'Control',
          icon: 'pi pi-sliders-h',
          path: '/control',
          permission: 'proxy:view',
          command: () => navigate('/control')
      },
      {
          label: 'Devices',
          icon: 'pi pi-desktop',
          path: '/devices',
          permission: 'device:view',
          command: () => navigate('/devices')
      },
      {
          label: 'Logs',
          icon: 'pi pi-list',
          path: '/logs',
          permission: 'logs:view',
          command: () => navigate('/logs')
      },
      {
          label: 'System',
          icon: 'pi pi-info-circle',
          path: '/system',
          permission: 'system:view',
          command: () => navigate('/system')
      },
      {
          label: 'Settings',
          icon: 'pi pi-cog',
          path: '/config',
          permission: 'config:view',
          command: () => navigate('/config')
      },
      {
          label: 'Users',
          icon: 'pi pi-users',
          path: '/users',
          permission: 'user:view',
          command: () => navigate('/users')
      },
      {
          label: 'Audit Log',
          icon: 'pi pi-history',
          path: '/audit',
          permission: 'audit:view',
          command: () => navigate('/audit')
      }
  ];

  const items = computed(() => {
      return allItems.filter(item => {
          if (!item.permission) return true;
          if (auth.isAdmin) return true;
          return auth.hasPermission(item.permission);
      });
  });

  const logout = async () => {
      await auth.logout();
      router.push('/login');
  }

  const isActiveRoute = (path) => {
      if (path === '/') return route.path === '/';
      return route.path.startsWith(path);
  };

  const checkMobile = debounce(() => {
      isMobile.value = window.innerWidth < 768;
  }, 150);

  onMounted(() => {
      checkMobile();
      window.addEventListener('resize', checkMobile);
  });

  onUnmounted(() => {
      window.removeEventListener('resize', checkMobile);
  });
</script>

<template>
    <div class="min-h-screen flex flex-col bg-transparent">
        <header class="px-4 py-3 z-10">
            <Menubar :model="isMobile ? [] : items" class="glass-card border border-white/10 rounded-2xl shadow-lg !bg-surface-800/40">
                 <template #start>
                   <div class="flex items-center gap-4 pl-2">
                       <Button v-if="isMobile" icon="pi pi-bars" text rounded @click="mobileMenuVisible = true" class="text-white hover:bg-white/10" />
                       <div class="flex items-center gap-2 cursor-pointer" @click="navigate('/')">
                           <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-500 to-blue-500 flex items-center justify-center shadow-lg shadow-primary-500/20">
                               <i class="pi pi-bolt text-white text-sm"></i>
                           </div>
                           <span class="text-xl font-bold tracking-tight text-white hidden sm:block">ModBridge</span>
                       </div>
                   </div>
                 </template>
                 <template #item="{ item, props }">
                     <a v-ripple class="flex items-center gap-2 px-4 py-2.5 hover:bg-white/10 rounded-xl cursor-pointer text-surface-200 transition-colors mx-1" :class="{'bg-white/10 text-white font-medium': isActiveRoute(item.path)}" v-bind="props.action">
                         <i :class="item.icon" class="text-lg"></i>
                         <span class="text-sm">{{ item.label }}</span>
                     </a>
                 </template>
                 <template #end>
                     <div class="flex items-center gap-3 pr-2">
                         <LanguageSelector class="hidden sm:flex" />
                         <div class="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-xl bg-surface-900/50 border border-white/5">
                             <i :class="appStore.darkMode ? 'pi pi-moon' : 'pi pi-sun'" class="text-surface-300 text-sm"></i>
                             <InputSwitch :modelValue="appStore.darkMode" @update:modelValue="(val) => appStore.toggleDarkMode(val)" class="scale-75" />
                         </div>
                         <div v-if="auth.user.username" class="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-xl bg-surface-900/50 border border-white/5">
                             <i class="pi pi-user text-surface-300 text-sm"></i>
                             <span class="text-surface-200 text-sm">{{ auth.user.username }}</span>
                             <span class="text-xs text-surface-400">({{ auth.user.role }})</span>
                         </div>
                         <Button icon="pi pi-power-off" severity="danger" rounded text @click="logout" class="hidden sm:flex hover:bg-red-500/20 w-10 h-10" />
                     </div>
                 </template>
            </Menubar>
        </header>

        <Sidebar v-model:visible="mobileMenuVisible" :baseZIndex="10000" class="glass-sidebar">
            <template #header>
                <div class="flex items-center gap-3 px-2">
                     <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-500 to-blue-500 flex items-center justify-center shadow-lg">
                         <i class="pi pi-bolt text-white text-sm"></i>
                     </div>
                     <span class="text-xl font-bold tracking-tight text-white">ModBridge</span>
                </div>
            </template>
            <div class="flex flex-col gap-2 h-full py-4">
                <div v-for="item in items" :key="item.label">
                    <Button
                        @click="item.command"
                        :label="item.label"
                        :icon="item.icon"
                        text
                        :class="['w-full text-left rounded-xl py-3 px-4', isActiveRoute(item.path) ? 'bg-primary-500/20 text-primary-300 font-medium border border-primary-500/20' : 'text-surface-200 hover:bg-white/5']"
                    />
                </div>

                <div class="mt-auto border-t border-white/10 pt-6 flex flex-col gap-4">
                    <div v-if="auth.user.username" class="flex items-center gap-2 px-4 py-2 rounded-xl bg-surface-900/50 border border-white/5">
                        <i class="pi pi-user text-surface-400 text-sm"></i>
                        <span class="text-surface-200 text-sm">{{ auth.user.username }}</span>
                        <span class="text-xs text-surface-400">({{ auth.user.role }})</span>
                    </div>
                     <div class="flex items-center justify-between px-4 py-3 rounded-xl bg-surface-900/50 border border-white/5">
                        <span class="text-surface-200 font-medium text-sm">Theme</span>
                        <div class="flex items-center gap-3">
                            <i :class="appStore.darkMode ? 'pi pi-moon text-surface-400' : 'pi pi-sun text-surface-400'"></i>
                            <InputSwitch :modelValue="appStore.darkMode" @update:modelValue="(val) => appStore.toggleDarkMode(val)" class="scale-90" />
                        </div>
                     </div>
                     <LanguageSelector class="w-full px-2" />
                    <Button
                        @click="logout"
                        label="Logout"
                        icon="pi pi-power-off"
                        severity="danger"
                        text
                        class="w-full text-left rounded-xl py-3 px-4 hover:bg-red-500/10"
                    />
                </div>
            </div>
        </Sidebar>

        <main class="flex-grow text-white w-full max-w-7xl mx-auto p-4 pt-0">
             <router-view></router-view>
        </main>
    </div>
</template>

<style scoped>
:deep(.p-menubar) {
    padding: 0.5rem;
    backdrop-filter: blur(24px) !important;
    -webkit-backdrop-filter: blur(24px) !important;
}
:deep(.p-menubar .p-menubar-button) {
    color: white;
}
:deep(.p-sidebar) {
    background: rgba(17, 24, 39, 0.7) !important;
    backdrop-filter: blur(24px) !important;
    -webkit-backdrop-filter: blur(24px) !important;
    border-right: 1px solid rgba(255, 255, 255, 0.1);
    color: white;
}
:deep(.p-sidebar-header) {
    padding: 1.5rem 1.5rem 0.5rem;
}
</style>
