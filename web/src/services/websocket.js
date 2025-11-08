class WebSocketService {
  constructor() {
    this.ws = null
    this.url = null
    this.reconnectInterval = 5000
    this.reconnectTimeout = null
    this.listeners = new Map()
    this.isIntentionallyClosed = false
  }

  connect(url = 'ws://localhost:8080/ws') {
    this.url = url
    this.isIntentionallyClosed = false

    try {
      this.ws = new WebSocket(url)

      this.ws.onopen = () => {
        console.log('WebSocket connected')
        this.notifyListeners('open', {})
      }

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          this.notifyListeners('message', data)
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error)
        }
      }

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        this.notifyListeners('error', error)
      }

      this.ws.onclose = () => {
        console.log('WebSocket closed')
        this.notifyListeners('close', {})

        // Attempt to reconnect if not intentionally closed
        if (!this.isIntentionallyClosed) {
          this.scheduleReconnect()
        }
      }
    } catch (error) {
      console.error('Failed to create WebSocket:', error)
      this.scheduleReconnect()
    }
  }

  scheduleReconnect() {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout)
    }

    this.reconnectTimeout = setTimeout(() => {
      console.log('Attempting to reconnect WebSocket...')
      this.connect(this.url)
    }, this.reconnectInterval)
  }

  send(data) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data))
      return true
    }
    console.warn('WebSocket is not connected')
    return false
  }

  close() {
    this.isIntentionallyClosed = true

    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout)
      this.reconnectTimeout = null
    }

    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, [])
    }
    this.listeners.get(event).push(callback)
  }

  off(event, callback) {
    if (!this.listeners.has(event)) return

    const callbacks = this.listeners.get(event)
    const index = callbacks.indexOf(callback)
    if (index > -1) {
      callbacks.splice(index, 1)
    }
  }

  notifyListeners(event, data) {
    if (!this.listeners.has(event)) return

    const callbacks = this.listeners.get(event)
    callbacks.forEach((callback) => {
      try {
        callback(data)
      } catch (error) {
        console.error(`Error in WebSocket listener for event '${event}':`, error)
      }
    })
  }

  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN
  }
}

// Create singleton instance
const websocketService = new WebSocketService()

export default websocketService
