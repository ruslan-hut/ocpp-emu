// Escape HTML entities to prevent XSS.
export function escapeHtml(str) {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

// formatPayload pretty-prints a payload as JSON, falling back to String().
export function formatPayload(payload) {
  if (!payload) return 'N/A'
  try {
    return JSON.stringify(payload, null, 2)
  } catch {
    return String(payload)
  }
}

// highlightJSON tokenizes a JSON payload (object or string) and returns HTML with
// <span> wrappers for keys, strings, numbers, booleans, null, and punctuation.
export function highlightJSON(payload) {
  if (!payload) return '<span class="json-null">N/A</span>'

  try {
    const json = typeof payload === 'string' ? payload : JSON.stringify(payload, null, 2)

    let result = ''
    let i = 0

    while (i < json.length) {
      const char = json[i]

      // Whitespace (preserve formatting)
      if (char === ' ' || char === '\n' || char === '\t' || char === '\r') {
        result += char
        i++
        continue
      }

      // Strings (keys or values)
      if (char === '"') {
        let str = '"'
        i++
        while (i < json.length && json[i] !== '"') {
          if (json[i] === '\\' && i + 1 < json.length) {
            str += json[i] + json[i + 1]
            i += 2
          } else {
            str += json[i]
            i++
          }
        }
        str += '"'
        i++

        // Check if this is a key (followed by colon) or a value
        let lookAhead = i
        while (lookAhead < json.length && (json[lookAhead] === ' ' || json[lookAhead] === '\n' || json[lookAhead] === '\t')) {
          lookAhead++
        }

        if (json[lookAhead] === ':') {
          result += `<span class="json-key">${escapeHtml(str)}</span>`
        } else {
          result += `<span class="json-string">${escapeHtml(str)}</span>`
        }
        continue
      }

      // Numbers
      if (char === '-' || (char >= '0' && char <= '9')) {
        let num = ''
        while (i < json.length && /[-0-9.eE+]/.test(json[i])) {
          num += json[i]
          i++
        }
        result += `<span class="json-number">${escapeHtml(num)}</span>`
        continue
      }

      // Booleans and null
      if (json.slice(i, i + 4) === 'true') {
        result += '<span class="json-boolean">true</span>'
        i += 4
        continue
      }
      if (json.slice(i, i + 5) === 'false') {
        result += '<span class="json-boolean">false</span>'
        i += 5
        continue
      }
      if (json.slice(i, i + 4) === 'null') {
        result += '<span class="json-null">null</span>'
        i += 4
        continue
      }

      // Brackets and punctuation
      if (char === '{' || char === '}') {
        result += `<span class="json-brace">${char}</span>`
        i++
        continue
      }
      if (char === '[' || char === ']') {
        result += `<span class="json-bracket">${char}</span>`
        i++
        continue
      }
      if (char === ':') {
        result += '<span class="json-colon">:</span>'
        i++
        continue
      }
      if (char === ',') {
        result += '<span class="json-comma">,</span>'
        i++
        continue
      }

      // Any other character
      result += escapeHtml(char)
      i++
    }

    return result
  } catch {
    return escapeHtml(String(payload))
  }
}
