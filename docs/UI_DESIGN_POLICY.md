# UI Design Policy

This document defines the mandatory UI design standards for the OCPP Emulator frontend. All new components and modifications must adhere to these guidelines.

---

## Design Principles

1. **Information Density First** - Prioritize showing more data in less space
2. **Horizontal Space Utilization** - Default to multi-column layouts on desktop
3. **Consistent Visual Language** - All values from design tokens, no hardcoded colors/spacing
4. **Desktop-First** - Optimize for desktop with mobile fallbacks
5. **Performance** - Minimize DOM depth, use CSS Grid/Flexbox

---

## Design Tokens

All styling MUST use CSS variables from `web/src/styles/design-tokens.css`. Never hardcode colors, spacing, or other values.

### Color Usage

```css
/* CORRECT */
color: var(--text-primary);
background: var(--bg-surface-1);
border-color: var(--border-emphasis);

/* INCORRECT - Never do this */
color: #111827;
background: #f9fafb;
border-color: #d1d5db;
```

### Semantic Colors

| Token | Light Mode | Dark Mode | Usage |
|-------|------------|-----------|-------|
| `--bg-base` | `#ffffff` | `#0f0f0f` | Page background |
| `--bg-surface-1` | `#f9fafb` | `#1a1a1a` | Cards, panels |
| `--bg-surface-2` | `#f3f4f6` | `#242424` | Nested elements, headers |
| `--bg-surface-3` | `#e5e7eb` | `#2e2e2e` | Hover states, deeper nesting |
| `--text-primary` | `#111827` | `#f0f0f0` | Main text |
| `--text-secondary` | `#4b5563` | `#a0a0a0` | Secondary text |
| `--text-muted` | `#6b7280` | `#737373` | Subtle text, labels |
| `--border-default` | `#e5e7eb` | `#333333` | Standard borders |
| `--border-emphasis` | `#d1d5db` | `#444444` | Prominent borders (cards) |

### Status Colors

Use the color scale tokens for status indicators:

```css
/* Success states */
background: var(--color-success-100);
color: var(--color-success-700);
border-color: var(--color-success-500);

/* Danger states */
background: var(--color-danger-100);
color: var(--color-danger-700);
border-color: var(--color-danger-500);

/* Warning states */
background: var(--color-warning-100);
color: var(--color-warning-700);
border-color: var(--color-warning-500);

/* Info/Primary states */
background: var(--color-primary-100);
color: var(--color-primary-700);
border-color: var(--color-primary-500);
```

### Spacing Scale

Always use spacing tokens (4px base unit):

| Token | Value | Usage |
|-------|-------|-------|
| `--space-1` | 4px | Tight gaps |
| `--space-2` | 8px | Standard gap |
| `--space-3` | 12px | Component padding |
| `--space-4` | 16px | Section padding |
| `--space-6` | 24px | Large spacing |
| `--space-8` | 32px | Section margins |

---

## Component Sizes

### Compact Controls (Desktop-First)

| Component | Height | Notes |
|-----------|--------|-------|
| Button (sm) | 28px | Icon buttons, secondary actions |
| Button (md) | 34px | Primary buttons (default) |
| Button (lg) | 40px | Hero actions only |
| Input/Select | 34px | All form controls |
| Badge (sm) | 16px | Inline indicators |
| Badge (md) | 20px | Standard badges |

### Typography

Base font size is 14px for desktop:

| Token | Size | Usage |
|-------|------|-------|
| `--text-xs` | 12px | Labels, badges |
| `--text-sm` | 13px | Secondary text |
| `--text-base` | 14px | Body text (default) |
| `--text-lg` | 16px | Subheadings |
| `--text-xl` | 18px | Section titles |
| `--text-2xl` | 20px | Page titles |

---

## Border & Radius Standards

### Border Usage

- **Cards and panels**: `1px solid var(--border-emphasis)` - must be visible
- **Nested dividers**: `1px solid var(--border-default)`
- **Selected/Active states**: `1px solid var(--color-primary-500)` with ring
- **Dashed borders**: Only for empty states and dropzones

### Border Radius

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm` | 2px | Badges, small elements |
| `--radius-md` | 4px | Buttons, inputs |
| `--radius-lg` | 6px | Cards, panels |
| `--radius-xl` | 8px | Modals, large containers |

---

## Shadow Standards

Use shadow tokens for elevation:

```css
--shadow-xs: 0 1px 2px rgba(0, 0, 0, 0.05);    /* Subtle lift */
--shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.1);     /* Cards */
--shadow-md: 0 4px 6px rgba(0, 0, 0, 0.1);     /* Dropdowns */
--shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.1);   /* Modals */
```

---

## Layout Standards

### Content Width

- Maximum content width: `1800px` (`--content-max-width`)
- Ultra-wide breakpoint support (1920px+)

### Grid Layouts

Desktop grids should maximize horizontal space:

```css
/* Station cards - 4+ columns on xl screens */
grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));

/* Three-panel layout */
grid-template-columns: 320px 1fr 380px;

/* Form grids - 3-4 columns */
grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
```

### Breakpoints (Desktop-First)

```css
@media (max-width: 1536px) { /* 2xl - Large desktop */ }
@media (max-width: 1280px) { /* xl - Desktop */ }
@media (max-width: 1024px) { /* lg - Small desktop */ }
@media (max-width: 768px)  { /* md - Tablet */ }
@media (max-width: 640px)  { /* sm - Mobile */ }
```

---

## Dark Mode Requirements

### Automatic Theme Support

Dark mode is handled automatically via CSS variables. Do NOT create separate dark mode overrides unless absolutely necessary.

```css
/* CORRECT - Uses tokens that auto-switch */
.card {
  background: var(--bg-surface-1);
  border: 1px solid var(--border-emphasis);
  color: var(--text-primary);
}

/* INCORRECT - Manual overrides */
.card { background: #f9fafb; }
[data-theme="dark"] .card { background: #1a1a1a; }
```

### Exception: Buttons with Colored Backgrounds

**Buttons are the ONE exception** where dark mode overrides are required. This is because:

1. Design tokens like `--color-success-500` are lighter in dark mode (for text visibility)
2. Buttons use these colors as backgrounds with white text
3. Light backgrounds + white text = poor contrast in dark mode

**Solution:** All button dark mode overrides are centralized in `components.css` using design tokens:

```css
/* In components.css - Dark Mode Button Overrides */
/* In dark mode, use 300-400 range for button backgrounds (inverted scale) */
[data-theme="dark"] .btn-primary,
[data-theme="dark"] .btn-save,
[data-theme="dark"] .btn-send {
  background: var(--color-primary-400);  /* #3b82f6 in dark mode */
  color: white;
}

[data-theme="dark"] .btn--success,
[data-theme="dark"] .btn-start {
  background: var(--color-success-400);  /* #10b981 in dark mode */
  color: white;
}
```

**Important:** In dark mode, the color scale is inverted:
- Use `--color-{type}-300` to `--color-{type}-400` for button backgrounds
- Use `--color-{type}-500` to `--color-{type}-600` for text on dark backgrounds

**Rule:** Never add button dark mode overrides in page-specific CSS. All button overrides belong in `components.css`.

### Theme Detection

The system uses `data-theme` attribute and `prefers-color-scheme`:

```css
/* System preference (default) */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) { /* dark tokens */ }
}

/* Explicit theme */
[data-theme="dark"] { /* dark tokens */ }
```

---

## Component Patterns

### Card Pattern

```css
.card {
  background: var(--bg-surface-1);
  border: 1px solid var(--border-emphasis);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xs);
  padding: var(--space-3);
}

.card:hover {
  border-color: var(--color-neutral-400);
  box-shadow: var(--shadow-sm);
}

.card.selected {
  border-color: var(--color-primary-500);
  background: var(--color-primary-100);
  box-shadow: 0 0 0 1px var(--color-primary-500);
}
```

### Button Pattern

**IMPORTANT:** All button styles are centralized in `web/src/styles/components.css`. Use existing classes instead of creating new button styles.

#### Button Class Naming Convention

| Class Pattern | Usage |
|--------------|-------|
| `.btn` | Base button class |
| `.btn--{variant}` | BEM modifier (e.g., `.btn--primary`, `.btn--success`) |
| `.btn-{action}` | Action-specific (e.g., `.btn-save`, `.btn-delete`, `.btn-start`) |
| `.btn-{action}--{size}` | Size variants (e.g., `.btn-send--compact`) |

#### Available Button Variants

| Variant | Light Mode | Dark Mode | Usage |
|---------|-----------|-----------|-------|
| Primary | `#3b82f6` | `#2563eb` | Main actions, submit |
| Success | `#10b981` | `#059669` | Start, save, confirm |
| Danger | `#ef4444` | `#dc2626` | Delete, stop, destructive |
| Warning | `#f59e0b` | `#d97706` | Caution actions |
| Secondary | Surface bg | `#2e2e2e` | Cancel, secondary actions |

#### Base Button Structure

```css
.btn {
  height: var(--btn-height-md);
  padding: 0 var(--space-4);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  transition: all var(--transition-fast);
}

.btn-primary {
  background: var(--color-primary-500);
  color: white;
}

.btn-secondary {
  background: var(--bg-base);
  border: 1px solid var(--border-emphasis);
  color: var(--text-primary);
}
```

#### Adding New Button Styles

When adding new button styles:

1. **Check if an existing class works** - Most buttons fit into primary, success, danger, warning, or secondary
2. **Add to `components.css`** - Never define button styles in page-specific CSS
3. **Include dark mode override** - Buttons with colored backgrounds need explicit dark mode styles (see Dark Mode section)
4. **Use the naming convention** - Follow `.btn-{action}` pattern

### Form Control Pattern

```css
.input {
  height: var(--input-height-md);
  padding: 0 var(--space-3);
  border: 1px solid var(--border-emphasis);
  border-radius: var(--radius-md);
  background: var(--bg-base);
  color: var(--text-primary);
  font-size: var(--text-base);
}

.input:focus {
  border-color: var(--color-primary-500);
  box-shadow: 0 0 0 2px var(--color-primary-100);
  outline: none;
}
```

### Badge Pattern

```css
.badge {
  height: var(--badge-height-md);
  padding: 0 var(--space-2);
  font-size: var(--text-xs);
  font-weight: var(--font-medium);
  border-radius: var(--radius-sm);
  border: 1px solid;
}

.badge-success {
  background: var(--color-success-100);
  color: var(--color-success-700);
  border-color: var(--color-success-500);
}
```

---

## File Organization

```
web/src/
├── styles/
│   ├── design-tokens.css    # All CSS variables (source of truth)
│   ├── utilities.css        # Utility classes
│   └── components.css       # Shared component styles (buttons, badges, cards)
│                            # ↳ Includes ALL button styles and dark mode overrides
├── components/
│   └── ui/                  # Reusable UI components
└── pages/
    └── *.css                # Page-specific styles (use tokens only)
                             # ↳ NO button definitions here - use components.css classes
```

### Style Precedence

1. **design-tokens.css** - Variables only, loaded first
2. **components.css** - Reusable component styles (buttons, forms, badges)
3. **page/*.css** - Page-specific layout and structure only

---

## Checklist for New Components

Before submitting UI changes, verify:

- [ ] All colors use design tokens (no hex values)
- [ ] All spacing uses `--space-*` tokens
- [ ] Borders use `--border-emphasis` for cards, `--border-default` for dividers
- [ ] Component works in both light and dark modes
- [ ] No manual dark mode overrides (except buttons - see Dark Mode section)
- [ ] Follows compact sizing (34px buttons, 14px base font)
- [ ] Responsive breakpoints follow desktop-first approach
- [ ] Shadows use `--shadow-*` tokens
- [ ] Button styles use existing classes from `components.css`
- [ ] New button variants added to `components.css` with dark mode overrides

---

*Policy Version: 1.1*
*Last Updated: 2026-01-02*
