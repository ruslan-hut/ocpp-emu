import { useState, useEffect } from 'react'
import { healthAPI, stationsAPI, messagesAPI } from '../services/api'
import './Dashboard.css'

function Dashboard() {
  const [health, setHealth] = useState(null)
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 5000) // Refresh every 5 seconds
    return () => clearInterval(interval)
  }, [])

  const fetchData = async () => {
    try {
      const [healthRes, stationsRes, messagesRes] = await Promise.all([
        healthAPI.getHealth(),
        stationsAPI.getAll(),
        messagesAPI.getStats(),
      ])

      setHealth(healthRes.data)
      setStats({
        stations: stationsRes.data,
        messages: messagesRes.data,
      })
      setError(null)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

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
          <div className="stat-value">{stats?.stations.count || 0}</div>
          <div className="stat-label">Total Stations</div>
          <div className="stat-detail">
            Connected: {health?.stations?.connected || 0}
          </div>
        </div>

        <div className="stat-card">
          <h3>Messages</h3>
          <div className="stat-value">{stats?.messages.total || 0}</div>
          <div className="stat-label">Total Messages</div>
          <div className="stat-detail">
            Sent: {stats?.messages.sent || 0} | Received: {stats?.messages.received || 0}
          </div>
        </div>

        <div className="stat-card">
          <h3>Message Buffer</h3>
          <div className="stat-value">{stats?.messages.buffered || 0}</div>
          <div className="stat-label">Buffered Messages</div>
          <div className="stat-detail">
            Dropped: {stats?.messages.dropped || 0}
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
