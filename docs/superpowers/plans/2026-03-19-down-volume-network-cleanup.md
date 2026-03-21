# Fix `rook down -v` Volume and Network Cleanup

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `rook down -v` remove named volumes associated with workspace containers, and make `rook down` always clean up the workspace network.

**Architecture:** Add two new functions to `internal/runner/`: `ContainerVolumes()` inspects a container to extract its named volume names, and `RemoveNetwork()` removes a network by name. The `down` CLI command orchestrates: stop containers → remove named volumes (if `-v`) → remove network.

**Tech Stack:** Go, `docker`/`podman` CLI (`inspect`, `volume rm`, `network rm`)

---

### Task 1: Add `ContainerVolumes` function

**Files:**
- Modify: `internal/runner/docker.go`
- Test: `internal/runner/docker_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestContainerVolumes(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	runtime := runner.ContainerRuntime
	containerName := "rook-test-cvol"

	// Clean up
	exec.Command(runtime, "rm", "-f", containerName).Run()
	exec.Command(runtime, "volume", "rm", "-f", "rook-test-namedvol").Run()

	// Create container with a named volume and a bind mount
	cmd := exec.Command(runtime, "run", "-d", "--name", containerName,
		"-v", "rook-test-namedvol:/data",
		"-v", "/tmp:/hostmount",
		"alpine:latest", "sleep", "300")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	defer func() {
		exec.Command(runtime, "rm", "-f", containerName).Run()
		exec.Command(runtime, "volume", "rm", "-f", "rook-test-namedvol").Run()
	}()

	vols, err := runner.ContainerVolumes(containerName)
	if err != nil {
		t.Fatalf("ContainerVolumes failed: %v", err)
	}

	// Should contain the named volume, NOT the bind mount
	found := false
	for _, v := range vols {
		if v == "rook-test-namedvol" {
			found = true
		}
		if v == "/tmp" {
			t.Error("bind mount should not be returned as a named volume")
		}
	}
	if !found {
		t.Errorf("expected 'rook-test-namedvol' in volumes, got %v", vols)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/runner/ -run TestContainerVolumes -v`
Expected: FAIL — `runner.ContainerVolumes` undefined

- [ ] **Step 3: Implement `ContainerVolumes`**

Add to `internal/runner/docker.go`:

```go
// ContainerVolumes returns the names of named volumes mounted on a container.
// Bind mounts are excluded — only Docker/Podman-managed volumes are returned.
// Note: relies on .Mounts[].Type == "volume" which requires Podman 4.x+ or any Docker version.
func ContainerVolumes(containerName string) ([]string, error) {
	// Use Go template to extract volume names; named volumes have Type "volume", bind mounts have Type "bind"
	output, err := exec.Command(ContainerRuntime, "inspect",
		"-f", `{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{"\n"}}{{end}}{{end}}`,
		containerName).Output()
	if err != nil {
		return nil, fmt.Errorf("inspecting volumes for %s: %w", containerName, err)
	}
	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/runner/ -run TestContainerVolumes -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/docker.go internal/runner/docker_test.go
git commit -m "feat: add ContainerVolumes to inspect named volumes on a container"
```

---

### Task 2: Add `RemoveVolumes` function

**Files:**
- Modify: `internal/runner/docker.go`
- Test: `internal/runner/docker_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRemoveVolumes(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	runtime := runner.ContainerRuntime

	// Create a volume
	exec.Command(runtime, "volume", "create", "rook-test-rmvol").Run()

	// Verify it exists
	if err := exec.Command(runtime, "volume", "inspect", "rook-test-rmvol").Run(); err != nil {
		t.Fatalf("volume should exist: %v", err)
	}

	runner.RemoveVolumes([]string{"rook-test-rmvol"})

	// Volume should be gone
	if err := exec.Command(runtime, "volume", "inspect", "rook-test-rmvol").Run(); err == nil {
		t.Error("volume should have been removed")
		exec.Command(runtime, "volume", "rm", "-f", "rook-test-rmvol").Run()
	}
}

func TestRemoveVolumes_Empty(t *testing.T) {
	// Should not panic or error on empty input
	runner.RemoveVolumes(nil)
	runner.RemoveVolumes([]string{})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/runner/ -run TestRemoveVolumes -v`
Expected: FAIL — `runner.RemoveVolumes` undefined

- [ ] **Step 3: Implement `RemoveVolumes`**

Add to `internal/runner/docker.go`:

```go
// RemoveVolumes removes the given named volumes.
func RemoveVolumes(names []string) {
	for _, name := range names {
		exec.Command(ContainerRuntime, "volume", "rm", name).Run()
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/runner/ -run TestRemoveVolumes -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/docker.go internal/runner/docker_test.go
git commit -m "feat: add RemoveVolumes for named volume cleanup"
```

---

### Task 3: Add `RemoveNetwork` function

**Files:**
- Modify: `internal/runner/docker.go`
- Test: `internal/runner/docker_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRemoveNetwork(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	runtime := runner.ContainerRuntime
	netName := "rook-test-net"

	// Create a network
	exec.Command(runtime, "network", "create", netName).Run()

	// Verify it exists
	if err := exec.Command(runtime, "network", "inspect", netName).Run(); err != nil {
		t.Fatalf("network should exist: %v", err)
	}

	runner.RemoveNetwork(netName)

	// Network should be gone
	if err := exec.Command(runtime, "network", "inspect", netName).Run(); err == nil {
		t.Error("network should have been removed")
		exec.Command(runtime, "network", "rm", netName).Run()
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/runner/ -run TestRemoveNetwork -v`
Expected: FAIL — `runner.RemoveNetwork` undefined

- [ ] **Step 3: Implement `RemoveNetwork`**

Add to `internal/runner/docker.go`:

```go
// RemoveNetwork removes a container network by name.
func RemoveNetwork(name string) {
	exec.Command(ContainerRuntime, "network", "rm", name).Run()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/runner/ -run TestRemoveNetwork -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/docker.go internal/runner/docker_test.go
git commit -m "feat: add RemoveNetwork for workspace network cleanup"
```

---

### Task 4: Wire volume and network cleanup into `rook down`

**Files:**
- Modify: `internal/cli/down.go`

- [ ] **Step 1: Update `down.go` to collect volumes, remove them, and remove network**

```go
package cli

import (
	"fmt"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newDownCmd() *cobra.Command {
	var removeVolumes bool

	cmd := &cobra.Command{
		Use:   "down [workspace]",
		Short: "Stop all services in workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}
			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}
			prefix := fmt.Sprintf("rook_%s_", wsName)
			containers, err := runner.FindContainers(prefix)
			if err != nil {
				return fmt.Errorf("finding containers: %w", err)
			}
			if len(containers) == 0 {
				fmt.Printf("No running containers found for %s.\n", wsName)
				// Still clean up the network
				networkName := fmt.Sprintf("rook_%s", wsName)
				runner.RemoveNetwork(networkName)
				return nil
			}

			// Collect named volumes before removing containers (must inspect while containers exist)
			seen := map[string]bool{}
			if removeVolumes {
				for _, name := range containers {
					vols, err := runner.ContainerVolumes(name)
					if err == nil {
						for _, v := range vols {
							seen[v] = true
						}
					}
				}
			}

			for _, name := range containers {
				fmt.Printf("Stopping %s...\n", name)
				runner.StopContainerWithVolumes(name, removeVolumes)
			}

			// Remove named volumes after containers are gone (deduplicated)
			if removeVolumes && len(seen) > 0 {
				volumes := make([]string, 0, len(seen))
				for v := range seen {
					volumes = append(volumes, v)
				}
				fmt.Printf("Removing %d volume(s)...\n", len(volumes))
				runner.RemoveVolumes(volumes)
			}

			// Clean up the workspace network
			networkName := fmt.Sprintf("rook_%s", wsName)
			runner.RemoveNetwork(networkName)

			fmt.Printf("Stopped %d container(s) for %s.\n", len(containers), wsName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&removeVolumes, "volumes", "v", false, "Remove volumes associated with containers")

	return cmd
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/cli/`
Expected: SUCCESS

- [ ] **Step 3: Run full test suite**

Run: `go test ./...`
Expected: All pass

- [ ] **Step 4: Commit**

```bash
git add internal/cli/down.go
git commit -m "fix: rook down -v now removes named volumes and always cleans up network"
```

---

### Task 5: Update existing volume test to verify named volume removal

**Files:**
- Modify: `internal/runner/docker_test.go`

- [ ] **Step 1: Update `TestStopContainerWithVolumes` comment**

The existing test at line 68-74 correctly documents that `docker rm -v` doesn't remove named volumes. No logic change needed — just update the comment to reference the new `RemoveVolumes` function for that purpose.

```go
// Named volumes are NOT removed by `docker rm -v` (only anonymous volumes).
// Use runner.RemoveVolumes() for named volume cleanup — see TestRemoveVolumes.
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/runner/ -v`
Expected: All pass

- [ ] **Step 3: Commit**

```bash
git add internal/runner/docker_test.go
git commit -m "docs: clarify named vs anonymous volume removal in test comments"
```
