# Rename Rook to Rook — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rename the project from "Rook" to "Rook" across the entire codebase — Go module, imports, plugin IDs, config paths, frontend, CI, and documentation.

**Architecture:** This is a mechanical rename with no behavioral changes. The rename touches Go module paths, string constants, frontend localStorage keys, CI artifact names, and documentation. Each task is independent once the Go module path is updated (Task 1), which all other Go files depend on.

**Tech Stack:** Go, Svelte, Wails, GitHub Actions

---

### Task 1: Rename Go module and update all import paths

**Files:**
- Modify: `go.mod:1`
- Modify: All `.go` files with `github.com/andybarilla/rook` imports

**Step 1: Update go.mod module path**

In `go.mod`, change line 1:
```
module github.com/andybarilla/rook
```
to:
```
module github.com/andybarilla/rook
```

**Step 2: Find and replace all Go import paths**

Run project-wide find-and-replace across all `.go` files:
- `github.com/andybarilla/rook` → `github.com/andybarilla/rook`

This affects ~50 files with import statements like:
```go
"github.com/andybarilla/rook/internal/config"
```
→
```go
"github.com/andybarilla/rook/internal/config"
```

**Step 3: Verify the build compiles**

Run: `cd /home/andy/dev/andybarilla/rook && go build ./...`
Expected: Clean build with no errors

**Step 4: Run all Go tests**

Run: `cd /home/andy/dev/andybarilla/rook && go test ./...`
Expected: All tests pass

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: rename Go module from rook to rook"
```

---

### Task 2: Rename app constants and plugin IDs

**Files:**
- Modify: `internal/config/paths.go:9,36`
- Modify: `main.go:20`
- Modify: `app.go:39`
- Modify: `internal/ssl/ssl.go` (ID method)
- Modify: `internal/php/php.go` (ID method)
- Modify: `internal/node/node.go` (ID method)
- Modify: `internal/databases/databases.go` (ID method)

**Step 1: Update appName constant**

In `internal/config/paths.go`:
- Line 9: `const appName = "rook"` → `const appName = "rook"`
- Line 36: `"rook.log"` → `"rook.log"`

**Step 2: Update window title**

In `main.go`:
- Line 20: `Title: "rook"` → `Title: "rook"`

**Step 3: Update log prefix**

In `app.go`:
- Line 39: `"[rook] "` → `"[rook] "`

**Step 4: Update plugin IDs**

In each plugin file, update the `ID()` method return value:
- `internal/ssl/ssl.go`: `"rook-ssl"` → `"rook-ssl"`
- `internal/php/php.go`: `"rook-php"` → `"rook-php"`
- `internal/node/node.go`: `"rook-node"` → `"rook-node"`
- `internal/databases/databases.go`: `"rook-databases"` → `"rook-databases"`

**Step 5: Update test assertions**

Update all test files that assert on plugin IDs or the old name:
- `internal/ssl/ssl_test.go`: `"rook-ssl"` → `"rook-ssl"`
- `internal/php/php_test.go`: `"rook-php"` → `"rook-php"`
- `internal/node/node_test.go`: `"rook-node"` → `"rook-node"`
- `internal/databases/databases_test.go`: `"rook-databases"` → `"rook-databases"`
- `internal/core/core_test.go`: all plugin ID references
- `internal/discovery/discovery_test.go`: manifest plugin ID references
- `internal/external/plugin_test.go`: mock manifest references
- `internal/config/paths_test.go`: any path assertions containing "rook"

**Step 6: Run all Go tests**

Run: `cd /home/andy/dev/andybarilla/rook && go test ./...`
Expected: All tests pass

**Step 7: Commit**

```bash
git add -A
git commit -m "refactor: rename app constants and plugin IDs to rook"
```

---

### Task 3: Update configuration and CI files

**Files:**
- Modify: `wails.json:3-4`
- Modify: `.github/workflows/release.yml:19,22,25`

**Step 1: Update wails.json**

```json
"name": "rook" → "name": "rook"
"outputfilename": "rook" → "outputfilename": "rook"
```

**Step 2: Update release workflow archive names**

In `.github/workflows/release.yml`:
- Line 19: `rook-linux-amd64.tar.gz` → `rook-linux-amd64.tar.gz`
- Line 22: `rook-darwin-arm64.tar.gz` → `rook-darwin-arm64.tar.gz`
- Line 25: `rook-windows-amd64.zip` → `rook-windows-amd64.zip`

**Step 3: Commit**

```bash
git add wails.json .github/workflows/release.yml
git commit -m "refactor: rename build config and CI artifacts to rook"
```

---

### Task 4: Update frontend files

**Files:**
- Modify: `frontend/index.html:6`
- Modify: `frontend/src/lib/theme.js:7,17`
- Modify: `frontend/src/lib/theme.test.js`
- Modify: `frontend/src/SiteList.svelte:19,23`
- Modify: `frontend/src/ServiceList.svelte:42`

**Step 1: Update HTML title**

In `frontend/index.html`:
- `<title>rook</title>` → `<title>rook</title>`

**Step 2: Update localStorage keys**

In `frontend/src/lib/theme.js`:
- `'rook-theme'` → `'rook-theme'` (all occurrences)

In `frontend/src/SiteList.svelte`:
- `'rook-view'` → `'rook-view'` (all occurrences)

**Step 3: Update GitHub URL**

In `frontend/src/ServiceList.svelte`:
- `https://github.com/andybarilla/rook#services` → `https://github.com/andybarilla/rook#services`

**Step 4: Update frontend tests**

In `frontend/src/lib/theme.test.js`:
- `'rook-theme'` → `'rook-theme'` (all occurrences)

**Step 5: Run frontend tests**

Run: `cd /home/andy/dev/andybarilla/rook/frontend && npm test`
Expected: All tests pass

**Step 6: Commit**

```bash
git add frontend/
git commit -m "refactor: rename frontend references to rook"
```

---

### Task 5: Update README, CLAUDE.md, and LEARNINGS.md

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `LEARNINGS.md`

**Step 1: Update README.md**

Find-and-replace across the file:
- `Rook` → `Rook` (capitalized project name)
- `rook` → `rook` (lowercase, in URLs, paths, code references)
- `github.com/andybarilla/rook` → `github.com/andybarilla/rook`
- `getrook.dev` → `getrook.dev`

Review each replacement manually — some context-dependent replacements may need care.

**Step 2: Update CLAUDE.md**

- `Rook` → `Rook` in the project description

**Step 3: Update LEARNINGS.md**

- `rook` → `rook` where it references the app name, log prefixes, etc.

**Step 4: Commit**

```bash
git add README.md CLAUDE.md LEARNINGS.md
git commit -m "docs: rename project references to Rook"
```

---

### Task 6: Update ROADMAP.md and task files

**Files:**
- Modify: `docs/ROADMAP.md`
- Modify: `docs/tasks/*.md` (all task files referencing rook)

**Step 1: Update ROADMAP.md**

Find-and-replace:
- `Rook` → `Rook`
- `rook-ssl` → `rook-ssl`
- `rook-php` → `rook-php`
- `rook-node` → `rook-node`
- `rook-databases` → `rook-databases`
- `rook-sdk` → `rook-sdk`
- `rook-cli` → `rook-cli`
- File references like `rook-core-design.md` → `rook-core-design.md` (only if files are actually renamed in Task 7)

**Step 2: Update task files**

Find-and-replace `rook` → `rook` in all files under `docs/tasks/`.

**Step 3: Commit**

```bash
git add docs/ROADMAP.md docs/tasks/
git commit -m "docs: rename roadmap and task references to Rook"
```

---

### Task 7: Rename and update design/plan docs

**Files:**
- Rename: All files under `docs/plans/` with "rook" in the filename
- Modify: Content within all plan docs

**Step 1: Rename files with "rook" in the name**

```bash
cd /home/andy/dev/andybarilla/rook/docs/plans
for f in *rook*; do git mv "$f" "${f//rook/rook}"; done
```

Files affected:
- `2026-03-03-rook-core-design.md` → `2026-03-03-rook-core-design.md`
- `2026-03-03-rook-core.md` → `2026-03-03-rook-core.md`
- `2026-03-04-rook-databases-design.md` → `2026-03-04-rook-databases-design.md`
- `2026-03-04-rook-databases.md` → `2026-03-04-rook-databases.md`
- `2026-03-04-rook-node-design.md` → `2026-03-04-rook-node-design.md`
- `2026-03-04-rook-node-plan.md` → `2026-03-04-rook-node-plan.md`
- `2026-03-04-rook-php-design.md` → `2026-03-04-rook-php-design.md`
- `2026-03-04-rook-php.md` → `2026-03-04-rook-php.md`
- `2026-03-04-rook-ssl-design.md` → `2026-03-04-rook-ssl-design.md`
- `2026-03-04-rook-ssl.md` → `2026-03-04-rook-ssl.md`
- `2026-03-08-rook-cli-design.md` → `2026-03-08-rook-cli-design.md`
- `2026-03-08-rook-cli-implementation.md` → `2026-03-08-rook-cli-implementation.md`

**Step 2: Find and replace content within all plan docs**

Across all `docs/plans/*.md` files:
- `rook` → `rook` (lowercase)
- `Rook` → `Rook` (capitalized)
- `getrook.dev` → `getrook.dev`

**Step 3: Update cross-references in ROADMAP.md and task files**

Update any file references in `docs/ROADMAP.md` and `docs/tasks/` that point to renamed plan files.

**Step 4: Commit**

```bash
git add docs/plans/ docs/ROADMAP.md docs/tasks/
git commit -m "docs: rename plan files and update content references to Rook"
```

---

### Task 8: Rename task files with "rook" in the name

**Files:**
- Rename: `docs/tasks/009-rook-databases.md` → `docs/tasks/009-rook-databases.md`
- Rename: `docs/tasks/011-rook-node.md` → `docs/tasks/011-rook-node.md`
- Modify: `docs/ROADMAP.md` (update references to renamed task files)

**Step 1: Rename task files**

```bash
cd /home/andy/dev/andybarilla/rook/docs/tasks
git mv 009-rook-databases.md 009-rook-databases.md
git mv 011-rook-node.md 011-rook-node.md
```

**Step 2: Update ROADMAP.md references**

Update the roadmap entries that reference these task files.

**Step 3: Commit**

```bash
git add docs/tasks/ docs/ROADMAP.md
git commit -m "docs: rename task files to rook"
```

---

### Task 9: Final verification

**Step 1: Search for any remaining "rook" references**

Run: `grep -ri "rook" --include="*.go" --include="*.json" --include="*.yml" --include="*.html" --include="*.js" --include="*.svelte" --include="*.md" --include="*.toml" .`

Review results — anything remaining should be intentional (e.g., the rename design doc itself) or in `go.sum`.

**Step 2: Build the full project**

Run: `cd /home/andy/dev/andybarilla/rook && wails build -tags webkit2_41`
Expected: Clean build producing `build/bin/rook`

**Step 3: Run all tests**

Run: `go test ./... && cd frontend && npm test`
Expected: All tests pass

**Step 4: Commit any stragglers**

If any remaining references were found and fixed:
```bash
git add -A
git commit -m "refactor: fix remaining rook references"
```
