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
  const [viewMode, setViewMode] = useState(() => localStorage.getItem('stationsViewMode') || 'list')

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

  const handleViewModeChange = (mode) => {
    setViewMode(mode)
    localStorage.setItem('stationsViewMode', mode)
  }

  const getProtocolClass = (version) => {
    if (version === 'ocpp2.1' || version === '2.1') return 'ocpp21'
    if (version === 'ocpp2.0.1' || version === '2.0.1') return 'ocpp201'
    return 'ocpp16'
  }

  const renderStationCard = (station, isHorizontal = false) => (
    <div key={station.stationId} className={`station-card ${isHorizontal ? 'station-card--horizontal' : ''}`}>
      <div className="station-card__main">
        <div className="station-header">
          <div className="station-header__info">
            <h3>{station.name}</h3>
            <span className="station-id-inline">{station.stationId}</span>
          </div>
          <span className={`status-badge ${station.runtimeState?.connectionStatus}`}>
            {station.runtimeState?.connectionStatus || 'unknown'}
          </span>
        </div>

        <div className="station-info">
          <div className="station-info__grid">
            <div className="info-cell">
              <span className="info-cell__label">Vendor</span>
              <span className="info-cell__value">{station.vendor}</span>
            </div>
            <div className="info-cell">
              <span className="info-cell__label">Model</span>
              <span className="info-cell__value">{station.model}</span>
            </div>
            <div className="info-cell">
              <span className="info-cell__label">Protocol</span>
              <span className={`protocol-badge ${getProtocolClass(station.protocolVersion)}`}>
                {station.protocolVersion?.toUpperCase() || 'OCPP1.6'}
              </span>
            </div>
            <div className="info-cell">
              <span className="info-cell__label">Connectors</span>
              <span className="info-cell__value">{station.connectors?.length || 0}</span>
            </div>
          </div>
          <div className="station-url">
            <span className="info-cell__label">CSMS:</span>
            <span className="info-cell__value url">{station.csmsUrl || 'Not configured'}</span>
          </div>
        </div>
      </div>

      <div className="station-actions">
        {station.runtimeState?.connectionStatus === 'connected' ? (
          <button className="btn-action btn-stop" onClick={() => handleStop(station.stationId)}>
            Stop
          </button>
        ) : (
          <button
            className="btn-action btn-start"
            onClick={() => handleStart(station.stationId)}
            disabled={!station.enabled}
          >
            Start
          </button>
        )}
        <button className="btn-action btn-connectors" onClick={() => handleViewConnectors(station.stationId)}>
          Connectors
        </button>
        <button className="btn-action btn-config" onClick={() => handleConfigureStation(station)}>
          Configure
        </button>
        <button className="btn-action btn-edit" onClick={() => handleEditStation(station)}>
          Edit
        </button>
      </div>
    </div>
  )

  const renderTableView = () => (
    <div className="stations-table-wrapper">
      <table className="stations-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Station ID</th>
            <th>Status</th>
            <th>Protocol</th>
            <th>Vendor / Model</th>
            <th>Connectors</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {stations.map((station) => (
            <tr key={station.stationId}>
              <td className="cell-name">{station.name}</td>
              <td className="cell-id">{station.stationId}</td>
              <td>
                <span className={`status-badge status-badge--sm ${station.runtimeState?.connectionStatus}`}>
                  {station.runtimeState?.connectionStatus || 'unknown'}
                </span>
              </td>
              <td>
                <span className={`protocol-badge ${getProtocolClass(station.protocolVersion)}`}>
                  {station.protocolVersion?.toUpperCase() || 'OCPP1.6'}
                </span>
              </td>
              <td className="cell-vendor">{station.vendor} / {station.model}</td>
              <td className="cell-connectors">{station.connectors?.length || 0}</td>
              <td className="cell-actions">
                {station.runtimeState?.connectionStatus === 'connected' ? (
                  <button className="btn-action btn-action--sm btn-stop" onClick={() => handleStop(station.stationId)}>
                    Stop
                  </button>
                ) : (
                  <button
                    className="btn-action btn-action--sm btn-start"
                    onClick={() => handleStart(station.stationId)}
                    disabled={!station.enabled}
                  >
                    Start
                  </button>
                )}
                <button className="btn-action btn-action--sm btn-connectors" onClick={() => handleViewConnectors(station.stationId)}>
                  Connectors
                </button>
                <button className="btn-action btn-action--sm btn-config" onClick={() => handleConfigureStation(station)}>
                  Config
                </button>
                <button className="btn-action btn-action--sm btn-edit" onClick={() => handleEditStation(station)}>
                  Edit
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )

  if (loading) {
    return <div className="loading">Loading stations...</div>
  }

  if (error) {
    return <div className="error">Error loading stations: {error}</div>
  }

  return (
    <div className="stations">
      <div className="page-header">
        <div className="page-header__title">
          <h2>Charging Stations</h2>
          <span className="station-count">{stations.length} station{stations.length !== 1 ? 's' : ''}</span>
        </div>
        <div className="header-actions">
          {stations.length > 0 && (
            <div className="view-toggle">
              <button
                className={`view-toggle__btn ${viewMode === 'list' ? 'active' : ''}`}
                onClick={() => handleViewModeChange('list')}
                title="List view"
              >
                <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                  <rect x="1" y="2" width="14" height="3" rx="1"/>
                  <rect x="1" y="7" width="14" height="3" rx="1"/>
                  <rect x="1" y="12" width="14" height="2" rx="1"/>
                </svg>
              </button>
              <button
                className={`view-toggle__btn ${viewMode === 'grid' ? 'active' : ''}`}
                onClick={() => handleViewModeChange('grid')}
                title="Grid view"
              >
                <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                  <rect x="1" y="1" width="6" height="6" rx="1"/>
                  <rect x="9" y="1" width="6" height="6" rx="1"/>
                  <rect x="1" y="9" width="6" height="6" rx="1"/>
                  <rect x="9" y="9" width="6" height="6" rx="1"/>
                </svg>
              </button>
              <button
                className={`view-toggle__btn ${viewMode === 'table' ? 'active' : ''}`}
                onClick={() => handleViewModeChange('table')}
                title="Table view"
              >
                <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                  <rect x="1" y="1" width="14" height="2"/>
                  <rect x="1" y="5" width="5" height="2"/>
                  <rect x="8" y="5" width="7" height="2"/>
                  <rect x="1" y="9" width="5" height="2"/>
                  <rect x="8" y="9" width="7" height="2"/>
                  <rect x="1" y="13" width="5" height="2"/>
                  <rect x="8" y="13" width="7" height="2"/>
                </svg>
              </button>
            </div>
          )}
          <button className="btn-secondary btn-secondary--sm" onClick={() => setShowTemplates(true)}>
            Templates
          </button>
          <button className="btn-secondary btn-secondary--sm" onClick={() => setShowImportExport(true)}>
            Import/Export
          </button>
          <button className="btn-primary btn-primary--sm" onClick={handleCreateStation}>
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
      ) : viewMode === 'table' ? (
        renderTableView()
      ) : (
        <div className={`stations-${viewMode === 'grid' ? 'grid' : 'list'}`}>
          {stations.map((station) => renderStationCard(station, viewMode === 'list'))}
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
