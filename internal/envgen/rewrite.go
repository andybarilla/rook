package envgen

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Rewrite detects host and port values in a string and replaces them
// with rook template tags for the given service name.
func Rewrite(value string, serviceName string) (string, error) {
	hostTag := fmt.Sprintf("{{.Host.%s}}", serviceName)
	portTag := fmt.Sprintf("{{.Port.%s}}", serviceName)

	// URL
	if strings.Contains(value, "://") {
		return rewriteURL(value, hostTag, portTag)
	}

	// Host:Port — split on last colon, validate port side is numeric
	if host, port, ok := splitHostPort(value); ok {
		_ = host // validated by splitHostPort
		_ = port
		return hostTag + ":" + portTag, nil
	}

	// Bare port (numeric string)
	if _, err := strconv.Atoi(value); err == nil {
		return portTag, nil
	}

	// Bare host (localhost or IPv4)
	if isKnownHost(value) {
		return hostTag, nil
	}

	return "", fmt.Errorf("cannot detect host or port in value %q", value)
}

func rewriteURL(value string, hostTag string, portTag string) (string, error) {
	u, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	host := u.Hostname()
	port := u.Port()

	if host == "" && port == "" {
		return "", fmt.Errorf("cannot detect host or port in value %q", value)
	}

	result := value
	if host != "" && port != "" {
		oldHostPort := host + ":" + port
		newHostPort := hostTag + ":" + portTag
		result = strings.Replace(result, oldHostPort, newHostPort, 1)
	} else if host != "" {
		result = strings.Replace(result, host, hostTag, 1)
	} else if port != "" {
		result = strings.Replace(result, ":"+port, ":"+portTag, 1)
	}

	return result, nil
}

// splitHostPort splits "host:port" where port is numeric.
// Returns false if the value doesn't match this pattern.
func splitHostPort(value string) (string, string, bool) {
	idx := strings.LastIndex(value, ":")
	if idx < 1 || idx == len(value)-1 {
		return "", "", false
	}
	host := value[:idx]
	port := value[idx+1:]
	if _, err := strconv.Atoi(port); err != nil {
		return "", "", false
	}
	return host, port, true
}

func isKnownHost(value string) bool {
	if value == "localhost" {
		return true
	}
	ip := net.ParseIP(value)
	return ip != nil && ip.To4() != nil
}
