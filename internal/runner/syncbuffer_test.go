package runner

import (
	"sync"
	"testing"
)

func TestSyncBuffer_WriteAndRead(t *testing.T) {
	var buf syncBuffer
	buf.Write([]byte("hello "))
	buf.Write([]byte("world"))

	data := buf.Bytes()
	if string(data) != "hello world" {
		t.Errorf("got %q", string(data))
	}
}

func TestSyncBuffer_Len(t *testing.T) {
	var buf syncBuffer
	if buf.Len() != 0 {
		t.Error("expected 0 length")
	}
	buf.Write([]byte("abc"))
	if buf.Len() != 3 {
		t.Errorf("expected 3, got %d", buf.Len())
	}
}

func TestSyncBuffer_ConcurrentAccess(t *testing.T) {
	var buf syncBuffer
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			buf.Write([]byte("x"))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = buf.Bytes()
			_ = buf.Len()
		}
	}()

	wg.Wait()
	if buf.Len() != 1000 {
		t.Errorf("expected 1000 bytes, got %d", buf.Len())
	}
}
