package runner

import (
	"bytes"
	"sync"
)

// syncBuffer is a thread-safe bytes.Buffer for capturing process output.
// Writes and reads are serialized via a mutex.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// Bytes returns a copy of the buffer contents.
func (b *syncBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	data := make([]byte, b.buf.Len())
	copy(data, b.buf.Bytes())
	return data
}

// Len returns the number of bytes in the buffer.
func (b *syncBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}
