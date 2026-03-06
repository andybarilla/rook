# Accessibility Improvements — Design

**Date:** 2026-03-05
**Status:** Approved
**Roadmap item:** Accessibility (color contrast, keyboard shortcuts)

## Context

The app uses DaisyUI dark theme with low-opacity text classes (`/30` through `/60`) that fail WCAG AA contrast requirements. There are no keyboard shortcuts for common actions, and focus management is minimal.

## Design

### 1. Color Contrast — WCAG AA (4.5:1 normal text, 3:1 large/decorative)

Current opacity classes on DaisyUI dark theme (~`#a6adbb` on ~`#1d232a`) fall below WCAG AA thresholds.

Opacity mapping:

| Current | New | Rationale |
|---|---|---|
| `text-base-content/30` | `text-base-content/50` | Decorative icons — 3:1 sufficient |
| `text-base-content/40` | `text-base-content/60` | Subtitles |
| `text-base-content/50` | `text-base-content/70` | Labels, muted text |
| `text-base-content/60` | `text-base-content/70` | Section headers, secondary text |

### 2. Keyboard Shortcuts

| Key | Action | Context |
|---|---|---|
| `Ctrl+N` | Open Add Site form and focus Path input | Always |
| `Escape` | Close Add Site form if open; dismiss top toast | Global (ConfirmModal already handles Escape) |
| `Ctrl+Enter` | Submit Add Site form | When form is open |

Single `svelte:window` keydown handler in `App.svelte`.

### 3. Focus Management

- **Add Site form open:** Focus the Path input automatically (via exposed `focus()` method or `focusOnOpen` prop).
- **ConfirmModal open:** Trap focus within the modal. Focus the Cancel button on open. Tab cycles between Cancel and Confirm only.
- **ConfirmModal close:** Return focus to the triggering Remove button.

## Files Changed

- **Modified:** `frontend/src/App.svelte` — keydown handler, focus orchestration
- **Modified:** `frontend/src/AddSiteForm.svelte` — expose focus method, Escape to close
- **Modified:** `frontend/src/lib/ConfirmModal.svelte` — focus trap, return focus
- **Modified:** `frontend/src/lib/ToastContainer.svelte` — Escape dismisses top toast
- **Modified:** `frontend/src/SiteList.svelte` — contrast fixes
- **Modified:** `frontend/src/ServiceList.svelte` — contrast fixes
- **Modified:** `frontend/src/lib/EmptyState.svelte` — contrast fixes
