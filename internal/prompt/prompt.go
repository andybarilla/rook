package prompt

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Prompter handles interactive user prompts.
// The interface exists so init can be tested with a mock.
type Prompter interface {
	// Select shows numbered options and returns selected indices (1-based input, 0-based output).
	// Empty input returns nil (skip). Accepts comma-separated numbers.
	Select(message string, options []string) ([]int, error)
	// Confirm asks a yes/no question. Returns true for y/yes.
	Confirm(message string, defaultYes bool) (bool, error)
	// Input asks for free-form text. Returns default if skipped.
	Input(message string, defaultValue string) (string, error)
	// InputList asks for comma-separated values. Returns nil if skipped.
	InputList(message string) ([]string, error)
}

// StdinPrompter implements Prompter by reading from an io.Reader and writing to an io.Writer.
type StdinPrompter struct {
	scanner *bufio.Scanner
	out     io.Writer
}

// NewStdinPrompter creates a Prompter that reads from r and writes prompts to w.
func NewStdinPrompter(r io.Reader, w io.Writer) *StdinPrompter {
	return &StdinPrompter{
		scanner: bufio.NewScanner(r),
		out:     w,
	}
}

func (p *StdinPrompter) readLine() (string, error) {
	if p.scanner.Scan() {
		return strings.TrimSpace(p.scanner.Text()), nil
	}
	if err := p.scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

// Select prints numbered options, reads comma-separated 1-based numbers, returns 0-based indices.
// Empty input returns nil. Invalid or out-of-range numbers return an error.
func (p *StdinPrompter) Select(message string, options []string) ([]int, error) {
	fmt.Fprintln(p.out, message)
	for i, opt := range options {
		fmt.Fprintf(p.out, "  %d) %s\n", i+1, opt)
	}
	fmt.Fprint(p.out, "Choose [comma-separated numbers, or empty to skip]: ")

	line, err := p.readLine()
	if err != nil {
		return nil, err
	}
	if line == "" {
		return nil, nil
	}

	var indices []int
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid selection %q: not a number", part)
		}
		if n < 1 || n > len(options) {
			return nil, fmt.Errorf("selection %d out of range (1-%d)", n, len(options))
		}
		indices = append(indices, n-1)
	}
	return indices, nil
}

// Confirm asks a yes/no question. Prints [Y/n] or [y/N] based on defaultYes.
// "y"/"yes" returns true, "n"/"no" returns false, empty returns the default.
func (p *StdinPrompter) Confirm(message string, defaultYes bool) (bool, error) {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}
	fmt.Fprintf(p.out, "%s %s ", message, hint)

	line, err := p.readLine()
	if err != nil {
		return false, err
	}
	if line == "" {
		return defaultYes, nil
	}
	lower := strings.ToLower(line)
	return lower == "y" || lower == "yes", nil
}

// Input asks for free-form text. Shows default in brackets if non-empty.
// Empty input returns the default value.
func (p *StdinPrompter) Input(message string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(p.out, "%s [%s] ", message, defaultValue)
	} else {
		fmt.Fprintf(p.out, "%s ", message)
	}

	line, err := p.readLine()
	if err != nil {
		return "", err
	}
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}

// InputList asks for comma-separated values. Trims whitespace from each value.
// Empty input returns nil.
func (p *StdinPrompter) InputList(message string) ([]string, error) {
	fmt.Fprintf(p.out, "%s ", message)

	line, err := p.readLine()
	if err != nil {
		return nil, err
	}
	if line == "" {
		return nil, nil
	}

	var result []string
	for _, part := range strings.Split(line, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result, nil
}
