import { useState } from 'react'

/**
 * useSort manages a click-to-sort field/order with localStorage persistence and
 * returns the sorted items. Clicking a field cycles asc -> desc -> none.
 *
 * @param {Array} items - the array to sort (sorted by `item[field]`)
 * @param {string} storageKey - prefix for the persisted keys (`<key>SortBy` / `<key>SortOrder`)
 */
export function useSort(items, storageKey) {
  const byKey = `${storageKey}SortBy`
  const orderKey = `${storageKey}SortOrder`

  const [sortBy, setSortBy] = useState(() => localStorage.getItem(byKey) || null)
  const [sortOrder, setSortOrder] = useState(() => localStorage.getItem(orderKey) || 'asc')

  const handleSort = (field) => {
    if (sortBy === field) {
      // Toggle through: asc -> desc -> none
      if (sortOrder === 'asc') {
        setSortOrder('desc')
        localStorage.setItem(orderKey, 'desc')
      } else {
        setSortBy(null)
        setSortOrder('asc')
        localStorage.removeItem(byKey)
        localStorage.setItem(orderKey, 'asc')
      }
    } else {
      setSortBy(field)
      setSortOrder('asc')
      localStorage.setItem(byKey, field)
      localStorage.setItem(orderKey, 'asc')
    }
  }

  let sortedItems = items
  if (sortBy) {
    sortedItems = [...items].sort((a, b) => {
      let aValue = a[sortBy]
      let bValue = b[sortBy]

      // Case-insensitive string comparison
      if (typeof aValue === 'string') aValue = aValue.toLowerCase()
      if (typeof bValue === 'string') bValue = bValue.toLowerCase()

      if (aValue < bValue) return sortOrder === 'asc' ? -1 : 1
      if (aValue > bValue) return sortOrder === 'asc' ? 1 : -1
      return 0
    })
  }

  return { sortBy, sortOrder, handleSort, sortedItems }
}
