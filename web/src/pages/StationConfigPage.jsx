import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { stationsAPI } from '../services/api'
import { useAuth } from '../contexts/AuthContext'
import './StationConfigPage.css'

const PROTOCOL_VERSIONS = ['ocpp1.6', 'ocpp2.0.1', 'ocpp2.1']

function StationConfigPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { isAdmin } = useAuth()

  const [activeTab, setActiveTab] = useState('general')
  const [config, setConfig] = useState(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchStation()
  }, [id])

  const fetchStation = async () => {
    try {
      setLoading(true)
      const response = await stationsAPI.getById(id)
      setConfig(response.data)
    } catch (err) {
      setError(`Failed to load station: ${err.message}`)
    } finally {
      setLoading(false)
    }
  }

  const handleChange = (field, value) => {
    setConfig(prev => ({ ...prev, [field]: value }))
  }

  const handleNestedChange = (parent, field, value) => {
    setConfig(prev => ({
      ...prev,
      [parent]: { ...prev[parent], [field]: value }
    }))
  }

  const handleSave = async () => {
    try {
      setSaving(true)
      setError(null)
      await stationsAPI.update(id, config)
      navigate('/stations')
    } catch (err) {
      setError(`Failed to save configuration: ${err.message}`)
      setSaving(false)
    }
  }

  if (!isAdmin) {
    return (
      <div className="station-config-page">
        <div className="error-message">You do not have permission to access this page.</div>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="station-config-page">
        <div className="loading">Loading station...</div>
      </div>
    )
  }

  if (!config) {
    return (
      <div className="station-config-page">
        <div className="error-message">Station not found</div>
      </div>
    )
  }

  const tabs = [
    { id: 'general', label: 'General' },
    { id: 'connection', label: 'Connection' },
    { id: 'simulation', label: 'Simulation' },
    { id: 'advanced', label: 'Advanced' }
  ]

  return (
    <div className="station-config-page">
      <div className="page-header">
        <nav className="breadcrumb">
          <Link to="/stations">Stations</Link>
          <span className="breadcrumb-separator">/</span>
          <span>Configure</span>
        </nav>
        <div className="page-header__row">
          <div>
            <h2>Quick Configuration</h2>
            <p className="page-subtitle">{config.name} ({config.stationId})</p>
          </div>
          <Link to={`/stations/${id}/edit`} className="btn btn--sm btn--secondary">
            Full Edit
          </Link>
        </div>
      </div>

      {error && <div className="error-message">{error}</div>}

      <div className="config-container">
        <div className="config-tabs">
          {tabs.map(tab => (
            <button
              key={tab.id}
              className={`tab-btn ${activeTab === tab.id ? 'active' : ''}`}
              onClick={() => setActiveTab(tab.id)}
            >
              {tab.label}
            </button>
          ))}
        </div>

        <div className="config-body">
          {/* General Tab */}
          {activeTab === 'general' && (
            <div className="config-tab">
              <div className="config-section">
                <h3>Basic Information</h3>
                <div className="config-grid">
                  <div className="config-field">
                    <label>Station Name</label>
                    <input
                      type="text"
                      value={config.name || ''}
                      onChange={(e) => handleChange('name', e.target.value)}
                    />
                  </div>

                  <div className="config-field">
                    <label>Station ID</label>
                    <input
                      type="text"
                      value={config.stationId || ''}
                      disabled
                    />
                    <small>Cannot be changed</small>
                  </div>

                  <div className="config-field">
                    <label>Protocol Version</label>
                    <select
                      value={config.protocolVersion || 'ocpp1.6'}
                      onChange={(e) => handleChange('protocolVersion', e.target.value)}
                    >
                      {PROTOCOL_VERSIONS.map(v => (
                        <option key={v} value={v}>{v.toUpperCase()}</option>
                      ))}
                    </select>
                  </div>

                  <div className="config-field config-field--checkboxes">
                    <label className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={config.enabled || false}
                        onChange={(e) => handleChange('enabled', e.target.checked)}
                      />
                      Enabled
                    </label>
                    <label className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={config.autoStart || false}
                        onChange={(e) => handleChange('autoStart', e.target.checked)}
                      />
                      Auto Start
                    </label>
                  </div>
                </div>
              </div>

              <div className="config-section">
                <h3>Device Information</h3>
                <div className="config-grid">
                  <div className="config-field">
                    <label>Vendor</label>
                    <input
                      type="text"
                      value={config.vendor || ''}
                      onChange={(e) => handleChange('vendor', e.target.value)}
                    />
                  </div>

                  <div className="config-field">
                    <label>Model</label>
                    <input
                      type="text"
                      value={config.model || ''}
                      onChange={(e) => handleChange('model', e.target.value)}
                    />
                  </div>

                  <div className="config-field">
                    <label>Serial Number</label>
                    <input
                      type="text"
                      value={config.serialNumber || ''}
                      onChange={(e) => handleChange('serialNumber', e.target.value)}
                    />
                  </div>

                  <div className="config-field">
                    <label>Firmware Version</label>
                    <input
                      type="text"
                      value={config.firmwareVersion || ''}
                      onChange={(e) => handleChange('firmwareVersion', e.target.value)}
                    />
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Connection Tab */}
          {activeTab === 'connection' && (
            <div className="config-tab">
              <div className="config-section">
                <h3>CSMS Connection</h3>
                <div className="config-grid">
                  <div className="config-field config-field--wide">
                    <label>CSMS URL</label>
                    <input
                      type="text"
                      value={config.csmsUrl || ''}
                      onChange={(e) => handleChange('csmsUrl', e.target.value)}
                      placeholder="ws://localhost:8080/ocpp"
                    />
                    <small>WebSocket URL of the Central System</small>
                  </div>

                  <div className="config-field">
                    <label>Authentication Type</label>
                    <select
                      value={config.csmsAuth?.type || 'none'}
                      onChange={(e) => handleNestedChange('csmsAuth', 'type', e.target.value)}
                    >
                      <option value="none">None</option>
                      <option value="basic">Basic Auth</option>
                    </select>
                  </div>

                  {config.csmsAuth?.type === 'basic' && (
                    <>
                      <div className="config-field">
                        <label>Username</label>
                        <input
                          type="text"
                          value={config.csmsAuth?.username || ''}
                          onChange={(e) => handleNestedChange('csmsAuth', 'username', e.target.value)}
                        />
                      </div>

                      <div className="config-field">
                        <label>Password</label>
                        <input
                          type="password"
                          value={config.csmsAuth?.password || ''}
                          onChange={(e) => handleNestedChange('csmsAuth', 'password', e.target.value)}
                        />
                      </div>
                    </>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Simulation Tab */}
          {activeTab === 'simulation' && (
            <div className="config-tab">
              <div className="config-section">
                <h3>Timing</h3>
                <div className="config-grid">
                  <div className="config-field">
                    <label>Boot Delay (sec)</label>
                    <input
                      type="number"
                      value={config.simulation?.bootDelay || 0}
                      onChange={(e) => handleNestedChange('simulation', 'bootDelay', parseInt(e.target.value))}
                      min="0"
                    />
                    <small>Delay before BootNotification</small>
                  </div>

                  <div className="config-field">
                    <label>Heartbeat Interval (sec)</label>
                    <input
                      type="number"
                      value={config.simulation?.heartbeatInterval || 300}
                      onChange={(e) => handleNestedChange('simulation', 'heartbeatInterval', parseInt(e.target.value))}
                      min="1"
                    />
                  </div>
                </div>
              </div>

              <div className="config-section">
                <h3>Charging Simulation</h3>
                <div className="config-grid">
                  <div className="config-field">
                    <label>Default ID Tag</label>
                    <input
                      type="text"
                      value={config.simulation?.defaultIdTag || 'DEFAULT_TAG'}
                      onChange={(e) => handleNestedChange('simulation', 'defaultIdTag', e.target.value)}
                    />
                  </div>

                  <div className="config-field">
                    <label>Energy Rate (W)</label>
                    <input
                      type="number"
                      value={config.simulation?.energyDeliveryRate || 7000}
                      onChange={(e) => handleNestedChange('simulation', 'energyDeliveryRate', parseInt(e.target.value))}
                      min="0"
                    />
                  </div>

                  <div className="config-field">
                    <label>Meter Variance (0-1)</label>
                    <input
                      type="number"
                      step="0.01"
                      value={config.simulation?.meterValueVariance || 0.1}
                      onChange={(e) => handleNestedChange('simulation', 'meterValueVariance', parseFloat(e.target.value))}
                      min="0"
                      max="1"
                    />
                  </div>

                  <div className="config-field config-field--checkboxes">
                    <label className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={config.simulation?.randomizeMeterValues || false}
                        onChange={(e) => handleNestedChange('simulation', 'randomizeMeterValues', e.target.checked)}
                      />
                      Randomize Meters
                    </label>
                    <label className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={config.simulation?.statusNotificationOnChange ?? true}
                        onChange={(e) => handleNestedChange('simulation', 'statusNotificationOnChange', e.target.checked)}
                      />
                      Status on Change
                    </label>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Advanced Tab */}
          {activeTab === 'advanced' && (
            <div className="config-tab">
              <div className="config-section">
                <h3>Meter Values</h3>
                <div className="config-grid">
                  <div className="config-field">
                    <label>Interval (sec)</label>
                    <input
                      type="number"
                      value={config.meterValuesConfig?.interval || 60}
                      onChange={(e) => handleNestedChange('meterValuesConfig', 'interval', parseInt(e.target.value))}
                      min="1"
                    />
                  </div>

                  <div className="config-field">
                    <label>Aligned Interval (sec)</label>
                    <input
                      type="number"
                      value={config.meterValuesConfig?.alignedDataInterval || 900}
                      onChange={(e) => handleNestedChange('meterValuesConfig', 'alignedDataInterval', parseInt(e.target.value))}
                      min="1"
                    />
                  </div>
                </div>
              </div>

              <div className="config-section">
                <h3>Hardware Identifiers</h3>
                <div className="config-grid">
                  <div className="config-field">
                    <label>ICCID (SIM)</label>
                    <input
                      type="text"
                      value={config.iccid || ''}
                      onChange={(e) => handleChange('iccid', e.target.value)}
                    />
                  </div>

                  <div className="config-field">
                    <label>IMSI</label>
                    <input
                      type="text"
                      value={config.imsi || ''}
                      onChange={(e) => handleChange('imsi', e.target.value)}
                    />
                  </div>
                </div>
              </div>

              <div className="config-section">
                <h3>Connectors</h3>
                <p className="config-note">
                  Connector configuration is managed through the full edit page.
                </p>
                <Link to={`/stations/${id}/edit`} className="btn btn--sm btn--secondary">
                  Edit Connectors
                </Link>
              </div>
            </div>
          )}
        </div>

        <div className="config-actions">
          <Link to="/stations" className="btn btn--secondary">
            Cancel
          </Link>
          <button className="btn btn--primary" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving...' : 'Save Configuration'}
          </button>
        </div>
      </div>
    </div>
  )
}

export default StationConfigPage
