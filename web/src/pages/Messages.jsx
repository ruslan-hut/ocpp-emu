import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
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
  const [selectedMessage, setSelectedMessage] = useState(null)
  const [splitPosition, setSplitPosition] = useState(() => {
    const saved = localStorage.getItem('messagesSplitPosition')
    return saved ? parseInt(saved, 10) : 50
  })
  const [isResizing, setIsResizing] = useState(false)
  const [filters, setFilters] = useState({
    direction: 'all',
    stationId: '',
    messageType: 'all',
    action: '',
    searchQuery: '',
    limit: 100,
  })

  const wsRef = useRef(null)
  const messagesEndRef = useRef(null)
  const exportMenuRef = useRef(null)
  const splitContainerRef = useRef(null)

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

  // JSON syntax highlighter - returns HTML with spans for different token types
  const highlightJSON = (payload) => {
    if (!payload) return '<span class="json-null">N/A</span>'

    try {
      const json = typeof payload === 'string' ? payload : JSON.stringify(payload, null, 2)

      // Tokenize and highlight JSON
      let result = ''
      let i = 0

      while (i < json.length) {
        const char = json[i]

        // Whitespace (preserve formatting)
        if (char === ' ' || char === '\n' || char === '\t' || char === '\r') {
          result += char
          i++
          continue
        }

        // Strings (keys or values)
        if (char === '"') {
          let str = '"'
          i++
          while (i < json.length && json[i] !== '"') {
            if (json[i] === '\\' && i + 1 < json.length) {
              str += json[i] + json[i + 1]
              i += 2
            } else {
              str += json[i]
              i++
            }
          }
          str += '"'
          i++

          // Check if this is a key (followed by colon) or a value
          let lookAhead = i
          while (lookAhead < json.length && (json[lookAhead] === ' ' || json[lookAhead] === '\n' || json[lookAhead] === '\t')) {
            lookAhead++
          }

          if (json[lookAhead] === ':') {
            result += `<span class="json-key">${escapeHtml(str)}</span>`
          } else {
            result += `<span class="json-string">${escapeHtml(str)}</span>`
          }
          continue
        }

        // Numbers
        if (char === '-' || (char >= '0' && char <= '9')) {
          let num = ''
          while (i < json.length && /[-0-9.eE+]/.test(json[i])) {
            num += json[i]
            i++
          }
          result += `<span class="json-number">${escapeHtml(num)}</span>`
          continue
        }

        // Booleans and null
        if (json.slice(i, i + 4) === 'true') {
          result += '<span class="json-boolean">true</span>'
          i += 4
          continue
        }
        if (json.slice(i, i + 5) === 'false') {
          result += '<span class="json-boolean">false</span>'
          i += 5
          continue
        }
        if (json.slice(i, i + 4) === 'null') {
          result += '<span class="json-null">null</span>'
          i += 4
          continue
        }

        // Brackets and punctuation
        if (char === '{' || char === '}') {
          result += `<span class="json-brace">${char}</span>`
          i++
          continue
        }
        if (char === '[' || char === ']') {
          result += `<span class="json-bracket">${char}</span>`
          i++
          continue
        }
        if (char === ':') {
          result += '<span class="json-colon">:</span>'
          i++
          continue
        }
        if (char === ',') {
          result += '<span class="json-comma">,</span>'
          i++
          continue
        }

        // Any other character
        result += escapeHtml(char)
        i++
      }

      return result
    } catch {
      return escapeHtml(String(payload))
    }
  }

  // Escape HTML entities to prevent XSS
  const escapeHtml = (str) => {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;')
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
      limit: 100,
    })
  }

  // Resize handlers for split panel
  const handleResizeStart = useCallback((e) => {
    e.preventDefault()
    setIsResizing(true)
  }, [])

  const handleResizeMove = useCallback((e) => {
    if (!isResizing || !splitContainerRef.current) return

    const container = splitContainerRef.current
    const containerRect = container.getBoundingClientRect()
    const newPosition = ((e.clientX - containerRect.left) / containerRect.width) * 100

    // Clamp between 30% and 70%
    const clampedPosition = Math.min(Math.max(newPosition, 30), 70)
    setSplitPosition(clampedPosition)
  }, [isResizing])

  const handleResizeEnd = useCallback(() => {
    if (isResizing) {
      setIsResizing(false)
      localStorage.setItem('messagesSplitPosition', splitPosition.toString())
    }
  }, [isResizing, splitPosition])

  useEffect(() => {
    if (isResizing) {
      document.addEventListener('mousemove', handleResizeMove)
      document.addEventListener('mouseup', handleResizeEnd)
      document.body.style.cursor = 'col-resize'
      document.body.style.userSelect = 'none'
    }

    return () => {
      document.removeEventListener('mousemove', handleResizeMove)
      document.removeEventListener('mouseup', handleResizeEnd)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
  }, [isResizing, handleResizeMove, handleResizeEnd])

  const handleMessageSelect = (message, index) => {
    setSelectedMessage({ ...message, index })
  }

  const formatTimestampCompact = (timestamp) => {
    if (!timestamp) return ''
    const date = new Date(timestamp)
    return date.toLocaleTimeString('en-US', { hour12: false })
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
    <div className="messages messages--desktop">
      {/* Fixed Header Bar */}
      <div className="messages-header">
        <div className="messages-header__top">
          <h2>OCPP Messages</h2>
          <div className="header-actions">
            <div className="live-updates-toggle">
              <label>
                <input
                  type="checkbox"
                  checked={liveUpdates}
                  onChange={(e) => setLiveUpdates(e.target.checked)}
                />
                <span>Live</span>
                {wsConnected && <span className="ws-indicator connected">●</span>}
                {!wsConnected && liveUpdates && <span className="ws-indicator disconnected">●</span>}
              </label>
            </div>

            {/* Inline Stats */}
            {(stats || messageCounts.total > 0) && (
              <div className="stats-inline">
                <span className="stat-inline">
                  <span className="stat-inline__value">{displayStats.total}</span>
                  <span className="stat-inline__label">total</span>
                </span>
                <span className="stat-inline stat-inline--sent">
                  <span className="stat-inline__value">{displayStats.sent}</span>
                  <span className="stat-inline__label">sent</span>
                </span>
                <span className="stat-inline stat-inline--received">
                  <span className="stat-inline__value">{displayStats.received}</span>
                  <span className="stat-inline__label">recv</span>
                </span>
              </div>
            )}

            <div className="export-dropdown" ref={exportMenuRef}>
              <button
                className="btn btn--sm btn--secondary"
                onClick={() => setShowExportMenu(!showExportMenu)}
                disabled={filteredMessages.length === 0}
              >
                Export
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

            <button className="btn btn--sm btn--danger" onClick={handleClearMessages}>
              Clear
            </button>
          </div>
        </div>

        {/* Compact Filter Bar */}
        <div className="filters-bar">
          <input
            type="text"
            className="search-input search-input--compact"
            placeholder="Search messages..."
            value={filters.searchQuery}
            onChange={(e) => handleFilterChange('searchQuery', e.target.value)}
          />

          <div className="filters-inline">
            <select
              className="filter-select"
              value={filters.direction}
              onChange={(e) => handleFilterChange('direction', e.target.value)}
            >
              <option value="all">All Directions</option>
              <option value="sent">Sent</option>
              <option value="received">Received</option>
            </select>

            <select
              className="filter-select"
              value={filters.messageType}
              onChange={(e) => handleFilterChange('messageType', e.target.value)}
            >
              <option value="all">All Types</option>
              <option value="Call">Call</option>
              <option value="CallResult">CallResult</option>
              <option value="CallError">CallError</option>
            </select>

            <input
              type="text"
              className="filter-input"
              list="action-suggestions"
              placeholder="Action..."
              value={filters.action}
              onChange={(e) => handleFilterChange('action', e.target.value)}
            />
            <datalist id="action-suggestions">
              {uniqueActions.map((action) => (
                <option key={action} value={action} />
              ))}
            </datalist>

            <input
              type="text"
              className="filter-input"
              placeholder="Station ID..."
              value={filters.stationId}
              onChange={(e) => handleFilterChange('stationId', e.target.value)}
            />

            <select
              className="filter-select filter-select--sm"
              value={filters.limit}
              onChange={(e) => handleFilterChange('limit', parseInt(e.target.value))}
            >
              <option value="50">50</option>
              <option value="100">100</option>
              <option value="200">200</option>
              <option value="500">500</option>
            </select>

            {hasActiveFilters() && (
              <button className="btn-clear-filters--sm" onClick={handleClearFilters}>
                Clear
              </button>
            )}
          </div>

          <div className="results-count">
            {filteredMessages.length} / {messages.length}
          </div>
        </div>
      </div>

      {/* Split Panel Content */}
      <div className="messages-content" ref={splitContainerRef}>
        {/* Message List Panel */}
        <div
          className="messages-list-panel"
          style={{ width: `${splitPosition}%` }}
        >
          {filteredMessages.length === 0 ? (
            <div className="empty-state empty-state--compact">
              <p>{messages.length === 0 ? 'No messages found' : 'No messages match filters'}</p>
              {messages.length > 0 && hasActiveFilters() && (
                <button className="btn-link" onClick={handleClearFilters}>
                  Clear filters
                </button>
              )}
            </div>
          ) : (
            <div className="messages-list messages-list--compact">
              {filteredMessages.map((message, index) => (
                <div
                  key={index}
                  className={`message-row ${message.direction} ${selectedMessage?.index === index ? 'selected' : ''}`}
                  onClick={() => handleMessageSelect(message, index)}
                >
                  <span className={`direction-indicator ${message.direction}`}>
                    {message.direction === 'sent' ? '→' : '←'}
                  </span>
                  <span className="message-row__time">
                    {formatTimestampCompact(message.timestamp)}
                  </span>
                  <span className={`message-row__type message-row__type--${message.messageType?.toLowerCase()}`}>
                    {message.messageType === 'CallResult' ? 'Result' : message.messageType === 'CallError' ? 'Error' : message.messageType}
                  </span>
                  <span className="message-row__action">
                    {message.action || '-'}
                  </span>
                  <span className="message-row__station">
                    {message.stationId}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Resize Handle */}
        <div
          className={`resize-handle ${isResizing ? 'resizing' : ''}`}
          onMouseDown={handleResizeStart}
        >
          <div className="resize-handle__grip" />
        </div>

        {/* Message Detail Panel */}
        <div
          className="messages-detail-panel"
          style={{ width: `${100 - splitPosition}%` }}
        >
          {selectedMessage ? (
            <div className="message-detail">
              <div className="message-detail__header">
                <div className="message-detail__title">
                  <span className={`direction-badge direction-badge--lg ${selectedMessage.direction}`}>
                    {selectedMessage.direction}
                  </span>
                  <span className="message-detail__action">{selectedMessage.action || 'N/A'}</span>
                  <span className={`message-type-badge message-type-badge--${selectedMessage.messageType?.toLowerCase()}`}>
                    {selectedMessage.messageType}
                  </span>
                </div>
                <button className="btn-close-detail" onClick={() => setSelectedMessage(null)}>
                  ×
                </button>
              </div>

              <div className="message-detail__meta">
                <div className="meta-row">
                  <span className="meta-label">Timestamp</span>
                  <span className="meta-value">{formatTimestamp(selectedMessage.timestamp)}</span>
                </div>
                <div className="meta-row">
                  <span className="meta-label">Station ID</span>
                  <span className="meta-value meta-value--mono">{selectedMessage.stationId}</span>
                </div>
                <div className="meta-row">
                  <span className="meta-label">Message ID</span>
                  <span className="meta-value meta-value--mono">{selectedMessage.messageId || 'N/A'}</span>
                </div>
                {selectedMessage.correlationId && (
                  <div className="meta-row">
                    <span className="meta-label">Correlation ID</span>
                    <span className="meta-value meta-value--mono">{selectedMessage.correlationId}</span>
                  </div>
                )}
                <div className="meta-row">
                  <span className="meta-label">Protocol</span>
                  <span className="meta-value">{selectedMessage.protocolVersion || 'OCPP 1.6'}</span>
                </div>
                {selectedMessage.errorCode && (
                  <div className="meta-row meta-row--error">
                    <span className="meta-label">Error Code</span>
                    <span className="meta-value meta-value--error">{selectedMessage.errorCode}</span>
                  </div>
                )}
                {selectedMessage.errorDescription && (
                  <div className="meta-row meta-row--error">
                    <span className="meta-label">Error Description</span>
                    <span className="meta-value meta-value--error">{selectedMessage.errorDescription}</span>
                  </div>
                )}
              </div>

              <div className="message-detail__payload">
                <div className="payload-header">
                  <span className="payload-title">Payload</span>
                  <button
                    className="btn-copy"
                    onClick={() => {
                      navigator.clipboard.writeText(formatPayload(selectedMessage.payload))
                    }}
                  >
                    Copy
                  </button>
                </div>
                <pre
                    className="payload-content payload-content--highlighted"
                    dangerouslySetInnerHTML={{ __html: highlightJSON(selectedMessage.payload) }}
                  />
              </div>
            </div>
          ) : (
            <div className="message-detail__empty">
              <div className="empty-detail-icon">📋</div>
              <p>Select a message to view details</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default Messages
