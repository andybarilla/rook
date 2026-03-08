# DaisyUI + Tailwind CSS Integration Design

**Date:** 2026-03-04
**Status:** Proposed
**Roadmap item:** Integrate DaisyUI + Tailwind CSS (framework-agnostic component classes + dark theme)

## Context

The Rook frontend is a Svelte 3 app running under Wails (plain Vite, no SvelteKit). All styling is hand-written CSS (~200 lines across 4 components + global styles). As we add UI/UX improvements (error UX, form improvements, accessibility), a component library will accelerate development.

DaisyUI was chosen because:
- Pure CSS — no JS dependency, works with any Svelte version including 3.x
- Framework-agnostic — no SvelteKit requirement (unlike Skeleton UI)
- Built on Tailwind — utility classes for custom tweaks
- Built-in dark theme that closely matches the existing color scheme
- Pre-built component classes: alerts, toasts, modals, tables, badges, buttons, forms

## Installation

Add dev dependencies:
- `tailwindcss` + `postcss` + `autoprefixer`
- `daisyui`

Create `tailwind.config.js`:
- Content paths: `./index.html`, `./src/**/*.{svelte,js}`
- DaisyUI plugin with `dark` theme as default
- Customize primary color to match existing green accent (`#66cc99`)

Create `postcss.config.js` with Tailwind and Autoprefixer plugins.

Add Tailwind directives (`@tailwind base/components/utilities`) to `style.css`.

## Component Migration

Replace hand-written CSS with DaisyUI classes:

| Current | DaisyUI Replacement |
|---|---|
| `.global-error` (red banner) | `alert alert-error` |
| `.site-table` / `.service-table` | `table table-zebra` |
| `.btn-add` (green button) | `btn btn-success btn-sm` |
| `.btn-remove` | `btn btn-ghost btn-sm` |
| `.btn-action` (start/stop) | `btn btn-outline btn-sm` |
| `.add-form` (bordered box) | `card bg-base-200` |
| `input[type="text"]` | `input input-bordered input-sm` |
| `checkbox` | `checkbox checkbox-sm` |
| `.status-running` / `.status-stopped` | `badge badge-success` / `badge badge-ghost` |
| `.empty` placeholder text | `text-base-content/50` |

## Theme Configuration

Use DaisyUI's built-in `dark` theme as the sole theme. Customize:
- `primary`: `#66cc99` (existing green accent)
- Keep other dark theme defaults (they match the current `rgba(27,38,54)` background)

## What Stays the Same

- Component structure: App.svelte, SiteList.svelte, AddSiteForm.svelte, ServiceList.svelte
- All Svelte logic (props, events, reactive statements)
- Layout structure and hierarchy
- Wails bindings and backend integration

## What Changes

- Scoped `<style>` blocks get simplified or removed as styling moves to class attributes
- `style.css` gets Tailwind directives, loses hand-written resets (Tailwind handles this)
- HTML markup gains DaisyUI/Tailwind class names

## Benefits for Subsequent Tasks

This integration directly enables:
- **Error UX task**: Use DaisyUI `alert` + `toast` components
- **Form improvements**: Use DaisyUI `modal` for delete confirmation, `card` for collapsible form
- **Accessibility**: DaisyUI components have better default contrast ratios and ARIA attributes
