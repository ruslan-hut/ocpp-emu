import { useState } from 'react'
import { stationsAPI } from '../services/api'
import './ConnectorCard.css'

function ConnectorCard({ stationId, connector, isStationConnected = true, onUpdate }) {
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
        setTimeout(onUpdate, 500)
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

  const formatDuration = (startTime) => {
    if (!startTime) return '--'
    const start = new Date(startTime)
    const now = new Date()
    const diffMs = now - start
    const diffMins = Math.floor(diffMs / 60000)
    const hours = Math.floor(diffMins / 60)
    const mins = diffMins % 60
    return `${hours}h ${mins}m`
  }

  const hasError = connector.errorCode && connector.errorCode !== 'NoError'

  return (
    <div className={`connector-card ${hasError ? 'has-error' : ''}`}>
      <div className="connector-card__header">
        <div className="connector-card__title">
          <span className="connector-num">#{connector.id}</span>
          <span className="connector-type">{connector.type}</span>
        </div>
        <span className={`connector-state ${getStateColor(connector.state)}`}>
          {connector.state}
        </span>
      </div>

      <div className="connector-card__body">
        <div className="connector-card__stats">
          <div className="stat">
            <span className="stat-label">Power</span>
            <span className="stat-value">{(connector.maxPower / 1000).toFixed(1)} kW</span>
          </div>
          {connector.transaction && (
            <>
              <div className="stat">
                <span className="stat-label">Energy</span>
                <span className="stat-value">
                  {((connector.transaction.currentMeter - connector.transaction.startMeterValue) / 1000).toFixed(2)} kWh
                </span>
              </div>
              <div className="stat">
                <span className="stat-label">Duration</span>
                <span className="stat-value">{formatDuration(connector.transaction.startTime)}</span>
              </div>
            </>
          )}
          {hasError && (
            <div className="stat stat--error">
              <span className="stat-label">Error</span>
              <span className="stat-value">{connector.errorCode}</span>
            </div>
          )}
        </div>

        <div className="connector-card__actions">
          {!isStationConnected && (
            <span className="warn-icon" title="Station not connected">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                <path d="M8 1L1 14h14L8 1zm0 3.5l4.5 8h-9L8 4.5zM7.25 7v3h1.5V7h-1.5zm0 4v1.5h1.5V11h-1.5z"/>
              </svg>
            </span>
          )}

          {showChargeForm ? (
            <div className="charge-form">
              <input
                type="text"
                placeholder="ID Tag"
                value={idTag}
                onChange={(e) => setIdTag(e.target.value)}
                disabled={loading || !isStationConnected}
              />
              <button
                className="btn btn--xs btn--primary"
                onClick={handleStartCharging}
                disabled={loading || !isStationConnected}
              >
                {loading ? '...' : 'Go'}
              </button>
              <button
                className="btn btn--xs btn--secondary"
                onClick={() => setShowChargeForm(false)}
                disabled={loading}
              >
                X
              </button>
            </div>
          ) : (
            <>
              {connector.state === 'Available' && (
                <button
                  className="btn btn--xs btn--success"
                  onClick={() => setShowChargeForm(true)}
                  disabled={loading || !isStationConnected}
                  title={!isStationConnected ? 'Station not connected' : 'Start charging'}
                >
                  Start
                </button>
              )}
              {connector.state === 'Charging' && (
                <button
                  className="btn btn--xs btn--danger"
                  onClick={handleStopCharging}
                  disabled={loading || !isStationConnected}
                  title={!isStationConnected ? 'Station not connected' : 'Stop charging'}
                >
                  Stop
                </button>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

export default ConnectorCard
