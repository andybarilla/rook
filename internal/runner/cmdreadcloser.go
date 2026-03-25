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

func (c *cmdReadCloser) Close() error {
	err := c.ReadCloser.Close()
	if c.cmd != nil {
		c.cmd.Wait()
	}
	return err
}
