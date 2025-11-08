import { useState, useEffect } from 'react'
import { stationsAPI } from '../services/api'
import './Stations.css'

function Stations() {
  const [stations, setStations] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchStations()
  }, [])

  const fetchStations = async () => {
    try {
      const response = await stationsAPI.getAll()
      setStations(response.data.stations || [])
      setError(null)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleStart = async (stationId) => {
    try {
      await stationsAPI.start(stationId)
      fetchStations()
    } catch (err) {
      alert(`Failed to start station: ${err.message}`)
    }
  }

  const handleStop = async (stationId) => {
    try {
      await stationsAPI.stop(stationId)
      fetchStations()
    } catch (err) {
      alert(`Failed to stop station: ${err.message}`)
    }
  }

  const handleDelete = async (stationId) => {
    if (!confirm(`Are you sure you want to delete station ${stationId}?`)) {
      return
    }

    try {
      await stationsAPI.delete(stationId)
      fetchStations()
    } catch (err) {
      alert(`Failed to delete station: ${err.message}`)
    }
  }

  if (loading) {
    return <div className="loading">Loading stations...</div>
  }

  if (error) {
    return <div className="error">Error loading stations: {error}</div>
  }

  return (
    <div className="stations">
      <div className="page-header">
        <h2>Charging Stations</h2>
        <button className="btn-primary">+ Add Station</button>
      </div>

      {stations.length === 0 ? (
        <div className="empty-state">
          <p>No stations configured</p>
          <button className="btn-primary">Create Your First Station</button>
        </div>
      ) : (
        <div className="stations-grid">
          {stations.map((station) => (
            <div key={station.stationId} className="station-card">
              <div className="station-header">
                <h3>{station.name}</h3>
                <span className={`status-badge ${station.runtimeState?.connectionStatus}`}>
                  {station.runtimeState?.connectionStatus || 'unknown'}
                </span>
              </div>

              <div className="station-info">
                <div className="info-row">
                  <span className="label">ID:</span>
                  <span className="value">{station.stationId}</span>
                </div>
                <div className="info-row">
                  <span className="label">Vendor:</span>
                  <span className="value">{station.vendor}</span>
                </div>
                <div className="info-row">
                  <span className="label">Model:</span>
                  <span className="value">{station.model}</span>
                </div>
                <div className="info-row">
                  <span className="label">Protocol:</span>
                  <span className="value">{station.protocolVersion}</span>
                </div>
                <div className="info-row">
                  <span className="label">Connectors:</span>
                  <span className="value">{station.connectors?.length || 0}</span>
                </div>
                <div className="info-row">
                  <span className="label">Enabled:</span>
                  <span className="value">{station.enabled ? 'Yes' : 'No'}</span>
                </div>
              </div>

              <div className="station-actions">
                {station.runtimeState?.connectionStatus === 'connected' ? (
                  <button
                    className="btn-secondary"
                    onClick={() => handleStop(station.stationId)}
                  >
                    Stop
                  </button>
                ) : (
                  <button
                    className="btn-primary"
                    onClick={() => handleStart(station.stationId)}
                    disabled={!station.enabled}
                  >
                    Start
                  </button>
                )}
                <button className="btn-secondary">Edit</button>
                <button
                  className="btn-danger"
                  onClick={() => handleDelete(station.stationId)}
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default Stations
