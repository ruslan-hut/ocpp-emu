import { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import './TemplatesManager.css'

// Default templates
const DEFAULT_TEMPLATES = [
  {
    name: 'AC 22kW Charger',
    description: 'Standard AC charger with 22kW Type2 connector',
    protocolVersion: 'ocpp1.6',
    vendor: 'Generic',
    model: 'AC-22',
    firmwareVersion: '1.0.0',
    connectors: [
      { id: 1, type: 'Type2', maxPower: 22000, status: 'Available' }
    ],
    supportedProfiles: ['Core', 'SmartCharging'],
    meterValuesConfig: {
      interval: 60,
      measurands: ['Energy.Active.Import.Register', 'Power.Active.Import'],
      alignedDataInterval: 900
    },
    simulation: {
      bootDelay: 0,
      heartbeatInterval: 300,
      statusNotificationOnChange: true,
      defaultIdTag: 'DEFAULT_TAG',
      energyDeliveryRate: 7000,
      randomizeMeterValues: true,
      meterValueVariance: 0.1
    }
  },
  {
    name: 'DC 50kW Fast Charger',
    description: 'DC fast charger with CCS connector',
    protocolVersion: 'ocpp1.6',
    vendor: 'Generic',
    model: 'DC-50',
    firmwareVersion: '1.0.0',
    connectors: [
      { id: 1, type: 'CCS', maxPower: 50000, status: 'Available' }
    ],
    supportedProfiles: ['Core', 'SmartCharging'],
    meterValuesConfig: {
      interval: 30,
      measurands: ['Energy.Active.Import.Register', 'Power.Active.Import', 'Current.Import'],
      alignedDataInterval: 900
    },
    simulation: {
      bootDelay: 0,
      heartbeatInterval: 300,
      statusNotificationOnChange: true,
      defaultIdTag: 'DEFAULT_TAG',
      energyDeliveryRate: 45000,
      randomizeMeterValues: true,
      meterValueVariance: 0.05
    }
  },
  {
    name: 'Dual Port AC Charger',
    description: 'Dual port AC charger with two Type2 connectors',
    protocolVersion: 'ocpp1.6',
    vendor: 'Generic',
    model: 'AC-DUAL-22',
    firmwareVersion: '1.0.0',
    connectors: [
      { id: 1, type: 'Type2', maxPower: 22000, status: 'Available' },
      { id: 2, type: 'Type2', maxPower: 22000, status: 'Available' }
    ],
    supportedProfiles: ['Core', 'SmartCharging', 'Reservation'],
    meterValuesConfig: {
      interval: 60,
      measurands: ['Energy.Active.Import.Register', 'Power.Active.Import'],
      alignedDataInterval: 900
    },
    simulation: {
      bootDelay: 0,
      heartbeatInterval: 300,
      statusNotificationOnChange: true,
      defaultIdTag: 'DEFAULT_TAG',
      energyDeliveryRate: 7000,
      randomizeMeterValues: true,
      meterValueVariance: 0.1
    }
  }
]

function TemplatesManager({ onClose, onSelectTemplate }) {
  const [templates, setTemplates] = useState([])
  const [selectedTemplate, setSelectedTemplate] = useState(null)
  const [editingTemplate, setEditingTemplate] = useState(null)
  const [templateName, setTemplateName] = useState('')
  const [templateDescription, setTemplateDescription] = useState('')

  useEffect(() => {
    // Load templates from localStorage
    const saved = localStorage.getItem('stationTemplates')
    if (saved) {
      try {
        setTemplates(JSON.parse(saved))
      } catch (e) {
        console.error('Failed to load templates:', e)
        setTemplates([...DEFAULT_TEMPLATES])
      }
    } else {
      setTemplates([...DEFAULT_TEMPLATES])
    }
  }, [])

  const saveTemplates = (newTemplates) => {
    setTemplates(newTemplates)
    localStorage.setItem('stationTemplates', JSON.stringify(newTemplates))
  }

  const handleSaveAsTemplate = () => {
    if (!templateName.trim()) {
      alert('Please enter a template name')
      return
    }

    const newTemplate = {
      ...editingTemplate,
      name: templateName,
      description: templateDescription
    }

    const updated = [...templates, newTemplate]
    saveTemplates(updated)
    setEditingTemplate(null)
    setTemplateName('')
    setTemplateDescription('')
  }

  const handleDeleteTemplate = (index) => {
    if (confirm(`Delete template "${templates[index].name}"?`)) {
      const updated = templates.filter((_, i) => i !== index)
      saveTemplates(updated)
    }
  }

  const handleSelectTemplate = (template) => {
    onSelectTemplate(template)
    onClose()
  }

  const handleExportTemplate = (template) => {
    const dataStr = JSON.stringify(template, null, 2)
    const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr)
    const exportFileDefaultName = `template-${template.name.toLowerCase().replace(/\s+/g, '-')}.json`

    const linkElement = document.createElement('a')
    linkElement.setAttribute('href', dataUri)
    linkElement.setAttribute('download', exportFileDefaultName)
    linkElement.click()
  }

  const handleImportTemplate = (e) => {
    const file = e.target.files[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (event) => {
      try {
        const template = JSON.parse(event.target.result)
        const updated = [...templates, template]
        saveTemplates(updated)
        alert('Template imported successfully!')
      } catch (err) {
        alert('Failed to import template: Invalid JSON file')
      }
    }
    reader.readAsText(file)
  }

  return (
    <div className="templates-overlay">
      <div className="templates-container">
        <div className="templates-header">
          <h2>Station Templates</h2>
          <button className="close-btn" onClick={onClose}>×</button>
        </div>

        <div className="templates-actions">
          <label className="btn-secondary import-btn">
            <input
              type="file"
              accept=".json"
              onChange={handleImportTemplate}
              style={{ display: 'none' }}
            />
            Import Template
          </label>
        </div>

        <div className="templates-grid">
          {templates.map((template, index) => (
            <div
              key={index}
              className={`template-card ${selectedTemplate === index ? 'selected' : ''}`}
              onClick={() => setSelectedTemplate(index)}
            >
              <div className="template-header">
                <h3>{template.name}</h3>
                <span className="template-protocol">{template.protocolVersion}</span>
              </div>

              {template.description && (
                <p className="template-description">{template.description}</p>
              )}

              <div className="template-details">
                <div className="detail-item">
                  <span className="detail-label">Vendor:</span>
                  <span>{template.vendor || 'N/A'}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Model:</span>
                  <span>{template.model || 'N/A'}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Connectors:</span>
                  <span>{template.connectors?.length || 0}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Max Power:</span>
                  <span>{Math.max(...(template.connectors || []).map(c => c.maxPower)) / 1000} kW</span>
                </div>
              </div>

              <div className="template-actions">
                <button
                  className="btn-small btn-use"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleSelectTemplate(template)
                  }}
                >
                  Use Template
                </button>
                <button
                  className="btn-small btn-export"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleExportTemplate(template)
                  }}
                >
                  Export
                </button>
                {index >= DEFAULT_TEMPLATES.length && (
                  <button
                    className="btn-small btn-delete"
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDeleteTemplate(index)
                    }}
                  >
                    Delete
                  </button>
                )}
              </div>
            </div>
          ))}

          {templates.length === 0 && (
            <div className="empty-templates">
              <p>No templates available</p>
              <p>Import a template or create one from an existing station</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

TemplatesManager.propTypes = {
  onClose: PropTypes.func.isRequired,
  onSelectTemplate: PropTypes.func.isRequired
}

export default TemplatesManager
