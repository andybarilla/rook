# UI Redesign — Figma-Inspired Incremental Reskin

**Date:** 2026-03-07
**Status:** Approved
**Approach:** Incremental reskin of existing Svelte + DaisyUI components

## Context

A Figma Make prototype ("Multi-language Herd frontend") was created as visual inspiration for a UI refresh. The Figma uses React + shadcn/ui + Tailwind, but the implementation stays in Svelte 3 + DaisyUI + Tailwind. The Figma is directional, not a pixel-perfect spec.

**Figma reference:** `https://www.figma.com/make/9J1S5mIZ79NJ2JPYmiZBGs/Multi-language-Herd-frontend`

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Framework | Keep Svelte + DaisyUI | No value in rewriting; Figma is just visual reference |
| Navigation | Top tab bar | Lighter than sidebar, fits existing single-page architecture |
| Sites layout | Table + card view with toggle | Best of both worlds |
| Add Site UX | Modal dialog | Cleaner than collapsible form, less visual clutter |
| Services layout | Cards only | Only 3 services max; cards work better than table |
| Settings | Minimal — only backend-supported features | No fake UI |
| Theme | Light + Dark with toggle | DaisyUI theme switching, persisted to localStorage |
| Accent color | Purple (replacing green) | Match Figma direction |
| Approach | Incremental reskin | Preserve existing tests, less risk |

## Layout & Navigation

- Top tab bar with 3 tabs: Sites, Services, Settings
- Small Rook logo/name on the left, tabs next to it
- DaisyUI `tabs` component (bordered or lifted variant)
- Active tab highlighted with purple accent
- Simple reactive variable to show/hide sections (no routing library)
- Version number moved to Settings tab

## Sites Tab

### Header
- "Sites" title + subtitle on the left
- Purple "Add Site" button on the right
- Search input below (filters by domain or path, client-side)

### View Toggle
- Small grid/list icon toggle to switch between card and table views
- Preference persisted to localStorage

### Table View (existing, refreshed)
- Same columns: Domain, Path, PHP Version, Node Version, TLS, Actions
- Improved styling: cleaner row spacing, colored language badges, green/gray status dots for TLS

### Card View (new)
- Responsive grid (1/2/3 columns depending on viewport)
- Each card shows:
  - Language emoji + site name + `.test` URL as clickable link
  - Badges for language version and TLS status
  - Path (truncated)
  - Port info + action buttons (open in browser, remove)
- Language badge colors: purple=PHP, green=Node.js, blue=Python, red=Ruby, cyan=Go

### Add Site Dialog
- DaisyUI `modal` triggered by "Add Site" button
- Same fields as current: Path, Domain, PHP Version, Node Version, TLS checkbox
- Not expanding to Figma's language/framework picker (backend doesn't support those concepts)

### Empty State
- Existing EmptyState component, works in both views

## Services Tab

### Header
- "Services" title + subtitle on the left
- Running count on the right (e.g., "2/3 Running")

### Layout
- Card layout only (no table toggle — max 3 services, cards are better)
- Each card shows:
  - Emoji icon (MySQL: 💾, PostgreSQL: 🐘, Redis: 🔴)
  - Service name + description
  - Status badge (green "running" / gray "stopped")
  - Port number
  - Play/Pause button to start/stop
  - Auto-start badge (display-only, backend doesn't expose toggle)
- Services with `enabled: false` shown dimmed or in separate "Available" section

### Empty State
- Existing EmptyState component for when no services are enabled

## Settings Tab

Intentionally minimal — only features the backend supports today.

### Appearance (Card)
- Light/Dark theme toggle
- DaisyUI theme switching, persisted to localStorage

### About (Card)
- App version
- List of active plugins (display-only)

### Not Included
- Language version management (no backend API)
- Sites directory configuration (no backend API)
- Default domain suffix configuration (no backend API)
- Cache clearing / config export (no backend API)

## Visual Styling & Theme

### Themes
- **Light** (default): White/neutral backgrounds, dark text, purple accent
- **Dark**: Similar to current dark theme but with purple accent instead of green
- Toggle in Settings, persisted to localStorage

### Colors
- Accent: Purple (replaces green #66cc99)
- Language badges: Purple=PHP, Green=Node.js, Blue=Python, Red=Ruby, Cyan=Go
- Status: Green=running, Gray=stopped
- Cards: Subtle border, light hover shadow

### Typography
- Keep Nunito font

### Accessibility
- All existing accessibility work preserved (WCAG AA contrast, keyboard shortcuts, focus management)
- Ensure both light and dark themes meet contrast requirements

## Backend Changes

Minimal backend changes needed:

- **Optional:** Expose `Plugins()` method via Wails binding so Settings can display active plugins
- No other new backend APIs required

## What's Not Changing

- Svelte 3 framework
- DaisyUI component library
- Tailwind CSS
- Wails backend integration
- Existing test infrastructure
- Keyboard shortcuts (Ctrl+N, Escape, Ctrl+Enter)
- Toast notification system
- Confirmation modals for destructive actions
- Error message mapping
