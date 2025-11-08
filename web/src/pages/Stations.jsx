import { useState, useEffect } from 'react'
import { stationsAPI } from '../services/api'
import StationForm from '../components/StationForm'
import TemplatesManager from '../components/TemplatesManager'
import ImportExport from '../components/ImportExport'
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

  useEffect(() => {
    fetchStations()
    loadTemplates()
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
                  className="btn-action btn-edit"
                  onClick={() => handleEditStation(station)}
                >
                  ✏️ Edit
                </button>
                <button
                  className="btn-action btn-template"
                  onClick={() => handleSaveAsTemplate(station)}
                  title="Save as template"
                >
                  📋
                </button>
                <button
                  className="btn-action btn-delete"
                  onClick={() => handleDelete(station.stationId)}
                >
                  🗑️ Delete
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
    </div>
  )
}

export default Stations
