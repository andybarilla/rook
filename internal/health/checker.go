package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type CheckType string

const (
	TypeCommand CheckType = "command"
	TypeHTTP    CheckType = "http"
	TypeTCP     CheckType = "tcp"
)

type Check struct {
	Type   CheckType
	Target string
}

type Config struct {
	Interval time.Duration
	Timeout  time.Duration
	Retries  int
}

func DefaultConfig() Config {
	return Config{Interval: 2 * time.Second, Timeout: 30 * time.Second, Retries: 15}
}

func Parse(s string) (Check, error) {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return Check{Type: TypeHTTP, Target: s}, nil
	}
	if strings.HasPrefix(s, "tcp://") {
		return Check{Type: TypeTCP, Target: strings.TrimPrefix(s, "tcp://")}, nil
	}
	return Check{Type: TypeCommand, Target: s}, nil
}

func ParseFromService(hc any) (Check, Config, error) {
	cfg := DefaultConfig()
	switch v := hc.(type) {
	case string:
		check, err := Parse(v)
		return check, cfg, err
	case map[string]any:
		if test, ok := v["test"].(string); ok {
			check, err := Parse(test)
			if err != nil {
				return Check{}, cfg, err
			}
			if interval, ok := v["interval"].(string); ok {
				if d, err := time.ParseDuration(interval); err == nil {
					cfg.Interval = d
				}
			}
			if timeout, ok := v["timeout"].(string); ok {
				if d, err := time.ParseDuration(timeout); err == nil {
					cfg.Timeout = d
				}
			}
			if retries, ok := v["retries"].(int); ok {
				cfg.Retries = retries
			}
			return check, cfg, nil
		}
		return Check{}, cfg, fmt.Errorf("structured healthcheck missing 'test' field")
	case nil:
		return Check{}, cfg, fmt.Errorf("no healthcheck defined")
	default:
		return Check{}, cfg, fmt.Errorf("unsupported healthcheck type: %T", hc)
	}
}

func Run(ctx context.Context, check Check) error {
	switch check.Type {
	case TypeHTTP:
		req, err := http.NewRequestWithContext(ctx, "GET", check.Target, nil)
		if err != nil {
			return err
		}
		resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("health check returned status %d", resp.StatusCode)
		}
		return nil
	case TypeTCP:
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", check.Target)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	case TypeCommand:
		return exec.CommandContext(ctx, "sh", "-c", check.Target).Run()
	default:
		return fmt.Errorf("unknown check type: %s", check.Type)
	}
}

func WaitUntilHealthy(ctx context.Context, check Check, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	if err := Run(ctx, check); err == nil {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check timed out: %w", ctx.Err())
		case <-ticker.C:
			if err := Run(ctx, check); err == nil {
				return nil
			}
		}
	}
}
