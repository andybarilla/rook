package health_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/health"
)

func TestParseCheck_Command(t *testing.T) {
	check, _ := health.Parse("pg_isready -U skeetr")
	if check.Type != health.TypeCommand {
		t.Errorf("expected command, got %s", check.Type)
	}
}

func TestParseCheck_HTTP(t *testing.T) {
	check, _ := health.Parse("http://localhost:8080/health")
	if check.Type != health.TypeHTTP {
		t.Errorf("expected http, got %s", check.Type)
	}
}

func TestParseCheck_TCP(t *testing.T) {
	check, _ := health.Parse("tcp://localhost:5432")
	if check.Type != health.TypeTCP {
		t.Errorf("expected tcp, got %s", check.Type)
	}
}

func TestHTTPCheck_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()
	err := health.Run(context.Background(), health.Check{Type: health.TypeHTTP, Target: srv.URL})
	if err != nil {
		t.Errorf("expected healthy: %v", err)
	}
}

func TestHTTPCheck_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	err := health.Run(context.Background(), health.Check{Type: health.TypeHTTP, Target: srv.URL})
	if err == nil {
		t.Error("expected error for 500")
	}
}

func TestTCPCheck_Healthy(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	err := health.Run(context.Background(), health.Check{Type: health.TypeTCP, Target: ln.Addr().String()})
	if err != nil {
		t.Errorf("expected healthy: %v", err)
	}
}

func TestTCPCheck_Unhealthy(t *testing.T) {
	err := health.Run(context.Background(), health.Check{Type: health.TypeTCP, Target: "127.0.0.1:1"})
	if err == nil {
		t.Error("expected error for closed port")
	}
}

func TestWaitForHealthy_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := health.WaitUntilHealthy(ctx, health.Check{Type: health.TypeTCP, Target: "127.0.0.1:1"}, 50*time.Millisecond)
	if err == nil {
		t.Error("expected timeout")
	}
}

func TestWaitForHealthy_EventuallyHealthy(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	go func() {
		time.Sleep(100 * time.Millisecond)
		ln2, err := net.Listen("tcp", addr)
		if err != nil {
			fmt.Printf("test listener error: %v\n", err)
			return
		}
		defer ln2.Close()
		time.Sleep(2 * time.Second)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := health.WaitUntilHealthy(ctx, health.Check{Type: health.TypeTCP, Target: addr}, 50*time.Millisecond)
	if err != nil {
		t.Errorf("expected eventual health: %v", err)
	}
}
