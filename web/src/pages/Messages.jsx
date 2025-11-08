import { useState, useEffect, useRef } from 'react'
import { messagesAPI } from '../services/api'
import './Messages.css'

// Use relative WebSocket URL for Docker/production, full URL for development
const WS_URL = import.meta.env.VITE_WS_URL ||
  (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host

function Messages() {
  const [messages, setMessages] = useState([])
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [liveUpdates, setLiveUpdates] = useState(true)
  const [wsConnected, setWsConnected] = useState(false)
  const [filters, setFilters] = useState({
    direction: 'all',
    stationId: '',
    limit: 50,
  })

  const wsRef = useRef(null)
  const messagesEndRef = useRef(null)

  useEffect(() => {
    fetchMessages()
    fetchStats()
  }, [filters])

  // WebSocket connection for real-time updates
  useEffect(() => {
    if (!liveUpdates) {
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
      setWsConnected(false)
      return
    }

    connectWebSocket()

    return () => {
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [liveUpdates, filters.stationId])

  const connectWebSocket = () => {
    try {
      // Build WebSocket URL with filters
      let wsUrl = `${WS_URL}/api/ws/messages`
      const params = new URLSearchParams()
      if (filters.stationId) {
        params.append('stationId', filters.stationId)
      }
      if (params.toString()) {
        wsUrl += `?${params.toString()}`
      }

      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        console.log('WebSocket connected')
        setWsConnected(true)
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)

          if (data.type === 'welcome') {
            console.log('Received welcome message:', data.message)
          } else if (data.type === 'ocpp_message') {
            handleNewMessage(data.message)
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        setWsConnected(false)
      }

      ws.onclose = () => {
        console.log('WebSocket closed')
        setWsConnected(false)

        // Attempt to reconnect after 5 seconds if live updates are still enabled
        if (liveUpdates) {
          setTimeout(() => {
            if (liveUpdates) {
              connectWebSocket()
            }
          }, 5000)
        }
      }
    } catch (err) {
      console.error('Failed to create WebSocket:', err)
    }
  }

  const handleNewMessage = (message) => {
    // Convert message entry format to storage format
    const formattedMessage = {
      stationId: message.StationID,
      direction: message.Direction,
      messageType: message.MessageType,
      action: message.Action,
      messageId: message.MessageID,
      protocolVersion: message.ProtocolVersion,
      payload: message.Payload,
      timestamp: message.Timestamp,
      correlationId: message.CorrelationID,
      errorCode: message.ErrorCode,
      errorDescription: message.ErrorDesc,
    }

    // Apply direction filter
    if (filters.direction !== 'all' && formattedMessage.direction !== filters.direction) {
      return
    }

    setMessages((prev) => {
      // Add new message at the beginning
      const newMessages = [formattedMessage, ...prev]

      // Limit the number of messages in memory
      if (newMessages.length > filters.limit) {
        return newMessages.slice(0, filters.limit)
      }

      return newMessages
    })

    // Update stats
    setStats((prev) => {
      if (!prev) return prev
      return {
        ...prev,
        total: (prev.total || 0) + 1,
        sent: formattedMessage.direction === 'sent' ? (prev.sent || 0) + 1 : prev.sent,
        received: formattedMessage.direction === 'received' ? (prev.received || 0) + 1 : prev.received,
      }
    })

    // Auto-scroll to top where new messages appear
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

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
        <div className="header-actions">
          <div className="live-updates-toggle">
            <label>
              <input
                type="checkbox"
                checked={liveUpdates}
                onChange={(e) => setLiveUpdates(e.target.checked)}
              />
              <span>Live Updates</span>
              {wsConnected && <span className="ws-indicator connected">●</span>}
              {!wsConnected && liveUpdates && <span className="ws-indicator disconnected">●</span>}
            </label>
          </div>
          <button className="btn-danger" onClick={handleClearMessages}>
            Clear All Messages
          </button>
        </div>
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
