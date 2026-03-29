package prompt_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andybarilla/rook/internal/prompt"
)

func TestSelect(t *testing.T) {
	t.Run("single_choice", func(t *testing.T) {
		r := strings.NewReader("2\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		indices, err := p.Select("Pick one:", []string{"alpha", "beta", "gamma"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indices) != 1 || indices[0] != 1 {
			t.Fatalf("expected [1], got %v", indices)
		}
		out := w.String()
		if !strings.Contains(out, "1) alpha") {
			t.Fatalf("expected numbered options in output, got: %s", out)
		}
		if !strings.Contains(out, "2) beta") {
			t.Fatalf("expected numbered options in output, got: %s", out)
		}
	})

	t.Run("multiple_choices", func(t *testing.T) {
		r := strings.NewReader("1,3\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		indices, err := p.Select("Pick:", []string{"a", "b", "c"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indices) != 2 || indices[0] != 0 || indices[1] != 2 {
			t.Fatalf("expected [0, 2], got %v", indices)
		}
	})

	t.Run("empty_input_returns_nil", func(t *testing.T) {
		r := strings.NewReader("\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		indices, err := p.Select("Pick:", []string{"a", "b"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if indices != nil {
			t.Fatalf("expected nil, got %v", indices)
		}
	})

	t.Run("out_of_range_returns_error", func(t *testing.T) {
		r := strings.NewReader("5\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		_, err := p.Select("Pick:", []string{"a", "b"})
		if err == nil {
			t.Fatal("expected error for out-of-range selection")
		}
	})

	t.Run("zero_returns_error", func(t *testing.T) {
		r := strings.NewReader("0\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		_, err := p.Select("Pick:", []string{"a", "b"})
		if err == nil {
			t.Fatal("expected error for zero input")
		}
	})

	t.Run("non_numeric_returns_error", func(t *testing.T) {
		r := strings.NewReader("abc\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		_, err := p.Select("Pick:", []string{"a", "b"})
		if err == nil {
			t.Fatal("expected error for non-numeric input")
		}
	})

	t.Run("spaces_around_numbers", func(t *testing.T) {
		r := strings.NewReader(" 1 , 2 \n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		indices, err := p.Select("Pick:", []string{"a", "b", "c"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indices) != 2 || indices[0] != 0 || indices[1] != 1 {
			t.Fatalf("expected [0, 1], got %v", indices)
		}
	})
}

func TestConfirm(t *testing.T) {
	t.Run("yes_input", func(t *testing.T) {
		r := strings.NewReader("y\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.Confirm("Continue?", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result {
			t.Fatal("expected true for 'y' input")
		}
	})

	t.Run("yes_full_word", func(t *testing.T) {
		r := strings.NewReader("yes\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.Confirm("Continue?", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result {
			t.Fatal("expected true for 'yes' input")
		}
	})

	t.Run("no_input", func(t *testing.T) {
		r := strings.NewReader("n\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.Confirm("Continue?", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result {
			t.Fatal("expected false for 'n' input")
		}
	})

	t.Run("no_full_word", func(t *testing.T) {
		r := strings.NewReader("no\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.Confirm("Continue?", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result {
			t.Fatal("expected false for 'no' input")
		}
	})

	t.Run("empty_returns_default_yes", func(t *testing.T) {
		r := strings.NewReader("\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.Confirm("Continue?", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result {
			t.Fatal("expected true for empty input with defaultYes=true")
		}
		if !strings.Contains(w.String(), "[Y/n]") {
			t.Fatalf("expected [Y/n] in prompt, got: %s", w.String())
		}
	})

	t.Run("empty_returns_default_no", func(t *testing.T) {
		r := strings.NewReader("\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.Confirm("Continue?", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result {
			t.Fatal("expected false for empty input with defaultYes=false")
		}
		if !strings.Contains(w.String(), "[y/N]") {
			t.Fatalf("expected [y/N] in prompt, got: %s", w.String())
		}
	})

	t.Run("case_insensitive", func(t *testing.T) {
		r := strings.NewReader("Y\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.Confirm("Continue?", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result {
			t.Fatal("expected true for 'Y' input")
		}
	})
}

func TestInput(t *testing.T) {
	t.Run("returns_typed_value", func(t *testing.T) {
		r := strings.NewReader("hello\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.Input("Name:", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "hello" {
			t.Fatalf("expected 'hello', got '%s'", result)
		}
	})

	t.Run("empty_returns_default", func(t *testing.T) {
		r := strings.NewReader("\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.Input("Name:", "fallback")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "fallback" {
			t.Fatalf("expected 'fallback', got '%s'", result)
		}
		if !strings.Contains(w.String(), "[fallback]") {
			t.Fatalf("expected default in brackets, got: %s", w.String())
		}
	})

	t.Run("no_default_shows_no_brackets", func(t *testing.T) {
		r := strings.NewReader("val\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.Input("Name:", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "val" {
			t.Fatalf("expected 'val', got '%s'", result)
		}
		if strings.Contains(w.String(), "[") {
			t.Fatalf("expected no brackets for empty default, got: %s", w.String())
		}
	})
}

func TestInputList(t *testing.T) {
	t.Run("comma_separated_values", func(t *testing.T) {
		r := strings.NewReader("foo, bar , baz\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.InputList("Tags:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 || result[0] != "foo" || result[1] != "bar" || result[2] != "baz" {
			t.Fatalf("expected [foo bar baz], got %v", result)
		}
	})

	t.Run("empty_returns_nil", func(t *testing.T) {
		r := strings.NewReader("\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.InputList("Tags:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Fatalf("expected nil, got %v", result)
		}
	})

	t.Run("single_value", func(t *testing.T) {
		r := strings.NewReader("only\n")
		w := &bytes.Buffer{}
		p := prompt.NewStdinPrompter(r, w)

		result, err := p.InputList("Tags:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0] != "only" {
			t.Fatalf("expected [only], got %v", result)
		}
	})
}
