import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { stationsAPI } from '../services/api'
import { useAuth } from '../contexts/AuthContext'
import TemplatesManager from '../components/TemplatesManager'
import ImportExport from '../components/ImportExport'
import ConnectorCard from '../components/ConnectorCard'
import './Stations.css'

function Stations() {
  const { isAdmin } = useAuth()
  const [stations, setStations] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showTemplates, setShowTemplates] = useState(false)
  const [showImportExport, setShowImportExport] = useState(false)
  const [templates, setTemplates] = useState([])
  const [expandedStations, setExpandedStations] = useState(new Set())
  const [connectorsMap, setConnectorsMap] = useState({})
  const [connectorsLoading, setConnectorsLoading] = useState({})
  const [sortBy, setSortBy] = useState(() => localStorage.getItem('stationsSortBy') || null)
  const [sortOrder, setSortOrder] = useState(() => localStorage.getItem('stationsSortOrder') || 'asc')

  useEffect(() => {
    fetchStations()
    loadTemplates()
  }, [])

  useEffect(() => {
    if (expandedStations.size === 0) return

    const intervalId = setInterval(() => {
      expandedStations.forEach(stationId => {
        refreshConnectors(stationId, true)
      })
    }, 5000)

    return () => clearInterval(intervalId)
  }, [expandedStations])

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

  const handleStart = async (stationId, e) => {
    e?.stopPropagation()
    try {
      await stationsAPI.start(stationId)
      fetchStations()
    } catch (err) {
      alert(`Failed to start station: ${err.message}`)
    }
  }

  const handleStop = async (stationId, e) => {
    e?.stopPropagation()
    try {
      await stationsAPI.stop(stationId)
      fetchStations()
    } catch (err) {
      alert(`Failed to stop station: ${err.message}`)
    }
  }

  const handleTemplateSelect = (template) => {
    setShowTemplates(false)
    window.location.href = `/stations/new?template=${encodeURIComponent(template.name)}`
  }

  const toggleExpanded = async (stationId) => {
    const newExpanded = new Set(expandedStations)
    if (newExpanded.has(stationId)) {
      newExpanded.delete(stationId)
    } else {
      newExpanded.add(stationId)
      // Always refresh connectors when expanding
      await loadConnectors(stationId)
    }
    setExpandedStations(newExpanded)
  }

  const loadConnectors = async (stationId) => {
    setConnectorsLoading(prev => ({ ...prev, [stationId]: true }))
    try {
      const response = await stationsAPI.getConnectors(stationId)
      const sortedConnectors = (response.data.connectors || []).sort((a, b) => a.id - b.id)
      setConnectorsMap(prev => ({ ...prev, [stationId]: sortedConnectors }))
    } catch (err) {
      console.error(`Failed to load connectors for ${stationId}:`, err)
      setConnectorsMap(prev => ({ ...prev, [stationId]: [] }))
    } finally {
      setConnectorsLoading(prev => ({ ...prev, [stationId]: false }))
    }
  }

  const refreshConnectors = async (stationId, silent = false) => {
    if (!silent) {
      setConnectorsLoading(prev => ({ ...prev, [stationId]: true }))
    }
    try {
      const response = await stationsAPI.getConnectors(stationId)
      const sortedConnectors = (response.data.connectors || []).sort((a, b) => a.id - b.id)
      setConnectorsMap(prev => ({ ...prev, [stationId]: sortedConnectors }))
    } catch (err) {
      console.error(`Failed to refresh connectors for ${stationId}:`, err)
    } finally {
      if (!silent) {
        setConnectorsLoading(prev => ({ ...prev, [stationId]: false }))
      }
    }
  }

  const handleSort = (field) => {
    if (sortBy === field) {
      if (sortOrder === 'asc') {
        setSortOrder('desc')
        localStorage.setItem('stationsSortOrder', 'desc')
      } else {
        setSortBy(null)
        setSortOrder('asc')
        localStorage.removeItem('stationsSortBy')
        localStorage.setItem('stationsSortOrder', 'asc')
      }
    } else {
      setSortBy(field)
      setSortOrder('asc')
      localStorage.setItem('stationsSortBy', field)
      localStorage.setItem('stationsSortOrder', 'asc')
    }
  }

  const getSortedStations = () => {
    if (!sortBy) return stations

    return [...stations].sort((a, b) => {
      let aValue = a[sortBy]
      let bValue = b[sortBy]

      if (typeof aValue === 'string') aValue = aValue.toLowerCase()
      if (typeof bValue === 'string') bValue = bValue.toLowerCase()

      if (aValue < bValue) return sortOrder === 'asc' ? -1 : 1
      if (aValue > bValue) return sortOrder === 'asc' ? 1 : -1
      return 0
    })
  }

  const getProtocolClass = (version) => {
    if (version === 'ocpp2.1' || version === '2.1') return 'ocpp21'
    if (version === 'ocpp2.0.1' || version === '2.0.1') return 'ocpp201'
    return 'ocpp16'
  }

  if (loading) {
    return <div className="stations"><div className="loading">Loading stations...</div></div>
  }

  if (error) {
    return <div className="stations"><div className="error">Error loading stations: {error}</div></div>
  }

  const sortedStations = getSortedStations()

  return (
    <div className="stations">
      <div className="page-header">
        <div className="page-header__title">
          <h2>Charging Stations</h2>
          <span className="station-count">{stations.length} station{stations.length !== 1 ? 's' : ''}</span>
        </div>
        <div className="header-actions">
          {isAdmin && (
            <>
              <button className="btn btn--sm btn--secondary" onClick={() => setShowTemplates(true)}>
                Templates
              </button>
              <button className="btn btn--sm btn--secondary" onClick={() => setShowImportExport(true)}>
                Import/Export
              </button>
              <Link to="/stations/new" className="btn btn--sm btn--primary">
                + Add Station
              </Link>
            </>
          )}
        </div>
      </div>

      {stations.length === 0 ? (
        <div className="empty-state">
          <h3>No stations configured</h3>
          <p>{isAdmin ? 'Get started by creating your first charging station' : 'No stations available to view'}</p>
          {isAdmin && (
            <div className="empty-state-actions">
              <Link to="/stations/new" className="btn btn-primary">
                Create New Station
              </Link>
              <button className="btn btn-secondary" onClick={() => setShowTemplates(true)}>
                Use Template
              </button>
              <button className="btn btn-secondary" onClick={() => setShowImportExport(true)}>
                Import Stations
              </button>
            </div>
          )}
        </div>
      ) : (
        <div className="stations-list">
          {sortedStations.map((station) => {
            const isExpanded = expandedStations.has(station.stationId)
            const stationConnectors = connectorsMap[station.stationId] || []
            const isLoadingConnectors = connectorsLoading[station.stationId]
            const isConnected = station.runtimeState?.connectionStatus === 'connected'

            return (
              <div key={station.stationId} className={`station-card ${isExpanded ? 'expanded' : ''}`}>
                <div className="station-card__main" onClick={() => toggleExpanded(station.stationId)}>
                  <div className="station-card__data">
                    <div className="station-card__expand">
                      <span className={`expand-icon ${isExpanded ? 'expanded' : ''}`}>
                        <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor">
                          <path d="M4 2l4 4-4 4" />
                        </svg>
                      </span>
                    </div>

                    <div className="station-card__status">
                      <span
                        className={`status-dot ${station.runtimeState?.connectionStatus || 'unknown'}`}
                        title={station.runtimeState?.connectionStatus || 'unknown'}
                      />
                    </div>

                    <div className="station-card__field station-card__protocol">
                      {/*<span className="field-label">Protocol</span>*/}
                      <span className={`protocol-badge ${getProtocolClass(station.protocolVersion)}`}>
                        {station.protocolVersion?.toUpperCase() || 'OCPP1.6'}
                      </span>
                    </div>

                    <div className="station-card__field station-card__name">
                      <span className="field-label">Name</span>
                      <span className="field-value">{station.name}</span>
                    </div>

                    <div className="station-card__field station-card__id">
                      <span className="field-label">Station ID</span>
                      <span className="field-value field-value--mono">{station.stationId}</span>
                    </div>

                    <div className="station-card__field station-card__vendor">
                      <span className="field-label">Vendor / Model</span>
                      <span className="field-value">{station.vendor} / {station.model}</span>
                    </div>

                    <div className="station-card__field station-card__connectors-count">
                      <span className="field-label">Connectors</span>
                      <span className="field-value">{station.connectors?.length || 0}</span>
                    </div>
                  </div>

                  <div className="station-card__actions">
                    {isConnected ? (
                      <button
                        className="btn btn--sm btn-stop"
                        onClick={(e) => handleStop(station.stationId, e)}
                      >
                        Stop
                      </button>
                    ) : (
                      <button
                        className="btn btn--sm btn-start"
                        onClick={(e) => handleStart(station.stationId, e)}
                        disabled={!station.enabled}
                      >
                        Start
                      </button>
                    )}
                    {isAdmin && (
                      <Link
                        to={`/stations/${station.stationId}/edit`}
                        className="btn btn--sm btn-edit"
                        onClick={(e) => e.stopPropagation()}
                      >
                        Edit
                      </Link>
                    )}
                  </div>
                </div>

                {isExpanded && (
                  <div className="station-card__connectors">
                    <div className="connectors-header">
                      <span className="connectors-title">Connectors</span>
                      <button
                        className="btn btn--xs btn--secondary"
                        onClick={(e) => {
                          e.stopPropagation()
                          refreshConnectors(station.stationId, false)
                        }}
                        disabled={isLoadingConnectors}
                      >
                        Refresh
                      </button>
                    </div>
                    {isLoadingConnectors ? (
                      <div className="connectors-loading">Loading connectors...</div>
                    ) : stationConnectors.length === 0 ? (
                      <div className="connectors-empty">No connectors configured</div>
                    ) : (
                      <div className="connectors-scroll">
                        {stationConnectors.map((connector) => (
                          <ConnectorCard
                            key={connector.id}
                            stationId={station.stationId}
                            connector={connector}
                            isStationConnected={isConnected}
                            onUpdate={() => refreshConnectors(station.stationId, true)}
                          />
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
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
    </div>
  )
}

export default Stations
