import { useState, useEffect, useRef } from 'react'
import { healthAPI, stationsAPI, messagesAPI } from '../services/api'
import './Dashboard.css'

function Dashboard() {
  const [health, setHealth] = useState(null)
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const isInitialLoad = useRef(true)

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 5000) // Refresh every 5 seconds
    return () => clearInterval(interval)
  }, [])

  const fetchData = async () => {
    if (isInitialLoad.current) {
      setLoading(true)
    }

    const [healthRes, stationsRes, messagesRes] = await Promise.allSettled([
      healthAPI.getHealth(),
      stationsAPI.getAll(),
      messagesAPI.getStats(),
    ])

    const errors = []

    if (healthRes.status === 'fulfilled') {
      setHealth(healthRes.value.data)
    } else {
      console.warn('Failed to load health', healthRes.reason)
      errors.push(healthRes.reason?.message || 'Failed to load health')
    }

    if (stationsRes.status === 'fulfilled') {
      setStats((prev) => ({
        ...(prev || {}),
        stations: stationsRes.value.data,
      }))
    } else {
      console.warn('Failed to load stations', stationsRes.reason)
      errors.push(stationsRes.reason?.message || 'Failed to load stations')
    }

    if (messagesRes.status === 'fulfilled') {
      setStats((prev) => ({
        ...(prev || {}),
        messages: messagesRes.value.data,
      }))
    } else {
      console.warn('Failed to load message stats', messagesRes.reason)
      errors.push(messagesRes.reason?.message || 'Failed to load message stats')
    }

    setError(errors.length ? errors.join('; ') : null)
    setLoading(false)
    isInitialLoad.current = false
  }

  const stationStats = stats?.stations ?? {}
  const messageStats = stats?.messages ?? {}

  if (loading) {
    return <div className="loading">Loading dashboard...</div>
  }

  if (error) {
    return <div className="error">Error loading dashboard: {error}</div>
  }

  return (
    <div className="dashboard">
      <h2>Dashboard</h2>

      <div className="stats-grid">
        <div className="stat-card">
          <h3>System Health</h3>
          <div className="stat-value">{health?.status || 'Unknown'}</div>
          <div className="stat-label">Status</div>
          <div className="stat-detail">Database: {health?.database || 'Unknown'}</div>
        </div>

        <div className="stat-card">
          <h3>Stations</h3>
          <div className="stat-value">{stationStats.count ?? 0}</div>
          <div className="stat-label">Total Stations</div>
          <div className="stat-detail">
            Connected: {health?.stations?.connected || 0}
          </div>
        </div>

        <div className="stat-card">
          <h3>Messages</h3>
          <div className="stat-value">{messageStats.total ?? 0}</div>
          <div className="stat-label">Total Messages</div>
          <div className="stat-detail">
            Sent: {messageStats.sent ?? 0} | Received: {messageStats.received ?? 0}
          </div>
        </div>

        <div className="stat-card">
          <h3>Message Buffer</h3>
          <div className="stat-value">{messageStats.buffered ?? 0}</div>
          <div className="stat-label">Buffered Messages</div>
          <div className="stat-detail">
            Dropped: {messageStats.dropped ?? 0}
          </div>
        </div>
      </div>

      <div className="info-section">
        <h3>Quick Info</h3>
        <div className="info-grid">
          <div className="info-item">
            <strong>Version:</strong> {health?.version || 'Unknown'}
          </div>
          <div className="info-item">
            <strong>Charging Stations:</strong> {health?.stations?.charging || 0}
          </div>
          <div className="info-item">
            <strong>Available Stations:</strong> {health?.stations?.available || 0}
          </div>
          <div className="info-item">
            <strong>Faulted Stations:</strong> {health?.stations?.faulted || 0}
          </div>
        </div>
      </div>
    </div>
  )
}

export default Dashboard
