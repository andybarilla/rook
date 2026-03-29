package discovery

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// LocalSignal describes a detected local service signal.
type LocalSignal struct {
	Type    string // "go", "node", "python", "rust", "makefile", "procfile"
	File    string // the file that triggered detection
	Name    string // suggested service name
	Command string // suggested run command
}

// ScanLocalSignals looks for language/framework files at the top level of dir
// and returns suggested local services.
func ScanLocalSignals(dir string) []LocalSignal {
	var signals []LocalSignal

	signals = append(signals, scanGo(dir)...)
	signals = append(signals, scanNode(dir)...)
	signals = append(signals, scanMakefile(dir)...)
	signals = append(signals, scanProcfile(dir)...)
	signals = append(signals, scanPython(dir)...)
	signals = append(signals, scanRust(dir)...)

	return signals
}

func scanGo(dir string) []LocalSignal {
	gomod := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(gomod); err != nil {
		return nil
	}

	// Check for cmd/ subdirectories
	cmdDir := filepath.Join(dir, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err == nil {
		var signals []LocalSignal
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			signals = append(signals, LocalSignal{
				Type:    "go",
				File:    "go.mod",
				Name:    e.Name(),
				Command: "go run ./cmd/" + e.Name(),
			})
		}
		if len(signals) > 0 {
			return signals
		}
	}

	// No cmd dirs — suggest go run . with module name
	name := filepath.Base(dir)
	data, err := os.ReadFile(gomod)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "module ") {
				parts := strings.Split(strings.TrimPrefix(line, "module "), "/")
				name = parts[len(parts)-1]
				break
			}
		}
	}

	return []LocalSignal{{
		Type:    "go",
		File:    "go.mod",
		Name:    name,
		Command: "go run .",
	}}
}

func scanNode(dir string) []LocalSignal {
	pkgPath := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}

	var pkg struct {
		Name    string            `json:"name"`
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	name := pkg.Name
	if name == "" {
		name = filepath.Base(dir)
	}

	if _, ok := pkg.Scripts["dev"]; ok {
		return []LocalSignal{{
			Type:    "node",
			File:    "package.json",
			Name:    name,
			Command: "npm run dev",
		}}
	}
	if _, ok := pkg.Scripts["start"]; ok {
		return []LocalSignal{{
			Type:    "node",
			File:    "package.json",
			Name:    name,
			Command: "npm start",
		}}
	}

	return nil
}

func scanMakefile(dir string) []LocalSignal {
	makePath := filepath.Join(dir, "Makefile")
	f, err := os.Open(makePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var signals []LocalSignal
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for _, target := range []string{"dev", "run"} {
			if strings.HasPrefix(line, target+":") {
				signals = append(signals, LocalSignal{
					Type:    "makefile",
					File:    "Makefile",
					Name:    filepath.Base(dir),
					Command: "make " + target,
				})
			}
		}
	}
	return signals
}

func scanProcfile(dir string) []LocalSignal {
	procPath := filepath.Join(dir, "Procfile")
	f, err := os.Open(procPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var signals []LocalSignal
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		signals = append(signals, LocalSignal{
			Type:    "procfile",
			File:    "Procfile",
			Name:    strings.TrimSpace(parts[0]),
			Command: strings.TrimSpace(parts[1]),
		})
	}
	return signals
}

func scanPython(dir string) []LocalSignal {
	for _, f := range []string{"pyproject.toml", "setup.py"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return []LocalSignal{{
				Type: "python",
				File: f,
				Name: filepath.Base(dir),
			}}
		}
	}
	return nil
}

func scanRust(dir string) []LocalSignal {
	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err != nil {
		return nil
	}
	return []LocalSignal{{
		Type:    "rust",
		File:    "Cargo.toml",
		Name:    filepath.Base(dir),
		Command: "cargo run",
	}}
}
