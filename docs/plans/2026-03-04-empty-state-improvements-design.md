# Empty State Improvements — Design

**Date:** 2026-03-04
**Status:** Approved
**Roadmap item:** Empty state improvements (actionable guidance for sites and services)

## Context

Both the Sites and Services sections show plain muted text when empty ("No sites registered. Add one below." / "No database services configured."). New users get no visual cue or clear call-to-action to get started.

## Design

### 1. New Shared Component: `EmptyState.svelte`

**Location:** `frontend/src/lib/EmptyState.svelte`

A reusable centered empty-state component with:
- **Icon slot** — accepts an inline SVG via a named slot
- **`message` prop** — short headline (e.g., "No sites yet")
- **`subtitle` prop** (optional) — secondary guidance text
- **`actionLabel` prop** (optional) — CTA button text; renders a ghost-style button when provided
- **`on:action` event** — dispatched when the CTA button is clicked

Layout: centered `py-10`, icon at 40px muted, message below, subtitle below that, button at bottom.

### 2. Sites Empty State

Replace the `<p>` in `SiteList.svelte` with `<EmptyState>`:
- **Icon:** Globe SVG (inline, simple outline style)
- **Message:** "No sites yet"
- **Subtitle:** "Add your first site to start developing locally."
- **CTA:** "Add Site" button

The CTA dispatches an `action` event from SiteList. App.svelte handles this by opening the AddSiteForm collapse and focusing the Path input.

**AddSiteForm change:** Accept `collapseOpen` as a bindable prop so App.svelte can control it from outside.

### 3. Services Empty State

Replace the `<p>` in `ServiceList.svelte` with `<EmptyState>`:
- **Icon:** Database SVG (inline, simple outline style)
- **Message:** "No services available"
- **Subtitle:** "Install database plugins to manage MySQL, PostgreSQL, and Redis."
- **CTA:** "Setup Guide" button that opens documentation URL via `window.open()` or Wails `BrowserOpenURL`

## Files Changed

- **New:** `frontend/src/lib/EmptyState.svelte` — shared empty state component
- **Modified:** `frontend/src/SiteList.svelte` — use EmptyState, dispatch CTA event
- **Modified:** `frontend/src/ServiceList.svelte` — use EmptyState, link to docs
- **Modified:** `frontend/src/App.svelte` — handle site CTA to open AddSiteForm
- **Modified:** `frontend/src/AddSiteForm.svelte` — expose `collapseOpen` as bindable prop
