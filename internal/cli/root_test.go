package cli_test

import (
	"bytes"
	"testing"

	"github.com/andybarilla/rook/internal/cli"
)

func TestRootCommandShowsHelp(t *testing.T) {
	var buf bytes.Buffer
	cmd := cli.NewRootCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("rook")) {
		t.Errorf("help output missing 'rook': %s", out)
	}
}

func TestRootCommandJSONFlag(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"--json", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	jsonFlag := cmd.Flag("json")
	if jsonFlag == nil {
		t.Fatal("expected --json flag to exist")
	}
}
