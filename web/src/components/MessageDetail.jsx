import { formatTimestamp } from '../utils/dateFormatter'
import { formatPayload, highlightJSON } from '../utils/jsonFormatter'

// MessageDetail renders the detail panel for a single OCPP message, or an empty
// placeholder when no message is selected.
function MessageDetail({ message, onClose }) {
  if (!message) {
    return (
      <div className="message-detail__empty">
        <div className="empty-detail-icon">📋</div>
        <p>Select a message to view details</p>
      </div>
    )
  }

  return (
    <div className="message-detail">
      <div className="message-detail__header">
        <div className="message-detail__title">
          <span className={`direction-badge direction-badge--lg ${message.direction}`}>
            {message.direction}
          </span>
          <span className="message-detail__action">{message.action || 'N/A'}</span>
          <span className={`message-type-badge message-type-badge--${message.messageType?.toLowerCase()}`}>
            {message.messageType}
          </span>
        </div>
        <button className="btn-close-detail" onClick={onClose}>
          ×
        </button>
      </div>

      <div className="message-detail__meta">
        <div className="meta-row">
          <span className="meta-label">Timestamp</span>
          <span className="meta-value">{formatTimestamp(message.timestamp)}</span>
        </div>
        <div className="meta-row">
          <span className="meta-label">Station ID</span>
          <span className="meta-value meta-value--mono">{message.stationId}</span>
        </div>
        <div className="meta-row">
          <span className="meta-label">Message ID</span>
          <span className="meta-value meta-value--mono">{message.messageId || 'N/A'}</span>
        </div>
        {message.correlationId && (
          <div className="meta-row">
            <span className="meta-label">Correlation ID</span>
            <span className="meta-value meta-value--mono">{message.correlationId}</span>
          </div>
        )}
        <div className="meta-row">
          <span className="meta-label">Protocol</span>
          <span className="meta-value">{message.protocolVersion || 'OCPP 1.6'}</span>
        </div>
        {message.errorCode && (
          <div className="meta-row meta-row--error">
            <span className="meta-label">Error Code</span>
            <span className="meta-value meta-value--error">{message.errorCode}</span>
          </div>
        )}
        {message.errorDescription && (
          <div className="meta-row meta-row--error">
            <span className="meta-label">Error Description</span>
            <span className="meta-value meta-value--error">{message.errorDescription}</span>
          </div>
        )}
      </div>

      <div className="message-detail__payload">
        <div className="payload-header">
          <span className="payload-title">Payload</span>
          <button
            className="btn-copy"
            onClick={() => {
              navigator.clipboard.writeText(formatPayload(message.payload))
            }}
          >
            Copy
          </button>
        </div>
        <pre
          className="payload-content payload-content--highlighted"
          dangerouslySetInnerHTML={{ __html: highlightJSON(message.payload) }}
        />
      </div>
    </div>
  )
}

export default MessageDetail
