package runner

import (
	"io"
	"os/exec"
)

// cmdReadCloser wraps an io.ReadCloser with an optional exec.Cmd.
// Close() closes the reader and waits for the command to exit.
type cmdReadCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

// NewCmdReadCloser wraps an io.ReadCloser with an exec.Cmd that is waited on Close.
func NewCmdReadCloser(r io.ReadCloser, cmd *exec.Cmd) io.ReadCloser {
	return &cmdReadCloser{ReadCloser: r, cmd: cmd}
}

func (c *cmdReadCloser) Close() error {
	err := c.ReadCloser.Close()
	if c.cmd != nil {
		c.cmd.Wait()
	}
	return err
}
