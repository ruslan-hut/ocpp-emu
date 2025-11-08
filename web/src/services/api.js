import axios from 'axios'

// Use relative URL in production/Docker (goes through nginx proxy)
// Use full URL in development (direct to backend)
const API_BASE_URL = import.meta.env.VITE_API_URL || ''

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000, // Increased timeout to 30 seconds
})

// Request interceptor
api.interceptors.request.use(
  (config) => {
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
api.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

// Health API
export const healthAPI = {
  getHealth: () => api.get('/api/health'),
}

// Stations API
export const stationsAPI = {
  getAll: () => api.get('/api/stations'),
  getById: (id) => api.get(`/api/stations/${id}`),
  create: (station) => api.post('/api/stations', station),
  update: (id, station) => api.put(`/api/stations/${id}`, station),
  delete: (id) => api.delete(`/api/stations/${id}`),
  start: (id) => api.patch(`/api/stations/${id}/start`),
  stop: (id) => api.patch(`/api/stations/${id}/stop`),
}

// Messages API
export const messagesAPI = {
  getAll: (params) => api.get('/api/messages', { params }),
  getMessages: (params) => api.get('/api/messages', { params }),
  searchMessages: (params) => api.get('/api/messages/search', { params }),
  getStats: () => api.get('/api/messages/stats'),
  clear: () => api.delete('/api/messages'),
}

// Connections API
export const connectionsAPI = {
  getAll: () => api.get('/api/connections'),
}

export default api
