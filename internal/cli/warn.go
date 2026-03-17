package cli

import (
	"fmt"
	"io"
)

const colorYellow = "\033[33m"

type warnings []string

func (w *warnings) add(format string, args ...any) {
	*w = append(*w, fmt.Sprintf(format, args...))
}

func (w warnings) print(out io.Writer) {
	for _, msg := range w {
		fmt.Fprintf(out, "%s⚠ %s%s\n", colorYellow, msg, colorReset)
	}
}
