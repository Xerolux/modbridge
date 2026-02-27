import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

export const useAuthStore = defineStore('auth', () => {
  const isAuthenticated = ref(false)
  const checking = ref(false)

  const checkAuth = async () => {
    if (checking.value) return isAuthenticated.value
    checking.value = true
    try {
      // Just check health or a protected endpoint to verify session
      await axios.get('/api/proxies')
      isAuthenticated.value = true
    } catch (e) {
      isAuthenticated.value = false
    } finally {
      checking.value = false
    }
    return isAuthenticated.value
  }

  const login = async (password) => {
    try {
      await axios.post('/api/login', { password })
      isAuthenticated.value = true
      return true
    } catch (e) {
      isAuthenticated.value = false
      return false
    }
  }

  const logout = async () => {
    // Clear session cookie by setting it to expire in the past
    document.cookie = 'session_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;'
    document.cookie = 'csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;'
    isAuthenticated.value = false
  }

  return { isAuthenticated, checkAuth, login, logout }
})
