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

  // Get selected station object
  const selectedStationObj = stations.find(s => s.stationId === selectedStation)

  // Get protocol version from selected station
  const protocolVersion = selectedStationObj?.protocolVersion || 'ocpp1.6'
  const isOcpp201 = protocolVersion === 'ocpp2.0.1' || protocolVersion === 'ocpp2.1'

  // Get templates based on protocol version
  const messageTemplates = isOcpp201 ? ocpp201Templates : ocpp16Templates

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

  return (
    <div className="message-crafter">
      <div className="page-header">
        <div>
          <h2>Message Crafter</h2>
          <p>Craft and send custom OCPP messages for testing</p>
        </div>
        <button
          className="btn-templates"
          onClick={() => setShowTemplateLibrary(true)}
        >
          📚 Template Library
        </button>
      </div>

      <div className="crafter-container">
        {/* Station Selection */}
        <div className="crafter-section">
          <h3>1. Select Station</h3>
          <select
            value={selectedStation}
            onChange={(e) => setSelectedStation(e.target.value)}
            className="station-select"
          >
            <option value="">-- Select a station --</option>
            {connectedStations.map(station => (
              <option key={station.stationId} value={station.stationId}>
                {station.name} ({station.stationId})
              </option>
            ))}
          </select>

          {selectedStation && selectedStationObj && (
            <div className="station-info">
              <div className="info-row">
                <span className="label">Protocol:</span>
                <span className={`protocol-badge ${isOcpp201 ? 'ocpp201' : 'ocpp16'}`}>
                  {protocolVersion.toUpperCase()}
                </span>
              </div>
              <div className="info-row">
                <span className="label">Status:</span>
                <span className="value status-connected">Connected</span>
              </div>
              {isOcpp201 && (
                <div className="protocol-note">
                  Using OCPP 2.0.1 message templates (TransactionEvent, IdToken, etc.)
                </div>
              )}
            </div>
          )}

          {connectedStations.length === 0 && (
            <div className="warning">
              ⚠️ No connected stations found. Start a station first.
            </div>
          )}
        </div>

        {/* Message Type */}
        <div className="crafter-section">
          <h3>2. Message Type</h3>
          <div className="message-type-selector">
            <button
              className={`type-btn ${messageType === 'Call' ? 'active' : ''}`}
              onClick={() => setMessageType('Call')}
            >
              Call (2)
            </button>
            <button
              className={`type-btn ${messageType === 'CallResult' ? 'active' : ''}`}
              onClick={() => setMessageType('CallResult')}
            >
              CallResult (3)
            </button>
            <button
              className={`type-btn ${messageType === 'CallError' ? 'active' : ''}`}
              onClick={() => setMessageType('CallError')}
            >
              CallError (4)
            </button>
          </div>
        </div>

        {/* Message Details */}
        {messageType === 'Call' && (
          <div className="crafter-section">
            <h3>3. Action & Payload</h3>
            <div className="form-group">
              <label>Action</label>
              <select
                value={action}
                onChange={(e) => handleActionChange(e.target.value)}
                className="action-select"
              >
                {Object.keys(messageTemplates).map(actionName => (
                  <option key={actionName} value={actionName}>{actionName}</option>
                ))}
              </select>
            </div>

            <div className="form-group">
              <label>Unique ID</label>
              <input
                type="text"
                value={uniqueId}
                onChange={(e) => setUniqueId(e.target.value)}
                className="unique-id-input"
              />
            </div>

            <div className="form-group">
              <div className="editor-header">
                <label>Payload (JSON)</label>
                <button
                  type="button"
                  className="btn-format"
                  onClick={formatJSON}
                  title="Format JSON"
                >
                  Format
                </button>
              </div>
              <div className="monaco-editor-container">
                <Editor
                  height="300px"
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
                    formatOnType: true
                  }}
                />
              </div>
              {jsonError && (
                <div className="json-error">
                  ⚠️ {jsonError}
                </div>
              )}
            </div>
          </div>
        )}

        {messageType === 'CallResult' && (
          <div className="crafter-section">
            <h3>3. Response Details</h3>
            <div className="form-group">
              <label>Unique ID (from original Call)</label>
              <input
                type="text"
                value={uniqueId}
                onChange={(e) => setUniqueId(e.target.value)}
                className="unique-id-input"
              />
            </div>

            <div className="form-group">
              <div className="editor-header">
                <label>Result Payload (JSON)</label>
                <button
                  type="button"
                  className="btn-format"
                  onClick={formatJSON}
                  title="Format JSON"
                >
                  Format
                </button>
              </div>
              <div className="monaco-editor-container">
                <Editor
                  height="300px"
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
                    formatOnType: true
                  }}
                />
              </div>
              {jsonError && (
                <div className="json-error">
                  ⚠️ {jsonError}
                </div>
              )}
            </div>
          </div>
        )}

        {messageType === 'CallError' && (
          <div className="crafter-section">
            <h3>3. Error Details</h3>
            <div className="form-group">
              <label>Unique ID (from original Call)</label>
              <input
                type="text"
                value={uniqueId}
                onChange={(e) => setUniqueId(e.target.value)}
                className="unique-id-input"
              />
            </div>

            <div className="form-group">
              <div className="editor-header">
                <label>Error Details (JSON)</label>
                <button
                  type="button"
                  className="btn-format"
                  onClick={formatJSON}
                  title="Format JSON"
                >
                  Format
                </button>
              </div>
              <div className="monaco-editor-container">
                <Editor
                  height="300px"
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
                    formatOnType: true
                  }}
                />
              </div>
              {jsonError && (
                <div className="json-error">
                  ⚠️ {jsonError}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Validation Settings */}
        <div className="crafter-section">
          <h3>4. Validation (Optional)</h3>
          <div className="validation-controls">
            <div className="validation-toggle">
              <label className="toggle-label">
                <input
                  type="checkbox"
                  checked={validationEnabled}
                  onChange={(e) => setValidationEnabled(e.target.checked)}
                />
                <span>Enable OCPP Message Validation</span>
              </label>
            </div>

            {validationEnabled && (
              <div className="validation-mode">
                <label>Validation Mode:</label>
                <div className="mode-selector">
                  <button
                    className={`mode-btn ${validationMode === ValidationMode.STRICT ? 'active' : ''}`}
                    onClick={() => setValidationMode(ValidationMode.STRICT)}
                  >
                    Strict
                  </button>
                  <button
                    className={`mode-btn ${validationMode === ValidationMode.LENIENT ? 'active' : ''}`}
                    onClick={() => setValidationMode(ValidationMode.LENIENT)}
                  >
                    Lenient
                  </button>
                </div>
                <p className="mode-description">
                  {validationMode === ValidationMode.STRICT
                    ? 'Enforce full OCPP spec compliance - all errors and warnings must be resolved'
                    : 'Allow testing of edge cases - warnings are allowed, only errors block sending'}
                </p>
              </div>
            )}

            {validationEnabled && validationResult && (
              <div className={`validation-result ${
                validationResult.valid ? 'valid' :
                validationResult.errors.length > 0 ? 'error' : 'warning'
              }`}>
                <div className="validation-header">
                  {validationResult.valid && validationResult.errors.length === 0 && validationResult.warnings.length === 0 && (
                    <span>✅ Message is valid</span>
                  )}
                  {validationResult.errors.length > 0 && (
                    <span>❌ {validationResult.errors.length} error{validationResult.errors.length > 1 ? 's' : ''} found</span>
                  )}
                  {validationResult.errors.length === 0 && validationResult.warnings.length > 0 && (
                    <span>⚠️ {validationResult.warnings.length} warning{validationResult.warnings.length > 1 ? 's' : ''}</span>
                  )}
                </div>

                {validationResult.errors.length > 0 && (
                  <div className="validation-messages">
                    <strong>Errors:</strong>
                    <ul>
                      {validationResult.errors.map((err, i) => (
                        <li key={i}>
                          <code>{err.field}</code>: {err.message}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {validationResult.warnings.length > 0 && (
                  <div className="validation-messages">
                    <strong>Warnings:</strong>
                    <ul>
                      {validationResult.warnings.map((warn, i) => (
                        <li key={i}>
                          <code>{warn.field}</code>: {warn.message}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Message Preview */}
        <div className="crafter-section">
          <h3>Message Preview</h3>
          <pre className="message-preview">
            {formatMessage()}
          </pre>
        </div>

        {/* Send Button */}
        <div className="crafter-section">
          <button
            className="btn-send"
            onClick={handleSend}
            disabled={!selectedStation || sending}
          >
            {sending ? 'Sending...' : '📤 Send Message'}
          </button>
        </div>

        {/* Result */}
        {result && (
          <div className={`result ${result.success ? 'success' : 'error'}`}>
            {result.success ? (
              <>
                <div className="result-header">✅ {result.message}</div>
                <div className="result-details">
                  <strong>Sent:</strong>
                  <pre>{JSON.stringify(result.sentMessage, null, 2)}</pre>
                </div>
              </>
            ) : (
              <div className="result-header">❌ {result.error}</div>
            )}
          </div>
        )}
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
