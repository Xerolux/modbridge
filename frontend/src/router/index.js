import { createRouter, createWebHashHistory } from 'vue-router'
 import { useAuthStore } from '../stores/auth'

 const Dashboard = () => import('../views/Dashboard.vue')
 const Login = () => import('../views/Login.vue')
 const Control = () => import('../views/Control.vue')
 const Config = () => import('../views/Config.vue')
 const Logs = () => import('../views/Logs.vue')
 const Devices = () => import('../views/Devices.vue')
 const SystemInfo = () => import('../views/SystemInfo.vue')
 const Users = () => import('../views/Users/Users.vue')
 const Audit = () => import('../views/Audit/Audit.vue')
 const Layout = () => import('../components/Layout.vue')

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

  if (!requiresAuth) return true

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
