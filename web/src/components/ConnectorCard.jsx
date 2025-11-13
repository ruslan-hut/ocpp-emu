import { useState } from 'react'
import { stationsAPI } from '../services/api'
import './ConnectorCard.css'

function ConnectorCard({ stationId, connector, onUpdate }) {
  const [loading, setLoading] = useState(false)
  const [showChargeForm, setShowChargeForm] = useState(false)
  const [idTag, setIdTag] = useState('USER001')

  const getStateColor = (state) => {
    switch (state) {
      case 'Available':
        return 'success'
      case 'Charging':
        return 'charging'
      case 'Preparing':
        return 'preparing'
      case 'Finishing':
        return 'finishing'
      case 'Faulted':
        return 'error'
      case 'Unavailable':
        return 'unavailable'
      default:
        return 'default'
    }
  }

  const handleStartCharging = async () => {
    if (!idTag.trim()) {
      alert('Please enter an ID Tag')
      return
    }

    setLoading(true)
    try {
      await stationsAPI.startCharging(stationId, connector.id, idTag)
      setShowChargeForm(false)
      if (onUpdate) {
        setTimeout(onUpdate, 500) // Give time for state to update
      }
    } catch (err) {
      alert(`Failed to start charging: ${err.response?.data?.error || err.message}`)
    } finally {
      setLoading(false)
    }
  }

  const handleStopCharging = async () => {
    if (!confirm('Stop charging session?')) {
      return
    }

    setLoading(true)
    try {
      await stationsAPI.stopCharging(stationId, connector.id, 'Local')
      if (onUpdate) {
        setTimeout(onUpdate, 500)
      }
    } catch (err) {
      alert(`Failed to stop charging: ${err.response?.data?.error || err.message}`)
    } finally {
      setLoading(false)
    }
  }

  const formatDateTime = (dateString) => {
    if (!dateString) return 'N/A'
    const date = new Date(dateString)
    return date.toLocaleString()
  }

  const formatDuration = (startTime) => {
    if (!startTime) return 'N/A'
    const start = new Date(startTime)
    const now = new Date()
    const diffMs = now - start
    const diffMins = Math.floor(diffMs / 60000)
    const hours = Math.floor(diffMins / 60)
    const mins = diffMins % 60
    return `${hours}h ${mins}m`
  }

  return (
    <div className="connector-card">
      <div className="connector-header">
        <div className="connector-id">
          <span className="label">Connector {connector.id}</span>
          <span className="type">{connector.type}</span>
        </div>
        <span className={`connector-state ${getStateColor(connector.state)}`}>
          {connector.state}
        </span>
      </div>

      <div className="connector-info">
        <div className="info-item">
          <span className="label">Max Power:</span>
          <span className="value">{connector.maxPower} W</span>
        </div>
        {connector.errorCode && connector.errorCode !== 'NoError' && (
          <div className="info-item error">
            <span className="label">Error:</span>
            <span className="value">{connector.errorCode}</span>
          </div>
        )}
      </div>

      {connector.transaction && (
        <div className="transaction-info">
          <h4>Active Transaction</h4>
          <div className="transaction-details">
            <div className="detail-row">
              <span className="label">Transaction ID:</span>
              <span className="value">{connector.transaction.id}</span>
            </div>
            <div className="detail-row">
              <span className="label">ID Tag:</span>
              <span className="value">{connector.transaction.idTag}</span>
            </div>
            <div className="detail-row">
              <span className="label">Started:</span>
              <span className="value">{formatDateTime(connector.transaction.startTime)}</span>
            </div>
            <div className="detail-row">
              <span className="label">Duration:</span>
              <span className="value">{formatDuration(connector.transaction.startTime)}</span>
            </div>
            <div className="detail-row">
              <span className="label">Energy:</span>
              <span className="value">
                {connector.transaction.currentMeter - connector.transaction.startMeterValue} Wh
              </span>
            </div>
          </div>
        </div>
      )}

      <div className="connector-actions">
        {connector.state === 'Available' && !showChargeForm && (
          <button
            className="btn-action btn-start-charging"
            onClick={() => setShowChargeForm(true)}
            disabled={loading}
          >
            ⚡ Start Charging
          </button>
        )}

        {connector.state === 'Charging' && (
          <button
            className="btn-action btn-stop-charging"
            onClick={handleStopCharging}
            disabled={loading}
          >
            ⏹ Stop Charging
          </button>
        )}

        {showChargeForm && (
          <div className="charge-form">
            <input
              type="text"
              placeholder="ID Tag (e.g., USER001)"
              value={idTag}
              onChange={(e) => setIdTag(e.target.value)}
              disabled={loading}
            />
            <div className="form-actions">
              <button
                className="btn-action btn-start"
                onClick={handleStartCharging}
                disabled={loading}
              >
                {loading ? 'Starting...' : 'Start'}
              </button>
              <button
                className="btn-action btn-cancel"
                onClick={() => setShowChargeForm(false)}
                disabled={loading}
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default ConnectorCard
