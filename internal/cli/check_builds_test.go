package cli

import (
	"testing"
)

func TestCheckBuildsCmd_Help(t *testing.T) {
	// Verify command is registered and has help
	cmd := NewCheckBuildsCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	if cmd.Use != "check-builds [workspace]" {
		t.Errorf("Use: got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected non-empty Short")
	}
}
