package external

import (
	"io"
	"os/exec"
)

type execProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (p *execProcess) Stdin() io.WriteCloser  { return p.stdin }
func (p *execProcess) Stdout() io.ReadCloser  { return p.stdout }
func (p *execProcess) Kill() error             { return p.cmd.Process.Kill() }
func (p *execProcess) Wait() error             { return p.cmd.Wait() }

func ExecProcessStarter(exePath string) (Process, error) {
	cmd := exec.Command(exePath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &execProcess{cmd: cmd, stdin: stdin, stdout: stdout}, nil
}
