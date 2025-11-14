import { useState } from 'react'
import PropTypes from 'prop-types'
import { stationsAPI } from '../services/api'
import './ImportExport.css'

function ImportExport({ stations, onClose, onImportComplete }) {
  const [importing, setImporting] = useState(false)
  const [importResults, setImportResults] = useState(null)

  const handleExportAll = () => {
    const dataStr = JSON.stringify(stations, null, 2)
    const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr)
    const exportFileDefaultName = `stations-export-${new Date().toISOString().split('T')[0]}.json`

    const linkElement = document.createElement('a')
    linkElement.setAttribute('href', dataUri)
    linkElement.setAttribute('download', exportFileDefaultName)
    linkElement.click()
  }

  const handleExportSingle = (station) => {
    const dataStr = JSON.stringify(station, null, 2)
    const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr)
    const exportFileDefaultName = `station-${station.stationId}.json`

    const linkElement = document.createElement('a')
    linkElement.setAttribute('href', dataUri)
    linkElement.setAttribute('download', exportFileDefaultName)
    linkElement.click()
  }

  const handleImport = async (e) => {
    const file = e.target.files[0]
    if (!file) return

    setImporting(true)
    setImportResults(null)

    const reader = new FileReader()
    reader.onload = async (event) => {
      try {
        const data = JSON.parse(event.target.result)
        const stationsToImport = Array.isArray(data) ? data : [data]

        const results = {
          total: stationsToImport.length,
          successful: 0,
          failed: 0,
          errors: []
        }

        for (const station of stationsToImport) {
          try {
            // Remove MongoDB IDs and runtime state
            const cleanStation = {
              ...station,
              _id: undefined,
              id: undefined,
              runtimeState: undefined,
              createdAt: undefined,
              updatedAt: undefined
            }

            await stationsAPI.create(cleanStation)
            results.successful++
          } catch (err) {
            results.failed++
            results.errors.push({
              stationId: station.stationId || 'Unknown',
              error: err.response?.data?.error || err.message
            })
          }
        }

        setImportResults(results)
        if (results.successful > 0) {
          onImportComplete()
        }
      } catch (err) {
        setImportResults({
          total: 0,
          successful: 0,
          failed: 1,
          errors: [{ stationId: 'N/A', error: 'Invalid JSON file' }]
        })
      } finally {
        setImporting(false)
      }
    }
    reader.readAsText(file)
  }

  return (
    <div className="import-export-overlay">
      <div className="import-export-container">
        <div className="import-export-header">
          <h2>Import / Export Stations</h2>
          <button className="close-btn" onClick={onClose}>×</button>
        </div>

        <div className="import-export-content">
          {/* Export Section */}
          <section className="ie-section">
            <h3>Export Stations</h3>
            <p className="ie-description">
              Export station configurations to JSON files for backup or transfer
            </p>

            <div className="export-actions">
              <button className="btn-primary" onClick={handleExportAll}>
                Export All Stations ({stations.length})
              </button>
            </div>

            {stations.length > 0 && (
              <div className="export-list">
                <h4>Export Individual Stations:</h4>
                <div className="station-list">
                  {stations.map((station) => (
                    <div key={station.stationId} className="station-item">
                      <div className="station-info">
                        <span className="station-id">{station.stationId}</span>
                        <span className="station-name">{station.name}</span>
                      </div>
                      <button
                        className="btn-small"
                        onClick={() => handleExportSingle(station)}
                      >
                        Export
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </section>

          {/* Import Section */}
          <section className="ie-section">
            <h3>Import Stations</h3>
            <p className="ie-description">
              Import station configurations from JSON files. You can import a single station or multiple stations at once.
            </p>

            <div className="import-area">
              <label className="import-button">
                <input
                  type="file"
                  accept=".json"
                  onChange={handleImport}
                  style={{ display: 'none' }}
                  disabled={importing}
                />
                <span>{importing ? 'Importing...' : 'Choose File to Import'}</span>
              </label>
              <p className="import-note">
                Accepts JSON files exported from this system
              </p>
            </div>

            {importResults && (
              <div className={`import-results ${importResults.failed > 0 ? 'has-errors' : 'success'}`}>
                <h4>Import Results</h4>
                <div className="results-summary">
                  <div className="result-item">
                    <span className="label">Total:</span>
                    <span className="value">{importResults.total}</span>
                  </div>
                  <div className="result-item success-item">
                    <span className="label">Successful:</span>
                    <span className="value">{importResults.successful}</span>
                  </div>
                  {importResults.failed > 0 && (
                    <div className="result-item error-item">
                      <span className="label">Failed:</span>
                      <span className="value">{importResults.failed}</span>
                    </div>
                  )}
                </div>

                {importResults.errors.length > 0 && (
                  <div className="import-errors">
                    <h5>Errors:</h5>
                    {importResults.errors.map((err, i) => (
                      <div key={i} className="error-item">
                        <strong>{err.stationId}:</strong> {err.error}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </section>
        </div>

        <div className="import-export-footer">
          <button className="btn-secondary" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  )
}

ImportExport.propTypes = {
  stations: PropTypes.array.isRequired,
  onClose: PropTypes.func.isRequired,
  onImportComplete: PropTypes.func.isRequired
}

export default ImportExport
