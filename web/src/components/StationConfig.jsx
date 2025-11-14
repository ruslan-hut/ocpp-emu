import { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import './StationConfig.css'

const PROTOCOL_VERSIONS = ['ocpp1.6', 'ocpp2.0.1', 'ocpp2.1']

function StationConfig({ station, onSave, onClose }) {
  const [activeTab, setActiveTab] = useState('general')
  const [config, setConfig] = useState(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (station) {
      setConfig({ ...station })
    }
  }, [station])

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
      await onSave(config)
      onClose()
    } catch (err) {
      alert(`Failed to save configuration: ${err.message}`)
    } finally {
      setSaving(false)
    }
  }

  if (!config) {
    return null
  }

  return (
    <div className="station-config-overlay" onClick={onClose}>
      <div className="station-config-modal" onClick={(e) => e.stopPropagation()}>
        <div className="station-config-header">
          <h2>⚙️ Station Configuration</h2>
          <p className="config-subtitle">{config.name} ({config.stationId})</p>
          <button className="btn-close" onClick={onClose}>×</button>
        </div>

        <div className="station-config-tabs">
          <button
            className={`tab-btn ${activeTab === 'general' ? 'active' : ''}`}
            onClick={() => setActiveTab('general')}
          >
            General
          </button>
          <button
            className={`tab-btn ${activeTab === 'connection' ? 'active' : ''}`}
            onClick={() => setActiveTab('connection')}
          >
            Connection
          </button>
          <button
            className={`tab-btn ${activeTab === 'simulation' ? 'active' : ''}`}
            onClick={() => setActiveTab('simulation')}
          >
            Simulation
          </button>
          <button
            className={`tab-btn ${activeTab === 'advanced' ? 'active' : ''}`}
            onClick={() => setActiveTab('advanced')}
          >
            Advanced
          </button>
        </div>

        <div className="station-config-body">
          {/* General Tab */}
          {activeTab === 'general' && (
            <div className="config-tab">
              <div className="config-section">
                <h3>Basic Information</h3>

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
                    className="disabled-input"
                  />
                  <small>Station ID cannot be changed</small>
                </div>

                <div className="config-field">
                  <label>Protocol Version</label>
                  <select
                    value={config.protocolVersion || 'ocpp1.6'}
                    onChange={(e) => handleChange('protocolVersion', e.target.value)}
                  >
                    {PROTOCOL_VERSIONS.map(version => (
                      <option key={version} value={version}>{version.toUpperCase()}</option>
                    ))}
                  </select>
                </div>

                <div className="config-field checkbox-field">
                  <label>
                    <input
                      type="checkbox"
                      checked={config.enabled || false}
                      onChange={(e) => handleChange('enabled', e.target.checked)}
                    />
                    <span>Station Enabled</span>
                  </label>
                  <small>Disabled stations cannot be started</small>
                </div>

                <div className="config-field checkbox-field">
                  <label>
                    <input
                      type="checkbox"
                      checked={config.autoStart || false}
                      onChange={(e) => handleChange('autoStart', e.target.checked)}
                    />
                    <span>Auto-start on Application Launch</span>
                  </label>
                </div>
              </div>

              <div className="config-section">
                <h3>Device Information</h3>

                <div className="config-field">
                  <label>Vendor</label>
                  <input
                    type="text"
                    value={config.vendor || ''}
                    onChange={(e) => handleChange('vendor', e.target.value)}
                    placeholder="e.g., ChargePoint Vendor"
                  />
                </div>

                <div className="config-field">
                  <label>Model</label>
                  <input
                    type="text"
                    value={config.model || ''}
                    onChange={(e) => handleChange('model', e.target.value)}
                    placeholder="e.g., CP-500"
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
                    placeholder="e.g., 1.0.0"
                  />
                </div>
              </div>
            </div>
          )}

          {/* Connection Tab */}
          {activeTab === 'connection' && (
            <div className="config-tab">
              <div className="config-section">
                <h3>CSMS Connection</h3>

                <div className="config-field">
                  <label>CSMS URL</label>
                  <input
                    type="text"
                    value={config.csmsUrl || ''}
                    onChange={(e) => handleChange('csmsUrl', e.target.value)}
                    placeholder="ws://localhost:8080/ocpp or wss://csms.example.com/ocpp"
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
          )}

          {/* Simulation Tab */}
          {activeTab === 'simulation' && (
            <div className="config-tab">
              <div className="config-section">
                <h3>Timing Settings</h3>

                <div className="config-field">
                  <label>Boot Delay (seconds)</label>
                  <input
                    type="number"
                    value={config.simulation?.bootDelay || 0}
                    onChange={(e) => handleNestedChange('simulation', 'bootDelay', parseInt(e.target.value))}
                    min="0"
                  />
                  <small>Delay before sending BootNotification</small>
                </div>

                <div className="config-field">
                  <label>Heartbeat Interval (seconds)</label>
                  <input
                    type="number"
                    value={config.simulation?.heartbeatInterval || 300}
                    onChange={(e) => handleNestedChange('simulation', 'heartbeatInterval', parseInt(e.target.value))}
                    min="1"
                  />
                </div>
              </div>

              <div className="config-section">
                <h3>Charging Simulation</h3>

                <div className="config-field">
                  <label>Default ID Tag</label>
                  <input
                    type="text"
                    value={config.simulation?.defaultIdTag || 'DEFAULT_TAG'}
                    onChange={(e) => handleNestedChange('simulation', 'defaultIdTag', e.target.value)}
                  />
                  <small>Default RFID tag for charging sessions</small>
                </div>

                <div className="config-field">
                  <label>Energy Delivery Rate (Wh/s)</label>
                  <input
                    type="number"
                    value={config.simulation?.energyDeliveryRate || 7000}
                    onChange={(e) => handleNestedChange('simulation', 'energyDeliveryRate', parseInt(e.target.value))}
                    min="0"
                  />
                  <small>Simulated power consumption rate</small>
                </div>

                <div className="config-field checkbox-field">
                  <label>
                    <input
                      type="checkbox"
                      checked={config.simulation?.randomizeMeterValues || false}
                      onChange={(e) => handleNestedChange('simulation', 'randomizeMeterValues', e.target.checked)}
                    />
                    <span>Randomize Meter Values</span>
                  </label>
                  <small>Add variance to simulated readings</small>
                </div>

                {config.simulation?.randomizeMeterValues && (
                  <div className="config-field">
                    <label>Meter Value Variance (0.0 - 1.0)</label>
                    <input
                      type="number"
                      step="0.01"
                      value={config.simulation?.meterValueVariance || 0.1}
                      onChange={(e) => handleNestedChange('simulation', 'meterValueVariance', parseFloat(e.target.value))}
                      min="0"
                      max="1"
                    />
                  </div>
                )}

                <div className="config-field checkbox-field">
                  <label>
                    <input
                      type="checkbox"
                      checked={config.simulation?.statusNotificationOnChange || true}
                      onChange={(e) => handleNestedChange('simulation', 'statusNotificationOnChange', e.target.checked)}
                    />
                    <span>Send StatusNotification on Change</span>
                  </label>
                </div>
              </div>
            </div>
          )}

          {/* Advanced Tab */}
          {activeTab === 'advanced' && (
            <div className="config-tab">
              <div className="config-section">
                <h3>Meter Values Configuration</h3>

                <div className="config-field">
                  <label>Meter Values Interval (seconds)</label>
                  <input
                    type="number"
                    value={config.meterValuesConfig?.interval || 60}
                    onChange={(e) => handleNestedChange('meterValuesConfig', 'interval', parseInt(e.target.value))}
                    min="1"
                  />
                </div>

                <div className="config-field">
                  <label>Aligned Data Interval (seconds)</label>
                  <input
                    type="number"
                    value={config.meterValuesConfig?.alignedDataInterval || 900}
                    onChange={(e) => handleNestedChange('meterValuesConfig', 'alignedDataInterval', parseInt(e.target.value))}
                    min="1"
                  />
                </div>
              </div>

              <div className="config-section">
                <h3>Hardware Identifiers</h3>

                <div className="config-field">
                  <label>ICCID (SIM Card)</label>
                  <input
                    type="text"
                    value={config.iccid || ''}
                    onChange={(e) => handleChange('iccid', e.target.value)}
                    placeholder="e.g., 89310410106543789301"
                  />
                </div>

                <div className="config-field">
                  <label>IMSI (Mobile Network)</label>
                  <input
                    type="text"
                    value={config.imsi || ''}
                    onChange={(e) => handleChange('imsi', e.target.value)}
                    placeholder="e.g., 310410123456789"
                  />
                </div>
              </div>

              <div className="config-section">
                <h3>Connectors</h3>
                <p className="config-note">
                  Connector configuration is managed through the full station form.
                  Click "Full Configuration" below to access connector settings.
                </p>
              </div>
            </div>
          )}
        </div>

        <div className="station-config-footer">
          <button className="btn-cancel" onClick={onClose}>
            Cancel
          </button>
          <button className="btn-save" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving...' : 'Save Configuration'}
          </button>
        </div>
      </div>
    </div>
  )
}

StationConfig.propTypes = {
  station: PropTypes.object.isRequired,
  onSave: PropTypes.func.isRequired,
  onClose: PropTypes.func.isRequired
}

export default StationConfig
