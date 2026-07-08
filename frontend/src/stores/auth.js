import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from '../axios.js'
import { onUnauthorized } from '../utils/authEvents'

export const useAuthStore = defineStore('auth', () => {
  const isAuthenticated = ref(false)
  const mustChangePassword = ref(false)
  const user = ref({
    userId: '',
    username: '',
    role: '',
    permissions: []
  })

  let checkPromise = null
  // Auth is verified exactly once per page load (a full reload resets this).
  // After that, trust the in-memory state and react to 401 via onUnauthorized.
  // This avoids the bug where a short time-based cache returned "authenticated"
  // while the server session had already ended.
  let verifiedThisLoad = false

  const isAdmin = computed(() => user.value.role === 'admin')
  const isOperator = computed(() => user.value.role === 'techniker' || user.value.role === 'operator' || user.value.role === 'admin')

  const hasPermission = (permission) => {
    if (isAdmin.value) return true
    return user.value.permissions.includes(permission)
  }

  const resetState = () => {
    isAuthenticated.value = false
    mustChangePassword.value = false
    user.value = { userId: '', username: '', role: '', permissions: [] }
    checkPromise = null
    verifiedThisLoad = false
  }

  onUnauthorized(resetState)

  const checkAuth = async () => {
    if (isAuthenticated.value && verifiedThisLoad) {
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
        mustChangePassword.value = !!res.data.must_change_password
        isAuthenticated.value = true
        verifiedThisLoad = true
      } catch {
        // The response interceptor already emits onUnauthorized on 401, which
        // calls resetState(). Nothing else to do here.
        isAuthenticated.value = false
        user.value = { userId: '', username: '', role: '', permissions: [] }
        mustChangePassword.value = false
        verifiedThisLoad = false
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
      mustChangePassword.value = !!res.data.force_password_change
      const meRes = await axios.get('/api/me')
      user.value = {
        userId: meRes.data.user_id || res.data.user_id || 'admin',
        username: meRes.data.username || res.data.username || 'admin',
        role: meRes.data.role || res.data.role || 'admin',
        permissions: meRes.data.permissions || []
      }
      mustChangePassword.value = mustChangePassword.value || !!meRes.data.must_change_password
      isAuthenticated.value = true
      verifiedThisLoad = true
      return { success: true, mustChangePassword: mustChangePassword.value }
    } catch (e) {
      resetState()
      const message = e.response?.data?.trim() || e.message || 'Login failed'
      return { success: false, message }
    }
  }

  const logout = async () => {
    // Ask the server to invalidate the session so it ends immediately (instead
    // of lingering until its natural expiry). Cookie clearing is the fallback.
    try {
      await axios.post('/api/logout')
    } catch {
      // ignore — we clear cookies client-side regardless
    }
    resetState()
    const secureFlag = window.location.protocol === 'https:' ? '; Secure' : ''
    document.cookie = `session_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/; SameSite=Lax${secureFlag}`
    document.cookie = `csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/; SameSite=Lax${secureFlag}`
  }

  return {
    isAuthenticated,
    mustChangePassword,
    user,
    isAdmin,
    isOperator,
    hasPermission,
    checkAuth,
    login,
    logout
  }
})
