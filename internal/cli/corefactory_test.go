package cli_test

import (
	"testing"

	"github.com/andybarilla/flock/internal/cli"
)

func TestNewCoreReturnsNonNil(t *testing.T) {
	c, cleanup, err := cli.NewCore()
	if err != nil {
		t.Fatalf("NewCore: %v", err)
	}
	defer cleanup()

	if c == nil {
		t.Fatal("expected non-nil Core")
	}
}

func TestNewCoreListSitesEmpty(t *testing.T) {
	c, cleanup, err := cli.NewCore()
	if err != nil {
		t.Fatalf("NewCore: %v", err)
	}
	defer cleanup()

	sites := c.Sites()
	_ = sites
}
