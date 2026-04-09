import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from '../axios.js'
import { onUnauthorized } from '../utils/authEvents'

export const useAuthStore = defineStore('auth', () => {
  const isAuthenticated = ref(false)
  const user = ref({
    userId: '',
    username: '',
    role: '',
    permissions: []
  })

  let checkPromise = null

  const isAdmin = computed(() => user.value.role === 'admin')
  const isOperator = computed(() => user.value.role === 'operator' || user.value.role === 'admin')

  const hasPermission = (permission) => {
    if (isAdmin.value) return true
    return user.value.permissions.includes(permission)
  }

  const resetState = () => {
    isAuthenticated.value = false
    user.value = { userId: '', username: '', role: '', permissions: [] }
    checkPromise = null
  }

  onUnauthorized(resetState)

  const checkAuth = async () => {
    if (checkPromise) return checkPromise
    checkPromise = (async () => {
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
        } catch {
          // Ignore error
        }
        isAuthenticated.value = false
        user.value = { userId: '', username: '', role: '', permissions: [] }
      } finally {
        checkPromise = null
      }
      return isAuthenticated.value
    })()
    return checkPromise
  }

  const login = async (payload) => {
    try {
      const res = await axios.post('/api/login', payload)
      const meRes = await axios.get('/api/me')
      user.value = {
        userId: meRes.data.user_id || res.data.user_id || 'admin',
        username: meRes.data.username || res.data.username || 'admin',
        role: meRes.data.role || res.data.role || 'admin',
        permissions: meRes.data.permissions || []
      }
      isAuthenticated.value = true
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
