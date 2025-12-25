import { createContext, useContext, useState, useEffect, useCallback } from 'react'

/**
 * Theme modes:
 * - 'light': Force light theme
 * - 'dark': Force dark theme
 * - 'system': Follow system preference
 */

const THEME_STORAGE_KEY = 'ocpp-emu-theme'

const ThemeContext = createContext(null)

/**
 * Get the effective theme based on mode and system preference
 */
function getEffectiveTheme(mode) {
  if (mode === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return mode
}

/**
 * Get initial theme mode from localStorage or default to 'system'
 */
function getInitialMode() {
  if (typeof window === 'undefined') return 'system'

  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY)
    if (stored === 'light' || stored === 'dark' || stored === 'system') {
      return stored
    }
  } catch (e) {
    console.warn('Failed to read theme from localStorage:', e)
  }

  return 'system'
}

/**
 * Apply theme to document
 */
function applyTheme(theme) {
  const root = document.documentElement

  // Set data-theme attribute for CSS
  root.setAttribute('data-theme', theme)

  // Update meta theme-color for mobile browsers
  const metaThemeColor = document.querySelector('meta[name="theme-color"]')
  if (metaThemeColor) {
    metaThemeColor.setAttribute('content', theme === 'dark' ? '#0f0f0f' : '#667eea')
  }
}

export function ThemeProvider({ children }) {
  const [mode, setModeState] = useState(getInitialMode)
  const [theme, setTheme] = useState(() => getEffectiveTheme(getInitialMode()))

  // Update effective theme when mode changes
  useEffect(() => {
    const effectiveTheme = getEffectiveTheme(mode)
    setTheme(effectiveTheme)
    applyTheme(effectiveTheme)
  }, [mode])

  // Listen for system theme changes when in 'system' mode
  useEffect(() => {
    if (mode !== 'system') return

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

    const handleChange = (e) => {
      const newTheme = e.matches ? 'dark' : 'light'
      setTheme(newTheme)
      applyTheme(newTheme)
    }

    mediaQuery.addEventListener('change', handleChange)
    return () => mediaQuery.removeEventListener('change', handleChange)
  }, [mode])

  // Apply theme on initial mount
  useEffect(() => {
    applyTheme(theme)
  }, [])

  const setMode = useCallback((newMode) => {
    setModeState(newMode)
    try {
      localStorage.setItem(THEME_STORAGE_KEY, newMode)
    } catch (e) {
      console.warn('Failed to save theme to localStorage:', e)
    }
  }, [])

  // Toggle between light and dark (skipping system)
  const toggleTheme = useCallback(() => {
    setMode(theme === 'dark' ? 'light' : 'dark')
  }, [theme, setMode])

  // Cycle through: light -> dark -> system -> light
  const cycleTheme = useCallback(() => {
    const nextMode = mode === 'light' ? 'dark' : mode === 'dark' ? 'system' : 'light'
    setMode(nextMode)
  }, [mode, setMode])

  const value = {
    theme,      // Current effective theme: 'light' | 'dark'
    mode,       // Current mode: 'light' | 'dark' | 'system'
    setMode,    // Set mode explicitly
    toggleTheme, // Toggle between light/dark
    cycleTheme, // Cycle through light/dark/system
    isDark: theme === 'dark',
    isLight: theme === 'light',
    isSystem: mode === 'system',
  }

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}

export default ThemeContext
