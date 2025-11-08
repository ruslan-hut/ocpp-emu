import { useState, useEffect } from 'react'
import { messagesAPI } from '../services/api'
import './Messages.css'

function Messages() {
  const [messages, setMessages] = useState([])
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [filters, setFilters] = useState({
    direction: 'all',
    stationId: '',
    limit: 50,
  })

  useEffect(() => {
    fetchMessages()
    fetchStats()
  }, [filters])

  const fetchMessages = async () => {
    try {
      const params = {}
      if (filters.stationId) params.station_id = filters.stationId
      if (filters.direction !== 'all') params.direction = filters.direction
      params.limit = filters.limit

      const response = await messagesAPI.getAll(params)
      setMessages(response.data.messages || [])
      setError(null)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const fetchStats = async () => {
    try {
      const response = await messagesAPI.getStats()
      setStats(response.data)
    } catch (err) {
      console.error('Failed to fetch stats:', err)
    }
  }

  const handleFilterChange = (key, value) => {
    setFilters((prev) => ({ ...prev, [key]: value }))
  }

  const handleClearMessages = async () => {
    if (!confirm('Are you sure you want to clear all messages?')) {
      return
    }

    try {
      await messagesAPI.clear()
      fetchMessages()
      fetchStats()
    } catch (err) {
      alert(`Failed to clear messages: ${err.message}`)
    }
  }

  const formatTimestamp = (timestamp) => {
    if (!timestamp) return 'N/A'
    return new Date(timestamp).toLocaleString()
  }

  const formatPayload = (payload) => {
    if (!payload) return 'N/A'
    try {
      return JSON.stringify(payload, null, 2)
    } catch {
      return String(payload)
    }
  }

  if (loading) {
    return <div className="loading">Loading messages...</div>
  }

  if (error) {
    return <div className="error">Error loading messages: {error}</div>
  }

  return (
    <div className="messages">
      <div className="page-header">
        <h2>OCPP Messages</h2>
        <button className="btn-danger" onClick={handleClearMessages}>
          Clear All Messages
        </button>
      </div>

      {stats && (
        <div className="message-stats">
          <div className="stat-item">
            <span className="stat-label">Total:</span>
            <span className="stat-value">{stats.total || 0}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Sent:</span>
            <span className="stat-value">{stats.sent || 0}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Received:</span>
            <span className="stat-value">{stats.received || 0}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Buffered:</span>
            <span className="stat-value">{stats.buffered || 0}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Dropped:</span>
            <span className="stat-value">{stats.dropped || 0}</span>
          </div>
        </div>
      )}

      <div className="filters">
        <div className="filter-group">
          <label htmlFor="direction">Direction:</label>
          <select
            id="direction"
            value={filters.direction}
            onChange={(e) => handleFilterChange('direction', e.target.value)}
          >
            <option value="all">All</option>
            <option value="sent">Sent</option>
            <option value="received">Received</option>
          </select>
        </div>

        <div className="filter-group">
          <label htmlFor="stationId">Station ID:</label>
          <input
            id="stationId"
            type="text"
            placeholder="Filter by station..."
            value={filters.stationId}
            onChange={(e) => handleFilterChange('stationId', e.target.value)}
          />
        </div>

        <div className="filter-group">
          <label htmlFor="limit">Limit:</label>
          <select
            id="limit"
            value={filters.limit}
            onChange={(e) => handleFilterChange('limit', parseInt(e.target.value))}
          >
            <option value="25">25</option>
            <option value="50">50</option>
            <option value="100">100</option>
            <option value="200">200</option>
          </select>
        </div>
      </div>

      {messages.length === 0 ? (
        <div className="empty-state">
          <p>No messages found</p>
        </div>
      ) : (
        <div className="messages-list">
          {messages.map((message, index) => (
            <div key={index} className={`message-card ${message.direction}`}>
              <div className="message-header">
                <div className="message-info">
                  <span className={`direction-badge ${message.direction}`}>
                    {message.direction}
                  </span>
                  <span className="message-type">{message.messageType}</span>
                  <span className="message-station">{message.stationId}</span>
                </div>
                <div className="message-timestamp">
                  {formatTimestamp(message.timestamp)}
                </div>
              </div>

              <div className="message-details">
                <div className="detail-row">
                  <span className="detail-label">Message ID:</span>
                  <span className="detail-value">{message.messageId || 'N/A'}</span>
                </div>
                <div className="detail-row">
                  <span className="detail-label">Action:</span>
                  <span className="detail-value">{message.action || 'N/A'}</span>
                </div>
              </div>

              {message.payload && (
                <details className="message-payload">
                  <summary>View Payload</summary>
                  <pre>{formatPayload(message.payload)}</pre>
                </details>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default Messages
