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
  getConnectors: (id) => api.get(`/api/stations/${id}/connectors`),
  startCharging: (id, connectorId, idTag) =>
    api.post(`/api/stations/${id}/charge`, { connectorId, idTag }),
  stopCharging: (id, connectorId, reason) =>
    api.post(`/api/stations/${id}/stop-charge`, { connectorId, reason: reason || 'Local' }),
  sendMessage: (id, message) =>
    api.post(`/api/stations/${id}/send-message`, { message }),
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

// Scenarios API
export const scenariosAPI = {
  getAll: (params) => api.get('/api/scenarios', { params }),
  getById: (id) => api.get(`/api/scenarios/${id}`),
  create: (scenario) => api.post('/api/scenarios', scenario),
  update: (id, scenario) => api.put(`/api/scenarios/${id}`, scenario),
  delete: (id) => api.delete(`/api/scenarios/${id}`),
  execute: (id, stationId) => api.post(`/api/scenarios/${id}/execute`, { stationId }),
}

// Executions API
export const executionsAPI = {
  getAll: (params) => api.get('/api/executions', { params }),
  getById: (id) => api.get(`/api/executions/${id}`),
  pause: (id) => api.post(`/api/executions/${id}/pause`),
  resume: (id) => api.post(`/api/executions/${id}/resume`),
  stop: (id) => api.post(`/api/executions/${id}/stop`),
  delete: (id) => api.delete(`/api/executions/${id}`),
}

export default api
