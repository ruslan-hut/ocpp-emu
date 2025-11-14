import { useState, useEffect, useRef } from 'react'
import Editor from '@monaco-editor/react'
import { stationsAPI } from '../services/api'
import TemplateLibrary from '../components/TemplateLibrary'
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
  const editorRef = useRef(null)

  // OCPP 1.6 message templates
  const messageTemplates = {
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
  }, [action])

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
  const selectedStationObj = stations.find(s => s.stationId === selectedStation)

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
                <span className="value">{selectedStationObj.protocolVersion}</span>
              </div>
              <div className="info-row">
                <span className="label">Status:</span>
                <span className="value status-connected">Connected</span>
              </div>
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
