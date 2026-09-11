import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { navigation } from './navigation'

const Dashboard = () => import(/* webpackChunkName: "dashboard" */ '../views/Dashboard.vue')
const Login = () => import(/* webpackChunkName: "login" */ '../views/Login.vue')
const AccountRecovery = () => import(/* webpackChunkName: "account-recovery" */ '../views/AccountRecovery.vue')
const Control = () => import(/* webpackChunkName: "control" */ '../views/Control.vue')
const Config = () => import(/* webpackChunkName: "config" */ '../views/Config.vue')
const Logs = () => import(/* webpackChunkName: "logs" */ '../views/Logs.vue')
const Devices = () => import(/* webpackChunkName: "devices" */ '../views/Devices.vue')
const SystemInfo = () => import(/* webpackChunkName: "system" */ '../views/SystemInfo.vue')
const Users = () => import(/* webpackChunkName: "users" */ '../views/Users/Users.vue')
const Audit = () => import(/* webpackChunkName: "audit" */ '../views/Audit/Audit.vue')
const ChangePassword = () => import(/* webpackChunkName: "change-password" */ '../views/ChangePassword.vue')
const Layout = () => import(/* webpackChunkName: "layout" */ '../components/Layout.vue')

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: { preload: true }
  },
  {
    path: '/account-recovery',
    name: 'AccountRecovery',
    component: AccountRecovery
  },
  {
    path: '/change-password',
    name: 'ChangePassword',
    component: ChangePassword,
    meta: { requiresAuth: true, allowDuringMustChange: true }
  },
  {
    path: '/',
    component: Layout,
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: Dashboard,
        meta: { preload: true }
      },
      {
        path: '/control',
        name: 'Control',
        component: Control,
        meta: { permission: 'proxy:view' }
      },
      {
        path: '/devices',
        name: 'Devices',
        component: Devices,
        meta: { permission: 'device:view' }
      },
      {
        path: '/config',
        name: 'Config',
        component: Config,
        meta: { permission: 'config:view' }
      },
      {
        path: '/logs',
        name: 'Logs',
        component: Logs,
        meta: { permission: 'logs:view' }
      },
      {
        path: '/system',
        name: 'System',
        component: SystemInfo,
        meta: { permission: 'system:view' }
      },
      {
        path: '/users',
        name: 'Users',
        component: Users,
        meta: { permission: 'user:view' }
      },
      {
        path: '/audit',
        name: 'Audit',
        component: Audit,
        meta: { permission: 'audit:view' }
      }
    ]
  },
  // Unknown routes previously rendered a completely blank page (no view at
  // all). Redirect them home instead so a stale bookmark or typo never looks
  // like a broken app.
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    return savedPosition || { top: 0 }
  }
})

router.beforeEach(() => {
  navigation.loading = true
  navigation.failedPath = null
})

router.afterEach(() => {
  navigation.loading = false
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth)

  if (!requiresAuth) {
    return true
  }

  const valid = await auth.checkAuth()
  if (!valid) {
    return { path: '/login', replace: true }
  }

  // Force password change on first login: only the dedicated change-password
  // route is reachable until the flag is cleared.
  if (auth.mustChangePassword && !to.meta.allowDuringMustChange) {
    return { path: '/change-password', replace: true }
  }

  const requiredPermission = to.meta.permission
  if (requiredPermission && !auth.hasPermission(requiredPermission)) {
    return { path: '/', replace: true }
  }

  return true
})

router.onError((error, to) => {
  console.error('Router navigation error:', error)
  navigation.loading = false
  navigation.failedPath = to?.fullPath || '/'
})

export default router
