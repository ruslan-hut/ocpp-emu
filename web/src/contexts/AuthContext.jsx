import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { authAPI } from '../services/auth'

const AUTH_STORAGE_KEY = 'ocpp-emu-auth'

const AuthContext = createContext(null)

/**
 * Get initial auth state from localStorage
 */
function getInitialAuth() {
  if (typeof window === 'undefined') return null

  try {
    const stored = localStorage.getItem(AUTH_STORAGE_KEY)
    if (stored) {
      const parsed = JSON.parse(stored)
      // Check if token is expired
      if (parsed.expiresAt && new Date(parsed.expiresAt) > new Date()) {
        return parsed
      }
      // Token expired, clear storage
      localStorage.removeItem(AUTH_STORAGE_KEY)
    }
  } catch (e) {
    console.warn('Failed to read auth from localStorage:', e)
    localStorage.removeItem(AUTH_STORAGE_KEY)
  }

  return null
}

export function AuthProvider({ children }) {
  const [auth, setAuth] = useState(getInitialAuth)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  // Check auth status on mount
  useEffect(() => {
    const checkAuth = async () => {
      const storedAuth = getInitialAuth()
      if (storedAuth) {
        try {
          // Verify token is still valid
          await authAPI.me(storedAuth.token)
          setAuth(storedAuth)
        } catch (e) {
          // Token is invalid, clear auth
          console.warn('Token validation failed:', e)
          localStorage.removeItem(AUTH_STORAGE_KEY)
          setAuth(null)
        }
      }
      setLoading(false)
    }

    checkAuth()
  }, [])

  const login = useCallback(async (username, password) => {
    setError(null)
    try {
      const response = await authAPI.login(username, password)
      const { token, expiresAt, user } = response.data

      const authData = {
        token,
        expiresAt,
        user
      }

      setAuth(authData)
      localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(authData))

      return { success: true }
    } catch (e) {
      const message = e.response?.data?.error || 'Login failed'
      setError(message)
      return { success: false, error: message }
    }
  }, [])

  const logout = useCallback(() => {
    // Optionally notify server
    if (auth?.token) {
      authAPI.logout(auth.token).catch(() => {})
    }

    setAuth(null)
    setError(null)
    localStorage.removeItem(AUTH_STORAGE_KEY)
  }, [auth])

  const clearError = useCallback(() => {
    setError(null)
  }, [])

  const value = {
    user: auth?.user || null,
    token: auth?.token || null,
    loading,
    error,
    login,
    logout,
    clearError,
    isAuthenticated: !!auth?.token,
    isAdmin: auth?.user?.role === 'admin',
    isViewer: auth?.user?.role === 'viewer',
  }

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

export default AuthContext
