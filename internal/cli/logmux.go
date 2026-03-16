package cli

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

var logColors = []string{
	"\033[32m", // green
	"\033[33m", // yellow
	"\033[34m", // blue
	"\033[35m", // purple
	"\033[36m", // cyan
	"\033[31m", // red
}

const colorReset = "\033[0m"

type logMux struct {
	mu  sync.Mutex
	out io.Writer
}

func newLogMux(out io.Writer) *logMux {
	return &logMux{out: out}
}

func (m *logMux) addStream(service string, r io.ReadCloser, colorIdx int) {
	defer r.Close()
	color := logColors[colorIdx%len(logColors)]
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m.mu.Lock()
		fmt.Fprintf(m.out, "%s[%-12s]%s %s\n", color, service, colorReset, line)
		m.mu.Unlock()
	}
}
