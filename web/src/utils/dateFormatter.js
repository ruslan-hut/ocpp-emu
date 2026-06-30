// formatTimestamp renders a timestamp as a locale date+time string.
export function formatTimestamp(timestamp) {
  if (!timestamp) return 'N/A'
  return new Date(timestamp).toLocaleString()
}

// formatTimestampCompact renders just the time (24-hour).
export function formatTimestampCompact(timestamp) {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleTimeString('en-US', { hour12: false })
}
