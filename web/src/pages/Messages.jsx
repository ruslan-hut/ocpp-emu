import { useState, useEffect, useRef, useMemo } from 'react'
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
  const [showExportMenu, setShowExportMenu] = useState(false)
  const [filters, setFilters] = useState({
    direction: 'all',
    stationId: '',
    messageType: 'all',
    action: '',
    searchQuery: '',
    limit: 50,
  })

  const wsRef = useRef(null)
  const messagesEndRef = useRef(null)
  const exportMenuRef = useRef(null)

  // Helper function to format payload for search
  const getSearchablePayload = (payload) => {
    if (!payload) return ''
    try {
      return JSON.stringify(payload)
    } catch {
      return String(payload)
    }
  }

  // Filter messages based on search and filters
  const filteredMessages = useMemo(() => {
    return messages.filter((message) => {
      // Message type filter
      if (filters.messageType !== 'all' && message.messageType !== filters.messageType) {
        return false
      }

      // Action filter
      if (filters.action && message.action && !message.action.toLowerCase().includes(filters.action.toLowerCase())) {
        return false
      }

      // Search query - search in multiple fields
      if (filters.searchQuery) {
        const query = filters.searchQuery.toLowerCase()
        const searchableText = [
          message.stationId,
          message.messageType,
          message.action,
          message.messageId,
          message.errorCode,
          message.errorDescription,
          getSearchablePayload(message.payload)
        ].join(' ').toLowerCase()

        if (!searchableText.includes(query)) {
          return false
        }
      }

      return true
    })
  }, [messages, filters.messageType, filters.action, filters.searchQuery])

  const messageCounts = useMemo(() => {
    return filteredMessages.reduce(
      (acc, message) => {
        acc.total += 1
        if (message.direction === 'sent') {
          acc.sent += 1
        } else if (message.direction === 'received') {
          acc.received += 1
        }
        return acc
      },
      { total: 0, sent: 0, received: 0 }
    )
  }, [filteredMessages])

  const displayStats = useMemo(() => {
    const chooseValue = (statValue, fallback) => {
      if (statValue === undefined || statValue === null) {
        return fallback
      }
      return Math.max(statValue, fallback)
    }

    return {
      total: chooseValue(stats?.total, messageCounts.total),
      sent: chooseValue(stats?.sent, messageCounts.sent),
      received: chooseValue(stats?.received, messageCounts.received),
      buffered: stats?.buffered ?? 0,
      dropped: stats?.dropped ?? 0,
    }
  }, [stats, messageCounts])

  const normalizeMessage = (message) => ({
    stationId: message.stationId ?? message.StationID ?? '',
    direction: message.direction ?? message.Direction ?? '',
    messageType: message.messageType ?? message.MessageType ?? '',
    action: message.action ?? message.Action ?? '',
    messageId: message.messageId ?? message.MessageID ?? '',
    protocolVersion: message.protocolVersion ?? message.ProtocolVersion ?? '',
    payload: message.payload ?? message.Payload ?? null,
    timestamp: message.timestamp ?? message.Timestamp ?? null,
    correlationId: message.correlationId ?? message.CorrelationID ?? '',
    errorCode: message.errorCode ?? message.ErrorCode ?? '',
    errorDescription:
      message.errorDescription ??
      message.ErrorDescription ??
      message.ErrorDesc ??
      message.errorDesc ??
      '',
  })

  useEffect(() => {
    fetchMessages()
    fetchStats()
  }, [filters])

  // Close export menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (exportMenuRef.current && !exportMenuRef.current.contains(event.target)) {
        setShowExportMenu(false)
      }
    }

    if (showExportMenu) {
      document.addEventListener('mousedown', handleClickOutside)
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [showExportMenu])

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
    const formattedMessage = normalizeMessage(message)

    // Apply direction filter
    if (filters.direction !== 'all' && formattedMessage.direction !== filters.direction) {
      return
    }
    // Apply station filter
    if (filters.stationId && formattedMessage.stationId !== filters.stationId) {
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
      if (filters.stationId) params.stationId = filters.stationId
      if (filters.direction !== 'all') params.direction = filters.direction
      params.limit = filters.limit

      const response = await messagesAPI.getAll(params)
      const normalized = (response.data.messages || []).map((message) => normalizeMessage(message))
      setMessages(normalized)
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

  // Get unique actions from messages for autocomplete
  const uniqueActions = useMemo(() => {
    const actions = new Set()
    messages.forEach((message) => {
      if (message.action) {
        actions.add(message.action)
      }
    })
    return Array.from(actions).sort()
  }, [messages])

  const handleClearFilters = () => {
    setFilters({
      direction: 'all',
      stationId: '',
      messageType: 'all',
      action: '',
      searchQuery: '',
      limit: 50,
    })
  }

  const hasActiveFilters = () => {
    return (
      filters.direction !== 'all' ||
      filters.stationId !== '' ||
      filters.messageType !== 'all' ||
      filters.action !== '' ||
      filters.searchQuery !== ''
    )
  }

  // Export functionality
  const exportToJSON = () => {
    const dataStr = JSON.stringify(filteredMessages, null, 2)
    const dataBlob = new Blob([dataStr], { type: 'application/json' })
    const url = URL.createObjectURL(dataBlob)
    const link = document.createElement('a')
    link.href = url
    link.download = `ocpp-messages-${new Date().toISOString()}.json`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }

  const exportToCSV = () => {
    // CSV headers
    const headers = [
      'Timestamp',
      'Station ID',
      'Direction',
      'Message Type',
      'Action',
      'Message ID',
      'Protocol Version',
      'Error Code',
      'Error Description',
      'Payload'
    ]

    // Convert messages to CSV rows
    const rows = filteredMessages.map((message) => [
      message.timestamp || '',
      message.stationId || '',
      message.direction || '',
      message.messageType || '',
      message.action || '',
      message.messageId || '',
      message.protocolVersion || '',
      message.errorCode || '',
      message.errorDescription || '',
      message.payload ? JSON.stringify(message.payload) : ''
    ])

    // Escape CSV fields (handle commas, quotes, newlines)
    const escapeCSV = (field) => {
      if (field === null || field === undefined) return ''
      const str = String(field)
      if (str.includes(',') || str.includes('"') || str.includes('\n')) {
        return `"${str.replace(/"/g, '""')}"`
      }
      return str
    }

    // Build CSV content
    const csvContent = [
      headers.map(escapeCSV).join(','),
      ...rows.map((row) => row.map(escapeCSV).join(','))
    ].join('\n')

    // Download
    const dataBlob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(dataBlob)
    const link = document.createElement('a')
    link.href = url
    link.download = `ocpp-messages-${new Date().toISOString()}.csv`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }

  const handleExport = (format) => {
    if (filteredMessages.length === 0) {
      alert('No messages to export')
      return
    }

    if (format === 'json') {
      exportToJSON()
    } else if (format === 'csv') {
      exportToCSV()
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

          <div className="export-dropdown" ref={exportMenuRef}>
            <button
              className="btn-export"
              onClick={() => setShowExportMenu(!showExportMenu)}
              disabled={filteredMessages.length === 0}
            >
              📥 Export ({filteredMessages.length})
            </button>
            {showExportMenu && (
              <div className="export-menu">
                <button onClick={() => { handleExport('json'); setShowExportMenu(false); }}>
                  Export as JSON
                </button>
                <button onClick={() => { handleExport('csv'); setShowExportMenu(false); }}>
                  Export as CSV
                </button>
              </div>
            )}
          </div>

          <button className="btn-danger" onClick={handleClearMessages}>
            Clear All Messages
          </button>
        </div>
      </div>

      {(stats || messageCounts.total > 0) && (
        <div className="message-stats">
          <div className="stat-item">
            <span className="stat-label">Total:</span>
            <span className="stat-value">{displayStats.total}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Sent:</span>
            <span className="stat-value">{displayStats.sent}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Received:</span>
            <span className="stat-value">{displayStats.received}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Buffered:</span>
            <span className="stat-value">{displayStats.buffered}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Dropped:</span>
            <span className="stat-value">{displayStats.dropped}</span>
          </div>
        </div>
      )}

      <div className="filters-container">
        {/* Search Bar */}
        <div className="search-bar">
          <input
            type="text"
            className="search-input"
            placeholder="🔍 Search messages (station, action, payload, error...)..."
            value={filters.searchQuery}
            onChange={(e) => handleFilterChange('searchQuery', e.target.value)}
          />
          {hasActiveFilters() && (
            <button className="btn-clear-filters" onClick={handleClearFilters}>
              Clear Filters
            </button>
          )}
        </div>

        {/* Filters Row */}
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
            <label htmlFor="messageType">Message Type:</label>
            <select
              id="messageType"
              value={filters.messageType}
              onChange={(e) => handleFilterChange('messageType', e.target.value)}
            >
              <option value="all">All</option>
              <option value="Call">Call (2)</option>
              <option value="CallResult">CallResult (3)</option>
              <option value="CallError">CallError (4)</option>
            </select>
          </div>

          <div className="filter-group">
            <label htmlFor="action">Action:</label>
            <input
              id="action"
              type="text"
              list="action-suggestions"
              placeholder="Filter by action..."
              value={filters.action}
              onChange={(e) => handleFilterChange('action', e.target.value)}
            />
            <datalist id="action-suggestions">
              {uniqueActions.map((action) => (
                <option key={action} value={action} />
              ))}
            </datalist>
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

        {/* Active Filter Tags */}
        {hasActiveFilters() && (
          <div className="active-filters">
            <span className="active-filters-label">Active filters:</span>
            {filters.direction !== 'all' && (
              <span className="filter-tag">
                Direction: {filters.direction}
                <button onClick={() => handleFilterChange('direction', 'all')}>×</button>
              </span>
            )}
            {filters.messageType !== 'all' && (
              <span className="filter-tag">
                Type: {filters.messageType}
                <button onClick={() => handleFilterChange('messageType', 'all')}>×</button>
              </span>
            )}
            {filters.action && (
              <span className="filter-tag">
                Action: {filters.action}
                <button onClick={() => handleFilterChange('action', '')}>×</button>
              </span>
            )}
            {filters.stationId && (
              <span className="filter-tag">
                Station: {filters.stationId}
                <button onClick={() => handleFilterChange('stationId', '')}>×</button>
              </span>
            )}
            {filters.searchQuery && (
              <span className="filter-tag">
                Search: "{filters.searchQuery}"
                <button onClick={() => handleFilterChange('searchQuery', '')}>×</button>
              </span>
            )}
          </div>
        )}

        {/* Results Count */}
        <div className="results-info">
          Showing {filteredMessages.length} of {messages.length} messages
        </div>
      </div>

      {filteredMessages.length === 0 ? (
        <div className="empty-state">
          <p>{messages.length === 0 ? 'No messages found' : 'No messages match your filters'}</p>
          {messages.length > 0 && hasActiveFilters() && (
            <button className="btn-secondary" onClick={handleClearFilters}>
              Clear All Filters
            </button>
          )}
        </div>
      ) : (
        <div className="messages-list">
          {filteredMessages.map((message, index) => (
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
