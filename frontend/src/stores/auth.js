import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from '../axios.js'
import { onUnauthorized } from '../utils/authEvents'

export const useAuthStore = defineStore('auth', () => {
  const AUTH_CHECK_CACHE_MS = 5000
  const isAuthenticated = ref(false)
  const user = ref({
    userId: '',
    username: '',
    role: '',
    permissions: []
  })

  let checkPromise = null
  let lastAuthCheckAt = 0

  const isAdmin = computed(() => user.value.role === 'admin')
  const isOperator = computed(() => user.value.role === 'techniker' || user.value.role === 'operator' || user.value.role === 'admin')

  const hasPermission = (permission) => {
    if (isAdmin.value) return true
    return user.value.permissions.includes(permission)
  }

  const resetState = () => {
    isAuthenticated.value = false
    user.value = { userId: '', username: '', role: '', permissions: [] }
    checkPromise = null
    lastAuthCheckAt = 0
  }

  onUnauthorized(resetState)

  const checkAuth = async () => {
    if (isAuthenticated.value && Date.now() - lastAuthCheckAt < AUTH_CHECK_CACHE_MS) {
      return true
    }

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
        lastAuthCheckAt = Date.now()
      } catch {
        try {
          await axios.get('/api/status')
        } catch {
          // Ignore error
        }
        isAuthenticated.value = false
        user.value = { userId: '', username: '', role: '', permissions: [] }
        lastAuthCheckAt = 0
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
      lastAuthCheckAt = Date.now()
      return { success: true }
    } catch (e) {
      isAuthenticated.value = false
      lastAuthCheckAt = 0
      const message = e.response?.data?.trim() || e.message || 'Login failed'
      return { success: false, message }
    }
  }

  const logout = async () => {
    const secureFlag = window.location.protocol === 'https:' ? '; Secure' : ''
    document.cookie = `session_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/; SameSite=Lax${secureFlag}`
    document.cookie = `csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/; SameSite=Lax${secureFlag}`
    isAuthenticated.value = false
    user.value = { userId: '', username: '', role: '', permissions: [] }
    lastAuthCheckAt = 0
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
