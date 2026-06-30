import { formatTimestampCompact } from '../utils/dateFormatter'

// MessageList renders the filterable list of OCPP messages, or an empty-state
// when there are none. Selection and clear-filters are delegated to the parent.
function MessageList({ messages, filteredMessages, selectedIndex, onSelect, hasActiveFilters, onClearFilters }) {
  if (filteredMessages.length === 0) {
    return (
      <div className="empty-state empty-state--compact">
        <p>{messages.length === 0 ? 'No messages found' : 'No messages match filters'}</p>
        {messages.length > 0 && hasActiveFilters && (
          <button className="btn-link" onClick={onClearFilters}>
            Clear filters
          </button>
        )}
      </div>
    )
  }

  return (
    <div className="messages-list messages-list--compact">
      {filteredMessages.map((message, index) => (
        <div
          key={index}
          className={`message-row ${message.direction} ${selectedIndex === index ? 'selected' : ''}`}
          onClick={() => onSelect(message, index)}
        >
          <span className={`direction-indicator ${message.direction}`}>
            {message.direction === 'sent' ? '→' : '←'}
          </span>
          <span className="message-row__time">
            {formatTimestampCompact(message.timestamp)}
          </span>
          <span className={`message-row__type message-row__type--${message.messageType?.toLowerCase()}`}>
            {message.messageType === 'CallResult' ? 'Result' : message.messageType === 'CallError' ? 'Error' : message.messageType}
          </span>
          <span className="message-row__action">
            {message.action || '-'}
          </span>
          <span className="message-row__station">
            {message.stationId}
          </span>
        </div>
      ))}
    </div>
  )
}

export default MessageList
