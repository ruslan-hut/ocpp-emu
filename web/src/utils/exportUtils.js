// downloadBlob triggers a browser download of content as a file.
function downloadBlob(content, type, filename) {
  const blob = new Blob([content], { type })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

// escapeCSV quotes a field when it contains a comma, quote, or newline.
function escapeCSV(field) {
  if (field === null || field === undefined) return ''
  const str = String(field)
  if (str.includes(',') || str.includes('"') || str.includes('\n')) {
    return `"${str.replace(/"/g, '""')}"`
  }
  return str
}

// exportMessagesToJSON downloads the given OCPP messages as a JSON file.
export function exportMessagesToJSON(messages) {
  const dataStr = JSON.stringify(messages, null, 2)
  downloadBlob(dataStr, 'application/json', `ocpp-messages-${new Date().toISOString()}.json`)
}

// exportMessagesToCSV downloads the given OCPP messages as a CSV file.
export function exportMessagesToCSV(messages) {
  const headers = [
    'Timestamp',
    'Station ID',
    'Direction',
    'Message Type',
    'Action',
    'Message ID',
    'Protocol Version',
    'Error Code',
    'Error Description',
    'Payload'
  ]

  const rows = messages.map((message) => [
    message.timestamp || '',
    message.stationId || '',
    message.direction || '',
    message.messageType || '',
    message.action || '',
    message.messageId || '',
    message.protocolVersion || '',
    message.errorCode || '',
    message.errorDescription || '',
    message.payload ? JSON.stringify(message.payload) : ''
  ])

  const csvContent = [
    headers.map(escapeCSV).join(','),
    ...rows.map((row) => row.map(escapeCSV).join(','))
  ].join('\n')

  downloadBlob(csvContent, 'text/csv;charset=utf-8;', `ocpp-messages-${new Date().toISOString()}.csv`)
}
