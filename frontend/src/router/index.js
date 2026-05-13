import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const Dashboard = () => import(/* webpackChunkName: "dashboard" */ '../views/Dashboard.vue')
const Login = () => import(/* webpackChunkName: "login" */ '../views/Login.vue')
const Control = () => import(/* webpackChunkName: "control" */ '../views/Control.vue')
const Config = () => import(/* webpackChunkName: "config" */ '../views/Config.vue')
const Logs = () => import(/* webpackChunkName: "logs" */ '../views/Logs.vue')
const Devices = () => import(/* webpackChunkName: "devices" */ '../views/Devices.vue')
const SystemInfo = () => import(/* webpackChunkName: "system" */ '../views/SystemInfo.vue')
const Users = () => import(/* webpackChunkName: "users" */ '../views/Users/Users.vue')
const Audit = () => import(/* webpackChunkName: "audit" */ '../views/Audit/Audit.vue')
const Layout = () => import(/* webpackChunkName: "layout" */ '../components/Layout.vue')

const prefetchMainRoutes = () => {
  Dashboard()
  Layout()
  Control()
  Config()
}

 const routes = [
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: { preload: true }
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
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth)

  if (!requiresAuth) {
    // Prefetch main app chunks while user is on the login screen
    if (to.name === 'Login') {
      setTimeout(prefetchMainRoutes, 300)
    }
    return true
  }

  const valid = await auth.checkAuth()
  if (!valid) {
    return { path: '/login', replace: true }
  }

  const requiredPermission = to.meta.permission
  if (requiredPermission && !auth.hasPermission(requiredPermission)) {
    return { path: '/', replace: true }
  }

  return true
})

router.onError((error) => {
  console.error('Router navigation error:', error)
  const message = String(error?.message || '')
  if (/failed to fetch dynamically imported module|importing a module script failed|loading chunk|chunkloaderror/i.test(message)) {
    window.location.reload()
  }
})

export default router
