import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { healthAPI, stationsAPI, messagesAPI } from '../services/api'
import './Dashboard.css'

function Dashboard() {
  const navigate = useNavigate()
  const [health, setHealth] = useState(null)
  const [stats, setStats] = useState(null)
  const [stations, setStations] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const isInitialLoad = useRef(true)
  const [sortBy, setSortBy] = useState(() => localStorage.getItem('dashboardSortBy') || null)
  const [sortOrder, setSortOrder] = useState(() => localStorage.getItem('dashboardSortOrder') || 'asc')

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
      const stationData = stationsRes.value.data
      setStations(stationData.stations || [])
      setStats((prev) => ({
        ...(prev || {}),
        stations: stationData,
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

  const getStatusClass = (status) => {
    switch (status) {
      case 'connected': return 'status--connected'
      case 'connecting': return 'status--connecting'
      case 'disconnected':
      case 'not_connected': return 'status--disconnected'
      default: return 'status--unknown'
    }
  }

  const handleSort = (field) => {
    if (sortBy === field) {
      // Toggle through: asc -> desc -> none
      if (sortOrder === 'asc') {
        setSortOrder('desc')
        localStorage.setItem('dashboardSortOrder', 'desc')
      } else {
        setSortBy(null)
        setSortOrder('asc')
        localStorage.removeItem('dashboardSortBy')
        localStorage.setItem('dashboardSortOrder', 'asc')
      }
    } else {
      setSortBy(field)
      setSortOrder('asc')
      localStorage.setItem('dashboardSortBy', field)
      localStorage.setItem('dashboardSortOrder', 'asc')
    }
  }

  const getSortedStations = () => {
    if (!sortBy) return stations

    return [...stations].sort((a, b) => {
      let aValue = a[sortBy]
      let bValue = b[sortBy]

      // Handle case-insensitive string comparison for stationId
      if (typeof aValue === 'string') aValue = aValue.toLowerCase()
      if (typeof bValue === 'string') bValue = bValue.toLowerCase()

      if (aValue < bValue) return sortOrder === 'asc' ? -1 : 1
      if (aValue > bValue) return sortOrder === 'asc' ? 1 : -1
      return 0
    })
  }

  if (loading) {
    return <div className="loading">Loading dashboard...</div>
  }

  if (error) {
    return <div className="error">Error loading dashboard: {error}</div>
  }

  const connectedCount = stations.filter(s => s.runtimeState?.connectionStatus === 'connected').length
  const chargingCount = health?.stations?.charging || 0
  const availableCount = health?.stations?.available || 0
  const faultedCount = health?.stations?.faulted || 0

  return (
    <div className="dashboard dashboard--desktop">
      {/* Header */}
      <div className="dashboard-header">
        <h2>Dashboard</h2>
        <div className="dashboard-header__status">
          <span className={`system-status system-status--${health?.status?.toLowerCase() || 'unknown'}`}>
            {health?.status || 'Unknown'}
          </span>
          <span className="last-update">Auto-refresh: 5s</span>
        </div>
      </div>

      {/* Stats Row - Compact Cards */}
      <div className="stats-row">
        <div className="stat-widget">
          <div className="stat-widget__icon stat-widget__icon--primary">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" />
            </svg>
          </div>
          <div className="stat-widget__content">
            <div className="stat-widget__value">{stationStats.count ?? 0}</div>
            <div className="stat-widget__label">Total Stations</div>
          </div>
          <div className="stat-widget__detail">
            <span className="detail-highlight">{connectedCount}</span> connected
          </div>
        </div>

        <div className="stat-widget">
          <div className="stat-widget__icon stat-widget__icon--success">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M7 18c-1.1 0-1.99.9-1.99 2S5.9 22 7 22s2-.9 2-2-.9-2-2-2zM1 2v2h2l3.6 7.59-1.35 2.45c-.16.28-.25.61-.25.96 0 1.1.9 2 2 2h12v-2H7.42c-.14 0-.25-.11-.25-.25l.03-.12.9-1.63h7.45c.75 0 1.41-.41 1.75-1.03l3.58-6.49c.08-.14.12-.31.12-.48 0-.55-.45-1-1-1H5.21l-.94-2H1z" />
            </svg>
          </div>
          <div className="stat-widget__content">
            <div className="stat-widget__value">{chargingCount}</div>
            <div className="stat-widget__label">Charging</div>
          </div>
          <div className="stat-widget__detail">
            <span className="detail-highlight">{availableCount}</span> available
          </div>
        </div>

        <div className="stat-widget">
          <div className="stat-widget__icon stat-widget__icon--info">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M20 2H4c-1.1 0-1.99.9-1.99 2L2 22l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm-7 12h-2v-2h2v2zm0-4h-2V6h2v4z" />
            </svg>
          </div>
          <div className="stat-widget__content">
            <div className="stat-widget__value">{messageStats.total ?? 0}</div>
            <div className="stat-widget__label">Messages</div>
          </div>
          <div className="stat-widget__detail">
            <span className="detail-sent">{messageStats.sent ?? 0}</span>
            <span className="detail-sep">/</span>
            <span className="detail-received">{messageStats.received ?? 0}</span>
          </div>
        </div>

        <div className="stat-widget">
          <div className="stat-widget__icon stat-widget__icon--warning">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M15 1H9v2h6V1zm-4 13h2V8h-2v6zm8.03-6.61l1.42-1.42c-.43-.51-.9-.99-1.41-1.41l-1.42 1.42C16.07 4.74 14.12 4 12 4c-4.97 0-9 4.03-9 9s4.02 9 9 9 9-4.03 9-9c0-2.12-.74-4.07-1.97-5.61zM12 20c-3.87 0-7-3.13-7-7s3.13-7 7-7 7 3.13 7 7-3.13 7-7 7z" />
            </svg>
          </div>
          <div className="stat-widget__content">
            <div className="stat-widget__value">{messageStats.buffered ?? 0}</div>
            <div className="stat-widget__label">Buffered</div>
          </div>
          <div className="stat-widget__detail">
            <span className={messageStats.dropped > 0 ? 'detail-danger' : ''}>{messageStats.dropped ?? 0}</span> dropped
          </div>
        </div>

        {faultedCount > 0 && (
          <div className="stat-widget stat-widget--alert">
            <div className="stat-widget__icon stat-widget__icon--danger">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z" />
              </svg>
            </div>
            <div className="stat-widget__content">
              <div className="stat-widget__value">{faultedCount}</div>
              <div className="stat-widget__label">Faulted</div>
            </div>
          </div>
        )}
      </div>

      {/* Main Content - Two Columns */}
      <div className="dashboard-content">
        {/* Station Overview */}
        <div className="dashboard-panel">
          <div className="panel-header">
            <h3>Station Overview</h3>
            <div className="panel-header-actions">
              <button
                className="sort-btn"
                onClick={() => handleSort('stationId')}
                title="Sort by Station ID"
              >
                Sort by ID
                {sortBy === 'stationId' && (
                  <span className="sort-indicator">
                    {sortOrder === 'asc' ? ' ▲' : ' ▼'}
                  </span>
                )}
              </button>
              <button className="btn-link" onClick={() => navigate('/stations')}>
                View All
              </button>
            </div>
          </div>
          <div className="station-list">
            {stations.length === 0 ? (
              <div className="empty-panel">
                <p>No stations configured</p>
                <button className="btn btn--sm btn--primary" onClick={() => navigate('/stations')}>
                  Add Station
                </button>
              </div>
            ) : (
              getSortedStations().slice(0, 8).map((station) => (
                <div key={station.stationId} className="station-row">
                  <div className="station-row__status">
                    <span className={`status-dot ${getStatusClass(station.runtimeState?.connectionStatus)}`} />
                  </div>
                  <div className="station-row__info">
                    <span className="station-row__name">{station.name}</span>
                    <span className="station-row__id">{station.stationId}</span>
                  </div>
                  <div className="station-row__meta">
                    <span className="station-row__protocol">{station.protocolVersion?.toUpperCase() || 'OCPP1.6'}</span>
                    <span className="station-row__connectors">{station.connectors?.length || 0} conn.</span>
                  </div>
                </div>
              ))
            )}
            {stations.length > 8 && (
              <div className="more-items">
                +{stations.length - 8} more stations
              </div>
            )}
          </div>
        </div>

        {/* System Info & Quick Stats */}
        <div className="dashboard-panel">
          <div className="panel-header">
            <h3>System Status</h3>
          </div>
          <div className="system-info">
            <div className="info-row">
              <span className="info-row__label">Database</span>
              <span className={`info-row__value info-row__value--${health?.database?.toLowerCase() || 'unknown'}`}>
                {health?.database || 'Unknown'}
              </span>
            </div>
            <div className="info-row">
              <span className="info-row__label">Connected Stations</span>
              <span className="info-row__value">{connectedCount} / {stationStats.count ?? 0}</span>
            </div>
            <div className="info-row">
              <span className="info-row__label">Available</span>
              <span className="info-row__value">{availableCount}</span>
            </div>
            <div className="info-row">
              <span className="info-row__label">Charging</span>
              <span className="info-row__value info-row__value--success">{chargingCount}</span>
            </div>
            <div className="info-row">
              <span className="info-row__label">Faulted</span>
              <span className={`info-row__value ${faultedCount > 0 ? 'info-row__value--danger' : ''}`}>
                {faultedCount}
              </span>
            </div>
            <div className="info-row">
              <span className="info-row__label">Unavailable</span>
              <span className="info-row__value">{health?.stations?.unavailable || 0}</span>
            </div>
          </div>

          <div className="panel-divider" />

          <div className="panel-header">
            <h3>Message Stats</h3>
            <button className="btn-link" onClick={() => navigate('/messages')}>
              View Messages
            </button>
          </div>
          <div className="message-stats-compact">
            <div className="msg-stat">
              <span className="msg-stat__value msg-stat__value--sent">{messageStats.sent ?? 0}</span>
              <span className="msg-stat__label">Sent</span>
            </div>
            <div className="msg-stat">
              <span className="msg-stat__value msg-stat__value--received">{messageStats.received ?? 0}</span>
              <span className="msg-stat__label">Received</span>
            </div>
            <div className="msg-stat">
              <span className="msg-stat__value">{messageStats.buffered ?? 0}</span>
              <span className="msg-stat__label">Buffered</span>
            </div>
            <div className="msg-stat">
              <span className={`msg-stat__value ${messageStats.dropped > 0 ? 'msg-stat__value--danger' : ''}`}>
                {messageStats.dropped ?? 0}
              </span>
              <span className="msg-stat__label">Dropped</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Dashboard
