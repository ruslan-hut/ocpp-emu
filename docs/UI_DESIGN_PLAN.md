# UI Design Improvement Plan

## Executive Summary

This document outlines a comprehensive UI redesign strategy to transform the OCPP Emulator from a mobile-first vertical layout to a desktop-optimized, information-dense interface while maintaining a seamless, consistent experience across all pages.

---

## Current State Analysis

### Issues Identified

1. **Underutilized Horizontal Space**
   - Main content max-width: 1200px leaves 20-40% blank on wide monitors
   - Station cards minimum 350px wastes space (could fit more per row)
   - Modals capped at 900px when screens are 1920px+
   - No ultra-wide breakpoint (1600px+)

2. **Vertical-Heavy Layout**
   - Single-column forms when 2-3 columns would fit
   - Stacked filters instead of horizontal toolbar
   - Large padding/margins designed for touch (20-24px)
   - Info rows with excessive line-height

3. **Inconsistent Design System**
   - No CSS variables (colors hardcoded across 15+ files)
   - No spacing scale (mix of px/rem values: 4, 8, 10, 12, 15, 16, 20, 24px)
   - Button styles duplicated in multiple CSS files
   - Badge patterns repeated 5+ times without shared component

4. **Inefficient Data Display**
   - Station info uses full-width rows for small values
   - Message list cards too tall for the data shown
   - Connector details expanded by default (should be compact)

5. **Fragmented Dark Theme Implementation**
   - Dark mode scattered across 8 separate CSS files
   - No centralized color management - colors hardcoded in each `@media (prefers-color-scheme: dark)` block
   - Inconsistent dark colors used:
     - Backgrounds: `#1a1a1a`, `#2a2a2a`, `#2d2d2d`, `#3a3a3a` (no standard scale)
     - Text: `#e0e0e0`, `#fff`, `#999`, `#cce5ff` (inconsistent)
     - Borders: `#444`, `#3a3a3a` (varies)
   - Some components missing dark mode entirely (Layout header, some modals)
   - No user toggle - relies solely on system preference
   - Active/focus states sometimes break in dark mode

---

## Dark Theme Analysis

### Current Dark Mode Coverage

| File | Dark Mode | Issues |
|------|-----------|--------|
| `index.css` | Partial | Only light mode defined explicitly |
| `Layout.css` | Missing | Header gradient doesn't adapt |
| `Dashboard.css` | Yes | Basic coverage |
| `Stations.css` | Missing | Cards don't adapt |
| `Messages.css` | Yes | Most comprehensive |
| `MessageCrafter.css` | Yes | Good coverage |
| `StationForm.css` | Yes | Modal adapts |
| `StationConfig.css` | Yes | Tabs adapt |
| `ConnectorCard.css` | Yes | Uses CSS variables (good!) |
| `TemplateLibrary.css` | Yes | Modal adapts |

### Color Audit - Current Dark Theme Colors

```
Background Hierarchy (light → dark):
├── Surface 1: #2a2a2a, #2d2d2d (cards, modals)
├── Surface 2: #1a1a1a (inputs, nested elements)
├── Surface 3: #3a3a3a (hover states)
└── Base: #121212 (page background - implied)

Text Hierarchy:
├── Primary: #ffffff, #e0e0e0
├── Secondary: #999999, #aaaaaa
├── Muted: #666666
└── Accent: #cce5ff (info)

Border Colors:
├── Default: #444444
├── Subtle: #3a3a3a
└── Focus: #667eea (purple accent)

Status Colors (need dark variants):
├── Success: #10b981 → needs dark bg variant
├── Warning: #f59e0b → needs dark bg variant
├── Danger: #ef4444 → needs dark bg variant
└── Info: #3b82f6 → needs dark bg variant
```

### Proposed Dark Theme Color System

```css
/* Dark Theme Colors - Semantic Tokens */
[data-theme="dark"],
.dark {
  /* Backgrounds */
  --bg-base: #0f0f0f;
  --bg-surface-1: #1a1a1a;
  --bg-surface-2: #242424;
  --bg-surface-3: #2e2e2e;
  --bg-elevated: #383838;

  /* Text */
  --text-primary: #f0f0f0;
  --text-secondary: #a0a0a0;
  --text-muted: #666666;
  --text-inverse: #0f0f0f;

  /* Borders */
  --border-default: #333333;
  --border-muted: #262626;
  --border-emphasis: #444444;

  /* Status - Dark Variants (reduced saturation, adjusted lightness) */
  --color-success-dark-bg: #052e16;
  --color-success-dark-text: #4ade80;
  --color-success-dark-border: #166534;

  --color-warning-dark-bg: #422006;
  --color-warning-dark-text: #fbbf24;
  --color-warning-dark-border: #a16207;

  --color-danger-dark-bg: #450a0a;
  --color-danger-dark-text: #f87171;
  --color-danger-dark-border: #991b1b;

  --color-info-dark-bg: #172554;
  --color-info-dark-text: #60a5fa;
  --color-info-dark-border: #1e40af;

  /* Protocol badges - Dark variants */
  --color-ocpp16-dark-bg: #1e3a5f;
  --color-ocpp16-dark-text: #93c5fd;

  --color-ocpp201-dark-bg: #14532d;
  --color-ocpp201-dark-text: #86efac;

  --color-ocpp21-dark-bg: #431407;
  --color-ocpp21-dark-text: #fdba74;

  /* Accent colors */
  --accent-primary: #60a5fa;
  --accent-primary-hover: #3b82f6;

  /* Shadows (more subtle in dark mode) */
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.3);
  --shadow-md: 0 2px 4px rgba(0, 0, 0, 0.4);
  --shadow-lg: 0 4px 8px rgba(0, 0, 0, 0.5);
}
```

### Theme Switching Strategy

```javascript
// Theme modes
type Theme = 'light' | 'dark' | 'system';

// Implementation options:
// 1. CSS class on <html> element: class="dark"
// 2. Data attribute: data-theme="dark"
// 3. CSS custom properties swap

// Recommended: data-theme attribute
<html data-theme="dark">
```

```css
/* Base (light) tokens */
:root {
  --bg-base: #ffffff;
  --bg-surface-1: #f9fafb;
  --text-primary: #111827;
  /* ... */
}

/* Dark theme override */
[data-theme="dark"] {
  --bg-base: #0f0f0f;
  --bg-surface-1: #1a1a1a;
  --text-primary: #f0f0f0;
  /* ... */
}

/* System preference (default when no explicit theme) */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --bg-base: #0f0f0f;
    /* ... dark tokens ... */
  }
}
```

---

## Design System Specification

### 1. CSS Custom Properties (Design Tokens)

```css
:root {
  /* ===== COLORS ===== */
  /* Primary */
  --color-primary-50: #eff6ff;
  --color-primary-100: #dbeafe;
  --color-primary-500: #3b82f6;
  --color-primary-600: #2563eb;
  --color-primary-700: #1d4ed8;

  /* Success */
  --color-success-50: #ecfdf5;
  --color-success-100: #d1fae5;
  --color-success-500: #10b981;
  --color-success-600: #059669;
  --color-success-700: #047857;

  /* Warning */
  --color-warning-50: #fffbeb;
  --color-warning-100: #fef3c7;
  --color-warning-500: #f59e0b;
  --color-warning-600: #d97706;

  /* Danger */
  --color-danger-50: #fef2f2;
  --color-danger-100: #fee2e2;
  --color-danger-500: #ef4444;
  --color-danger-600: #dc2626;

  /* Neutral */
  --color-neutral-50: #f9fafb;
  --color-neutral-100: #f3f4f6;
  --color-neutral-200: #e5e7eb;
  --color-neutral-300: #d1d5db;
  --color-neutral-400: #9ca3af;
  --color-neutral-500: #6b7280;
  --color-neutral-600: #4b5563;
  --color-neutral-700: #374151;
  --color-neutral-800: #1f2937;
  --color-neutral-900: #111827;

  /* Protocol Colors */
  --color-ocpp16: #1565c0;
  --color-ocpp16-bg: #e3f2fd;
  --color-ocpp201: #2e7d32;
  --color-ocpp201-bg: #e8f5e9;
  --color-ocpp21: #e65100;
  --color-ocpp21-bg: #fff3e0;

  /* ===== SPACING SCALE ===== */
  --space-0: 0;
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;

  /* ===== TYPOGRAPHY ===== */
  --font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  --font-mono: 'SF Mono', Monaco, 'Cascadia Code', monospace;

  --text-xs: 0.75rem;    /* 12px */
  --text-sm: 0.8125rem;  /* 13px */
  --text-base: 0.875rem; /* 14px - default for desktop */
  --text-lg: 1rem;       /* 16px */
  --text-xl: 1.125rem;   /* 18px */
  --text-2xl: 1.25rem;   /* 20px */
  --text-3xl: 1.5rem;    /* 24px */

  --font-normal: 400;
  --font-medium: 500;
  --font-semibold: 600;
  --font-bold: 700;

  --leading-tight: 1.25;
  --leading-normal: 1.4;
  --leading-relaxed: 1.5;

  /* ===== BORDERS & RADIUS ===== */
  --radius-sm: 3px;
  --radius-md: 4px;
  --radius-lg: 6px;
  --radius-xl: 8px;

  --border-width: 1px;
  --border-color: var(--color-neutral-200);

  /* ===== SHADOWS ===== */
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 2px 4px rgba(0, 0, 0, 0.1);
  --shadow-lg: 0 4px 8px rgba(0, 0, 0, 0.1);
  --shadow-xl: 0 8px 16px rgba(0, 0, 0, 0.15);

  /* ===== LAYOUT ===== */
  --content-max-width: 1600px;
  --sidebar-width: 240px;
  --header-height: 48px;
  --toolbar-height: 40px;

  /* ===== TRANSITIONS ===== */
  --transition-fast: 0.1s ease;
  --transition-normal: 0.2s ease;
  --transition-slow: 0.3s ease;

  /* ===== Z-INDEX SCALE ===== */
  --z-dropdown: 100;
  --z-sticky: 200;
  --z-modal-backdrop: 900;
  --z-modal: 1000;
  --z-tooltip: 1100;
}

/* Dark Mode Overrides */
@media (prefers-color-scheme: dark) {
  :root {
    --color-neutral-50: #18181b;
    --color-neutral-100: #27272a;
    --color-neutral-200: #3f3f46;
    --color-neutral-300: #52525b;
    --color-neutral-700: #d4d4d8;
    --color-neutral-800: #e4e4e7;
    --color-neutral-900: #fafafa;
    --border-color: var(--color-neutral-300);
  }
}
```

### 2. Breakpoints Strategy

| Breakpoint | Width | Target | Layout Changes |
|------------|-------|--------|----------------|
| `xs` | < 640px | Mobile | Single column, stacked |
| `sm` | 640px+ | Large phone | 2-column grids |
| `md` | 768px+ | Tablet | Sidebar visible |
| `lg` | 1024px+ | Laptop | 3-column grids |
| `xl` | 1280px+ | Desktop | 4-column grids |
| `2xl` | 1536px+ | Large desktop | 5+ columns, expanded panels |
| `3xl` | 1920px+ | Ultra-wide | Max density, side panels |

```css
/* Desktop-first media queries */
@media (max-width: 1536px) { /* 2xl */ }
@media (max-width: 1280px) { /* xl */ }
@media (max-width: 1024px) { /* lg */ }
@media (max-width: 768px)  { /* md - tablet */ }
@media (max-width: 640px)  { /* sm - mobile */ }
```

### 3. Layout System

#### App Shell (Desktop-Optimized)

```
┌─────────────────────────────────────────────────────────────────┐
│ Header (48px) - Compact with logo, nav, quick actions          │
├───────────┬─────────────────────────────────────────────────────┤
│           │                                                     │
│  Sidebar  │              Main Content Area                      │
│  (240px)  │         (flex: 1, max-width: 1600px)               │
│           │                                                     │
│  - Nav    │  ┌─────────────────────────────────────────────┐   │
│  - Quick  │  │ Page Toolbar (40px) - filters, actions      │   │
│    Stats  │  ├─────────────────────────────────────────────┤   │
│           │  │                                             │   │
│           │  │           Page Content                      │   │
│           │  │                                             │   │
│           │  └─────────────────────────────────────────────┘   │
│           │                                                     │
├───────────┴─────────────────────────────────────────────────────┤
│ Status Bar (24px) - connection status, last sync               │
└─────────────────────────────────────────────────────────────────┘
```

#### Key Layout Classes

```css
/* Container with max-width and auto margins */
.container {
  width: 100%;
  max-width: var(--content-max-width);
  margin: 0 auto;
  padding: 0 var(--space-4);
}

/* Flexible content area */
.content {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

/* Page with optional sidebar */
.page-layout {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-4);
}

@media (min-width: 1280px) {
  .page-layout--with-sidebar {
    grid-template-columns: 280px 1fr;
  }

  .page-layout--with-panel {
    grid-template-columns: 1fr 320px;
  }

  .page-layout--three-column {
    grid-template-columns: 240px 1fr 300px;
  }
}

/* Toolbar - horizontal action bar */
.toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  height: var(--toolbar-height);
  padding: 0 var(--space-3);
  background: var(--color-neutral-50);
  border-bottom: var(--border-width) solid var(--border-color);
}

.toolbar__group {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.toolbar__separator {
  width: 1px;
  height: 20px;
  background: var(--border-color);
  margin: 0 var(--space-2);
}

.toolbar__spacer {
  flex: 1;
}
```

---

## Component Specifications

### 1. Compact Data Table

Replace vertical info-rows with horizontal data tables for dense information:

```
┌──────────────────────────────────────────────────────────────────┐
│ Station: CP-001          Model: ModelX         Protocol: 2.0.1  │
│ Status: ● Connected      Vendor: VendorName    Connectors: 2    │
│ URL: wss://csms.example.com/ocpp                                │
└──────────────────────────────────────────────────────────────────┘
```

```css
.data-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--space-1) var(--space-4);
  font-size: var(--text-sm);
}

.data-grid__item {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  min-width: 0;
}

.data-grid__label {
  color: var(--color-neutral-500);
  font-size: var(--text-xs);
  white-space: nowrap;
}

.data-grid__value {
  color: var(--color-neutral-800);
  font-weight: var(--font-medium);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
```

### 2. Compact Station Card

Reduce from 350px minimum to 260px with tighter layout:

```
┌─────────────────────────────────┐
│ CP-001        ● Connected  2.0.1│
├─────────────────────────────────┤
│ ModelX · VendorName             │
│ Connectors: 2 (Type2, CCS)      │
│ wss://csms.example.com/...      │
├─────────────────────────────────┤
│ [Start] [Stop] [Config] [···]  │
└─────────────────────────────────┘
```

```css
.station-card {
  display: flex;
  flex-direction: column;
  background: var(--color-neutral-50);
  border: var(--border-width) solid var(--border-color);
  border-radius: var(--radius-lg);
  font-size: var(--text-sm);
  min-width: 260px;
}

.station-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--space-3);
  border-bottom: var(--border-width) solid var(--border-color);
  gap: var(--space-2);
}

.station-card__title {
  font-weight: var(--font-semibold);
  font-size: var(--text-base);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.station-card__badges {
  display: flex;
  gap: var(--space-1);
  flex-shrink: 0;
}

.station-card__body {
  padding: var(--space-2) var(--space-3);
  flex: 1;
}

.station-card__meta {
  color: var(--color-neutral-600);
  font-size: var(--text-xs);
  line-height: var(--leading-normal);
}

.station-card__url {
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  color: var(--color-neutral-500);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.station-card__actions {
  display: flex;
  gap: var(--space-1);
  padding: var(--space-2) var(--space-3);
  border-top: var(--border-width) solid var(--border-color);
  background: var(--color-neutral-100);
}
```

### 3. Compact Buttons

Three sizes: small (default), medium, large

```css
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-1);
  font-family: inherit;
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  line-height: 1;
  white-space: nowrap;
  border: var(--border-width) solid transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

/* Sizes */
.btn--sm {
  height: 26px;
  padding: 0 var(--space-2);
  font-size: var(--text-xs);
}

.btn--md {
  height: 32px;
  padding: 0 var(--space-3);
}

.btn--lg {
  height: 38px;
  padding: 0 var(--space-4);
  font-size: var(--text-base);
}

/* Variants */
.btn--primary {
  background: var(--color-primary-500);
  color: white;
}

.btn--primary:hover {
  background: var(--color-primary-600);
}

.btn--secondary {
  background: var(--color-neutral-100);
  border-color: var(--border-color);
  color: var(--color-neutral-700);
}

.btn--secondary:hover {
  background: var(--color-neutral-200);
}

.btn--success {
  background: var(--color-success-500);
  color: white;
}

.btn--danger {
  background: var(--color-danger-500);
  color: white;
}

.btn--ghost {
  background: transparent;
  color: var(--color-neutral-600);
}

.btn--ghost:hover {
  background: var(--color-neutral-100);
}

/* Icon-only button */
.btn--icon {
  width: 32px;
  padding: 0;
}

.btn--icon.btn--sm {
  width: 26px;
}
```

### 4. Compact Badges

```css
.badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  height: 20px;
  padding: 0 var(--space-2);
  font-size: var(--text-xs);
  font-weight: var(--font-medium);
  line-height: 1;
  border-radius: var(--radius-sm);
  white-space: nowrap;
}

.badge--sm {
  height: 16px;
  padding: 0 4px;
  font-size: 10px;
}

/* Status badges */
.badge--connected {
  background: var(--color-success-100);
  color: var(--color-success-700);
}

.badge--disconnected {
  background: var(--color-danger-100);
  color: var(--color-danger-700);
}

.badge--connecting {
  background: var(--color-warning-100);
  color: var(--color-warning-700);
}

/* Protocol badges */
.badge--ocpp16 {
  background: var(--color-ocpp16-bg);
  color: var(--color-ocpp16);
}

.badge--ocpp201 {
  background: var(--color-ocpp201-bg);
  color: var(--color-ocpp201);
}

.badge--ocpp21 {
  background: var(--color-ocpp21-bg);
  color: var(--color-ocpp21);
}

/* With dot indicator */
.badge--dot::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}
```

### 5. Compact Form Controls

```css
.form-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.form-field__label {
  font-size: var(--text-xs);
  font-weight: var(--font-medium);
  color: var(--color-neutral-600);
}

.form-field__input,
.form-field__select {
  height: 32px;
  padding: 0 var(--space-2);
  font-size: var(--text-sm);
  border: var(--border-width) solid var(--border-color);
  border-radius: var(--radius-md);
  background: white;
}

.form-field__input:focus,
.form-field__select:focus {
  outline: none;
  border-color: var(--color-primary-500);
  box-shadow: 0 0 0 2px var(--color-primary-100);
}

/* Compact horizontal form */
.form-row {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.form-row .form-field {
  min-width: 120px;
  flex: 1;
}

/* Dense form grid */
.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--space-3);
}

@media (min-width: 1280px) {
  .form-grid--dense {
    grid-template-columns: repeat(4, 1fr);
  }
}
```

### 6. Compact Message List

Replace tall cards with condensed rows:

```
┌────────────────────────────────────────────────────────────────────┐
│ → Heartbeat                    12:45:32  ✓ Success   2ms  [View]  │
│ ← HeartbeatResponse            12:45:32  ✓                        │
│ → StatusNotification           12:45:30  ✓ Success   5ms  [View]  │
│ → TransactionEvent [Started]   12:45:28  ✓ Success  12ms  [View]  │
└────────────────────────────────────────────────────────────────────┘
```

```css
.message-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-sm);
  border-bottom: var(--border-width) solid var(--border-color);
  transition: background var(--transition-fast);
}

.message-row:hover {
  background: var(--color-neutral-50);
}

.message-row__direction {
  width: 16px;
  color: var(--color-neutral-400);
}

.message-row__direction--sent { color: var(--color-success-500); }
.message-row__direction--received { color: var(--color-primary-500); }

.message-row__action {
  font-weight: var(--font-medium);
  min-width: 180px;
}

.message-row__time {
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  color: var(--color-neutral-500);
  min-width: 70px;
}

.message-row__status {
  min-width: 80px;
}

.message-row__latency {
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  color: var(--color-neutral-400);
  min-width: 40px;
  text-align: right;
}

.message-row__actions {
  margin-left: auto;
}
```

---

## Page-Specific Layouts

### Dashboard (Desktop-Optimized)

```
┌────────────────────────────────────────────────────────────────────┐
│ OCPP Emulator                    [Stations] [Messages] [Settings] │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────┐        │
│  │ Stations │ Connected│ Charging │ Messages │ Errors   │        │
│  │    10    │     8    │     3    │   1,234  │    12    │        │
│  └──────────┴──────────┴──────────┴──────────┴──────────┘        │
│                                                                    │
│  ┌─────────────────────────────────┬───────────────────────────┐  │
│  │ Active Stations                 │ Recent Messages           │  │
│  │ ┌─────┬─────┬─────┬─────┐      │ → Heartbeat      12:45   │  │
│  │ │CP001│CP002│CP003│CP004│      │ ← HeartbeatResp  12:45   │  │
│  │ │ ●On │ ●On │ ●Off│ ●Chg│      │ → StatusNotif    12:44   │  │
│  │ └─────┴─────┴─────┴─────┘      │ → TransactionEv  12:43   │  │
│  │ ┌─────┬─────┬─────┬─────┐      │                           │  │
│  │ │CP005│CP006│CP007│CP008│      │ [View All Messages →]     │  │
│  │ └─────┴─────┴─────┴─────┘      │                           │  │
│  └─────────────────────────────────┴───────────────────────────┘  │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

### Stations Page (Desktop-Optimized)

```
┌────────────────────────────────────────────────────────────────────┐
│ Stations                      [+ Add] [Import] [Start All] [Stop] │
├────────────────────────────────────────────────────────────────────┤
│ Filter: [All ▾] [Connected ▾] [Protocol ▾]    Search: [________] │
├────────────────────────────────────────────────────────────────────┤
│ ┌─────────────┬─────────────┬─────────────┬─────────────┐        │
│ │ CP-001      │ CP-002      │ CP-003      │ CP-004      │        │
│ │ ● Connected │ ● Connected │ ○ Offline   │ ⚡ Charging │        │
│ │ ModelX 2.0.1│ ModelY 1.6  │ ModelX 2.1  │ ModelZ 2.0.1│        │
│ │ [▶][⚙][···]│ [▶][⚙][···]│ [▶][⚙][···]│ [■][⚙][···]│        │
│ ├─────────────┼─────────────┼─────────────┼─────────────┤        │
│ │ CP-005      │ CP-006      │ CP-007      │ CP-008      │        │
│ │ ○ Offline   │ ● Connected │ ● Connected │ ○ Offline   │        │
│ │ ModelX 1.6  │ ModelY 2.0.1│ ModelX 2.0.1│ ModelZ 1.6  │        │
│ │ [▶][⚙][···]│ [▶][⚙][···]│ [▶][⚙][···]│ [▶][⚙][···]│        │
│ └─────────────┴─────────────┴─────────────┴─────────────┘        │
│                                                                    │
│ Showing 8 of 10 stations                          [1] [2] [→]     │
└────────────────────────────────────────────────────────────────────┘
```

### Messages Page (Desktop-Optimized)

```
┌────────────────────────────────────────────────────────────────────┐
│ Messages                                    [Export] [Clear] [⚙]  │
├────────────────────────────────────────────────────────────────────┤
│ Station:[All▾] Direction:[All▾] Action:[All▾] [________] [Search]│
├────────────────────────────────────────────────────────────────────┤
│  Dir │ Action              │ Station │ Time     │ Status │ Resp  │
│ ─────┼─────────────────────┼─────────┼──────────┼────────┼───────│
│  →   │ Heartbeat           │ CP-001  │ 12:45:32 │ ✓ OK   │  2ms  │
│  ←   │ HeartbeatResponse   │ CP-001  │ 12:45:32 │        │       │
│  →   │ StatusNotification  │ CP-002  │ 12:45:30 │ ✓ OK   │  5ms  │
│  →   │ TransactionEvent    │ CP-001  │ 12:45:28 │ ✓ OK   │ 12ms  │
│  ←   │ TransactionEventRes │ CP-001  │ 12:45:28 │        │       │
│  →   │ Authorize           │ CP-003  │ 12:45:25 │ ✓ OK   │  8ms  │
├────────────────────────────────────────────────────────────────────┤
│                              Selected Message Details              │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │ {                                                            │ │
│  │   "timestamp": "2024-01-15T12:45:32Z",                      │ │
│  │   "currentTime": "2024-01-15T12:45:32Z"                     │ │
│  │ }                                                            │ │
│  └──────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
```

### Message Crafter (Desktop-Optimized)

```
┌────────────────────────────────────────────────────────────────────┐
│ Message Crafter                           [Templates] [History]   │
├────────────────────────────────────────────────────────────────────┤
│ Station: [CP-001 ▾]  Action: [Heartbeat ▾]  Type: [Call ▾]       │
├──────────────────────────────────┬─────────────────────────────────┤
│ Payload Editor                   │ Response / Preview             │
│ ┌──────────────────────────────┐ │ ┌─────────────────────────────┐│
│ │{                             │ │ │ Status: ✓ Sent Successfully ││
│ │                              │ │ │ Response Time: 12ms         ││
│ │}                             │ │ ├─────────────────────────────┤│
│ │                              │ │ │{                            ││
│ │                              │ │ │  "currentTime": "2024..."   ││
│ │                              │ │ │}                            ││
│ └──────────────────────────────┘ │ └─────────────────────────────┘│
│ [Validate]                       │                                 │
├──────────────────────────────────┴─────────────────────────────────┤
│                                                  [Send Message →] │
└────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Phases

### Phase 1: Foundation (Week 1)
- [ ] Create `design-tokens.css` with all CSS variables
- [ ] Create `utilities.css` with spacing, layout utilities
- [ ] Create `components.css` with button, badge, form styles
- [ ] Update `index.css` to import new design system files
- [ ] Reduce base font-size from 16px to 14px

### Phase 2: Layout System (Week 1-2)
- [ ] Redesign `Layout.jsx` with compact header (48px)
- [ ] Add optional sidebar support
- [ ] Implement toolbar component
- [ ] Update max-width from 1200px to 1600px
- [ ] Add 1536px and 1920px breakpoints

### Phase 3: Component Updates (Week 2)
- [ ] Refactor buttons to use new `.btn` classes
- [ ] Refactor badges to use new `.badge` classes
- [ ] Refactor form controls to compact sizes
- [ ] Create shared `DataGrid` component
- [ ] Create shared `Toolbar` component

### Phase 4: Page Updates (Week 2-3)
- [ ] Redesign Dashboard with compact stat cards
- [ ] Redesign Stations page with 4-column grid
- [ ] Redesign Messages page with table layout
- [ ] Redesign Message Crafter with split view
- [ ] Update all modals to use new component styles

### Phase 5: Polish (Week 3)
- [ ] Dark mode verification and fixes
- [ ] Keyboard navigation improvements
- [ ] Animation/transition refinements
- [ ] Cross-browser testing
- [ ] Performance optimization

---

## Design Principles

1. **Information Density First**
   - Prioritize showing more data in less space
   - Use compact controls (32px height vs 40px)
   - Reduce padding from 20px to 12-16px

2. **Horizontal Space Utilization**
   - Default to multi-column layouts on desktop
   - Use grid with 4+ columns on xl+ screens
   - Side-by-side panels for detail views

3. **Consistent Visual Language**
   - All colors from design tokens (no hardcoded values)
   - All spacing from spacing scale
   - Consistent border-radius and shadows

4. **Progressive Enhancement**
   - Desktop-first with mobile fallbacks
   - Touch-friendly on tablet (40px min touch targets)
   - Keyboard accessible throughout

5. **Performance**
   - Minimize DOM depth
   - Use CSS Grid/Flexbox (no JS layouts)
   - Lazy load modals and heavy components

---

## File Structure

```
web/src/
├── styles/
│   ├── design-tokens.css    # CSS variables
│   ├── utilities.css        # Utility classes
│   ├── components.css       # Shared component styles
│   └── index.css            # Main entry (imports all)
├── components/
│   ├── ui/
│   │   ├── Button.jsx       # Unified button component
│   │   ├── Badge.jsx        # Unified badge component
│   │   ├── Input.jsx        # Form input component
│   │   ├── Select.jsx       # Form select component
│   │   ├── DataGrid.jsx     # Compact data display
│   │   ├── Toolbar.jsx      # Page toolbar
│   │   └── Modal.jsx        # Unified modal
│   └── layout/
│       ├── Layout.jsx       # App shell
│       ├── Header.jsx       # Compact header
│       ├── Sidebar.jsx      # Optional sidebar
│       └── StatusBar.jsx    # Bottom status bar
└── pages/
    └── ... (existing pages, updated)
```

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Stations visible (1920px) | 4-5 | 6-8 |
| Messages visible without scroll | 8-10 | 15-20 |
| Form fields per row | 1-2 | 3-4 |
| Header height | 60px+ | 48px |
| Button height | 40px | 32px |
| Card padding | 20px | 12px |
| Base font size | 16px | 14px |
| Max content width | 1200px | 1600px |

---

*Document Version: 1.0*
*Created: 2025-12-25*
