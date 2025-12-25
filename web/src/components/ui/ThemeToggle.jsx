import { useTheme } from '../../contexts/ThemeContext'
import './ThemeToggle.css'

/**
 * Theme toggle button with three modes: light, dark, system
 * Click to cycle through modes, with visual indicator for current state
 */
function ThemeToggle({ size = 'md', showLabel = false }) {
  const { theme, mode, cycleTheme } = useTheme()

  const getIcon = () => {
    if (mode === 'system') {
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
          <line x1="8" y1="21" x2="16" y2="21" />
          <line x1="12" y1="17" x2="12" y2="21" />
        </svg>
      )
    }
    if (theme === 'dark') {
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
      )
    }
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="12" r="5" />
        <line x1="12" y1="1" x2="12" y2="3" />
        <line x1="12" y1="21" x2="12" y2="23" />
        <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
        <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
        <line x1="1" y1="12" x2="3" y2="12" />
        <line x1="21" y1="12" x2="23" y2="12" />
        <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
        <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
      </svg>
    )
  }

  const getLabel = () => {
    if (mode === 'system') return 'System'
    if (theme === 'dark') return 'Dark'
    return 'Light'
  }

  const getTitle = () => {
    const current = mode === 'system' ? `System (${theme})` : mode
    const next = mode === 'light' ? 'dark' : mode === 'dark' ? 'system' : 'light'
    return `Theme: ${current}. Click for ${next}`
  }

  return (
    <button
      className={`theme-toggle theme-toggle--${size}`}
      onClick={cycleTheme}
      title={getTitle()}
      aria-label={getTitle()}
    >
      <span className="theme-toggle__icon">
        {getIcon()}
      </span>
      {showLabel && (
        <span className="theme-toggle__label">{getLabel()}</span>
      )}
      {mode === 'system' && (
        <span className="theme-toggle__indicator" title="Following system preference" />
      )}
    </button>
  )
}

export default ThemeToggle
