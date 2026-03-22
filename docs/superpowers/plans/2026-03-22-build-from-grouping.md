# Build-From Grouping in Rebuild Prompt — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Group `build_from` consumers under their source service in the `rook up` rebuild prompt, and remove stale consumer containers before reconnect when their source is being rebuilt.

**Architecture:** Extract the rebuild prompt display into a testable helper function that takes an `io.Writer`, stale services map, and workspace services to produce grouped output. Add a consumer container removal step between `ForceBuild` flag setting and `orch.Reconnect` in `up.go`.

**Tech Stack:** Go stdlib (`sort`, `strings`, `io`, `fmt`, `testing`)

---

### Task 1: Extract and group the rebuild prompt display

**Files:**
- Create: `internal/cli/rebuild_prompt.go`
- Create: `internal/cli/rebuild_prompt_test.go`

- [ ] **Step 1: Write failing tests for `formatRebuildPrompt`**

In `internal/cli/rebuild_prompt_test.go`:

```go
package cli

import (
	"bytes"
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

func TestFormatRebuildPrompt_SingleServiceNoConsumers(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{"api": {"Dockerfile modified"}}
	services := map[string]workspace.Service{
		"api": {Build: "."},
	}
	formatRebuildPrompt(&buf, stale, services, nil)

	got := buf.String()
	want := "1 service(s) need rebuild:\n  - api (Dockerfile modified)\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatRebuildPrompt_WithSingleConsumer(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{"api": {"Dockerfile modified"}}
	services := map[string]workspace.Service{
		"api":    {Build: "."},
		"worker": {BuildFrom: "api"},
	}
	resolved := []string{"api", "worker"}
	formatRebuildPrompt(&buf, stale, services, resolved)

	got := buf.String()
	want := "1 service(s) need rebuild:\n  - api (Dockerfile modified)\n    also used by: worker\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatRebuildPrompt_WithMultipleConsumers(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{"api": {"context files changed"}}
	services := map[string]workspace.Service{
		"api":       {Build: "."},
		"worker":    {BuildFrom: "api"},
		"scheduler": {BuildFrom: "api"},
	}
	resolved := []string{"api", "worker", "scheduler"}
	formatRebuildPrompt(&buf, stale, services, resolved)

	got := buf.String()
	want := "1 service(s) need rebuild:\n  - api (context files changed)\n    also used by: scheduler, worker\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatRebuildPrompt_ConsumerNotInProfile(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{"api": {"Dockerfile modified"}}
	services := map[string]workspace.Service{
		"api":    {Build: "."},
		"worker": {BuildFrom: "api"},
	}
	resolved := []string{"api"} // worker not in profile
	formatRebuildPrompt(&buf, stale, services, resolved)

	got := buf.String()
	want := "1 service(s) need rebuild:\n  - api (Dockerfile modified)\n"
	if got != want {
		t.Errorf("consumer not in profile should be excluded, got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatRebuildPrompt_NoReason(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{"api": {}}
	services := map[string]workspace.Service{
		"api": {Build: "."},
	}
	formatRebuildPrompt(&buf, stale, services, nil)

	got := buf.String()
	want := "1 service(s) need rebuild:\n  - api\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatRebuildPrompt_MultipleStaleSources(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{
		"api":      {"Dockerfile modified"},
		"frontend": {"context files changed"},
	}
	services := map[string]workspace.Service{
		"api":      {Build: "."},
		"worker":   {BuildFrom: "api"},
		"frontend": {Build: "./web"},
		"ssr":      {BuildFrom: "frontend"},
	}
	resolved := []string{"api", "worker", "frontend", "ssr"}
	formatRebuildPrompt(&buf, stale, services, resolved)

	got := buf.String()
	want := "2 service(s) need rebuild:\n  - api (Dockerfile modified)\n    also used by: worker\n  - frontend (context files changed)\n    also used by: ssr\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestFormatRebuildPrompt -v`
Expected: FAIL — `formatRebuildPrompt` undefined

- [ ] **Step 3: Implement `formatRebuildPrompt`**

In `internal/cli/rebuild_prompt.go`:

```go
package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andybarilla/rook/internal/workspace"
)

// formatRebuildPrompt writes the grouped stale-build prompt to w.
// resolvedServices is the list of service names in the active profile;
// if nil, all services in the workspace are considered.
func formatRebuildPrompt(w io.Writer, staleServices map[string][]string, services map[string]workspace.Service, resolvedServices []string) {
	// Build set of resolved service names for filtering
	resolvedSet := make(map[string]bool)
	if resolvedServices != nil {
		for _, name := range resolvedServices {
			resolvedSet[name] = true
		}
	}

	// Build reverse map: source -> sorted consumers (filtered by profile)
	consumers := make(map[string][]string)
	for name, svc := range services {
		if svc.BuildFrom == "" {
			continue
		}
		if _, stale := staleServices[svc.BuildFrom]; !stale {
			continue
		}
		if resolvedServices != nil && !resolvedSet[name] {
			continue
		}
		consumers[svc.BuildFrom] = append(consumers[svc.BuildFrom], name)
	}
	for source := range consumers {
		sort.Strings(consumers[source])
	}

	// Sort stale service names for deterministic output
	names := make([]string, 0, len(staleServices))
	for name := range staleServices {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintf(w, "%d service(s) need rebuild:\n", len(staleServices))
	for _, name := range names {
		reasons := staleServices[name]
		if len(reasons) > 0 {
			fmt.Fprintf(w, "  - %s (%s)\n", name, reasons[0])
		} else {
			fmt.Fprintf(w, "  - %s\n", name)
		}
		if deps := consumers[name]; len(deps) > 0 {
			fmt.Fprintf(w, "    also used by: %s\n", strings.Join(deps, ", "))
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestFormatRebuildPrompt -v`
Expected: PASS (all 6 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/cli/rebuild_prompt.go internal/cli/rebuild_prompt_test.go
git commit -m "feat(cli): add formatRebuildPrompt with build_from grouping"
```

---

### Task 2: Wire `formatRebuildPrompt` into `up.go` and add consumer container removal

**Files:**
- Modify: `internal/cli/up.go`

- [ ] **Step 1: Write failing test for `buildFromConsumers` helper**

Add to `internal/cli/rebuild_prompt_test.go`:

```go
func TestBuildFromConsumers_ReturnsConsumersOfRebuiltSources(t *testing.T) {
	services := map[string]workspace.Service{
		"api":       {Build: ".", ForceBuild: true},
		"worker":    {BuildFrom: "api"},
		"scheduler": {BuildFrom: "api"},
		"frontend":  {Build: "./web"},
	}
	resolved := []string{"api", "worker", "scheduler", "frontend"}

	got := buildFromConsumers(services, resolved)

	if len(got) != 2 {
		t.Fatalf("expected 2 consumers, got %d: %v", len(got), got)
	}
	// Should be sorted
	if got[0] != "scheduler" || got[1] != "worker" {
		t.Errorf("expected [scheduler worker], got %v", got)
	}
}

func TestBuildFromConsumers_ExcludesNonProfileConsumers(t *testing.T) {
	services := map[string]workspace.Service{
		"api":    {Build: ".", ForceBuild: true},
		"worker": {BuildFrom: "api"},
	}
	resolved := []string{"api"} // worker not in profile

	got := buildFromConsumers(services, resolved)

	if len(got) != 0 {
		t.Errorf("expected 0 consumers (worker not in profile), got %v", got)
	}
}

func TestBuildFromConsumers_IgnoresNonRebuiltSources(t *testing.T) {
	services := map[string]workspace.Service{
		"api":    {Build: "."},           // not ForceBuild
		"worker": {BuildFrom: "api"},
	}
	resolved := []string{"api", "worker"}

	got := buildFromConsumers(services, resolved)

	if len(got) != 0 {
		t.Errorf("expected 0 consumers (source not rebuilt), got %v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestBuildFromConsumers -v`
Expected: FAIL — `buildFromConsumers` undefined

- [ ] **Step 3: Implement `buildFromConsumers`**

Add to `internal/cli/rebuild_prompt.go`:

```go
// buildFromConsumers returns sorted names of build_from consumers whose
// source has ForceBuild=true, filtered to only services in resolvedServices.
func buildFromConsumers(services map[string]workspace.Service, resolvedServices []string) []string {
	resolvedSet := make(map[string]bool)
	for _, name := range resolvedServices {
		resolvedSet[name] = true
	}

	var result []string
	for name, svc := range services {
		if svc.BuildFrom == "" {
			continue
		}
		if !resolvedSet[name] {
			continue
		}
		source, ok := services[svc.BuildFrom]
		if !ok || !source.ForceBuild {
			continue
		}
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestBuildFromConsumers -v`
Expected: PASS (all 3 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/cli/rebuild_prompt.go internal/cli/rebuild_prompt_test.go
git commit -m "feat(cli): add buildFromConsumers helper for stale container removal"
```

---

### Task 3: Wire into `up.go`

**Files:**
- Modify: `internal/cli/up.go`

- [ ] **Step 1: Resolve profile early and replace inline prompt**

In `up.go`, the local variable `profile` (line 53) shadows the `profile` package name. Rename it to `profileName` throughout `up.go` (it's used on lines 53-58 and passed to `orch.Up` on line 305).

Then, immediately after the `profileName` block (after line 58), add profile resolution:

```go
			resolvedServices, err := profilepkg.Resolve(*ws, profileName)
			if err != nil {
				return fmt.Errorf("resolving profile %q: %w", profileName, err)
			}
```

Add import alias: `profilepkg "github.com/andybarilla/rook/internal/profile"`

Then replace lines 90-97 (the `fmt.Printf` stale services display block, keeping line 89 "Checking for stale builds...") with:

```go
			fmt.Println()
			formatRebuildPrompt(os.Stdout, staleServices, ws.Services, resolvedServices)
```

Also update the auto-rebuild message (line 120) to use `formatRebuildPrompt` for the missing-image subset, or at minimum keep it as-is since that path only lists source services being rebuilt (not consumers). The grouped display is most valuable in the initial prompt.

- [ ] **Step 2: Add consumer container removal before `orch.Reconnect`**

In `up.go`, between the `--build` flag block (line 298) and `orch := cctx.newOrchestrator(wsName)` (line 300), insert:

```go
			// Remove stale build_from consumer containers so they aren't
			// reconnected with an outdated image.
			staleConsumers := buildFromConsumers(ws.Services, resolvedServices)
			for _, name := range staleConsumers {
				runner.StopContainer(containerPrefix + name)
			}
```

Note: `resolvedServices` is already available from the earlier resolution. `containerPrefix` is already computed at line 166 and is in scope here.

- [ ] **Step 3: Update remaining references**

Update the `orch.Up` call and the "Starting..." print to use `profileName` instead of `profile`:

```go
			fmt.Printf("Starting %s (profile: %s)...\n", wsName, profileName)
			if err := orch.Up(ctx, *ws, profileName); err != nil {
```

- [ ] **Step 4: Run all existing tests**

Run: `go test ./internal/cli/ -v`
Expected: PASS (all existing + new tests)

- [ ] **Step 5: Commit**

```bash
git add internal/cli/up.go
git commit -m "feat(cli): wire grouped rebuild prompt and consumer container removal into up"
```

---

### Task 4: Manual smoke test

- [ ] **Step 1: Verify with a workspace that has `build_from` services**

If you have a workspace with `build_from` services (e.g., kern-app with api/worker sharing a build):

1. Modify the Dockerfile to trigger stale detection
2. Run `rook up` and verify the prompt shows grouped output:
   ```
   1 service(s) need rebuild:
     - api (Dockerfile modified)
       also used by: worker
   ```
3. Accept the rebuild and verify worker's container is recreated with the new image

- [ ] **Step 2: Verify without `build_from` services**

Run `rook up` on a workspace with no `build_from` services and verify the prompt looks identical to before (regression check).
