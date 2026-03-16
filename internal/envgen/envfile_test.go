package envgen_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/envgen"
)

func TestParseEnvFile_BasicKeyValue(t *testing.T) {
	path := writeEnvFile(t, "DB_HOST=localhost\nDB_PORT=5432\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["DB_HOST"] != "localhost" {
		t.Errorf("DB_HOST: got %q", result["DB_HOST"])
	}
	if result["DB_PORT"] != "5432" {
		t.Errorf("DB_PORT: got %q", result["DB_PORT"])
	}
}

func TestParseEnvFile_CommentsAndBlankLines(t *testing.T) {
	path := writeEnvFile(t, "# this is a comment\n\nKEY=value\n\n# another comment\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 key, got %d: %v", len(result), result)
	}
	if result["KEY"] != "value" {
		t.Errorf("KEY: got %q", result["KEY"])
	}
}

func TestParseEnvFile_DoubleQuotes(t *testing.T) {
	path := writeEnvFile(t, `KEY="value with spaces"`+"\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value with spaces" {
		t.Errorf("KEY: got %q", result["KEY"])
	}
}

func TestParseEnvFile_SingleQuotes(t *testing.T) {
	path := writeEnvFile(t, `KEY='value with spaces'`+"\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value with spaces" {
		t.Errorf("KEY: got %q", result["KEY"])
	}
}

func TestParseEnvFile_ExportPrefix(t *testing.T) {
	path := writeEnvFile(t, "export KEY=value\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value" {
		t.Errorf("KEY: got %q", result["KEY"])
	}
}

func TestParseEnvFile_EmptyValue(t *testing.T) {
	path := writeEnvFile(t, "KEY=\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := result["KEY"]; !ok || v != "" {
		t.Errorf("KEY: expected empty string, got %q (ok=%v)", v, ok)
	}
}

func TestParseEnvFile_DuplicateKeysLastWins(t *testing.T) {
	path := writeEnvFile(t, "KEY=first\nKEY=second\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "second" {
		t.Errorf("KEY: got %q, want 'second'", result["KEY"])
	}
}

func TestParseEnvFile_NoEqualsSkipped(t *testing.T) {
	path := writeEnvFile(t, "VALID=yes\nINVALIDLINE\nALSO_VALID=yes\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 keys, got %d: %v", len(result), result)
	}
}

func TestParseEnvFile_ValueWithEquals(t *testing.T) {
	path := writeEnvFile(t, "DATABASE_URL=postgres://u:p@host:5432/db?sslmode=disable\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["DATABASE_URL"] != "postgres://u:p@host:5432/db?sslmode=disable" {
		t.Errorf("DATABASE_URL: got %q", result["DATABASE_URL"])
	}
}

func TestParseEnvFile_FileNotFound(t *testing.T) {
	_, err := envgen.ParseEnvFile("/nonexistent/.env")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
