package discovery

import (
	"encoding/json"
	"strconv"
	"strings"
)

// DockerfileSignals holds structured information extracted from a Dockerfile.
type DockerfileSignals struct {
	ExposedPorts []int    // from EXPOSE directives
	AptPackages  []string // from apt-get install / apk add
	Stages       []string // named FROM stages (the alias after AS)
	EntryCmd     string   // CMD or ENTRYPOINT value
	InferredDeps []string // e.g., "postgres" inferred from postgresql-client
}

// packageToDep maps known apt/apk package names to service dependencies.
var packageToDep = map[string]string{
	"postgresql-client":     "postgres",
	"libpq-dev":             "postgres",
	"redis-tools":           "redis",
	"redis-server":          "redis",
	"mysql-client":          "mysql",
	"default-mysql-client":  "mysql",
	"libmysqlclient-dev":    "mysql",
	"mongodb-clients":       "mongo",
}

// ParseDockerfile extracts structured signals from a Dockerfile.
// Never returns an error — worst case returns a zero-value struct.
func ParseDockerfile(content []byte) DockerfileSignals {
	var sig DockerfileSignals
	hasEntrypoint := false

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		upper := strings.ToUpper(trimmed)

		switch {
		case strings.HasPrefix(upper, "EXPOSE "):
			sig.ExposedPorts = append(sig.ExposedPorts, parseExposePorts(trimmed[7:])...)

		case strings.HasPrefix(upper, "FROM "):
			if stage := parseFromStage(trimmed); stage != "" {
				sig.Stages = append(sig.Stages, stage)
			}

		case strings.HasPrefix(upper, "RUN "):
			sig.AptPackages = append(sig.AptPackages, parseAptPackages(trimmed[4:])...)

		case strings.HasPrefix(upper, "CMD "):
			if !hasEntrypoint {
				sig.EntryCmd = parseCommand(trimmed[4:])
			}

		case strings.HasPrefix(upper, "ENTRYPOINT "):
			sig.EntryCmd = parseCommand(trimmed[11:])
			hasEntrypoint = true
		}
	}

	// Infer dependencies from packages
	seen := make(map[string]bool)
	for _, pkg := range sig.AptPackages {
		if dep, ok := packageToDep[pkg]; ok && !seen[dep] {
			sig.InferredDeps = append(sig.InferredDeps, dep)
			seen[dep] = true
		}
	}

	return sig
}

// parseExposePorts parses "8080 3000/tcp" into [8080, 3000].
func parseExposePorts(s string) []int {
	var ports []int
	for _, token := range strings.Fields(s) {
		// Strip protocol suffix like /tcp, /udp
		portStr := strings.Split(token, "/")[0]
		if port, err := strconv.Atoi(portStr); err == nil {
			ports = append(ports, port)
		}
	}
	return ports
}

// parseFromStage extracts the alias from "FROM image AS alias".
func parseFromStage(line string) string {
	parts := strings.Fields(line)
	for i, p := range parts {
		if strings.EqualFold(p, "AS") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// parseAptPackages extracts package names from apt-get install or apk add commands.
func parseAptPackages(runCmd string) []string {
	// Handle line continuations
	runCmd = strings.ReplaceAll(runCmd, "\\\n", " ")

	// Look for apt-get install or apk add
	var packages []string
	for _, pattern := range []string{"apt-get install", "apk add"} {
		idx := strings.Index(strings.ToLower(runCmd), pattern)
		if idx < 0 {
			continue
		}
		rest := runCmd[idx+len(pattern):]
		for _, token := range strings.Fields(rest) {
			// Skip flags
			if strings.HasPrefix(token, "-") {
				continue
			}
			// Stop at && or ; (next command)
			if token == "&&" || token == ";" {
				break
			}
			packages = append(packages, token)
		}
	}
	return packages
}

// parseCommand parses CMD/ENTRYPOINT value in JSON array or shell form.
func parseCommand(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") {
		var parts []string
		if err := json.Unmarshal([]byte(s), &parts); err == nil {
			return strings.Join(parts, " ")
		}
	}
	return s
}
