package buildcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// HashFile returns the SHA256 hash of a file's contents.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("hashing file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hashing file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
