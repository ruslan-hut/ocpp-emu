import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_URL || ''

// Separate axios instance for auth (no interceptors that might cause redirects)
const authAxios = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 10000,
})

export const authAPI = {
  /**
   * Login with username and password
   * @param {string} username
   * @param {string} password
   * @returns {Promise<{data: {token: string, expiresAt: string, user: {username: string, role: string}}}>}
   */
  login: (username, password) =>
    authAxios.post('/api/auth/login', { username, password }),

  /**
   * Logout (optional server notification)
   * @param {string} token
   * @returns {Promise}
   */
  logout: (token) =>
    authAxios.post('/api/auth/logout', {}, {
      headers: { Authorization: `Bearer ${token}` }
    }),

  /**
   * Get current user info
   * @param {string} token
   * @returns {Promise<{data: {username: string, role: string}}>}
   */
  me: (token) =>
    authAxios.get('/api/auth/me', {
      headers: { Authorization: `Bearer ${token}` }
    }),

  /**
   * Refresh token
   * @param {string} token
   * @returns {Promise<{data: {token: string, expiresAt: string, user: {username: string, role: string}}}>}
   */
  refresh: (token) =>
    authAxios.post('/api/auth/refresh', {}, {
      headers: { Authorization: `Bearer ${token}` }
    }),
}

export default authAPI
