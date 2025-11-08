import { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import './StationForm.css'

const PROTOCOL_VERSIONS = ['ocpp1.6', 'ocpp2.0.1', 'ocpp2.1']
const CONNECTOR_TYPES = ['Type2', 'CCS', 'CHAdeMO', 'Type1', 'GB/T']
const SUPPORTED_PROFILES = [
  'Core',
  'FirmwareManagement',
  'LocalAuthListManagement',
  'Reservation',
  'SmartCharging',
  'RemoteTrigger'
]
const MEASURANDS = [
  'Energy.Active.Import.Register',
  'Power.Active.Import',
  'Current.Import',
  'Voltage',
  'SoC',
  'Temperature'
]

function StationForm({ station, onSubmit, onCancel, templates = [] }) {
  const [formData, setFormData] = useState({
    stationId: '',
    name: '',
    enabled: true,
    autoStart: false,
    protocolVersion: 'ocpp1.6',
    vendor: '',
    model: '',
    serialNumber: '',
    firmwareVersion: '',
    iccid: '',
    imsi: '',
    connectors: [
      { id: 1, type: 'Type2', maxPower: 22000, status: 'Available' }
    ],
    supportedProfiles: ['Core'],
    meterValuesConfig: {
      interval: 60,
      measurands: ['Energy.Active.Import.Register', 'Power.Active.Import'],
      alignedDataInterval: 900
    },
    csmsUrl: '',
    csmsAuth: {
      type: 'basic',
      username: '',
      password: ''
    },
    simulation: {
      bootDelay: 0,
      heartbeatInterval: 300,
      statusNotificationOnChange: true,
      defaultIdTag: 'DEFAULT_TAG',
      energyDeliveryRate: 7000,
      randomizeMeterValues: true,
      meterValueVariance: 0.1
    },
    tags: []
  })

  const [newTag, setNewTag] = useState('')

  useEffect(() => {
    if (station) {
      setFormData({
        ...station,
        csmsAuth: station.csmsAuth || { type: 'basic', username: '', password: '' }
      })
    }
  }, [station])

  const handleChange = (field, value) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  const handleNestedChange = (parent, field, value) => {
    setFormData(prev => ({
      ...prev,
      [parent]: { ...prev[parent], [field]: value }
    }))
  }

  const handleConnectorChange = (index, field, value) => {
    const newConnectors = [...formData.connectors]
    newConnectors[index] = { ...newConnectors[index], [field]: value }
    setFormData(prev => ({ ...prev, connectors: newConnectors }))
  }

  const addConnector = () => {
    const newId = formData.connectors.length > 0
      ? Math.max(...formData.connectors.map(c => c.id)) + 1
      : 1
    setFormData(prev => ({
      ...prev,
      connectors: [...prev.connectors, {
        id: newId,
        type: 'Type2',
        maxPower: 22000,
        status: 'Available'
      }]
    }))
  }

  const removeConnector = (index) => {
    if (formData.connectors.length <= 1) {
      alert('Station must have at least one connector')
      return
    }
    setFormData(prev => ({
      ...prev,
      connectors: prev.connectors.filter((_, i) => i !== index)
    }))
  }

  const handleProfileToggle = (profile) => {
    const profiles = formData.supportedProfiles || []
    if (profiles.includes(profile)) {
      setFormData(prev => ({
        ...prev,
        supportedProfiles: profiles.filter(p => p !== profile)
      }))
    } else {
      setFormData(prev => ({
        ...prev,
        supportedProfiles: [...profiles, profile]
      }))
    }
  }

  const handleMeasurandToggle = (measurand) => {
    const measurands = formData.meterValuesConfig.measurands || []
    if (measurands.includes(measurand)) {
      handleNestedChange('meterValuesConfig', 'measurands',
        measurands.filter(m => m !== measurand)
      )
    } else {
      handleNestedChange('meterValuesConfig', 'measurands',
        [...measurands, measurand]
      )
    }
  }

  const addTag = () => {
    if (newTag && !formData.tags.includes(newTag)) {
      setFormData(prev => ({
        ...prev,
        tags: [...(prev.tags || []), newTag]
      }))
      setNewTag('')
    }
  }

  const removeTag = (tag) => {
    setFormData(prev => ({
      ...prev,
      tags: prev.tags.filter(t => t !== tag)
    }))
  }

  const loadTemplate = (template) => {
    setFormData({
      ...template,
      stationId: '', // Clear station ID for new station
      name: `${template.name} (Copy)`,
    })
  }

  const handleSubmit = (e) => {
    e.preventDefault()
    onSubmit(formData)
  }

  return (
    <div className="station-form-overlay">
      <div className="station-form-container">
        <div className="form-header">
          <h2>{station ? 'Edit Station' : 'Create New Station'}</h2>
          <button className="close-btn" onClick={onCancel}>×</button>
        </div>

        {!station && templates.length > 0 && (
          <div className="templates-section">
            <label>Load from Template:</label>
            <select onChange={(e) => e.target.value && loadTemplate(JSON.parse(e.target.value))}>
              <option value="">-- Select Template --</option>
              {templates.map((t, i) => (
                <option key={i} value={JSON.stringify(t)}>{t.name}</option>
              ))}
            </select>
          </div>
        )}

        <form onSubmit={handleSubmit}>
          {/* Basic Information */}
          <section className="form-section">
            <h3>Basic Information</h3>
            <div className="form-grid">
              <div className="form-field">
                <label>Station ID *</label>
                <input
                  type="text"
                  value={formData.stationId}
                  onChange={(e) => handleChange('stationId', e.target.value)}
                  required
                  disabled={!!station}
                  placeholder="e.g., CP001"
                />
              </div>

              <div className="form-field">
                <label>Station Name *</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => handleChange('name', e.target.value)}
                  required
                  placeholder="e.g., Main Street Station 1"
                />
              </div>

              <div className="form-field">
                <label>Protocol Version *</label>
                <select
                  value={formData.protocolVersion}
                  onChange={(e) => handleChange('protocolVersion', e.target.value)}
                  required
                >
                  {PROTOCOL_VERSIONS.map(v => (
                    <option key={v} value={v}>{v}</option>
                  ))}
                </select>
              </div>

              <div className="form-field checkbox-field">
                <label>
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) => handleChange('enabled', e.target.checked)}
                  />
                  Enabled
                </label>
              </div>

              <div className="form-field checkbox-field">
                <label>
                  <input
                    type="checkbox"
                    checked={formData.autoStart}
                    onChange={(e) => handleChange('autoStart', e.target.checked)}
                  />
                  Auto Start
                </label>
              </div>
            </div>
          </section>

          {/* Station Details */}
          <section className="form-section">
            <h3>Station Details</h3>
            <div className="form-grid">
              <div className="form-field">
                <label>Vendor *</label>
                <input
                  type="text"
                  value={formData.vendor}
                  onChange={(e) => handleChange('vendor', e.target.value)}
                  required
                  placeholder="e.g., ABB, Schneider Electric"
                />
              </div>

              <div className="form-field">
                <label>Model *</label>
                <input
                  type="text"
                  value={formData.model}
                  onChange={(e) => handleChange('model', e.target.value)}
                  required
                  placeholder="e.g., Terra AC"
                />
              </div>

              <div className="form-field">
                <label>Serial Number</label>
                <input
                  type="text"
                  value={formData.serialNumber}
                  onChange={(e) => handleChange('serialNumber', e.target.value)}
                  placeholder="e.g., SN123456789"
                />
              </div>

              <div className="form-field">
                <label>Firmware Version</label>
                <input
                  type="text"
                  value={formData.firmwareVersion}
                  onChange={(e) => handleChange('firmwareVersion', e.target.value)}
                  placeholder="e.g., 1.2.3"
                />
              </div>

              <div className="form-field">
                <label>ICCID</label>
                <input
                  type="text"
                  value={formData.iccid}
                  onChange={(e) => handleChange('iccid', e.target.value)}
                  placeholder="SIM card ICCID"
                />
              </div>

              <div className="form-field">
                <label>IMSI</label>
                <input
                  type="text"
                  value={formData.imsi}
                  onChange={(e) => handleChange('imsi', e.target.value)}
                  placeholder="SIM card IMSI"
                />
              </div>
            </div>
          </section>

          {/* Connectors */}
          <section className="form-section">
            <div className="section-header">
              <h3>Connectors</h3>
              <button type="button" className="btn-small" onClick={addConnector}>
                + Add Connector
              </button>
            </div>

            {formData.connectors.map((connector, index) => (
              <div key={index} className="connector-item">
                <div className="connector-header">
                  <span>Connector {connector.id}</span>
                  {formData.connectors.length > 1 && (
                    <button
                      type="button"
                      className="btn-remove"
                      onClick={() => removeConnector(index)}
                    >
                      Remove
                    </button>
                  )}
                </div>
                <div className="form-grid">
                  <div className="form-field">
                    <label>Type</label>
                    <select
                      value={connector.type}
                      onChange={(e) => handleConnectorChange(index, 'type', e.target.value)}
                    >
                      {CONNECTOR_TYPES.map(t => (
                        <option key={t} value={t}>{t}</option>
                      ))}
                    </select>
                  </div>

                  <div className="form-field">
                    <label>Max Power (W)</label>
                    <input
                      type="number"
                      value={connector.maxPower}
                      onChange={(e) => handleConnectorChange(index, 'maxPower', parseInt(e.target.value))}
                      min="0"
                    />
                  </div>
                </div>
              </div>
            ))}
          </section>

          {/* CSMS Connection */}
          <section className="form-section">
            <h3>CSMS Connection</h3>
            <div className="form-grid">
              <div className="form-field full-width">
                <label>CSMS URL *</label>
                <input
                  type="url"
                  value={formData.csmsUrl}
                  onChange={(e) => handleChange('csmsUrl', e.target.value)}
                  required
                  placeholder="ws://localhost:9000/ocpp"
                />
              </div>

              <div className="form-field">
                <label>Auth Type</label>
                <select
                  value={formData.csmsAuth?.type || 'basic'}
                  onChange={(e) => handleNestedChange('csmsAuth', 'type', e.target.value)}
                >
                  <option value="basic">Basic Auth</option>
                  <option value="bearer">Bearer Token</option>
                  <option value="none">None</option>
                </select>
              </div>

              {formData.csmsAuth?.type === 'basic' && (
                <>
                  <div className="form-field">
                    <label>Username</label>
                    <input
                      type="text"
                      value={formData.csmsAuth.username}
                      onChange={(e) => handleNestedChange('csmsAuth', 'username', e.target.value)}
                    />
                  </div>

                  <div className="form-field">
                    <label>Password</label>
                    <input
                      type="password"
                      value={formData.csmsAuth.password}
                      onChange={(e) => handleNestedChange('csmsAuth', 'password', e.target.value)}
                    />
                  </div>
                </>
              )}
            </div>
          </section>

          {/* Supported Profiles */}
          <section className="form-section">
            <h3>Supported Profiles</h3>
            <div className="checkbox-group">
              {SUPPORTED_PROFILES.map(profile => (
                <label key={profile} className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.supportedProfiles?.includes(profile)}
                    onChange={() => handleProfileToggle(profile)}
                  />
                  {profile}
                </label>
              ))}
            </div>
          </section>

          {/* Meter Values Configuration */}
          <section className="form-section">
            <h3>Meter Values Configuration</h3>
            <div className="form-grid">
              <div className="form-field">
                <label>Interval (seconds)</label>
                <input
                  type="number"
                  value={formData.meterValuesConfig.interval}
                  onChange={(e) => handleNestedChange('meterValuesConfig', 'interval', parseInt(e.target.value))}
                  min="1"
                />
              </div>

              <div className="form-field">
                <label>Aligned Data Interval (seconds)</label>
                <input
                  type="number"
                  value={formData.meterValuesConfig.alignedDataInterval}
                  onChange={(e) => handleNestedChange('meterValuesConfig', 'alignedDataInterval', parseInt(e.target.value))}
                  min="0"
                />
              </div>
            </div>

            <div className="form-field full-width">
              <label>Measurands</label>
              <div className="checkbox-group">
                {MEASURANDS.map(measurand => (
                  <label key={measurand} className="checkbox-label">
                    <input
                      type="checkbox"
                      checked={formData.meterValuesConfig.measurands?.includes(measurand)}
                      onChange={() => handleMeasurandToggle(measurand)}
                    />
                    {measurand}
                  </label>
                ))}
              </div>
            </div>
          </section>

          {/* Simulation Settings */}
          <section className="form-section">
            <h3>Simulation Settings</h3>
            <div className="form-grid">
              <div className="form-field">
                <label>Boot Delay (seconds)</label>
                <input
                  type="number"
                  value={formData.simulation.bootDelay}
                  onChange={(e) => handleNestedChange('simulation', 'bootDelay', parseInt(e.target.value))}
                  min="0"
                />
              </div>

              <div className="form-field">
                <label>Heartbeat Interval (seconds)</label>
                <input
                  type="number"
                  value={formData.simulation.heartbeatInterval}
                  onChange={(e) => handleNestedChange('simulation', 'heartbeatInterval', parseInt(e.target.value))}
                  min="1"
                />
              </div>

              <div className="form-field">
                <label>Default ID Tag</label>
                <input
                  type="text"
                  value={formData.simulation.defaultIdTag}
                  onChange={(e) => handleNestedChange('simulation', 'defaultIdTag', e.target.value)}
                />
              </div>

              <div className="form-field">
                <label>Energy Delivery Rate (W)</label>
                <input
                  type="number"
                  value={formData.simulation.energyDeliveryRate}
                  onChange={(e) => handleNestedChange('simulation', 'energyDeliveryRate', parseInt(e.target.value))}
                  min="0"
                />
              </div>

              <div className="form-field">
                <label>Meter Value Variance</label>
                <input
                  type="number"
                  value={formData.simulation.meterValueVariance}
                  onChange={(e) => handleNestedChange('simulation', 'meterValueVariance', parseFloat(e.target.value))}
                  step="0.01"
                  min="0"
                  max="1"
                />
              </div>

              <div className="form-field checkbox-field">
                <label>
                  <input
                    type="checkbox"
                    checked={formData.simulation.statusNotificationOnChange}
                    onChange={(e) => handleNestedChange('simulation', 'statusNotificationOnChange', e.target.checked)}
                  />
                  Status Notification on Change
                </label>
              </div>

              <div className="form-field checkbox-field">
                <label>
                  <input
                    type="checkbox"
                    checked={formData.simulation.randomizeMeterValues}
                    onChange={(e) => handleNestedChange('simulation', 'randomizeMeterValues', e.target.checked)}
                  />
                  Randomize Meter Values
                </label>
              </div>
            </div>
          </section>

          {/* Tags */}
          <section className="form-section">
            <h3>Tags</h3>
            <div className="tags-input">
              <input
                type="text"
                value={newTag}
                onChange={(e) => setNewTag(e.target.value)}
                placeholder="Add tag..."
                onKeyPress={(e) => e.key === 'Enter' && (e.preventDefault(), addTag())}
              />
              <button type="button" className="btn-small" onClick={addTag}>Add</button>
            </div>
            <div className="tags-list">
              {formData.tags?.map(tag => (
                <span key={tag} className="tag">
                  {tag}
                  <button type="button" onClick={() => removeTag(tag)}>×</button>
                </span>
              ))}
            </div>
          </section>

          {/* Form Actions */}
          <div className="form-actions">
            <button type="button" className="btn-secondary" onClick={onCancel}>
              Cancel
            </button>
            <button type="submit" className="btn-primary">
              {station ? 'Update Station' : 'Create Station'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

StationForm.propTypes = {
  station: PropTypes.object,
  onSubmit: PropTypes.func.isRequired,
  onCancel: PropTypes.func.isRequired,
  templates: PropTypes.array
}

export default StationForm
