import { useState, useEffect } from 'react'
import { stationsAPI } from '../services/api'
import StationForm from '../components/StationForm'
import StationConfig from '../components/StationConfig'
import TemplatesManager from '../components/TemplatesManager'
import ImportExport from '../components/ImportExport'
import ConnectorCard from '../components/ConnectorCard'
import './Stations.css'

function Stations() {
  const [stations, setStations] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showForm, setShowForm] = useState(false)
  const [editingStation, setEditingStation] = useState(null)
  const [showTemplates, setShowTemplates] = useState(false)
  const [showImportExport, setShowImportExport] = useState(false)
  const [templates, setTemplates] = useState([])
  const [showConnectors, setShowConnectors] = useState(false)
  const [selectedStationId, setSelectedStationId] = useState(null)
  const [connectors, setConnectors] = useState([])
  const [connectorsLoading, setConnectorsLoading] = useState(false)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [refreshInterval, setRefreshInterval] = useState(5)
  const [showConfig, setShowConfig] = useState(false)
  const [configuringStation, setConfiguringStation] = useState(null)

  useEffect(() => {
    fetchStations()
    loadTemplates()
  }, [])

  // Auto-refresh connectors
  useEffect(() => {
    if (!showConnectors || !autoRefresh || !selectedStationId) {
      return
    }

    const intervalId = setInterval(() => {
      handleRefreshConnectors(true) // silent refresh
    }, refreshInterval * 1000)

    return () => clearInterval(intervalId)
  }, [showConnectors, autoRefresh, refreshInterval, selectedStationId])

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

  const loadTemplates = () => {
    const saved = localStorage.getItem('stationTemplates')
    if (saved) {
      try {
        setTemplates(JSON.parse(saved))
      } catch (e) {
        console.error('Failed to load templates:', e)
      }
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

  const handleDelete = async (station) => {
    const stationId = station.stationId || station
    if (!confirm(`Are you sure you want to delete station ${stationId}?`)) {
      return
    }

    try {
      await stationsAPI.delete(stationId)
      setShowForm(false)
      setEditingStation(null)
      fetchStations()
    } catch (err) {
      alert(`Failed to delete station: ${err.message}`)
    }
  }

  const handleCreateStation = () => {
    setEditingStation(null)
    setShowForm(true)
  }

  const handleEditStation = (station) => {
    setEditingStation(station)
    setShowForm(true)
  }

  const handleFormSubmit = async (formData) => {
    try {
      if (editingStation) {
        await stationsAPI.update(editingStation.stationId, formData)
      } else {
        await stationsAPI.create(formData)
      }
      setShowForm(false)
      setEditingStation(null)
      fetchStations()
    } catch (err) {
      alert(`Failed to save station: ${err.response?.data?.error || err.message}`)
    }
  }

  const handleFormCancel = () => {
    setShowForm(false)
    setEditingStation(null)
  }

  const handleTemplateSelect = (template) => {
    setEditingStation({
      ...template,
      stationId: '',
      name: `${template.name} (New)`,
      csmsUrl: '',
      enabled: true,
      autoStart: false
    })
    setShowTemplates(false)
    setShowForm(true)
  }

  const handleSaveAsTemplate = (station) => {
    const templateName = prompt('Enter template name:')
    if (!templateName) return

    const template = {
      name: templateName,
      description: `Template based on ${station.name}`,
      ...station,
      stationId: undefined,
      name: undefined,
      csmsUrl: undefined,
      enabled: undefined,
      autoStart: undefined,
      runtimeState: undefined,
      createdAt: undefined,
      updatedAt: undefined,
      _id: undefined,
      id: undefined
    }

    const existingTemplates = templates || []
    const newTemplates = [...existingTemplates, template]
    setTemplates(newTemplates)
    localStorage.setItem('stationTemplates', JSON.stringify(newTemplates))
    alert('Template saved successfully!')
  }

  const handleViewConnectors = async (stationId) => {
    setSelectedStationId(stationId)
    setShowConnectors(true)
    setConnectorsLoading(true)
    try {
      const response = await stationsAPI.getConnectors(stationId)
      const sortedConnectors = (response.data.connectors || []).sort((a, b) => a.id - b.id)
      setConnectors(sortedConnectors)
    } catch (err) {
      alert(`Failed to load connectors: ${err.message}`)
      setConnectors([])
    } finally {
      setConnectorsLoading(false)
    }
  }

  const handleRefreshConnectors = async (silent = false) => {
    if (!selectedStationId) return
    if (!silent) {
      setConnectorsLoading(true)
    }
    try {
      const response = await stationsAPI.getConnectors(selectedStationId)
      const sortedConnectors = (response.data.connectors || []).sort((a, b) => a.id - b.id)
      setConnectors(sortedConnectors)
    } catch (err) {
      console.error('Failed to refresh connectors:', err)
    } finally {
      if (!silent) {
        setConnectorsLoading(false)
      }
    }
  }

  const handleCloseConnectors = () => {
    setShowConnectors(false)
    setSelectedStationId(null)
    setConnectors([])
  }

  const handleConfigureStation = (station) => {
    setConfiguringStation(station)
    setShowConfig(true)
  }

  const handleSaveConfig = async (updatedConfig) => {
    try {
      await stationsAPI.update(updatedConfig.stationId, updatedConfig)
      setShowConfig(false)
      setConfiguringStation(null)
      fetchStations()
    } catch (err) {
      throw new Error(err.response?.data?.error || err.message)
    }
  }

  const handleCloseConfig = () => {
    setShowConfig(false)
    setConfiguringStation(null)
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
        <div className="header-actions">
          <button className="btn-secondary" onClick={() => setShowTemplates(true)}>
            📋 Templates
          </button>
          <button className="btn-secondary" onClick={() => setShowImportExport(true)}>
            📥📤 Import/Export
          </button>
          <button className="btn-primary" onClick={handleCreateStation}>
            + Add Station
          </button>
        </div>
      </div>

      {stations.length === 0 ? (
        <div className="empty-state">
          <h3>No stations configured</h3>
          <p>Get started by creating your first charging station</p>
          <div className="empty-state-actions">
            <button className="btn-primary" onClick={handleCreateStation}>
              Create New Station
            </button>
            <button className="btn-secondary" onClick={() => setShowTemplates(true)}>
              Use Template
            </button>
            <button className="btn-secondary" onClick={() => setShowImportExport(true)}>
              Import Stations
            </button>
          </div>
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
                  <span className="value station-id">{station.stationId}</span>
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
                  <span className="value">
                    {station.connectors?.length || 0}
                    {station.connectors && station.connectors.length > 0 && (
                      <span className="connector-types">
                        {' '}({station.connectors.map(c => c.type).join(', ')})
                      </span>
                    )}
                  </span>
                </div>
                <div className="info-row">
                  <span className="label">CSMS:</span>
                  <span className="value url">{station.csmsUrl || 'Not configured'}</span>
                </div>
                <div className="info-row">
                  <span className="label">Enabled:</span>
                  <span className="value">
                    <span className={`badge ${station.enabled ? 'enabled' : 'disabled'}`}>
                      {station.enabled ? 'Yes' : 'No'}
                    </span>
                  </span>
                </div>
                {station.tags && station.tags.length > 0 && (
                  <div className="info-row">
                    <span className="label">Tags:</span>
                    <span className="value tags">
                      {station.tags.map(tag => (
                        <span key={tag} className="tag">{tag}</span>
                      ))}
                    </span>
                  </div>
                )}
              </div>

              <div className="station-actions">
                {station.runtimeState?.connectionStatus === 'connected' ? (
                  <button
                    className="btn-action btn-stop"
                    onClick={() => handleStop(station.stationId)}
                  >
                    ⏹ Stop
                  </button>
                ) : (
                  <button
                    className="btn-action btn-start"
                    onClick={() => handleStart(station.stationId)}
                    disabled={!station.enabled}
                  >
                    ▶ Start
                  </button>
                )}
                <button
                  className="btn-action btn-connectors"
                  onClick={() => handleViewConnectors(station.stationId)}
                  title="Manage connectors"
                >
                  🔌 Connectors
                </button>
                <button
                  className="btn-action btn-config"
                  onClick={() => handleConfigureStation(station)}
                  title="Configure station settings"
                >
                  ⚙️ Configure
                </button>
                <button
                  className="btn-action btn-edit"
                  onClick={() => handleEditStation(station)}
                  title="Full configuration editor"
                >
                  ✏️ Edit
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showForm && (
        <StationForm
          station={editingStation}
          onSubmit={handleFormSubmit}
          onCancel={handleFormCancel}
          onDelete={editingStation ? handleDelete : undefined}
          onSaveAsTemplate={editingStation ? handleSaveAsTemplate : undefined}
          templates={templates}
        />
      )}

      {showTemplates && (
        <TemplatesManager
          onClose={() => setShowTemplates(false)}
          onSelectTemplate={handleTemplateSelect}
        />
      )}

      {showImportExport && (
        <ImportExport
          stations={stations}
          onClose={() => setShowImportExport(false)}
          onImportComplete={fetchStations}
        />
      )}

      {showConnectors && (
        <div className="modal-overlay" onClick={handleCloseConnectors}>
          <div className="modal-content connectors-modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>Connectors - {selectedStationId}</h2>
              <div className="modal-header-actions">
                <div className="auto-refresh-controls">
                  <label className="auto-refresh-toggle">
                    <input
                      type="checkbox"
                      checked={autoRefresh}
                      onChange={(e) => setAutoRefresh(e.target.checked)}
                    />
                    <span>Auto-refresh</span>
                    {autoRefresh && <span className="refresh-indicator">●</span>}
                  </label>
                  {autoRefresh && (
                    <select
                      className="refresh-interval-select"
                      value={refreshInterval}
                      onChange={(e) => setRefreshInterval(Number(e.target.value))}
                    >
                      <option value="3">3s</option>
                      <option value="5">5s</option>
                      <option value="10">10s</option>
                      <option value="30">30s</option>
                    </select>
                  )}
                </div>
                <button
                  className="btn-secondary"
                  onClick={() => handleRefreshConnectors(false)}
                  disabled={connectorsLoading}
                >
                  🔄 Refresh
                </button>
                <button className="btn-close" onClick={handleCloseConnectors}>
                  ✕
                </button>
              </div>
            </div>
            <div className="modal-body">
              {connectorsLoading ? (
                <div className="loading">Loading connectors...</div>
              ) : connectors.length === 0 ? (
                <div className="empty-state">
                  <p>No connectors configured for this station</p>
                </div>
              ) : (
                <div className="connectors-grid">
                  {connectors.map((connector) => {
                    const station = stations.find(s => s.stationId === selectedStationId)
                    const isConnected = station?.runtimeState?.connectionStatus === 'connected'
                    return (
                      <ConnectorCard
                        key={connector.id}
                        stationId={selectedStationId}
                        connector={connector}
                        isStationConnected={isConnected}
                        onUpdate={() => handleRefreshConnectors(true)}
                      />
                    )
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {showConfig && configuringStation && (
        <StationConfig
          station={configuringStation}
          onSave={handleSaveConfig}
          onClose={handleCloseConfig}
        />
      )}
    </div>
  )
}

export default Stations
