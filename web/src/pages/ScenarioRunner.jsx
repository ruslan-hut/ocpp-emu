import { useState, useEffect, useRef, useCallback } from 'react'
import { scenariosAPI, executionsAPI, stationsAPI } from '../services/api'
import './ScenarioRunner.css'

const WS_URL = import.meta.env.VITE_WS_URL ||
  (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host

function ScenarioRunner() {
  const [scenarios, setScenarios] = useState([])
  const [executions, setExecutions] = useState([])
  const [stations, setStations] = useState([])
  const [selectedScenario, setSelectedScenario] = useState(null)
  const [selectedStation, setSelectedStation] = useState('')
  const [activeExecution, setActiveExecution] = useState(null)
  const [loading, setLoading] = useState(true)
  const [executing, setExecuting] = useState(false)
  const [error, setError] = useState(null)
  const [wsConnected, setWsConnected] = useState(false)
  const [liveMessages, setLiveMessages] = useState([])

  const wsRef = useRef(null)

  // Load scenarios and stations
  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      setLoading(true)
      const [scenariosRes, stationsRes, executionsRes] = await Promise.all([
        scenariosAPI.getAll(),
        stationsAPI.getAll(),
        executionsAPI.getAll({ status: 'running' }),
      ])
      setScenarios(scenariosRes.data || [])
      setStations(stationsRes.data || [])
      setExecutions(executionsRes.data || [])

      // Check if there's an active execution
      const active = (executionsRes.data || []).find(e =>
        e.status === 'running' || e.status === 'paused'
      )
      if (active) {
        setActiveExecution(active)
        // Select the scenario for the active execution
        const scenario = (scenariosRes.data || []).find(s => s.scenarioId === active.scenarioId)
        if (scenario) setSelectedScenario(scenario)
      }
    } catch (err) {
      console.error('Failed to load data:', err)
      setError('Failed to load data')
    } finally {
      setLoading(false)
    }
  }

  // WebSocket connection for real-time updates
  useEffect(() => {
    if (!activeExecution) return

    const stationId = activeExecution.stationId
    const wsUrl = `${WS_URL}/api/ws/messages?stationId=${encodeURIComponent(stationId)}`

    const connect = () => {
      wsRef.current = new WebSocket(wsUrl)

      wsRef.current.onopen = () => {
        setWsConnected(true)
      }

      wsRef.current.onclose = () => {
        setWsConnected(false)
      }

      wsRef.current.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)

          if (data.type === 'ocpp_message') {
            setLiveMessages(prev => [data.message, ...prev].slice(0, 50))
          } else if (data.type === 'scenario_progress') {
            // Update execution progress
            setActiveExecution(prev => {
              if (prev && prev.executionId === data.progress.executionId) {
                return { ...prev, ...data.progress }
              }
              return prev
            })

            // Refresh if completed or failed
            if (data.progress.status === 'completed' || data.progress.status === 'failed') {
              setTimeout(loadData, 1000)
            }
          }
        } catch (err) {
          console.error('WebSocket message parse error:', err)
        }
      }

      wsRef.current.onerror = (err) => {
        console.error('WebSocket error:', err)
      }
    }

    connect()

    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [activeExecution?.executionId, activeExecution?.stationId])

  // Poll for execution updates
  useEffect(() => {
    if (!activeExecution) return

    const pollInterval = setInterval(async () => {
      try {
        const res = await executionsAPI.getById(activeExecution.executionId)
        setActiveExecution(res.data)

        if (res.data.status === 'completed' || res.data.status === 'failed' || res.data.status === 'cancelled') {
          clearInterval(pollInterval)
          setTimeout(loadData, 1000)
        }
      } catch (err) {
        console.error('Failed to poll execution:', err)
      }
    }, 2000)

    return () => clearInterval(pollInterval)
  }, [activeExecution?.executionId])

  const handleExecute = async () => {
    if (!selectedScenario || !selectedStation) {
      setError('Please select a scenario and station')
      return
    }

    try {
      setExecuting(true)
      setError(null)
      setLiveMessages([])
      const res = await scenariosAPI.execute(selectedScenario.scenarioId, selectedStation)
      setActiveExecution(res.data)
    } catch (err) {
      console.error('Failed to execute scenario:', err)
      setError(err.response?.data || 'Failed to execute scenario')
    } finally {
      setExecuting(false)
    }
  }

  const handlePause = async () => {
    if (!activeExecution) return
    try {
      await executionsAPI.pause(activeExecution.executionId)
      const res = await executionsAPI.getById(activeExecution.executionId)
      setActiveExecution(res.data)
    } catch (err) {
      console.error('Failed to pause:', err)
      setError('Failed to pause execution')
    }
  }

  const handleResume = async () => {
    if (!activeExecution) return
    try {
      await executionsAPI.resume(activeExecution.executionId)
      const res = await executionsAPI.getById(activeExecution.executionId)
      setActiveExecution(res.data)
    } catch (err) {
      console.error('Failed to resume:', err)
      setError('Failed to resume execution')
    }
  }

  const handleStop = async () => {
    if (!activeExecution) return
    try {
      await executionsAPI.stop(activeExecution.executionId)
      setActiveExecution(null)
      loadData()
    } catch (err) {
      console.error('Failed to stop:', err)
      setError('Failed to stop execution')
    }
  }

  const getStepStatusIcon = (status) => {
    switch (status) {
      case 'success': return '✓'
      case 'failed': return '✗'
      case 'running': return '●'
      case 'pending': return '○'
      case 'skipped': return '−'
      default: return '○'
    }
  }

  const getStepStatusClass = (status) => {
    return `step-status step-${status}`
  }

  const formatDuration = (ms) => {
    if (!ms) return '-'
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(1)}s`
  }

  const getStatusClass = (status) => {
    switch (status) {
      case 'running': return 'status-running'
      case 'paused': return 'status-paused'
      case 'completed': return 'status-completed'
      case 'failed': return 'status-failed'
      case 'cancelled': return 'status-cancelled'
      default: return 'status-pending'
    }
  }

  if (loading) {
    return (
      <div className="scenario-runner loading">
        <div className="loading-spinner"></div>
        <p>Loading scenarios...</p>
      </div>
    )
  }

  return (
    <div className="scenario-runner">
      <div className="sr-header">
        <h1>Scenario Runner</h1>
        <div className="sr-header-status">
          {activeExecution && (
            <span className={`execution-status ${getStatusClass(activeExecution.status)}`}>
              {activeExecution.status.toUpperCase()}
            </span>
          )}
          <span className={`ws-status ${wsConnected ? 'connected' : 'disconnected'}`}>
            {wsConnected ? '● Live' : '○ Offline'}
          </span>
        </div>
      </div>

      {error && (
        <div className="error-banner">
          <span>{error}</span>
          <button onClick={() => setError(null)}>×</button>
        </div>
      )}

      <div className="sr-content">
        {/* Left Panel - Scenario Selection */}
        <div className="sr-panel sr-scenarios">
          <h2>Scenarios</h2>
          <div className="scenario-list">
            {scenarios.length === 0 ? (
              <p className="empty-message">No scenarios available</p>
            ) : (
              scenarios.map(scenario => (
                <div
                  key={scenario.scenarioId}
                  className={`scenario-item ${selectedScenario?.scenarioId === scenario.scenarioId ? 'selected' : ''}`}
                  onClick={() => setSelectedScenario(scenario)}
                >
                  <div className="scenario-item-header">
                    <span className="scenario-name">{scenario.name}</span>
                    {scenario.isBuiltin && <span className="builtin-badge">Built-in</span>}
                  </div>
                  <p className="scenario-desc">{scenario.description}</p>
                  <div className="scenario-meta">
                    <span>{scenario.steps?.length || 0} steps</span>
                    {scenario.tags && scenario.tags.length > 0 && (
                      <span className="tags">
                        {scenario.tags.slice(0, 3).map(tag => (
                          <span key={tag} className="tag">{tag}</span>
                        ))}
                      </span>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Middle Panel - Execution View */}
        <div className="sr-panel sr-execution">
          <h2>Execution</h2>

          {/* Controls */}
          <div className="execution-controls">
            <select
              value={selectedStation}
              onChange={(e) => setSelectedStation(e.target.value)}
              disabled={activeExecution}
            >
              <option value="">Select Station</option>
              {stations.map(station => (
                <option key={station.stationId} value={station.stationId}>
                  {station.stationId} - {station.name || 'Unnamed'}
                </option>
              ))}
            </select>

            <div className="control-buttons">
              {!activeExecution ? (
                <button
                  className="btn btn-primary"
                  onClick={handleExecute}
                  disabled={!selectedScenario || !selectedStation || executing}
                >
                  {executing ? 'Starting...' : 'Execute'}
                </button>
              ) : (
                <>
                  {activeExecution.status === 'running' && (
                    <button className="btn btn-secondary" onClick={handlePause}>
                      Pause
                    </button>
                  )}
                  {activeExecution.status === 'paused' && (
                    <button className="btn btn-primary" onClick={handleResume}>
                      Resume
                    </button>
                  )}
                  <button className="btn btn-danger" onClick={handleStop}>
                    Stop
                  </button>
                </>
              )}
            </div>
          </div>

          {/* Progress */}
          {activeExecution && (
            <div className="execution-progress">
              <div className="progress-bar">
                <div
                  className="progress-fill"
                  style={{ width: `${activeExecution.percentage || 0}%` }}
                />
              </div>
              <div className="progress-text">
                Step {activeExecution.currentStep + 1} of {activeExecution.totalSteps}
                {activeExecution.currentStepDesc && (
                  <span className="current-step-desc"> - {activeExecution.currentStepDesc}</span>
                )}
              </div>
            </div>
          )}

          {/* Steps */}
          <div className="execution-steps">
            {selectedScenario ? (
              selectedScenario.steps.map((step, index) => {
                const result = activeExecution?.results?.[index]
                return (
                  <div key={index} className={`step ${result?.status || 'pending'}`}>
                    <span className={getStepStatusClass(result?.status || 'pending')}>
                      {getStepStatusIcon(result?.status || 'pending')}
                    </span>
                    <div className="step-content">
                      <div className="step-header">
                        <span className="step-type">{step.type}</span>
                        <span className="step-duration">{formatDuration(result?.duration)}</span>
                      </div>
                      <p className="step-desc">{step.description || `Step ${index + 1}`}</p>
                      {result?.error && (
                        <p className="step-error">{result.error}</p>
                      )}
                    </div>
                  </div>
                )
              })
            ) : (
              <p className="empty-message">Select a scenario to view steps</p>
            )}
          </div>
        </div>

        {/* Right Panel - Live Messages */}
        <div className="sr-panel sr-messages">
          <h2>Live Messages</h2>
          <div className="message-feed">
            {liveMessages.length === 0 ? (
              <p className="empty-message">
                {activeExecution ? 'Waiting for messages...' : 'Start a scenario to see messages'}
              </p>
            ) : (
              liveMessages.map((msg, index) => (
                <div key={index} className={`message-item ${msg.direction}`}>
                  <div className="message-header">
                    <span className={`direction ${msg.direction}`}>
                      {msg.direction === 'sent' ? '→' : '←'}
                    </span>
                    <span className="action">{msg.action || msg.messageType}</span>
                    <span className="time">
                      {new Date(msg.timestamp).toLocaleTimeString()}
                    </span>
                  </div>
                  {msg.payload && (
                    <pre className="message-payload">
                      {JSON.stringify(msg.payload, null, 2)}
                    </pre>
                  )}
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default ScenarioRunner
