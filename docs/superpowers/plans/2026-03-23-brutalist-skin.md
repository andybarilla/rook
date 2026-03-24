# Brutalist Skin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the indigo/blue-gray visual skin with a brutalist design — black background, pink (#ff2d78) for active states, amber (#d4a017) for attention states, JetBrains Mono font, zero border-radius everywhere.

**Architecture:** Pure CSS/styling changes. Update Tailwind v4 `@theme` tokens in `theme.css`, add `@font-face` declarations, add global border-radius reset, then update Tailwind utility classes in each component to use new tokens. No layout or functionality changes.

**Tech Stack:** React, Tailwind CSS v4, JetBrains Mono (woff2, bundled)

---

### Task 1: Add JetBrains Mono font files

**Files:**
- Create: `cmd/rook-gui/frontend/src/assets/fonts/JetBrainsMono-Regular.woff2`
- Create: `cmd/rook-gui/frontend/src/assets/fonts/JetBrainsMono-SemiBold.woff2`
- Create: `cmd/rook-gui/frontend/src/assets/fonts/JetBrainsMono-Bold.woff2`

- [ ] **Step 1: Create assets/fonts directory and copy font files**

```bash
mkdir -p cmd/rook-gui/frontend/src/assets/fonts
cp ~/dev/andybarilla/jackdaw-brutalist-skin/src/assets/fonts/JetBrainsMono-Regular.woff2 cmd/rook-gui/frontend/src/assets/fonts/
cp ~/dev/andybarilla/jackdaw-brutalist-skin/src/assets/fonts/JetBrainsMono-SemiBold.woff2 cmd/rook-gui/frontend/src/assets/fonts/
cp ~/dev/andybarilla/jackdaw-brutalist-skin/src/assets/fonts/JetBrainsMono-Bold.woff2 cmd/rook-gui/frontend/src/assets/fonts/
```

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/assets/fonts/
git commit -m "chore: add JetBrains Mono woff2 font files"
```

---

### Task 2: Update theme.css — tokens, font-face, border-radius reset

**Files:**
- Modify: `cmd/rook-gui/frontend/src/theme.css`

- [ ] **Step 1: Replace theme.css with brutalist tokens and font setup**

Replace the entire file with:

```css
@import "tailwindcss";

@font-face {
  font-family: 'JetBrains Mono';
  src: url('./assets/fonts/JetBrainsMono-Regular.woff2') format('woff2');
  font-weight: 400;
  font-style: normal;
  font-display: swap;
}

@font-face {
  font-family: 'JetBrains Mono';
  src: url('./assets/fonts/JetBrainsMono-SemiBold.woff2') format('woff2');
  font-weight: 600;
  font-style: normal;
  font-display: swap;
}

@font-face {
  font-family: 'JetBrains Mono';
  src: url('./assets/fonts/JetBrainsMono-Bold.woff2') format('woff2');
  font-weight: 700;
  font-style: normal;
  font-display: swap;
}

@theme {
  /* Backgrounds */
  --color-rook-bg: #000000;
  --color-rook-sidebar: #111111;
  --color-rook-card: #111111;
  --color-rook-input: #0a0a0a;

  /* Borders */
  --color-rook-border: #222222;
  --color-rook-border-active: #ff2d7830;
  --color-rook-border-attention: #d4a01730;

  /* Text */
  --color-rook-text: #d4d4d4;
  --color-rook-text-secondary: #999999;
  --color-rook-muted: #666666;

  /* Status / Accent */
  --color-rook-active: #ff2d78;
  --color-rook-active-hover: #ff4d8e;
  --color-rook-attention: #d4a017;
  --color-rook-success: #3fb950;
  --color-rook-error: #f85149;
  --color-rook-idle: #444444;

  /* Glow effects */
  --shadow-rook-glow-active: 0 0 12px #ff2d7810;
  --shadow-rook-glow-attention: 0 0 12px #d4a01710;
}

*:not(.rounded-full) {
  border-radius: 0 !important;
}

body {
  margin: 0;
  font-family: 'JetBrains Mono', monospace;
  background-color: var(--color-rook-bg);
  color: var(--color-rook-text);
}
```

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/theme.css
git commit -m "feat: brutalist theme tokens, JetBrains Mono font, zero border-radius"
```

---

### Task 3: Update Sidebar component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/Sidebar.tsx`

- [ ] **Step 1: Update color mappings and remove rounded classes**

Changes:
- `borderColors.running`: `border-l-rook-running` → `border-l-rook-active`
- `borderColors.partial`: `border-l-rook-partial` → `border-l-rook-attention`
- `statusText.running`: `text-rook-running` → `text-rook-active`
- `statusText.partial`: `text-rook-partial` → `text-rook-attention`
- Remove `rounded-md` from workspace buttons and add-workspace button (CSS reset handles it, but clean up classes)
- Add glow box-shadow to workspace buttons: `shadow-rook-glow-active` for running, `shadow-rook-glow-attention` for partial

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/Sidebar.tsx
git commit -m "feat: sidebar brutalist colors"
```

---

### Task 4: Update StatusDot component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/StatusDot.tsx`

- [ ] **Step 1: Update color map to brutalist palette**

Changes:
- `running`: `bg-rook-running` → `bg-rook-active`
- `starting`: `bg-rook-partial` → `bg-rook-attention`
- `partial`: `bg-rook-partial` → `bg-rook-attention`
- `crashed`: `bg-rook-crashed` → `bg-rook-error`
- `stopped`: `bg-rook-stopped` → `bg-rook-idle`
- Keep `rounded-full` — status dots stay circular (CSS reset excludes `.rounded-full` via `*:not(.rounded-full)` selector)

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/StatusDot.tsx
git commit -m "feat: status dot brutalist colors, preserve circular shape"
```

---

### Task 5: Update LogViewer component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/LogViewer.tsx`

- [ ] **Step 1: Update colors**

Changes:
- `SERVICE_COLORS`: Replace `text-rook-running` with `text-rook-active`, `text-rook-partial` with `text-rook-attention`, keep the rest but replace `text-blue-400`/`text-purple-400`/`text-cyan-400`/`text-pink-400` all with `text-rook-active` (icon shape distinguishes, not color — same principle applies to log service labels)
- Log background: `bg-[#111122]` → `bg-rook-input` (maps to `#0a0a0a`)
- Active tab border: `border-rook-accent` stays (now points to pink via theme)
- "Jump to bottom" button: `bg-rook-accent` stays (now pink)
- Filter input: remove `rounded`

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/LogViewer.tsx
git commit -m "feat: log viewer brutalist colors"
```

---

### Task 6: Update ServiceList component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/ServiceList.tsx`

- [ ] **Step 1: Update color references**

Changes:
- `text-rook-crashed` on stop link → `text-rook-error`
- `text-rook-running` on start link → `text-rook-active`
- Remove `rounded-md` from service cards
- Add glow box-shadow to service cards based on status: `shadow-rook-glow-active` for running/starting

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/ServiceList.tsx
git commit -m "feat: service list brutalist colors"
```

---

### Task 7: Update ConfirmDialog component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/ConfirmDialog.tsx`

- [ ] **Step 1: Update colors, remove rounded classes**

Changes:
- Default confirm button: `bg-rook-accent hover:bg-rook-accent/80` → `bg-rook-active hover:bg-rook-active-hover`
- Remove `rounded-lg` from dialog, `rounded` from buttons

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/ConfirmDialog.tsx
git commit -m "feat: confirm dialog brutalist colors"
```

---

### Task 8: Update Toast component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/Toast.tsx`

- [ ] **Step 1: Update colors**

Changes:
- `bg-red-600` → `bg-rook-error`
- `bg-green-600` → `bg-rook-success`
- `bg-rook-accent` stays (now pink)
- Remove `rounded-md`

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/Toast.tsx
git commit -m "feat: toast brutalist colors"
```

---

### Task 9: Update ContextMenu component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/ContextMenu.tsx`

- [ ] **Step 1: Update colors**

Changes:
- `text-rook-crashed` → `text-rook-error`
- `hover:bg-red-500/10` → `hover:bg-rook-error/10`
- Remove `rounded-md`

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/ContextMenu.tsx
git commit -m "feat: context menu brutalist colors"
```

---

### Task 10: Update BuildStatusBadge component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/BuildStatusBadge.tsx`

- [ ] **Step 1: Update colors**

Changes:
- `bg-orange-500/20 text-orange-400 border border-orange-500/30` → `bg-rook-attention/20 text-rook-attention border border-rook-attention/30`
- Remove `rounded`

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/BuildStatusBadge.tsx
git commit -m "feat: build badge brutalist colors"
```

---

### Task 11: Update DiscoveryWizard component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/DiscoveryWizard.tsx`

- [ ] **Step 1: Update colors, remove rounded classes**

Changes:
- `bg-rook-accent` on buttons → `bg-rook-active`
- `text-rook-running` on success message → `text-rook-success`
- `text-rook-crashed` on error → `text-rook-error`
- Remove `rounded-lg`, `rounded` from dialog and inputs

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/DiscoveryWizard.tsx
git commit -m "feat: discovery wizard brutalist colors"
```

---

### Task 12: Update RebuildDialog component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/RebuildDialog.tsx`

- [ ] **Step 1: Update colors**

Changes:
- `bg-rook-accent hover:bg-rook-accent/80` → `bg-rook-active hover:bg-rook-active-hover`
- Remove `rounded-lg`, `rounded`

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/RebuildDialog.tsx
git commit -m "feat: rebuild dialog brutalist colors"
```

---

### Task 13: Update DiscoverDiffDialog component

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/DiscoverDiffDialog.tsx`

- [ ] **Step 1: Update colors**

Changes:
- `bg-rook-accent hover:bg-rook-accent/80` → `bg-rook-active hover:bg-rook-active-hover`
- Remove `rounded-lg`, `rounded`

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/DiscoverDiffDialog.tsx
git commit -m "feat: discover diff dialog brutalist colors"
```

---

### Task 14: Update Dashboard page

**Files:**
- Modify: `cmd/rook-gui/frontend/src/pages/Dashboard.tsx`

- [ ] **Step 1: Update colors**

Changes:
- `StatCard` colors: `text-rook-running` → `text-rook-active`, `text-rook-partial` → `text-rook-attention`
- `text-rook-crashed` on reset link → `text-rook-error`
- Remove `rounded-md` from stat cards and port table
- Checkbox `accent-color` — add inline style `style={{ accentColor: 'var(--color-rook-active)' }}`

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/pages/Dashboard.tsx
git commit -m "feat: dashboard brutalist colors"
```

---

### Task 15: Update WorkspaceDetail page

**Files:**
- Modify: `cmd/rook-gui/frontend/src/pages/WorkspaceDetail.tsx`

- [ ] **Step 1: Update colors**

Changes:
- Stop All button: `bg-rook-crashed` → `bg-rook-error`
- Start button: `bg-rook-running text-rook-bg` → `bg-rook-active text-black`
- Active tab border: `border-rook-accent` stays (now pink)
- Re-scan button: remove `rounded`

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/pages/WorkspaceDetail.tsx
git commit -m "feat: workspace detail brutalist colors"
```

---

### Task 16: Update BuildsTab page

**Files:**
- Modify: `cmd/rook-gui/frontend/src/pages/BuildsTab.tsx`

- [ ] **Step 1: Update colors**

Changes:
- `text-rook-running` on ✅ → `text-rook-success`
- `text-orange-400` on ⚠️ and "Needs rebuild" text → `text-rook-attention`
- `text-rook-crashed` → `text-rook-error`
- Remove `rounded-md` from service cards

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/pages/BuildsTab.tsx
git commit -m "feat: builds tab brutalist colors"
```

---

### Task 17: Update ProfileSwitcher and EnvViewer

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/ProfileSwitcher.tsx`
- Modify: `cmd/rook-gui/frontend/src/components/EnvViewer.tsx`

- [ ] **Step 1: Remove rounded classes**

Both components only need `rounded` class removals — colors already use theme tokens that are now correct.

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/ProfileSwitcher.tsx cmd/rook-gui/frontend/src/components/EnvViewer.tsx
git commit -m "feat: profile switcher and env viewer brutalist cleanup"
```

---

### Task 18: Update ManifestEditor

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/ManifestEditor.tsx`

- [ ] **Step 1: Remove rounded class**

Remove `rounded-md` from the card div.

- [ ] **Step 2: Commit**

```bash
git add cmd/rook-gui/frontend/src/components/ManifestEditor.tsx
git commit -m "feat: manifest editor brutalist cleanup"
```

---

### Task 19: Verify build

- [ ] **Step 1: Run frontend build to check for errors**

```bash
cd cmd/rook-gui/frontend && npm run build
```

Expected: Clean build, no errors.

- [ ] **Step 2: Fix any issues found**

- [ ] **Step 3: Final commit if any fixes needed**
