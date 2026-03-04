# Add Site Form Improvements — Design

**Date:** 2026-03-04
**Status:** Approved
**Roadmap items:**
- Add Site form improvements (collapsible form, better layout, confirmation on remove)
- Site table: show Node Version column

## Context

The collapsible form and confirmation modal are already implemented. This task covers the remaining "better layout" improvement plus the Node Version column as a quick win.

## Design

### 1. AddSiteForm — Grouped Sections for Visual Hierarchy

**Problem:** All fields have identical styling — no visual distinction between required and optional fields.

**Solution:** Split fields into two labeled groups inside the collapse:

- **Section 1 — "Site" (required):** Path and Domain fields. Use `input-md` (instead of current `input-sm`) to give them visual prominence.
- **Divider:** A DaisyUI `<div class="divider text-xs">Options</div>` to separate sections.
- **Section 2 — "Options" (optional):** PHP Version, Node Version, and TLS. Keep `input-sm` to visually de-emphasize. Add muted helper text "All optional" near the section label.
- **Submit button** stays at the bottom, unchanged.

This creates a clear top-to-bottom flow: required fields (prominent) -> options (subdued) -> action.

### 2. SiteList — Node Version Column

Add a **Node** column between PHP and TLS in the site table, displaying `site.node_version || '—'`. Mirror the existing PHP column. Update both the data table and the skeleton loading state.

## Files Changed

- `frontend/src/AddSiteForm.svelte` — grouped sections with divider
- `frontend/src/SiteList.svelte` — add Node Version column
