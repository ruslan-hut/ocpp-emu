import { useState, useEffect, useRef } from 'react'
import Editor from '@monaco-editor/react'
import { stationsAPI } from '../services/api'
import TemplateLibrary from '../components/TemplateLibrary'
import { ocppValidator, ValidationMode, OCPPProtocol } from '../services/ocppValidator'
import './MessageCrafter.css'

function MessageCrafter() {
  const [stations, setStations] = useState([])
  const [selectedStation, setSelectedStation] = useState('')
  const [messageType, setMessageType] = useState('Call')
  const [uniqueId, setUniqueId] = useState('')
  const [action, setAction] = useState('Heartbeat')
  const [payload, setPayload] = useState('{}')
  const [sending, setSending] = useState(false)
  const [result, setResult] = useState(null)
  const [jsonError, setJsonError] = useState(null)
  const [showTemplateLibrary, setShowTemplateLibrary] = useState(false)
  const [validationEnabled, setValidationEnabled] = useState(true)
  const [validationMode, setValidationMode] = useState(ValidationMode.STRICT)
  const [validationResult, setValidationResult] = useState(null)
  const editorRef = useRef(null)

  // OCPP 1.6 message templates
  const ocpp16Templates = {
    Heartbeat: '{}',
    BootNotification: JSON.stringify({
      chargePointVendor: "VendorName",
      chargePointModel: "ModelX"
    }, null, 2),
    StatusNotification: JSON.stringify({
      connectorId: 1,
      errorCode: "NoError",
      status: "Available"
    }, null, 2),
    Authorize: JSON.stringify({
      idTag: "TAG123456"
    }, null, 2),
    StartTransaction: JSON.stringify({
      connectorId: 1,
      idTag: "TAG123456",
      meterStart: 0,
      timestamp: new Date().toISOString()
    }, null, 2),
    StopTransaction: JSON.stringify({
      transactionId: 1,
      meterStop: 1000,
      timestamp: new Date().toISOString()
    }, null, 2),
    MeterValues: JSON.stringify({
      connectorId: 1,
      transactionId: 1,
      meterValue: [{
        timestamp: new Date().toISOString(),
        sampledValue: [{
          value: "1000",
          context: "Sample.Periodic",
          measurand: "Energy.Active.Import.Register",
          unit: "Wh"
        }]
      }]
    }, null, 2),
    DataTransfer: JSON.stringify({
      vendorId: "VendorName",
      messageId: "CustomMessage",
      data: "test data"
    }, null, 2)
  }

  // OCPP 2.0.1 message templates
  const ocpp201Templates = {
    Heartbeat: '{}',
    BootNotification: JSON.stringify({
      reason: "PowerUp",
      chargingStation: {
        model: "ModelX",
        vendorName: "VendorName",
        serialNumber: "SN123456",
        firmwareVersion: "1.0.0"
      }
    }, null, 2),
    StatusNotification: JSON.stringify({
      timestamp: new Date().toISOString(),
      connectorStatus: "Available",
      evseId: 1,
      connectorId: 1
    }, null, 2),
    Authorize: JSON.stringify({
      idToken: {
        idToken: "TAG123456",
        type: "ISO14443"
      }
    }, null, 2),
    TransactionEvent: JSON.stringify({
      eventType: "Started",
      timestamp: new Date().toISOString(),
      triggerReason: "Authorized",
      seqNo: 0,
      transactionInfo: {
        transactionId: "TX-" + Date.now(),
        chargingState: "Charging"
      },
      idToken: {
        idToken: "TAG123456",
        type: "ISO14443"
      },
      evse: {
        id: 1,
        connectorId: 1
      }
    }, null, 2),
    NotifyReport: JSON.stringify({
      requestId: 1,
      generatedAt: new Date().toISOString(),
      seqNo: 0,
      reportData: [{
        component: {
          name: "ChargingStation"
        },
        variable: {
          name: "Model"
        },
        variableAttribute: [{
          type: "Actual",
          value: "ModelX",
          mutability: "ReadOnly"
        }]
      }]
    }, null, 2),
    GetVariables: JSON.stringify({
      getVariableData: [{
        component: {
          name: "ChargingStation"
        },
        variable: {
          name: "Model"
        },
        attributeType: "Actual"
      }]
    }, null, 2),
    SetVariables: JSON.stringify({
      setVariableData: [{
        component: {
          name: "ChargingStation"
        },
        variable: {
          name: "AllowNewSessionsPendingFirmwareUpdate"
        },
        attributeType: "Actual",
        attributeValue: "true"
      }]
    }, null, 2),
    SignCertificate: JSON.stringify({
      csr: "-----BEGIN CERTIFICATE REQUEST-----\nMIIBIjANBgkqh...\n-----END CERTIFICATE REQUEST-----",
      certificateType: "ChargingStationCertificate"
    }, null, 2),
    Get15118EVCertificate: JSON.stringify({
      iso15118SchemaVersion: "urn:iso:15118:2:2013:MsgDef",
      action: "Install",
      exiRequest: "base64-encoded-exi-request"
    }, null, 2),
    SecurityEventNotification: JSON.stringify({
      type: "FirmwareUpdated",
      timestamp: new Date().toISOString(),
      techInfo: "Firmware updated to version 1.1.0"
    }, null, 2),
    DataTransfer: JSON.stringify({
      vendorId: "VendorName",
      messageId: "CustomMessage",
      data: "test data"
    }, null, 2),
    LogStatusNotification: JSON.stringify({
      status: "Uploaded",
      requestId: 1
    }, null, 2),
    FirmwareStatusNotification: JSON.stringify({
      status: "Installed",
      requestId: 1
    }, null, 2)
  }

  // OCPP 2.1 message templates (extends 2.0.1 with additional messages)
  const ocpp21Templates = {
    // Inherited from 2.0.1
    ...ocpp201Templates,

    // OCPP 2.1 Cost and Tariff
    CostUpdated: JSON.stringify({
      totalCost: 12.50,
      transactionId: "TX-123456"
    }, null, 2),
    NotifyCustomerInformation: JSON.stringify({
      data: "Customer information data",
      seqNo: 0,
      requestId: 1,
      tbc: false,
      generatedAt: new Date().toISOString()
    }, null, 2),
    NotifyEVChargingNeeds: JSON.stringify({
      evseId: 1,
      chargingNeeds: {
        requestedEnergyTransfer: "DC",
        departureTime: new Date(Date.now() + 3600000).toISOString(),
        dcChargingParameters: {
          evMaxCurrent: 300,
          evMaxVoltage: 500,
          evMaxPower: 150000,
          stateOfCharge: 20,
          evEnergyCapacity: 75000
        }
      }
    }, null, 2),

    // OCPP 2.1 Display Messages
    SetDisplayMessage: JSON.stringify({
      message: {
        id: 1,
        priority: "NormalCycle",
        state: "Idle",
        message: {
          format: "UTF8",
          content: "Welcome to the charging station"
        }
      }
    }, null, 2),
    GetDisplayMessages: JSON.stringify({
      requestId: 1,
      priority: "NormalCycle"
    }, null, 2),
    ClearDisplayMessage: JSON.stringify({
      id: 1
    }, null, 2),
    NotifyDisplayMessages: JSON.stringify({
      requestId: 1,
      tbc: false,
      messageInfo: [{
        id: 1,
        priority: "NormalCycle",
        message: {
          format: "UTF8",
          content: "Station message"
        }
      }]
    }, null, 2),

    // OCPP 2.1 Reservations
    ReserveNow: JSON.stringify({
      id: 1,
      expiryDateTime: new Date(Date.now() + 3600000).toISOString(),
      evseId: 1,
      idToken: {
        idToken: "TAG123456",
        type: "ISO14443"
      }
    }, null, 2),
    CancelReservation: JSON.stringify({
      reservationId: 1
    }, null, 2),

    // OCPP 2.1 Charging Profiles
    SetChargingProfile: JSON.stringify({
      evseId: 1,
      chargingProfile: {
        id: 1,
        stackLevel: 0,
        chargingProfilePurpose: "TxDefaultProfile",
        chargingProfileKind: "Relative",
        chargingSchedule: [{
          id: 1,
          chargingRateUnit: "W",
          chargingSchedulePeriod: [{
            startPeriod: 0,
            limit: 11000
          }]
        }]
      }
    }, null, 2),
    GetChargingProfiles: JSON.stringify({
      requestId: 1,
      evseId: 1,
      chargingProfile: {
        chargingProfilePurpose: "TxDefaultProfile"
      }
    }, null, 2),
    ClearChargingProfile: JSON.stringify({
      chargingProfileId: 1
    }, null, 2),
    GetCompositeSchedule: JSON.stringify({
      evseId: 1,
      duration: 3600,
      chargingRateUnit: "W"
    }, null, 2),
    ReportChargingProfiles: JSON.stringify({
      requestId: 1,
      chargingLimitSource: "EMS",
      evseId: 1,
      tbc: false,
      chargingProfile: [{
        id: 1,
        stackLevel: 0,
        chargingProfilePurpose: "TxDefaultProfile",
        chargingProfileKind: "Relative",
        chargingSchedule: [{
          id: 1,
          chargingRateUnit: "W",
          chargingSchedulePeriod: [{
            startPeriod: 0,
            limit: 11000
          }]
        }]
      }]
    }, null, 2),
    NotifyChargingLimit: JSON.stringify({
      chargingLimit: {
        chargingLimitSource: "EMS",
        isGridCritical: false
      },
      chargingSchedule: [{
        id: 1,
        chargingRateUnit: "W",
        chargingSchedulePeriod: [{
          startPeriod: 0,
          limit: 11000
        }]
      }]
    }, null, 2),
    ClearedChargingLimit: JSON.stringify({
      chargingLimitSource: "EMS",
      evseId: 1
    }, null, 2),

    // OCPP 2.1 Local Authorization
    GetLocalListVersion: JSON.stringify({}, null, 2),
    SendLocalList: JSON.stringify({
      versionNumber: 1,
      updateType: "Full",
      localAuthorizationList: [{
        idToken: {
          idToken: "TAG123456",
          type: "ISO14443"
        },
        idTokenInfo: {
          status: "Accepted"
        }
      }]
    }, null, 2),

    // OCPP 2.1 Firmware Management
    UpdateFirmware: JSON.stringify({
      requestId: 1,
      firmware: {
        location: "https://example.com/firmware/v1.2.0.bin",
        retrieveDateTime: new Date().toISOString(),
        signingCertificate: "",
        signature: ""
      },
      retries: 3,
      retryInterval: 60
    }, null, 2),
    SetNetworkProfile: JSON.stringify({
      configurationSlot: 1,
      connectionData: {
        ocppVersion: "OCPP21",
        ocppTransport: "JSON",
        ocppCsmsUrl: "wss://csms.example.com/ocpp",
        messageTimeout: 30,
        securityProfile: 1,
        ocppInterface: "Wired0"
      }
    }, null, 2),
    GetLog: JSON.stringify({
      logType: "DiagnosticsLog",
      requestId: 1,
      log: {
        remoteLocation: "https://example.com/logs/upload"
      }
    }, null, 2)
  }

  // Get selected station object
  const selectedStationObj = stations.find(s => s.stationId === selectedStation)

  // Get protocol version from selected station
  const protocolVersion = selectedStationObj?.protocolVersion || 'ocpp1.6'
  const isOcpp21 = protocolVersion === 'ocpp2.1' || protocolVersion === '2.1' || protocolVersion === 'ocpp21'
  const isOcpp201 = protocolVersion === 'ocpp2.0.1' || protocolVersion === '2.0.1' || protocolVersion === 'ocpp201'

  // Get templates based on protocol version
  const messageTemplates = isOcpp21 ? ocpp21Templates : (isOcpp201 ? ocpp201Templates : ocpp16Templates)

  useEffect(() => {
    fetchStations()
  }, [])

  useEffect(() => {
    // Generate a new unique ID whenever action or messageType changes
    setUniqueId(generateUniqueId())
  }, [action, messageType])

  useEffect(() => {
    // Update payload when action changes
    if (messageTemplates[action]) {
      setPayload(messageTemplates[action])
    }
  }, [action, messageTemplates])

  // Reset action to first available when station changes (protocol might differ)
  useEffect(() => {
    const availableActions = Object.keys(messageTemplates)
    if (!availableActions.includes(action)) {
      setAction(availableActions[0] || 'Heartbeat')
    }
  }, [selectedStation, messageTemplates])

  // Update validator protocol when station changes
  useEffect(() => {
    ocppValidator.setProtocol(protocolVersion)
  }, [protocolVersion])

  const fetchStations = async () => {
    try {
      const response = await stationsAPI.getAll()
      const stationList = response.data.stations || []
      setStations(stationList)

      // Select first connected station
      const connected = stationList.find(s => s.runtimeState?.connectionStatus === 'connected')
      if (connected) {
        setSelectedStation(connected.stationId)
      }
    } catch (err) {
      console.error('Failed to load stations:', err)
    }
  }

  const generateUniqueId = () => {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
  }

  const handleActionChange = (newAction) => {
    setAction(newAction)
  }

  const handleEditorMount = (editor) => {
    editorRef.current = editor
  }

  const handleEditorChange = (value) => {
    setPayload(value || '{}')
    // Validate JSON
    try {
      JSON.parse(value || '{}')
      setJsonError(null)
    } catch (err) {
      setJsonError(err.message)
    }
  }

  // Validate message whenever it changes
  useEffect(() => {
    if (!validationEnabled) {
      setValidationResult(null)
      return
    }

    try {
      const message = buildMessage()
      ocppValidator.setMode(validationMode)
      const result = ocppValidator.validateMessage(message)
      setValidationResult(result)
    } catch (err) {
      // If buildMessage fails, don't show validation errors
      setValidationResult(null)
    }
  }, [payload, action, uniqueId, messageType, validationEnabled, validationMode])

  const formatJSON = () => {
    try {
      const parsed = JSON.parse(payload)
      const formatted = JSON.stringify(parsed, null, 2)
      setPayload(formatted)
      setJsonError(null)
    } catch (err) {
      setJsonError(err.message)
    }
  }

  const handleTemplateSelect = (template) => {
    setAction(template.action)
    setPayload(template.payload)
    setJsonError(null)
  }

  const buildMessage = () => {
    try {
      const payloadObj = JSON.parse(payload)

      if (messageType === 'Call') {
        return [2, uniqueId, action, payloadObj]
      } else if (messageType === 'CallResult') {
        return [3, uniqueId, payloadObj]
      } else if (messageType === 'CallError') {
        // For CallError: [4, uniqueId, errorCode, errorDescription, errorDetails]
        return [4, uniqueId, "GenericError", "Error description", payloadObj]
      }
    } catch (err) {
      throw new Error(`Invalid JSON payload: ${err.message}`)
    }
  }

  const handleSend = async () => {
    if (!selectedStation) {
      setResult({ success: false, error: 'Please select a station' })
      return
    }

    // Check validation if enabled
    if (validationEnabled && validationResult) {
      if (validationResult.errors.length > 0) {
        setResult({
          success: false,
          error: `Cannot send message: ${validationResult.errors.length} validation error${validationResult.errors.length > 1 ? 's' : ''} found`
        })
        return
      }

      if (validationMode === ValidationMode.STRICT && validationResult.warnings.length > 0) {
        setResult({
          success: false,
          error: `Cannot send message in strict mode: ${validationResult.warnings.length} warning${validationResult.warnings.length > 1 ? 's' : ''} found`
        })
        return
      }
    }

    try {
      setSending(true)
      setResult(null)

      const message = buildMessage()

      const response = await fetch(`http://localhost:8080/api/stations/${selectedStation}/send-message`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ message })
      })

      const data = await response.json()

      if (response.ok) {
        setResult({
          success: true,
          message: 'Message sent successfully',
          sentMessage: message
        })

        // Generate new unique ID for next message
        setUniqueId(generateUniqueId())
      } else {
        setResult({
          success: false,
          error: data.error || 'Failed to send message'
        })
      }
    } catch (err) {
      setResult({
        success: false,
        error: err.message || 'Failed to send message'
      })
    } finally {
      setSending(false)
    }
  }

  const formatMessage = () => {
    try {
      const message = buildMessage()
      return JSON.stringify(message, null, 2)
    } catch (err) {
      return `Error: ${err.message}`
    }
  }

  const connectedStations = stations.filter(s => s.runtimeState?.connectionStatus === 'connected')

  // Determine validation status for inline display
  const getValidationStatus = () => {
    if (!validationEnabled || !validationResult) return null
    if (validationResult.errors.length > 0) return 'error'
    if (validationResult.warnings.length > 0) return 'warning'
    return 'valid'
  }

  return (
    <div className="message-crafter message-crafter--desktop">
      {/* Compact Header */}
      <div className="crafter-header">
        <div className="crafter-header__title">
          <h2>Message Crafter</h2>
          <span className="crafter-header__subtitle">Craft and send custom OCPP messages</span>
        </div>
        <div className="crafter-header__actions">
          <button
            className="btn-templates--compact"
            onClick={() => setShowTemplateLibrary(true)}
          >
            Templates
          </button>
        </div>
      </div>

      {/* Three Column Layout */}
      <div className="crafter-layout">
        {/* Left Column - Settings */}
        <div className="crafter-column crafter-column--settings">
          {/* Station Selection */}
          <div className="settings-group">
            <label className="settings-label">Station</label>
            <select
              value={selectedStation}
              onChange={(e) => setSelectedStation(e.target.value)}
              className="settings-select"
            >
              <option value="">Select station...</option>
              {connectedStations.map(station => (
                <option key={station.stationId} value={station.stationId}>
                  {station.name}
                </option>
              ))}
            </select>
            {selectedStation && selectedStationObj && (
              <div className="station-meta">
                <span className={`protocol-tag ${isOcpp21 ? 'ocpp21' : isOcpp201 ? 'ocpp201' : 'ocpp16'}`}>
                  {protocolVersion.toUpperCase()}
                </span>
                <span className="status-tag status-tag--connected">Connected</span>
              </div>
            )}
            {connectedStations.length === 0 && (
              <div className="warning-inline">No connected stations</div>
            )}
          </div>

          {/* Message Type */}
          <div className="settings-group">
            <label className="settings-label">Message Type</label>
            <div className="type-selector">
              <button
                className={`type-btn--compact ${messageType === 'Call' ? 'active' : ''}`}
                onClick={() => setMessageType('Call')}
              >
                Call
              </button>
              <button
                className={`type-btn--compact ${messageType === 'CallResult' ? 'active' : ''}`}
                onClick={() => setMessageType('CallResult')}
              >
                Result
              </button>
              <button
                className={`type-btn--compact ${messageType === 'CallError' ? 'active' : ''}`}
                onClick={() => setMessageType('CallError')}
              >
                Error
              </button>
            </div>
          </div>

          {/* Action (for Call type) */}
          {messageType === 'Call' && (
            <div className="settings-group">
              <label className="settings-label">Action</label>
              <select
                value={action}
                onChange={(e) => handleActionChange(e.target.value)}
                className="settings-select"
              >
                {Object.keys(messageTemplates).map(actionName => (
                  <option key={actionName} value={actionName}>{actionName}</option>
                ))}
              </select>
            </div>
          )}

          {/* Unique ID */}
          <div className="settings-group">
            <label className="settings-label">Message ID</label>
            <input
              type="text"
              value={uniqueId}
              onChange={(e) => setUniqueId(e.target.value)}
              className="settings-input settings-input--mono"
              placeholder="Auto-generated"
            />
          </div>

          {/* Validation Toggle */}
          <div className="settings-group">
            <label className="settings-label">Validation</label>
            <div className="validation-row">
              <label className="toggle-compact">
                <input
                  type="checkbox"
                  checked={validationEnabled}
                  onChange={(e) => setValidationEnabled(e.target.checked)}
                />
                <span>{validationEnabled ? 'Enabled' : 'Disabled'}</span>
              </label>
              {validationEnabled && (
                <div className="mode-toggle">
                  <button
                    className={`mode-btn--sm ${validationMode === ValidationMode.STRICT ? 'active' : ''}`}
                    onClick={() => setValidationMode(ValidationMode.STRICT)}
                  >
                    Strict
                  </button>
                  <button
                    className={`mode-btn--sm ${validationMode === ValidationMode.LENIENT ? 'active' : ''}`}
                    onClick={() => setValidationMode(ValidationMode.LENIENT)}
                  >
                    Lenient
                  </button>
                </div>
              )}
            </div>
          </div>

          {/* Send Button */}
          <div className="settings-group settings-group--send">
            <button
              className="btn-send--compact"
              onClick={handleSend}
              disabled={!selectedStation || sending}
            >
              {sending ? 'Sending...' : 'Send Message'}
            </button>
          </div>
        </div>

        {/* Middle Column - Editor */}
        <div className="crafter-column crafter-column--editor">
          <div className="editor-panel">
            <div className="editor-panel__header">
              <span className="editor-panel__title">
                {messageType === 'Call' ? 'Payload' : messageType === 'CallResult' ? 'Result Payload' : 'Error Details'}
              </span>
              <div className="editor-panel__actions">
                <button
                  type="button"
                  className="btn-action--xs"
                  onClick={formatJSON}
                  title="Format JSON"
                >
                  Format
                </button>
              </div>
            </div>
            <div className="editor-container">
              <Editor
                height="100%"
                defaultLanguage="json"
                value={payload}
                onChange={handleEditorChange}
                onMount={handleEditorMount}
                theme="vs-dark"
                options={{
                  minimap: { enabled: false },
                  fontSize: 13,
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                  tabSize: 2,
                  formatOnPaste: true,
                  formatOnType: true,
                  wordWrap: 'on'
                }}
              />
            </div>
            {/* Inline Validation Status */}
            {jsonError && (
              <div className="editor-status editor-status--error">
                JSON Error: {jsonError}
              </div>
            )}
            {!jsonError && validationEnabled && validationResult && (
              <div className={`editor-status editor-status--${getValidationStatus()}`}>
                {validationResult.errors.length > 0 && (
                  <span>{validationResult.errors.length} error{validationResult.errors.length > 1 ? 's' : ''}</span>
                )}
                {validationResult.errors.length === 0 && validationResult.warnings.length > 0 && (
                  <span>{validationResult.warnings.length} warning{validationResult.warnings.length > 1 ? 's' : ''}</span>
                )}
                {validationResult.errors.length === 0 && validationResult.warnings.length === 0 && (
                  <span>Valid</span>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Right Column - Preview & Results */}
        <div className="crafter-column crafter-column--preview">
          {/* Message Preview */}
          <div className="preview-panel">
            <div className="preview-panel__header">
              <span className="preview-panel__title">Message Preview</span>
            </div>
            <pre className="preview-content">
              {formatMessage()}
            </pre>
          </div>

          {/* Validation Details (expanded) */}
          {validationEnabled && validationResult && (validationResult.errors.length > 0 || validationResult.warnings.length > 0) && (
            <div className="validation-panel">
              <div className="validation-panel__header">
                <span className="validation-panel__title">Validation Issues</span>
              </div>
              <div className="validation-panel__content">
                {validationResult.errors.map((err, i) => (
                  <div key={`err-${i}`} className="validation-item validation-item--error">
                    <code>{err.field}</code>
                    <span>{err.message}</span>
                  </div>
                ))}
                {validationResult.warnings.map((warn, i) => (
                  <div key={`warn-${i}`} className="validation-item validation-item--warning">
                    <code>{warn.field}</code>
                    <span>{warn.message}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Result */}
          {result && (
            <div className={`result-panel result-panel--${result.success ? 'success' : 'error'}`}>
              <div className="result-panel__header">
                {result.success ? 'Message Sent' : 'Error'}
              </div>
              {result.success ? (
                <div className="result-panel__content">
                  <pre>{JSON.stringify(result.sentMessage, null, 2)}</pre>
                </div>
              ) : (
                <div className="result-panel__error">
                  {result.error}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Template Library Modal */}
      {showTemplateLibrary && (
        <TemplateLibrary
          onSelectTemplate={handleTemplateSelect}
          onClose={() => setShowTemplateLibrary(false)}
          currentPayload={payload}
          currentAction={action}
        />
      )}
    </div>
  )
}

export default MessageCrafter
