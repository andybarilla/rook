package envgen

import (
	"fmt"
	"net/url"
	"strings"
)

// Rewrite detects host and port values in a string and replaces them
// with rook template tags for the given service name.
func Rewrite(value string, serviceName string) (string, error) {
	hostTag := fmt.Sprintf("{{.Host.%s}}", serviceName)
	portTag := fmt.Sprintf("{{.Port.%s}}", serviceName)

	if strings.Contains(value, "://") {
		return rewriteURL(value, hostTag, portTag)
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
