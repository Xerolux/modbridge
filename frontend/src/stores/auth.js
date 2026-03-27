import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from '../axios.js'

export const useAuthStore = defineStore('auth', () => {
  const isAuthenticated = ref(false)
  const checking = ref(false)
  const user = ref({
    userId: '',
    username: '',
    role: '',
    permissions: []
  })

  const isAdmin = computed(() => user.value.role === 'admin')
  const isOperator = computed(() => user.value.role === 'operator' || user.value.role === 'admin')
  const isViewer = computed(() => true)

  const hasPermission = (permission) => {
    if (isAdmin.value) return true
    return user.value.permissions.includes(permission)
  }

  const checkAuth = async () => {
    if (checking.value) return isAuthenticated.value
    checking.value = true
    try {
      const res = await axios.get('/api/me')
      user.value = {
        userId: res.data.user_id || '',
        username: res.data.username || '',
        role: res.data.role || 'admin',
        permissions: res.data.permissions || []
      }
      isAuthenticated.value = true
    } catch {
      try {
        await axios.get('/api/status')
        isAuthenticated.value = true
        user.value = { userId: 'admin', username: 'admin', role: 'admin', permissions: [] }
      } catch {
        isAuthenticated.value = false
        user.value = { userId: '', username: '', role: '', permissions: [] }
      }
    } finally {
      checking.value = false
    }
    return isAuthenticated.value
  }

  const login = async (payload) => {
    try {
      const res = await axios.post('/api/login', payload)
      user.value = {
        userId: res.data.user_id || 'admin',
        username: res.data.username || 'admin',
        role: res.data.role || 'admin',
        permissions: []
      }
      isAuthenticated.value = true
      if (res.data.role) {
        const meRes = await axios.get('/api/me')
        if (meRes.data.permissions) {
          user.value.permissions = meRes.data.permissions
        }
      }
      return { success: true }
    } catch (e) {
      isAuthenticated.value = false
      const message = e.response?.data?.trim() || e.message || 'Login failed'
      return { success: false, message }
    }
  }

  const logout = async () => {
    document.cookie = 'session_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;'
    document.cookie = 'csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;'
    isAuthenticated.value = false
    user.value = { userId: '', username: '', role: '', permissions: [] }
  }

  return {
    isAuthenticated,
    user,
    isAdmin,
    isOperator,
    hasPermission,
    checkAuth,
    login,
    logout
  }
})
