package cli

import (
	"context"
	"io"
	"os"
	"time"
)

// tailFile opens a file and returns a reader that streams its content,
// including existing data and any new data appended after opening.
// The reader closes when the context is cancelled.
func tailFile(path string, ctx context.Context) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		defer f.Close()
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				if _, writeErr := pw.Write(buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil && err != io.EOF {
				return
			}
			// At EOF — poll for new data
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
		}
	}()
	return pr, nil
}
